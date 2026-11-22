package app

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"flag"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/store"
	"github.com/kaszasg/kinex-mark/internal/storetest"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

// testIterations is the PBKDF2 cost used by the tests (9.2.1).
const testIterations = 1000

// waitLimit is the deadlock guard of every channel wait in these tests.
const waitLimit = 15 * time.Second

func TestMain(m *testing.M) {
	hasherIterations = testIterations
	os.Exit(m.Run())
}

// logEntry is one captured log record.
type logEntry struct {
	level slog.Level
	msg   string
	attrs map[string]string
}

// logSink is a slog.Handler that keeps every record in memory and reports
// the address of the "server started" record on a channel.
type logSink struct {
	mu       sync.Mutex
	entries  []logEntry
	started  chan string
	jobsIdle chan struct{} // the scheduler finished an archive and purge run
}

func newLogSink() *logSink {
	return &logSink{started: make(chan string, 4), jobsIdle: make(chan struct{}, 1)}
}

func (s *logSink) Enabled(context.Context, slog.Level) bool { return true }

func (s *logSink) Handle(_ context.Context, r slog.Record) error {
	e := logEntry{level: r.Level, msg: r.Message, attrs: map[string]string{}}
	r.Attrs(func(a slog.Attr) bool {
		e.attrs[a.Key] = a.Value.String()
		return true
	})
	s.mu.Lock()
	s.entries = append(s.entries, e)
	s.mu.Unlock()
	switch r.Message {
	case "server started":
		s.started <- e.attrs["addr"]
	case "purge run", "purge run failed":
		select {
		case s.jobsIdle <- struct{}{}:
		default:
		}
	}
	return nil
}

func (s *logSink) WithAttrs([]slog.Attr) slog.Handler { return s }

func (s *logSink) WithGroup(string) slog.Handler { return s }

func (s *logSink) logger() *slog.Logger { return slog.New(s) }

// find returns the first record with the message.
func (s *logSink) find(msg string) (logEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if e.msg == msg {
			return e, true
		}
	}
	return logEntry{}, false
}

// mustFind is find that fails the test when the record is missing.
func (s *logSink) mustFind(t *testing.T, msg string) logEntry {
	t.Helper()
	e, ok := s.find(msg)
	if !ok {
		t.Fatalf("log record %q not found; log:\n%s", msg, s.text())
	}
	return e
}

// text renders every record, for leak checks and failure messages.
func (s *logSink) text() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b strings.Builder
	for _, e := range s.entries {
		b.WriteString(e.level.String() + " " + e.msg)
		for k, v := range e.attrs {
			b.WriteString(" " + k + "=" + v)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// loadConfig builds a validated configuration the way main does, on a
// loopback address with a free port.
func loadConfig(t *testing.T, dir string, env map[string]string, args ...string) config.Config {
	t.Helper()
	fs := flag.NewFlagSet("kinex-mark", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	all := append([]string{"-addr", "127.0.0.1:0", "-data", dir}, args...)
	cfg, err := config.Load(fs, all, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg
}

// running is a server started by startApp.
type running struct {
	addr string
	sink *logSink
	stop func() error
}

// startApp runs Run in the background and waits for the "server started"
// record. The server is stopped by cancelling Run's context, at the latest
// when the test ends.
//
// stop first waits until the startup archive and purge run has finished:
// a store query cancelled in mid-flight can leave the SQLite file handle
// open (reported in the B7 handoff), and the temporary directory could then
// not be removed on Windows.
func startApp(t *testing.T, cfg config.Config) *running {
	t.Helper()
	sink := newLogSink()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg, "test-version", sink.logger()) }()

	var once sync.Once
	var stopErr error
	stop := func() error {
		once.Do(func() {
			select {
			case <-sink.jobsIdle:
			case stopErr = <-done:
				cancel()
				if stopErr == nil {
					stopErr = errors.New("Run returned before it was stopped")
				}
				return
			case <-time.After(waitLimit):
				stopErr = errors.New("the startup archive and purge run did not finish")
			}
			cancel()
			select {
			case err := <-done:
				if stopErr == nil {
					stopErr = err
				}
			case <-time.After(waitLimit):
				stopErr = errors.New("Run did not return after cancel")
			}
		})
		return stopErr
	}
	t.Cleanup(func() { _ = stop() })

	select {
	case addr := <-sink.started:
		return &running{addr: addr, sink: sink, stop: stop}
	case err := <-done:
		cancel()
		t.Fatalf("Run returned before the server started: %v\nlog:\n%s", err, sink.text())
	case <-time.After(waitLimit):
		t.Fatalf("server did not start; log:\n%s", sink.text())
	}
	return nil
}

// openStore opens the data directory's database for assertions.
func openStore(t *testing.T, dir string) *store.Store {
	t.Helper()
	return storetest.NewStore(t, storetest.WithDir(dir))
}

// getStatus performs a GET and returns the status code.
func getStatus(t *testing.T, client *http.Client, url string) int {
	t.Helper()
	res, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	return res.StatusCode
}

// plainClient talks to the test server without any proxy.
func plainClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}}
}

