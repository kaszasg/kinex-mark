package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/store"
	"github.com/kaszasg/kinex-mark/internal/storetest"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

// errInjected is the failure of faultStore.
var errInjected = errors.New("injected store failure")

// faultStore wraps a Store and fails the methods named in fail.
type faultStore struct {
	Store
	fail map[string]bool
}

func (f faultStore) SessionByToken(ctx context.Context, hash []byte) (model.Session, model.User, error) {
	if f.fail["SessionByToken"] {
		return model.Session{}, model.User{}, errInjected
	}
	return f.Store.SessionByToken(ctx, hash)
}

func (f faultStore) Settings(ctx context.Context) (model.SystemSettings, error) {
	if f.fail["Settings"] {
		return model.SystemSettings{}, errInjected
	}
	return f.Store.Settings(ctx)
}

func (f faultStore) DeleteSessionByToken(ctx context.Context, hash []byte) error {
	if f.fail["DeleteSessionByToken"] {
		return errInjected
	}
	return f.Store.DeleteSessionByToken(ctx, hash)
}

func (f faultStore) TouchSession(ctx context.Context, id int64) error {
	if f.fail["TouchSession"] {
		return errInjected
	}
	return f.Store.TouchSession(ctx, id)
}

func (f faultStore) Consents(ctx context.Context, userID int64) ([]model.Consent, error) {
	if f.fail["Consents"] {
		return nil, errInjected
	}
	return f.Store.Consents(ctx, userID)
}

func (f faultStore) CreateSession(ctx context.Context, n model.NewSession) (model.Session, error) {
	if f.fail["CreateSession"] {
		return model.Session{}, errInjected
	}
	return f.Store.CreateSession(ctx, n)
}

func (f faultStore) Ping(ctx context.Context) error {
	if f.fail["Ping"] {
		return errInjected
	}
	return f.Store.Ping(ctx)
}

// guardFixture is a server over a real store whose per-route guard is
// tested with an echo handler, independent of the real route handlers.
type guardFixture struct {
	t    *testing.T
	s    *server
	st   *store.Store
	fail map[string]bool // methods of faultStore that fail
	clk  *clock.Fake
	ctx  context.Context
}

// echoIdentity is the response of echoHandler.
type echoIdentity struct {
	UserID    int64      `json:"userId"`
	SessionID int64      `json:"sessionId"`
	ViewerID  int64      `json:"viewerId"`
	Role      model.Role `json:"role"`
	LastSeen  time.Time  `json:"lastSeen"`
}

// echoHandler answers the identity found in the request context.
func echoHandler(_ *server, w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, echoIdentity{
		UserID:    userFrom(r).ID,
		SessionID: sessionFrom(r).ID,
		ViewerID:  viewerFrom(r).UserID,
		Role:      userFrom(r).Role,
		LastSeen:  sessionFrom(r).LastSeenAt,
	})
}

// newGuardFixture builds the fixture; edit may change the Deps.
func newGuardFixture(t *testing.T, edit ...func(*Deps)) *guardFixture {
	t.Helper()
	clk := testutil.NewClock()
	st := storetest.NewStore(t, storetest.WithClock(clk))
	fail := map[string]bool{}
	d := minimalDeps(t)
	d.Store = faultStore{Store: st, fail: fail}
	d.Clock = clk
	for _, e := range edit {
		e(&d)
	}
	s, err := newServer(d)
	if err != nil {
		t.Fatal(err)
	}
	return &guardFixture{t: t, s: s, st: st, fail: fail, clk: clk, ctx: context.Background()}
}

// session creates a 168-hour session of the user and returns its token.
func (f *guardFixture) session(userID int64) string {
	f.t.Helper()
	token, hash, err := auth.NewToken()
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.st.CreateSession(f.ctx, model.NewSession{
		TokenHash: hash, UserID: userID, ExpiresAt: f.clk.Now().Add(168 * time.Hour), IP: "127.0.0.1",
	}); err != nil {
		f.t.Fatal(err)
	}
	return token
}

// call runs the guard of a route with access level a (deny code deny) for
// a request carrying token as the plain session cookie ("" for none).
func (f *guardFixture) call(a access, deny, token string) *httptest.ResponseRecorder {
	f.t.Helper()
	rt := route{method: http.MethodGet, pattern: "/api/echo", access: a, deny: deny, handle: echoHandler}
	r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/echo", nil)
	if token != "" {
		r.AddCookie(&http.Cookie{Name: sessionCookiePlain, Value: token})
	}
	rec := httptest.NewRecorder()
	f.s.guard(rt).ServeHTTP(rec, r)
	return rec
}

