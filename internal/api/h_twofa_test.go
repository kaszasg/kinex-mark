package api

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// twofaRecoveryCode is the displayed recovery code shape (6.6).
var twofaRecoveryCode = regexp.MustCompile(`^[abcdefghjkmnpqrstuvwxyz23456789]{5}-[abcdefghjkmnpqrstuvwxyz23456789]{5}$`)

func TestTwoFactorSetup(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	setup := func(pw any) *http.Response { return c.post("/api/me/2fa/setup", pw) }

	expectError(t, setup("{"), http.StatusBadRequest, codeInvalidJSON)
	expectFields(t, setup(map[string]string{}), map[string]string{"password": fieldRequired})
	body := expectError(t, setup(map[string]string{"password": "wrong password!"}), http.StatusBadRequest, codeCurrentPasswordInvalid)
	if body.Fields["currentPassword"] != fieldIncorrect {
		t.Errorf("fields %v", body.Fields)
	}

	res := setup(map[string]string{"password": authPassword})
	got := decodeStatus[twoFactorSetupResponse](t, res, http.StatusOK)
	if len(got.Secret) != 32 || !got.ExpiresAt.Equal(e.clock.Now().Add(pendingTOTPTTL)) {
		t.Errorf("setup %+v", got)
	}
	if want := auth.TOTPURI("Kinex Mark", u.Username, got.Secret); got.OTPAuthURI != want {
		t.Errorf("uri %q, want %q", got.OTPAuthURI, want)
	}
	if !strings.HasPrefix(got.OTPAuthURI, "otpauth://totp/Kinex%20Mark:"+u.Username+"?secret=") {
		t.Errorf("uri %q", got.OTPAuthURI)
	}
	twofaJSONKeys(t, readBody(t, res), "expiresAt,otpauthUri,secret")
	// The secret is stored sealed, never in clear text.
	if text := authTableText(t, e, "users"); strings.Contains(text, got.Secret) {
		t.Error("pending secret stored in clear text")
	}

	// Already enabled.
	secret, _ := authEnable2FA(t, e, c)
	if secret == got.Secret {
		t.Error("a second setup reused the secret")
	}
	expectError(t, setup(map[string]string{"password": authPassword}), http.StatusConflict, codeTOTPAlreadyEnabled)
}

// twofaJSONKeys compares the sorted top-level keys with a comma list.
func twofaJSONKeys(t *testing.T, body []byte, want string) {
	t.Helper()
	if got := strings.Join(jsonKeys(t, body), ","); got != want {
		t.Errorf("keys %s, want %s", got, want)
	}
}

func TestTwoFactorEnable(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	other, _ := e.createSession(t, u.ID)
	enable := func(code any) *http.Response { return c.post("/api/me/2fa/enable", code) }

	expectFields(t, enable(map[string]string{}), map[string]string{"code": fieldRequired})
	expectFields(t, enable(map[string]string{"code": "12345"}), map[string]string{"code": fieldInvalidFormat})
	expectError(t, enable(map[string]string{"code": "123456"}), http.StatusConflict, codeTOTPSetupRequired)

	// An expired setup is gone.
	setup := decodeStatus[twoFactorSetupResponse](t, c.post("/api/me/2fa/setup", map[string]string{"password": authPassword}), http.StatusOK)
	e.clock.Advance(pendingTOTPTTL + time.Second)
	expectError(t, enable(map[string]string{"code": authCode(t, e, setup.Secret)}), http.StatusConflict, codeTOTPSetupRequired)

	setup = decodeStatus[twoFactorSetupResponse](t, c.post("/api/me/2fa/setup", map[string]string{"password": authPassword}), http.StatusOK)
	expectError(t, enable(map[string]string{"code": "000000"}), http.StatusUnauthorized, codeInvalidTOTP)
	rc := decodeStatus[recoveryCodesResponse](t, enable(map[string]string{"code": authCode(t, e, setup.Secret)}), http.StatusOK)
	if len(rc.RecoveryCodes) != 10 {
		t.Fatalf("codes %v", rc.RecoveryCodes)
	}
	seen := map[string]bool{}
	for _, code := range rc.RecoveryCodes {
		if !twofaRecoveryCode.MatchString(code) || seen[code] {
			t.Errorf("recovery code %q", code)
		}
		seen[code] = true
	}
	authExpectAudit(t, e, model.Event2FAEnabled, 1)

	// The other session is gone, the current one stays.
	me := decodeStatus[meResponse](t, c.get("/api/auth/me"), http.StatusOK)
	if !me.TOTPEnabled || me.RecoveryCodesRemaining != 10 {
		t.Errorf("me %+v", me)
	}
	stale := e.client(t)
	stale.setSessionToken(other)
	expectError(t, stale.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)

	// Enabled: no second enable.
	expectError(t, enable(map[string]string{"code": "123456"}), http.StatusConflict, codeTOTPAlreadyEnabled)
	// Recovery codes are stored as keyed MACs only.
	if text := authTableText(t, e, "recovery_codes"); strings.Contains(text, rc.RecoveryCodes[0]) ||
		strings.Contains(text, auth.NormalizeRecoveryCode(rc.RecoveryCodes[0])) {
		t.Error("recovery code stored in clear text")
	}
}

