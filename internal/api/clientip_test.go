package api

import (
	"bytes"
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/auth"
)

// newUnitServer returns a server over minimalDeps with trusted proxies and
// a captured log.
func newUnitServer(t *testing.T, trusted ...string) (*server, *bytes.Buffer) {
	t.Helper()
	d := minimalDeps(t)
	for _, e := range trusted {
		d.Config.TrustedProxies = append(d.Config.TrustedProxies, netip.MustParsePrefix(e))
	}
	var buf bytes.Buffer
	d.Log = slog.New(slog.NewTextHandler(&buf, nil))
	s, err := newServer(d)
	if err != nil {
		t.Fatal(err)
	}
	return s, &buf
}

// proxyRequest builds a request from peer with the given headers.
func proxyRequest(peer string, headers ...string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "http://app.local/api/x", nil)
	r.RemoteAddr = peer
	for i := 0; i+1 < len(headers); i += 2 {
		r.Header.Add(headers[i], headers[i+1])
	}
	return r
}

func TestPeerAddr(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"1.2.3.4:5":          "1.2.3.4",
		"[::ffff:1.2.3.4]:5": "1.2.3.4",
		"[2001:db8::1]:443":  "2001:db8::1",
		"[fe80::1%eth0]:80":  "fe80::1%eth0",
		"1.2.3.4":            "1.2.3.4",
		"::1":                "::1",
		"[fe80::2%en0]":      "fe80::2",
		"garbage":            "invalid IP",
		"":                   "invalid IP",
	}
	for remote, want := range cases {
		r := proxyRequest(remote)
		if got := peerAddr(r).String(); got != want {
			t.Errorf("peerAddr(%q) = %q, want %q", remote, got, want)
		}
	}
}

func TestClientAddr(t *testing.T) {
	t.Parallel()
	s, logs := newUnitServer(t, "127.0.0.1/32", "10.0.0.0/8", "fd00::/8")
	cases := []struct {
		name string
		r    *http.Request
		want string
	}{
		{"no proxy headers", proxyRequest("203.0.113.9:1"), "203.0.113.9"},
		{"untrusted peer ignores XFF", proxyRequest("203.0.113.9:1", headerForwardedFor, "1.1.1.1"), "203.0.113.9"},
		{"trusted peer single hop", proxyRequest("127.0.0.1:1", headerForwardedFor, "198.51.100.7"), "198.51.100.7"},
		{"mapped IPv6 client unmapped", proxyRequest("127.0.0.1:1", headerForwardedFor, "::ffff:203.0.113.5"), "203.0.113.5"},
		{"rightmost untrusted wins", proxyRequest("127.0.0.1:1", headerForwardedFor, "6.6.6.6, 1.1.1.1, 10.0.0.2"), "1.1.1.1"},
		{"multiple header lines", proxyRequest("127.0.0.1:1", headerForwardedFor, "6.6.6.6", headerForwardedFor, "1.1.1.1, 10.0.0.9"), "1.1.1.1"},
		{"all trusted: leftmost", proxyRequest("127.0.0.1:1", headerForwardedFor, "10.0.0.3, 10.0.0.2"), "10.0.0.3"},
		{"garbage stops the walk", proxyRequest("127.0.0.1:1", headerForwardedFor, "1.1.1.1, garbage, 10.0.0.2"), "10.0.0.2"},
		{"garbage at the end", proxyRequest("127.0.0.1:1", headerForwardedFor, "garbage"), "127.0.0.1"},
		{"trusted IPv6 proxy", proxyRequest("[fd00::1]:1", headerForwardedFor, "2001:db8::5"), "2001:db8::5"},
		{"trusted mapped peer", proxyRequest("[::ffff:10.1.2.3]:1", headerForwardedFor, "8.8.8.8"), "8.8.8.8"},
		{"trusted peer without XFF", proxyRequest("10.0.0.1:1"), "10.0.0.1"},
	}
	for _, c := range cases {
		if got := s.clientAddr(c.r).String(); got != c.want {
			t.Errorf("%s: clientAddr = %q, want %q", c.name, got, c.want)
		}
	}
	// 9.2.3: an IPv4-mapped client gets its own /32 throttle key.
	mapped := s.clientAddr(proxyRequest("127.0.0.1:1", headerForwardedFor, "::ffff:203.0.113.5"))
	other := s.clientAddr(proxyRequest("127.0.0.1:1", headerForwardedFor, "::ffff:203.0.113.6"))
	if auth.IPThrottleKey(mapped) != auth.IPThrottleKey(netip.MustParseAddr("203.0.113.5")) ||
		auth.IPThrottleKey(mapped) == auth.IPThrottleKey(other) {
		t.Errorf("throttle keys %q %q", auth.IPThrottleKey(mapped), auth.IPThrottleKey(other))
	}
	if got := s.clientIP(proxyRequest("garbage")); got != "" {
		t.Errorf("clientIP of an invalid peer = %q", got)
	}
	if got := s.clientIP(proxyRequest("[::ffff:1.2.3.4]:9")); got != "1.2.3.4" {
		t.Errorf("clientIP = %q", got)
	}
	if n := strings.Count(logs.String(), untrustedForwardWarning); n != 1 {
		t.Errorf("warning logged %d times:\n%s", n, logs)
	}
	if strings.Contains(logs.String(), "1.1.1.1") || !strings.Contains(logs.String(), "peer=203.0.113.9") {
		t.Errorf("warning content:\n%s", logs)
	}
}

func TestRequestSchemeAndHost(t *testing.T) {
	t.Parallel()
	s, _ := newUnitServer(t, "127.0.0.1/32")
	cases := []struct {
		name         string
		r            *http.Request
		scheme, host string
	}{
		{"plain", proxyRequest("203.0.113.9:1"), "http", "app.local"},
		{"untrusted forwarded", proxyRequest("203.0.113.9:1", headerForwardedProto, "https", headerForwardedHost, "x.example"), "http", "app.local"},
		{"trusted https", proxyRequest("127.0.0.1:1", headerForwardedProto, "HTTPS", headerForwardedHost, "x.example:8443"), "https", "x.example:8443"},
		{"trusted list", proxyRequest("127.0.0.1:1", headerForwardedProto, "http, https", headerForwardedHost, " a.example , b.example"), "http", "a.example"},
		{"trusted unknown proto", proxyRequest("127.0.0.1:1", headerForwardedProto, "gopher"), "http", "app.local"},
		{"trusted empty host", proxyRequest("127.0.0.1:1", headerForwardedHost, ""), "http", "app.local"},
	}
	for _, c := range cases {
		if got := s.requestScheme(c.r); got != c.scheme {
			t.Errorf("%s: scheme %q, want %q", c.name, got, c.scheme)
		}
		if got := s.requestHost(c.r); got != c.host {
			t.Errorf("%s: host %q, want %q", c.name, got, c.host)
		}
	}
	tlsReq := proxyRequest("127.0.0.1:1", headerForwardedProto, "http")
	tlsReq.TLS = &tls.ConnectionState{}
	if s.requestScheme(tlsReq) != "https" {
		t.Error("TLS request is not https")
	}
	if s.isTrusted(netip.Addr{}) {
		t.Error("invalid address trusted")
	}
}
