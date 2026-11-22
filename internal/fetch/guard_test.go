package fetch

import (
	"errors"
	"net/netip"
	"testing"
)

var policies = []Policy{
	{},
	{AllowPrivate: true},
	{AllowLoopback: true},
	{AllowPrivate: true, AllowLoopback: true},
}

func TestCheckIPTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		addr  string
		class addrClass
	}{
		// always blocked
		{"0.0.0.0", classBlocked},
		{"0.1.2.3", classBlocked},
		{"::", classBlocked},
		{"224.0.0.1", classBlocked},
		{"239.255.255.250", classBlocked},
		{"ff02::1", classBlocked},
		{"240.0.0.1", classBlocked},
		{"255.255.255.255", classBlocked},
		{"169.254.169.254", classBlocked},
		{"169.254.170.2", classBlocked},
		{"100.100.100.200", classBlocked},
		{"168.63.129.16", classBlocked},
		{"fd00:ec2::254", classBlocked},
		{"fd20:ce::254", classBlocked},
		{"192.0.2.1", classBlocked},
		{"198.51.100.7", classBlocked},
		{"203.0.113.9", classBlocked},
		{"2001:db8::1", classBlocked},
		{"::ffff:169.254.169.254", classBlocked},
		{"64:ff9b::a9fe:a9fe", classBlocked}, // NAT64 of 169.254.169.254
		{"2002:c000:201::", classBlocked},    // 6to4 of 192.0.2.1
		{"::a9fe:a9fe", classBlocked},        // IPv4-compatible metadata address
		// loopback and link-local: AllowLoopback only
		{"127.0.0.1", classLoopback},
		{"127.255.255.254", classLoopback},
		{"::1", classLoopback},
		{"169.254.1.1", classLoopback},
		{"fe80::1", classLoopback},
		{"fe80::1%eth0", classLoopback},
		{"::ffff:127.0.0.1", classLoopback},
		{"64:ff9b::7f00:1", classLoopback},
		{"64:ff9b:1::7f00:1", classLoopback},
		{"2002:7f00:1::", classLoopback},
		{"2001:0:4136:e378:8000:63bf:80ff:fffe", classLoopback}, // Teredo, client ^80ff:fffe = 127.0.0.1
		{"::7f00:1", classLoopback},
		// private: AllowPrivate
		{"10.0.0.1", classPrivate},
		{"172.16.0.1", classPrivate},
		{"172.31.255.255", classPrivate},
		{"192.168.1.1", classPrivate},
		{"100.64.0.1", classPrivate},
		{"100.127.255.255", classPrivate},
		{"fc00::1", classPrivate},
		{"fd12:3456::1", classPrivate},
		{"192.0.0.8", classPrivate},
		{"198.18.0.1", classPrivate},
		{"198.19.255.255", classPrivate},
		{"::ffff:10.0.0.1", classPrivate},
		{"64:ff9b::a00:1", classPrivate},
		{"64:ff9b:1::c0a8:101", classPrivate},
		{"2002:a00:1::", classPrivate},
		{"2001:0:4136:e378:8000:63bf:f5ff:fffe", classPrivate}, // Teredo client 10.0.0.1
		{"::a00:1", classPrivate},
		// public
		{"93.184.216.34", classPublic},
		{"1.1.1.1", classPublic},
		{"172.32.0.1", classPublic},
		{"100.128.0.1", classPublic},
		{"198.20.0.1", classPublic},
		{"2606:4700:4700::1111", classPublic},
		{"64:ff9b::808:808", classPublic},
		{"2002:808:808::", classPublic},
		{"::ffff:8.8.8.8", classPublic},
	}
	for _, c := range cases {
		ip := netip.MustParseAddr(c.addr)
		for _, p := range policies {
			want := allowedClass(p, c.class)
			err := CheckIP(p, ip)
			if (err == nil) != want {
				t.Errorf("CheckIP(%+v, %s) = %v, want allowed=%v", p, c.addr, err, want)
				continue
			}
			if err != nil {
				var fe *Error
				if !errors.As(err, &fe) || fe.Code != codeBlockedAddress || fe.Temporary {
					t.Errorf("CheckIP(%+v, %s) error = %#v, want blocked_address", p, c.addr, err)
				}
			}
		}
	}
}

