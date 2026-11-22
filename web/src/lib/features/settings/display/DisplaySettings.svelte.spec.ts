// Settings > Display (plan 8.7.6): every control saves at once through the preferences store.
import { screen, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { setMode } from 'mode-watcher';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import type { Preferences } from '$lib/api/types.js';
import { i18n, t } from '$lib/i18n/index.js';
import { PreferencesStore } from '$lib/state/preferences.svelte.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe, makePreferences } from '$lib/test/fixtures.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import DisplaySettings from './DisplaySettings.svelte';
import { normalizeSpace } from '$lib/test/text.js';

vi.mock('mode-watcher', async (importOriginal) => ({
	...(await importOriginal<typeof import('mode-watcher')>()),
	setMode: vi.fn()
}));

const user = () => userEvent.setup({ pointerEventsCheck: 0 });

function option(text: string): HTMLElement {
	const found = screen
		.queryAllByRole('option', { hidden: true })
		.find((el) => el.textContent?.trim() === text);
	if (!found) throw new Error(`no option "${text}"`);
	return found;
}

let store: PreferencesStore;

function setup(partial: Partial<Preferences> = {}) {
	const api = mockApi({
		'PATCH /api/me/preferences': (req) =>
			json({ ...session.me?.preferences, ...(req.body as object) })
	});
	store = new PreferencesStore(session);
	const update = vi.spyOn(store, 'update');
	renderWithApp(
		DisplaySettings,
		{ store },
		{ me: makeMe('developer', { preferences: makePreferences(partial) }) }
	);
	return { api, update };
}

beforeEach(() => {
	vi.mocked(setMode).mockClear();
});

afterEach(async () => {
	await store?.flush();
	vi.unstubAllGlobals();
});

describe('DisplaySettings', () => {
	test('shows loading rows before the preferences exist', () => {
		store = new PreferencesStore(session);
		renderWithApp(DisplaySettings, { store }, { me: null });
		expect(screen.getByRole('status')).toHaveTextContent(t('common.states.loading'));
		expect(screen.queryByRole('switch')).toBeNull();
	});

	test('reflects the stored preferences in three cards', () => {
		setup({ theme: 'light', viewMode: 'tiles', sort: 'title', tagMatch: 'all', showUrl: false });
		for (const key of [
			'settings.display.appearance.title',
			'settings.display.list.title',
			'settings.display.archive.title'
		] as const) {
			expect(screen.getByRole('heading', { level: 2, name: t(key) })).toBeInTheDocument();
		}
		const checked = (group: string, name: string) =>
			within(screen.getByRole('group', { name: group })).getByRole('radio', { name });
		expect(
			checked(t('settings.display.theme.label'), t('settings.display.theme.light'))
		).toHaveAttribute('aria-checked', 'true');
		expect(
			checked(t('settings.display.viewMode.label'), t('settings.display.viewMode.tiles'))
		).toHaveAttribute('aria-checked', 'true');
		expect(
			checked(t('settings.display.sort.label'), t('settings.display.sort.title'))
		).toHaveAttribute('aria-checked', 'true');
		expect(
			checked(t('settings.display.tagMatch.label'), t('settings.display.tagMatch.all'))
		).toHaveAttribute('aria-checked', 'true');
		expect(screen.getByRole('switch', { name: t('settings.display.fields.url') })).toHaveAttribute(
			'aria-checked',
			'false'
		);
		expect(
			screen.getByRole('switch', { name: t('settings.display.archived.toggle') })
		).toHaveAttribute('aria-checked', 'false');
		expect(
			screen.getByRole('combobox', { name: t('settings.display.language.label') })
		).toHaveTextContent(t('nav.locale.hu'));
	});

	test('the theme applies at once and is saved in one request', async () => {
		const { api, update } = setup();
		await user().click(screen.getByRole('radio', { name: t('settings.display.theme.light') }));
		expect(update).toHaveBeenCalledWith({ theme: 'light' });
		expect(setMode).toHaveBeenLastCalledWith('light');
		await store.flush();
		expect(api.calls('PATCH /api/me/preferences')[0].body).toEqual({ theme: 'light' });
	});

	test('clicking the chosen option again changes nothing', async () => {
		const { update } = setup({ theme: 'dark' });
		const dark = screen.getByRole('radio', { name: t('settings.display.theme.dark') });
		await user().click(dark);
		expect(update).not.toHaveBeenCalled();
		expect(dark).toHaveAttribute('aria-checked', 'true');
	});

	test('view, sort and tag match save their values', async () => {
		const { update } = setup();
		const u = user();
		await u.click(screen.getByRole('radio', { name: t('settings.display.viewMode.tiles') }));
		await u.click(screen.getByRole('radio', { name: t('settings.display.sort.created') }));
		await u.click(screen.getByRole('radio', { name: t('settings.display.tagMatch.all') }));
		expect(update.mock.calls).toEqual([
			[{ viewMode: 'tiles' }],
			[{ sort: 'created' }],
			[{ tagMatch: 'all' }]
		]);
		expect(screen.getByRole('radio', { name: t('settings.display.sort.created') })).toHaveAttribute(
			'aria-checked',
			'true'
		);
	});

	test('the field switches save and reshape the preview', async () => {
		const { update } = setup();
		const preview = screen.getByTestId('display-preview');
		expect(within(preview).getByTestId('preview-label')).toHaveTextContent(
			t('settings.display.preview.description')
		);
		expect(within(preview).getByTestId('preview-title')).toHaveTextContent(
			t('settings.display.preview.title')
		);
		expect(within(preview).getByTestId('preview-url')).toBeInTheDocument();
		expect(preview.querySelector('svg')).not.toBeNull();

		const u = user();
		await u.click(screen.getByRole('switch', { name: t('settings.display.fields.description') }));
		expect(update).toHaveBeenLastCalledWith({ showDescription: false });
		expect(within(preview).getByTestId('preview-label')).toHaveTextContent(
			t('settings.display.preview.title')
		);
		expect(within(preview).queryByTestId('preview-title')).toBeNull();

		await u.click(screen.getByRole('switch', { name: t('settings.display.fields.title') }));
		await u.click(screen.getByRole('switch', { name: t('settings.display.fields.icon') }));
		expect(within(preview).getByTestId('preview-label')).toHaveTextContent(
			t('settings.display.preview.url')
		);
		expect(within(preview).queryByTestId('preview-url')).toBeNull();
		expect(preview.querySelector('svg')).toBeNull();
		expect(update).toHaveBeenCalledWith({ showIcon: false });
	});

	test('the archived switch saves the default', async () => {
		const { update } = setup();
		const archived = screen.getByRole('switch', { name: t('settings.display.archived.toggle') });
		await user().click(archived);
		expect(update).toHaveBeenCalledWith({ showArchived: true });
		expect(archived).toHaveAttribute('aria-checked', 'true');
	});

	test('the language select switches the interface language', async () => {
		const { update } = setup();
		const u = user();
		await u.click(screen.getByRole('combobox', { name: t('settings.display.language.label') }));
		await vi.waitFor(() => option('English'));
		await u.click(option('English'));
		expect(update).toHaveBeenCalledWith({ locale: 'en' });
		expect(i18n.locale).toBe('en');
		expect(await screen.findByText(t('settings.display.intro'))).toBeInTheDocument();
		expect(t('settings.display.intro')).toMatch(/^Every change/);
	});

	test('choosing the current language again saves nothing', async () => {
		const { update } = setup();
		const u = user();
		await u.click(screen.getByRole('combobox', { name: t('settings.display.language.label') }));
		await vi.waitFor(() => option('Magyar'));
		await u.click(option('Magyar'));
		expect(update).not.toHaveBeenCalled();
	});

	test('the archiving period saves through the preferences', async () => {
		const { update } = setup({ archiveAfterDays: null, systemArchiveAfterDays: 90 });
		expect(
			screen.getByText(normalizeSpace(t('settings.display.archive.current', { count: 90 })))
		).toBeInTheDocument();
		const u = user();
		await u.click(screen.getByRole('combobox', { name: t('settings.display.archive.label') }));
		await vi.waitFor(() => option(t('settings.display.archive.never')));
		await u.click(option(t('settings.display.archive.never')));
		expect(update).toHaveBeenCalledWith({ archiveAfterDays: 0 });
		expect(
			screen.getByText(t('settings.display.archive.current', { count: 0 }))
		).toBeInTheDocument();
	});

	test('a failed save reverts the control and shows a toast', async () => {
		const error = vi.spyOn(toast, 'error').mockImplementation(() => 1);
		const { api } = setup({ viewMode: 'list' });
		api.route('PATCH /api/me/preferences', apiError(500, 'internal_error'));
		const tiles = screen.getByRole('radio', { name: t('settings.display.viewMode.tiles') });
		await user().click(tiles);
		expect(tiles).toHaveAttribute('aria-checked', 'true');
		await store.flush();
		await vi.waitFor(() => expect(tiles).toHaveAttribute('aria-checked', 'false'));
		expect(error).toHaveBeenCalledOnce();
	});
});
