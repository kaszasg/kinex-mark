// The links page: states, toolbar, tag filter, list, dialogs and keyboard shortcuts (plan 8.7.3).
import { fireEvent, screen, within } from '@testing-library/svelte';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import type { LinkItem, Preferences } from '$lib/api/types.js';
import { errorTitle, t } from '$lib/i18n/index.js';
import { LOADING_DELAY_MS } from '$lib/components/app/LoadingRows.svelte';
import { linkStats } from '$lib/state/link-stats.svelte.js';
import { PAGE_SIZE } from '$lib/state/links.svelte.js';
import { preferences } from '$lib/state/preferences.svelte.js';
import { session } from '$lib/state/session.svelte.js';
import { tags } from '$lib/state/tags.svelte.js';
import { setTestPage } from '$lib/test/app-state.svelte.js';
import { makeLink, makeMe, makePreferences, makeTag } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent, type MockRoute } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import { modifierKeyLabel } from '$lib/utils/keyboard.js';
import LinksPage from './LinksPage.svelte';
import { VISIBLE_TAG_CHIPS } from './TagFilterBar.svelte';
import {
	findFloating,
	makeDetail,
	makePage,
	makeStore,
	queryFloating,
	user
} from './test-helpers.js';

vi.mock('svelte-sonner', () => ({
	toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn(), info: vi.fn() })
}));

beforeEach(() => {
	vi.mocked(toast.success).mockClear();
	vi.mocked(toast.error).mockClear();
	vi.mocked(goto).mockClear();
	vi.spyOn(linkStats, 'refresh').mockResolvedValue();
	tags.reset();
});

afterEach(async () => {
	await preferences.flush();
	vi.useRealTimers();
	vi.unstubAllGlobals();
	tags.reset();
	session.clear();
});

type Options = {
	items?: LinkItem[];
	total?: number;
	prefs?: Partial<Preferences>;
	routes?: Record<string, MockRoute>;
	path?: string;
	listRoute?: MockRoute;
};

async function setup(options: Options = {}) {
	setTestPage(options.path ?? '/');
	const items = options.items ?? [];
	const api = mockApi({
		'GET /api/links': options.listRoute ?? json(makePage(items, options.total ?? items.length)),
		'PATCH /api/me/preferences': json(makePreferences(options.prefs)),
		...options.routes
	});
	const { store } = makeStore();
	const result = renderWithApp(
		LinksPage,
		{ store },
		{ me: makeMe('developer', { preferences: makePreferences(options.prefs) }) }
	);
	await vi.waitFor(() => expect(store.status).not.toBe('loading'));
	return { ...result, api, store };
}

function anchorOf(id: number): HTMLAnchorElement {
	return document.querySelector<HTMLAnchorElement>(`[data-link-id="${id}"] [data-link-anchor]`)!;
}

function key(name: string, target: Element | Document | Window = document.body) {
	return fireEvent.keyDown(target, { key: name });
}

