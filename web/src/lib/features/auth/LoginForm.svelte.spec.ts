import { render, screen, waitFor } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import { i18n, t } from '$lib/i18n/index.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe } from '$lib/test/fixtures.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import LoginForm from './LoginForm.svelte';

const CHALLENGE = {
	status: 'totp_required',
	challenge: 'chal-1',
	expiresAt: '2026-11-21T10:05:00Z'
};

beforeEach(() => {
	vi.mocked(goto).mockClear();
	session.clear();
});

afterEach(() => {
	vi.unstubAllGlobals();
	session.clear();
});

const username = () => screen.getByLabelText(t('auth.login.username'));
const password = () => screen.getByLabelText(t('auth.login.password'));
// The busy spinner adds its label to the button name, so only the end of the name is matched.
const submitButton = () =>
	screen.getByRole('button', { name: new RegExp(`${t('auth.login.submit')}$`) });
/** toHaveTextContent collapses no-break spaces in the element text only. */
const plain = (text: string): string => text.replace(/ /g, ' ');

async function signIn(user = 'kanna', pass = 'correct horse battery'): Promise<void> {
	await userEvent.type(username(), user);
	await userEvent.type(password(), pass);
	await userEvent.click(submitButton());
}

async function reachTotpStep(routes: Parameters<typeof mockApi>[0] = {}) {
	const api = mockApi({ 'POST /api/auth/login': json(CHALLENGE), ...routes });
	const view = renderWithApp(LoginForm, {});
	await signIn();
	const code = await screen.findByLabelText(t('auth.totp.code'));
	return { api, code, unmount: view.unmount };
}

