// Package fetch is the metadata fetcher: SSRF guard, URL normalisation,
// title and icon extraction (plan 3.3.6, 6.8, 7.3).
//
// HTML is only ever read with html.NewTokenizer. The tree builder
// (html.Parse*) and html.Render must not be called on untrusted input: the
// golang.org/x/net/html advisories so far have hit them, not the tokenizer.
package fetch

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

// Default option values (6.8).
const (
	DefaultTimeout      = 10 * time.Second
	DefaultMaxHTMLBytes = 1 << 20
	DefaultMaxIconBytes = 64 << 10
	DefaultMaxRedirects = 5
	DefaultUserAgent    = "KinexMark/dev (+metadata fetcher)"

	dialTimeout           = 5 * time.Second
	tlsHandshakeTimeout   = 5 * time.Second
	responseHeaderTimeout = 5 * time.Second
	maxResponseHeaderSize = 64 << 10
	maxIconCandidates     = 3

	acceptHTML  = "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5"
	acceptImage = "image/avif,image/webp,image/png,image/svg+xml,image/*;q=0.8,*/*;q=0.5"
)

// Fetch error codes (7.3 table; identical to model.FetchErr*).
const (
	codeTimeout          = "timeout"
	codeDNS              = "dns"
	codeConnection       = "connection"
	codeTLS              = "tls"
	codeHTTPStatus       = "http_status"
	codeTooLarge         = "too_large"
	codeNotHTML          = "not_html"
	codeTooManyRedirects = "too_many_redirects"
	codeBlockedAddress   = "blocked_address"
	codeBlockedPort      = "blocked_port"
)

// Policy says which non-public address ranges may be dialled (6.8).
type Policy struct{ AllowPrivate, AllowLoopback bool }

// Options configures a Fetcher. Zero values take the defaults above.
type Options struct {
	Timeout      time.Duration // whole fetch (page and icons), default 10 s
	MaxHTMLBytes int64         // default 1 MiB
	MaxIconBytes int64         // default 64 KiB
	MaxRedirects int           // default 5
	UserAgent    string        // "KinexMark/<version> (+metadata fetcher)"
	// Policy may change at run time (system setting); it is read again on
	// every dial. Nil means Policy{} (only public addresses).
	Policy func() Policy
	// DialAllow narrows the dial check further after CheckIP. Tests only
	// (for example: only the fake origin's port); nil in production.
	DialAllow func(netip.AddrPort) bool
}

// Fetcher downloads a page's title and icon under the SSRF rules of 6.8.
// It is safe for concurrent use.
type Fetcher struct {
	opts      Options
	transport *http.Transport
	client    *http.Client
}

// Result is the outcome of a fetch that reached the page's server.
type Result struct {
	FinalURL string // page URL after redirects
	// Title is cleaned with CleanText (no control or bidi characters), at
	// most 500 runes, and may be empty.
	Title string
	Icon  *Icon // nil when no valid icon was found
	// PageErr is set when the server answered but the page itself was
	// unusable (http_status, not_html, too_large); the /favicon.ico fallback
	// ran anyway. With not_html the link counts as fetched (state ok).
	PageErr *Error
}

// Icon is a validated icon (6.8).
type Icon struct {
	Data        []byte
	ContentType string // image/png, image/gif, image/jpeg, image/webp, image/x-icon or image/svg+xml
	SHA256      string // lower-case hex of Data
}

// Error is a fetch failure. Code is one of the 7.3 codes; Temporary marks
// failures worth a retry (timeout, dns, connection, 5xx and 429).
//
// Error() never contains the URL (the wrapped *url.Error does), so it is safe
// to log; still, 7.3 asks callers to log only Code and the host.
type Error struct {
	Code      string
	Temporary bool
	Err       error
}

func (e *Error) Error() string { return "metadata fetch failed: " + e.Code }

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error { return e.Err }

func newError(code string, temporary bool, err error) *Error {
	return &Error{Code: code, Temporary: temporary, Err: err}
}

