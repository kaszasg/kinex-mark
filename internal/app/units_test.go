package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/fetch"
	"github.com/kaszasg/kinex-mark/internal/model"
)

var errBoom = errors.New("boom")

func testKeyring(t *testing.T, b byte) *auth.Keyring {
	t.Helper()
	kr, err := auth.NewKeyring(bytes.Repeat([]byte{b}, auth.KeyBytes))
	if err != nil {
		t.Fatal(err)
	}
	return kr
}

// fakeKeyCheck is a keyCheckStore with injectable failures.
type fakeKeyCheck struct {
	stored          string
	readErr, setErr error
}

func (f *fakeKeyCheck) KeyCheck(context.Context) (string, error) { return f.stored, f.readErr }

func (f *fakeKeyCheck) SetKeyCheck(_ context.Context, v string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.stored = v
	return nil
}

func TestVerifyKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	kr := testKeyring(t, 1)

	empty := &fakeKeyCheck{}
	if err := verifyKey(ctx, empty, kr); err != nil || empty.stored != kr.CheckValue() {
		t.Errorf("first use: err %v, stored %q", err, empty.stored)
	}
	if err := verifyKey(ctx, empty, kr); err != nil {
		t.Errorf("same key: %v", err)
	}
	if err := verifyKey(ctx, empty, testKeyring(t, 2)); !errors.Is(err, errKeyMismatch) {
		t.Errorf("other key: %v, want %v", err, errKeyMismatch)
	}
	if err := verifyKey(ctx, &fakeKeyCheck{readErr: errBoom}, kr); !errors.Is(err, errBoom) || !strings.HasPrefix(err.Error(), "read key check: ") {
		t.Errorf("read failure: %v", err)
	}
	if err := verifyKey(ctx, &fakeKeyCheck{setErr: errBoom}, kr); !errors.Is(err, errBoom) || !strings.HasPrefix(err.Error(), "store key check: ") {
		t.Errorf("write failure: %v", err)
	}
}

// fakeClosing is a closingStore with injectable failures.
type fakeClosing struct {
	pingErr, closeErr error
	calls             []string
}

func (f *fakeClosing) Ping(context.Context) error {
	f.calls = append(f.calls, "ping")
	return f.pingErr
}

func (f *fakeClosing) Close() error {
	f.calls = append(f.calls, "close")
	return f.closeErr
}

func TestCloseStore(t *testing.T) {
	t.Parallel()
	ok := &fakeClosing{}
	sink := newLogSink()
	closeStore(ok, sink.logger())
	if strings.Join(ok.calls, ",") != "ping,close" || sink.text() != "" {
		t.Errorf("calls %v, log %q", ok.calls, sink.text())
	}

	failing := &fakeClosing{pingErr: errBoom, closeErr: errBoom}
	sink = newLogSink()
	closeStore(failing, sink.logger())
	if strings.Join(failing.calls, ",") != "ping,close" {
		t.Errorf("calls = %v; Close must run even when Ping fails", failing.calls)
	}
	sink.mustFind(t, "database still busy at shutdown")
	sink.mustFind(t, "close store failed")
}

// fakeSettings is a settingsReader.
type fakeSettings struct {
	allowPrivate bool
	err          error
	calls        int
}

func (f *fakeSettings) Settings(context.Context) (model.SystemSettings, error) {
	f.calls++
	s := model.DefaultSystemSettings()
	s.AllowPrivateFetch = f.allowPrivate
	return s, f.err
}