// secretKey returns a valid APP_SECRET_KEY value filled with b.
func secretKey(b byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{b}, auth.KeyBytes))
}

// auditEntries returns the audit rows of one event.
func auditEntries(t *testing.T, st *store.Store, event string) []model.AuditEntry {
	t.Helper()
	rows, _, err := st.Audit(context.Background(), model.AuditFilter{Event: event}, model.PageReq{Limit: 50})
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	return rows
}

// verifyPassword reports whether password matches the user's stored hash.
func verifyPassword(t *testing.T, creds model.Credentials, password string) bool {
	t.Helper()
	ok, _, err := auth.NewHasher(testIterations, 1).Verify(creds.PasswordHash, password)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	return ok
}

func TestRunGeneratedAdminHealthAndShutdown(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "data")
	cfg := loadConfig(t, dir, nil)
	app := startApp(t, cfg)

	started := app.sink.mustFind(t, "server started")
	if started.attrs["version"] != "test-version" || started.attrs["tls"] != "false" || started.attrs["data"] != dir {
		t.Errorf("server started attrs = %v", started.attrs)
	}
	created := app.sink.mustFind(t, "secret key created")
	if created.attrs["path"] != filepath.Join(dir, config.SecretKeyFileName) {
		t.Errorf("secret key path = %q", created.attrs["path"])
	}
	if _, err := os.Stat(filepath.Join(dir, config.SecretKeyFileName)); err != nil {
		t.Errorf("key file: %v", err)
	}
	gen := app.sink.mustFind(t, "initial admin password generated")
	if gen.level != slog.LevelWarn || gen.attrs["username"] != config.DefaultAdminUser {
		t.Errorf("generated password record = %+v", gen)
	}
	password := gen.attrs["password"]
	if len(password) != generatedPasswordLength {
		t.Errorf("generated password length = %d, want %d", len(password), generatedPasswordLength)
	}

	if code := getStatus(t, plainClient(), "http://"+app.addr+"/healthz"); code != http.StatusOK {
		t.Errorf("GET /healthz = %d, want 200", code)
	}
	live := cfg
	live.Addr = app.addr
	if code := CheckHealth(live); code != 0 {
		t.Errorf("CheckHealth on a live server = %d, want 0", code)
	}

	if err := app.stop(); err != nil {
		t.Fatalf("Run after cancel: %v", err)
	}
	app.sink.mustFind(t, "shutting down")
	if code := CheckHealth(live); code != 1 {
		t.Errorf("CheckHealth on a stopped server = %d, want 1", code)
	}

	st := openStore(t, dir)
	ctx := context.Background()
	creds, err := st.Credentials(ctx, model.UsernameKey(config.DefaultAdminUser))
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if creds.User.Role != model.RoleAdmin || !creds.User.MustChangePassword || !creds.User.Active {
		t.Errorf("admin = %+v", creds.User)
	}
	if !verifyPassword(t, creds, password) {
		t.Error("the logged password does not match the stored hash")
	}
	rows := auditEntries(t, st, model.EventBootstrapAdminCreated)
	if len(rows) != 1 {
		t.Fatalf("bootstrap audit rows = %d, want 1", len(rows))
	}
	if rows[0].Details[model.DetailGeneratedPassword] != true || rows[0].Actor.ID != nil ||
		rows[0].Target == nil || rows[0].Target.ID == nil || *rows[0].Target.ID != creds.User.ID {
		t.Errorf("bootstrap audit = %+v", rows[0])
	}
	if kc, err := st.KeyCheck(ctx); err != nil || kc == "" {
		t.Errorf("KeyCheck = %q, %v; want a stored value", kc, err)
	}
}

