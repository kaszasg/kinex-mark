// Link actions shared by menus, the details sheet and the shortcuts (plan 8.7.3).
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { errorMessage, t } from '$lib/i18n/index.js';
import { linkStats } from '$lib/state/link-stats.svelte.js';
import { tags } from '$lib/state/tags.svelte.js';
import { makeLink, makeMe } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import {
	canRun,
	deleteLink,
	linkActions,
	openLink,
	refreshLink,
	runLinkAction,
	toggleArchive,
	togglePin,
	type LinkActionId,
	type LinkUi
} from './actions.js';
import { makeDetail, makeStore } from './test-helpers.js';

vi.mock('svelte-sonner', () => ({
	toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn() })
}));

type ToastOptions = { action?: { label: string; onClick: () => void } };

beforeEach(() => {
	vi.mocked(toast.success).mockClear();
	vi.mocked(toast.error).mockClear();
	vi.spyOn(linkStats, 'refresh').mockResolvedValue();
	vi.spyOn(tags, 'refresh').mockResolvedValue().mockClear();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

function ui(): LinkUi {
	return {
		details: vi.fn(),
		edit: vi.fn(),
		tags: vi.fn(),
		share: vi.fn(),
		confirmDelete: vi.fn()
	};
}

describe('linkActions', () => {
	test('labels follow the pinned and archived state', () => {
		const actions = linkActions(
			makeLink({ pinned: true, archivedAt: '2026-01-01T00:00:00Z' }),
			makeMe()
		);
		expect(actions.find((action) => action.id === 'pin')?.label).toBe('links.actions.unpin');
		expect(actions.find((action) => action.id === 'archive')?.label).toBe('links.actions.restore');
		expect(actions.at(-1)).toMatchObject({ id: 'delete', destructive: true });
	});

	test('without a viewer there is no share action', () => {
		expect(canRun('share', makeLink(), null)).toBe(false);
		expect(canRun('edit', makeLink(), null)).toBe(true);
	});
});

describe('runLinkAction', () => {
	test('dialog actions go to the page', () => {
		const handlers = ui();
		const link = makeLink();
		const { store } = makeStore();
		const map: [LinkActionId, keyof LinkUi][] = [
			['details', 'details'],
			['edit', 'edit'],
			['tags', 'tags'],
			['share', 'share'],
			['delete', 'confirmDelete']
		];
		for (const [id, handler] of map) {
			runLinkAction(id, link, handlers, store);
			expect(handlers[handler]).toHaveBeenCalledWith(link);
		}
	});

	test('direct actions call the API', async () => {
		const link = makeLink({ id: 2 });
		const api = mockApi({
			'POST /api/links/2/visit': json({ useCount: 1 }),
			'POST /api/links/2/pin': json({ ...link, pinned: true }),
			'POST /api/links/2/archive': json({ ...link, archivedAt: '2026-02-01T00:00:00Z' }),
			'POST /api/links/2/refresh': json(makeDetail(link, { fetchStatus: 'pending' }))
		});
		const { store } = makeStore();
		store.items = [link];
		vi.spyOn(store, 'watchPending').mockImplementation(() => {});
		const open = vi.spyOn(window, 'open').mockImplementation(() => null);
		const handlers = ui();
		runLinkAction('open', link, handlers, store);
		expect(open).toHaveBeenCalledWith(link.url, '_blank', 'noopener,noreferrer');
		runLinkAction('copyUrl', link, handlers, store);
		runLinkAction('copyMarkdown', link, handlers, store);
		runLinkAction('refresh', link, handlers, store);
		runLinkAction('pin', link, handlers, store);
		runLinkAction('archive', link, handlers, store);
		await vi.waitFor(() => expect(api.calls('POST /api/links/2/archive')).toHaveLength(1));
		await vi.waitFor(() => expect(api.calls('POST /api/links/2/visit')).toHaveLength(3));
		expect(api.calls('POST /api/links/2/pin')).toHaveLength(1);
		expect(api.calls('POST /api/links/2/refresh')).toHaveLength(1);
		expect(store.watchPending).toHaveBeenCalledWith(2);
	});
});

describe('toggles', () => {
	test('unpinning and pinning an archived link use their own toasts; failures return null', async () => {
		const pinned = makeLink({ id: 3, pinned: true });
		const archived = makeLink({ id: 4, archivedAt: '2026-01-01T00:00:00Z' });
		mockApi({
			'DELETE /api/links/3/pin': json({ ...pinned, pinned: false }),
			'POST /api/links/4/pin': json({ ...archived, pinned: true, archivedAt: null }),
			'POST /api/links/5/pin': apiError(500, 'internal_error')
		});
		const { store } = makeStore();
		await togglePin(pinned, store);
		expect(toast.success).toHaveBeenLastCalledWith(t('links.toast.unpinned'));
		await togglePin(archived, store);
		expect(toast.success).toHaveBeenLastCalledWith(t('links.toast.pinnedRestored'));
		expect(await togglePin(makeLink({ id: 5 }), store)).toBeNull();
	});

	test('archiving offers an undo; restoring confirms', async () => {
		const link = makeLink({ id: 6 });
		const api = mockApi({
			'POST /api/links/6/archive': json({ ...link, archivedAt: '2026-02-01T00:00:00Z' }),
			'DELETE /api/links/6/archive': json(link),
			'POST /api/links/7/archive': apiError(500, 'internal_error')
		});
		const { store } = makeStore();
		await toggleArchive(link, store);
		const options = vi.mocked(toast.success).mock.calls.at(-1)?.[1] as ToastOptions;
		expect(options.action?.label).toBe(t('links.toast.undo'));
		options.action!.onClick();
		await vi.waitFor(() => expect(api.calls('DELETE /api/links/6/archive')).toHaveLength(1));
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenLastCalledWith(t('links.toast.restored'))
		);
		vi.mocked(toast.success).mockClear();
		await toggleArchive({ ...link, archivedAt: '2026-02-01T00:00:00Z' }, store);
		expect(toast.success).toHaveBeenLastCalledWith(t('links.toast.restored'));
		expect(await toggleArchive(makeLink({ id: 7 }), store)).toBeNull();
	});
});

describe('refresh and delete', () => {
	test('a failed refresh shows the error', async () => {
		mockApi({ 'POST /api/links/8/refresh': apiError(429, 'too_many_attempts') });
		const { store } = makeStore();
		await refreshLink(makeLink({ id: 8 }), store);
		expect(toast.error).toHaveBeenCalledWith(
			errorMessage({ code: 'too_many_attempts', details: {} })
		);
	});

	test('delete removes the link, closes its details and refreshes the counter', async () => {
		const link = makeLink({ id: 9 });
		mockApi({ 'DELETE /api/links/9': noContent() });
		const { store } = makeStore();
		store.items = [link];
		store.detailsId = 9;
		await deleteLink(link, store);
		expect(store.find(9)).toBeUndefined();
		expect(store.detailsId).toBeNull();
		expect(toast.success).toHaveBeenCalledWith(t('links.toast.deleted'));
		expect(linkStats.refresh).toHaveBeenCalled();
		expect(tags.refresh).toHaveBeenCalled();
	});

	test('a failed delete throws to the confirmation dialog', async () => {
		mockApi({ 'DELETE /api/links/9': apiError(404, 'not_found') });
		const { store } = makeStore();
		await expect(deleteLink(makeLink({ id: 9 }), store)).rejects.toMatchObject({
			code: 'not_found'
		});
	});
});
