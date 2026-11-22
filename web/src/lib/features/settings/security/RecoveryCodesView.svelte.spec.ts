// One-time view of recovery codes: copy, .txt download, print and the saved checkbox (plan 8.7.6).
import { screen, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { toast } from 'svelte-sonner';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { fmt, i18n, t } from '$lib/i18n/index.js';
import { saveText } from '$lib/utils/download.js';
import { renderWithApp } from '$lib/test/render.js';
import { makeMe, makePreferences } from '$lib/test/fixtures.js';
import RecoveryCodesView, { PRINT_CLASS, recoveryCodesFilename } from './RecoveryCodesView.svelte';

vi.mock('$lib/utils/download.js', () => ({ saveText: vi.fn(), saveBlob: vi.fn() }));

const user = () => userEvent.setup({ pointerEventsCheck: 0 });
const CODES = ['abcde-fghjk', 'mnpqr-stuvw', 'xyz23-45678'];
const AT = new Date('2026-11-21T10:15:00.000Z');

afterEach(() => {
	vi.mocked(saveText).mockClear();
	document.documentElement.classList.remove(PRINT_CLASS);
});

describe('RecoveryCodesView', () => {
	test('lists the codes in two mono columns with a print-only heading', () => {
		renderWithApp(
			RecoveryCodesView,
			{ codes: CODES, username: 'kanna', generatedAt: AT },
			{ me: makeMe() }
		);
		const list = screen.getByRole('list', { name: t('security.recovery.listLabel') });
		expect(list).toHaveClass('grid-cols-2', 'font-mono');
		expect(
			within(list)
				.getAllByRole('listitem')
				.map((li) => li.textContent)
		).toEqual(CODES);
		expect(
			screen.getByText(t('security.recovery.printHeading', { username: 'kanna' }))
		).toHaveClass('kxm-print-only');
	});

	test('copies all codes, one per line', async () => {
		const success = vi.spyOn(toast, 'success');
		renderWithApp(
			RecoveryCodesView,
			{ codes: CODES, username: 'kanna', generatedAt: AT },
			{ me: makeMe() }
		);
		await user().click(screen.getByRole('button', { name: t('security.recovery.copy') }));
		expect(await navigator.clipboard.readText()).toBe(CODES.join('\n'));
		expect(success).toHaveBeenCalledWith(t('security.recovery.copied'));
	});

	test('downloads a text file in the active language', async () => {
		renderWithApp(
			RecoveryCodesView,
			{ codes: CODES, username: 'kanna', generatedAt: AT },
			{ me: makeMe('developer', { preferences: makePreferences({ locale: 'en' }) }) }
		);
		expect(i18n.locale).toBe('en');
		await user().click(screen.getByRole('button', { name: 'Download as .txt' }));
		expect(saveText).toHaveBeenCalledOnce();
		const [text, filename] = vi.mocked(saveText).mock.calls[0];
		expect(filename).toBe('kinex-mark-recovery-codes-kanna.txt');
		expect(text.split('\n')).toEqual([
			'Kinex Mark recovery codes',
			'',
			'Account: kanna. Each code can be used once instead of an authentication code.',
			`Created: ${fmt.dateTime(AT)}`,
			t('security.recovery.file.warning'),
			'',
			...CODES,
			''
		]);
	});

	test('prints with only the code sheet visible', async () => {
		let during = false;
		const print = vi.fn(() => {
			during = document.documentElement.classList.contains(PRINT_CLASS);
		});
		vi.stubGlobal('print', print);
		try {
			renderWithApp(
				RecoveryCodesView,
				{ codes: CODES, username: 'kanna', generatedAt: AT },
				{ me: makeMe() }
			);
			await user().click(screen.getByRole('button', { name: t('security.recovery.print') }));
			expect(print).toHaveBeenCalledOnce();
			expect(during).toBe(true);
			expect(document.documentElement.classList.contains(PRINT_CLASS)).toBe(false);
		} finally {
			vi.unstubAllGlobals();
		}
	});

	test('the saved checkbox is bindable', async () => {
		const view = renderWithApp(
			RecoveryCodesView,
			{ codes: CODES, username: 'kanna', generatedAt: AT, saved: false },
			{ me: makeMe() }
		);
		const box = screen.getByRole('checkbox', { name: t('security.recovery.saved') });
		expect(box).not.toBeChecked();
		await user().click(box);
		expect(box).toBeChecked();
		await view.rerender({ saved: false });
		expect(box).not.toBeChecked();
	});

	test('file name', () => {
		expect(recoveryCodesFilename('a.b')).toBe('kinex-mark-recovery-codes-a.b.txt');
	});
});