describe('states', () => {
	test('loading shows skeleton rows', async () => {
		vi.useFakeTimers();
		setTestPage('/');
		mockApi({ 'GET /api/links': () => new Promise(() => {}) });
		const { store } = makeStore();
		renderWithApp(LinksPage, { store }, { me: makeMe() });
		await vi.advanceTimersByTimeAsync(LOADING_DELAY_MS);
		expect(screen.getAllByTestId('loading-row')).toHaveLength(8);
		expect(store.status).toBe('loading');
	});

	test('loading in the tile view shows six tiles', async () => {
		vi.useFakeTimers();
		setTestPage('/');
		mockApi({ 'GET /api/links': () => new Promise(() => {}) });
		const { store } = makeStore();
		renderWithApp(
			LinksPage,
			{ store },
			{ me: makeMe('developer', { preferences: makePreferences({ viewMode: 'tiles' }) }) }
		);
		await vi.advanceTimersByTimeAsync(LOADING_DELAY_MS);
		expect(screen.getAllByTestId('loading-row')).toHaveLength(6);
	});

	test('an empty collection invites capture with the paste shortcut', async () => {
		await setup();
		expect(screen.getByText(t('links.empty.collection.title'))).toBeInTheDocument();
		expect(
			screen.getByText(t('links.empty.collection.text', { shortcut: `${modifierKeyLabel()} V` }))
		).toBeInTheDocument();
	});

	test('an empty filtered result offers to clear the filters', async () => {
		const { store } = await setup({ path: '/?q=zzz' });
		expect(screen.getByText(t('links.empty.filtered.title'))).toBeInTheDocument();
		expect(screen.getByTestId('links-live')).toHaveTextContent(t('links.results', { count: 0 }));
		// The page summary does not claim an empty collection.
		expect(screen.queryByText(t('links.summary', { count: 0, pinned: 0, archived: 0 }))).toBeNull();
		expect(screen.getAllByText(t('links.results', { count: 0 })).length).toBeGreaterThan(1);
		const clear = screen.getAllByRole('button', { name: t('links.filters.clear') });
		await user().click(clear.at(-1)!);
		expect(store.query.q).toBe('');
		expect(goto).toHaveBeenLastCalledWith('/', { replace: true, reset: false });
	});

	test('the archived-only view has its own empty state', async () => {
		await setup({ path: '/?archived=only' });
		expect(screen.getByText(t('links.empty.archived.title'))).toBeInTheDocument();
	});

	test('a load error offers a retry', async () => {
		const { api } = await setup({ listRoute: apiError(500, 'internal_error') });
		const alert = screen.getByRole('alert');
		expect(alert).toHaveTextContent(errorTitle({ code: 'internal_error', details: {} }));
		api.route('GET /api/links', json(makePage([makeLink({ id: 1, title: 'Back' })])));
		await user().click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		expect(await screen.findByText('Back')).toBeInTheDocument();
	});

	test('a populated list shows the summary and the pinned group first', async () => {
		const pinned = makeLink({ id: 1, pinned: true, description: 'Pinned one' });
		const other = makeLink({ id: 2, description: 'Other one' });
		await setup({ items: [pinned, other] });
		expect(
			screen.getByText(t('links.summary', { count: 2, pinned: 1, archived: 0 }))
		).toBeVisible();
		const pinnedGroup = screen.getByRole('region', { name: t('links.groups.pinned') });
		const otherGroup = screen.getByRole('region', { name: t('links.groups.others') });
		expect(within(pinnedGroup).getByText('Pinned one')).toBeInTheDocument();
		expect(within(otherGroup).getByText('Other one')).toBeInTheDocument();
		expect(screen.getByRole('button', { name: t('links.toolbar.add') })).toBeInTheDocument();
	});

	test('other orders have no group headings; archived links show a longer summary', async () => {
		await setup({
			items: [makeLink({ id: 1, pinned: true }), makeLink({ id: 2 })],
			prefs: { sort: 'created', showArchived: true, viewMode: 'tiles' }
		});
		expect(screen.queryByRole('region', { name: t('links.groups.pinned') })).toBeNull();
		expect(screen.getByRole('region', { name: t('links.list.label') })).toBeInTheDocument();
		expect(
			screen.getByText(t('links.summaryArchived', { count: 2, pinned: 1, archived: 0 }))
		).toBeInTheDocument();
	});

	test('the end of a long list is announced; loading more shows a spinner', async () => {
		const items = [makeLink({ id: 1 })];
		const { store } = await setup({ items, total: items.length });
		store.total = PAGE_SIZE + 1;
		store.items = Array.from({ length: PAGE_SIZE + 1 }, (_, index) => makeLink({ id: index + 10 }));
		await vi.waitFor(() => expect(screen.getByText(t('links.list.allLoaded'))).toBeInTheDocument());
		store.total = PAGE_SIZE + 5;
		await vi.waitFor(() => expect(screen.getByTestId('links-sentinel')).toBeInTheDocument());
		store.status = 'loading-more';
		expect(await screen.findByText(t('common.states.loadingMore'))).toBeInTheDocument();
	}, 30_000);
});

