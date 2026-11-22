// The i18n runtime: t(), plurals, fallbacks, locale switching, error texts (plan 8.5).
import { afterEach, describe, expect, test, vi } from 'vitest';
import { ApiError } from '$lib/api/errors.js';
import {
	catalogue,
	errorMessage,
	errorParams,
	errorText,
	errorTitle,
	fieldMessage,
	fmt,
	hasMessage,
	i18n,
	isLocale,
	knownErrorCode,
	LOCALE_STORAGE_KEY,
	t,
	tDynamic
} from './index.js';

afterEach(() => {
	vi.restoreAllMocks();
	localStorage.clear();
});

describe('t', () => {
	test('translates in the active locale', () => {
		expect(t('nav.links')).toBe('Linkek');
		i18n.setLocale('en', { persist: false });
		expect(t('nav.links')).toBe('Links');
		i18n.setLocale('de', { persist: false });
		expect(t('nav.tags')).toBe('Labels');
	});

	test('fills placeholders and formats numbers by locale', () => {
		expect(t('common.tagPicker.limit', { max: 20 })).toContain('20');
		i18n.setLocale('en', { persist: false });
		expect(t('fields.range', { min: 1000, max: 36500 })).toBe(
			'Enter a value between 1,000 and 36,500.'
		);
	});

	test('formats Date placeholders as dates', () => {
		i18n.setLocale('de', { persist: false });
		expect(t('fields.range', { min: new Date(2026, 10, 5), max: 'x' })).toContain('05.11.2026');
	});

	test('selects plural forms', () => {
		i18n.setLocale('en', { persist: false });
		expect(t('common.units.days', { count: 1 })).toBe('1 day');
		expect(t('common.units.days', { count: 3 })).toBe('3 days');
		i18n.setLocale('hu', { persist: false });
		expect(t('common.units.days', { count: 3 })).toBe('3 nap');
	});

	test('a plural without a numeric count uses the other form', () => {
		i18n.setLocale('en', { persist: false });
		expect(t('common.units.days')).toBe('{count} days');
	});

	test('leaves a missing placeholder visible', () => {
		expect(t('common.tagPicker.limit')).toContain('{max}');
	});

	test('falls back to the key for an unknown key', () => {
		expect(t('nope.missing' as never)).toBe('nope.missing');
	});

	test('falls back to Hungarian when a locale lacks the key', () => {
		const en = catalogue('en') as unknown as { nav: Record<string, unknown> };
		const saved = en.nav.links;
		delete en.nav.links;
		try {
			i18n.setLocale('en', { persist: false });
			expect(t('nav.links')).toBe('Linkek');
		} finally {
			en.nav.links = saved;
		}
	});

	test('a key pointing at a namespace object is returned as is', () => {
		expect(t('nav.groups' as never)).toBe('nav.groups');
	});
});

describe('hasMessage and tDynamic', () => {
	test('hasMessage checks leaves of the Hungarian catalogue', () => {
		expect(hasMessage('errors.not_found.title')).toBe(true);
		expect(hasMessage('errors.not_found')).toBe(false);
		expect(hasMessage('errors.unknown_code.title')).toBe(false);
	});

	test('tDynamic returns the fallback or the key itself', () => {
		expect(tDynamic('roles.admin')).toBe('Adminisztrátor');
		expect(tDynamic('roles.robot')).toBe('roles.robot');
		expect(tDynamic('roles.robot', undefined, 'robot')).toBe('robot');
	});
});

describe('setLocale', () => {
	test('persists the choice and updates <html lang>', () => {
		i18n.setLocale('de');
		expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('de');
		expect(document.documentElement.lang).toBe('de');
		expect(i18n.explicit).toBeNull();
	});

	test('records an explicit choice until it is cleared', () => {
		i18n.setLocale('en', { explicit: true, persist: false });
		expect(i18n.explicit).toBe('en');
		i18n.clearExplicit();
		expect(i18n.explicit).toBeNull();
	});

	test('fmt follows the active locale', () => {
		expect(fmt.number(12480)).toBe('12 480');
		i18n.setLocale('en', { persist: false });
		expect(fmt.number(12480)).toBe('12,480');
		expect(fmt.duration(2)).toBe('2 days');
	});

	test('isLocale accepts the three locales only', () => {
		expect(isLocale('hu')).toBe(true);
		expect(isLocale('fr')).toBe(false);
		expect(isLocale(undefined)).toBe(false);
	});
});

describe('error texts', () => {
	test('title and text of a known code', () => {
		const err = new ApiError(404, 'not_found');
		expect(errorTitle(err)).toBe(t('errors.not_found.title'));
		expect(errorText(err)).toBe(t('errors.not_found.text'));
		expect(errorMessage(err)).toBe(`${errorTitle(err)} ${errorText(err)}`);
	});

	test('unknown server codes fall back to internal_error', () => {
		const err = new ApiError(500, 'brand_new_code' as never);
		expect(knownErrorCode(err)).toBe('internal_error');
		expect(errorTitle(err)).toBe(t('errors.internal_error.title'));
	});

	test('too_many_attempts rounds the wait up to whole minutes', () => {
		expect(
			errorParams(new ApiError(429, 'too_many_attempts', {}, { retryAfterSeconds: 61 }))
		).toEqual({
			minutes: 2
		});
		expect(
			errorParams(new ApiError(429, 'too_many_attempts', {}, { retryAfterSeconds: 5 }))
		).toEqual({
			minutes: 1
		});
		expect(errorParams(new ApiError(429, 'too_many_attempts'))).toEqual({ minutes: 1 });
		i18n.setLocale('en', { persist: false });
		expect(
			errorText(new ApiError(429, 'too_many_attempts', {}, { retryAfterSeconds: 120 }))
		).toContain('2');
	});

	test('setting_locked lists the locked keys', () => {
		expect(
			errorParams(new ApiError(409, 'setting_locked', {}, { keys: ['sessionIdleMinutes', 'x'] }))
		).toEqual({ keys: 'sessionIdleMinutes, x' });
		expect(errorParams(new ApiError(409, 'setting_locked', {}, {}))).toEqual({ keys: '' });
	});
});

describe('fieldMessage', () => {
	test('per-field override first', () => {
		expect(fieldMessage('url', 'invalid_format')).toBe(t('fields.byField.url.invalid_format'));
	});

	test('generic field text otherwise', () => {
		expect(fieldMessage('title', 'too_long')).toBe(t('fields.too_long'));
	});

	test('range text when bounds are known', () => {
		expect(fieldMessage('archiveAfterDays', 'out_of_range', { min: 1, max: 3650 })).toBe(
			t('fields.range', { min: 1, max: 3650 })
		);
		expect(fieldMessage('archiveAfterDays', 'out_of_range')).toBe(t('fields.out_of_range'));
	});

	test('unknown codes read as invalid', () => {
		expect(fieldMessage('title', 'weird')).toBe(t('fields.invalid'));
	});
});

describe('dev warnings', () => {
	test('a missing key warns in development', () => {
		const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
		t('missing.key' as never);
		expect(warn).toHaveBeenCalled();
	});
});
