import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { goto } from '$app/navigation';
import type { LinkDetail, LinkItem, LinkPage } from '$lib/api/types.js';
import { sortLinks } from '$lib/links/order.js';
import { FALLBACK_DEFAULTS } from '$lib/links/query.js';
import { makeLink, makeMe } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, type MockRequest } from '$lib/test/mock-api.js';
import {
	LEAVE_DELAY_MS,
	LinksStore,
	PAGE_SIZE,
	PENDING_MAX_MS,
	PENDING_POLL_MS,
	SEARCH_DEBOUNCE_MS,
	links,
	toApiError,
	toLinkItem
} from './links.svelte.js';
import { session } from './session.svelte.js';

function page(items: LinkItem[], total = items.length, pinned = 0, archived = 0): LinkPage {
	return { items, total, limit: PAGE_SIZE, offset: 0, counts: { pinned, archived } } as LinkPage;
}

function detail(link: LinkItem, patch: Partial<LinkDetail> = {}): LinkDetail {
	return {
		...link,
		titleSource: 'fetched',
		tags: [],
		fetchError: null,
		fetchedAt: null,
		fetchAttempts: 1,
		shares: [],
		...patch
	};
}

function setup() {
	let tagListener: (() => void) | null = null;
	const tags = {
		byId: new Map<number, { name: string }>([[7, { name: 'infra' }]]),
		onTagsChanged(listener: () => void) {
			tagListener = listener;
			return () => {};
		},
		refresh: vi.fn(async () => {})
	};
	const linkStats = { refresh: vi.fn(async () => {}) };
	const store = new LinksStore({ tags, linkStats });
	return { store, linkStats, tags, fireTagsChanged: () => tagListener?.() };
}

async function flush(): Promise<void> {
	for (let i = 0; i < 5; i++) await Promise.resolve();
	await new Promise((resolve) => setTimeout(resolve, 0));
}

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
});

describe('helpers', () => {
	it('toLinkItem drops the detail-only fields', () => {
		const link = makeLink();
		expect(toLinkItem(detail(link))).toEqual(link);
	});

	it('toApiError wraps unknown errors as network errors', () => {
		expect(toApiError(new Error('x')).code).toBe('network_error');
	});
});

