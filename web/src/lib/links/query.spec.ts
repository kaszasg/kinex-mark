import { describe, expect, it } from 'vitest';
import { makeLink, makePreferences } from '$lib/test/fixtures.js';
import {
	defaultQuery,
	defaultsFromPreferences,
	FALLBACK_DEFAULTS,
	hasActiveFilters,
	linkIdFromParams,
	matchesQuery,
	MAX_FILTER_TAGS,
	MAX_QUERY_LENGTH,
	queryFromParams,
	queryHref,
	queryToParams,
	sameQuery,
	toLinkQuery,
	type ListQuery,
	type QueryDefaults
} from './query.js';

const defaults: QueryDefaults = FALLBACK_DEFAULTS;

function params(search: string): URLSearchParams {
	return new URLSearchParams(search);
}

describe('defaultsFromPreferences', () => {
	it('maps the preferences and falls back without them', () => {
		expect(defaultsFromPreferences(null)).toBe(FALLBACK_DEFAULTS);
		expect(
			defaultsFromPreferences(
				makePreferences({ sort: 'title', tagMatch: 'all', showArchived: true })
			)
		).toEqual({ sort: 'title', tagMatch: 'all', archived: 'include' });
		expect(defaultsFromPreferences(makePreferences()).archived).toBe('exclude');
	});
});

describe('queryFromParams', () => {
	it('reads every parameter', () => {
		const query = queryFromParams(
			params('q=%20grafana%20&tags=3,1,3,x,0&match=all&scope=own&archived=only&sort=title'),
			defaults
		);
		expect(query).toEqual({
			q: 'grafana',
			tags: [3, 1],
			tagMatch: 'all',
			untagged: false,
			scope: 'own',
			archived: 'only',
			sort: 'title'
		});
	});

	it('uses the defaults for missing or invalid values', () => {
		const own: QueryDefaults = { sort: 'created', tagMatch: 'all', archived: 'include' };
		expect(queryFromParams(params('scope=bad&sort=bad&archived=bad&match=bad'), own)).toEqual(
			defaultQuery(own)
		);
	});

	it('accepts untagged=true or 1, and the tags win over it', () => {
		expect(queryFromParams(params('untagged=true'), defaults).untagged).toBe(true);
		expect(queryFromParams(params('untagged=1'), defaults).untagged).toBe(true);
		expect(queryFromParams(params('untagged=yes'), defaults).untagged).toBe(false);
		expect(queryFromParams(params('untagged=true&tags=2'), defaults).untagged).toBe(false);
	});

	it('applies the limits', () => {
		const many = Array.from({ length: 30 }, (_, index) => index + 1).join(',');
		expect(queryFromParams(params(`tags=${many}`), defaults).tags).toHaveLength(MAX_FILTER_TAGS);
		const long = 'é'.repeat(MAX_QUERY_LENGTH + 20);
		expect([...queryFromParams(params(`q=${long}`), defaults).q]).toHaveLength(MAX_QUERY_LENGTH);
	});
});

describe('queryToParams and queryHref', () => {
	it('writes only the deviations from the defaults', () => {
		expect(queryHref(defaultQuery(defaults), defaults)).toBe('/');
		const query: ListQuery = {
			q: ' x ',
			tags: [1, 2],
			tagMatch: 'all',
			untagged: false,
			scope: 'shared',
			archived: 'include',
			sort: 'created'
		};
		expect(queryToParams(query, defaults, 7).toString()).toBe(
			'q=x&tags=1%2C2&match=all&scope=shared&archived=include&sort=created&link=7'
		);
	});

	it('writes untagged only without tags and round-trips', () => {
		const query = { ...defaultQuery(defaults), untagged: true };
		const href = queryHref(query, defaults);
		expect(href).toBe('/?untagged=true');
		expect(queryFromParams(new URL(href, 'http://x').searchParams, defaults)).toEqual(query);
	});
});

describe('linkIdFromParams', () => {
	it('reads a positive integer only', () => {
		expect(linkIdFromParams(params('link=12'))).toBe(12);
		expect(linkIdFromParams(params('link=0'))).toBeNull();
		expect(linkIdFromParams(params('link=-1'))).toBeNull();
		expect(linkIdFromParams(params('link=abc'))).toBeNull();
		expect(linkIdFromParams(params('link=99999999999999999999'))).toBeNull();
		expect(linkIdFromParams(params(''))).toBeNull();
	});
});

