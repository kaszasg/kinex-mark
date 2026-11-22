// Maps API errors of the security and privacy forms to messages under their fields (plan 8.7.6).
import { tick } from 'svelte';
import { ApiError, type ErrorCode } from '$lib/api/errors.js';
import { errorMessage, fieldMessage } from '$lib/i18n/index.js';

/** Messages shown under named fields, plus the error left for an ErrorAlert (null: none). */
export type FormErrors = { fields: Record<string, string>; alert: unknown };

export const NO_ERRORS: FormErrors = { fields: {}, alert: null };

/**
 * Splits an error: codes listed in `byCode` go under that field as one sentence (title + text),
 * validation_failed field codes go under the known fields, everything else to the alert.
 */
export function formErrors(
	err: unknown,
	byCode: Partial<Record<ErrorCode, string>>,
	fieldNames: readonly string[]
): FormErrors {
	if (!(err instanceof ApiError)) return { fields: {}, alert: err };
	const target = byCode[err.code];
	if (target) return { fields: { [target]: errorMessage(err) }, alert: null };
	if (err.code !== 'validation_failed') return { fields: {}, alert: err };
	const fields: Record<string, string> = {};
	let unmatched = Object.keys(err.fields).length === 0;
	for (const [name, code] of Object.entries(err.fields)) {
		if (fieldNames.includes(name)) fields[name] = fieldMessage(name, code);
		else unmatched = true;
	}
	return { fields, alert: unmatched ? err : null };
}

/** Moves the focus to the first invalid control inside `root` once the DOM has updated. */
export async function focusFirstInvalid(root: HTMLElement | null | undefined): Promise<void> {
	await tick();
	root?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
}