describe('LinksStore loading', () => {
	it('syncs the URL into the query and loads the first page', async () => {
		const items = [makeLink({ pinned: true }), makeLink()];
		const api = mockApi({ 'GET /api/links': json(page(items, 2, 1, 0)) });
		const { store } = setup();
		store.sync(new URLSearchParams('q=graf&tags=7&scope=own&link=9'), FALLBACK_DEFAULTS);
		expect(store.status).toBe('loading');
		expect(store.detailsId).toBe(9);
		await flush();
		expect(store.status).toBe('ready');
		expect(store.items).toHaveLength(2);
		expect(store.pinned).toHaveLength(1);
		expect(store.others).toHaveLength(1);
		expect(store.counts).toEqual({ pinned: 1, archived: 0 });
		const url = api.calls('GET /api/links')[0].url;
		expect(url.searchParams.get('q')).toBe('graf');
		expect(url.searchParams.get('tags')).toBe('7');
		expect(url.searchParams.get('scope')).toBe('own');
		expect(url.searchParams.get('limit')).toBe(String(PAGE_SIZE));
		expect(url.searchParams.get('offset')).toBe('0');
	});

	it('does not reload when the synced query is unchanged', async () => {
		const api = mockApi({ 'GET /api/links': json(page([])) });
		const { store } = setup();
		store.sync(new URLSearchParams('q=a'), FALLBACK_DEFAULTS);
		await flush();
		store.sync(new URLSearchParams('q=a&link=3'), FALLBACK_DEFAULTS);
		await flush();
		expect(api.calls('GET /api/links')).toHaveLength(1);
		expect(store.detailsId).toBe(3);
		store.sync(new URLSearchParams('q=b'), FALLBACK_DEFAULTS);
		await flush();
		expect(api.calls('GET /api/links')).toHaveLength(2);
	});

	it('keeps only the newest response', async () => {
		const first = makeLink({ title: 'first' });
		const second = makeLink({ title: 'second' });
		let release: (() => void) | null = null;
		mockApi({
			'GET /api/links': (req: MockRequest) => {
				if (req.url.searchParams.get('q') === 'slow') {
					return new Promise((resolve) => {
						release = () => resolve(json(page([first])));
					});
				}
				return json(page([second]));
			}
		});
		const { store } = setup();
		store.sync(new URLSearchParams('q=slow'), FALLBACK_DEFAULTS);
		await flush();
		store.sync(new URLSearchParams('q=fast'), FALLBACK_DEFAULTS);
		await flush();
		release!();
		await flush();
		expect(store.items.map((link) => link.title)).toEqual(['second']);
	});

	it('reports a load error', async () => {
		mockApi({ 'GET /api/links': apiError(500, 'internal_error') });
		const { store } = setup();
		await store.reload();
		expect(store.status).toBe('error');
		expect(store.error?.code).toBe('internal_error');
	});

	it('appends the next page without duplicates and survives a failure', async () => {
		// Equal creation times: the server orders by id, descending (plan 4.8).
		const a = makeLink({ id: 3 });
		const b = makeLink({ id: 2 });
		const c = makeLink({ id: 1 });
		const api = mockApi({
			'GET /api/links': (req: MockRequest) =>
				req.url.searchParams.get('offset') === '0' ? json(page([a, b], 3)) : json(page([b, c], 3))
		});
		const { store } = setup();
		await store.reload();
		await store.loadMore();
		expect(api.calls('GET /api/links')[1].url.searchParams.get('offset')).toBe('2');
		expect(store.items.map((link) => link.id)).toEqual([3, 2, 1]);
		await store.loadMore();
		expect(api.calls('GET /api/links')).toHaveLength(2);

		api.route('GET /api/links', apiError(500, 'internal_error'));
		store.total = 10;
		await store.loadMore();
		expect(store.status).toBe('ready');
	});

	it('loadMore after an unpin still loads every link and completes', async () => {
		// A server that sorts like plan 4.8 over 205 links; link 1 (the oldest) is pinned.
		const server = Array.from({ length: PAGE_SIZE + 5 }, (_, index) =>
			makeLink({
				id: index + 1,
				pinned: index === 0,
				createdAt: new Date(Date.UTC(2026, 0, 1, 0, index)).toISOString()
			})
		);
		const api = mockApi({
			'GET /api/links': (req: MockRequest) => {
				const offset = Number(req.url.searchParams.get('offset'));
				const limit = Number(req.url.searchParams.get('limit'));
				const sorted = sortLinks(server, 'usage');
				return json(page(sorted.slice(offset, offset + limit), sorted.length));
			},
			'DELETE /api/links/1/pin': () => {
				server[0] = { ...server[0], pinned: false };
				return json(server[0]);
			}
		});
		const { store } = setup();
		await store.reload();
		expect(store.items).toHaveLength(PAGE_SIZE);
		expect(store.items[0].id).toBe(1);
		await store.pin(1, false);
		await store.loadMore();
		expect(api.calls('GET /api/links')[1].url.searchParams.get('offset')).toBe(
			String(PAGE_SIZE - 1)
		);
		expect(store.items).toHaveLength(PAGE_SIZE + 5);
		expect(store.items.length).toBe(store.total);
		expect(store.items.map((link) => link.id)).toEqual(
			sortLinks(server, 'usage').map((link) => link.id)
		);
	});

	it('loadMore counts a locally inserted link that sorts behind the window', async () => {
		const server = Array.from({ length: 4 }, (_, index) =>
			makeLink({ id: index + 1, createdAt: new Date(Date.UTC(2026, 0, 2, 0, index)).toISOString() })
		);
		const api = mockApi({
			'GET /api/links': (req: MockRequest) => {
				const offset = Number(req.url.searchParams.get('offset'));
				const sorted = sortLinks(server, 'usage');
				return json(page(sorted.slice(offset, offset === 0 ? 2 : undefined), sorted.length));
			}
		});
		const { store } = setup();
		await store.reload();
		expect(store.items.map((link) => link.id)).toEqual([4, 3]);
		const old = makeLink({ id: 9, createdAt: '2025-01-01T00:00:00Z' });
		server.push(old);
		store.upsert(old);
		await store.loadMore();
		expect(api.calls('GET /api/links')[1].url.searchParams.get('offset')).toBe('2');
		expect(store.items.map((link) => link.id)).toEqual([4, 3, 2, 1, 9]);
	});

	it('reloads after a tag change once it has loaded', async () => {
		const api = mockApi({ 'GET /api/links': json(page([])) });
		const { store, fireTagsChanged } = setup();
		fireTagsChanged();
		expect(api.calls('GET /api/links')).toHaveLength(0);
		await store.reload();
		fireTagsChanged();
		await flush();
		expect(api.calls('GET /api/links')).toHaveLength(2);
	});
});

