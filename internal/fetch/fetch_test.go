package fetch

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

// openPolicy reaches the loopback origin; the dial filter keeps every other
// loopback port closed.
func openPolicy() Policy { return Policy{AllowPrivate: true, AllowLoopback: true} }

func newTestFetcher(srv *httptest.Server, mods ...func(*Options)) *Fetcher {
	opts := Options{Policy: openPolicy, DialAllow: testutil.AllowOnly(srv), Timeout: 5 * time.Second}
	for _, m := range mods {
		m(&opts)
	}
	return New(opts)
}

// counter wraps a handler and counts its requests.
type counter struct{ n atomic.Int64 }

func (c *counter) wrap(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.n.Add(1)
		h(w, r)
	}
}

func fetchErr(t *testing.T, err error) *Error {
	t.Helper()
	var fe *Error
	if !errors.As(err, &fe) {
		t.Fatalf("error = %v (%T), want *fetch.Error", err, err)
	}
	return fe
}

func mustFetch(t *testing.T, f *Fetcher, rawURL string) Result {
	t.Helper()
	res, err := f.Fetch(context.Background(), rawURL, "hu")
	if err != nil {
		t.Fatalf("Fetch(%s): %v", rawURL, err)
	}
	return res
}

func TestFetchTitles(t *testing.T) {
	t.Parallel()
	// "Árvíztűrő Šíp" in windows-1250 and in ISO-8859-2 (Š differs).
	const want = "Árvíztűrő Šíp"
	const cp1250 = "\xc1rv\xedzt\xfbr\xf5 \x8a\xedp"
	const latin2 = "\xc1rv\xedzt\xfbr\xf5 \xa9\xedp"
	long := strings.Repeat("é", 600)
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{
		"/plain":         testutil.HTMLPage("Hello &amp; World &lt;3", "utf-8"),
		"/og":            testutil.HTMLPage("  ", "utf-8", `<meta property="og:title" content="OG &quot;Title&quot;">`),
		"/cp1250-header": testutil.HTMLPage(cp1250, "windows-1250"),
		"/latin2-header": testutil.HTMLPage(latin2, "ISO-8859-2"),
		"/cp1250-meta":   testutil.HTMLPage(cp1250, "", `<meta charset="windows-1250">`),
		"/latin2-meta":   testutil.HTMLPage(latin2, "", `<meta http-equiv="Content-Type" content="text/html; charset=iso-8859-2">`),
		"/utf8-nodecl":   testutil.HTMLPage(want, ""),
		"/long":          testutil.HTMLPage(long, "utf-8"),
		"/dirty":         testutil.HTMLPage("Bank‮\x01 \t login⁦", "utf-8"),
		"/notitle":       testutil.HTMLPage("", "utf-8"),
		"/xhtml": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/xhtml+xml")
			_, _ = io.WriteString(w, `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>XHTML</title></head></html>`)
		},
		"/badparam": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset")
			_, _ = io.WriteString(w, `<title>Bad parameter</title>`)
		},
		"/sniffed": func(w http.ResponseWriter, _ *http.Request) {
			w.Header()["Content-Type"] = nil // no header at all: the fetcher sniffs
			_, _ = io.WriteString(w, `<!DOCTYPE html><html><head><title>Sniffed</title></head></html>`)
		},
	})
	f := newTestFetcher(srv)
	cases := map[string]string{
		"/plain":         "Hello & World <3",
		"/og":            `OG "Title"`,
		"/cp1250-header": want,
		"/latin2-header": want,
		"/cp1250-meta":   want,
		"/latin2-meta":   want,
		"/utf8-nodecl":   want,
		"/long":          strings.Repeat("é", 500),
		"/dirty":         "Bank login",
		"/notitle":       "",
		"/xhtml":         "XHTML",
		"/sniffed":       "Sniffed",
		"/badparam":      "Bad parameter",
	}
	for path, wantTitle := range cases {
		res := mustFetch(t, f, srv.URL+path)
		if res.Title != wantTitle || res.PageErr != nil {
			t.Errorf("%s: title %q, pageErr %v; want %q", path, res.Title, res.PageErr, wantTitle)
		}
		if res.FinalURL != srv.URL+path {
			t.Errorf("%s: FinalURL = %s", path, res.FinalURL)
		}
	}
}

