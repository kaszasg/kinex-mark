// Details sheet and share dialog (plan 8.7.3).
import { fireEvent, screen, within } from '@testing-library/svelte';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import type { LinkDetail, LinkItem, Me, Share } from '$lib/api/types.js';
import { errorTitle, fmt, t } from '$lib/i18n/index.js';
import { linkStats } from '$lib/state/link-stats.svelte.js';
import { tags } from '$lib/state/tags.svelte.js';
import { makeLink, makeMe, makeTag } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import LinkDetailsSheet from './LinkDetailsSheet.svelte';
import ShareDialog, { DIRECTORY_DEBOUNCE_MS, DIRECTORY_QUERY_MAX } from './ShareDialog.svelte';
import { findFloating, makeDetail, makeStore, user } from './test-helpers.js';

vi.mock('svelte-sonner', () => ({
	toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn(), info: vi.fn() })
}));

const DAY = 86_400_000;
/** toHaveTextContent collapses whitespace (also the narrow no-break spaces of dates). */
const text = (value: string) => value.replace(/\s+/g, ' ').trim();

beforeEach(() => {
	vi.mocked(toast.success).mockClear();
	vi.mocked(toast.error).mockClear();
	vi.spyOn(linkStats, 'refresh').mockResolvedValue();
	tags.reset();
});

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
	tags.reset();
});

const share = (patch: Partial<Share>): Share => ({
	id: 1,
	targetType: 'all',
	targetRole: null,
	targetUser: null,
	createdAt: '2026-01-05T10:00:00.000Z',
	...patch
});

function openSheet(
	detail: LinkDetail,
	options: { me?: Me; inList?: boolean; routes?: Record<string, unknown> } = {}
) {
	const api = mockApi({
		[`GET /api/links/${detail.id}`]: json(detail),
		...(options.routes as Record<string, never>)
	});
	const { store } = makeStore();
	if (options.inList) store.items = [makeLink({ ...detail })];
	const onaction = vi.fn();
	const onclose = vi.fn();
	const result = renderWithApp(
		LinkDetailsSheet,
		{
			id: detail.id,
			me: options.me ?? makeMe(),
			effectiveArchiveAfterDays: 90,
			onaction,
			onclose,
			store
		},
		{ me: options.me ?? makeMe() }
	);
	return { ...result, api, store, onaction, onclose };
}

const due = new Date(Date.now() + 20 * DAY).toISOString();

function ownDetail(patch: Partial<LinkDetail> = {}): LinkDetail {
	const link = makeLink({
		id: 3,
		title: 'Grafana',
		description: 'Éles műszerfal\nmásodik sor',
		useCount: 12,
		lastUsedAt: new Date(Date.now() - DAY).toISOString(),
		archiveDueAt: due,
		tagIds: [1]
	});
	return makeDetail(link, {
		tags: [makeTag({ id: 1, name: 'infra' })],
		fetchAttempts: 2,
		shares: [
			share({ id: 1 }),
			share({ id: 2, targetType: 'role', targetRole: 'developer' }),
			share({
				id: 3,
				targetType: 'user',
				targetUser: { id: 4, username: 'peter', displayName: 'Nagy Péter' }
			})
		],
		...patch
	});
}

