package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// csrfProbe is a state-changing public route whose handler is reached
// once the CSRF and host checks pass. The tests only assert that neither
// check rejected the request, so they hold for the stub and the real
// handler alike.
const csrfProbe = "/api/auth/logout"

// expectPassedChecks fails when the host or CSRF check rejected res.
func expectPassedChecks(t *testing.T, res *http.Response) {
	t.Helper()
	body := readBody(t, res)
	if res.StatusCode == http.StatusMisdirectedRequest || strings.Contains(string(body), `"code":"`+codeCSRFRejected+`"`) {
		t.Fatalf("%s %s rejected: %d %s", res.Request.Method, res.Request.URL.Path, res.StatusCode, body)
	}
}

func TestCSRF(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t)
	reject := func(res *http.Response) {
		t.Helper()
		expectError(t, res, http.StatusForbidden, codeCSRFRejected)
	}

	// 1. The custom header is required on every unsafe method.
	noHeader := c.withHeader(headerKinexRequest, "")
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		reject(noHeader.do(m, csrfProbe, nil))
	}
	reject(noHeader.post("/api/auth/login", map[string]string{"username": "a", "password": "b"}))
	reject(noHeader.post("/api/nope", nil)) // before the 404 catch-all
	reject(c.withHeader(headerKinexRequest, "true").post(csrfProbe, nil))

	// Safe methods and non-API paths need no header.
	expectStatus(t, noHeader.get("/api/public/config"), http.StatusOK)
	expectStatus(t, noHeader.do(http.MethodHead, "/healthz", nil), http.StatusOK)
	expectStatus(t, noHeader.post("/login", nil), http.StatusOK) // SPA
	expectError(t, noHeader.do(http.MethodOptions, "/api/links", nil), http.StatusNotFound, codeEndpointNotFound)

	// 2. Origin must match; without Origin, Sec-Fetch-Site must be same-origin or none.
	expectPassedChecks(t, c.post(csrfProbe, nil))
	reject(c.withHeader("Origin", "http://evil.example").post(csrfProbe, nil))
	reject(c.withHeader("Origin", "null").post(csrfProbe, nil))
	reject(c.withHeader("Origin", strings.Replace(e.srv.URL, "http://", "https://", 1)).post(csrfProbe, nil))
	reject(c.withHeader("Origin", e.srv.URL+"/path").post(csrfProbe, nil))
	expectPassedChecks(t, c.withHeader("Origin", e.srv.URL+"/").post(csrfProbe, nil))
	expectPassedChecks(t, c.withHeader("Origin", strings.ToUpper(e.srv.URL)).post(csrfProbe, nil))

	noOrigin := c.withHeader("Origin", "")
	expectPassedChecks(t, noOrigin.post(csrfProbe, nil)) // non-browser client
	reject(noOrigin.withHeader("Sec-Fetch-Site", "cross-site").post(csrfProbe, nil))
	reject(noOrigin.withHeader("Sec-Fetch-Site", "same-site").post(csrfProbe, nil))
	expectPassedChecks(t, noOrigin.withHeader("Sec-Fetch-Site", "same-origin").post(csrfProbe, nil))
	expectPassedChecks(t, noOrigin.withHeader("Sec-Fetch-Site", "none").post(csrfProbe, nil))
	// A matching Origin wins over Sec-Fetch-Site.
	expectPassedChecks(t, c.withHeader("Sec-Fetch-Site", "cross-site").post(csrfProbe, nil))
}

func TestCSRFWithBaseURL(t *testing.T) {
	t.Parallel()
	e := newEnv(t, withBaseURL("https://mark.example.com"))
	c := e.client(t).withHeader("Host", "mark.example.com")
	expectPassedChecks(t, c.withHeader("Origin", "https://mark.example.com").post(csrfProbe, nil))
	expectPassedChecks(t, c.withHeader("Origin", "https://MARK.example.com:443").post(csrfProbe, nil))
	expectError(t, c.withHeader("Origin", e.srv.URL).post(csrfProbe, nil), http.StatusForbidden, codeCSRFRejected)
	expectError(t, c.withHeader("Origin", "http://mark.example.com").post(csrfProbe, nil), http.StatusForbidden, codeCSRFRejected)
	expectError(t, c.withHeader("Origin", "https://mark.example.com:8443").post(csrfProbe, nil), http.StatusForbidden, codeCSRFRejected)
}

func TestCSRFBehindProxy(t *testing.T) {
	t.Parallel()
	const origin = "https://app.example.com"
	proxied := func(c *apiClient) *apiClient {
		return c.withHeader("X-Forwarded-Proto", "https").
			withHeader("X-Forwarded-Host", "app.example.com").
			withHeader("X-Forwarded-For", "198.51.100.7").
			withHeader("Origin", origin)
	}

	// Trusted proxy: the forwarded scheme and host form the expected origin.
	trusted := newEnv(t, withTrustedProxy("127.0.0.1"), withAddr(":8080"))
	expectPassedChecks(t, proxied(trusted.client(t)).post(csrfProbe, nil))
	expectError(t, proxied(trusted.client(t)).withHeader("Origin", trusted.srv.URL).post(csrfProbe, nil),
		http.StatusForbidden, codeCSRFRejected)
	if n := trusted.logs.count(untrustedForwardWarning); n != 0 {
		t.Errorf("trusted proxy logged %d warnings", n)
	}

	// Untrusted peer: the headers have no effect and one warning is logged.
	untrusted := newEnv(t)
	for i := 0; i < 3; i++ {
		expectError(t, proxied(untrusted.client(t)).post(csrfProbe, nil), http.StatusForbidden, codeCSRFRejected)
	}
	// Without the foreign Origin the request passes on its real host.
	expectPassedChecks(t, proxied(untrusted.client(t)).withHeader("Origin", untrusted.srv.URL).post(csrfProbe, nil))
	logs := untrusted.logs.String()
	if n := untrusted.logs.count(untrustedForwardWarning); n != 1 {
		t.Fatalf("warning logged %d times, want 1:\n%s", n, logs)
	}
	if !strings.Contains(logs, "peer=127.0.0.1") || strings.Contains(logs, "app.example.com") || strings.Contains(logs, "198.51.100.7") {
		t.Fatalf("warning must name the peer and no header value:\n%s", logs)
	}
}

