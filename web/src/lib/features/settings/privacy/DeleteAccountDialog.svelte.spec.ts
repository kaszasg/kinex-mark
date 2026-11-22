// Account deletion of Settings > Privacy (plan 8.7.6, GDPR erasure).
import { screen, waitFor, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import { ApiError } from '$lib/api/errors.js';
import type { PrivacyStatus } from '$lib/api/types.js';
import { errorMessage, fieldMessage, t } from '$lib/i18n/index.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import DeleteAccountDialog from './DeleteAccountDialog.svelte';

const user = () => userEvent.setup({ pointerEventsCheck: 0 });

function privacy(ownSharedLinkCount: number): PrivacyStatus {
	return { currentVersion: 1, consentRequired: false, consents: [], ownSharedLinkCount };
}

function message(code: ConstructorParameters<typeof ApiError>[1]): string {
	return errorMessage(new ApiError(400, code));
}

const confirmLabel = () => t('security.deleteAccount.confirm', { username: 'kanna' });
const submitButton = () => screen.getByRole('button', { name: t('security.deleteAccount.submit') });

async function openDialog(): Promise<HTMLElement> {
	await user().click(screen.getByRole('button', { name: t('security.deleteAccount.open') }));
	return screen.findByRole('alertdialog');
}

beforeEach(() => {
	vi.mocked(goto).mockClear();
	session.clear();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('DeleteAccountDialog', () => {
	test('with two-factor sign-in: warning, code field, username gate and success', async () => {
		const api = mockApi({
			'GET /api/me/privacy': json(privacy(3)),
			'POST /api/me/delete': noContent()
		});
		renderWithApp(DeleteAccountDialog, {}, { me: makeMe('developer', { totpEnabled: true }) });
		const card = screen.getByRole('region', { name: t('security.deleteAccount.title') });
		expect(within(card).getByText(t('security.deleteAccount.text'))).toBeInTheDocument();

		const dialog = await openDialog();
		expect(dialog).toHaveTextContent(t('security.deleteAccount.dialogTitle'));
		expect(dialog).toHaveTextContent(t('security.deleteAccount.dialogText'));
		expect(
			await within(dialog).findByText(t('security.deleteAccount.shared.title', { count: 3 }))
		).toBeInTheDocument();
		expect(dialog).toHaveTextContent(t('security.deleteAccount.shared.text'));

		const submit = submitButton();
		expect(submit).toBeDisabled();
		await user().type(screen.getByLabelText(confirmLabel()), 'kann');
		expect(submit).toBeDisabled();
		await user().type(screen.getByLabelText(confirmLabel()), 'a');
		expect(submit).toBeEnabled();

		// Client-side required fields.
		await user().click(submit);
		expect(screen.getAllByText(fieldMessage('password', 'required'))).toHaveLength(2);
		expect(api.calls('POST /api/me/delete')).toHaveLength(0);

		await user().type(
			screen.getByLabelText(t('security.deleteAccount.password')),
			'Secret-pass-123'
		);
		await user().type(screen.getByLabelText(t('security.deleteAccount.code')), ' 123456 ');
		await user().click(submit);

		await waitFor(() => expect(goto).toHaveBeenCalledWith('/login?reason=deleted'));
		expect(api.calls('POST /api/me/delete')[0].body).toEqual({
			password: 'Secret-pass-123',
			confirm: 'kanna',
			code: '123456'
		});
		expect(session.status).toBe('anonymous');
		expect(session.lastClearReason).toBe('deleted');
	});

	test('opening the dialog focuses the password field', async () => {
		mockApi({ 'GET /api/me/privacy': json(privacy(0)) });
		renderWithApp(DeleteAccountDialog, {}, { me: makeMe('developer', { totpEnabled: true }) });
		const dialog = await openDialog();
		await waitFor(() =>
			expect(within(dialog).getByLabelText(t('security.deleteAccount.password'))).toHaveFocus()
		);
		expect(dialog).not.toHaveFocus();
	});

	test('without two-factor sign-in there is no code field and no warning', async () => {
		const api = mockApi({
			'GET /api/me/privacy': json(privacy(0)),
			'POST /api/me/delete': noContent()
		});
		renderWithApp(DeleteAccountDialog, {}, { me: makeMe('customer', { totpEnabled: false }) });
		const dialog = await openDialog();
		await waitFor(() => expect(api.calls('GET /api/me/privacy')).toHaveLength(1));
		await waitFor(() => expect(within(dialog).queryByRole('status')).not.toBeInTheDocument());
		expect(within(dialog).queryByLabelText(t('security.deleteAccount.code'))).toBeNull();
		expect(dialog).not.toHaveTextContent(t('security.deleteAccount.shared.text'));

		await user().type(screen.getByLabelText(t('security.deleteAccount.password')), 'pass');
		await user().type(screen.getByLabelText(confirmLabel()), 'kanna');
		await user().click(submitButton());
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/login?reason=deleted'));
		expect(api.calls('POST /api/me/delete')[0].body).toEqual({
			password: 'pass',
			confirm: 'kanna'
		});
	});

	test('server errors go under their fields or into an alert', async () => {
		const api = mockApi({
			'GET /api/me/privacy': json(privacy(0)),
			'POST /api/me/delete': apiError(400, 'current_password_invalid')
		});
		renderWithApp(DeleteAccountDialog, {}, { me: makeMe('admin', { totpEnabled: true }) });
		await openDialog();
		await user().type(screen.getByLabelText(t('security.deleteAccount.password')), 'wrong');
		await user().type(screen.getByLabelText(t('security.deleteAccount.code')), '123456');
		await user().type(screen.getByLabelText(confirmLabel()), 'kanna');
		await user().click(submitButton());
		expect(await screen.findByText(message('current_password_invalid'))).toBeInTheDocument();
		expect(screen.getByLabelText(t('security.deleteAccount.password'))).toHaveAttribute(
			'aria-invalid',
			'true'
		);

		api.route('POST /api/me/delete', apiError(401, 'invalid_totp'));
		await user().type(screen.getByLabelText(t('security.deleteAccount.code')), '654321');
		await user().click(submitButton());
		expect(await screen.findByText(message('invalid_totp'))).toBeInTheDocument();
		expect(screen.getByLabelText(t('security.deleteAccount.code'))).toHaveValue('');

		api.route('POST /api/me/delete', apiError(400, 'confirmation_mismatch'));
		await user().type(screen.getByLabelText(t('security.deleteAccount.code')), '111111');
		await user().click(submitButton());
		expect(await screen.findByText(message('confirmation_mismatch'))).toBeInTheDocument();

		api.route('POST /api/me/delete', apiError(400, 'validation_failed', { code: 'required' }));
		await user().type(screen.getByLabelText(t('security.deleteAccount.code')), '222222');
		await user().click(submitButton());
		expect(await screen.findByText(fieldMessage('code', 'required'))).toBeInTheDocument();

		api.route('POST /api/me/delete', apiError(409, 'last_admin'));
		await user().type(screen.getByLabelText(t('security.deleteAccount.code')), '333333');
		await user().click(submitButton());
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.last_admin.title'));
		expect(goto).not.toHaveBeenCalled();
		expect(session.status).toBe('authenticated');
	});

	test('too many attempts shows an alert', async () => {
		mockApi({
			'GET /api/me/privacy': json(privacy(0)),
			'POST /api/me/delete': apiError(429, 'too_many_attempts', undefined, {
				retryAfterSeconds: 300
			})
		});
		renderWithApp(DeleteAccountDialog, {}, { me: makeMe() });
		await openDialog();
		await user().type(screen.getByLabelText(t('security.deleteAccount.password')), 'pass');
		await user().type(screen.getByLabelText(confirmLabel()), 'kanna');
		await user().click(submitButton());
		// toHaveTextContent collapses the element's NBSP, so the expectation is collapsed too.
		expect(await screen.findByRole('alert')).toHaveTextContent(
			t('errors.too_many_attempts.text', { minutes: 5 }).replace(/\s+/g, ' ')
		);
	});

	test('a failed shared-link count shows an alert with retry', async () => {
		const api = mockApi({ 'GET /api/me/privacy': apiError(500, 'internal_error') });
		renderWithApp(DeleteAccountDialog, {}, { me: makeMe() });
		const dialog = await openDialog();
		const alert = await within(dialog).findByRole('alert');
		api.route('GET /api/me/privacy', json(privacy(1)));
		await user().click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		expect(
			await within(dialog).findByText(t('security.deleteAccount.shared.title', { count: 1 }))
		).toBeInTheDocument();
	});

	test('cancel closes the dialog and reopening starts empty', async () => {
		mockApi({ 'GET /api/me/privacy': json(privacy(0)) });
		renderWithApp(DeleteAccountDialog, {}, { me: makeMe() });
		await openDialog();
		await user().type(screen.getByLabelText(confirmLabel()), 'kanna');
		await user().click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
		await openDialog();
		expect(screen.getByLabelText(confirmLabel())).toHaveValue('');
		await user().keyboard('{Escape}');
		await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
	});
});
