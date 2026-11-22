// Locale-aware formatting (plan 8.5.2, GRAPHICS.md 1.8).
import { describe, expect, test } from 'vitest';
import {
	createFormatters,
	formatDate,
	formatDateTime,
	formatNumber,
	formatRelative,
	formatTime,
	MISSING,
	NBSP
} from './format.js';
import { detectLocale, fromLanguages, readStoredLocale, writeStoredLocale } from './detect.js';
import { isPlural, selectPlural } from './plural.js';

const at = (y: number, m: number, d: number, h = 0, min = 0, s = 0) =>
	new Date(y, m - 1, d, h, min, s);

describe('formatNumber', () => {
	test.each([
		['hu', 1284, '1284'],
		['hu', 12480, `12${NBSP}480`],
		['en', 1284, '1,284'],
		['de', 1284, '1.284'],
		['de', 12480, '12.480']
	] as const)('%s %d -> %s', (locale, n, expected) => {
		expect(formatNumber(locale, n)).toBe(expected);
	});

	test('fixed fraction digits', () => {
		expect(formatNumber('hu', 24.8, { fraction: 1 })).toBe('24,8');
		expect(formatNumber('en', 24.8, { fraction: 2 })).toBe('24.80');
	});
});

describe('dates and times', () => {
	const date = at(2026, 11, 5, 14, 32, 8);

	test.each([
		['hu', '2026. 11. 05.'],
		['en', '5 Nov 2026'],
		['de', '05.11.2026']
	] as const)('formatDate %s', (locale, expected) => {
		expect(formatDate(locale, date).replace(/ /g, ' ')).toBe(expected);
	});

	test('formatTime is 24-hour with seconds', () => {
		expect(formatTime('hu', date)).toBe('14:32:08');
		expect(formatTime('en', date)).toBe('14:32:08');
	});

	test('formatDateTime joins date and minute', () => {
		expect(formatDateTime('hu', date).replace(/ /g, ' ')).toBe('2026. 11. 05. 14:32');
		expect(formatDateTime('en', date).replace(/ /g, ' ')).toBe('5 Nov 2026, 14:32');
		expect(formatDateTime('de', date).replace(/ /g, ' ')).toBe('05.11.2026, 14:32');
	});

	test('invalid values render the missing marker', () => {
		expect(formatDate('hu', 'nonsense')).toBe(MISSING);
		expect(formatTime('hu', Number.NaN)).toBe(MISSING);
		expect(formatDateTime('en', 'nonsense')).toBe(MISSING);
		expect(formatRelative('de', 'nonsense')).toBe(MISSING);
	});

	test('accepts ISO strings and timestamps', () => {
		expect(formatTime('hu', date.getTime())).toBe('14:32:08');
		expect(formatDate('de', date.toISOString())).toBe('05.11.2026');
	});
});

describe('formatRelative', () => {
	const now = at(2026, 11, 21, 12, 0);
	const plain = (text: string) => text.replace(/ /g, ' ');

	test('Hungarian narrow forms', () => {
		expect(plain(formatRelative('hu', at(2026, 11, 21, 11, 55), now))).toBe('5 perce');
		expect(plain(formatRelative('hu', at(2026, 11, 21, 10, 0), now))).toBe('2 órája');
		expect(plain(formatRelative('hu', at(2026, 11, 18, 9, 0), now))).toBe('3 napja');
		expect(formatRelative('hu', at(2026, 11, 20, 9, 0), now)).toBe('tegnap');
	});

	test('English and German long forms', () => {
		expect(plain(formatRelative('en', at(2026, 11, 18), now))).toBe('3 days ago');
		expect(plain(formatRelative('de', at(2026, 11, 18), now))).toBe('vor 3 Tagen');
		expect(plain(formatRelative('en', now, now))).toBe('this minute');
	});

	test('months and years', () => {
		expect(plain(formatRelative('en', at(2026, 8, 1), now))).toBe('3 months ago');
		expect(plain(formatRelative('en', at(2024, 6, 1), now))).toBe('2 years ago');
	});

	test('defaults to the current time', () => {
		expect(plain(formatRelative('en', new Date()))).toBe('this minute');
	});
});

describe('createFormatters', () => {
	test('follows the locale source and formats durations through the callback', () => {
		let locale: 'hu' | 'en' = 'hu';
		const fmt = createFormatters(
			() => locale,
			(count) => `${count} d`
		);
		expect(fmt.number(1284)).toBe('1284');
		locale = 'en';
		expect(fmt.number(1284)).toBe('1,284');
		expect(fmt.duration(3)).toBe('3 d');
		expect(fmt.missing).toBe(MISSING);
		expect(fmt.date(at(2026, 1, 2))).toContain('2026');
		expect(fmt.time(at(2026, 1, 2, 3, 4, 5))).toBe('03:04:05');
		expect(fmt.dateTime(at(2026, 1, 2, 3, 4))).toContain('03:04');
		expect(fmt.relative(at(2026, 1, 2), at(2026, 1, 3))).toBe('yesterday');
	});
});

describe('plural', () => {
	const message = { zero: 'none', one: 'one item', other: 'many items' };

	test('isPlural recognises plural objects', () => {
		expect(isPlural(message)).toBe(true);
		expect(isPlural('text')).toBe(false);
		expect(isPlural(null)).toBe(false);
		expect(isPlural({ one: 'x' })).toBe(false);
	});

	test('zero wins for 0, then Intl categories', () => {
		expect(selectPlural('en', message, 0)).toBe('none');
		expect(selectPlural('en', message, 1)).toBe('one item');
		expect(selectPlural('en', message, 2)).toBe('many items');
		expect(selectPlural('en', { one: 'a', other: 'b' }, 0)).toBe('b');
		expect(selectPlural('hu', message, 1)).toBe('one item');
		expect(selectPlural('de', message, 5)).toBe('many items');
	});
});

describe('detectLocale', () => {
	test('stored choice, then browser languages, then hu', () => {
		expect(detectLocale('de', ['en-US'])).toBe('de');
		expect(detectLocale(null, ['fr-FR', 'en-GB'])).toBe('en');
		expect(detectLocale(null, ['fr'])).toBe('hu');
		expect(detectLocale(null, [])).toBe('hu');
	});

	test('fromLanguages reads the language prefix', () => {
		expect(fromLanguages(['DE_at'])).toBe('de');
		expect(fromLanguages(['hu-HU'])).toBe('hu');
		expect(fromLanguages(['it'])).toBeNull();
	});

	test('reads and writes the stored locale', () => {
		writeStoredLocale('en');
		expect(readStoredLocale()).toBe('en');
		localStorage.setItem('kxm-locale', 'xx');
		expect(readStoredLocale()).toBeNull();
		localStorage.removeItem('kxm-locale');
	});

	test('storage failures are ignored', () => {
		const original = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
		Object.defineProperty(globalThis, 'localStorage', {
			configurable: true,
			get() {
				throw new Error('blocked');
			}
		});
		try {
			expect(readStoredLocale()).toBeNull();
			expect(() => writeStoredLocale('de')).not.toThrow();
		} finally {
			if (original) Object.defineProperty(globalThis, 'localStorage', original);
		}
	});
});