describe('LinksStore query changes', () => {
	beforeEach(() => vi.mocked(goto).mockClear());

	it('writes the URL and debounces a search-only change', async () => {
		vi.useFakeTimers();
		const api = mockApi({ 'GET /api/links': json(page([])) });
		const { store } = setup();
		store.setQuery({ q: 'a' });
		store.setQuery({ q: 'ab' });
		expect(vi.mocked(goto)).toHaveBeenLastCalledWith('/?q=ab', { replace: true, reset: false });
		expect(api.calls('GET /api/links')).toHaveLength(0);
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);
		expect(api.calls('GET /api/links')).toHaveLength(1);
		expect(api.calls('GET /api/links')[0].url.searchParams.get('q')).toBe('ab');
	});

	it('reloads other changes at once and ignores no-op changes', async () => {
		const api = mockApi({ 'GET /api/links': json(page([])) });
		const { store } = setup();
		store.setQuery({ scope: 'shared' });
		await flush();
		expect(api.calls('GET /api/links')).toHaveLength(1);
		store.setQuery({ scope: 'shared' });
		await flush();
		expect(api.calls('GET /api/links')).toHaveLength(1);
		store.setQuery({ untagged: true });
		store.setQuery({ tags: [7] });
		expect(store.query.untagged).toBe(false);
	});

	it('clears the filters and keeps the preferred archived mode', async () => {
		mockApi({ 'GET /api/links': json(page([])) });
		const { store } = setup();
		store.defaults = { ...FALLBACK_DEFAULTS, archived: 'include' };
		store.setQuery({ q: 'x', tags: [7], scope: 'own', archived: 'only' });
		store.clearFilters();
		expect(store.query).toMatchObject({
			q: '',
			tags: [],
			scope: 'all',
			archived: 'include'
		});
		store.setQuery({ archived: 'exclude' });
		store.clearFilters();
		expect(store.query.archived).toBe('exclude');
	});

	it('opens and closes the details sheet through the URL', () => {
		const { store } = setup();
		store.openDetails(5);
		expect(vi.mocked(goto)).toHaveBeenLastCalledWith('/?link=5', { replace: true, reset: false });
		store.closeDetails();
		expect(store.detailsId).toBeNull();
		expect(vi.mocked(goto)).toHaveBeenLastCalledWith('/', { replace: true, reset: false });
		const calls = vi.mocked(goto).mock.calls.length;
		store.closeDetails();
		expect(vi.mocked(goto).mock.calls.length).toBe(calls);
	});
});

