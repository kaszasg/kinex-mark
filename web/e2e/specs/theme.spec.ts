// Theme (plan 9.4.2 theme.spec.ts): dark by default, light after the toggle and after a reload;
// signed in, the choice is saved to the account and a fresh browser profile follows it.
import { test, expect } from '../support/fixtures.ts';
import { gotoLinks, msg, withApi } from './lib/app.ts';

const DARK = /(^|\s)dark(\s|$)/;
const LIGHT = /(^|\s)light(\s|$)/;

test('the sign-in page is dark by default and keeps the light theme', async ({ page }) => {
	await page.goto('/login');
	const html = page.locator('html');
	await expect(html).toHaveClass(DARK);
	await page.getByRole('button', { name: msg('nav.theme.toggle'), exact: true }).click();
	await expect(html).toHaveClass(LIGHT);
	await expect(html).not.toHaveClass(DARK);
	await page.reload();
	await expect(html).toHaveClass(LIGHT);
	await page.getByRole('button', { name: msg('nav.theme.toggle'), exact: true }).click();
	await expect(html).toHaveClass(DARK);
});

test('signed in, the theme is saved to the account', async ({ createUser, openPage }) => {
	const user = await createUser('customer');
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	const html = page.locator('html');
	await expect(html).toHaveClass(DARK);
	const saved = page.waitForResponse(
		(response) =>
			response.request().method() === 'PATCH' &&
			new URL(response.url()).pathname === '/api/me/preferences' &&
			response.status() === 200
	);
	await page.getByRole('button', { name: msg('nav.theme.toggle'), exact: true }).click();
	await expect(html).toHaveClass(LIGHT);
	await saved;
	await page.reload();
	await expect(html).toHaveClass(LIGHT);

	const stored = await withApi(user, async (api) => (await api.get('/api/me/preferences')).json());
	expect(stored).toMatchObject({ theme: 'light' });
	const fresh = await openPage(user.storageState);
	await gotoLinks(fresh);
	await expect(fresh.locator('html')).toHaveClass(LIGHT);
});