// sessions returns the user's stored sessions.
func (f *guardFixture) sessions(userID int64) []model.Session {
	f.t.Helper()
	list, err := f.st.Sessions(f.ctx, userID)
	if err != nil {
		f.t.Fatal(err)
	}
	return list
}

// expectRecError checks status and code of an error recorded by rec.
func expectRecError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d: %s", rec.Code, status, rec.Body.String())
	}
	if got := decodeErrorBody(t, rec).Code; got != code {
		t.Fatalf("code %q, want %q", got, code)
	}
}

// clearsCookie reports whether rec deletes the cookie name.
func clearsCookie(rec *httptest.ResponseRecorder, name string) bool {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name && c.MaxAge < 0 && c.Value == "" {
			return true
		}
	}
	return false
}

func TestGuardAuthentication(t *testing.T) {
	t.Parallel()
	f := newGuardFixture(t)
	u := storetest.SeedUser(t, f.st, model.RoleDeveloper)
	token := f.session(u.ID)

	// Public routes run without a session, ignore any cookie and see the
	// zero identity.
	rec := f.call(accessPublic, "", "bogus")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"userId":0`) || len(rec.Result().Cookies()) != 0 {
		t.Fatalf("public: %d %s %v", rec.Code, rec.Body.String(), rec.Result().Cookies())
	}

	// No cookie: 401 without touching the cookie.
	rec = f.call(accessExempt, "", "")
	expectRecError(t, rec, http.StatusUnauthorized, codeUnauthenticated)
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("no cookie must not be cleared")
	}

	// Unknown token: 401 and the cookie is cleared.
	rec = f.call(accessUser, "", "unknown-token")
	expectRecError(t, rec, http.StatusUnauthorized, codeUnauthenticated)
	if !clearsCookie(rec, sessionCookiePlain) {
		t.Fatal("unknown token: cookie not cleared")
	}

	// Valid session: the handler sees user, session and viewer.
	for _, a := range []access{accessExempt, accessUser, accessSharer} {
		rec = f.call(a, codeForbidden, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", a, rec.Code, rec.Body.String())
		}
		var got echoIdentity
		decodeRec(t, rec, &got)
		if got.UserID != u.ID || got.ViewerID != u.ID || got.Role != model.RoleDeveloper || got.SessionID == 0 {
			t.Fatalf("%s: identity %+v", a, got)
		}
	}
}

func TestGuardRolesAndGates(t *testing.T) {
	t.Parallel()
	f := newGuardFixture(t)
	customer := storetest.SeedUser(t, f.st, model.RoleCustomer)
	developer := storetest.SeedUser(t, f.st, model.RoleDeveloper)
	admin := storetest.SeedUser(t, f.st, model.RoleAdmin)
	tokens := map[model.Role]string{
		model.RoleCustomer:  f.session(customer.ID),
		model.RoleDeveloper: f.session(developer.ID),
		model.RoleAdmin:     f.session(admin.ID),
	}

	// 5.13: D denies customers with the route's code, A denies non-admins.
	matrix := []struct {
		a      access
		deny   string
		denied map[model.Role]bool
	}{
		{accessExempt, "", nil},
		{accessUser, "", nil},
		{accessSharer, codeShareNotAllowed, map[model.Role]bool{model.RoleCustomer: true}},
		{accessSharer, codeForbidden, map[model.Role]bool{model.RoleCustomer: true}},
		{accessAdmin, codeForbidden, map[model.Role]bool{model.RoleCustomer: true, model.RoleDeveloper: true}},
	}
	for _, m := range matrix {
		for role, token := range tokens {
			rec := f.call(m.a, m.deny, token)
			if m.denied[role] {
				expectRecError(t, rec, http.StatusForbidden, m.deny)
			} else if rec.Code != http.StatusOK {
				t.Fatalf("%s/%s: %d %s", m.a, role, rec.Code, rec.Body.String())
			}
		}
	}

	// Password-change gate: B, D and A routes are closed, K routes open.
	mustChange := storetest.SeedNewUser(t, f.st, model.NewUser{
		Username: "newcomer", Role: model.RoleAdmin, PasswordHash: storetest.SeedPasswordHash, MustChangePassword: true,
	})
	mt := f.session(mustChange.ID)
	for _, a := range []access{accessUser, accessSharer, accessAdmin} {
		expectRecError(t, f.call(a, codeForbidden, mt), http.StatusForbidden, codePasswordChangeRequired)
	}
	if rec := f.call(accessExempt, "", mt); rec.Code != http.StatusOK {
		t.Fatalf("exempt route behind the password gate: %d", rec.Code)
	}

	// Consent gate: a user without consent for the current notice.
	noConsent := storetest.SeedNewUser(t, f.st, model.NewUser{
		Username: "unconsented", Role: model.RoleCustomer, PasswordHash: storetest.SeedPasswordHash,
	})
	nt := f.session(noConsent.ID)
	expectRecError(t, f.call(accessUser, "", nt), http.StatusForbidden, codeConsentRequired)
	if rec := f.call(accessExempt, "", nt); rec.Code != http.StatusOK {
		t.Fatalf("exempt route behind the consent gate: %d", rec.Code)
	}
	// The password gate comes first.
	if err := f.st.SetPassword(f.ctx, noConsent.ID, storetest.SeedPasswordHash, true); err != nil {
		t.Fatal(err)
	}
	expectRecError(t, f.call(accessUser, "", nt), http.StatusForbidden, codePasswordChangeRequired)

	// A new notice version closes the gate for everyone until acknowledged;
	// the gate runs before the role check.
	set, err := f.st.BumpPrivacyNotice(f.ctx, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	expectRecError(t, f.call(accessUser, "", tokens[model.RoleAdmin]), http.StatusForbidden, codeConsentRequired)
	expectRecError(t, f.call(accessAdmin, codeForbidden, tokens[model.RoleCustomer]), http.StatusForbidden, codeConsentRequired)
	if err := f.st.RecordConsent(f.ctx, admin.ID, "marketing", set.PrivacyNoticeVersion); err == nil {
		// Another consent kind never satisfies the gate (the store may reject it).
		expectRecError(t, f.call(accessUser, "", tokens[model.RoleAdmin]), http.StatusForbidden, codeConsentRequired)
	}
	if err := f.st.RecordConsent(f.ctx, admin.ID, model.ConsentPrivacyNotice, set.PrivacyNoticeVersion); err != nil {
		t.Fatal(err)
	}
	if rec := f.call(accessAdmin, codeForbidden, tokens[model.RoleAdmin]); rec.Code != http.StatusOK {
		t.Fatalf("after consent: %d %s", rec.Code, rec.Body.String())
	}
}

func TestGuardExpiry(t *testing.T) {
	t.Parallel()

	t.Run("idle", func(t *testing.T) {
		t.Parallel()
		f := newGuardFixture(t)
		u := storetest.SeedUser(t, f.st, model.RoleCustomer)
		token := f.session(u.ID)
		f.clk.Advance(719 * time.Minute)
		if rec := f.call(accessUser, "", token); rec.Code != http.StatusOK {
			t.Fatalf("719 idle minutes: %d", rec.Code)
		}
		f.clk.Advance(720 * time.Minute) // idle limit (default 720) reached since the touch
		rec := f.call(accessUser, "", token)
		expectRecError(t, rec, http.StatusUnauthorized, codeUnauthenticated)
		if !clearsCookie(rec, sessionCookiePlain) || len(f.sessions(u.ID)) != 0 {
			t.Fatal("expired session not deleted or cookie not cleared")
		}
		expectRecError(t, f.call(accessUser, "", token), http.StatusUnauthorized, codeUnauthenticated)
	})

	t.Run("absolute", func(t *testing.T) {
		t.Parallel()
		f := newGuardFixture(t)
		u := storetest.SeedUser(t, f.st, model.RoleCustomer)
		token := f.session(u.ID) // expires after 168 hours
		start := f.clk.Now()
		for f.clk.Now().Sub(start) < 160*time.Hour {
			f.clk.Advance(10 * time.Hour) // active use keeps the idle timer fresh
			if rec := f.call(accessUser, "", token); rec.Code != http.StatusOK {
				t.Fatalf("after %v: %d", f.clk.Now().Sub(start), rec.Code)
			}
		}
		f.clk.Set(start.Add(168 * time.Hour))
		expectRecError(t, f.call(accessUser, "", token), http.StatusUnauthorized, codeUnauthenticated)
		if len(f.sessions(u.ID)) != 0 {
			t.Fatal("session not deleted")
		}
	})

	t.Run("lowered settings apply to existing sessions", func(t *testing.T) {
		t.Parallel()
		f := newGuardFixture(t)
		u := storetest.SeedUser(t, f.st, model.RoleCustomer)
		token := f.session(u.ID)
		hours, idle := 1, 15
		if _, err := f.st.UpdateSettings(f.ctx, model.SettingsPatch{SessionAbsoluteHours: &hours}, 0); err != nil {
			t.Fatal(err)
		}
		f.clk.Advance(59 * time.Minute)
		if rec := f.call(accessUser, "", token); rec.Code != http.StatusOK {
			t.Fatalf("within the lowered absolute limit: %d", rec.Code)
		}
		f.clk.Advance(time.Minute)
		expectRecError(t, f.call(accessUser, "", token), http.StatusUnauthorized, codeUnauthenticated)

		token = f.session(u.ID)
		if _, err := f.st.UpdateSettings(f.ctx, model.SettingsPatch{SessionIdleMinutes: &idle}, 0); err != nil {
			t.Fatal(err)
		}
		f.clk.Advance(15 * time.Minute)
		expectRecError(t, f.call(accessUser, "", token), http.StatusUnauthorized, codeUnauthenticated)
	})

	t.Run("inactive user", func(t *testing.T) {
		t.Parallel()
		f := newGuardFixture(t)
		u := storetest.SeedUser(t, f.st, model.RoleCustomer)
		token := f.session(u.ID)
		off := false
		if _, err := f.st.UpdateUser(f.ctx, u.ID, model.UserPatch{Active: &off}); err != nil {
			t.Fatal(err)
		}
		rec := f.call(accessExempt, "", token)
		expectRecError(t, rec, http.StatusUnauthorized, codeUnauthenticated)
		if !clearsCookie(rec, sessionCookiePlain) || len(f.sessions(u.ID)) != 0 {
			t.Fatal("session of an inactive user not deleted")
		}
	})
}

func TestGuardTouch(t *testing.T) {
	t.Parallel()
	f := newGuardFixture(t)
	u := storetest.SeedUser(t, f.st, model.RoleCustomer)
	token := f.session(u.ID)
	created := f.sessions(u.ID)[0].LastSeenAt

	f.clk.Advance(59 * time.Second)
	f.call(accessUser, "", token)
	if got := f.sessions(u.ID)[0].LastSeenAt; !got.Equal(created) {
		t.Fatalf("touched within a minute: %v", got)
	}
	f.clk.Advance(time.Second)
	rec := f.call(accessUser, "", token)
	var id echoIdentity
	decodeRec(t, rec, &id)
	now := f.clk.Now()
	if got := f.sessions(u.ID)[0].LastSeenAt; !got.Equal(now) || !id.LastSeen.Equal(now) {
		t.Fatalf("last seen %v (handler %v), want %v", got, id.LastSeen, now)
	}
}

func TestGuardStoreFailures(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"SessionByToken", "Settings", "DeleteSessionByToken", "TouchSession", "Consents"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			f := newGuardFixture(t)
			u := storetest.SeedUser(t, f.st, model.RoleCustomer)
			token := f.session(u.ID)
			if method == "DeleteSessionByToken" {
				f.clk.Advance(24 * time.Hour) // idle expiry triggers the delete
			}
			if method == "TouchSession" {
				f.clk.Advance(time.Hour)
			}
			f.fail[method] = true
			expectRecError(t, f.call(accessUser, "", token), http.StatusInternalServerError, codeInternalError)
		})
	}
}

func TestCookieName(t *testing.T) {
	t.Parallel()
	if sessionCookieName(true) != "__Host-kxm_session" || sessionCookieName(false) != "kxm_session" {
		t.Fatal("cookie names")
	}
	// In secure mode only the __Host- cookie counts.
	f := newGuardFixture(t, func(d *Deps) { d.Config.SecureCookies = config.SecureCookiesTrue })
	u := storetest.SeedUser(t, f.st, model.RoleCustomer)
	token := f.session(u.ID)
	expectRecError(t, f.call(accessUser, "", token), http.StatusUnauthorized, codeUnauthenticated) // plain name
	r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/echo", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookieSecure, Value: token})
	rec := httptest.NewRecorder()
	f.s.guard(route{access: accessUser, handle: echoHandler}).ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("secure cookie: %d %s", rec.Code, rec.Body.String())
	}
}

func TestStartSession(t *testing.T) {
	t.Parallel()
	f := newGuardFixture(t, func(d *Deps) {
		d.Config.TrustedProxies = nil
	})
	u := storetest.SeedUser(t, f.st, model.RoleCustomer)
	old := f.session(u.ID)

	r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/auth/login", nil)
	r.RemoteAddr = "[::ffff:192.0.2.10]:4321"
	r.Header.Set("User-Agent", strings.Repeat("ü", 200)) // 400 bytes
	r.AddCookie(&http.Cookie{Name: sessionCookiePlain, Value: old})
	rec := httptest.NewRecorder()
	sess, err := f.s.startSession(f.ctx, rec, r, u)
	if err != nil {
		t.Fatal(err)
	}
	list := f.sessions(u.ID)
	if len(list) != 1 || list[0].ID != sess.ID {
		t.Fatalf("old session not rotated: %+v", list)
	}
	if sess.IP != "192.0.2.10" || len(sess.UserAgent) > model.SessionUserAgentMaxLen || !strings.HasPrefix(sess.UserAgent, "üü") {
		t.Fatalf("session ip %q ua %d bytes", sess.IP, len(sess.UserAgent))
	}
	if !sess.ExpiresAt.Equal(f.clk.Now().Add(168 * time.Hour)) {
		t.Fatalf("expires %v", sess.ExpiresAt)
	}
	ck := rec.Result().Cookies()
	if len(ck) != 1 {
		t.Fatalf("cookies %v", ck)
	}
	c := ck[0]
	if c.Name != sessionCookiePlain || c.Path != "/" || !c.HttpOnly || c.Secure || c.SameSite != http.SameSiteStrictMode ||
		c.MaxAge != 168*3600 || c.Domain != "" || len(c.Value) != 43 {
		t.Fatalf("cookie %+v", c)
	}
	// The new token authenticates.
	if rec := f.call(accessUser, "", c.Value); rec.Code != http.StatusOK {
		t.Fatalf("new token: %d", rec.Code)
	}

	for _, method := range []string{"DeleteSessionByToken", "Settings", "CreateSession"} {
		f.fail[method] = true
		if _, err := f.s.startSession(f.ctx, httptest.NewRecorder(), r, u); !errors.Is(err, errInjected) {
			t.Errorf("%s failure: %v", method, err)
		}
		f.fail[method] = false
	}
}

func TestSessionCookieAttributesSecure(t *testing.T) {
	t.Parallel()
	f := newGuardFixture(t, func(d *Deps) { d.Config.SecureCookies = config.SecureCookiesTrue })
	r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	rec := httptest.NewRecorder()
	f.s.setSessionCookie(rec, r, "tok", f.clk.Now().Add(-time.Second)) // already past: Max-Age 1
	f.s.clearSessionCookie(rec, r)
	ck := rec.Result().Cookies()
	if len(ck) != 2 {
		t.Fatalf("cookies %v", ck)
	}
	set, cleared := ck[0], ck[1]
	if set.Name != sessionCookieSecure || !set.Secure || set.MaxAge != 1 || !set.HttpOnly {
		t.Fatalf("set %+v", set)
	}
	if cleared.Name != sessionCookieSecure || !cleared.Secure || cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Fatalf("cleared %+v", cleared)
	}
	if !strings.Contains(rec.Header().Values("Set-Cookie")[1], "Max-Age=0") {
		t.Fatalf("clear header %q", rec.Header().Values("Set-Cookie")[1])
	}
}

func TestTruncateBytes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"abc", 5, "abc"},
		{"abcdef", 3, "abc"},
		{"aéb", 2, "a"},
		{"éé", 3, "é"},
		{"", 0, ""},
	}
	for _, c := range cases {
		if got := truncateBytes(c.in, c.n); got != c.want {
			t.Errorf("truncateBytes(%q, %d) = %q", c.in, c.n, got)
		}
	}
}

func TestSessionExpired(t *testing.T) {
	t.Parallel()
	now := testutil.T0
	set := model.DefaultSystemSettings()
	fresh := model.Session{CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(168 * time.Hour)}
	if sessionExpired(fresh, set, now) {
		t.Error("fresh session expired")
	}
	if !sessionExpired(fresh, set, now.Add(12*time.Hour)) {
		t.Error("idle limit not applied")
	}
	early := fresh
	early.ExpiresAt = now.Add(time.Minute)
	if !sessionExpired(early, set, now.Add(time.Minute)) {
		t.Error("stored expiry not applied")
	}
}