func TestFetchRequestHeaders(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var seen []http.Header
	record := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			seen = append(seen, r.Header.Clone())
			mu.Unlock()
			h(w, r)
		}
	}
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{
		"/start": record(testutil.RedirectTo("/page")),
		"/page":  record(testutil.HTMLPage("Page", "utf-8")),
	})
	f := newTestFetcher(srv, func(o *Options) { o.UserAgent = "KinexMark/1.2.3 (+metadata fetcher)" })
	for lang, wantLang := range map[string]string{"hu": "hu,en;q=0.5", "de": "de,en;q=0.5", "en": "en", "": "en", "fr": "en"} {
		mu.Lock()
		seen = nil
		mu.Unlock()
		res, err := f.Fetch(context.Background(), srv.URL+"/start", lang)
		if err != nil || res.Title != "Page" || res.FinalURL != srv.URL+"/page" {
			t.Fatalf("Fetch = %+v, %v", res, err)
		}
		mu.Lock()
		if len(seen) != 2 {
			t.Fatalf("saw %d requests, want 2", len(seen))
		}
		for _, h := range seen {
			if h.Get("User-Agent") != "KinexMark/1.2.3 (+metadata fetcher)" ||
				h.Get("Accept") != "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5" ||
				h.Get("Accept-Language") != wantLang || h.Get("Cookie") != "" ||
				h.Get("Authorization") != "" || h.Get("Referer") != "" {
				t.Errorf("lang %q: request headers %v", lang, h)
			}
		}
		mu.Unlock()
	}
}

func TestFetchStripsCredentialsOnRedirect(t *testing.T) {
	t.Parallel()
	var auth atomic.Value
	auth.Store("unset")
	var srv *httptest.Server
	srv = testutil.NewOrigin(t, map[string]http.HandlerFunc{
		"/start": func(w http.ResponseWriter, r *http.Request) {
			u, _ := url.Parse(srv.URL + "/target")
			u.User = url.UserPassword("user", "secret")
			testutil.RedirectTo(u.String())(w, r)
		},
		"/target": func(w http.ResponseWriter, r *http.Request) {
			auth.Store(r.Header.Get("Authorization"))
			testutil.HTMLPage("Target", "utf-8")(w, r)
		},
	})
	f := newTestFetcher(srv)
	res := mustFetch(t, f, strings.Replace(srv.URL, "http://", "http://user:pw@", 1)+"/start")
	if res.Title != "Target" || auth.Load() != "" {
		t.Fatalf("title %q, Authorization %q", res.Title, auth.Load())
	}
}

