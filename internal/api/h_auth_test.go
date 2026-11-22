package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/storetest"
)

// ---------------------------------------------------------------------------
// Shared helpers of the account tests (h_auth, h_twofa, h_me, h_privacy,
// h_admin_*). Every name starts with "account" or "auth" to stay clear of
// the link tests in the same package.

// authPassword is the password of every account seeded by authSeed.
const authPassword = "Correct-Horse-2026"

// authNewPassword satisfies the policy and differs from authPassword.
const authNewPassword = "Battery-Staple-9137"

// authSeq makes seeded usernames unique.
var authSeq atomic.Int64

// authUsername returns a fresh valid username.
func authUsername(prefix string) string {
	return fmt.Sprintf("%s%d", prefix, authSeq.Add(1))
}

// authSeed creates an active account of role with authPassword that has
// acknowledged the current privacy notice.
func authSeed(t *testing.T, e *testEnv, role model.Role) model.User {
	t.Helper()
	return e.seedPasswordUser(t, authUsername("acct"), role, authPassword)
}

// authLogin posts the first sign-in step.
func authLogin(c *apiClient, username, password string) *http.Response {
	c.t.Helper()
	return c.post("/api/auth/login", map[string]string{"username": username, "password": password})
}

// authSignIn seeds an account and signs it in through the login handler.
func authSignIn(t *testing.T, e *testEnv, role model.Role) (*apiClient, model.User) {
	t.Helper()
	u := authSeed(t, e, role)
	c := e.client(t)
	res := decodeStatus[loginOKResponse](t, authLogin(c, u.Username, authPassword), http.StatusOK)
	if res.Status != loginStatusOK {
		t.Fatalf("login status %q", res.Status)
	}
	return c, u
}

// authCode returns the TOTP code of secret at the env's current time.
func authCode(t *testing.T, e *testEnv, secret string) string {
	t.Helper()
	code, err := auth.TOTPCode(secret, e.clock.Now())
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	return code
}

// authNextCode moves the clock to the next TOTP step and returns its code,
// so that the code is newer than the last accepted one.
func authNextCode(t *testing.T, e *testEnv, secret string) string {
	t.Helper()
	e.clock.Advance(time.Duration(auth.TOTPPeriod) * time.Second)
	return authCode(t, e, secret)
}

// authEnable2FA enables 2FA for the signed-in client through the handlers
// and returns the secret and the recovery codes.
func authEnable2FA(t *testing.T, e *testEnv, c *apiClient) (string, []string) {
	t.Helper()
	setup := decodeStatus[twoFactorSetupResponse](t, c.post("/api/me/2fa/setup", map[string]string{"password": authPassword}), http.StatusOK)
	rc := decodeStatus[recoveryCodesResponse](t, c.post("/api/me/2fa/enable", map[string]string{"code": authCode(t, e, setup.Secret)}), http.StatusOK)
	return setup.Secret, rc.RecoveryCodes
}

// authChallenge signs in a 2FA account up to the challenge.
func authChallenge(t *testing.T, c *apiClient, username string) loginChallengeResponse {
	t.Helper()
	ch := decodeStatus[loginChallengeResponse](t, authLogin(c, username, authPassword), http.StatusOK)
	if ch.Status != loginStatusTOTPRequired || ch.Challenge == "" {
		t.Fatalf("challenge response %+v", ch)
	}
	return ch
}

// authTOTP posts the second sign-in step with a TOTP code.
func authTOTP(c *apiClient, challenge, code string) *http.Response {
	c.t.Helper()
	return c.post("/api/auth/login/totp", map[string]string{"challenge": challenge, "code": code})
}

