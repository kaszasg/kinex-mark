// Form model of the system settings (plan 8.7.8).
import { describe, expect, test } from 'vitest';
import {
	draftPatch,
	isDirty,
	parseWhole,
	rangeParams,
	toDraft,
	validateDraft
} from './settings-model.js';
import { makeSettings } from './test-helpers.js';

describe('settings model', () => {
	test('a fresh draft is clean and mirrors the settings', () => {
		const saved = makeSettings();
		const draft = toDraft(saved);
		expect(draft.defaultArchiveDays).toBe('90');
		expect(draft.privacyControllerName).toBe('Kinexium');
		expect(draft.allowPrivateFetch).toBe(false);
		expect(isDirty(draft, saved)).toBe(false);
		expect(validateDraft(draft)).toEqual({});
	});

	test('the patch holds only changed fields, numbers parsed and text trimmed', () => {
		const saved = makeSettings();
		const draft = toDraft(saved);
		draft.defaultArchiveDays = ' 30 ';
		draft.allowPrivateFetch = true;
		draft.privacyDpoContact = '  dpo@example.com  ';
		draft.privacyControllerName = 'Kinexium  ';
		expect(draftPatch(draft, saved)).toEqual({
			defaultArchiveDays: 30,
			allowPrivateFetch: true,
			privacyDpoContact: 'dpo@example.com'
		});
		expect(isDirty(draft, saved)).toBe(true);
	});

	test('numbers: required, whole, in range', () => {
		const draft = toDraft(makeSettings());
		draft.defaultArchiveDays = '';
		draft.sessionIdleMinutes = '14';
		draft.sessionAbsoluteHours = '2.5';
		draft.auditRetentionDays = '3651';
		expect(validateDraft(draft)).toEqual({
			defaultArchiveDays: { code: 'required' },
			sessionIdleMinutes: { code: 'out_of_range', params: { min: 15, max: 10080 } },
			sessionAbsoluteHours: { code: 'invalid' },
			auditRetentionDays: { code: 'out_of_range', params: { min: 30, max: 3650 } }
		});
		draft.defaultArchiveDays = '0';
		draft.sessionIdleMinutes = '10080';
		draft.sessionAbsoluteHours = '1';
		draft.auditRetentionDays = '30';
		expect(validateDraft(draft)).toEqual({});
	});

	test('text: rune limits and the e-mail shape', () => {
		const draft = toDraft(makeSettings());
		draft.privacyControllerName = 'é'.repeat(200);
		draft.privacySupervisoryAuthority = 'x'.repeat(301);
		draft.privacyControllerEmail = 'not-an-address';
		expect(validateDraft(draft)).toEqual({
			privacySupervisoryAuthority: { code: 'too_long' },
			privacyControllerEmail: { code: 'invalid_format' }
		});
		draft.privacyControllerEmail = '';
		draft.privacySupervisoryAuthority = '';
		expect(validateDraft(draft)).toEqual({});
	});

	test('helpers', () => {
		expect(parseWhole(' 42 ')).toBe(42);
		expect(parseWhole('-1')).toBe(-1);
		expect(parseWhole('4e2')).toBeNull();
		expect(rangeParams('sessionAbsoluteHours')).toEqual({ min: 1, max: 2160 });
		expect(rangeParams('privacyControllerName')).toBeUndefined();
	});
});
