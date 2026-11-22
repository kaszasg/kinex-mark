package api

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newLoggedServer returns a server over minimalDeps logging to a buffer at
// the given level.
func newLoggedServer(t *testing.T, level slog.Level) (*server, *bytes.Buffer) {
	t.Helper()
	d := minimalDeps(t)
	var buf bytes.Buffer
	d.Log = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level}))
	s, err := newServer(d)
	if err != nil {
		t.Fatal(err)
	}
	return s, &buf
}

func TestRecoverer(t *testing.T) {
	t.Parallel()
	s, logs := newLoggedServer(t, slog.LevelInfo)

	h := s.recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/x", nil))
	expectRecError(t, rec, http.StatusInternalServerError, codeInternalError)
	if out := logs.String(); !strings.Contains(out, "msg=panic") || !strings.Contains(out, "err=boom") || !strings.Contains(out, "stack=") {
		t.Fatalf("panic log: %s", out)
	}

	// A panic after the response started cannot change it.
	h = s.recoverer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		panic("late")
	}))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusAccepted || rec.Body.Len() != 0 {
		t.Fatalf("late panic: %d %q", rec.Code, rec.Body.String())
	}

	// http.ErrAbortHandler is passed on to net/http.
	h = s.recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	func() {
		defer func() {
			if v := recover(); v != http.ErrAbortHandler {
				t.Fatalf("recovered %v", v)
			}
		}()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}()
}

func TestRecovererInChain(t *testing.T) {
	t.Parallel()
	// A panicking store method ends as 500 with the security headers set.
	e := newEnv(t, withDeps(func(d *Deps) { d.Store = nilStore{} }))
	res := e.client(t).get("/healthz")
	expectError(t, res, http.StatusInternalServerError, codeInternalError)
	expectSecurityHeaders(t, res, false)
	if !strings.Contains(e.logs.String(), "msg=panic") {
		t.Fatalf("no panic log:\n%s", e.logs)
	}
}

func TestAccessLog(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t)
	expectStatus(t, c.get("/api/public/config?q=private-search-term"), http.StatusOK)
	expectStatus(t, c.get("/api/auth/me?q=private-search-term"), http.StatusUnauthorized)
	expectStatus(t, c.get("/missing/page"), http.StatusOK)
	out := e.logs.String()
	for _, want := range []string{
		`level=DEBUG msg=request method=GET route="GET /api/public/config" status=200 duration=`,
		`level=DEBUG msg=request method=GET route="GET /api/auth/me" status=401 duration=`,
		`level=DEBUG msg=request method=GET route=/ status=200 duration=`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("access log lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "private-search-term") || strings.Contains(out, "/missing/page") {
		t.Errorf("access log leaks the path or query:\n%s", out)
	}

	// Above Debug level nothing is logged.
	s, logs := newLoggedServer(t, slog.LevelInfo)
	h := s.accessLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTeapot || logs.Len() != 0 {
		t.Fatalf("info level: %d %q", rec.Code, logs.String())
	}
}

func TestStatusWriter(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec}
	if sw.code() != http.StatusOK {
		t.Fatal("default code")
	}
	if _, err := sw.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	sw.WriteHeader(http.StatusTeapot) // ignored by the recorder, not recorded
	if sw.code() != http.StatusOK || sw.Unwrap() != rec {
		t.Fatalf("code %d", sw.code())
	}
	sw2 := &statusWriter{ResponseWriter: httptest.NewRecorder()}
	sw2.WriteHeader(http.StatusCreated)
	if sw2.code() != http.StatusCreated {
		t.Fatalf("code %d", sw2.code())
	}
}

func TestLimitBody(t *testing.T) {
	t.Parallel()
	var readErr error
	var n int64
	h := limitBody(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		n, readErr = io.Copy(io.Discard, r.Body)
	}))
	big := bytes.Repeat([]byte("a"), maxRequestBytes+1)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/links/import", bytes.NewReader(big)))
	var mbe *http.MaxBytesError
	if !errors.As(readErr, &mbe) || mbe.Limit != maxRequestBytes || n != maxRequestBytes {
		t.Fatalf("over the limit: n=%d err=%v", n, readErr)
	}
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/links/import", bytes.NewReader(big[:maxRequestBytes])))
	if readErr != nil || n != maxRequestBytes {
		t.Fatalf("at the limit: n=%d err=%v", n, readErr)
	}
	// No body stays no body.
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if readErr != nil || n != 0 {
		t.Fatalf("no body: n=%d err=%v", n, readErr)
	}
}