// authDB opens a second connection to the env's database for inspection.
func authDB(t *testing.T, e *testEnv) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+e.dbPath())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// authTableText returns every value of every row of table as text, one row
// per line, for data minimisation checks.
func authTableText(t *testing.T, e *testEnv, table string) string {
	t.Helper()
	rows, err := authDB(t, e).Query("SELECT * FROM " + table)
	if err != nil {
		t.Fatalf("query %s: %v", table, err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		for _, v := range vals {
			if bs, ok := v.([]byte); ok {
				v = string(bs)
			}
			fmt.Fprintf(&b, "%v|", v)
		}
		b.WriteString("\n")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// authThrottleRow is a login_throttle row read from the database.
type authThrottleRow struct {
	Failures int
	Lockouts int
	Locked   bool
	UserID   sql.NullInt64
}

// authThrottle returns the login_throttle row of key, and whether it exists.
func authThrottle(t *testing.T, e *testEnv, key string) (authThrottleRow, bool) {
	t.Helper()
	var row authThrottleRow
	var locked sql.NullString
	err := authDB(t, e).QueryRow(`SELECT failures, lockouts, locked_until, user_id FROM login_throttle WHERE key = ?`, key).
		Scan(&row.Failures, &row.Lockouts, &locked, &row.UserID)
	if err == sql.ErrNoRows {
		return row, false
	}
	if err != nil {
		t.Fatalf("throttle row: %v", err)
	}
	row.Locked = locked.Valid
	return row, true
}

// authUserKey is the "u:" throttle key of a username in the test keyring.
func authUserKey(t *testing.T, username string) string {
	t.Helper()
	return auth.UserThrottleKey(newTestKeyring(t), model.UsernameKey(username))
}

// authLoopbackIPKey is the throttle key of the test client's address.
var authLoopbackIPKey = auth.IPThrottleKey(netip.MustParseAddr("127.0.0.1"))

// authAudit returns the audit entries of event (newest first).
func authAudit(t *testing.T, e *testEnv, event string) []model.AuditEntry {
	t.Helper()
	items, _, err := e.store.Audit(e.ctx(), model.AuditFilter{Event: event}, model.PageReq{Limit: 200})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	return items
}

// authExpectAudit fails unless exactly n entries of event exist and returns them.
func authExpectAudit(t *testing.T, e *testEnv, event string, n int) []model.AuditEntry {
	t.Helper()
	items := authAudit(t, e, event)
	if len(items) != n {
		t.Fatalf("audit %s: %d entries, want %d: %+v", event, len(items), n, items)
	}
	return items
}

// ---------------------------------------------------------------------------
// Fault injection

// accountFaultStore wraps a Store and fails one call of one method: the
// nth call (1-based, counted from arm) of the armed method returns err.
type accountFaultStore struct {
	Store
	mu     sync.Mutex
	method string
	nth    int
	calls  int
	err    error
}

// arm makes the nth next call of method fail with err (errInjected when nil).
func (f *accountFaultStore) arm(method string, nth int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err == nil {
		err = errInjected
	}
	f.method, f.nth, f.calls, f.err = method, nth, 0, err
}

// hit counts a call of method and reports the error to return, if any.
func (f *accountFaultStore) hit(method string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if method != f.method {
		return nil
	}
	f.calls++
	if f.calls == f.nth {
		return f.err
	}
	return nil
}

func (f *accountFaultStore) Settings(ctx context.Context) (model.SystemSettings, error) {
	if err := f.hit("Settings"); err != nil {
		return model.SystemSettings{}, err
	}
	return f.Store.Settings(ctx)
}

func (f *accountFaultStore) UpdateSettings(ctx context.Context, p model.SettingsPatch, by int64) (model.SystemSettings, error) {
	if err := f.hit("UpdateSettings"); err != nil {
		return model.SystemSettings{}, err
	}
	return f.Store.UpdateSettings(ctx, p, by)
}

func (f *accountFaultStore) BumpPrivacyNotice(ctx context.Context, by int64) (model.SystemSettings, error) {
	if err := f.hit("BumpPrivacyNotice"); err != nil {
		return model.SystemSettings{}, err
	}
	return f.Store.BumpPrivacyNotice(ctx, by)
}

func (f *accountFaultStore) CreateUser(ctx context.Context, u model.NewUser) (model.User, error) {
	if err := f.hit("CreateUser"); err != nil {
		return model.User{}, err
	}
	return f.Store.CreateUser(ctx, u)
}

func (f *accountFaultStore) UserByID(ctx context.Context, id int64) (model.User, error) {
	if err := f.hit("UserByID"); err != nil {
		return model.User{}, err
	}
	return f.Store.UserByID(ctx, id)
}

func (f *accountFaultStore) Credentials(ctx context.Context, key string) (model.Credentials, error) {
	if err := f.hit("Credentials"); err != nil {
		return model.Credentials{}, err
	}
	return f.Store.Credentials(ctx, key)
}

func (f *accountFaultStore) AdminUsers(ctx context.Context, fl model.UserFilter, p model.PageReq) ([]model.AdminUser, int, error) {
	if err := f.hit("AdminUsers"); err != nil {
		return nil, 0, err
	}
	return f.Store.AdminUsers(ctx, fl, p)
}

func (f *accountFaultStore) AdminUser(ctx context.Context, id int64) (model.AdminUser, error) {
	if err := f.hit("AdminUser"); err != nil {
		return model.AdminUser{}, err
	}
	return f.Store.AdminUser(ctx, id)
}

func (f *accountFaultStore) UpdateUser(ctx context.Context, id int64, p model.UserPatch) (model.User, error) {
	if err := f.hit("UpdateUser"); err != nil {
		return model.User{}, err
	}
	return f.Store.UpdateUser(ctx, id, p)
}

func (f *accountFaultStore) SetPassword(ctx context.Context, id int64, hash string, must bool) error {
	if err := f.hit("SetPassword"); err != nil {
		return err
	}
	return f.Store.SetPassword(ctx, id, hash, must)
}

func (f *accountFaultStore) RecordLogin(ctx context.Context, id int64) error {
	if err := f.hit("RecordLogin"); err != nil {
		return err
	}
	return f.Store.RecordLogin(ctx, id)
}

func (f *accountFaultStore) Directory(ctx context.Context, exclude int64, q string, limit int) ([]model.DirectoryUser, error) {
	if err := f.hit("Directory"); err != nil {
		return nil, err
	}
	return f.Store.Directory(ctx, exclude, q, limit)
}

func (f *accountFaultStore) DeleteUser(ctx context.Context, id int64, e model.AuditEntry) error {
	if err := f.hit("DeleteUser"); err != nil {
		return err
	}
	return f.Store.DeleteUser(ctx, id, e)
}

func (f *accountFaultStore) OwnSharedLinkCount(ctx context.Context, id int64) (int, error) {
	if err := f.hit("OwnSharedLinkCount"); err != nil {
		return 0, err
	}
	return f.Store.OwnSharedLinkCount(ctx, id)
}

func (f *accountFaultStore) SetPendingTOTP(ctx context.Context, id int64, sealed string, exp time.Time) error {
	if err := f.hit("SetPendingTOTP"); err != nil {
		return err
	}
	return f.Store.SetPendingTOTP(ctx, id, sealed, exp)
}

func (f *accountFaultStore) PendingTOTP(ctx context.Context, id int64) (string, time.Time, error) {
	if err := f.hit("PendingTOTP"); err != nil {
		return "", time.Time{}, err
	}
	return f.Store.PendingTOTP(ctx, id)
}

func (f *accountFaultStore) EnableTOTP(ctx context.Context, id int64, sealed string, step int64, hashes []string) error {
	if err := f.hit("EnableTOTP"); err != nil {
		return err
	}
	return f.Store.EnableTOTP(ctx, id, sealed, step, hashes)
}

func (f *accountFaultStore) DisableTOTP(ctx context.Context, id int64) error {
	if err := f.hit("DisableTOTP"); err != nil {
		return err
	}
	return f.Store.DisableTOTP(ctx, id)
}

func (f *accountFaultStore) AdvanceTOTPStep(ctx context.Context, id, step int64) (bool, error) {
	if err := f.hit("AdvanceTOTPStep"); err != nil {
		return false, err
	}
	return f.Store.AdvanceTOTPStep(ctx, id, step)
}

func (f *accountFaultStore) ReplaceRecoveryCodes(ctx context.Context, id int64, hashes []string) error {
	if err := f.hit("ReplaceRecoveryCodes"); err != nil {
		return err
	}
	return f.Store.ReplaceRecoveryCodes(ctx, id, hashes)
}

func (f *accountFaultStore) UseRecoveryCode(ctx context.Context, id int64, hash string) (bool, error) {
	if err := f.hit("UseRecoveryCode"); err != nil {
		return false, err
	}
	return f.Store.UseRecoveryCode(ctx, id, hash)
}

func (f *accountFaultStore) RecoveryCodesRemaining(ctx context.Context, id int64) (int, error) {
	if err := f.hit("RecoveryCodesRemaining"); err != nil {
		return 0, err
	}
	return f.Store.RecoveryCodesRemaining(ctx, id)
}

func (f *accountFaultStore) CreateSession(ctx context.Context, n model.NewSession) (model.Session, error) {
	if err := f.hit("CreateSession"); err != nil {
		return model.Session{}, err
	}
	return f.Store.CreateSession(ctx, n)
}

func (f *accountFaultStore) SessionByToken(ctx context.Context, hash []byte) (model.Session, model.User, error) {
	if err := f.hit("SessionByToken"); err != nil {
		return model.Session{}, model.User{}, err
	}
	return f.Store.SessionByToken(ctx, hash)
}

func (f *accountFaultStore) Sessions(ctx context.Context, id int64) ([]model.Session, error) {
	if err := f.hit("Sessions"); err != nil {
		return nil, err
	}
	return f.Store.Sessions(ctx, id)
}

func (f *accountFaultStore) DeleteSession(ctx context.Context, userID, id int64) error {
	if err := f.hit("DeleteSession"); err != nil {
		return err
	}
	return f.Store.DeleteSession(ctx, userID, id)
}

func (f *accountFaultStore) DeleteSessionByToken(ctx context.Context, hash []byte) error {
	if err := f.hit("DeleteSessionByToken"); err != nil {
		return err
	}
	return f.Store.DeleteSessionByToken(ctx, hash)
}

func (f *accountFaultStore) DeleteUserSessions(ctx context.Context, userID, except int64) (int, error) {
	if err := f.hit("DeleteUserSessions"); err != nil {
		return 0, err
	}
	return f.Store.DeleteUserSessions(ctx, userID, except)
}

func (f *accountFaultStore) CreateChallenge(ctx context.Context, hash []byte, userID int64, ip string, exp time.Time) error {
	if err := f.hit("CreateChallenge"); err != nil {
		return err
	}
	return f.Store.CreateChallenge(ctx, hash, userID, ip, exp)
}

func (f *accountFaultStore) ChallengeAttempt(ctx context.Context, hash []byte, max int) (model.Challenge, error) {
	if err := f.hit("ChallengeAttempt"); err != nil {
		return model.Challenge{}, err
	}
	return f.Store.ChallengeAttempt(ctx, hash, max)
}

func (f *accountFaultStore) DeleteChallenge(ctx context.Context, hash []byte) error {
	if err := f.hit("DeleteChallenge"); err != nil {
		return err
	}
	return f.Store.DeleteChallenge(ctx, hash)
}

func (f *accountFaultStore) ThrottleUpdate(ctx context.Context, keys []model.ThrottleKey,
	fn func(row model.ThrottleRow) (model.ThrottleRow, error)) error {
	if err := f.hit("ThrottleUpdate"); err != nil {
		return err
	}
	return f.Store.ThrottleUpdate(ctx, keys, fn)
}

func (f *accountFaultStore) ClearThrottle(ctx context.Context, key string) error {
	if err := f.hit("ClearThrottle"); err != nil {
		return err
	}
	return f.Store.ClearThrottle(ctx, key)
}

func (f *accountFaultStore) ClearUserThrottle(ctx context.Context, id int64) error {
	if err := f.hit("ClearUserThrottle"); err != nil {
		return err
	}
	return f.Store.ClearUserThrottle(ctx, id)
}

func (f *accountFaultStore) Preferences(ctx context.Context, id int64) (model.Preferences, error) {
	if err := f.hit("Preferences"); err != nil {
		return model.Preferences{}, err
	}
	return f.Store.Preferences(ctx, id)
}

func (f *accountFaultStore) UpdatePreferences(ctx context.Context, id int64, p model.PreferencesPatch) (model.Preferences, error) {
	if err := f.hit("UpdatePreferences"); err != nil {
		return model.Preferences{}, err
	}
	return f.Store.UpdatePreferences(ctx, id, p)
}

func (f *accountFaultStore) AppendAudit(ctx context.Context, e model.AuditEntry) error {
	if err := f.hit("AppendAudit"); err != nil {
		return err
	}
	return f.Store.AppendAudit(ctx, e)
}

func (f *accountFaultStore) Audit(ctx context.Context, fl model.AuditFilter, p model.PageReq) ([]model.AuditEntry, int, error) {
	if err := f.hit("Audit"); err != nil {
		return nil, 0, err
	}
	return f.Store.Audit(ctx, fl, p)
}

func (f *accountFaultStore) RecordConsent(ctx context.Context, id int64, kind string, version int) error {
	if err := f.hit("RecordConsent"); err != nil {
		return err
	}
	return f.Store.RecordConsent(ctx, id, kind, version)
}

func (f *accountFaultStore) Consents(ctx context.Context, id int64) ([]model.Consent, error) {
	if err := f.hit("Consents"); err != nil {
		return nil, err
	}
	return f.Store.Consents(ctx, id)
}

func (f *accountFaultStore) ExportUser(ctx context.Context, id int64) (model.UserExport, error) {
	if err := f.hit("ExportUser"); err != nil {
		return model.UserExport{}, err
	}
	return f.Store.ExportUser(ctx, id)
}

// accountFaultEnv is an env whose store is wrapped by an accountFaultStore.
func accountFaultEnv(t *testing.T, opts ...envOption) (*testEnv, *accountFaultStore) {
	t.Helper()
	fs := &accountFaultStore{}
	opts = append(opts, withDeps(func(d *Deps) {
		fs.Store = d.Store
		d.Store = fs
	}))
	return newEnv(t, opts...), fs
}

// accountFault is one fault injection case: the nth call of method fails
// during the request, which must then answer status (500 unless set).
// prepare runs before the fault is armed (sign-in, seeding) and returns
// the request; nth counts only the calls of that request.
type accountFault struct {
	name    string
	method  string
	nth     int
	err     error
	status  int
	code    string
	prepare func() func() *http.Response
}

// runAccountFaults prepares each case, arms the fault, sends the request
// and checks the answer.
func runAccountFaults(t *testing.T, fs *accountFaultStore, cases []accountFault) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			call := c.prepare()
			fs.arm(c.method, c.nth, c.err)
			res := call()
			fs.arm("", 0, nil)
			status, code := c.status, c.code
			if status == 0 {
				status, code = http.StatusInternalServerError, codeInternalError
			}
			expectError(t, res, status, code)
		})
	}
}