describe('LinksStore list changes', () => {
	async function loaded(items: LinkItem[], counts = { pinned: 0, archived: 0 }) {
		mockApi({
			'GET /api/links': json(page(items, items.length, counts.pinned, counts.archived))
		});
		const ctx = setup();
		await ctx.store.reload();
		return ctx;
	}

	it('upserts: replaces, inserts matching links with highlight, skips hidden ones', async () => {
		const existing = makeLink({ id: 1, title: 'old' });
		const { store } = await loaded([existing]);
		expect(store.upsert({ ...existing, title: 'new' })).toBe(true);
		expect(store.find(1)?.title).toBe('new');
		expect(store.upsert(makeLink({ id: 2, pinned: true }), { highlight: true })).toBe(true);
		expect(store.items[0].id).toBe(2);
		expect(store.highlightId).toBe(2);
		expect(store.total).toBe(2);
		expect(store.counts.pinned).toBe(1);
		expect(store.upsert(makeLink({ id: 3, archivedAt: '2026-01-01T00:00:00Z' }))).toBe(false);
		expect(store.find(3)).toBeUndefined();
	});

	it('removes a link and its highlight', async () => {
		const link = makeLink({ id: 1, pinned: true });
		const { store } = await loaded([link], { pinned: 1, archived: 0 });
		store.highlight(1);
		store.remove(1);
		store.remove(99);
		expect(store.items).toHaveLength(0);
		expect(store.counts.pinned).toBe(0);
		expect(store.highlightId).toBeNull();
	});

	it('pins optimistically and keeps the server answer', async () => {
		const a = makeLink({ id: 1, useCount: 5 });
		const b = makeLink({ id: 2, useCount: 1 });
		const { store } = await loaded([a, b]);
		let resolve: ((value: unknown) => void) | null = null;
		mockApi({
			'POST /api/links/2/pin': () =>
				new Promise((done) => {
					resolve = () => done(json({ ...b, pinned: true, useCount: 2 }));
				})
		});
		const pending = store.pin(2, true);
		expect(store.items[0].id).toBe(2);
		expect(store.items[0].pinned).toBe(true);
		resolve!(null);
		const result = await pending;
		expect(result?.useCount).toBe(2);
		expect(store.find(2)?.useCount).toBe(2);
	});

	it('reverts a failed pin', async () => {
		const link = makeLink({ id: 1 });
		const { store } = await loaded([link]);
		mockApi({ 'POST /api/links/1/pin': apiError(500, 'internal_error') });
		expect(await store.pin(1, true)).toBeNull();
		expect(store.find(1)?.pinned).toBe(false);
	});

	it('pinning an archived link restores it and refreshes the counters', async () => {
		const link = makeLink({ id: 1, archivedAt: '2026-01-01T00:00:00Z' });
		mockApi({ 'GET /api/links': json(page([link])) });
		const { store, linkStats, tags } = setup();
		store.setQuery({ archived: 'include' });
		await flush();
		mockApi({ 'POST /api/links/1/pin': json({ ...link, pinned: true, archivedAt: null }) });
		const pending = store.pin(1, true);
		expect(store.find(1)?.archivedAt).toBeNull();
		await pending;
		expect(linkStats.refresh).toHaveBeenCalled();
		expect(tags.refresh).toHaveBeenCalled();
		mockApi({ 'DELETE /api/links/1/pin': json({ ...link, pinned: false, archivedAt: null }) });
		expect((await store.pin(1, false))?.pinned).toBe(false);
	});

	it('archives: the link leaves the list after the delay unless archived links show', async () => {
		vi.useFakeTimers();
		const link = makeLink({ id: 1 });
		const { store, linkStats, tags } = await loaded([link]);
		const archived = { ...link, archivedAt: '2026-01-02T00:00:00Z', archiveReason: 'manual' };
		mockApi({ 'POST /api/links/1/archive': json(archived) });
		const result = await store.archive(1, true);
		expect(result?.archivedAt).not.toBeNull();
		expect(store.find(1)).toBeDefined();
		await vi.advanceTimersByTimeAsync(LEAVE_DELAY_MS);
		expect(store.find(1)).toBeUndefined();
		expect(linkStats.refresh).toHaveBeenCalled();
		// The sidebar's frequent tags count active links only.
		expect(tags.refresh).toHaveBeenCalled();
	});

	it("the undo after the leave puts the restored link back (plan 8.6, archived 'exclude')", async () => {
		vi.useFakeTimers();
		const kept = makeLink({ id: 5, createdAt: '2026-01-01T00:00:00Z' });
		const link = makeLink({ id: 6, createdAt: '2026-01-02T00:00:00Z' });
		const { store } = await loaded([link, kept]);
		mockApi({
			'POST /api/links/6/archive': json({
				...link,
				archivedAt: '2026-01-03T00:00:00Z',
				archiveReason: 'manual'
			}),
			'DELETE /api/links/6/archive': json({ ...link, archivedAt: null, archiveReason: null })
		});
		await store.archive(6, true);
		await vi.advanceTimersByTimeAsync(LEAVE_DELAY_MS);
		expect(store.items.map((item) => item.id)).toEqual([5]);
		expect(store.total).toBe(1);
		const restored = await store.archive(6, false);
		expect(restored?.archivedAt).toBeNull();
		expect(store.items.map((item) => item.id)).toEqual([6, 5]);
		expect(store.total).toBe(2);
		await vi.advanceTimersByTimeAsync(LEAVE_DELAY_MS);
		expect(store.find(6)).toBeDefined();
	});

	it('a server answer for a link outside the filter does not enter the list', async () => {
		const { store } = await loaded([]);
		mockApi({
			'POST /api/links/8/archive': json(makeLink({ id: 8, archivedAt: '2026-01-03T00:00:00Z' }))
		});
		await store.archive(8, true);
		expect(store.items).toHaveLength(0);
		expect(store.total).toBe(0);
	});

	it('restores a failed archive, also after the link left', async () => {
		vi.useFakeTimers();
		const link = makeLink({ id: 1 });
		const { store } = await loaded([link]);
		let fail: (() => void) | null = null;
		mockApi({
			'POST /api/links/1/archive': () =>
				new Promise((done) => {
					fail = () => done(apiError(500, 'internal_error'));
				})
		});
		const pending = store.archive(1, true);
		await vi.advanceTimersByTimeAsync(LEAVE_DELAY_MS);
		expect(store.find(1)).toBeUndefined();
		fail!();
		expect(await pending).toBeNull();
		expect(store.find(1)?.archivedAt).toBeNull();
	});

	it('restores an archived link', async () => {
		const link = makeLink({ id: 1, archivedAt: '2026-01-01T00:00:00Z' });
		mockApi({ 'GET /api/links': json(page([link], 1, 0, 1)) });
		const { store } = setup();
		store.setQuery({ archived: 'only' });
		await flush();
		mockApi({ 'DELETE /api/links/1/archive': json({ ...link, archivedAt: null }) });
		const result = await store.archive(1, false);
		expect(result?.archivedAt).toBeNull();
		expect(store.counts.archived).toBe(0);
	});

	it('counts a visit optimistically and applies the server state', async () => {
		const link = makeLink({ id: 1, useCount: 2, archivedAt: '2026-01-01T00:00:00Z' });
		mockApi({ 'GET /api/links': json(page([link])) });
		const { store, linkStats } = setup();
		store.setQuery({ archived: 'include' });
		await flush();
		const api = mockApi({
			'POST /api/links/1/visit': json({
				pinned: false,
				useCount: 9,
				lastUsedAt: '2026-02-01T00:00:00Z',
				archivedAt: null,
				archiveReason: null,
				inactiveSince: '2026-02-01T00:00:00Z',
				archiveDueAt: null
			})
		});
		store.visit(1);
		expect(store.find(1)?.useCount).toBe(3);
		expect(store.find(1)?.archivedAt).toBeNull();
		await flush();
		expect(store.find(1)?.useCount).toBe(9);
		expect(api.calls('POST /api/links/1/visit')[0].init.keepalive).toBe(true);
		expect(linkStats.refresh).toHaveBeenCalled();
	});

	it('reverts a failed visit', async () => {
		const link = makeLink({ id: 1, useCount: 2 });
		const { store } = await loaded([link]);
		mockApi({ 'POST /api/links/1/visit': apiError(500, 'internal_error') });
		store.visit(1);
		await flush();
		expect(store.find(1)?.useCount).toBe(2);
	});

	it('visits a link outside the list without changing the list', async () => {
		const { store } = await loaded([]);
		const api = mockApi({ 'POST /api/links/5/visit': json({ useCount: 1 }) });
		store.visit(5);
		await flush();
		expect(api.calls('POST /api/links/5/visit')).toHaveLength(1);
		expect(store.items).toHaveLength(0);
	});
});

