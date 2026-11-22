// Form model of the system settings (plan 8.7.8; ranges from the SystemSettings notes of 5.5).
import type { SystemSettingsPatch } from '$lib/api/admin.js';
import type { FieldCode } from '$lib/api/errors.js';
import type { SystemSettings } from '$lib/api/types.js';
import type { Params } from '$lib/i18n/index.js';

export type NumberKey =
	'defaultArchiveDays' | 'sessionIdleMinutes' | 'sessionAbsoluteHours' | 'auditRetentionDays';
export type TextKey =
	| 'privacyControllerName'
	| 'privacyControllerAddress'
	| 'privacyControllerEmail'
	| 'privacyDpoContact'
	| 'privacySupervisoryAuthority';
export type EditableKey = NumberKey | TextKey | 'allowPrivateFetch';

export type NumberSpec = { min: number; max: number; unit: 'days' | 'minutes' | 'hours' };

export const NUMBER_SPECS: Record<NumberKey, NumberSpec> = {
	defaultArchiveDays: { min: 0, max: 3650, unit: 'days' },
	sessionIdleMinutes: { min: 15, max: 10080, unit: 'minutes' },
	sessionAbsoluteHours: { min: 1, max: 2160, unit: 'hours' },
	auditRetentionDays: { min: 30, max: 3650, unit: 'days' }
};

/** Maximum length in Unicode code points (the server counts runes). */
export const TEXT_LIMITS: Record<TextKey, number> = {
	privacyControllerName: 200,
	privacyControllerAddress: 300,
	privacyControllerEmail: 254,
	privacyDpoContact: 300,
	privacySupervisoryAuthority: 300
};

export const NUMBER_KEYS = Object.keys(NUMBER_SPECS) as NumberKey[];
export const TEXT_KEYS = Object.keys(TEXT_LIMITS) as TextKey[];
export const EDITABLE_KEYS: readonly EditableKey[] = [
	...NUMBER_KEYS,
	'allowPrivateFetch',
	...TEXT_KEYS
];

/** The editable form state: numbers as typed text, so a half-typed value is not lost. */
export type SettingsDraft = Record<NumberKey, string> &
	Record<TextKey, string> & { allowPrivateFetch: boolean };

export type FieldIssue = { code: FieldCode; params?: Params };

export function toDraft(settings: SystemSettings): SettingsDraft {
	const draft = { allowPrivateFetch: settings.allowPrivateFetch } as SettingsDraft;
	for (const key of NUMBER_KEYS) draft[key] = String(settings[key]);
	for (const key of TEXT_KEYS) draft[key] = settings[key];
	return draft;
}

const runeCount = (value: string): number => [...value].length;
const EMAIL_SHAPE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Parses an integer field; null when the text is not a whole number. */
export function parseWhole(value: string): number | null {
	const trimmed = value.trim();
	return /^-?\d{1,9}$/.test(trimmed) ? Number(trimmed) : null;
}

/** Client-side checks with the server's field codes; the server stays authoritative. */
export function validateDraft(draft: SettingsDraft): Partial<Record<EditableKey, FieldIssue>> {
	const issues: Partial<Record<EditableKey, FieldIssue>> = {};
	for (const key of NUMBER_KEYS) {
		const { min, max } = NUMBER_SPECS[key];
		if (draft[key].trim() === '') {
			issues[key] = { code: 'required' };
			continue;
		}
		const value = parseWhole(draft[key]);
		if (value === null) issues[key] = { code: 'invalid' };
		else if (value < min || value > max)
			issues[key] = { code: 'out_of_range', params: { min, max } };
	}
	for (const key of TEXT_KEYS) {
		const value = draft[key].trim();
		if (runeCount(value) > TEXT_LIMITS[key]) issues[key] = { code: 'too_long' };
		else if (key === 'privacyControllerEmail' && value !== '' && !EMAIL_SHAPE.test(value))
			issues[key] = { code: 'invalid_format' };
	}
	return issues;
}

/** The changed fields of the draft against the saved settings (numbers parsed, text trimmed). */
export function draftPatch(draft: SettingsDraft, saved: SystemSettings): SystemSettingsPatch {
	const patch: SystemSettingsPatch = {};
	for (const key of NUMBER_KEYS) {
		const value = parseWhole(draft[key]);
		if (value !== saved[key]) patch[key] = value ?? Number.NaN;
	}
	if (draft.allowPrivateFetch !== saved.allowPrivateFetch)
		patch.allowPrivateFetch = draft.allowPrivateFetch;
	for (const key of TEXT_KEYS) {
		const value = draft[key].trim();
		if (value !== saved[key]) patch[key] = value;
	}
	return patch;
}

/** True when the draft differs from the saved settings. */
export function isDirty(draft: SettingsDraft, saved: SystemSettings): boolean {
	return Object.keys(draftPatch(draft, saved)).length > 0;
}

/** Range parameters for an out_of_range code reported by the server on a number field. */
export function rangeParams(key: string): Params | undefined {
	const spec = NUMBER_SPECS[key as NumberKey];
	return spec ? { min: spec.min, max: spec.max } : undefined;
}
