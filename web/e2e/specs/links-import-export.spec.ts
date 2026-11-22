// Import and export (plan 9.4.2 links-import-export.spec.ts): JSON and HTML downloads with their
// file names and formats, re-import of the JSON by another user with the summary, and a broken file.
import { readFile } from 'node:fs/promises';
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import {
	apiCreateLink,
	apiCreateTag,
	fixtureUrl,
	gotoLinks,
	msg,
	toast,
	withApi
} from './lib/app.ts';

async function exportAs(page: Page, itemKey: string) {
	await page.getByRole('button', { name: msg('links.importExport.label'), exact: true }).click();
	const downloadPromise = page.waitForEvent('download');
	await page.getByRole('menuitem', { name: msg(itemKey), exact: true }).click();
	const download = await downloadPromise;
	await expect(toast(page, msg('links.toast.exported'))).toBeVisible();
	return {
		filename: download.suggestedFilename(),
		text: await readFile(await download.path(), 'utf8')
	};
}

/** The value of one row of the import summary. */
function summaryValue(page: Page, labelKey: string) {
	return page
		.getByRole('dialog', { name: msg('links.importExport.resultTitle') })
		.locator('dt')
		.filter({ hasText: msg(labelKey) })
		.locator('xpath=following-sibling::dd[1]');
}

test('export as JSON and HTML, import into another account', async ({
	createUser,
	openPage
}, testInfo) => {
	const owner = await createUser('developer');
	const receiver = await createUser('developer');
	const urls = ['a', 'b', 'c'].map((suffix) => fixtureUrl(testInfo.testId, suffix));
	await withApi(owner, async (api) => {
		const tag = await apiCreateTag(api, 'exported-tag', 'violet');
		await apiCreateLink(api, { url: urls[0], description: 'First export', tagIds: [tag.id] });
		await apiCreateLink(api, { url: urls[1], description: 'Second export', tagIds: [tag.id] });
		await apiCreateLink(api, { url: urls[2], description: 'Third export' });
	});
	// The receiver already has one of them.
	await withApi(receiver, (api) => apiCreateLink(api, { url: urls[2], description: 'Own copy' }));

	const page = await openPage(owner.storageState);
	await gotoLinks(page);
	const json = await exportAs(page, 'links.importExport.exportJson');
	expect(json.filename).toMatch(/^kinex-mark-links-\d{8}\.json$/);
	const data = JSON.parse(json.text) as {
		format: string;
		links: { url: string; description: string; tags: string[] }[];
	};
	expect(data.format).toBe('kinex-mark-links/1');
	expect(data.links.map((link) => link.url).sort()).toEqual([...urls].sort());
	const first = data.links.find((link) => link.url === urls[0]);
	expect(first).toMatchObject({ description: 'First export', tags: ['exported-tag'] });

	const html = await exportAs(page, 'links.importExport.exportHtml');
	expect(html.filename).toMatch(/^kinex-mark-links-\d{8}\.html$/);
	expect(html.text).toMatch(/^<!DOCTYPE NETSCAPE-Bookmark-file-1>/i);
	for (const url of urls) expect(html.text).toContain(`HREF="${url}"`);

	const receiverPage = await openPage(receiver.storageState);
	await gotoLinks(receiverPage);
	await expect(receiverPage.locator('li.kxm-list-row')).toHaveCount(1);
	await receiverPage.getByTestId('import-input').setInputFiles({
		name: json.filename,
		mimeType: 'application/json',
		buffer: Buffer.from(json.text, 'utf8')
	});
	await expect(
		receiverPage.getByRole('dialog', { name: msg('links.importExport.resultTitle') })
	).toBeVisible();
	await expect(summaryValue(receiverPage, 'links.importExport.total')).toHaveText('3');
	await expect(summaryValue(receiverPage, 'links.importExport.created')).toHaveText('2');
	await expect(summaryValue(receiverPage, 'links.importExport.duplicates')).toHaveText('1');
	await expect(summaryValue(receiverPage, 'links.importExport.invalid')).toHaveText('0');
	await receiverPage
		.getByRole('button', { name: msg('common.actions.close'), exact: true })
		.click();
	await expect(receiverPage.locator('li.kxm-list-row')).toHaveCount(3);
	const tags = await withApi(
		receiver,
		async (api) =>
			((await (await api.get('/api/tags')).json()) as { items: { name: string }[] }).items
	);
	expect(tags.map((tag) => tag.name)).toEqual(['exported-tag']);
});

test('a file that is neither format is refused', async ({ createUser, openPage, expectHttp }) => {
	const user = await createUser('customer');
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	expectHttp(400, '/api/links/import', 'POST');
	await page.getByTestId('import-input').setInputFiles({
		name: 'notes.json',
		mimeType: 'application/json',
		buffer: Buffer.from('{"hello": "world"}', 'utf8')
	});
	await expect(toast(page, msg('errors.import_invalid.title'))).toBeVisible();
	await expect(page.getByText(msg('links.empty.collection.title'))).toBeVisible();
});