describe('toolbar', () => {
	test('scope, sort, view and archived switches change the query and the preferences', async () => {
		const { api, store } = await setup({ items: [makeLink({ id: 1 })] });
		const u = user();
		await u.click(screen.getByRole('radio', { name: t('links.scope.shared') }));
		expect(store.query.scope).toBe('shared');
		await vi.waitFor(() =>
			expect(api.calls('GET /api/links').at(-1)?.url.searchParams.get('scope')).toBe('shared')
		);

		await u.click(screen.getByRole('combobox', { name: t('links.toolbar.sort') }));
		await u.click(await findFloating('option', t('links.sort.title')));
		expect(store.query.sort).toBe('title');
		expect(preferences.current?.sort).toBe('title');
		expect(goto).toHaveBeenLastCalledWith('/?scope=shared', { replace: true, reset: false });

		await u.click(screen.getByRole('switch'));
		expect(store.query.archived).toBe('include');
		expect(preferences.current?.showArchived).toBe(true);

		await u.click(screen.getByRole('radio', { name: t('links.view.tiles') }));
		expect(preferences.current?.viewMode).toBe('tiles');

		await u.click(screen.getByRole('button', { name: t('links.toolbar.display') }));
		await u.click(await findFloating('menuitemcheckbox', t('links.fields.url')));
		expect(preferences.current?.showUrl).toBe(false);
		// A checkbox item keeps the menu open: the shortcuts item is still there.
		await u.click(await findFloating('menuitem', t('links.toolbar.shortcuts')));
		await vi.waitFor(() =>
			expect(queryFloating('menuitem', t('links.toolbar.shortcuts'))).toBeNull()
		);
		await preferences.flush();
		expect(api.calls('PATCH /api/me/preferences').length).toBeGreaterThan(0);
	});
});

describe('single-select toggle groups', () => {
	test('clicking the selected item keeps it selected (scope, view, tag match)', async () => {
		tags.items = [makeTag({ id: 1, name: 'alpha' }), makeTag({ id: 2, name: 'beta' })];
		const { store } = await setup({ items: [makeLink({ id: 1 })] });
		const u = user();
		const bar = screen.getByRole('group', { name: t('links.filters.label') });
		await u.click(within(bar).getByRole('button', { name: 'alpha' }));
		await u.click(within(bar).getByRole('button', { name: 'beta' }));
		const scope = screen.getByRole('group', { name: t('links.toolbar.scope') });
		const all = within(scope).getByRole('radio', { name: t('links.scope.all') });
		expect(all).toHaveAttribute('aria-checked', 'true');
		await u.click(all);
		expect(all).toHaveAttribute('aria-checked', 'true');
		expect(store.query.scope).toBe('all');

		const list = screen.getByRole('radio', { name: t('links.view.list') });
		await u.click(list);
		expect(list).toHaveAttribute('aria-checked', 'true');
		expect(preferences.current?.viewMode).toBe('list');

		const any = within(bar).getByRole('radio', { name: t('links.filters.match.any') });
		expect(any).toHaveAttribute('aria-checked', 'true');
		await u.click(any);
		expect(any).toHaveAttribute('aria-checked', 'true');
		expect(store.query.tagMatch).toBe('any');
	});
});

describe('mobile toolbar', () => {
	test('below 768 px it takes two rows: scope and view, then sort, display and archived', async () => {
		await setup({ items: [makeLink({ id: 1 })] });
		const toolbar = screen.getByRole('group', { name: t('links.toolbar.label') });
		const byLabel = (key: 'links.toolbar.scope' | 'links.toolbar.view') =>
			toolbar.querySelector(`[aria-label="${t(key)}"]`)!;
		const scope = byLabel('links.toolbar.scope');
		const view = byLabel('links.toolbar.view');
		const sort = screen.getByRole('combobox', { name: t('links.toolbar.sort') });
		const display = screen.getByRole('button', { name: t('links.toolbar.display') });
		const archived = screen.getByRole('switch').parentElement!;
		const lineBreak = toolbar.querySelector('.basis-full')!;
		const order = (el: Element) =>
			Number(
				[...el.classList]
					.find((c) => c.startsWith('max-md:order-'))
					?.split('-')
					.at(-1) ?? 0
			);
		expect(order(scope)).toBe(0);
		expect(order(view)).toBe(1);
		expect(order(lineBreak)).toBe(2);
		for (const el of [sort, display, archived]) expect(order(el)).toBe(3);
		expect(lineBreak).toHaveClass('h-0', 'max-md:block', 'hidden');
		// The display menu is an icon button on phones; its name stays for assistive technology.
		expect(display.querySelector('[class~="max-md:sr-only"]')).toHaveTextContent(
			t('links.toolbar.display')
		);
	});
});

