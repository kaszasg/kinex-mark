// Adding links (plan 9.4.2 links-capture.spec.ts): the add dialog with tags, quick capture by
// drag-and-drop and by paste, the no-URL message, pasting into a text field, and duplicates.
import type { Page } from '@playwright/test';
import { test, expect, uniqueUrl } from '../support/fixtures.ts';
import { dropUrl, pasteText } from '../support/helpers.ts';
import {
	addLinkNames,
	apiCreateLink,
	apiCreateTag,
	fixtureTitle,
	gotoLinks,
	msg,
	rowByLabel,
	toast,
	withApi
} from './lib/app.ts';

/** Counts the quick-capture requests the page sends. */
function countQuickCaptures(page: Page): () => number {
	let count = 0;
	page.on('request', (request) => {
		if (new URL(request.url()).pathname === '/api/links/quick') count += 1;
	});
	return () => count;
}

test('a link added in the dialog gets its tags and description', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	await withApi(user, async (api) => {
		await apiCreateTag(api, 'docs', 'blue');
		await apiCreateTag(api, 'ops', 'green');
	});
	const url = uniqueUrl(testInfo);
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	await expect(page.getByText(msg('links.empty.collection.title'))).toBeVisible();

	const names = addLinkNames();
	await page.getByRole('button', { name: names.open, exact: true }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('heading', { name: msg('links.dialog.addTitle') })).toBeVisible();
	await dialog.getByLabel(names.url, { exact: true }).fill(url);
	await dialog.getByLabel(names.description, { exact: true }).fill('Runbook of the release');
	await dialog.getByRole('button', { name: msg('links.dialog.tags'), exact: true }).click();
	const picker = page.getByRole('listbox', { name: msg('common.tagPicker.label') });
	await picker.getByRole('option', { name: 'docs' }).click();
	await picker.getByRole('option', { name: 'ops' }).click();
	await page.keyboard.press('Escape');
	await expect(picker).toBeHidden();
	await expect(dialog).toBeVisible();
	await expect(
		dialog.getByRole('button', { name: msg('common.tagPicker.remove', { name: 'docs' }) })
	).toBeVisible();

	const created = page.waitForResponse(
		(response) =>
			response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/links'
	);
	await dialog.getByRole('button', { name: names.submit, exact: true }).click();
	expect((await created).status()).toBe(201);
	await expect(dialog).toBeHidden();
	await expect(toast(page, msg('links.toast.saved'))).toBeVisible();

	const row = rowByLabel(page, 'Runbook of the release');
	await expect(row).toBeVisible();
	await expect(row.getByText('docs', { exact: true })).toBeVisible();
	await expect(row.getByText('ops', { exact: true })).toBeVisible();
	await expect(page.getByText(msg('links.summary', { count: 1, pinned: 0 }))).toBeVisible();
	// The fetched title shows as the secondary line under the description.
	await expect(row.locator('[data-secondary="title"]')).toHaveText(fixtureTitle(url), {
		timeout: 20_000
	});
});

test('a link dragged onto the window is saved with its anchor text', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('customer');
	const url = uniqueUrl(testInfo);
	const page = await openPage(user.storageState);
	await gotoLinks(page);

	await dropUrl(page, url, {
		title: 'Dragged handbook',
		beforeDrop: async () => {
			const overlay = page.getByTestId('capture-overlay');
			await expect(overlay).toBeVisible();
			await expect(overlay).toContainText(msg('capture.overlay.title'));
		}
	});
	await expect(page.getByTestId('capture-overlay')).toBeHidden();
	await expect(toast(page, msg('capture.toast.saved'))).toBeVisible();
	const row = rowByLabel(page, 'Dragged handbook');
	await expect(row).toBeVisible();
	await expect(row.locator('[data-link-anchor]')).toHaveAttribute('href', url);

	// The same link again: the toast says it exists, nothing new is created.
	await dropUrl(page, url, { title: 'Dragged handbook' });
	await expect(toast(page, msg('capture.toast.exists'))).toBeVisible();
	await expect(page.locator('li.kxm-list-row')).toHaveCount(1);
});

test('pasted text with a URL is captured, text without one is refused', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	const url = uniqueUrl(testInfo);
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	const captures = countQuickCaptures(page);

	await pasteText(page, `Notes for tomorrow: ${url}, please read.`);
	await expect(toast(page, msg('capture.toast.saved'))).toBeVisible();
	// The label becomes the fetched title once the metadata arrives.
	await expect(rowByLabel(page, fixtureTitle(url))).toBeVisible({ timeout: 20_000 });
	expect(captures()).toBe(1);

	await pasteText(page, 'just some words without any address');
	await expect(toast(page, msg('errors.no_url_found.title'))).toBeVisible();
	expect(captures()).toBe(1);
	await expect(page.locator('li.kxm-list-row')).toHaveCount(1);
});

test('pasting into a text field does not capture', async ({ createUser, openPage }, testInfo) => {
	const user = await createUser('developer');
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	const captures = countQuickCaptures(page);
	const search = page.getByRole('searchbox', { name: msg('nav.search.label') });
	await search.click();
	await pasteText(page, uniqueUrl(testInfo), { target: search });
	// Give a would-be capture time to start.
	await page.waitForTimeout(500);
	expect(captures()).toBe(0);
	await expect(page.getByText(msg('links.empty.collection.title'))).toBeVisible();
});

test('a duplicate URL shows a warning and highlights the existing link', async ({
	createUser,
	openPage,
	expectHttp
}, testInfo) => {
	const user = await createUser('developer');
	const url = uniqueUrl(testInfo);
	await withApi(user, (api) => apiCreateLink(api, { url, description: 'The original' }));
	const page = await openPage(user.storageState);
	await gotoLinks(page);

	expectHttp(409, '/api/links', 'POST');
	const names = addLinkNames();
	await page.getByRole('button', { name: names.open, exact: true }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByLabel(names.url, { exact: true }).fill(url);
	await dialog.getByRole('button', { name: names.submit, exact: true }).click();
	await expect(dialog.getByText(msg('errors.link_duplicate.title'))).toBeVisible();
	await dialog.getByRole('button', { name: msg('links.dialog.showExisting'), exact: true }).click();
	await expect(dialog).toBeHidden();
	const row = rowByLabel(page, 'The original');
	await expect(row).toHaveClass(/kxm-flash/);
	await expect(page.locator('li.kxm-list-row')).toHaveCount(1);
});
