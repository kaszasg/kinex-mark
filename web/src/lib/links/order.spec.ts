import { describe, expect, it } from 'vitest';
import { makeLink } from '$lib/test/fixtures.js';
import { compareCodePoints, compareLinks, sortLabel, sortLinks } from './order.js';

describe('compareCodePoints', () => {
	it('orders by code point, not by locale', () => {
		expect(compareCodePoints('a', 'b')).toBeLessThan(0);
		expect(compareCodePoints('Z', 'a')).toBeLessThan(0);
		expect(compareCodePoints('z', 'á')).toBeLessThan(0);
		expect(compareCodePoints('ab', 'a')).toBeGreaterThan(0);
		expect(compareCodePoints('😀', '￿')).toBeGreaterThan(0);
		expect(compareCodePoints('same', 'same')).toBe(0);
	});
});

describe('sortLabel', () => {
	it('uses description, then title, then host, lower case', () => {
		expect(sortLabel({ description: 'Desc', title: 'T', host: 'h' })).toBe('desc');
		expect(sortLabel({ description: '', title: 'Title', host: 'h' })).toBe('title');
		expect(sortLabel({ description: '', title: '', host: 'Host.io' })).toBe('host.io');
	});

	it('folds accents, so an accented label sorts next to its base letter', () => {
		expect(sortLabel({ description: 'Árvíz', title: '', host: 'h' })).toBe('arviz');
		const zebra = makeLink({ id: 1, description: 'Zebra' });
		const flood = makeLink({ id: 2, description: 'Árvíz' });
		const apple = makeLink({ id: 3, description: 'alma' });
		expect(sortLinks([zebra, flood, apple], 'title').map((link) => link.id)).toEqual([3, 2, 1]);
	});
});

describe('compareLinks', () => {
	it('puts pinned links first in every order', () => {
		const pinned = makeLink({ pinned: true, useCount: 0 });
		const used = makeLink({ useCount: 10 });
		for (const sort of ['usage', 'created', 'title'] as const) {
			expect(sortLinks([used, pinned], sort)[0]).toBe(pinned);
		}
	});

	it('orders by usage, then last use (missing last), then creation, then id', () => {
		const a = makeLink({ id: 1, useCount: 5, lastUsedAt: '2026-01-02T00:00:00Z' });
		const b = makeLink({ id: 2, useCount: 5, lastUsedAt: '2026-01-03T00:00:00Z' });
		const c = makeLink({ id: 3, useCount: 5, lastUsedAt: null });
		const d = makeLink({ id: 4, useCount: 9 });
		const e = makeLink({ id: 5, useCount: 5, lastUsedAt: null, createdAt: '2026-02-01T00:00:00Z' });
		const f = makeLink({ id: 6, useCount: 5, lastUsedAt: null });
		expect(sortLinks([a, b, c, d, e, f], 'usage').map((link) => link.id)).toEqual([
			4, 2, 1, 5, 6, 3
		]);
	});

	it('orders by creation time for the newest-first order', () => {
		const older = makeLink({ id: 1, createdAt: '2026-01-01T00:00:00Z', useCount: 9 });
		const newer = makeLink({ id: 2, createdAt: '2026-01-02T00:00:00Z' });
		expect(sortLinks([older, newer], 'created').map((link) => link.id)).toEqual([2, 1]);
	});

	it('orders by sort label for the title order', () => {
		const b = makeLink({ id: 1, description: '', title: 'beta' });
		const a = makeLink({ id: 2, description: 'Alpha', title: 'zzz' });
		const host = makeLink({ id: 3, description: '', title: '', host: 'aaa.io' });
		expect(sortLinks([b, a, host], 'title').map((link) => link.id)).toEqual([3, 2, 1]);
	});

	it('treats an unparsable time as missing', () => {
		const bad = makeLink({ id: 1, useCount: 1, lastUsedAt: 'not a date' });
		const good = makeLink({ id: 2, useCount: 1, lastUsedAt: '2026-01-01T00:00:00Z' });
		expect(compareLinks(bad, good, 'usage')).toBeGreaterThan(0);
		expect(compareLinks(good, bad, 'usage')).toBeLessThan(0);
	});

	it('does not change the input array', () => {
		const items = [makeLink({ id: 1 }), makeLink({ id: 2 })];
		const copy = [...items];
		sortLinks(items, 'usage');
		expect(items).toEqual(copy);
	});
});
