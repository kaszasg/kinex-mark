// List and tile view, the four display switches and the label rule (plan 9.4.2 links-view.spec.ts).
import type { Page } from '@playwright/test';
import { test, expect, uniqueUrl } from '../support/fixtures.ts';
import { FIXTURE_ORIGIN } from '../support/constants.mjs';
import { apiCreateLink, gotoLinks, msg, rowByLabel, waitFetched, withApi } from './lib/app.ts';

const viewGroup = (page: Page) => page.getByRole('group', { name: msg('links.toolbar.view') });

async function toggleField(
	page: Page,
	key: 'icon' | 'url' | 'title' | 'description'
): Promise<void> {
	await page.getByRole('button', { name: msg('links.toolbar.display'), exact: true }).click();
	const menu = page.getByRole('menu');
	const saved = page.waitForResponse(
		(response) =>
			response.request().method() === 'PATCH' &&
			new URL(response.url()).pathname === '/api/me/preferences'
	);
	await menu
		.getByRole('menuitemcheckbox', { name: msg(`links.fields.${key}`), exact: true })
		.click();
	expect((await saved).status()).toBe(200);
	await page.keyboard.press('Escape');
	await expect(menu).toBeHidden();
}

test('the tile view is kept after a reload and saved on the server', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	await withApi(user, (api) =>
		apiCreateLink(api, { url: uniqueUrl(testInfo), description: 'Tile candidate' })
	);
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	await expect(
		viewGroup(page).getByRole('radio', { name: msg('links.view.list') })
	).toHaveAttribute('aria-checked', 'true');
	await expect(page.locator('li.kxm-list-row')).toHaveCount(1);

	const saved = page.waitForResponse(
		(response) =>
			response.request().method() === 'PATCH' &&
			new URL(response.url()).pathname === '/api/me/preferences'
	);
	await viewGroup(page)
		.getByRole('radio', { name: msg('links.view.tiles') })
		.click();
	expect((await saved).status()).toBe(200);
	await expect(page.locator('li.kxm-list-row')).toHaveCount(0);
	const tile = page.locator('li[data-link-id]');
	await expect(tile).toHaveCount(1);
	await expect(tile.locator('[data-link-anchor]')).toHaveText('Tile candidate');

	await page.reload();
	await expect(page.locator('li[data-link-id]')).toHaveCount(1);
	await expect(page.locator('li.kxm-list-row')).toHaveCount(0);
	await expect(
		viewGroup(page).getByRole('radio', { name: msg('links.view.tiles') })
	).toHaveAttribute('aria-checked', 'true');
	const prefs = await withApi(user, async (api) => (await api.get('/api/me/preferences')).json());
	expect(prefs).toMatchObject({ viewMode: 'tiles' });

	// "v" switches back to the list.
	await page.locator('body').press('v');
	await expect(page.locator('li.kxm-list-row')).toHaveCount(1);
});

test('the display switches change the label and the secondary lines, also after a reload', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('customer');
	const url = uniqueUrl(testInfo);
	await withApi(user, async (api) => {
		const link = await apiCreateLink(api, {
			url,
			title: 'Manual title',
			description: 'Short description'
		});
		await waitFetched(api, link.id);
	});
	const pretty = url.replace('http://', '');
	// A URL label longer than 40 characters collapses to the host (plan 8.7.3).
	const urlLabel = [...pretty].length <= 40 ? pretty : new URL(url).hostname;
	const page = await openPage(user.storageState);
	await gotoLinks(page);

	// Description first, the title and the URL as secondary lines.
	let row = rowByLabel(page, 'Short description');
	await expect(row.locator('[data-secondary="title"]')).toHaveText('Manual title');
	await expect(row.locator('[data-secondary="url"]')).toHaveText(pretty);
	await expect(row.locator('img')).toHaveCount(1);

	await toggleField(page, 'description');
	row = rowByLabel(page, 'Manual title');
	await expect(row).toBeVisible();
	await expect(row.locator('[data-secondary="description"]')).toHaveCount(0);
	await expect(row.locator('[data-secondary="url"]')).toBeVisible();

	await toggleField(page, 'title');
	// Without description and title the label is the URL, and no URL line repeats it.
	row = rowByLabel(page, urlLabel);
	await expect(row).toBeVisible();
	await expect(row.locator('[data-secondary]')).toHaveCount(0);

	await toggleField(page, 'icon');
	await expect(row.locator('img')).toHaveCount(0);

	await page.reload();
	row = rowByLabel(page, urlLabel);
	await expect(row).toBeVisible();
	await expect(row.locator('img')).toHaveCount(0);
	await expect(row.locator('[data-secondary]')).toHaveCount(0);

	// Back on, URL switched off: the description label with the title line only.
	await toggleField(page, 'description');
	await toggleField(page, 'title');
	await toggleField(page, 'url');
	row = rowByLabel(page, 'Short description');
	await expect(row.locator('[data-secondary="title"]')).toHaveText('Manual title');
	await expect(row.locator('[data-secondary="url"]')).toHaveCount(0);
	await page.reload();
	row = rowByLabel(page, 'Short description');
	await expect(row.locator('[data-secondary="url"]')).toHaveCount(0);
});

test('the label follows description, title, URL and shows the domain for a long URL', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	const longUrl = `${FIXTURE_ORIGIN}/plain?${testInfo.testId}=${'segment-'.repeat(8)}`;
	await withApi(user, async (api) => {
		await apiCreateLink(api, {
			url: uniqueUrl(testInfo, 'a'),
			title: 'Title only',
			description: ''
		});
		await apiCreateLink(api, {
			url: uniqueUrl(testInfo, 'b'),
			title: 'Hidden title',
			description: 'Description wins'
		});
		const long = await apiCreateLink(api, { url: longUrl });
		// /plain is not a web page: no title arrives, so the URL stays the label.
		await waitFetched(api, long.id);
	});
	const page = await openPage(user.storageState);
	await gotoLinks(page);

	await expect(rowByLabel(page, 'Title only')).toBeVisible();
	const described = rowByLabel(page, 'Description wins');
	await expect(described.locator('[data-secondary="title"]')).toHaveText('Hidden title');
	const host = new URL(FIXTURE_ORIGIN).hostname;
	const long = rowByLabel(page, host);
	await expect(long).toBeVisible();
	await expect(long.locator('[data-link-anchor]')).toHaveAttribute('title', longUrl);
});