func TestFetchIcons(t *testing.T) {
	t.Parallel()
	big := append(append([]byte{}, pngBytes...), make([]byte, 64<<10)...)
	var hits sync.Map // path -> *counter
	count := func(path string, h http.HandlerFunc) http.HandlerFunc {
		c, _ := hits.LoadOrStore(path, &counter{})
		return c.(*counter).wrap(h)
	}
	hitCount := func(path string) int64 {
		c, ok := hits.Load(path)
		if !ok {
			return 0
		}
		return c.(*counter).n.Load()
	}
	dataPNG := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{
		"/order": testutil.HTMLPage("Order", "utf-8",
			`<link rel="apple-touch-icon" href="/apple.png">`,
			`<link rel="icon" sizes="16x16" href="/16.png">`,
			`<link rel="icon" sizes="32x32" href="/32.gif">`,
			`<link rel="icon" href="/plain.png">`),
		"/base": testutil.HTMLPage("Base", "utf-8",
			`<base href="/static/">`,
			`<link rel="icon" href="icon.png">`),
		"/relative/page": testutil.HTMLPage("Relative", "utf-8", `<link rel="icon" href="../rel.ico">`),
		"/data":          testutil.HTMLPage("Data", "utf-8", `<link rel="icon" href="`+dataPNG+`">`),
		"/fallback":      testutil.HTMLPage("Fallback", "utf-8"),
		"/toolarge": testutil.HTMLPage("Large", "utf-8",
			`<link rel="icon" sizes="32x32" href="/big.png">`,
			`<link rel="icon" href="/chunked-big.png">`),
		"/fake": testutil.HTMLPage("Fake", "utf-8",
			`<link rel="icon" sizes="32x32" href="/fake.png">`,
			`<link rel="icon" href="/evil.svg">`,
			`<link rel="icon" sizes="any" type="image/svg+xml" href="/good.svg">`),
		"/limit3": testutil.HTMLPage("Limit", "utf-8",
			`<link rel="icon" sizes="32x32" href="/missing1.png">`,
			`<link rel="icon" href="/missing2.png">`,
			`<link rel="icon" sizes="16x16" href="/missing3.png">`),
		"/apple.png":       count("/apple.png", testutil.Icon("image/png", pngBytes)),
		"/16.png":          count("/16.png", testutil.Icon("image/png", pngBytes)),
		"/32.gif":          count("/32.gif", testutil.Icon("image/gif", gifBytes)),
		"/plain.png":       count("/plain.png", testutil.Icon("image/png", pngBytes)),
		"/static/icon.png": count("/static/icon.png", testutil.Icon("image/png", pngBytes)),
		"/rel.ico":         count("/rel.ico", testutil.Icon("image/vnd.microsoft.icon", icoBytes)),
		"/favicon.ico":     count("/favicon.ico", testutil.Icon("image/x-icon", icoBytes)),
		"/big.png":         count("/big.png", testutil.Icon("image/png", big)),
		"/chunked-big.png": count("/chunked-big.png", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			for i := 0; i < 3; i++ {
				_, _ = w.Write(big[:30000])
				w.(http.Flusher).Flush()
			}
		}),
		"/fake.png":     count("/fake.png", testutil.Icon("image/png", []byte("<!doctype html><title>x</title>"))),
		"/evil.svg":     count("/evil.svg", testutil.Icon("image/svg+xml", []byte(`<svg><script>alert(1)</script></svg>`))),
		"/good.svg":     count("/good.svg", testutil.Icon("text/plain", svgBytes)),
		"/missing1.png": count("/missing1.png", testutil.Status(http.StatusNotFound)),
		"/missing2.png": count("/missing2.png", testutil.Status(http.StatusNotFound)),
		"/missing3.png": count("/missing3.png", testutil.Status(http.StatusNotFound)),
	})
	f := newTestFetcher(srv)

	check := func(path, wantType string, wantData []byte) {
		t.Helper()
		res := mustFetch(t, f, srv.URL+path)
		switch {
		case wantType == "" && res.Icon != nil:
			t.Errorf("%s: icon %s, want none", path, res.Icon.ContentType)
		case wantType != "" && (res.Icon == nil || res.Icon.ContentType != wantType || string(res.Icon.Data) != string(wantData)):
			t.Errorf("%s: icon %+v, want %s", path, res.Icon, wantType)
		}
	}
	check("/order", "image/gif", gifBytes)
	if hitCount("/apple.png")+hitCount("/16.png")+hitCount("/plain.png") != 0 {
		t.Error("/order: lower-ranked icons were downloaded after a valid one")
	}
	check("/base", "image/png", pngBytes)
	if hitCount("/static/icon.png") != 1 {
		t.Error("/base: icon not resolved against <base href>")
	}
	check("/relative/page", "image/x-icon", icoBytes)
	before := hitCount("/favicon.ico")
	check("/data", "image/png", pngBytes)
	if hitCount("/favicon.ico") != before {
		t.Error("/data: the data: icon should have been used without a download")
	}
	check("/fallback", "image/x-icon", icoBytes)
	if hitCount("/favicon.ico") != before+1 {
		t.Error("/fallback: /favicon.ico not requested")
	}
	check("/toolarge", "image/x-icon", icoBytes)
	if hitCount("/big.png") != 1 || hitCount("/chunked-big.png") != 1 {
		t.Error("/toolarge: both oversized candidates should have been tried")
	}
	check("/fake", "image/svg+xml", svgBytes)
	if hitCount("/fake.png") != 1 || hitCount("/evil.svg") != 1 {
		t.Error("/fake: rejected candidates should have been tried first")
	}
	favBefore := hitCount("/favicon.ico")
	check("/limit3", "", nil)
	if hitCount("/missing1.png") != 1 || hitCount("/missing2.png") != 1 || hitCount("/missing3.png") != 1 ||
		hitCount("/favicon.ico") != favBefore {
		t.Error("/limit3: at most three candidates may be downloaded")
	}
}

