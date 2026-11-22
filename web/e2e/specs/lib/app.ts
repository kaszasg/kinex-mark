// Shared helpers of the E2E specs (plan 9.4.2): API shortcuts that prepare data for the user under
// test, catalogue lookups in every locale, and the accessible names of recurring UI parts. The
// support layer (../../support) stays the single source of the harness; this module only builds
// on it. Specs keep UI texts out of their source by resolving them from the catalogues.
import { fileURLToPath, pathToFileURL } from 'node:url';
import { expect, type APIRequestContext, type Locator, type Page } from '@playwright/test';
import { expectStatus, newApiContext, type StorageState } from '../../support/api.ts';
import { FIXTURE_ORIGIN } from '../../support/constants.mjs';
import { type CreatedUser } from '../../support/fixtures.ts';
import { msg, type MessageParams } from '../../support/i18n.ts';

export { msg };

export type UiLocale = 'hu' | 'en' | 'de';

type Plural = { zero?: string; one: string; other: string };
type Node = string | Plural | { [key: string]: Node };

async function loadCatalogue(locale: 'en' | 'de'): Promise<Record<string, Node>> {
	const file = fileURLToPath(
		new URL(`../../../src/lib/i18n/messages/${locale}/index.ts`, import.meta.url)
	);
	const module = (await import(pathToFileURL(file).href)) as { default: Record<string, Node> };
	return module.default;
}

const catalogues: Record<'en' | 'de', Record<string, Node>> = {
	en: await loadCatalogue('en'),
	de: await loadCatalogue('de')
};

const NUMBER_LOCALE = { en: 'en-GB', de: 'de-DE' } as const;

/** The text of a message key in the given locale, with placeholders and plurals resolved. */
export function msgIn(locale: UiLocale, key: string, params: MessageParams = {}): string {
	if (locale === 'hu') return msg(key, params);
	let node: Node | undefined = catalogues[locale];
	for (const part of key.split('.')) {
		node = typeof node === 'object' && !('one' in node && 'other' in node) ? node[part] : undefined;
		if (node === undefined) throw new Error(`unknown message key: ${key} (${locale})`);
	}
	let text: string;
	if (typeof node === 'string') {
		text = node;
	} else if ('one' in node && 'other' in node && typeof node.one === 'string') {
		const plural = node as Plural;
		const count = Number(params.count);
		text =
			count === 0 && plural.zero !== undefined
				? plural.zero
				: new Intl.PluralRules(locale).select(count) === 'one'
					? plural.one
					: plural.other;
	} else {
		throw new Error(`message key ${key} is a namespace`);
	}
	const numbers = new Intl.NumberFormat(NUMBER_LOCALE[locale]);
	return text.replace(/\{(\w+)\}/g, (match, name: string) => {
		if (!(name in params)) return match;
		const value = params[name];
		return typeof value === 'number' ? numbers.format(value).replace(/[  ]/g, ' ') : value;
	});
}

