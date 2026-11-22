// Audit log (plan 9.4.2 admin-audit.spec.ts): filters by event group, result, actor and period,
// paging, and localized event names. The entries come from an account made for this test.
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import { expectStatus, newApiContext, signIn } from '../support/api.ts';
import { API_HEADERS } from '../support/constants.mjs';
import { msg } from './lib/app.ts';

/** A day in YYYY-MM-DD form in the Budapest time zone, `offset` days from today. */
function budapestDay(offset = 0): string {
	const date = new Date(Date.now() + offset * 86_400_000);
	return new Intl.DateTimeFormat('en-CA', {
		timeZone: 'Europe/Budapest',
		year: 'numeric',
		month: '2-digit',
		day: '2-digit'
	}).format(date);
}

const bodyRows = (page: Page) => page.getByRole('table').locator('tbody tr');

async function pick(page: Page, labelKey: string, optionName: string): Promise<void> {
	await page.getByLabel(msg(labelKey), { exact: true }).click();
	await page.getByRole('option', { name: optionName, exact: true }).click();
}

test('filters, paging and localized event names', async ({
	asAdmin,
	adminApi,
	createUser,
	expectHttp
}) => {
	test.setTimeout(120_000);
	const user = await createUser('developer');
	// 30 sign-ins and sign-outs give more than one page (50 rows) for this actor alone.
	for (let i = 0; i < 30; i++) {
		const api = await newApiContext();
		try {
			await signIn(api, user.username, user.password);
			await expectStatus(await api.post('/api/auth/logout'), 204, 'sign out');
		} finally {
			await api.dispose();
		}
	}
	// One failed sign-in on the account.
	const failing = await newApiContext();
	try {
		const response = await failing.post('/api/auth/login', {
			data: { username: user.username, password: 'Not-This-Passphrase-1' },
			headers: { ...API_HEADERS }
		});
		expect(response.status()).toBe(401);
	} finally {
		await failing.dispose();
	}

	const page = asAdmin;
	expectHttp(401, '/api/auth/login', 'POST');
	await page.goto('/admin/audit');
	await expect(page.getByRole('heading', { level: 1, name: msg('audit.pageTitle') })).toBeVisible();
	await expect(bodyRows(page).first()).toBeVisible();

	// Actor: the picker searches the user list.
	await page.getByLabel(msg('audit.filters.actor'), { exact: true }).click();
	await page.getByPlaceholder(msg('audit.filters.actorSearch')).fill(user.username);
	await page.getByRole('option', { name: user.username }).click();
	await expect(page).toHaveURL(new RegExp(`actorId=${user.id}`));
	await expect(bodyRows(page)).toHaveCount(50);
	const count = async (query: string) => {
		const response = await adminApi.get(`/api/admin/audit?actorId=${user.id}&limit=1${query}`);
		await expectStatus(response, 200, 'count audit rows');
		return ((await response.json()) as { total: number }).total;
	};
	// 30 sign-ins, 30 sign-outs and a failure, plus what the account setup logged.
	const total = await count('');
	expect(total).toBeGreaterThanOrEqual(61);
	expect(total).toBeLessThan(100);
	await expect(page.getByText(msg('admin.pager.range', { from: 1, to: 50, total }))).toBeVisible();

	// Paging.
	await page.getByRole('button', { name: msg('admin.pager.next'), exact: true }).click();
	await expect(page).toHaveURL(/[?&]page=2/);
	await expect(bodyRows(page)).toHaveCount(total - 50);
	await expect(
		page.getByText(msg('admin.pager.range', { from: 51, to: total, total }))
	).toBeVisible();

	// Event group: sign-ins only (the logouts are auth.* too, the failed attempt as well).
	await pick(page, 'audit.filters.event', msg('audit.filters.groups.auth'));
	await expect(page).toHaveURL(/event=auth\.\*/);
	await expect(page).not.toHaveURL(/[?&]page=/);
	const authTotal = await count('&event=auth.*');
	expect(authTotal).toBeGreaterThanOrEqual(61);
	await expect(
		page.getByText(msg('admin.pager.range', { from: 1, to: 50, total: authTotal }))
	).toBeVisible();

	// Result: the one failure, with its localized name.
	await pick(page, 'audit.filters.result', msg('audit.result.failure'));
	await expect(page).toHaveURL(/success=false/);
	await expect(bodyRows(page)).toHaveCount(1);
	await expect(bodyRows(page).first()).toContainText(msg('audit.events.auth.login.failure'));
	await expect(bodyRows(page).first()).toContainText(msg('audit.result.failure'));

	await pick(page, 'audit.filters.result', msg('audit.result.success'));
	await expect(bodyRows(page).first()).toContainText(
		new RegExp(`${msg('audit.events.auth.login.success')}|${msg('audit.events.auth.logout')}`)
	);

	// Period: today keeps the rows, tomorrow has none.
	await page.getByLabel(msg('audit.filters.from'), { exact: true }).fill(budapestDay());
	await expect(page).toHaveURL(new RegExp(`from=${budapestDay()}`));
	await expect(bodyRows(page)).toHaveCount(50);
	await page.getByLabel(msg('audit.filters.from'), { exact: true }).fill(budapestDay(1));
	await expect(page.getByText(msg('audit.empty.title'))).toBeVisible();

	// The URL holds the filters: a reload keeps them.
	await page.reload();
	await expect(page.getByText(msg('audit.empty.title'))).toBeVisible();
	await page
		.getByRole('button', { name: msg('audit.filters.clear'), exact: true })
		.first()
		.click();
	await expect(page).toHaveURL(/\/admin\/audit$/);
	await expect(bodyRows(page)).toHaveCount(50);
});