func TestFetchPolicy(t *testing.T) {
	t.Parallel()
	yes, no := true, false
	tests := []struct {
		name      string
		envValue  *bool
		loopback  bool
		setting   bool
		err       error
		want      fetch.Policy
		wantCalls int
	}{
		{name: "setting off", want: fetch.Policy{}, wantCalls: 1},
		{name: "setting on", setting: true, want: fetch.Policy{AllowPrivate: true}, wantCalls: 1},
		{name: "environment on wins", envValue: &yes, want: fetch.Policy{AllowPrivate: true}},
		{name: "environment off wins", envValue: &no, setting: true, want: fetch.Policy{}},
		{name: "loopback from environment", loopback: true, want: fetch.Policy{AllowLoopback: true}, wantCalls: 1},
		{name: "settings error blocks private", setting: true, err: errBoom, loopback: true, want: fetch.Policy{AllowLoopback: true}, wantCalls: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			st := &fakeSettings{allowPrivate: tc.setting, err: tc.err}
			sink := newLogSink()
			cfg := config.Config{AllowPrivateFetch: tc.envValue, AllowLoopbackFetch: tc.loopback}
			if got := fetchPolicy(st, cfg, sink.logger())(); got != tc.want {
				t.Errorf("policy = %+v, want %+v", got, tc.want)
			}
			if st.calls != tc.wantCalls {
				t.Errorf("settings reads = %d, want %d", st.calls, tc.wantCalls)
			}
			_, warned := sink.find("read fetch policy failed, private addresses stay blocked")
			if warned != (tc.err != nil) {
				t.Errorf("warning logged = %v", warned)
			}
		})
	}
}

func TestUserAgent(t *testing.T) {
	t.Parallel()
	if got := userAgent("1.2.3"); got != "KinexMark/1.2.3 (+metadata fetcher)" {
		t.Errorf("userAgent = %q", got)
	}
}

// fakeBootstrap is a bootstrapStore with injectable failures.
type fakeBootstrap struct {
	users                       int
	countErr, createErr, logErr error
	created                     []model.NewUser
	audits                      []model.AuditEntry
}

func (f *fakeBootstrap) CountUsers(context.Context) (int, error) { return f.users, f.countErr }

func (f *fakeBootstrap) CreateUser(_ context.Context, n model.NewUser) (model.User, error) {
	if f.createErr != nil {
		return model.User{}, f.createErr
	}
	f.created = append(f.created, n)
	return model.User{ID: 7, Username: n.Username, Role: n.Role}, nil
}

func (f *fakeBootstrap) AppendAudit(_ context.Context, e model.AuditEntry) error {
	f.audits = append(f.audits, e)
	return f.logErr
}

func TestBootstrapAdmin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := auth.NewHasher(testIterations, 1)
	cfg := config.Config{AdminUser: "admin"}

	t.Run("non-empty database without password is silent", func(t *testing.T) {
		t.Parallel()
		st := &fakeBootstrap{users: 1}
		sink := newLogSink()
		if err := bootstrapAdmin(ctx, st, h, cfg, sink.logger()); err != nil {
			t.Fatal(err)
		}
		if len(st.created) != 0 || sink.text() != "" {
			t.Errorf("created %v, log %q", st.created, sink.text())
		}
	})

	t.Run("generated password", func(t *testing.T) {
		t.Parallel()
		st := &fakeBootstrap{}
		sink := newLogSink()
		if err := bootstrapAdmin(ctx, st, h, cfg, sink.logger()); err != nil {
			t.Fatal(err)
		}
		if len(st.created) != 1 || !st.created[0].MustChangePassword || st.created[0].Role != model.RoleAdmin {
			t.Fatalf("created = %+v", st.created)
		}
		pw := sink.mustFind(t, "initial admin password generated").attrs["password"]
		if code := auth.CheckPasswordPolicy(pw, "admin", ""); code != "" {
			t.Errorf("generated password breaks the policy: %s", code)
		}
		if ok, _, err := h.Verify(st.created[0].PasswordHash, pw); err != nil || !ok {
			t.Errorf("hash does not verify: %v", err)
		}
		a := st.audits[0]
		if a.Event != model.EventBootstrapAdminCreated || a.Target.Name != "admin" || *a.Target.ID != 7 {
			t.Errorf("audit = %+v", a)
		}
	})

	failures := []struct {
		name   string
		st     *fakeBootstrap
		cfg    config.Config
		prefix string
	}{
		{"count fails", &fakeBootstrap{countErr: errBoom}, cfg, "bootstrap admin: count users: "},
		{"create fails", &fakeBootstrap{createErr: errBoom}, cfg, "bootstrap admin: create user: "},
		{"audit fails", &fakeBootstrap{logErr: errBoom}, cfg, "bootstrap admin: audit: "},
	}
	for _, tc := range failures {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := bootstrapAdmin(ctx, tc.st, h, tc.cfg, newLogSink().logger())
			if !errors.Is(err, errBoom) || !strings.HasPrefix(err.Error(), tc.prefix) {
				t.Errorf("err = %v, want prefix %q", err, tc.prefix)
			}
		})
	}

	t.Run("password containing the username is rejected", func(t *testing.T) {
		t.Parallel()
		weak := config.Config{AdminUser: "admin", AdminPassword: "my-admin-password-2026"}
		st := &fakeBootstrap{}
		if err := bootstrapAdmin(ctx, st, h, weak, newLogSink().logger()); !errors.Is(err, errWeakAdminPassword) {
			t.Errorf("err = %v, want %v", err, errWeakAdminPassword)
		}
		if len(st.created) != 0 {
			t.Error("a user was created")
		}
	})
}

