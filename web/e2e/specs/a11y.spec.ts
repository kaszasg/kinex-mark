// Accessibility (plan 9.4.2 a11y.spec.ts): axe on the main pages in the dark and the light theme
// with no serious or critical violation, keyboard-only sign-in and link capture, the page
// shortcuts, and the hero image on the sign-in page.
import AxeBuilder from '@axe-core/playwright';
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import { BASE_URL } from '../support/constants.mjs';
import {
	apiCreateLink,
	apiDeleteLink,
	apiPatchPreferences,
	fixtureUrl,
	gotoLinks,
	msg,
	toast,
	withApi
} from './lib/app.ts';

const SIGNED_IN_PAGES = [
	'/',
	'/tags',
	'/settings/profile',
	'/settings/display',
	'/settings/security',
	'/settings/privacy',
	'/admin/users',
	'/admin/settings',
	'/admin/audit'
];

async function seriousViolations(page: Page): Promise<string[]> {
	const results = await new AxeBuilder({ page }).analyze();
	return results.violations
		.filter((violation) => violation.impact === 'serious' || violation.impact === 'critical')
		.map(
			(violation) =>
				`${violation.id} (${violation.impact}): ${violation.nodes
					.slice(0, 3)
					.map((node) => node.target.join(' '))
					.join(', ')}`
		);
}

async function settle(page: Page): Promise<void> {
	await expect(page.getByRole('heading', { level: 1 }).first()).toBeVisible();
	await expect(page.locator('[aria-busy="true"]')).toHaveCount(0);
	// Let the entry transitions end, so colour contrast is measured on the final colours.
	await page.waitForTimeout(300);
}

for (const theme of ['dark', 'light'] as const) {
	test(`no serious or critical axe violation in the ${theme} theme`, async ({
		page,
		asAdmin,
		adminApi
	}, testInfo) => {
		test.setTimeout(150_000);
		const sample = await apiCreateLink(adminApi, {
			url: fixtureUrl(testInfo.testId),
			description: 'Accessibility sample'
		});
		const found: Record<string, string[]> = {};
		try {
			await page.goto('/login');
			if (theme === 'light') {
				await page.getByRole('button', { name: msg('nav.theme.toggle'), exact: true }).click();
			}
			await expect(page.locator('html')).toHaveClass(new RegExp(`(^|\\s)${theme}(\\s|$)`));
			for (const path of ['/login', '/privacy']) {
				await page.goto(path);
				await settle(page);
				found[`anonymous ${path}`] = await seriousViolations(page);
			}

			await apiPatchPreferences(adminApi, { theme });
			for (const path of SIGNED_IN_PAGES) {
				await asAdmin.goto(path);
				await expect(asAdmin.locator('html')).toHaveClass(new RegExp(`(^|\\s)${theme}(\\s|$)`));
				await settle(asAdmin);
				found[path] = await seriousViolations(asAdmin);
			}
		} finally {
			await apiDeleteLink(adminApi, sample.id);
		}
		const failing = Object.fromEntries(
			Object.entries(found).filter(([, violations]) => violations.length > 0)
		);
		expect(failing).toEqual({});
	});
}

