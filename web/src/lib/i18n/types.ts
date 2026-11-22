// Catalogue and key types (plan 8.5.1). The Hungarian catalogue is the type source.
import type hu from './messages/hu/index.js';

/** A plural message: selected by Intl.PluralRules; `zero` overrides count === 0. */
export type Plural = { zero?: string; one: string; other: string };

/** Same shape as the Hungarian catalogue, with every leaf widened to string / Plural. */
export type Shape<T> = {
	[K in keyof T]: T[K] extends string ? string : T[K] extends Plural ? Plural : Shape<T[K]>;
};
export type Catalogue = Shape<typeof hu>;

/** Dot paths to every leaf ("links.toolbar.search"); a Plural object counts as a leaf. */
type Leaves<T, P extends string = ''> = {
	[K in keyof T & string]: T[K] extends string | Plural ? `${P}${K}` : Leaves<T[K], `${P}${K}.`>;
}[keyof T & string];
export type MessageKey = Leaves<typeof hu>;
export type Locale = 'hu' | 'en' | 'de';
export type Params = Record<string, string | number | Date>;

export const LOCALES: readonly Locale[] = ['hu', 'en', 'de'];

/** Reports whether a string is a supported locale. */
export function isLocale(value: unknown): value is Locale {
	return value === 'hu' || value === 'en' || value === 'de';
}
