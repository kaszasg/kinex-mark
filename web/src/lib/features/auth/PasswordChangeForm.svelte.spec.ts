import { screen, waitFor } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import { t } from '$lib/i18n/index.js';
import { publicConfig } from '$lib/state/public-config.svelte.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe, makePublicConfig } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import PasswordChangeForm from './PasswordChangeForm.svelte';

const gateUser = () =>
	makeMe('customer', { mustChangePassword: true, privacyConsentRequired: true });

beforeEach(() => {
	vi.mocked(goto).mockClear();
	publicConfig.config = null;
});

afterEach(() => {
	vi.unstubAllGlobals();
	session.clear();
	publicConfig.config = null;
});

const current = () => screen.getByLabelText(t('auth.passwordChange.current'));
const next = () => screen.getByLabelText(t('auth.passwordChange.new'));
const repeat = () => screen.getByLabelText(t('auth.passwordChange.confirm'));
const submit = () =>
	screen.getByRole('button', { name: new RegExp(`${t('auth.passwordChange.submit')}$`) });

async function fill(cur: string, pass: string, again = pass): Promise<void> {
	if (cur) await userEvent.type(current(), cur);
	if (pass) await userEvent.type(next(), pass);
	if (again) await userEvent.type(repeat(), again);
	await userEvent.click(submit());
}

