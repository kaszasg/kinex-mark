package api

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/kaszasg/kinex-mark/internal/bookmarks"
	"github.com/kaszasg/kinex-mark/internal/fetch"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Export formats of GET /api/links/export (5.9.11).
const (
	exportFormatJSON = "json"
	exportFormatHTML = "html"
)

// exportFormats lists the accepted values of the format parameter.
var exportFormats = []string{exportFormatJSON, exportFormatHTML}

// Media types of the bookmark files.
const (
	mediaJSON = "application/json"
	mediaHTML = "text/html"
)

// exportLinks answers GET /api/links/export?format=json|html: the viewer's
// own links with the viewer's tag names as a download (5.9.11).
func (s *server) exportLinks(w http.ResponseWriter, r *http.Request) {
	q := newQuery(r)
	format := q.enum("format", "", exportFormats)
	if format == "" {
		q.invalid("format")
	}
	if q.writeIfInvalid(w) {
		return
	}
	links, err := s.deps.Store.ExportLinks(r.Context(), userFrom(r).ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	entries := make([]bookmarks.Entry, 0, len(links))
	for _, l := range links {
		added := l.CreatedAt.UTC()
		entries = append(entries, bookmarks.Entry{
			URL:         l.URL,
			Title:       l.Title,
			Description: l.Description,
			Tags:        nonNil(l.Tags),
			AddedAt:     &added,
		})
	}
	now := s.now().UTC()
	var buf bytes.Buffer
	contentType := mediaJSON + "; charset=utf-8"
	if format == exportFormatHTML {
		contentType = mediaHTML + "; charset=utf-8"
		err = bookmarks.WriteNetscape(&buf, entries)
	} else {
		err = bookmarks.WriteJSON(&buf, now, entries)
	}
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", contentType)
	h.Set("Content-Disposition", `attachment; filename="kinex-mark-links-`+now.Format("20060102")+"."+format+`"`)
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// importLinks answers POST /api/links/import (5.9.11). The raw body is a
// kinex-mark-links/1 JSON file (application/json) or a Netscape bookmark
// file (text/html), at most model.MaxImportBytes and
// model.MaxImportEntries entries. Entries with an invalid URL are skipped
// and counted as invalid; the new links stay pending for the sweeper.
func (s *server) importLinks(w http.ResponseWriter, r *http.Request) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || (media != mediaJSON && media != mediaHTML) {
		writeError(w, newError(codeImportInvalid))
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, model.MaxImportBytes))
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeError(w, newError(codePayloadTooLarge))
			return
		}
		writeFailure(w, r, s.log, err)
		return
	}
	var entries []bookmarks.Entry
	if media == mediaJSON {
		entries, err = bookmarks.ParseJSON(bytes.NewReader(body), model.MaxImportEntries)
	} else {
		entries, err = bookmarks.ParseNetscape(bytes.NewReader(body), model.MaxImportEntries)
	}
	if err != nil {
		switch {
		case errors.Is(err, bookmarks.ErrTooManyEntries):
			err = newError(codeImportTooManyEntries)
		case errors.Is(err, bookmarks.ErrInvalid):
			err = newError(codeImportInvalid)
		}
		writeFailure(w, r, s.log, err)
		return
	}
	items := make([]model.ImportEntry, 0, len(entries))
	dropped := 0
	for _, e := range entries {
		u, host, nerr := fetch.NormalizeURL(e.URL)
		if nerr != nil {
			dropped++
			continue
		}
		items = append(items, model.ImportEntry{
			URL:         u,
			Host:        host,
			Title:       fetch.CleanText(e.Title, false),
			Description: fetch.CleanText(e.Description, true),
			Tags:        e.Tags,
			AddedAt:     e.AddedAt,
		})
	}
	res, _, err := s.deps.Store.ImportLinks(r.Context(), userFrom(r).ID, items)
	if err != nil {
		if errors.Is(err, model.ErrLimitReached) {
			err = limitReached(limitLinks)
		}
		writeFailure(w, r, s.log, err)
		return
	}
	res.Total += dropped
	res.Invalid += dropped
	writeJSON(w, http.StatusOK, res)
}
