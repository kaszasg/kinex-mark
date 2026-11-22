package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/storetest"
)

// privacySharedLink creates a link of owner shared with recipient and
// returns its id.
func privacySharedLink(t *testing.T, e *testEnv, owner, recipient model.User) int64 {
	t.Helper()
	l, err := e.store.CreateLink(e.ctx(), model.NewLink{
		OwnerID: owner.ID, URL: "https://example.com/" + owner.Username, Host: "example.com", Title: "Shared",
	})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	rid := recipient.ID
	if _, err := e.store.CreateShare(e.ctx(), owner.Viewer(), l.ID, model.ShareTarget{TargetType: model.ShareUser, TargetUserID: &rid}); err != nil {
		t.Fatalf("create share: %v", err)
	}
	return l.ID
}

// privacyVisibleShared counts the links shared with u.
func privacyVisibleShared(t *testing.T, e *testEnv, u model.User) int {
	t.Helper()
	page, err := e.store.Links(e.ctx(), u.Viewer(), model.LinkQuery{
		Scope: model.ScopeShared, Archived: model.ArchivedInclude, Sort: model.SortCreated, TagMatch: model.TagMatchAny, Limit: 100,
	})
	if err != nil {
		t.Fatalf("links: %v", err)
	}
	return page.Total
}

// privacyAuditNames returns actor_name and target_name of every audit row.
func privacyAuditNames(t *testing.T, e *testEnv) string {
	t.Helper()
	rows, err := authDB(t, e).Query(`SELECT COALESCE(actor_name, ''), COALESCE(target_name, '') FROM audit_log`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var b strings.Builder
	for rows.Next() {
		var a, tn string
		if err := rows.Scan(&a, &tn); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&b, "%s|%s\n", a, tn)
	}
	return b.String()
}

func TestPrivacyStatusAndConsent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := e.loginAs(t, model.RoleDeveloper)
	other := e.seedUser(t, model.RoleCustomer)
	privacySharedLink(t, e, u, other)

	res := c.get("/api/me/privacy")
	st := decodeStatus[privacyStatusResponse](t, res, http.StatusOK)
	if st.CurrentVersion != 1 || st.ConsentRequired || len(st.Consents) != 1 || st.Consents[0].Version != 1 || st.OwnSharedLinkCount != 1 {
		t.Errorf("status %+v", st)
	}
	if keys := strings.Join(jsonKeys(t, readBody(t, res)), ","); keys != "consentRequired,consents,currentVersion,ownSharedLinkCount" {
		t.Errorf("keys %s", keys)
	}

	// Acknowledging the current version again changes nothing.
	expectStatus(t, c.post("/api/me/privacy/consent", map[string]int{"version": 1}), http.StatusNoContent)
	authExpectAudit(t, e, model.EventPrivacyConsentAccepted, 0)

	expectFields(t, c.post("/api/me/privacy/consent", map[string]any{}), map[string]string{"version": fieldRequired})
	expectError(t, c.post("/api/me/privacy/consent", map[string]any{"version": "1"}), http.StatusBadRequest, codeInvalidJSON)
	expectError(t, c.post("/api/me/privacy/consent", map[string]int{"version": 2}), http.StatusConflict, codeConsentVersionMismatch)

	// After a bump everyone must acknowledge again.
	if _, err := e.store.BumpPrivacyNotice(e.ctx(), 0); err != nil {
		t.Fatal(err)
	}
	expectError(t, c.get("/api/me/sessions"), http.StatusForbidden, codeConsentRequired)
	st = decodeStatus[privacyStatusResponse](t, c.get("/api/me/privacy"), http.StatusOK)
	if st.CurrentVersion != 2 || !st.ConsentRequired || len(st.Consents) != 1 {
		t.Errorf("status after bump %+v", st)
	}
	expectError(t, c.post("/api/me/privacy/consent", map[string]int{"version": 1}), http.StatusConflict, codeConsentVersionMismatch)
	expectStatus(t, c.post("/api/me/privacy/consent", map[string]int{"version": 2}), http.StatusNoContent)
	a := authExpectAudit(t, e, model.EventPrivacyConsentAccepted, 1)[0]
	if a.Details[model.DetailVersion] != float64(2) || a.Actor.ID == nil || *a.Actor.ID != u.ID {
		t.Errorf("consent audit %+v", a)
	}
	expectStatus(t, c.get("/api/me/sessions"), http.StatusOK)
	st = decodeStatus[privacyStatusResponse](t, c.get("/api/me/privacy"), http.StatusOK)
	if st.ConsentRequired || len(st.Consents) != 2 {
		t.Errorf("status after consent %+v", st)
	}
}

