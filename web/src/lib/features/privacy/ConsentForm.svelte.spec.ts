import { screen, waitFor } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import { fmt, t } from '$lib/i18n/index.js';
import { publicConfig } from '$lib/state/public-config.svelte.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe, makePublicConfig } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import ConsentForm from './ConsentForm.svelte';

const gateUser = () =>
	makeMe('developer', { privacyConsentRequired: true, privacyNoticeVersion: 3 });
const configV = (version: number, updatedAt: string | null = '2026-11-05T09:00:00Z') =>
	makePublicConfig({
		privacy: { ...makePublicConfig().privacy, noticeVersion: version, updatedAt }
	});

beforeEach(() => {
	vi.mocked(goto).mockClear();
	publicConfig.config = null;
});

afterEach(() => {
	vi.unstubAllGlobals();
	session.clear();
	publicConfig.config = null;
});

const checkbox = () => screen.getByRole('checkbox', { name: t('privacy.consent.checkbox') });
const submit = () =>
	screen.getByRole('button', { name: new RegExp(`${t('privacy.consent.submit')}$`) });

describe('ConsentForm', () => {
	test('shows the loading rows, then the notice with its version and date', async () => {
		let release: () => void = () => {};
		const gate = new Promise<void>((resolve) => (release = resolve));
		mockApi({
			'GET /api/public/config': async () => {
				await gate;
				return json(configV(3));
			}
		});
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		expect(
			screen.getByRole('heading', { level: 1, name: t('privacy.consent.title') })
		).toBeVisible();
		expect(screen.getByText(t('privacy.consent.text'))).toBeInTheDocument();
		expect(screen.queryByRole('checkbox')).toBeNull();
		release();
		expect(await screen.findByRole('region', { name: t('privacy.consent.title') })).toHaveAttribute(
			'tabindex',
			'0'
		);
		const version = document.querySelector('[data-notice-version]');
		expect(version).toHaveTextContent(t('privacy.version', { version: 3 }));
		// toHaveTextContent collapses no-break spaces in the element text only.
		expect(version).toHaveTextContent(fmt.date('2026-11-05T09:00:00Z').replace(/ /g, ' '));
		expect(screen.getAllByRole('heading', { level: 3 }).length).toBe(12);
		const full = screen.getByRole('link', { name: new RegExp(t('privacy.consent.openFull')) });
		expect(full).toHaveAttribute('href', '/privacy');
		expect(full).toHaveAttribute('target', '_blank');
		expect(full).toHaveAttribute('rel', 'noopener');
	});

	test('a notice without a publication date shows only the version', async () => {
		mockApi({ 'GET /api/public/config': json(configV(1, null)) });
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		await screen.findByRole('checkbox');
		expect(document.querySelector('[data-notice-version]')?.textContent?.trim()).toBe(
			t('privacy.version', { version: 1 })
		);
	});

	test('a failed load shows the error with a working retry', async () => {
		const api = mockApi({ 'GET /api/public/config': apiError(500, 'internal_error') });
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(t('errors.network_error.title'));
		api.route('GET /api/public/config', json(configV(3)));
		await userEvent.click(screen.getByRole('button', { name: t('common.actions.retry') }));
		expect(await screen.findByRole('checkbox')).toBeInTheDocument();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	test('the button stays disabled until the notice is acknowledged', async () => {
		const api = mockApi({ 'GET /api/public/config': json(configV(3)) });
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		await screen.findByRole('checkbox');
		expect(submit()).toBeDisabled();
		await userEvent.click(screen.getByText(t('privacy.consent.checkbox')));
		expect(checkbox()).toBeChecked();
		expect(submit()).toBeEnabled();
		await userEvent.click(checkbox());
		expect(submit()).toBeDisabled();
		expect(api.calls('POST /api/me/privacy/consent')).toHaveLength(0);
	});

	test('acknowledging sends the shown version, refreshes the user and continues', async () => {
		const api = mockApi({
			'GET /api/public/config': json(configV(3)),
			'POST /api/me/privacy/consent': noContent(),
			'GET /api/auth/me': json(makeMe('developer'))
		});
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		await userEvent.click(await screen.findByRole('checkbox'));
		await userEvent.click(submit());
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/', { replaceState: true }));
		expect(api.calls('POST /api/me/privacy/consent')[0].body).toEqual({ version: 3 });
		expect(session.gate).toBeNull();
	});

	test('a failed refresh still closes the consent gate locally', async () => {
		mockApi({
			'GET /api/public/config': json(configV(3)),
			'POST /api/me/privacy/consent': noContent(),
			'GET /api/auth/me': apiError(500, 'internal_error')
		});
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		await userEvent.click(await screen.findByRole('checkbox'));
		await userEvent.click(submit());
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/', { replaceState: true }));
		expect(session.me?.privacyConsentRequired).toBe(false);
	});

	test('a newer notice reloads the text, unticks the box and says so', async () => {
		let version = 3;
		const api = mockApi({
			'GET /api/public/config': () => json(configV(version)),
			'POST /api/me/privacy/consent': () => {
				version = 4;
				return apiError(409, 'consent_version_mismatch');
			}
		});
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		await userEvent.click(await screen.findByRole('checkbox'));
		await userEvent.click(submit());
		expect(await screen.findByText(t('errors.consent_version_mismatch.title'))).toBeInTheDocument();
		await waitFor(() =>
			expect(document.querySelector('[data-notice-version]')).toHaveTextContent(
				t('privacy.version', { version: 4 })
			)
		);
		expect(checkbox()).not.toBeChecked();
		expect(submit()).toBeDisabled();
		expect(screen.queryByRole('alert')).toBeNull();
		expect(api.calls('GET /api/public/config')).toHaveLength(2);
		expect(goto).not.toHaveBeenCalled();

		await userEvent.click(checkbox());
		api.route('POST /api/me/privacy/consent', noContent());
		api.route('GET /api/auth/me', json(makeMe('developer')));
		await userEvent.click(submit());
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/', { replaceState: true }));
		expect(api.calls('POST /api/me/privacy/consent')[1].body).toEqual({ version: 4 });
	});

	test('a failed reload after a newer notice shows the error', async () => {
		let calls = 0;
		mockApi({
			'GET /api/public/config': () => {
				calls += 1;
				return calls === 1 ? json(configV(3)) : apiError(500, 'internal_error');
			},
			'POST /api/me/privacy/consent': apiError(409, 'consent_version_mismatch')
		});
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		await userEvent.click(await screen.findByRole('checkbox'));
		await userEvent.click(submit());
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.internal_error.title'));
	});

	test('other errors appear as an alert and the form stays', async () => {
		mockApi({
			'GET /api/public/config': json(configV(3)),
			'POST /api/me/privacy/consent': apiError(500, 'internal_error')
		});
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		await userEvent.click(await screen.findByRole('checkbox'));
		await userEvent.click(submit());
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.internal_error.title'));
		expect(checkbox()).toBeChecked();
		await waitFor(() => expect(checkbox()).toHaveFocus());
		expect(goto).not.toHaveBeenCalled();
	});

	test('an already loaded configuration of the current version is used at once', () => {
		const api = mockApi();
		publicConfig.config = configV(3);
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		expect(screen.getByRole('checkbox')).toBeInTheDocument();
		expect(api.calls('GET /api/public/config')).toHaveLength(0);
	});

	test('a cached notice older than the current version is loaded again before it is shown (E1)', async () => {
		const api = mockApi({
			'GET /api/public/config': json(configV(3)),
			'POST /api/me/privacy/consent': noContent(),
			'GET /api/auth/me': json(makeMe('developer'))
		});
		// The tab cached version 2; the notice was bumped to 3 while the user was signed in.
		publicConfig.config = configV(2);
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		expect(document.querySelector('[data-notice-version]')).toBeNull();
		await waitFor(() =>
			expect(document.querySelector('[data-notice-version]')).toHaveTextContent(
				t('privacy.version', { version: 3 })
			)
		);
		await userEvent.click(checkbox());
		await userEvent.click(submit());
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/', { replaceState: true }));
		expect(api.calls('POST /api/me/privacy/consent')).toHaveLength(1);
		expect(api.calls('POST /api/me/privacy/consent')[0].body).toEqual({ version: 3 });
	});

	test('a failed reload of an outdated notice shows the error with a retry', async () => {
		mockApi({ 'GET /api/public/config': apiError(500, 'internal_error') });
		publicConfig.config = configV(2);
		renderWithApp(ConsentForm, {}, { me: gateUser() });
		expect(
			await screen.findByRole('button', { name: t('common.actions.retry') })
		).toBeInTheDocument();
		expect(screen.queryByRole('checkbox')).toBeNull();
	});
});
