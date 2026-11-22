// Smoke test of the real binary with the embedded UI (plan 9.4.2): the sign-in page loads in
// the dark theme and in Hungarian, and the console stays clean (consoleGuard).
import { test, expect } from '../support/fixtures.ts';

test('the sign-in page loads with the dark theme, Hungarian language and a clean console', async ({
	page
}) => {
	const me = page.waitForResponse(
		(response) => new URL(response.url()).pathname === '/api/auth/me'
	);
	await page.goto('/');

	expect((await me).status()).toBe(401);
	await expect(page).toHaveURL(/\/login(\?|$)/);
	await expect(page).toHaveTitle(/Kinex Mark/);
	const html = page.locator('html');
	await expect(html).toHaveClass(/(^|\s)dark(\s|$)/);
	await expect(html).toHaveAttribute('lang', 'hu');
});

test('the API rejects anonymous requests to the current user', async ({ request }) => {
	const response = await request.get('/api/auth/me');
	expect(response.status()).toBe(401);
	expect(await response.json()).toMatchObject({ code: 'unauthenticated' });
});