// New returns a Fetcher with the transport rules of 6.8: no proxy, no
// keep-alive (every request dials again, so the dial check always sees the
// current policy), dial, TLS and response-header timeouts of 5 s.
func New(opts Options) *Fetcher {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.MaxHTMLBytes <= 0 {
		opts.MaxHTMLBytes = DefaultMaxHTMLBytes
	}
	if opts.MaxIconBytes <= 0 {
		opts.MaxIconBytes = DefaultMaxIconBytes
	}
	if opts.MaxRedirects <= 0 {
		opts.MaxRedirects = DefaultMaxRedirects
	}
	if opts.UserAgent == "" {
		opts.UserAgent = DefaultUserAgent
	}
	if opts.Policy == nil {
		opts.Policy = func() Policy { return Policy{} }
	}
	f := &Fetcher{opts: opts}
	dialer := &net.Dialer{Timeout: dialTimeout, Control: f.control}
	f.transport = &http.Transport{
		Proxy:                  nil,
		DialContext:            dialer.DialContext,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:    tlsHandshakeTimeout,
		ResponseHeaderTimeout:  responseHeaderTimeout,
		DisableKeepAlives:      true,
		MaxResponseHeaderBytes: maxResponseHeaderSize,
	}
	f.client = &http.Client{
		Transport:     f.transport,
		CheckRedirect: f.checkRedirect,
	}
	return f
}

// Fetch downloads rawURL and extracts the title and the icon (6.8).
//
// A non-nil error is always an *Error and means the page could not be
// retrieved at all (timeout, dns, connection, tls, too_many_redirects,
// blocked_address, blocked_port); no icon was tried then. When the server
// answered, err is nil and Result.PageErr reports an unusable page.
func (f *Fetcher) Fetch(ctx context.Context, rawURL, acceptLanguage string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, f.opts.Timeout)
	defer cancel()

	u, ferr := checkTarget(rawURL)
	if ferr != nil {
		return Result{}, ferr
	}
	pg, err := f.loadPage(ctx, u, acceptLanguage)
	if err != nil {
		return Result{}, err
	}
	res := Result{FinalURL: pg.final.String(), Title: pg.title, PageErr: pg.err}
	res.Icon = f.findIcon(ctx, pg.iconCandidates())
	return res, nil
}

// page is what loadPage learned about the page.
type page struct {
	final *url.URL
	base  *url.URL
	title string
	links []linkTag
	err   *Error
}

func (f *Fetcher) loadPage(ctx context.Context, u *url.URL, lang string) (page, *Error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return page{}, newError(codeBlockedAddress, false, err)
	}
	req.Header.Set("User-Agent", f.opts.UserAgent)
	req.Header.Set("Accept", acceptHTML)
	req.Header.Set("Accept-Language", acceptLanguageHeader(lang))
	resp, err := f.client.Do(req)
	if err != nil {
		return page{}, classify(ctx, err)
	}
	defer resp.Body.Close()

	pg := page{final: resp.Request.URL, base: resp.Request.URL}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		temporary := resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests
		pg.err = newError(codeHTTPStatus, temporary, nil)
		return pg, nil
	}

	body := &io.LimitedReader{R: resp.Body, N: f.opts.MaxHTMLBytes}
	contentType := resp.Header.Get("Content-Type")
	var head []byte
	if contentType == "" {
		// Sniff like a browser; the sniffed bytes are replayed below.
		head = make([]byte, 512)
		n, rerr := io.ReadFull(body, head)
		head = head[:n]
		if rerr != nil && !errors.Is(rerr, io.EOF) && !errors.Is(rerr, io.ErrUnexpectedEOF) {
			return page{}, classify(ctx, rerr)
		}
		contentType = http.DetectContentType(head)
	}
	if !isHTML(contentType) {
		pg.err = newError(codeNotHTML, false, nil)
		return pg, nil
	}

	var src io.Reader = body
	if len(head) > 0 {
		src = io.MultiReader(bytes.NewReader(head), body)
	}
	decoded, err := charset.NewReader(src, contentType)
	if err != nil {
		return page{}, classify(ctx, err)
	}
	meta, err := parseHead(decoded)
	if err != nil {
		return page{}, classify(ctx, err)
	}
	pg.title = CleanText(meta.title, false)
	if pg.title == "" {
		pg.title = CleanText(meta.ogTitle, false)
	}
	pg.links = meta.links
	if b := resolveBase(pg.final, meta.baseHref); b != nil {
		pg.base = b
	}
	if pg.title == "" && !meta.complete && body.N == 0 {
		// The limit was reached; one more byte decides whether the page
		// was cut off.
		var probe [1]byte
		if n, _ := io.ReadFull(resp.Body, probe[:]); n > 0 {
			pg.err = newError(codeTooLarge, false, nil)
		}
	}
	return pg, nil
}

