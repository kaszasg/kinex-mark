package api

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/kaszasg/kinex-mark/internal/fetch"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Link list query (5.9.1, 5.4).
const (
	linkListDefaultLimit = 200
	linkListMaxLimit     = 500
	linkListMaxTags      = 20
)

// Request field names of the link endpoints (5.5 NewLink, LinkPatch).
const (
	fieldNameURL         = "url"
	fieldNameTitle       = "title"
	fieldNameDescription = "description"
	fieldNameTagIDs      = "tagIds"
	fieldNameText        = "text"
)

// newLinkRequest is the body of POST /api/links (5.5 NewLink).
type newLinkRequest struct {
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	TagIDs      []int64 `json:"tagIds"`
	Pinned      bool    `json:"pinned"`
}

// linkPatchRequest is the body of PATCH /api/links/{id} (5.5 LinkPatch).
type linkPatchRequest struct {
	URL         *string `json:"url"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

// quickCaptureRequest is the body of POST /api/links/quick (5.9.3).
type quickCaptureRequest struct {
	Text  string `json:"text"`
	Title string `json:"title"`
}

// archiveDays reads the viewer's effective archive period once per request;
// toLinkItem derives archiveDueAt from it (5.5).
func (s *server) archiveDays(r *http.Request) (int, error) {
	p, err := s.deps.Store.Preferences(r.Context(), userFrom(r).ID)
	if err != nil {
		return 0, err
	}
	return p.EffectiveArchiveAfterDays, nil
}

// writeLinkDetail answers with the detail view of a link.
func (s *server) writeLinkDetail(w http.ResponseWriter, r *http.Request, status int, d model.LinkDetail) {
	days, err := s.archiveDays(r)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, status, toLinkDetail(d, days))
}

// writeLinkItem loads the viewer's current view of a link and answers with
// it (the personal state endpoints, 5.9.5).
func (s *server) writeLinkItem(w http.ResponseWriter, r *http.Request, id int64) {
	item, err := s.deps.Store.LinkItem(r.Context(), viewerFrom(r), id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	days, err := s.archiveDays(r)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, toLinkItem(item, days))
}

// enqueue queues a metadata fetch. A full queue is not an error: the link is
// pending in the database and the sweeper picks it up (7.3).
func (s *server) enqueue(id int64) {
	if s.deps.Jobs != nil {
		s.deps.Jobs.Enqueue(id)
	}
}

// normalizeURLField runs fetch.NormalizeURL and records its field code.
func normalizeURLField(f fieldErrors, field, raw string) (string, string) {
	u, host, err := fetch.NormalizeURL(raw)
	if err != nil {
		// NormalizeURL always returns a *fetch.URLError; invalid_format is
		// only the fallback for an unexpected error type.
		code := fieldInvalidFormat
		var ue *fetch.URLError
		if errors.As(err, &ue) {
			code = ue.Code
		}
		f.add(field, code)
		return "", ""
	}
	return u, host
}

// tagConflict maps model.ErrConflict of CreateLink, UpdateLink and
// SetLinkTags: a tag id that is not the viewer's own (5.9.2, 5.9.5).
func tagConflict(err error) error {
	if errors.Is(err, model.ErrConflict) {
		return validationError(map[string]string{fieldNameTagIDs: fieldUnknown})
	}
	return err
}

// listLinks answers GET /api/links (5.9.1).
func (s *server) listLinks(w http.ResponseWriter, r *http.Request) {
	q := newQuery(r)
	lq := model.LinkQuery{
		Q:        strings.ToLower(q.text("q", model.SearchMaxRunes)),
		TagIDs:   q.ids("tags", linkListMaxTags),
		TagMatch: q.enum("tagMatch", model.TagMatchAny, model.TagMatchModes),
		Scope:    q.enum("scope", model.ScopeAll, model.LinkScopes),
		Archived: q.enum("archived", model.ArchivedExclude, model.ArchivedModes),
		Sort:     q.enum("sort", model.SortUsage, model.SortOrders),
	}
	if untagged := q.boolean("untagged"); untagged != nil {
		lq.Untagged = *untagged
		if q.has("tags") {
			q.invalid("untagged")
		}
	}
	page := q.page(linkListDefaultLimit, linkListMaxLimit)
	lq.Limit, lq.Offset = page.Limit, page.Offset
	if q.writeIfInvalid(w) {
		return
	}
	res, err := s.deps.Store.Links(r.Context(), viewerFrom(r), lq)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	days, err := s.archiveDays(r)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, toLinkPage(res, lq, days))
}

// createLink answers POST /api/links (5.9.2).
func (s *server) createLink(w http.ResponseWriter, r *http.Request) {
	var req newLinkRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	u, host := normalizeURLField(f, fieldNameURL, req.URL)
	title := f.text(fieldNameTitle, req.Title, 0, model.TitleMaxRunes, false)
	desc := f.text(fieldNameDescription, req.Description, 0, model.DescriptionMaxLen, true)
	f.add(fieldNameTagIDs, checkIDList(req.TagIDs, model.MaxTagsPerLink))
	if f.writeIfAny(w) {
		return
	}
	d, err := s.deps.Store.CreateLink(r.Context(), model.NewLink{
		OwnerID:     userFrom(r).ID,
		URL:         u,
		Host:        host,
		Title:       title,
		Description: desc,
		TagIDs:      req.TagIDs,
		Pinned:      req.Pinned,
	})
	if err != nil {
		if errors.Is(err, model.ErrLimitReached) {
			err = limitReached(limitLinks)
		}
		writeFailure(w, r, s.log, tagConflict(err))
		return
	}
	s.enqueue(d.ID)
	s.writeLinkDetail(w, r, http.StatusCreated, d)
}

// quickCapture answers POST /api/links/quick (5.9.3): the URL is extracted
// from dropped or pasted text. An URL the viewer already owns answers 200
// with the existing link and created: false.
func (s *server) quickCapture(w http.ResponseWriter, r *http.Request) {
	var req quickCaptureRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	switch {
	case strings.TrimSpace(req.Text) == "":
		f.add(fieldNameText, fieldRequired)
	case utf8.RuneCountInString(req.Text) > model.QuickTextMaxRunes:
		f.add(fieldNameText, fieldTooLong)
	}
	if f.writeIfAny(w) {
		return
	}
	raw, ok := fetch.ExtractURL(req.Text)
	if !ok {
		writeError(w, newError(codeNoURLFound))
		return
	}
	u, host := normalizeURLField(f, fieldNameURL, raw)
	if f.writeIfAny(w) {
		return
	}
	// The title is the text of a dropped anchor: untrusted page content, so
	// it is cleaned and cut like a fetched title instead of being rejected.
	title := fetch.CleanText(req.Title, false)
	d, err := s.deps.Store.CreateLink(r.Context(), model.NewLink{
		OwnerID: userFrom(r).ID,
		URL:     u,
		Host:    host,
		Title:   title,
	})
	var dup *model.DuplicateError
	switch {
	case errors.As(err, &dup) && dup.Kind == model.DuplicateLink:
		existing, lerr := s.deps.Store.Link(r.Context(), viewerFrom(r), dup.ExistingID)
		if lerr != nil {
			writeFailure(w, r, s.log, lerr)
			return
		}
		s.writeQuickCapture(w, r, http.StatusOK, existing, false)
	case errors.Is(err, model.ErrLimitReached):
		writeError(w, limitReached(limitLinks))
	case err != nil:
		writeFailure(w, r, s.log, err)
	default:
		s.enqueue(d.ID)
		s.writeQuickCapture(w, r, http.StatusCreated, d, true)
	}
}

// writeQuickCapture answers with a QuickCaptureResult.
func (s *server) writeQuickCapture(w http.ResponseWriter, r *http.Request, status int, d model.LinkDetail, created bool) {
	days, err := s.archiveDays(r)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, status, QuickCaptureResult{Link: toLinkDetail(d, days), Created: created})
}

// getLink answers GET /api/links/{id} (5.9.4).
func (s *server) getLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	d, err := s.deps.Store.Link(r.Context(), viewerFrom(r), id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.writeLinkDetail(w, r, http.StatusOK, d)
}

// updateLink answers PATCH /api/links/{id} (5.9.4). A URL change or the
// removal of a manual title marks the link for fetching; it is queued then.
func (s *server) updateLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req linkPatchRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	var p model.LinkPatch
	if req.URL != nil {
		u, host := normalizeURLField(f, fieldNameURL, *req.URL)
		p.URL, p.Host = &u, &host
	}
	if req.Title != nil {
		title := f.text(fieldNameTitle, *req.Title, 0, model.TitleMaxRunes, false)
		p.Title = &title
	}
	if req.Description != nil {
		desc := f.text(fieldNameDescription, *req.Description, 0, model.DescriptionMaxLen, true)
		p.Description = &desc
	}
	if f.writeIfAny(w) {
		return
	}
	d, err := s.deps.Store.UpdateLink(r.Context(), viewerFrom(r), id, p)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if d.FetchStatus == model.FetchPending && (p.URL != nil || (p.Title != nil && *p.Title == "")) {
		s.enqueue(d.ID)
	}
	s.writeLinkDetail(w, r, http.StatusOK, d)
}

// deleteLink answers DELETE /api/links/{id} (5.9.4).
func (s *server) deleteLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := s.deps.Store.DeleteLink(r.Context(), viewerFrom(r), id); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeNoContent(w)
}

// refreshLink answers POST /api/links/{id}/refresh (5.9.7): the link is
// marked for fetching again and queued; 202 with the detail view.
func (s *server) refreshLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	v := viewerFrom(r)
	if err := s.deps.Store.ResetFetch(r.Context(), v, id); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.enqueue(id)
	d, err := s.deps.Store.Link(r.Context(), v, id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.writeLinkDetail(w, r, http.StatusAccepted, d)
}
