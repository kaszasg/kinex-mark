// Privacy (plan 9.4.2 privacy.spec.ts): the public notice with the controller data, the GDPR data
// export, and account deletion by a developer whose shared link then disappears for the recipient.
import { readFile } from 'node:fs/promises';
import { test, expect } from '../support/fixtures.ts';
import { expectStatus, newApiContext } from '../support/api.ts';
import { API_HEADERS } from '../support/constants.mjs';
import {
	apiCreateLink,
	apiCreateTag,
	fixtureUrl,
	gotoLinks,
	msg,
	rowByLabel,
	withApi
} from './lib/app.ts';

/** Every key of a JSON value, at any depth. */
function allKeys(value: unknown, keys = new Set<string>()): Set<string> {
	if (Array.isArray(value)) {
		for (const item of value) allKeys(item, keys);
	} else if (value !== null && typeof value === 'object') {
		for (const [key, item] of Object.entries(value)) {
			keys.add(key);
			allKeys(item, keys);
		}
	}
	return keys;
}

test('the notice is public and names the controller from the settings', async ({
	page,
	adminApi
}) => {
	const response = await adminApi.get('/api/admin/settings');
	await expectStatus(response, 200, 'read settings');
	const { settings } = (await response.json()) as { settings: Record<string, unknown> };
	if (settings.privacyControllerName !== 'Example Operator Ltd.') {
		const patch = await adminApi.patch('/api/admin/settings', {
			data: { privacyControllerName: 'Example Operator Ltd.' }
		});
		await expectStatus(patch, 200, 'set the controller name');
	}

	await page.goto('/privacy');
	await expect(page).toHaveURL(/\/privacy$/);
	await expect(page.getByRole('heading', { level: 1, name: msg('privacy.title') })).toBeVisible();
	await expect(page.getByText('Example Operator Ltd.')).toBeVisible();
	for (const section of [
		'controller',
		'legalBasis',
		'recipients',
		'transfers',
		'automatedDecisions',
		'adminAccess',
		'rights'
	]) {
		await expect(
			page.getByRole('heading', { name: msg(`privacy.sections.${section}.title`), exact: true })
		).toBeVisible();
	}
	await expect(page.getByText(msg('privacy.rights.access'))).toBeVisible();
	// Without a configured authority the notice names the generic one.
	await expect(
		page.getByText(
			msg('privacy.rights.complaint', { authority: msg('privacy.rights.defaultAuthority') })
		)
	).toBeVisible();
});

test('the data export holds own links and tags and no secrets', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	const url = fixtureUrl(testInfo.testId, 'export');
	await withApi(user, async (api) => {
		const tag = await apiCreateTag(api, 'export-tag', 'ochre');
		await apiCreateLink(api, { url, description: 'Exported link', tagIds: [tag.id] });
	});
	const page = await openPage(user.storageState);
	await page.goto('/settings/privacy');
	const downloadPromise = page.waitForEvent('download');
	await page
		.getByRole('button', { name: msg('security.privacy.export.submit'), exact: true })
		.click();
	const download = await downloadPromise;
	expect(download.suggestedFilename()).toMatch(
		new RegExp(`^kinex-mark-export-${user.username}-\\d{8}\\.json$`)
	);
	const data = JSON.parse(await readFile(await download.path(), 'utf8')) as {
		format: string;
		user: { username: string };
		tags: { name: string }[];
		links: { url: string; tags: string[] }[];
	};
	expect(data.format).toBe('kinex-mark-export/1');
	expect(data.user.username).toBe(user.username);
	expect(data.tags.map((tag) => tag.name)).toEqual(['export-tag']);
	expect(data.links).toHaveLength(1);
	expect(data.links[0]).toMatchObject({ url, tags: ['export-tag'] });
	// No secret: no password hash, TOTP secret, recovery code or session token under any key.
	const keys = [...allKeys(data)];
	expect(keys.filter((key) => /hash|secret|token|^totp$|recovery/i.test(key))).toEqual([]);
});

test('a developer deletes the account; the shared link disappears for the recipient', async ({
	createUser,
	openPage,
	adminApi
}, testInfo) => {
	const started = Date.now() - 1000;
	const developer = await createUser('developer');
	const recipient = await createUser('customer');
	await withApi(developer, async (api) => {
		const link = await apiCreateLink(api, {
			url: fixtureUrl(testInfo.testId, 'gone'),
			description: 'Soon gone'
		});
		const share = await api.post(`/api/links/${link.id}/shares`, {
			data: { targetType: 'user', targetUserId: recipient.id }
		});
		await expectStatus(share, 201, 'share the link');
	});
	const recipientPage = await openPage(recipient.storageState);
	await gotoLinks(recipientPage);
	await expect(rowByLabel(recipientPage, 'Soon gone')).toBeVisible();

	const page = await openPage(developer.storageState);
	await page.goto('/settings/privacy');
	await page.getByRole('button', { name: msg('security.deleteAccount.open'), exact: true }).click();
	const dialog = page.getByRole('alertdialog');
	await expect(
		dialog.getByText(msg('security.deleteAccount.shared.title', { count: 1 }))
	).toBeVisible();
	await dialog
		.getByLabel(msg('security.deleteAccount.password'), { exact: true })
		.fill(developer.password);
	const confirmLabel = msg('security.deleteAccount.confirm', { username: developer.username });
	await dialog.getByLabel(confirmLabel, { exact: true }).fill('someone-else');
	const submit = dialog.getByRole('button', {
		name: msg('security.deleteAccount.submit'),
		exact: true
	});
	// The form keeps the button disabled; the server refuses a wrong name on its own as well.
	await expect(submit).toBeDisabled();
	await withApi(developer, async (api) => {
		const refused = await api.post('/api/me/delete', {
			data: { password: developer.password, confirm: 'someone-else' }
		});
		expect(refused.status()).toBe(400);
		expect(await refused.json()).toMatchObject({ code: 'confirmation_mismatch' });
	});

	await dialog.getByLabel(confirmLabel, { exact: true }).fill(developer.username);
	await submit.click();
	await expect(page).toHaveURL(/\/login\?reason=deleted/);
	await expect(page.getByText(msg('auth.accountDeleted.title'))).toBeVisible();

	const anonymous = await newApiContext();
	try {
		const login = await anonymous.post('/api/auth/login', {
			data: { username: developer.username, password: developer.password },
			headers: { ...API_HEADERS }
		});
		expect(login.status()).toBe(401);
		expect(await login.json()).toMatchObject({ code: 'invalid_credentials' });
	} finally {
		await anonymous.dispose();
	}

	const audit = await adminApi.get('/api/admin/audit?event=user.self_deleted&limit=200');
	await expectStatus(audit, 200, 'read the audit log');
	const entries = ((await audit.json()) as { items: { at: string }[] }).items.filter(
		(entry) => Date.parse(entry.at) >= started
	);
	expect(entries).toHaveLength(1);
	// The entry is pseudonymised: the deleted user name appears nowhere in it.
	expect(JSON.stringify(entries[0])).not.toContain(developer.username);

	await recipientPage.reload();
	await expect(
		recipientPage.getByRole('heading', { level: 1, name: msg('links.pageTitle') })
	).toBeVisible();
	await expect(rowByLabel(recipientPage, 'Soon gone')).toHaveCount(0);
});
