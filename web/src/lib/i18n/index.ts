// Public surface of the i18n layer (plan 8.5). The reactive runtime lives in runtime.svelte.ts
// because runes compile only in .svelte(.ts) modules.
export { i18n, t, hasMessage, tDynamic, fmt, catalogue } from './runtime.svelte.js';
export {
	errorTitle,
	errorText,
	errorMessage,
	errorParams,
	fieldMessage,
	knownErrorCode
} from './errors.js';
export { detectLocale, readStoredLocale, writeStoredLocale, LOCALE_STORAGE_KEY } from './detect.js';
export { MISSING, NBSP, type Formatters } from './format.js';
export {
	LOCALES,
	isLocale,
	type Catalogue,
	type Locale,
	type MessageKey,
	type Params,
	type Plural,
	type Shape
} from './types.js';