describe('PasswordChangeForm', () => {
	test('the policy help comes from the public configuration', async () => {
		mockApi({
			'GET /api/public/config': json(
				makePublicConfig({ passwordPolicy: { minLength: 12, maxLength: 128, minDistinct: 6 } })
			)
		});
		renderWithApp(PasswordChangeForm, { mode: 'gate' }, { me: gateUser() });
		const help = t('auth.passwordPolicy', { min: 12, distinct: 6 });
		expect(await screen.findByText(help)).toBeInTheDocument();
		expect(next()).toHaveAccessibleDescription(help);
		expect(current()).toHaveAttribute('autocomplete', 'current-password');
		expect(next()).toHaveAttribute('autocomplete', 'new-password');
		expect(repeat()).toHaveAttribute('autocomplete', 'new-password');
	});

	test('empty fields are required and nothing is sent', async () => {
		const api = mockApi();
		renderWithApp(PasswordChangeForm, { mode: 'gate' }, { me: gateUser() });
		await userEvent.click(submit());
		expect(screen.getAllByText(t('fields.required'))).toHaveLength(3);
		expect(current()).toHaveFocus();
		expect(api.calls('POST /api/auth/password')).toHaveLength(0);
	});

	test('the repetition must match before a request is made', async () => {
		const api = mockApi();
		renderWithApp(PasswordChangeForm, { mode: 'gate' }, { me: gateUser() });
		await fill('temporary-pass-1', 'brand new passphrase', 'brand new passphrasE');
		expect(screen.getByText(t('auth.passwordChange.mismatch'))).toBeInTheDocument();
		expect(repeat()).toHaveAttribute('aria-invalid', 'true');
		expect(repeat()).toHaveFocus();
		expect(api.calls('POST /api/auth/password')).toHaveLength(0);
	});

	test('current_password_invalid appears under the current field', async () => {
		mockApi({ 'POST /api/auth/password': apiError(400, 'current_password_invalid') });
		renderWithApp(PasswordChangeForm, { mode: 'gate' }, { me: gateUser() });
		await fill('wrong', 'brand new passphrase');
		await waitFor(() => expect(current()).toHaveAttribute('aria-invalid', 'true'));
		expect(current()).toHaveAccessibleDescription(
			`${t('errors.current_password_invalid.title')} ${t('errors.current_password_invalid.text')}`
		);
		expect(current()).toHaveValue('');
		await waitFor(() => expect(current()).toHaveFocus());
		expect(screen.queryByRole('alert')).toBeNull();
	});

	test.each([
		'password_too_short',
		'password_too_simple',
		'password_contains_username',
		'password_reused'
	] as const)('validation_failed %s appears under the new password', async (code) => {
		mockApi({
			'POST /api/auth/password': apiError(400, 'validation_failed', { newPassword: code })
		});
		renderWithApp(PasswordChangeForm, { mode: 'gate' }, { me: gateUser() });
		await fill('temporary-pass-1', 'short');
		await waitFor(() => expect(next()).toHaveAttribute('aria-invalid', 'true'));
		expect(next()).toHaveAccessibleDescription(expect.stringContaining(t(`fields.${code}`)));
		await waitFor(() => expect(next()).toHaveFocus());
		expect(screen.queryByRole('alert')).toBeNull();
	});

	test('validation_failed without a known field shows the alert', async () => {
		mockApi({ 'POST /api/auth/password': apiError(400, 'validation_failed') });
		renderWithApp(PasswordChangeForm, { mode: 'gate' }, { me: gateUser() });
		await fill('temporary-pass-1', 'brand new passphrase');
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.validation_failed.title'));
	});

	test('too_many_attempts shows the alert', async () => {
		mockApi({
			'POST /api/auth/password': apiError(429, 'too_many_attempts', undefined, {
				retryAfterSeconds: 600
			})
		});
		renderWithApp(PasswordChangeForm, { mode: 'gate' }, { me: gateUser() });
		await fill('temporary-pass-1', 'brand new passphrase');
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.too_many_attempts.title'));
		expect(submit()).toBeEnabled();
	});

	test('gate mode: success refreshes the user, toasts and goes to the next gate', async () => {
		const success = vi.spyOn(toast, 'success');
		const onSuccess = vi.fn();
		const api = mockApi({
			'POST /api/auth/password': noContent(),
			'GET /api/auth/me': json(makeMe('customer', { privacyConsentRequired: true }))
		});
		renderWithApp(PasswordChangeForm, { mode: 'gate', onSuccess }, { me: gateUser() });
		await fill('temporary-pass-1', 'brand new passphrase');
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/consent', { replaceState: true }));
		expect(api.calls('POST /api/auth/password')[0].body).toEqual({
			currentPassword: 'temporary-pass-1',
			newPassword: 'brand new passphrase'
		});
		expect(success).toHaveBeenCalledWith(t('auth.passwordChange.saved'));
		expect(onSuccess).toHaveBeenCalledTimes(1);
		expect(session.gate).toBe('consent');
	});

	test('gate mode: without a further gate the links page follows', async () => {
		mockApi({
			'POST /api/auth/password': noContent(),
			'GET /api/auth/me': json(makeMe('customer'))
		});
		renderWithApp(PasswordChangeForm, { mode: 'gate' }, { me: gateUser() });
		await fill('temporary-pass-1', 'brand new passphrase');
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/', { replaceState: true }));
	});

	test('gate mode: a failed refresh still closes the password gate locally', async () => {
		mockApi({
			'POST /api/auth/password': noContent(),
			'GET /api/auth/me': apiError(500, 'internal_error')
		});
		renderWithApp(
			PasswordChangeForm,
			{ mode: 'gate' },
			{ me: makeMe('customer', { mustChangePassword: true }) }
		);
		await fill('temporary-pass-1', 'brand new passphrase');
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/', { replaceState: true }));
		expect(session.me?.mustChangePassword).toBe(false);
	});

	test('settings mode: success toasts, clears the fields and calls onSuccess', async () => {
		const success = vi.spyOn(toast, 'success');
		const onSuccess = vi.fn();
		mockApi({ 'POST /api/auth/password': noContent() });
		renderWithApp(PasswordChangeForm, { mode: 'settings', onSuccess }, { me: makeMe() });
		await fill('old passphrase 1', 'brand new passphrase');
		await waitFor(() => expect(onSuccess).toHaveBeenCalledTimes(1));
		expect(success).toHaveBeenCalledWith(t('auth.passwordChange.savedOtherSessions'));
		expect(current()).toHaveValue('');
		expect(next()).toHaveValue('');
		expect(repeat()).toHaveValue('');
		expect(submit()).toBeEnabled();
		expect(goto).not.toHaveBeenCalled();
	});

	test('keeps the mode markers the shell tests rely on', () => {
		mockApi();
		const { container } = renderWithApp(PasswordChangeForm, { mode: 'gate', onSuccess: () => {} });
		expect(container.querySelector('[data-mode="gate"]')).toHaveAttribute(
			'data-has-success',
			'true'
		);
	});
});