describe('toLinkQuery', () => {
	it('maps to the API query', () => {
		expect(toLinkQuery(defaultQuery(defaults))).toEqual({
			q: undefined,
			tags: undefined,
			tagMatch: 'any',
			untagged: undefined,
			scope: 'all',
			archived: 'exclude',
			sort: 'usage'
		});
		expect(toLinkQuery({ ...defaultQuery(defaults), q: ' a ', tags: [4] })).toMatchObject({
			q: 'a',
			tags: [4]
		});
		expect(toLinkQuery({ ...defaultQuery(defaults), untagged: true }).untagged).toBe(true);
	});

	it('sends the search text the way the server accepts it', () => {
		// A pasted tab, a bidi override and an over-long text used to fail with invalid_query.
		expect(toLinkQuery({ ...defaultQuery(defaults), q: 'graf\tana\u202e' }).q).toBe('graf ana');
		const long = 'x'.repeat(MAX_QUERY_LENGTH + 5);
		expect(toLinkQuery({ ...defaultQuery(defaults), q: long }).q).toHaveLength(MAX_QUERY_LENGTH);
		expect(toLinkQuery({ ...defaultQuery(defaults), q: '\u0000\u200f' }).q).toBeUndefined();
		expect(queryFromParams(params('q=a%09b%E2%80%8E'), defaults).q).toBe('a b');
		expect(queryHref({ ...defaultQuery(defaults), q: 'a\nb' }, defaults)).toBe('/?q=a+b');
	});
});

describe('sameQuery', () => {
	it('compares every field', () => {
		const base = defaultQuery(defaults);
		expect(sameQuery(base, { ...base })).toBe(true);
		expect(sameQuery(base, { ...base, tags: [1] })).toBe(false);
		expect(sameQuery({ ...base, tags: [1, 2] }, { ...base, tags: [2, 1] })).toBe(false);
		expect(sameQuery(base, { ...base, q: 'x' })).toBe(false);
		expect(sameQuery(base, { ...base, sort: 'title' })).toBe(false);
	});
});

describe('matchesQuery', () => {
	const base = defaultQuery(defaults);
	const names = new Map([
		[1, 'infra'],
		[2, 'docs']
	]);
	const tagName = (id: number) => names.get(id);

	it('filters the archived state', () => {
		const archived = makeLink({ archivedAt: '2026-01-01T00:00:00Z' });
		const active = makeLink();
		expect(matchesQuery(archived, base)).toBe(false);
		expect(matchesQuery(archived, { ...base, archived: 'include' })).toBe(true);
		expect(matchesQuery(active, { ...base, archived: 'only' })).toBe(false);
	});

	it('filters the scope', () => {
		const shared = makeLink({ isOwner: false });
		expect(matchesQuery(shared, { ...base, scope: 'own' })).toBe(false);
		expect(matchesQuery(makeLink(), { ...base, scope: 'shared' })).toBe(false);
		expect(matchesQuery(shared, { ...base, scope: 'shared' })).toBe(true);
	});

	it('filters tags with any and all, and untagged', () => {
		const link = makeLink({ tagIds: [1] });
		expect(matchesQuery(link, { ...base, tags: [1, 2] })).toBe(true);
		expect(matchesQuery(link, { ...base, tags: [1, 2], tagMatch: 'all' })).toBe(false);
		expect(matchesQuery(link, { ...base, tags: [2] })).toBe(false);
		expect(matchesQuery(link, { ...base, untagged: true })).toBe(false);
		expect(matchesQuery(makeLink({ tagIds: [] }), { ...base, untagged: true })).toBe(true);
	});

	it('searches in the texts and the tag names', () => {
		const link = makeLink({ title: 'Grafana', tagIds: [2] });
		expect(matchesQuery(link, { ...base, q: 'GRAF' }, tagName)).toBe(true);
		expect(matchesQuery(link, { ...base, q: 'docs' }, tagName)).toBe(true);
		expect(matchesQuery(link, { ...base, q: 'docs' })).toBe(false);
		expect(matchesQuery(link, { ...base, q: 'nothing' }, tagName)).toBe(false);
	});

	it('ignores accents like the server search', () => {
		const link = makeLink({ description: 'Árvízvédelmi útmutató', tagIds: [] });
		expect(matchesQuery(link, { ...base, q: 'arvizvedelmi' })).toBe(true);
		expect(matchesQuery(link, { ...base, q: 'ÚTMUTATÓ' })).toBe(true);
		expect(matchesQuery(makeLink({ title: 'arviz' }), { ...base, q: 'árvíz' })).toBe(true);
	});
});

describe('hasActiveFilters', () => {
	it('counts search, tags, untagged, scope and archived-only', () => {
		const base = defaultQuery(defaults);
		expect(hasActiveFilters(base)).toBe(false);
		expect(hasActiveFilters({ ...base, archived: 'include' })).toBe(false);
		expect(hasActiveFilters({ ...base, q: ' ' })).toBe(false);
		expect(hasActiveFilters({ ...base, q: 'x' })).toBe(true);
		expect(hasActiveFilters({ ...base, tags: [1] })).toBe(true);
		expect(hasActiveFilters({ ...base, untagged: true })).toBe(true);
		expect(hasActiveFilters({ ...base, scope: 'own' })).toBe(true);
		expect(hasActiveFilters({ ...base, archived: 'only' })).toBe(true);
	});
});
