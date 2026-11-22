package model

import "time"

// Audit event codes (6.10).
const (
	EventBootstrapAdminCreated  = "bootstrap.admin_created"
	EventLoginSuccess           = "auth.login.success"
	EventLoginFailure           = "auth.login.failure"
	EventLoginLocked            = "auth.login.locked"
	EventLogout                 = "auth.logout"
	EventRecoveryCodeUsed       = "auth.recovery_code.used"
	EventPasswordChanged        = "auth.password.changed"
	Event2FAEnabled             = "auth.2fa.enabled"
	Event2FADisabled            = "auth.2fa.disabled"
	Event2FARecoveryRegenerated = "auth.2fa.recovery_regenerated"
	EventSessionRevoked         = "session.revoked"
	EventSessionRevokedOthers   = "session.revoked_others"
	EventUserCreated            = "user.created"
	EventUserUpdated            = "user.updated"
	EventUserDeactivated        = "user.deactivated"
	EventUserActivated          = "user.activated"
	EventUserDeleted            = "user.deleted"
	EventUserPasswordReset      = "user.password_reset"
	EventUser2FAReset           = "user.2fa_reset"
	EventUserUnlocked           = "user.unlocked"
	EventUserSessionsRevoked    = "user.sessions_revoked"
	EventUserSelfDeleted        = "user.self_deleted"
	EventSettingsUpdated        = "settings.updated"
	EventPrivacyNoticeBumped    = "privacy.notice_bumped"
	EventPrivacyConsentAccepted = "privacy.consent_accepted"
	EventGDPRExport             = "gdpr.export"
	EventLinkShared             = "link.shared"
	EventLinkUnshared           = "link.unshared"
)

// AuditEvents lists every audit event code (28 codes, 6.10). The contract
// golden (9.6) is generated from it.
var AuditEvents = []string{
	EventBootstrapAdminCreated,
	EventLoginSuccess,
	EventLoginFailure,
	EventLoginLocked,
	EventLogout,
	EventRecoveryCodeUsed,
	EventPasswordChanged,
	Event2FAEnabled,
	Event2FADisabled,
	Event2FARecoveryRegenerated,
	EventSessionRevoked,
	EventSessionRevokedOthers,
	EventUserCreated,
	EventUserUpdated,
	EventUserDeactivated,
	EventUserActivated,
	EventUserDeleted,
	EventUserPasswordReset,
	EventUser2FAReset,
	EventUserUnlocked,
	EventUserSessionsRevoked,
	EventUserSelfDeleted,
	EventSettingsUpdated,
	EventPrivacyNoticeBumped,
	EventPrivacyConsentAccepted,
	EventGDPRExport,
	EventLinkShared,
	EventLinkUnshared,
}

// Audit detail keys: the closed key set of AuditEntry.Details (6.10).
const (
	DetailReason            = "reason"
	DetailMethod            = "method"
	DetailScope             = "scope"
	DetailChanged           = "changed"
	DetailRole              = "role"
	DetailActive            = "active"
	DetailRemaining         = "remaining"
	DetailSessionID         = "sessionId"
	DetailCount             = "count"
	DetailVersion           = "version"
	DetailTargetType        = "targetType"
	DetailTargetRole        = "targetRole"
	DetailTargetUserID      = "targetUserId"
	DetailGeneratedPassword = "generatedPassword"
	DetailVia               = "via"
)

// AuditDetailKeys lists the closed detail key set in the order of 6.10.
var AuditDetailKeys = []string{
	DetailReason, DetailMethod, DetailScope, DetailChanged, DetailRole, DetailActive,
	DetailRemaining, DetailSessionID, DetailCount, DetailVersion, DetailTargetType,
	DetailTargetRole, DetailTargetUserID, DetailGeneratedPassword, DetailVia,
}

// Enumerated audit detail values.
const (
	ReasonCredentials  = "credentials"
	ReasonUnknownUser  = "unknown_user"
	ReasonDisabled     = "disabled"
	ReasonTOTP         = "totp"
	MethodPassword     = "password"
	MethodTOTP         = "totp"
	MethodRecoveryCode = "recovery_code"
	ScopeUser          = "user"
	ScopeIP            = "ip"
	ViaCLI             = "cli"
)

// AuditDetailValues lists the enumerated values of the detail keys that the
// interface translates (8.5.8). The role and boolean values are covered by
// the roles list and common words.
var AuditDetailValues = map[string][]string{
	DetailReason:     {ReasonCredentials, ReasonUnknownUser, ReasonDisabled, ReasonTOTP},
	DetailMethod:     {MethodPassword, MethodTOTP, MethodRecoveryCode},
	DetailScope:      {ScopeUser, ScopeIP},
	DetailVia:        {ViaCLI},
	DetailTargetType: {string(ShareAll), string(ShareRole), string(ShareUser)},
}

// AuditTargetType is the kind of object an audit entry refers to.
type AuditTargetType string

// Audit target types (audit_log.target_type).
const (
	TargetUser     AuditTargetType = "user"
	TargetLink     AuditTargetType = "link"
	TargetSession  AuditTargetType = "session"
	TargetSettings AuditTargetType = "settings"
	TargetTag      AuditTargetType = "tag"
)

// Audit column limits (4.5).
const (
	AuditEventMaxLen = 64
	AuditNameMaxLen  = 64
)

// AuditActor is the actor of an audit entry. Username is a snapshot
// (audit_log.actor_name), pseudonymised when the user is deleted (6.11).
type AuditActor struct {
	ID       *int64 `json:"id"`
	Username string `json:"username"`
}

// AuditTarget is the object of an audit entry; Name is a snapshot.
type AuditTarget struct {
	Type AuditTargetType `json:"type"`
	ID   *int64          `json:"id"`
	Name string          `json:"name"`
}

// AuditEntry is an audit log row (5.5 AuditEntry). Details never contain
// names, URLs or secrets (6.10); a nil map is stored and returned as {}.
type AuditEntry struct {
	ID      int64          `json:"id"`
	At      time.Time      `json:"at"`
	Event   string         `json:"event"`
	Success bool           `json:"success"`
	Actor   AuditActor     `json:"actor"`
	Target  *AuditTarget   `json:"target"`
	IP      string         `json:"ip"`
	Details map[string]any `json:"details"`
}

// AuditFilter is the admin audit list filter (5.12).
type AuditFilter struct {
	Event   string // exact code, or a prefix ending in "*" ("auth.*"); "" for any
	ActorID *int64
	Success *bool
	From    *time.Time // inclusive
	To      *time.Time // inclusive
}