func TestPrivacyPublicConfig(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	name, addr := "Kinex Kft.", "1111 Budapest\nFő utca 1."
	if _, err := e.store.UpdateSettings(e.ctx(), model.SettingsPatch{
		PrivacyControllerName: &name, PrivacyControllerAddress: &addr,
	}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := e.store.BumpPrivacyNotice(e.ctx(), 0); err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Privacy publicPrivacy `json:"privacy"`
	}
	decodeInto(t, expectStatus(t, e.client(t).get("/api/public/config"), http.StatusOK), &cfg)
	p := cfg.Privacy
	if p.ControllerName != name || p.ControllerAddress != addr || p.NoticeVersion != 2 || p.UpdatedAt == nil ||
		p.AuditRetentionDays != 180 || p.SessionIdleMinutes != 720 || p.SessionAbsoluteHours == 0 {
		t.Errorf("privacy %+v", p)
	}
}

func TestPrivacyExport(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleDeveloper)
	other := e.seedUser(t, model.RoleCustomer)
	privacySharedLink(t, e, u, other)

	res := c.get("/api/me/export")
	body := expectStatus(t, res, http.StatusOK)
	want := fmt.Sprintf(`attachment; filename="kinex-mark-export-%s-%s.json"`, u.Username, e.clock.Now().UTC().Format("20060102"))
	if got := res.Header.Get("Content-Disposition"); got != want {
		t.Errorf("Content-Disposition %q, want %q", got, want)
	}
	var exp model.UserExport
	decodeInto(t, body, &exp)
	if exp.Format != model.UserExportFormat || exp.User.ID != u.ID || exp.User.Username != u.Username || exp.User.Role != model.RoleDeveloper {
		t.Errorf("export user %+v", exp.User)
	}
	if len(exp.Links) != 1 || len(exp.Sessions) != 1 || len(exp.Consents) != 1 {
		t.Errorf("export counts links %d sessions %d consents %d", len(exp.Links), len(exp.Sessions), len(exp.Consents))
	}
	if len(exp.AuditEvents) == 0 {
		t.Error("no own audit events")
	}
	// No secrets in the export.
	lower := strings.ToLower(string(body))
	for _, word := range []string{"password_hash", "passwordhash", "totp_secret", "totpsecret", "tokenhash", "token_hash", authPassword} {
		if strings.Contains(lower, strings.ToLower(word)) {
			t.Errorf("export contains %q", word)
		}
	}
	// Every list is an array, never null.
	var raw map[string]json.RawMessage
	decodeInto(t, body, &raw)
	for _, k := range []string{"consents", "sessions", "tags", "links", "sharedWithMe", "auditEvents"} {
		if !strings.HasPrefix(string(raw[k]), "[") {
			t.Errorf("%s is %s", k, raw[k])
		}
	}
	authExpectAudit(t, e, model.EventGDPRExport, 1)

	// Exempt from the gates: available while the password must change.
	u2 := storetest.SeedNewUser(t, e.store, model.NewUser{
		Username: authUsername("exp"), Role: model.RoleCustomer, PasswordHash: storetest.SeedPasswordHash, MustChangePassword: true,
	})
	expectStatus(t, e.loginUser(t, u2).get("/api/me/export"), http.StatusOK)
}

func TestPrivacyDeleteAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleDeveloper)
	recipient := e.seedUser(t, model.RoleCustomer)
	privacySharedLink(t, e, u, recipient)
	if privacyVisibleShared(t, e, recipient) != 1 {
		t.Fatal("share not visible")
	}
	del := func(body any) *http.Response { return c.post("/api/me/delete", body) }

	expectFields(t, del(map[string]string{}), map[string]string{"password": fieldRequired, "confirm": fieldRequired})
	expectError(t, del(map[string]string{"password": authPassword, "confirm": "someone-else"}), http.StatusBadRequest, codeConfirmationMismatch)
	expectError(t, del(map[string]string{"password": authPassword, "confirm": strings.ToUpper(u.Username)}), http.StatusBadRequest, codeConfirmationMismatch)
	expectError(t, del(map[string]string{"password": "wrong password!", "confirm": u.Username}), http.StatusBadRequest, codeCurrentPasswordInvalid)
	expectError(t, del(map[string]string{"password": authPassword, "confirm": u.Username, "extra": "x"}), http.StatusBadRequest, codeInvalidJSON)

	res := del(map[string]string{"password": authPassword, "confirm": " " + u.Username + " "})
	expectStatus(t, res, http.StatusNoContent)
	if ck := responseCookie(res, e.cookieName()); ck == nil || ck.MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", ck)
	}
	expectError(t, c.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	if _, err := e.store.UserByID(e.ctx(), u.ID); err != model.ErrNotFound {
		t.Errorf("user still there: %v", err)
	}
	// The recipient lost the shared link.
	if n := privacyVisibleShared(t, e, recipient); n != 0 {
		t.Errorf("recipient still sees %d shared links", n)
	}
	// The audit trail is pseudonymised, the deletion itself included.
	names := privacyAuditNames(t, e)
	if strings.Contains(strings.ToLower(names), strings.ToLower(u.Username)) {
		t.Errorf("audit still names the user:\n%s", names)
	}
	alias := fmt.Sprintf("deleted-user-%d", u.ID)
	d := authExpectAudit(t, e, model.EventUserSelfDeleted, 1)[0]
	if d.Actor.Username != alias || d.Target == nil || d.Target.Name != alias || d.IP != "127.0.0.1" {
		t.Errorf("self delete audit %+v", d)
	}
	// The account cannot sign in any more.
	expectError(t, authLogin(e.client(t), u.Username, authPassword), http.StatusUnauthorized, codeInvalidCredentials)
}

