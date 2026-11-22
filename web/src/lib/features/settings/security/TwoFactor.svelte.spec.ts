// Two-factor sign-in in Settings > Security: card, enrolment and step-up dialogs (plan 8.7.6).
import { screen, waitFor, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { errorMessage, fieldMessage, t } from '$lib/i18n/index.js';
import { ApiError } from '$lib/api/errors.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import TwoFactorCard from './TwoFactorCard.svelte';
import TwoFactorConfirmDialog from './TwoFactorConfirmDialog.svelte';
import TwoFactorEnrollDialog, { groupSecret } from './TwoFactorEnrollDialog.svelte';

const user = () => userEvent.setup({ pointerEventsCheck: 0 });

const SECRET = 'JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP';
const SETUP = {
	secret: SECRET,
	otpauthUri: `otpauth://totp/Kinex%20Mark:kanna?secret=${SECRET}&issuer=Kinex%20Mark&algorithm=SHA1&digits=6&period=30`,
	expiresAt: '2026-11-21T12:10:00.000Z'
};
const CODES = Array.from({ length: 10 }, (_, i) => `abcde-fgh${String(i).padStart(2, '0')}`);

function message(code: ConstructorParameters<typeof ApiError>[1]): string {
	return errorMessage(new ApiError(400, code));
}

function dialog(): HTMLElement {
	return screen.getByRole('dialog');
}

beforeEach(() => {
	session.clear();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('groupSecret', () => {
	test('groups the secret by four characters', () => {
		expect(groupSecret('JBSWY3DPEHPK3PXP')).toBe('JBSW Y3DP EHPK 3PXP');
		expect(groupSecret('ABCDEF')).toBe('ABCD EF');
		expect(groupSecret('')).toBe('');
	});
});

describe('TwoFactorEnrollDialog', () => {
	async function toScanStep(): Promise<void> {
		await user().type(
			screen.getByLabelText(t('security.enroll.password.label')),
			'Secret-pass-123'
		);
		await user().click(screen.getByRole('button', { name: t('security.enroll.password.submit') }));
		await screen.findByRole('img', { name: t('security.enroll.scan.qr') });
	}

	function codeInput(): HTMLElement {
		return screen.getByLabelText(t('security.enroll.scan.code'));
	}

	test('walks through the three steps and refreshes the session at the end', async () => {
		const me = makeMe('developer', { totpEnabled: false });
		const api = mockApi({
			'POST /api/me/2fa/setup': json(SETUP),
			'POST /api/me/2fa/enable': json({ recoveryCodes: CODES }),
			'GET /api/auth/me': json({ ...me, totpEnabled: true, recoveryCodesRemaining: 10 })
		});
		const success = vi.spyOn(toast, 'success');
		const onEnabled = vi.fn();
		renderWithApp(TwoFactorEnrollDialog, { open: true, onEnabled }, { me });

		// Step 1: password.
		expect(within(dialog()).getByText('1 / 3')).toBeInTheDocument();
		expect(
			within(dialog()).getByText(t('security.enroll.stepLabel', { current: '1', total: '3' }))
		).toBeInTheDocument();
		expect(
			screen.getByRole('heading', { name: t('security.enroll.password.title') })
		).toBeInTheDocument();
		expect(dialog().querySelectorAll('[data-step-done]')).toHaveLength(1);
		await toScanStep();
		expect(api.calls('POST /api/me/2fa/setup')[0].body).toEqual({ password: 'Secret-pass-123' });

		// Step 2: QR code, manual key and the code.
		expect(within(dialog()).getByText('2 / 3')).toBeInTheDocument();
		const qr = screen.getByRole('img', { name: t('security.enroll.scan.qr') });
		expect(qr.tagName.toLowerCase()).toBe('svg');
		expect(qr).toHaveAttribute('width', '176');
		expect(qr.querySelector('path')?.getAttribute('d')).toBeTruthy();
		const secret = screen.getByText(groupSecret(SECRET));
		// Lines break only between the groups of four, never inside one.
		expect(secret).toHaveClass('break-normal');
		expect(secret).not.toHaveClass('break-all');
		expect(
			screen.getByRole('button', { name: t('security.enroll.scan.copySecret') })
		).toBeInTheDocument();
		await user().type(codeInput(), '123456');
		await user().click(screen.getByRole('button', { name: t('security.enroll.scan.submit') }));

		// Step 3: recovery codes; "Done" waits for the checkbox.
		const list = await screen.findByRole('list', { name: t('security.recovery.listLabel') });
		expect(api.calls('POST /api/me/2fa/enable')[0].body).toEqual({ code: '123456' });
		expect(within(dialog()).getByText('3 / 3')).toBeInTheDocument();
		expect(within(list).getAllByRole('listitem')).toHaveLength(10);
		const done = screen.getByRole('button', { name: t('common.actions.done') });
		expect(done).toBeDisabled();
		await user().keyboard('{Escape}');
		expect(screen.getByRole('dialog')).toBeInTheDocument();
		await user().click(screen.getByRole('checkbox', { name: t('security.recovery.saved') }));
		expect(done).toBeEnabled();
		await user().click(done);

		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
		expect(success).toHaveBeenCalledWith(t('security.twoFactor.enabled'));
		expect(onEnabled).toHaveBeenCalledOnce();
		await waitFor(() => expect(session.me?.totpEnabled).toBe(true));
		expect(api.calls('GET /api/auth/me')).toHaveLength(1);
	});

	test('an empty password is caught before the request', async () => {
		const api = mockApi({});
		renderWithApp(TwoFactorEnrollDialog, { open: true }, { me: makeMe() });
		await user().click(screen.getByRole('button', { name: t('security.enroll.password.submit') }));
		const input = screen.getByLabelText(t('security.enroll.password.label'));
		expect(input).toHaveAttribute('aria-invalid', 'true');
		expect(screen.getByText(fieldMessage('password', 'required'))).toBeInTheDocument();
		await waitFor(() => expect(input).toHaveFocus());
		expect(api.requests).toHaveLength(0);
	});

	test('a wrong password is shown under the field', async () => {
		mockApi({ 'POST /api/me/2fa/setup': apiError(400, 'current_password_invalid') });
		renderWithApp(TwoFactorEnrollDialog, { open: true }, { me: makeMe() });
		await user().type(screen.getByLabelText(t('security.enroll.password.label')), 'wrong-pass');
		await user().click(screen.getByRole('button', { name: t('security.enroll.password.submit') }));
		expect(await screen.findByText(message('current_password_invalid'))).toBeInTheDocument();
		expect(screen.getByLabelText(t('security.enroll.password.label'))).toHaveAttribute(
			'aria-invalid',
			'true'
		);
	});

	test('a throttled setup shows an alert', async () => {
		mockApi({
			'POST /api/me/2fa/setup': apiError(429, 'too_many_attempts', undefined, {
				retryAfterSeconds: 120
			})
		});
		renderWithApp(TwoFactorEnrollDialog, { open: true }, { me: makeMe() });
		await user().type(screen.getByLabelText(t('security.enroll.password.label')), 'pass');
		await user().click(screen.getByRole('button', { name: t('security.enroll.password.submit') }));
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(t('errors.too_many_attempts.title'));
	});

	test('a short or wrong code stays on the scan step', async () => {
		mockApi({
			'POST /api/me/2fa/setup': json(SETUP),
			'POST /api/me/2fa/enable': apiError(401, 'invalid_totp')
		});
		renderWithApp(TwoFactorEnrollDialog, { open: true }, { me: makeMe() });
		await toScanStep();
		await user().click(screen.getByRole('button', { name: t('security.enroll.scan.submit') }));
		expect(screen.getByText(fieldMessage('code', 'required'))).toBeInTheDocument();
		await user().type(codeInput(), '123');
		await user().click(screen.getByRole('button', { name: t('security.enroll.scan.submit') }));
		expect(screen.getByText(fieldMessage('code', 'invalid_format'))).toBeInTheDocument();
		await user().type(codeInput(), '456');
		await user().click(screen.getByRole('button', { name: t('security.enroll.scan.submit') }));
		expect(await screen.findByText(message('invalid_totp'))).toBeInTheDocument();
		expect(screen.getByRole('img', { name: t('security.enroll.scan.qr') })).toBeInTheDocument();
	});

	test('an expired setup goes back to the password step', async () => {
		mockApi({
			'POST /api/me/2fa/setup': json(SETUP),
			'POST /api/me/2fa/enable': apiError(409, 'totp_setup_required')
		});
		renderWithApp(TwoFactorEnrollDialog, { open: true }, { me: makeMe() });
		await toScanStep();
		await user().type(codeInput(), '123456');
		await user().click(screen.getByRole('button', { name: t('security.enroll.scan.submit') }));
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(t('errors.totp_setup_required.title'));
		expect(screen.getByLabelText(t('security.enroll.password.label'))).toBeInTheDocument();
		expect(within(dialog()).getByText('1 / 3')).toBeInTheDocument();
	});

	test('cancel and Escape close the dialog before the codes exist', async () => {
		mockApi({ 'POST /api/me/2fa/setup': json(SETUP) });
		const view = renderWithApp(TwoFactorEnrollDialog, { open: true }, { me: makeMe() });
		await user().click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());

		await view.rerender({ open: true });
		await toScanStep();
		await user().keyboard('{Escape}');
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());

		// Reopening starts over at step 1.
		await view.rerender({ open: true });
		expect(await screen.findByLabelText(t('security.enroll.password.label'))).toHaveValue('');
	});
});

describe('TwoFactorConfirmDialog', () => {
	test('turning off needs the password and a code, then refreshes the session', async () => {
		const me = makeMe('developer', { totpEnabled: true, recoveryCodesRemaining: 5 });
		const api = mockApi({
			'POST /api/me/2fa/disable': noContent(),
			'GET /api/auth/me': json({ ...me, totpEnabled: false, recoveryCodesRemaining: 0 })
		});
		const success = vi.spyOn(toast, 'success');
		const onDone = vi.fn();
		renderWithApp(TwoFactorConfirmDialog, { open: true, mode: 'disable', onDone }, { me });

		expect(screen.getByText(t('security.confirm.disable.title'))).toBeInTheDocument();
		const submit = screen.getByRole('button', { name: t('security.confirm.disable.submit') });
		await user().click(submit);
		expect(screen.getAllByText(fieldMessage('code', 'required'))).toHaveLength(2);
		expect(screen.getByLabelText(t('security.confirm.password'))).toHaveAttribute(
			'aria-invalid',
			'true'
		);
		expect(screen.getByLabelText(t('security.confirm.disable.code'))).toHaveAttribute(
			'aria-invalid',
			'true'
		);
		expect(api.requests).toHaveLength(0);

		await user().type(screen.getByLabelText(t('security.confirm.password')), 'Secret-pass-123');
		await user().type(screen.getByLabelText(t('security.confirm.disable.code')), ' abcde-fghjk ');
		await user().click(submit);

		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
		expect(api.calls('POST /api/me/2fa/disable')[0].body).toEqual({
			password: 'Secret-pass-123',
			code: 'abcde-fghjk'
		});
		expect(success).toHaveBeenCalledWith(t('security.twoFactor.disabled'));
		expect(onDone).toHaveBeenCalledOnce();
		await waitFor(() => expect(session.me?.totpEnabled).toBe(false));
	});

	test('wrong password and wrong code go under their fields', async () => {
		const me = makeMe('developer', { totpEnabled: true });
		const api = mockApi({ 'POST /api/me/2fa/disable': apiError(400, 'current_password_invalid') });
		renderWithApp(TwoFactorConfirmDialog, { open: true, mode: 'disable' }, { me });
		await user().type(screen.getByLabelText(t('security.confirm.password')), 'wrong');
		await user().type(screen.getByLabelText(t('security.confirm.disable.code')), '123456');
		await user().click(screen.getByRole('button', { name: t('security.confirm.disable.submit') }));
		expect(await screen.findByText(message('current_password_invalid'))).toBeInTheDocument();

		api.route('POST /api/me/2fa/disable', apiError(401, 'invalid_totp'));
		await user().type(screen.getByLabelText(t('security.confirm.disable.code')), '654321');
		await user().click(screen.getByRole('button', { name: t('security.confirm.disable.submit') }));
		expect(await screen.findByText(message('invalid_totp'))).toBeInTheDocument();
		expect(screen.getByLabelText(t('security.confirm.disable.code'))).toHaveAttribute(
			'aria-invalid',
			'true'
		);
		expect(screen.getByLabelText(t('security.confirm.disable.code'))).toHaveValue('');
	});

	test('a state conflict shows an alert', async () => {
		mockApi({ 'POST /api/me/2fa/disable': apiError(409, 'totp_not_enabled') });
		renderWithApp(
			TwoFactorConfirmDialog,
			{ open: true, mode: 'disable' },
			{ me: makeMe('developer', { totpEnabled: true }) }
		);
		await user().type(screen.getByLabelText(t('security.confirm.password')), 'pass');
		await user().type(screen.getByLabelText(t('security.confirm.disable.code')), '123456');
		await user().click(screen.getByRole('button', { name: t('security.confirm.disable.submit') }));
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.totp_not_enabled.title'));
	});

	test('new recovery codes need a six-digit code and are shown once', async () => {
		const me = makeMe('developer', { totpEnabled: true, recoveryCodesRemaining: 1 });
		const api = mockApi({
			'POST /api/me/2fa/recovery-codes': json({ recoveryCodes: CODES }),
			'GET /api/auth/me': json({ ...me, recoveryCodesRemaining: 10 })
		});
		const success = vi.spyOn(toast, 'success');
		const onDone = vi.fn();
		renderWithApp(TwoFactorConfirmDialog, { open: true, mode: 'regenerate', onDone }, { me });

		expect(screen.getByText(t('security.confirm.regenerate.title'))).toBeInTheDocument();
		await user().type(screen.getByLabelText(t('security.confirm.password')), 'Secret-pass-123');
		await user().type(screen.getByLabelText(t('security.confirm.regenerate.code')), 'abc');
		await user().click(
			screen.getByRole('button', { name: t('security.confirm.regenerate.submit') })
		);
		expect(screen.getByText(fieldMessage('code', 'invalid_format'))).toBeInTheDocument();
		expect(api.requests).toHaveLength(0);

		await user().clear(screen.getByLabelText(t('security.confirm.regenerate.code')));
		await user().type(screen.getByLabelText(t('security.confirm.regenerate.code')), '123456');
		await user().click(
			screen.getByRole('button', { name: t('security.confirm.regenerate.submit') })
		);
		const list = await screen.findByRole('list', { name: t('security.recovery.listLabel') });
		expect(within(list).getAllByRole('listitem')).toHaveLength(10);
		expect(api.calls('POST /api/me/2fa/recovery-codes')[0].body).toEqual({
			password: 'Secret-pass-123',
			code: '123456'
		});
		expect(success).toHaveBeenCalledWith(t('security.twoFactor.regenerated'));

		// The dialog stays until the codes are saved.
		await user().keyboard('{Escape}');
		expect(screen.getByRole('dialog')).toBeInTheDocument();
		const done = screen.getByRole('button', { name: t('common.actions.done') });
		expect(done).toBeDisabled();
		await user().click(screen.getByRole('checkbox', { name: t('security.recovery.saved') }));
		await user().click(done);
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
		expect(onDone).toHaveBeenCalledOnce();
		await waitFor(() => expect(session.me?.recoveryCodesRemaining).toBe(10));
	});

	test('Escape after saving the codes closes the dialog', async () => {
		mockApi({
			'POST /api/me/2fa/recovery-codes': json({ recoveryCodes: CODES }),
			'GET /api/auth/me': json(makeMe('developer', { totpEnabled: true }))
		});
		const onDone = vi.fn();
		renderWithApp(
			TwoFactorConfirmDialog,
			{ open: true, mode: 'regenerate', onDone },
			{ me: makeMe('developer', { totpEnabled: true }) }
		);
		await user().type(screen.getByLabelText(t('security.confirm.password')), 'pass');
		await user().type(screen.getByLabelText(t('security.confirm.regenerate.code')), '123456');
		await user().click(
			screen.getByRole('button', { name: t('security.confirm.regenerate.submit') })
		);
		await user().click(await screen.findByRole('checkbox', { name: t('security.recovery.saved') }));
		await user().keyboard('{Escape}');
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
		expect(onDone).toHaveBeenCalledOnce();
	});

	test('cancel closes without a request', async () => {
		const api = mockApi({});
		renderWithApp(
			TwoFactorConfirmDialog,
			{ open: true, mode: 'disable' },
			{ me: makeMe('developer', { totpEnabled: true }) }
		);
		await user().click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
		expect(api.requests).toHaveLength(0);
	});
});

describe('TwoFactorCard', () => {
	test('turned off: status word and the enrol action', async () => {
		renderWithApp(TwoFactorCard, {}, { me: makeMe('developer', { totpEnabled: false }) });
		const card = screen.getByRole('region', { name: t('security.twoFactor.title') });
		expect(within(card).getByText(t('status.disabled'))).toBeInTheDocument();
		expect(within(card).queryByText(t('security.twoFactor.remaining'))).not.toBeInTheDocument();
		await user().click(within(card).getByRole('button', { name: t('security.twoFactor.enable') }));
		expect(await screen.findByRole('dialog')).toHaveTextContent(
			t('security.enroll.password.title')
		);
	});

	test('turned on with few codes left: count, warning and both actions', async () => {
		renderWithApp(
			TwoFactorCard,
			{},
			{ me: makeMe('developer', { totpEnabled: true, recoveryCodesRemaining: 3 }) }
		);
		const card = screen.getByRole('region', { name: t('security.twoFactor.title') });
		expect(within(card).getByText(t('status.enabled'))).toBeInTheDocument();
		expect(within(card).getByTestId('recovery-remaining')).toHaveTextContent('3');
		expect(within(card).getByText(t('security.twoFactor.lowCodes.title'))).toBeInTheDocument();
		expect(within(card).queryByRole('button', { name: t('security.twoFactor.enable') })).toBeNull();

		await user().click(
			within(card).getByRole('button', { name: t('security.twoFactor.regenerate') })
		);
		expect(await screen.findByRole('dialog')).toHaveTextContent(
			t('security.confirm.regenerate.title')
		);
		await user().click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());

		await user().click(within(card).getByRole('button', { name: t('security.twoFactor.disable') }));
		expect(await screen.findByRole('dialog')).toHaveTextContent(
			t('security.confirm.disable.title')
		);
	});

	test('enough codes: no warning', () => {
		renderWithApp(
			TwoFactorCard,
			{},
			{ me: makeMe('developer', { totpEnabled: true, recoveryCodesRemaining: 8 }) }
		);
		expect(screen.queryByText(t('security.twoFactor.lowCodes.title'))).not.toBeInTheDocument();
		expect(screen.getByTestId('recovery-remaining')).toHaveTextContent('8');
	});
});
