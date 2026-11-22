// URL state of the users table and the shared page arithmetic (plan 8.7.7, 5.4).
import { describe, expect, test } from 'vitest';
import { pageCount, pageOffset, pageRange, parsePage } from './paging.js';
import {
	DEFAULT_USERS_FILTER,
	hasUserFilters,
	isLocked,
	parseUsersFilter,
	usersApiQuery,
	usersFilterSearch
} from './users-query.js';

const params = (search: string) => new URLSearchParams(search);

describe('paging', () => {
	test.each([
		[null, 1],
		['', 1],
		['3', 3],
		['0', 1],
		['-2', 1],
		['1.5', 1],
		['abc', 1],
		['1234567', 1]
	])('parsePage(%s) = %d', (raw, expected) => {
		expect(parsePage(raw)).toBe(expected);
	});

	test('offset, count and the shown range', () => {
		expect(pageOffset(1)).toBe(0);
		expect(pageOffset(3)).toBe(100);
		expect(pageOffset(0)).toBe(0);
		expect(pageCount(0)).toBe(1);
		expect(pageCount(50)).toBe(1);
		expect(pageCount(51)).toBe(2);
		expect(pageRange(1, 0)).toEqual({ from: 0, to: 0 });
		expect(pageRange(1, 132)).toEqual({ from: 1, to: 50 });
		expect(pageRange(3, 132)).toEqual({ from: 101, to: 132 });
		expect(pageRange(2, 30, 10)).toEqual({ from: 11, to: 20 });
	});
});

describe('users filter', () => {
	test('defaults when the URL has nothing or rubbish', () => {
		expect(parseUsersFilter(params(''))).toEqual(DEFAULT_USERS_FILTER);
		expect(parseUsersFilter(params('role=root&status=gone&page=x'))).toEqual(DEFAULT_USERS_FILTER);
	});

	test('reads every parameter and trims the search text', () => {
		expect(parseUsersFilter(params('q=%20anna%20&role=developer&status=all&page=2'))).toEqual({
			q: 'anna',
			role: 'developer',
			status: 'all',
			page: 2
		});
		expect(parseUsersFilter(params(`q=${'a'.repeat(80)}`)).q).toHaveLength(64);
	});

	test('round-trips through the query string, leaving defaults out', () => {
		expect(usersFilterSearch(DEFAULT_USERS_FILTER)).toBe('');
		const filter = { q: 'kovacs anna', role: 'admin', status: 'inactive', page: 4 } as const;
		const search = usersFilterSearch(filter);
		expect(search).toBe('?q=kovacs+anna&role=admin&status=inactive&page=4');
		expect(parseUsersFilter(params(search.slice(1)))).toEqual(filter);
	});

	test('knows when filters are set (the page does not count)', () => {
		expect(hasUserFilters({ ...DEFAULT_USERS_FILTER, page: 3 })).toBe(false);
		expect(hasUserFilters({ ...DEFAULT_USERS_FILTER, q: 'x' })).toBe(true);
		expect(hasUserFilters({ ...DEFAULT_USERS_FILTER, role: 'customer' })).toBe(true);
		expect(hasUserFilters({ ...DEFAULT_USERS_FILTER, status: 'all' })).toBe(true);
	});

	test('builds the API query with limit and offset', () => {
		expect(usersApiQuery(DEFAULT_USERS_FILTER)).toEqual({
			q: undefined,
			role: undefined,
			status: 'active',
			limit: 50,
			offset: 0
		});
		expect(usersApiQuery({ q: 'x', role: 'customer', status: 'all', page: 3 })).toEqual({
			q: 'x',
			role: 'customer',
			status: 'all',
			limit: 50,
			offset: 100
		});
	});

	test('a lock counts only while it lies in the future', () => {
		const now = Date.parse('2026-01-05T10:00:00Z');
		expect(isLocked(null, now)).toBe(false);
		expect(isLocked('2026-01-05T09:59:59Z', now)).toBe(false);
		expect(isLocked('2026-01-05T10:15:00Z', now)).toBe(true);
		expect(isLocked('not a date', now)).toBe(false);
	});
});
