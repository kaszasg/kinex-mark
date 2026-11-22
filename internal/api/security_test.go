package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/config"
)

func TestBuildCSP(t *testing.T) {
	t.Parallel()
	const want = "default-src 'self'; script-src 'self' 'sha256-a' 'sha256-b'; style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; " +
		"form-action 'self'; frame-ancestors 'none'; manifest-src 'self'; worker-src 'self'"
	if got := buildCSP([]string{"'sha256-a'", "'sha256-b'"}, false); got != want {
		t.Errorf("CSP\n got %q\nwant %q", got, want)
	}
	if got := buildCSP([]string{"'sha256-a'", "'sha256-b'"}, true); got != want+"; upgrade-insecure-requests" {
		t.Errorf("secure CSP %q", got)
	}
	if got := buildCSP(nil, false); !strings.Contains(got, "script-src 'self';") {
		t.Errorf("CSP without hashes %q", got)
	}
}

// expectSecurityHeaders checks the 6.9 header set; hsts tells whether
// Strict-Transport-Security and upgrade-insecure-requests are expected.
func expectSecurityHeaders(t *testing.T, res *http.Response, hsts bool) {
	t.Helper()
	want := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"Referrer-Policy":              "no-referrer",
		"X-Frame-Options":              "DENY",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           "camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()",
		"X-Robots-Tag":                 "noindex, nofollow",
	}
	for k, v := range want {
		if got := res.Header.Get(k); got != v {
			t.Errorf("%s %s: %s = %q, want %q", res.Request.Method, res.Request.URL.Path, k, got, v)
		}
	}
	csp := res.Header.Get("Content-Security-Policy")
	wantCSP := buildCSP([]string{cspHash(testInlineScript1), cspHash(testInlineScript2)}, hsts)
	if csp != wantCSP {
		t.Errorf("%s: CSP\n got %q\nwant %q", res.Request.URL.Path, csp, wantCSP)
	}
	if strings.Contains(strings.Split(strings.Split(csp, "script-src")[1], ";")[0], "unsafe-inline") {
		t.Errorf("script-src allows unsafe-inline: %q", csp)
	}
	if got := res.Header.Get("Strict-Transport-Security"); (got == "max-age=31536000") != hsts || (!hsts && got != "") {
		t.Errorf("%s: HSTS %q, want present=%v", res.Request.URL.Path, got, hsts)
	}
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t)
	for _, res := range []*http.Response{
		c.get("/healthz"),
		c.get("/api/public/config"),
		c.get("/api/nope"),
		c.get("/api/auth/me"), // 401
		c.withHeader(headerKinexRequest, "").post("/api/auth/logout", nil), // 403 csrf
		c.withHeader("Host", "evil.example").get("/api/public/config"),     // 421
		c.get("/"),
		c.get("/_app/immutable/x.js"),
		c.get("/_app/missing.js"),
	} {
		expectSecurityHeaders(t, res, false)
	}
}

func TestSecureModeHeaders(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		opts    []envOption
		headers map[string]string
		secure  bool
	}{
		{"auto over http", nil, nil, false},
		{"auto over TLS", []envOption{withTLS()}, nil, true},
		{"auto with https base URL, no proxy trust", []envOption{withBaseURL("https://mark.example.com")}, nil, true},
		{"auto with http base URL", []envOption{withBaseURL("http://mark.example.com")}, nil, false},
		{"always", []envOption{withSecureCookies(config.SecureCookiesTrue)}, nil, true},
		{"never, even over TLS", []envOption{withSecureCookies(config.SecureCookiesFalse), withTLS()}, nil, false},
		{"trusted proxy https", []envOption{withTrustedProxy("127.0.0.1")}, map[string]string{"X-Forwarded-Proto": "https"}, true},
		{"trusted proxy http", []envOption{withTrustedProxy("127.0.0.0/8")}, map[string]string{"X-Forwarded-Proto": "http"}, false},
		{"untrusted proxy https", nil, map[string]string{"X-Forwarded-Proto": "https"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t, tc.opts...)
			c := e.client(t)
			for k, v := range tc.headers {
				c = c.withHeader(k, v)
			}
			expectSecurityHeaders(t, c.get("/healthz"), tc.secure)
			expectSecurityHeaders(t, c.get("/"), tc.secure)
		})
	}
}
