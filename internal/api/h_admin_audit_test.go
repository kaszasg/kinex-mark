package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestAdminAuditList(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, admin := e.loginAs(t, model.RoleAdmin)
	u := authSeed(t, e, model.RoleCustomer)
	authLogin(e.client(t), u.Username, "wrong password!")
	e.clock.Advance(time.Hour)
	mid := e.clock.Now()
	decodeStatus[loginOKResponse](t, authLogin(e.client(t), u.Username, authPassword), http.StatusOK)
	expectStatus(t, c.post(adminUsersPath(u.ID, "/unlock"), nil), http.StatusNoContent)

	type page = model.Page[model.AuditEntry]
	all := decodeStatus[page](t, c.get("/api/admin/audit"), http.StatusOK)
	if all.Total != 3 || all.Limit != 50 || all.Items[0].Event != model.EventUserUnlocked {
		t.Fatalf("audit %+v", all)
	}
	// Newest first; details always an object.
	res := c.get("/api/admin/audit?event=" + model.EventUserUnlocked)
	body := expectStatus(t, res, http.StatusOK)
	if !strings.Contains(string(body), `"details":{}`) {
		t.Errorf("details not an object: %s", body)
	}

	get := func(q string) page {
		t.Helper()
		return decodeStatus[page](t, c.get("/api/admin/audit?"+q), http.StatusOK)
	}
	if p := get("event=auth.*"); p.Total != 2 {
		t.Errorf("auth.* %+v", p)
	}
	if p := get("success=false"); p.Total != 1 || p.Items[0].Event != model.EventLoginFailure {
		t.Errorf("failures %+v", p)
	}
	if p := get(fmt.Sprintf("actorId=%d", admin.ID)); p.Total != 1 {
		t.Errorf("by admin %+v", p)
	}
	if p := get("from=" + url.QueryEscape(mid.Format(time.RFC3339))); p.Total != 2 {
		t.Errorf("from %+v", p)
	}
	if p := get("to=" + url.QueryEscape(mid.Add(-time.Second).Format(time.RFC3339))); p.Total != 1 {
		t.Errorf("to %+v", p)
	}
	if p := get("limit=1&offset=2"); len(p.Items) != 1 || p.Items[0].Event != model.EventLoginFailure || p.Total != 3 {
		t.Errorf("page %+v", p)
	}

	res = c.get("/api/admin/audit?event=Auth.*&actorId=0&success=yes&from=yesterday&limit=0")
	b := expectError(t, res, http.StatusBadRequest, codeInvalidQuery)
	for _, k := range []string{"event", "actorId", "success", "from", "limit"} {
		if b.Fields[k] != fieldInvalid {
			t.Errorf("field %s: %q", k, b.Fields[k])
		}
	}
	from := url.QueryEscape(mid.Format(time.RFC3339))
	to := url.QueryEscape(mid.Add(-time.Minute).Format(time.RFC3339))
	b = expectError(t, c.get("/api/admin/audit?from="+from+"&to="+to), http.StatusBadRequest, codeInvalidQuery)
	if b.Fields["to"] != fieldInvalid {
		t.Errorf("fields %v", b.Fields)
	}
}

func TestAdminAuditEventFilter(t *testing.T) {
	t.Parallel()
	for v, want := range map[string]bool{
		"auth.login.success": true,
		"auth.*":             true,
		"2fa_x.y":            true,
		"*":                  false,
		"":                   false,
		"auth*x":             false,
		"Auth":               false,
		"auth login":         false,
		strings.Repeat("a", model.AuditEventMaxLen):       true,
		strings.Repeat("a", model.AuditEventMaxLen+1):     false,
		strings.Repeat("a", model.AuditEventMaxLen) + "*": true,
	} {
		if got := validAuditEventFilter(v); got != want {
			t.Errorf("validAuditEventFilter(%q) = %v", v, got)
		}
	}
}

func TestAdminAuditFault(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	runAccountFaults(t, fs, []accountFault{{
		name: "audit", method: "Audit", nth: 1,
		prepare: func() func() *http.Response {
			c, _ := e.loginAs(t, model.RoleAdmin)
			return func() *http.Response { return c.get("/api/admin/audit") }
		},
	}})
}
