package api

import (
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

// headerKinexRequest is the custom request header every state-changing API
// call must carry (6.3). A plain HTML form or sendBeacon cannot send it and
// another origin cannot send it without a CORS preflight, which is never
// granted.
const headerKinexRequest = "X-Kinex-Request"

// isAPIPath reports whether p is under /api/ (or is /api itself).
func isAPIPath(p string) bool {
	return p == "/api" || strings.HasPrefix(p, "/api/")
}

// isUnsafeMethod reports whether m changes state (6.3).
func isUnsafeMethod(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// defaultPort returns the implicit port of a scheme.
func defaultPort(scheme string) string {
	if strings.EqualFold(scheme, "https") {
		return "443"
	}
	return "80"
}

// splitHost splits "host", "host:port", "[v6]" or "[v6]:port" into a
// lower-case host name (without brackets) and a port ("" when absent).
func splitHost(hostport string) (string, string) {
	hostport = strings.ToLower(strings.TrimSpace(hostport))
	if h, p, err := net.SplitHostPort(hostport); err == nil {
		return strings.Trim(h, "[]"), p
	}
	return strings.Trim(hostport, "[]"), ""
}

// canonicalHost returns host[:port] in a comparable form: lower case, the
// default port of scheme removed, IPv6 in brackets.
func canonicalHost(hostport, scheme string) string {
	h, p := splitHost(hostport)
	if p == defaultPort(scheme) {
		p = ""
	}
	if strings.Contains(h, ":") {
		h = "[" + h + "]"
	}
	if p == "" {
		return h
	}
	return h + ":" + p
}

// isLoopbackHostName reports whether h (without port or brackets) is
// localhost, 127.0.0.1 or ::1, the names accepted on a loopback listener.
func isLoopbackHostName(h string) bool {
	switch h {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// listensOnLoopback reports whether the configured listen address is a
// loopback address (127.0.0.0/8, ::1 or localhost).
func listensOnLoopback(addr string) bool {
	h, _ := splitHost(addr)
	if h == "localhost" {
		return true
	}
	a, err := netip.ParseAddr(h)
	return err == nil && a.Unmap().IsLoopback()
}

// hostAllowed applies the host check of 6.3 to r.
func (s *server) hostAllowed(r *http.Request) bool {
	if base := s.cfg.BaseURL; base != nil {
		return canonicalHost(s.requestHost(r), base.Scheme) == canonicalHost(base.Host, base.Scheme)
	}
	if s.loopbackOnly {
		h, _ := splitHost(s.requestHost(r))
		return isLoopbackHostName(h)
	}
	return true
}

// hostCheck rejects /api/ requests whose Host does not belong to this server
// with 421 host_mismatch (6.3, DNS rebinding). /healthz and the static files
// are not checked, because the health check calls 127.0.0.1.
func (s *server) hostCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) && !s.hostAllowed(r) {
			writeError(w, newError(codeHostMismatch))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// expectedOrigin returns the scheme and canonical host of the origin that
// state-changing requests must come from: APP_BASE_URL when set, otherwise
// the scheme and host this request was addressed to.
func (s *server) expectedOrigin(r *http.Request) (string, string) {
	if base := s.cfg.BaseURL; base != nil {
		scheme := strings.ToLower(base.Scheme)
		return scheme, canonicalHost(base.Host, scheme)
	}
	scheme := s.requestScheme(r)
	return scheme, canonicalHost(s.requestHost(r), scheme)
}

// originMatches reports whether an Origin header value is the expected origin.
func (s *server) originMatches(r *http.Request, origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	scheme, host := s.expectedOrigin(r)
	got := strings.ToLower(u.Scheme)
	return got == scheme && canonicalHost(u.Host, got) == host
}

// csrfAllowed applies the CSRF rule of 6.3 to a state-changing request.
func (s *server) csrfAllowed(r *http.Request) bool {
	if r.Header.Get(headerKinexRequest) != "1" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		return s.originMatches(r, origin)
	}
	switch site := r.Header.Get("Sec-Fetch-Site"); site {
	case "", "same-origin", "none":
		return true
	}
	return false
}

// csrf rejects state-changing /api/ requests without the custom header or
// from a foreign origin with 403 csrf_rejected (6.3). Sign-in is included.
func (s *server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) && isUnsafeMethod(r.Method) && !s.csrfAllowed(r) {
			writeError(w, newError(codeCSRFRejected))
			return
		}
		next.ServeHTTP(w, r)
	})
}
