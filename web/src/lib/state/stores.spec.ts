// Preferences, public config, tags and link-stats stores (plan 8.6).
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { setMode } from 'mode-watcher';
import { toast } from 'svelte-sonner';
import { i18n } from '$lib/i18n/index.js';
import { makeMe, makePreferences, makePublicConfig, makeTag } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { linkStats, LinkStatsStore, LINK_STATS_INTERVAL_MS } from './link-stats.svelte.js';
import { preferences, PREFERENCES_DEBOUNCE_MS, PreferencesStore } from './preferences.svelte.js';
import { PublicConfigStore } from './public-config.svelte.js';
import { session, SessionStore } from './session.svelte.js';
import { compareTagNames, tags, TagsStore } from './tags.svelte.js';

vi.mock('mode-watcher', async (importOriginal) => ({
	...(await importOriginal<typeof import('mode-watcher')>()),
	setMode: vi.fn()
}));

beforeEach(() => {
	vi.mocked(setMode).mockClear();
});

afterEach(() => {
	vi.useRealTimers();
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});

describe('PreferencesStore', () => {
	function signedIn() {
		const sessionStore = new SessionStore();
		sessionStore.setMe(makeMe('developer', { preferences: makePreferences({ theme: 'dark' }) }));
		return { sessionStore, prefs: new PreferencesStore(sessionStore) };
	}

	test('merges changes within the window into one PATCH', async () => {
		vi.useFakeTimers();
		const api = mockApi({
			'PATCH /api/me/preferences': (req) =>
				json(makePreferences({ ...(req.body as object), theme: 'light', showUrl: false }))
		});
		const { sessionStore, prefs } = signedIn();
		prefs.update({ theme: 'light' });
		prefs.update({ showUrl: false });
		expect(sessionStore.me?.preferences.theme).toBe('light');
		expect(setMode).toHaveBeenLastCalledWith('light');
		expect(prefs.saving).toBe(true);
		await vi.advanceTimersByTimeAsync(PREFERENCES_DEBOUNCE_MS);
		expect(api.requests).toHaveLength(1);
		expect(api.requests[0].body).toEqual({ theme: 'light', showUrl: false });
		expect(prefs.saving).toBe(false);
		expect(prefs.current?.showUrl).toBe(false);
	});

	test('a locale change applies at once', async () => {
		mockApi({ 'PATCH /api/me/preferences': json(makePreferences({ locale: 'de' })) });
		const { prefs } = signedIn();
		prefs.update({ locale: 'de' });
		expect(i18n.locale).toBe('de');
		await prefs.flush();
	});

	test('a failed save reverts every field but the locale and shows a toast', async () => {
		const error = vi.spyOn(toast, 'error').mockImplementation(() => 1);
		mockApi({ 'PATCH /api/me/preferences': apiError(400, 'validation_failed') });
		const { sessionStore, prefs } = signedIn();
		prefs.update({ theme: 'light', locale: 'en' });
		await prefs.flush();
		expect(sessionStore.me?.preferences.theme).toBe('dark');
		expect(sessionStore.me?.preferences.locale).toBe('en');
		expect(setMode).toHaveBeenLastCalledWith('dark');
		expect(error).toHaveBeenCalledOnce();
	});

	test('a network failure is reported too', async () => {
		const error = vi.spyOn(toast, 'error').mockImplementation(() => 1);
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw new TypeError('offline');
			})
		);
		const { prefs } = signedIn();
		prefs.update({ showTitle: false });
		await prefs.flush();
		expect(error).toHaveBeenCalledOnce();
	});

	test('a non-API exception is reported as a network error', async () => {
		const error = vi.spyOn(toast, 'error').mockImplementation(() => 1);
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => new Response('{not json', { status: 200 }))
		);
		const { prefs } = signedIn();
		prefs.update({ showTitle: false });
		await prefs.flush();
		expect(error).toHaveBeenCalledOnce();
	});

	test('changes made during a running save stay on top', async () => {
		let release: (() => void) | undefined;
		let calls = 0;
		mockApi({
			'PATCH /api/me/preferences': (req) => {
				calls += 1;
				if (calls > 1) return json(makePreferences(req.body as object));
				return new Promise((resolve) => {
					release = () => resolve(json(makePreferences({ showIcon: false })));
				});
			}
		});
		const { sessionStore, prefs } = signedIn();
		prefs.update({ showIcon: false });
		const running = prefs.flush();
		await vi.waitFor(() => expect(release).toBeDefined());
		prefs.update({ sort: 'title' });
		release?.();
		await running;
		expect(sessionStore.me?.preferences.sort).toBe('title');
		expect(sessionStore.me?.preferences.showIcon).toBe(false);
		await prefs.flush();
		expect(calls).toBe(2);
	});

	test('does nothing while signed out or without changes', async () => {
		const api = mockApi({});
		const prefs = new PreferencesStore(new SessionStore());
		prefs.update({ theme: 'light' });
		await prefs.flush();
		expect(prefs.current).toBeNull();
		expect(api.requests).toHaveLength(0);
	});

	test('reset drops a change that waits for the debounce', async () => {
		vi.useFakeTimers();
		const api = mockApi({ 'PATCH /api/me/preferences': json(makePreferences()) });
		const { prefs } = signedIn();
		prefs.update({ showUrl: false });
		prefs.reset();
		expect(prefs.saving).toBe(false);
		await vi.advanceTimersByTimeAsync(PREFERENCES_DEBOUNCE_MS);
		expect(api.requests).toHaveLength(0);
	});

	test('a server answer after sign-out is dropped', async () => {
		mockApi({ 'PATCH /api/me/preferences': json(makePreferences()) });
		const { sessionStore, prefs } = signedIn();
		prefs.update({ theme: 'light' });
		sessionStore.clear();
		await prefs.flush();
		expect(sessionStore.me).toBeNull();
	});
});

