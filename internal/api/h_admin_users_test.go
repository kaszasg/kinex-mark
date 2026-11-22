package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/storetest"
)

// adminUsersPath is the admin route of one user.
func adminUsersPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/admin/users/%d%s", id, suffix)
}

// adminUsersStepUp is the AdminStepUp body with the right password.
var adminUsersStepUp = map[string]string{"currentPassword": authPassword}

func TestAdminUsersList(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, admin := authSignIn(t, e, model.RoleAdmin)
	prefix := authUsername("lst")
	mk := func(suffix string, role model.Role) model.User {
		return storetest.SeedNewUser(t, e.store, model.NewUser{
			Username: prefix + suffix, DisplayName: "Lista " + suffix, Role: role, PasswordHash: storetest.SeedPasswordHash,
		})
	}
	dev := mk("a", model.RoleDeveloper)
	cust := mk("b", model.RoleCustomer)
	privacySharedLink(t, e, dev, cust)
	off := false
	if _, err := e.store.UpdateUser(e.ctx(), cust.ID, model.UserPatch{Active: &off}); err != nil {
		t.Fatal(err)
	}

	type page = model.Page[model.AdminUser]
	all := decodeStatus[page](t, c.get("/api/admin/users"), http.StatusOK)
	if all.Total < 3 || all.Limit != 50 || all.Offset != 0 {
		t.Errorf("page %+v", all)
	}
	got := decodeStatus[page](t, c.get("/api/admin/users?q="+strings.ToUpper(prefix)), http.StatusOK)
	if got.Total != 2 || got.Items[0].ID != dev.ID || got.Items[0].LinkCount != 1 || got.Items[0].SharedLinkCount != 1 {
		t.Fatalf("search %+v", got)
	}
	got = decodeStatus[page](t, c.get("/api/admin/users?q="+prefix+"&status=inactive"), http.StatusOK)
	if got.Total != 1 || got.Items[0].ID != cust.ID || got.Items[0].Active {
		t.Errorf("inactive %+v", got)
	}
	got = decodeStatus[page](t, c.get("/api/admin/users?q="+prefix+"&status=active&role=developer"), http.StatusOK)
	if got.Total != 1 || got.Items[0].ID != dev.ID {
		t.Errorf("active developers %+v", got)
	}
	got = decodeStatus[page](t, c.get("/api/admin/users?q="+prefix+"&limit=1&offset=1"), http.StatusOK)
	if got.Total != 2 || len(got.Items) != 1 || got.Items[0].ID != cust.ID || got.Limit != 1 || got.Offset != 1 {
		t.Errorf("second page %+v", got)
	}
	got = decodeStatus[page](t, c.get("/api/admin/users?status=all&q="+admin.Username), http.StatusOK)
	if got.Total != 1 || got.Items[0].SessionCount != 1 || got.Items[0].LastLoginAt == nil {
		t.Errorf("admin row %+v", got)
	}

	res := c.get("/api/admin/users?role=root&status=gone&limit=201&offset=-1&q=" + strings.Repeat("q", 65))
	b := expectError(t, res, http.StatusBadRequest, codeInvalidQuery)
	for _, k := range []string{"role", "status", "limit", "offset", "q"} {
		if b.Fields[k] != fieldInvalid {
			t.Errorf("field %s: %q", k, b.Fields[k])
		}
	}
	// Not for non-admins.
	dc, _ := e.loginAs(t, model.RoleDeveloper)
	expectError(t, dc.get("/api/admin/users"), http.StatusForbidden, codeForbidden)
}

