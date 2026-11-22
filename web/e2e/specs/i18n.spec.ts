// Languages (plan 9.4.2 i18n.spec.ts): hu/en/de on the sign-in page without reload, kept after a
// reload, saved to the server after sign-in, the switch in the app top bar and in admin, and the
// date and number formats per language.
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import { BASE_URL } from '../support/constants.mjs';
import { gotoLinks, msgIn, withApi, type UiLocale } from './lib/app.ts';

const NAMES: Record<UiLocale, string> = { hu: 'Magyar', en: 'English', de: 'Deutsch' };

async function expectLoginIn(page: Page, locale: UiLocale): Promise<void> {
	await expect(page.locator('html')).toHaveAttribute('lang', locale);
	await expect(
		page.getByRole('heading', { level: 1, name: msgIn(locale, 'auth.login.title'), exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: msgIn(locale, 'auth.login.submit'), exact: true })
	).toBeVisible();
	await expect(
		page.getByLabel(msgIn(locale, 'auth.login.username'), { exact: true })
	).toBeVisible();
	await expect(page).toHaveTitle(new RegExp(msgIn(locale, 'auth.login.title')));
}

async function pickOnLogin(page: Page, from: UiLocale, to: UiLocale): Promise<void> {
	await page
		.getByRole('radiogroup', { name: msgIn(from, 'nav.locale.label') })
		.or(page.getByRole('group', { name: msgIn(from, 'nav.locale.label') }))
		.getByRole('radio', { name: NAMES[to], exact: true })
		.click();
}

/** Picks a language in the top bar menu. */
async function pickInTopbar(page: Page, from: UiLocale, to: UiLocale): Promise<void> {
	await page.getByRole('button', { name: msgIn(from, 'nav.locale.label'), exact: true }).click();
	await page.getByRole('menuitemradio', { name: NAMES[to], exact: true }).click();
	await expect(page.locator('html')).toHaveAttribute('lang', to);
}

function preferencesSaved(page: Page) {
	return page.waitForResponse(
		(response) =>
			response.request().method() === 'PATCH' &&
			new URL(response.url()).pathname === '/api/me/preferences' &&
			response.status() === 200
	);
}

/** The date format of the app for one language (format.ts), with no-break spaces. */
function appDate(locale: UiLocale, value: string): string {
	const options: Intl.DateTimeFormatOptions =
		locale === 'en'
			? { day: 'numeric', month: 'short', year: 'numeric' }
			: { year: 'numeric', month: '2-digit', day: '2-digit' };
	const tag = { hu: 'hu', en: 'en-GB', de: 'de-DE' }[locale];
	return new Intl.DateTimeFormat(tag, { ...options, timeZone: 'Europe/Budapest' })
		.format(new Date(value))
		.replace(/[   ]/g, ' ');
}

test('the sign-in page switches between hu, en and de without reload', async ({ page }) => {
	await page.goto('/login');
	await expectLoginIn(page, 'hu');
	const marker = await page.evaluate(
		() => ((window as unknown as { kxmMark: number }).kxmMark = 1)
	);
	expect(marker).toBe(1);

	await pickOnLogin(page, 'hu', 'en');
	await expectLoginIn(page, 'en');
	await pickOnLogin(page, 'en', 'de');
	await expectLoginIn(page, 'de');
	// Same document: no reload happened.
	expect(await page.evaluate(() => (window as unknown as { kxmMark?: number }).kxmMark)).toBe(1);

	await page.reload();
	await expectLoginIn(page, 'de');
	await pickOnLogin(page, 'de', 'hu');
	await expectLoginIn(page, 'hu');
});