/** Escapes a text for use inside a RegExp. */
export function escapeRegExp(text: string): string {
	return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** A RegExp that matches the exact text (whole string). */
export function exactText(text: string): RegExp {
	return new RegExp(`^\\s*${escapeRegExp(text)}\\s*$`);
}

// ---------------------------------------------------------------------------------------------
// Fixture origin
// ---------------------------------------------------------------------------------------------

export const ALPHA_URL = `${FIXTURE_ORIGIN}/alpha`;
export const BETA_URL = `${FIXTURE_ORIGIN}/beta`;
export const PLAIN_URL = `${FIXTURE_ORIGIN}/plain`;
/** Title of the /alpha fixture page. */
export const ALPHA_TITLE = 'Alpha docs';

/** The title the fixture origin serves for a /p/<id> URL. */
export function fixtureTitle(url: string): string {
	return `Fixture ${new URL(url).pathname.split('/').pop()}`;
}

/** A fixture URL that is unique per call (several links of one test). */
export function fixtureUrl(testId: string, suffix = 'main'): string {
	const id = `${testId}-${suffix}`.replace(/[^A-Za-z0-9._-]/g, '-');
	return `${FIXTURE_ORIGIN}/p/${id}`;
}

// ---------------------------------------------------------------------------------------------
// API shortcuts for the signed-in user under test
// ---------------------------------------------------------------------------------------------

export interface ApiLink {
	id: number;
	url: string;
	title: string;
	description: string;
	pinned: boolean;
	useCount: number;
	archivedAt: string | null;
	archiveDueAt: string | null;
	fetchStatus: string;
	tagIds: number[];
}

export interface ApiTag {
	id: number;
	name: string;
	color: string;
}

/** API context signed in as the given user (storage state of createUser or a state file). */
export function apiAs(state: string | StorageState): Promise<APIRequestContext> {
	return newApiContext(state);
}

/** Runs fn with an API context of the user and disposes it afterwards. */
export async function withApi<T>(
	user: CreatedUser | string,
	fn: (api: APIRequestContext) => Promise<T>
): Promise<T> {
	const api = await apiAs(typeof user === 'string' ? user : user.storageState);
	try {
		return await fn(api);
	} finally {
		await api.dispose();
	}
}

export async function apiCreateLink(
	api: APIRequestContext,
	link: { url: string; title?: string; description?: string; tagIds?: number[]; pinned?: boolean }
): Promise<ApiLink> {
	const response = await api.post('/api/links', { data: link });
	await expectStatus(response, 201, `create link ${link.url}`);
	return (await response.json()) as ApiLink;
}

export async function apiGetLink(api: APIRequestContext, id: number): Promise<ApiLink> {
	const response = await api.get(`/api/links/${id}`);
	await expectStatus(response, 200, `get link ${id}`);
	return (await response.json()) as ApiLink;
}

export async function apiDeleteLink(api: APIRequestContext, id: number): Promise<void> {
	const response = await api.delete(`/api/links/${id}`);
	await expectStatus(response, [204, 404], `delete link ${id}`);
}

export async function apiCreateTag(
	api: APIRequestContext,
	name: string,
	color = 'neutral'
): Promise<ApiTag> {
	const response = await api.post('/api/tags', { data: { name, color } });
	await expectStatus(response, 201, `create tag ${name}`);
	return (await response.json()) as ApiTag;
}

export async function apiVisit(api: APIRequestContext, id: number, times = 1): Promise<void> {
	for (let i = 0; i < times; i++) {
		const response = await api.post(`/api/links/${id}/visit`);
		await expectStatus(response, 200, `visit link ${id}`);
	}
}

export async function apiPatchPreferences(
	api: APIRequestContext,
	patch: Record<string, unknown>
): Promise<void> {
	const response = await api.patch('/api/me/preferences', { data: patch });
	await expectStatus(response, 200, 'update preferences');
}

/** Waits until the server has finished the metadata fetch of a link. */
export async function waitFetched(api: APIRequestContext, id: number): Promise<ApiLink> {
	let link = await apiGetLink(api, id);
	await expect
		.poll(
			async () => {
				link = await apiGetLink(api, id);
				return link.fetchStatus;
			},
			{ timeout: 20_000, intervals: [250, 500, 1000] }
		)
		.not.toBe('pending');
	return link;
}

// ---------------------------------------------------------------------------------------------
// UI
// ---------------------------------------------------------------------------------------------

/** Accessible names of the add-link flow for support/helpers.ts addLink(). */
export function addLinkNames(): {
	open: string;
	url: string;
	title: string;
	description: string;
	submit: string;
} {
	return {
		open: msg('links.toolbar.add'),
		url: msg('links.dialog.url'),
		title: msg('links.dialog.title'),
		description: msg('links.dialog.description'),
		submit: msg('links.dialog.save')
	};
}

/** Opens the links page and waits until the list (or the empty state) has rendered. */
export async function gotoLinks(page: Page, path = '/'): Promise<void> {
	await page.goto(path);
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();
	await expect(page.locator('[aria-busy="true"]')).toHaveCount(0);
}

/** The labels of the list rows in display order (the anchor text of every row). */
export function rowLabels(page: Page): Promise<string[]> {
	return page
		.locator('li.kxm-list-row [data-link-anchor]')
		.evaluateAll((anchors) => anchors.map((anchor) => (anchor.textContent ?? '').trim()));
}

/** The row-menu button name of a link label. */
export function menuName(label: string): string {
	return msg('links.row.menu', { label });
}

/** Opens the action menu of the row whose label is given and clicks one item. */
export async function rowAction(page: Page, label: string, actionKey: string): Promise<void> {
	const row = page.locator('li.kxm-list-row').filter({
		has: page.locator('[data-link-anchor]', { hasText: exactText(label) })
	});
	await row.hover();
	await row.getByRole('button', { name: menuName(label), exact: true }).click();
	const menu = page.getByRole('menu');
	await expect(menu).toBeVisible();
	await menu.getByRole('menuitem', { name: msg(actionKey), exact: true }).click();
}

/** The list row whose anchor text is exactly the label. */
export function rowByLabel(page: Page, label: string): Locator {
	return page.locator('li.kxm-list-row').filter({
		has: page.locator('[data-link-anchor]', { hasText: exactText(label) })
	});
}

/** The visible toast with the given text. */
export function toast(page: Page, text: string | RegExp): Locator {
	return page.locator('[data-sonner-toast]').filter({ hasText: text }).first();
}

/** The details sheet (dialog) of a link. */
export function detailsSheet(page: Page): Locator {
	return page.getByRole('dialog').filter({ has: page.locator('[data-testid="details-usage"]') });
}

/** The use count shown in the details sheet's usage metric. */
export async function detailsUseCount(page: Page): Promise<number> {
	const metric = detailsSheet(page).locator('[data-testid="details-usage"] .data-lg').first();
	return Number((await metric.innerText()).replace(/\D/g, ''));
}

/** Signs in through the sign-in form (no second factor). */
export async function signInUi(page: Page, username: string, password: string): Promise<void> {
	await page.getByLabel(msg('auth.login.username'), { exact: true }).fill(username);
	await page.getByLabel(msg('auth.login.password'), { exact: true }).fill(password);
	await page.getByRole('button', { name: msg('auth.login.submit'), exact: true }).click();
}

/** Opens the sidebar user menu of the signed-in user. */
export async function openUserMenu(page: Page, locale: UiLocale = 'hu'): Promise<Locator> {
	const suffix = msgIn(locale, 'nav.userMenu').split('{role}')[1] ?? '';
	await page.getByRole('button', { name: new RegExp(`${escapeRegExp(suffix)}$`) }).click();
	const menu = page.getByRole('menu');
	await expect(menu).toBeVisible();
	return menu;
}

/** Signs out through the user menu and waits for the sign-in page. */
export async function signOutUi(page: Page): Promise<void> {
	const menu = await openUserMenu(page);
	await menu.getByRole('menuitem', { name: msg('nav.logout') }).click();
	await expect(page).toHaveURL(/\/login(\?|$)/);
}

/** Formats a date like fmt.date('hu') in Europe/Budapest (yyyy. mm. dd.). */
export function huDate(value: string | Date): string {
	return new Intl.DateTimeFormat('hu', {
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		timeZone: 'Europe/Budapest'
	})
		.format(new Date(value))
		.replace(/[  ]/g, ' ');
}
