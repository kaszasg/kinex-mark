// Usage ranking and pinning (plan 9.4.2 links-ranking.spec.ts): opened links move up, pinned
// links come first, usage orders both groups, and one keyboard opening counts exactly one use.
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import {
	apiCreateLink,
	detailsSheet,
	detailsUseCount,
	fixtureUrl,
	gotoLinks,
	msg,
	rowAction,
	rowByLabel,
	rowLabels,
	withApi
} from './lib/app.ts';

/** Opens a link by clicking its label; waits for the counted visit and closes the new tab. */
async function openByClick(page: Page, label: string): Promise<void> {
	const popup = page.context().waitForEvent('page');
	const visit = page.waitForResponse(
		(response) =>
			response.request().method() === 'POST' && /\/api\/links\/\d+\/visit$/.test(response.url())
	);
	await rowByLabel(page, label).locator('[data-link-anchor]').click();
	expect((await visit).status()).toBe(200);
	await (await popup).close();
}

async function pinByButton(page: Page, label: string): Promise<void> {
	const row = rowByLabel(page, label);
	await row.hover();
	const pinned = page.waitForResponse(
		(response) => response.request().method() === 'POST' && /\/pin$/.test(response.url())
	);
	await row.getByRole('button', { name: msg('links.actions.pin'), exact: true }).click();
	expect((await pinned).status()).toBe(200);
	await expect(
		row.getByRole('button', { name: msg('links.actions.unpin'), exact: true })
	).toHaveAttribute('aria-pressed', 'true');
}

/** Labels per group: the pinned group first, then the others. */
async function groups(page: Page): Promise<{ pinned: string[]; others: string[] }> {
	const read = (id: string) =>
		page
			.locator(`section[aria-labelledby="${id}"] li.kxm-list-row [data-link-anchor]`)
			.allInnerTexts();
	return { pinned: await read('links-group-pinned'), others: await read('links-group-others') };
}

test('opening, pinning and usage decide the order', async ({ createUser, openPage }, testInfo) => {
	test.setTimeout(120_000);
	const user = await createUser('developer');
	// Created C, B, A: without uses the newest comes first (A, B, C).
	await withApi(user, async (api) => {
		for (const name of ['C', 'B', 'A']) {
			await apiCreateLink(api, {
				url: fixtureUrl(testInfo.testId, name),
				description: `Link ${name}`
			});
		}
	});
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	expect(await rowLabels(page)).toEqual(['Link A', 'Link B', 'Link C']);

	await openByClick(page, 'Link C');
	await openByClick(page, 'Link C');
	await openByClick(page, 'Link B');
	await expect.poll(() => rowLabels(page)).toEqual(['Link C', 'Link B', 'Link A']);

	await pinByButton(page, 'Link A');
	await pinByButton(page, 'Link C');
	await openByClick(page, 'Link C');
	await expect
		.poll(() => groups(page))
		.toEqual({ pinned: ['Link C', 'Link A'], others: ['Link B'] });
	await expect(
		page.getByRole('heading', { level: 2, name: msg('links.groups.pinned') })
	).toBeVisible();

	for (let i = 0; i < 3; i++) await openByClick(page, 'Link B');
	await expect
		.poll(() => groups(page))
		.toEqual({ pinned: ['Link C', 'Link A'], others: ['Link B'] });

	// The server returns the same order.
	await page.reload();
	await expect(page.locator('li.kxm-list-row')).toHaveCount(3);
	expect(await groups(page)).toEqual({ pinned: ['Link C', 'Link A'], others: ['Link B'] });
	await expect(page.getByText(msg('links.summary', { count: 3, pinned: 2 }))).toBeVisible();
});

test('a keyboard opening counts exactly one use', async ({ createUser, openPage }, testInfo) => {
	const user = await createUser('customer');
	await withApi(user, (api) =>
		apiCreateLink(api, { url: fixtureUrl(testInfo.testId, 'k'), description: 'Keyboard link' })
	);
	const page = await openPage(user.storageState);
	await gotoLinks(page);

	await rowAction(page, 'Keyboard link', 'links.actions.details');
	await expect(detailsSheet(page)).toBeVisible();
	expect(await detailsUseCount(page)).toBe(0);
	await page.keyboard.press('Escape');
	await expect(detailsSheet(page)).toBeHidden();

	await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
	await page.keyboard.press('j');
	const anchor = rowByLabel(page, 'Keyboard link').locator('[data-link-anchor]');
	await expect(anchor).toBeFocused();
	let visits = 0;
	page.on('request', (request) => {
		if (request.method() === 'POST' && /\/api\/links\/\d+\/visit$/.test(request.url())) visits += 1;
	});
	const popup = page.context().waitForEvent('page');
	await page.keyboard.press('Enter');
	await (await popup).close();
	await expect.poll(() => visits).toBe(1);

	await rowAction(page, 'Keyboard link', 'links.actions.details');
	await expect.poll(() => detailsUseCount(page)).toBe(1);
	await page.keyboard.press('Escape');

	await expect(detailsSheet(page)).toBeHidden();
	await gotoLinks(page);
	await rowAction(page, 'Keyboard link', 'links.actions.details');
	await expect.poll(() => detailsUseCount(page)).toBe(1);
	expect(visits).toBe(1);
});