func TestAdminUsersCreate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, admin := authSignIn(t, e, model.RoleAdmin)
	name := authUsername("new")

	res := c.post("/api/admin/users", map[string]any{"username": " " + name + " ", "displayName": " Új Ember ", "role": "developer", "locale": "en"})
	created := decodeStatus[adminUserCreatedResponse](t, res, http.StatusCreated)
	u := created.User
	if u.Username != name || u.DisplayName != "Új Ember" || u.Role != model.RoleDeveloper || !u.Active || !u.MustChangePassword ||
		u.TOTPEnabled || created.TemporaryPassword == nil || len(*created.TemporaryPassword) != generatedPasswordLength {
		t.Fatalf("created %+v / %v", u, created.TemporaryPassword)
	}
	a := authExpectAudit(t, e, model.EventUserCreated, 1)[0]
	if a.Details[model.DetailRole] != "developer" || a.Target == nil || a.Target.ID == nil || *a.Target.ID != u.ID ||
		a.Actor.ID == nil || *a.Actor.ID != admin.ID {
		t.Errorf("created audit %+v", a)
	}
	prefs, err := e.store.Preferences(e.ctx(), u.ID)
	if err != nil || prefs.Locale != model.LocaleEN {
		t.Errorf("locale %+v %v", prefs, err)
	}
	// The generated password signs in and must be changed.
	nc := e.client(t)
	me := decodeStatus[loginOKResponse](t, authLogin(nc, name, *created.TemporaryPassword), http.StatusOK).User
	if !me.MustChangePassword {
		t.Error("must change flag missing")
	}

	// Given password: no temporary password in the answer.
	name2 := authUsername("new")
	res = c.post("/api/admin/users", map[string]any{"username": name2, "role": "customer", "password": authNewPassword})
	created = decodeStatus[adminUserCreatedResponse](t, res, http.StatusCreated)
	if created.TemporaryPassword != nil || created.User.DisplayName != "" {
		t.Errorf("created %+v", created)
	}
	if !strings.Contains(string(readBody(t, res)), `"temporaryPassword":null`) {
		t.Error("temporaryPassword not null")
	}
	decodeStatus[loginOKResponse](t, authLogin(e.client(t), name2, authNewPassword), http.StatusOK)
	if p, _ := e.store.Preferences(e.ctx(), created.User.ID); p.Locale != model.LocaleHU {
		t.Errorf("default locale %s", p.Locale)
	}

	expectError(t, c.post("/api/admin/users", map[string]any{"username": strings.ToUpper(name), "role": "customer"}), http.StatusConflict, codeUsernameTaken)
	expectFields(t, c.post("/api/admin/users", map[string]any{}), map[string]string{"username": fieldRequired, "role": fieldRequired})
	expectFields(t, c.post("/api/admin/users", map[string]any{
		"username": "ab", "role": "root", "locale": "fr", "displayName": strings.Repeat("x", 101),
	}), map[string]string{"username": fieldTooShort, "role": fieldInvalid, "locale": fieldInvalid, "displayName": fieldTooLong})
	expectFields(t, c.post("/api/admin/users", map[string]any{"username": "bad name", "role": "customer"}),
		map[string]string{"username": fieldInvalidFormat})
	expectFields(t, c.post("/api/admin/users", map[string]any{"username": strings.Repeat("a", 65), "role": "customer"}),
		map[string]string{"username": fieldTooLong})
	expectFields(t, c.post("/api/admin/users", map[string]any{"username": "validname", "role": "customer", "password": "short"}),
		map[string]string{"password": fieldPasswordTooShort})
}

func TestAdminUsersGet(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := e.loginAs(t, model.RoleAdmin)
	u := e.seedUser(t, model.RoleCustomer)
	res := c.get(adminUsersPath(u.ID, ""))
	got := decodeStatus[model.AdminUser](t, res, http.StatusOK)
	if got.ID != u.ID || got.Username != u.Username {
		t.Errorf("user %+v", got)
	}
	want := "active,createdAt,displayName,id,lastLoginAt,linkCount,lockedUntil,mustChangePassword,role,sessionCount," +
		"sharedLinkCount,totpEnabled,updatedAt,username"
	if keys := strings.Join(jsonKeys(t, readBody(t, res)), ","); keys != want {
		t.Errorf("keys %s", keys)
	}
	expectError(t, c.get("/api/admin/users/999999"), http.StatusNotFound, codeNotFound)
	expectError(t, c.get("/api/admin/users/x"), http.StatusBadRequest, codeInvalidID)
}

