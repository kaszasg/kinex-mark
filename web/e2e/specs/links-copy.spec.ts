// Copy URL and copy as Markdown (plan 9.4.2 links-copy.spec.ts): the clipboard holds the exact
// text, and every copy counts as a use that moves the link up in the ranking.
import { test, expect, uniqueUrl } from '../support/fixtures.ts';
import { BASE_URL } from '../support/constants.mjs';
import {
	ALPHA_TITLE,
	ALPHA_URL,
	apiCreateLink,
	detailsSheet,
	detailsUseCount,
	gotoLinks,
	msg,
	rowAction,
	rowLabels,
	toast,
	waitFetched,
	withApi
} from './lib/app.ts';

test('copying writes the URL and the Markdown link and counts as a use', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	await withApi(user, async (api) => {
		const alpha = await apiCreateLink(api, { url: ALPHA_URL });
		const fetched = await waitFetched(api, alpha.id);
		expect(fetched.title).toBe(ALPHA_TITLE);
		// A newer link that is never used: without uses it would rank first.
		await apiCreateLink(api, { url: uniqueUrl(testInfo), description: 'Never used' });
	});
	const page = await openPage(user.storageState);
	await page
		.context()
		.grantPermissions(['clipboard-read', 'clipboard-write'], { origin: BASE_URL });
	await gotoLinks(page);
	expect(await rowLabels(page)).toEqual(['Never used', ALPHA_TITLE]);

	const readClipboard = () => page.evaluate(() => navigator.clipboard.readText());

	await rowAction(page, ALPHA_TITLE, 'links.actions.copyUrl');
	await expect(toast(page, msg('links.toast.urlCopied'))).toBeVisible();
	expect(await readClipboard()).toBe(ALPHA_URL);
	await expect.poll(() => rowLabels(page)).toEqual([ALPHA_TITLE, 'Never used']);

	await rowAction(page, ALPHA_TITLE, 'links.actions.copyMarkdown');
	await expect(toast(page, msg('links.toast.markdownCopied'))).toBeVisible();
	expect(await readClipboard()).toBe(`[${ALPHA_TITLE}](${ALPHA_URL})`);

	await rowAction(page, ALPHA_TITLE, 'links.actions.details');
	await expect(detailsSheet(page)).toBeVisible();
	await expect.poll(() => detailsUseCount(page)).toBe(2);

	// The copy buttons of the details sheet count as well.
	await detailsSheet(page)
		.getByRole('button', { name: msg('links.actions.copyMarkdown'), exact: true })
		.click();
	await expect.poll(() => detailsUseCount(page)).toBe(3);
	expect(await readClipboard()).toBe(`[${ALPHA_TITLE}](${ALPHA_URL})`);

	// The counts are the server's.
	await page.reload();
	await expect(detailsSheet(page)).toBeVisible();
	await expect.poll(() => detailsUseCount(page)).toBe(3);
	await page.keyboard.press('Escape');
	expect(await rowLabels(page)).toEqual([ALPHA_TITLE, 'Never used']);
});