test('the language chosen before sign-in is saved to the account', async ({
	page,
	createUser,
	openPage
}) => {
	const user = await createUser('developer');
	await page.goto('/login');
	await pickOnLogin(page, 'hu', 'en');
	await expectLoginIn(page, 'en');
	await page.getByLabel(msgIn('en', 'auth.login.username'), { exact: true }).fill(user.username);
	await page.getByLabel(msgIn('en', 'auth.login.password'), { exact: true }).fill(user.password);
	await page.getByRole('button', { name: msgIn('en', 'auth.login.submit'), exact: true }).click();
	await expect(page).toHaveURL(`${BASE_URL}/`);
	await expect(
		page.getByRole('heading', { level: 1, name: msgIn('en', 'links.pageTitle'), exact: true })
	).toBeVisible();
	await expect
		.poll(() =>
			withApi(
				user,
				async (api) =>
					((await (await api.get('/api/me/preferences')).json()) as { locale: string }).locale
			)
		)
		.toBe('en');

	// A fresh browser profile (Hungarian browser locale, empty storage) follows the account.
	const fresh = await openPage(user.storageState);
	await fresh.goto('/');
	await expect(fresh.locator('html')).toHaveAttribute('lang', 'en');
	await expect(
		fresh.getByRole('heading', { level: 1, name: msgIn('en', 'links.pageTitle'), exact: true })
	).toBeVisible();
});

test('the top bar switches the language in the app and in admin', async ({ asAdmin }) => {
	const page = asAdmin;
	await gotoLinks(page);
	let saved = preferencesSaved(page);
	await pickInTopbar(page, 'hu', 'de');
	await expect(
		page.getByRole('heading', { level: 1, name: msgIn('de', 'links.pageTitle'), exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('navigation', { name: msgIn('de', 'nav.sidebarTitle'), exact: true })
	).toBeVisible();
	await saved;

	// German keyboards label the keys Strg and Entf; the shortcuts table follows the language.
	await page
		.getByRole('button', { name: msgIn('de', 'links.toolbar.display'), exact: true })
		.click();
	await page
		.getByRole('menuitem', { name: msgIn('de', 'links.toolbar.shortcuts'), exact: true })
		.click();
	const shortcuts = page.getByRole('dialog', { name: msgIn('de', 'links.shortcuts.title') });
	await expect(shortcuts).toBeVisible();
	const keys = (await shortcuts.locator('kbd[data-slot="kbd"]').allInnerTexts()).map((key) =>
		key.trim()
	);
	for (const key of ['ctrl', 'delete', 'enter', 'esc']) {
		expect(keys).toContain(msgIn('de', `common.keys.${key}`));
	}
	expect(keys).not.toContain(msgIn('en', 'common.keys.ctrl'));
	expect(keys).not.toContain(msgIn('en', 'common.keys.delete'));
	await page.keyboard.press('Escape');
	await expect(shortcuts).toBeHidden();

	await page.goto('/admin/users');
	await expect(page.locator('html')).toHaveAttribute('lang', 'de');
	await expect(
		page.getByRole('heading', { level: 1, name: msgIn('de', 'admin.users.pageTitle'), exact: true })
	).toBeVisible();
	saved = preferencesSaved(page);
	await pickInTopbar(page, 'de', 'en');
	await expect(
		page.getByRole('heading', { level: 1, name: msgIn('en', 'admin.users.pageTitle'), exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('columnheader', { name: msgIn('en', 'admin.users.table.role'), exact: true })
	).toBeVisible();
	await saved;
});

test('dates and numbers follow the language', async ({ asAdmin, adminApi }) => {
	const me = (await (await adminApi.get('/api/auth/me')).json()) as { createdAt: string };
	expect(me.createdAt).toBeTruthy();
	const page = asAdmin;
	const numbers: Record<UiLocale, string> = { hu: '10 080', en: '10,080', de: '10.080' };
	let current: UiLocale = 'hu';
	for (const locale of ['hu', 'en', 'de'] as UiLocale[]) {
		if (locale !== current) {
			// The choice is saved after a short debounce; the next navigation waits for it.
			const saved = preferencesSaved(page);
			await pickInTopbar(page, current, locale);
			await saved;
			current = locale;
		}
		await page.goto('/settings/profile');
		await expect(page.locator('html')).toHaveAttribute('lang', locale);
		// The profile shows the creation time as date and minute; the date part is per language.
		await expect(page.locator(`time[datetime="${me.createdAt}"]`)).toContainText(
			appDate(locale, me.createdAt)
		);
		await page.goto('/admin/settings');
		await expect(
			page.getByText(msgIn(locale, 'admin.settings.fields.sessionIdleMinutes'), { exact: true })
		).toBeVisible();
		await expect(page.locator('main')).toContainText(numbers[locale]);
	}
});
