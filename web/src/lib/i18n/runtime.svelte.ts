// Reactive i18n runtime (plan 8.5.1): the active locale, t(), hasMessage(), tDynamic() and fmt.
import de from './messages/de/index.js';
import en from './messages/en/index.js';
import hu from './messages/hu/index.js';
import { detectLocale, writeStoredLocale } from './detect.js';
import { createFormatters, formatDate, formatNumber } from './format.js';
import { isPlural, selectPlural } from './plural.js';
import type { Catalogue, Locale, MessageKey, Params } from './types.js';

const catalogues: Record<Locale, Catalogue> = { hu, en, de };

const dev = import.meta.env.DEV;

function warn(message: string): void {
	if (dev) console.warn(`[i18n] ${message}`);
}

/** Walks a dot path through a catalogue; undefined when any segment is missing. */
function lookup(catalogue: unknown, key: string): unknown {
	let node: unknown = catalogue;
	for (const part of key.split('.')) {
		if (typeof node !== 'object' || node === null || !(part in node)) return undefined;
		node = (node as Record<string, unknown>)[part];
	}
	return node;
}

function isLeaf(value: unknown): boolean {
	return typeof value === 'string' || isPlural(value);
}

class I18n {
	/** The active locale; reading it in a template re-renders on change. */
	locale = $state<Locale>(detectLocale());
	/** A locale the user picked by hand before signing in; it wins over the stored preference. */
	explicit = $state<Locale | null>(null);

	/** Switches the locale synchronously; persists the choice and updates <html lang>. */
	setLocale(next: Locale, opts: { persist?: boolean; explicit?: boolean } = {}): void {
		this.locale = next;
		if (opts.explicit) this.explicit = next;
		if (opts.persist ?? true) writeStoredLocale(next);
		if (typeof document !== 'undefined') document.documentElement.lang = next;
	}

	/** Forgets the explicit pre-sign-in choice once it has been applied. */
	clearExplicit(): void {
		this.explicit = null;
	}
}

export const i18n = new I18n();

function interpolate(
	locale: Locale,
	text: string,
	params: Params | undefined,
	key: string
): string {
	return text.replace(/\{(\w+)\}/g, (match, name: string) => {
		const value = params?.[name];
		if (value === undefined) {
			warn(`missing placeholder "${name}" for "${key}"`);
			return match;
		}
		if (typeof value === 'number') return formatNumber(locale, value);
		if (value instanceof Date) return formatDate(locale, value);
		return value;
	});
}

function render(locale: Locale, message: unknown, params: Params | undefined, key: string): string {
	if (typeof message === 'string') return interpolate(locale, message, params, key);
	if (isPlural(message)) {
		const count = params?.count;
		if (typeof count !== 'number') warn(`plural "${key}" needs a numeric count`);
		const form = selectPlural(locale, message, typeof count === 'number' ? count : Number.NaN);
		return interpolate(locale, form, params, key);
	}
	return key;
}

function translate(key: string, params?: Params): string {
	const locale = i18n.locale;
	let message = lookup(catalogues[locale], key);
	if (!isLeaf(message)) {
		warn(`missing "${key}" in ${locale}`);
		message = lookup(catalogues.hu, key);
		if (!isLeaf(message)) return key;
		return render('hu', message, params, key);
	}
	return render(locale, message, params, key);
}

/** Translates a key of the catalogue; reactive to i18n.locale. */
export function t(key: MessageKey, params?: Params): string {
	return translate(key, params);
}

/** Runtime check for keys built from server codes (errors, fields.byField, audit, fetch.errors). */
export function hasMessage(key: string): key is MessageKey {
	return isLeaf(lookup(catalogues.hu, key));
}

/** t() for a dynamic key: the fallback (default: the key itself, shown in mono by the caller) when missing. */
export function tDynamic(key: string, params?: Params, fallback?: string): string {
	return hasMessage(key) ? translate(key, params) : (fallback ?? key);
}

/** Formatters bound to the active locale (plan 8.5.2). */
export const fmt = createFormatters(
	() => i18n.locale,
	(count) => translate('common.units.days', { count })
);

/** The catalogue of one locale, for tests and tooling. */
export function catalogue(locale: Locale): Catalogue {
	return catalogues[locale];
}
