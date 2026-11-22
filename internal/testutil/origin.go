package testutil

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"strings"
	"testing"
)

// NewOrigin starts a fake HTTP origin on the loopback interface (9.2.4).
// routes maps net/http.ServeMux patterns to handlers; the server is closed
// when the test ends. Tests reach it with
// Policy{AllowPrivate: true, AllowLoopback: true} and
// Options.DialAllow = AllowOnly(srv), so no other loopback port is reachable.
func NewOrigin(t testing.TB, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, h := range routes {
		mux.HandleFunc(pattern, h)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// AllowOnly returns a dial filter that admits only the origin's own address
// and port.
func AllowOnly(srv *httptest.Server) func(netip.AddrPort) bool {
	want, err := netip.ParseAddrPort(srv.Listener.Addr().String())
	if err != nil {
		panic(fmt.Sprintf("testutil: origin address %q: %v", srv.Listener.Addr(), err))
	}
	want = netip.AddrPortFrom(want.Addr().Unmap(), want.Port())
	return func(ap netip.AddrPort) bool {
		return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port()) == want
	}
}

// HTMLPage serves a small HTML document. title is written as raw bytes (it
// may hold pre-encoded or entity text); charset, when not empty, goes into
// the Content-Type header; each element of headExtra (icon links, meta
// tags) is written verbatim into the head after the title.
func HTMLPage(title, charset string, headExtra ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		ct := "text/html"
		if charset != "" {
			ct += "; charset=" + charset
		}
		w.Header().Set("Content-Type", ct)
		var b strings.Builder
		b.WriteString("<!doctype html><html><head><title>")
		b.WriteString(title)
		b.WriteString("</title>")
		for _, s := range headExtra {
			b.WriteString(s)
		}
		b.WriteString("</head><body><p>body</p></body></html>")
		_, _ = w.Write([]byte(b.String()))
	}
}

// Redirect answers n redirects in a row (302 to the same path with a hop
// counter) and then a page titled "Redirected".
func Redirect(n int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		left := n
		if s := r.URL.Query().Get("hop"); s != "" {
			left, _ = strconv.Atoi(s)
		}
		if left > 0 {
			http.Redirect(w, r, r.URL.Path+"?hop="+strconv.Itoa(left-1), http.StatusFound)
			return
		}
		HTMLPage("Redirected", "utf-8")(w, r)
	}
}

// RedirectTo answers a 302 redirect to target.
func RedirectTo(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", target)
		w.WriteHeader(http.StatusFound)
	}
}

// Slow blocks until the request's context ends (client timeout or close).
func Slow() http.HandlerFunc {
	return func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}
}

// Big serves an HTML document of at least size bytes: the head holds a
// comment filler of size bytes, followed by <title>Big page</title>, so the
// title lies beyond a size limit smaller than size.
func Big(size int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html><html><head><!--"))
		chunk := []byte(strings.Repeat("x", 4096))
		for left := size; left > 0; left -= len(chunk) {
			if left < len(chunk) {
				chunk = chunk[:left]
			}
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
		_, _ = w.Write([]byte("--><title>Big page</title></head><body></body></html>"))
	}
}

// Icon serves data with the given Content-Type.
func Icon(contentType string, data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		_, _ = w.Write(data)
	}
}

// Status answers code with a small HTML body.
func Status(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(code)
		_, _ = fmt.Fprintf(w, "<!doctype html><title>%s</title>", html.EscapeString(http.StatusText(code)))
	}
}