describe('LoginForm, credentials step', () => {
	test('shows the title, the subtitle and the help, and focuses the username', async () => {
		renderWithApp(LoginForm, {});
		expect(screen.getByRole('heading', { level: 1, name: t('auth.login.title') })).toBeVisible();
		expect(screen.getByText(t('auth.login.subtitle'))).toBeInTheDocument();
		expect(screen.getByText(t('auth.login.noAccount'))).toBeInTheDocument();
		await waitFor(() => expect(username()).toHaveFocus());
		expect(username()).toHaveAttribute('autocomplete', 'username');
		expect(username()).toHaveAttribute('autocapitalize', 'off');
		expect(username()).toHaveAttribute('spellcheck', 'false');
		expect(password()).toHaveAttribute('autocomplete', 'current-password');
	});

	test('empty fields show required errors without a request', async () => {
		const api = mockApi();
		renderWithApp(LoginForm, {});
		await userEvent.click(submitButton());
		expect(api.requests).toHaveLength(0);
		expect(username()).toHaveAttribute('aria-invalid', 'true');
		expect(password()).toHaveAttribute('aria-invalid', 'true');
		expect(screen.getAllByText(t('fields.required'))).toHaveLength(2);
		expect(username()).toHaveFocus();

		await userEvent.type(username(), 'kanna');
		await userEvent.click(submitButton());
		expect(api.requests).toHaveLength(0);
		expect(password()).toHaveFocus();
	});

	test('a successful sign-in sends the trimmed name and goes to the validated next target', async () => {
		const me = makeMe();
		const api = mockApi({ 'POST /api/auth/login': json({ status: 'ok', user: me }) });
		renderWithApp(LoginForm, { next: '/tags?x=1' });
		await signIn('  kanna ', 'secret password');
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/tags?x=1', { replaceState: true }));
		expect(api.calls('POST /api/auth/login')[0].body).toEqual({
			username: 'kanna',
			password: 'secret password'
		});
		expect(session.me?.id).toBe(me.id);
		expect(submitButton()).toBeDisabled();
	});

	test('a foreign next target is replaced by the links page', async () => {
		mockApi({ 'POST /api/auth/login': json({ status: 'ok', user: makeMe() }) });
		renderWithApp(LoginForm, { next: 'https://evil.example/' });
		await signIn();
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/', { replaceState: true }));
	});

	test('an open gate wins over next', async () => {
		mockApi({
			'POST /api/auth/login': json({
				status: 'ok',
				user: makeMe('customer', { mustChangePassword: true })
			})
		});
		renderWithApp(LoginForm, { next: '/tags' });
		await signIn();
		await waitFor(() =>
			expect(goto).toHaveBeenCalledWith('/change-password', { replaceState: true })
		);
	});

	test('invalid_credentials shows the alert, clears and focuses the password', async () => {
		mockApi({ 'POST /api/auth/login': apiError(401, 'invalid_credentials') });
		renderWithApp(LoginForm, {});
		await signIn();
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(t('errors.invalid_credentials.title'));
		expect(alert).toHaveTextContent(t('errors.invalid_credentials.text'));
		expect(password()).toHaveValue('');
		await waitFor(() => expect(password()).toHaveFocus());
		expect(username()).toHaveValue('kanna');
		expect(goto).not.toHaveBeenCalled();
	});

	test('account_disabled shows its alert', async () => {
		mockApi({ 'POST /api/auth/login': apiError(403, 'account_disabled') });
		renderWithApp(LoginForm, {});
		await signIn();
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.account_disabled.title'));
	});

	test('network_error shows its alert and the form can be sent again', async () => {
		vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')));
		renderWithApp(LoginForm, {});
		await signIn();
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.network_error.title'));
		expect(submitButton()).toBeEnabled();
	});

	test('validation_failed marks the reported fields', async () => {
		mockApi({
			'POST /api/auth/login': apiError(400, 'validation_failed', { password: 'too_long' })
		});
		renderWithApp(LoginForm, {});
		await signIn();
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.validation_failed.title'));
		expect(password()).toHaveAttribute('aria-invalid', 'true');
		expect(screen.getByText(t('fields.too_long'))).toBeInTheDocument();
		await waitFor(() => expect(password()).toHaveFocus());
	});

	test('too_many_attempts disables the button and shows the remaining minutes', async () => {
		mockApi({
			'POST /api/auth/login': apiError(429, 'too_many_attempts', undefined, {
				retryAfterSeconds: 110
			})
		});
		renderWithApp(LoginForm, {});
		await signIn();
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(t('errors.too_many_attempts.title'));
		expect(alert).toHaveTextContent(plain(t('errors.too_many_attempts.text', { minutes: 2 })));
		expect(submitButton()).toBeDisabled();
	});

	test('the lock ends after the retry time and enables the button again', async () => {
		mockApi({
			'POST /api/auth/login': apiError(429, 'too_many_attempts', undefined, {
				retryAfterSeconds: 1
			})
		});
		renderWithApp(LoginForm, {});
		await signIn();
		await waitFor(() => expect(submitButton()).toBeDisabled());
		await waitFor(() => expect(submitButton()).toBeEnabled(), { timeout: 3000 });
		expect(screen.queryByRole('alert')).toBeNull();
	});
});

