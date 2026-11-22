// Package bookmarks reads and writes bookmark files: the Netscape HTML
// format of the browsers and the kinex-mark-links/1 JSON format (plan 3.3.7).
//
// The package only parses; validating and cleaning URLs, titles,
// descriptions and tag names is the caller's job (fetch.NormalizeURL,
// fetch.CleanText). HTML is read with html.NewTokenizer only: the tree
// builder (html.Parse*) and html.Render must not be called, as the
// golang.org/x/net/html advisories so far have hit them, not the tokenizer.
package bookmarks

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Entry is one bookmark.
type Entry struct {
	URL, Title, Description string
	Tags                    []string
	AddedAt                 *time.Time
}

// Errors of the parsers: ErrInvalid is an unusable import file
// (import_invalid), ErrTooManyEntries a file above the entry limit
// (import_too_many_entries).
var (
	ErrInvalid        = errors.New("bookmarks: file could not be read")
	ErrTooManyEntries = errors.New("bookmarks: too many entries")
)

const netscapeDoctype = "netscape-bookmark-file-1"

// maxUnixSeconds keeps ADD_DATE inside the year 9999.
const maxUnixSeconds = 253402300799

// ParseNetscape reads a Netscape bookmark file. Every <A HREF> is an entry;
// ADD_DATE is Unix seconds; the comma-separated TAGS attribute and the name
// of the nearest enclosing <H3> folder become tags (the folder hierarchy is
// flattened); the <DD> text after the entry is its description.
//
// It returns ErrInvalid when the input is not a bookmark file (no Netscape
// doctype and no <A HREF> inside a <DL>) and ErrTooManyEntries when it holds
// more than maxEntries entries (maxEntries <= 0: no limit).
func ParseNetscape(r io.Reader, maxEntries int) ([]Entry, error) {
	z := html.NewTokenizer(r)
	var (
		entries   []Entry
		doctype   bool
		folders   []string // folder name per open <DL>
		pending   string   // name of the last <H3>, waiting for its <DL>
		inH3      bool
		h3        strings.Builder
		cur       *Entry // open <A>
		title     strings.Builder
		lastEntry bool // the last list item was an <A>, so a <DD> describes it
		inDD      bool
		desc      strings.Builder
	)
	closeDD := func() {
		if inDD && lastEntry && len(entries) > 0 {
			entries[len(entries)-1].Description = strings.TrimSpace(desc.String())
		}
		inDD = false
		desc.Reset()
	}
	closeA := func() bool {
		if cur == nil {
			return true
		}
		cur.Title = strings.TrimSpace(title.String())
		entries = append(entries, *cur)
		cur, lastEntry = nil, true
		return maxEntries <= 0 || len(entries) <= maxEntries
	}
	nearestFolder := func() string {
		if len(folders) == 0 {
			return ""
		}
		return folders[len(folders)-1]
	}
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if err := z.Err(); !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("read bookmark file: %w", err)
			}
			if !closeA() {
				return nil, ErrTooManyEntries
			}
			closeDD()
			if !doctype && len(entries) == 0 {
				return nil, ErrInvalid
			}
			return entries, nil
		case html.DoctypeToken:
			if strings.EqualFold(strings.TrimSpace(string(z.Text())), netscapeDoctype) {
				doctype = true
			}
		case html.TextToken:
			switch {
			case cur != nil:
				title.Write(z.Text())
			case inH3:
				h3.Write(z.Text())
			case inDD:
				desc.Write(z.Text())
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			switch string(name) {
			case "dl":
				closeDD()
				folders = append(folders, pending)
				pending, lastEntry = "", false
			case "dt":
				closeDD()
			case "h3":
				closeDD()
				inH3, lastEntry = true, false
				h3.Reset()
			case "a":
				closeDD()
				if !closeA() {
					return nil, ErrTooManyEntries
				}
				attrs := readAttrs(z, hasAttr)
				href, ok := attrs["href"]
				if !ok || len(folders) == 0 && !doctype {
					continue
				}
				e := Entry{URL: strings.TrimSpace(href)}
				e.Tags = addTags(e.Tags, strings.Split(attrs["tags"], ",")...)
				e.Tags = addTags(e.Tags, nearestFolder())
				if s, ok := attrs["add_date"]; ok {
					e.AddedAt = parseAddDate(s)
				}
				cur = &e
				title.Reset()
			case "dd":
				closeDD()
				inDD = true
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "a":
				if !closeA() {
					return nil, ErrTooManyEntries
				}
			case "h3":
				if inH3 {
					pending = strings.TrimSpace(h3.String())
				}
				inH3 = false
			case "dl":
				closeDD()
				if !closeA() {
					return nil, ErrTooManyEntries
				}
				if len(folders) > 0 {
					folders = folders[:len(folders)-1]
				}
				pending, lastEntry = "", false
			}
		}
	}
}