func TestHostCheck(t *testing.T) {
	t.Parallel()
	port := func(e *testEnv) string {
		u, err := url.Parse(e.srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		return u.Port()
	}

	t.Run("base URL", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, withBaseURL("https://mark.example.com"))
		c := e.client(t)
		expectError(t, c.get("/api/public/config"), http.StatusMisdirectedRequest, codeHostMismatch) // 127.0.0.1:port
		expectError(t, c.withHeader("Host", "evil.example").get("/api/public/config"), http.StatusMisdirectedRequest, codeHostMismatch)
		expectError(t, c.withHeader("Host", "mark.example.com:8443").get("/api/public/config"), http.StatusMisdirectedRequest, codeHostMismatch)
		expectError(t, c.withHeader("Host", "evil.example").get("/api/nope"), http.StatusMisdirectedRequest, codeHostMismatch)
		expectError(t, c.withHeader("Host", "evil.example").post(csrfProbe, nil), http.StatusMisdirectedRequest, codeHostMismatch)
		expectStatus(t, c.withHeader("Host", "mark.example.com").get("/api/public/config"), http.StatusOK)
		expectStatus(t, c.withHeader("Host", "Mark.Example.com:443").get("/api/public/config"), http.StatusOK)
		// /healthz and the static files are not checked.
		expectStatus(t, c.withHeader("Host", "evil.example").get("/healthz"), http.StatusOK)
		expectStatus(t, c.withHeader("Host", "evil.example").get("/"), http.StatusOK)
	})

	t.Run("base URL with port behind trusted proxy", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, withBaseURL("http://intranet:8081"), withTrustedProxy("127.0.0.1"))
		c := e.client(t)
		expectStatus(t, c.withHeader("X-Forwarded-Host", "intranet:8081").get("/api/public/config"), http.StatusOK)
		expectError(t, c.withHeader("X-Forwarded-Host", "intranet").get("/api/public/config"), http.StatusMisdirectedRequest, codeHostMismatch)
		expectStatus(t, c.withHeader("Host", "intranet:8081").get("/api/public/config"), http.StatusOK)
	})

	t.Run("loopback listener", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t) // 127.0.0.1:0
		c := e.client(t)
		p := port(e)
		for _, host := range []string{"127.0.0.1:" + p, "localhost:" + p, "[::1]:" + p, "LOCALHOST:5173", "localhost"} {
			expectStatus(t, c.withHeader("Host", host).get("/api/public/config"), http.StatusOK)
		}
		for _, host := range []string{"evil.example", "evil.example:" + p, "127.0.0.2:" + p, "localhost.evil.example"} {
			expectError(t, c.withHeader("Host", host).get("/api/public/config"), http.StatusMisdirectedRequest, codeHostMismatch)
		}
		expectStatus(t, c.withHeader("Host", "evil.example").get("/healthz"), http.StatusOK)
	})

	t.Run("non-loopback listener", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, withAddr(":8080"))
		expectStatus(t, e.client(t).withHeader("Host", "evil.example").get("/api/public/config"), http.StatusOK)
	})
}

func TestHostHelpers(t *testing.T) {
	t.Parallel()
	canon := []struct{ in, scheme, want string }{
		{"Example.COM", "https", "example.com"},
		{"example.com:443", "https", "example.com"},
		{"example.com:443", "http", "example.com:443"},
		{"example.com:80", "http", "example.com"},
		{"[::1]:8080", "http", "[::1]:8080"},
		{"[::1]", "http", "[::1]"},
		{"::1", "http", "[::1]"},
		{" localhost:5173 ", "http", "localhost:5173"},
	}
	for _, c := range canon {
		if got := canonicalHost(c.in, c.scheme); got != c.want {
			t.Errorf("canonicalHost(%q, %q) = %q, want %q", c.in, c.scheme, got, c.want)
		}
	}
	loop := map[string]bool{
		"127.0.0.1:8080": true, "127.0.0.5:1": true, "[::1]:8080": true, "localhost:8080": true,
		":8080": false, "0.0.0.0:8080": false, "192.168.1.2:80": false, "[::]:8080": false, "example.com:80": false,
	}
	for addr, want := range loop {
		if got := listensOnLoopback(addr); got != want {
			t.Errorf("listensOnLoopback(%q) = %v", addr, got)
		}
	}
	if !isAPIPath("/api") || !isAPIPath("/api/x") || isAPIPath("/apix") || isAPIPath("/") {
		t.Error("isAPIPath")
	}
}

func TestOriginMatchesRejectsMalformed(t *testing.T) {
	t.Parallel()
	s, err := newServer(minimalDeps(t))
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "http://example.com/api/x", nil)
	for _, o := range []string{"http://example.com?x=1", "http://example.com#f", "http://u@example.com", "example.com", "http://", "%zz"} {
		if s.originMatches(r, o) {
			t.Errorf("origin %q accepted", o)
		}
	}
	if !s.originMatches(r, "http://example.com") || !s.originMatches(r, "http://example.com:80") {
		t.Error("matching origin rejected")
	}
}
