package fetch

import (
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// MaxTextRunes is the length CleanText truncates to (title and description
// limits, 5.9.2).
const MaxTextRunes = 500

// linkTag is one <link> element of the head.
type linkTag struct {
	rel   []string // lower-case rel tokens
	href  string
	sizes string
	typ   string
}

// headMeta is what parseHead extracts.
type headMeta struct {
	title    string // text of the first <title>
	ogTitle  string // first <meta property="og:title">
	baseHref string // first <base href>
	links    []linkTag
	complete bool // stopped at </head> or <body>, not at the end of input
}

// parseHead tokenizes the document until </head> or the first <body>
// (6.8). It never builds a tree (no html.Parse, see the package comment).
// The only error is a read error other than io.EOF.
func parseHead(r io.Reader) (headMeta, error) {
	var m headMeta
	z := html.NewTokenizer(r)
	inTitle, titleDone := false, false
	var title strings.Builder
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if err := z.Err(); !errors.Is(err, io.EOF) {
				return m, err
			}
			if inTitle {
				m.title = title.String()
			}
			return m, nil
		case html.TextToken:
			if inTitle {
				title.Write(z.Text())
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			switch string(name) {
			case "title":
				if !titleDone && tt == html.StartTagToken {
					inTitle = true
				}
			case "meta":
				attrs := readAttrs(z, hasAttr)
				prop := strings.ToLower(strings.TrimSpace(attrs["property"]))
				if prop == "" {
					prop = strings.ToLower(strings.TrimSpace(attrs["name"]))
				}
				if prop == "og:title" && m.ogTitle == "" {
					m.ogTitle = attrs["content"]
				}
			case "base":
				attrs := readAttrs(z, hasAttr)
				if href, ok := attrs["href"]; ok && m.baseHref == "" {
					m.baseHref = strings.TrimSpace(href)
				}
			case "link":
				attrs := readAttrs(z, hasAttr)
				m.links = append(m.links, linkTag{
					rel:   strings.Fields(strings.ToLower(attrs["rel"])),
					href:  strings.TrimSpace(attrs["href"]),
					sizes: strings.ToLower(strings.TrimSpace(attrs["sizes"])),
					typ:   strings.ToLower(strings.TrimSpace(attrs["type"])),
				})
			case "body":
				m.complete = true
				if inTitle {
					m.title = title.String()
				}
				return m, nil
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "title":
				if inTitle {
					inTitle, titleDone = false, true
					m.title = title.String()
				}
			case "head":
				m.complete = true
				return m, nil
			}
		}
	}
}

// readAttrs collects the attributes of the current tag (first one wins,
// keys lower-cased by the tokenizer, values unescaped).
func readAttrs(z *html.Tokenizer, more bool) map[string]string {
	attrs := map[string]string{}
	for more {
		var k, v []byte
		k, v, more = z.TagAttr()
		key := string(k)
		if _, seen := attrs[key]; !seen {
			attrs[key] = string(v)
		}
	}
	return attrs
}

// resolveBase resolves a <base href> against the page URL. It returns nil
// when there is no usable http(s) base.
func resolveBase(pageURL *url.URL, href string) *url.URL {
	if href == "" {
		return nil
	}
	ref, err := url.Parse(href)
	if err != nil {
		return nil
	}
	b := pageURL.ResolveReference(ref)
	if b.Scheme != "http" && b.Scheme != "https" || b.Host == "" {
		return nil
	}
	return b
}

// isBidiControl reports the bidi control characters of 6.8 (Unicode
// Bidi_Control): the marks ALM, LRM and RLM, the embeddings, overrides and
// isolates.
func isBidiControl(r rune) bool {
	return r == 0x061C || r == 0x200E || r == 0x200F || (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069)
}

// isControl reports C0, DEL and C1 control characters.
func isControl(r rune) bool {
	return r < 0x20 || r == 0x7F || (r >= 0x80 && r <= 0x9F)
}

// CleanText normalises untrusted text for display (6.8): whitespace runs
// (including tab, CR and the other whitespace controls) collapse to one
// space; the remaining C0 and C1 control characters, U+007F, the bidi
// control characters (isBidiControl) and invalid UTF-8 are removed; the result
// is trimmed and cut to at most MaxTextRunes runes. With keepNewlines, line
// breaks (\n, \r\n) survive: spaces around them are dropped and at most one
// empty line is kept between paragraphs.
func CleanText(s string, keepNewlines bool) string {
	if keepNewlines {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}
	var b strings.Builder
	b.Grow(len(s))
	runes := 0
	pendingSpace, pendingNL := false, 0
	for i, r := range s {
		switch {
		case r == utf8.RuneError && isInvalidAt(s, i):
			continue
		case keepNewlines && r == '\n':
			pendingNL++
			continue
		case unicode.IsSpace(r):
			pendingSpace = true
			continue
		case isControl(r) || isBidiControl(r):
			continue
		}
		sep := ""
		if runes > 0 {
			switch {
			case pendingNL >= 2:
				sep = "\n\n"
			case pendingNL == 1:
				sep = "\n"
			case pendingSpace:
				sep = " "
			}
		}
		if runes+len(sep)+1 > MaxTextRunes {
			break
		}
		b.WriteString(sep)
		b.WriteRune(r)
		runes += len(sep) + 1
		pendingSpace, pendingNL = false, 0
	}
	return b.String()
}

func isInvalidAt(s string, i int) bool {
	_, size := utf8.DecodeRuneInString(s[i:])
	return size <= 1
}
