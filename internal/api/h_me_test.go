package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestMeUpdateProfile(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := e.loginAs(t, model.RoleCustomer)

	res := c.patch("/api/me/profile", map[string]string{"displayName": "  Kovács Éva  "})
	me := decodeStatus[meResponse](t, res, http.StatusOK)
	if me.DisplayName != "Kovács Éva" || me.ID != u.ID || me.SessionID == 0 {
		t.Errorf("me %+v", me)
	}
	// An empty body changes nothing.
	me = decodeStatus[meResponse](t, c.patch("/api/me/profile", map[string]any{}), http.StatusOK)
	if me.DisplayName != "Kovács Éva" {
		t.Errorf("display name %q", me.DisplayName)
	}
	// The name may be cleared.
	me = decodeStatus[meResponse](t, c.patch("/api/me/profile", map[string]string{"displayName": ""}), http.StatusOK)
	if me.DisplayName != "" {
		t.Errorf("display name %q", me.DisplayName)
	}
	expectFields(t, c.patch("/api/me/profile", map[string]string{"displayName": strings.Repeat("é", 101)}),
		map[string]string{"displayName": fieldTooLong})
	expectFields(t, c.patch("/api/me/profile", map[string]string{"displayName": "a\u0000b"}),
		map[string]string{"displayName": fieldInvalid})
	expectError(t, c.patch("/api/me/profile", map[string]any{"username": "other"}), http.StatusBadRequest, codeInvalidJSON)
	expectError(t, c.patch("/api/me/profile", map[string]any{"displayName": 5}), http.StatusBadRequest, codeInvalidJSON)
}

func TestMePreferences(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := e.loginAs(t, model.RoleCustomer)

	res := c.get("/api/me/preferences")
	body := expectStatus(t, res, http.StatusOK)
	assertJSON(t, body, `{"locale":"hu","theme":"dark","viewMode":"list","showIcon":true,"showUrl":true,
		"showTitle":true,"showDescription":true,"showArchived":false,"sort":"usage","tagMatch":"any",
		"archiveAfterDays":null,"effectiveArchiveAfterDays":90,"systemArchiveAfterDays":90}`)

	got := decodeStatus[model.Preferences](t, c.patch("/api/me/preferences", map[string]any{
		"locale": "en", "theme": "light", "viewMode": "tiles", "showIcon": false, "sort": "title",
		"tagMatch": "all", "archiveAfterDays": 0,
	}), http.StatusOK)
	if got.Locale != model.LocaleEN || got.Theme != "light" || got.ViewMode != "tiles" || got.ShowIcon ||
		got.Sort != "title" || got.TagMatch != "all" || got.ArchiveAfterDays == nil || *got.ArchiveAfterDays != 0 ||
		got.EffectiveArchiveAfterDays != 0 || !got.ShowURL {
		t.Errorf("preferences %+v", got)
	}
	// null resets the archive period to the system default.
	got = decodeStatus[model.Preferences](t, c.patch("/api/me/preferences", map[string]any{"archiveAfterDays": nil}), http.StatusOK)
	if got.ArchiveAfterDays != nil || got.EffectiveArchiveAfterDays != 90 || got.Locale != model.LocaleEN {
		t.Errorf("preferences after reset %+v", got)
	}

	expectFields(t, c.patch("/api/me/preferences", map[string]any{
		"locale": "fr", "theme": "pink", "viewMode": "grid", "sort": "random", "tagMatch": "none", "archiveAfterDays": 3651,
	}), map[string]string{
		"locale": fieldInvalid, "theme": fieldInvalid, "viewMode": fieldInvalid, "sort": fieldInvalid,
		"tagMatch": fieldInvalid, "archiveAfterDays": fieldOutOfRange,
	})
	expectFields(t, c.patch("/api/me/preferences", map[string]any{"archiveAfterDays": -1}),
		map[string]string{"archiveAfterDays": fieldOutOfRange})
	// The read-only fields are unknown in a patch.
	expectError(t, c.patch("/api/me/preferences", map[string]any{"effectiveArchiveAfterDays": 5}), http.StatusBadRequest, codeInvalidJSON)
	expectError(t, c.patch("/api/me/preferences", map[string]any{"systemArchiveAfterDays": 5}), http.StatusBadRequest, codeInvalidJSON)

	// The system default follows the setting.
	days := 30
	if _, err := e.store.UpdateSettings(e.ctx(), model.SettingsPatch{DefaultArchiveDays: &days}, 0); err != nil {
		t.Fatal(err)
	}
	got = decodeStatus[model.Preferences](t, c.get("/api/me/preferences"), http.StatusOK)
	if got.EffectiveArchiveAfterDays != 30 || got.SystemArchiveAfterDays != 30 {
		t.Errorf("preferences %+v", got)
	}
}

