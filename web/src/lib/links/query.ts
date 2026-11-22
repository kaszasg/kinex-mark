// LinkQuery <-> URL search parameters of the links page (plan 8.6).
// The URL holds only values that differ from the defaults; invalid values are dropped.
import type { LinkItem, LinkQuery, Preferences } from '$lib/api/types.js';
import { cleanQueryText } from '$lib/utils/text.js';
import { foldText } from './fold.js';

export type Scope = NonNullable<LinkQuery['scope']>;
export type ArchivedMode = NonNullable<LinkQuery['archived']>;
export type SortMode = NonNullable<LinkQuery['sort']>;
export type TagMatch = NonNullable<LinkQuery['tagMatch']>;

/** A fully specified query (every filter present), the state of the links page. */
export type ListQuery = {
	q: string;
	tags: number[];
	tagMatch: TagMatch;
	untagged: boolean;
	scope: Scope;
	archived: ArchivedMode;
	sort: SortMode;
};

/** Defaults that come from the preferences; the URL omits them. */
export type QueryDefaults = { sort: SortMode; tagMatch: TagMatch; archived: ArchivedMode };
/** URL search parameters as read (also the read-only kind of $app/state). */
export type ReadableParams = Pick<URLSearchParams, 'get'>;

export const SCOPES: readonly Scope[] = ['all', 'own', 'shared'];
export const ARCHIVED_MODES: readonly ArchivedMode[] = ['exclude', 'include', 'only'];
export const SORTS: readonly SortMode[] = ['usage', 'created', 'title'];
export const TAG_MATCHES: readonly TagMatch[] = ['any', 'all'];
/** The server's limits (plan 5.9.1). */
export const MAX_FILTER_TAGS = 20;
export const MAX_QUERY_LENGTH = 200;

export const FALLBACK_DEFAULTS: QueryDefaults = {
	sort: 'usage',
	tagMatch: 'any',
	archived: 'exclude'
};

export function defaultsFromPreferences(prefs: Preferences | null | undefined): QueryDefaults {
	if (!prefs) return FALLBACK_DEFAULTS;
	return {
		sort: prefs.sort,
		tagMatch: prefs.tagMatch,
		archived: prefs.showArchived ? 'include' : 'exclude'
	};
}

export function defaultQuery(defaults: QueryDefaults = FALLBACK_DEFAULTS): ListQuery {
	return {
		q: '',
		tags: [],
		tagMatch: defaults.tagMatch,
		untagged: false,
		scope: 'all',
		archived: defaults.archived,
		sort: defaults.sort
	};
}

function oneOf<T extends string>(value: string | null, allowed: readonly T[]): T | undefined {
	return value !== null && (allowed as readonly string[]).includes(value)
		? (value as T)
		: undefined;
}

function parseTags(value: string | null): number[] {
	if (!value) return [];
	const ids: number[] = [];
	for (const part of value.split(',')) {
		const trimmed = part.trim();
		if (!/^\d+$/.test(trimmed)) continue;
		const id = Number(trimmed);
		if (id > 0 && Number.isSafeInteger(id) && !ids.includes(id)) ids.push(id);
	}
	return ids.slice(0, MAX_FILTER_TAGS);
}

/**
 * The search text as the server accepts it (plan 5.9.1 and 5.1): 0-200 runes without control or
 * bidi runes. A pasted tab would otherwise fail the whole list load with invalid_query.
 */
export function searchText(q: string): string {
	return cleanQueryText(q, MAX_QUERY_LENGTH);
}

export function queryFromParams(params: ReadableParams, defaults: QueryDefaults): ListQuery {
	const query = defaultQuery(defaults);
	query.q = searchText(params.get('q') ?? '');
	query.tags = parseTags(params.get('tags'));
	query.tagMatch = oneOf(params.get('match'), TAG_MATCHES) ?? defaults.tagMatch;
	const untagged = params.get('untagged');
	// "untagged" and "tags" exclude each other (invalid_query): the tags win.
	query.untagged = query.tags.length === 0 && (untagged === 'true' || untagged === '1');
	query.scope = oneOf(params.get('scope'), SCOPES) ?? 'all';
	query.archived = oneOf(params.get('archived'), ARCHIVED_MODES) ?? defaults.archived;
	query.sort = oneOf(params.get('sort'), SORTS) ?? defaults.sort;
	return query;
}

