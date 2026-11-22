import { describe, expect, test } from 'vitest';
import { GATE_PATHS, parseLoginReason, signedInTarget } from './redirect.js';

const ORIGIN = 'http://localhost';
const open = { mustChangePassword: false, privacyConsentRequired: false };

describe('signedInTarget', () => {
	test('the password gate comes before the consent gate and before next', () => {
		expect(
			signedInTarget({ mustChangePassword: true, privacyConsentRequired: true }, '/tags', ORIGIN)
		).toBe(GATE_PATHS.password);
	});

	test('the consent gate comes before next', () => {
		expect(signedInTarget({ ...open, privacyConsentRequired: true }, '/tags', ORIGIN)).toBe(
			'/consent'
		);
	});

	test.each([
		['/tags?x=1#a', '/tags?x=1#a'],
		[null, '/'],
		[undefined, '/'],
		['', '/'],
		['https://evil.example/', '/'],
		['//evil.example', '/'],
		['/\\evil.example', '/'],
		['/login', '/'],
		['/login?next=/tags', '/'],
		['/login#top', '/']
	])('next %s leads to %s', (next, target) => {
		expect(signedInTarget(open, next, ORIGIN)).toBe(target);
	});
});

describe('parseLoginReason', () => {
	test.each([
		['expired', 'expired'],
		['deleted', 'deleted'],
		['logout', null],
		['', null],
		[null, null],
		[undefined, null]
	])('%s gives %s', (value, reason) => {
		expect(parseLoginReason(value)).toBe(reason);
	});
});
