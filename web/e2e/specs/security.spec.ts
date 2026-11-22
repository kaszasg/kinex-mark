// Security and role boundaries (plan 9.4.2 security.spec.ts): CSP on every page response, 401
// signed out, customer and developer restrictions with direct navigation to /admin, IDOR from the
// browser, same-origin requests only, and the 404 page.
import type { Page, Response } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import { newApiContext } from '../support/api.ts';
import { BASE_URL } from '../support/constants.mjs';
import { apiCreateLink, fixtureUrl, gotoLinks, menuName, msg, withApi } from './lib/app.ts';

/** Collects the document responses of a page, to check their CSP header. */
function collectDocuments(page: Page): Response[] {
	const documents: Response[] = [];
	page.on('response', (response) => {
		if (response.request().resourceType() === 'document') documents.push(response);
	});
	return documents;
}

function scriptSrc(csp: string): string {
	const directive = csp
		.split(';')
		.map((part) => part.trim())
		.find((part) => part.startsWith('script-src'));
	return directive ?? '';
}

function sidebar(page: Page) {
	return page.getByRole('navigation', { name: msg('nav.sidebarTitle') });
}

test('every page carries a CSP without unsafe-inline scripts', async ({ asAdmin, page }) => {
	const anonymousDocs = collectDocuments(page);
	await page.goto('/login');
	await page.goto('/privacy');
	const adminDocs = collectDocuments(asAdmin);
	for (const path of [
		'/',
		'/tags',
		'/settings/profile',
		'/settings/display',
		'/settings/security',
		'/settings/privacy',
		'/admin/users',
		'/admin/settings',
		'/admin/audit'
	]) {
		await asAdmin.goto(path);
		await expect(asAdmin.getByRole('heading', { level: 1 })).toBeVisible();
	}
	const documents = [...anonymousDocs, ...adminDocs];
	expect(documents.length).toBeGreaterThanOrEqual(11);
	for (const response of documents) {
		const csp = (await response.allHeaders())['content-security-policy'] ?? '';
		expect(csp, response.url()).toContain("default-src 'self'");
		const scripts = scriptSrc(csp);
		expect(scripts, response.url()).not.toBe('');
		expect(scripts, response.url()).not.toContain("'unsafe-inline'");
	}
});

test('signed out, the API answers 401', async () => {
	const api = await newApiContext();
	try {
		for (const path of ['/api/links', '/api/tags', '/api/me/preferences', '/api/admin/users']) {
			const response = await api.get(path);
			expect(response.status(), path).toBe(401);
			expect(await response.json()).toMatchObject({ code: 'unauthenticated' });
		}
	} finally {
		await api.dispose();
	}
});

test('a customer has no admin menu, no admin pages and no user directory', async ({
	createUser,
	openPage,
	expectHttp
}) => {
	const user = await createUser('customer');
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	await expect(
		sidebar(page).getByRole('link', { name: msg('nav.links'), exact: true })
	).toBeVisible();
	await expect(sidebar(page).getByText(msg('nav.groups.admin'), { exact: true })).toHaveCount(0);
	await expect(sidebar(page).getByRole('link', { name: msg('nav.adminUsers') })).toHaveCount(0);

	for (const path of ['/admin/users', '/admin', '/admin/settings']) {
		await page.goto(path);
		await expect(
			page.getByRole('heading', { level: 1, name: msg('nav.errorPage.forbidden.title') })
		).toBeVisible();
		await expect(page.getByText('403', { exact: true })).toBeVisible();
	}

	expectHttp(403, '/api/users/directory', 'GET');
	const status = await page.evaluate(async () => {
		const response = await fetch('/api/users/directory', {
			headers: { 'X-Kinex-Request': '1', Accept: 'application/json' }
		});
		return response.status;
	});
	expect(status).toBe(403);
});

