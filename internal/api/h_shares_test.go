package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestShareCreateListDelete(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	a, aUser := linkTestLoginAs(t, e, model.RoleDeveloper)
	cust, custUser := linkTestLoginAs(t, e, model.RoleCustomer)
	dev, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	d := linkTestCreate(t, a, map[string]any{"url": "https://share.example/secret", "title": "Titkos cím"})
	p := linkTestPath(d.ID, "/shares")

	if list := decode[listResponse[model.Share]](t, a.get(p)); list.Items == nil || len(list.Items) != 0 {
		t.Fatalf("initial shares = %+v", list.Items)
	}
	expectError(t, cust.get(linkTestPath(d.ID)), http.StatusNotFound, codeNotFound)

	e.clock.Advance(time.Minute)
	byUser := decodeStatus[model.Share](t, a.post(p, map[string]any{"targetType": "user", "targetUserId": custUser.ID}), http.StatusCreated)
	if byUser.TargetType != model.ShareUser || byUser.TargetUser == nil || byUser.TargetUser.ID != custUser.ID ||
		byUser.TargetRole != nil || !byUser.CreatedAt.Equal(e.clock.Now()) {
		t.Errorf("user share = %+v", byUser)
	}
	byRole := decodeStatus[model.Share](t, a.post(p, map[string]any{"targetType": "role", "targetRole": "developer"}), http.StatusCreated)
	if byRole.TargetType != model.ShareRole || byRole.TargetRole == nil || *byRole.TargetRole != model.RoleDeveloper || byRole.TargetUser != nil {
		t.Errorf("role share = %+v", byRole)
	}
	all := decodeStatus[model.Share](t, a.post(p, map[string]any{"targetType": "all"}), http.StatusCreated)

	got := decode[LinkDetail](t, cust.get(linkTestPath(d.ID)))
	if got.SharedVia == nil || *got.SharedVia != model.ShareUser || got.SharedAt == nil || got.IsOwner {
		t.Errorf("recipient view = %+v", got)
	}
	if got := decode[LinkDetail](t, dev.get(linkTestPath(d.ID))); got.SharedVia == nil || *got.SharedVia != model.ShareRole {
		t.Errorf("developer view via = %v", got.SharedVia)
	}
	owner := decode[LinkDetail](t, a.get(linkTestPath(d.ID)))
	if owner.ShareCount != 3 || len(owner.Shares) != 3 {
		t.Errorf("owner view: shareCount %d shares %d", owner.ShareCount, len(owner.Shares))
	}
	if list := decode[listResponse[model.Share]](t, a.get(p)); len(list.Items) != 3 {
		t.Errorf("shares = %+v", list.Items)
	}

	t.Run("duplicates", func(t *testing.T) {
		expectError(t, a.post(p, map[string]any{"targetType": "all"}), http.StatusConflict, codeShareDuplicate)
		expectError(t, a.post(p, map[string]any{"targetType": "role", "targetRole": "developer"}), http.StatusConflict, codeShareDuplicate)
		expectError(t, a.post(p, map[string]any{"targetType": "user", "targetUserId": custUser.ID}), http.StatusConflict, codeShareDuplicate)
	})

	t.Run("audit", func(t *testing.T) {
		rows, _, err := e.store.Audit(e.ctx(), model.AuditFilter{Event: model.EventLinkShared}, model.PageReq{Limit: 10})
		if err != nil || len(rows) != 3 {
			t.Fatalf("link.shared rows = %d (%v)", len(rows), err)
		}
		// Newest first: all, role, user.
		r := rows[2]
		if r.Actor.ID == nil || *r.Actor.ID != aUser.ID || r.Actor.Username != aUser.Username || r.Target == nil ||
			r.Target.Type != model.TargetLink || r.Target.ID == nil || *r.Target.ID != d.ID || r.Target.Name != "" ||
			r.IP != "127.0.0.1" || !r.Success {
			t.Errorf("audit row = %+v", r)
		}
		if r.Details["targetType"] != "user" || r.Details["targetRole"] != nil || r.Details["targetUserId"] != float64(custUser.ID) {
			t.Errorf("user share details = %v", r.Details)
		}
		if rows[1].Details["targetRole"] != "developer" || rows[1].Details["targetUserId"] != nil {
			t.Errorf("role share details = %v", rows[1].Details)
		}
		for _, row := range rows {
			text := strings.ToLower(row.Target.Name + row.Actor.Username)
			for k, v := range row.Details {
				text += k + " " + strings.ToLower(strconv.Quote(linkTestDetailString(v)))
			}
			if strings.Contains(text, "share.example") || strings.Contains(text, "titkos") {
				t.Errorf("audit row leaks the link: %+v", row)
			}
		}
	})

	t.Run("delete", func(t *testing.T) {
		res := a.del(p + "/" + strconv.FormatInt(byUser.ID, 10))
		expectStatus(t, res, http.StatusNoContent)
		if len(readBody(t, res)) != 0 {
			t.Error("204 with a body")
		}
		// The customer still sees the link through the "all" share.
		if got := decode[LinkDetail](t, cust.get(linkTestPath(d.ID))); got.SharedVia == nil || *got.SharedVia != model.ShareAll {
			t.Errorf("via after removing the user share = %v", got.SharedVia)
		}
		expectError(t, a.del(p+"/"+strconv.FormatInt(byUser.ID, 10)), http.StatusNotFound, codeNotFound)
		expectStatus(t, a.del(p+"/"+strconv.FormatInt(all.ID, 10)), http.StatusNoContent)
		expectError(t, cust.get(linkTestPath(d.ID)), http.StatusNotFound, codeNotFound)
		if list := decode[listResponse[model.Share]](t, a.get(p)); len(list.Items) != 1 || list.Items[0].ID != byRole.ID {
			t.Errorf("remaining shares = %+v", list.Items)
		}
		rows, _, err := e.store.Audit(e.ctx(), model.AuditFilter{Event: model.EventLinkUnshared}, model.PageReq{Limit: 10})
		if err != nil || len(rows) != 2 {
			t.Fatalf("link.unshared rows = %d (%v)", len(rows), err)
		}
		if rows[1].Details["targetType"] != "user" || rows[1].Details["targetUserId"] != float64(custUser.ID) ||
			rows[0].Details["targetType"] != "all" || rows[0].Target == nil || *rows[0].Target.ID != d.ID {
			t.Errorf("unshare rows = %+v / %+v", rows[0], rows[1])
		}
	})
}