func TestRunAdminFromEnvironmentPassword(t *testing.T) {
	t.Parallel()
	const password = "Correct-Horse-Battery-9"
	dir := t.TempDir()
	env := map[string]string{config.EnvAdminPassword: password}
	cfg := loadConfig(t, dir, env, "-admin-user", "Root.Admin")

	first := startApp(t, cfg)
	rec := first.sink.mustFind(t, "bootstrap admin created from APP_ADMIN_PASSWORD; remove the variable from the environment")
	if rec.level != slog.LevelWarn || rec.attrs["username"] != "Root.Admin" {
		t.Errorf("bootstrap record = %+v", rec)
	}
	if _, ok := first.sink.find("initial admin password generated"); ok {
		t.Error("a password was generated although APP_ADMIN_PASSWORD is set")
	}
	if err := first.stop(); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if strings.Contains(first.sink.text(), password) {
		t.Error("the environment password reached the log")
	}

	second := startApp(t, cfg)
	rec = second.sink.mustFind(t, "APP_ADMIN_PASSWORD is set but ignored on a non-empty database; remove it")
	if rec.level != slog.LevelWarn {
		t.Errorf("ignored-password record level = %v", rec.level)
	}
	if err := second.stop(); err != nil {
		t.Fatalf("second run: %v", err)
	}

	st := openStore(t, dir)
	ctx := context.Background()
	if n, err := st.CountUsers(ctx); err != nil || n != 1 {
		t.Fatalf("CountUsers = %d, %v; want 1", n, err)
	}
	creds, err := st.Credentials(ctx, model.UsernameKey("root.admin"))
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if creds.User.MustChangePassword || creds.User.Role != model.RoleAdmin {
		t.Errorf("admin = %+v", creds.User)
	}
	if !verifyPassword(t, creds, password) {
		t.Error("the environment password does not match the stored hash")
	}
	rows := auditEntries(t, st, model.EventBootstrapAdminCreated)
	if len(rows) != 1 || rows[0].Details[model.DetailGeneratedPassword] != false {
		t.Errorf("bootstrap audit = %+v", rows)
	}
}

func TestRunRejectsWeakEnvironmentPassword(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := loadConfig(t, dir, map[string]string{config.EnvAdminPassword: "short"})
	err := Run(context.Background(), cfg, "test", newLogSink().logger())
	if !errors.Is(err, errWeakAdminPassword) {
		t.Fatalf("Run = %v, want %v", err, errWeakAdminPassword)
	}
	if err.Error() != "bootstrap admin: password does not meet the policy" {
		t.Errorf("error text = %q", err.Error())
	}
	st := openStore(t, dir)
	if n, err := st.CountUsers(context.Background()); err != nil || n != 0 {
		t.Errorf("CountUsers = %d, %v; want 0", n, err)
	}
}

func TestRunStopsOnKeyMismatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	app := startApp(t, loadConfig(t, dir, nil))
	if err := app.stop(); err != nil {
		t.Fatalf("first run: %v", err)
	}

	cfg := loadConfig(t, dir, map[string]string{config.EnvSecretKey: secretKey(7)})
	err := Run(context.Background(), cfg, "test", newLogSink().logger())
	if !errors.Is(err, errKeyMismatch) {
		t.Fatalf("Run with another key = %v, want %v", err, errKeyMismatch)
	}
	if !strings.Contains(err.Error(), "restore the original key file or APP_SECRET_KEY") {
		t.Errorf("error text = %q", err.Error())
	}
	if _, err := os.Stat(filepath.Join(dir, config.SecretKeyFileName)); err != nil {
		t.Errorf("the original key file must stay: %v", err)
	}
}