func TestPrivacyDeleteAccountTwoFactor(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	secret, _ := authEnable2FA(t, e, c)
	body := map[string]string{"password": authPassword, "confirm": u.Username}
	expectFields(t, c.post("/api/me/delete", body), map[string]string{"code": fieldRequired})
	body["code"] = "000000"
	expectError(t, c.post("/api/me/delete", body), http.StatusUnauthorized, codeInvalidTOTP)
	if _, err := e.store.UserByID(e.ctx(), u.ID); err != nil {
		t.Fatalf("user deleted by a wrong code: %v", err)
	}
	body["code"] = authNextCode(t, e, secret)
	expectStatus(t, c.post("/api/me/delete", body), http.StatusNoContent)
	if _, err := e.store.UserByID(e.ctx(), u.ID); err != model.ErrNotFound {
		t.Errorf("user still there: %v", err)
	}
}

func TestPrivacyDeleteLastAdmin(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleAdmin)
	// Other seeded admins would make this one not the last: remove them.
	list, _, err := e.store.AdminUsers(e.ctx(), model.UserFilter{Role: model.RoleAdmin, Status: model.UserStatusActive}, model.PageReq{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	off := false
	for _, a := range list {
		if a.ID != u.ID {
			if _, err := e.store.UpdateUser(e.ctx(), a.ID, model.UserPatch{Active: &off}); err != nil {
				t.Fatal(err)
			}
		}
	}
	expectError(t, c.post("/api/me/delete", map[string]string{"password": authPassword, "confirm": u.Username}), http.StatusConflict, codeLastAdmin)
}

func TestPrivacyFaults(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	call := func(method, path string, body func(u model.User) any) func() func() *http.Response {
		return func() func() *http.Response {
			c, u := authSignIn(t, e, model.RoleCustomer)
			return func() *http.Response { return c.do(method, path, body(u)) }
		}
	}
	none := func(model.User) any { return nil }
	consent := func(model.User) any { return map[string]int{"version": 1} }
	remove := func(u model.User) any { return map[string]string{"password": authPassword, "confirm": u.Username} }
	needsConsent := func() func() *http.Response {
		u := storetest.SeedNewUser(t, e.store, model.NewUser{
			Username: authUsername("cons"), Role: model.RoleCustomer, PasswordHash: storetest.SeedPasswordHash,
		})
		c := e.loginUser(t, u)
		return func() *http.Response { return c.post("/api/me/privacy/consent", map[string]int{"version": 1}) }
	}
	get, post := http.MethodGet, http.MethodPost
	runAccountFaults(t, fs, []accountFault{
		{name: "status settings", method: "Settings", nth: 2, prepare: call(get, "/api/me/privacy", none)},
		{name: "status consents", method: "Consents", nth: 1, prepare: call(get, "/api/me/privacy", none)},
		{name: "status shared", method: "OwnSharedLinkCount", nth: 1, prepare: call(get, "/api/me/privacy", none)},
		{name: "consent settings", method: "Settings", nth: 2, prepare: call(post, "/api/me/privacy/consent", consent)},
		{name: "consent check", method: "Consents", nth: 1, prepare: call(post, "/api/me/privacy/consent", consent)},
		{name: "consent record", method: "RecordConsent", nth: 1, prepare: needsConsent},
		{name: "export", method: "ExportUser", nth: 1, prepare: call(get, "/api/me/export", none)},
		{name: "delete step-up", method: "Credentials", nth: 1, prepare: call(post, "/api/me/delete", remove)},
		{name: "delete store", method: "DeleteUser", nth: 1, prepare: call(post, "/api/me/delete", remove)},
	})
}

func TestPrivacyExportContents(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	adminC, _ := authSignIn(t, e, model.RoleAdmin)
	_, u := authSignIn(t, e, model.RoleCustomer)
	owner := e.seedUser(t, model.RoleDeveloper)

	// A link shared with u that u has visited, then the share is revoked.
	l, err := e.store.CreateLink(e.ctx(), model.NewLink{OwnerID: owner.ID, URL: "https://example.org/x", Host: "example.org", Title: "X"})
	if err != nil {
		t.Fatal(err)
	}
	uid := u.ID
	sh, err := e.store.CreateShare(e.ctx(), owner.Viewer(), l.ID, model.ShareTarget{TargetType: model.ShareUser, TargetUserID: &uid})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.store.RecordVisit(e.ctx(), u.Viewer(), l.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.store.DeleteShare(e.ctx(), owner.Viewer(), l.ID, sh.ID); err != nil {
		t.Fatal(err)
	}
	// The admin resets u's password (u is the target), then u fails to sign
	// in until locked (u is the actor).
	tmp := decodeStatus[temporaryPasswordResponse](t, adminC.post(adminUsersPath(u.ID, "/reset-password"), adminUsersStepUp), http.StatusOK)
	for i := 0; i < 5; i++ {
		authLogin(e.client(t), u.Username, "wrong password!")
	}
	// Sign in with a fresh session made directly (the account is locked).
	uc := e.loginUser(t, u)
	var exp model.UserExport
	decodeInto(t, expectStatus(t, uc.get("/api/me/export"), http.StatusOK), &exp)

	var target, actor bool
	for _, ev := range exp.AuditEvents {
		if ev.Event == model.EventUserPasswordReset && ev.Role == model.ExportAuditTarget {
			target = true
		}
		if ev.Event == model.EventLoginFailure && ev.Role == model.ExportAuditActor {
			actor = true
		}
	}
	if !target || !actor {
		t.Errorf("audit events %+v", exp.AuditEvents)
	}
	if exp.Security.LockedUntil == nil {
		t.Error("security.lockedUntil empty after the lock")
	}
	if len(exp.SharedWithMe) != 1 || exp.SharedWithMe[0].LinkID != l.ID || exp.SharedWithMe[0].URL != nil {
		t.Errorf("sharedWithMe %+v", exp.SharedWithMe)
	}
	if tmp.TemporaryPassword == nil || strings.Contains(fmt.Sprint(exp), *tmp.TemporaryPassword) {
		t.Error("temporary password in the export")
	}
}

func TestPrivacyDeletionPseudonymisesEverything(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	adminC, _ := authSignIn(t, e, model.RoleAdmin)
	for _, self := range []bool{true, false} {
		c, u := authSignIn(t, e, model.RoleDeveloper)
		display := "Rejtett Személy " + u.Username
		decodeStatus[meResponse](t, c.patch("/api/me/profile", map[string]string{"displayName": display}), http.StatusOK)
		// Rows naming the account: admin actions, a failed sign-in, and a
		// failure row without actor id that carries the name only.
		expectStatus(t, adminC.post(adminUsersPath(u.ID, "/unlock"), nil), http.StatusNoContent)
		decodeStatus[model.AdminUser](t, adminC.patch(adminUsersPath(u.ID, ""), map[string]string{"displayName": display + "!"}), http.StatusOK)
		authLogin(e.client(t), u.Username, "wrong password!")
		if err := e.store.AppendAudit(e.ctx(), model.AuditEntry{
			Event: model.EventLoginFailure, Actor: model.AuditActor{Username: strings.ToUpper(u.Username)},
			Details: map[string]any{model.DetailReason: model.ReasonCredentials},
		}); err != nil {
			t.Fatal(err)
		}
		if self {
			expectStatus(t, c.post("/api/me/delete", map[string]string{"password": authPassword, "confirm": u.Username}), http.StatusNoContent)
		} else {
			data, _ := json.Marshal(adminUsersStepUp)
			expectStatus(t, adminC.doRequest(http.MethodDelete, adminUsersPath(u.ID, ""), data, nil), http.StatusNoContent)
		}
		text := strings.ToLower(authTableText(t, e, "audit_log"))
		for _, secret := range []string{u.Username, display} {
			if strings.Contains(text, strings.ToLower(secret)) {
				t.Errorf("self=%v: audit_log still contains %q:\n%s", self, secret, text)
			}
		}
		for _, table := range []string{"sessions", "login_throttle", "consent_records", "user_preferences", "links"} {
			var n int
			col := "user_id"
			if table == "links" {
				col = "owner_id"
			}
			if err := authDB(t, e).QueryRow("SELECT COUNT(*) FROM "+table+" WHERE "+col+" = ?", u.ID).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Errorf("self=%v: %d rows left in %s", self, n, table)
			}
		}
	}
}