// linkTestDetailString renders an audit detail value for the leak check.
func linkTestDetailString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(v.(float64), 'f', -1, 64)
}

func TestShareValidation(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	a, aUser := linkTestLoginAs(t, e, model.RoleAdmin)
	inactive := e.seedUser(t, model.RoleCustomer)
	active := false
	if _, err := e.store.UpdateUser(e.ctx(), inactive.ID, model.UserPatch{Active: &active}); err != nil {
		t.Fatal(err)
	}
	other := e.seedUser(t, model.RoleCustomer)
	d := linkTestCreateURL(t, a, "https://validate.example/")
	p := linkTestPath(d.ID, "/shares")

	cases := []struct {
		name string
		body map[string]any
		want map[string]string
	}{
		{"missing type", map[string]any{}, map[string]string{"targetType": "required"}},
		{"unknown type", map[string]any{"targetType": "group"}, map[string]string{"targetType": "invalid"}},
		{"role without role", map[string]any{"targetType": "role"}, map[string]string{"targetRole": "required"}},
		{"role with user", map[string]any{"targetType": "role", "targetRole": "customer", "targetUserId": other.ID},
			map[string]string{"targetUserId": "not_allowed"}},
		{"unknown role", map[string]any{"targetType": "role", "targetRole": "guest"}, map[string]string{"targetRole": "invalid"}},
		{"user without user", map[string]any{"targetType": "user"}, map[string]string{"targetUserId": "required"}},
		{"user with role", map[string]any{"targetType": "user", "targetUserId": other.ID, "targetRole": "customer"},
			map[string]string{"targetRole": "not_allowed"}},
		{"user id zero", map[string]any{"targetType": "user", "targetUserId": 0}, map[string]string{"targetUserId": "invalid"}},
		{"all with both", map[string]any{"targetType": "all", "targetRole": "admin", "targetUserId": other.ID},
			map[string]string{"targetRole": "not_allowed", "targetUserId": "not_allowed"}},
		{"unknown user", map[string]any{"targetType": "user", "targetUserId": 424242}, map[string]string{"targetUserId": "unknown"}},
		{"inactive user", map[string]any{"targetType": "user", "targetUserId": inactive.ID}, map[string]string{"targetUserId": "unknown"}},
		{"owner itself", map[string]any{"targetType": "user", "targetUserId": aUser.ID}, map[string]string{"targetUserId": "not_allowed"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expectFields(t, a.post(p, tc.body), tc.want)
		})
	}
	expectError(t, a.post(p, map[string]any{"targetType": "all", "extra": true}), http.StatusBadRequest, codeInvalidJSON)
	expectError(t, a.post(p, map[string]any{"targetType": "role", "targetRole": 3}), http.StatusBadRequest, codeInvalidJSON)
	expectError(t, a.post("/api/links/x/shares", map[string]any{"targetType": "all"}), http.StatusBadRequest, codeInvalidID)
	expectError(t, a.post(linkTestPath(424242, "/shares"), map[string]any{"targetType": "all"}), http.StatusNotFound, codeNotFound)
	expectError(t, a.get("/api/links/0/shares"), http.StatusBadRequest, codeInvalidID)
	expectError(t, a.get(linkTestPath(424242, "/shares")), http.StatusNotFound, codeNotFound)
	expectError(t, a.del(linkTestPath(d.ID, "/shares/x")), http.StatusBadRequest, codeInvalidID)
	expectError(t, a.del("/api/links/x/shares/1"), http.StatusBadRequest, codeInvalidID)
	expectError(t, a.del(linkTestPath(d.ID, "/shares/424242")), http.StatusNotFound, codeNotFound)
	if list := decode[listResponse[model.Share]](t, a.get(p)); len(list.Items) != 0 {
		t.Errorf("rejected requests created shares: %+v", list.Items)
	}
	if rows, _, err := e.store.Audit(e.ctx(), model.AuditFilter{Event: "link.*"}, model.PageReq{Limit: 10}); err != nil || len(rows) != 0 {
		t.Errorf("audit rows after rejected requests = %d (%v)", len(rows), err)
	}

	// Customers cannot share at all (route level), even their own links.
	cust, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	own := linkTestCreateURL(t, cust, "https://customer.example/")
	expectError(t, cust.get(linkTestPath(own.ID, "/shares")), http.StatusForbidden, codeShareNotAllowed)
	expectError(t, cust.post(linkTestPath(own.ID, "/shares"), map[string]any{"targetType": "all"}), http.StatusForbidden, codeShareNotAllowed)
	expectError(t, cust.del(linkTestPath(own.ID, "/shares/1")), http.StatusForbidden, codeShareNotAllowed)
	if got := decode[LinkDetail](t, cust.get(linkTestPath(own.ID))); len(got.Shares) != 0 {
		t.Errorf("customer detail shares = %v", got.Shares)
	}
}

func TestShareAuditFailureIsLogged(t *testing.T) {
	t.Parallel()
	e, faults := linkTestFaultEnv(t)
	a, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	d := linkTestCreateURL(t, a, "https://audit.example/")
	faults.set("AppendAudit", errors.New("injected audit failure"))
	share := decodeStatus[model.Share](t, a.post(linkTestPath(d.ID, "/shares"), map[string]any{"targetType": "all"}), http.StatusCreated)
	expectStatus(t, a.del(linkTestPath(d.ID, "/shares/", strconv.FormatInt(share.ID, 10))), http.StatusNoContent)
	if n := e.logs.count("audit write failed"); n != 2 {
		t.Errorf("audit failure log lines = %d, want 2\n%s", n, e.logs.String())
	}
	if strings.Contains(e.logs.String(), "audit.example") {
		t.Error("the log contains the link URL")
	}
}
