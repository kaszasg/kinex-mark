// Package api implements the HTTP layer: middleware, the route table, the
// JSON handlers and the embedded single-page application.
package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/config"
)

// server carries the dependencies and the per-process state of the handlers.
type server struct {
	deps  Deps
	log   *slog.Logger
	clock clock.Clock
	cfg   config.Config

	ui        *spa
	csp       string // Content-Security-Policy outside secure mode
	cspSecure string // the same with upgrade-insecure-requests

	// loopbackOnly: no APP_BASE_URL and a loopback listen address, so the
	// host check accepts loopback names only (6.3).
	loopbackOnly bool
	// forwardWarn makes the untrusted X-Forwarded-* warning one-shot (6.2).
	forwardWarn sync.Once
	// lockAudits limits auth.login.locked to one entry per key and interval.
	lockAudits lockAuditLimiter
}

// New builds the complete HTTP handler: the global middleware chain of 6.1
// (recoverer, securityHeaders, accessLog, hostCheck, csrf, body limit), the
// routes of routes.go behind their per-route checks (guard), the JSON 404
// catch-all under /api/ and the SPA handler. Store, Hasher, Keyring and a UI
// with index.html are required; Clock defaults to the real clock and Log to
// a discarding logger.
func New(d Deps) (http.Handler, error) {
	s, err := newServer(d)
	if err != nil {
		return nil, err
	}
	return s.handler(), nil
}

// newServer validates d, applies the defaults and prepares the SPA and the
// CSP.
func newServer(d Deps) (*server, error) {
	switch {
	case d.Store == nil:
		return nil, errors.New("api: store is required")
	case d.Hasher == nil:
		return nil, errors.New("api: hasher is required")
	case d.Keyring == nil:
		return nil, errors.New("api: keyring is required")
	}
	if d.Clock == nil {
		d.Clock = clock.Real{}
	}
	if d.Log == nil {
		d.Log = slog.New(slog.DiscardHandler)
	}
	ui, err := newSPA(d.UI)
	if err != nil {
		return nil, err
	}
	hashes, err := inlineScriptHashes(ui.index)
	if err != nil {
		return nil, fmt.Errorf("api: %w", err)
	}
	s := &server{
		deps:         d,
		log:          d.Log,
		clock:        d.Clock,
		cfg:          d.Config,
		ui:           ui,
		csp:          buildCSP(hashes, false),
		cspSecure:    buildCSP(hashes, true),
		loopbackOnly: d.Config.BaseURL == nil && listensOnLoopback(d.Config.Addr),
	}
	return s, nil
}

// handler assembles the mux and the global middleware chain (6.1).
func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	for _, rt := range routes() {
		mux.HandleFunc(rt.key(), s.guard(rt))
	}
	// Unknown API routes and unregistered methods get JSON 404, not index.html.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, newError(codeEndpointNotFound))
	})
	mux.Handle("/", s.ui)

	var h http.Handler = limitBody(mux)
	h = s.csrf(h)
	h = s.hostCheck(h)
	h = s.accessLog(h)
	h = s.securityHeaders(h)
	h = s.recoverer(h)
	return h
}

// now returns the current time of the injected clock (UTC).
func (s *server) now() time.Time { return s.clock.Now() }