func TestFetchRedirects(t *testing.T) {
	t.Parallel()
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{
		"/r5":  testutil.Redirect(5),
		"/r6":  testutil.Redirect(6),
		"/ftp": testutil.RedirectTo("ftp://example.com/file"),
		"/port": func(w http.ResponseWriter, r *http.Request) {
			testutil.RedirectTo("http://127.0.0.1:25/")(w, r)
		},
	})
	f := newTestFetcher(srv)
	res := mustFetch(t, f, srv.URL+"/r5")
	if res.Title != "Redirected" || res.FinalURL != srv.URL+"/r5?hop=0" {
		t.Fatalf("5 redirects: %+v", res)
	}
	_, err := f.Fetch(context.Background(), srv.URL+"/r6", "en")
	if fe := fetchErr(t, err); fe.Code != codeTooManyRedirects || fe.Temporary {
		t.Fatalf("6 redirects: %+v", fe)
	}
	_, err = f.Fetch(context.Background(), srv.URL+"/ftp", "en")
	if fe := fetchErr(t, err); fe.Code != codeBlockedAddress {
		t.Fatalf("redirect to ftp: %+v", fe)
	}
	_, err = f.Fetch(context.Background(), srv.URL+"/port", "en")
	if fe := fetchErr(t, err); fe.Code != codeBlockedPort || fe.Temporary {
		t.Fatalf("redirect to port 25: %+v", fe)
	}
}

func TestFetchTimeout(t *testing.T) {
	t.Parallel()
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{"/slow": testutil.Slow()})
	f := newTestFetcher(srv, func(o *Options) { o.Timeout = 300 * time.Millisecond })
	start := time.Now()
	_, err := f.Fetch(context.Background(), srv.URL+"/slow", "en")
	if fe := fetchErr(t, err); fe.Code != codeTimeout || !fe.Temporary {
		t.Fatalf("slow origin: %+v", fe)
	}
	if d := time.Since(start); d > 4*time.Second {
		t.Fatalf("timeout took %v", d)
	}
}

func TestFetchHTMLLimit(t *testing.T) {
	t.Parallel()
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{
		"/big":   testutil.Big(2 << 20),
		"/small": testutil.Big(100 << 10),
		"/exact": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			doc := "<!doctype html><html><head><!--"
			doc += strings.Repeat("x", (1<<20)-len(doc)-len("-->"))
			_, _ = io.WriteString(w, doc+"-->")
		},
		"/titlefirst": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, "<!doctype html><html><head><title>Early</title><!--")
			_, _ = w.Write([]byte(strings.Repeat("y", 2<<20)))
		},
		"/favicon.ico": testutil.Icon("image/x-icon", icoBytes),
	})
	f := newTestFetcher(srv)
	res := mustFetch(t, f, srv.URL+"/big")
	if res.Title != "" || res.PageErr == nil || res.PageErr.Code != codeTooLarge || res.PageErr.Temporary {
		t.Fatalf("2 MiB page: %+v", res)
	}
	if res.Icon == nil {
		t.Error("2 MiB page: the favicon fallback should still run")
	}
	if res := mustFetch(t, f, srv.URL+"/small"); res.Title != "Big page" || res.PageErr != nil {
		t.Fatalf("100 KiB page: %+v", res)
	}
	if res := mustFetch(t, f, srv.URL+"/exact"); res.Title != "" || res.PageErr != nil {
		t.Fatalf("exactly 1 MiB without title: %+v", res)
	}
	if res := mustFetch(t, f, srv.URL+"/titlefirst"); res.Title != "Early" || res.PageErr != nil {
		t.Fatalf("truncated page with an early title: %+v", res)
	}
}

