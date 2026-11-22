// Client-side ordering (plan 8.7.3): an exact copy of the ORDER BY of plan 4.8, used only to
// reorder the loaded window after an optimistic change.
import type { LinkItem, LinkQuery } from '$lib/api/types.js';
import { foldText } from './fold.js';

export type LinkSort = NonNullable<LinkQuery['sort']>;
type Orderable = Pick<
	LinkItem,
	'id' | 'pinned' | 'useCount' | 'lastUsedAt' | 'createdAt' | 'description' | 'title' | 'host'
>;

/** Compares by Unicode code point, like the server's byte order on UTF-8 text. */
export function compareCodePoints(a: string, b: string): number {
	const left = [...a];
	const right = [...b];
	const length = Math.min(left.length, right.length);
	for (let i = 0; i < length; i++) {
		const diff = (left[i].codePointAt(0) ?? 0) - (right[i].codePointAt(0) ?? 0);
		if (diff !== 0) return diff;
	}
	return left.length - right.length;
}

/** The server's sort_label: description, otherwise title, otherwise host, folded (foldText). */
export function sortLabel(link: Pick<LinkItem, 'description' | 'title' | 'host'>): string {
	return foldText(link.description || link.title || link.host);
}

function time(value: string | null): number | null {
	if (value === null) return null;
	const ms = Date.parse(value);
	return Number.isNaN(ms) ? null : ms;
}

/** Descending time order with missing values last. */
function compareTimeDesc(a: string | null, b: string | null): number {
	const left = time(a);
	const right = time(b);
	if (left === right) return 0;
	if (left === null) return 1;
	if (right === null) return -1;
	return right - left;
}

export function compareLinks(a: Orderable, b: Orderable, sort: LinkSort): number {
	if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
	if (sort === 'usage') {
		if (a.useCount !== b.useCount) return b.useCount - a.useCount;
		const used = compareTimeDesc(a.lastUsedAt, b.lastUsedAt);
		if (used !== 0) return used;
	} else if (sort === 'title') {
		const label = compareCodePoints(sortLabel(a), sortLabel(b));
		if (label !== 0) return label;
	}
	const created = compareTimeDesc(a.createdAt, b.createdAt);
	if (created !== 0) return created;
	return b.id - a.id;
}

/** A sorted copy. */
export function sortLinks<T extends Orderable>(items: readonly T[], sort: LinkSort): T[] {
	return [...items].sort((a, b) => compareLinks(a, b, sort));
}
