// The links page store (plan 8.6): the loaded window of the filtered list, the query that the
// URL mirrors, optimistic personal-state changes and polling of pending metadata.
import { untrack } from 'svelte';
import { toast } from 'svelte-sonner';
import { goto } from '$app/navigation';
import { ApiError } from '$lib/api/errors.js';
import * as linksApi from '$lib/api/links.js';
import type { LinkDetail, LinkItem, LinkState } from '$lib/api/types.js';
import { compareLinks, sortLinks } from '$lib/links/order.js';
import {
	defaultQuery,
	FALLBACK_DEFAULTS,
	linkIdFromParams,
	matchesQuery,
	queryFromParams,
	queryHref,
	sameQuery,
	toLinkQuery,
	type ListQuery,
	type QueryDefaults,
	type ReadableParams
} from '$lib/links/query.js';
import { linkStats as linkStatsStore } from './link-stats.svelte.js';
import { session } from './session.svelte.js';
import { tags as tagsStore } from './tags.svelte.js';
import { toastError } from '$lib/utils/toast-error.js';

/** One page of the list (plan 5.9.1). */
export const PAGE_SIZE = 200;
/** Typing in the search waits this long before the list reloads. */
export const SEARCH_DEBOUNCE_MS = 200;
/** Pending metadata is polled at this interval (plan 5.9.8)... */
export const PENDING_POLL_MS = 1500;
/** ...for at most this long per link. */
export const PENDING_MAX_MS = 20_000;
/** At most this many GET /api/links/{id} requests run per poll. */
export const PENDING_BATCH = 5;
/** An item that left the filtered set (archived under "exclude") disappears after this delay. */
export const LEAVE_DELAY_MS = 300;

export type LinksStatus = 'idle' | 'loading' | 'loading-more' | 'ready' | 'error';
export type LinkCounts = { pinned: number; archived: number };

type TagSource = {
	onTagsChanged(listener: () => void): () => void;
	readonly byId: Map<number, { name: string }>;
	/** Reloads the tag counts (the sidebar's frequent tags follow them). */
	refresh(): Promise<void>;
};
type StatsSource = { refresh(): Promise<void> };
export type LinksStoreDeps = { tags?: TagSource; linkStats?: StatsSource };

/** The LinkItem fields of a LinkItem or LinkDetail: the list never keeps detail-only data. */
export function toLinkItem(link: LinkItem | LinkDetail): LinkItem {
	return {
		id: link.id,
		url: link.url,
		host: link.host,
		title: link.title,
		description: link.description,
		iconUrl: link.iconUrl,
		fetchStatus: link.fetchStatus,
		tagIds: link.tagIds,
		pinned: link.pinned,
		useCount: link.useCount,
		lastUsedAt: link.lastUsedAt,
		archivedAt: link.archivedAt,
		archiveReason: link.archiveReason,
		inactiveSince: link.inactiveSince,
		archiveDueAt: link.archiveDueAt,
		isOwner: link.isOwner,
		owner: link.owner,
		shareCount: link.shareCount,
		sharedVia: link.sharedVia,
		sharedAt: link.sharedAt,
		createdAt: link.createdAt,
		updatedAt: link.updatedAt
	};
}

function isAbort(err: unknown): boolean {
	return (err as { name?: unknown } | null)?.name === 'AbortError';
}

export function toApiError(err: unknown): ApiError {
	return err instanceof ApiError ? err : new ApiError(0, 'network_error');
}

function showError(err: unknown): void {
	toastError(err);
}

export class LinksStore {
	items = $state<LinkItem[]>([]);
	total = $state(0);
	counts = $state<LinkCounts>({ pinned: 0, archived: 0 });
	status = $state<LinksStatus>('idle');
	error = $state<ApiError | null>(null);
	query = $state<ListQuery>(defaultQuery());
	defaults = $state<QueryDefaults>(FALLBACK_DEFAULTS);
	/** The link whose details sheet is open (?link=<id>). */
	detailsId = $state<number | null>(null);
	highlightId = $state<number | null>(null);
	/** The latest LinkDetail seen by polling, for an open details sheet. */
	lastDetail = $state<LinkDetail | null>(null);
	/** A link the edit dialog should open for (quick capture toast action). */
	editRequest = $state<LinkItem | null>(null);
	/** Text for the page's polite live region. */
	announcement = $state('');

	readonly pinned = $derived(this.items.filter((link) => link.pinned));
	readonly others = $derived(this.items.filter((link) => !link.pinned));

