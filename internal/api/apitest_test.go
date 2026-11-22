package api

// Shared HTTP test harness of the api package (plan 9.2.3). Every HTTP
// test file of the package (B5 and B6) builds on these helpers instead of
// defining its own server, client or assertions.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/store"
	"github.com/kaszasg/kinex-mark/internal/storetest"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

// Test UI (fstest.MapFS): an index.html with two inline scripts (the theme
// bootstrap and a SvelteKit-like start script) plus one external script,
// an immutable asset, a font and robots.txt.
const (
	testInlineScript1 = "\n\t\t\ttry {\n\t\t\t\tlocalStorage.getItem('mode-watcher-mode') || localStorage.setItem('mode-watcher-mode', 'dark');\n\t\t\t} catch {}\n\t\t"
	testInlineScript2 = "\n\t\t\t\t{\n\t\t\t\t\t__sveltekit_test = { base: \"\" };\n\t\t\t\t\timport(\"/_app/immutable/x.js\").then((kit) => kit.start());\n\t\t\t\t}\n\t\t\t"
	testIndexHTML     = "<!doctype html>\n<html lang=\"hu\" class=\"dark\">\n\t<head>\n\t\t<meta charset=\"utf-8\" />\n\t\t<script>" +
		testInlineScript1 + "</script>\n\t\t<script type=\"module\" src=\"/_app/immutable/x.js\"></script>\n\t</head>\n\t<body>\n\t\t<div style=\"display: contents\">\n\t\t\t<script>" +
		testInlineScript2 + "</script>\n\t\t</div>\n\t</body>\n</html>\n"
	testAssetJS   = "export const start = () => {};\n"
	testFontBytes = "wOF2-test-font"
	testRobotsTxt = "User-agent: *\nDisallow: /\n"
)

// testUI returns a fresh copy of the test UI file system.
func testUI() fstest.MapFS {
	return fstest.MapFS{
		"index.html":                     {Data: []byte(testIndexHTML)},
		"_app/immutable/x.js":            {Data: []byte(testAssetJS)},
		"fonts/HankenGrotesk-Test.woff2": {Data: []byte(testFontBytes)},
		"robots.txt":                     {Data: []byte(testRobotsTxt)},
		"favicon.svg":                    {Data: []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>")},
	}
}

// testKey is the fixed 32-byte server key of the test keyring.
var testKey = []byte("kinex-mark-test-key-0123456789ab")

// testHasher is the cheap password hasher of the tests (9.2.1).
var testHasher = auth.NewHasher(1000, 4)

// newTestKeyring returns the keyring of testKey.
func newTestKeyring(t testing.TB) *auth.Keyring {
	t.Helper()
	kr, err := auth.NewKeyring(testKey)
	if err != nil {
		t.Fatalf("keyring: %v", err)
	}
	return kr
}

// fakeJobs implements Jobs and records the calls.
type fakeJobs struct {
	mu           sync.Mutex
	ids          []int64 // Enqueue arguments in call order
	full         bool    // Enqueue reports a full queue
	archiveCalls int
	archiveCount int   // RunArchive result
	archiveErr   error // RunArchive error
}

// Enqueue records the link id; it returns false while full is set.
func (f *fakeJobs) Enqueue(linkID int64) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ids = append(f.ids, linkID)
	return !f.full
}

// RunArchive counts the call and returns archiveCount, archiveErr.
func (f *fakeJobs) RunArchive(context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.archiveCalls++
	return f.archiveCount, f.archiveErr
}

// enqueued returns a copy of the recorded Enqueue ids.
func (f *fakeJobs) enqueued() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int64(nil), f.ids...)
}

// setFull switches the simulated queue state.
func (f *fakeJobs) setFull(full bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.full = full
}

// archiveRuns returns how often RunArchive was called.
func (f *fakeJobs) archiveRuns() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.archiveCalls
}