func TestAdminUsersUpdate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, admin := authSignIn(t, e, model.RoleAdmin)
	u := e.seedUser(t, model.RoleCustomer)
	tok, _ := e.createSession(t, u.ID)
	path := adminUsersPath(u.ID, "")

	// Display name only: no step-up, only the field name is audited.
	got := decodeStatus[model.AdminUser](t, c.patch(path, map[string]any{"displayName": "Titkos Név"}), http.StatusOK)
	if got.DisplayName != "Titkos Név" {
		t.Errorf("user %+v", got)
	}
	upd := authExpectAudit(t, e, model.EventUserUpdated, 1)[0]
	assertJSON(t, mustJSON(t, upd.Details), `{"changed":["displayName"]}`)
	// An unchanged value is not audited.
	decodeStatus[model.AdminUser](t, c.patch(path, map[string]any{"displayName": "Titkos Név"}), http.StatusOK)
	authExpectAudit(t, e, model.EventUserUpdated, 1)

	// Role and active need the password.
	expectFields(t, c.patch(path, map[string]any{"role": "developer"}), map[string]string{"currentPassword": fieldRequired})
	expectFields(t, c.patch(path, map[string]any{"active": false, "currentPassword": ""}), map[string]string{"currentPassword": fieldRequired})
	expectFields(t, c.patch(path, map[string]any{"role": "root", "currentPassword": authPassword}), map[string]string{"role": fieldInvalid})
	b := expectError(t, c.patch(path, map[string]any{"role": "developer", "currentPassword": "wrong password!"}), http.StatusBadRequest, codeCurrentPasswordInvalid)
	if b.Fields["currentPassword"] != fieldIncorrect {
		t.Errorf("fields %v", b.Fields)
	}
	got = decodeStatus[model.AdminUser](t, c.patch(path, map[string]any{"role": "developer", "currentPassword": authPassword}), http.StatusOK)
	if got.Role != model.RoleDeveloper {
		t.Errorf("role %s", got.Role)
	}
	upd = authExpectAudit(t, e, model.EventUserUpdated, 2)[0]
	assertJSON(t, mustJSON(t, upd.Details), `{"changed":["role"],"role":{"from":"customer","to":"developer"}}`)

	// Deactivation ends the sessions.
	got = decodeStatus[model.AdminUser](t, c.patch(path, map[string]any{"active": false, "currentPassword": authPassword}), http.StatusOK)
	if got.Active || got.SessionCount != 0 {
		t.Errorf("deactivated %+v", got)
	}
	authExpectAudit(t, e, model.EventUserDeactivated, 1)
	stale := e.client(t)
	stale.setSessionToken(tok)
	expectError(t, stale.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	decodeStatus[model.AdminUser](t, c.patch(path, map[string]any{"active": true, "currentPassword": authPassword}), http.StatusOK)
	authExpectAudit(t, e, model.EventUserActivated, 1)

	// Own account: role and active are self_modification, the name is fine.
	own := adminUsersPath(admin.ID, "")
	expectError(t, c.patch(own, map[string]any{"role": "customer", "currentPassword": authPassword}), http.StatusConflict, codeSelfModification)
	expectError(t, c.patch(own, map[string]any{"active": false, "currentPassword": authPassword}), http.StatusConflict, codeSelfModification)
	decodeStatus[model.AdminUser](t, c.patch(own, map[string]any{"role": "admin", "active": true, "currentPassword": authPassword}), http.StatusOK)
	decodeStatus[model.AdminUser](t, c.patch(own, map[string]any{"displayName": "Főnök"}), http.StatusOK)

	expectError(t, c.patch(adminUsersPath(999999, ""), map[string]any{"displayName": "x"}), http.StatusNotFound, codeNotFound)
	expectError(t, c.patch("/api/admin/users/0", map[string]any{}), http.StatusBadRequest, codeInvalidID)
	expectError(t, c.patch(path, map[string]any{"username": "x"}), http.StatusBadRequest, codeInvalidJSON)
	expectFields(t, c.patch(path, map[string]any{"displayName": strings.Repeat("x", 101)}), map[string]string{"displayName": fieldTooLong})
}