describe('LinkDetailsSheet', () => {
	test('shows the URL readable, as the list does; open and copy keep the encoded URL (E2)', async () => {
		const url = 'https://b%C3%BCcher.example.invalid/stra%C3%9Fe/%C3%B6ffnungszeiten?ő=ű';
		openSheet(ownDetail({ url }));
		const dialog = await screen.findByRole('dialog');
		await within(dialog).findByRole('heading', { name: 'Éles műszerfal' });
		expect(screen.getByTestId('details-url')).toHaveTextContent(
			'https://bücher.example.invalid/straße/öffnungszeiten?ő=ű'
		);
		expect(within(dialog).getByRole('link', { name: t('links.actions.open') })).toHaveAttribute(
			'href',
			url
		);
	});

	test('loads and shows every block of an own link', async () => {
		openSheet(ownDetail());
		const dialog = await screen.findByRole('dialog');
		expect(within(dialog).getByRole('status')).toHaveTextContent(t('common.states.loading'));
		expect(
			await within(dialog).findByRole('heading', { name: 'Éles műszerfal' })
		).toBeInTheDocument();
		expect(screen.getByTestId('details-url')).toHaveTextContent(
			'https://grafana.example.net/d/abc'
		);
		expect(dialog).toHaveTextContent('Grafana');
		expect(dialog).toHaveTextContent('infra');
		expect(dialog).toHaveTextContent(fmt.number(12));
		expect(dialog).toHaveTextContent(text(fmt.date(due)));
		expect(dialog).toHaveTextContent(text(t('links.details.dueIn', { count: 20 })));
		expect(dialog).toHaveTextContent(t('fetch.status.ok'));
		expect(dialog).toHaveTextContent(t('links.share.everyone'));
		expect(dialog).toHaveTextContent(
			t('links.share.roleTarget', { role: t('roles.plural.developer') })
		);
		expect(dialog).toHaveTextContent('Nagy Péter');
		for (const name of [
			t('links.actions.edit'),
			t('links.actions.archive'),
			t('links.details.deleteLink'),
			t('links.actions.refresh'),
			t('links.details.manageShares')
		]) {
			expect(within(dialog).getByRole('button', { name })).toBeInTheDocument();
		}
		expect(dialog).not.toHaveTextContent(t('links.details.fields.owner'));
	});

	test('the usage metrics stack in one column so the dates are not clipped', async () => {
		openSheet(ownDetail());
		const usage = await screen.findByTestId('details-usage');
		// Three cards side by side leave about 120 px per card, too narrow for a data-lg date.
		expect(usage).toHaveClass('grid-cols-1');
		expect(usage.className).not.toMatch(/auto-fit|grid-cols-[23]/);
		expect(usage.children).toHaveLength(3);
		expect(usage).toHaveTextContent(text(fmt.date(due)));
	});

	test('a load error offers a retry', async () => {
		const { api } = openSheet(ownDetail(), {
			routes: { 'GET /api/links/3': apiError(500, 'internal_error') }
		});
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(errorTitle({ code: 'internal_error', details: {} }));
		api.route('GET /api/links/3', json(ownDetail()));
		await user().click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		await vi.waitFor(() => expect(api.calls('GET /api/links/3')).toHaveLength(2));
		expect(await screen.findByRole('heading', { name: 'Éles műszerfal' })).toBeInTheDocument();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	test('a shared link shows the owner and the share time, without owner actions', async () => {
		const detail = makeDetail(
			makeLink({
				id: 3,
				isOwner: false,
				title: 'Shared',
				description: '',
				owner: { id: 2, username: 'peter', displayName: 'Nagy Péter' },
				sharedAt: '2026-01-04T10:00:00.000Z'
			})
		);
		openSheet(detail);
		const dialog = await screen.findByRole('dialog');
		await within(dialog).findByRole('heading', { name: 'Shared' });
		expect(dialog).toHaveTextContent(t('links.details.fields.owner'));
		expect(dialog).toHaveTextContent('Nagy Péter');
		expect(dialog).toHaveTextContent(t('links.details.times.sharedAt'));
		expect(dialog).toHaveTextContent(text(fmt.dateTime('2026-01-04T10:00:00.000Z')));
		expect(within(dialog).queryByRole('button', { name: t('links.actions.edit') })).toBeNull();
		expect(
			within(dialog).queryByRole('button', { name: t('links.details.deleteLink') })
		).toBeNull();
		expect(within(dialog).queryByRole('button', { name: t('links.actions.refresh') })).toBeNull();
		expect(dialog).not.toHaveTextContent(t('links.details.sections.sharing'));
	});

	test.each([
		['pending', 'fetch.status.pending'],
		['failed', 'fetch.status.failed'],
		['blocked', 'fetch.status.blocked']
	] as const)('shows the %s metadata state', async (status, key) => {
		openSheet(
			ownDetail({
				fetchStatus: status,
				fetchError:
					status === 'failed' ? 'timeout' : status === 'blocked' ? 'blocked_address' : null
			})
		);
		const badge = await screen.findByTestId('details-fetch');
		expect(badge).toHaveTextContent(t(key));
		if (status === 'failed') expect(badge).toHaveTextContent(t('fetch.errors.timeout'));
		if (status === 'pending') {
			expect(screen.getByRole('button', { name: t('links.actions.refresh') })).toBeDisabled();
		}
	});

	test('archived, pinned and never-archiving links show the right archive metric', async () => {
		const archived = openSheet(
			ownDetail({
				archivedAt: '2026-01-04T10:00:00.000Z',
				archiveReason: 'auto',
				archiveDueAt: null
			})
		);
		const dialog = await screen.findByRole('dialog');
		await within(dialog).findByRole('heading', { name: 'Éles műszerfal' });
		expect(dialog).toHaveTextContent(t('links.details.archivedNote'));
		expect(dialog).toHaveTextContent(t('links.details.archiveReason.auto'));
		expect(within(dialog).getByRole('button', { name: t('links.actions.restore') })).toBeVisible();
		archived.unmount();

		const pinned = openSheet(ownDetail({ pinned: true, lastUsedAt: null }));
		await screen.findByRole('heading', { name: 'Éles műszerfal' });
		expect(screen.getByRole('dialog')).toHaveTextContent(t('links.details.never'));
		expect(screen.getByRole('dialog')).toHaveTextContent(t('links.details.neverUsed'));
		pinned.unmount();

		openSheet(ownDetail({ archiveDueAt: new Date(Date.now() + 1000).toISOString() }));
		await screen.findByRole('heading', { name: 'Éles műszerfal' });
		expect(screen.getByRole('dialog')).toHaveTextContent(
			text(t('links.details.dueIn', { count: 1 }))
		);
	});

	test('the action buttons run their actions and count uses', async () => {
		const detail = ownDetail();
		const { api, onaction, onclose, store } = openSheet(detail, {
			inList: true,
			routes: {
				'POST /api/links/3/visit': json({ useCount: 13 }),
				'POST /api/links/3/pin': json({ ...makeLink({ ...detail }), pinned: true }),
				'POST /api/links/3/archive': json({
					...makeLink({ ...detail }),
					archivedAt: '2026-02-01T00:00:00Z'
				}),
				'POST /api/links/3/refresh': json({ ...detail, fetchStatus: 'pending' })
			}
		});
		vi.spyOn(store, 'watchPending').mockImplementation(() => {});
		const dialog = await screen.findByRole('dialog');
		await within(dialog).findByRole('heading', { name: 'Éles műszerfal' });
		const u = user();

		const open = within(dialog).getByRole('link', { name: t('links.actions.open') });
		expect(open).toHaveAttribute('href', detail.url);
		open.addEventListener('click', (event) => event.preventDefault());
		await u.click(open);
		await vi.waitFor(() => expect(api.calls('POST /api/links/3/visit')).toHaveLength(1));

		await u.click(within(dialog).getAllByRole('button', { name: t('links.actions.copyUrl') })[0]);
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.urlCopied')));
		expect(await navigator.clipboard.readText()).toBe(detail.url);

		await u.click(within(dialog).getByRole('button', { name: t('links.actions.copyMarkdown') }));
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(t('links.toast.markdownCopied'))
		);
		expect(await navigator.clipboard.readText()).toBe(
			'[Éles műszerfal](https://grafana.example.net/d/abc)'
		);

		await u.click(within(dialog).getAllByRole('button', { name: t('links.actions.copyUrl') })[1]);
		await vi.waitFor(() => expect(api.calls('POST /api/links/3/visit')).toHaveLength(4));

		await u.click(within(dialog).getByRole('button', { name: t('links.actions.pin') }));
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.pinned')));

		await u.click(within(dialog).getByRole('button', { name: t('links.actions.refresh') }));
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.refreshing')));

		await u.click(within(dialog).getByRole('button', { name: t('links.details.editTags') }));
		expect(onaction).toHaveBeenLastCalledWith('tags', expect.objectContaining({ id: 3 }));
		await u.click(within(dialog).getByRole('button', { name: t('links.details.manageShares') }));
		expect(onaction).toHaveBeenLastCalledWith('share', expect.objectContaining({ id: 3 }));
		await u.click(within(dialog).getByRole('button', { name: t('links.actions.edit') }));
		expect(onaction).toHaveBeenLastCalledWith('edit', expect.objectContaining({ id: 3 }));
		await u.click(within(dialog).getByRole('button', { name: t('links.details.deleteLink') }));
		expect(onaction).toHaveBeenLastCalledWith('delete', expect.objectContaining({ id: 3 }));

		await u.click(within(dialog).getByRole('button', { name: t('links.actions.archive') }));
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(t('links.toast.archived'), expect.anything())
		);

		await u.click(within(dialog).getByRole('button', { name: t('common.actions.close') }));
		expect(onclose).toHaveBeenCalled();
	});

	test('without a list copy the sheet counts the use and applies pin answers itself', async () => {
		const detail = ownDetail({ useCount: 1 });
		openSheet(detail, {
			routes: {
				'POST /api/links/3/visit': json({ useCount: 2 }),
				'POST /api/links/3/pin': json({ ...makeLink({ ...detail }), pinned: true })
			}
		});
		const dialog = await screen.findByRole('dialog');
		await within(dialog).findByRole('heading', { name: 'Éles műszerfal' });
		const u = user();
		await u.click(within(dialog).getByRole('button', { name: t('links.actions.copyMarkdown') }));
		await vi.waitFor(() => expect(dialog).toHaveTextContent(fmt.number(2)));
		await u.click(within(dialog).getByRole('button', { name: t('links.actions.pin') }));
		expect(
			await within(dialog).findByRole('button', { name: t('links.actions.unpin') })
		).toHaveAttribute('aria-pressed', 'true');
	});

	test('polling results update the open sheet', async () => {
		const { store } = openSheet(ownDetail({ fetchStatus: 'pending' }));
		await screen.findByRole('heading', { name: 'Éles műszerfal' });
		store.lastDetail = ownDetail({ fetchStatus: 'ok', title: 'Fresh title' });
		await vi.waitFor(() => expect(screen.getByRole('dialog')).toHaveTextContent('Fresh title'));
	});

	test('a closed sheet renders nothing', () => {
		mockApi({});
		const { store } = makeStore();
		renderWithApp(LinkDetailsSheet, {
			id: null,
			me: makeMe(),
			effectiveArchiveAfterDays: 90,
			onaction: vi.fn(),
			onclose: vi.fn(),
			store
		});
		expect(screen.queryByRole('dialog')).toBeNull();
	});
});

