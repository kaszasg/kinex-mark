import { render, screen, waitFor } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { fmt, t } from '$lib/i18n/index.js';
import { publicConfig } from '$lib/state/public-config.svelte.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe, makePublicConfig } from '$lib/test/fixtures.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import PrivacyPage from './PrivacyPage.svelte';

const config = makePublicConfig({
	privacy: {
		...makePublicConfig().privacy,
		noticeVersion: 2,
		updatedAt: '2026-10-01T08:00:00Z',
		controllerName: 'Kinexium'
	}
});

/** toHaveTextContent collapses no-break spaces in the element text only. */
const plain = (text: string): string => text.replace(/ /g, ' ');

beforeEach(() => {
	publicConfig.config = null;
});

afterEach(() => {
	vi.unstubAllGlobals();
	session.clear();
	publicConfig.config = null;
});

describe('PrivacyPage', () => {
	test('loads the notice without a session and shows version and date', async () => {
		mockApi({
			'GET /api/public/config': json(config),
			'GET /api/auth/me': apiError(401, 'unauthenticated')
		});
		renderWithApp(PrivacyPage, {});
		expect(document.title).toBe(`${t('privacy.title')} – Kinex Mark`);
		expect(screen.getByRole('heading', { level: 1, name: t('privacy.title') })).toBeVisible();
		expect(
			await screen.findByRole('heading', { level: 2, name: t('privacy.sections.controller.title') })
		).toBeInTheDocument();
		expect(document.querySelector('[data-notice-version]')).toHaveTextContent(
			t('privacy.version', { version: 2 })
		);
		expect(document.querySelector('[data-notice-updated]')).toHaveTextContent(
			plain(t('privacy.updatedAt', { date: fmt.date('2026-10-01T08:00:00Z') }))
		);
		expect(screen.queryByRole('link', { name: t('privacy.backToApp') })).toBeNull();
	});

	test('a signed-in user gets the link back to the app', async () => {
		mockApi({ 'GET /api/public/config': json(config) });
		renderWithApp(PrivacyPage, {}, { me: makeMe() });
		expect(screen.getByRole('link', { name: t('privacy.backToApp') })).toHaveAttribute('href', '/');
		await screen.findByRole('heading', { level: 2, name: t('privacy.sections.rights.title') });
	});

	test('the back link appears once the session probe finds a user', async () => {
		const api = mockApi({
			'GET /api/public/config': json(config),
			'GET /api/auth/me': json(makeMe())
		});
		// A fresh page load: the session has not been probed yet.
		session.clear();
		session.status = 'unknown';
		render(PrivacyPage, {});
		expect(await screen.findByRole('link', { name: t('privacy.backToApp') })).toBeVisible();
		expect(api.calls('GET /api/auth/me')).toHaveLength(1);
	});

	test('an anonymous probe keeps the back link hidden', async () => {
		const api = mockApi({
			'GET /api/public/config': json(config),
			'GET /api/auth/me': apiError(401, 'unauthenticated')
		});
		session.clear();
		session.status = 'unknown';
		render(PrivacyPage, {});
		await waitFor(() => expect(api.calls('GET /api/auth/me')).toHaveLength(1));
		await screen.findByRole('heading', { level: 2, name: t('privacy.sections.data.title') });
		expect(screen.queryByRole('link', { name: t('privacy.backToApp') })).toBeNull();
	});

	test('shows the loading rows while the notice loads', async () => {
		let release: () => void = () => {};
		const pending = new Promise<void>((resolve) => (release = resolve));
		mockApi({
			'GET /api/public/config': async () => {
				await pending;
				return json(config);
			}
		});
		renderWithApp(PrivacyPage, {});
		expect(screen.getByRole('status')).toHaveTextContent(t('common.states.loading'));
		expect(await screen.findAllByTestId('loading-row')).toHaveLength(8);
		release();
		expect(
			await screen.findByRole('heading', { level: 2, name: t('privacy.sections.data.title') })
		).toBeInTheDocument();
		expect(screen.queryByTestId('loading-row')).toBeNull();
	});

	test('a failed load shows the error with a working retry', async () => {
		const api = mockApi({ 'GET /api/public/config': apiError(500, 'internal_error') });
		renderWithApp(PrivacyPage, {});
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(t('errors.network_error.title'));
		expect(document.querySelector('[data-notice-version]')).toBeNull();
		api.route('GET /api/public/config', json(config));
		await userEvent.click(screen.getByRole('button', { name: t('common.actions.retry') }));
		await waitFor(() => expect(screen.queryByRole('alert')).toBeNull());
		expect(
			screen.getByRole('heading', { level: 2, name: t('privacy.sections.data.title') })
		).toBeInTheDocument();
	});

	test('an already loaded configuration renders at once', () => {
		const api = mockApi({ 'GET /api/auth/me': apiError(401, 'unauthenticated') });
		publicConfig.config = config;
		renderWithApp(PrivacyPage, {});
		expect(
			screen.getByRole('heading', { level: 2, name: t('privacy.sections.storage.title') })
		).toBeInTheDocument();
		expect(api.calls('GET /api/public/config')).toHaveLength(0);
	});
});
