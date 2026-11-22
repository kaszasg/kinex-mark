package model

import "time"

// System setting keys: the JSON field names of SystemSettings, which are also
// the system_settings.key values (4.5).
const (
	SettingDefaultArchiveDays          = "defaultArchiveDays"
	SettingAuditRetentionDays          = "auditRetentionDays"
	SettingSessionIdleMinutes          = "sessionIdleMinutes"
	SettingSessionAbsoluteHours        = "sessionAbsoluteHours"
	SettingAllowPrivateFetch           = "allowPrivateFetch"
	SettingPrivacyControllerName       = "privacyControllerName"
	SettingPrivacyControllerAddress    = "privacyControllerAddress"
	SettingPrivacyControllerEmail      = "privacyControllerEmail"
	SettingPrivacyDpoContact           = "privacyDpoContact"
	SettingPrivacySupervisoryAuthority = "privacySupervisoryAuthority"
	SettingPrivacyNoticeVersion        = "privacyNoticeVersion"
	SettingPrivacyNoticeUpdatedAt      = "privacyNoticeUpdatedAt"
)

// IntRange is an inclusive integer range.
type IntRange struct{ Min, Max int }

// Contains reports whether v lies within the range.
func (r IntRange) Contains(v int) bool { return v >= r.Min && v <= r.Max }

// Ranges and limits of the editable system settings (5.5 SystemSettings).
var (
	DefaultArchiveDaysRange   = IntRange{0, 3650}
	AuditRetentionDaysRange   = IntRange{30, 3650}
	SessionIdleMinutesRange   = IntRange{15, 10080}
	SessionAbsoluteHoursRange = IntRange{1, 2160}
)

// Rune limits of the privacy text settings.
const (
	PrivacyControllerNameMaxLen    = 200
	PrivacyControllerAddressMaxLen = 300
	PrivacyControllerEmailMaxLen   = 254
	PrivacyDpoContactMaxLen        = 300
	PrivacySupervisoryAuthMaxLen   = 300
)

// SystemSettings are the admin-editable settings (5.5 SystemSettings).
type SystemSettings struct {
	DefaultArchiveDays          int        `json:"defaultArchiveDays"`
	AuditRetentionDays          int        `json:"auditRetentionDays"`
	SessionIdleMinutes          int        `json:"sessionIdleMinutes"`
	SessionAbsoluteHours        int        `json:"sessionAbsoluteHours"`
	AllowPrivateFetch           bool       `json:"allowPrivateFetch"`
	PrivacyControllerName       string     `json:"privacyControllerName"`
	PrivacyControllerAddress    string     `json:"privacyControllerAddress"`
	PrivacyControllerEmail      string     `json:"privacyControllerEmail"`
	PrivacyDpoContact           string     `json:"privacyDpoContact"`
	PrivacySupervisoryAuthority string     `json:"privacySupervisoryAuthority"`
	PrivacyNoticeVersion        int        `json:"privacyNoticeVersion"`   // read-only, bumped by the bump endpoint
	PrivacyNoticeUpdatedAt      *time.Time `json:"privacyNoticeUpdatedAt"` // read-only
}

// DefaultSystemSettings returns the built-in defaults; the store merges the
// stored non-default values over them (4.5).
func DefaultSystemSettings() SystemSettings {
	return SystemSettings{
		DefaultArchiveDays:   90,
		AuditRetentionDays:   180,
		SessionIdleMinutes:   720,
		SessionAbsoluteHours: 168,
		AllowPrivateFetch:    false,
		PrivacyNoticeVersion: 1,
	}
}

// SettingsPatch is a partial update of the editable settings
// (PATCH /api/admin/settings); nil fields stay unchanged. The read-only
// fields are not part of it, so DisallowUnknownFields rejects them.
type SettingsPatch struct {
	DefaultArchiveDays          *int    `json:"defaultArchiveDays,omitempty"`
	AuditRetentionDays          *int    `json:"auditRetentionDays,omitempty"`
	SessionIdleMinutes          *int    `json:"sessionIdleMinutes,omitempty"`
	SessionAbsoluteHours        *int    `json:"sessionAbsoluteHours,omitempty"`
	AllowPrivateFetch           *bool   `json:"allowPrivateFetch,omitempty"`
	PrivacyControllerName       *string `json:"privacyControllerName,omitempty"`
	PrivacyControllerAddress    *string `json:"privacyControllerAddress,omitempty"`
	PrivacyControllerEmail      *string `json:"privacyControllerEmail,omitempty"`
	PrivacyDpoContact           *string `json:"privacyDpoContact,omitempty"`
	PrivacySupervisoryAuthority *string `json:"privacySupervisoryAuthority,omitempty"`
}

// Keys returns the setting keys present in the patch, in declaration order.
// It feeds the settings.updated audit details and the setting_locked check.
func (p SettingsPatch) Keys() []string {
	keys := []string{}
	add := func(set bool, key string) {
		if set {
			keys = append(keys, key)
		}
	}
	add(p.DefaultArchiveDays != nil, SettingDefaultArchiveDays)
	add(p.AuditRetentionDays != nil, SettingAuditRetentionDays)
	add(p.SessionIdleMinutes != nil, SettingSessionIdleMinutes)
	add(p.SessionAbsoluteHours != nil, SettingSessionAbsoluteHours)
	add(p.AllowPrivateFetch != nil, SettingAllowPrivateFetch)
	add(p.PrivacyControllerName != nil, SettingPrivacyControllerName)
	add(p.PrivacyControllerAddress != nil, SettingPrivacyControllerAddress)
	add(p.PrivacyControllerEmail != nil, SettingPrivacyControllerEmail)
	add(p.PrivacyDpoContact != nil, SettingPrivacyDpoContact)
	add(p.PrivacySupervisoryAuthority != nil, SettingPrivacySupervisoryAuthority)
	return keys
}

// Consent kinds (consent_records.kind).
const ConsentPrivacyNotice = "privacy_notice"

// Consent is an acknowledgement of a privacy notice version (4.5).
type Consent struct {
	Kind       string    `json:"kind"`
	Version    int       `json:"version"`
	AcceptedAt time.Time `json:"acceptedAt"`
}

// PurgeResult counts the rows removed by Store.Purge (7.4).
type PurgeResult struct {
	Sessions    int64
	Challenges  int64
	Throttle    int64
	PendingTOTP int64
	Audit       int64
	Icons       int64
}
