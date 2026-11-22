// Add, edit and tag dialogs of the links page (plan 8.7.3).
import { screen } from '@testing-library/svelte';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import type { LinkItem } from '$lib/api/types.js';
import { errorText, errorTitle, t } from '$lib/i18n/index.js';
import { linkStats } from '$lib/state/link-stats.svelte.js';
import { tags } from '$lib/state/tags.svelte.js';
import { makeLink, makeMe, makeTag } from '$lib/test/fixtures.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import LinkDialog from './LinkDialog.svelte';
import LinkTagsDialog from './LinkTagsDialog.svelte';
import { getFloating, makeDetail, makeStore, user } from './test-helpers.js';

vi.mock('svelte-sonner', () => ({
	toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn(), info: vi.fn() })
}));

beforeEach(() => {
	vi.mocked(toast.success).mockClear();
	vi.mocked(toast.error).mockClear();
	vi.spyOn(linkStats, 'refresh').mockResolvedValue();
	vi.spyOn(tags, 'refresh').mockResolvedValue().mockClear();
	tags.reset();
});

afterEach(() => {
	vi.unstubAllGlobals();
	tags.reset();
});

function openDialog(link: LinkItem | null = null) {
	const { store } = makeStore();
	const onShowExisting = vi.fn();
	const result = renderWithApp(
		LinkDialog,
		{ open: true, link, store, onShowExisting },
		{ me: makeMe() }
	);
	return { ...result, store, onShowExisting };
}

const urlField = () => screen.getByLabelText(t('links.dialog.url'));
const titleField = () => screen.getByLabelText(t('links.dialog.title'));
const descriptionField = () => screen.getByLabelText(t('links.dialog.description'));
const save = () => screen.getByRole('button', { name: t('links.dialog.save') });

describe('LinkDialog: adding', () => {
	test('saves a new link with every field and closes', async () => {
		const created = makeDetail(makeLink({ id: 41, pinned: true, fetchStatus: 'pending' }));
		const api = mockApi({ 'POST /api/links': json(created, 201) });
		const { store } = openDialog();
		const watch = vi.spyOn(store, 'watchPending').mockImplementation(() => {});
		expect(screen.getByRole('heading', { name: t('links.dialog.addTitle') })).toBeInTheDocument();
		const u = user();
		await u.type(urlField(), 'https://grafana.example.net');
		await u.type(titleField(), 'Grafana');
		await u.type(descriptionField(), 'Éles');
		expect(screen.getByTestId('description-counter')).toHaveTextContent(
			t('links.dialog.counter', { count: 4 })
		);
		await u.click(screen.getByLabelText(t('links.dialog.pin')));
		await u.click(save());
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.saved')));
		expect(api.calls('POST /api/links')[0].body).toEqual({
			url: 'https://grafana.example.net',
			title: 'Grafana',
			description: 'Éles',
			tagIds: [],
			pinned: true
		});
		expect(store.find(41)).toBeDefined();
		expect(store.highlightId).toBe(41);
		expect(watch).toHaveBeenCalledWith(41);
		expect(linkStats.refresh).toHaveBeenCalled();
		// Without tags no tag count changes.
		expect(tags.refresh).not.toHaveBeenCalled();
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});

	test('a new link with tags refreshes the tag counts', async () => {
		tags.items = [makeTag({ id: 3, name: 'docs' })];
		const created = makeDetail(makeLink({ id: 42, tagIds: [3] }));
		const api = mockApi({ 'POST /api/links': json(created, 201) });
		openDialog();
		const u = user();
		await u.type(urlField(), 'https://docs.example.net');
		await u.click(screen.getByRole('button', { name: t('common.tagPicker.label') }));
		await u.click(getFloating('option', 'docs'));
		await u.click(save());
		await vi.waitFor(() => expect(api.calls('POST /api/links')).toHaveLength(1));
		expect(api.calls('POST /api/links')[0].body).toMatchObject({ tagIds: [3] });
		await vi.waitFor(() => expect(tags.refresh).toHaveBeenCalled());
	});

	test('requires the URL without sending a request', async () => {
		const api = mockApi({});
		openDialog();
		await user().click(save());
		expect(urlField()).toHaveAttribute('aria-invalid', 'true');
		await vi.waitFor(() => expect(urlField()).toHaveFocus());
		expect(api.requests).toHaveLength(0);
	});

	test('shows the server field errors and focuses the first invalid field', async () => {
		mockApi({
			'POST /api/links': apiError(422, 'validation_failed', {
				description: 'too_long',
				title: 'too_long'
			})
		});
		openDialog();
		const u = user();
		await u.type(urlField(), 'https://a.io');
		await u.click(save());
		await vi.waitFor(() => expect(titleField()).toHaveAttribute('aria-invalid', 'true'));
		expect(descriptionField()).toHaveAttribute('aria-invalid', 'true');
		await vi.waitFor(() => expect(titleField()).toHaveFocus());
	});

	test('warns about a duplicate and shows the existing link', async () => {
		mockApi({
			'POST /api/links': apiError(409, 'link_duplicate', undefined, { linkId: 12 })
		});
		const { onShowExisting } = openDialog();
		const u = user();
		await u.type(urlField(), 'https://a.io');
		await u.click(save());
		const title = errorTitle({ code: 'link_duplicate', details: {} });
		expect(await screen.findByText(title)).toBeInTheDocument();
		expect(
			screen.getByText(errorText({ code: 'link_duplicate', details: {} }))
		).toBeInTheDocument();
		await u.click(screen.getByRole('button', { name: t('links.dialog.showExisting') }));
		expect(onShowExisting).toHaveBeenCalledWith(12);
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});

	test('a duplicate without a link id and other errors show an error alert', async () => {
		mockApi({ 'POST /api/links': apiError(409, 'link_duplicate') });
		openDialog();
		const u = user();
		await u.type(urlField(), 'https://a.io');
		await u.click(save());
		expect(await screen.findByRole('alert')).toBeInTheDocument();
		expect(screen.queryByRole('button', { name: t('links.dialog.showExisting') })).toBeNull();
	});

	test('a server error keeps the dialog open with an alert', async () => {
		mockApi({ 'POST /api/links': apiError(500, 'internal_error') });
		openDialog();
		const u = user();
		await u.type(urlField(), 'https://a.io');
		await u.click(save());
		expect(await screen.findByRole('alert')).toHaveTextContent(
			errorTitle({ code: 'internal_error', details: {} })
		);
		expect(screen.getByRole('dialog')).toBeInTheDocument();
	});

	test('cancel closes without a request', async () => {
		const api = mockApi({});
		openDialog();
		await user().click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
		expect(api.requests).toHaveLength(0);
	});
});