// mustJSON marshals v for a JSON comparison.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAdminUsersLastAdmin(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	c, _ := authSignIn(t, e, model.RoleAdmin)
	other := e.seedUser(t, model.RoleAdmin)
	// The store refuses the change of the last active admin.
	fs.arm("UpdateUser", 1, model.ErrLastAdmin)
	expectError(t, c.patch(adminUsersPath(other.ID, ""), map[string]any{"role": "customer", "currentPassword": authPassword}),
		http.StatusConflict, codeLastAdmin)
	fs.arm("DeleteUser", 1, model.ErrLastAdmin)
	b, _ := mustJSONBody(adminUsersStepUp)
	expectError(t, c.doRequest(http.MethodDelete, adminUsersPath(other.ID, ""), b, nil), http.StatusConflict, codeLastAdmin)
	fs.arm("", 0, nil)
}

// mustJSONBody returns v as a JSON request body.
func mustJSONBody(v any) ([]byte, error) { return json.Marshal(v) }

func TestAdminUsersStepUpThrottle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := authSignIn(t, e, model.RoleAdmin)
	u := e.seedUser(t, model.RoleCustomer)
	for i := 0; i < 5; i++ {
		expectError(t, c.post(adminUsersPath(u.ID, "/reset-password"), map[string]string{"currentPassword": "wrong password!"}),
			http.StatusBadRequest, codeCurrentPasswordInvalid)
	}
	res := c.post(adminUsersPath(u.ID, "/reset-password"), adminUsersStepUp)
	expectError(t, res, http.StatusTooManyRequests, codeTooManyAttempts)
	if res.Header.Get("Retry-After") == "" {
		t.Error("no Retry-After")
	}
}

func TestAdminUsersDelete(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, admin := authSignIn(t, e, model.RoleAdmin)
	victim := authSeed(t, e, model.RoleDeveloper)
	recipient := e.seedUser(t, model.RoleCustomer)
	privacySharedLink(t, e, victim, recipient)
	// The victim signs in once, so that an audit row names it.
	decodeStatus[loginOKResponse](t, authLogin(e.client(t), victim.Username, authPassword), http.StatusOK)
	del := func(id int64, body any) *http.Response {
		data, err := mustJSONBody(body)
		if err != nil {
			t.Fatal(err)
		}
		return c.doRequest(http.MethodDelete, adminUsersPath(id, ""), data, nil)
	}

	expectFields(t, del(victim.ID, map[string]string{}), map[string]string{"currentPassword": fieldRequired})
	expectError(t, del(victim.ID, map[string]string{"currentPassword": "wrong password!"}), http.StatusBadRequest, codeCurrentPasswordInvalid)
	expectError(t, del(admin.ID, adminUsersStepUp), http.StatusConflict, codeSelfModification)
	expectError(t, del(999999, adminUsersStepUp), http.StatusNotFound, codeNotFound)
	expectStatus(t, del(victim.ID, adminUsersStepUp), http.StatusNoContent)

	if _, err := e.store.UserByID(e.ctx(), victim.ID); err != model.ErrNotFound {
		t.Errorf("user still there: %v", err)
	}
	if n := privacyVisibleShared(t, e, recipient); n != 0 {
		t.Errorf("recipient still sees %d links", n)
	}
	names := privacyAuditNames(t, e)
	if strings.Contains(strings.ToLower(names), strings.ToLower(victim.Username)) {
		t.Errorf("audit still names the user:\n%s", names)
	}
	d := authExpectAudit(t, e, model.EventUserDeleted, 1)[0]
	alias := fmt.Sprintf("deleted-user-%d", victim.ID)
	if d.Target == nil || d.Target.Name != alias || d.Actor.Username != admin.Username {
		t.Errorf("delete audit %+v", d)
	}
	expectError(t, del(victim.ID, adminUsersStepUp), http.StatusNotFound, codeNotFound)
}

