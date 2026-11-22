package api

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Session cookie names (6.2): the __Host- prefix in secure mode.
const (
	sessionCookieSecure = "__Host-kxm_session"
	sessionCookiePlain  = "kxm_session"
)

// touchInterval is the minimum time between two last_seen_at updates (6.2).
const touchInterval = time.Minute

// sessionCookieName returns the session cookie name of a security mode.
func sessionCookieName(secure bool) string {
	if secure {
		return sessionCookieSecure
	}
	return sessionCookiePlain
}

// sessionToken returns the session token presented by r, if any.
func (s *server) sessionToken(r *http.Request) (string, bool) {
	c, err := r.Cookie(sessionCookieName(s.secureMode(r)))
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}

// setSessionCookie sets the session cookie (6.2): Path=/, HttpOnly,
// SameSite=Strict, Secure in secure mode, Max-Age until the absolute expiry,
// no Domain.
func (s *server) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	secure := s.secureMode(r)
	maxAge := int(math.Ceil(expires.Sub(s.now()).Seconds()))
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName(secure),
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// clearSessionCookie deletes the session cookie in the browser.
func (s *server) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	secure := s.secureMode(r)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName(secure),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// truncateBytes cuts s to at most n bytes without splitting a rune.
func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// startSession creates a session for u and sets its cookie (sign-in, 6.2).
// A valid session cookie presented with r is deleted first (rotation).
// The absolute expiry comes from the current sessionAbsoluteHours setting.
func (s *server) startSession(ctx context.Context, w http.ResponseWriter, r *http.Request, u model.User) (model.Session, error) {
	if old, ok := s.sessionToken(r); ok {
		if err := s.deps.Store.DeleteSessionByToken(ctx, auth.HashToken(old)); err != nil {
			return model.Session{}, err
		}
	}
	set, err := s.deps.Store.Settings(ctx)
	if err != nil {
		return model.Session{}, err
	}
	token, hash, err := auth.NewToken()
	if err != nil {
		return model.Session{}, err
	}
	expires := s.now().Add(time.Duration(set.SessionAbsoluteHours) * time.Hour)
	sess, err := s.deps.Store.CreateSession(ctx, model.NewSession{
		TokenHash: hash,
		UserID:    u.ID,
		ExpiresAt: expires,
		IP:        truncateBytes(s.clientIP(r), model.SessionIPMaxLen),
		UserAgent: truncateBytes(r.UserAgent(), model.SessionUserAgentMaxLen),
	})
	if err != nil {
		return model.Session{}, err
	}
	s.setSessionCookie(w, r, token, sess.ExpiresAt)
	return sess, nil
}

// sessionExpired reports whether a session is past its absolute expiry, its
// absolute lifetime under the current setting, or its idle limit (6.2).
func sessionExpired(sess model.Session, set model.SystemSettings, now time.Time) bool {
	idle := time.Duration(set.SessionIdleMinutes) * time.Minute
	absolute := time.Duration(set.SessionAbsoluteHours) * time.Hour
	return !now.Before(sess.ExpiresAt) ||
		!now.Before(sess.CreatedAt.Add(absolute)) ||
		!now.Before(sess.LastSeenAt.Add(idle))
}

// errNoSession reports a missing, unknown, expired or inactive session.
var errNoSession = errors.New("no valid session")

// authenticate loads the session of r and its user with one query (6.2).
// A missing cookie is errNoSession. An unknown or expired session, or one
// whose user is inactive, is deleted, its cookie is cleared and the error is
// errNoSession. last_seen_at is refreshed at most once per touchInterval.
// It also returns the current system settings, which requireReady reuses.
func (s *server) authenticate(w http.ResponseWriter, r *http.Request) (authInfo, model.SystemSettings, error) {
	ctx := r.Context()
	token, ok := s.sessionToken(r)
	if !ok {
		return authInfo{}, model.SystemSettings{}, errNoSession
	}
	hash := auth.HashToken(token)
	sess, user, err := s.deps.Store.SessionByToken(ctx, hash)
	if errors.Is(err, model.ErrNotFound) {
		s.clearSessionCookie(w, r)
		return authInfo{}, model.SystemSettings{}, errNoSession
	}
	if err != nil {
		return authInfo{}, model.SystemSettings{}, err
	}
	set, err := s.deps.Store.Settings(ctx)
	if err != nil {
		return authInfo{}, model.SystemSettings{}, err
	}
	now := s.now()
	if !user.Active || sessionExpired(sess, set, now) {
		if err := s.deps.Store.DeleteSessionByToken(ctx, hash); err != nil {
			return authInfo{}, model.SystemSettings{}, err
		}
		s.clearSessionCookie(w, r)
		return authInfo{}, model.SystemSettings{}, errNoSession
	}
	if now.Sub(sess.LastSeenAt) >= touchInterval {
		if err := s.deps.Store.TouchSession(ctx, sess.ID); err != nil {
			return authInfo{}, model.SystemSettings{}, err
		}
		sess.LastSeenAt = now
	}
	return authInfo{User: user, Session: sess}, set, nil
}

// consentRequired reports whether the user has not acknowledged the current
// privacy notice version (6.2, Me.privacyConsentRequired).
func (s *server) consentRequired(ctx context.Context, userID int64, set model.SystemSettings) (bool, error) {
	consents, err := s.deps.Store.Consents(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, c := range consents {
		if c.Kind == model.ConsentPrivacyNotice && c.Version == set.PrivacyNoticeVersion {
			return false, nil
		}
	}
	return true, nil
}

// requireReady is the gate of every B, D and A route (6.2): 403
// password_change_required while a password change is pending, otherwise
// 403 consent_required while the current privacy notice is not
// acknowledged. It returns nil when the user may proceed.
func (s *server) requireReady(ctx context.Context, u model.User, set model.SystemSettings) error {
	if u.MustChangePassword {
		return newError(codePasswordChangeRequired)
	}
	required, err := s.consentRequired(ctx, u.ID, set)
	if err != nil {
		return err
	}
	if required {
		return newError(codeConsentRequired)
	}
	return nil
}

// requireRole returns the route's deny error when the role is below its
// access level (5.13), nil otherwise.
func requireRole(rt route, role model.Role) *apiError {
	if rt.permits(role) {
		return nil
	}
	return newError(rt.deny)
}

// guard wraps a route handler with the per-route checks (6.1): for every
// non-public route authenticate (401 unauthenticated), then requireReady on
// gated routes, then the role check. The handler finds the identity through
// userFrom, sessionFrom and viewerFrom.
func (s *server) guard(rt route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rt.needsSession() {
			rt.handle(s, w, r)
			return
		}
		a, set, err := s.authenticate(w, r)
		if errors.Is(err, errNoSession) {
			writeError(w, newError(codeUnauthenticated))
			return
		}
		if err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		if rt.gated() {
			if err := s.requireReady(r.Context(), a.User, set); err != nil {
				writeFailure(w, r, s.log, err)
				return
			}
		}
		if denied := requireRole(rt, a.User.Role); denied != nil {
			writeError(w, denied)
			return
		}
		rt.handle(s, w, r.WithContext(withAuth(r.Context(), a)))
	}
}