describe('LinkDialog: GRAPHICS.md fields (10.9) and the plan 8.7.3 description', () => {
	test('fields are radius-md and transparent; the description shows three rows', () => {
		openDialog();
		for (const field of [urlField(), titleField(), descriptionField()]) {
			expect(field).toHaveClass('rounded-md', 'bg-transparent!');
			expect(field).not.toHaveClass('rounded-lg');
		}
		// rows=3 has no effect with field-sizing-content: the minimum height keeps three lines.
		expect(descriptionField()).toHaveAttribute('rows', '3');
		expect(descriptionField()).toHaveClass('min-h-20');
		expect(descriptionField()).not.toHaveClass('min-h-0');
	});
});

describe('LinkDialog: editing', () => {
	test('sends only the changed fields and keeps the tags', async () => {
		const link = makeLink({ id: 5, title: 'Old', description: 'Desc', tagIds: [1] });
		const api = mockApi({
			'PATCH /api/links/5': json(makeDetail({ ...link, title: 'New' }))
		});
		const { store } = openDialog(link);
		store.items = [link];
		expect(screen.getByRole('heading', { name: t('links.dialog.editTitle') })).toBeInTheDocument();
		expect(urlField()).toHaveValue(link.url);
		expect(screen.queryByLabelText(t('links.dialog.pin'))).toBeNull();
		const u = user();
		await u.clear(titleField());
		await u.type(titleField(), 'New');
		await u.click(save());
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.saved')));
		expect(api.calls('PATCH /api/links/5')[0].body).toEqual({ title: 'New' });
		expect(api.calls('PUT /api/links/5/tags')).toHaveLength(0);
		expect(store.find(5)?.title).toBe('New');
	});

	test('the URL field shows the readable URL; saving it unchanged sends no URL (E2)', async () => {
		const url = 'https://b%C3%BCcher.example.invalid/stra%C3%9Fe/%C3%B6ffnungszeiten?ő=ű';
		const link = makeLink({ id: 5, url, title: 'Old' });
		const api = mockApi({ 'PATCH /api/links/5': json(makeDetail({ ...link, title: 'New' })) });
		openDialog(link);
		expect(urlField()).toHaveValue('https://bücher.example.invalid/straße/öffnungszeiten?ő=ű');
		const u = user();
		await u.clear(titleField());
		await u.type(titleField(), 'New');
		await u.click(save());
		await vi.waitFor(() => expect(api.calls('PATCH /api/links/5')).toHaveLength(1));
		expect(api.calls('PATCH /api/links/5')[0].body).toEqual({ title: 'New' });
	});

	test('a pending URL change starts the metadata polling', async () => {
		const link = makeLink({ id: 5 });
		mockApi({
			'PATCH /api/links/5': json(
				makeDetail({ ...link, url: 'https://b.io', fetchStatus: 'pending' })
			)
		});
		const { store } = openDialog(link);
		const watch = vi.spyOn(store, 'watchPending').mockImplementation(() => {});
		const u = user();
		await u.clear(urlField());
		await u.type(urlField(), 'https://b.io');
		await u.click(save());
		await vi.waitFor(() => expect(watch).toHaveBeenCalledWith(5));
	});

	test('saving without changes sends nothing', async () => {
		const api = mockApi({});
		openDialog(makeLink({ id: 5 }));
		await user().click(save());
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalled());
		expect(api.requests).toHaveLength(0);
	});

	test('changed tags are saved separately', async () => {
		tags.items = [makeTag({ id: 1, name: 'infra' }), makeTag({ id: 2, name: 'docs' })];
		const link = makeLink({ id: 5, tagIds: [1] });
		const api = mockApi({ 'PUT /api/links/5/tags': json({ ...link, tagIds: [1, 2] }) });
		openDialog(link);
		const u = user();
		await u.click(screen.getByRole('button', { name: t('common.tagPicker.label') }));
		await u.click(getFloating('option', 'docs'));
		await u.click(save());
		await vi.waitFor(() => expect(api.calls('PUT /api/links/5/tags')).toHaveLength(1));
		expect(api.calls('PUT /api/links/5/tags')[0].body).toEqual({ tagIds: [1, 2] });
		expect(api.calls('PATCH /api/links/5')).toHaveLength(0);
		// The sidebar's frequent tags follow the new counts.
		expect(tags.refresh).toHaveBeenCalled();
	});

	test('a saved PATCH reaches the list even when the tag update fails', async () => {
		tags.items = [makeTag({ id: 1, name: 'infra' }), makeTag({ id: 2, name: 'docs' })];
		const link = makeLink({ id: 5, title: 'Old', tagIds: [1] });
		const api = mockApi({
			'PATCH /api/links/5': json(makeDetail({ ...link, title: 'New' })),
			'PUT /api/links/5/tags': apiError(500, 'internal_error')
		});
		const { store } = openDialog(link);
		store.items = [link];
		const u = user();
		await u.clear(titleField());
		await u.type(titleField(), 'New');
		await u.click(screen.getByRole('button', { name: t('common.tagPicker.label') }));
		await u.click(getFloating('option', 'docs'));
		await u.click(save());
		await vi.waitFor(() => expect(api.calls('PUT /api/links/5/tags')).toHaveLength(1));
		await vi.waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument());
		expect(api.calls('PATCH /api/links/5')[0].body).toEqual({ title: 'New' });
		expect(toast.success).not.toHaveBeenCalled();
		expect(store.find(5)?.title).toBe('New');
	});
});

