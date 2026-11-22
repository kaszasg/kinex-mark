// Settings > Profile (plan 8.7.6): read-only account data and the display name form.
import { screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { setMode } from 'mode-watcher';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import type { Me } from '$lib/api/types.js';
import { fmt, t } from '$lib/i18n/index.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe, makePreferences } from '$lib/test/fixtures.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import ProfileForm, { DISPLAY_NAME_MAX } from './ProfileForm.svelte';

vi.mock('mode-watcher', async (importOriginal) => ({
	...(await importOriginal<typeof import('mode-watcher')>()),
	setMode: vi.fn()
}));

const user = () => userEvent.setup({ pointerEventsCheck: 0 });

function setup(partial: Partial<Me> = {}) {
	const me = makeMe('developer', {
		createdAt: '2026-03-02T08:15:00.000Z',
		lastLoginAt: '2026-11-20T07:40:00.000Z',
		...partial
	});
	renderWithApp(ProfileForm, {}, { me });
	const input = screen.getByLabelText(t('settings.profile.displayName'));
	const save = screen.getByRole('button', { name: t('settings.profile.save') });
	return { me, input, save };
}

beforeEach(() => {
	vi.mocked(setMode).mockClear();
	vi.spyOn(toast, 'success').mockImplementation(() => 1);
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('ProfileForm', () => {
	test('shows loading rows before the session is known', () => {
		renderWithApp(ProfileForm, {}, { me: null });
		expect(screen.getByRole('status')).toHaveTextContent(t('common.states.loading'));
		expect(screen.queryByRole('textbox')).toBeNull();
	});

	test('shows the account data with machine values in mono', () => {
		const { input, save } = setup();
		expect(
			screen.getByRole('heading', { level: 2, name: t('settings.profile.title') })
		).toBeVisible();
		expect(screen.getByTestId('profile-username')).toHaveTextContent('kanna');
		expect(screen.getByTestId('profile-username')).toHaveClass('font-mono');
		expect(screen.getByText(t('roles.developer'))).toBeInTheDocument();
		const times = [...document.querySelectorAll('time')];
		expect(times.map((el) => el.getAttribute('datetime'))).toEqual([
			'2026-03-02T08:15:00.000Z',
			'2026-11-20T07:40:00.000Z'
		]);
		expect(times[0].textContent?.trim()).toBe(fmt.dateTime('2026-03-02T08:15:00.000Z'));
		expect(input).toHaveValue('Kovács Anna');
		expect(save).toBeDisabled();
	});

	test('a missing last sign-in shows the dash', () => {
		setup({ lastLoginAt: null });
		expect(screen.getByTestId('profile-last-login')).toHaveTextContent(fmt.missing);
	});

	test('saves the trimmed display name and keeps the session preferences', async () => {
		const api = mockApi({
			'PATCH /api/me/profile': (req) =>
				json(
					makeMe('developer', {
						displayName: (req.body as { displayName: string }).displayName,
						preferences: makePreferences({ theme: 'light' })
					})
				)
		});
		const { input, save } = setup();
		const u = user();
		await u.clear(input);
		await u.type(input, '  Kovács Anna Mária  ');
		expect(save).toBeEnabled();
		await u.click(save);
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('settings.profile.saved')));
		expect(api.calls('PATCH /api/me/profile')[0].body).toEqual({
			displayName: 'Kovács Anna Mária'
		});
		expect(session.me?.displayName).toBe('Kovács Anna Mária');
		expect(session.me?.preferences.theme).toBe('dark');
		expect(input).toHaveValue('Kovács Anna Mária');
		expect(save).toBeDisabled();
	});

	test('an empty display name is allowed', async () => {
		const api = mockApi({
			'PATCH /api/me/profile': json(makeMe('developer', { displayName: '' }))
		});
		const { input, save } = setup();
		await user().clear(input);
		await user().click(save);
		await vi.waitFor(() => expect(api.calls('PATCH /api/me/profile')).toHaveLength(1));
		expect(api.calls('PATCH /api/me/profile')[0].body).toEqual({ displayName: '' });
	});

	test('a name over 100 characters is rejected before sending', async () => {
		const api = mockApi({});
		const { input, save } = setup();
		await user().clear(input);
		await user().type(input, 'x'.repeat(DISPLAY_NAME_MAX + 1));
		await user().click(save);
		expect(screen.getByText(t('fields.too_long'))).toBeInTheDocument();
		expect(input).toHaveAttribute('aria-invalid', 'true');
		expect(api.requests).toHaveLength(0);
		await user().type(input, '{Backspace}');
		expect(input).not.toHaveAttribute('aria-invalid');
	});

	test('a server field error lands on the field', async () => {
		mockApi({
			'PATCH /api/me/profile': apiError(400, 'validation_failed', { displayName: 'invalid' })
		});
		const { input, save } = setup();
		await user().type(input, 'x');
		await user().click(save);
		expect(await screen.findByText(t('fields.invalid'))).toBeInTheDocument();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	test('other failures show the error alert and re-enable the button', async () => {
		mockApi({ 'PATCH /api/me/profile': apiError(500, 'internal_error') });
		const { input, save } = setup();
		await user().type(input, 'x');
		await user().click(save);
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.internal_error.title'));
		expect(save).toBeEnabled();
	});

	test('a network failure reads as a network error and shows the spinner meanwhile', async () => {
		let fail: (reason: unknown) => void = () => {};
		vi.stubGlobal(
			'fetch',
			vi.fn(() => new Promise((_, reject) => (fail = reject)))
		);
		const { input, save } = setup();
		await user().type(input, 'x');
		await user().click(save);
		expect(save).toBeDisabled();
		expect(screen.getByLabelText(t('common.states.loading'))).toBeInTheDocument();
		fail(new TypeError('offline'));
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.network_error.title'));
	});
});
