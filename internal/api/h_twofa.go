package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Two-factor setup, enable, disable and recovery codes (5.8, 6.5, 6.6).

// pendingTOTPTTL is the lifetime of a setup secret awaiting confirmation (5.8).
const pendingTOTPTTL = 10 * time.Minute

// twoFactorSetupResponse is TwoFactorSetup (5.5).
type twoFactorSetupResponse struct {
	Secret     string    `json:"secret"`
	OTPAuthURI string    `json:"otpauthUri"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// recoveryCodesResponse is RecoveryCodes (5.5).
type recoveryCodesResponse struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

// passwordRequest is a body with the caller's password.
type passwordRequest struct {
	Password string `json:"password"`
}

// codeRequest is a body with a TOTP code.
type codeRequest struct {
	Code string `json:"code"`
}

// passwordCodeRequest is a body with the caller's password and a code.
type passwordCodeRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

// validate checks that both values are present and of sane length.
func (req passwordCodeRequest) validate() fieldErrors {
	f := fieldErrors{}
	f.add("password", checkPasswordInput(req.Password))
	f.add("code", checkCodeInput(req.Code))
	return f
}

// setupTwoFactor is POST /api/me/2fa/setup: after the password step-up a new
// secret is sealed into the pending slot for pendingTOTPTTL. The secret and
// the URI appear only in this response.
func (s *server) setupTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req passwordRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	f.add("password", checkPasswordInput(req.Password))
	if f.writeIfAny(w) {
		return
	}
	ctx := r.Context()
	u := userFrom(r)
	if u.TOTPEnabled {
		writeError(w, newError(codeTOTPAlreadyEnabled))
		return
	}
	if err := s.stepUp(ctx, u, s.passwordStep(req.Password)); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	sealed, err := s.deps.Keyring.Seal([]byte(secret), totpAssociatedData(u.ID))
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	expires := s.now().Add(pendingTOTPTTL)
	if err := s.deps.Store.SetPendingTOTP(ctx, u.ID, sealed, expires); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, twoFactorSetupResponse{
		Secret:     secret,
		OTPAuthURI: auth.TOTPURI(totpIssuer, u.Username, secret),
		ExpiresAt:  expires,
	})
}

// newRecoveryCodes generates a fresh set of codes and their stored hashes.
func (s *server) newRecoveryCodes() ([]string, []string, error) {
	codes, err := auth.NewRecoveryCodes(recoveryCodeCount)
	if err != nil {
		return nil, nil, err
	}
	return codes, s.recoveryCodeHashes(codes), nil
}

// enableTwoFactor is POST /api/me/2fa/enable: the code is checked against
// the pending secret; on success 2FA is on, ten recovery codes are returned
// once and the other sessions are deleted.
func (s *server) enableTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req codeRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := fieldErrors{}
	if req.Code == "" {
		f.add("code", fieldRequired)
	} else if !isTOTPCode(req.Code) {
		f.add("code", fieldInvalidFormat)
	}
	if f.writeIfAny(w) {
		return
	}
	ctx := r.Context()
	u := userFrom(r)
	if u.TOTPEnabled {
		writeError(w, newError(codeTOTPAlreadyEnabled))
		return
	}
	sealed, _, err := s.deps.Store.PendingTOTP(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, newError(codeTOTPSetupRequired))
		return
	}
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	secret, err := s.deps.Keyring.Open(sealed, totpAssociatedData(u.ID))
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	step, ok := auth.VerifyTOTP(string(secret), req.Code, s.now(), 0)
	if !ok {
		writeError(w, newError(codeInvalidTOTP))
		return
	}
	codes, hashes, err := s.newRecoveryCodes()
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	err = s.deps.Store.EnableTOTP(ctx, u.ID, sealed, step, hashes)
	if errors.Is(err, model.ErrConflict) {
		writeError(w, newError(codeTOTPAlreadyEnabled))
		return
	}
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if _, err := s.deps.Store.DeleteUserSessions(ctx, u.ID, sessionFrom(r).ID); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.Event2FAEnabled, u, nil, nil)
	writeJSON(w, http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}

// secondFactorStep is the step-up check of password plus a second factor.
// A wrong password is 400 current_password_invalid, a wrong code 401
// invalid_totp; both count as a failure on the user's throttle key.
// allowRecovery also accepts a recovery code.
func (s *server) secondFactorStep(r *http.Request, password, code string, allowRecovery bool) stepUpCheck {
	return func(c model.Credentials) (*apiError, error) {
		if ok, _ := s.passwordMatches(c, password); !ok {
			return currentPasswordError(), nil
		}
		var ok bool
		var err error
		if allowRecovery {
			_, ok, err = s.verifySecondFactor(r.Context(), c, code)
		} else {
			ok, err = s.verifyTOTP(r.Context(), c, code)
		}
		if err != nil {
			return nil, err
		}
		if !ok {
			return newError(codeInvalidTOTP), nil
		}
		return nil, nil
	}
}

// disableTwoFactor is POST /api/me/2fa/disable: password and a TOTP or
// recovery code; the secrets and codes are removed and the other sessions
// deleted.
func (s *server) disableTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req passwordCodeRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.validate().writeIfAny(w) {
		return
	}
	ctx := r.Context()
	u := userFrom(r)
	if !u.TOTPEnabled {
		writeError(w, newError(codeTOTPNotEnabled))
		return
	}
	if err := s.stepUp(ctx, u, s.secondFactorStep(r, req.Password, req.Code, true)); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.deps.Store.DisableTOTP(ctx, u.ID); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if _, err := s.deps.Store.DeleteUserSessions(ctx, u.ID, sessionFrom(r).ID); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.Event2FADisabled, u, nil, nil)
	writeNoContent(w)
}

// regenerateRecoveryCodes is POST /api/me/2fa/recovery-codes: password and a
// TOTP code (not a recovery code); the old codes are replaced.
func (s *server) regenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	var req passwordCodeRequest
	if !readJSON(w, r, &req) {
		return
	}
	f := req.validate()
	if !isTOTPCode(req.Code) {
		f.add("code", fieldInvalidFormat) // after required/too_long: the first code wins
	}
	if f.writeIfAny(w) {
		return
	}
	ctx := r.Context()
	u := userFrom(r)
	if !u.TOTPEnabled {
		writeError(w, newError(codeTOTPNotEnabled))
		return
	}
	if err := s.stepUp(ctx, u, s.secondFactorStep(r, req.Password, req.Code, false)); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	codes, hashes, err := s.newRecoveryCodes()
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	if err := s.deps.Store.ReplaceRecoveryCodes(ctx, u.ID, hashes); err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	s.auditUser(r, model.Event2FARecoveryRegenerated, u, nil, nil)
	writeJSON(w, http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}
