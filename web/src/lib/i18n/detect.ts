// Initial locale choice before sign-in (plan 8.5.1): stored choice, browser languages, then hu.
import { isLocale, type Locale } from './types.js';

export const LOCALE_STORAGE_KEY = 'kxm-locale';

/** The locale stored by an earlier choice, or null (storage may be unavailable). */
export function readStoredLocale(): Locale | null {
	try {
		const value = globalThis.localStorage?.getItem(LOCALE_STORAGE_KEY);
		return isLocale(value) ? value : null;
	} catch {
		return null;
	}
}

/** Stores the locale choice; storage failures are ignored. */
export function writeStoredLocale(locale: Locale): void {
	try {
		globalThis.localStorage?.setItem(LOCALE_STORAGE_KEY, locale);
	} catch {
		// Private mode or blocked storage: the choice lasts for this page only.
	}
}

/** The first supported language prefix (hu, en, de) of the browser's preference list. */
export function fromLanguages(languages: readonly string[]): Locale | null {
	for (const language of languages) {
		const prefix = language.toLowerCase().split(/[-_]/)[0];
		if (isLocale(prefix)) return prefix;
	}
	return null;
}

/** Stored choice, then navigator.languages, then hu. */
export function detectLocale(
	stored: Locale | null = readStoredLocale(),
	languages: readonly string[] = globalThis.navigator?.languages ?? []
): Locale {
	return stored ?? fromLanguages(languages) ?? 'hu';
}
