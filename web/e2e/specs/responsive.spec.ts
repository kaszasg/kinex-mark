// Phone layout (plan 9.4.2 responsive.spec.ts, mobile project, Pixel 7): the off-canvas sidebar,
// no horizontal page scroll on the main pages, and the row actions through the row menu.
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import {
	apiCreateLink,
	apiDeleteLink,
	fixtureUrl,
	gotoLinks,
	menuName,
	msg,
	toast,
	withApi
} from './lib/app.ts';

async function expectNoHorizontalScroll(page: Page, path: string): Promise<void> {
	const sizes = await page.evaluate(() => ({
		scroll: document.documentElement.scrollWidth,
		inner: window.innerWidth
	}));
	expect(sizes.scroll, `${path}: ${JSON.stringify(sizes)}`).toBeLessThanOrEqual(sizes.inner);
}

test('the sidebar opens off-canvas and closes again', async ({ asDeveloper }) => {
	const page = asDeveloper;
	await gotoLinks(page);
	const nav = page.getByRole('navigation', { name: msg('nav.sidebarTitle') });
	await expect(nav.getByRole('link', { name: msg('nav.tags'), exact: true })).toBeHidden();
	await page.getByRole('button', { name: msg('nav.toggleSidebar'), exact: true }).click();
	await expect(nav.getByRole('link', { name: msg('nav.tags'), exact: true })).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(nav.getByRole('link', { name: msg('nav.tags'), exact: true })).toBeHidden();

	// Following a link closes the sidebar too.
	await page.getByRole('button', { name: msg('nav.toggleSidebar'), exact: true }).click();
	await nav.getByRole('link', { name: msg('nav.tags'), exact: true }).click();
	await expect(page.getByRole('heading', { level: 1, name: msg('tags.pageTitle') })).toBeVisible();
	await expect(nav.getByRole('link', { name: msg('nav.tags'), exact: true })).toBeHidden();
});

test('the main pages do not scroll sideways', async ({ page, asAdmin, adminApi }, testInfo) => {
	for (const path of ['/login', '/privacy']) {
		await page.goto(path);
		await expect(page.getByRole('heading', { level: 1 }).first()).toBeVisible();
		await expectNoHorizontalScroll(page, path);
	}
	const sample = await apiCreateLink(adminApi, {
		url: `${fixtureUrl(testInfo.testId)}?with=${'long-query-value-'.repeat(8)}`,
		description: 'A rather long description that must wrap or truncate on a phone screen'
	});
	try {
		for (const path of [
			'/',
			'/tags',
			'/settings/profile',
			'/settings/display',
			'/settings/security',
			'/settings/privacy',
			'/admin/users',
			'/admin/settings',
			'/admin/audit'
		]) {
			await asAdmin.goto(path);
			await expect(asAdmin.getByRole('heading', { level: 1 }).first()).toBeVisible();
			await expect(asAdmin.locator('[aria-busy="true"]')).toHaveCount(0);
			await expectNoHorizontalScroll(asAdmin, path);
		}
	} finally {
		await apiDeleteLink(adminApi, sample.id);
	}
});

test('row actions are reachable from the row menu', async ({ createUser, openPage }, testInfo) => {
	const user = await createUser('developer');
	await withApi(user, (api) =>
		apiCreateLink(api, { url: fixtureUrl(testInfo.testId), description: 'Phone link' })
	);
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	const row = page.locator('li.kxm-list-row').first();
	const menuButton = row.getByRole('button', { name: menuName('Phone link'), exact: true });
	await expect(menuButton).toBeVisible();
	await menuButton.tap();
	const menu = page.getByRole('menu');
	for (const key of [
		'links.actions.details',
		'links.actions.edit',
		'links.actions.share',
		'links.actions.archive',
		'links.actions.delete'
	]) {
		await expect(menu.getByRole('menuitem', { name: msg(key), exact: true })).toBeVisible();
	}
	await menu.getByRole('menuitem', { name: msg('links.actions.archive'), exact: true }).tap();
	await expect(toast(page, msg('links.toast.archived'))).toBeVisible();
	await expect(page.locator('li.kxm-list-row')).toHaveCount(0);
});

test('a long unbroken label stays inside the tag dialog on a 360 px screen', async ({
	createUser,
	openPage
}, testInfo) => {
	const label = `Kinexmarkhosszucimke${'x'.repeat(140)}`;
	const user = await createUser('developer');
	await withApi(user, (api) =>
		apiCreateLink(api, { url: fixtureUrl(testInfo.testId), description: label })
	);
	const page = await openPage(user.storageState);
	await page.setViewportSize({ width: 360, height: 740 });
	await gotoLinks(page);
	await page.getByRole('button', { name: menuName(label), exact: true }).tap();
	await page
		.getByRole('menu')
		.getByRole('menuitem', { name: msg('links.actions.tags'), exact: true })
		.tap();
	const dialog = page.getByRole('dialog', { name: msg('links.tagsDialog.title') });
	await expect(dialog).toBeVisible();
	// The box keeps its width (phase E finding: long text widened the dialog past the screen).
	const box = await dialog.evaluate((el) => {
		const rect = el.getBoundingClientRect();
		return { left: rect.left, right: rect.right, scroll: el.scrollWidth, client: el.clientWidth };
	});
	expect(box.left, JSON.stringify(box)).toBeGreaterThanOrEqual(0);
	expect(box.right, JSON.stringify(box)).toBeLessThanOrEqual(360);
	expect(box.scroll, JSON.stringify(box)).toBeLessThanOrEqual(box.client);
	for (const name of [msg('common.actions.cancel'), msg('links.tagsDialog.save')]) {
		await expect(dialog.getByRole('button', { name, exact: true })).toBeInViewport();
	}
	await expectNoHorizontalScroll(page, 'tag dialog');
});
