// Tags screen of plan 8.7.5: loading, error, empty and populated states, create, edit, delete.
import { screen, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import type { Tag } from '$lib/api/types.js';
import { fmt, t } from '$lib/i18n/index.js';
import { TagsStore } from '$lib/state/tags.svelte.js';
import { makeMe, makeTag } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import TagsPage from './TagsPage.svelte';

const user = () => userEvent.setup({ pointerEventsCheck: 0 });

const infra = (): Tag =>
	makeTag({
		id: 3,
		name: 'infra',
		color: 'ochre',
		linkCount: 12,
		activeLinkCount: 9,
		updatedAt: '2026-11-05T13:32:00.000Z'
	});
const docs = (): Tag =>
	makeTag({ id: 5, name: 'docs', color: 'blue', linkCount: 1, activeLinkCount: 1 });

function renderPage(store = new TagsStore(), me: ReturnType<typeof makeMe> | null = null) {
	renderWithApp(TagsPage, { store }, { me });
	return store;
}

beforeEach(() => {
	vi.spyOn(toast, 'success').mockImplementation(() => 1);
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('TagsPage states', () => {
	test('shows the loading rows while the first request runs', async () => {
		mockApi({ 'GET /api/tags': () => new Promise(() => {}) });
		renderPage();
		expect(await screen.findByRole('status')).toHaveTextContent(t('common.states.loading'));
		expect(screen.getByRole('heading', { level: 1, name: t('tags.pageTitle') })).toBeVisible();
		expect(screen.queryByRole('table')).toBeNull();
		expect(screen.queryByText(t('tags.summary', { count: 0 }))).toBeNull();
	});

	test('lists the tags in a table with counts, colour names and a filtered links link', async () => {
		mockApi({ 'GET /api/tags': json({ items: [infra(), docs()] }) });
		renderPage();
		const table = await screen.findByRole('table', { name: t('tags.table.caption') });
		const rows = within(table).getAllByTestId('tag-row');
		expect(rows.map((row) => within(row).getAllByRole('cell')[0].textContent?.trim())).toEqual([
			'docs',
			'infra'
		]);
		expect(screen.getByText(t('tags.summary', { count: 2 }))).toBeInTheDocument();
		const row = rows[1];
		expect(within(row).getByText(t('tagColors.ochre'))).toBeInTheDocument();
		const link = within(row).getByRole('link', {
			name: t('tags.table.showLinks', { count: 12, name: 'infra' })
		});
		expect(link).toHaveAttribute('href', '/?tags=3&archived=include');
		expect(link).toHaveTextContent('12');
		expect(within(row).getByText('9')).toBeInTheDocument();
		const time = row.querySelector('time');
		expect(time?.textContent?.trim()).toBe(fmt.date('2026-11-05T13:32:00.000Z'));
		expect(time).toHaveAttribute('datetime', '2026-11-05T13:32:00.000Z');
		expect(
			within(row).getByRole('button', { name: t('tags.table.edit', { name: 'infra' }) })
		).toBeInTheDocument();
	});

	test('shows the empty state when there is no tag', async () => {
		mockApi({ 'GET /api/tags': json({ items: [] }) });
		renderPage();
		expect(await screen.findByText(t('tags.empty.title'))).toBeInTheDocument();
		expect(screen.getByText(t('tags.empty.text'))).toBeInTheDocument();
		expect(screen.getByText(t('tags.summary', { count: 0 }))).toBeInTheDocument();
		expect(screen.queryByRole('table')).toBeNull();
	});

	test('shows the error with a retry that loads again', async () => {
		const api = mockApi({ 'GET /api/tags': apiError(500, 'internal_error') });
		renderPage();
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(t('errors.internal_error.title'));
		api.route('GET /api/tags', json({ items: [docs()] }));
		await user().click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		expect(await screen.findByRole('table')).toBeInTheDocument();
		expect(api.calls('GET /api/tags')).toHaveLength(2);
	});

	test('a signed-in visit reuses the per-user load of the store', async () => {
		const api = mockApi({ 'GET /api/tags': json({ items: [docs()] }) });
		const store = new TagsStore();
		await store.ensure(1);
		renderPage(store, makeMe());
		expect(await screen.findByRole('table')).toBeInTheDocument();
		expect(api.calls('GET /api/tags')).toHaveLength(1);
	});
});

describe('TagsPage actions', () => {
	test('adds a tag with the chosen colour', async () => {
		const api = mockApi({
			'GET /api/tags': json({ items: [infra()] }),
			'POST /api/tags': (req) =>
				json(makeTag({ id: 9, ...(req.body as { name: string; color: Tag['color'] }) }), 201)
		});
		renderPage();
		await screen.findByRole('table');
		const u = user();
		await u.click(screen.getByRole('button', { name: t('tags.add') }));
		const dialog = await screen.findByRole('dialog');
		expect(within(dialog).getByText(t('tags.dialog.createTitle'))).toBeInTheDocument();
		await u.type(within(dialog).getByLabelText(t('tags.dialog.name')), '  ops  ');
		await u.click(within(dialog).getByRole('radio', { name: t('tagColors.green') }));
		await u.click(within(dialog).getByRole('button', { name: t('tags.dialog.save') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
		expect(api.calls('POST /api/tags')[0].body).toEqual({ name: 'ops', color: 'green' });
		expect(toast.success).toHaveBeenCalledWith(t('tags.dialog.saved'));
		expect(screen.getAllByTestId('tag-row')).toHaveLength(2);
		expect(screen.getByText(t('tags.summary', { count: 2 }))).toBeInTheDocument();
	});

	test('edits a tag and sends only the changed fields', async () => {
		const api = mockApi({
			'GET /api/tags': json({ items: [infra()] }),
			'PATCH /api/tags/3': (req) => json({ ...infra(), ...(req.body as object) })
		});
		renderPage();
		await screen.findByRole('table');
		const u = user();
		await u.click(screen.getByRole('button', { name: t('tags.table.edit', { name: 'infra' }) }));
		const dialog = await screen.findByRole('dialog');
		expect(within(dialog).getByText(t('tags.dialog.editTitle'))).toBeInTheDocument();
		const input = within(dialog).getByLabelText(t('tags.dialog.name'));
		expect(input).toHaveValue('infra');
		expect(within(dialog).getByRole('radio', { name: t('tagColors.ochre') })).toHaveAttribute(
			'aria-checked',
			'true'
		);
		await u.clear(input);
		await u.type(input, 'platform');
		await u.click(within(dialog).getByRole('button', { name: t('tags.dialog.save') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
		expect(api.calls('PATCH /api/tags/3')[0].body).toEqual({ name: 'platform' });
		expect(screen.getByText('platform')).toBeInTheDocument();
	});

	test('deletes a tag after the confirmation naming it and its link count', async () => {
		const api = mockApi({
			'GET /api/tags': json({ items: [infra(), docs()] }),
			'DELETE /api/tags/3': noContent()
		});
		renderPage();
		await screen.findByRole('table');
		const u = user();
		await u.click(screen.getByRole('button', { name: t('tags.table.delete', { name: 'infra' }) }));
		const dialog = await screen.findByRole('alertdialog');
		expect(dialog).toHaveTextContent(t('tags.delete.title'));
		expect(dialog).toHaveTextContent(t('tags.delete.text', { name: 'infra', count: 12 }));
		await u.click(within(dialog).getByRole('button', { name: t('tags.delete.confirm') }));
		await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull());
		expect(api.calls('DELETE /api/tags/3')).toHaveLength(1);
		expect(toast.success).toHaveBeenCalledWith(t('tags.delete.deleted'));
		expect(screen.getAllByTestId('tag-row')).toHaveLength(1);
	});

	test('a failed delete stays in the dialog with the error', async () => {
		mockApi({
			'GET /api/tags': json({ items: [infra()] }),
			'DELETE /api/tags/3': apiError(404, 'not_found')
		});
		renderPage();
		await screen.findByRole('table');
		const u = user();
		await u.click(screen.getByRole('button', { name: t('tags.table.delete', { name: 'infra' }) }));
		const dialog = await screen.findByRole('alertdialog');
		await u.click(within(dialog).getByRole('button', { name: t('tags.delete.confirm') }));
		expect(await within(dialog).findByText(t('errors.not_found.title'))).toBeInTheDocument();
		expect(screen.getAllByTestId('tag-row')).toHaveLength(1);
	});

	test('the delete text has its own wording for an unused tag', () => {
		expect(t('tags.delete.text', { name: 'x', count: 0 })).not.toContain('0');
		expect(t('tags.delete.text', { name: 'x', count: 2 })).toContain('2');
	});
});