describe('LinkTagsDialog', () => {
	function openTags(link: LinkItem) {
		const { store } = makeStore();
		store.items = [link];
		renderWithApp(LinkTagsDialog, { open: true, link, store }, { me: makeMe() });
		return store;
	}

	test('a long label truncates inside the 440 px box; the footer buttons stay (E2)', async () => {
		const long =
			'A Kubernetes hivatalos dokumentacioja kezdoknek, haladoknak es uzemeltetoknek '.repeat(7);
		openTags(makeLink({ id: 9, description: long }));
		const dialog = await screen.findByRole('dialog');
		const label = screen.getByTestId('tags-dialog-label');
		expect(label).toHaveClass('block', 'min-w-0', 'truncate');
		expect(dialog).toHaveClass('grid-cols-[minmax(0,1fr)]', '*:min-w-0');
		expect(label.closest('[data-slot="dialog-header"]')).toHaveClass('min-w-0');
		expect(screen.getByRole('button', { name: t('links.tagsDialog.save') })).toBeInTheDocument();
	});

	test('saves the personal tags of a shared link', async () => {
		tags.items = [makeTag({ id: 1, name: 'infra' })];
		const link = makeLink({ id: 9, isOwner: false, title: 'Shared', description: '' });
		const api = mockApi({ 'PUT /api/links/9/tags': json({ ...link, tagIds: [1] }) });
		const store = openTags(link);
		expect(screen.getByRole('dialog')).toHaveTextContent('Shared');
		const u = user();
		await u.click(screen.getByRole('button', { name: t('common.tagPicker.label') }));
		await u.click(getFloating('option', 'infra'));
		await u.click(screen.getByRole('button', { name: t('links.tagsDialog.save') }));
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.tagsSaved')));
		expect(api.calls('PUT /api/links/9/tags')[0].body).toEqual({ tagIds: [1] });
		expect(store.find(9)?.tagIds).toEqual([1]);
		expect(tags.refresh).toHaveBeenCalled();
	});

	test('shows a field error and a general error', async () => {
		const link = makeLink({ id: 9 });
		const api = mockApi({
			'PUT /api/links/9/tags': apiError(422, 'validation_failed', { tagIds: 'out_of_range' })
		});
		openTags(link);
		const u = user();
		await u.click(screen.getByRole('button', { name: t('links.tagsDialog.save') }));
		await vi.waitFor(() => expect(document.getElementById('link-tags-error')).not.toBeNull());
		api.route('PUT /api/links/9/tags', apiError(500, 'internal_error'));
		await u.click(screen.getByRole('button', { name: t('links.tagsDialog.save') }));
		expect(await screen.findByRole('alert')).toBeInTheDocument();
		await u.click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});
});