describe('PublicConfigStore', () => {
	test('loads once and caches', async () => {
		const api = mockApi({ 'GET /api/public/config': json(makePublicConfig()) });
		const store = new PublicConfigStore();
		const [a, b] = await Promise.all([store.ensure(), store.ensure()]);
		expect(a).toEqual(b);
		await store.ensure();
		expect(api.requests).toHaveLength(1);
		expect(store.config?.appName).toBe('Kinex Mark');
	});

	test('a failure resolves null and retries later', async () => {
		const api = mockApi({ 'GET /api/public/config': apiError(500, 'internal_error') });
		const store = new PublicConfigStore();
		await expect(store.ensure()).resolves.toBeNull();
		api.route('GET /api/public/config', json(makePublicConfig()));
		await expect(store.ensure()).resolves.not.toBeNull();
	});

	test('refresh loads past the cache once and keeps the copy on failure', async () => {
		let version = 1;
		const api = mockApi({
			'GET /api/public/config': () =>
				json(
					makePublicConfig({ privacy: { ...makePublicConfig().privacy, noticeVersion: version } })
				)
		});
		const store = new PublicConfigStore();
		await store.ensure();
		version = 2;
		const [a, b] = await Promise.all([store.refresh(), store.refresh()]);
		expect(a).toBe(b);
		expect(api.calls('GET /api/public/config')).toHaveLength(2);
		expect(store.config?.privacy.noticeVersion).toBe(2);
		api.route('GET /api/public/config', apiError(500, 'internal_error'));
		await expect(store.refresh()).resolves.toBeNull();
		expect(store.config?.privacy.noticeVersion).toBe(2);
	});
});