/** Search parameters with only the non-default values, plus the open details sheet. */
export function queryToParams(
	query: ListQuery,
	defaults: QueryDefaults,
	linkId: number | null = null
): URLSearchParams {
	const params = new URLSearchParams();
	const q = searchText(query.q);
	if (q) params.set('q', q);
	if (query.tags.length > 0) params.set('tags', query.tags.join(','));
	else if (query.untagged) params.set('untagged', 'true');
	if (query.tagMatch !== defaults.tagMatch) params.set('match', query.tagMatch);
	if (query.scope !== 'all') params.set('scope', query.scope);
	if (query.archived !== defaults.archived) params.set('archived', query.archived);
	if (query.sort !== defaults.sort) params.set('sort', query.sort);
	if (linkId !== null) params.set('link', String(linkId));
	return params;
}

/** The links page URL for a query. */
export function queryHref(
	query: ListQuery,
	defaults: QueryDefaults,
	linkId: number | null = null
): string {
	const search = queryToParams(query, defaults, linkId).toString();
	return search ? `/?${search}` : '/';
}

/** The open details sheet: ?link=<id>. */
export function linkIdFromParams(params: ReadableParams): number | null {
	const raw = params.get('link');
	if (!raw || !/^\d+$/.test(raw)) return null;
	const id = Number(raw);
	return id > 0 && Number.isSafeInteger(id) ? id : null;
}

/** The API query of a list query (the API module maps it to the 5.9.1 parameters). */
export function toLinkQuery(query: ListQuery): LinkQuery {
	return {
		q: searchText(query.q) || undefined,
		tags: query.tags.length > 0 ? query.tags : undefined,
		tagMatch: query.tagMatch,
		untagged: query.tags.length === 0 && query.untagged ? true : undefined,
		scope: query.scope,
		archived: query.archived,
		sort: query.sort
	};
}

export function sameQuery(a: ListQuery, b: ListQuery): boolean {
	return (
		a.q === b.q &&
		a.tagMatch === b.tagMatch &&
		a.untagged === b.untagged &&
		a.scope === b.scope &&
		a.archived === b.archived &&
		a.sort === b.sort &&
		a.tags.length === b.tags.length &&
		a.tags.every((id, index) => b.tags[index] === id)
	);
}

type Matchable = Pick<
	LinkItem,
	'url' | 'host' | 'title' | 'description' | 'tagIds' | 'isOwner' | 'archivedAt'
>;

/**
 * Whether a link belongs to the filtered set, evaluated in the browser after a capture or an
 * optimistic change. Mirrors the server filter of plan 4.8 (search in the texts and the tag names).
 */
export function matchesQuery(
	link: Matchable,
	query: ListQuery,
	tagName: (id: number) => string | undefined = () => undefined
): boolean {
	if (query.archived === 'exclude' && link.archivedAt !== null) return false;
	if (query.archived === 'only' && link.archivedAt === null) return false;
	if (query.scope === 'own' && !link.isOwner) return false;
	if (query.scope === 'shared' && link.isOwner) return false;
	if (query.tags.length > 0) {
		const hit =
			query.tagMatch === 'all'
				? query.tags.every((id) => link.tagIds.includes(id))
				: query.tags.some((id) => link.tagIds.includes(id));
		if (!hit) return false;
	} else if (query.untagged && link.tagIds.length > 0) {
		return false;
	}
	const needle = foldText(searchText(query.q));
	if (needle) {
		const names = link.tagIds.map((id) => tagName(id) ?? '');
		const haystack = foldText(
			[link.title, link.description, link.url, link.host, ...names].join('\n')
		);
		if (!haystack.includes(needle)) return false;
	}
	return true;
}

/** A filter narrows the collection: search, tags, untagged, scope or archived-only. */
export function hasActiveFilters(query: ListQuery): boolean {
	return (
		searchText(query.q) !== '' ||
		query.tags.length > 0 ||
		query.untagged ||
		query.scope !== 'all' ||
		query.archived === 'only'
	);
}
