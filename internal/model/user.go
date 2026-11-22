package model

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

// Username rules (5.12): 3-64 characters from [A-Za-z0-9._-], unique
// case-insensitively.
const (
	UsernameMinLen    = 3
	UsernameMaxLen    = 64
	DisplayNameMaxLen = 100
)

// UsernameKey returns the case-insensitive uniqueness key of a username
// (users.username_key). COLLATE NOCASE only folds ASCII, so the key is
// computed in Go.
func UsernameKey(username string) string {
	return strings.ToLower(username)
}

// ValidUsername reports whether s satisfies the username rule.
func ValidUsername(s string) bool {
	if len(s) < UsernameMinLen || len(s) > UsernameMaxLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

// User is a user account without secrets. The api package builds the Me,
// AdminUser and export shapes from it.
type User struct {
	ID                 int64      `json:"id"`
	Username           string     `json:"username"`
	UsernameKey        string     `json:"-"`
	DisplayName        string     `json:"displayName"`
	Role               Role       `json:"role"`
	MustChangePassword bool       `json:"mustChangePassword"`
	Active             bool       `json:"active"`
	TOTPEnabled        bool       `json:"totpEnabled"`
	PasswordChangedAt  time.Time  `json:"passwordChangedAt"`
	LastLoginAt        *time.Time `json:"lastLoginAt"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// Ref returns the public reference of the user.
func (u User) Ref() UserRef {
	return UserRef{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName}
}

// Viewer returns the visibility identity of the user.
func (u User) Viewer() Viewer {
	return Viewer{UserID: u.ID, Role: u.Role}
}

// Credentials is the sign-in view of an account: the user plus the secrets
// needed to verify a password or a TOTP code.
type Credentials struct {
	User         User
	PasswordHash string // auth.Hasher encoded hash
	TOTPSecret   string // Keyring.Seal output (associated data "totp:<id>"); "" when 2FA is off
	TOTPLastStep int64  // last accepted TOTP step (replay protection)
}

// NewUser is the input of Store.CreateUser. The store derives the username
// key and creates the preferences row in the same transaction.
type NewUser struct {
	Username           string
	DisplayName        string
	Role               Role
	PasswordHash       string
	MustChangePassword bool
	Locale             Locale // initial user_preferences.locale; "" means DefaultLocale
}

// UserPatch is the input of Store.UpdateUser; nil fields stay unchanged.
type UserPatch struct {
	DisplayName *string
	Role        *Role
	Active      *bool
}

// UserRef is the public reference of a user (5.5 UserRef).
type UserRef struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

// DirectoryUser is an entry of the sharing user directory (5.5 DirectoryUser).
type DirectoryUser struct {
	UserRef
	Role Role `json:"role"`
}

// User status filter values of UserFilter.Status.
const (
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
	UserStatusAll      = "all"
)

// UserFilter is the admin user list filter (5.12).
type UserFilter struct {
	Query  string // lower-cased search text, "" for none
	Role   Role   // "" for any role
	Status string // UserStatusActive, UserStatusInactive or UserStatusAll ("" = all)
}

// AdminUser is a row of the admin user list (5.5 AdminUser).
type AdminUser struct {
	ID                 int64      `json:"id"`
	Username           string     `json:"username"`
	DisplayName        string     `json:"displayName"`
	Role               Role       `json:"role"`
	Active             bool       `json:"active"`
	MustChangePassword bool       `json:"mustChangePassword"`
	TOTPEnabled        bool       `json:"totpEnabled"`
	LockedUntil        *time.Time `json:"lockedUntil"`
	LastLoginAt        *time.Time `json:"lastLoginAt"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	LinkCount          int        `json:"linkCount"`
	SharedLinkCount    int        `json:"sharedLinkCount"`
	SessionCount       int        `json:"sessionCount"`
}

// Preference values (user_preferences CHECK constraints, 4.2).
const (
	ThemeDark   = "dark"
	ThemeLight  = "light"
	ThemeSystem = "system"

	ViewList  = "list"
	ViewTiles = "tiles"

	SortUsage   = "usage"
	SortCreated = "created"
	SortTitle   = "title"

	TagMatchAny = "any"
	TagMatchAll = "all"

	ArchiveAfterDaysMax = 3650
)

// Allowed values of the enumerated preferences, in display order.
var (
	Themes        = []string{ThemeDark, ThemeLight, ThemeSystem}
	ViewModes     = []string{ViewList, ViewTiles}
	SortOrders    = []string{SortUsage, SortCreated, SortTitle}
	TagMatchModes = []string{TagMatchAny, TagMatchAll}
)

// Preferences are the per-user display settings (5.5 Preferences).
type Preferences struct {
	Locale           Locale `json:"locale"`
	Theme            string `json:"theme"`
	ViewMode         string `json:"viewMode"`
	ShowIcon         bool   `json:"showIcon"`
	ShowURL          bool   `json:"showUrl"`
	ShowTitle        bool   `json:"showTitle"`
	ShowDescription  bool   `json:"showDescription"`
	ShowArchived     bool   `json:"showArchived"`
	Sort             string `json:"sort"`
	TagMatch         string `json:"tagMatch"`
	ArchiveAfterDays *int   `json:"archiveAfterDays"` // nil: the system default
	// EffectiveArchiveAfterDays is ArchiveAfterDays or, when nil, the
	// defaultArchiveDays system setting; 0 means never. Read-only.
	EffectiveArchiveAfterDays int `json:"effectiveArchiveAfterDays"`
	// SystemArchiveAfterDays is the current defaultArchiveDays system setting. Read-only.
	SystemArchiveAfterDays int `json:"systemArchiveAfterDays"`
}

// DefaultPreferences returns the column defaults of user_preferences for a
// given system archive period.
func DefaultPreferences(systemArchiveDays int) Preferences {
	return Preferences{
		Locale:                    DefaultLocale,
		Theme:                     ThemeDark,
		ViewMode:                  ViewList,
		ShowIcon:                  true,
		ShowURL:                   true,
		ShowTitle:                 true,
		ShowDescription:           true,
		ShowArchived:              false,
		Sort:                      SortUsage,
		TagMatch:                  TagMatchAny,
		EffectiveArchiveAfterDays: systemArchiveDays,
		SystemArchiveAfterDays:    systemArchiveDays,
	}
}

// PreferencesPatch is a partial preferences update (PATCH /api/me/preferences).
// Absent JSON fields stay nil; ArchiveAfterDays distinguishes absent from null.
type PreferencesPatch struct {
	Locale           *Locale     `json:"locale,omitempty"`
	Theme            *string     `json:"theme,omitempty"`
	ViewMode         *string     `json:"viewMode,omitempty"`
	ShowIcon         *bool       `json:"showIcon,omitempty"`
	ShowURL          *bool       `json:"showUrl,omitempty"`
	ShowTitle        *bool       `json:"showTitle,omitempty"`
	ShowDescription  *bool       `json:"showDescription,omitempty"`
	ShowArchived     *bool       `json:"showArchived,omitempty"`
	Sort             *string     `json:"sort,omitempty"`
	TagMatch         *string     `json:"tagMatch,omitempty"`
	ArchiveAfterDays NullableInt `json:"archiveAfterDays"`
}

// NullableInt is a JSON integer field that distinguishes an absent field
// (Set == false) from an explicit null (Set == true, Value == nil).
type NullableInt struct {
	Set   bool
	Value *int
}

// UnmarshalJSON records that the field was present and decodes its value.
func (n *NullableInt) UnmarshalJSON(b []byte) error {
	n.Set = true
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		n.Value = nil
		return nil
	}
	var v int
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}

// MarshalJSON encodes the value, or null when unset or nil.
func (n NullableInt) MarshalJSON() ([]byte, error) {
	if !n.Set || n.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.Value)
}
