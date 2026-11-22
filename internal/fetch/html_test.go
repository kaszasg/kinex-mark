package fetch

import (
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCleanText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		in, want string
		keepNL   bool
	}{
		{"empty", "", "", false},
		{"trim and collapse", "  Hello \t  World  ", "Hello World", false},
		{"C0 controls", "a\x00b\x01c\x1fd\x07", "abcd", false},
		{"DEL", "a\x7fb", "ab", false},
		{"C1 controls", "a\u0080b\u009fc\u0085d", "abc d", false},
		{"arabic letter mark", "adm؜in", "admin", false},
		{"bidi overrides", "safe‮gnp.exe‬", "safegnp.exe", false},
		{"bidi marks and isolates", "‎a‏b⁦c⁧d⁨e⁩f‪g‫h‭", "abcdefgh", false},
		{"newlines collapse without keep", "line1\nline2\r\nline3", "line1 line2 line3", false},
		{"newlines kept", "line1\nline2\r\nline3", "line1\nline2\nline3", true},
		{"spaces around newlines", "  a  \n   b  ", "a\nb", true},
		{"blank lines capped", "a\n\n\n\n\nb", "a\n\nb", true},
		{"lone CR removed as space", "a\rb", "a b", true},
		{"tab is whitespace", "a\tb", "a b", true},
		{"invalid utf8 dropped", "a\xffb\xc3", "ab", false},
		{"replacement char kept", "a�b", "a�b", false},
		{"accented text", "Árvíztűrő", "Árvíztűrő", false},
		{"nbsp is whitespace", "a  b", "a b", false},
		{"controls between spaces", "a \x00 b", "a b", false},
	}
	for _, c := range cases {
		if got := CleanText(c.in, c.keepNL); got != c.want {
			t.Errorf("%s: CleanText(%q, %v) = %q, want %q", c.name, c.in, c.keepNL, got, c.want)
		}
	}
}

func TestCleanTextTruncates(t *testing.T) {
	t.Parallel()
	got := CleanText(strings.Repeat("é", 600), false)
	if n := utf8.RuneCountInString(got); n != MaxTextRunes {
		t.Fatalf("got %d runes, want %d", n, MaxTextRunes)
	}
	// A separator never ends the result.
	got = CleanText(strings.Repeat("a", 499)+" b c", false)
	if got != strings.Repeat("a", 499) {
		t.Fatalf("truncation left %q at the end", got[len(got)-3:])
	}
	got = CleanText(strings.Repeat("a", 498)+"\n\nb", true)
	if got != strings.Repeat("a", 498) {
		t.Fatalf("truncation kept a trailing newline: %q", got[len(got)-3:])
	}
}

func TestParseHead(t *testing.T) {
	t.Parallel()
	doc := `<!doctype html><html><head>
<base href="/assets/"><base href="/ignored/">
<title>First &amp; best</title><title>Second</title>
<meta property="og:title" content="OG one"><meta property="og:title" content="OG two">
<link rel="Shortcut Icon" href=" /favicon.png " sizes="32X32" type="IMAGE/PNG">
<link rel="stylesheet" href="/s.css">
<script>document.write("<title>no</title>")</script>
</head><body><link rel="icon" href="/late.png"></body></html>`
	m, err := parseHead(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if m.title != "First & best" || m.ogTitle != "OG one" || m.baseHref != "/assets/" || !m.complete {
		t.Fatalf("parseHead = %+v", m)
	}
	if len(m.links) != 2 || m.links[0].href != "/favicon.png" || m.links[0].sizes != "32x32" ||
		m.links[0].typ != "image/png" || strings.Join(m.links[0].rel, " ") != "shortcut icon" {
		t.Fatalf("links = %+v", m.links)
	}
}

func TestParseHeadVariants(t *testing.T) {
	t.Parallel()
	// The body start ends the head; an unclosed title at EOF still counts.
	m, _ := parseHead(strings.NewReader(`<title>a<b>c</title>`))
	if m.complete || m.title != "a<b>c" {
		t.Errorf("markup inside title: %+v", m)
	}
	m, _ = parseHead(strings.NewReader(`<head><title>Unclosed`))
	if m.complete || m.title != "Unclosed" {
		t.Errorf("unclosed title: %+v", m)
	}
	m, _ = parseHead(strings.NewReader(`<head><meta name="og:title" content="By name"></head>`))
	if m.ogTitle != "By name" {
		t.Errorf("og:title by name: %+v", m)
	}
	m, _ = parseHead(strings.NewReader(`<head><title>Only head</title><body>`))
	if !m.complete || m.title != "Only head" {
		t.Errorf("body start: %+v", m)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestParseHeadReadError(t *testing.T) {
	t.Parallel()
	_, err := parseHead(io.MultiReader(strings.NewReader("<title>x"), failingReader{}))
	if err == nil || err.Error() != "boom" {
		t.Fatalf("err = %v, want boom", err)
	}
}

func TestResolveBase(t *testing.T) {
	t.Parallel()
	pageURL, _ := url.Parse("https://example.com/a/b.html")
	for href, want := range map[string]string{
		"":                       "",
		"/static/":               "https://example.com/static/",
		"sub/":                   "https://example.com/a/sub/",
		"https://cdn.example/x/": "https://cdn.example/x/",
		"javascript:alert(1)":    "",
		"data:text/html,x":       "",
		"http://[::1":            "",
		"http:":                  "",
	} {
		got := resolveBase(pageURL, href)
		if (got == nil && want != "") || (got != nil && got.String() != want) {
			t.Errorf("resolveBase(%q) = %v, want %q", href, got, want)
		}
	}
}
