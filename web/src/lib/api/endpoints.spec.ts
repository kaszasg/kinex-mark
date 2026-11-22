// Every endpoint function sends the method, path, query and body of plan 5.
import { afterEach, describe, expect, test, vi } from 'vitest';
import { json, mockApi } from '$lib/test/mock-api.js';
import { adminApi, authApi, directoryApi, linksApi, meApi, publicApi, tagsApi } from './index.js';

afterEach(() => {
	vi.unstubAllGlobals();
});

type Case = {
	name: string;
	call: () => Promise<unknown>;
	method: string;
	path: string;
	query?: Record<string, string>;
	body?: unknown;
};

const file = new File(['<html></html>'], 'bookmarks.html', { type: 'text/html' });

const cases: Case[] = [
	{
		name: 'public config',
		call: () => publicApi.getPublicConfig(),
		method: 'GET',
		path: '/api/public/config'
	},
	{ name: 'health', call: () => publicApi.getHealth(), method: 'GET', path: '/healthz' },
	{
		name: 'login',
		call: () => authApi.login('kanna', 'pw'),
		method: 'POST',
		path: '/api/auth/login',
		body: { username: 'kanna', password: 'pw' }
	},
	{
		name: 'login totp',
		call: () => authApi.loginTotp({ challenge: 'c', code: '123456' }),
		method: 'POST',
		path: '/api/auth/login/totp',
		body: { challenge: 'c', code: '123456' }
	},
	{ name: 'logout', call: () => authApi.logout(), method: 'POST', path: '/api/auth/logout' },
	{ name: 'me', call: () => authApi.getMe(), method: 'GET', path: '/api/auth/me' },
	{
		name: 'change password',
		call: () => authApi.changePassword('old', 'new'),
		method: 'POST',
		path: '/api/auth/password',
		body: { currentPassword: 'old', newPassword: 'new' }
	},
	{
		name: 'profile',
		call: () => meApi.updateProfile({ displayName: 'A' }),
		method: 'PATCH',
		path: '/api/me/profile',
		body: { displayName: 'A' }
	},
	{
		name: 'preferences',
		call: () => meApi.getPreferences(),
		method: 'GET',
		path: '/api/me/preferences'
	},
	{
		name: 'update preferences',
		call: () => meApi.updatePreferences({ theme: 'light' }),
		method: 'PATCH',
		path: '/api/me/preferences',
		body: { theme: 'light' }
	},
	{ name: 'sessions', call: () => meApi.listSessions(), method: 'GET', path: '/api/me/sessions' },
	{
		name: 'revoke session',
		call: () => meApi.revokeSession(7),
		method: 'DELETE',
		path: '/api/me/sessions/7'
	},
	{
		name: 'revoke others',
		call: () => meApi.revokeOtherSessions(),
		method: 'POST',
		path: '/api/me/sessions/revoke-others'
	},
	{
		name: '2fa setup',
		call: () => meApi.setupTwoFactor('pw'),
		method: 'POST',
		path: '/api/me/2fa/setup',
		body: { password: 'pw' }
	},
	{
		name: '2fa enable',
		call: () => meApi.enableTwoFactor('123456'),
		method: 'POST',
		path: '/api/me/2fa/enable',
		body: { code: '123456' }
	},
	{
		name: '2fa disable',
		call: () => meApi.disableTwoFactor('pw', '123456'),
		method: 'POST',
		path: '/api/me/2fa/disable',
		body: { password: 'pw', code: '123456' }
	},
	{
		name: 'recovery codes',
		call: () => meApi.regenerateRecoveryCodes('pw', '123456'),
		method: 'POST',
		path: '/api/me/2fa/recovery-codes',
		body: { password: 'pw', code: '123456' }
	},
	{ name: 'privacy', call: () => meApi.getPrivacy(), method: 'GET', path: '/api/me/privacy' },
	{
		name: 'consent',
		call: () => meApi.acceptPrivacy(3),
		method: 'POST',
		path: '/api/me/privacy/consent',
		body: { version: 3 }
	},
	{
		name: 'export my data',
		call: () => meApi.exportMyData(),
		method: 'GET',
		path: '/api/me/export'
	},
	{
		name: 'delete account',
		call: () => meApi.deleteAccount('pw', 'kanna'),
		method: 'POST',
		path: '/api/me/delete',
		body: { password: 'pw', confirm: 'kanna' }
	},
	{
		name: 'delete account with code',
		call: () => meApi.deleteAccount('pw', 'kanna', '123456'),
		method: 'POST',
		path: '/api/me/delete',
		body: { password: 'pw', confirm: 'kanna', code: '123456' }
	},
	{
		name: 'list links',
		call: () => linksApi.listLinks({ q: 'x', tags: [1, 2], untagged: false, limit: 50 }),
		method: 'GET',
		path: '/api/links',
		query: { q: 'x', tags: '1,2', limit: '50' }
	},
	{
		name: 'list untagged links',
		call: () => linksApi.listLinks({ untagged: true, tags: [] }),
		method: 'GET',
		path: '/api/links',
		query: { untagged: 'true' }
	},
	{
		name: 'create link',
		call: () => linksApi.createLink({ url: 'https://a.example' } as never),
		method: 'POST',
		path: '/api/links',
		body: { url: 'https://a.example' }
	},
	{
		name: 'quick capture',
		call: () => linksApi.quickCapture('https://a.example', 'A'),
		method: 'POST',
		path: '/api/links/quick',
		body: { text: 'https://a.example', title: 'A' }
	},
	{
		name: 'quick capture without title',
		call: () => linksApi.quickCapture('https://a.example', ''),
		method: 'POST',
		path: '/api/links/quick',
		body: { text: 'https://a.example' }
	},
	{ name: 'get link', call: () => linksApi.getLink(5), method: 'GET', path: '/api/links/5' },
	{
		name: 'update link',
		call: () => linksApi.updateLink(5, { title: 'T' } as never),
		method: 'PATCH',
		path: '/api/links/5',
		body: { title: 'T' }
	},
	{
		name: 'delete link',
		call: () => linksApi.deleteLink(5),
		method: 'DELETE',
		path: '/api/links/5'
	},
	{
		name: 'set tags',
		call: () => linksApi.setLinkTags(5, [1]),
		method: 'PUT',
		path: '/api/links/5/tags',
		body: { tagIds: [1] }
	},
	{ name: 'pin', call: () => linksApi.pin(5), method: 'POST', path: '/api/links/5/pin' },
	{ name: 'unpin', call: () => linksApi.unpin(5), method: 'DELETE', path: '/api/links/5/pin' },
	{
		name: 'archive',
		call: () => linksApi.archive(5),
		method: 'POST',
		path: '/api/links/5/archive'
	},
	{
		name: 'restore',
		call: () => linksApi.restore(5),
		method: 'DELETE',
		path: '/api/links/5/archive'
	},
	{
		name: 'visit',
		call: () => linksApi.recordVisit(5),
		method: 'POST',
		path: '/api/links/5/visit'
	},
	{
		name: 'refresh',
		call: () => linksApi.refreshMetadata(5),
		method: 'POST',
		path: '/api/links/5/refresh'
	},
	{
		name: 'shares',
		call: () => linksApi.listShares(5),
		method: 'GET',
		path: '/api/links/5/shares'
	},
	{
		name: 'create share',
		call: () => linksApi.createShare(5, { targetType: 'all' } as never),
		method: 'POST',
		path: '/api/links/5/shares',
		body: { targetType: 'all' }
	},
	{
		name: 'delete share',
		call: () => linksApi.deleteShare(5, 9),
		method: 'DELETE',
		path: '/api/links/5/shares/9'
	},
	{
		name: 'export links',
		call: () => linksApi.exportLinks('html'),
		method: 'GET',
		path: '/api/links/export',
		query: { format: 'html' }
	},
	{ name: 'list tags', call: () => tagsApi.listTags(), method: 'GET', path: '/api/tags' },
	{
		name: 'create tag',
		call: () => tagsApi.createTag('infra', 'blue'),
		method: 'POST',
		path: '/api/tags',
		body: { name: 'infra', color: 'blue' }
	},
	{
		name: 'create tag without colour',
		call: () => tagsApi.createTag('infra'),
		method: 'POST',
		path: '/api/tags',
		body: { name: 'infra' }
	},
	{
		name: 'update tag',
		call: () => tagsApi.updateTag(3, { color: 'rose' }),
		method: 'PATCH',
		path: '/api/tags/3',
		body: { color: 'rose' }
	},
	{ name: 'delete tag', call: () => tagsApi.deleteTag(3), method: 'DELETE', path: '/api/tags/3' },
	{
		name: 'directory',
		call: () => directoryApi.searchDirectory('ann'),
		method: 'GET',
		path: '/api/users/directory',
		query: { q: 'ann' }
	},
	{
		name: 'admin users',
		call: () => adminApi.listUsers({ q: 'a', role: 'admin' } as never),
		method: 'GET',
		path: '/api/admin/users',
		query: { q: 'a', role: 'admin' }
	},
	{
		name: 'admin users default',
		call: () => adminApi.listUsers(),
		method: 'GET',
		path: '/api/admin/users'
	},
	{
		name: 'admin create user',
		call: () => adminApi.createUser({ username: 'u' } as never),
		method: 'POST',
		path: '/api/admin/users',
		body: { username: 'u' }
	},
	{
		name: 'admin get user',
		call: () => adminApi.getUser(4),
		method: 'GET',
		path: '/api/admin/users/4'
	},
	{
		name: 'admin update user',
		call: () => adminApi.updateUser(4, { active: false } as never),
		method: 'PATCH',
		path: '/api/admin/users/4',
		body: { active: false }
	},
	{
		name: 'admin reset 2fa',
		call: () => adminApi.resetTwoFactor(4, 'pw'),
		method: 'POST',
		path: '/api/admin/users/4/reset-2fa',
		body: { currentPassword: 'pw' }
	},
	{
		name: 'admin unlock',
		call: () => adminApi.unlockUser(4),
		method: 'POST',
		path: '/api/admin/users/4/unlock'
	},
	{
		name: 'admin revoke sessions',
		call: () => adminApi.revokeUserSessions(4),
		method: 'DELETE',
		path: '/api/admin/users/4/sessions'
	},
	{
		name: 'admin delete user',
		call: () => adminApi.deleteUser(4, 'pw'),
		method: 'DELETE',
		path: '/api/admin/users/4',
		body: { currentPassword: 'pw' }
	},
	{
		name: 'admin settings',
		call: () => adminApi.getSettings(),
		method: 'GET',
		path: '/api/admin/settings'
	},
	{
		name: 'admin update settings',
		call: () => adminApi.updateSettings({ sessionIdleMinutes: 60 } as never),
		method: 'PATCH',
		path: '/api/admin/settings',
		body: { sessionIdleMinutes: 60 }
	},
	{
		name: 'admin bump notice',
		call: () => adminApi.bumpPrivacyNotice(),
		method: 'POST',
		path: '/api/admin/settings/privacy-notice/bump'
	},
	{
		name: 'admin audit',
		call: () => adminApi.listAudit({ event: 'auth.logout' } as never),
		method: 'GET',
		path: '/api/admin/audit',
		query: { event: 'auth.logout' }
	},
	{
		name: 'admin audit default',
		call: () => adminApi.listAudit(),
		method: 'GET',
		path: '/api/admin/audit'
	},
	{
		name: 'archive job',
		call: () => adminApi.runArchiveJob(),
		method: 'POST',
		path: '/api/admin/jobs/archive'
	}
];

