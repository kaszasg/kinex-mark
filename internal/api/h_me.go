package api

import (
	"net/http"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Profile, preferences and sessions of the signed-in user (5.8).

// sessionResponse is Session (5.5): a stored session plus the current flag.
type sessionResponse struct {
	ID         int64     `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"userAgent"`
	Current    bool      `json:"current"`
}

// revokedResponse is {revoked: number}.
type revokedResponse struct {
	Revoked int `json:"revoked"`
}

// profileRequest is the body of PATCH /api/me/profile.
type profileRequest struct {
	DisplayName *string `json:"displayName"`
}

// updateProfile is PATCH /api/me/profile: the display name (0-100 runes).
func (s *server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var req profileRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	var patch model.UserPatch
	if req.DisplayName != nil {
		name := f.text("displayName", *req.DisplayName, 0, model.DisplayNameMaxLen, false)
		patch.DisplayName = &name
	}
	if f.writeIfAny(w) {
		return
	}
	u, err := s.deps.Store.UpdateUser(r.Context(), userFrom(r).ID, patch)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.writeMe(w, r, u, sessionFrom(r).ID)
}

// getPreferences is GET /api/me/preferences.
func (s *server) getPreferences(w http.ResponseWriter, r *http.Request) {
	prefs, err := s.deps.Store.Preferences(r.Context(), userFrom(r).ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

// validatePreferences checks the enumerated values and the archive period of
// a preferences patch.
func validatePreferences(p model.PreferencesPatch) fieldErrors {
	f := fieldErrors{}
	if p.Locale != nil && !p.Locale.Valid() {
		f.add("locale", fieldInvalid)
	}
	enums := []struct {
		field   string
		value   *string
		allowed []string
	}{
		{"theme", p.Theme, model.Themes},
		{"viewMode", p.ViewMode, model.ViewModes},
		{"sort", p.Sort, model.SortOrders},
		{"tagMatch", p.TagMatch, model.TagMatchModes},
	}
	for _, e := range enums {
		if e.value != nil {
			f.oneOf(e.field, *e.value, e.allowed)
		}
	}
	if p.ArchiveAfterDays.Set && p.ArchiveAfterDays.Value != nil {
		f.intRange("archiveAfterDays", *p.ArchiveAfterDays.Value, model.IntRange{Min: 0, Max: model.ArchiveAfterDaysMax})
	}
	return f
}

// updatePreferences is PATCH /api/me/preferences: a partial update; the
// read-only effective and system archive periods are unknown fields.
func (s *server) updatePreferences(w http.ResponseWriter, r *http.Request) {
	var p model.PreferencesPatch
	if !readJSON(w, r, &p) {
		return
	}
	if validatePreferences(p).writeIfAny(w) {
		return
	}
	prefs, err := s.deps.Store.UpdatePreferences(r.Context(), userFrom(r).ID, p)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

// listSessions is GET /api/me/sessions, newest first.
func (s *server) listSessions(w http.ResponseWriter, r *http.Request) {
	list, err := s.deps.Store.Sessions(r.Context(), userFrom(r).ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	current := sessionFrom(r).ID
	items := make([]sessionResponse, len(list))
	for i, sess := range list {
		items[i] = sessionResponse{
			ID:         sess.ID,
			CreatedAt:  sess.CreatedAt,
			LastSeenAt: sess.LastSeenAt,
			ExpiresAt:  sess.ExpiresAt,
			IP:         sess.IP,
			UserAgent:  sess.UserAgent,
			Current:    sess.ID == current,
		}
	}
	writeList(w, items)
}

// revokeSession is DELETE /api/me/sessions/{id}: another user's or an
// unknown session is 404; revoking the current one also clears the cookie.
func (s *server) revokeSession(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	u := userFrom(r)
	if err := s.deps.Store.DeleteSession(r.Context(), u.ID, id); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if id == sessionFrom(r).ID {
		s.clearSessionCookie(w, r)
	}
	s.auditUser(r, model.EventSessionRevoked, u,
		&model.AuditTarget{Type: model.TargetSession, ID: &id},
		map[string]any{model.DetailSessionID: id})
	writeNoContent(w)
}

// revokeOtherSessions is POST /api/me/sessions/revoke-others.
func (s *server) revokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	n, err := s.deps.Store.DeleteUserSessions(r.Context(), u.ID, sessionFrom(r).ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.EventSessionRevokedOthers, u, nil, map[string]any{model.DetailCount: n})
	writeJSON(w, http.StatusOK, revokedResponse{Revoked: n})
}
