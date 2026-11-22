package api

import (
	"context"
	"net/http"
	"time"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Public endpoints (5.6).

// healthTimeout bounds the database ping of /healthz (TECHNOLOGY.md 4.5).
const healthTimeout = 2 * time.Second

// appName is PublicConfig.appName.
const appName = "Kinex Mark"

// healthResponse is the body of GET /healthz.
type healthResponse struct {
	Status string `json:"status"`
}

// passwordPolicy is PublicConfig.passwordPolicy.
type passwordPolicy struct {
	MinLength   int `json:"minLength"`
	MaxLength   int `json:"maxLength"`
	MinDistinct int `json:"minDistinct"`
}

// publicPrivacy is PublicConfig.privacy.
type publicPrivacy struct {
	ControllerName       string     `json:"controllerName"`
	ControllerAddress    string     `json:"controllerAddress"`
	ControllerEmail      string     `json:"controllerEmail"`
	DpoContact           string     `json:"dpoContact"`
	SupervisoryAuthority string     `json:"supervisoryAuthority"`
	NoticeVersion        int        `json:"noticeVersion"`
	UpdatedAt            *time.Time `json:"updatedAt"`
	AuditRetentionDays   int        `json:"auditRetentionDays"`
	SessionIdleMinutes   int        `json:"sessionIdleMinutes"`
	SessionAbsoluteHours int        `json:"sessionAbsoluteHours"`
}

// publicConfigResponse is the PublicConfig shape of 5.5.
type publicConfigResponse struct {
	AppName        string           `json:"appName"`
	Version        string           `json:"version"`
	Locales        []model.Locale   `json:"locales"`
	DefaultLocale  model.Locale     `json:"defaultLocale"`
	TagColors      []model.TagColor `json:"tagColors"`
	PasswordPolicy passwordPolicy   `json:"passwordPolicy"`
	Privacy        publicPrivacy    `json:"privacy"`
}

// healthz answers 200 {"status":"ok"} when the database answers a ping
// within healthTimeout, otherwise 500 internal_error. HEAD is served too.
func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
	defer cancel()
	if err := s.deps.Store.Ping(ctx); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// publicConfig answers the PublicConfig of 5.5: application facts, the
// password policy and the privacy notice data of the system settings.
func (s *server) publicConfig(w http.ResponseWriter, r *http.Request) {
	set, err := s.deps.Store.Settings(r.Context())
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	version := s.deps.Version
	if version == "" {
		version = "dev"
	}
	writeJSON(w, http.StatusOK, publicConfigResponse{
		AppName:       appName,
		Version:       version,
		Locales:       model.Locales,
		DefaultLocale: model.DefaultLocale,
		TagColors:     model.TagColors,
		PasswordPolicy: passwordPolicy{
			MinLength:   auth.MinPasswordRunes,
			MaxLength:   auth.MaxPasswordRunes,
			MinDistinct: auth.MinDistinctRunes,
		},
		Privacy: publicPrivacy{
			ControllerName:       set.PrivacyControllerName,
			ControllerAddress:    set.PrivacyControllerAddress,
			ControllerEmail:      set.PrivacyControllerEmail,
			DpoContact:           set.PrivacyDpoContact,
			SupervisoryAuthority: set.PrivacySupervisoryAuthority,
			NoticeVersion:        set.PrivacyNoticeVersion,
			UpdatedAt:            set.PrivacyNoticeUpdatedAt,
			AuditRetentionDays:   set.AuditRetentionDays,
			SessionIdleMinutes:   set.SessionIdleMinutes,
			SessionAbsoluteHours: set.SessionAbsoluteHours,
		},
	})
}