describe('ShareDialog', () => {
	function openShare(link: LinkItem, routes: Record<string, unknown>) {
		const api = mockApi(routes as Record<string, never>);
		const { store } = makeStore();
		store.items = [link];
		renderWithApp(ShareDialog, { open: true, link, store }, { me: makeMe() });
		return { api, store };
	}

	test('lists the shares, revokes one and adds a share for everyone', async () => {
		const link = makeLink({ id: 3, shareCount: 2 });
		const { api, store } = openShare(link, {
			'GET /api/links/3/shares': json({
				items: [
					share({ id: 2, targetType: 'role', targetRole: 'customer' }),
					share({
						id: 3,
						targetType: 'user',
						targetUser: { id: 4, username: 'peter', displayName: 'Nagy Péter' }
					})
				]
			}),
			'DELETE /api/links/3/shares/2': noContent(),
			'POST /api/links/3/shares': json(share({ id: 9 }), 201)
		});
		const dialog = await screen.findByRole('dialog');
		const roleText = t('links.share.roleTarget', { role: t('roles.plural.customer') });
		expect(await within(dialog).findByText(roleText)).toBeInTheDocument();
		expect(dialog).toHaveTextContent('peter');
		const u = user();
		await u.click(
			within(dialog).getByRole('button', { name: t('links.share.revoke', { target: roleText }) })
		);
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.unshared')));
		expect(api.calls('DELETE /api/links/3/shares/2')).toHaveLength(1);
		expect(store.find(3)?.shareCount).toBe(1);

		await u.click(within(dialog).getByRole('button', { name: t('links.share.add') }));
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.shared')));
		expect(api.calls('POST /api/links/3/shares')[0].body).toEqual({ targetType: 'all' });
		expect(dialog).toHaveTextContent(t('links.share.everyone'));
		expect(store.find(3)?.shareCount).toBe(2);

		await u.click(within(dialog).getByRole('button', { name: t('common.actions.done') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});

	test('shares with a role and reports a duplicate under the picker', async () => {
		const link = makeLink({ id: 3 });
		const { api } = openShare(link, {
			'GET /api/links/3/shares': json({ items: [] }),
			'POST /api/links/3/shares': apiError(409, 'share_duplicate')
		});
		const dialog = await screen.findByRole('dialog');
		expect(await within(dialog).findByText(t('links.share.empty'))).toBeInTheDocument();
		const u = user();
		await u.click(within(dialog).getByRole('radio', { name: t('links.share.type.role') }));
		await u.click(within(dialog).getByRole('button', { name: t('links.share.add') }));
		expect(document.getElementById('share-picker-error')).not.toBeNull();
		expect(api.calls('POST /api/links/3/shares')).toHaveLength(0);

		await u.click(screen.getByRole('combobox', { name: t('links.share.role') }));
		await u.click(await findFloating('option', t('roles.plural.developer')));
		await u.click(within(dialog).getByRole('button', { name: t('links.share.add') }));
		await vi.waitFor(() => expect(api.calls('POST /api/links/3/shares')).toHaveLength(1));
		expect(api.calls('POST /api/links/3/shares')[0].body).toEqual({
			targetType: 'role',
			targetRole: 'developer'
		});
		expect(
			await screen.findByText(errorTitle({ code: 'share_duplicate', details: {} }), {
				exact: false
			})
		).toBeInTheDocument();
	});

	test('the directory search text stays within the 64 runes of the API', async () => {
		const link = makeLink({ id: 3 });
		const { api } = openShare(link, {
			'GET /api/links/3/shares': json({ items: [] }),
			'GET /api/users/directory': json({ items: [] })
		});
		const dialog = await screen.findByRole('dialog');
		await within(dialog).findByText(t('links.share.empty'));
		const u = user();
		await u.click(within(dialog).getByRole('radio', { name: t('links.share.type.user') }));
		await u.click(within(dialog).getByRole('button', { name: t('links.share.user') }));
		const search = screen.getByRole('combobox', { hidden: true });
		expect(search).toHaveAttribute('maxlength', String(DIRECTORY_QUERY_MAX));
		// A paste can bypass maxlength: the client still clamps and cleans the text.
		await fireEvent.input(search, { target: { value: `kovacs\t${'a'.repeat(80)}` } });
		await new Promise((resolve) => setTimeout(resolve, DIRECTORY_DEBOUNCE_MS + 50));
		await vi.waitFor(() => expect(api.calls('GET /api/users/directory').length).toBeGreaterThan(1));
		const q = api.calls('GET /api/users/directory').at(-1)!.url.searchParams.get('q')!;
		expect([...q]).toHaveLength(DIRECTORY_QUERY_MAX);
		expect(q.startsWith('kovacs a')).toBe(true);
	});

	test('every opening of the user picker starts with an empty search (E2)', async () => {
		const link = makeLink({ id: 3 });
		const { api } = openShare(link, {
			'GET /api/links/3/shares': json({ items: [] }),
			'GET /api/users/directory': json({ items: [] })
		});
		const dialog = await screen.findByRole('dialog');
		await within(dialog).findByText(t('links.share.empty'));
		const u = user();
		await u.click(within(dialog).getByRole('radio', { name: t('links.share.type.user') }));
		const picker = within(dialog).getByRole('button', { name: t('links.share.user') });
		await u.click(picker);
		await u.type(screen.getByRole('combobox', { hidden: true }), 'kan');
		await u.keyboard('{Escape}');
		await vi.waitFor(() => expect(screen.queryByRole('combobox', { hidden: true })).toBeNull());
		await u.click(picker);
		const input = (await vi.waitFor(() =>
			screen.getByRole('combobox', { hidden: true })
		)) as HTMLInputElement;
		expect(input.value).toBe('');
		await vi.waitFor(() =>
			expect(api.calls('GET /api/users/directory').at(-1)?.url.searchParams.get('q') ?? '').toBe('')
		);
	});

	test('searches the directory and shares with one user', async () => {
		const link = makeLink({ id: 3 });
		const { api } = openShare(link, {
			'GET /api/links/3/shares': json({ items: [] }),
			'GET /api/users/directory': (req: { url: URL }) =>
				json({
					items:
						req.url.searchParams.get('q') === 'zz'
							? []
							: [{ id: 4, username: 'peter', displayName: 'Nagy Péter', role: 'developer' }]
				}),
			'POST /api/links/3/shares': apiError(422, 'validation_failed', { targetUserId: 'invalid' })
		});
		const dialog = await screen.findByRole('dialog');
		await within(dialog).findByText(t('links.share.empty'));
		const u = user();
		await u.click(within(dialog).getByRole('radio', { name: t('links.share.type.user') }));
		await u.click(within(dialog).getByRole('button', { name: t('links.share.add') }));
		expect(api.calls('POST /api/links/3/shares')).toHaveLength(0);
		await u.click(within(dialog).getByRole('button', { name: t('links.share.user') }));
		await vi.waitFor(() => expect(api.calls('GET /api/users/directory')).toHaveLength(1));
		await u.type(screen.getByRole('combobox', { hidden: true }), 'zz');
		await new Promise((resolve) => setTimeout(resolve, DIRECTORY_DEBOUNCE_MS + 50));
		await vi.waitFor(() =>
			expect(screen.getByText(t('links.share.userEmpty'))).toBeInTheDocument()
		);
		await u.clear(screen.getByRole('combobox', { hidden: true }));
		await new Promise((resolve) => setTimeout(resolve, DIRECTORY_DEBOUNCE_MS + 50));
		const option = await vi.waitFor(() => {
			const found = screen
				.queryAllByRole('option', { hidden: true })
				.find((el) => el.textContent?.includes('Nagy Péter'));
			if (!found) throw new Error('no option');
			return found;
		});
		await u.click(option);
		expect(within(dialog).getByRole('button', { name: t('links.share.user') })).toHaveTextContent(
			'Nagy Péter'
		);
		await u.click(within(dialog).getByRole('button', { name: t('links.share.add') }));
		await vi.waitFor(() => expect(api.calls('POST /api/links/3/shares')).toHaveLength(1));
		expect(api.calls('POST /api/links/3/shares')[0].body).toEqual({
			targetType: 'user',
			targetUserId: 4
		});
		await vi.waitFor(() =>
			expect(document.getElementById('share-picker-error')?.textContent?.trim()).not.toBe('')
		);
	});

	test('a failed load offers a retry; failed revoke and add show errors', async () => {
		const link = makeLink({ id: 3 });
		const { api } = openShare(link, {
			'GET /api/links/3/shares': apiError(500, 'internal_error')
		});
		const dialog = await screen.findByRole('dialog');
		const alert = await within(dialog).findByRole('alert');
		expect(within(dialog).getByRole('button', { name: t('links.share.add') })).toBeDisabled();
		api.route('GET /api/links/3/shares', json({ items: [share({ id: 1 })] }));
		api.route('DELETE /api/links/3/shares/1', apiError(500, 'internal_error'));
		api.route('POST /api/links/3/shares', apiError(500, 'internal_error'));
		const u = user();
		await u.click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		await u.click(
			await within(dialog).findByRole('button', {
				name: t('links.share.revoke', { target: t('links.share.everyone') })
			})
		);
		await vi.waitFor(() => expect(toast.error).toHaveBeenCalled());
		await u.click(within(dialog).getByRole('button', { name: t('links.share.add') }));
		expect(await within(dialog).findByRole('alert')).toBeInTheDocument();
	});
});