func TestMeSessions(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	e.clock.Advance(time.Minute)
	otherToken, otherSess := e.createSession(t, u.ID)
	stranger := e.seedUser(t, model.RoleCustomer)
	_, strangerSess := e.createSession(t, stranger.ID)

	list := decodeStatus[listResponse[sessionResponse]](t, c.get("/api/me/sessions"), http.StatusOK).Items
	if len(list) != 2 || list[0].ID != otherSess.ID || list[0].Current || !list[1].Current {
		t.Fatalf("sessions %+v", list)
	}
	if list[1].IP != "127.0.0.1" || list[1].UserAgent == "" || list[1].ExpiresAt.IsZero() {
		t.Errorf("session %+v", list[1])
	}
	meItemKeys(t, c.get("/api/me/sessions"), "createdAt,current,expiresAt,id,ip,lastSeenAt,userAgent")

	expectError(t, c.del("/api/me/sessions/abc"), http.StatusBadRequest, codeInvalidID)
	expectError(t, c.del("/api/me/sessions/0"), http.StatusBadRequest, codeInvalidID)
	expectError(t, c.del(fmt.Sprintf("/api/me/sessions/%d", strangerSess.ID)), http.StatusNotFound, codeNotFound)
	expectError(t, c.del("/api/me/sessions/999999"), http.StatusNotFound, codeNotFound)

	expectStatus(t, c.del(fmt.Sprintf("/api/me/sessions/%d", otherSess.ID)), http.StatusNoContent)
	stale := e.client(t)
	stale.setSessionToken(otherToken)
	expectError(t, stale.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	rev := authExpectAudit(t, e, model.EventSessionRevoked, 1)[0]
	if rev.Target == nil || rev.Target.Type != model.TargetSession || rev.Target.ID == nil || *rev.Target.ID != otherSess.ID ||
		rev.Details[model.DetailSessionID] != float64(otherSess.ID) {
		t.Errorf("revoke audit %+v", rev)
	}

	// Revoking the current session signs out.
	current := list[1].ID
	res := c.del(fmt.Sprintf("/api/me/sessions/%d", current))
	expectStatus(t, res, http.StatusNoContent)
	if ck := responseCookie(res, e.cookieName()); ck == nil || ck.MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", ck)
	}
	expectError(t, c.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	// The stranger's session is untouched.
	if l, _ := e.store.Sessions(e.ctx(), stranger.ID); len(l) != 1 {
		t.Errorf("stranger sessions %+v", l)
	}
}

// meItemKeys checks the keys of the first item of a list response.
func meItemKeys(t *testing.T, res *http.Response, want string) {
	t.Helper()
	var body struct {
		Items []map[string]any `json:"items"`
	}
	decodeInto(t, expectStatus(t, res, http.StatusOK), &body)
	if len(body.Items) == 0 {
		t.Fatal("empty list")
	}
	keys := make([]string, 0, len(body.Items[0]))
	for k := range body.Items[0] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if got := strings.Join(keys, ","); got != want {
		t.Errorf("keys %s, want %s", got, want)
	}
}

func TestMeRevokeOtherSessions(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := e.loginAs(t, model.RoleDeveloper)
	for i := 0; i < 3; i++ {
		e.createSession(t, u.ID)
	}
	got := decodeStatus[revokedResponse](t, c.post("/api/me/sessions/revoke-others", nil), http.StatusOK)
	if got.Revoked != 3 {
		t.Errorf("revoked %d", got.Revoked)
	}
	if l, _ := e.store.Sessions(e.ctx(), u.ID); len(l) != 1 {
		t.Errorf("sessions left %d", len(l))
	}
	a := authExpectAudit(t, e, model.EventSessionRevokedOthers, 1)[0]
	if a.Details[model.DetailCount] != float64(3) {
		t.Errorf("audit %+v", a)
	}
	got = decodeStatus[revokedResponse](t, c.post("/api/me/sessions/revoke-others", nil), http.StatusOK)
	if got.Revoked != 0 {
		t.Errorf("second revoke %d", got.Revoked)
	}
	expectStatus(t, c.get("/api/auth/me"), http.StatusOK)
}

func TestMeFaults(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	call := func(method, path string, body any) func() func() *http.Response {
		return func() func() *http.Response {
			c, _ := e.loginAs(t, model.RoleCustomer)
			return func() *http.Response { return c.do(method, path, body) }
		}
	}
	revokeOne := func() func() *http.Response {
		c, u := e.loginAs(t, model.RoleCustomer)
		_, s := e.createSession(t, u.ID)
		return func() *http.Response { return c.del(fmt.Sprintf("/api/me/sessions/%d", s.ID)) }
	}
	name := map[string]string{"displayName": "x"}
	runAccountFaults(t, fs, []accountFault{
		{name: "profile update", method: "UpdateUser", nth: 1, prepare: call(http.MethodPatch, "/api/me/profile", name)},
		{name: "profile me", method: "Preferences", nth: 1, prepare: call(http.MethodPatch, "/api/me/profile", name)},
		{name: "profile settings", method: "Settings", nth: 2, prepare: call(http.MethodPatch, "/api/me/profile", name)},
		{name: "preferences get", method: "Preferences", nth: 1, prepare: call(http.MethodGet, "/api/me/preferences", nil)},
		{name: "preferences patch", method: "UpdatePreferences", nth: 1, prepare: call(http.MethodPatch, "/api/me/preferences", map[string]string{"theme": "light"})},
		{name: "sessions list", method: "Sessions", nth: 1, prepare: call(http.MethodGet, "/api/me/sessions", nil)},
		{name: "session revoke", method: "DeleteSession", nth: 1, prepare: revokeOne},
		{name: "revoke others", method: "DeleteUserSessions", nth: 1, prepare: call(http.MethodPost, "/api/me/sessions/revoke-others", nil)},
	})
}