// ---------------------------------------------------------------------------
// Sign-in (9.2.3 group 3)

func TestLoginSuccess(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	u := authSeed(t, e, model.RoleDeveloper)
	c := e.client(t)

	// The username is matched case-insensitively.
	res := authLogin(c, strings.ToUpper(u.Username), authPassword)
	body := decodeStatus[loginOKResponse](t, res, http.StatusOK)
	if body.Status != loginStatusOK || body.User.ID != u.ID || body.User.Username != u.Username {
		t.Fatalf("login body %+v", body)
	}
	me := body.User
	if me.Role != model.RoleDeveloper || !me.Permissions.CanShare || !me.Permissions.CanListUsers || me.Permissions.CanAdmin {
		t.Errorf("permissions %+v", me.Permissions)
	}
	if me.SessionID == 0 || me.LastLoginAt == nil || !me.LastLoginAt.Equal(e.clock.Now()) || me.PrivacyConsentRequired {
		t.Errorf("me %+v", me)
	}
	if me.Preferences.Locale != model.LocaleHU || me.PrivacyNoticeVersion != 1 || me.TOTPEnabled || me.RecoveryCodesRemaining != 0 {
		t.Errorf("me %+v", me)
	}
	ck := responseCookie(res, e.cookieName())
	if ck == nil || !ck.HttpOnly || ck.SameSite != http.SameSiteStrictMode || ck.Path != "/" || ck.Secure {
		t.Fatalf("session cookie %+v", ck)
	}
	wantKeys := []string{"createdAt", "displayName", "id", "lastLoginAt", "mustChangePassword", "permissions",
		"preferences", "privacyConsentRequired", "privacyNoticeVersion", "recoveryCodesRemaining", "role",
		"sessionId", "totpEnabled", "username"}
	var raw map[string]json.RawMessage
	decodeInto(t, readBody(t, res), &raw)
	if got := jsonKeys(t, raw["user"]); strings.Join(got, ",") != strings.Join(wantKeys, ",") {
		t.Errorf("Me keys %v, want %v", got, wantKeys)
	}

	got := decodeStatus[meResponse](t, c.get("/api/auth/me"), http.StatusOK)
	if got.SessionID != me.SessionID || got.ID != u.ID {
		t.Errorf("GET me %+v", got)
	}
	ok := authExpectAudit(t, e, model.EventLoginSuccess, 1)[0]
	if ok.Actor.ID == nil || *ok.Actor.ID != u.ID || ok.Details[model.DetailMethod] != model.MethodPassword || ok.IP != "127.0.0.1" {
		t.Errorf("success audit %+v", ok)
	}
	// The user key is cleared after a full sign-in.
	if _, exists := authThrottle(t, e, authUserKey(t, u.Username)); exists {
		t.Error("user throttle row kept after success")
	}
}

