package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/storetest"
)

// Self-tests of the shared harness in apitest_test.go.

func TestHarnessSessions(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if e.cookieName() != sessionCookiePlain || e.secure() {
		t.Fatal("plain HTTP env must use the plain cookie")
	}

	anon := e.client(t)
	if anon.sessionToken() != "" {
		t.Fatal("anonymous client has a token")
	}
	expectError(t, anon.get("/api/me/preferences"), http.StatusUnauthorized, codeUnauthenticated)

	c, u := e.loginAs(t, model.RoleDeveloper)
	if c.sessionToken() == "" || u.Role != model.RoleDeveloper || !u.Active {
		t.Fatalf("loginAs: token %q user %+v", c.sessionToken(), u)
	}
	expectPermitted(t, c.get("/api/me/preferences"))
	expectPermitted(t, c.get("/api/users/directory"))
	sessions, err := e.store.Sessions(e.ctx(), u.ID)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("sessions %v %v", sessions, err)
	}

	// loginUser with a user who has not passed the gates.
	fresh := storetest.SeedNewUser(t, e.store, model.NewUser{
		Username: "fresh", Role: model.RoleCustomer, PasswordHash: storetest.SeedPasswordHash, MustChangePassword: true,
	})
	expectError(t, e.loginUser(t, fresh).get("/api/links"), http.StatusForbidden, codePasswordChangeRequired)

	// seedPasswordUser: real hash and consent.
	pu := e.seedPasswordUser(t, "alice", model.RoleCustomer, "correct horse battery")
	creds, err := e.store.Credentials(e.ctx(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _, err := testHasher.Verify(creds.PasswordHash, "correct horse battery"); !ok || err != nil {
		t.Fatalf("password hash does not verify: %v", err)
	}
	expectPermitted(t, e.loginUser(t, pu).get("/api/links"))

	// Advancing the shared fake clock past the idle limit signs out.
	e.clock.Advance(13 * time.Hour)
	expectError(t, c.get("/api/me/preferences"), http.StatusUnauthorized, codeUnauthenticated)
	if c.sessionToken() != "" {
		t.Fatal("cleared cookie still in the jar")
	}
}

func TestHarnessSecureEnv(t *testing.T) {
	t.Parallel()
	e := newEnv(t, withTLS())
	if e.cookieName() != sessionCookieSecure {
		t.Fatal("TLS env must use the __Host- cookie")
	}
	c, _ := e.loginAs(t, model.RoleAdmin)
	expectPermitted(t, c.get("/api/admin/settings"))
	if newEnv(t, withBaseURL("https://x.example")).cookieName() != sessionCookieSecure ||
		newEnv(t, withSecureCookies(config.SecureCookiesTrue)).cookieName() != sessionCookieSecure ||
		newEnv(t, withTLS(), withSecureCookies(config.SecureCookiesFalse)).cookieName() != sessionCookiePlain {
		t.Fatal("cookie name per mode")
	}
	var edited bool
	newEnv(t, withConfig(func(*config.Config) { edited = true }), withUI(testUI()))
	if !edited {
		t.Fatal("withConfig not applied")
	}
}

func TestHarnessClientHeaders(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t)
	if c.header.Get(headerKinexRequest) != "1" || c.header.Get("Origin") != e.srv.URL || c.header.Get("Accept") != "application/json" {
		t.Fatalf("default headers %v", c.header)
	}
	d := c.withHeader("Origin", "").withHeader("X-Test", "1")
	if d.header.Get("Origin") != "" || d.header.Get("X-Test") != "1" || c.header.Get("X-Test") != "" {
		t.Fatal("withHeader must copy")
	}
	var seen *http.Request
	for _, body := range []any{"{}", []byte("{}"), bytes.NewReader([]byte("{}")), map[string]int{}} {
		res := c.doRequest(http.MethodPost, csrfProbe, body, func(r *http.Request) { seen = r })
		if seen.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("Content-Type %q", seen.Header.Get("Content-Type"))
		}
		expectPermitted(t, res)
	}
	c.withHeader("Content-Type", "text/html").doRequest(http.MethodPost, csrfProbe, "<html>", func(r *http.Request) { seen = r })
	if seen.Header.Get("Content-Type") != "text/html" {
		t.Fatal("explicit Content-Type replaced")
	}
	res := c.get("/healthz")
	if string(readBody(t, res)) != string(readBody(t, res)) {
		t.Fatal("body not re-readable")
	}
	for _, m := range []*http.Response{c.put("/api/nope", nil), c.patch("/api/nope", nil), c.del("/api/nope")} {
		expectError(t, m, http.StatusNotFound, codeEndpointNotFound)
	}
	if rec := e.serve(httptest.NewRequest(http.MethodGet, "/healthz", nil)); rec.Code != http.StatusOK {
		t.Fatalf("serve: %d", rec.Code)
	}
}

func TestHarnessAssertions(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/api/x", nil)
	mk := func(status int, body string) *http.Response {
		h := http.Header{}
		h.Set("Content-Type", "application/json; charset=utf-8")
		h.Add("Set-Cookie", "a=1; Path=/")
		return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(bytes.NewBufferString(body)), Request: req}
	}
	expectFields(t, mk(400, `{"error":"validation failed","code":"validation_failed","fields":{"url":"required"}}`),
		map[string]string{"url": "required"})
	eb := expectError(t, mk(409, `{"error":"x","code":"link_duplicate","details":{"linkId":4}}`), 409, codeLinkDuplicate)
	if eb.Details["linkId"] != float64(4) {
		t.Fatalf("details %v", eb.Details)
	}
	assertJSON(t, []byte(`{"b":[1,2],"a":null}`), `{"a": null, "b": [1, 2]}`)
	if got := jsonKeys(t, []byte(`{"z":1,"a":2,"m":3}`)); !reflect.DeepEqual(got, []string{"a", "m", "z"}) {
		t.Fatalf("keys %v", got)
	}
	if c := responseCookie(mk(200, "{}"), "a"); c == nil || c.Value != "1" {
		t.Fatalf("cookie %v", c)
	}
	if responseCookie(mk(200, "{}"), "b") != nil {
		t.Fatal("missing cookie found")
	}
	var v struct{ A int }
	decodeInto(t, []byte(`{"A":3}`), &v)
	if v.A != 3 {
		t.Fatal("decodeInto")
	}
}

func TestFakeJobs(t *testing.T) {
	t.Parallel()
	j := &fakeJobs{archiveCount: 7, archiveErr: errors.New("x")}
	if !j.Enqueue(1) {
		t.Fatal("empty queue reported full")
	}
	j.setFull(true)
	if j.Enqueue(2) {
		t.Fatal("full queue accepted")
	}
	if got := j.enqueued(); !reflect.DeepEqual(got, []int64{1, 2}) {
		t.Fatalf("enqueued %v", got)
	}
	if n, err := j.RunArchive(context.Background()); n != 7 || err == nil || j.archiveRuns() != 1 {
		t.Fatalf("RunArchive %d %v %d", n, err, j.archiveRuns())
	}
	var lb logBuffer
	_, _ = lb.Write([]byte("a x\nb\nc x\n"))
	if lb.count("x") != 2 {
		t.Fatal("logBuffer.count")
	}
}
