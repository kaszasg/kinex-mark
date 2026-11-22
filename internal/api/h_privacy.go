package api

import (
	"net/http"
	"strings"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Privacy status, notice acknowledgement, GDPR export and account deletion
// (5.8, 5.8.1, 6.11, 6.14).

// exportDateLayout is the date part of the export file name.
const exportDateLayout = "20060102"

// privacyStatusResponse is PrivacyStatus (5.5).
type privacyStatusResponse struct {
	CurrentVersion     int             `json:"currentVersion"`
	ConsentRequired    bool            `json:"consentRequired"`
	Consents           []model.Consent `json:"consents"`
	OwnSharedLinkCount int             `json:"ownSharedLinkCount"`
}

// privacyStatus is GET /api/me/privacy.
func (s *server) privacyStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := userFrom(r)
	set, err := s.deps.Store.Settings(ctx)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	all, err := s.deps.Store.Consents(ctx, u.ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	consents := []model.Consent{}
	required := true
	for _, c := range all {
		if c.Kind != model.ConsentPrivacyNotice {
			continue
		}
		consents = append(consents, c)
		if c.Version == set.PrivacyNoticeVersion {
			required = false
		}
	}
	shared, err := s.deps.Store.OwnSharedLinkCount(ctx, u.ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, privacyStatusResponse{
		CurrentVersion:     set.PrivacyNoticeVersion,
		ConsentRequired:    required,
		Consents:           consents,
		OwnSharedLinkCount: shared,
	})
}

// consentRequest is the body of POST /api/me/privacy/consent.
type consentRequest struct {
	Version *int `json:"version"`
}

// acceptPrivacy is POST /api/me/privacy/consent: the acknowledgement of the
// current notice version; any other version is 409 consent_version_mismatch.
func (s *server) acceptPrivacy(w http.ResponseWriter, r *http.Request) {
	var req consentRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.Version == nil {
		writeValidation(w, map[string]string{"version": fieldRequired})
		return
	}
	ctx := r.Context()
	u := userFrom(r)
	set, err := s.deps.Store.Settings(ctx)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if *req.Version != set.PrivacyNoticeVersion {
		writeError(w, newError(codeConsentVersionMismatch))
		return
	}
	required, err := s.consentRequired(ctx, u.ID, set)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if required {
		if err := s.deps.Store.RecordConsent(ctx, u.ID, model.ConsentPrivacyNotice, set.PrivacyNoticeVersion); err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		s.auditUser(r, model.EventPrivacyConsentAccepted, u, nil,
			map[string]any{model.DetailVersion: set.PrivacyNoticeVersion})
	}
	writeNoContent(w)
}

// exportMyData is GET /api/me/export: the UserExport of 5.8.1 as a JSON
// attachment.
func (s *server) exportMyData(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	exp, err := s.deps.Store.ExportUser(r.Context(), u.ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	exp.Consents = nonNil(exp.Consents)
	exp.Sessions = nonNil(exp.Sessions)
	exp.Tags = nonNil(exp.Tags)
	exp.Links = nonNil(exp.Links)
	exp.SharedWithMe = nonNil(exp.SharedWithMe)
	exp.AuditEvents = nonNil(exp.AuditEvents)
	s.auditUser(r, model.EventGDPRExport, u, nil, nil)
	// Usernames are [A-Za-z0-9._-] only (5.12), so the name needs no quoting.
	name := "kinex-mark-export-" + u.Username + "-" + s.now().UTC().Format(exportDateLayout) + ".json"
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	writeJSON(w, http.StatusOK, exp)
}

// deleteAccountRequest is the body of POST /api/me/delete.
type deleteAccountRequest struct {
	Password string `json:"password"`
	Confirm  string `json:"confirm"`
	Code     string `json:"code"`
}

// deleteAccount is POST /api/me/delete (5.8, 6.11): password, the username
// as confirmation and, with 2FA on, a TOTP or recovery code; the last active
// admin cannot delete itself.
func (s *server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	var req deleteAccountRequest
	if !readJSON(w, r, &req) {
		return
	}
	u := userFrom(r)
	f := fieldErrors{}
	f.add("password", checkPasswordInput(req.Password))
	if strings.TrimSpace(req.Confirm) == "" {
		f.add("confirm", fieldRequired)
	}
	if u.TOTPEnabled {
		f.add("code", checkCodeInput(req.Code))
	}
	if f.writeIfAny(w) {
		return
	}
	if strings.TrimSpace(req.Confirm) != u.Username {
		writeError(w, newError(codeConfirmationMismatch))
		return
	}
	ctx := r.Context()
	check := s.passwordStep(req.Password)
	if u.TOTPEnabled {
		check = s.secondFactorStep(r, req.Password, req.Code, true)
	}
	if err := s.stepUp(ctx, u, check); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	entry := model.AuditEntry{
		Event:   model.EventUserSelfDeleted,
		Success: true,
		Actor:   auditActor(u),
		Target:  userTarget(u),
		IP:      truncateBytes(s.clientIP(r), model.SessionIPMaxLen),
	}
	if err := s.deps.Store.DeleteUser(ctx, u.ID, entry); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.clearSessionCookie(w, r)
	writeNoContent(w)
}
