// Import, export and the keyboard shortcuts dialog (plan 8.7.3).
import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { errorMessage, fmt, i18n, t } from '$lib/i18n/index.js';
import { linkStats } from '$lib/state/link-stats.svelte.js';
import { tags } from '$lib/state/tags.svelte.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import { saveBlob } from '$lib/utils/download.js';
import ImportExportMenu, { IMPORT_MAX_BYTES } from './ImportExportMenu.svelte';
import ShortcutsDialog, { SHORTCUTS } from './ShortcutsDialog.svelte';
import { findFloating, makeStore, user } from './test-helpers.js';

vi.mock('$lib/utils/download.js', () => ({ saveBlob: vi.fn(), saveText: vi.fn() }));

vi.mock('svelte-sonner', () => ({
	toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn() })
}));

beforeEach(() => {
	vi.mocked(toast.success).mockClear();
	vi.mocked(toast.error).mockClear();
	vi.spyOn(linkStats, 'refresh').mockResolvedValue();
	vi.spyOn(tags, 'load').mockResolvedValue();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

function setup() {
	const { store } = makeStore();
	const reload = vi.spyOn(store, 'reload').mockResolvedValue();
	render(ImportExportMenu, { store });
	return { store, reload };
}

async function openMenu() {
	await user().click(screen.getByRole('button', { name: t('links.importExport.label') }));
}

function choose(file: File) {
	const input = screen.getByTestId('import-input') as HTMLInputElement;
	Object.defineProperty(input, 'files', { configurable: true, value: [file] });
	return fireEvent.change(input);
}

describe('ImportExportMenu', () => {
	test.each([
		['json', 'links.importExport.exportJson'],
		['html', 'links.importExport.exportHtml']
	] as const)('exports %s and saves the file', async (format, label) => {
		vi.mocked(saveBlob).mockClear();
		const api = mockApi({
			'GET /api/links/export': {
				status: 200,
				body: new Blob(['data'], { type: 'application/octet-stream' }),
				headers: { 'Content-Disposition': `attachment; filename="kinex-mark-links.${format}"` }
			}
		});
		setup();
		await openMenu();
		await user().click(await findFloating('menuitem', t(label)));
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('links.toast.exported')));
		expect(api.calls('GET /api/links/export')[0].url.searchParams.get('format')).toBe(format);
		expect(saveBlob).toHaveBeenCalledWith(expect.anything(), `kinex-mark-links.${format}`);
	});

	test('a failed export shows an error toast', async () => {
		mockApi({ 'GET /api/links/export': apiError(500, 'internal_error') });
		setup();
		await openMenu();
		await user().click(await findFloating('menuitem', t('links.importExport.exportJson')));
		await vi.waitFor(() =>
			expect(toast.error).toHaveBeenCalledWith(
				errorMessage({ code: 'internal_error', details: {} })
			)
		);
	});

	test('the import item opens the file picker', async () => {
		setup();
		const click = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {});
		await openMenu();
		await user().click(await findFloating('menuitem', t('links.importExport.import')));
		expect(click).toHaveBeenCalled();
	});

	test('imports a file and shows the result', async () => {
		const api = mockApi({
			'POST /api/links/import': json({ total: 10, created: 7, duplicates: 2, invalid: 1 })
		});
		const { reload } = setup();
		await choose(new File(['{}'], 'links.json', { type: 'application/json' }));
		const dialog = await screen.findByRole('dialog');
		expect(
			within(dialog).getByRole('heading', { name: t('links.importExport.resultTitle') })
		).toBeInTheDocument();
		expect(dialog).toHaveTextContent(t('links.importExport.created'));
		expect(dialog).toHaveTextContent(fmt.number(7));
		expect(api.calls('POST /api/links/import')[0].headers.get('Content-Type')).toContain(
			'application/json'
		);
		expect(reload).toHaveBeenCalled();
		expect(linkStats.refresh).toHaveBeenCalled();
		expect(tags.load).toHaveBeenCalled();
		await user().click(within(dialog).getByRole('button', { name: t('common.actions.close') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
		// The dialog opened from the file picker: the focus goes back to the menu button (plan 8.9).
		await vi.waitFor(() =>
			expect(screen.getByRole('button', { name: t('links.importExport.label') })).toHaveFocus()
		);
	});

	test('a file over 2 MiB is refused without a request', async () => {
		const api = mockApi({});
		setup();
		const big = new File(['x'], 'big.html', { type: 'text/html' });
		Object.defineProperty(big, 'size', { value: IMPORT_MAX_BYTES + 1 });
		await choose(big);
		expect(toast.error).toHaveBeenCalledWith(
			errorMessage({ code: 'payload_too_large', details: {} })
		);
		expect(api.requests).toHaveLength(0);
	});

	test('an invalid import shows an error toast; an empty choice does nothing', async () => {
		const api = mockApi({ 'POST /api/links/import': apiError(422, 'import_invalid') });
		const { reload } = setup();
		await choose(new File(['nope'], 'links.html', { type: 'text/html' }));
		await vi.waitFor(() =>
			expect(toast.error).toHaveBeenCalledWith(
				errorMessage({ code: 'import_invalid', details: {} })
			)
		);
		expect(reload).not.toHaveBeenCalled();
		const input = screen.getByTestId('import-input') as HTMLInputElement;
		Object.defineProperty(input, 'files', { configurable: true, value: [] });
		await fireEvent.change(input);
		expect(api.calls('POST /api/links/import')).toHaveLength(1);
	});

	test('a file over the entry limit names the limit instead of an unreadable file', async () => {
		mockApi({ 'POST /api/links/import': apiError(400, 'import_too_many_entries') });
		setup();
		await choose(new File(['many'], 'links.json', { type: 'application/json' }));
		await vi.waitFor(() =>
			expect(toast.error).toHaveBeenCalledWith(
				errorMessage({ code: 'import_too_many_entries', details: {} })
			)
		);
		expect(errorMessage({ code: 'import_too_many_entries', details: {} })).not.toEqual(
			errorMessage({ code: 'import_invalid', details: {} })
		);
	});
});

describe('ShortcutsDialog', () => {
	test('lists every shortcut and closes with the close button', async () => {
		render(ShortcutsDialog, { open: true });
		const dialog = await screen.findByRole('dialog');
		expect(
			within(dialog).getByRole('heading', { name: t('links.shortcuts.title') })
		).toBeInTheDocument();
		const rows = within(dialog).getAllByRole('row');
		expect(rows).toHaveLength(SHORTCUTS.length);
		for (const row of SHORTCUTS) {
			expect(dialog).toHaveTextContent(t(row.label));
		}
		expect(dialog.querySelector('[data-shortcut="next"]')).toHaveTextContent(
			t('links.shortcuts.or')
		);
		expect(dialog.querySelector('[data-shortcut="paste"]')).toHaveTextContent(/Ctrl|⌘/);
		await user().click(within(dialog).getByRole('button', { name: t('common.actions.close') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});

	test('German key names: Strg, Entf (D-close leftover 2)', async () => {
		vi.spyOn(navigator, 'platform', 'get').mockReturnValue('Win32');
		i18n.setLocale('de', { persist: false });
		try {
			render(ShortcutsDialog, { open: true });
			const dialog = await screen.findByRole('dialog');
			expect(dialog.querySelector('[data-shortcut="paste"]')).toHaveTextContent('Strg');
			expect(dialog.querySelector('[data-shortcut="delete"]')).toHaveTextContent('Entf');
			expect(dialog.querySelector('[data-shortcut="delete"]')).not.toHaveTextContent('Delete');
			expect(dialog.querySelector('[data-shortcut="clear"]')).toHaveTextContent('Esc');
		} finally {
			i18n.setLocale('hu', { persist: false });
		}
	});

	test('renders nothing while closed', () => {
		render(ShortcutsDialog, {});
		expect(screen.queryByRole('dialog')).toBeNull();
	});
});