func TestRunRemovesFreshKeyFileOnMismatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	app := startApp(t, loadConfig(t, dir, map[string]string{config.EnvSecretKey: secretKey(1)}))
	if err := app.stop(); err != nil {
		t.Fatalf("first run: %v", err)
	}
	keyFile := filepath.Join(dir, config.SecretKeyFileName)
	if _, err := os.Stat(keyFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("APP_SECRET_KEY must not write a key file: %v", err)
	}

	sink := newLogSink()
	err := Run(context.Background(), loadConfig(t, dir, nil), "test", sink.logger())
	if !errors.Is(err, errKeyMismatch) {
		t.Fatalf("Run with a new key file = %v, want %v", err, errKeyMismatch)
	}
	if _, err := os.Stat(keyFile); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the freshly generated key file must be removed: %v", err)
	}
	sink.mustFind(t, "secret key removed")
}

func TestRunTLS(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	certFile, keyFile := writeSelfSignedCert(t, t.TempDir())
	cfg := loadConfig(t, dir, nil, "-tls-cert", certFile, "-tls-key", keyFile)
	app := startApp(t, cfg)
	if got := app.sink.mustFind(t, "server started").attrs["tls"]; got != "true" {
		t.Errorf("tls attr = %q, want true", got)
	}

	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		Proxy:           nil,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed test certificate
	}}
	if code := getStatus(t, client, "https://"+app.addr+"/healthz"); code != http.StatusOK {
		t.Errorf("GET https /healthz = %d, want 200", code)
	}
	res, err := client.Get("https://" + app.addr + "/healthz")
	if err == nil {
		if v := res.TLS.Version; v < tls.VersionTLS12 {
			t.Errorf("TLS version = %x", v)
		}
		res.Body.Close()
	}
	old := &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11} //nolint:gosec // probing the minimum version
	if conn, err := tls.Dial("tcp", app.addr, old); err == nil {
		conn.Close()
		t.Error("a TLS 1.1 handshake succeeded")
	}

	live := cfg
	live.Addr = app.addr
	if code := CheckHealth(live); code != 0 {
		t.Errorf("CheckHealth over TLS = %d, want 0", code)
	}
	if err := app.stop(); err != nil {
		t.Fatalf("Run after cancel: %v", err)
	}
}