func TestLoginValidation(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t)
	expectError(t, c.post("/api/auth/login", "{"), http.StatusBadRequest, codeInvalidJSON)
	expectError(t, c.post("/api/auth/login", `{"username":"a","password":"b","extra":1}`), http.StatusBadRequest, codeInvalidJSON)
	expectFields(t, c.post("/api/auth/login", map[string]string{}), map[string]string{"username": fieldRequired, "password": fieldRequired})
	expectFields(t, authLogin(c, "   ", "x"), map[string]string{"username": fieldRequired})
	expectFields(t, authLogin(c, strings.Repeat("a", 65), "x"), map[string]string{"username": fieldTooLong})
	expectFields(t, authLogin(c, "bad‮name", "x"), map[string]string{"username": fieldInvalid})
	expectFields(t, authLogin(c, "someone", strings.Repeat("é", 1025)), map[string]string{"password": fieldTooLong})
	// Validation failures do not touch the throttle.
	if _, exists := authThrottle(t, e, authLoopbackIPKey); exists {
		t.Error("validation failure reserved an attempt")
	}
}

func TestLoginFailures(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	u := authSeed(t, e, model.RoleCustomer)
	c := e.client(t)

	// Wrong password: the actor comes from the account record, not the input.
	expectError(t, authLogin(c, strings.ToUpper(u.Username), "wrong password!"), http.StatusUnauthorized, codeInvalidCredentials)
	f := authExpectAudit(t, e, model.EventLoginFailure, 1)[0]
	if f.Actor.ID == nil || *f.Actor.ID != u.ID || f.Actor.Username != u.Username || f.Details[model.DetailReason] != model.ReasonCredentials || f.Success {
		t.Errorf("credentials failure audit %+v", f)
	}
	if row, _ := authThrottle(t, e, authUserKey(t, u.Username)); row.Failures != 1 || !row.UserID.Valid || row.UserID.Int64 != u.ID {
		t.Errorf("user throttle row %+v", row)
	}
	if row, _ := authThrottle(t, e, authLoopbackIPKey); row.Failures != 1 {
		t.Errorf("ip throttle row %+v", row)
	}

	// Unknown username: no actor, no name.
	expectError(t, authLogin(c, "nobody-here", authPassword), http.StatusUnauthorized, codeInvalidCredentials)
	f = authExpectAudit(t, e, model.EventLoginFailure, 2)[0]
	if f.Actor.ID != nil || f.Actor.Username != "" || f.Details[model.DetailReason] != model.ReasonUnknownUser {
		t.Errorf("unknown user audit %+v", f)
	}
	if row, _ := authThrottle(t, e, authUserKey(t, "nobody-here")); row.Failures != 1 || row.UserID.Valid {
		t.Errorf("unknown user throttle row %+v", row)
	}

	// Inactive account: correct password is 403 and refunds both keys.
	inactive := false
	if _, err := e.store.UpdateUser(e.ctx(), u.ID, model.UserPatch{Active: &inactive}); err != nil {
		t.Fatal(err)
	}
	expectError(t, authLogin(c, u.Username, authPassword), http.StatusForbidden, codeAccountDisabled)
	f = authExpectAudit(t, e, model.EventLoginFailure, 3)[0]
	if f.Details[model.DetailReason] != model.ReasonDisabled || f.Actor.Username != u.Username {
		t.Errorf("disabled audit %+v", f)
	}
	if row, _ := authThrottle(t, e, authUserKey(t, u.Username)); row.Failures != 1 {
		t.Errorf("disabled attempt was not refunded: %+v", row)
	}
	// Inactive account with a wrong password is an ordinary failure.
	expectError(t, authLogin(c, u.Username, "wrong password!"), http.StatusUnauthorized, codeInvalidCredentials)
	if responseCookie(authLogin(c, u.Username, authPassword), e.cookieName()) != nil {
		t.Error("disabled account got a session cookie")
	}
}

func TestLoginUserThrottle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	u := authSeed(t, e, model.RoleCustomer)
	c := e.client(t)
	for i := 0; i < 5; i++ {
		expectError(t, authLogin(c, u.Username, "wrong password!"), http.StatusUnauthorized, codeInvalidCredentials)
	}
	// Locked: even the correct password is refused, with Retry-After.
	res := authLogin(c, u.Username, authPassword)
	body := expectError(t, res, http.StatusTooManyRequests, codeTooManyAttempts)
	if res.Header.Get("Retry-After") != "900" || body.Details[detailRetryAfterSeconds] != float64(900) {
		t.Errorf("Retry-After %q, details %v", res.Header.Get("Retry-After"), body.Details)
	}
	expectError(t, authLogin(c, u.Username, authPassword), http.StatusTooManyRequests, codeTooManyAttempts)
	locked := authExpectAudit(t, e, model.EventLoginLocked, 1)[0]
	if locked.Details[model.DetailScope] != model.ScopeUser || locked.Actor.Username != u.Username {
		t.Errorf("locked audit %+v", locked)
	}
	au, err := e.store.AdminUser(e.ctx(), u.ID)
	if err != nil || au.LockedUntil == nil {
		t.Fatalf("admin user lock %+v %v", au, err)
	}
	// After the lock the account signs in again.
	e.clock.Advance(15*time.Minute + time.Second)
	decodeStatus[loginOKResponse](t, authLogin(c, u.Username, authPassword), http.StatusOK)

	// The same lock applies to an unknown username, so the answer does not
	// reveal whether the account exists.
	for i := 0; i < 5; i++ {
		expectError(t, authLogin(c, "ghost-user", "wrong password!"), http.StatusUnauthorized, codeInvalidCredentials)
	}
	expectError(t, authLogin(c, "ghost-user", "wrong password!"), http.StatusTooManyRequests, codeTooManyAttempts)
	ghost := authExpectAudit(t, e, model.EventLoginLocked, 2)[0]
	if ghost.Actor.ID != nil || ghost.Actor.Username != "" {
		t.Errorf("unknown user locked audit %+v", ghost)
	}
}

func TestLoginIPThrottle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t)
	// 6 unknown names x 5 failures = 30 failures from one address.
	for n := 0; n < 6; n++ {
		for i := 0; i < 5; i++ {
			expectError(t, authLogin(c, fmt.Sprintf("ghost%d", n), "wrong password!"), http.StatusUnauthorized, codeInvalidCredentials)
		}
	}
	u := authSeed(t, e, model.RoleAdmin)
	expectError(t, authLogin(c, u.Username, authPassword), http.StatusTooManyRequests, codeTooManyAttempts)
	locked := authExpectAudit(t, e, model.EventLoginLocked, 1)[0]
	if locked.Details[model.DetailScope] != model.ScopeIP {
		t.Errorf("ip lock audit %+v", locked)
	}
	if row, _ := authThrottle(t, e, authLoopbackIPKey); !row.Locked {
		t.Errorf("ip row %+v", row)
	}
}

func TestLoginConcurrentFailures(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	u := authSeed(t, e, model.RoleCustomer)
	const n = 20
	statuses := make(chan int, n)
	var wg sync.WaitGroup
	body := fmt.Sprintf(`{"username":%q,"password":"wrong password!"}`, u.Username)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPost, e.url("/api/auth/login"), strings.NewReader(body))
			if err != nil {
				statuses <- 0
				return
			}
			req.Header.Set(headerKinexRequest, "1")
			req.Header.Set("Content-Type", "application/json")
			res, err := e.srv.Client().Do(req)
			if err != nil {
				statuses <- 0
				return
			}
			_ = res.Body.Close()
			statuses <- res.StatusCode
		}()
	}
	wg.Wait()
	close(statuses)
	var unauthorized, throttled int
	for s := range statuses {
		switch s {
		case http.StatusUnauthorized:
			unauthorized++
		case http.StatusTooManyRequests:
			throttled++
		default:
			t.Errorf("unexpected status %d", s)
		}
	}
	if unauthorized > 5 || unauthorized+throttled != n {
		t.Errorf("401: %d, 429: %d; want at most 5 verifications", unauthorized, throttled)
	}
	// Confirm locks at the threshold and starts a new window (6.7), so the
	// row holds the lock instead of the five counted failures.
	row, _ := authThrottle(t, e, authUserKey(t, u.Username))
	if !row.Locked || row.Lockouts != 1 {
		t.Errorf("throttle row after the burst %+v", row)
	}
}

