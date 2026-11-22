package api

import (
	"context"
	"net/http"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// linkTagsRequest is the body of PUT /api/links/{id}/tags (5.9.5).
type linkTagsRequest struct {
	TagIDs *[]int64 `json:"tagIds"`
}

// setLinkTags answers PUT /api/links/{id}/tags: replaces the viewer's own
// tags on a visible link (0-20 own tag ids).
func (s *server) setLinkTags(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req linkTagsRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	var ids []int64
	if req.TagIDs == nil {
		f.add(fieldNameTagIDs, fieldRequired)
	} else {
		ids = *req.TagIDs
		f.add(fieldNameTagIDs, checkIDList(ids, model.MaxTagsPerLink))
	}
	if f.writeIfAny(w) {
		return
	}
	if err := s.deps.Store.SetLinkTags(r.Context(), viewerFrom(r), id, nonNil(ids)); err != nil {
		writeFailure(w, r, s.log, tagConflict(err))
		return
	}
	s.writeLinkItem(w, r, id)
}

// pinLink answers POST /api/links/{id}/pin; pinning also restores an
// archived link (4.9).
func (s *server) pinLink(w http.ResponseWriter, r *http.Request) {
	s.changeLinkState(w, r, func(ctx context.Context, v model.Viewer, id int64) error {
		return s.deps.Store.SetPinned(ctx, v, id, true)
	})
}

// unpinLink answers DELETE /api/links/{id}/pin.
func (s *server) unpinLink(w http.ResponseWriter, r *http.Request) {
	s.changeLinkState(w, r, func(ctx context.Context, v model.Viewer, id int64) error {
		return s.deps.Store.SetPinned(ctx, v, id, false)
	})
}

// archiveLink answers POST /api/links/{id}/archive (manual archiving).
func (s *server) archiveLink(w http.ResponseWriter, r *http.Request) {
	s.changeLinkState(w, r, func(ctx context.Context, v model.Viewer, id int64) error {
		return s.deps.Store.SetArchived(ctx, v, id, true)
	})
}

// restoreLink answers DELETE /api/links/{id}/archive.
func (s *server) restoreLink(w http.ResponseWriter, r *http.Request) {
	s.changeLinkState(w, r, func(ctx context.Context, v model.Viewer, id int64) error {
		return s.deps.Store.SetArchived(ctx, v, id, false)
	})
}

// changeLinkState runs an idempotent personal state change on a visible
// link and answers with the viewer's LinkItem.
func (s *server) changeLinkState(w http.ResponseWriter, r *http.Request,
	change func(ctx context.Context, v model.Viewer, id int64) error) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := change(r.Context(), viewerFrom(r), id); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.writeLinkItem(w, r, id)
}

// visitLink answers POST /api/links/{id}/visit: every call records exactly
// one use (open or copy, 5.9.5) and answers with the new LinkState.
func (s *server) visitLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	st, err := s.deps.Store.RecordVisit(r.Context(), viewerFrom(r), id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	days, err := s.archiveDays(r)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, toLinkState(st, days))
}
