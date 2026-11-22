package fetch

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	t.Parallel()
	// 2040 bytes of input with spaces and an accented path: percent-encoding
	// makes the serialised URL longer than 2048 bytes.
	longAccented := "https://example.com/" + strings.Repeat("á ", 2020/3) + "x"
	for len(longAccented) < 2040 {
		longAccented += "x"
	}
	cases := []struct {
		in, want, host, code string
	}{
		{"", "", "", URLRequired},
		{"   \t ", "", "", URLRequired},
		{"https://example.com/" + strings.Repeat("a", 2030), "", "", URLTooLong},
		{longAccented, "", "", URLTooLong},
		{"example.com", "", "", URLInvalidFormat},
		{"http://", "", "", URLInvalidFormat},
		{"http:///path", "", "", URLInvalidFormat},
		{"javascript:alert(1)", "", "", URLInvalidFormat},
		{"http://exa mple.com/", "", "", URLInvalidFormat},
		{"ftp://example.com/file", "", "", URLNotAllowed},
		{"file://server/share", "", "", URLNotAllowed},
		{"https://user:secret@example.com/", "", "", URLCredentialsInURL},
		{"https://user@example.com/", "", "", URLCredentialsInURL},
		{"http://:80/", "", "", URLInvalidFormat},
		{"  HTTPS://Example.COM  ", "https://example.com/", "example.com", ""},
		{"http://example.com:80/a", "http://example.com/a", "example.com", ""},
		{"https://example.com:443", "https://example.com/", "example.com", ""},
		{"http://example.com:443/", "http://example.com:443/", "example.com", ""},
		{"https://example.com:8443/x", "https://example.com:8443/x", "example.com", ""},
		{"https://example.com?b=2&a=1#Frag", "https://example.com/?b=2&a=1#Frag", "example.com", ""},
		{"https://example.com/Path/To?Q=A%20B", "https://example.com/Path/To?Q=A%20B", "example.com", ""},
		{"https://@example.com/", "https://example.com/", "example.com", ""},
		{"http://[::1]:80/", "http://[::1]/", "::1", ""},
		{"http://[2001:DB8::1]:8080/", "http://[2001:db8::1]:8080/", "2001:db8::1", ""},
		{"https://example.com/a b", "https://example.com/a%20b", "example.com", ""},
		{"http://example.com:/", "http://example.com/", "example.com", ""},
	}
	for _, c := range cases {
		got, host, err := NormalizeURL(c.in)
		if c.code != "" {
			var ue *URLError
			if !errors.As(err, &ue) || ue.Code != c.code {
				t.Errorf("NormalizeURL(%.60q) error = %v, want %s", c.in, err, c.code)
			}
			continue
		}
		if err != nil || got != c.want || host != c.host {
			t.Errorf("NormalizeURL(%q) = %q, %q, %v; want %q, %q", c.in, got, host, err, c.want, c.host)
		}
	}
	if len(longAccented) != 2040 {
		t.Fatalf("test input is %d bytes, want 2040", len(longAccented))
	}
}

func TestNormalizeURLLimitAfterSerialisation(t *testing.T) {
	t.Parallel()
	// Exactly 2048 bytes of plain ASCII is accepted.
	in := "https://example.com/" + strings.Repeat("a", 2048-len("https://example.com/"))
	got, _, err := NormalizeURL(in)
	if err != nil || len(got) != 2048 {
		t.Fatalf("NormalizeURL(2048 bytes) = %d bytes, %v", len(got), err)
	}
	if (&URLError{Code: URLTooLong}).Error() != "invalid url: too_long" {
		t.Error("unexpected URLError text")
	}
}

func TestExtractURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"https://example.com/a", "https://example.com/a", true},
		{"# comment\r\nhttps://example.com/list\r\nhttps://other.example/", "https://example.com/list", true},
		{"#https://comment.example/\nhttp://real.example/", "http://real.example/", true},
		{"Look at this: https://example.com/page.", "https://example.com/page", true},
		{"(see https://example.com/x)", "https://example.com/x", true},
		{"https://en.wikipedia.org/wiki/Go_(language)", "https://en.wikipedia.org/wiki/Go_(language)", true},
		{"https://en.wikipedia.org/wiki/Go_(language)).", "https://en.wikipedia.org/wiki/Go_(language)", true},
		{"<https://example.com/angle>", "https://example.com/angle", true},
		{"\"https://example.com/q\",", "https://example.com/q", true},
		{"HTTPS://EXAMPLE.COM/UP", "HTTPS://EXAMPLE.COM/UP", true},
		{"http://[::1]:8080/", "http://[::1]:8080/", true},
		{"text first\nthen https://example.com/second", "https://example.com/second", true},
		{"example.com", "https://example.com", true},
		{"example.com/path?q=1", "https://example.com/path?q=1", true},
		{"sub.example.co.uk:8080/x", "https://sub.example.co.uk:8080/x", true},
		{"példa.hu", "https://példa.hu", true},
		{"example.com, and more", "https://example.com", true},
		{"https://. example.org", "", false},
		{"just some words", "", false},
		{"", "", false},
		{"# only a comment", "", false},
		{"localhost", "", false},
		{"1.2", "", false},
		{"https://", "", false},
	}
	for _, c := range cases {
		got, ok := ExtractURL(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ExtractURL(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}
