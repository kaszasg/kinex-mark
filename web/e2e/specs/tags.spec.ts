// Tags page (plan 9.4.2 tags.spec.ts): all seven colours, rename, duplicate name, deletion that
// keeps the links, and the link count that opens the filtered list.
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import {
	apiCreateLink,
	apiCreateTag,
	exactText,
	fixtureUrl,
	gotoLinks,
	msg,
	rowLabels,
	toast,
	withApi
} from './lib/app.ts';

const COLORS = ['blue', 'ochre', 'green', 'violet', 'rose', 'neutral', 'contrast'] as const;

async function createTag(page: Page, name: string, color: string): Promise<void> {
	await page.getByRole('button', { name: msg('tags.add'), exact: true }).click();
	const dialog = page.getByRole('dialog', { name: msg('tags.dialog.createTitle') });
	await dialog.getByLabel(msg('tags.dialog.name'), { exact: true }).fill(name);
	await dialog.getByRole('radio', { name: msg(`tagColors.${color}`), exact: true }).click();
	await dialog.getByRole('button', { name: msg('tags.dialog.save'), exact: true }).click();
	await expect(dialog).toBeHidden();
}

function tagRow(page: Page, name: string) {
	return page.getByRole('row').filter({ has: page.getByRole('cell', { name: exactText(name) }) });
}

test('tags in all seven colours, rename and duplicate name', async ({
	createUser,
	openPage,
	expectHttp
}) => {
	const user = await createUser('developer');
	const page = await openPage(user.storageState);
	await page.goto('/tags');
	await expect(page.getByText(msg('tags.empty.title'))).toBeVisible();

	for (const color of COLORS) await createTag(page, `tag-${color}`, color);
	await expect(page.getByRole('row')).toHaveCount(COLORS.length + 1);
	for (const color of COLORS) {
		await expect(tagRow(page, `tag-${color}`)).toContainText(msg(`tagColors.${color}`));
	}
	await expect(page.getByText(msg('tags.summary', { count: 7 }))).toBeVisible();

	await page.getByRole('button', { name: msg('tags.table.edit', { name: 'tag-blue' }) }).click();
	const edit = page.getByRole('dialog', { name: msg('tags.dialog.editTitle') });
	await edit.getByLabel(msg('tags.dialog.name'), { exact: true }).fill('tag-sky');
	await edit.getByRole('button', { name: msg('tags.dialog.save'), exact: true }).click();
	await expect(toast(page, msg('tags.dialog.saved')).last()).toBeVisible();
	await expect(tagRow(page, 'tag-sky')).toBeVisible();
	await expect(tagRow(page, 'tag-blue')).toHaveCount(0);

	expectHttp(409, '/api/tags', 'POST');
	await page.getByRole('button', { name: msg('tags.add'), exact: true }).click();
	const create = page.getByRole('dialog', { name: msg('tags.dialog.createTitle') });
	await create.getByLabel(msg('tags.dialog.name'), { exact: true }).fill('TAG-SKY');
	await create.getByRole('button', { name: msg('tags.dialog.save'), exact: true }).click();
	await expect(create.getByText(msg('errors.tag_duplicate.title'))).toBeVisible();
	await create.getByRole('button', { name: msg('common.actions.cancel'), exact: true }).click();
	await expect(page.getByRole('row')).toHaveCount(COLORS.length + 1);
});

test('deleting a tag keeps the links, the count opens the filtered list', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('customer');
	await withApi(user, async (api) => {
		const keep = await apiCreateTag(api, 'keep-me', 'green');
		const drop = await apiCreateTag(api, 'drop-me', 'rose');
		await apiCreateLink(api, {
			url: fixtureUrl(testInfo.testId, '1'),
			description: 'Tagged twice',
			tagIds: [keep.id, drop.id]
		});
		await apiCreateLink(api, {
			url: fixtureUrl(testInfo.testId, '2'),
			description: 'Tagged once',
			tagIds: [keep.id]
		});
		await apiCreateLink(api, { url: fixtureUrl(testInfo.testId, '3'), description: 'Untagged' });
	});
	const page = await openPage(user.storageState);
	await page.goto('/tags');

	await page.getByRole('button', { name: msg('tags.table.delete', { name: 'drop-me' }) }).click();
	const confirm = page.getByRole('alertdialog');
	await expect(confirm).toContainText(msg('tags.delete.text', { name: 'drop-me', count: 1 }));
	await confirm.getByRole('button', { name: msg('tags.delete.confirm'), exact: true }).click();
	await expect(toast(page, msg('tags.delete.deleted'))).toBeVisible();
	await expect(tagRow(page, 'drop-me')).toHaveCount(0);

	await page
		.getByRole('link', { name: msg('tags.table.showLinks', { count: 2, name: 'keep-me' }) })
		.click();
	await expect(page).toHaveURL(/\/\?tags=\d+/);
	await expect
		.poll(async () => [...(await rowLabels(page))].sort())
		.toEqual(['Tagged once', 'Tagged twice']);

	await gotoLinks(page);
	await expect(page.locator('li.kxm-list-row')).toHaveCount(3);
});