func TestFetchPageErrors(t *testing.T) {
	t.Parallel()
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{
		"/pdf":         testutil.Icon("application/pdf", []byte("%PDF-1.4 binary")),
		"/img":         testutil.Icon("image/png", pngBytes),
		"/badtype":     testutil.Icon("///", []byte("<title>x</title>")),
		"/404":         testutil.Status(http.StatusNotFound),
		"/500":         testutil.Status(http.StatusInternalServerError),
		"/503":         testutil.Status(http.StatusServiceUnavailable),
		"/429":         testutil.Status(http.StatusTooManyRequests),
		"/403":         testutil.Status(http.StatusForbidden),
		"/304":         func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotModified) },
		"/favicon.ico": testutil.Icon("image/x-icon", icoBytes),
	})
	f := newTestFetcher(srv)
	cases := []struct {
		path      string
		code      string
		temporary bool
	}{
		{"/pdf", codeNotHTML, false},
		{"/img", codeNotHTML, false},
		{"/badtype", codeNotHTML, false},
		{"/404", codeHTTPStatus, false},
		{"/403", codeHTTPStatus, false},
		{"/304", codeHTTPStatus, false},
		{"/500", codeHTTPStatus, true},
		{"/503", codeHTTPStatus, true},
		{"/429", codeHTTPStatus, true},
	}
	for _, c := range cases {
		res := mustFetch(t, f, srv.URL+c.path)
		if res.PageErr == nil || res.PageErr.Code != c.code || res.PageErr.Temporary != c.temporary {
			t.Errorf("%s: PageErr %+v, want %s temporary=%v", c.path, res.PageErr, c.code, c.temporary)
			continue
		}
		if res.Title != "" || res.Icon == nil || res.Icon.ContentType != "image/x-icon" {
			t.Errorf("%s: title %q icon %v; want the favicon fallback", c.path, res.Title, res.Icon)
		}
	}
}

// TestFetchLoopbackBlockedAtDial proves the dial-level block: with
// AllowLoopback false the loopback origin is never contacted.
func TestFetchLoopbackBlockedAtDial(t *testing.T) {
	t.Parallel()
	var c counter
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{"/": c.wrap(testutil.HTMLPage("Secret", "utf-8"))})
	for _, p := range []Policy{{}, {AllowPrivate: true}} {
		f := New(Options{Policy: func() Policy { return p }, Timeout: 5 * time.Second})
		res, err := f.Fetch(context.Background(), srv.URL+"/", "en")
		if fe := fetchErr(t, err); fe.Code != codeBlockedAddress || fe.Temporary {
			t.Fatalf("policy %+v: %+v", p, fe)
		}
		if res.Title != "" || res.Icon != nil {
			t.Fatalf("policy %+v: result %+v", p, res)
		}
	}
	if c.n.Load() != 0 {
		t.Fatalf("the blocked origin received %d requests", c.n.Load())
	}
	// localhost resolves to loopback addresses only, so it is blocked too.
	port := srv.URL[strings.LastIndex(srv.URL, ":")+1:]
	_, err := New(Options{Timeout: 5 * time.Second}).Fetch(context.Background(), "http://localhost:"+port+"/", "en")
	if fe := fetchErr(t, err); fe.Code != codeBlockedAddress && fe.Code != codeDNS {
		t.Fatalf("localhost: %+v", fe)
	}
	if c.n.Load() != 0 {
		t.Fatalf("the blocked origin received %d requests", c.n.Load())
	}
}

