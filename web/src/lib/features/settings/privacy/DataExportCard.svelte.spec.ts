// Data export card of Settings > Privacy (plan 8.7.6, GDPR access and portability).
import { screen, waitFor, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { toast } from 'svelte-sonner';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { t } from '$lib/i18n/index.js';
import { saveBlob } from '$lib/utils/download.js';
import { makeMe } from '$lib/test/fixtures.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import DataExportCard from './DataExportCard.svelte';

vi.mock('$lib/utils/download.js', () => ({ saveText: vi.fn(), saveBlob: vi.fn() }));

afterEach(() => {
	vi.mocked(saveBlob).mockClear();
	vi.unstubAllGlobals();
});

describe('DataExportCard', () => {
	test('lists the contents and saves the export under the server file name', async () => {
		const filename = 'kinex-mark-export-kanna-20261121.json';
		mockApi({
			'GET /api/me/export': json({ format: 'kinex-mark-export/1' }, 200, {
				'Content-Disposition': `attachment; filename="${filename}"`
			})
		});
		const success = vi.spyOn(toast, 'success');
		renderWithApp(DataExportCard, {}, { me: makeMe() });
		const card = screen.getByRole('region', { name: t('security.privacy.export.title') });
		expect(within(card).getAllByRole('listitem')).toHaveLength(8);
		expect(within(card).getByText(t('security.privacy.export.items.audit'))).toBeInTheDocument();
		expect(within(card).getByText(t('security.privacy.export.excluded'))).toBeInTheDocument();

		await userEvent.click(
			within(card).getByRole('button', { name: t('security.privacy.export.submit') })
		);
		await waitFor(() => expect(saveBlob).toHaveBeenCalledOnce());
		const [blob, name] = vi.mocked(saveBlob).mock.calls[0];
		expect(name).toBe(filename);
		expect(JSON.parse(await blob.text())).toEqual({ format: 'kinex-mark-export/1' });
		expect(success).toHaveBeenCalledWith(t('security.privacy.export.done'));
	});

	test('a failed export shows an alert and can be repeated', async () => {
		const api = mockApi({ 'GET /api/me/export': apiError(500, 'internal_error') });
		renderWithApp(DataExportCard, {}, { me: makeMe() });
		const button = screen.getByRole('button', { name: t('security.privacy.export.submit') });
		await userEvent.click(button);
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.internal_error.title'));
		expect(saveBlob).not.toHaveBeenCalled();

		api.route('GET /api/me/export', json({}));
		await userEvent.click(button);
		await waitFor(() => expect(screen.queryByRole('alert')).not.toBeInTheDocument());
		expect(saveBlob).toHaveBeenCalledOnce();
	});
});