func TestAdminUsersResetPassword(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := authSignIn(t, e, model.RoleAdmin)
	uc, u := authSignIn(t, e, model.RoleCustomer)
	for i := 0; i < 5; i++ {
		authLogin(e.client(t), u.Username, "wrong password!")
	}
	path := adminUsersPath(u.ID, "/reset-password")

	expectFields(t, c.post(path, map[string]string{}), map[string]string{"currentPassword": fieldRequired})
	expectFields(t, c.post(path, map[string]any{"currentPassword": authPassword, "password": "xx" + u.Username + "-xx1234"}),
		map[string]string{"password": fieldPasswordContainsUsername})
	expectError(t, c.post(adminUsersPath(999999, "/reset-password"), adminUsersStepUp), http.StatusNotFound, codeNotFound)

	got := decodeStatus[temporaryPasswordResponse](t, c.post(path, adminUsersStepUp), http.StatusOK)
	if got.TemporaryPassword == nil || len(*got.TemporaryPassword) != generatedPasswordLength {
		t.Fatalf("reset %+v", got)
	}
	authExpectAudit(t, e, model.EventUserPasswordReset, 1)
	// Sessions gone, lock lifted, the new password must be changed.
	expectError(t, uc.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	me := decodeStatus[loginOKResponse](t, authLogin(e.client(t), u.Username, *got.TemporaryPassword), http.StatusOK).User
	if !me.MustChangePassword {
		t.Error("must change flag missing")
	}

	// A given password.
	got = decodeStatus[temporaryPasswordResponse](t, c.post(path, map[string]any{"currentPassword": authPassword, "password": authNewPassword}), http.StatusOK)
	if got.TemporaryPassword != nil {
		t.Errorf("temporary password %v", *got.TemporaryPassword)
	}
	decodeStatus[loginOKResponse](t, authLogin(e.client(t), u.Username, authNewPassword), http.StatusOK)
}

func TestAdminUsersResetTwoFactor(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := authSignIn(t, e, model.RoleAdmin)
	uc, u := authSignIn(t, e, model.RoleCustomer)
	path := adminUsersPath(u.ID, "/reset-2fa")
	expectError(t, c.post(path, adminUsersStepUp), http.StatusConflict, codeTOTPNotEnabled)
	authEnable2FA(t, e, uc)
	expectFields(t, c.post(path, map[string]string{}), map[string]string{"currentPassword": fieldRequired})
	expectError(t, c.post(path, map[string]string{"currentPassword": "wrong password!"}), http.StatusBadRequest, codeCurrentPasswordInvalid)
	expectError(t, c.post(adminUsersPath(999999, "/reset-2fa"), adminUsersStepUp), http.StatusNotFound, codeNotFound)
	expectStatus(t, c.post(path, adminUsersStepUp), http.StatusNoContent)
	authExpectAudit(t, e, model.EventUser2FAReset, 1)
	expectError(t, uc.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	me := decodeStatus[loginOKResponse](t, authLogin(e.client(t), u.Username, authPassword), http.StatusOK).User
	if me.TOTPEnabled {
		t.Error("2FA still on")
	}
}

func TestAdminUsersUnlock(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := e.loginAs(t, model.RoleAdmin)
	u := authSeed(t, e, model.RoleCustomer)
	for i := 0; i < 5; i++ {
		authLogin(e.client(t), u.Username, "wrong password!")
	}
	expectError(t, authLogin(e.client(t), u.Username, authPassword), http.StatusTooManyRequests, codeTooManyAttempts)
	if au := decodeStatus[model.AdminUser](t, c.get(adminUsersPath(u.ID, "")), http.StatusOK); au.LockedUntil == nil {
		t.Error("lockedUntil not shown")
	}
	expectStatus(t, c.post(adminUsersPath(u.ID, "/unlock"), nil), http.StatusNoContent)
	authExpectAudit(t, e, model.EventUserUnlocked, 1)
	if au := decodeStatus[model.AdminUser](t, c.get(adminUsersPath(u.ID, "")), http.StatusOK); au.LockedUntil != nil {
		t.Error("still locked")
	}
	decodeStatus[loginOKResponse](t, authLogin(e.client(t), u.Username, authPassword), http.StatusOK)
	expectError(t, c.post(adminUsersPath(999999, "/unlock"), nil), http.StatusNotFound, codeNotFound)
}

func TestAdminUsersRevokeSessions(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, admin := e.loginAs(t, model.RoleAdmin)
	u := e.seedUser(t, model.RoleCustomer)
	for i := 0; i < 2; i++ {
		e.createSession(t, u.ID)
	}
	got := decodeStatus[revokedResponse](t, c.del(adminUsersPath(u.ID, "/sessions")), http.StatusOK)
	if got.Revoked != 2 {
		t.Errorf("revoked %d", got.Revoked)
	}
	a := authExpectAudit(t, e, model.EventUserSessionsRevoked, 1)[0]
	if a.Details[model.DetailCount] != float64(2) {
		t.Errorf("audit %+v", a)
	}
	// Own account: the current session stays.
	e.createSession(t, admin.ID)
	got = decodeStatus[revokedResponse](t, c.del(adminUsersPath(admin.ID, "/sessions")), http.StatusOK)
	if got.Revoked != 1 {
		t.Errorf("own revoked %d", got.Revoked)
	}
	expectStatus(t, c.get("/api/auth/me"), http.StatusOK)
	expectError(t, c.del(adminUsersPath(999999, "/sessions")), http.StatusNotFound, codeNotFound)
}

func TestAdminUsersFaults(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	// prepare signs in a fresh admin and seeds a target user.
	call := func(method, suffix string, body func(target model.User) any, twoFA bool) func() func() *http.Response {
		return func() func() *http.Response {
			c, _ := authSignIn(t, e, model.RoleAdmin)
			var target model.User
			if twoFA {
				var tc *apiClient
				tc, target = authSignIn(t, e, model.RoleCustomer)
				authEnable2FA(t, e, tc)
			} else {
				target = e.seedUser(t, model.RoleCustomer)
			}
			path := "/api/admin/users"
			if suffix != "-" {
				path = adminUsersPath(target.ID, suffix)
			}
			var data []byte
			if body != nil {
				var err error
				if data, err = mustJSONBody(body(target)); err != nil {
					t.Fatal(err)
				}
			}
			return func() *http.Response { return c.doRequest(method, path, data, nil) }
		}
	}
	stepUp := func(model.User) any { return adminUsersStepUp }
	newUser := func(model.User) any { return map[string]string{"username": authUsername("flt"), "role": "customer"} }
	deact := func(model.User) any { return map[string]any{"active": false, "currentPassword": authPassword} }
	get, post, patch, del := http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete
	runAccountFaults(t, fs, []accountFault{
		{name: "list", method: "AdminUsers", nth: 1, prepare: call(get, "-", nil, false)},
		{name: "create", method: "CreateUser", nth: 1, prepare: call(post, "-", newUser, false)},
		{name: "create reload", method: "AdminUser", nth: 1, prepare: call(post, "-", newUser, false)},
		{name: "get", method: "AdminUser", nth: 1, prepare: call(get, "", nil, false)},
		{name: "update load", method: "UserByID", nth: 1, prepare: call(patch, "", deact, false)},
		{name: "update step-up", method: "Credentials", nth: 1, prepare: call(patch, "", deact, false)},
		{name: "update store", method: "UpdateUser", nth: 1, prepare: call(patch, "", deact, false)},
		{name: "update sessions", method: "DeleteUserSessions", nth: 1, prepare: call(patch, "", deact, false)},
		{name: "update reload", method: "AdminUser", nth: 1, prepare: call(patch, "", deact, false)},
		{name: "delete load", method: "UserByID", nth: 1, prepare: call(del, "", stepUp, false)},
		{name: "delete step-up", method: "Credentials", nth: 1, prepare: call(del, "", stepUp, false)},
		{name: "delete store", method: "DeleteUser", nth: 1, prepare: call(del, "", stepUp, false)},
		{name: "reset step-up", method: "Credentials", nth: 1, prepare: call(post, "/reset-password", stepUp, false)},
		{name: "reset set", method: "SetPassword", nth: 1, prepare: call(post, "/reset-password", stepUp, false)},
		{name: "reset sessions", method: "DeleteUserSessions", nth: 1, prepare: call(post, "/reset-password", stepUp, false)},
		{name: "reset throttle", method: "ClearUserThrottle", nth: 1, prepare: call(post, "/reset-password", stepUp, false)},
		{name: "2fa load", method: "UserByID", nth: 1, prepare: call(post, "/reset-2fa", stepUp, true)},
		{name: "2fa step-up", method: "Credentials", nth: 1, prepare: call(post, "/reset-2fa", stepUp, true)},
		{name: "2fa disable", method: "DisableTOTP", nth: 1, prepare: call(post, "/reset-2fa", stepUp, true)},
		{name: "2fa sessions", method: "DeleteUserSessions", nth: 1, prepare: call(post, "/reset-2fa", stepUp, true)},
		{name: "unlock", method: "ClearUserThrottle", nth: 1, prepare: call(post, "/unlock", nil, false)},
		{name: "revoke", method: "DeleteUserSessions", nth: 1, prepare: call(del, "/sessions", nil, false)},
	})
}

func TestAdminUsersGeneratedPasswordsDiffer(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := e.loginAs(t, model.RoleAdmin)
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		got := decodeStatus[adminUserCreatedResponse](t, c.post("/api/admin/users", map[string]string{"username": authUsername("gen"), "role": "customer"}), http.StatusCreated)
		if seen[*got.TemporaryPassword] {
			t.Fatal("generated password repeated")
		}
		seen[*got.TemporaryPassword] = true
		if got.User.CreatedAt.IsZero() || got.User.CreatedAt.After(e.clock.Now().Add(time.Second)) {
			t.Errorf("createdAt %v", got.User.CreatedAt)
		}
	}
}

// TestAdminUsersStepUpCheckOrder pins the check order of the step-up routes
// with an {id}: the path id first (invalid_id), then the body (invalid_json,
// validation_failed), then the account (not_found), as on PATCH and on every
// other {id} route (5.1).
func TestAdminUsersStepUpCheckOrder(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := e.loginAs(t, model.RoleAdmin)
	routes := []struct {
		method, suffix string
	}{
		{http.MethodDelete, ""},
		{http.MethodPost, "/reset-password"},
		{http.MethodPost, "/reset-2fa"},
	}
	for _, rt := range routes {
		t.Run(rt.method+rt.suffix, func(t *testing.T) {
			call := func(id string, body any) *http.Response {
				return c.doRequest(rt.method, "/api/admin/users/"+id+rt.suffix, body, nil)
			}
			// A bad id wins over a missing, malformed or incomplete body.
			for _, id := range []string{"x", "0", "-3"} {
				expectError(t, call(id, nil), http.StatusBadRequest, codeInvalidID)
				expectError(t, call(id, "{"), http.StatusBadRequest, codeInvalidID)
				expectError(t, call(id, map[string]string{}), http.StatusBadRequest, codeInvalidID)
				expectError(t, call(id, adminUsersStepUp), http.StatusBadRequest, codeInvalidID)
			}
			// A well-formed id: the body is checked before the account is loaded.
			expectError(t, call("999999", nil), http.StatusBadRequest, codeInvalidJSON)
			expectFields(t, call("999999", map[string]string{}), map[string]string{"currentPassword": fieldRequired})
			expectError(t, call("999999", adminUsersStepUp), http.StatusNotFound, codeNotFound)
		})
	}
}

// TestAdminUsersUpdateAuditSplit pins the 5.12 reading of 6.10: a PATCH that
// changes role and active together writes user.updated with the role value
// only, and the activation change as its own user.deactivated event without
// details.
func TestAdminUsersUpdateAuditSplit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := authSignIn(t, e, model.RoleAdmin)
	u := e.seedUser(t, model.RoleCustomer)
	got := decodeStatus[model.AdminUser](t, c.patch(adminUsersPath(u.ID, ""),
		map[string]any{"role": "developer", "active": false, "currentPassword": authPassword}), http.StatusOK)
	if got.Role != model.RoleDeveloper || got.Active {
		t.Fatalf("user %+v", got)
	}
	upd := authExpectAudit(t, e, model.EventUserUpdated, 1)[0]
	assertJSON(t, mustJSON(t, upd.Details), `{"changed":["role"],"role":{"from":"customer","to":"developer"}}`)
	if _, ok := upd.Details[model.DetailActive]; ok {
		t.Errorf("user.updated repeats the activation change: %v", upd.Details)
	}
	deact := authExpectAudit(t, e, model.EventUserDeactivated, 1)[0]
	if len(deact.Details) != 0 {
		t.Errorf("user.deactivated details %v", deact.Details)
	}
	authExpectAudit(t, e, model.EventUserActivated, 0)
}

