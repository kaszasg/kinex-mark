package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Sign-in, sign-out, me and password change (5.7), plus the account helpers
// shared by the other account and admin handlers: login throttling (6.7),
// password step-up, second-factor checks, audit writing and the Me shape.

const (
	// challengeTTL is the lifetime of a second-step sign-in challenge (5.7).
	challengeTTL = 5 * time.Minute
	// challengeMaxAttempts is the number of codes one challenge accepts.
	challengeMaxAttempts = 5
	// passwordInputMaxRunes bounds a typed password before verification (5.7).
	passwordInputMaxRunes = 1024
	// codeInputMaxRunes bounds a typed TOTP or recovery code.
	codeInputMaxRunes = 64
	// challengeInputMaxRunes bounds a presented challenge token.
	challengeInputMaxRunes = 128
	// lockAuditInterval: auth.login.locked is written at most once per key
	// in this interval (6.10).
	lockAuditInterval = 15 * time.Minute
	// totpIssuer is the issuer of the otpauth URI (6.5).
	totpIssuer = "Kinex Mark"
	// recoveryCodeCount is the number of recovery codes per generation (6.6).
	recoveryCodeCount = 10
	// recoveryCodePurpose is the keyring MAC purpose of recovery code hashes (6.6).
	recoveryCodePurpose = "recovery-code"
)

// Login response statuses (5.5 LoginResponse).
const (
	loginStatusOK           = "ok"
	loginStatusTOTPRequired = "totp_required"
)

// ---------------------------------------------------------------------------
// Response shapes

// permissions is Me.permissions.
type permissions struct {
	CanShare     bool `json:"canShare"`
	CanListUsers bool `json:"canListUsers"`
	CanAdmin     bool `json:"canAdmin"`
}

// meResponse is the Me shape of 5.5.
type meResponse struct {
	ID                     int64             `json:"id"`
	Username               string            `json:"username"`
	DisplayName            string            `json:"displayName"`
	Role                   model.Role        `json:"role"`
	MustChangePassword     bool              `json:"mustChangePassword"`
	TOTPEnabled            bool              `json:"totpEnabled"`
	RecoveryCodesRemaining int               `json:"recoveryCodesRemaining"`
	PrivacyConsentRequired bool              `json:"privacyConsentRequired"`
	PrivacyNoticeVersion   int               `json:"privacyNoticeVersion"`
	Permissions            permissions       `json:"permissions"`
	Preferences            model.Preferences `json:"preferences"`
	SessionID              int64             `json:"sessionId"`
	CreatedAt              time.Time         `json:"createdAt"`
	LastLoginAt            *time.Time        `json:"lastLoginAt"`
}

// loginOKResponse is LoginResponse with status "ok".
type loginOKResponse struct {
	Status string     `json:"status"`
	User   meResponse `json:"user"`
}