func readAttrs(z *html.Tokenizer, more bool) map[string]string {
	attrs := map[string]string{}
	for more {
		var k, v []byte
		k, v, more = z.TagAttr()
		if _, seen := attrs[string(k)]; !seen {
			attrs[string(k)] = string(v)
		}
	}
	return attrs
}

// MaxTagsPerEntry bounds the tag names kept per entry. A link takes at most
// 20 tags (model.MaxTagsPerLink); the margin leaves room for names that the
// caller's cleaning turns into duplicates or empty strings. The bound keeps
// the per-entry work small whatever the file holds (5.9.11).
const MaxTagsPerEntry = 100

// addTags appends the trimmed, non-empty names that are not yet present, up
// to MaxTagsPerEntry names in all; the rest are dropped. It runs in linear
// time.
func addTags(tags []string, names ...string) []string {
	if len(tags) >= MaxTagsPerEntry {
		return tags
	}
	seen := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		seen[t] = struct{}{}
	}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		tags = append(tags, n)
		if len(tags) >= MaxTagsPerEntry {
			break
		}
	}
	return tags
}

func parseAddDate(s string) *time.Time {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n <= 0 || n > maxUnixSeconds {
		return nil
	}
	t := time.Unix(n, 0).UTC()
	return &t
}

// WriteNetscape writes entries in the standard Netscape form. The header and
// the title are language-neutral ("Kinex Mark"); the backend writes no
// localised text (D10). Tag names containing a comma cannot be represented
// in the TAGS attribute and are split when read back.
func WriteNetscape(w io.Writer, entries []Entry) error {
	bw := bufio.NewWriter(w)
	bw.WriteString("<!DOCTYPE NETSCAPE-Bookmark-file-1>\n" +
		"<!-- This is an automatically generated file.\n" +
		"     It will be read and overwritten.\n" +
		"     DO NOT EDIT! -->\n" +
		"<META HTTP-EQUIV=\"Content-Type\" CONTENT=\"text/html; charset=UTF-8\">\n" +
		"<TITLE>Kinex Mark</TITLE>\n" +
		"<H1>Kinex Mark</H1>\n" +
		"<DL><p>\n")
	for _, e := range entries {
		bw.WriteString("    <DT><A HREF=\"")
		bw.WriteString(html.EscapeString(e.URL))
		bw.WriteString("\"")
		if e.AddedAt != nil {
			bw.WriteString(" ADD_DATE=\"")
			bw.WriteString(strconv.FormatInt(e.AddedAt.Unix(), 10))
			bw.WriteString("\"")
		}
		if len(e.Tags) > 0 {
			bw.WriteString(" TAGS=\"")
			bw.WriteString(html.EscapeString(strings.Join(e.Tags, ",")))
			bw.WriteString("\"")
		}
		bw.WriteString(">")
		bw.WriteString(html.EscapeString(e.Title))
		bw.WriteString("</A>\n")
		if e.Description != "" {
			bw.WriteString("    <DD>")
			bw.WriteString(html.EscapeString(e.Description))
			bw.WriteString("\n")
		}
	}
	bw.WriteString("</DL><p>\n")
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("write bookmark file: %w", err)
	}
	return nil
}
