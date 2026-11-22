// The /admin/users screen: states, URL filters, paging and every row action (plan 8.7.7).
import { screen, within } from '@testing-library/svelte';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import { ApiError } from '$lib/api/errors.js';
import type { AdminUser } from '$lib/api/types.js';
import { errorMessage, errorTitle, t } from '$lib/i18n/index.js';
import { setTestPage } from '$lib/test/app-state.svelte.js';
import { makeMe } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent, type MockRequest } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import UsersScreen from './UsersScreen.svelte';
import { findFloating, getFloating, makeAdminUser, page, user } from './test-helpers.js';

vi.mock('svelte-sonner', async (importOriginal) => ({
	...(await importOriginal<typeof import('svelte-sonner')>()),
	toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn() })
}));

const me = makeMe('admin', { id: 1, username: 'kanna' });
const self = makeAdminUser({ id: 1, username: 'kanna', role: 'admin' });
const future = new Date(Date.now() + 15 * 60_000).toISOString();

function listRoute(users: AdminUser[], total = users.length) {
	return (req: MockRequest) =>
		json(page(users, total, Number(req.url.searchParams.get('offset') ?? 0)));
}

function render(path = '/admin/users') {
	setTestPage(path);
	return renderWithApp(UsersScreen, {}, { me });
}

async function openMenu(u: ReturnType<typeof user>, username: string, item: string) {
	await u.click(screen.getByRole('button', { name: t('admin.users.actions.menu', { username }) }));
	await u.click(await findFloating('menuitem', item));
}

