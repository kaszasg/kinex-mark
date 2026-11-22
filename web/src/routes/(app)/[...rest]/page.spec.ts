// The (app) catch-all route of plan 8.7.11.
import { isHttpError } from '@sveltejs/kit';
import { describe, expect, test } from 'vitest';
import { load } from './+page.js';

describe('(app)/[...rest]', () => {
	test('an unknown path is a 404 of the signed-in frame', () => {
		let thrown: unknown;
		try {
			void load({} as Parameters<typeof load>[0]);
		} catch (err) {
			thrown = err;
		}
		expect(isHttpError(thrown, 404)).toBe(true);
	});
});