// logBuffer is a goroutine-safe log sink.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends p.
func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns everything logged so far.
func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// count returns how many logged lines contain substr.
func (b *logBuffer) count(substr string) int {
	n := 0
	for _, line := range strings.Split(b.String(), "\n") {
		if strings.Contains(line, substr) {
			n++
		}
	}
	return n
}

// testEnv is one API server over a real store in t.TempDir().
type testEnv struct {
	t       *testing.T
	clock   *clock.Fake
	store   *store.Store
	jobs    *fakeJobs
	srv     *httptest.Server // api.New(Deps{...}) with the testUI file system
	handler http.Handler     // the same handler, for direct ServeHTTP calls
	cfg     config.Config
	deps    Deps
	logs    *logBuffer // every log line of the server, Debug level included
	tls     bool
}

// dbPath returns the database file of the env's store (cfg.DataDir/app.db).
func (e *testEnv) dbPath() string { return filepath.Join(e.cfg.DataDir, store.DBFileName) }

// envSettings collects the options of newEnv.
type envSettings struct {
	cfg     config.Config
	tls     bool
	ui      fstest.MapFS
	version string
	mutate  []func(*Deps)
}

// envOption configures newEnv.
type envOption func(*envSettings)

// withBaseURL sets APP_BASE_URL (for example "https://mark.example.com").
func withBaseURL(raw string) envOption {
	return func(s *envSettings) {
		u, err := url.Parse(raw)
		if err != nil {
			panic(err)
		}
		s.cfg.BaseURL = &url.URL{Scheme: u.Scheme, Host: u.Host}
	}
}

// withTrustedProxy sets APP_TRUSTED_PROXY from IPs or CIDRs. The test
// client connects from 127.0.0.1, so withTrustedProxy("127.0.0.1") makes it
// a trusted proxy.
func withTrustedProxy(entries ...string) envOption {
	return func(s *envSettings) {
		for _, e := range entries {
			var p netip.Prefix
			if strings.Contains(e, "/") {
				p = netip.MustParsePrefix(e)
			} else {
				a := netip.MustParseAddr(e)
				p = netip.PrefixFrom(a, a.BitLen())
			}
			s.cfg.TrustedProxies = append(s.cfg.TrustedProxies, p)
		}
	}
}

// withAddr sets the configured listen address (APP_ADDR) used by the host
// check. The default is "127.0.0.1:0", a loopback listener.
func withAddr(addr string) envOption {
	return func(s *envSettings) { s.cfg.Addr = addr }
}

// withSecureCookies sets APP_SECURE_COOKIES.
func withSecureCookies(mode config.SecureCookies) envOption {
	return func(s *envSettings) { s.cfg.SecureCookies = mode }
}

// withTLS serves the test server over TLS (httptest.NewTLSServer).
func withTLS() envOption {
	return func(s *envSettings) { s.tls = true }
}

// withConfig edits the configuration directly.
func withConfig(fn func(*config.Config)) envOption {
	return func(s *envSettings) { fn(&s.cfg) }
}

// withUI replaces the test UI file system.
func withUI(ui fstest.MapFS) envOption {
	return func(s *envSettings) { s.ui = ui }
}

// withDeps edits the dependencies just before api.New (for example to wrap
// the store with a failing fake).
func withDeps(fn func(*Deps)) envOption {
	return func(s *envSettings) { s.mutate = append(s.mutate, fn) }
}