describe('LinksStore pending metadata', () => {
	it('polls until the metadata arrives', async () => {
		vi.useFakeTimers();
		const link = makeLink({ id: 1, fetchStatus: 'pending' });
		mockApi({ 'GET /api/links': json(page([link])) });
		const { store } = setup();
		await store.reload();
		let ready = false;
		const api = mockApi({
			'GET /api/links/1': () =>
				json(detail(link, ready ? { fetchStatus: 'ok', title: 'Fetched' } : {}))
		});
		store.watchPending(1);
		store.watchPending(1);
		expect(store.watching).toEqual([1]);
		await vi.advanceTimersByTimeAsync(PENDING_POLL_MS);
		expect(api.calls('GET /api/links/1')).toHaveLength(1);
		expect(store.watching).toEqual([1]);
		ready = true;
		await vi.advanceTimersByTimeAsync(PENDING_POLL_MS);
		expect(store.find(1)?.title).toBe('Fetched');
		expect(store.lastDetail?.fetchStatus).toBe('ok');
		expect(store.watching).toEqual([]);
		await vi.advanceTimersByTimeAsync(PENDING_POLL_MS * 3);
		expect(api.calls('GET /api/links/1')).toHaveLength(2);
	});

	it('polls at most five links per tick, round robin', async () => {
		vi.useFakeTimers();
		const ids = [1, 2, 3, 4, 5, 6, 7];
		const routes: Record<string, ReturnType<typeof json>> = {};
		for (const id of ids) {
			routes[`GET /api/links/${id}`] = json(detail(makeLink({ id, fetchStatus: 'pending' })));
		}
		const api = mockApi(routes);
		const { store } = setup();
		for (const id of ids) store.watchPending(id);
		await vi.advanceTimersByTimeAsync(PENDING_POLL_MS);
		expect(api.requests).toHaveLength(5);
		await vi.advanceTimersByTimeAsync(PENDING_POLL_MS);
		const second = api.requests.slice(5).map((req) => req.path);
		expect(second).toContain('/api/links/6');
		expect(second).toContain('/api/links/7');
	});

	it('drops a deleted link and gives up after the time limit', async () => {
		vi.useFakeTimers();
		mockApi({
			'GET /api/links/1': apiError(404, 'not_found'),
			'GET /api/links/2': json(detail(makeLink({ id: 2, fetchStatus: 'pending' })))
		});
		const { store } = setup();
		store.watchPending(1);
		store.watchPending(2);
		await vi.advanceTimersByTimeAsync(PENDING_POLL_MS);
		expect(store.watching).toEqual([2]);
		await vi.advanceTimersByTimeAsync(PENDING_MAX_MS);
		expect(store.watching).toEqual([]);
	});
});

