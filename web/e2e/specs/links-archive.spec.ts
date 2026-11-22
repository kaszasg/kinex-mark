// Archiving (plan 9.4.2 links-archive.spec.ts): manual archive, the "show archived" switch,
// restoring by opening, by the menu and by pinning, the admin's archive run and the archive
// date in the details. Time-based auto-archiving itself is proven by the Go store and jobs
// tests with a fake clock (plan 9.4.2); here only what a browser can drive deterministically.
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import {
	apiCreateLink,
	apiGetLink,
	apiPatchPreferences,
	detailsSheet,
	fixtureUrl,
	gotoLinks,
	huDate,
	msg,
	rowAction,
	rowByLabel,
	toast,
	withApi
} from './lib/app.ts';

const archivedSwitch = (page: Page) =>
	page.getByRole('switch', { name: msg('links.archivedToggle'), exact: true });

const archivedBadge = (page: Page, label: string) =>
	rowByLabel(page, label).locator('[data-marker="archived"]');

test('archived links hide, show with the switch and come back', async ({
	createUser,
	openPage
}, testInfo) => {
	test.setTimeout(120_000);
	const user = await createUser('developer');
	await withApi(user, async (api) => {
		for (const name of ['one', 'two', 'three']) {
			await apiCreateLink(api, {
				url: fixtureUrl(testInfo.testId, name),
				description: `Archive ${name}`
			});
		}
	});
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	await expect(page.getByText(msg('links.summary', { count: 3, pinned: 0 }))).toBeVisible();

	for (const name of ['one', 'two', 'three']) {
		await rowAction(page, `Archive ${name}`, 'links.actions.archive');
		await expect(toast(page, msg('links.toast.archived')).last()).toBeVisible();
		await expect(rowByLabel(page, `Archive ${name}`)).toHaveCount(0);
	}
	await expect(page.getByText(msg('links.empty.collection.title'))).toBeVisible();

	await archivedSwitch(page).click();
	await expect(archivedSwitch(page)).toBeChecked();
	for (const name of ['one', 'two', 'three']) {
		await expect(archivedBadge(page, `Archive ${name}`)).toHaveText(msg('links.row.archived'));
	}
	await expect(
		page.getByText(msg('links.summaryArchived', { count: 3, pinned: 0, archived: 3 }))
	).toBeVisible();

	// Opening restores.
	const popup = page.context().waitForEvent('page');
	await rowByLabel(page, 'Archive one').locator('[data-link-anchor]').click();
	await (await popup).close();
	await expect(archivedBadge(page, 'Archive one')).toHaveCount(0);

	// The menu restores.
	await rowAction(page, 'Archive two', 'links.actions.restore');
	await expect(toast(page, msg('links.toast.restored'))).toBeVisible();
	await expect(archivedBadge(page, 'Archive two')).toHaveCount(0);

	// Pinning restores.
	await rowAction(page, 'Archive three', 'links.actions.pin');
	await expect(toast(page, msg('links.toast.pinnedRestored'))).toBeVisible();
	await expect(archivedBadge(page, 'Archive three')).toHaveCount(0);
	await expect(
		page.getByText(msg('links.summaryArchived', { count: 3, pinned: 1, archived: 0 }))
	).toBeVisible();

	await archivedSwitch(page).click();
	await expect(archivedSwitch(page)).not.toBeChecked();
	await expect(page.locator('li.kxm-list-row')).toHaveCount(3);
	await expect(page.getByText(msg('links.summary', { count: 3, pinned: 1 }))).toBeVisible();
});

test('the undo action of the archive toast puts the link back', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('customer');
	await withApi(user, (api) =>
		apiCreateLink(api, { url: fixtureUrl(testInfo.testId, 'u'), description: 'Undo me' })
	);
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	await rowAction(page, 'Undo me', 'links.actions.archive');
	const archived = toast(page, msg('links.toast.archived'));
	await expect(archived).toBeVisible();
	await expect(rowByLabel(page, 'Undo me')).toHaveCount(0);
	await archived.getByRole('button', { name: msg('links.toast.undo'), exact: true }).click();
	await expect(toast(page, msg('links.toast.restored'))).toBeVisible();
	await expect(rowByLabel(page, 'Undo me')).toBeVisible();
});

test('the details show the archive date of the personal archive period', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	const link = await withApi(user, async (api) => {
		await apiPatchPreferences(api, { archiveAfterDays: 30 });
		const created = await apiCreateLink(api, {
			url: fixtureUrl(testInfo.testId, 'due'),
			description: 'Due later'
		});
		return apiGetLink(api, created.id);
	});
	expect(link.archiveDueAt).not.toBeNull();
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	// The expiry meter of the row names the same date.
	await expect(rowByLabel(page, 'Due later').locator('[data-expiry-slot]')).toBeVisible();

	await rowAction(page, 'Due later', 'links.actions.details');
	const usage = detailsSheet(page).getByTestId('details-usage');
	await expect(usage).toContainText(msg('links.details.metrics.archive'));
	await expect(usage).toContainText(huDate(link.archiveDueAt as string));
	await expect(usage).toContainText(msg('links.details.dueIn', { count: 30 }));

	// A pinned link is never archived.
	await detailsSheet(page)
		.getByRole('button', { name: msg('links.actions.pin'), exact: true })
		.click();
	await expect(usage).toContainText(msg('links.details.never'));
});

test('the admin archive run reports that nothing was due', async ({ asAdmin }) => {
	await asAdmin.goto('/admin/settings');
	const run = asAdmin.getByRole('button', { name: msg('admin.settings.runArchive'), exact: true });
	const answered = asAdmin.waitForResponse(
		(response) => new URL(response.url()).pathname === '/api/admin/jobs/archive'
	);
	await run.click();
	expect((await answered).status()).toBe(200);
	await expect(toast(asAdmin, msg('admin.settings.archived', { count: 0 }))).toBeVisible();
});
