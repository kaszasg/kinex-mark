// Metadata and the details sheet (plan 9.4.2 links-detail.spec.ts): fetched titles and icons
// from the local fixture origin, failed fetches, every block of the sheet, editing, refreshing
// and deleting a link.
import { readFile } from 'node:fs/promises';
import type { Page } from '@playwright/test';
import { test, expect, uniqueUrl } from '../support/fixtures.ts';
import {
	ALPHA_TITLE,
	ALPHA_URL,
	BETA_URL,
	PLAIN_URL,
	addLinkNames,
	apiCreateLink,
	apiGetLink,
	detailsSheet,
	exactText,
	fixtureTitle,
	gotoLinks,
	msg,
	rowAction,
	rowByLabel,
	toast,
	waitFetched,
	withApi
} from './lib/app.ts';
import { addLink } from '../support/helpers.ts';

/** The title of the windows-1250 fixture page, read from the fixture file. */
async function betaTitle(): Promise<string> {
	const html = await readFile(new URL('../fixtures/site/beta.html', import.meta.url), 'utf8');
	const match = /<title>([^<]*)<\/title>/.exec(html);
	if (!match) throw new Error('beta.html has no title');
	return match[1].trim();
}

async function iconLoaded(page: Page, label: string, linkId: number): Promise<string> {
	const icon = rowByLabel(page, label).locator('img');
	await expect(icon).toHaveAttribute('src', new RegExp(`^/api/links/${linkId}/icon`));
	await expect
		.poll(() => icon.evaluate((img) => (img as HTMLImageElement).naturalWidth))
		.toBeGreaterThan(0);
	return (await icon.getAttribute('src')) ?? '';
}

test('titles and icons arrive from the fixture pages, failures are named', async ({
	createUser,
	openPage
}) => {
	const user = await createUser('developer');
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	const names = addLinkNames();
	const alphaId = await addLink(page, { url: ALPHA_URL }, names);
	expect(alphaId).toBeDefined();
	// The row turns from the URL into the fetched title within the polling window.
	await expect(rowByLabel(page, ALPHA_TITLE)).toBeVisible({ timeout: 20_000 });
	await iconLoaded(page, ALPHA_TITLE, alphaId as number);

	await addLink(page, { url: BETA_URL }, names);
	await expect(rowByLabel(page, await betaTitle())).toBeVisible({ timeout: 20_000 });

	const plainId = await addLink(page, { url: PLAIN_URL }, names);
	await withApi(user, (api) => waitFetched(api, plainId as number));
	await page.goto(`/?link=${plainId}`);
	// A text file is fetched, but it has no title: the reason is named.
	await expect(detailsSheet(page).getByTestId('details-fetch')).toContainText(
		msg('fetch.errors.not_html')
	);
	await page.keyboard.press('Escape');
	await expect(rowByLabel(page, PLAIN_URL.replace('http://', ''))).toBeVisible();
});

test('the details sheet shows every block and edits change the label', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	const link = await withApi(user, async (api) => {
		const created = await apiCreateLink(api, { url: ALPHA_URL });
		return waitFetched(api, created.id);
	});
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	await rowAction(page, ALPHA_TITLE, 'links.actions.details');
	const sheet = detailsSheet(page);
	await expect(sheet.getByRole('heading', { name: ALPHA_TITLE })).toBeVisible();
	await expect(page).toHaveURL(new RegExp(`[?&]link=${link.id}`));
	for (const key of ['data', 'usage', 'times', 'metadata', 'sharing']) {
		await expect(
			sheet.getByRole('heading', { level: 3, name: msg(`links.details.sections.${key}`) })
		).toBeVisible();
	}
	await expect(sheet.getByTestId('details-url')).toHaveText(ALPHA_URL);
	await expect(sheet.getByText(msg('links.details.noTags'))).toBeVisible();
	await expect(sheet.getByTestId('details-fetch')).toContainText(msg('fetch.status.ok'));
	await expect(sheet.getByText(msg('links.share.empty'))).toBeVisible();
	for (const key of ['created', 'updated', 'lastUsed', 'inactiveSince', 'archivedAt']) {
		await expect(
			sheet.getByRole('term').filter({ hasText: exactText(msg(`links.details.times.${key}`)) })
		).toBeVisible();
	}

	// Editing the description makes it the label.
	await sheet.getByRole('button', { name: msg('links.actions.edit'), exact: true }).click();
	const dialog = page.getByRole('dialog', { name: msg('links.dialog.editTitle') });
	await dialog.getByLabel(msg('links.dialog.description'), { exact: true }).fill('Alpha handbook');
	await dialog.getByRole('button', { name: msg('links.dialog.save'), exact: true }).click();
	await expect(dialog).toBeHidden();
	await expect(sheet.getByRole('heading', { name: 'Alpha handbook' })).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(rowByLabel(page, 'Alpha handbook')).toBeVisible();

	// A new URL brings the new page's title and icon.
	const before = await iconLoaded(page, 'Alpha handbook', link.id);
	const url = uniqueUrl(testInfo);
	await rowAction(page, 'Alpha handbook', 'links.actions.edit');
	await dialog.getByLabel(msg('links.dialog.description'), { exact: true }).fill('');
	await dialog.getByLabel(msg('links.dialog.url'), { exact: true }).fill(url);
	await dialog.getByRole('button', { name: msg('links.dialog.save'), exact: true }).click();
	await expect(dialog).toBeHidden();
	await expect(rowByLabel(page, fixtureTitle(url))).toBeVisible({ timeout: 20_000 });
	const after = await iconLoaded(page, fixtureTitle(url), link.id);
	expect(after).not.toBe(before);
});

test('a manual title survives a metadata refresh, deletion asks first', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('customer');
	const url = uniqueUrl(testInfo);
	const link = await withApi(user, async (api) => {
		const created = await apiCreateLink(api, { url, title: 'My own name' });
		return waitFetched(api, created.id);
	});
	expect(link.title).toBe('My own name');
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	await rowAction(page, 'My own name', 'links.actions.details');
	const sheet = detailsSheet(page);
	const refreshed = page.waitForResponse(
		(response) => response.request().method() === 'POST' && /\/refresh$/.test(response.url())
	);
	await sheet.getByRole('button', { name: msg('links.actions.refresh'), exact: true }).click();
	expect((await refreshed).status()).toBe(202);
	await expect(toast(page, msg('links.toast.refreshing'))).toBeVisible();
	await withApi(user, async (api) => {
		const done = await waitFetched(api, link.id);
		expect(done.fetchStatus).toBe('ok');
		expect((await apiGetLink(api, link.id)).title).toBe('My own name');
	});
	await expect(sheet.getByTestId('details-fetch')).toContainText(msg('fetch.status.ok'), {
		timeout: 20_000
	});
	await expect(sheet.getByRole('heading', { name: 'My own name' })).toBeVisible();

	await sheet.getByRole('button', { name: msg('links.details.deleteLink'), exact: true }).click();
	const confirm = page.getByRole('alertdialog');
	await expect(confirm).toContainText(msg('links.delete.text', { label: 'My own name' }));
	await confirm.getByRole('button', { name: msg('links.delete.confirm'), exact: true }).click();
	await expect(toast(page, msg('links.toast.deleted'))).toBeVisible();
	await expect(sheet).toBeHidden();
	await expect(page.getByText(msg('links.empty.collection.title'))).toBeVisible();
	await withApi(user, async (api) => {
		expect((await api.get(`/api/links/${link.id}`)).status()).toBe(404);
	});
});