func TestLoginDataMinimisation(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t)
	const typed = "Tavasz2026!jelszo"
	expectError(t, authLogin(c, typed, "anything at all"), http.StatusUnauthorized, codeInvalidCredentials)
	for i := 0; i < 5; i++ {
		authLogin(c, typed, "anything at all")
	}
	for _, table := range []string{"audit_log", "login_throttle"} {
		text := authTableText(t, e, table)
		if strings.Contains(strings.ToLower(text), strings.ToLower(typed)) {
			t.Errorf("%s contains the typed name:\n%s", table, text)
		}
		if text == "" {
			t.Errorf("%s is empty", table)
		}
	}
}

func TestLoginMappedIPv6Client(t *testing.T) {
	t.Parallel()
	e := newEnv(t, withTrustedProxy("127.0.0.1"))
	c := e.client(t).withHeader("X-Forwarded-For", "::ffff:203.0.113.5")
	expectError(t, authLogin(c, "ghost-user", "wrong password!"), http.StatusUnauthorized, codeInvalidCredentials)
	v4 := auth.IPThrottleKey(netip.MustParseAddr("203.0.113.5"))
	if row, ok := authThrottle(t, e, v4); !ok || row.Failures != 1 {
		t.Errorf("no /32 key %q: %+v", v4, row)
	}
	if _, ok := authThrottle(t, e, auth.IPThrottleKey(netip.MustParseAddr("::"))); ok {
		t.Error("the mapped client shares the ::/64 key")
	}
	if f := authExpectAudit(t, e, model.EventLoginFailure, 1)[0]; f.IP != "203.0.113.5" {
		t.Errorf("audit ip %q", f.IP)
	}
}

func TestLoginRehashesOldPassword(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	old, err := auth.NewHasher(500, 1).Hash(authPassword)
	if err != nil {
		t.Fatal(err)
	}
	u := storetest.SeedNewUser(t, e.store, model.NewUser{Username: authUsername("old"), Role: model.RoleCustomer, PasswordHash: old})
	decodeStatus[loginOKResponse](t, authLogin(e.client(t), u.Username, authPassword), http.StatusOK)
	cred, err := e.store.Credentials(e.ctx(), u.UsernameKey)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cred.PasswordHash, "$1000$") {
		t.Errorf("hash not upgraded: %s", cred.PasswordHash)
	}
	if cred.User.MustChangePassword {
		t.Error("rehash changed the must-change flag")
	}
	// A failing rehash write is logged and the sign-in still succeeds.
	e2, fs := accountFaultEnv(t)
	u2 := storetest.SeedNewUser(t, e2.store, model.NewUser{Username: authUsername("old"), Role: model.RoleCustomer, PasswordHash: old})
	fs.arm("SetPassword", 1, nil)
	decodeStatus[loginOKResponse](t, authLogin(e2.client(t), u2.Username, authPassword), http.StatusOK)
	if e2.logs.count("password rehash failed") != 1 {
		t.Errorf("rehash failure not logged:\n%s", e2.logs)
	}
}

func TestLoginRotatesSession(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	first := c.sessionToken()
	decodeStatus[loginOKResponse](t, authLogin(c, u.Username, authPassword), http.StatusOK)
	second := c.sessionToken()
	if first == "" || second == "" || first == second {
		t.Fatalf("tokens %q %q", first, second)
	}
	stale := e.client(t)
	stale.setSessionToken(first)
	expectError(t, stale.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	sessions, err := e.store.Sessions(e.ctx(), u.ID)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("sessions after rotation: %d %v", len(sessions), err)
	}
}

func TestLogout(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	token := c.sessionToken()
	res := c.post("/api/auth/logout", nil)
	expectStatus(t, res, http.StatusNoContent)
	if ck := responseCookie(res, e.cookieName()); ck == nil || ck.MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", ck)
	}
	if list, _ := e.store.Sessions(e.ctx(), u.ID); len(list) != 0 {
		t.Errorf("session row kept: %+v", list)
	}
	// The copied token is dead.
	copied := e.client(t)
	copied.setSessionToken(token)
	expectError(t, copied.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	out := authExpectAudit(t, e, model.EventLogout, 1)[0]
	if out.Actor.ID == nil || *out.Actor.ID != u.ID {
		t.Errorf("logout audit %+v", out)
	}

	// Without a cookie, with an unknown token and with an expired session:
	// 204, no audit.
	expectStatus(t, e.client(t).post("/api/auth/logout", nil), http.StatusNoContent)
	expectStatus(t, copied.post("/api/auth/logout", nil), http.StatusNoContent)
	expired := e.client(t)
	tok, _ := e.createSession(t, u.ID)
	expired.setSessionToken(tok)
	e.clock.Advance(13 * time.Hour)
	expectStatus(t, expired.post("/api/auth/logout", nil), http.StatusNoContent)
	authExpectAudit(t, e, model.EventLogout, 1)
	if list, _ := e.store.Sessions(e.ctx(), u.ID); len(list) != 0 {
		t.Errorf("expired session row kept: %+v", list)
	}
}

