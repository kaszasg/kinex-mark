package api

import (
	"net/http"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Admin user management (5.12).

const (
	// adminUsersDefaultLimit and adminUsersMaxLimit page the user list (5.4).
	adminUsersDefaultLimit = 50
	adminUsersMaxLimit     = 200
	// adminUserQueryMaxRunes bounds the search text of the user list.
	adminUserQueryMaxRunes = 64
	// generatedPasswordLength is the length of a password the server
	// generates for a new user or a reset (5.12).
	generatedPasswordLength = 16
)

// adminUserCreatedResponse is AdminUserCreated (5.5).
type adminUserCreatedResponse struct {
	User              model.AdminUser `json:"user"`
	TemporaryPassword *string         `json:"temporaryPassword"`
}

// temporaryPasswordResponse is the body of a password reset.
type temporaryPasswordResponse struct {
	TemporaryPassword *string `json:"temporaryPassword"`
}

// roleNames lists the roles as strings for enum checks.
func roleNames() []string {
	out := make([]string, len(model.Roles))
	for i, r := range model.Roles {
		out[i] = string(r)
	}
	return out
}

// adminListUsers is GET /api/admin/users?q=&role=&status=&limit=&offset=.
func (s *server) adminListUsers(w http.ResponseWriter, r *http.Request) {
	q := newQuery(r)
	filter := model.UserFilter{
		Query:  q.text("q", adminUserQueryMaxRunes),
		Role:   model.Role(q.enum("role", "", roleNames())),
		Status: q.enum("status", model.UserStatusAll, []string{model.UserStatusActive, model.UserStatusInactive, model.UserStatusAll}),
	}
	page := q.page(adminUsersDefaultLimit, adminUsersMaxLimit)
	if q.writeIfInvalid(w) {
		return
	}
	items, total, err := s.deps.Store.AdminUsers(r.Context(), filter, page)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, model.NewPage(items, total, page))
}

// adminTarget reads the {id} of an admin user route and loads that account;
// it answers 400 invalid_id or 404 not_found itself and then reports false.
func (s *server) adminTarget(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return model.User{}, false
	}
	return s.loadAdminTarget(w, r, id)
}

// loadAdminTarget loads the account id of an admin user route whose {id} is
// already parsed; it answers 404 not_found itself and then reports false.
// Handlers with a body parse the {id} first, then the body, and load the
// account last (the order of 5.1 and of every other {id} route).
func (s *server) loadAdminTarget(w http.ResponseWriter, r *http.Request, id int64) (model.User, bool) {
	u, err := s.deps.Store.UserByID(r.Context(), id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return model.User{}, false
	}
	return u, true
}

// writeAdminUser answers the AdminUser row of id with status.
func (s *server) writeAdminUser(w http.ResponseWriter, r *http.Request, id int64, status int) {
	au, err := s.deps.Store.AdminUser(r.Context(), id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, status, au)
}

// newPassword returns the hash of password, or of a generated password that
// is also returned (5.12: no password given means a generated one).
func (s *server) newPassword(password string) (hash string, generated *string, err error) {
	if password == "" {
		p, err := auth.GeneratePassword(generatedPasswordLength)
		if err != nil {
			return "", nil, err
		}
		password, generated = p, &p
	}
	hash, err = s.deps.Hasher.Hash(password)
	return hash, generated, err
}

// adminCreateUserRequest is NewAdminUser (5.5).
type adminCreateUserRequest struct {
	Username    string  `json:"username"`
	DisplayName *string `json:"displayName"`
	Role        string  `json:"role"`
	Password    *string `json:"password"`
	Locale      *string `json:"locale"`
}

// adminCreateUser is POST /api/admin/users: a new account that must change
// its password at the first sign-in.
func (s *server) adminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req adminCreateUserRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	username, code := checkUsername(req.Username)
	f.add("username", code)
	display := ""
	if req.DisplayName != nil {
		display = f.text("displayName", *req.DisplayName, 0, model.DisplayNameMaxLen, false)
	}
	role := model.Role(req.Role)
	if req.Role == "" {
		f.add("role", fieldRequired)
	} else if !role.Valid() {
		f.add("role", fieldInvalid)
	}
	locale := model.DefaultLocale
	if req.Locale != nil && *req.Locale != "" {
		locale = model.Locale(*req.Locale)
		if !locale.Valid() {
			f.add("locale", fieldInvalid)
		}
	}
	password := ""
	if req.Password != nil {
		password = *req.Password
	}
	if password != "" {
		f.add("password", auth.CheckPasswordPolicy(password, username, ""))
	}
	if f.writeIfAny(w) {
		return
	}
	hash, generated, err := s.newPassword(password)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	ctx := r.Context()
	u, err := s.deps.Store.CreateUser(ctx, model.NewUser{
		Username:           username,
		DisplayName:        display,
		Role:               role,
		PasswordHash:       hash,
		MustChangePassword: true,
		Locale:             locale,
	})
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.EventUserCreated, userFrom(r), userTarget(u), map[string]any{model.DetailRole: string(u.Role)})
	au, err := s.deps.Store.AdminUser(ctx, u.ID)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, adminUserCreatedResponse{User: au, TemporaryPassword: generated})
}