// newEnv starts an API server on a random loopback port over a fresh store
// with a fake clock at testutil.T0. Everything is closed when t ends.
func newEnv(t *testing.T, opts ...envOption) *testEnv {
	t.Helper()
	set := envSettings{
		cfg: config.Config{
			Addr:          "127.0.0.1:0",
			DataDir:       t.TempDir(),
			SecureCookies: config.SecureCookiesAuto,
			LogLevel:      slog.LevelDebug,
			LogFormat:     "text",
			FetchWorkers:  config.DefaultFetchWorkers,
			FetchTimeout:  config.DefaultFetchTimeout,
			AdminUser:     config.DefaultAdminUser,
		},
		ui:      testUI(),
		version: "test",
	}
	for _, o := range opts {
		o(&set)
	}
	clk := testutil.NewClock()
	// The store lives in cfg.DataDir, so tests can inspect <DataDir>/app.db.
	st := storetest.NewStore(t, storetest.WithClock(clk), storetest.WithDir(set.cfg.DataDir))
	logs := &logBuffer{}
	e := &testEnv{
		t:     t,
		clock: clk,
		store: st,
		jobs:  &fakeJobs{},
		cfg:   set.cfg,
		logs:  logs,
		tls:   set.tls,
	}
	e.deps = Deps{
		Store:   st,
		Jobs:    e.jobs,
		Hasher:  testHasher,
		Keyring: newTestKeyring(t),
		Clock:   clk,
		Log:     slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Config:  set.cfg,
		Version: set.version,
		UI:      set.ui,
	}
	for _, fn := range set.mutate {
		fn(&e.deps)
	}
	h, err := New(e.deps)
	if err != nil {
		t.Fatalf("api.New: %v", err)
	}
	e.handler = h
	if set.tls {
		e.srv = httptest.NewTLSServer(h)
	} else {
		e.srv = httptest.NewServer(h)
	}
	t.Cleanup(e.srv.Close)
	return e
}

// ctx returns a context for direct store calls in tests.
func (e *testEnv) ctx() context.Context { return context.Background() }

// secure reports whether the server answers the default test client in
// secure mode (6.2), which decides the session cookie name.
func (e *testEnv) secure() bool {
	switch e.cfg.SecureCookies {
	case config.SecureCookiesTrue:
		return true
	case config.SecureCookiesFalse:
		return false
	}
	return e.tls || (e.cfg.BaseURL != nil && e.cfg.BaseURL.Scheme == "https")
}

// cookieName is the session cookie name the server uses for the default
// test client.
func (e *testEnv) cookieName() string { return sessionCookieName(e.secure()) }

// url returns the absolute URL of path on the test server.
func (e *testEnv) url(path string) string { return e.srv.URL + path }

