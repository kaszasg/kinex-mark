// The three catalogues have the same keys, plural forms and placeholders (plan 8.5.9).
import { describe, expect, test } from 'vitest';
import { catalogue, LOCALES, type Locale } from './index.js';
import { isPlural } from './plural.js';

type Leaf = string | { zero?: string; one: string; other: string };

function leaves(node: unknown, prefix = '', out = new Map<string, Leaf>()): Map<string, Leaf> {
	if (typeof node === 'string' || isPlural(node)) {
		out.set(prefix, node as Leaf);
		return out;
	}
	if (typeof node === 'object' && node !== null) {
		for (const [key, value] of Object.entries(node)) {
			leaves(value, prefix ? `${prefix}.${key}` : key, out);
		}
	}
	return out;
}

function placeholders(leaf: Leaf): string[] {
	const texts = typeof leaf === 'string' ? [leaf] : [leaf.one, leaf.other];
	const names = new Set<string>();
	for (const text of texts) for (const m of text.matchAll(/\{(\w+)\}/g)) names.add(m[1]);
	return [...names].sort();
}

const all = Object.fromEntries(LOCALES.map((l) => [l, leaves(catalogue(l))])) as Record<
	Locale,
	Map<string, Leaf>
>;

describe('catalogue parity', () => {
	test.each(['en', 'de'] as const)('%s has exactly the Hungarian keys', (locale) => {
		expect([...all[locale].keys()].sort()).toEqual([...all.hu.keys()].sort());
	});

	test.each(['en', 'de'] as const)('%s uses the same leaf kind (text or plural)', (locale) => {
		const mismatched = [...all.hu].filter(
			([key, leaf]) => typeof leaf !== typeof all[locale].get(key)
		);
		expect(mismatched.map(([key]) => key)).toEqual([]);
	});

	test.each(['en', 'de'] as const)('%s has the same placeholders', (locale) => {
		const mismatched = [...all.hu]
			.filter(([key, leaf]) => {
				const other = all[locale].get(key);
				return other !== undefined && placeholders(leaf).join() !== placeholders(other).join();
			})
			.map(([key]) => key);
		expect(mismatched).toEqual([]);
	});

	test.each([...LOCALES])('%s has no empty text', (locale) => {
		const empty = [...all[locale]]
			.filter(([, leaf]) =>
				typeof leaf === 'string'
					? leaf.trim() === ''
					: leaf.one.trim() === '' || leaf.other.trim() === ''
			)
			.map(([key]) => key);
		expect(empty).toEqual([]);
	});

	test.each([...LOCALES])('%s plural messages carry the {count} placeholder', (locale) => {
		const without = [...all[locale]]
			.filter(([, leaf]) => typeof leaf !== 'string' && !leaf.other.includes('{count}'))
			.map(([key]) => key);
		expect(without).toEqual([]);
	});

	test.each([...LOCALES])('%s texts follow the tone rules (no "!" and no emoji)', (locale) => {
		const offending = [...all[locale]]
			.filter(([, leaf]) => {
				const text = typeof leaf === 'string' ? leaf : `${leaf.zero ?? ''}${leaf.one}${leaf.other}`;
				return text.includes('!') || /\p{Extended_Pictographic}/u.test(text);
			})
			.map(([key]) => key);
		expect(offending).toEqual([]);
	});

	test.each([...LOCALES])(
		'%s keeps a value and its unit together with U+00A0 (plan 8.5.4, GRAPHICS.md 1.8)',
		(locale) => {
			const unit =
				/(?:\{\w+\}|\d) (?:nap|perc|óra|másodperc|bájt|MB|KB|kB|day|minute|hour|second|byte|Tag|Minute|Stunde|Sekunde|Byte)/u;
			const offending = [...all[locale]]
				.filter(([, leaf]) => {
					const text =
						typeof leaf === 'string' ? leaf : `${leaf.zero ?? ''} ${leaf.one} ${leaf.other}`;
					return unit.test(text);
				})
				.map(([key]) => key);
			expect(offending).toEqual([]);
		}
	);

	test('Hungarian texts never speak as "we" (GRAPHICS.md 1.2)', () => {
		// First-person plural endings; third-person pronouns ("köztük", "rájuk") end the same way.
		const we = /\p{L}+(?:tuk|tük|juk|jük|unk|ünk)(?!\p{L})/gu;
		const allowed = new Set(['köztük', 'rájuk']);
		const offending = [...all.hu].flatMap(([key, leaf]) => {
			const text = typeof leaf === 'string' ? leaf : `${leaf.zero ?? ''} ${leaf.one} ${leaf.other}`;
			return [...text.matchAll(we)]
				.map((match) => match[0])
				.filter((word) => !allowed.has(word.toLowerCase()))
				.map((word) => `${key}: ${word}`);
		});
		expect(offending).toEqual([]);
	});

	test('Hungarian "cím" means the page title next to the URL field (plan 2.3 G10)', () => {
		// The URL checks and the metadata status sit next to the "Cím" (title) field.
		const scoped = (key: string) =>
			key.startsWith('fields.byField.url.') || key.startsWith('fetch.status.');
		const bare = /(?<![\p{L}-])(?<!hálózati )cím(?!\p{L}*-)(?!ke|zett)/iu;
		const offending = [...all.hu]
			.filter(([key, leaf]) => scoped(key) && typeof leaf === 'string' && bare.test(leaf))
			.map(([key]) => key);
		expect(offending).toEqual([]);
	});

	// One term per concept (plan 8.5.4): the second factor, its code, the icon and tag matching.
	test.each([
		['en', [/two-factor authentication/i, /authenticator code/i]],
		['de', [/Authentifizierungscode/, /(?<!\p{L})Icon/u, /Label-Abgleich/]]
	] as const)('%s uses one term per concept', (locale, variants) => {
		const offending = [...all[locale]]
			.filter(([, leaf]) => {
				const text =
					typeof leaf === 'string' ? leaf : `${leaf.zero ?? ''} ${leaf.one} ${leaf.other}`;
				return variants.some((variant) => variant.test(text));
			})
			.map(([key]) => key);
		expect(offending).toEqual([]);
	});
});