func TestSessionExpiryOverHTTP(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := authSignIn(t, e, model.RoleCustomer)
	// Idle expiry (720 minutes by default).
	e.clock.Advance(719 * time.Minute)
	expectStatus(t, c.get("/api/auth/me"), http.StatusOK)
	e.clock.Advance(720 * time.Minute)
	expectError(t, c.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)

	// Absolute expiry under a lowered setting.
	hours := 1
	if _, err := e.store.UpdateSettings(e.ctx(), model.SettingsPatch{SessionAbsoluteHours: &hours}, 0); err != nil {
		t.Fatal(err)
	}
	c2, _ := authSignIn(t, e, model.RoleCustomer)
	e.clock.Advance(59 * time.Minute)
	expectStatus(t, c2.get("/api/auth/me"), http.StatusOK)
	e.clock.Advance(2 * time.Minute)
	expectError(t, c2.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
}

// ---------------------------------------------------------------------------
// Second sign-in step

func TestTwoFactorLogin(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleDeveloper)
	secret, codes := authEnable2FA(t, e, c)
	if len(codes) != recoveryCodeCount {
		t.Fatalf("recovery codes %v", codes)
	}

	// The password step answers a challenge and no session.
	anon := e.client(t)
	res := authLogin(anon, u.Username, authPassword)
	ch := decodeStatus[loginChallengeResponse](t, res, http.StatusOK)
	if ch.Status != loginStatusTOTPRequired || len(ch.Challenge) != 43 || !ch.ExpiresAt.Equal(e.clock.Now().Add(challengeTTL)) {
		t.Fatalf("challenge %+v", ch)
	}
	if responseCookie(res, e.cookieName()) != nil {
		t.Fatal("password step set a session cookie")
	}
	if keys := jsonKeys(t, readBody(t, res)); strings.Join(keys, ",") != "challenge,expiresAt,status" {
		t.Errorf("challenge keys %v", keys)
	}
	// With 2FA on, the password step keeps its reservation on the user key.
	if row, _ := authThrottle(t, e, authUserKey(t, u.Username)); row.Failures != 1 {
		t.Errorf("user key after the password step %+v", row)
	}

	code := authNextCode(t, e, secret)
	ok := decodeStatus[loginOKResponse](t, authTOTP(anon, ch.Challenge, code), http.StatusOK)
	if !ok.User.TOTPEnabled || ok.User.RecoveryCodesRemaining != 10 || anon.sessionToken() == "" {
		t.Fatalf("totp login %+v", ok.User)
	}
	if _, exists := authThrottle(t, e, authUserKey(t, u.Username)); exists {
		t.Error("user key kept after the second factor")
	}
	if s := authAudit(t, e, model.EventLoginSuccess)[0]; s.Details[model.DetailMethod] != model.MethodTOTP {
		t.Errorf("success audit %+v", s)
	}
	// The challenge is single use.
	expectError(t, authTOTP(anon, ch.Challenge, code), http.StatusUnauthorized, codeChallengeInvalid)

	// A replayed code (same step) is refused.
	again := e.client(t)
	ch2 := authChallenge(t, again, u.Username)
	expectError(t, authTOTP(again, ch2.Challenge, code), http.StatusUnauthorized, codeInvalidTOTP)
	f := authAudit(t, e, model.EventLoginFailure)[0]
	if f.Details[model.DetailReason] != model.ReasonTOTP || f.Actor.Username != u.Username {
		t.Errorf("totp failure audit %+v", f)
	}

	// A recovery code works once.
	rec := map[string]string{"challenge": ch2.Challenge, "recoveryCode": strings.ToUpper(codes[0])}
	got := decodeStatus[loginOKResponse](t, again.post("/api/auth/login/totp", rec), http.StatusOK)
	if got.User.RecoveryCodesRemaining != 9 {
		t.Errorf("remaining %d", got.User.RecoveryCodesRemaining)
	}
	used := authExpectAudit(t, e, model.EventRecoveryCodeUsed, 1)[0]
	if used.Details[model.DetailRemaining] != float64(9) {
		t.Errorf("recovery audit %+v", used)
	}
	if s := authAudit(t, e, model.EventLoginSuccess)[0]; s.Details[model.DetailMethod] != model.MethodRecoveryCode {
		t.Errorf("success audit %+v", s)
	}
	third := e.client(t)
	ch3 := authChallenge(t, third, u.Username)
	rec["challenge"] = ch3.Challenge
	expectError(t, third.post("/api/auth/login/totp", rec), http.StatusUnauthorized, codeInvalidTOTP)
	// An empty normalised recovery code never matches.
	rec["recoveryCode"] = "--"
	expectError(t, third.post("/api/auth/login/totp", rec), http.StatusUnauthorized, codeInvalidTOTP)
}

func TestTwoFactorSecondFactorCannotBeBruteForced(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	authEnable2FA(t, e, c)
	anon := e.client(t)
	ch := authChallenge(t, anon, u.Username)
	for i := 0; i < 4; i++ {
		expectError(t, authTOTP(anon, ch.Challenge, "000000"), http.StatusUnauthorized, codeInvalidTOTP)
	}
	// 1 password reservation + 4 code failures = 5: the user key is locked.
	expectError(t, authLogin(anon, u.Username, authPassword), http.StatusTooManyRequests, codeTooManyAttempts)
	// The TOTP failures counted on the address key too.
	if row, _ := authThrottle(t, e, authLoopbackIPKey); row.Failures != 4 {
		t.Errorf("ip key after four code failures %+v", row)
	}
}

func TestTwoFactorChallengeLimits(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	secret, _ := authEnable2FA(t, e, c)
	anon := e.client(t)

	// Expiry after five minutes.
	ch := authChallenge(t, anon, u.Username)
	e.clock.Advance(challengeTTL + time.Second)
	expectError(t, authTOTP(anon, ch.Challenge, authCode(t, e, secret)), http.StatusUnauthorized, codeChallengeInvalid)

	// Five attempts per challenge (the throttle is cleared between the
	// attempts so that only the challenge limit applies).
	ch = authChallenge(t, anon, u.Username)
	for i := 1; i <= 5; i++ {
		if err := e.store.ClearUserThrottle(e.ctx(), u.ID); err != nil {
			t.Fatal(err)
		}
		res := authTOTP(anon, ch.Challenge, "000000")
		if i < 5 {
			expectError(t, res, http.StatusUnauthorized, codeInvalidTOTP)
		} else {
			expectError(t, res, http.StatusUnauthorized, codeChallengeInvalid)
		}
	}
	if err := e.store.ClearUserThrottle(e.ctx(), u.ID); err != nil {
		t.Fatal(err)
	}
	expectError(t, authTOTP(anon, ch.Challenge, authNextCode(t, e, secret)), http.StatusUnauthorized, codeChallengeInvalid)

	// One live challenge per user: the older one dies with the next sign-in.
	first := authChallenge(t, anon, u.Username)
	second := authChallenge(t, e.client(t), u.Username)
	expectError(t, authTOTP(anon, first.Challenge, authNextCode(t, e, secret)), http.StatusUnauthorized, codeChallengeInvalid)
	decodeStatus[loginOKResponse](t, authTOTP(anon, second.Challenge, authCode(t, e, secret)), http.StatusOK)
}

func TestTwoFactorStepLockedKeys(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	secret, _ := authEnable2FA(t, e, c)

	// Locked user key: the challenge stays but the step is 429.
	anon := e.client(t)
	ch := authChallenge(t, anon, u.Username)
	for i := 0; i < 4; i++ {
		expectError(t, authLogin(e.client(t), u.Username, "wrong password!"), http.StatusUnauthorized, codeInvalidCredentials)
	}
	expectError(t, authTOTP(anon, ch.Challenge, authNextCode(t, e, secret)), http.StatusTooManyRequests, codeTooManyAttempts)
	if len(authAudit(t, e, model.EventLoginLocked)) != 1 {
		t.Error("locked second step not audited")
	}

	// Locked address key.
	e2 := newEnv(t)
	c2, u2 := authSignIn(t, e2, model.RoleCustomer)
	secret2, _ := authEnable2FA(t, e2, c2)
	anon2 := e2.client(t)
	ch2 := authChallenge(t, anon2, u2.Username)
	for n := 0; n < 6; n++ {
		for i := 0; i < 5; i++ {
			authLogin(anon2, fmt.Sprintf("ghost%d", n), "wrong password!")
		}
	}
	expectError(t, authTOTP(anon2, ch2.Challenge, authNextCode(t, e2, secret2)), http.StatusTooManyRequests, codeTooManyAttempts)
}

func TestTwoFactorStepAccountChanged(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	secret, _ := authEnable2FA(t, e, c)
	anon := e.client(t)
	ch := authChallenge(t, anon, u.Username)
	inactive := false
	if _, err := e.store.UpdateUser(e.ctx(), u.ID, model.UserPatch{Active: &inactive}); err != nil {
		t.Fatal(err)
	}
	expectError(t, authTOTP(anon, ch.Challenge, authNextCode(t, e, secret)), http.StatusUnauthorized, codeChallengeInvalid)
	// The challenge is gone.
	active := true
	if _, err := e.store.UpdateUser(e.ctx(), u.ID, model.UserPatch{Active: &active}); err != nil {
		t.Fatal(err)
	}
	expectError(t, authTOTP(anon, ch.Challenge, authNextCode(t, e, secret)), http.StatusUnauthorized, codeChallengeInvalid)
}