describe('tag filter', () => {
	test('filters by tags, untagged and the match mode, and clears them', async () => {
		tags.items = [
			makeTag({ id: 1, name: 'infra', activeLinkCount: 5 }),
			makeTag({ id: 2, name: 'docs', activeLinkCount: 3 })
		];
		const { store } = await setup({ items: [makeLink({ id: 1 })] });
		const bar = screen.getByRole('group', { name: t('links.filters.label') });
		const u = user();
		await u.click(within(bar).getByRole('button', { name: 'infra' }));
		expect(store.query.tags).toEqual([1]);
		expect(within(bar).queryByRole('radiogroup')).toBeNull();
		await u.click(within(bar).getByRole('button', { name: 'docs' }));
		expect(store.query.tags).toEqual([1, 2]);
		await u.click(within(bar).getByRole('radio', { name: t('links.filters.match.all') }));
		expect(store.query.tagMatch).toBe('all');
		expect(preferences.current?.tagMatch).toBe('all');
		await u.click(within(bar).getByRole('button', { name: 'docs' }));
		expect(store.query.tags).toEqual([1]);
		await u.click(within(bar).getByRole('button', { name: t('links.filters.untagged') }));
		expect(store.query).toMatchObject({ untagged: true, tags: [] });
		await u.click(within(bar).getByRole('button', { name: t('links.filters.clear') }));
		expect(store.query.untagged).toBe(false);
		expect(within(bar).queryByRole('button', { name: t('links.filters.clear') })).toBeNull();
	});

	test('rarely used tags sit in the searchable "more" popover', async () => {
		tags.items = Array.from({ length: VISIBLE_TAG_CHIPS + 2 }, (_, index) =>
			makeTag({ id: index + 1, name: `tag${index + 1}`, activeLinkCount: 100 - index })
		);
		const { store } = await setup({ items: [makeLink({ id: 1 })] });
		const u = user();
		await u.click(screen.getByRole('button', { name: t('links.filters.more', { count: 2 }) }));
		const search = screen.getByPlaceholderText(t('links.filters.search'));
		await u.type(search, 'tag14');
		expect(queryFloating('option', 'tag13')).toBeNull();
		await u.click(await findFloating('option', 'tag14'));
		expect(store.query.tags).toEqual([14]);
		await u.type(search, 'nothing');
		await vi.waitFor(() => expect(screen.getByText(t('links.filters.empty'))).toBeInTheDocument());
		// Every opening starts with an empty search (E2).
		await u.keyboard('{Escape}');
		await vi.waitFor(() =>
			expect(screen.queryByPlaceholderText(t('links.filters.search'))).toBeNull()
		);
		await u.click(screen.getByRole('button', { name: t('links.filters.more', { count: 1 }) }));
		expect(
			(await screen.findByPlaceholderText(t('links.filters.search'))) as HTMLInputElement
		).toHaveValue('');
	});

	test('below 768 px the tag row scrolls in a ScrollArea; from 768 px it wraps (plan 8.9)', async () => {
		tags.items = [makeTag({ id: 1, name: 'infra' }), makeTag({ id: 2, name: 'docs' })];
		const { unmount } = await setup({ items: [makeLink({ id: 1 })] });
		let bar = screen.getByRole('group', { name: t('links.filters.label') });
		const area = bar.closest('[data-slot="scroll-area"]');
		expect(area).not.toBeNull();
		expect(area?.querySelector('[data-slot="scroll-area-viewport"]')).toContainElement(bar);
		expect(bar).toHaveClass('w-max');
		expect(bar).not.toHaveClass('overflow-x-auto');
		unmount();

		const real = window.matchMedia;
		vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
			...real(query),
			matches: query.includes('min-width: 768px')
		}));
		await setup({ items: [makeLink({ id: 1 })] });
		bar = screen.getByRole('group', { name: t('links.filters.label') });
		expect(bar.closest('[data-slot="scroll-area"]')).toBeNull();
		expect(bar).toHaveClass('flex-wrap');
	});

	test('no tags and no filters: no filter row', async () => {
		await setup({ items: [makeLink({ id: 1 })] });
		expect(screen.queryByRole('group', { name: t('links.filters.label') })).toBeNull();
	});
});

