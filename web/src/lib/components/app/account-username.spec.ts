// Password forms carry the account's username for password managers (D-close leftover 3).
import { screen } from '@testing-library/svelte';
import type { Component } from 'svelte';
import { afterEach, describe, expect, test, vi } from 'vitest';
import StepUpDialog from '$lib/features/admin/StepUpDialog.svelte';
import PasswordChangeForm from '$lib/features/auth/PasswordChangeForm.svelte';
import DeleteAccountDialog from '$lib/features/settings/privacy/DeleteAccountDialog.svelte';
import TwoFactorConfirmDialog from '$lib/features/settings/security/TwoFactorConfirmDialog.svelte';
import TwoFactorEnrollDialog from '$lib/features/settings/security/TwoFactorEnrollDialog.svelte';
import { t } from '$lib/i18n/index.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe } from '$lib/test/fixtures.js';
import { mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import userEvent from '@testing-library/user-event';

afterEach(() => {
	session.clear();
	vi.unstubAllGlobals();
});

type Case = {
	name: string;
	component: Component<Record<string, unknown>>;
	props: Record<string, unknown>;
	open?: () => Promise<void>;
};

const cases: Case[] = [
	{
		name: 'PasswordChangeForm',
		component: PasswordChangeForm as unknown as Case['component'],
		props: { mode: 'settings' }
	},
	{
		name: 'StepUpDialog',
		component: StepUpDialog as unknown as Case['component'],
		props: { open: true, title: 'Reset?', confirmLabel: 'Reset', onConfirm: () => {} }
	},
	{
		name: 'DeleteAccountDialog',
		component: DeleteAccountDialog as unknown as Case['component'],
		props: {},
		open: async () => {
			await userEvent
				.setup({ pointerEventsCheck: 0 })
				.click(screen.getByRole('button', { name: t('security.deleteAccount.open') }));
		}
	},
	{
		name: 'TwoFactorConfirmDialog',
		component: TwoFactorConfirmDialog as unknown as Case['component'],
		props: { open: true, mode: 'disable' }
	},
	{
		name: 'TwoFactorEnrollDialog',
		component: TwoFactorEnrollDialog as unknown as Case['component'],
		props: { open: true }
	}
];

describe('password forms', () => {
	test.each(cases)(
		'$name has a hidden, read-only username field in the form of its password field',
		async ({ component, props, open }) => {
			mockApi({});
			renderWithApp(component, props, { me: makeMe('developer', { username: 'kanna' }) });
			await open?.();
			const password = await vi.waitFor(() => {
				const found = document.querySelector<HTMLInputElement>(
					'input[autocomplete="current-password"]'
				);
				if (!found) throw new Error('no password field yet');
				return found;
			});
			const username = password.form?.querySelector<HTMLInputElement>(
				'input[autocomplete="username"]'
			);
			expect(username).toBeTruthy();
			expect(username).toHaveValue('kanna');
			expect(username).toHaveAttribute('readonly');
			expect(username).toHaveAttribute('hidden');
			expect(username).toHaveAttribute('tabindex', '-1');
			expect(username).toHaveAttribute('aria-hidden', 'true');
			expect(username).not.toBeVisible();
		}
	);
});
