package fetch

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

var (
	pngBytes  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10\x00\x00\x00\x10\x08\x06\x00\x00\x00")
	gifBytes  = []byte("GIF89a\x10\x00\x10\x00\x80\x00\x00")
	jpegBytes = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	webpBytes = []byte("RIFF\x24\x00\x00\x00WEBPVP8 \x18\x00\x00\x00")
	icoBytes  = []byte("\x00\x00\x01\x00\x01\x00\x10\x10\x00\x00\x01\x00\x20\x00")
	svgBytes  = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect width="16" height="16"/></svg>`)
)

func TestValidateIcon(t *testing.T) {
	t.Parallel()
	const limit = 64 << 10
	cases := []struct {
		name string
		data []byte
		want string // "" = rejected
	}{
		{"png", pngBytes, "image/png"},
		{"gif", gifBytes, "image/gif"},
		{"jpeg", jpegBytes, "image/jpeg"},
		{"webp", webpBytes, "image/webp"},
		{"ico", icoBytes, "image/x-icon"},
		{"svg", svgBytes, "image/svg+xml"},
		{"svg with xml declaration", []byte("\xef\xbb\xbf<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!-- c -->\n<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>\n"), "image/svg+xml"},
		{"svg without namespace", []byte("<SVG></SVG>"), ""},
		{"svg with encoded javascript url", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><use href="&#106;avascript:alert(1)"/></svg>`), ""},
		{"svg with foreignObject", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><p xmlns="http://www.w3.org/1999/xhtml">x</p></foreignObject></svg>`), ""},
		{"svg with script", []byte(`<svg><script>alert(1)</script></svg>`), ""},
		{"svg with javascript url", []byte(`<svg><a href="JavaScript:alert(1)"><rect/></a></svg>`), ""},
		{"svg with event attribute", []byte(`<svg onload="alert(1)"></svg>`), ""},
		{"svg with slash event attribute", []byte(`<svg/OnLoad = "alert(1)"></svg>`), ""},
		{"xml without svg", []byte(`<?xml version="1.0"?><html></html>`), ""},
		{"html disguised", []byte(`<!doctype html><html><body>not an icon</body></html>`), ""},
		{"text", []byte("hello"), ""},
		{"bmp", []byte("BM\x00\x00\x00\x00\x00\x00\x00\x00\x36\x00\x00\x00"), ""},
		{"empty", nil, ""},
		{"too large", append(append([]byte{}, pngBytes...), make([]byte, limit)...), ""},
	}
	for _, c := range cases {
		icon := validateIcon(c.data, limit)
		switch {
		case c.want == "" && icon != nil:
			t.Errorf("%s: accepted as %s", c.name, icon.ContentType)
		case c.want != "" && icon == nil:
			t.Errorf("%s: rejected, want %s", c.name, c.want)
		case icon != nil:
			sum := sha256.Sum256(c.data)
			if icon.ContentType != c.want || icon.SHA256 != hex.EncodeToString(sum[:]) || !bytes.Equal(icon.Data, c.data) {
				t.Errorf("%s: got %s %s", c.name, icon.ContentType, icon.SHA256)
			}
		}
	}
}

func TestDecodeDataIcon(t *testing.T) {
	t.Parallel()
	enc := base64.StdEncoding.EncodeToString(pngBytes)
	cases := []struct {
		ref  string
		want []byte
	}{
		{"data:image/png;base64," + enc, pngBytes},
		{"DATA:Image/PNG;BASE64," + enc, pngBytes},
		{"data:image/png;base64," + enc[:10] + "\n " + enc[10:], pngBytes},
		{"data:image/png;base64," + strings.TrimRight(enc, "="), pngBytes},
		{"data:image/png;base64," + strings.ReplaceAll(enc, "/", "%2F"), pngBytes},
		{"data:image/svg+xml,<svg></svg>", nil},
		{"data:text/plain;base64," + enc, nil},
		{"data:image/png;base64", nil},
		{"data:image/png;base64,***", nil},
		{"data:image/png;base64," + strings.Repeat("A", 4*base64.StdEncoding.EncodedLen(1024)), nil},
	}
	for _, c := range cases {
		got := decodeDataIcon(c.ref, 1024)
		if !bytes.Equal(got, c.want) {
			t.Errorf("decodeDataIcon(%.40q) = %d bytes, want %d", c.ref, len(got), len(c.want))
		}
	}
}

func TestSizeDistance(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]int{
		"16x16":         16,
		"48x48 16x16":   16,
		"30x30 64x64":   2,
		"any":           sizeUnknown,
		"bogus":         sizeUnknown + 1,
		"0x0":           sizeUnknown + 1,
		"":              sizeUnknown + 1,
		"192x192 any":   160,
		"180x180 33x33": 1,
	} {
		if got := sizeDistance(in); got != want {
			t.Errorf("sizeDistance(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestIconCandidatesOrder(t *testing.T) {
	t.Parallel()
	final, _ := url.Parse("https://example.com/dir/page.html")
	base, _ := url.Parse("https://cdn.example.com/static/")
	p := page{final: final, base: base, links: []linkTag{
		{rel: []string{"apple-touch-icon"}, href: "apple.png"},
		{rel: []string{"icon"}, href: "big.png", sizes: "192x192"},
		{rel: []string{"icon"}, href: "vector.svg", sizes: "any", typ: "image/svg+xml"},
		{rel: []string{"icon"}, href: "small.png", sizes: "16x16"},
		{rel: []string{"shortcut", "icon"}, href: "plain.ico"},
		{rel: []string{"icon"}, href: "/abs/32.png", sizes: "16x16 32x32"},
		{rel: []string{"stylesheet"}, href: "style.css"},
		{rel: []string{"icon"}, href: ""},
		{rel: []string{"icon"}, href: "javascript:alert(1)"},
		{rel: []string{"icon"}, href: "http://[::1"},
		{rel: []string{"apple-touch-icon-precomposed"}, href: "apple.png"},
		{rel: []string{"icon"}, href: "data:image/png;base64,AAAA", sizes: "48x48"},
		{rel: []string{"icon"}, href: "https://other.example/x.png#frag", sizes: "any"},
	}}
	want := []string{
		"https://cdn.example.com/abs/32.png",
		"https://cdn.example.com/static/plain.ico",
		"https://cdn.example.com/static/vector.svg",
		"https://cdn.example.com/static/small.png",
		"data:image/png;base64,AAAA",
		"https://cdn.example.com/static/big.png",
		"https://other.example/x.png",
		"https://cdn.example.com/static/apple.png",
		"https://example.com/favicon.ico",
	}
	if got := p.iconCandidates(); !reflect.DeepEqual(got, want) {
		t.Fatalf("iconCandidates =\n%q\nwant\n%q", got, want)
	}
}