// checkRedirect limits redirects and checks scheme and port on every hop
// (6.8). It also strips credentials and the Referer from the next request.
func (f *Fetcher) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > f.opts.MaxRedirects {
		return newError(codeTooManyRedirects, false, errors.New("stopped after too many redirects"))
	}
	if ferr := checkURL(req.URL); ferr != nil {
		return ferr
	}
	req.URL.User = nil
	req.Header.Del("Referer")
	req.Header.Del("Authorization")
	req.Header.Del("Cookie")
	return nil
}

// checkTarget parses the URL to fetch and applies the scheme and port rules.
func checkTarget(rawURL string) (*url.URL, *Error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, newError(codeBlockedAddress, false, err)
	}
	if ferr := checkURL(u); ferr != nil {
		return nil, ferr
	}
	u.User = nil
	return u, nil
}

// checkURL allows http and https with a host, on port 80, 443 or 1024-65535.
func checkURL(u *url.URL) *Error {
	if u.Scheme != "http" && u.Scheme != "https" || u.Hostname() == "" {
		return newError(codeBlockedAddress, false, errors.New("only http and https URLs with a host are fetched"))
	}
	if p := u.Port(); p != "" {
		if !allowedPortString(p) {
			return newError(codeBlockedPort, false, errors.New("port not allowed"))
		}
	}
	// On some systems (Windows among them) the dialer rewrites an
	// unspecified destination (0.0.0.0, ::) to a local address before the
	// dial hook sees it, so the always-blocked unspecified literal is
	// refused here.
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && ip.Unmap().IsUnspecified() {
		return newError(codeBlockedAddress, false, errors.New("unspecified address"))
	}
	return nil
}

func isHTML(contentType string) bool {
	// A broken parameter (for example "text/html; charset") still yields the
	// media type, as in browsers.
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil && !errors.Is(err, mime.ErrInvalidMediaParameter) {
		return false
	}
	return mt == "text/html" || mt == "application/xhtml+xml"
}

// acceptLanguageHeader builds "hu,en;q=0.5" from the owner's language.
func acceptLanguageHeader(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "hu":
		return "hu,en;q=0.5"
	case "de":
		return "de,en;q=0.5"
	default:
		return "en"
	}
}

// classify maps a transport or read error to a 7.3 code.
func classify(ctx context.Context, err error) *Error {
	var fe *Error
	if errors.As(err, &fe) {
		return fe
	}
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return newError(codeTimeout, true, err)
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return newError(codeDNS, true, err)
	}
	if isTLSError(err) {
		return newError(codeTLS, false, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return newError(codeTimeout, true, err)
	}
	return newError(codeConnection, true, err)
}

func isTLSError(err error) bool {
	var (
		verifyErr    *tls.CertificateVerificationError
		recordErr    tls.RecordHeaderError
		alertErr     tls.AlertError
		unknownAuth  x509.UnknownAuthorityError
		hostnameErr  x509.HostnameError
		invalidErr   x509.CertificateInvalidError
		systemErr    x509.SystemRootsError
		constraintEr x509.ConstraintViolationError
	)
	return errors.As(err, &verifyErr) || errors.As(err, &recordErr) || errors.As(err, &alertErr) ||
		errors.As(err, &unknownAuth) || errors.As(err, &hostnameErr) || errors.As(err, &invalidErr) ||
		errors.As(err, &systemErr) || errors.As(err, &constraintEr)
}
