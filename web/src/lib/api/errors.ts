// Stable error codes of the API contract (plan 5.3) and the client-side ApiError.

/** Every error code of plan 5.3, plus the client-only network_error. */
export type ErrorCode =
	| 'endpoint_not_found'
	| 'invalid_json'
	| 'invalid_id'
	| 'invalid_query'
	| 'validation_failed'
	| 'payload_too_large'
	| 'not_found'
	| 'unauthenticated'
	| 'forbidden'
	| 'csrf_rejected'
	| 'host_mismatch'
	| 'password_change_required'
	| 'consent_required'
	| 'invalid_credentials'
	| 'account_disabled'
	| 'too_many_attempts'
	| 'challenge_invalid'
	| 'invalid_totp'
	| 'current_password_invalid'
	| 'totp_already_enabled'
	| 'totp_not_enabled'
	| 'totp_setup_required'
	| 'link_duplicate'
	| 'link_limit_reached'
	| 'not_owner'
	| 'share_not_allowed'
	| 'share_duplicate'
	| 'no_url_found'
	| 'import_invalid'
	| 'import_too_many_entries'
	| 'tag_duplicate'
	| 'tag_limit_reached'
	| 'username_taken'
	| 'last_admin'
	| 'self_modification'
	| 'setting_locked'
	| 'consent_version_mismatch'
	| 'confirmation_mismatch'
	| 'not_implemented'
	| 'internal_error'
	| 'network_error';

/** Every field code of plan 5.3 (the values of an error body's `fields`). */
export type FieldCode =
	| 'required'
	| 'too_short'
	| 'too_long'
	| 'invalid'
	| 'invalid_format'
	| 'not_allowed'
	| 'out_of_range'
	| 'unknown'
	| 'duplicate'
	| 'credentials_in_url'
	| 'incorrect'
	| 'password_too_short'
	| 'password_too_long'
	| 'password_contains_username'
	| 'password_too_simple'
	| 'password_reused';

/** The codes that make the client hand control to the registered auth-failure handler. */
export type AuthFailureCode = 'unauthenticated' | 'password_change_required' | 'consent_required';

/** An API failure: HTTP status, stable code, field codes and machine details (plan 5.2). */
export class ApiError extends Error {
	/**
	 * Set by the API client when the registered auth-failure handler took the error over (sign-in
	 * page or gate page): the user sees that page's message, so callers raise no toast for it.
	 */
	authHandled = false;

	constructor(
		readonly status: number,
		readonly code: ErrorCode,
		readonly fields: Record<string, FieldCode> = {},
		readonly details: Record<string, unknown> = {}
	) {
		super(code);
		this.name = 'ApiError';
	}

	/** The field code reported for one request field, if any. */
	field(name: string): FieldCode | undefined {
		return this.fields[name];
	}

	/** Reports whether the error carries one of the given codes. */
	is(...codes: ErrorCode[]): boolean {
		return codes.includes(this.code);
	}
}

/** The code assumed when an error body carries none (plan 5.3 client fallback). */
export function codeFromStatus(status: number): ErrorCode {
	switch (status) {
		case 401:
			return 'unauthenticated';
		case 403:
			return 'forbidden';
		case 404:
			return 'not_found';
		case 413:
			return 'payload_too_large';
		case 429:
			return 'too_many_attempts';
		default:
			return 'internal_error';
	}
}

/** Narrows an unknown value (a caught exception) to ApiError. */
export function isApiError(value: unknown): value is ApiError {
	return value instanceof ApiError;
}