// TestFetchRedirectToOtherLoopbackPort: the origin redirects to another
// loopback port; both the test dial filter and the real policy block it.
func TestFetchRedirectToOtherLoopbackPort(t *testing.T) {
	t.Parallel()
	var other counter
	internal := testutil.NewOrigin(t, map[string]http.HandlerFunc{"/": other.wrap(testutil.HTMLPage("Internal admin", "utf-8"))})
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{"/": testutil.RedirectTo(internal.URL + "/")})

	// 1. Dial filter: the policy would allow loopback, AllowOnly does not.
	f := newTestFetcher(srv)
	_, err := f.Fetch(context.Background(), srv.URL+"/", "en")
	if fe := fetchErr(t, err); fe.Code != codeBlockedAddress {
		t.Fatalf("dial filter: %+v", fe)
	}

	// 2. Real policy without a dial filter: loopback is allowed for the first
	// dial (the origin) only, as if the setting changed in between; the
	// redirect needs a new dial, which the policy then refuses.
	var dials atomic.Int64
	g := New(Options{Timeout: 5 * time.Second, Policy: func() Policy {
		if dials.Add(1) == 1 {
			return openPolicy()
		}
		return Policy{AllowPrivate: true}
	}})
	_, err = g.Fetch(context.Background(), srv.URL+"/", "en")
	if fe := fetchErr(t, err); fe.Code != codeBlockedAddress {
		t.Fatalf("real policy: %+v", fe)
	}
	if other.n.Load() != 0 {
		t.Fatalf("the internal service received %d requests", other.n.Load())
	}
}

func TestFetchPolicyChangeTakesEffect(t *testing.T) {
	t.Parallel()
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{"/": testutil.HTMLPage("Hi", "utf-8")})
	var allow atomic.Bool
	allow.Store(true)
	f := newTestFetcher(srv, func(o *Options) {
		o.Policy = func() Policy { return Policy{AllowPrivate: true, AllowLoopback: allow.Load()} }
	})
	if !f.transport.DisableKeepAlives || f.transport.Proxy != nil {
		t.Fatal("the transport must disable keep-alives and ignore proxy variables")
	}
	if res := mustFetch(t, f, srv.URL+"/"); res.Title != "Hi" {
		t.Fatalf("first fetch: %+v", res)
	}
	allow.Store(false)
	_, err := f.Fetch(context.Background(), srv.URL+"/", "en")
	if fe := fetchErr(t, err); fe.Code != codeBlockedAddress {
		t.Fatalf("after the policy change: %+v", fe)
	}
}

func TestFetchIgnoresProxyEnvironment(t *testing.T) {
	// Not parallel: it reads the process environment through http.ProxyFromEnvironment.
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{"/": testutil.HTMLPage("Direct", "utf-8")})
	f := newTestFetcher(srv)
	t.Setenv("HTTP_PROXY", "http://192.0.2.1:3128")
	t.Setenv("http_proxy", "http://192.0.2.1:3128")
	if res := mustFetch(t, f, srv.URL+"/"); res.Title != "Direct" {
		t.Fatalf("fetch with proxy variables: %+v", res)
	}
}

func TestFetchBlockedPortAndBadURL(t *testing.T) {
	t.Parallel()
	f := New(Options{Policy: openPolicy})
	cases := map[string]string{
		"http://127.0.0.1:22/":     codeBlockedPort,
		"http://127.0.0.1:0/":      codeBlockedPort,
		"https://example.com:8/":   codeBlockedPort,
		"ftp://example.com/":       codeBlockedAddress,
		"http:///nohost":           codeBlockedAddress,
		"http://example.com:%zz/":  codeBlockedAddress,
		"http://169.254.169.254/":  codeBlockedAddress,
		"http://[fd00:ec2::254]/":  codeBlockedAddress,
		"http://0.0.0.0:8081/":     codeBlockedAddress,
		"http://[::]:8081/":        codeBlockedAddress,
		"http://[::ffff:0.0.0.0]/": codeBlockedAddress,
		"http://192.0.2.10:8080/x": codeBlockedAddress,
	}
	for raw, code := range cases {
		_, err := f.Fetch(context.Background(), raw, "en")
		if fe := fetchErr(t, err); fe.Code != code {
			t.Errorf("%s: %+v, want %s", raw, fe, code)
		}
	}
}

func TestFetchTLSError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewUnstartedServer(testutil.HTMLPage("TLS", "utf-8"))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0) // the expected handshake failure
	srv.StartTLS()
	t.Cleanup(srv.Close)
	f := newTestFetcher(srv)
	_, err := f.Fetch(context.Background(), srv.URL+"/", "en")
	if fe := fetchErr(t, err); fe.Code != codeTLS || fe.Temporary {
		t.Fatalf("self-signed certificate: %+v", fe)
	}
}

