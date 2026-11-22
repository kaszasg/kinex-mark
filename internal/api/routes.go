package api

import (
	"net/http"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// access is the access level of a route (5.13).
type access int

// Access levels: N, K, B, D and A of the 5.13 matrix.
const (
	accessPublic access = iota // N: anyone, no session needed
	accessExempt               // K: signed in, exempt from the password-change and consent gates
	accessUser                 // B: any signed-in role, gates apply
	accessSharer               // D: admin and developer, gates apply
	accessAdmin                // A: admin only, gates apply
)

// String returns the 5.13 letter of the level.
func (a access) String() string {
	switch a {
	case accessPublic:
		return "N"
	case accessExempt:
		return "K"
	case accessUser:
		return "B"
	case accessSharer:
		return "D"
	case accessAdmin:
		return "A"
	}
	return "?"
}

// handlerFunc is a route handler bound to the server at registration.
type handlerFunc func(s *server, w http.ResponseWriter, r *http.Request)

// route is one row of the route table. The mux registration and the
// permission matrix test (routes_test.go) both walk this table.
type route struct {
	method  string
	pattern string
	access  access
	// deny is the error code for a signed-in user whose role is below the
	// level: share_not_allowed or forbidden on D routes, forbidden on A
	// routes, "" otherwise.
	deny   string
	handle handlerFunc
}

// key is the ServeMux pattern, for example "GET /api/links/{id}".
func (rt route) key() string { return rt.method + " " + rt.pattern }

// needsSession reports whether the route requires a valid session.
func (rt route) needsSession() bool { return rt.access != accessPublic }

// gated reports whether the password-change and consent gates apply (6.2).
func (rt route) gated() bool { return rt.access >= accessUser }

// permits reports whether a signed-in user of the role may call the route.
func (rt route) permits(role model.Role) bool {
	switch rt.access {
	case accessSharer:
		return role.CanShare()
	case accessAdmin:
		return role.IsAdmin()
	}
	return true
}

// routes returns the route table (5.6-5.13). A GET pattern also serves HEAD.
func routes() []route {
	const (
		share  = codeShareNotAllowed
		forbid = codeForbidden
	)
	return []route{
		// 5.6 public
		{http.MethodGet, "/healthz", accessPublic, "", (*server).healthz},
		{http.MethodGet, "/api/public/config", accessPublic, "", (*server).publicConfig},

		// 5.7 authentication
		{http.MethodPost, "/api/auth/login", accessPublic, "", (*server).login},
		{http.MethodPost, "/api/auth/login/totp", accessPublic, "", (*server).loginTOTP},
		{http.MethodPost, "/api/auth/logout", accessPublic, "", (*server).logout},
		{http.MethodGet, "/api/auth/me", accessExempt, "", (*server).me},
		{http.MethodPost, "/api/auth/password", accessExempt, "", (*server).changePassword},

		// 5.8 own account
		{http.MethodPatch, "/api/me/profile", accessUser, "", (*server).updateProfile},
		{http.MethodGet, "/api/me/preferences", accessExempt, "", (*server).getPreferences},
		{http.MethodPatch, "/api/me/preferences", accessExempt, "", (*server).updatePreferences},
		{http.MethodGet, "/api/me/sessions", accessUser, "", (*server).listSessions},
		{http.MethodDelete, "/api/me/sessions/{id}", accessUser, "", (*server).revokeSession},
		{http.MethodPost, "/api/me/sessions/revoke-others", accessUser, "", (*server).revokeOtherSessions},
		{http.MethodPost, "/api/me/2fa/setup", accessUser, "", (*server).setupTwoFactor},
		{http.MethodPost, "/api/me/2fa/enable", accessUser, "", (*server).enableTwoFactor},
		{http.MethodPost, "/api/me/2fa/disable", accessUser, "", (*server).disableTwoFactor},
		{http.MethodPost, "/api/me/2fa/recovery-codes", accessUser, "", (*server).regenerateRecoveryCodes},
		{http.MethodGet, "/api/me/privacy", accessExempt, "", (*server).privacyStatus},
		{http.MethodPost, "/api/me/privacy/consent", accessExempt, "", (*server).acceptPrivacy},
		{http.MethodGet, "/api/me/export", accessExempt, "", (*server).exportMyData},
		{http.MethodPost, "/api/me/delete", accessExempt, "", (*server).deleteAccount},

		// 5.9 links
		{http.MethodGet, "/api/links", accessUser, "", (*server).listLinks},
		{http.MethodPost, "/api/links", accessUser, "", (*server).createLink},
		{http.MethodPost, "/api/links/quick", accessUser, "", (*server).quickCapture},
		{http.MethodGet, "/api/links/export", accessUser, "", (*server).exportLinks},
		{http.MethodPost, "/api/links/import", accessUser, "", (*server).importLinks},
		{http.MethodGet, "/api/links/{id}", accessUser, "", (*server).getLink},
		{http.MethodPatch, "/api/links/{id}", accessUser, "", (*server).updateLink},
		{http.MethodDelete, "/api/links/{id}", accessUser, "", (*server).deleteLink},
		{http.MethodPost, "/api/links/{id}/refresh", accessUser, "", (*server).refreshLink},
		{http.MethodPut, "/api/links/{id}/tags", accessUser, "", (*server).setLinkTags},
		{http.MethodPost, "/api/links/{id}/pin", accessUser, "", (*server).pinLink},
		{http.MethodDelete, "/api/links/{id}/pin", accessUser, "", (*server).unpinLink},
		{http.MethodPost, "/api/links/{id}/archive", accessUser, "", (*server).archiveLink},
		{http.MethodDelete, "/api/links/{id}/archive", accessUser, "", (*server).restoreLink},
		{http.MethodPost, "/api/links/{id}/visit", accessUser, "", (*server).visitLink},
		{http.MethodGet, "/api/links/{id}/icon", accessUser, "", (*server).linkIcon},
		{http.MethodGet, "/api/links/{id}/shares", accessSharer, share, (*server).listShares},
		{http.MethodPost, "/api/links/{id}/shares", accessSharer, share, (*server).createShare},
		{http.MethodDelete, "/api/links/{id}/shares/{shareId}", accessSharer, share, (*server).deleteShare},

		// 5.10 tags
		{http.MethodGet, "/api/tags", accessUser, "", (*server).listTags},
		{http.MethodPost, "/api/tags", accessUser, "", (*server).createTag},
		{http.MethodPatch, "/api/tags/{id}", accessUser, "", (*server).updateTag},
		{http.MethodDelete, "/api/tags/{id}", accessUser, "", (*server).deleteTag},

		// 5.11 user directory
		{http.MethodGet, "/api/users/directory", accessSharer, forbid, (*server).searchDirectory},

		// 5.12 admin
		{http.MethodGet, "/api/admin/users", accessAdmin, forbid, (*server).adminListUsers},
		{http.MethodPost, "/api/admin/users", accessAdmin, forbid, (*server).adminCreateUser},
		{http.MethodGet, "/api/admin/users/{id}", accessAdmin, forbid, (*server).adminGetUser},
		{http.MethodPatch, "/api/admin/users/{id}", accessAdmin, forbid, (*server).adminUpdateUser},
		{http.MethodDelete, "/api/admin/users/{id}", accessAdmin, forbid, (*server).adminDeleteUser},
		{http.MethodPost, "/api/admin/users/{id}/reset-password", accessAdmin, forbid, (*server).adminResetPassword},
		{http.MethodPost, "/api/admin/users/{id}/reset-2fa", accessAdmin, forbid, (*server).adminResetTwoFactor},
		{http.MethodPost, "/api/admin/users/{id}/unlock", accessAdmin, forbid, (*server).adminUnlockUser},
		{http.MethodDelete, "/api/admin/users/{id}/sessions", accessAdmin, forbid, (*server).adminRevokeSessions},
		{http.MethodGet, "/api/admin/settings", accessAdmin, forbid, (*server).adminGetSettings},
		{http.MethodPatch, "/api/admin/settings", accessAdmin, forbid, (*server).adminUpdateSettings},
		{http.MethodPost, "/api/admin/settings/privacy-notice/bump", accessAdmin, forbid, (*server).adminBumpPrivacyNotice},
		{http.MethodGet, "/api/admin/audit", accessAdmin, forbid, (*server).adminListAudit},
		{http.MethodPost, "/api/admin/jobs/archive", accessAdmin, forbid, (*server).adminRunArchive},
	}
}