// fakeAdmin is an adminStore that records calls and can fail one of them.
type fakeAdmin struct {
	credsErr error
	failOn   string
	calls    []string
	audits   []model.AuditEntry
}

func (f *fakeAdmin) call(name string) error {
	f.calls = append(f.calls, name)
	if f.failOn == name {
		return errBoom
	}
	return nil
}

func (f *fakeAdmin) Credentials(_ context.Context, key string) (model.Credentials, error) {
	if f.credsErr != nil {
		return model.Credentials{}, f.credsErr
	}
	return model.Credentials{User: model.User{ID: 3, Username: "Alice", UsernameKey: key}}, nil
}

func (f *fakeAdmin) ClearUserThrottle(context.Context, int64) error { return f.call("throttle") }

func (f *fakeAdmin) SetPassword(context.Context, int64, string, bool) error {
	return f.call("password")
}

func (f *fakeAdmin) DeleteUserSessions(context.Context, int64, int64) (int, error) {
	return 0, f.call("sessions")
}

func (f *fakeAdmin) AppendAudit(_ context.Context, e model.AuditEntry) error {
	f.audits = append(f.audits, e)
	return f.call("audit")
}

// failWriter fails every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errBoom }

func TestRunAdminCommandSteps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := auth.NewHasher(testIterations, 1)
	unlock := config.AdminCommand{Kind: config.AdminCommandUnlockUser, Username: "alice"}
	reset := config.AdminCommand{Kind: config.AdminCommandResetPassword, Username: "alice"}

	t.Run("reset order", func(t *testing.T) {
		t.Parallel()
		st := &fakeAdmin{}
		if err := runAdminCommand(ctx, st, h, reset, io.Discard); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(st.calls, ","); got != "password,sessions,throttle,audit" {
			t.Errorf("calls = %s", got)
		}
		if st.audits[0].Event != model.EventUserPasswordReset || st.audits[0].Target.Name != "Alice" {
			t.Errorf("audit = %+v", st.audits[0])
		}
	})

	tests := []struct {
		name   string
		st     *fakeAdmin
		cmd    config.AdminCommand
		out    io.Writer
		prefix string
	}{
		{"credentials fail", &fakeAdmin{credsErr: errBoom}, unlock, io.Discard, "load user: "},
		{"unlock throttle fails", &fakeAdmin{failOn: "throttle"}, unlock, io.Discard, "clear throttle: "},
		{"unlock audit fails", &fakeAdmin{failOn: "audit"}, unlock, io.Discard, "audit: "},
		{"unlock output fails", &fakeAdmin{}, unlock, failWriter{}, "boom"},
		{"reset password fails", &fakeAdmin{failOn: "password"}, reset, io.Discard, "set password: "},
		{"reset sessions fail", &fakeAdmin{failOn: "sessions"}, reset, io.Discard, "delete sessions: "},
		{"reset throttle fails", &fakeAdmin{failOn: "throttle"}, reset, io.Discard, "clear throttle: "},
		{"reset audit fails", &fakeAdmin{failOn: "audit"}, reset, io.Discard, "audit: "},
		{"reset output fails", &fakeAdmin{}, reset, failWriter{}, "boom"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := runAdminCommand(ctx, tc.st, h, tc.cmd, tc.out)
			if !errors.Is(err, errBoom) || !strings.HasPrefix(err.Error(), tc.prefix) {
				t.Errorf("err = %v, want prefix %q", err, tc.prefix)
			}
		})
	}

	t.Run("unknown user", func(t *testing.T) {
		t.Parallel()
		err := runAdminCommand(ctx, &fakeAdmin{credsErr: model.ErrNotFound}, h, unlock, io.Discard)
		if !errors.Is(err, errUnknownUser) {
			t.Errorf("err = %v, want %v", err, errUnknownUser)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		t.Parallel()
		st := &fakeAdmin{}
		err := runAdminCommand(ctx, st, h, config.AdminCommand{Kind: "explode", Username: "alice"}, io.Discard)
		if !errors.Is(err, errUnknownCommand) || len(st.calls) != 0 {
			t.Errorf("err = %v, calls %v", err, st.calls)
		}
	})
}

