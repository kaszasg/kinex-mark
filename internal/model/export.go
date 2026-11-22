package model

import "time"

// UserExportFormat is the format marker of the personal data export.
const UserExportFormat = "kinex-mark-export/1"

// LinksExportFormat is the format marker of the JSON link export (3.3.7).
const LinksExportFormat = "kinex-mark-links/1"

// Import limits (5.9.11).
const (
	MaxImportBytes   = 2 << 20
	MaxImportEntries = 5000
)

// UserExport is the GDPR self-service export (5.8.1). It never contains
// password hashes, TOTP secrets, recovery codes, session tokens or icon data.
type UserExport struct {
	Format       string             `json:"format"`
	ExportedAt   time.Time          `json:"exportedAt"`
	User         ExportUser         `json:"user"`
	Preferences  Preferences        `json:"preferences"`
	Consents     []Consent          `json:"consents"`
	Sessions     []Session          `json:"sessions"`
	Tags         []ExportTag        `json:"tags"`
	Links        []ExportOwnLink    `json:"links"`
	SharedWithMe []ExportSharedLink `json:"sharedWithMe"`
	AuditEvents  []ExportAuditEvent `json:"auditEvents"`
	Security     ExportSecurity     `json:"security"`
}

// ExportUser is the account part of UserExport.
type ExportUser struct {
	ID                int64      `json:"id"`
	Username          string     `json:"username"`
	DisplayName       string     `json:"displayName"`
	Role              Role       `json:"role"`
	Active            bool       `json:"active"`
	TOTPEnabled       bool       `json:"totpEnabled"`
	CreatedAt         time.Time  `json:"createdAt"`
	LastLoginAt       *time.Time `json:"lastLoginAt"`
	PasswordChangedAt time.Time  `json:"passwordChangedAt"`
}

// ExportTag is a tag in UserExport.
type ExportTag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Color     TagColor  `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
}

// ExportShare is a share of an own link in UserExport.
type ExportShare struct {
	TargetType     ShareTargetType `json:"targetType"`
	TargetRole     *Role           `json:"targetRole"`
	TargetUsername *string         `json:"targetUsername"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// ExportOwnLink is an own link in UserExport.
type ExportOwnLink struct {
	ID          int64         `json:"id"`
	URL         string        `json:"url"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	Tags        []string      `json:"tags"`
	Pinned      bool          `json:"pinned"`
	UseCount    int64         `json:"useCount"`
	LastUsedAt  *time.Time    `json:"lastUsedAt"`
	ArchivedAt  *time.Time    `json:"archivedAt"`
	Shares      []ExportShare `json:"shares"`
}

// ExportSharedLink is the personal state on a link owned by someone else.
// URL is nil when the link is no longer visible to the user (5.8.1).
type ExportSharedLink struct {
	LinkID     int64      `json:"linkId"`
	URL        *string    `json:"url"`
	Tags       []string   `json:"tags"`
	Pinned     bool       `json:"pinned"`
	UseCount   int64      `json:"useCount"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	ArchivedAt *time.Time `json:"archivedAt"`
}

// Roles of the user in an exported audit event.
const (
	ExportAuditActor  = "actor"
	ExportAuditTarget = "target"
)

// ExportAuditEvent is an audit event in which the user is the actor or the target.
type ExportAuditEvent struct {
	At      time.Time `json:"at"`
	Event   string    `json:"event"`
	Success bool      `json:"success"`
	IP      string    `json:"ip"`
	Role    string    `json:"role"` // ExportAuditActor or ExportAuditTarget
}

// ExportSecurity is the current sign-in lock of the account.
type ExportSecurity struct {
	LockedUntil *time.Time `json:"lockedUntil"`
}

// ExportedLink is an own link prepared for the bookmark export
// (GET /api/links/export), with the owner's tag names.
type ExportedLink struct {
	URL         string
	Title       string
	Description string
	Tags        []string
	CreatedAt   time.Time
}

// ImportEntry is a bookmark to import. The caller normalises URL and Host
// with fetch.NormalizeURL, cleans Title and Description with fetch.CleanText,
// drops invalid entries and counts them in ImportResult.Invalid itself.
type ImportEntry struct {
	URL         string
	Host        string
	Title       string
	Description string
	Tags        []string
	AddedAt     *time.Time
}

// ImportResult summarises an import (5.5 ImportResult).
type ImportResult struct {
	Total      int `json:"total"`
	Created    int `json:"created"`
	Duplicates int `json:"duplicates"`
	Invalid    int `json:"invalid"`
}
