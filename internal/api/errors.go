package api

import (
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Stable error codes (5.3). The interface translates them; the message is
// for logs and developers only.
const (
	codeEndpointNotFound       = "endpoint_not_found"
	codeInvalidJSON            = "invalid_json"
	codeInvalidID              = "invalid_id"
	codeInvalidQuery           = "invalid_query"
	codeValidationFailed       = "validation_failed"
	codePayloadTooLarge        = "payload_too_large"
	codeNotFound               = "not_found"
	codeUnauthenticated        = "unauthenticated"
	codeForbidden              = "forbidden"
	codeCSRFRejected           = "csrf_rejected"
	codeHostMismatch           = "host_mismatch"
	codePasswordChangeRequired = "password_change_required"
	codeConsentRequired        = "consent_required"
	codeInvalidCredentials     = "invalid_credentials"
	codeAccountDisabled        = "account_disabled"
	codeTooManyAttempts        = "too_many_attempts"
	codeChallengeInvalid       = "challenge_invalid"
	codeInvalidTOTP            = "invalid_totp"
	codeCurrentPasswordInvalid = "current_password_invalid"
	codeTOTPAlreadyEnabled     = "totp_already_enabled"
	codeTOTPNotEnabled         = "totp_not_enabled"
	codeTOTPSetupRequired      = "totp_setup_required"
	codeLinkDuplicate          = "link_duplicate"
	codeLinkLimitReached       = "link_limit_reached"
	codeNotOwner               = "not_owner"
	codeShareNotAllowed        = "share_not_allowed"
	codeShareDuplicate         = "share_duplicate"
	codeNoURLFound             = "no_url_found"
	codeImportInvalid          = "import_invalid"
	codeImportTooManyEntries   = "import_too_many_entries"
	codeTagDuplicate           = "tag_duplicate"
	codeTagLimitReached        = "tag_limit_reached"
	codeUsernameTaken          = "username_taken"
	codeLastAdmin              = "last_admin"
	codeSelfModification       = "self_modification"
	codeSettingLocked          = "setting_locked"
	codeConsentVersionMismatch = "consent_version_mismatch"
	codeConfirmationMismatch   = "confirmation_mismatch"
	codeNotImplemented         = "not_implemented"
	codeInternalError          = "internal_error"
)

// errorSpec binds a code to its HTTP status and log message.
type errorSpec struct {
	code    string
	status  int
	message string
}

// errorSpecs is the 5.3 table in order. The contract golden (9.6) lists the
// codes from here; the client-only network_error is not part of it.
var errorSpecs = []errorSpec{
	{codeEndpointNotFound, http.StatusNotFound, "endpoint not found"},
	{codeInvalidJSON, http.StatusBadRequest, "invalid JSON"},
	{codeInvalidID, http.StatusBadRequest, "invalid id"},
	{codeInvalidQuery, http.StatusBadRequest, "invalid query"},
	{codeValidationFailed, http.StatusBadRequest, "validation failed"},
	{codePayloadTooLarge, http.StatusRequestEntityTooLarge, "payload too large"},
	{codeNotFound, http.StatusNotFound, "not found"},
	{codeUnauthenticated, http.StatusUnauthorized, "unauthenticated"},
	{codeForbidden, http.StatusForbidden, "forbidden"},
	{codeCSRFRejected, http.StatusForbidden, "csrf rejected"},
	{codeHostMismatch, http.StatusMisdirectedRequest, "host mismatch"},
	{codePasswordChangeRequired, http.StatusForbidden, "password change required"},
	{codeConsentRequired, http.StatusForbidden, "consent required"},
	{codeInvalidCredentials, http.StatusUnauthorized, "invalid credentials"},
	{codeAccountDisabled, http.StatusForbidden, "account disabled"},
	{codeTooManyAttempts, http.StatusTooManyRequests, "too many attempts"},
	{codeChallengeInvalid, http.StatusUnauthorized, "challenge invalid"},
	{codeInvalidTOTP, http.StatusUnauthorized, "invalid totp code"},
	{codeCurrentPasswordInvalid, http.StatusBadRequest, "current password invalid"},
	{codeTOTPAlreadyEnabled, http.StatusConflict, "totp already enabled"},
	{codeTOTPNotEnabled, http.StatusConflict, "totp not enabled"},
	{codeTOTPSetupRequired, http.StatusConflict, "totp setup required"},
	{codeLinkDuplicate, http.StatusConflict, "link already exists"},
	{codeLinkLimitReached, http.StatusConflict, "link limit reached"},
	{codeNotOwner, http.StatusForbidden, "not owner"},
	{codeShareNotAllowed, http.StatusForbidden, "sharing not allowed"},
	{codeShareDuplicate, http.StatusConflict, "share already exists"},
	{codeNoURLFound, http.StatusBadRequest, "no url found"},
	{codeImportInvalid, http.StatusBadRequest, "import file invalid"},
	{codeImportTooManyEntries, http.StatusBadRequest, "import file has too many entries"},
	{codeTagDuplicate, http.StatusConflict, "tag already exists"},
	{codeTagLimitReached, http.StatusConflict, "tag limit reached"},
	{codeUsernameTaken, http.StatusConflict, "username taken"},
	{codeLastAdmin, http.StatusConflict, "last active admin"},
	{codeSelfModification, http.StatusConflict, "self modification not allowed"},
	{codeSettingLocked, http.StatusConflict, "setting locked by environment"},
	{codeConsentVersionMismatch, http.StatusConflict, "consent version mismatch"},
	{codeConfirmationMismatch, http.StatusBadRequest, "confirmation mismatch"},
	{codeNotImplemented, http.StatusNotImplemented, "not implemented"},
	{codeInternalError, http.StatusInternalServerError, "internal error"},
}

var errorSpecByCode = func() map[string]errorSpec {
	m := make(map[string]errorSpec, len(errorSpecs))
	for _, s := range errorSpecs {
		m[s.code] = s
	}
	return m
}()

// errorCodes returns every server error code in table order.
func errorCodes() []string {
	codes := make([]string, len(errorSpecs))
	for i, s := range errorSpecs {
		codes[i] = s.code
	}
	return codes
}

// Field codes: the values of the "fields" object (5.3).
const (
	fieldRequired                 = "required"
	fieldTooShort                 = "too_short"
	fieldTooLong                  = "too_long"
	fieldInvalid                  = "invalid"
	fieldInvalidFormat            = "invalid_format"
	fieldNotAllowed               = "not_allowed"
	fieldOutOfRange               = "out_of_range"
	fieldUnknown                  = "unknown"
	fieldDuplicate                = "duplicate"
	fieldCredentialsInURL         = "credentials_in_url"
	fieldIncorrect                = "incorrect"
	fieldPasswordTooShort         = "password_too_short"
	fieldPasswordTooLong          = "password_too_long"
	fieldPasswordContainsUsername = "password_contains_username"
	fieldPasswordTooSimple        = "password_too_simple"
	fieldPasswordReused           = "password_reused"
)

// fieldCodes lists every field code in the order of 5.3.
var fieldCodes = []string{
	fieldRequired, fieldTooShort, fieldTooLong, fieldInvalid, fieldInvalidFormat,
	fieldNotAllowed, fieldOutOfRange, fieldUnknown, fieldDuplicate, fieldCredentialsInURL,
	fieldIncorrect, fieldPasswordTooShort, fieldPasswordTooLong, fieldPasswordContainsUsername,
	fieldPasswordTooSimple, fieldPasswordReused,
}

// Detail keys of the "details" object (5.2).
const (
	detailLinkID            = "linkId"
	detailTagID             = "tagId"
	detailRetryAfterSeconds = "retryAfterSeconds"
	detailKeys              = "keys"
)

// apiError is an error response: status, stable code, English message and
// the optional fields and details objects (5.2). It implements error, so
// validators and handlers can return it through ordinary error paths.
type apiError struct {
	Status     int
	Code       string
	Message    string
	Fields     map[string]string
	Details    map[string]any
	RetryAfter int // seconds; sets the Retry-After header when > 0
}

// Error implements the error interface.
func (e *apiError) Error() string { return e.Code + ": " + e.Message }

// newError returns the error of a code from the 5.3 table. An unknown code
// is a programming error and yields internal_error.
func newError(code string) *apiError {
	s, ok := errorSpecByCode[code]
	if !ok {
		s = errorSpecByCode[codeInternalError]
	}
	return &apiError{Status: s.status, Code: s.code, Message: s.message}
}

// withFields returns a copy of e with the given field codes.
func (e *apiError) withFields(fields map[string]string) *apiError {
	c := *e
	c.Fields = fields
	return &c
}

// withDetail returns a copy of e with one more detail entry.
func (e *apiError) withDetail(key string, value any) *apiError {
	c := *e
	c.Details = make(map[string]any, len(e.Details)+1)
	for k, v := range e.Details {
		c.Details[k] = v
	}
	c.Details[key] = value
	return &c
}

// validationError is 400 validation_failed with field codes.
func validationError(fields map[string]string) *apiError {
	return newError(codeValidationFailed).withFields(fields)
}

// invalidQueryError is 400 invalid_query; fields maps parameter names to "invalid".
func invalidQueryError(fields map[string]string) *apiError {
	return newError(codeInvalidQuery).withFields(fields)
}

// currentPasswordError is 400 current_password_invalid with
// fields.currentPassword = "incorrect" (5.2).
func currentPasswordError() *apiError {
	return newError(codeCurrentPasswordInvalid).withFields(map[string]string{"currentPassword": fieldIncorrect})
}

// tooManyAttempts is 429 too_many_attempts with details.retryAfterSeconds
// and the Retry-After header, rounded up to whole seconds (at least 1).
func tooManyAttempts(retryAfter time.Duration) *apiError {
	secs := int(math.Ceil(retryAfter.Seconds()))
	if secs < 1 {
		secs = 1
	}
	e := newError(codeTooManyAttempts).withDetail(detailRetryAfterSeconds, secs)
	e.RetryAfter = secs
	return e
}

// settingLocked is 409 setting_locked with details.keys.
func settingLocked(keys []string) *apiError {
	return newError(codeSettingLocked).withDetail(detailKeys, keys)
}

// Limit kinds of limitReached.
const (
	limitLinks = "link"
	limitTags  = "tag"
)

// limitReached maps model.ErrLimitReached for a resource kind (limitLinks or
// limitTags); the store error alone does not tell which limit was hit.
func limitReached(kind string) *apiError {
	if kind == limitTags {
		return newError(codeTagLimitReached)
	}
	return newError(codeLinkLimitReached)
}

// mapModelError maps the generic model errors to API errors. It reports
// false for errors that need handler context (model.ErrLimitReached,
// model.ErrConflict) or are unexpected; those end as internal_error unless
// the handler maps them first.
func mapModelError(err error) (*apiError, bool) {
	var dup *model.DuplicateError
	switch {
	case err == nil:
		return nil, false
	case errors.Is(err, model.ErrNotFound):
		return newError(codeNotFound), true
	case errors.Is(err, model.ErrNotOwner):
		return newError(codeNotOwner), true
	case errors.Is(err, model.ErrLastAdmin):
		return newError(codeLastAdmin), true
	case errors.As(err, &dup):
		switch dup.Kind {
		case model.DuplicateLink:
			return newError(codeLinkDuplicate).withDetail(detailLinkID, dup.ExistingID), true
		case model.DuplicateTag:
			return newError(codeTagDuplicate).withDetail(detailTagID, dup.ExistingID), true
		case model.DuplicateShare:
			return newError(codeShareDuplicate), true
		case model.DuplicateUsername:
			return newError(codeUsernameTaken), true
		}
	}
	return nil, false
}
