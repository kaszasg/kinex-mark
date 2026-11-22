// User administration (plan 9.4.2 admin-users.spec.ts) on target accounts made by createUser:
// create, edit with a role change, password reset with step-up, deactivate and activate, 2FA
// reset, unlock, delete, the disabled actions of the own row, and the last-admin guard.
import type { APIRequestContext, Page } from '@playwright/test';
import { test, expect, type CreatedUser } from '../support/fixtures.ts';
import { expectStatus, newApiContext } from '../support/api.ts';
import { ADMIN_PASSWORD, ADMIN_USERNAME, API_HEADERS } from '../support/constants.mjs';
import { totpAt } from '../support/totp.ts';
import { msg, toast, withApi } from './lib/app.ts';

async function openUsers(page: Page, username: string): Promise<void> {
	await page.goto(`/admin/users?q=${encodeURIComponent(username)}&status=all`);
	await expect(
		page.getByRole('heading', { level: 1, name: msg('admin.users.pageTitle') })
	).toBeVisible();
	await expect(userRow(page, username)).toBeVisible();
}

function userRow(page: Page, username: string) {
	return page
		.getByRole('row')
		.filter({ has: page.getByRole('cell', { name: username, exact: true }) });
}

async function userAction(page: Page, username: string, actionKey: string): Promise<void> {
	await userRow(page, username)
		.getByRole('button', { name: msg('admin.users.actions.menu', { username }), exact: true })
		.click();
	await page.getByRole('menuitem', { name: msg(actionKey), exact: true }).click();
}

/** Fills the step-up password of the open confirmation and confirms. */
async function confirmWithPassword(page: Page, confirmLabel: string, password = ADMIN_PASSWORD) {
	const dialog = page.getByRole('alertdialog');
	await dialog.getByLabel(msg('common.stepUp.label'), { exact: true }).fill(password);
	await dialog.getByRole('button', { name: confirmLabel, exact: true }).click();
	return dialog;
}

async function login(username: string, password: string): Promise<number> {
	const api = await newApiContext();
	try {
		const response = await api.post('/api/auth/login', {
			data: { username, password },
			headers: { ...API_HEADERS }
		});
		return response.status();
	} finally {
		await api.dispose();
	}
}

async function meStatus(user: CreatedUser): Promise<number> {
	return withApi(user, async (api: APIRequestContext) => (await api.get('/api/auth/me')).status());
}

test('create a user and change the role with the admin password', async ({ asAdmin }) => {
	const page = asAdmin;
	const username = `e2e-adm-${Date.now().toString(36)}`;
	await page.goto('/admin/users');
	await page.getByRole('button', { name: msg('admin.users.add'), exact: true }).click();
	const dialog = page.getByRole('dialog', { name: msg('admin.userDialog.createTitle') });
	await dialog.getByLabel(msg('admin.userDialog.fields.username'), { exact: true }).fill(username);
	await dialog
		.getByLabel(msg('admin.userDialog.fields.displayName'), { exact: true })
		.fill('Created In UI');
	await dialog.getByLabel(msg('admin.userDialog.fields.role'), { exact: true }).click();
	await page.getByRole('option', { name: msg('roles.developer'), exact: true }).click();
	await dialog.getByRole('button', { name: msg('admin.userDialog.create'), exact: true }).click();
	await expect(toast(page, msg('admin.users.toast.created', { username }))).toBeVisible();
	const temp = page.getByRole('dialog', { name: msg('admin.tempPassword.title') });
	await expect(temp.getByTestId('temporary-password')).toHaveText(/^\S{16}$/);
	await temp.getByRole('button', { name: msg('admin.tempPassword.done'), exact: true }).click();

	await openUsers(page, username);
	await expect(userRow(page, username)).toContainText(msg('roles.developer'));
	await userAction(page, username, 'admin.users.actions.edit');
	const edit = page.getByRole('dialog', { name: msg('admin.userDialog.editTitle', { username }) });
	await edit.getByLabel(msg('admin.userDialog.fields.role'), { exact: true }).click();
	await page.getByRole('option', { name: msg('roles.customer'), exact: true }).click();
	const save = edit.getByRole('button', { name: msg('admin.userDialog.save'), exact: true });
	await expect(save).toBeDisabled();
	await edit.getByLabel(msg('common.stepUp.label'), { exact: true }).fill(ADMIN_PASSWORD);
	await save.click();
	await expect(toast(page, msg('admin.users.toast.updated', { username }))).toBeVisible();
	await expect(userRow(page, username)).toContainText(msg('roles.customer'));
});

