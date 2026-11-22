package api

import (
	"errors"
	"net/http"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Request field names of the tag endpoints (5.10).
const (
	fieldNameName  = "name"
	fieldNameColor = "color"
)

// newTagRequest is the body of POST /api/tags.
type newTagRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color"`
}

// tagPatchRequest is the body of PATCH /api/tags/{id}.
type tagPatchRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// tagColorField validates an optional color; nil stays nil.
func tagColorField(f fieldErrors, raw *string) *model.TagColor {
	if raw == nil {
		return nil
	}
	c := model.TagColor(*raw)
	if !c.Valid() {
		f.add(fieldNameColor, fieldInvalid)
		return nil
	}
	return &c
}

// listTags answers GET /api/tags: the viewer's own tags by name with the
// counts of the links currently visible to the viewer.
func (s *server) listTags(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	tags, err := s.deps.Store.Tags(r.Context(), u.ID, u.Role)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeList(w, tags)
}

// createTag answers POST /api/tags (name 1-40 runes, color default neutral).
func (s *server) createTag(w http.ResponseWriter, r *http.Request) {
	var req newTagRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	name := f.text(fieldNameName, req.Name, 1, model.TagNameMaxLen, false)
	color := model.DefaultTagColor
	if c := tagColorField(f, req.Color); c != nil {
		color = *c
	}
	if f.writeIfAny(w) {
		return
	}
	tag, err := s.deps.Store.CreateTag(r.Context(), userFrom(r).ID, name, color)
	if err != nil {
		if errors.Is(err, model.ErrLimitReached) {
			err = limitReached(limitTags)
		}
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

// updateTag answers PATCH /api/tags/{id}: rename or recolor an own tag.
func (s *server) updateTag(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req tagPatchRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	var p model.TagPatch
	if req.Name != nil {
		name := f.text(fieldNameName, *req.Name, 1, model.TagNameMaxLen, false)
		p.Name = &name
	}
	p.Color = tagColorField(f, req.Color)
	if f.writeIfAny(w) {
		return
	}
	tag, err := s.deps.Store.UpdateTag(r.Context(), userFrom(r).ID, id, p)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

// deleteTag answers DELETE /api/tags/{id}; the tag leaves every link.
func (s *server) deleteTag(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := s.deps.Store.DeleteTag(r.Context(), userFrom(r).ID, id); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeNoContent(w)
}
