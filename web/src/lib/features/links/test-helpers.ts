// Test helpers of the links feature specs: floating content, user events, stores and fixtures.
import { screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import type { LinkDetail, LinkItem, LinkPage } from '$lib/api/types.js';
import { LinksStore } from '$lib/state/links.svelte.js';
import { T0 } from '$lib/test/fixtures.js';

export const user = () => userEvent.setup({ pointerEventsCheck: 0 });

/**
 * Floating content is visibility:hidden in jsdom (no layout), so its text does not count as an
 * accessible name: floating items are found by role and text instead.
 */
export function queryFloating(role: string, text: string): HTMLElement | null {
	return (
		screen.queryAllByRole(role, { hidden: true }).find((el) => el.textContent?.trim() === text) ??
		null
	);
}

export function getFloating(role: string, text: string): HTMLElement {
	const el = queryFloating(role, text);
	if (!el) throw new Error(`no ${role} with text "${text}"`);
	return el;
}

export function findFloating(role: string, text: string): Promise<HTMLElement> {
	return vi.waitFor(() => getFloating(role, text));
}

/** A links store with stub tag and counter dependencies. */
export function makeStore(tagNames: Record<number, string> = {}) {
	const linkStats = { refresh: vi.fn(async () => {}) };
	const tags = {
		byId: new Map(Object.entries(tagNames).map(([id, name]) => [Number(id), { name }])),
		onTagsChanged: () => () => {},
		refresh: vi.fn(async () => {})
	};
	return { store: new LinksStore({ tags, linkStats }), linkStats, tags };
}

export function makeDetail(link: LinkItem, patch: Partial<LinkDetail> = {}): LinkDetail {
	return {
		...link,
		titleSource: 'fetched',
		tags: [],
		fetchError: null,
		fetchedAt: T0,
		fetchAttempts: 1,
		shares: [],
		...patch
	};
}

export function makePage(
	items: LinkItem[],
	total = items.length,
	counts = { pinned: items.filter((link) => link.pinned).length, archived: 0 }
): LinkPage {
	return { items, total, limit: 200, offset: 0, counts } as LinkPage;
}