// serve runs req through the handler without the network, so the test
// controls RemoteAddr, Host and TLS.
func (e *testEnv) serve(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

// settings returns the current system settings.
func (e *testEnv) settings(t *testing.T) model.SystemSettings {
	t.Helper()
	s, err := e.store.Settings(e.ctx())
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	return s
}

// seedUser creates an active user of role who passes both gates
// (storetest.SeedUser). Its password hash is not valid, so it cannot sign
// in with a password; use seedPasswordUser for that.
func (e *testEnv) seedUser(t *testing.T, role model.Role) model.User {
	t.Helper()
	return storetest.SeedUser(t, e.store, role)
}

// seedPasswordUser creates an active user of role with a real hash of
// password who has acknowledged the current privacy notice.
func (e *testEnv) seedPasswordUser(t *testing.T, username string, role model.Role, password string) model.User {
	t.Helper()
	hash, err := testHasher.Hash(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := storetest.SeedNewUser(t, e.store, model.NewUser{Username: username, Role: role, PasswordHash: hash})
	if err := e.store.RecordConsent(e.ctx(), u.ID, model.ConsentPrivacyNotice, e.settings(t).PrivacyNoticeVersion); err != nil {
		t.Fatalf("record consent: %v", err)
	}
	return u
}

// apiClient is an HTTP client of the test server with its own cookie jar.
// By default every request carries X-Kinex-Request: 1, Accept:
// application/json and the server's Origin, like the frontend (5.1).
type apiClient struct {
	t      *testing.T
	env    *testEnv
	http   *http.Client
	header http.Header
}

// client returns an anonymous client.
func (e *testEnv) client(t *testing.T) *apiClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	// srv.Client() returns the same *http.Client on every call: copy it, so
	// that every apiClient keeps its own cookie jar.
	hc := *e.srv.Client()
	hc.Jar = jar
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	h := http.Header{}
	h.Set(headerKinexRequest, "1")
	h.Set("Accept", "application/json")
	h.Set("Origin", e.srv.URL)
	return &apiClient{t: t, env: e, http: &hc, header: h}
}

// loginAs seeds a user of role (storetest.SeedUser: active, consent given)
// and returns a client signed in as that user. The session is created
// directly with store.CreateSession and its cookie set by hand, so the
// helper does not depend on the sign-in handler (11.5).
func (e *testEnv) loginAs(t *testing.T, role model.Role) (*apiClient, model.User) {
	t.Helper()
	u := e.seedUser(t, role)
	return e.loginUser(t, u), u
}

// loginUser returns a client signed in as an existing user (for example one
// from storetest.SeedNewUser, which has not passed the gates).
func (e *testEnv) loginUser(t *testing.T, u model.User) *apiClient {
	t.Helper()
	c := e.client(t)
	token, _ := e.createSession(t, u.ID)
	c.setSessionToken(token)
	return c
}

// createSession stores a new session of the user, valid for the current
// sessionAbsoluteHours, and returns its token and row.
func (e *testEnv) createSession(t *testing.T, userID int64) (string, model.Session) {
	t.Helper()
	token, hash, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	hours := e.settings(t).SessionAbsoluteHours
	sess, err := e.store.CreateSession(e.ctx(), model.NewSession{
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: e.clock.Now().Add(time.Duration(hours) * time.Hour),
		IP:        "127.0.0.1",
		UserAgent: "apitest",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return token, sess
}

// setSessionToken puts a session token into the client's cookie jar under
// the server's cookie name (without the Secure flag, so the jar sends it
// over plain HTTP too).
func (c *apiClient) setSessionToken(token string) {
	u, err := url.Parse(c.env.srv.URL)
	if err != nil {
		c.t.Fatal(err)
	}
	c.http.Jar.SetCookies(u, []*http.Cookie{{Name: c.env.cookieName(), Value: token, Path: "/"}})
}

// sessionToken returns the session token in the client's jar ("" if none).
func (c *apiClient) sessionToken() string {
	u, err := url.Parse(c.env.srv.URL)
	if err != nil {
		c.t.Fatal(err)
	}
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == c.env.cookieName() {
			return ck.Value
		}
	}
	return ""
}

// withHeader returns a copy of the client (same cookie jar) whose requests
// carry the header; an empty value removes a default header. "Host" sets
// the request's Host.
func (c *apiClient) withHeader(key, value string) *apiClient {
	cp := *c
	cp.header = c.header.Clone()
	if value == "" {
		cp.header.Del(key)
	} else {
		cp.header.Set(key, value)
	}
	return &cp
}

// do sends a request and returns the response with the body fully read, so
// callers never need to close it and may read it more than once through
// readBody, decode or expectError. body: nil sends none; []byte, string
// and io.Reader are sent as is; anything else is JSON-encoded. A body gets
// Content-Type: application/json unless the client sets one.
func (c *apiClient) do(method, path string, body any) *http.Response {
	c.t.Helper()
	return c.doRequest(method, path, body, nil)
}

// doRequest is do with a hook that may edit the request before it is sent.
func (c *apiClient) doRequest(method, path string, body any, edit func(*http.Request)) *http.Response {
	c.t.Helper()
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case []byte:
		rd = bytes.NewReader(b)
	case string:
		rd = strings.NewReader(b)
	case io.Reader:
		rd = b
	default:
		data, err := json.Marshal(b)
		if err != nil {
			c.t.Fatalf("encode request body: %v", err)
		}
		rd = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.env.url(path), rd)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	for k, vs := range c.header {
		if http.CanonicalHeaderKey(k) == "Host" {
			req.Host = vs[0]
			continue
		}
		req.Header[k] = append([]string(nil), vs...)
	}
	if rd != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if edit != nil {
		edit(req)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	data, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		c.t.Fatalf("%s %s: read body: %v", method, path, err)
	}
	res.Body = io.NopCloser(bytes.NewReader(data))
	return res
}

// get, post, put, patch and del are shorthands of do.
func (c *apiClient) get(path string) *http.Response {
	c.t.Helper()
	return c.do(http.MethodGet, path, nil)
}

func (c *apiClient) post(path string, body any) *http.Response {
	c.t.Helper()
	return c.do(http.MethodPost, path, body)
}

func (c *apiClient) put(path string, body any) *http.Response {
	c.t.Helper()
	return c.do(http.MethodPut, path, body)
}

func (c *apiClient) patch(path string, body any) *http.Response {
	c.t.Helper()
	return c.do(http.MethodPatch, path, body)
}

func (c *apiClient) del(path string) *http.Response {
	c.t.Helper()
	return c.do(http.MethodDelete, path, nil)
}

// readBody returns the response body; it can be called repeatedly on a
// response from do.
func readBody(t *testing.T, res *http.Response) []byte {
	t.Helper()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	res.Body = io.NopCloser(bytes.NewReader(data))
	return data
}

// expectStatus fails the test unless res has the status; it returns the body.
func expectStatus(t *testing.T, res *http.Response, status int) []byte {
	t.Helper()
	body := readBody(t, res)
	if res.StatusCode != status {
		t.Fatalf("%s %s: status %d, want %d; body: %s", res.Request.Method, res.Request.URL.Path, res.StatusCode, status, body)
	}
	return body
}

// expectJSONContentType fails unless the response is JSON (5.1).
func expectJSONContentType(t *testing.T, res *http.Response) {
	t.Helper()
	if ct := res.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("%s %s: Content-Type %q, want application/json; charset=utf-8", res.Request.Method, res.Request.URL.Path, ct)
	}
}