func TestTwoFactorEnableConflict(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	c, _ := authSignIn(t, e, model.RoleCustomer)
	setup := decodeStatus[twoFactorSetupResponse](t, c.post("/api/me/2fa/setup", map[string]string{"password": authPassword}), http.StatusOK)
	// A concurrent enable won the race: the store reports a conflict.
	fs.arm("EnableTOTP", 1, model.ErrConflict)
	expectError(t, c.post("/api/me/2fa/enable", map[string]string{"code": authCode(t, e, setup.Secret)}), http.StatusConflict, codeTOTPAlreadyEnabled)
}

func TestTwoFactorDisable(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	disable := func(pw, code string) *http.Response {
		return c.post("/api/me/2fa/disable", map[string]string{"password": pw, "code": code})
	}
	expectError(t, disable(authPassword, "123456"), http.StatusConflict, codeTOTPNotEnabled)
	secret, codes := authEnable2FA(t, e, c)
	other, _ := e.createSession(t, u.ID)

	expectFields(t, c.post("/api/me/2fa/disable", map[string]string{}), map[string]string{"password": fieldRequired, "code": fieldRequired})
	expectFields(t, disable(authPassword, strings.Repeat("1", 65)), map[string]string{"code": fieldTooLong})
	expectError(t, disable("wrong password!", authNextCode(t, e, secret)), http.StatusBadRequest, codeCurrentPasswordInvalid)
	expectError(t, disable(authPassword, "000000"), http.StatusUnauthorized, codeInvalidTOTP)
	expectError(t, disable(authPassword, "abcde-23456"), http.StatusUnauthorized, codeInvalidTOTP)
	// The wrong code counted on the user key, a correct step-up refunds.
	if row, _ := authThrottle(t, e, authUserKey(t, u.Username)); row.Failures != 3 {
		t.Errorf("user key %+v", row)
	}

	// A recovery code is accepted as the second factor.
	expectStatus(t, disable(authPassword, codes[0]), http.StatusNoContent)
	authExpectAudit(t, e, model.Event2FADisabled, 1)
	me := decodeStatus[meResponse](t, c.get("/api/auth/me"), http.StatusOK)
	if me.TOTPEnabled || me.RecoveryCodesRemaining != 0 {
		t.Errorf("me %+v", me)
	}
	stale := e.client(t)
	stale.setSessionToken(other)
	expectError(t, stale.get("/api/auth/me"), http.StatusUnauthorized, codeUnauthenticated)
	// Signing in needs no second step any more.
	decodeStatus[loginOKResponse](t, authLogin(e.client(t), u.Username, authPassword), http.StatusOK)

	// With a TOTP code.
	secret, _ = authEnable2FA(t, e, c)
	expectStatus(t, disable(authPassword, authNextCode(t, e, secret)), http.StatusNoContent)
}

func TestTwoFactorRecoveryCodes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := authSignIn(t, e, model.RoleCustomer)
	regen := func(pw, code string) *http.Response {
		return c.post("/api/me/2fa/recovery-codes", map[string]string{"password": pw, "code": code})
	}
	expectError(t, regen(authPassword, "123456"), http.StatusConflict, codeTOTPNotEnabled)
	secret, old := authEnable2FA(t, e, c)

	expectFields(t, regen("", ""), map[string]string{"password": fieldRequired, "code": fieldRequired})
	// Only a TOTP code, not a recovery code.
	expectFields(t, regen(authPassword, old[0]), map[string]string{"code": fieldInvalidFormat})
	expectError(t, regen("wrong password!", authNextCode(t, e, secret)), http.StatusBadRequest, codeCurrentPasswordInvalid)
	expectError(t, regen(authPassword, "000000"), http.StatusUnauthorized, codeInvalidTOTP)

	fresh := decodeStatus[recoveryCodesResponse](t, regen(authPassword, authNextCode(t, e, secret)), http.StatusOK)
	if len(fresh.RecoveryCodes) != 10 || fresh.RecoveryCodes[0] == old[0] {
		t.Fatalf("codes %v", fresh.RecoveryCodes)
	}
	authExpectAudit(t, e, model.Event2FARecoveryRegenerated, 1)

	// The old codes are void, the new ones work.
	anon := e.client(t)
	ch := authChallenge(t, anon, u.Username)
	expectError(t, anon.post("/api/auth/login/totp", map[string]string{"challenge": ch.Challenge, "recoveryCode": old[1]}),
		http.StatusUnauthorized, codeInvalidTOTP)
	decodeStatus[loginOKResponse](t, anon.post("/api/auth/login/totp", map[string]string{"challenge": ch.Challenge, "recoveryCode": fresh.RecoveryCodes[1]}),
		http.StatusOK)
}