func TestCheckIPPolicyMatrix(t *testing.T) {
	t.Parallel()
	// Loopback and link-local open only with AllowLoopback; AllowPrivate does
	// not open them, and AllowLoopback does not open private ranges.
	loop := netip.MustParseAddr("127.0.0.1")
	link := netip.MustParseAddr("169.254.10.10")
	priv := netip.MustParseAddr("10.1.2.3")
	if CheckIP(Policy{AllowPrivate: true}, loop) == nil || CheckIP(Policy{AllowPrivate: true}, link) == nil {
		t.Error("AllowPrivate must not open loopback or link-local")
	}
	if CheckIP(Policy{AllowLoopback: true}, loop) != nil || CheckIP(Policy{AllowLoopback: true}, link) != nil {
		t.Error("AllowLoopback must open loopback and link-local")
	}
	if CheckIP(Policy{AllowLoopback: true}, priv) == nil {
		t.Error("AllowLoopback must not open private ranges")
	}
	all := Policy{AllowPrivate: true, AllowLoopback: true}
	if CheckIP(all, netip.MustParseAddr("169.254.169.254")) == nil {
		t.Error("the metadata address must stay blocked under every policy")
	}
}

func TestCheckIPInvalid(t *testing.T) {
	t.Parallel()
	if CheckIP(Policy{AllowPrivate: true, AllowLoopback: true}, netip.Addr{}) == nil {
		t.Error("the zero address must be blocked")
	}
}

func TestAllowedPort(t *testing.T) {
	t.Parallel()
	for port, want := range map[uint16]bool{
		0: false, 21: false, 22: false, 25: false, 79: false, 80: true, 81: false, 443: true,
		444: false, 1023: false, 1024: true, 8443: true, 65535: true,
	} {
		if got := allowedPort(port); got != want {
			t.Errorf("allowedPort(%d) = %v, want %v", port, got, want)
		}
	}
	for s, want := range map[string]bool{"80": true, "443": true, "22": false, "65536": false, "x": false, "-1": false} {
		if got := allowedPortString(s); got != want {
			t.Errorf("allowedPortString(%q) = %v, want %v", s, got, want)
		}
	}
}

// TestControlDirect calls the dial hook with the addresses a rebinding DNS
// server could return; the hook sees the address actually dialled.
func TestControlDirect(t *testing.T) {
	t.Parallel()
	strict := New(Options{})
	open := New(Options{Policy: func() Policy { return Policy{AllowPrivate: true, AllowLoopback: true} }})
	filtered := New(Options{
		Policy:    func() Policy { return Policy{AllowPrivate: true, AllowLoopback: true} },
		DialAllow: func(ap netip.AddrPort) bool { return ap == netip.MustParseAddrPort("127.0.0.1:4000") },
	})
	cases := []struct {
		f       *Fetcher
		address string
		code    string // "" = allowed
	}{
		{strict, "93.184.216.34:443", ""},
		{strict, "127.0.0.1:443", codeBlockedAddress},
		{strict, "10.0.0.5:80", codeBlockedAddress},
		{strict, "[::1]:443", codeBlockedAddress},
		{strict, "[::ffff:127.0.0.1]:80", codeBlockedAddress},
		{strict, "[64:ff9b::7f00:1]:80", codeBlockedAddress},
		{strict, "93.184.216.34:22", codeBlockedPort},
		{strict, "not-an-address", codeBlockedAddress},
		{open, "127.0.0.1:8081", ""},
		{open, "169.254.169.254:80", codeBlockedAddress},
		{open, "[fd00:ec2::254]:80", codeBlockedAddress},
		{filtered, "127.0.0.1:4000", ""},
		{filtered, "[::ffff:127.0.0.1]:4000", ""},
		{filtered, "127.0.0.1:4001", codeBlockedAddress},
		{filtered, "93.184.216.34:443", codeBlockedAddress},
	}
	for _, c := range cases {
		err := c.f.control("tcp", c.address, nil)
		if c.code == "" {
			if err != nil {
				t.Errorf("control(%s) = %v, want nil", c.address, err)
			}
			continue
		}
		var fe *Error
		if !errors.As(err, &fe) || fe.Code != c.code {
			t.Errorf("control(%s) = %v, want %s", c.address, err, c.code)
		}
	}
}

func TestEmbeddedIPv4(t *testing.T) {
	t.Parallel()
	for addr, want := range map[string]string{
		"64:ff9b::7f00:1":                      "127.0.0.1",
		"64:ff9b:1::a00:1":                     "10.0.0.1",
		"2002:7f00:1::":                        "127.0.0.1",
		"2001:0:4136:e378:8000:63bf:80ff:fffe": "127.0.0.1",
		"::7f00:1":                             "127.0.0.1",
		"::1":                                  "",
		"2606:4700::1":                         "",
	} {
		got := embeddedIPv4(netip.MustParseAddr(addr))
		switch {
		case want == "" && len(got) != 0:
			t.Errorf("embeddedIPv4(%s) = %v, want none", addr, got)
		case want != "" && (len(got) != 1 || got[0].String() != want):
			t.Errorf("embeddedIPv4(%s) = %v, want %s", addr, got, want)
		}
	}
	if embeddedIPv4(netip.MustParseAddr("127.0.0.1")) != nil {
		t.Error("an IPv4 address carries no embedded address")
	}
}
