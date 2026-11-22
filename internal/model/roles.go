package model

// Role is a user role (D5).
type Role string

// The three roles.
const (
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleCustomer  Role = "customer"
)

// Roles lists every role in the canonical order (contract golden, 9.6).
var Roles = []Role{RoleAdmin, RoleDeveloper, RoleCustomer}

// Valid reports whether r is a known role.
func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleDeveloper, RoleCustomer:
		return true
	}
	return false
}

// CanShare reports whether the role may share links (admin, developer).
func (r Role) CanShare() bool { return r == RoleAdmin || r == RoleDeveloper }

// CanListUsers reports whether the role may read the user directory (admin, developer).
func (r Role) CanListUsers() bool { return r == RoleAdmin || r == RoleDeveloper }

// IsAdmin reports whether the role is admin.
func (r Role) IsAdmin() bool { return r == RoleAdmin }

// Locale is a user interface language.
type Locale string

// Supported locales.
const (
	LocaleHU Locale = "hu"
	LocaleEN Locale = "en"
	LocaleDE Locale = "de"
)

// DefaultLocale is the locale of new users and of the public pages.
const DefaultLocale = LocaleHU

// Locales lists the supported locales in the canonical order.
var Locales = []Locale{LocaleHU, LocaleEN, LocaleDE}

// Valid reports whether l is a supported locale.
func (l Locale) Valid() bool {
	switch l {
	case LocaleHU, LocaleEN, LocaleDE:
		return true
	}
	return false
}