describe('LoginForm, notices', () => {
	test.each([
		['expired', 'auth.expired.title', 'auth.expired.text'],
		['deleted', 'auth.accountDeleted.title', 'auth.accountDeleted.text']
	] as const)('reason=%s shows an info alert', (reason, title, text) => {
		renderWithApp(LoginForm, { reason });
		expect(screen.getByText(t(title))).toBeInTheDocument();
		expect(screen.getByText(t(text))).toBeInTheDocument();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	test('after a sign-out the login card says so, below the title and the subtitle', () => {
		session.clear('logout');
		render(LoginForm, {});
		const notice = screen.getByText(t('auth.logout.title'));
		const heading = screen.getByRole('heading', { level: 1, name: t('auth.login.title') });
		const subtitle = screen.getByText(t('auth.login.subtitle'));
		const follows = (a: Node, b: Node) =>
			Boolean(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);
		expect(follows(heading, subtitle)).toBe(true);
		expect(follows(subtitle, notice)).toBe(true);
	});

	test('a sign-in attempt hides the notice', async () => {
		mockApi({ 'POST /api/auth/login': apiError(401, 'invalid_credentials') });
		renderWithApp(LoginForm, { reason: 'expired' });
		await signIn();
		await screen.findByRole('alert');
		expect(screen.queryByText(t('auth.expired.title'))).toBeNull();
	});

	test('the texts follow the language switch', async () => {
		renderWithApp(LoginForm, {});
		i18n.setLocale('de', { persist: false });
		expect(await screen.findByRole('button', { name: 'Anmelden' })).toBeInTheDocument();
	});
});

describe('LoginForm, two-factor step', () => {
	test('totp_required opens the code step with focus and an announcement', async () => {
		const { code } = await reachTotpStep();
		expect(screen.getByRole('heading', { level: 1, name: t('auth.totp.title') })).toBeVisible();
		expect(screen.getByText(t('auth.totp.text'))).toBeInTheDocument();
		await waitFor(() => expect(code).toHaveFocus());
		expect(code).toHaveAttribute('inputmode', 'numeric');
		expect(code).toHaveAttribute('autocomplete', 'one-time-code');
		expect(document.querySelector('[aria-live="polite"]')).toHaveTextContent(t('auth.totp.title'));
		expect(screen.queryByLabelText(t('auth.login.password'))).toBeNull();
	});

	test('signing in with the code leaves no late read of the unmounted code field', async () => {
		const { code, unmount } = await reachTotpStep({
			'POST /api/auth/login/totp': json({ status: 'ok', user: makeMe() })
		});
		// Timers of an earlier test's (already unmounted) code field may still be due: let them pass.
		await new Promise((resolve) => setTimeout(resolve, 80));
		const warn = vi.spyOn(console, 'warn');
		await userEvent.type(code, '123456');
		await waitFor(() => expect(goto).toHaveBeenCalled());
		// The navigation unmounts the form; the PinInput timers (up to 50 ms) have run before it.
		unmount();
		await new Promise((resolve) => setTimeout(resolve, 80));
		const inert = warn.mock.calls.filter((call) => String(call[0]).includes('derived_inert'));
		expect(inert).toEqual([]);
	});

	test('the sixth digit submits the code and signs in', async () => {
		const me = makeMe();
		const { api, code } = await reachTotpStep({
			'POST /api/auth/login/totp': json({ status: 'ok', user: me })
		});
		await userEvent.type(code, '123456');
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/', { replaceState: true }));
		expect(api.calls('POST /api/auth/login/totp')).toHaveLength(1);
		expect(api.calls('POST /api/auth/login/totp')[0].body).toEqual({
			challenge: 'chal-1',
			code: '123456'
		});
	});

	test('the verify button reports an empty or short code without a request', async () => {
		const { api, code } = await reachTotpStep();
		const verify = screen.getByRole('button', { name: t('auth.totp.submit') });
		await userEvent.click(verify);
		expect(screen.getByText(t('fields.required'))).toBeInTheDocument();
		await userEvent.type(code, '123');
		await userEvent.click(verify);
		expect(screen.getByText(t('fields.invalid_format'))).toBeInTheDocument();
		expect(api.calls('POST /api/auth/login/totp')).toHaveLength(0);
	});

	test('invalid_totp clears the code, keeps the step and refocuses the field', async () => {
		const { code } = await reachTotpStep({
			'POST /api/auth/login/totp': apiError(401, 'invalid_totp')
		});
		await userEvent.type(code, '000000');
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.invalid_totp.title'));
		await waitFor(() => expect(code).toHaveValue(''));
		await waitFor(() => expect(code).toHaveFocus());
		expect(screen.getByRole('heading', { name: t('auth.totp.title') })).toBeInTheDocument();
	});

	test('challenge_invalid returns to the first step with an info alert', async () => {
		const { code } = await reachTotpStep({
			'POST /api/auth/login/totp': apiError(401, 'challenge_invalid')
		});
		await userEvent.type(code, '000000');
		expect(
			await screen.findByRole('heading', { level: 1, name: t('auth.login.title') })
		).toBeInTheDocument();
		expect(screen.getByText(t('errors.challenge_invalid.title'))).toBeInTheDocument();
		expect(screen.queryByRole('alert')).toBeNull();
		expect(password()).toHaveValue('');
		await waitFor(() => expect(username()).toHaveFocus());
	});

	test('too_many_attempts in the code step disables the button', async () => {
		const { code } = await reachTotpStep({
			'POST /api/auth/login/totp': apiError(429, 'too_many_attempts', undefined, {
				retryAfterSeconds: 300
			})
		});
		await userEvent.type(code, '000000');
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(plain(t('errors.too_many_attempts.text', { minutes: 5 })));
		expect(screen.getByRole('button', { name: t('auth.totp.submit') })).toBeDisabled();
	});

	test('other errors show their alert', async () => {
		const { code } = await reachTotpStep({
			'POST /api/auth/login/totp': apiError(500, 'internal_error')
		});
		await userEvent.type(code, '111111');
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.internal_error.title'));
	});

	test('a recovery code is sent in lower case and its toast reports the remaining codes', async () => {
		const info = vi.spyOn(toast, 'info');
		const { api } = await reachTotpStep({
			'POST /api/auth/login/totp': json({
				status: 'ok',
				user: makeMe('developer', { recoveryCodesRemaining: 7 })
			})
		});
		await userEvent.click(screen.getByRole('button', { name: t('auth.totp.useRecovery') }));
		const field = screen.getByLabelText(t('auth.recovery.code'));
		await waitFor(() => expect(field).toHaveFocus());
		expect(field).toHaveAttribute('placeholder', t('auth.recovery.placeholder'));
		expect(screen.getByText(t('auth.recovery.text'))).toBeInTheDocument();
		await userEvent.type(field, ' ABCDE-12345 ');
		await userEvent.click(screen.getByRole('button', { name: t('auth.totp.submit') }));
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/', { replaceState: true }));
		expect(api.calls('POST /api/auth/login/totp')[0].body).toEqual({
			challenge: 'chal-1',
			recoveryCode: 'abcde-12345'
		});
		expect(info).toHaveBeenCalledWith(t('auth.recovery.used', { count: 7 }));
	});

	test('few remaining recovery codes give a warning toast', async () => {
		const warning = vi.spyOn(toast, 'warning');
		await reachTotpStep({
			'POST /api/auth/login/totp': json({
				status: 'ok',
				user: makeMe('developer', { recoveryCodesRemaining: 2 })
			})
		});
		await userEvent.click(screen.getByRole('button', { name: t('auth.totp.useRecovery') }));
		await userEvent.type(screen.getByLabelText(t('auth.recovery.code')), 'abcde12345');
		await userEvent.click(screen.getByRole('button', { name: t('auth.totp.submit') }));
		await waitFor(() =>
			expect(warning).toHaveBeenCalledWith(t('auth.recovery.used', { count: 2 }), {
				description: t('auth.recovery.low')
			})
		);
	});

	test('an empty recovery code is required; switching back restores the code field', async () => {
		const { api } = await reachTotpStep();
		await userEvent.click(screen.getByRole('button', { name: t('auth.totp.useRecovery') }));
		await userEvent.click(screen.getByRole('button', { name: t('auth.totp.submit') }));
		expect(screen.getByText(t('fields.required'))).toBeInTheDocument();
		expect(api.calls('POST /api/auth/login/totp')).toHaveLength(0);
		await userEvent.click(screen.getByRole('button', { name: t('auth.totp.useCode') }));
		expect(screen.getByLabelText(t('auth.totp.code'))).toBeInTheDocument();
		expect(screen.queryByText(t('fields.required'))).toBeNull();
	});

	test('back to sign-in drops the challenge', async () => {
		await reachTotpStep();
		await userEvent.click(screen.getByRole('button', { name: t('auth.totp.back') }));
		expect(screen.getByRole('heading', { level: 1, name: t('auth.login.title') })).toBeVisible();
		expect(screen.queryByText(t('errors.challenge_invalid.title'))).toBeNull();
		expect(document.querySelector('[aria-live="polite"]')).toHaveTextContent('');
	});
});
