// Locale-aware formatters (plan 8.5.2, GRAPHICS.md 1.8). Pure functions plus a bound facade.
import type { Locale } from './types.js';

export const NBSP = ' ';
/** The marker of a missing value (U+2013), never zero. */
export const MISSING = '–';

const NUMBER_LOCALE: Record<Locale, string> = { hu: 'hu', en: 'en-GB', de: 'de-DE' };

const cache = new Map<string, Intl.NumberFormat | Intl.DateTimeFormat | Intl.RelativeTimeFormat>();

function cached<T extends Intl.NumberFormat | Intl.DateTimeFormat | Intl.RelativeTimeFormat>(
	key: string,
	create: () => T
): T {
	let value = cache.get(key) as T | undefined;
	if (!value) {
		value = create();
		cache.set(key, value);
	}
	return value;
}

/** Replaces the plain and narrow spaces Intl may emit with U+00A0. */
function nbsp(text: string): string {
	return text.replace(/[  ]/g, NBSP);
}

function toDate(value: Date | string | number): Date {
	return value instanceof Date ? value : new Date(value);
}

function isValid(date: Date): boolean {
	return !Number.isNaN(date.getTime());
}

/** Number with the locale's grouping; Hungarian groups from five digits with U+00A0. */
export function formatNumber(locale: Locale, n: number, opts: { fraction?: number } = {}): string {
	const key = `n:${locale}:${opts.fraction ?? ''}`;
	const format = cached(key, () => {
		const options: Intl.NumberFormatOptions = {};
		if (locale === 'hu') options.useGrouping = 'min2';
		if (opts.fraction !== undefined) {
			options.minimumFractionDigits = opts.fraction;
			options.maximumFractionDigits = opts.fraction;
		}
		return new Intl.NumberFormat(NUMBER_LOCALE[locale], options);
	});
	return nbsp(format.format(n));
}

/** Calendar date: 2026. 11. 05. / 5 Nov 2026 / 05.11.2026 */
export function formatDate(locale: Locale, value: Date | string | number): string {
	const date = toDate(value);
	if (!isValid(date)) return MISSING;
	const format = cached(`d:${locale}`, () =>
		locale === 'en'
			? new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })
			: new Intl.DateTimeFormat(NUMBER_LOCALE[locale], {
					year: 'numeric',
					month: '2-digit',
					day: '2-digit'
				})
	);
	return nbsp(format.format(date));
}

function clock(locale: Locale, date: Date, seconds: boolean): string {
	const format = cached(
		`t:${locale}:${seconds}`,
		() =>
			new Intl.DateTimeFormat(NUMBER_LOCALE[locale], {
				hour: '2-digit',
				minute: '2-digit',
				second: seconds ? '2-digit' : undefined,
				hourCycle: 'h23'
			})
	);
	return format.format(date);
}

/** 24-hour time with seconds: 14:32:08 */
export function formatTime(locale: Locale, value: Date | string | number): string {
	const date = toDate(value);
	return isValid(date) ? clock(locale, date, true) : MISSING;
}

/** Date and minute: 2026. 11. 05. 14:32 / 5 Nov 2026, 14:32 / 05.11.2026, 14:32 */
export function formatDateTime(locale: Locale, value: Date | string | number): string {
	const date = toDate(value);
	if (!isValid(date)) return MISSING;
	const separator = locale === 'hu' ? NBSP : ',' + NBSP;
	return formatDate(locale, date) + separator + clock(locale, date, false);
}

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;

function startOfDay(date: Date): number {
	return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

/**
 * Relative time with numeric 'auto': minutes within the hour, hours within the same day,
 * then calendar days (today, yesterday, 3 days ago), months and years.
 */
export function formatRelative(
	locale: Locale,
	value: Date | string | number,
	now: Date = new Date()
): string {
	const date = toDate(value);
	if (!isValid(date)) return MISSING;
	const format = cached(
		`r:${locale}`,
		() =>
			new Intl.RelativeTimeFormat(NUMBER_LOCALE[locale], {
				numeric: 'auto',
				style: locale === 'hu' ? 'narrow' : 'long'
			})
	);
	const diff = date.getTime() - now.getTime();
	const sameDay = startOfDay(date) === startOfDay(now);
	if (Math.abs(diff) < HOUR) return nbsp(format.format(Math.trunc(diff / MINUTE), 'minute'));
	if (sameDay) return nbsp(format.format(Math.trunc(diff / HOUR), 'hour'));
	const days = Math.round((startOfDay(date) - startOfDay(now)) / (24 * HOUR));
	if (Math.abs(days) < 30) return nbsp(format.format(days, 'day'));
	if (Math.abs(days) < 365) return nbsp(format.format(Math.trunc(days / 30), 'month'));
	return nbsp(format.format(Math.trunc(days / 365), 'year'));
}

export type Formatters = {
	number(n: number, opts?: { fraction?: number }): string;
	date(value: Date | string | number): string;
	time(value: Date | string | number): string;
	dateTime(value: Date | string | number): string;
	relative(value: Date | string | number, now?: Date): string;
	duration(days: number): string;
	readonly missing: string;
};

/** Binds the formatters to a locale source; `duration` uses the common.units.days plural message. */
export function createFormatters(
	getLocale: () => Locale,
	days: (count: number) => string
): Formatters {
	return {
		number: (n, opts) => formatNumber(getLocale(), n, opts),
		date: (value) => formatDate(getLocale(), value),
		time: (value) => formatTime(getLocale(), value),
		dateTime: (value) => formatDateTime(getLocale(), value),
		relative: (value, now) => formatRelative(getLocale(), value, now),
		duration: (count) => days(count),
		missing: MISSING
	};
}
