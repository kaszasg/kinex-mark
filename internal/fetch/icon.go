package fetch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Icon candidate tiers in the order of 6.8.
const (
	tierIcon32 = iota
	tierIconNoSizes
	tierIconSVG
	tierIconOther
	tierAppleTouch
	tierFavicon
)

type iconCandidate struct {
	ref   string // absolute http(s) URL or the raw data: URI
	tier  int
	dist  int // tierIconOther: distance of the closest size to 32 px
	order int // document order
}

const sizeUnknown = 1 << 20

// sizeDistance returns the smallest |width-32| among the sizes tokens.
func sizeDistance(sizes string) int {
	best := sizeUnknown + 1
	for _, tok := range strings.Fields(sizes) {
		if tok == "any" {
			best = min(best, sizeUnknown)
			continue
		}
		w, _, ok := strings.Cut(tok, "x")
		n, err := strconv.Atoi(w)
		if !ok || err != nil || n <= 0 {
			continue
		}
		d := n - 32
		if d < 0 {
			d = -d
		}
		best = min(best, d)
	}
	return best
}

func hasToken(tokens []string, want string) bool {
	for _, t := range tokens {
		if t == want {
			return true
		}
	}
	return false
}

// iconCandidates orders the page's icon links (6.8): icon with sizes 32x32,
// icon without sizes, icon of type SVG, any other icon by closeness to 32 px,
// apple-touch-icon, then /favicon.ico. Duplicates are dropped.
func (p page) iconCandidates() []string {
	var cands []iconCandidate
	for i, l := range p.links {
		if l.href == "" {
			continue
		}
		c := iconCandidate{order: i}
		switch {
		case hasToken(l.rel, "icon"):
			switch {
			case hasToken(strings.Fields(l.sizes), "32x32"):
				c.tier = tierIcon32
			case l.sizes == "":
				c.tier = tierIconNoSizes
			case l.typ == "image/svg+xml":
				c.tier = tierIconSVG
			default:
				c.tier = tierIconOther
				c.dist = sizeDistance(l.sizes)
			}
		case hasToken(l.rel, "apple-touch-icon"), hasToken(l.rel, "apple-touch-icon-precomposed"):
			c.tier = tierAppleTouch
		default:
			continue
		}
		ref, ok := resolveIconRef(p.base, l.href)
		if !ok {
			continue
		}
		c.ref = ref
		cands = append(cands, c)
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].tier != cands[j].tier {
			return cands[i].tier < cands[j].tier
		}
		if cands[i].dist != cands[j].dist {
			return cands[i].dist < cands[j].dist
		}
		return cands[i].order < cands[j].order
	})
	if fav, err := p.final.Parse("/favicon.ico"); err == nil {
		cands = append(cands, iconCandidate{ref: fav.String(), tier: tierFavicon})
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		if !seen[c.ref] {
			seen[c.ref] = true
			out = append(out, c.ref)
		}
	}
	return out
}

// resolveIconRef resolves href against the base. data: URIs are kept as
// they are; only http, https and data are accepted.
func resolveIconRef(base *url.URL, href string) (string, bool) {
	if len(href) >= 5 && strings.EqualFold(href[:5], "data:") {
		return href, true
	}
	ref, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	u := base.ResolveReference(ref)
	if u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return "", false
	}
	u.Fragment = ""
	u.RawFragment = ""
	return u.String(), true
}

// findIcon tries at most three candidates in order and returns the first
// valid icon, or nil.
func (f *Fetcher) findIcon(ctx context.Context, refs []string) *Icon {
	for i, ref := range refs {
		if i == maxIconCandidates || ctx.Err() != nil {
			break
		}
		var data []byte
		if strings.HasPrefix(strings.ToLower(ref), "data:") {
			data = decodeDataIcon(ref, f.opts.MaxIconBytes)
		} else {
			data = f.downloadIcon(ctx, ref)
		}
		if icon := validateIcon(data, f.opts.MaxIconBytes); icon != nil {
			return icon
		}
	}
	return nil
}

// downloadIcon fetches an icon through the guarded client. It returns nil on
// any failure or when the body exceeds the size limit.
func (f *Fetcher) downloadIcon(ctx context.Context, ref string) []byte {
	u, ferr := checkTarget(ref)
	if ferr != nil {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", f.opts.UserAgent)
	req.Header.Set("Accept", acceptImage)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 || resp.ContentLength > f.opts.MaxIconBytes {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, f.opts.MaxIconBytes+1))
	if err != nil || int64(len(data)) > f.opts.MaxIconBytes {
		return nil
	}
	return data
}

// decodeDataIcon decodes a data:image/...;base64,... URI.
func decodeDataIcon(ref string, maxBytes int64) []byte {
	meta, payload, ok := strings.Cut(ref[len("data:"):], ",")
	if !ok {
		return nil
	}
	meta = strings.ToLower(strings.TrimSpace(meta))
	if !strings.HasPrefix(meta, "image/") || !strings.HasSuffix(meta, ";base64") {
		return nil
	}
	// Generous bound (whitespace, percent-encoding); the decoded size is
	// checked exactly by validateIcon.
	if len(payload) > 2*base64.StdEncoding.EncodedLen(int(maxBytes)) {
		return nil
	}
	payload = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, payload)
	if unescaped, err := url.PathUnescape(payload); err == nil {
		payload = unescaped
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(payload, "="))
		if err != nil {
			return nil
		}
	}
	return data
}

var svgEventAttr = regexp.MustCompile(`[\s/"']on[a-z]+\s*=`)

// validateIcon checks size and type from the content (6.8) and returns the
// icon, or nil when it is rejected.
func validateIcon(data []byte, maxBytes int64) *Icon {
	if len(data) == 0 || int64(len(data)) > maxBytes {
		return nil
	}
	ct := ""
	switch sniffed := http.DetectContentType(data); sniffed {
	case "image/png", "image/gif", "image/jpeg", "image/webp", "image/x-icon":
		ct = sniffed
	case "image/vnd.microsoft.icon":
		ct = "image/x-icon"
	default:
		if isSafeSVG(data) {
			ct = "image/svg+xml"
		}
	}
	if ct == "" {
		return nil
	}
	sum := sha256.Sum256(data)
	return &Icon{Data: data, ContentType: ct, SHA256: hex.EncodeToString(sum[:])}
}

// isSafeSVG accepts content that starts with <svg (or <?xml ... <svg) and
// has no <script, javascript: or on...= event attribute (searched in lower
// case, 6.8). On top of that quick check the content must pass the
// structural allow-list check of svgStructureOK, which sees through
// character references, foreign namespaces and entity declarations.
func isSafeSVG(data []byte) bool {
	s := bytes.ToLower(bytes.TrimSpace(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	switch {
	case bytes.HasPrefix(s, []byte("<svg")):
	case bytes.HasPrefix(s, []byte("<?xml")) && bytes.Contains(s, []byte("<svg")):
	default:
		return false
	}
	if bytes.Contains(s, []byte("<script")) || bytes.Contains(s, []byte("javascript:")) {
		return false
	}
	if svgEventAttr.Match(s) {
		return false
	}
	return svgStructureOK(data)
}