func TestTwoFactorStepUpThrottle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := authSignIn(t, e, model.RoleCustomer)
	for i := 0; i < 5; i++ {
		expectError(t, c.post("/api/me/2fa/setup", map[string]string{"password": "wrong password!"}), http.StatusBadRequest, codeCurrentPasswordInvalid)
	}
	res := c.post("/api/me/2fa/setup", map[string]string{"password": authPassword})
	expectError(t, res, http.StatusTooManyRequests, codeTooManyAttempts)
	if res.Header.Get("Retry-After") == "" {
		t.Error("no Retry-After")
	}
}

func TestTwoFactorFaults(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	post := func(path string, body func(secret string) any, enabled bool) func() func() *http.Response {
		return func() func() *http.Response {
			c, _ := authSignIn(t, e, model.RoleCustomer)
			secret := ""
			if enabled {
				secret, _ = authEnable2FA(t, e, c)
			}
			return func() *http.Response { return c.post(path, body(secret)) }
		}
	}
	password := func(string) any { return map[string]string{"password": authPassword} }
	withCode := func(secret string) any {
		return map[string]string{"password": authPassword, "code": authNextCode(t, e, secret)}
	}
	withRecovery := func(string) any { return map[string]string{"password": authPassword, "code": "abcde-23456"} }
	enable := func() func() *http.Response {
		c, _ := authSignIn(t, e, model.RoleCustomer)
		setup := decodeStatus[twoFactorSetupResponse](t, c.post("/api/me/2fa/setup", map[string]string{"password": authPassword}), http.StatusOK)
		return func() *http.Response {
			return c.post("/api/me/2fa/enable", map[string]string{"code": authCode(t, e, setup.Secret)})
		}
	}
	runAccountFaults(t, fs, []accountFault{
		{name: "setup step-up", method: "Credentials", nth: 1, prepare: post("/api/me/2fa/setup", password, false)},
		{name: "setup pending", method: "SetPendingTOTP", nth: 1, prepare: post("/api/me/2fa/setup", password, false)},
		{name: "enable pending", method: "PendingTOTP", nth: 1, prepare: enable},
		{name: "enable store", method: "EnableTOTP", nth: 1, prepare: enable},
		{name: "enable sessions", method: "DeleteUserSessions", nth: 1, prepare: enable},
		{name: "disable step-up", method: "ThrottleUpdate", nth: 1, prepare: post("/api/me/2fa/disable", withCode, true)},
		{name: "disable advance", method: "AdvanceTOTPStep", nth: 1, prepare: post("/api/me/2fa/disable", withCode, true)},
		{name: "disable recovery", method: "UseRecoveryCode", nth: 1, prepare: post("/api/me/2fa/disable", withRecovery, true)},
		{name: "disable store", method: "DisableTOTP", nth: 1, prepare: post("/api/me/2fa/disable", withCode, true)},
		{name: "disable sessions", method: "DeleteUserSessions", nth: 1, prepare: post("/api/me/2fa/disable", withCode, true)},
		{name: "regenerate step-up", method: "Credentials", nth: 1, prepare: post("/api/me/2fa/recovery-codes", withCode, true)},
		{name: "regenerate store", method: "ReplaceRecoveryCodes", nth: 1, prepare: post("/api/me/2fa/recovery-codes", withCode, true)},
	})
}

func TestTwoFactorSealedSecretUnreadable(t *testing.T) {
	t.Parallel()
	// A keyring that cannot open the stored secret (wrong server key) is an
	// internal error, never a silent pass.
	e := newEnv(t)
	c, _ := authSignIn(t, e, model.RoleCustomer)
	setup := decodeStatus[twoFactorSetupResponse](t, c.post("/api/me/2fa/setup", map[string]string{"password": authPassword}), http.StatusOK)
	other, err := auth.NewKeyring([]byte("another-32-byte-key-for-the-test"))
	if err != nil {
		t.Fatal(err)
	}
	d := e.deps
	d.Keyring = other
	h, err := New(d)
	if err != nil {
		t.Fatal(err)
	}
	send := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1"+path, strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set(headerKinexRequest, "1")
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: e.cookieName(), Value: c.sessionToken()})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	expectRecError(t, send("/api/me/2fa/enable", `{"code":"`+authCode(t, e, setup.Secret)+`"}`), http.StatusInternalServerError, codeInternalError)

	// The same for the disable step-up once 2FA is on.
	decodeStatus[recoveryCodesResponse](t, c.post("/api/me/2fa/enable", map[string]string{"code": authCode(t, e, setup.Secret)}), http.StatusOK)
	expectRecError(t, send("/api/me/2fa/disable", `{"password":"`+authPassword+`","code":"123456"}`), http.StatusInternalServerError, codeInternalError)
}
