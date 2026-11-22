package fetch

import (
	"net/url"
	"regexp"
	"strings"
)

// MaxURLBytes is the longest URL accepted before and after normalisation
// (5.9.10, links.url CHECK).
const MaxURLBytes = 2048

// URL error codes (field codes of 5.9.2).
const (
	URLRequired         = "required"
	URLTooLong          = "too_long"
	URLInvalidFormat    = "invalid_format"
	URLNotAllowed       = "not_allowed"
	URLCredentialsInURL = "credentials_in_url"
)

const (
	defaultHTTPPort     = "80"
	defaultHTTPSPort    = "443"
	trailingPunctuation = ".,;:!?)]}>'\""
)

// URLError is a NormalizeURL failure; Code is the field code.
type URLError struct{ Code string }

func (e *URLError) Error() string { return "invalid url: " + e.Code }

// NormalizeURL validates and normalises a link URL (5.9.10): http or https
// only, no credentials, lower-case scheme and host, default port dropped,
// empty path replaced by "/", query and fragment unchanged. host is the
// lower-case host name without the port.
func NormalizeURL(raw string) (normalized string, host string, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", &URLError{Code: URLRequired}
	}
	if len(s) > MaxURLBytes {
		return "", "", &URLError{Code: URLTooLong}
	}
	u, perr := url.Parse(s)
	if perr != nil || u.Host == "" {
		return "", "", &URLError{Code: URLInvalidFormat}
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", &URLError{Code: URLNotAllowed}
	}
	if u.User != nil {
		if u.User.String() != "" {
			return "", "", &URLError{Code: URLCredentialsInURL}
		}
		u.User = nil
	}
	host = strings.ToLower(u.Hostname())
	if host == "" {
		return "", "", &URLError{Code: URLInvalidFormat}
	}
	port := u.Port()
	if u.Scheme == "http" && port == defaultHTTPPort || u.Scheme == "https" && port == defaultHTTPSPort {
		port = ""
	}
	hostPart := host
	if strings.Contains(host, ":") {
		hostPart = "[" + host + "]"
	}
	if port != "" {
		hostPart += ":" + port
	}
	u.Host = hostPart
	if u.Path == "" && u.RawPath == "" {
		u.Path = "/"
	}
	normalized = u.String()
	if len(normalized) > MaxURLBytes {
		return "", "", &URLError{Code: URLTooLong}
	}
	return normalized, host, nil
}

var (
	httpToken   = regexp.MustCompile(`(?i)https?://\S+`)
	domainToken = regexp.MustCompile(`^[\p{L}\p{N}.-]+\.[\p{L}]{2,63}(:\d+)?(/\S*)?$`)

	openingBracket = map[byte]byte{')': '(', ']': '[', '}': '{'}
)

// ExtractURL finds the URL in dropped or pasted text (5.9.3). Lines starting
// with "#" (text/uri-list comments) are skipped. The first http(s)://
// token wins, without trailing punctuation (a closing bracket only when it
// has no opening pair in the token). Without such a token, a domain-shaped
// first token gets an "https://" prefix.
func ExtractURL(text string) (string, bool) {
	firstToken := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if firstToken == "" {
			if f := strings.Fields(line); len(f) > 0 {
				firstToken = f[0]
			}
		}
		for _, m := range httpToken.FindAllString(line, -1) {
			tok := trimTrailing(m)
			if _, rest, _ := strings.Cut(tok, "://"); rest != "" {
				return tok, true
			}
		}
	}
	if firstToken != "" {
		tok := trimTrailing(firstToken)
		if domainToken.MatchString(tok) {
			return "https://" + tok, true
		}
	}
	return "", false
}

// trimTrailing removes trailing punctuation. A closing bracket stays when
// the token contains more opening than closing ones of its kind before it,
// as in https://en.wikipedia.org/wiki/Go_(language).
func trimTrailing(tok string) string {
	for tok != "" {
		c := tok[len(tok)-1]
		if !strings.ContainsRune(trailingPunctuation, rune(c)) {
			break
		}
		if open, ok := openingBracket[c]; ok {
			if strings.Count(tok, string(open)) >= strings.Count(tok, string(c)) {
				break
			}
		}
		tok = tok[:len(tok)-1]
	}
	return tok
}