test('a developer has no admin menu but can share', async ({ createUser, openPage }, testInfo) => {
	const user = await createUser('developer');
	await withApi(user, (api) =>
		apiCreateLink(api, { url: fixtureUrl(testInfo.testId), description: 'Shareable' })
	);
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	await expect(sidebar(page).getByText(msg('nav.groups.admin'), { exact: true })).toHaveCount(0);
	const row = page.locator('li.kxm-list-row').first();
	await row.hover();
	await row.getByRole('button', { name: menuName('Shareable'), exact: true }).click();
	await expect(
		page.getByRole('menuitem', { name: msg('links.actions.share'), exact: true })
	).toBeVisible();
	await page.keyboard.press('Escape');

	await page.goto('/admin/audit');
	await expect(
		page.getByRole('heading', { level: 1, name: msg('nav.errorPage.forbidden.title') })
	).toBeVisible();
});

test('an admin sees the admin menu group', async ({ asAdmin }) => {
	await gotoLinks(asAdmin);
	const nav = sidebar(asAdmin);
	await expect(nav.getByText(msg('nav.groups.admin'), { exact: true })).toBeVisible();
	for (const key of ['nav.adminUsers', 'nav.adminSettings', 'nav.adminAudit']) {
		await expect(nav.getByRole('link', { name: msg(key), exact: true })).toBeVisible();
	}
});

test('another user link is not found from the browser', async ({
	createUser,
	openPage,
	expectHttp
}, testInfo) => {
	const owner = await createUser('developer');
	const intruder = await createUser('developer');
	const link = await withApi(owner, (api) =>
		apiCreateLink(api, { url: fixtureUrl(testInfo.testId), description: 'Private' })
	);
	const page = await openPage(intruder.storageState);
	await gotoLinks(page);
	expectHttp(404, new RegExp(`^/api/links/${link.id}(/.*)?$`));
	const statuses = await page.evaluate(async (id) => {
		const headers = { 'X-Kinex-Request': '1', Accept: 'application/json' };
		const results: number[] = [];
		results.push((await fetch(`/api/links/${id}`, { headers })).status);
		results.push(
			(
				await fetch(`/api/links/${id}`, {
					method: 'PATCH',
					headers: { ...headers, 'Content-Type': 'application/json' },
					body: JSON.stringify({ description: 'Taken over' })
				})
			).status
		);
		results.push((await fetch(`/api/links/${id}`, { method: 'DELETE', headers })).status);
		results.push((await fetch(`/api/links/${id}/shares`, { headers })).status);
		return results;
	}, link.id);
	expect(statuses).toEqual([404, 404, 404, 404]);
	const unchanged = await withApi(owner, async (api) =>
		(await api.get(`/api/links/${link.id}`)).json()
	);
	expect(unchanged).toMatchObject({ description: 'Private' });
});

test('the app only talks to its own origin', async ({ createUser, openPage }, testInfo) => {
	const user = await createUser('developer');
	await withApi(user, (api) =>
		apiCreateLink(api, { url: fixtureUrl(testInfo.testId), description: 'External page' })
	);
	const page = await openPage(user.storageState);
	const foreign: string[] = [];
	page.on('request', (request) => {
		const url = request.url();
		if (url.startsWith('data:') || url.startsWith('blob:')) return;
		if (new URL(url).origin !== BASE_URL) foreign.push(url);
	});
	for (const path of ['/', '/tags', '/settings/display', '/settings/security', '/privacy']) {
		await page.goto(path);
		await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	}
	await page.goto('/');
	await expect(page.locator('li.kxm-list-row img').first()).toBeVisible({ timeout: 20_000 });
	expect(foreign).toEqual([]);
});

test('an unknown path shows the 404 page inside the app', async ({ asDeveloper }) => {
	const page = asDeveloper;
	await page.goto('/no-such-page');
	await expect(
		page.getByRole('heading', { level: 1, name: msg('nav.errorPage.notFound.title') })
	).toBeVisible();
	await expect(page.getByText('404', { exact: true })).toBeVisible();
	await page.getByRole('link', { name: msg('nav.errorPage.notFound.action'), exact: true }).click();
	await expect(page).toHaveURL(`${BASE_URL}/`);
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();
});