beforeEach(() => {
	vi.mocked(goto).mockReset();
	vi.mocked(goto).mockImplementation(async (url) => setTestPage(String(url)));
	vi.mocked(toast.success).mockClear();
	vi.mocked(toast.error).mockClear();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('UsersScreen states', () => {
	test('loading, then the table with the total in the subtitle', async () => {
		let release: () => void = () => undefined;
		const gate = new Promise<void>((resolve) => (release = resolve));
		mockApi({
			'GET /api/admin/users': async () => {
				await gate;
				return json(page([self, makeAdminUser({ username: 'peter' })], 2));
			}
		});
		render();
		expect(screen.getByRole('heading', { name: t('admin.users.pageTitle') })).toBeInTheDocument();
		expect(screen.getByRole('status')).toHaveTextContent(t('common.states.loading'));
		expect(screen.getByRole('button', { name: t('admin.users.add') })).toBeInTheDocument();
		release();
		expect(await screen.findByText('peter')).toBeInTheDocument();
		expect(screen.getByText(t('admin.users.summary', { count: 2 }))).toBeInTheDocument();
		expect(screen.getByText('1–2 / 2')).toBeInTheDocument();
	});

	test('an error replaces the content and retry loads again', async () => {
		let fail = true;
		mockApi({
			'GET /api/admin/users': () =>
				fail ? apiError(500, 'internal_error') : json(page([makeAdminUser({ username: 'back' })]))
		});
		render();
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(errorTitle(new ApiError(500, 'internal_error')));
		fail = false;
		const u = user();
		await u.click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		expect(await screen.findByText('back')).toBeInTheDocument();
	});

	test('no match: the empty state offers to clear the filters', async () => {
		mockApi({ 'GET /api/admin/users': listRoute([]) });
		render('/admin/users?q=nobody&role=customer');
		expect(await screen.findByText(t('admin.users.empty.title'))).toBeInTheDocument();
		const u = user();
		const clear = screen.getAllByRole('button', { name: t('admin.users.filters.clear') });
		expect(clear).toHaveLength(2);
		await u.click(clear[1]);
		expect(goto).toHaveBeenLastCalledWith('/admin/users', { replace: false, reset: false });
	});

	test('the default view has no clear button in the empty state', async () => {
		mockApi({ 'GET /api/admin/users': listRoute([]) });
		render();
		expect(await screen.findByText(t('admin.users.empty.title'))).toBeInTheDocument();
		expect(screen.queryByRole('button', { name: t('admin.users.filters.clear') })).toBeNull();
	});
});

describe('UsersScreen filters and paging', () => {
	test('the URL drives the request; filter changes go to the URL', async () => {
		const api = mockApi({ 'GET /api/admin/users': listRoute([self], 120) });
		render('/admin/users?q=kan&role=admin&status=all&page=2');
		expect(await screen.findByText('kanna')).toBeInTheDocument();
		const query = api.calls('GET /api/admin/users')[0].url.searchParams;
		expect(Object.fromEntries(query)).toEqual({
			q: 'kan',
			role: 'admin',
			status: 'all',
			limit: '50',
			offset: '50'
		});
		expect(screen.getByLabelText(t('admin.users.filters.search'))).toHaveValue('kan');
		const u = user();
		await u.click(screen.getByLabelText(t('admin.users.filters.role')));
		await u.click(await findFloating('option', t('admin.users.filters.allRoles')));
		expect(goto).toHaveBeenLastCalledWith('/admin/users?q=kan&status=all', {
			replace: false,
			reset: false
		});
		await u.click(screen.getByLabelText(t('admin.users.filters.status')));
		await u.click(await findFloating('option', t('admin.users.filters.statusInactive')));
		expect(goto).toHaveBeenLastCalledWith('/admin/users?q=kan&status=inactive', {
			replace: false,
			reset: false
		});
		await vi.waitFor(() => expect(api.calls('GET /api/admin/users')).toHaveLength(3));
	});

	test('typing searches after a pause and replaces the history entry', async () => {
		const api = mockApi({ 'GET /api/admin/users': listRoute([self]) });
		render();
		await screen.findByText('kanna');
		const u = user();
		await u.type(screen.getByLabelText(t('admin.users.filters.search')), 'pet');
		await vi.waitFor(() =>
			expect(goto).toHaveBeenLastCalledWith('/admin/users?q=pet', { replace: true, reset: false })
		);
		expect(goto).toHaveBeenCalledTimes(1);
		await vi.waitFor(() =>
			expect(api.calls('GET /api/admin/users').at(-1)?.url.searchParams.get('q')).toBe('pet')
		);
		await u.type(screen.getByLabelText(t('admin.users.filters.search')), 'er{Enter}');
		expect(goto).toHaveBeenLastCalledWith('/admin/users?q=peter', { replace: true, reset: false });
	});

	test('the pager moves to another page', async () => {
		mockApi({ 'GET /api/admin/users': listRoute([self], 75) });
		render();
		await screen.findByText('kanna');
		const u = user();
		await u.click(screen.getByRole('button', { name: t('admin.pager.next') }));
		expect(goto).toHaveBeenLastCalledWith('/admin/users?page=2', { replace: false, reset: false });
	});

	test('a stale answer does not overwrite a newer one', async () => {
		let first = true;
		let releaseFirst: () => void = () => undefined;
		mockApi({
			'GET /api/admin/users': async () => {
				if (first) {
					first = false;
					await new Promise<void>((resolve) => (releaseFirst = resolve));
					return json(page([makeAdminUser({ username: 'stale' })]));
				}
				return json(page([makeAdminUser({ username: 'fresh' })]));
			}
		});
		render();
		setTestPage('/admin/users?role=customer');
		expect(await screen.findByText('fresh')).toBeInTheDocument();
		releaseFirst();
		await new Promise((resolve) => setTimeout(resolve, 20));
		expect(screen.queryByText('stale')).toBeNull();
	});
});

describe('UsersScreen actions', () => {
	test('creating a user shows the temporary password once and reloads', async () => {
		const created = makeAdminUser({ username: 'fresh.one' });
		const api = mockApi({
			'GET /api/admin/users': listRoute([self]),
			'POST /api/admin/users': json({ user: created, temporaryPassword: 'Gen-Pass-0123456' }, 201)
		});
		render();
		await screen.findByText('kanna');
		const u = user();
		await u.click(screen.getByRole('button', { name: t('admin.users.add') }));
		await u.type(screen.getByLabelText(t('admin.userDialog.fields.username')), 'fresh.one');
		await u.click(screen.getByRole('button', { name: t('admin.userDialog.create') }));
		const dialog = await screen.findByRole('dialog', { name: t('admin.tempPassword.title') });
		expect(within(dialog).getByTestId('temporary-password')).toHaveTextContent('Gen-Pass-0123456');
		expect(within(dialog).getByText(t('admin.tempPassword.warningTitle'))).toBeInTheDocument();
		expect(
			within(dialog).getByRole('button', { name: t('admin.tempPassword.copy') })
		).toBeInTheDocument();
		expect(toast.success).toHaveBeenCalledWith(
			t('admin.users.toast.created', { username: 'fresh.one' })
		);
		await vi.waitFor(() => expect(api.calls('GET /api/admin/users')).toHaveLength(2));
		await u.click(within(dialog).getByRole('button', { name: t('admin.tempPassword.done') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});

	test('a typed password gives no temporary password dialog', async () => {
		const created = makeAdminUser({ username: 'typed' });
		mockApi({
			'GET /api/admin/users': listRoute([self]),
			'POST /api/admin/users': json({ user: created, temporaryPassword: null }, 201)
		});
		render();
		await screen.findByText('kanna');
		const u = user();
		await u.click(screen.getByRole('button', { name: t('admin.users.add') }));
		await u.type(screen.getByLabelText(t('admin.userDialog.fields.username')), 'typed');
		await u.click(screen.getByRole('button', { name: t('admin.userDialog.create') }));
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalled());
		expect(screen.queryByRole('dialog', { name: t('admin.tempPassword.title') })).toBeNull();
	});

	test('editing a user saves and confirms with a toast', async () => {
		const peter = makeAdminUser({ id: 7, username: 'peter', displayName: 'Peter' });
		mockApi({
			'GET /api/admin/users': listRoute([self, peter]),
			'PATCH /api/admin/users/7': json({ ...peter, displayName: 'Peter S' })
		});
		render();
		await screen.findByText('peter');
		const u = user();
		await openMenu(u, 'peter', t('admin.users.actions.edit'));
		await u.type(screen.getByLabelText(t('admin.userDialog.fields.displayName')), ' S');
		await u.click(screen.getByRole('button', { name: t('admin.userDialog.save') }));
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(
				t('admin.users.toast.updated', { username: 'peter' })
			)
		);
	});

	test('password reset needs the own password and shows the new temporary password', async () => {
		const peter = makeAdminUser({ id: 7, username: 'peter' });
		const api = mockApi({
			'GET /api/admin/users': listRoute([self, peter]),
			'POST /api/admin/users/7/reset-password': (req) =>
				(req.body as { currentPassword: string }).currentPassword === 'mine'
					? json({ temporaryPassword: 'Reset-Pass-987654' })
					: apiError(400, 'current_password_invalid', { currentPassword: 'incorrect' })
		});
		render();
		await screen.findByText('peter');
		const u = user();
		await openMenu(u, 'peter', t('admin.users.actions.resetPassword'));
		const dialog = await screen.findByRole('alertdialog', {
			name: t('admin.users.confirm.resetPassword.title', { username: 'peter' })
		});
		expect(within(dialog).getByText(t('admin.users.confirm.resetPassword.text'))).toBeVisible();
		const confirm = within(dialog).getByRole('button', {
			name: t('admin.users.confirm.resetPassword.action')
		});
		expect(confirm).toBeDisabled();
		const password = within(dialog).getByLabelText(t('common.stepUp.label'));
		await u.type(password, 'wrong');
		await u.click(confirm);
		expect(
			await within(dialog).findByText(errorMessage(new ApiError(400, 'current_password_invalid')))
		).toBeInTheDocument();
		await u.clear(password);
		await u.type(password, 'mine');
		await u.click(confirm);
		const temp = await screen.findByRole('dialog', { name: t('admin.tempPassword.title') });
		expect(within(temp).getByTestId('temporary-password')).toHaveTextContent('Reset-Pass-987654');
		expect(toast.success).toHaveBeenCalledWith(
			t('admin.users.toast.passwordReset', { username: 'peter' })
		);
		expect(api.calls('POST /api/admin/users/7/reset-password').at(-1)?.body).toEqual({
			currentPassword: 'mine'
		});
	});

	test('delete warns about shared links and sends the own password', async () => {
		const sharer = makeAdminUser({ id: 9, username: 'sharer', sharedLinkCount: 3 });
		const api = mockApi({
			'GET /api/admin/users': listRoute([self, sharer]),
			'DELETE /api/admin/users/9': noContent()
		});
		render();
		await screen.findByText('sharer');
		const u = user();
		await openMenu(u, 'sharer', t('admin.users.actions.delete'));
		const dialog = await screen.findByRole('alertdialog', {
			name: t('admin.users.confirm.delete.title', { username: 'sharer' })
		});
		expect(
			within(dialog).getByText(t('admin.users.confirm.delete.sharedTitle', { count: 3 }))
		).toBeInTheDocument();
		await u.type(within(dialog).getByLabelText(t('common.stepUp.label')), 'mine');
		await u.click(
			within(dialog).getByRole('button', { name: t('admin.users.confirm.delete.action') })
		);
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(
				t('admin.users.toast.deleted', { username: 'sharer' })
			)
		);
		expect(api.calls('DELETE /api/admin/users/9')[0].body).toEqual({ currentPassword: 'mine' });
	});

	test('delete without shared links has no warning; last_admin is an alert', async () => {
		const other = makeAdminUser({ id: 9, username: 'other', role: 'admin' });
		mockApi({
			'GET /api/admin/users': listRoute([self, other]),
			'DELETE /api/admin/users/9': apiError(409, 'last_admin')
		});
		render();
		await screen.findByText('other');
		const u = user();
		await openMenu(u, 'other', t('admin.users.actions.delete'));
		const dialog = await screen.findByRole('alertdialog');
		expect(within(dialog).queryByText(/0/)).toBeNull();
		await u.type(within(dialog).getByLabelText(t('common.stepUp.label')), 'mine');
		await u.click(
			within(dialog).getByRole('button', { name: t('admin.users.confirm.delete.action') })
		);
		expect(await within(dialog).findByRole('alert')).toHaveTextContent(
			errorTitle(new ApiError(409, 'last_admin'))
		);
	});

	test.each([
		['deactivate', { active: false }, 'deactivated', {}],
		['activate', { active: true }, 'activated', { active: false }]
	] as const)('%s goes through the step-up PATCH', async (action, body, toastKey, partial) => {
		const peter = makeAdminUser({ id: 7, username: 'peter', ...partial });
		const api = mockApi({
			'GET /api/admin/users': listRoute([self, peter]),
			'PATCH /api/admin/users/7': json({ ...peter, ...body })
		});
		render();
		await screen.findByText('peter');
		const u = user();
		await openMenu(u, 'peter', t(`admin.users.actions.${action}`));
		const dialog = await screen.findByRole('alertdialog', {
			name: t(`admin.users.confirm.${action}.title`, { username: 'peter' })
		});
		await u.type(within(dialog).getByLabelText(t('common.stepUp.label')), 'mine');
		await u.click(
			within(dialog).getByRole('button', { name: t(`admin.users.confirm.${action}.action`) })
		);
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(
				t(`admin.users.toast.${toastKey}`, { username: 'peter' })
			)
		);
		expect(api.calls('PATCH /api/admin/users/7')[0].body).toEqual({
			...body,
			currentPassword: 'mine'
		});
	});

	test('2FA reset goes through step-up', async () => {
		const peter = makeAdminUser({ id: 7, username: 'peter', totpEnabled: true });
		const api = mockApi({
			'GET /api/admin/users': listRoute([self, peter]),
			'POST /api/admin/users/7/reset-2fa': noContent()
		});
		render();
		await screen.findByText('peter');
		const u = user();
		await openMenu(u, 'peter', t('admin.users.actions.resetTwoFactor'));
		const dialog = await screen.findByRole('alertdialog');
		expect(within(dialog).getByText(t('admin.users.confirm.resetTwoFactor.text'))).toBeVisible();
		await u.type(within(dialog).getByLabelText(t('common.stepUp.label')), 'mine');
		await u.click(
			within(dialog).getByRole('button', { name: t('admin.users.confirm.resetTwoFactor.action') })
		);
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(
				t('admin.users.toast.twoFactorReset', { username: 'peter' })
			)
		);
		expect(api.calls('POST /api/admin/users/7/reset-2fa')[0].body).toEqual({
			currentPassword: 'mine'
		});
	});

	test('unlock and revoking sessions run at once without a password', async () => {
		const locked = makeAdminUser({ id: 7, username: 'locked', lockedUntil: future });
		const api = mockApi({
			'GET /api/admin/users': listRoute([self, locked]),
			'POST /api/admin/users/7/unlock': noContent(),
			'DELETE /api/admin/users/7/sessions': json({ revoked: 2 })
		});
		render();
		await screen.findByText('locked');
		const u = user();
		await openMenu(u, 'locked', t('admin.users.actions.unlock'));
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(
				t('admin.users.toast.unlocked', { username: 'locked' })
			)
		);
		await openMenu(u, 'locked', t('admin.users.actions.revokeSessions'));
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(
				t('admin.users.toast.sessionsRevoked', { count: 2 })
			)
		);
		expect(api.calls('POST /api/admin/users/7/unlock')).toHaveLength(1);
		expect(screen.queryByRole('alertdialog')).toBeNull();
	});

	test('a failed direct action is a toast', async () => {
		mockApi({
			'GET /api/admin/users': listRoute([self, makeAdminUser({ id: 7, username: 'gone' })]),
			'DELETE /api/admin/users/7/sessions': apiError(404, 'not_found')
		});
		render();
		await screen.findByText('gone');
		const u = user();
		await openMenu(u, 'gone', t('admin.users.actions.revokeSessions'));
		await vi.waitFor(() =>
			expect(toast.error).toHaveBeenCalledWith(errorMessage(new ApiError(404, 'not_found')))
		);
	});

	test('a direct action that fails without an API error reports a network error', async () => {
		const target = makeAdminUser({ id: 7, username: 'net', lockedUntil: future });
		mockApi({ 'GET /api/admin/users': listRoute([self, target]) });
		render();
		await screen.findByText('net');
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw new TypeError('offline');
			})
		);
		const u = user();
		await openMenu(u, 'net', t('admin.users.actions.unlock'));
		await vi.waitFor(() =>
			expect(toast.error).toHaveBeenCalledWith(errorMessage(new ApiError(0, 'network_error')))
		);
	});

	test('the own row is marked as such in the menu', async () => {
		mockApi({ 'GET /api/admin/users': listRoute([self]) });
		render();
		await screen.findByText('kanna');
		const u = user();
		await u.click(
			screen.getByRole('button', { name: t('admin.users.actions.menu', { username: 'kanna' }) })
		);
		await findFloating('menuitem', t('admin.users.actions.edit'));
		expect(getFloating('menuitem', t('admin.users.actions.delete'))).toHaveAttribute(
			'aria-disabled',
			'true'
		);
	});
});