func TestRunStartErrors(t *testing.T) {
	t.Parallel()

	t.Run("data dir is a file", func(t *testing.T) {
		t.Parallel()
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := loadConfig(t, file, nil)
		err := Run(context.Background(), cfg, "test", newLogSink().logger())
		if err == nil || !strings.HasPrefix(err.Error(), "create data dir: ") {
			t.Errorf("Run = %v, want a create data dir error", err)
		}
	})

	t.Run("invalid secret key", func(t *testing.T) {
		t.Parallel()
		cfg := loadConfig(t, t.TempDir(), nil)
		cfg.SecretKey = "not base64!"
		err := Run(context.Background(), cfg, "test", newLogSink().logger())
		if err == nil || !strings.HasPrefix(err.Error(), "load secret key: ") || strings.Contains(err.Error(), "not base64!") {
			t.Errorf("Run = %v, want a load secret key error without the value", err)
		}
	})

	t.Run("database path is a directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, store.DBFileName), 0o750); err != nil {
			t.Fatal(err)
		}
		err := Run(context.Background(), loadConfig(t, dir, nil), "test", newLogSink().logger())
		if err == nil || !strings.HasPrefix(err.Error(), "open store: ") {
			t.Errorf("Run = %v, want an open store error", err)
		}
	})

	t.Run("busy port", func(t *testing.T) {
		t.Parallel()
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		cfg := loadConfig(t, t.TempDir(), nil, "-addr", ln.Addr().String())
		sink := newLogSink()
		err = Run(context.Background(), cfg, "test", sink.logger())
		if err == nil || !strings.HasPrefix(err.Error(), "listen: ") {
			t.Errorf("Run = %v, want a listen error", err)
		}
		for _, msg := range []string{"server started", "archive run", "archive run failed"} {
			if _, ok := sink.find(msg); ok {
				t.Errorf("%q logged although the port was busy", msg)
			}
		}
	})

	t.Run("unreadable certificate", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		cert := filepath.Join(tmp, "cert.pem")
		key := filepath.Join(tmp, "key.pem")
		for _, f := range []string{cert, key} {
			if err := os.WriteFile(f, []byte("not a pem file"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		cfg := loadConfig(t, t.TempDir(), nil, "-tls-cert", cert, "-tls-key", key)
		err := Run(context.Background(), cfg, "test", newLogSink().logger())
		if err == nil || !strings.HasPrefix(err.Error(), "load tls certificate: ") {
			t.Errorf("Run = %v, want a certificate error", err)
		}
	})
}

// seedLockedUser creates a customer with a throttle row and returns it.
func seedLockedUser(t *testing.T, st *store.Store) (model.User, string) {
	t.Helper()
	u := storetest.SeedUser(t, st, model.RoleCustomer)
	key := "u:" + strings.Repeat("a", 64)
	locked := testutil.T0.Add(time.Hour)
	err := st.ThrottleUpdate(context.Background(), []model.ThrottleKey{{Key: key, UserID: &u.ID}},
		func(row model.ThrottleRow) (model.ThrottleRow, error) {
			row.Failures = 5
			row.LockedUntil = &locked
			row.Lockouts = 1
			return row, nil
		})
	if err != nil {
		t.Fatalf("ThrottleUpdate: %v", err)
	}
	return u, key
}

// throttleFailures reads the failure count of a throttle key (0 for a fresh
// row, that is, when the row was deleted).
func throttleFailures(t *testing.T, st *store.Store, key string, userID int64) (int, bool) {
	t.Helper()
	var failures int
	var locked bool
	err := st.ThrottleUpdate(context.Background(), []model.ThrottleKey{{Key: key, UserID: &userID}},
		func(row model.ThrottleRow) (model.ThrottleRow, error) {
			failures = row.Failures
			locked = row.LockedUntil != nil
			return row, nil
		})
	if err != nil {
		t.Fatalf("ThrottleUpdate: %v", err)
	}
	return failures, locked
}

// assertCLIAudit checks the single audit row of a recovery command.
func assertCLIAudit(t *testing.T, st *store.Store, event string, u model.User) {
	t.Helper()
	rows := auditEntries(t, st, event)
	if len(rows) != 1 {
		t.Fatalf("%s rows = %d, want 1", event, len(rows))
	}
	r := rows[0]
	if r.Actor.ID != nil || r.Actor.Username != "" || !r.Success {
		t.Errorf("%s actor = %+v, success %v", event, r.Actor, r.Success)
	}
	if r.Target == nil || r.Target.Type != model.TargetUser || r.Target.ID == nil || *r.Target.ID != u.ID {
		t.Errorf("%s target = %+v", event, r.Target)
	}
	if len(r.Details) != 1 || r.Details[model.DetailVia] != model.ViaCLI {
		t.Errorf("%s details = %v, want {via: cli}", event, r.Details)
	}
}

func TestRunAdminCommandUnlockUser(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st := openStore(t, dir)
	u, key := seedLockedUser(t, st)
	if n, locked := throttleFailures(t, st, key, u.ID); n != 5 || !locked {
		t.Fatalf("seeded throttle row = %d, locked %v", n, locked)
	}

	cfg := loadConfig(t, dir, nil, "-unlock-user", strings.ToUpper(u.Username))
	var out bytes.Buffer
	sink := newLogSink()
	if code := RunAdminCommand(context.Background(), cfg, &out, sink.logger()); code != 0 {
		t.Fatalf("RunAdminCommand = %d, want 0; log:\n%s", code, sink.text())
	}
	if want := "Sign-in lock cleared for user " + u.Username + ".\n"; out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
	if n, locked := throttleFailures(t, st, key, u.ID); n != 0 || locked {
		t.Errorf("throttle row after unlock = %d failures, locked %v; want a fresh row", n, locked)
	}
	assertCLIAudit(t, st, model.EventUserUnlocked, u)
	if _, ok := sink.find("server started"); ok {
		t.Error("an admin command must not start the server")
	}
}

func TestRunAdminCommandResetPassword(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st := openStore(t, dir)
	ctx := context.Background()
	u, key := seedLockedUser(t, st)
	for i := range 2 {
		_, err := st.CreateSession(ctx, model.NewSession{
			TokenHash: bytes.Repeat([]byte{byte(i + 1)}, 32),
			UserID:    u.ID,
			ExpiresAt: testutil.T0.Add(time.Hour),
			IP:        "127.0.0.1",
			UserAgent: "test",
		})
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
	}
	if s, err := st.Sessions(ctx, u.ID); err != nil || len(s) != 2 {
		t.Fatalf("seeded sessions = %d, %v", len(s), err)
	}

	cfg := loadConfig(t, dir, nil, "-reset-password", u.Username)
	var out bytes.Buffer
	sink := newLogSink()
	if code := RunAdminCommand(ctx, cfg, &out, sink.logger()); code != 0 {
		t.Fatalf("RunAdminCommand = %d, want 0; log:\n%s", code, sink.text())
	}
	line, rest, _ := strings.Cut(out.String(), "\n")
	prefix := "New password for user " + u.Username + ": "
	if !strings.HasPrefix(line, prefix) || rest != "The password must be changed at the next sign-in.\n" {
		t.Fatalf("output = %q", out.String())
	}
	password := strings.TrimPrefix(line, prefix)
	if len(password) != generatedPasswordLength {
		t.Errorf("password length = %d, want %d", len(password), generatedPasswordLength)
	}
	if strings.Contains(sink.text(), password) {
		t.Error("the new password reached the log")
	}

	creds, err := st.Credentials(ctx, u.UsernameKey)
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if !creds.User.MustChangePassword {
		t.Error("must_change_password is not set")
	}
	if !verifyPassword(t, creds, password) {
		t.Error("the printed password does not match the stored hash")
	}
	if s, err := st.Sessions(ctx, u.ID); err != nil || len(s) != 0 {
		t.Errorf("sessions after reset = %d, %v; want 0", len(s), err)
	}
	if n, locked := throttleFailures(t, st, key, u.ID); n != 0 || locked {
		t.Errorf("throttle row after reset = %d failures, locked %v", n, locked)
	}
	assertCLIAudit(t, st, model.EventUserPasswordReset, u)
}

func TestRunAdminCommandFailures(t *testing.T) {
	t.Parallel()

	t.Run("unknown user", func(t *testing.T) {
		t.Parallel()
		cfg := loadConfig(t, t.TempDir(), nil, "-reset-password", "nobody")
		var out bytes.Buffer
		sink := newLogSink()
		if code := RunAdminCommand(context.Background(), cfg, &out, sink.logger()); code != 1 {
			t.Fatalf("RunAdminCommand = %d, want 1", code)
		}
		rec := sink.mustFind(t, adminCommandFailed)
		if rec.attrs["err"] != errUnknownUser.Error() || rec.attrs["command"] != config.AdminCommandResetPassword {
			t.Errorf("failure record = %+v", rec)
		}
		if out.Len() != 0 {
			t.Errorf("output = %q, want none", out.String())
		}
	})

	t.Run("no command", func(t *testing.T) {
		t.Parallel()
		cfg := loadConfig(t, t.TempDir(), nil)
		sink := newLogSink()
		if code := RunAdminCommand(context.Background(), cfg, io.Discard, sink.logger()); code != 1 {
			t.Fatalf("RunAdminCommand = %d, want 1", code)
		}
		sink.mustFind(t, adminCommandFailed)
	})

	t.Run("data dir is a file", func(t *testing.T) {
		t.Parallel()
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := loadConfig(t, file, nil, "-unlock-user", "admin")
		sink := newLogSink()
		if code := RunAdminCommand(context.Background(), cfg, io.Discard, sink.logger()); code != 1 {
			t.Fatalf("RunAdminCommand = %d, want 1", code)
		}
		if rec := sink.mustFind(t, adminCommandFailed); !strings.HasPrefix(rec.attrs["err"], "create data dir: ") {
			t.Errorf("failure record = %+v", rec)
		}
	})
}

// writeSelfSignedCert writes a loopback certificate and its key as PEM.
func writeSelfSignedCert(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}