describe('LinksStore misc', () => {
	it('highlights, requests edits, announces and resets', () => {
		const { store } = setup();
		store.highlight(3);
		store.clearHighlight(4);
		expect(store.highlightId).toBe(3);
		store.clearHighlight(3);
		expect(store.highlightId).toBeNull();
		const link = makeLink();
		store.requestEdit(detail(link));
		expect(store.editRequest).toEqual(link);
		store.announce('hello');
		expect(store.announcement).toBe('hello');
		store.watchPending(1);
		store.reset();
		expect(store.editRequest).toBeNull();
		expect(store.announcement).toBe('');
		expect(store.watching).toEqual([]);
		expect(store.status).toBe('idle');
	});
});

describe('LinksStore and the signed-in user', () => {
	afterEach(() => {
		session.clear();
		links.reset();
	});

	it('forgets the previous user list on sign-out, so the next user loads their own', async () => {
		const customer = makeMe('customer', { id: 31 });
		const developer = makeMe('developer', { id: 32 });
		const privateLink = makeLink({ id: 1, url: 'https://example.org/kunde-privat' });
		mockApi({ 'GET /api/links': json(page([privateLink])) });
		session.setMe(customer);
		links.sync(new URLSearchParams(), FALLBACK_DEFAULTS);
		await flush();
		expect(links.items.map((link) => link.id)).toEqual([1]);
		links.openDetails(1);

		session.clear('logout');
		expect(links.items).toEqual([]);
		expect(links.total).toBe(0);
		expect(links.status).toBe('idle');
		expect(links.detailsId).toBeNull();

		// Same URL, same query: the developer's list still loads from the server.
		const ownLink = makeLink({ id: 2, isOwner: true });
		const api = mockApi({ 'GET /api/links': json(page([ownLink])) });
		session.setMe(developer);
		links.sync(new URLSearchParams(), FALLBACK_DEFAULTS);
		await flush();
		expect(api.calls('GET /api/links')).toHaveLength(1);
		expect(links.items.map((link) => link.id)).toEqual([2]);
	});

	it('also forgets the list when another user signs in without a sign-out', async () => {
		mockApi({ 'GET /api/links': json(page([makeLink({ id: 1 })])) });
		session.setMe(makeMe('developer', { id: 41 }));
		await links.reload();
		expect(links.items).toHaveLength(1);
		// A refresh of the same user keeps the list.
		session.setMe(makeMe('developer', { id: 41 }));
		expect(links.items).toHaveLength(1);
		session.setMe(makeMe('customer', { id: 42 }));
		expect(links.items).toEqual([]);
	});

	it('drops a list answer that arrives after the sign-out', async () => {
		let answer: (() => void) | null = null;
		mockApi({
			'GET /api/links': () =>
				new Promise((done) => {
					answer = () => done(json(page([makeLink({ id: 1 })])));
				})
		});
		session.setMe(makeMe('customer', { id: 51 }));
		const pending = links.reload();
		session.clear('logout');
		answer!();
		await pending;
		expect(links.items).toEqual([]);
		expect(links.status).toBe('idle');
	});
});