// loginChallengeResponse is LoginResponse with status "totp_required".
type loginChallengeResponse struct {
	Status    string    `json:"status"`
	Challenge string    `json:"challenge"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// buildMe assembles the Me shape of u for the session sessionID.
func (s *server) buildMe(ctx context.Context, u model.User, sessionID int64, set model.SystemSettings) (meResponse, error) {
	prefs, err := s.deps.Store.Preferences(ctx, u.ID)
	if err != nil {
		return meResponse{}, err
	}
	remaining := 0
	if u.TOTPEnabled {
		if remaining, err = s.deps.Store.RecoveryCodesRemaining(ctx, u.ID); err != nil {
			return meResponse{}, err
		}
	}
	consent, err := s.consentRequired(ctx, u.ID, set)
	if err != nil {
		return meResponse{}, err
	}
	return meResponse{
		ID:                     u.ID,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		Role:                   u.Role,
		MustChangePassword:     u.MustChangePassword,
		TOTPEnabled:            u.TOTPEnabled,
		RecoveryCodesRemaining: remaining,
		PrivacyConsentRequired: consent,
		PrivacyNoticeVersion:   set.PrivacyNoticeVersion,
		Permissions: permissions{
			CanShare:     u.Role.CanShare(),
			CanListUsers: u.Role.CanListUsers(),
			CanAdmin:     u.Role.IsAdmin(),
		},
		Preferences: prefs,
		SessionID:   sessionID,
		CreatedAt:   u.CreatedAt,
		LastLoginAt: u.LastLoginAt,
	}, nil
}

// writeMe answers 200 Me for the user and session of the request.
func (s *server) writeMe(w http.ResponseWriter, r *http.Request, u model.User, sessionID int64) {
	set, err := s.deps.Store.Settings(r.Context())
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	me, err := s.buildMe(r.Context(), u, sessionID, set)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, me)
}

// ---------------------------------------------------------------------------
// Audit

// auditActor returns the actor reference of u.
func auditActor(u model.User) model.AuditActor {
	id := u.ID
	return model.AuditActor{ID: &id, Username: u.Username}
}

// userTarget returns the audit target of a user account.
func userTarget(u model.User) *model.AuditTarget {
	id := u.ID
	return &model.AuditTarget{Type: model.TargetUser, ID: &id, Name: u.Username}
}

// audit appends e with the client address of r. A failed write is logged and
// does not undo the action the entry records.
func (s *server) audit(r *http.Request, e model.AuditEntry) {
	if e.IP == "" {
		e.IP = truncateBytes(s.clientIP(r), model.SessionIPMaxLen)
	}
	if err := s.deps.Store.AppendAudit(r.Context(), e); err != nil {
		s.log.Error("audit write failed", "err", err, "event", e.Event)
	}
}

// auditUser appends a successful event whose actor is u.
func (s *server) auditUser(r *http.Request, event string, u model.User, target *model.AuditTarget, details map[string]any) {
	s.audit(r, model.AuditEntry{Event: event, Success: true, Actor: auditActor(u), Target: target, Details: details})
}

// lockAuditLimiter remembers when auth.login.locked was last written per
// throttle key, so a locked key is audited at most once per
// lockAuditInterval (6.10). The zero value is ready to use.
type lockAuditLimiter struct {
	mu   sync.Mutex
	last map[string]time.Time
}

// allow reports whether key may be audited at now and records it.
func (l *lockAuditLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.last == nil {
		l.last = map[string]time.Time{}
	}
	if t, ok := l.last[key]; ok && now.Sub(t) < lockAuditInterval {
		return false
	}
	for k, t := range l.last {
		if now.Sub(t) >= lockAuditInterval {
			delete(l.last, k)
		}
	}
	l.last[key] = now
	return true
}

// ---------------------------------------------------------------------------
// Login throttling (6.7)

// throttleKey is one throttle row with its rule and audit scope.
type throttleKey struct {
	key   model.ThrottleKey
	rule  auth.ThrottleRule
	scope string // model.ScopeUser or model.ScopeIP
}

// throttleBlock describes a refused attempt: the blocked keys and the longest
// remaining wait.
type throttleBlock struct {
	keys       []throttleKey
	retryAfter time.Duration
}

// errThrottleBlocked rolls back the reservation transaction of a blocked attempt.
var errThrottleBlocked = errors.New("throttle blocked")

// userThrottleKey is the "u:" key of an account (6.7); userID is 0 for an
// unknown username.
func (s *server) userThrottleKey(usernameKey string, userID int64) throttleKey {
	k := model.ThrottleKey{Key: auth.UserThrottleKey(s.deps.Keyring, usernameKey)}
	if userID != 0 {
		id := userID
		k.UserID = &id
	}
	return throttleKey{key: k, rule: auth.UserRule, scope: model.ScopeUser}
}

// signInThrottleKeys returns the user key and, when the client address is
// known, the IP key of a sign-in attempt.
func (s *server) signInThrottleKeys(r *http.Request, user throttleKey) []throttleKey {
	keys := []throttleKey{user}
	if addr := s.clientAddr(r); addr.IsValid() {
		keys = append(keys, throttleKey{
			key:   model.ThrottleKey{Key: auth.IPThrottleKey(addr)},
			rule:  auth.IPRule,
			scope: model.ScopeIP,
		})
	}
	return keys
}

// throttleState converts a stored row into the rule state.
func throttleState(row model.ThrottleRow) auth.ThrottleState {
	return auth.ThrottleState{
		Failures:    row.Failures,
		WindowStart: row.WindowStart,
		LockedUntil: row.LockedUntil,
		Lockouts:    row.Lockouts,
	}
}

// withThrottleState returns row carrying st.
func withThrottleState(row model.ThrottleRow, st auth.ThrottleState) model.ThrottleRow {
	row.Failures = st.Failures
	row.WindowStart = st.WindowStart
	row.LockedUntil = st.LockedUntil
	row.Lockouts = st.Lockouts
	return row
}

// rowRules maps the row keys of keys to their throttle keys.
func rowRules(keys []throttleKey) ([]model.ThrottleKey, map[string]throttleKey) {
	list := make([]model.ThrottleKey, len(keys))
	byKey := make(map[string]throttleKey, len(keys))
	for i, k := range keys {
		list[i] = k.key
		byKey[k.key.Key] = k
	}
	return list, byKey
}

// reserveAttempt checks every key and reserves one attempt on each, in one
// write transaction (6.7). When any key is blocked nothing is reserved and
// the block is returned.
func (s *server) reserveAttempt(ctx context.Context, keys []throttleKey) (*throttleBlock, error) {
	list, byKey := rowRules(keys)
	now := s.now()
	block := &throttleBlock{}
	seen := 0
	err := s.deps.Store.ThrottleUpdate(ctx, list, func(row model.ThrottleRow) (model.ThrottleRow, error) {
		seen++
		k := byKey[row.Key]
		st := throttleState(row)
		if blocked, after := k.rule.Check(st, now); blocked {
			block.keys = append(block.keys, k)
			block.retryAfter = max(block.retryAfter, after)
		} else {
			row = withThrottleState(row, k.rule.Reserve(st, now))
		}
		if seen == len(list) && len(block.keys) > 0 {
			return row, errThrottleBlocked
		}
		return row, nil
	})
	if errors.Is(err, errThrottleBlocked) {
		return block, nil
	}
	return nil, err
}

// confirmAttempt records a verified failure on every key (the reservation
// already counted it; Confirm locks at the threshold).
func (s *server) confirmAttempt(ctx context.Context, keys []throttleKey) error {
	list, byKey := rowRules(keys)
	now := s.now()
	return s.deps.Store.ThrottleUpdate(ctx, list, func(row model.ThrottleRow) (model.ThrottleRow, error) {
		return withThrottleState(row, byKey[row.Key].rule.Confirm(throttleState(row), now)), nil
	})
}

// refundAttempt returns the reservation of a verified success on every key.
func (s *server) refundAttempt(ctx context.Context, keys []throttleKey) error {
	list, byKey := rowRules(keys)
	return s.deps.Store.ThrottleUpdate(ctx, list, func(row model.ThrottleRow) (model.ThrottleRow, error) {
		return withThrottleState(row, byKey[row.Key].rule.Refund(throttleState(row))), nil
	})
}

// auditLocked writes auth.login.locked for every blocked key that was not
// audited in the last lockAuditInterval. actor is the account when known.
func (s *server) auditLocked(r *http.Request, b *throttleBlock, actor model.AuditActor) {
	for _, k := range b.keys {
		if !s.lockAudits.allow(k.key.Key, s.now()) {
			continue
		}
		s.audit(r, model.AuditEntry{
			Event:   model.EventLoginLocked,
			Success: false,
			Actor:   actor,
			Details: map[string]any{model.DetailScope: k.scope},
		})
	}
}

// ---------------------------------------------------------------------------
// Credential checks

// passwordMatches verifies password against the stored hash. A malformed
// stored hash never matches (seeded or corrupted rows).
func (s *server) passwordMatches(c model.Credentials, password string) (ok, needsRehash bool) {
	ok, needsRehash, err := s.deps.Hasher.Verify(c.PasswordHash, password)
	if err != nil {
		s.log.Warn("stored password hash is unusable", "user_id", c.User.ID, "err", err)
		return false, false
	}
	return ok, needsRehash
}

// isTOTPCode reports whether code has the shape of a TOTP code: 6 digits.
func isTOTPCode(code string) bool {
	if len(code) != auth.TOTPDigits {
		return false
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return false
		}
	}
	return true
}

// totpAssociatedData is the keyring associated data of a user's TOTP secret (6.5).
func totpAssociatedData(userID int64) string {
	return "totp:" + strconv.FormatInt(userID, 10)
}

// recoveryCodeHash is the stored form of a recovery code (6.6).
func (s *server) recoveryCodeHash(code string) string {
	return s.deps.Keyring.MAC(recoveryCodePurpose, []byte(auth.NormalizeRecoveryCode(code)))
}

// recoveryCodeHashes returns the stored forms of codes.
func (s *server) recoveryCodeHashes(codes []string) []string {
	out := make([]string, len(codes))
	for i, c := range codes {
		out[i] = s.recoveryCodeHash(c)
	}
	return out
}

// verifyTOTP checks a TOTP code against the enabled secret of c and records
// the accepted step, so the same code is never accepted twice (6.5).
func (s *server) verifyTOTP(ctx context.Context, c model.Credentials, code string) (bool, error) {
	if c.TOTPSecret == "" || !isTOTPCode(code) {
		return false, nil
	}
	secret, err := s.deps.Keyring.Open(c.TOTPSecret, totpAssociatedData(c.User.ID))
	if err != nil {
		return false, err
	}
	step, ok := auth.VerifyTOTP(string(secret), code, s.now(), c.TOTPLastStep)
	if !ok {
		return false, nil
	}
	return s.deps.Store.AdvanceTOTPStep(ctx, c.User.ID, step)
}

// verifyRecoveryCode uses an unused recovery code of the user (6.6).
func (s *server) verifyRecoveryCode(ctx context.Context, c model.Credentials, code string) (bool, error) {
	if auth.NormalizeRecoveryCode(code) == "" {
		return false, nil
	}
	return s.deps.Store.UseRecoveryCode(ctx, c.User.ID, s.recoveryCodeHash(code))
}

// verifySecondFactor accepts a TOTP code (6 digits) or a recovery code and
// returns the audit method of the accepted one.
func (s *server) verifySecondFactor(ctx context.Context, c model.Credentials, code string) (string, bool, error) {
	if isTOTPCode(code) {
		ok, err := s.verifyTOTP(ctx, c, code)
		return model.MethodTOTP, ok, err
	}
	ok, err := s.verifyRecoveryCode(ctx, c, code)
	return model.MethodRecoveryCode, ok, err
}

// stepUpCheck verifies a credential inside a reservation: a non-nil
// *apiError is a verified failure, an error an unexpected failure.
type stepUpCheck func(c model.Credentials) (*apiError, error)

// stepUp confirms a sensitive action of u (5.7, 5.12): it reserves an attempt
// on the user's throttle key, runs check on the current credentials, then
// confirms a verified failure or refunds a success. It returns nil on
// success, the 429 or the check's *apiError on refusal, or an unexpected error.
func (s *server) stepUp(ctx context.Context, u model.User, check stepUpCheck) error {
	keys := []throttleKey{s.userThrottleKey(u.UsernameKey, u.ID)}
	block, err := s.reserveAttempt(ctx, keys)
	if err != nil {
		return err
	}
	if block != nil {
		return tooManyAttempts(block.retryAfter)
	}
	cred, err := s.deps.Store.Credentials(ctx, u.UsernameKey)
	if err != nil {
		return err
	}
	refusal, err := check(cred)
	if err != nil {
		return err
	}
	if refusal != nil {
		if err := s.confirmAttempt(ctx, keys); err != nil {
			return err
		}
		return refusal
	}
	return s.refundAttempt(ctx, keys)
}

// passwordStep is the step-up check of the caller's own password; a wrong
// password is 400 current_password_invalid (5.12).
func (s *server) passwordStep(password string) stepUpCheck {
	return func(c model.Credentials) (*apiError, error) {
		if ok, _ := s.passwordMatches(c, password); !ok {
			return currentPasswordError(), nil
		}
		return nil, nil
	}
}

// checkPasswordInput validates a typed password (not trimmed): required and
// at most passwordInputMaxRunes runes.
func checkPasswordInput(p string) string {
	switch {
	case p == "":
		return fieldRequired
	case utf8.RuneCountInString(p) > passwordInputMaxRunes:
		return fieldTooLong
	}
	return ""
}

// checkCodeInput validates a typed TOTP or recovery code: required, at most
// codeInputMaxRunes runes.
func checkCodeInput(code string) string {
	switch {
	case code == "":
		return fieldRequired
	case utf8.RuneCountInString(code) > codeInputMaxRunes:
		return fieldTooLong
	}
	return ""
}

// ---------------------------------------------------------------------------
// Handlers

// loginRequest is the body of POST /api/auth/login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// login is POST /api/auth/login (5.7 steps 1-8).
func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	username := f.text("username", req.Username, 1, model.UsernameMaxLen, false)
	f.add("password", checkPasswordInput(req.Password))
	if f.writeIfAny(w) {
		return
	}
	ctx := r.Context()
	usernameKey := model.UsernameKey(username)

	// Step 1: the account, so that its "u:" row carries the user id.
	cred, err := s.deps.Store.Credentials(ctx, usernameKey)
	known := err == nil
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		writeFailure(w, r, s.log, err)
		return
	}
	userKey := s.userThrottleKey(usernameKey, cred.User.ID)
	keys := s.signInThrottleKeys(r, userKey)
	actor := model.AuditActor{}
	if known {
		actor = auditActor(cred.User)
	}

	// Step 2: reserve before any verification.
	block, err := s.reserveAttempt(ctx, keys)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if block != nil {
		s.auditLocked(r, block, actor)
		writeError(w, tooManyAttempts(block.retryAfter))
		return
	}

	// Step 3: unknown account. The typed name is never stored.
	if !known {
		s.deps.Hasher.DummyVerify(req.Password)
		if err := s.confirmAttempt(ctx, keys); err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		s.audit(r, model.AuditEntry{
			Event:   model.EventLoginFailure,
			Details: map[string]any{model.DetailReason: model.ReasonUnknownUser},
		})
		writeError(w, newError(codeInvalidCredentials))
		return
	}

	// Step 4: wrong password.
	ok, rehash := s.passwordMatches(cred, req.Password)
	if !ok {
		if err := s.confirmAttempt(ctx, keys); err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		s.audit(r, model.AuditEntry{
			Event:   model.EventLoginFailure,
			Actor:   actor,
			Details: map[string]any{model.DetailReason: model.ReasonCredentials},
		})
		writeError(w, newError(codeInvalidCredentials))
		return
	}

	// Step 5: correct password, inactive account.
	if !cred.User.Active {
		if err := s.refundAttempt(ctx, keys); err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		s.audit(r, model.AuditEntry{
			Event:   model.EventLoginFailure,
			Actor:   actor,
			Details: map[string]any{model.DetailReason: model.ReasonDisabled},
		})
		writeError(w, newError(codeAccountDisabled))
		return
	}

	// Step 6: refund the address key; the user key waits for the second
	// factor when 2FA is on.
	if err := s.refundAttempt(ctx, keys[1:]); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if rehash {
		s.rehashPassword(ctx, cred.User, req.Password)
	}

	// Step 7: second factor.
	if cred.User.TOTPEnabled {
		token, hash, err := auth.NewToken()
		if err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		expires := s.now().Add(challengeTTL)
		ip := truncateBytes(s.clientIP(r), model.SessionIPMaxLen)
		if err := s.deps.Store.CreateChallenge(ctx, hash, cred.User.ID, ip, expires); err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		writeJSON(w, http.StatusOK, loginChallengeResponse{
			Status:    loginStatusTOTPRequired,
			Challenge: token,
			ExpiresAt: expires,
		})
		return
	}

	// Step 8: session.
	if err := s.deps.Store.ClearThrottle(ctx, userKey.key.Key); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.completeLogin(w, r, cred.User, model.MethodPassword)
}

// rehashPassword stores a hash with the current iteration count after a
// successful verification of an older hash (6.4). It is best effort: a
// failure is logged and the sign-in continues.
func (s *server) rehashPassword(ctx context.Context, u model.User, password string) {
	hash, err := s.deps.Hasher.Hash(password)
	if err == nil {
		err = s.deps.Store.SetPassword(ctx, u.ID, hash, u.MustChangePassword)
	}
	if err != nil {
		s.log.Warn("password rehash failed", "user_id", u.ID, "err", err)
	}
}

// completeLogin creates the session (with rotation), records the sign-in,
// audits it and answers {status: "ok", user: Me}.
func (s *server) completeLogin(w http.ResponseWriter, r *http.Request, u model.User, method string) {
	ctx := r.Context()
	sess, err := s.startSession(ctx, w, r, u)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.deps.Store.RecordLogin(ctx, u.ID); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.EventLoginSuccess, u, nil, map[string]any{model.DetailMethod: method})
	fresh, err := s.deps.Store.UserByID(ctx, u.ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	set, err := s.deps.Store.Settings(ctx)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	me, err := s.buildMe(ctx, fresh, sess.ID, set)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, loginOKResponse{Status: loginStatusOK, User: me})
}

// loginTOTPRequest is the body of POST /api/auth/login/totp: exactly one of
// code and recoveryCode.
type loginTOTPRequest struct {
	Challenge    string  `json:"challenge"`
	Code         *string `json:"code"`
	RecoveryCode *string `json:"recoveryCode"`
}

// validate checks the request shape (400 validation_failed).
func (req loginTOTPRequest) validate() fieldErrors {
	f := fieldErrors{}
	switch {
	case req.Challenge == "":
		f.add("challenge", fieldRequired)
	case utf8.RuneCountInString(req.Challenge) > challengeInputMaxRunes:
		f.add("challenge", fieldTooLong)
	}
	switch {
	case req.Code != nil && req.RecoveryCode != nil:
		f.add("recoveryCode", fieldNotAllowed)
	case req.Code != nil:
		if !isTOTPCode(*req.Code) {
			f.add("code", fieldInvalidFormat)
		}
	case req.RecoveryCode != nil:
		f.add("recoveryCode", checkCodeInput(*req.RecoveryCode))
	default:
		f.add("code", fieldRequired)
	}
	return f
}

// loginTOTP is POST /api/auth/login/totp, the second sign-in step (5.7).
func (s *server) loginTOTP(w http.ResponseWriter, r *http.Request) {
	var req loginTOTPRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.validate().writeIfAny(w) {
		return
	}
	ctx := r.Context()
	hash := auth.HashToken(req.Challenge)

	// Step 1: reserve one attempt of the challenge.
	ch, err := s.deps.Store.ChallengeAttempt(ctx, hash, challengeMaxAttempts)
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, newError(codeChallengeInvalid))
		return
	}
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	user, err := s.deps.Store.UserByID(ctx, ch.UserID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}

	// Step 2: the user and address keys.
	keys := s.signInThrottleKeys(r, s.userThrottleKey(user.UsernameKey, user.ID))
	block, err := s.reserveAttempt(ctx, keys)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if block != nil {
		s.auditLocked(r, block, auditActor(user))
		writeError(w, tooManyAttempts(block.retryAfter))
		return
	}
	cred, err := s.deps.Store.Credentials(ctx, user.UsernameKey)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if !cred.User.Active || !cred.User.TOTPEnabled {
		// The account changed since the password step.
		s.dropChallenge(w, r, hash, keys)
		return
	}

	// Step 3: verify the code.
	method := model.MethodTOTP
	var ok bool
	if req.Code != nil {
		ok, err = s.verifyTOTP(ctx, cred, *req.Code)
	} else {
		method = model.MethodRecoveryCode
		ok, err = s.verifyRecoveryCode(ctx, cred, *req.RecoveryCode)
	}
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}

	// Step 4: failure on both keys.
	if !ok {
		if err := s.confirmAttempt(ctx, keys); err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		s.audit(r, model.AuditEntry{
			Event:   model.EventLoginFailure,
			Actor:   auditActor(cred.User),
			Details: map[string]any{model.DetailReason: model.ReasonTOTP},
		})
		if ch.Attempts >= challengeMaxAttempts {
			if err := s.deps.Store.DeleteChallenge(ctx, hash); err != nil {
				writeFailure(w, r, s.log, err)
				return
			}
			writeError(w, newError(codeChallengeInvalid))
			return
		}
		writeError(w, newError(codeInvalidTOTP))
		return
	}

	// Step 5: success.
	if err := s.deps.Store.DeleteChallenge(ctx, hash); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.deps.Store.ClearThrottle(ctx, keys[0].key.Key); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.refundAttempt(ctx, keys[1:]); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if method == model.MethodRecoveryCode {
		remaining, err := s.deps.Store.RecoveryCodesRemaining(ctx, cred.User.ID)
		if err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		s.auditUser(r, model.EventRecoveryCodeUsed, cred.User, nil, map[string]any{model.DetailRemaining: remaining})
	}
	s.completeLogin(w, r, cred.User, method)
}

// dropChallenge deletes a challenge that can no longer succeed, refunds the
// reservation and answers 401 challenge_invalid.
func (s *server) dropChallenge(w http.ResponseWriter, r *http.Request, hash []byte, keys []throttleKey) {
	ctx := r.Context()
	if err := s.deps.Store.DeleteChallenge(ctx, hash); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.refundAttempt(ctx, keys); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeError(w, newError(codeChallengeInvalid))
}

// logout is POST /api/auth/logout: the presented session row is deleted (if
// any), then the cookie (6.2). auth.logout is audited for a valid session.
func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := s.sessionToken(r); ok {
		ctx := r.Context()
		hash := auth.HashToken(token)
		sess, user, err := s.deps.Store.SessionByToken(ctx, hash)
		found := err == nil
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			writeFailure(w, r, s.log, err)
			return
		}
		valid := false
		if found {
			set, err := s.deps.Store.Settings(ctx)
			if err != nil {
				writeFailure(w, r, s.log, err)
				return
			}
			valid = user.Active && !sessionExpired(sess, set, s.now())
		}
		if err := s.deps.Store.DeleteSessionByToken(ctx, hash); err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
		if valid {
			s.auditUser(r, model.EventLogout, user, nil, nil)
		}
	}
	s.clearSessionCookie(w, r)
	writeNoContent(w)
}

// me is GET /api/auth/me.
func (s *server) me(w http.ResponseWriter, r *http.Request) {
	s.writeMe(w, r, userFrom(r), sessionFrom(r).ID)
}

// passwordChangeRequest is the body of POST /api/auth/password.
type passwordChangeRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// changePassword is POST /api/auth/password (5.7): the current password is
// confirmed through the throttle, the new one must satisfy the policy; the
// other sessions are deleted.
func (s *server) changePassword(w http.ResponseWriter, r *http.Request) {
	var req passwordChangeRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	f.add("currentPassword", checkPasswordInput(req.CurrentPassword))
	if req.NewPassword == "" {
		f.add("newPassword", fieldRequired)
	}
	if f.writeIfAny(w) {
		return
	}
	ctx := r.Context()
	u := userFrom(r)
	if err := s.stepUp(ctx, u, s.passwordStep(req.CurrentPassword)); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if code := auth.CheckPasswordPolicy(req.NewPassword, u.Username, req.CurrentPassword); code != "" {
		writeValidation(w, map[string]string{"newPassword": code})
		return
	}
	hash, err := s.deps.Hasher.Hash(req.NewPassword)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.deps.Store.SetPassword(ctx, u.ID, hash, false); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if _, err := s.deps.Store.DeleteUserSessions(ctx, u.ID, sessionFrom(r).ID); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.EventPasswordChanged, u, nil, nil)
	writeNoContent(w)
}
