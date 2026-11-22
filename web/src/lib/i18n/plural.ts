// Plural selection with cached Intl.PluralRules per locale (plan 8.5.1).
import type { Locale, Plural } from './types.js';

const rulesCache = new Map<Locale, Intl.PluralRules>();

function rulesFor(locale: Locale): Intl.PluralRules {
	let rules = rulesCache.get(locale);
	if (!rules) {
		rules = new Intl.PluralRules(locale);
		rulesCache.set(locale, rules);
	}
	return rules;
}

/** Reports whether a catalogue value is a plural object. */
export function isPlural(value: unknown): value is Plural {
	return (
		typeof value === 'object' &&
		value !== null &&
		typeof (value as Plural).one === 'string' &&
		typeof (value as Plural).other === 'string'
	);
}

/** Picks the plural form: `zero` for 0 when present, otherwise the Intl category (one or other). */
export function selectPlural(locale: Locale, message: Plural, count: number): string {
	if (count === 0 && message.zero !== undefined) return message.zero;
	const category = rulesFor(locale).select(count);
	return category === 'one' ? message.one : message.other;
}