describe('TagsStore', () => {
	const infra = makeTag({ id: 1, name: 'infra', activeLinkCount: 2 });
	const alpha = makeTag({ id: 2, name: 'Alpha', activeLinkCount: 9 });
	const idle = makeTag({ id: 3, name: 'idle', activeLinkCount: 0 });

	test('loads in name order and derives byId and frequent tags', async () => {
		const api = mockApi({ 'GET /api/tags': json({ items: [infra, idle, alpha] }) });
		const store = new TagsStore();
		await Promise.all([store.ensure(1), store.ensure(1)]);
		expect(store.items.map((t) => t.name)).toEqual(['Alpha', 'idle', 'infra']);
		expect(store.byId.get(1)?.name).toBe('infra');
		expect(store.frequent.map((t) => t.id)).toEqual([2, 1]);
		expect(store.status).toBe('ready');
		await store.ensure(1);
		expect(api.requests).toHaveLength(1);
		await store.ensure(2);
		expect(api.requests).toHaveLength(2);
	});

	test('a failed load reports the error and retries on ensure', async () => {
		const api = mockApi({ 'GET /api/tags': apiError(500, 'internal_error') });
		const store = new TagsStore();
		await store.ensure(1);
		expect(store.status).toBe('error');
		expect(store.error?.code).toBe('internal_error');
		api.route('GET /api/tags', json({ items: [] }));
		await store.ensure(1);
		expect(store.status).toBe('ready');
	});

	test('create, update and remove keep the order and notify listeners', async () => {
		const created = makeTag({ id: 9, name: 'beta', color: 'neutral' });
		const api = mockApi({
			'GET /api/tags': json({ items: [infra] }),
			'POST /api/tags': json(created, 201),
			'PATCH /api/tags/9': json({ ...created, name: 'zeta' }),
			'DELETE /api/tags/1': noContent()
		});
		const store = new TagsStore();
		await store.load();
		const listener = vi.fn();
		const off = store.onTagsChanged(listener);
		await store.create('beta');
		expect(api.calls('POST /api/tags')[0].body).toEqual({ name: 'beta', color: 'neutral' });
		expect(store.items.map((t) => t.name)).toEqual(['beta', 'infra']);
		await store.update(9, { name: 'zeta' });
		expect(store.items.map((t) => t.name)).toEqual(['infra', 'zeta']);
		await store.remove(1);
		expect(store.items.map((t) => t.name)).toEqual(['zeta']);
		expect(listener).toHaveBeenCalledTimes(3);
		off();
		store.reset();
		expect(store.items).toEqual([]);
		expect(store.status).toBe('idle');
	});

	test('refresh reloads the counts after a link change, but not before the first load', async () => {
		const api = mockApi({ 'GET /api/tags': json({ items: [infra] }) });
		const store = new TagsStore();
		await store.refresh();
		expect(api.requests).toHaveLength(0);
		await store.load();
		api.route('GET /api/tags', json({ items: [{ ...infra, activeLinkCount: 3 }] }));
		const refreshing = store.refresh();
		// The shown tags stay while the counts reload.
		expect(store.status).toBe('ready');
		await refreshing;
		expect(store.byId.get(1)?.activeLinkCount).toBe(3);
		expect(api.requests).toHaveLength(2);
	});

	test('a refresh during a running load loads once more afterwards', async () => {
		let release: (() => void) | undefined;
		let calls = 0;
		mockApi({
			'GET /api/tags': () => {
				calls += 1;
				if (calls === 1) return json({ items: [infra] });
				if (calls === 2) {
					return new Promise((resolve) => {
						release = () => resolve(json({ items: [infra] }));
					});
				}
				return json({ items: [{ ...infra, activeLinkCount: 5 }] });
			}
		});
		const store = new TagsStore();
		await store.load();
		const running = store.load();
		const refreshing = store.refresh();
		await vi.waitFor(() => expect(release).toBeDefined());
		release?.();
		await running;
		await refreshing;
		expect(calls).toBe(3);
		expect(store.byId.get(1)?.activeLinkCount).toBe(5);
	});

	test('reset drops an answer that arrives later and lets the next user load', async () => {
		let release: (() => void) | undefined;
		let calls = 0;
		mockApi({
			'GET /api/tags': () => {
				calls += 1;
				if (calls > 1) return json({ items: [alpha] });
				return new Promise((resolve) => {
					release = () => resolve(json({ items: [infra] }));
				});
			}
		});
		const store = new TagsStore();
		const first = store.ensure(1);
		await vi.waitFor(() => expect(release).toBeDefined());
		store.reset();
		await store.ensure(2);
		release?.();
		await first;
		expect(store.items.map((t) => t.name)).toEqual(['Alpha']);
		expect(calls).toBe(2);
	});

	test('compareTagNames breaks ties by id', () => {
		expect(compareTagNames({ id: 1, name: 'a' }, { id: 2, name: 'A' })).toBeLessThan(0);
		expect(compareTagNames({ id: 1, name: 'b' }, { id: 2, name: 'a' })).toBeGreaterThan(0);
	});
});

