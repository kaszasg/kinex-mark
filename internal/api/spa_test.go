package api

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

// cspHash returns the CSP source of an inline script body.
func cspHash(body string) string {
	sum := sha256.Sum256([]byte(body))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

func TestInlineScriptHashes(t *testing.T) {
	t.Parallel()
	want := []string{cspHash(testInlineScript1), cspHash(testInlineScript2)}
	cases := []struct {
		name string
		doc  string
		want []string
	}{
		{"test index", testIndexHTML, want},
		{"CRLF line breaks", strings.ReplaceAll(testIndexHTML, "\n", "\r\n"), want},
		{"lone CR", "<script>a\rb</script>", []string{cspHash("a\nb")}},
		{"upper-case tag", "<SCRIPT>x()</SCRIPT>", []string{cspHash("x()")}},
		{"external script skipped", `<script src="/a.js"></script><script type="module" SRC="/b.js">ignored()</script>`, nil},
		{"empty script skipped", "<script></script><script>y()</script>", []string{cspHash("y()")}},
		{"duplicates once", "<script>z()</script><script>z()</script>", []string{cspHash("z()")}},
		{"entities stay raw", "<script>if (a &amp;&amp; b) {}</script>", []string{cspHash("if (a &amp;&amp; b) {}")}},
		{"text outside scripts ignored", "<p>hello</p><style>p{}</style><div>text</div>", nil},
		{"attribute without src", `<script type="text/javascript" nonce="n">w()</script>`, []string{cspHash("w()")}},
		{"no scripts", "", nil},
	}
	for _, c := range cases {
		got, err := inlineScriptHashes([]byte(c.doc))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// failingReader returns an error after its data.
type failingReader struct{ data string }

func (r *failingReader) Read(p []byte) (int, error) {
	if r.data == "" {
		return 0, errors.New("disk on fire")
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestReadScriptHashesReaderError(t *testing.T) {
	t.Parallel()
	_, err := readScriptHashes(&failingReader{data: "<html><script>a()"})
	if err == nil || !strings.Contains(err.Error(), "disk on fire") {
		t.Fatalf("err = %v", err)
	}
}

// TestInlineScriptHashesRealBuild checks the real SvelteKit build when it
// exists (web/build is produced by npm run build before every Go build path,
// 10.1): exactly the two inline scripts (theme bootstrap and SvelteKit start)
// are hashed.
func TestInlineScriptHashesRealBuild(t *testing.T) {
	t.Parallel()
	doc, err := os.ReadFile(filepath.Join("..", "..", "web", "build", "index.html"))
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("web/build/index.html not built")
	}
	if err != nil {
		t.Fatal(err)
	}
	hashes, err := inlineScriptHashes(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 2 {
		t.Fatalf("got %d inline script hashes, want 2: %v", len(hashes), hashes)
	}
	// Recompute independently: the text between each <script> without src
	// and the next </script>.
	var want []string
	rest := strings.ReplaceAll(string(doc), "\r\n", "\n")
	for {
		i := strings.Index(rest, "<script>")
		if i < 0 {
			break
		}
		rest = rest[i+len("<script>"):]
		j := strings.Index(rest, "</script>")
		want = append(want, cspHash(rest[:j]))
		rest = rest[j:]
	}
	if !reflect.DeepEqual(hashes, want) {
		t.Fatalf("hashes %v, want %v", hashes, want)
	}
	csp := buildCSP(hashes, false)
	if !strings.Contains(csp, "script-src 'self' "+want[0]+" "+want[1]+";") {
		t.Fatalf("CSP %q lacks the build hashes", csp)
	}
}

func TestSPAServing(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t)
	cases := []struct {
		method, path string
		status       int
		body         string // exact body; "" skips the check
		cache        string // exact Cache-Control; "-" means absent
		contentType  string // prefix; "" skips the check
	}{
		{http.MethodGet, "/", 200, testIndexHTML, "no-cache", "text/html; charset=utf-8"},
		{http.MethodGet, "/index.html", 200, testIndexHTML, "no-cache", "text/html; charset=utf-8"},
		{http.MethodGet, "/login", 200, testIndexHTML, "no-cache", "text/html; charset=utf-8"},
		{http.MethodGet, "/settings/security?tab=1", 200, testIndexHTML, "no-cache", "text/html; charset=utf-8"},
		{http.MethodGet, "/fonts", 200, testIndexHTML, "no-cache", "text/html; charset=utf-8"},
		{http.MethodPost, "/login", 200, testIndexHTML, "no-cache", "text/html; charset=utf-8"},
		{http.MethodPost, "/healthz", 200, testIndexHTML, "no-cache", "text/html; charset=utf-8"},
		{http.MethodGet, "/_app/immutable/x.js", 200, testAssetJS, immutableCache, ""},
		{http.MethodGet, "/fonts/HankenGrotesk-Test.woff2", 200, testFontBytes, immutableCache, ""},
		{http.MethodGet, "/robots.txt", 200, testRobotsTxt, "-", "text/plain"},
		{http.MethodGet, "/favicon.svg", 200, "", "-", ""},
		{http.MethodGet, "/_app/immutable/missing.js", 404, "", "-", "text/plain"},
		{http.MethodGet, "/_app/version.json", 404, "", "-", "text/plain"},
		{http.MethodGet, "/_app/immutable", 404, "", "-", "text/plain"},
		{http.MethodGet, "/etc/passwd", 200, testIndexHTML, "no-cache", "text/html"},
	}
	for _, tc := range cases {
		res := c.do(tc.method, tc.path, nil)
		body := expectStatus(t, res, tc.status)
		if tc.body != "" && string(body) != tc.body {
			t.Errorf("%s %s: body %q, want %q", tc.method, tc.path, body, tc.body)
		}
		got := res.Header.Get("Cache-Control")
		if (tc.cache == "-" && got != "") || (tc.cache != "-" && got != tc.cache) {
			t.Errorf("%s %s: Cache-Control %q, want %q", tc.method, tc.path, got, tc.cache)
		}
		if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, tc.contentType) {
			t.Errorf("%s %s: Content-Type %q, want prefix %q", tc.method, tc.path, ct, tc.contentType)
		}
		if res.Header.Get("Content-Security-Policy") == "" {
			t.Errorf("%s %s: no CSP", tc.method, tc.path)
		}
	}
}

func TestSPAHead(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	res := e.client(t).do(http.MethodHead, "/links", nil)
	if body := expectStatus(t, res, http.StatusOK); len(body) != 0 {
		t.Fatalf("HEAD body %q", body)
	}
	if res.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("Cache-Control %q", res.Header.Get("Cache-Control"))
	}
}

func TestNewSPAErrors(t *testing.T) {
	t.Parallel()
	if _, err := newSPA(nil); err == nil {
		t.Error("nil UI accepted")
	}
	if _, err := newSPA(fstest.MapFS{}); err == nil || !strings.Contains(err.Error(), "index.html") {
		t.Errorf("missing index.html: %v", err)
	}
}