func TestNewHTTPServerLimits(t *testing.T) {
	t.Parallel()
	sink := newLogSink()
	srv := newHTTPServer("127.0.0.1:1", http.NotFoundHandler(), sink.logger())
	if srv.ReadHeaderTimeout != 5*time.Second || srv.ReadTimeout != 30*time.Second ||
		srv.WriteTimeout != 30*time.Second || srv.IdleTimeout != 2*time.Minute || srv.MaxHeaderBytes != 64<<10 {
		t.Errorf("limits = %v %v %v %v %d", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.WriteTimeout, srv.IdleTimeout, srv.MaxHeaderBytes)
	}
	srv.ErrorLog.Print("http: TLS handshake error from 127.0.0.1:1: EOF")
	if rec := sink.mustFind(t, "http: TLS handshake error from 127.0.0.1:1: EOF"); rec.level != slog.LevelWarn {
		t.Errorf("server error log level = %v, want WARN", rec.level)
	}
}

func TestNewLogger(t *testing.T) {
	t.Parallel()
	var text, js bytes.Buffer
	NewLogger(config.Config{LogLevel: slog.LevelWarn, LogFormat: "text"}, &text).Info("hidden")
	NewLogger(config.Config{LogLevel: slog.LevelWarn, LogFormat: "text"}, &text).Warn("shown", "k", "v")
	if got := text.String(); strings.Contains(got, "hidden") || !strings.Contains(got, "msg=shown k=v") {
		t.Errorf("text log = %q", got)
	}
	NewLogger(config.Config{LogLevel: slog.LevelDebug, LogFormat: "json"}, &js).Debug("event", "n", 1)
	var rec map[string]any
	if err := json.Unmarshal(js.Bytes(), &rec); err != nil || rec["msg"] != "event" || rec["level"] != "DEBUG" {
		t.Errorf("json log = %q (%v)", js.String(), err)
	}
}

func TestCheckHealth(t *testing.T) {
	t.Parallel()
	status := func(code int) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/healthz" {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(code)
		}))
		t.Cleanup(srv.Close)
		return srv
	}
	port := func(srv *httptest.Server) string {
		_, p, _ := net.SplitHostPort(srv.Listener.Addr().String())
		return p
	}
	ok := status(http.StatusOK)
	failing := status(http.StatusServiceUnavailable)
	closed := status(http.StatusOK)
	closedAddr := closed.Listener.Addr().String()
	closed.Close()

	tests := []struct {
		name string
		cfg  config.Config
		want int
	}{
		{"live", config.Config{Addr: ok.Listener.Addr().String()}, 0},
		{"empty host becomes loopback", config.Config{Addr: ":" + port(ok)}, 0},
		{"any address becomes loopback", config.Config{Addr: "0.0.0.0:" + port(ok)}, 0},
		{"not 200", config.Config{Addr: failing.Listener.Addr().String()}, 1},
		{"stopped", config.Config{Addr: closedAddr}, 1},
		{"invalid address", config.Config{Addr: "no-port"}, 1},
		{"tls against plain http", config.Config{Addr: ok.Listener.Addr().String(), TLSCertFile: "cert.pem"}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := CheckHealth(tc.cfg); got != tc.want {
				t.Errorf("CheckHealth(%q) = %d, want %d", tc.cfg.Addr, got, tc.want)
			}
		})
	}
}