test('a password reset needs the right admin password and ends the sessions', async ({
	asAdmin,
	createUser,
	expectHttp
}) => {
	const page = asAdmin;
	const target = await createUser('developer');
	await openUsers(page, target.username);

	expectHttp(400, /^\/api\/admin\/users\/\d+\/reset-password$/, 'POST');
	await userAction(page, target.username, 'admin.users.actions.resetPassword');
	const label = msg('admin.users.confirm.resetPassword.action');
	const dialog = await confirmWithPassword(page, label, 'Not-The-Admin-Password-1');
	await expect(dialog.getByText(msg('errors.current_password_invalid.title'))).toBeVisible();
	expect(await meStatus(target)).toBe(200);

	await dialog.getByLabel(msg('common.stepUp.label'), { exact: true }).fill(ADMIN_PASSWORD);
	await dialog.getByRole('button', { name: label, exact: true }).click();
	await expect(
		toast(page, msg('admin.users.toast.passwordReset', { username: target.username }))
	).toBeVisible();
	const temp = page.getByRole('dialog', { name: msg('admin.tempPassword.title') });
	const password = (await temp.getByTestId('temporary-password').innerText()).trim();
	await temp.getByRole('button', { name: msg('admin.tempPassword.done'), exact: true }).click();
	expect(await meStatus(target)).toBe(401);
	expect(await login(target.username, password)).toBe(200);
});

test('deactivate and activate an account', async ({ asAdmin, createUser }) => {
	const page = asAdmin;
	const target = await createUser('customer');
	await openUsers(page, target.username);

	await userAction(page, target.username, 'admin.users.actions.deactivate');
	await confirmWithPassword(page, msg('admin.users.confirm.deactivate.action'));
	await expect(
		toast(page, msg('admin.users.toast.deactivated', { username: target.username }))
	).toBeVisible();
	await expect(userRow(page, target.username)).toContainText(msg('status.inactive'));
	expect(await meStatus(target)).toBe(401);
	expect(await login(target.username, target.password)).toBe(403);

	await userAction(page, target.username, 'admin.users.actions.activate');
	await confirmWithPassword(page, msg('admin.users.confirm.activate.action'));
	await expect(
		toast(page, msg('admin.users.toast.activated', { username: target.username }))
	).toBeVisible();
	await expect(userRow(page, target.username)).toContainText(msg('status.active'));
	expect(await login(target.username, target.password)).toBe(200);
});

test('a 2FA reset turns two-factor sign-in off and ends the sessions', async ({
	asAdmin,
	createUser
}) => {
	const page = asAdmin;
	const target = await createUser('developer');
	await withApi(target, async (api) => {
		const setup = await api.post('/api/me/2fa/setup', { data: { password: target.password } });
		await expectStatus(setup, 200, '2fa setup');
		const { secret } = (await setup.json()) as { secret: string };
		const enable = await api.post('/api/me/2fa/enable', {
			data: { code: totpAt(secret, Date.now()).code }
		});
		await expectStatus(enable, 200, '2fa enable');
	});
	await openUsers(page, target.username);
	await expect(userRow(page, target.username)).toContainText(msg('admin.users.table.twoFactorOn'));

	await userAction(page, target.username, 'admin.users.actions.resetTwoFactor');
	await confirmWithPassword(page, msg('admin.users.confirm.resetTwoFactor.action'));
	await expect(
		toast(page, msg('admin.users.toast.twoFactorReset', { username: target.username }))
	).toBeVisible();
	await expect(userRow(page, target.username)).not.toContainText(
		msg('admin.users.table.twoFactorOn')
	);
	expect(await meStatus(target)).toBe(401);
	// The password alone signs in again.
	expect(await login(target.username, target.password)).toBe(200);
});