	#tags: TagSource;
	#stats: StatsSource;
	#abort: AbortController | null = null;
	#searchTimer: ReturnType<typeof setTimeout> | null = null;
	#leaving = new Map<number, ReturnType<typeof setTimeout>>();
	#pending = new Map<number, number>();
	#pollTimer: ReturnType<typeof setInterval> | null = null;
	#polling = false;
	/**
	 * The last link of the latest server page, as the server sorted it then. Every server item up to
	 * it is loaded, so the next page starts after the loaded items that still sort before it; items
	 * that a local change (unpin, edit, insert) moved behind it are fetched again (plan 8.6).
	 */
	#boundary: LinkItem | null = null;

	constructor(deps: LinksStoreDeps = {}) {
		this.#tags = deps.tags ?? tagsStore;
		this.#stats = deps.linkStats ?? linkStatsStore;
		// Tag changes alter the tag filter and the tag ids on the links.
		this.#tags.onTagsChanged(() => {
			if (this.status !== 'idle') void this.reload();
		});
	}

	find(id: number): LinkItem | undefined {
		return this.items.find((link) => link.id === id);
	}

	/** Applies the URL of the links page: reloads when the query differs from the current one. */
	sync(params: ReadableParams, defaults: QueryDefaults): void {
		untrack(() => {
			this.defaults = defaults;
			this.detailsId = linkIdFromParams(params);
			const next = queryFromParams(params, defaults);
			if (this.status !== 'idle' && sameQuery(next, this.query)) return;
			this.query = next;
			this.#cancelSearch();
			void this.reload();
		});
	}

	/** Changes the query: the URL follows, the search text reloads debounced, the rest at once. */
	setQuery(patch: Partial<ListQuery>): void {
		const next: ListQuery = { ...this.query, ...patch };
		if (next.tags.length > 0) next.untagged = false;
		if (sameQuery(next, this.query)) return;
		const searchOnly = Object.keys(patch).every((key) => key === 'q');
		this.query = next;
		this.#writeUrl();
		this.#cancelSearch();
		if (searchOnly) {
			this.#searchTimer = setTimeout(() => {
				this.#searchTimer = null;
				void this.reload();
			}, SEARCH_DEBOUNCE_MS);
		} else {
			void this.reload();
		}
	}

	/** Clears search, tags, scope and the archived-only view. */
	clearFilters(): void {
		this.setQuery({
			q: '',
			tags: [],
			untagged: false,
			scope: 'all',
			archived: this.query.archived === 'only' ? this.defaults.archived : this.query.archived
		});
	}

	openDetails(id: number): void {
		this.detailsId = id;
		this.#writeUrl();
	}

	closeDetails(): void {
		if (this.detailsId === null) return;
		this.detailsId = null;
		this.#writeUrl();
	}

	#writeUrl(): void {
		void goto(queryHref(this.query, this.defaults, this.detailsId), {
			replace: true,
			reset: false
		});
	}

	#cancelSearch(): void {
		if (this.#searchTimer) clearTimeout(this.#searchTimer);
		this.#searchTimer = null;
	}

	/** Loads the first page of the current query; a newer call aborts the older request. */
	async reload(): Promise<void> {
		this.#abort?.abort();
		const controller = new AbortController();
		this.#abort = controller;
		this.status = 'loading';
		this.error = null;
		try {
			const page = await linksApi.listLinks(
				{ ...toLinkQuery(this.query), limit: PAGE_SIZE, offset: 0 },
				controller.signal
			);
			if (controller !== this.#abort) return;
			this.#clearLeaving();
			this.items = page.items;
			this.#boundary = page.items.at(-1) ?? null;
			this.total = page.total;
			this.counts = { ...page.counts };
			this.status = 'ready';
		} catch (err) {
			if (controller !== this.#abort || isAbort(err)) return;
			this.error = toApiError(err);
			this.status = 'error';
		}
	}

	/** Appends the next page (the list's end sentinel calls it). */
	async loadMore(): Promise<void> {
		if (this.status !== 'ready' || this.items.length >= this.total) return;
		const controller = this.#abort ?? new AbortController();
		this.#abort = controller;
		this.status = 'loading-more';
		try {
			const page = await linksApi.listLinks(
				{ ...toLinkQuery(this.query), limit: PAGE_SIZE, offset: this.#nextOffset() },
				controller.signal
			);
			if (controller !== this.#abort) return;
			const known = new Set(this.items.map((link) => link.id));
			const merged = [...this.items, ...page.items.filter((link) => !known.has(link.id))];
			this.items = sortLinks(merged, this.query.sort);
			this.#boundary = page.items.at(-1) ?? this.#boundary;
			this.total = page.total;
			this.counts = { ...page.counts };
			this.status = 'ready';
		} catch (err) {
			if (controller !== this.#abort || isAbort(err)) return;
			this.status = 'ready';
			showError(err);
		}
	}

	/** The server offset of the next page: the loaded items that sort up to the boundary. */
	#nextOffset(): number {
		const boundary = this.#boundary;
		if (!boundary) return this.items.length;
		const sort = this.query.sort;
		return this.items.filter((link) => compareLinks(link, boundary, sort) <= 0).length;
	}

	#belongs(link: LinkItem): boolean {
		return matchesQuery(link, this.query, (id) => this.#tags.byId.get(id)?.name);
	}

	#adjustCounts(before: LinkItem | null, after: LinkItem | null): void {
		const pinned = (after?.pinned ? 1 : 0) - (before?.pinned ? 1 : 0);
		const archived = (after?.archivedAt ? 1 : 0) - (before?.archivedAt ? 1 : 0);
		if (pinned !== 0 || archived !== 0) {
			this.counts = {
				pinned: Math.max(0, this.counts.pinned + pinned),
				archived: Math.max(0, this.counts.archived + archived)
			};
		}
	}

	/** Replaces a loaded item and reorders; returns the previous version. */
	#replace(next: LinkItem): LinkItem | undefined {
		const index = this.items.findIndex((link) => link.id === next.id);
		if (index < 0) return undefined;
		const before = this.items[index];
		const items = [...this.items];
		items[index] = next;
		this.items = sortLinks(items, this.query.sort);
		this.#adjustCounts(before, next);
		return before;
	}

	#insert(link: LinkItem): void {
		this.items = sortLinks([...this.items, link], this.query.sort);
		this.total += 1;
		this.#adjustCounts(null, link);
	}

	/** An item that no longer matches the filter leaves the list after a short delay. */
	#scheduleLeave(link: LinkItem): void {
		this.#cancelLeave(link.id);
		if (this.#belongs(link)) return;
		this.#leaving.set(
			link.id,
			setTimeout(() => {
				this.#leaving.delete(link.id);
				const current = this.find(link.id);
				if (current && !this.#belongs(current)) this.remove(link.id);
			}, LEAVE_DELAY_MS)
		);
	}

	#cancelLeave(id: number): void {
		const timer = this.#leaving.get(id);
		if (timer) clearTimeout(timer);
		this.#leaving.delete(id);
	}

	#clearLeaving(): void {
		for (const timer of this.#leaving.values()) clearTimeout(timer);
		this.#leaving.clear();
	}

	/**
	 * Adds or replaces a link. A new link is inserted only when it matches the current filters.
	 * Returns whether the link is in the list afterwards.
	 */
	upsert(link: LinkItem | LinkDetail, opts: { highlight?: boolean } = {}): boolean {
		const item = toLinkItem(link);
		let visible = true;
		if (this.find(item.id)) this.#replace(item);
		else if (this.#belongs(item)) this.#insert(item);
		else visible = false;
		if (visible && opts.highlight) this.highlightId = item.id;
		return visible;
	}

	remove(id: number): void {
		const before = this.find(id);
		this.#cancelLeave(id);
		if (!before) return;
		this.items = this.items.filter((link) => link.id !== id);
		this.total = Math.max(0, this.total - 1);
		this.#adjustCounts(before, null);
		if (this.highlightId === id) this.highlightId = null;
	}

	/** Puts an earlier version back after a failed request (also when it already left the list). */
	#restore(before: LinkItem): void {
		this.#cancelLeave(before.id);
		if (this.find(before.id)) this.#replace(before);
		else this.#insert(before);
	}

	/**
	 * Applies the server's answer: replaces a loaded item, or puts back one that already left the
	 * list (the toast's undo after the 300 ms leave) when it matches the filters again.
	 */
	#applyServer(next: LinkItem): void {
		if (this.find(next.id)) {
			this.#replace(next);
			this.#scheduleLeave(next);
		} else if (this.#belongs(next)) {
			this.#insert(next);
		}
	}

	/** Optimistic pin or unpin; pinning an archived link restores it (plan 4.9). */
	async pin(id: number, pinned: boolean): Promise<LinkItem | null> {
		const before = this.find(id);
		if (before) {
			const optimistic: LinkItem = pinned
				? { ...before, pinned, archivedAt: null, archiveReason: null, archiveDueAt: null }
				: { ...before, pinned };
			this.#replace(optimistic);
			this.#scheduleLeave(optimistic);
		}
		try {
			const result = await (pinned ? linksApi.pin(id) : linksApi.unpin(id));
			this.#applyServer(result);
			if (before?.archivedAt && !result.archivedAt) this.refreshCounts();
			return result;
		} catch (err) {
			if (before) this.#restore(before);
			showError(err);
			return null;
		}
	}

	/** Optimistic archive or restore; under "exclude" the archived item leaves after 300 ms. */
	async archive(id: number, archived: boolean): Promise<LinkItem | null> {
		const before = this.find(id);
		if (before) {
			const optimistic: LinkItem = archived
				? {
						...before,
						pinned: false,
						archivedAt: new Date().toISOString(),
						archiveReason: 'manual',
						archiveDueAt: null
					}
				: { ...before, archivedAt: null, archiveReason: null };
			this.#replace(optimistic);
			this.#scheduleLeave(optimistic);
		}
		try {
			const result = await (archived ? linksApi.archive(id) : linksApi.restore(id));
			this.#applyServer(result);
			this.refreshCounts();
			return result;
		} catch (err) {
			if (before) this.#restore(before);
			showError(err);
			return null;
		}
	}

	/** One use (open or copy): optimistic count and time, keepalive POST, server state wins. */
	visit(id: number): void {
		const before = this.find(id);
		if (before) {
			const optimistic: LinkItem = {
				...before,
				useCount: before.useCount + 1,
				lastUsedAt: new Date().toISOString(),
				archivedAt: null,
				archiveReason: null
			};
			this.#replace(optimistic);
			this.#scheduleLeave(optimistic);
		}
		linksApi.recordVisit(id).then(
			(state: LinkState) => {
				const current = this.find(id);
				if (current) this.#applyServer({ ...current, ...state });
				if (before?.archivedAt) this.refreshCounts();
			},
			(err: unknown) => {
				if (before) this.#restore(before);
				showError(err);
			}
		);
	}

	/** Polls GET /api/links/{id} every 1.5 s while the metadata is pending, for up to 20 s. */
	watchPending(id: number): void {
		this.#pending.set(id, Date.now());
		if (!this.#pollTimer) {
			this.#pollTimer = setInterval(() => void this.#poll(), PENDING_POLL_MS);
		}
	}

	get watching(): number[] {
		return [...this.#pending.keys()];
	}

	#stopPolling(): void {
		if (this.#pollTimer) clearInterval(this.#pollTimer);
		this.#pollTimer = null;
	}

	async #poll(): Promise<void> {
		const now = Date.now();
		for (const [id, started] of this.#pending) {
			if (now - started >= PENDING_MAX_MS) this.#pending.delete(id);
		}
		if (this.#pending.size === 0) {
			this.#stopPolling();
			return;
		}
		if (this.#polling) return;
		this.#polling = true;
		// Round robin: the polled ids move to the end of the queue.
		const batch = [...this.#pending.entries()].slice(0, PENDING_BATCH);
		for (const [id, started] of batch) {
			this.#pending.delete(id);
			this.#pending.set(id, started);
		}
		await Promise.all(
			batch.map(async ([id]) => {
				try {
					const detail = await linksApi.getLink(id);
					this.lastDetail = detail;
					if (this.find(id)) this.#replace(toLinkItem(detail));
					if (detail.fetchStatus !== 'pending') this.#pending.delete(id);
				} catch (err) {
					if (err instanceof ApiError && err.status === 404) this.#pending.delete(id);
				}
			})
		);
		this.#polling = false;
		if (this.#pending.size === 0) this.#stopPolling();
	}

	/**
	 * Reloads the counters that depend on the user's links: the sidebar's link count and the tag
	 * counts behind "Frequent tags". Called after creating, tagging, archiving, restoring,
	 * deleting and importing links.
	 */
	refreshCounts(): void {
		void this.#stats.refresh();
		void this.#tags.refresh();
	}

	highlight(id: number): void {
		this.highlightId = id;
	}

	clearHighlight(id: number): void {
		if (this.highlightId === id) this.highlightId = null;
	}

	requestEdit(link: LinkItem | LinkDetail): void {
		this.editRequest = toLinkItem(link);
	}

	announce(text: string): void {
		this.announcement = text;
	}

	/** Forgets everything (sign-out, tests). */
	reset(): void {
		this.#abort?.abort();
		this.#abort = null;
		this.#cancelSearch();
		this.#clearLeaving();
		this.#boundary = null;
		this.#pending.clear();
		this.#stopPolling();
		this.items = [];
		this.total = 0;
		this.counts = { pinned: 0, archived: 0 };
		this.status = 'idle';
		this.error = null;
		this.query = defaultQuery();
		this.defaults = FALLBACK_DEFAULTS;
		this.detailsId = null;
		this.highlightId = null;
		this.lastDetail = null;
		this.editRequest = null;
		this.announcement = '';
	}
}

export const links = new LinksStore();

// The list belongs to the signed-in user: a sign-out or a user switch empties it (plan 8.6).
session.onUserChange(() => links.reset());
