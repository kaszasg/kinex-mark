package api

import (
	"net/http"
	"slices"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Admin system settings (5.12).

// adminSettingsResponse is AdminSettingsResponse (5.5).
type adminSettingsResponse struct {
	Settings    model.SystemSettings `json:"settings"`
	Defaults    model.SystemSettings `json:"defaults"`
	LockedByEnv []string             `json:"lockedByEnv"`
}

// effectiveSettings overlays the values fixed by the environment (6.8:
// APP_ALLOW_PRIVATE_FETCH decides over the stored setting).
func (s *server) effectiveSettings(set model.SystemSettings) model.SystemSettings {
	if s.cfg.AllowPrivateFetch != nil {
		set.AllowPrivateFetch = *s.cfg.AllowPrivateFetch
	}
	return set
}

// writeAdminSettings answers AdminSettingsResponse for set.
func (s *server) writeAdminSettings(w http.ResponseWriter, set model.SystemSettings) {
	writeJSON(w, http.StatusOK, adminSettingsResponse{
		Settings:    s.effectiveSettings(set),
		Defaults:    model.DefaultSystemSettings(),
		LockedByEnv: nonNil(s.cfg.EnvLockedSettings()),
	})
}

// adminGetSettings is GET /api/admin/settings.
func (s *server) adminGetSettings(w http.ResponseWriter, r *http.Request) {
	set, err := s.deps.Store.Settings(r.Context())
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.writeAdminSettings(w, set)
}

// validateSettings checks a settings patch (5.12) and stores the trimmed
// text values back into it.
func validateSettings(p *model.SettingsPatch) fieldErrors {
	f := fieldErrors{}
	ranges := []struct {
		key   string
		value *int
		r     model.IntRange
	}{
		{model.SettingDefaultArchiveDays, p.DefaultArchiveDays, model.DefaultArchiveDaysRange},
		{model.SettingAuditRetentionDays, p.AuditRetentionDays, model.AuditRetentionDaysRange},
		{model.SettingSessionIdleMinutes, p.SessionIdleMinutes, model.SessionIdleMinutesRange},
		{model.SettingSessionAbsoluteHours, p.SessionAbsoluteHours, model.SessionAbsoluteHoursRange},
	}
	for _, c := range ranges {
		if c.value != nil {
			f.intRange(c.key, *c.value, c.r)
		}
	}
	texts := []struct {
		key       string
		value     *string
		max       int
		multiline bool
	}{
		{model.SettingPrivacyControllerName, p.PrivacyControllerName, model.PrivacyControllerNameMaxLen, false},
		{model.SettingPrivacyControllerAddress, p.PrivacyControllerAddress, model.PrivacyControllerAddressMaxLen, true},
		{model.SettingPrivacyControllerEmail, p.PrivacyControllerEmail, model.PrivacyControllerEmailMaxLen, false},
		{model.SettingPrivacyDpoContact, p.PrivacyDpoContact, model.PrivacyDpoContactMaxLen, false},
		{model.SettingPrivacySupervisoryAuthority, p.PrivacySupervisoryAuthority, model.PrivacySupervisoryAuthMaxLen, false},
	}
	for _, c := range texts {
		if c.value != nil {
			*c.value = f.text(c.key, *c.value, 0, c.max, c.multiline)
		}
	}
	if e := p.PrivacyControllerEmail; e != nil && *e != "" && !isEmailShaped(*e) {
		f.add(model.SettingPrivacyControllerEmail, fieldInvalidFormat)
	}
	return f
}

// adminUpdateSettings is PATCH /api/admin/settings: a partial update of the
// editable settings; keys fixed by the environment are 409 setting_locked.
func (s *server) adminUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var p model.SettingsPatch
	if !readJSON(w, r, &p) {
		return
	}
	keys := p.Keys()
	locked := []string{}
	for _, k := range s.cfg.EnvLockedSettings() {
		if slices.Contains(keys, k) {
			locked = append(locked, k)
		}
	}
	if len(locked) > 0 {
		writeError(w, settingLocked(locked))
		return
	}
	if validateSettings(&p).writeIfAny(w) {
		return
	}
	admin := userFrom(r)
	set, err := s.deps.Store.UpdateSettings(r.Context(), p, admin.ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if len(keys) > 0 {
		s.auditUser(r, model.EventSettingsUpdated, admin,
			&model.AuditTarget{Type: model.TargetSettings}, map[string]any{model.DetailChanged: keys})
	}
	s.writeAdminSettings(w, set)
}

// adminBumpPrivacyNotice is POST /api/admin/settings/privacy-notice/bump:
// a new notice version, which every user has to acknowledge (consent gate).
func (s *server) adminBumpPrivacyNotice(w http.ResponseWriter, r *http.Request) {
	admin := userFrom(r)
	set, err := s.deps.Store.BumpPrivacyNotice(r.Context(), admin.ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.EventPrivacyNoticeBumped, admin,
		&model.AuditTarget{Type: model.TargetSettings}, map[string]any{model.DetailVersion: set.PrivacyNoticeVersion})
	s.writeAdminSettings(w, set)
}