test('unlock a locked account and delete an account', async ({ asAdmin, createUser }) => {
	const page = asAdmin;
	const target = await createUser('customer');
	for (let i = 0; i < 5; i++) {
		expect(await login(target.username, `Wrong-Passphrase-${i}xx`)).toBe(401);
	}
	expect(await login(target.username, target.password)).toBe(429);
	await openUsers(page, target.username);
	await expect(userRow(page, target.username)).toContainText(msg('status.locked'));

	await userAction(page, target.username, 'admin.users.actions.unlock');
	await expect(
		toast(page, msg('admin.users.toast.unlocked', { username: target.username }))
	).toBeVisible();
	await expect(userRow(page, target.username)).toContainText(msg('status.active'));
	expect(await login(target.username, target.password)).toBe(200);

	await userAction(page, target.username, 'admin.users.actions.delete');
	await confirmWithPassword(page, msg('admin.users.confirm.delete.action'));
	await expect(
		toast(page, msg('admin.users.toast.deleted', { username: target.username }))
	).toBeVisible();
	await expect(userRow(page, target.username)).toHaveCount(0);
	expect(await login(target.username, target.password)).toBe(401);
});

test('the own row cannot be deactivated or deleted', async ({ asAdmin }) => {
	const page = asAdmin;
	await openUsers(page, ADMIN_USERNAME);
	await userRow(page, ADMIN_USERNAME)
		.getByRole('button', { name: msg('admin.users.actions.menu', { username: ADMIN_USERNAME }) })
		.click();
	const menu = page.getByRole('menu');
	for (const key of ['admin.users.actions.deactivate', 'admin.users.actions.delete']) {
		await expect(menu.getByRole('menuitem', { name: msg(key), exact: true })).toHaveAttribute(
			'aria-disabled',
			'true'
		);
	}
	await expect(menu.getByText(msg('admin.users.actions.selfHint'))).toBeVisible();
	await page.keyboard.press('Escape');

	await userAction(page, ADMIN_USERNAME, 'admin.users.actions.edit');
	const edit = page.getByRole('dialog');
	await expect(
		edit.getByLabel(msg('admin.userDialog.fields.role'), { exact: true })
	).toBeDisabled();
	await expect(
		edit.getByRole('switch', { name: msg('admin.userDialog.fields.active') })
	).toBeDisabled();
});

test('the last active admin cannot delete the own account', async ({
	asAdmin,
	adminApi,
	expectHttp
}) => {
	// Other admins made by earlier tests are deactivated, so the seeded admin is the last one.
	const me = (await (await adminApi.get('/api/auth/me')).json()) as { id: number };
	const list = await adminApi.get('/api/admin/users?role=admin&status=active&limit=200');
	await expectStatus(list, 200, 'list admins');
	const admins = ((await list.json()) as { items: { id: number }[] }).items;
	for (const admin of admins.filter((item) => item.id !== me.id)) {
		const patch = await adminApi.patch(`/api/admin/users/${admin.id}`, {
			data: { active: false, currentPassword: ADMIN_PASSWORD }
		});
		await expectStatus(patch, 200, `deactivate admin ${admin.id}`);
	}
	const check = await adminApi.get('/api/admin/users?role=admin&status=active&limit=200');
	expect(((await check.json()) as { total: number }).total).toBe(1);

	const page = asAdmin;
	expectHttp(409, '/api/me/delete', 'POST');
	await page.goto('/settings/privacy');
	await page.getByRole('button', { name: msg('security.deleteAccount.open'), exact: true }).click();
	const dialog = page.getByRole('alertdialog');
	await dialog
		.getByLabel(msg('security.deleteAccount.password'), { exact: true })
		.fill(ADMIN_PASSWORD);
	await dialog
		.getByLabel(msg('security.deleteAccount.confirm', { username: ADMIN_USERNAME }), {
			exact: true
		})
		.fill(ADMIN_USERNAME);
	await dialog
		.getByRole('button', { name: msg('security.deleteAccount.submit'), exact: true })
		.click();
	await expect(dialog.getByText(msg('errors.last_admin.title'))).toBeVisible();
	expect((await adminApi.get('/api/auth/me')).status()).toBe(200);
});