describe('LinkStatsStore', () => {
	test('refresh reads the total of non-archived links and keeps it on failure', async () => {
		const api = mockApi({ 'GET /api/links': json({ items: [], total: 42, limit: 1, offset: 0 }) });
		const store = new LinkStatsStore();
		await store.refresh();
		expect(store.active).toBe(42);
		expect(Object.fromEntries(api.requests[0].url.searchParams)).toEqual({
			limit: '1',
			archived: 'exclude'
		});
		api.route('GET /api/links', apiError(500, 'internal_error'));
		await store.refresh();
		expect(store.active).toBe(42);
		store.reset();
		expect(store.active).toBeNull();
	});

	test('an answer that arrives after a reset is dropped', async () => {
		let release: (() => void) | undefined;
		mockApi({
			'GET /api/links': () =>
				new Promise((resolve) => {
					release = () => resolve(json({ items: [], total: 7, limit: 1, offset: 0 }));
				})
		});
		const store = new LinkStatsStore();
		const pending = store.refresh();
		await vi.waitFor(() => expect(release).toBeDefined());
		store.reset();
		release?.();
		await pending;
		expect(store.active).toBeNull();
	});

	test('start refreshes now and every minute while focused', async () => {
		vi.useFakeTimers();
		const api = mockApi({ 'GET /api/links': json({ items: [], total: 1, limit: 1, offset: 0 }) });
		const focus = vi.spyOn(document, 'hasFocus').mockReturnValue(true);
		const store = new LinkStatsStore();
		const stop = store.start();
		await vi.advanceTimersByTimeAsync(LINK_STATS_INTERVAL_MS);
		expect(api.requests).toHaveLength(2);
		focus.mockReturnValue(false);
		await vi.advanceTimersByTimeAsync(LINK_STATS_INTERVAL_MS);
		expect(api.requests).toHaveLength(2);
		stop();
		focus.mockReturnValue(true);
		await vi.advanceTimersByTimeAsync(LINK_STATS_INTERVAL_MS);
		expect(api.requests).toHaveLength(2);
	});
});

describe('per-user stores and the session', () => {
	afterEach(() => {
		session.clear();
	});

	test('a sign-out or another user empties the tags, the counter and pending preferences', async () => {
		vi.useFakeTimers();
		const api = mockApi({
			'GET /api/tags': json({ items: [makeTag({ id: 1, name: 'privat' })] }),
			'PATCH /api/me/preferences': json(makePreferences())
		});
		session.setMe(makeMe('customer', { id: 61 }));
		await tags.ensure(61);
		linkStats.active = 4;
		preferences.update({ showUrl: false });
		expect(tags.items).toHaveLength(1);

		session.clear('logout');
		expect(tags.items).toEqual([]);
		expect(tags.status).toBe('idle');
		expect(linkStats.active).toBeNull();
		await vi.advanceTimersByTimeAsync(PREFERENCES_DEBOUNCE_MS);
		expect(api.calls('PATCH /api/me/preferences')).toHaveLength(0);

		session.setMe(makeMe('developer', { id: 62 }));
		await tags.ensure(62);
		linkStats.active = 2;
		session.setMe(makeMe('admin', { id: 63 }));
		expect(tags.status).toBe('idle');
		expect(linkStats.active).toBeNull();
	});

	test('listeners can unsubscribe, and a refresh of the same user keeps the data', () => {
		const store = new SessionStore();
		const listener = vi.fn();
		const off = store.onUserChange(listener);
		store.clear();
		expect(listener).not.toHaveBeenCalled();
		store.setMe(makeMe('developer', { id: 5 }));
		store.setMe(makeMe('developer', { id: 5 }));
		expect(listener).not.toHaveBeenCalled();
		store.clear('logout');
		expect(listener).toHaveBeenCalledOnce();
		off();
		store.setMe(makeMe('developer', { id: 6 }));
		store.clear('logout');
		expect(listener).toHaveBeenCalledOnce();
	});
});