// decode decodes a JSON response body into T. Unknown fields are allowed;
// use assertJSON for an exact comparison.
func decode[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	var v T
	body := readBody(t, res)
	expectJSONContentType(t, res)
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("decode %T from %q: %v", v, body, err)
	}
	return v
}

// decodeStatus checks the status, then decodes the body into T.
func decodeStatus[T any](t *testing.T, res *http.Response, status int) T {
	t.Helper()
	expectStatus(t, res, status)
	return decode[T](t, res)
}

// decodeInto decodes a JSON document into dst.
func decodeInto(t *testing.T, data []byte, dst any) {
	t.Helper()
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatalf("decode %q: %v", data, err)
	}
}

// decodeRec decodes the JSON body of a recorded response into dst.
func decodeRec(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
}

// expectError checks status and error code of an error response (5.2) and
// returns the decoded body for checks of fields and details.
func expectError(t *testing.T, res *http.Response, status int, code string) errorBody {
	t.Helper()
	expectStatus(t, res, status)
	body := decode[errorBody](t, res)
	if body.Code != code {
		t.Fatalf("%s %s: code %q, want %q (error %q)", res.Request.Method, res.Request.URL.Path, body.Code, code, body.Error)
	}
	if body.Error == "" {
		t.Fatalf("%s %s: empty error message", res.Request.Method, res.Request.URL.Path)
	}
	return body
}

// expectFields checks a 400 validation_failed response with exactly the
// given field codes.
func expectFields(t *testing.T, res *http.Response, fields map[string]string) {
	t.Helper()
	body := expectError(t, res, http.StatusBadRequest, codeValidationFailed)
	if !reflect.DeepEqual(body.Fields, fields) {
		t.Fatalf("fields %v, want %v", body.Fields, fields)
	}
}

// assertJSON compares a JSON document with want (JSON text) semantically:
// key order and white space do not matter, every value must match exactly.
func assertJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("decode got %q: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("decode want %q: %v", want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("JSON mismatch\n got: %s\nwant: %s", got, want)
	}
}

// jsonKeys returns the sorted top-level keys of a JSON object, for shape
// checks against the 5.5 types.
func jsonKeys(t *testing.T, body []byte) []string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decode object %q: %v", body, err)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// responseCookie returns the Set-Cookie of name in res, or nil.
func responseCookie(res *http.Response, name string) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}
