// Session store: probe, sign-in, gates, permissions, auth failures (plan 8.6).
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import { setMode } from 'mode-watcher';
import { request } from '$lib/api/client.js';
import { ApiError } from '$lib/api/errors.js';
import { i18n } from '$lib/i18n/index.js';
import { makeMe, makePreferences, makePublicConfig } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { publicConfig } from './public-config.svelte.js';
import { SessionStore, session } from './session.svelte.js';

vi.mock('mode-watcher', async (importOriginal) => ({
	...(await importOriginal<typeof import('mode-watcher')>()),
	setMode: vi.fn()
}));

beforeEach(() => {
	session.clear();
	vi.mocked(goto).mockClear();
	vi.mocked(setMode).mockClear();
});

afterEach(() => {
	vi.unstubAllGlobals();
	window.history.replaceState(null, '', '/');
});

describe('ensure', () => {
	test('a signed-in probe stores the user once (single flight)', async () => {
		const me = makeMe('admin', { preferences: makePreferences({ locale: 'en', theme: 'light' }) });
		const api = mockApi({ 'GET /api/auth/me': json(me) });
		const store = new SessionStore();
		const [a, b] = await Promise.all([store.ensure(), store.ensure()]);
		expect(a).toEqual(me);
		expect(b).toEqual(me);
		expect(api.calls('GET /api/auth/me')).toHaveLength(1);
		expect(store.status).toBe('authenticated');
		expect(i18n.locale).toBe('en');
		expect(setMode).toHaveBeenCalledWith('light');
		await expect(store.ensure()).resolves.toEqual(me);
		expect(api.calls('GET /api/auth/me')).toHaveLength(1);
	});

	test('401 marks the session anonymous and is cached', async () => {
		const api = mockApi({ 'GET /api/auth/me': apiError(401, 'unauthenticated') });
		const store = new SessionStore();
		await expect(store.ensure()).resolves.toBeNull();
		expect(store.status).toBe('anonymous');
		await expect(store.ensure()).resolves.toBeNull();
		expect(api.requests).toHaveLength(1);
		expect(goto).not.toHaveBeenCalled();
	});

	test('other errors are rethrown and the probe can be retried', async () => {
		const api = mockApi({ 'GET /api/auth/me': apiError(500, 'internal_error') });
		const store = new SessionStore();
		await expect(store.ensure()).rejects.toBeInstanceOf(ApiError);
		expect(store.status).toBe('unknown');
		api.route('GET /api/auth/me', json(makeMe()));
		await expect(store.ensure()).resolves.not.toBeNull();
	});
});

describe('sign-in', () => {
	test('login with an ok answer signs in', async () => {
		const me = makeMe();
		mockApi({ 'POST /api/auth/login': json({ status: 'ok', user: me }) });
		const store = new SessionStore();
		await expect(store.login('kanna', 'pw')).resolves.toEqual({ status: 'ok', user: me });
		expect(store.me).toEqual(me);
	});

	test('login with a TOTP challenge waits for the second step', async () => {
		mockApi({ 'POST /api/auth/login': json({ status: 'totp_required', challenge: 'c1' }) });
		const store = new SessionStore();
		await store.login('kanna', 'pw');
		expect(store.me).toBeNull();
	});

	test('verifyTotp sends a code or a recovery code', async () => {
		const me = makeMe();
		const api = mockApi({ 'POST /api/auth/login/totp': json({ status: 'ok', user: me }) });
		const store = new SessionStore();
		await store.verifyTotp({ challenge: 'c', code: '123456' });
		await store.verifyTotp({ challenge: 'c', recoveryCode: 'abcd-efgh' });
		await store.verifyTotp({ challenge: 'c' });
		expect(api.requests.map((r) => r.body)).toEqual([
			{ challenge: 'c', code: '123456' },
			{ challenge: 'c', recoveryCode: 'abcd-efgh' },
			{ challenge: 'c', code: '' }
		]);
		expect(store.status).toBe('authenticated');
	});
});

describe('setMe and the explicit locale', () => {
	test('a hand-picked locale wins and is saved to the preference', async () => {
		const api = mockApi({ 'PATCH /api/me/preferences': json(makePreferences({ locale: 'de' })) });
		i18n.setLocale('de', { explicit: true, persist: false });
		const store = new SessionStore();
		store.setMe(makeMe('developer', { preferences: makePreferences({ locale: 'hu' }) }));
		expect(i18n.locale).toBe('de');
		expect(store.me?.preferences.locale).toBe('de');
		expect(i18n.explicit).toBeNull();
		await vi.waitFor(() => expect(api.calls('PATCH /api/me/preferences')).toHaveLength(1));
		expect(api.requests[0].body).toEqual({ locale: 'de' });
	});

	test('a failed preference save keeps the local choice', async () => {
		mockApi({ 'PATCH /api/me/preferences': apiError(500, 'internal_error') });
		i18n.setLocale('en', { explicit: true, persist: false });
		const store = new SessionStore();
		store.setMe(makeMe());
		await new Promise((resolve) => setTimeout(resolve, 0));
		expect(i18n.locale).toBe('en');
	});

	test('the same explicit locale needs no save', () => {
		const api = mockApi({});
		i18n.setLocale('hu', { explicit: true, persist: false });
		new SessionStore().setMe(makeMe());
		expect(api.requests).toHaveLength(0);
	});
});

