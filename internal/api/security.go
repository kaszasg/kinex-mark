package api

import (
	"net/http"
	"strings"

	"github.com/kaszasg/kinex-mark/internal/config"
)

// Security header values (6.9).
const (
	permissionsPolicy = "camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()"
	hstsValue         = "max-age=31536000"
)

// buildCSP returns the Content-Security-Policy of 6.9 with the given
// script hashes ('sha256-...' sources) in script-src. secure appends
// upgrade-insecure-requests.
func buildCSP(scriptHashes []string, secure bool) string {
	scriptSrc := "'self'"
	if len(scriptHashes) > 0 {
		scriptSrc += " " + strings.Join(scriptHashes, " ")
	}
	csp := "default-src 'self'; script-src " + scriptSrc +
		"; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'" +
		"; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'" +
		"; frame-ancestors 'none'; manifest-src 'self'; worker-src 'self'"
	if secure {
		csp += "; upgrade-insecure-requests"
	}
	return csp
}

// secureMode reports whether r is served in secure mode (6.2): always with
// APP_SECURE_COOKIES=true, never with false, and in auto mode when the
// server runs TLS itself, APP_BASE_URL is https, or a trusted proxy reports
// X-Forwarded-Proto: https.
func (s *server) secureMode(r *http.Request) bool {
	switch s.cfg.SecureCookies {
	case config.SecureCookiesTrue:
		return true
	case config.SecureCookiesFalse:
		return false
	}
	if r.TLS != nil || s.cfg.TLSCertFile != "" {
		return true
	}
	if s.cfg.BaseURL != nil && strings.EqualFold(s.cfg.BaseURL.Scheme, "https") {
		return true
	}
	return s.requestScheme(r) == "https"
}

// securityHeaders sets the 6.9 headers on every response. A handler may
// replace the CSP afterwards (the icon endpoint, 5.9.9).
func (s *server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secure := s.secureMode(r)
		h := w.Header()
		if secure {
			h.Set("Content-Security-Policy", s.cspSecure)
			h.Set("Strict-Transport-Security", hstsValue)
		} else {
			h.Set("Content-Security-Policy", s.csp)
		}
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Permissions-Policy", permissionsPolicy)
		h.Set("X-Robots-Tag", "noindex, nofollow")
		next.ServeHTTP(w, r)
	})
}
