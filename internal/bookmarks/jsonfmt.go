package bookmarks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// JSONFormat is the value of the "format" field of the JSON export.
const JSONFormat = "kinex-mark-links/1"

// timeLayout matches model.TimeLayout (RFC 3339 UTC, three fraction digits).
const timeLayout = "2006-01-02T15:04:05.000Z07:00"

type jsonFile struct {
	Format     string      `json:"format"`
	ExportedAt string      `json:"exportedAt"`
	Links      *[]jsonLink `json:"links"`
}

type jsonLink struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	AddedAt     *string  `json:"addedAt"`
}

// minAddedAt is the earliest accepted bookmark time (the Unix epoch, as for
// the Netscape ADD_DATE). An earlier value is treated as missing: a negative
// year in UTC cannot be stored as a model.TimeLayout timestamp and read back.
var minAddedAt = time.Unix(0, 0).UTC()

// ParseJSON reads a kinex-mark-links/1 file:
// {"format":"kinex-mark-links/1","exportedAt":"…","links":[{"url","title",
// "description","tags":[…],"addedAt"}]}. Unknown fields are ignored; an
// addedAt before 1970-01-01T00:00:00Z counts as missing. It returns
// ErrInvalid for malformed JSON, a different format, a missing links array
// or an invalid addedAt, and ErrTooManyEntries above maxEntries
// (maxEntries <= 0: no limit).
func ParseJSON(r io.Reader, maxEntries int) ([]Entry, error) {
	dec := json.NewDecoder(r)
	var f jsonFile
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: trailing data after the JSON document", ErrInvalid)
	}
	if f.Format != JSONFormat || f.Links == nil {
		return nil, ErrInvalid
	}
	if maxEntries > 0 && len(*f.Links) > maxEntries {
		return nil, ErrTooManyEntries
	}
	entries := make([]Entry, 0, len(*f.Links))
	for _, l := range *f.Links {
		e := Entry{URL: l.URL, Title: l.Title, Description: l.Description, Tags: addTags(nil, l.Tags...)}
		if l.AddedAt != nil {
			t, err := time.Parse(time.RFC3339Nano, *l.AddedAt)
			if err != nil {
				return nil, fmt.Errorf("%w: addedAt: %v", ErrInvalid, err)
			}
			t = t.UTC()
			if !t.Before(minAddedAt) {
				e.AddedAt = &t
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// WriteJSON writes entries as a kinex-mark-links/1 file (indented, with a
// trailing newline). Missing tags are written as [], a missing addedAt as
// null.
func WriteJSON(w io.Writer, exportedAt time.Time, entries []Entry) error {
	links := make([]jsonLink, 0, len(entries))
	for _, e := range entries {
		l := jsonLink{URL: e.URL, Title: e.Title, Description: e.Description, Tags: e.Tags}
		if l.Tags == nil {
			l.Tags = []string{}
		}
		if e.AddedAt != nil {
			s := e.AddedAt.UTC().Format(timeLayout)
			l.AddedAt = &s
		}
		links = append(links, l)
	}
	f := jsonFile{Format: JSONFormat, ExportedAt: exportedAt.UTC().Format(timeLayout), Links: &links}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(f); err != nil {
		return fmt.Errorf("write bookmark json: %w", err)
	}
	return nil
}
