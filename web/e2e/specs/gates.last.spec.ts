// Mandatory steps after sign-in (plan 9.4.2 gates.last.spec.ts, chromium-last project): the
// forced password change of a new account, the privacy notice gate, and the gate that a new
// notice version opens for everyone. Runs last because the version bump touches every user.
import { test, expect, acceptPrivacyNoticeForSeededUsers } from '../support/fixtures.ts';
import { expectStatus, newApiContext } from '../support/api.ts';
import { USER_PASSWORD } from '../support/constants.mjs';
import { msg, signInUi } from './lib/app.ts';

test.afterAll(async () => {
	await acceptPrivacyNoticeForSeededUsers((state) => newApiContext(state));
});

test('a new user changes the generated password and acknowledges the notice', async ({
	asAdmin,
	openPage
}) => {
	const username = `e2e-gate-${Date.now().toString(36)}`;
	const admin = asAdmin;
	await admin.goto('/admin/users');
	await admin.getByRole('button', { name: msg('admin.users.add'), exact: true }).click();
	const dialog = admin.getByRole('dialog');
	await dialog.getByLabel(msg('admin.userDialog.fields.username'), { exact: true }).fill(username);
	await dialog
		.getByLabel(msg('admin.userDialog.fields.displayName'), { exact: true })
		.fill('Gate Tester');
	await dialog.getByRole('button', { name: msg('admin.userDialog.create'), exact: true }).click();

	const tempDialog = admin.getByRole('dialog').filter({ hasText: msg('admin.tempPassword.title') });
	await expect(tempDialog).toBeVisible();
	const temporary = (await tempDialog.getByTestId('temporary-password').innerText()).trim();
	expect(temporary).toHaveLength(16);
	await tempDialog
		.getByRole('button', { name: msg('admin.tempPassword.done'), exact: true })
		.click();

	const page = await openPage();
	await page.goto('/login');
	await signInUi(page, username, temporary);
	await expect(page).toHaveURL(/\/change-password$/);
	await page.getByLabel(msg('auth.passwordChange.current'), { exact: true }).fill(temporary);
	await page.getByLabel(msg('auth.passwordChange.new'), { exact: true }).fill(USER_PASSWORD);
	await page.getByLabel(msg('auth.passwordChange.confirm'), { exact: true }).fill(USER_PASSWORD);
	await page.getByRole('button', { name: msg('auth.passwordChange.submit'), exact: true }).click();

	await expect(page).toHaveURL(/\/consent$/);
	await expect(
		page.getByRole('heading', { level: 1, name: msg('privacy.consent.title') })
	).toBeVisible();
	const submit = page.getByRole('button', { name: msg('privacy.consent.submit'), exact: true });
	await expect(submit).toBeDisabled();
	// The direct path to the app stays closed while the gate is open.
	await page.goto('/tags');
	await expect(page).toHaveURL(/\/consent$/);
	await page.getByLabel(msg('privacy.consent.checkbox')).check();
	await submit.click();
	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();
});

test('a new notice version sends the next request of a signed-in user to the gate', async ({
	createUser,
	openPage,
	adminApi,
	expectHttp
}) => {
	const user = await createUser('developer');
	const page = await openPage(user.storageState);
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();

	const bump = await adminApi.post('/api/admin/settings/privacy-notice/bump');
	await expectStatus(bump, 200, 'bump the privacy notice');
	// The seeded users acknowledge the new version at once, so later specs are not gated.
	await acceptPrivacyNoticeForSeededUsers((state) => newApiContext(state));

	expectHttp(403, /^\/api\//);
	await page
		.getByRole('group', { name: msg('links.toolbar.scope') })
		.getByRole('radio', { name: msg('links.scope.own') })
		.click();
	await expect(page).toHaveURL(/\/consent$/);

	// The full notice opens in a new tab without leaving the gate.
	const popupPromise = page.context().waitForEvent('page');
	await page.getByRole('link', { name: new RegExp(msg('privacy.consent.openFull')) }).click();
	const popup = await popupPromise;
	await expect(popup).toHaveURL(/\/privacy$/);
	await expect(popup.getByRole('heading', { level: 1, name: msg('privacy.title') })).toBeVisible();
	await popup.close();
});

// Regression (phase E finding E1): the SPA had cached the public configuration before the bump,
// so the gate showed the old notice version and the first acknowledgement was refused with 409
// consent_version_mismatch. The gate now reloads the notice; one acknowledgement is enough.
test('the gate shows the new notice version and one acknowledgement opens the app', async ({
	createUser,
	openPage,
	adminApi,
	expectHttp
}) => {
	const user = await createUser('developer');
	const page = await openPage(user.storageState);
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();

	const bump = await adminApi.post('/api/admin/settings/privacy-notice/bump');
	await expectStatus(bump, 200, 'bump the privacy notice');
	const version = ((await bump.json()) as { settings: { privacyNoticeVersion: number } }).settings
		.privacyNoticeVersion;
	await acceptPrivacyNoticeForSeededUsers((state) => newApiContext(state));

	expectHttp(403, /^\/api\//);
	await page
		.getByRole('group', { name: msg('links.toolbar.scope') })
		.getByRole('radio', { name: msg('links.scope.own') })
		.click();
	await expect(page).toHaveURL(/\/consent$/);
	await expect(page.locator('[data-notice-version]')).toContainText(
		msg('privacy.version', { version })
	);
	await page.getByLabel(msg('privacy.consent.checkbox')).check();
	await page.getByRole('button', { name: msg('privacy.consent.submit'), exact: true }).click();
	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();
});
