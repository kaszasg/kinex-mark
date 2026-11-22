package fetch

import (
	"errors"
	"net/netip"
	"strconv"
	"syscall"
)

// addrClass is the SSRF category of an address (6.8).
type addrClass int

const (
	classPublic   addrClass = iota
	classPrivate            // opened by Policy.AllowPrivate
	classLoopback           // opened by Policy.AllowLoopback only
	classBlocked            // never allowed
)

func mustPrefixes(ss ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(ss))
	for i, s := range ss {
		out[i] = netip.MustParsePrefix(s)
	}
	return out
}

var (
	// Always blocked, AllowPrivate and AllowLoopback included.
	alwaysBlocked = mustPrefixes(
		"0.0.0.0/8", "::/128", "224.0.0.0/4", "ff00::/8", "240.0.0.0/4", "255.255.255.255/32",
		// cloud metadata endpoints
		"169.254.169.254/32", "169.254.170.2/32", "100.100.100.200/32", "168.63.129.16/32",
		"fd00:ec2::254/128", "fd20:ce::254/128",
		// documentation ranges
		"192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24", "2001:db8::/32",
	)
	// Opened only by AllowLoopback (APP_ALLOW_LOOPBACK_FETCH).
	loopbackRanges = mustPrefixes("127.0.0.0/8", "::1/128", "169.254.0.0/16", "fe80::/10")
	// Opened by AllowPrivate (allowPrivateFetch).
	privateRanges = mustPrefixes(
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10", "fc00::/7",
		"192.0.0.0/24", "198.18.0.0/15",
	)

	nat64WellKnown = netip.MustParsePrefix("64:ff9b::/96")
	nat64Local     = netip.MustParsePrefix("64:ff9b:1::/48")
	sixToFour      = netip.MustParsePrefix("2002::/16")
	teredo         = netip.MustParsePrefix("2001::/32")
	ipv4Compatible = netip.MustParsePrefix("::/96")
)

func inAny(ip netip.Addr, ps []netip.Prefix) bool {
	for _, p := range ps {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func classOf(ip netip.Addr) addrClass {
	switch {
	case inAny(ip, alwaysBlocked):
		return classBlocked
	case inAny(ip, loopbackRanges):
		return classLoopback
	case inAny(ip, privateRanges):
		return classPrivate
	default:
		return classPublic
	}
}

func v4At(b [16]byte, idx ...int) netip.Addr {
	return netip.AddrFrom4([4]byte{b[idx[0]], b[idx[1]], b[idx[2]], b[idx[3]]})
}

// embeddedIPv4 returns the IPv4 addresses an IPv6 address carries: NAT64
// (64:ff9b::/96 and 64:ff9b:1::/48, the last 32 bits), 6to4, Teredo (client
// address, bitwise negated) and IPv4-compatible (::/96).
func embeddedIPv4(ip netip.Addr) []netip.Addr {
	if !ip.Is6() {
		return nil
	}
	b := ip.As16()
	var out []netip.Addr
	switch {
	case nat64WellKnown.Contains(ip), nat64Local.Contains(ip):
		out = append(out, v4At(b, 12, 13, 14, 15))
	case sixToFour.Contains(ip):
		out = append(out, v4At(b, 2, 3, 4, 5))
	case teredo.Contains(ip):
		out = append(out, netip.AddrFrom4([4]byte{^b[12], ^b[13], ^b[14], ^b[15]}))
	case ipv4Compatible.Contains(ip):
		// :: and ::1 are IPv6 addresses of their own, classified directly.
		v4 := v4At(b, 12, 13, 14, 15)
		if v4 != netip.AddrFrom4([4]byte{0, 0, 0, 0}) && v4 != netip.AddrFrom4([4]byte{0, 0, 0, 1}) {
			out = append(out, v4)
		}
	}
	return out
}

func allowedClass(p Policy, c addrClass) bool {
	switch c {
	case classPublic:
		return true
	case classPrivate:
		return p.AllowPrivate
	case classLoopback:
		return p.AllowLoopback
	default:
		return false
	}
}

// CheckIP returns nil when the policy allows dialling ip, otherwise an
// *Error with Code "blocked_address" (6.8). IPv4-mapped IPv6 addresses are
// unmapped first; IPv4 addresses embedded in IPv6 (NAT64, 6to4, Teredo,
// IPv4-compatible) are checked as well.
func CheckIP(p Policy, ip netip.Addr) error {
	ip = ip.Unmap().WithZone("")
	if !ip.IsValid() {
		return newError(codeBlockedAddress, false, errors.New("invalid address"))
	}
	if !allowedClass(p, classOf(ip)) {
		return newError(codeBlockedAddress, false, errors.New("address not allowed"))
	}
	for _, v4 := range embeddedIPv4(ip) {
		if !allowedClass(p, classOf(v4)) {
			return newError(codeBlockedAddress, false, errors.New("embedded address not allowed"))
		}
	}
	return nil
}

// allowedPort reports whether a destination port may be dialled:
// 80, 443 or 1024-65535 (6.8).
func allowedPort(port uint16) bool {
	return port == 80 || port == 443 || port >= 1024
}

func allowedPortString(s string) bool {
	n, err := strconv.ParseUint(s, 10, 16)
	return err == nil && allowedPort(uint16(n))
}

// control runs before every connection with the address actually dialled,
// so DNS rebinding and redirects cannot bypass the policy (6.8).
func (f *Fetcher) control(_, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return newError(codeBlockedAddress, false, err)
	}
	if !allowedPort(ap.Port()) {
		return newError(codeBlockedPort, false, errors.New("port not allowed"))
	}
	if err := CheckIP(f.opts.Policy(), ap.Addr()); err != nil {
		return err
	}
	if f.opts.DialAllow != nil && !f.opts.DialAllow(netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())) {
		return newError(codeBlockedAddress, false, errors.New("address not allowed by the dial filter"))
	}
	return nil
}
