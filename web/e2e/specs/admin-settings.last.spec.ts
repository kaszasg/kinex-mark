// System settings (plan 9.4.2 admin-settings.last.spec.ts, chromium-last project): a range error
// and a successful save, the environment-locked private fetch switch, the controller data on
// /privacy, and a privacy notice version bump. The seeded users acknowledge the new notice again.
import { test, expect, acceptPrivacyNoticeForSeededUsers } from '../support/fixtures.ts';
import { expectStatus, newApiContext } from '../support/api.ts';
import { msg, toast } from './lib/app.ts';

test.afterAll(async () => {
	await acceptPrivacyNoticeForSeededUsers((state) => newApiContext(state));
});

const CONTROLLER = {
	privacyControllerName: 'Settings Spec Operator Kft.',
	privacyControllerAddress: '1 Example Street, Budapest',
	privacyControllerEmail: 'privacy@example.test',
	privacySupervisoryAuthority: 'Example Data Protection Authority, dpa.example'
};

test('a value out of range is refused, valid values are saved and shown on /privacy', async ({
	asAdmin,
	adminApi,
	browser
}) => {
	const page = asAdmin;
	await page.goto('/admin/settings');
	await expect(
		page.getByRole('heading', { level: 1, name: msg('admin.settings.pageTitle') })
	).toBeVisible();
	const save = page.getByRole('button', { name: msg('admin.settings.save'), exact: true });

	const idle = page.getByLabel(msg('admin.settings.fields.sessionIdleMinutes'), { exact: true });
	await idle.fill('5');
	await expect(page.getByText(msg('admin.settings.unsaved'))).toBeVisible();
	await save.click();
	await expect(page.getByText(msg('fields.range', { min: 15, max: 10080 }))).toBeVisible();
	// The server checks the range on its own as well.
	const refused = await adminApi.patch('/api/admin/settings', { data: { sessionIdleMinutes: 5 } });
	expect(refused.status()).toBe(400);
	expect(await refused.json()).toMatchObject({
		code: 'validation_failed',
		fields: { sessionIdleMinutes: 'out_of_range' }
	});
	await page.getByRole('button', { name: msg('admin.settings.discard'), exact: true }).click();
	await expect(page.getByText(msg('admin.settings.noChanges'))).toBeVisible();

	await page
		.getByLabel(msg('admin.settings.fields.auditRetentionDays'), { exact: true })
		.fill('400');
	for (const [key, value] of Object.entries(CONTROLLER)) {
		await page.getByLabel(msg(`admin.settings.fields.${key}`), { exact: true }).fill(value);
	}
	await save.click();
	await expect(toast(page, msg('admin.settings.saved'))).toBeVisible();
	await page.reload();
	await expect(
		page.getByLabel(msg('admin.settings.fields.auditRetentionDays'), { exact: true })
	).toHaveValue('400');
	const stored = (await (await adminApi.get('/api/admin/settings')).json()) as {
		settings: Record<string, unknown>;
	};
	expect(stored.settings).toMatchObject({ auditRetentionDays: 400, ...CONTROLLER });

	const anonymous = await browser.newContext();
	try {
		const notice = await anonymous.newPage();
		await notice.goto('/privacy');
		for (const value of Object.values(CONTROLLER)) {
			await expect(notice.getByText(value, { exact: false }).first()).toBeVisible();
		}
	} finally {
		await anonymous.close();
	}
});

test('the private fetch switch is locked by the environment', async ({ asAdmin, adminApi }) => {
	const response = await adminApi.get('/api/admin/settings');
	await expectStatus(response, 200, 'read settings');
	const body = (await response.json()) as { lockedByEnv: string[] };
	expect(body.lockedByEnv).toContain('allowPrivateFetch');
	const patch = await adminApi.patch('/api/admin/settings', { data: { allowPrivateFetch: false } });
	expect(patch.status()).toBe(409);
	expect(await patch.json()).toMatchObject({ code: 'setting_locked' });

	const page = asAdmin;
	await page.goto('/admin/settings');
	await expect(
		page.getByRole('switch', { name: msg('admin.settings.fields.allowPrivateFetch') })
	).toBeDisabled();
	await expect(page.getByText(msg('admin.settings.locked.title'))).toBeVisible();
	await expect(page.getByText(msg('admin.settings.locked.privateFetch'))).toBeVisible();
});

test('a new notice version is issued after a confirmation', async ({
	asAdmin,
	adminApi,
	expectHttp
}) => {
	const before = (await (await adminApi.get('/api/admin/settings')).json()) as {
		settings: { privacyNoticeVersion: number };
	};
	const page = asAdmin;
	await page.goto('/admin/settings');
	await expect(page.getByTestId('notice-version')).toContainText(
		String(before.settings.privacyNoticeVersion)
	);
	// After the bump the admin's own next request may meet the consent gate.
	expectHttp(403, /^\/api\//);
	await page.getByRole('button', { name: msg('admin.settings.notice.bump'), exact: true }).click();
	const confirm = page.getByRole('alertdialog');
	await expect(confirm).toContainText(msg('admin.settings.notice.bumpTitle'));
	await confirm
		.getByRole('button', { name: msg('admin.settings.notice.bump'), exact: true })
		.click();
	const next = before.settings.privacyNoticeVersion + 1;
	await expect(toast(page, msg('admin.settings.notice.bumped', { version: next }))).toBeVisible();
	await expect(page.getByTestId('notice-version')).toContainText(String(next));
	await acceptPrivacyNoticeForSeededUsers((state) => newApiContext(state));
});