describe('gates and permissions', () => {
	test('password gate comes before consent', () => {
		const store = new SessionStore();
		expect(store.gate).toBeNull();
		store.setMe(makeMe('customer', { mustChangePassword: true, privacyConsentRequired: true }));
		expect(store.gate).toBe('password');
		store.setMe(makeMe('customer', { privacyConsentRequired: true }));
		expect(store.gate).toBe('consent');
		store.setMe(makeMe('customer'));
		expect(store.gate).toBeNull();
	});

	test('can() follows the permissions', () => {
		const store = new SessionStore();
		expect(store.can('share')).toBe(false);
		store.setMe(makeMe('admin'));
		expect([store.can('share'), store.can('listUsers'), store.can('admin')]).toEqual([
			true,
			true,
			true
		]);
		store.setMe(makeMe('developer'));
		expect([store.can('share'), store.can('listUsers'), store.can('admin')]).toEqual([
			true,
			true,
			false
		]);
		store.setMe(makeMe('customer'));
		expect([store.can('share'), store.can('listUsers'), store.can('admin')]).toEqual([
			false,
			false,
			false
		]);
	});
});

describe('logout, refresh and clear', () => {
	test('logout ignores server errors, clears and opens the login page', async () => {
		mockApi({ 'POST /api/auth/logout': apiError(500, 'internal_error') });
		const store = new SessionStore();
		store.setMe(makeMe());
		await store.logout();
		expect(store.me).toBeNull();
		expect(store.lastClearReason).toBe('logout');
		expect(goto).toHaveBeenCalledWith('/login');
	});

	test('refresh reloads the user', async () => {
		mockApi({ 'GET /api/auth/me': json(makeMe('admin', { displayName: 'New' })) });
		const store = new SessionStore();
		await expect(store.refresh()).resolves.toMatchObject({ displayName: 'New' });
		expect(store.me?.displayName).toBe('New');
	});

	test('clear forgets the user with a reason', () => {
		const store = new SessionStore();
		store.setMe(makeMe());
		store.clear('deleted');
		expect(store.status).toBe('anonymous');
		expect(store.lastClearReason).toBe('deleted');
		store.clear();
		expect(store.lastClearReason).toBeNull();
	});
});

describe('auth failures from the API client', () => {
	test('an expired session goes to the login page with the return path', async () => {
		window.history.replaceState(null, '', '/tags?x=1');
		session.setMe(makeMe());
		mockApi({ 'GET /api/tags': apiError(401, 'unauthenticated') });
		await expect(request('GET', '/tags')).rejects.toBeInstanceOf(ApiError);
		await vi.waitFor(() =>
			expect(goto).toHaveBeenCalledWith('/login?next=%2Ftags%3Fx%3D1&reason=expired')
		);
		expect(session.me).toBeNull();
		expect(session.lastClearReason).toBe('expired');
	});

	test('on the login page itself there is no redirect', async () => {
		window.history.replaceState(null, '', '/login');
		await session.handleAuthFailure('unauthenticated');
		expect(goto).not.toHaveBeenCalled();
	});

	test('an open gate refreshes the user and opens the gate page', async () => {
		mockApi({
			'GET /api/auth/me': json(makeMe('developer', { mustChangePassword: true })),
			'POST /api/auth/logout': noContent()
		});
		await session.handleAuthFailure('password_change_required');
		expect(goto).toHaveBeenCalledWith('/change-password');
		expect(session.gate).toBe('password');
	});

	test('consent_required reloads the public config, so the gate shows the bumped notice (E1)', async () => {
		publicConfig.config = makePublicConfig({
			privacy: { ...makePublicConfig().privacy, noticeVersion: 2 }
		});
		const api = mockApi({
			'GET /api/auth/me': json(
				makeMe('developer', { privacyConsentRequired: true, privacyNoticeVersion: 3 })
			),
			'GET /api/public/config': json(
				makePublicConfig({ privacy: { ...makePublicConfig().privacy, noticeVersion: 3 } })
			)
		});
		await session.handleAuthFailure('consent_required');
		expect(api.calls('GET /api/public/config')).toHaveLength(1);
		expect(publicConfig.config?.privacy.noticeVersion).toBe(3);
		expect(goto).toHaveBeenCalledWith('/consent');
		publicConfig.config = null;
	});

	test('a password gate leaves the cached public config alone', async () => {
		const api = mockApi({
			'GET /api/auth/me': json(makeMe('developer', { mustChangePassword: true }))
		});
		await session.handleAuthFailure('password_change_required');
		expect(api.calls('GET /api/public/config')).toHaveLength(0);
	});

	test('consent gate even when the refresh fails', async () => {
		mockApi({ 'GET /api/auth/me': apiError(500, 'internal_error') });
		await session.handleAuthFailure('consent_required');
		expect(goto).toHaveBeenCalledWith('/consent');
	});
});