test('sign-in and link capture work with the keyboard alone', async ({
	page,
	createUser
}, testInfo) => {
	const user = await createUser('developer');
	await page.goto('/login');
	const username = page.getByLabel(msg('auth.login.username'), { exact: true });
	// The form is reachable with Tab from the top of the page.
	for (
		let i = 0;
		i < 30 && !(await username.evaluate((el) => el === document.activeElement));
		i++
	) {
		await page.keyboard.press('Tab');
	}
	await expect(username).toBeFocused();
	await page.keyboard.type(user.username);
	await page.keyboard.press('Tab');
	await expect(page.getByLabel(msg('auth.login.password'), { exact: true })).toBeFocused();
	await page.keyboard.type(user.password);
	await page.keyboard.press('Enter');
	await expect(page).toHaveURL(`${BASE_URL}/`);
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();

	await page.locator('body').click({ position: { x: 5, y: 5 } });
	await page.keyboard.press('n');
	const dialog = page.getByRole('dialog', { name: msg('links.dialog.addTitle') });
	await expect(dialog).toBeVisible();
	const url = page.getByLabel(msg('links.dialog.url'), { exact: true });
	await expect(url).toBeFocused();
	await page.keyboard.type(fixtureUrl(testInfo.testId));
	await page.keyboard.press('Enter');
	await expect(toast(page, msg('links.toast.saved'))).toBeVisible();
	await expect(dialog).toBeHidden();
	await expect(page.locator('li.kxm-list-row')).toHaveCount(1);
});

test('the page shortcuts: search, new link, moving and pinning', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	await withApi(user, async (api) => {
		await apiCreateLink(api, { url: fixtureUrl(testInfo.testId, '1'), description: 'Older one' });
		await apiCreateLink(api, { url: fixtureUrl(testInfo.testId, '2'), description: 'Newer one' });
	});
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	const rows = page.locator('li.kxm-list-row');
	await expect(rows).toHaveCount(2);
	const firstLabel = (await rows.nth(0).locator('[data-link-anchor]').innerText()).trim();
	const secondLabel = (await rows.nth(1).locator('[data-link-anchor]').innerText()).trim();

	const search = page.getByRole('searchbox', { name: msg('nav.search.label') });
	await page.keyboard.press('/');
	await expect(search).toBeFocused();
	// Escape clears the typed search and its ?q filter on the desktop too; the focus stays.
	await page.keyboard.type('Older');
	await expect(page).toHaveURL(/[?&]q=Older/);
	await expect(rows).toHaveCount(1);
	await page.keyboard.press('Escape');
	await expect(search).toHaveValue('');
	await expect(page).not.toHaveURL(/[?&]q=/);
	await expect(search).toBeFocused();
	await expect(rows).toHaveCount(2);
	await page.keyboard.press('Escape');
	await page.locator('body').click({ position: { x: 5, y: 5 } });

	await page.keyboard.press('n');
	const dialog = page.getByRole('dialog', { name: msg('links.dialog.addTitle') });
	await expect(dialog).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(dialog).toBeHidden();
	// A click on empty page space gives <main> itself the focus; j and k still move.
	const main = page.locator('main');
	const box = await main.boundingBox();
	if (!box) throw new Error('main has no box');
	const bottom = Math.min(box.y + box.height, page.viewportSize()?.height ?? 900);
	await page.mouse.click(box.x + box.width / 2, bottom - 10);
	await expect(main).toBeFocused();

	await page.keyboard.press('j');
	await expect(rows.nth(0).locator('[data-link-anchor]')).toBeFocused();
	await page.keyboard.press('j');
	await expect(rows.nth(1).locator('[data-link-anchor]')).toBeFocused();
	await page.keyboard.press('k');
	await expect(rows.nth(0).locator('[data-link-anchor]')).toBeFocused();
	await page.keyboard.press('j');

	await page.keyboard.press('p');
	await expect(toast(page, msg('links.toast.pinned'))).toBeVisible();
	const pinned = page.locator('section[aria-labelledby="links-group-pinned"] li[data-link-id]');
	await expect(pinned).toHaveCount(1);
	await expect(pinned.locator('[data-link-anchor]')).toHaveText(secondLabel);
	expect(firstLabel).not.toBe(secondLabel);
});

test('the hero image of the sign-in page loads', async ({ page }) => {
	await page.goto('/login');
	const hero = page.getByRole('img', { name: msg('landing.hero.alt') });
	await expect(hero).toBeVisible();
	await expect(page.locator('[data-hero]')).toHaveAttribute('data-hero', 'loaded');
	expect(await hero.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBe(1600);
});