describe('dialogs and actions', () => {
	test('the add button opens the add dialog', async () => {
		await setup();
		await user().click(screen.getByRole('button', { name: t('links.toolbar.add') }));
		expect(
			await screen.findByRole('heading', { name: t('links.dialog.addTitle') })
		).toBeInTheDocument();
	});

	test('the menu opens the edit, tags and share dialogs and the details sheet', async () => {
		const link = makeLink({ id: 3, title: 'Grafana', description: '' });
		const { store } = await setup({
			items: [link],
			routes: {
				'GET /api/links/3/shares': json({ items: [] }),
				'GET /api/links/3': json(makeDetail(link))
			}
		});
		const u = user();
		const menu = () =>
			screen.getByRole('button', { name: t('links.row.menu', { label: 'Grafana' }) });

		await u.click(menu());
		await u.click(await findFloating('menuitem', t('links.actions.edit')));
		expect(await screen.findByRole('heading', { name: t('links.dialog.editTitle') })).toBeVisible();
		await u.click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

		await u.click(menu());
		await u.click(await findFloating('menuitem', t('links.actions.tags')));
		expect(await screen.findByRole('heading', { name: t('links.tagsDialog.title') })).toBeVisible();
		await u.click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

		await u.click(menu());
		await u.click(await findFloating('menuitem', t('links.actions.share')));
		expect(await screen.findByRole('heading', { name: t('links.share.title') })).toBeVisible();
		await u.click(screen.getByRole('button', { name: t('common.actions.done') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

		await u.click(menu());
		await u.click(await findFloating('menuitem', t('links.actions.details')));
		expect(store.detailsId).toBe(3);
		expect(goto).toHaveBeenLastCalledWith('/?link=3', { replace: true, reset: false });
		const sheet = await screen.findByRole('dialog');
		expect(await within(sheet).findByRole('heading', { name: 'Grafana' })).toBeInTheDocument();
		await u.click(within(sheet).getByRole('button', { name: t('common.actions.close') }));
		expect(store.detailsId).toBeNull();
	});

	test('delete asks for confirmation, then removes the link', async () => {
		const link = makeLink({ id: 4, title: 'Gone', description: '' });
		const { api, store } = await setup({
			items: [link],
			routes: { 'DELETE /api/links/4': noContent() }
		});
		const u = user();
		await u.click(screen.getByRole('button', { name: t('links.row.menu', { label: 'Gone' }) }));
		await u.click(await findFloating('menuitem', t('links.actions.delete')));
		const dialog = await screen.findByRole('alertdialog');
		expect(dialog).toHaveTextContent(t('links.delete.text', { label: 'Gone' }));
		await u.click(within(dialog).getByRole('button', { name: t('links.delete.confirm') }));
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.deleted')));
		expect(api.calls('DELETE /api/links/4')).toHaveLength(1);
		expect(store.find(4)).toBeUndefined();
	});

	test('the duplicate warning of the add dialog highlights the existing row', async () => {
		const link = makeLink({ id: 6, title: 'Known', description: '' });
		const { store } = await setup({
			items: [link],
			routes: {
				'POST /api/links': apiError(409, 'link_duplicate', undefined, { linkId: 6 })
			}
		});
		const u = user();
		await u.click(screen.getByRole('button', { name: t('links.toolbar.add') }));
		await u.type(await screen.findByLabelText(t('links.dialog.url')), link.url);
		await u.click(screen.getByRole('button', { name: t('links.dialog.save') }));
		await u.click(await screen.findByRole('button', { name: t('links.dialog.showExisting') }));
		expect(store.highlightId).toBe(6);
	});

	test('a quick capture edit request opens the edit dialog', async () => {
		const link = makeLink({ id: 7 });
		const { store } = await setup({ items: [link] });
		store.requestEdit(link);
		expect(await screen.findByRole('heading', { name: t('links.dialog.editTitle') })).toBeVisible();
		expect(store.editRequest).toBeNull();
	});
});

describe('dialogs opened from the details sheet', () => {
	/** True when b comes after a in the document, so it paints above a at the same z-index. */
	function above(a: Element, b: Element): boolean {
		return Boolean(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);
	}

	test('edit, tags, share and delete stack above the sheet and its overlay', async () => {
		const link = makeLink({ id: 5, title: 'Deep', description: '' });
		await setup({
			items: [link],
			path: '/?link=5',
			routes: {
				'GET /api/links/5': json(makeDetail(link)),
				'GET /api/links/5/shares': json({ items: [] })
			}
		});
		const sheet = await screen.findByRole('dialog');
		await within(sheet).findByRole('heading', { name: 'Deep' });
		const overlay = document.querySelector('[data-slot="sheet-overlay"]')!;
		expect(overlay).not.toBeNull();
		const u = user();
		const openFromSheet = async (name: string, heading: string, role = 'dialog') => {
			await u.click(within(sheet).getByRole('button', { name }));
			const title = await screen.findByRole('heading', { name: heading });
			const dialog = title.closest(`[role="${role}"]`)!;
			expect(dialog).not.toBe(sheet);
			expect(above(sheet, dialog)).toBe(true);
			expect(above(overlay, dialog)).toBe(true);
			return dialog;
		};

		let dialog = await openFromSheet(t('links.actions.edit'), t('links.dialog.editTitle'));
		await u.click(
			within(dialog as HTMLElement).getByRole('button', { name: t('common.actions.cancel') })
		);
		await vi.waitFor(() => expect(screen.getAllByRole('dialog')).toHaveLength(1));

		dialog = await openFromSheet(t('links.details.editTags'), t('links.tagsDialog.title'));
		await u.click(
			within(dialog as HTMLElement).getByRole('button', { name: t('common.actions.cancel') })
		);
		await vi.waitFor(() => expect(screen.getAllByRole('dialog')).toHaveLength(1));

		dialog = await openFromSheet(t('links.details.manageShares'), t('links.share.title'));
		await u.click(
			within(dialog as HTMLElement).getByRole('button', { name: t('common.actions.done') })
		);
		await vi.waitFor(() => expect(screen.getAllByRole('dialog')).toHaveLength(1));

		dialog = await openFromSheet(
			t('links.details.deleteLink'),
			t('links.delete.title'),
			'alertdialog'
		);
		await u.click(
			within(dialog as HTMLElement).getByRole('button', { name: t('common.actions.cancel') })
		);
		await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull());
		// The sheet is still open underneath.
		expect(screen.getByRole('dialog')).toBe(sheet);
	});
});

describe('keyboard shortcuts', () => {
	async function list() {
		const a = makeLink({ id: 1, description: 'First', useCount: 3 });
		const b = makeLink({ id: 2, description: 'Second', useCount: 1 });
		const ctx = await setup({
			items: [a, b],
			routes: {
				'POST /api/links/1/visit': json({ useCount: 4 }),
				'POST /api/links/2/visit': json({ useCount: 2 }),
				'POST /api/links/2/pin': json({ ...b, pinned: true }),
				'POST /api/links/2/archive': json({ ...b, archivedAt: '2026-02-01T00:00:00Z' }),
				'GET /api/links/2': json(makeDetail(b))
			}
		});
		return { ...ctx, a, b };
	}

	test('n opens the add dialog; v switches the view', async () => {
		await setup();
		await key('v');
		expect(preferences.current?.viewMode).toBe('tiles');
		await key('n');
		expect(
			await screen.findByRole('heading', { name: t('links.dialog.addTitle') })
		).toBeInTheDocument();
	});

	test('j, k and the arrows move between the links', async () => {
		await list();
		await key('j');
		expect(anchorOf(1)).toHaveFocus();
		await key('ArrowDown', anchorOf(1));
		expect(anchorOf(2)).toHaveFocus();
		await key('j', anchorOf(2));
		expect(anchorOf(2)).toHaveFocus();
		await key('k', anchorOf(2));
		expect(anchorOf(1)).toHaveFocus();
		await key('ArrowUp', anchorOf(1));
		expect(anchorOf(1)).toHaveFocus();
		(document.activeElement as HTMLElement).blur();
		await key('k');
		expect(anchorOf(2)).toHaveFocus();
	});

	test('j and the arrows also work while <main> itself has the focus (skip link, blank click)', async () => {
		const { container } = await list();
		// The page root's container stands in for <main tabindex="-1">.
		container.tabIndex = -1;
		container.focus();
		expect(document.activeElement).toBe(container);
		await key('j', container);
		expect(anchorOf(1)).toHaveFocus();
		container.focus();
		await key('ArrowUp', container);
		expect(anchorOf(2)).toHaveFocus();
	});

	test('a row archived by keyboard hands the focus to its neighbour; Undo brings it back (plan 8.9)', async () => {
		const a = makeLink({ id: 1, description: 'First', useCount: 3 });
		const b = makeLink({ id: 2, description: 'Second', useCount: 2 });
		const c = makeLink({ id: 3, description: 'Third', useCount: 1 });
		await setup({
			items: [a, b, c],
			routes: {
				'POST /api/links/2/archive': json({ ...b, archivedAt: '2026-02-01T00:00:00Z' }),
				'DELETE /api/links/2/archive': json(b),
				'POST /api/links/3/archive': json({ ...c, archivedAt: '2026-02-01T00:00:00Z' }),
				'GET /api/links/counts': json({ pinned: 0, archived: 1 })
			}
		});
		anchorOf(2).focus();
		await key('a', anchorOf(2));
		await vi.waitFor(() => expect(document.querySelector('[data-link-id="2"]')).toBeNull());
		await vi.waitFor(() => expect(anchorOf(3)).toHaveFocus());

		// The last row has no next neighbour: the previous one takes the focus.
		await key('a', anchorOf(3));
		await vi.waitFor(() => expect(document.querySelector('[data-link-id="3"]')).toBeNull());
		await vi.waitFor(() => expect(anchorOf(1)).toHaveFocus());

		// Undo from the toast: the toast button had the focus and goes away.
		const archivedToast = vi
			.mocked(toast.success)
			.mock.calls.find(([text]) => text === t('links.toast.archived'));
		const action = (archivedToast?.[1] as unknown as { action: { onClick: () => void } }).action;
		(document.activeElement as HTMLElement).blur();
		action.onClick();
		await vi.waitFor(() => expect(anchorOf(2)).toHaveFocus());
	});

	test('Enter on a focused row opens its link once', async () => {
		const { api } = await list();
		const row = document.querySelector<HTMLElement>('[data-link-id="2"]')!;
		const anchor = anchorOf(2);
		anchor.addEventListener('click', (event) => event.preventDefault());
		row.focus();
		await key('Enter', row);
		await vi.waitFor(() => expect(api.calls('POST /api/links/2/visit')).toHaveLength(1));
		// On the anchor itself the browser's native click opens it; the page adds nothing.
		anchor.focus();
		await key('Enter', anchor);
		expect(api.calls('POST /api/links/2/visit')).toHaveLength(1);
	});

	test('p, c, m, a, i, e and Delete act on the focused link', async () => {
		const { api, store } = await list();
		anchorOf(2).focus();
		await key('p', anchorOf(2));
		await vi.waitFor(() => expect(api.calls('POST /api/links/2/pin')).toHaveLength(1));

		anchorOf(2).focus();
		await key('c', anchorOf(2));
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.urlCopied')));
		anchorOf(2).focus();
		await key('m', anchorOf(2));
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(t('links.toast.markdownCopied'))
		);

		anchorOf(2).focus();
		await key('i', anchorOf(2));
		expect(store.detailsId).toBe(2);
		store.closeDetails();
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

		anchorOf(2).focus();
		await key('e', anchorOf(2));
		expect(await screen.findByRole('heading', { name: t('links.dialog.editTitle') })).toBeVisible();
		await user().click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

		anchorOf(2).focus();
		await key('Delete', anchorOf(2));
		expect(await screen.findByRole('alertdialog')).toBeInTheDocument();
		await user().click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull());

		anchorOf(2).focus();
		await key('a', anchorOf(2));
		await vi.waitFor(() => expect(api.calls('POST /api/links/2/archive')).toHaveLength(1));
	});

	test('shortcuts without a focused link, in text fields and with modifiers do nothing', async () => {
		const { api } = await list();
		await key('p');
		const input = document.createElement('input');
		document.body.append(input);
		input.focus();
		await key('n', input);
		await fireEvent.keyDown(document.body, { key: 'n', ctrlKey: true });
		await key('x');
		expect(screen.queryByRole('dialog')).toBeNull();
		expect(api.calls('POST /api/links/1/pin')).toHaveLength(0);
		input.remove();
	});

	test('a shared link has no edit or delete shortcut', async () => {
		const shared = makeLink({ id: 9, isOwner: false });
		await setup({ items: [shared] });
		anchorOf(9).focus();
		await key('e', anchorOf(9));
		await key('Delete', anchorOf(9));
		expect(screen.queryByRole('dialog')).toBeNull();
		expect(screen.queryByRole('alertdialog')).toBeNull();
	});
});

describe('URL sync', () => {
	test('the details sheet opens from ?link=', async () => {
		const link = makeLink({ id: 5, title: 'Deep', description: '' });
		await setup({
			items: [link],
			path: '/?link=5',
			routes: { 'GET /api/links/5': json(makeDetail(link)) }
		});
		const sheet = await screen.findByRole('dialog');
		expect(await within(sheet).findByRole('heading', { name: 'Deep' })).toBeInTheDocument();
	});
});