func TestLoginTOTPValidation(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t)
	post := func(body any) *http.Response { return c.post("/api/auth/login/totp", body) }
	expectError(t, post("[]"), http.StatusBadRequest, codeInvalidJSON)
	expectFields(t, post(map[string]string{}), map[string]string{"challenge": fieldRequired, "code": fieldRequired})
	expectFields(t, post(map[string]string{"challenge": strings.Repeat("x", 129), "code": "123456"}), map[string]string{"challenge": fieldTooLong})
	expectFields(t, post(map[string]string{"challenge": "x", "code": "12345a"}), map[string]string{"code": fieldInvalidFormat})
	expectFields(t, post(map[string]string{"challenge": "x", "code": "123456", "recoveryCode": "abcde-23456"}), map[string]string{"recoveryCode": fieldNotAllowed})
	expectFields(t, post(map[string]string{"challenge": "x", "recoveryCode": ""}), map[string]string{"recoveryCode": fieldRequired})
	expectFields(t, post(map[string]string{"challenge": "x", "recoveryCode": strings.Repeat("a", 65)}), map[string]string{"recoveryCode": fieldTooLong})
	expectError(t, post(map[string]string{"challenge": "unknown", "code": "123456"}), http.StatusUnauthorized, codeChallengeInvalid)
}

// ---------------------------------------------------------------------------
// Password change and the gates

