// Backend contract vs. translation catalogues (plan 8.5.9, 9.6).
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, test } from 'vitest';
import { catalogue, hasMessage, LOCALES, type Locale } from './index.js';

type Contract = {
	errorCodes: string[];
	fieldCodes: string[];
	auditEvents: string[];
	auditDetailKeys: string[];
	auditDetailValues: Record<string, string[]>;
	fetchErrors: string[];
	tagColors: string[];
	roles: string[];
	locales: string[];
};

// Vitest runs with web/ as the working directory.
const contract = JSON.parse(
	readFileSync(resolve(process.cwd(), '../internal/api/testdata/contract.json'), 'utf8')
) as Contract;

/** The twelve notice sections of plan 8.7.10 and the seven data subject rights. */
const PRIVACY_SECTIONS = [
	'controller',
	'data',
	'purposes',
	'legalBasis',
	'recipients',
	'transfers',
	'automatedDecisions',
	'retention',
	'storage',
	'fetching',
	'adminAccess',
	'rights'
] as const;
const PRIVACY_RIGHTS = [
	'access',
	'portability',
	'rectification',
	'erasure',
	'restriction',
	'objection',
	'complaint'
] as const;

function leaf(locale: Locale, key: string): unknown {
	let node: unknown = catalogue(locale);
	for (const part of key.split('.')) {
		if (typeof node !== 'object' || node === null) return undefined;
		node = (node as Record<string, unknown>)[part];
	}
	return node;
}

function missing(keys: string[]): string[] {
	const out: string[] = [];
	for (const locale of LOCALES) {
		for (const key of keys) {
			const value = leaf(locale, key);
			const ok =
				(typeof value === 'string' && value.trim() !== '') ||
				(typeof value === 'object' && value !== null && 'other' in value);
			if (!ok) out.push(`${locale}:${key}`);
		}
	}
	return out;
}

/** Every namespace is filled since wave 2 closed, so every group is always active. */
function group(name: string, keys: () => string[]): void {
	test(name, () => {
		expect(missing(keys())).toEqual([]);
	});
}

describe('contract golden file', () => {
	test('lists every group the catalogues are checked against', () => {
		expect(contract.errorCodes.length).toBeGreaterThan(0);
		expect(contract.fieldCodes.length).toBeGreaterThan(0);
		expect(contract.locales).toEqual([...LOCALES]);
	});
});

describe('active groups', () => {
	test('errors.<code>.title and .text exist for every error code and network_error', () => {
		const codes = [...contract.errorCodes, 'network_error'];
		const keys = codes.flatMap((code) => [`errors.${code}.title`, `errors.${code}.text`]);
		expect(missing(keys)).toEqual([]);
	});

	test('fields.<code> exists for every field code', () => {
		expect(missing(contract.fieldCodes.map((code) => `fields.${code}`))).toEqual([]);
	});

	test('roles.<role> exists for every role', () => {
		expect(missing(contract.roles.map((role) => `roles.${role}`))).toEqual([]);
	});

	test('nav.locale.<locale> exists for every locale', () => {
		expect(missing(contract.locales.map((locale) => `nav.locale.${locale}`))).toEqual([]);
	});

	test('hasMessage agrees with the error catalogue', () => {
		for (const code of contract.errorCodes) expect(hasMessage(`errors.${code}.title`)).toBe(true);
	});
});

describe('feature groups', () => {
	group('audit.events.<event> exists for every audit event', () =>
		contract.auditEvents.map((event) => `audit.events.${event}`)
	);
	group('audit.details.keys.<key> exists for every detail key', () =>
		contract.auditDetailKeys.map((key) => `audit.details.keys.${key}`)
	);
	group('audit.details.values.<key>.<value> exists for every enumerated value', () =>
		Object.entries(contract.auditDetailValues).flatMap(([key, values]) =>
			values.map((value) => `audit.details.values.${key}.${value}`)
		)
	);
	group('fetch.errors.<code> exists for every fetch error', () =>
		contract.fetchErrors.map((code) => `fetch.errors.${code}`)
	);
	group('tagColors.<color> exists for every tag colour', () =>
		contract.tagColors.map((color) => `tagColors.${color}`)
	);
	group('privacy.sections and privacy.rights exist for the notice', () => [
		...PRIVACY_SECTIONS.flatMap((s) => [
			`privacy.sections.${s}.title`,
			`privacy.sections.${s}.body`
		]),
		...PRIVACY_RIGHTS.map((r) => `privacy.rights.${r}`)
	]);
});

describe('ErrorCode union', () => {
	test('matches the golden error codes plus network_error', () => {
		const source = readFileSync(resolve(process.cwd(), 'src/lib/api/errors.ts'), 'utf8');
		const block = /export type ErrorCode =([\s\S]*?);/.exec(source)?.[1] ?? '';
		const literals = [...block.matchAll(/'([a-z_]+)'/g)].map((m) => m[1]).sort();
		expect(literals).toEqual([...contract.errorCodes, 'network_error'].sort());
	});

	test('FieldCode union matches the golden field codes', () => {
		const source = readFileSync(resolve(process.cwd(), 'src/lib/api/errors.ts'), 'utf8');
		const block = /export type FieldCode =([\s\S]*?);/.exec(source)?.[1] ?? '';
		const literals = [...block.matchAll(/'([a-z_]+)'/g)].map((m) => m[1]).sort();
		expect(literals).toEqual([...contract.fieldCodes].sort());
	});
});
