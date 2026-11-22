// Sharing (plan 9.4.2 sharing.spec.ts): a developer shares a link with a customer, who sees it
// read-only with personal pin and tags; a role share reaches every customer; revoking removes
// it. The shared link is deleted at the end, so no other spec sees the role share.
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import {
	apiCreateLink,
	apiCreateTag,
	apiDeleteLink,
	fixtureUrl,
	gotoLinks,
	menuName,
	msg,
	rowAction,
	rowByLabel,
	toast,
	withApi
} from './lib/app.ts';

async function shareDialog(page: Page, label: string) {
	await rowAction(page, label, 'links.actions.share');
	const dialog = page.getByRole('dialog', { name: msg('links.share.title') });
	await expect(dialog).toBeVisible();
	await expect(dialog.getByText(msg('links.share.empty'))).toBeVisible();
	return dialog;
}

test('a customer sees a shared link read-only, with personal pin and tags', async ({
	createUser,
	openPage
}, testInfo) => {
	test.setTimeout(120_000);
	const owner = await createUser('developer', { displayName: 'Share Owner' });
	const recipient = await createUser('customer', { displayName: 'Share Recipient' });
	const other = await createUser('customer', { displayName: 'Other Customer' });
	const link = await withApi(owner, (api) =>
		apiCreateLink(api, { url: fixtureUrl(testInfo.testId, 's'), description: 'Shared runbook' })
	);
	await withApi(recipient, (api) => apiCreateTag(api, 'my-reading', 'rose'));
	try {
		const ownerPage = await openPage(owner.storageState);
		await gotoLinks(ownerPage);
		let dialog = await shareDialog(ownerPage, 'Shared runbook');
		await dialog
			.getByRole('group', { name: msg('links.share.type.label') })
			.getByRole('radio', { name: msg('links.share.type.user'), exact: true })
			.click();
		await dialog.getByLabel(msg('links.share.user'), { exact: true }).click();
		await ownerPage
			.getByRole('combobox', { name: msg('links.share.userSearch') })
			.fill(recipient.username);
		await ownerPage.getByRole('option', { name: new RegExp(recipient.username) }).click();
		await dialog.getByRole('button', { name: msg('links.share.add'), exact: true }).click();
		await expect(toast(ownerPage, msg('links.toast.shared'))).toBeVisible();
		await expect(dialog.getByText('Share Recipient')).toBeVisible();
		await dialog.getByRole('button', { name: msg('common.actions.done'), exact: true }).click();

		const recipientPage = await openPage(recipient.storageState);
		await gotoLinks(recipientPage);
		const row = rowByLabel(recipientPage, 'Shared runbook');
		await expect(row).toBeVisible();
		await expect(row.locator('[data-marker="shared"]')).toContainText(
			msg('links.row.sharedBy', { owner: 'Share Owner' })
		);
		// Read-only: no edit, share or delete in the menu.
		await row.hover();
		await row.getByRole('button', { name: menuName('Shared runbook'), exact: true }).click();
		const menu = recipientPage.getByRole('menu');
		await expect(menu.getByRole('menuitem', { name: msg('links.actions.details') })).toBeVisible();
		for (const key of ['links.actions.edit', 'links.actions.share', 'links.actions.delete']) {
			await expect(menu.getByRole('menuitem', { name: msg(key), exact: true })).toHaveCount(0);
		}
		await recipientPage.keyboard.press('Escape');

		// Personal pin and tag.
		await row.hover();
		await row.getByRole('button', { name: msg('links.actions.pin'), exact: true }).click();
		await expect(toast(recipientPage, msg('links.toast.pinned'))).toBeVisible();
		await rowAction(recipientPage, 'Shared runbook', 'links.actions.tags');
		const tagsDialog = recipientPage.getByRole('dialog', { name: msg('links.tagsDialog.title') });
		await tagsDialog
			.getByRole('button', { name: msg('common.tagPicker.label'), exact: true })
			.click();
		await recipientPage
			.getByRole('listbox', { name: msg('common.tagPicker.label') })
			.getByRole('option', { name: 'my-reading' })
			.click();
		await recipientPage.keyboard.press('Escape');
		await tagsDialog
			.getByRole('button', { name: msg('links.tagsDialog.save'), exact: true })
			.click();
		await expect(toast(recipientPage, msg('links.toast.tagsSaved'))).toBeVisible();
		await expect(row.getByText('my-reading')).toBeVisible();
		await expect(row.locator('[data-marker="pinned"]')).toHaveCount(1);

		// The owner sees neither the recipient's pin nor the tag.
		await ownerPage.reload();
		const ownerRow = rowByLabel(ownerPage, 'Shared runbook');
		await expect(ownerRow).toBeVisible();
		await expect(ownerRow.locator('[data-marker="pinned"]')).toHaveCount(0);
		await expect(ownerRow.getByText('my-reading')).toHaveCount(0);

		// A role share reaches the other customer too.
		const otherPage = await openPage(other.storageState);
		await gotoLinks(otherPage);
		await expect(rowByLabel(otherPage, 'Shared runbook')).toHaveCount(0);
		await rowAction(ownerPage, 'Shared runbook', 'links.actions.share');
		dialog = ownerPage.getByRole('dialog', { name: msg('links.share.title') });
		await dialog
			.getByRole('group', { name: msg('links.share.type.label') })
			.getByRole('radio', { name: msg('links.share.type.role'), exact: true })
			.click();
		await dialog.getByLabel(msg('links.share.role'), { exact: true }).click();
		await ownerPage
			.getByRole('option', { name: msg('roles.plural.customer'), exact: true })
			.click();
		await dialog.getByRole('button', { name: msg('links.share.add'), exact: true }).click();
		const roleText = msg('links.share.roleTarget', { role: msg('roles.plural.customer') });
		await expect(dialog.getByText(roleText)).toBeVisible();
		await otherPage.reload();
		await expect(rowByLabel(otherPage, 'Shared runbook')).toBeVisible();

		// Revoking both shares removes the link for both customers.
		await dialog
			.getByRole('button', { name: msg('links.share.revoke', { target: roleText }), exact: true })
			.click();
		await expect(toast(ownerPage, msg('links.toast.unshared'))).toBeVisible();
		await dialog
			.getByRole('button', {
				name: msg('links.share.revoke', { target: 'Share Recipient' }),
				exact: true
			})
			.click();
		await expect(dialog.getByText(msg('links.share.empty'))).toBeVisible();
		await otherPage.reload();
		await recipientPage.reload();
		await expect(
			recipientPage.getByRole('heading', { level: 1, name: msg('links.pageTitle') })
		).toBeVisible();
		await expect(rowByLabel(otherPage, 'Shared runbook')).toHaveCount(0);
		await expect(rowByLabel(recipientPage, 'Shared runbook')).toHaveCount(0);
	} finally {
		await withApi(owner, (api) => apiDeleteLink(api, link.id));
	}
});
