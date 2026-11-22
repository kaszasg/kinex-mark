// Settings (plan 9.4.2 settings.spec.ts), each test with its own account: profile, display
// preferences with a custom archive period, sessions and revoking another device, password change.
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import { newApiContext, signIn } from '../support/api.ts';
import { API_HEADERS, BASE_URL } from '../support/constants.mjs';
import { escapeRegExp, msg, msgIn, toast, withApi } from './lib/app.ts';

/** Waits for the debounced preference PATCH that the next action triggers. */
function preferencesSaved(page: Page) {
	return page.waitForResponse(
		(response) =>
			response.request().method() === 'PATCH' &&
			new URL(response.url()).pathname === '/api/me/preferences' &&
			response.status() === 200
	);
}

test('the display name changes in the profile', async ({ createUser, openPage }) => {
	const user = await createUser('customer', { displayName: 'Before Rename' });
	const page = await openPage(user.storageState);
	await page.goto('/settings');
	await expect(page).toHaveURL(/\/settings\/profile$/);
	await expect(page.getByText(user.username, { exact: true })).toBeVisible();
	const field = page.getByLabel(msg('settings.profile.displayName'), { exact: true });
	await expect(field).toHaveValue('Before Rename');
	await field.fill('After Rename');
	await page.getByRole('button', { name: msg('settings.profile.save'), exact: true }).click();
	await expect(toast(page, msg('settings.profile.saved'))).toBeVisible();
	await expect(
		page.getByRole('button', {
			name: msg('nav.userMenu', { name: 'After Rename', role: msg('roles.customer') })
		})
	).toBeVisible();
	await page.reload();
	await expect(field).toHaveValue('After Rename');
});

test('display preferences are saved, including a custom archive period', async ({
	createUser,
	openPage
}) => {
	const user = await createUser('developer');
	const page = await openPage(user.storageState);
	await page.goto('/settings/display');
	const html = page.locator('html');
	await expect(html).toHaveClass(/(^|\s)dark(\s|$)/);

	let saved = preferencesSaved(page);
	await page.getByRole('radio', { name: msg('settings.display.theme.light'), exact: true }).click();
	await expect(html).toHaveClass(/(^|\s)light(\s|$)/);
	await saved;

	saved = preferencesSaved(page);
	await page
		.getByRole('radio', { name: msg('settings.display.viewMode.tiles'), exact: true })
		.click();
	await saved;

	saved = preferencesSaved(page);
	await page
		.getByRole('switch', { name: msg('settings.display.fields.description'), exact: true })
		.click();
	await saved;
	await expect(page.getByTestId('preview-label')).toHaveText(msg('settings.display.preview.title'));

	await page.getByLabel(msg('settings.display.archive.label'), { exact: true }).click();
	await page
		.getByRole('option', { name: msg('settings.display.archive.custom'), exact: true })
		.click();
	const days = page.getByLabel(msg('settings.display.archive.customLabel'), { exact: true });
	await expect(days).toBeFocused();
	await days.fill('45');
	saved = preferencesSaved(page);
	await days.press('Enter');
	await saved;
	await expect(
		page.getByText(msg('settings.display.archive.current', { count: 45 }))
	).toBeVisible();

	await page.reload();
	await expect(html).toHaveClass(/(^|\s)light(\s|$)/);
	await expect(
		page.getByRole('radio', { name: msg('settings.display.viewMode.tiles'), exact: true })
	).toHaveAttribute('aria-checked', 'true');
	await expect(
		page.getByRole('switch', { name: msg('settings.display.fields.description'), exact: true })
	).not.toBeChecked();
	await expect(
		page.getByText(msg('settings.display.archive.current', { count: 45 }))
	).toBeVisible();
	const prefs = await withApi(user, async (api) => (await api.get('/api/me/preferences')).json());
	expect(prefs).toMatchObject({
		theme: 'light',
		viewMode: 'tiles',
		showDescription: false,
		archiveAfterDays: 45,
		effectiveArchiveAfterDays: 45
	});

	// The language select switches the whole interface at once.
	saved = preferencesSaved(page);
	await page
		.getByRole('combobox', { name: msg('settings.display.language.label'), exact: true })
		.click();
	await page.getByRole('option', { name: 'English', exact: true }).click();
	await expect(html).toHaveAttribute('lang', 'en');
	await expect(
		page.getByRole('combobox', {
			name: msgIn('en', 'settings.display.language.label'),
			exact: true
		})
	).toBeVisible();
	await saved;
	const locale = await withApi(user, async (api) => (await api.get('/api/me/preferences')).json());
	expect(locale).toMatchObject({ locale: 'en' });
});

test('another session is listed and can be signed out', async ({ createUser, openPage }) => {
	const user = await createUser('developer');
	const other = await newApiContext();
	try {
		await signIn(other, user.username, user.password);
		const page = await openPage(user.storageState);
		await page.goto('/settings/security');
		const list = page.getByRole('list', { name: msg('security.sessions.listLabel') });
		await expect(list.getByRole('listitem')).toHaveCount(2);
		await expect(list.getByText(msg('security.sessions.current'), { exact: true })).toHaveCount(1);
		const foreign = list
			.getByRole('listitem')
			.filter({ hasNot: page.getByText(msg('security.sessions.current'), { exact: true }) });
		const revokePrefix = msg('security.sessions.revoke', { device: '' });
		await foreign
			.getByRole('button', { name: new RegExp(`^${escapeRegExp(revokePrefix)}`) })
			.click();
		await expect(list.getByRole('listitem')).toHaveCount(1);
		expect((await other.get('/api/auth/me')).status()).toBe(401);
		// The current session stays.
		expect((await page.request.get(`${BASE_URL}/api/auth/me`)).status()).toBe(200);
	} finally {
		await other.dispose();
	}
});

test('a password change signs out the other devices', async ({ createUser, openPage }) => {
	const user = await createUser('customer');
	const other = await newApiContext();
	const next = 'Changed-Passphrase-93Z';
	try {
		await signIn(other, user.username, user.password);
		const page = await openPage(user.storageState);
		await page.goto('/settings/security');
		await page.getByLabel(msg('auth.passwordChange.current'), { exact: true }).fill(user.password);
		await page.getByLabel(msg('auth.passwordChange.new'), { exact: true }).fill(next);
		await page.getByLabel(msg('auth.passwordChange.confirm'), { exact: true }).fill(next);
		await page
			.getByRole('button', { name: msg('auth.passwordChange.submit'), exact: true })
			.click();
		await expect(toast(page, msg('auth.passwordChange.savedOtherSessions'))).toBeVisible();
		expect((await other.get('/api/auth/me')).status()).toBe(401);
	} finally {
		await other.dispose();
	}
	const fresh = await newApiContext();
	try {
		const response = await fresh.post('/api/auth/login', {
			data: { username: user.username, password: next },
			headers: { ...API_HEADERS }
		});
		expect(response.status()).toBe(200);
	} finally {
		await fresh.dispose();
	}
});