func TestChangePassword(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	other, _ := e.createSession(t, u.ID)
	change := func(current, next string) *http.Response {
		return c.post("/api/auth/password", map[string]string{"currentPassword": current, "newPassword": next})
	}
	expectFields(t, c.post("/api/auth/password", map[string]string{}), map[string]string{"currentPassword": fieldRequired, "newPassword": fieldRequired})
	body := expectError(t, change("wrong password!", authNewPassword), http.StatusBadRequest, codeCurrentPasswordInvalid)
	if body.Fields["currentPassword"] != fieldIncorrect {
		t.Errorf("fields %v", body.Fields)
	}
	for next, code := range map[string]string{
		"short":                       fieldPasswordTooShort,
		strings.Repeat("ab", 65):      fieldPasswordTooLong,
		"xx" + u.Username + "-xx1234": fieldPasswordContainsUsername,
		"aaaaaaaaaaaaaaabbbbb":        fieldPasswordTooSimple,
		authPassword:                  fieldPasswordReused,
	} {
		expectFields(t, change(authPassword, next), map[string]string{"newPassword": code})
	}
	expectStatus(t, change(authPassword, authNewPassword), http.StatusNoContent)
	authExpectAudit(t, e, model.EventPasswordChanged, 1)

	// The current session stays, the other one is gone.
	expectStatus(t, c.get("/api/auth/me"), http.StatusOK)
	stale := e.client(t)
	stale.setSessionToken(other)
	expectError(t, stale.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	expectError(t, authLogin(e.client(t), u.Username, authPassword), http.StatusUnauthorized, codeInvalidCredentials)
	decodeStatus[loginOKResponse](t, authLogin(e.client(t), u.Username, authNewPassword), http.StatusOK)

	// Wrong current passwords count on the user key (the sign-in above
	// cleared it): after five failures even the right password is 429.
	for i := 0; i < 5; i++ {
		expectError(t, change("wrong password!", "Another-Pass-123"), http.StatusBadRequest, codeCurrentPasswordInvalid)
	}
	expectError(t, change(authNewPassword, "Another-Pass-123"), http.StatusTooManyRequests, codeTooManyAttempts)
}

func TestPasswordAndConsentGates(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	hash, err := testHasher.Hash(authPassword)
	if err != nil {
		t.Fatal(err)
	}
	u := storetest.SeedNewUser(t, e.store, model.NewUser{
		Username: authUsername("gate"), Role: model.RoleDeveloper, PasswordHash: hash, MustChangePassword: true,
	})
	c := e.client(t)
	me := decodeStatus[loginOKResponse](t, authLogin(c, u.Username, authPassword), http.StatusOK).User
	if !me.MustChangePassword || !me.PrivacyConsentRequired {
		t.Fatalf("me %+v", me)
	}
	// Exempt routes work, gated routes answer the password gate first.
	expectStatus(t, c.get("/api/me/preferences"), http.StatusOK)
	expectStatus(t, c.get("/api/me/privacy"), http.StatusOK)
	expectError(t, c.get("/api/me/sessions"), http.StatusForbidden, codePasswordChangeRequired)
	expectError(t, c.get("/api/links"), http.StatusForbidden, codePasswordChangeRequired)

	expectStatus(t, c.post("/api/auth/password", map[string]string{"currentPassword": authPassword, "newPassword": authNewPassword}), http.StatusNoContent)
	expectError(t, c.get("/api/me/sessions"), http.StatusForbidden, codeConsentRequired)
	status := decodeStatus[privacyStatusResponse](t, c.get("/api/me/privacy"), http.StatusOK)
	if !status.ConsentRequired {
		t.Fatalf("privacy status %+v", status)
	}
	expectStatus(t, c.post("/api/me/privacy/consent", map[string]int{"version": status.CurrentVersion}), http.StatusNoContent)
	expectStatus(t, c.get("/api/me/sessions"), http.StatusOK)
	me = decodeStatus[meResponse](t, c.get("/api/auth/me"), http.StatusOK)
	if me.MustChangePassword || me.PrivacyConsentRequired {
		t.Errorf("me after the gates %+v", me)
	}
}

func TestLockAuditLimiter(t *testing.T) {
	t.Parallel()
	var l lockAuditLimiter
	t0 := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	if !l.allow("a", t0) || l.allow("a", t0.Add(lockAuditInterval-time.Second)) {
		t.Fatal("interval not enforced")
	}
	if !l.allow("b", t0.Add(time.Minute)) {
		t.Fatal("other key limited")
	}
	if !l.allow("a", t0.Add(lockAuditInterval)) {
		t.Fatal("key not allowed after the interval")
	}
	// Old entries are pruned.
	l.allow("c", t0.Add(3*lockAuditInterval))
	if len(l.last) != 1 {
		t.Errorf("entries %v", l.last)
	}
}

func TestAuditWriteFailureIsLogged(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	u := authSeed(t, e, model.RoleCustomer)
	fs.arm("AppendAudit", 1, nil)
	decodeStatus[loginOKResponse](t, authLogin(e.client(t), u.Username, authPassword), http.StatusOK)
	if e.logs.count("audit write failed") != 1 || e.logs.count(model.EventLoginSuccess) != 1 {
		t.Errorf("log:\n%s", e.logs)
	}
}

func TestMalformedStoredHashNeverMatches(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	u := e.seedUser(t, model.RoleCustomer) // storetest.SeedPasswordHash is not a valid hash
	expectError(t, authLogin(e.client(t), u.Username, storetest.SeedPasswordHash), http.StatusUnauthorized, codeInvalidCredentials)
	if e.logs.count("stored password hash is unusable") != 1 {
		t.Errorf("log:\n%s", e.logs)
	}
}

func TestAuthFaults(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	plain := authSeed(t, e, model.RoleCustomer)
	twoFA, twoFAUser := authSignIn(t, e, model.RoleCustomer)
	secret, codes := authEnable2FA(t, e, twoFA)
	resetThrottle := func() {
		for _, id := range []int64{plain.ID, twoFAUser.ID} {
			if err := e.store.ClearUserThrottle(e.ctx(), id); err != nil {
				t.Fatal(err)
			}
		}
		if err := e.store.ClearThrottle(e.ctx(), authLoopbackIPKey); err != nil {
			t.Fatal(err)
		}
	}
	login := func(name, password string) func() func() *http.Response {
		return func() func() *http.Response {
			resetThrottle()
			c := e.client(t)
			return func() *http.Response { return authLogin(c, name, password) }
		}
	}
	// totp prepares a challenge and, for wrong > 0, that many failed codes
	// before the request.
	totp := func(useRecovery bool, wrong int) func() func() *http.Response {
		return func() func() *http.Response {
			resetThrottle()
			c := e.client(t)
			ch := authChallenge(t, c, twoFAUser.Username)
			for i := 0; i < wrong; i++ {
				_ = e.store.ClearUserThrottle(e.ctx(), twoFAUser.ID)
				expectError(t, authTOTP(c, ch.Challenge, "000000"), http.StatusUnauthorized, codeInvalidTOTP)
			}
			_ = e.store.ClearUserThrottle(e.ctx(), twoFAUser.ID)
			return func() *http.Response {
				switch {
				case wrong > 0:
					return authTOTP(c, ch.Challenge, "000000")
				case useRecovery:
					return c.post("/api/auth/login/totp", map[string]string{"challenge": ch.Challenge, "recoveryCode": codes[len(codes)-1]})
				}
				return authTOTP(c, ch.Challenge, authNextCode(t, e, secret))
			}
		}
	}
	signedIn := func(method, path string, body any) func() func() *http.Response {
		return func() func() *http.Response {
			resetThrottle()
			c, _ := authSignIn(t, e, model.RoleCustomer)
			return func() *http.Response { return c.do(method, path, body) }
		}
	}
	twoFAGet := func() func() *http.Response {
		return func() *http.Response { return twoFA.get("/api/auth/me") }
	}
	pw := map[string]string{"currentPassword": authPassword, "newPassword": authNewPassword}
	wrongPW := map[string]string{"currentPassword": "wrong password!", "newPassword": authNewPassword}
	post := http.MethodPost
	runAccountFaults(t, fs, []accountFault{
		{name: "login credentials", method: "Credentials", nth: 1, prepare: login(plain.Username, authPassword)},
		{name: "login reserve", method: "ThrottleUpdate", nth: 1, prepare: login(plain.Username, authPassword)},
		{name: "login unknown confirm", method: "ThrottleUpdate", nth: 2, prepare: login("ghost-user", authPassword)},
		{name: "login wrong confirm", method: "ThrottleUpdate", nth: 2, prepare: login(plain.Username, "wrong password!")},
		{name: "login ip refund", method: "ThrottleUpdate", nth: 2, prepare: login(plain.Username, authPassword)},
		{name: "login clear user key", method: "ClearThrottle", nth: 1, prepare: login(plain.Username, authPassword)},
		{name: "login session", method: "CreateSession", nth: 1, prepare: login(plain.Username, authPassword)},
		{name: "login record", method: "RecordLogin", nth: 1, prepare: login(plain.Username, authPassword)},
		{name: "login reload", method: "UserByID", nth: 1, prepare: login(plain.Username, authPassword)},
		{name: "login settings", method: "Settings", nth: 2, prepare: login(plain.Username, authPassword)},
		{name: "login me", method: "Preferences", nth: 1, prepare: login(plain.Username, authPassword)},
		{name: "login challenge", method: "CreateChallenge", nth: 1, prepare: login(twoFAUser.Username, authPassword)},
		{name: "totp attempt", method: "ChallengeAttempt", nth: 1, prepare: totp(false, 0)},
		{name: "totp user", method: "UserByID", nth: 1, prepare: totp(false, 0)},
		{name: "totp reserve", method: "ThrottleUpdate", nth: 1, prepare: totp(false, 0)},
		{name: "totp credentials", method: "Credentials", nth: 1, prepare: totp(false, 0)},
		{name: "totp advance", method: "AdvanceTOTPStep", nth: 1, prepare: totp(false, 0)},
		{name: "totp delete", method: "DeleteChallenge", nth: 1, prepare: totp(false, 0)},
		{name: "totp clear", method: "ClearThrottle", nth: 1, prepare: totp(false, 0)},
		{name: "totp refund", method: "ThrottleUpdate", nth: 2, prepare: totp(false, 0)},
		{name: "totp recovery", method: "UseRecoveryCode", nth: 1, prepare: totp(true, 0)},
		{name: "totp remaining", method: "RecoveryCodesRemaining", nth: 1, prepare: totp(true, 0)},
		{name: "totp confirm", method: "ThrottleUpdate", nth: 2, prepare: totp(false, 1)},
		{name: "totp drop", method: "DeleteChallenge", nth: 1, prepare: totp(false, 4)},
		{name: "logout lookup", method: "SessionByToken", nth: 1, prepare: signedIn(post, "/api/auth/logout", nil)},
		{name: "logout settings", method: "Settings", nth: 1, prepare: signedIn(post, "/api/auth/logout", nil)},
		{name: "logout delete", method: "DeleteSessionByToken", nth: 1, prepare: signedIn(post, "/api/auth/logout", nil)},
		{name: "me settings", method: "Settings", nth: 2, prepare: signedIn(http.MethodGet, "/api/auth/me", nil)},
		{name: "me consents", method: "Consents", nth: 1, prepare: signedIn(http.MethodGet, "/api/auth/me", nil)},
		{name: "me remaining", method: "RecoveryCodesRemaining", nth: 1, prepare: twoFAGet},
		{name: "password reserve", method: "ThrottleUpdate", nth: 1, prepare: signedIn(post, "/api/auth/password", pw)},
		{name: "password credentials", method: "Credentials", nth: 1, prepare: signedIn(post, "/api/auth/password", pw)},
		{name: "password refund", method: "ThrottleUpdate", nth: 2, prepare: signedIn(post, "/api/auth/password", pw)},
		{name: "password set", method: "SetPassword", nth: 1, prepare: signedIn(post, "/api/auth/password", pw)},
		{name: "password sessions", method: "DeleteUserSessions", nth: 1, prepare: signedIn(post, "/api/auth/password", pw)},
		{name: "password wrong confirm", method: "ThrottleUpdate", nth: 2, prepare: signedIn(post, "/api/auth/password", wrongPW)},
	})
}

func TestTwoFactorDropChallengeFaults(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	secret, _ := authEnable2FA(t, e, c)
	setActive := func(active bool) {
		if _, err := e.store.UpdateUser(e.ctx(), u.ID, model.UserPatch{Active: &active}); err != nil {
			t.Fatal(err)
		}
	}
	// prepare: a challenge, then the account is deactivated, so the second
	// step drops the challenge and refunds the reservation.
	prepare := func() func() *http.Response {
		setActive(true)
		anon := e.client(t)
		ch := authChallenge(t, anon, u.Username)
		setActive(false)
		return func() *http.Response { return authTOTP(anon, ch.Challenge, authNextCode(t, e, secret)) }
	}
	runAccountFaults(t, fs, []accountFault{
		{name: "delete", method: "DeleteChallenge", nth: 1, prepare: prepare},
		{name: "refund", method: "ThrottleUpdate", nth: 2, prepare: prepare},
	})
}
