// ApiError and field codes to localized text (plan 8.5.5, 8.5.6).
import type { ApiError } from '$lib/api/errors.js';
import { hasMessage, t, tDynamic } from './runtime.svelte.js';
import type { Params } from './types.js';

/** The catalogue code for an error: unknown server codes fall back to internal_error. */
export function knownErrorCode(err: Pick<ApiError, 'code'>): string {
	return hasMessage(`errors.${err.code}.title`) ? err.code : 'internal_error';
}

/** Placeholder values carried by the error details (minutes to wait, locked setting keys). */
export function errorParams(err: Pick<ApiError, 'code' | 'details'>): Params {
	const params: Params = {};
	if (err.code === 'too_many_attempts') {
		const seconds = Number(err.details.retryAfterSeconds);
		params.minutes = Number.isFinite(seconds) ? Math.max(1, Math.ceil(seconds / 60)) : 1;
	}
	if (err.code === 'setting_locked') {
		const keys = err.details.keys;
		params.keys = Array.isArray(keys) ? keys.map(String).join(', ') : '';
	}
	return params;
}

/** What happened: errors.<code>.title */
export function errorTitle(err: Pick<ApiError, 'code' | 'details'>): string {
	return tDynamic(`errors.${knownErrorCode(err)}.title`, errorParams(err));
}

/** What to do: errors.<code>.text */
export function errorText(err: Pick<ApiError, 'code' | 'details'>): string {
	return tDynamic(`errors.${knownErrorCode(err)}.text`, errorParams(err));
}

/** Title and text as one sentence pair, for toasts and inline messages. */
export function errorMessage(err: Pick<ApiError, 'code' | 'details'>): string {
	return `${errorTitle(err)} ${errorText(err)}`;
}

/** Message for a field code: the per-field override first, then the generic text. */
export function fieldMessage(field: string, code: string, params?: Params): string {
	if (code === 'out_of_range' && params?.min !== undefined && params.max !== undefined) {
		return t('fields.range', params);
	}
	const specific = `fields.byField.${field}.${code}`;
	if (hasMessage(specific)) return tDynamic(specific, params);
	return tDynamic(`fields.${code}`, params, t('fields.invalid'));
}
