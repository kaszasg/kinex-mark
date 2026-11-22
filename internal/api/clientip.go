package api

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// Forwarding headers set by a reverse proxy (6.2, 3.4 APP_TRUSTED_PROXY).
const (
	headerForwardedFor   = "X-Forwarded-For"
	headerForwardedProto = "X-Forwarded-Proto"
	headerForwardedHost  = "X-Forwarded-Host"
)

// untrustedForwardWarning is logged once per server when a peer outside
// APP_TRUSTED_PROXY sends X-Forwarded-* headers (6.2).
const untrustedForwardWarning = "request has X-Forwarded-* headers from an untrusted peer; set APP_TRUSTED_PROXY and APP_BASE_URL"

// peerAddr returns the address of the directly connected peer, unmapped
// (an IPv4-mapped IPv6 address becomes IPv4). The zero Addr when RemoteAddr
// cannot be parsed.
func peerAddr(r *http.Request) netip.Addr {
	if ap, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		return ap.Addr().Unmap()
	}
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = h
	}
	if a, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
		return a.WithZone("").Unmap()
	}
	return netip.Addr{}
}

// isTrusted reports whether a is inside one of the trusted proxy prefixes.
func (s *server) isTrusted(a netip.Addr) bool {
	if !a.IsValid() {
		return false
	}
	a = a.WithZone("").Unmap()
	for _, p := range s.cfg.TrustedProxies {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// hasForwardHeaders reports whether r carries any X-Forwarded-* header.
func hasForwardHeaders(r *http.Request) bool {
	h := r.Header
	return h.Get(headerForwardedFor) != "" || h.Get(headerForwardedProto) != "" || h.Get(headerForwardedHost) != ""
}

// fromTrustedProxy reports whether r came directly from a trusted proxy, so
// that its X-Forwarded-* headers count. Headers from any other peer are
// ignored; the first such request of the server logs one warning with the
// peer address (never the header values).
func (s *server) fromTrustedProxy(r *http.Request) bool {
	peer := peerAddr(r)
	if s.isTrusted(peer) {
		return true
	}
	if hasForwardHeaders(r) {
		s.forwardWarn.Do(func() {
			s.log.Warn(untrustedForwardWarning, "peer", peer.String())
		})
	}
	return false
}

// clientAddr returns the client address of r (unmapped). Behind a trusted
// proxy it is the first address of X-Forwarded-For, read from the right,
// that is not itself a trusted proxy; when every listed address is trusted,
// the leftmost one. An unparsable entry ends the walk at the last good
// address. Without a trusted proxy it is the peer address.
func (s *server) clientAddr(r *http.Request) netip.Addr {
	client := peerAddr(r)
	if !s.fromTrustedProxy(r) {
		return client
	}
	var hops []string
	for _, v := range r.Header.Values(headerForwardedFor) {
		hops = append(hops, strings.Split(v, ",")...)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			break
		}
		client = a.WithZone("").Unmap()
		if !s.isTrusted(client) {
			break
		}
	}
	return client
}

// clientIP returns the client address as text for sessions, challenges and
// the audit log; "" when unknown.
func (s *server) clientIP(r *http.Request) string {
	a := s.clientAddr(r)
	if !a.IsValid() {
		return ""
	}
	return a.String()
}

// firstHeaderValue returns the first comma-separated element of a header,
// trimmed.
func firstHeaderValue(r *http.Request, name string) string {
	v := r.Header.Get(name)
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

// requestScheme returns "https" when the server itself runs TLS or a
// trusted proxy reports X-Forwarded-Proto: https, "http" otherwise.
func (s *server) requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if s.fromTrustedProxy(r) {
		switch strings.ToLower(firstHeaderValue(r, headerForwardedProto)) {
		case "https":
			return "https"
		case "http":
			return "http"
		}
	}
	return "http"
}

// requestHost returns the host the client addressed: X-Forwarded-Host from
// a trusted proxy when present, otherwise r.Host.
func (s *server) requestHost(r *http.Request) string {
	if s.fromTrustedProxy(r) {
		if h := firstHeaderValue(r, headerForwardedHost); h != "" {
			return h
		}
	}
	return r.Host
}
