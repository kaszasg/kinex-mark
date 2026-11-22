package api

import (
	"errors"
	"net/http"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Request field names of POST /api/links/{id}/shares (5.5 ShareTarget).
const (
	fieldNameTargetType   = "targetType"
	fieldNameTargetRole   = "targetRole"
	fieldNameTargetUserID = "targetUserId"
)

// shareTargetRequest is the body of POST /api/links/{id}/shares.
type shareTargetRequest struct {
	TargetType   string  `json:"targetType"`
	TargetRole   *string `json:"targetRole"`
	TargetUserID *int64  `json:"targetUserId"`
}

// validate checks the field combination of 5.9.6: role needs targetRole and
// forbids targetUserId, user the other way round, all forbids both.
func (req shareTargetRequest) validate() (model.ShareTarget, fieldErrors) {
	f := fieldErrors{}
	t := model.ShareTarget{TargetType: model.ShareTargetType(req.TargetType)}
	switch {
	case req.TargetType == "":
		f.add(fieldNameTargetType, fieldRequired)
	case !t.TargetType.Valid():
		f.add(fieldNameTargetType, fieldInvalid)
	}
	wantRole := t.TargetType == model.ShareRole
	wantUser := t.TargetType == model.ShareUser
	switch {
	case req.TargetRole == nil:
		if wantRole {
			f.add(fieldNameTargetRole, fieldRequired)
		}
	case !wantRole:
		f.add(fieldNameTargetRole, fieldNotAllowed)
	default:
		role := model.Role(*req.TargetRole)
		if !role.Valid() {
			f.add(fieldNameTargetRole, fieldInvalid)
		}
		t.TargetRole = &role
	}
	switch {
	case req.TargetUserID == nil:
		if wantUser {
			f.add(fieldNameTargetUserID, fieldRequired)
		}
	case !wantUser:
		f.add(fieldNameTargetUserID, fieldNotAllowed)
	case *req.TargetUserID < 1:
		f.add(fieldNameTargetUserID, fieldInvalid)
	default:
		t.TargetUserID = req.TargetUserID
	}
	if !f.empty() {
		return model.ShareTarget{}, f
	}
	return t, nil
}

// listShares answers GET /api/links/{id}/shares (owner only; the guard has
// already rejected roles that cannot share).
func (s *server) listShares(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	shares, err := s.deps.Store.Shares(r.Context(), viewerFrom(r), id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeList(w, shares)
}

// createShare answers POST /api/links/{id}/shares (5.9.6) and audits
// link.shared.
func (s *server) createShare(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req shareTargetRequest
	if !readJSON(w, r, &req) {
		return
	}
	target, f := req.validate()
	if f.writeIfAny(w) {
		return
	}
	v := viewerFrom(r)
	share, err := s.deps.Store.CreateShare(r.Context(), v, id, target)
	if err != nil {
		if errors.Is(err, model.ErrConflict) {
			// The shape is valid, so the store rejected the target user:
			// the owner itself, or an unknown or inactive account.
			code := fieldUnknown
			if target.TargetUserID != nil && *target.TargetUserID == v.UserID {
				code = fieldNotAllowed
			}
			err = validationError(map[string]string{fieldNameTargetUserID: code})
		}
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditShare(r, model.EventLinkShared, id, target)
	writeJSON(w, http.StatusCreated, share)
}

// deleteShare answers DELETE /api/links/{id}/shares/{shareId} (5.9.6). The
// share must belong to the link in the path and the link to the viewer
// (4.7 nested ids); otherwise 404 (403 not_owner for a visible foreign link).
func (s *server) deleteShare(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	shareID, ok := pathID(w, r, "shareId")
	if !ok {
		return
	}
	v := viewerFrom(r)
	// The listing checks the ownership and supplies the audit details; the
	// delete statement below enforces the nested id rule on its own.
	shares, err := s.deps.Store.Shares(r.Context(), v, id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	var target *model.ShareTarget
	for _, sh := range shares {
		if sh.ID == shareID {
			target = &model.ShareTarget{TargetType: sh.TargetType, TargetRole: sh.TargetRole}
			if sh.TargetUser != nil {
				uid := sh.TargetUser.ID
				target.TargetUserID = &uid
			}
			break
		}
	}
	if target == nil {
		writeError(w, newError(codeNotFound))
		return
	}
	if err := s.deps.Store.DeleteShare(r.Context(), v, id, shareID); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditShare(r, model.EventLinkUnshared, id, *target)
	writeNoContent(w)
}

// auditShare appends link.shared or link.unshared (6.10). The entry carries
// no URL and no title; the target name stays empty. A failed audit write is
// logged: the share change itself has already been committed.
func (s *server) auditShare(r *http.Request, event string, linkID int64, t model.ShareTarget) {
	u := userFrom(r)
	actorID, targetID := u.ID, linkID
	var role, userID any
	if t.TargetRole != nil {
		role = string(*t.TargetRole)
	}
	if t.TargetUserID != nil {
		userID = *t.TargetUserID
	}
	err := s.deps.Store.AppendAudit(r.Context(), model.AuditEntry{
		Event:   event,
		Success: true,
		Actor:   model.AuditActor{ID: &actorID, Username: u.Username},
		Target:  &model.AuditTarget{Type: model.TargetLink, ID: &targetID},
		IP:      s.clientIP(r),
		Details: map[string]any{
			model.DetailTargetType:   string(t.TargetType),
			model.DetailTargetRole:   role,
			model.DetailTargetUserID: userID,
		},
	})
	if err != nil {
		s.log.Error("audit write failed", "event", event, "err", err)
	}
}