func TestFetchConnectionError(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	addr := netip.MustParseAddrPort(ln.Addr().String())
	f := New(Options{Policy: openPolicy, DialAllow: func(ap netip.AddrPort) bool { return ap == addr }})
	_, err = f.Fetch(context.Background(), "http://"+addr.String()+"/", "en")
	if fe := fetchErr(t, err); fe.Code != codeConnection || !fe.Temporary {
		t.Fatalf("closed connection: %+v", fe)
	}
}

func TestFetchContextCanceled(t *testing.T) {
	t.Parallel()
	srv := testutil.NewOrigin(t, map[string]http.HandlerFunc{"/": testutil.HTMLPage("x", "utf-8")})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newTestFetcher(srv).Fetch(ctx, srv.URL+"/", "en")
	if fe := fetchErr(t, err); fe.Code != codeTimeout || !fe.Temporary {
		t.Fatalf("canceled context: %+v", fe)
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestClassify(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wrap := func(err error) error { return &url.Error{Op: "Get", URL: "http://secret.example/?token=1", Err: err} }
	cases := []struct {
		err       error
		code      string
		temporary bool
	}{
		{wrap(&net.DNSError{Err: "no such host", Name: "x.invalid", IsNotFound: true}), codeDNS, true},
		{wrap(&net.OpError{Op: "dial", Err: timeoutErr{}}), codeTimeout, true},
		{wrap(context.DeadlineExceeded), codeTimeout, true},
		{wrap(io.ErrUnexpectedEOF), codeConnection, true},
		{wrap(&net.OpError{Op: "dial", Err: os.ErrPermission}), codeConnection, true},
		{wrap(newError(codeBlockedPort, false, nil)), codeBlockedPort, false},
	}
	for _, c := range cases {
		fe := classify(ctx, c.err)
		if fe.Code != c.code || fe.Temporary != c.temporary {
			t.Errorf("classify(%v) = %+v, want %s", c.err, fe, c.code)
		}
		if strings.Contains(fe.Error(), "secret.example") {
			t.Errorf("Error() leaks the URL: %q", fe.Error())
		}
		if c.code != codeBlockedPort && !errors.Is(fe, c.err) {
			t.Errorf("classify(%v) does not wrap the cause", c.err)
		}
	}
}

func TestErrorCodesMatchModel(t *testing.T) {
	t.Parallel()
	codes := []string{codeTimeout, codeDNS, codeConnection, codeTLS, codeHTTPStatus, codeTooLarge,
		codeNotHTML, codeTooManyRedirects, codeBlockedAddress, codeBlockedPort}
	if !slices.Equal(codes, model.FetchErrors) {
		t.Fatalf("fetch codes %v differ from model.FetchErrors %v", codes, model.FetchErrors)
	}
	e := &Error{Code: codeTLS, Err: errors.New("x509")}
	if e.Error() != "metadata fetch failed: tls" || errors.Unwrap(e).Error() != "x509" {
		t.Fatalf("Error() = %q", e.Error())
	}
}

func TestNewDefaults(t *testing.T) {
	t.Parallel()
	f := New(Options{})
	o := f.opts
	if o.Timeout != DefaultTimeout || o.MaxHTMLBytes != DefaultMaxHTMLBytes || o.MaxIconBytes != DefaultMaxIconBytes ||
		o.MaxRedirects != DefaultMaxRedirects || o.UserAgent != DefaultUserAgent || o.Policy() != (Policy{}) {
		t.Fatalf("defaults = %+v", o)
	}
	if f.transport.TLSHandshakeTimeout != 5*time.Second || f.transport.ResponseHeaderTimeout != 5*time.Second {
		t.Fatal("transport timeouts")
	}
	if got := acceptLanguageHeader(" HU "); got != "hu,en;q=0.5" {
		t.Fatalf("acceptLanguageHeader = %q", got)
	}
}