// adminGetUser is GET /api/admin/users/{id}.
func (s *server) adminGetUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	s.writeAdminUser(w, r, id, http.StatusOK)
}

// adminUpdateUserRequest is AdminUserPatch (5.5).
type adminUpdateUserRequest struct {
	DisplayName     *string `json:"displayName"`
	Role            *string `json:"role"`
	Active          *bool   `json:"active"`
	CurrentPassword *string `json:"currentPassword"`
}

// fromTo is the {from, to} value of an audited change (6.10).
type fromTo struct {
	From any `json:"from"`
	To   any `json:"to"`
}

// adminUpdateUser is PATCH /api/admin/users/{id}. Role and active changes
// need the admin's password (step-up), are refused on the admin's own account
// (self_modification) and for the last active admin (last_admin).
func (s *server) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req adminUpdateUserRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	var patch model.UserPatch
	if req.DisplayName != nil {
		name := f.text("displayName", *req.DisplayName, 0, model.DisplayNameMaxLen, false)
		patch.DisplayName = &name
	}
	if req.Role != nil {
		role := model.Role(*req.Role)
		if !role.Valid() {
			f.add("role", fieldInvalid)
		}
		patch.Role = &role
	}
	patch.Active = req.Active
	sensitive := req.Role != nil || req.Active != nil
	if sensitive && (req.CurrentPassword == nil || *req.CurrentPassword == "") {
		f.add("currentPassword", fieldRequired)
	}
	if f.writeIfAny(w) {
		return
	}
	ctx := r.Context()
	admin := userFrom(r)
	before, err := s.deps.Store.UserByID(ctx, id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	roleChanged := patch.Role != nil && *patch.Role != before.Role
	activeChanged := patch.Active != nil && *patch.Active != before.Active
	if before.ID == admin.ID && (roleChanged || activeChanged) {
		writeError(w, newError(codeSelfModification))
		return
	}
	if sensitive {
		if err := s.stepUp(ctx, admin, s.passwordStep(*req.CurrentPassword)); err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
	}
	after, err := s.deps.Store.UpdateUser(ctx, id, patch)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if activeChanged && !after.Active {
		if _, err := s.deps.Store.DeleteUserSessions(ctx, id, 0); err != nil {
			writeFailure(w, r, s.log, err)
			return
		}
	}
	s.auditUserUpdate(r, admin, before, after)
	s.writeAdminUser(w, r, id, http.StatusOK)
}

// auditUserUpdate audits the changes between before and after (6.10):
// user.updated lists the changed fields and carries values only for the
// non-personal role; activation changes are user.activated and
// user.deactivated (5.12), so user.updated never repeats them as an
// active {from, to} value. The active detail key stays in the closed key set
// of 6.10 (model.AuditDetailKeys) without being emitted here.
func (s *server) auditUserUpdate(r *http.Request, admin, before, after model.User) {
	var changed []string
	details := map[string]any{}
	if after.DisplayName != before.DisplayName {
		changed = append(changed, "displayName")
	}
	if after.Role != before.Role {
		changed = append(changed, "role")
		details[model.DetailRole] = fromTo{From: string(before.Role), To: string(after.Role)}
	}
	if len(changed) > 0 {
		details[model.DetailChanged] = changed
		s.auditUser(r, model.EventUserUpdated, admin, userTarget(after), details)
	}
	if after.Active != before.Active {
		event := model.EventUserActivated
		if !after.Active {
			event = model.EventUserDeactivated
		}
		s.auditUser(r, event, admin, userTarget(after), nil)
	}
}

// stepUpRequest is AdminStepUp (5.5).
type stepUpRequest struct {
	CurrentPassword string `json:"currentPassword"`
}

// validate requires the admin's password.
func (req stepUpRequest) validate() fieldErrors {
	f := fieldErrors{}
	f.add("currentPassword", checkPasswordInput(req.CurrentPassword))
	return f
}