func TestAdminUsersStepUpRefusalChangesNothing(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := authSignIn(t, e, model.RoleAdmin)
	uc, u := authSignIn(t, e, model.RoleCustomer)
	authEnable2FA(t, e, uc)
	wrong := map[string]string{"currentPassword": "wrong password!"}
	body, err := json.Marshal(wrong)
	if err != nil {
		t.Fatal(err)
	}
	expectError(t, c.doRequest(http.MethodDelete, adminUsersPath(u.ID, ""), body, nil), http.StatusBadRequest, codeCurrentPasswordInvalid)
	expectError(t, c.post(adminUsersPath(u.ID, "/reset-password"), wrong), http.StatusBadRequest, codeCurrentPasswordInvalid)
	expectError(t, c.post(adminUsersPath(u.ID, "/reset-2fa"), wrong), http.StatusBadRequest, codeCurrentPasswordInvalid)
	expectError(t, c.patch(adminUsersPath(u.ID, ""), map[string]any{"role": "developer", "active": false, "currentPassword": "wrong password!"}),
		http.StatusBadRequest, codeCurrentPasswordInvalid)

	au := decodeStatus[model.AdminUser](t, c.get(adminUsersPath(u.ID, "")), http.StatusOK)
	if au.Role != model.RoleCustomer || !au.Active || !au.TOTPEnabled || au.MustChangePassword || au.SessionCount == 0 {
		t.Errorf("user changed by refused actions: %+v", au)
	}
	expectStatus(t, uc.get("/api/auth/me"), http.StatusOK)
	for _, ev := range []string{model.EventUserDeleted, model.EventUserPasswordReset, model.EventUser2FAReset, model.EventUserUpdated, model.EventUserDeactivated} {
		authExpectAudit(t, e, ev, 0)
	}
	// The fifth wrong confirmation locks: even the right one is 429.
	expectError(t, c.post(adminUsersPath(u.ID, "/reset-2fa"), wrong), http.StatusBadRequest, codeCurrentPasswordInvalid)
	expectError(t, c.post(adminUsersPath(u.ID, "/reset-2fa"), adminUsersStepUp), http.StatusTooManyRequests, codeTooManyAttempts)
	if au := decodeStatus[model.AdminUser](t, c.get(adminUsersPath(u.ID, "")), http.StatusOK); !au.TOTPEnabled {
		t.Error("2FA reset despite the lock")
	}
}