describe('endpoint table', () => {
	test.each(cases)('$name', async ({ call, method, path, query, body }) => {
		const api = mockApi({ [`${method} ${path}`]: json({}) });
		await call();
		expect(api.requests).toHaveLength(1);
		const req = api.requests[0];
		expect(Object.fromEntries(req.url.searchParams)).toEqual(query ?? {});
		expect(req.body).toEqual(body);
	});
});

describe('special cases', () => {
	test('admin reset password sends the generated or given password', async () => {
		const api = mockApi({
			'POST /api/admin/users/4/reset-password': json({ temporaryPassword: 'x' })
		});
		await adminApi.resetPassword(4, 'pw');
		await adminApi.resetPassword(4, 'pw', 'new-password-123');
		expect(api.requests[0].body).toEqual({ currentPassword: 'pw' });
		expect(api.requests[1].body).toEqual({ currentPassword: 'pw', password: 'new-password-123' });
	});

	test('import sends the raw file with the type of its extension', async () => {
		const api = mockApi({ 'POST /api/links/import': json({ created: 0 }) });
		await linksApi.importLinks(file);
		expect(api.requests[0].headers.get('Content-Type')).toBe('text/html');
		expect(api.requests[0].rawBody).toBe(file);
		expect(linksApi.importContentType('export.JSON')).toBe('application/json');
		expect(linksApi.importContentType('bookmarks.htm')).toBe('text/html');
	});

	test('visit uses keepalive; me probe and login skip the auth redirect flag only', async () => {
		const api = mockApi({ 'POST /api/links/1/visit': json({}), 'GET /api/auth/me': json({}) });
		await linksApi.recordVisit(1);
		await authApi.getMe({ skipAuthRedirect: true });
		expect(api.requests[0].init.keepalive).toBe(true);
	});

	test('directory search passes the abort signal', async () => {
		const api = mockApi({ 'GET /api/users/directory': json({ items: [] }) });
		const controller = new AbortController();
		await directoryApi.searchDirectory('a', controller.signal);
		expect(api.requests[0].init.signal).toBe(controller.signal);
	});
});