// adminDeleteUser is DELETE /api/admin/users/{id} with an AdminStepUp body
// (6.11). The admin's own account is self_modification.
func (s *server) adminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req stepUpRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.validate().writeIfAny(w) {
		return
	}
	target, ok := s.loadAdminTarget(w, r, id)
	if !ok {
		return
	}
	admin := userFrom(r)
	if target.ID == admin.ID {
		writeError(w, newError(codeSelfModification))
		return
	}
	ctx := r.Context()
	if err := s.stepUp(ctx, admin, s.passwordStep(req.CurrentPassword)); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	entry := model.AuditEntry{
		Event:   model.EventUserDeleted,
		Success: true,
		Actor:   auditActor(admin),
		Target:  userTarget(target),
		IP:      truncateBytes(s.clientIP(r), model.SessionIPMaxLen),
	}
	if err := s.deps.Store.DeleteUser(ctx, target.ID, entry); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeNoContent(w)
}

// adminResetPasswordRequest is the body of the password reset.
type adminResetPasswordRequest struct {
	CurrentPassword string  `json:"currentPassword"`
	Password        *string `json:"password"`
}

// adminResetPassword is POST /api/admin/users/{id}/reset-password: a new
// (given or generated) password that must be changed at the next sign-in;
// every session and throttle row of the user is removed.
func (s *server) adminResetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req adminResetPasswordRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := stepUpRequest{CurrentPassword: req.CurrentPassword}.validate()
	if f.writeIfAny(w) {
		return
	}
	target, ok := s.loadAdminTarget(w, r, id)
	if !ok {
		return
	}
	password := ""
	if req.Password != nil {
		password = *req.Password
	}
	if password != "" {
		if code := auth.CheckPasswordPolicy(password, target.Username, ""); code != "" {
			writeValidation(w, map[string]string{"password": code})
			return
		}
	}
	ctx := r.Context()
	admin := userFrom(r)
	if err := s.stepUp(ctx, admin, s.passwordStep(req.CurrentPassword)); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	hash, generated, err := s.newPassword(password)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.deps.Store.SetPassword(ctx, target.ID, hash, true); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if _, err := s.deps.Store.DeleteUserSessions(ctx, target.ID, 0); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.deps.Store.ClearUserThrottle(ctx, target.ID); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.EventUserPasswordReset, admin, userTarget(target), nil)
	writeJSON(w, http.StatusOK, temporaryPasswordResponse{TemporaryPassword: generated})
}

// adminResetTwoFactor is POST /api/admin/users/{id}/reset-2fa: 2FA off and
// every session of the user deleted.
func (s *server) adminResetTwoFactor(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req stepUpRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.validate().writeIfAny(w) {
		return
	}
	target, ok := s.loadAdminTarget(w, r, id)
	if !ok {
		return
	}
	if !target.TOTPEnabled {
		writeError(w, newError(codeTOTPNotEnabled))
		return
	}
	ctx := r.Context()
	admin := userFrom(r)
	if err := s.stepUp(ctx, admin, s.passwordStep(req.CurrentPassword)); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.deps.Store.DisableTOTP(ctx, target.ID); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if _, err := s.deps.Store.DeleteUserSessions(ctx, target.ID, 0); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.EventUser2FAReset, admin, userTarget(target), nil)
	writeNoContent(w)
}

// adminUnlockUser is POST /api/admin/users/{id}/unlock: every throttle row
// of the user is removed (6.7).
func (s *server) adminUnlockUser(w http.ResponseWriter, r *http.Request) {
	target, ok := s.adminTarget(w, r)
	if !ok {
		return
	}
	if err := s.deps.Store.ClearUserThrottle(r.Context(), target.ID); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.EventUserUnlocked, userFrom(r), userTarget(target), nil)
	writeNoContent(w)
}

// adminRevokeSessions is DELETE /api/admin/users/{id}/sessions. On the
// admin's own account the current session is kept.
func (s *server) adminRevokeSessions(w http.ResponseWriter, r *http.Request) {
	target, ok := s.adminTarget(w, r)
	if !ok {
		return
	}
	admin := userFrom(r)
	var keep int64
	if target.ID == admin.ID {
		keep = sessionFrom(r).ID
	}
	n, err := s.deps.Store.DeleteUserSessions(r.Context(), target.ID, keep)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.EventUserSessionsRevoked, admin, userTarget(target), map[string]any{model.DetailCount: n})
	writeJSON(w, http.StatusOK, revokedResponse{Revoked: n})
}
