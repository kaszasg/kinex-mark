// Two-factor sign-in (plan 9.4.2 twofa.spec.ts): enrolment in Settings > Security with a code
// computed from the shown secret, sign-in with an authenticator code and with a recovery code,
// the one-time use of recovery codes, and turning 2FA off. Every test uses its own account.
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures.ts';
import { expectStatus } from '../support/api.ts';
import { freshCode, totpAt } from '../support/totp.ts';
import { msg, signInUi, signOutUi, toast, withApi } from './lib/app.ts';

async function enterTotp(page: Page, code: string): Promise<void> {
	const field = page.getByLabel(msg('auth.totp.code'), { exact: true });
	await expect(field).toBeFocused();
	// The sixth digit submits the step.
	await field.pressSequentially(code);
}

/** Relative luminance of a computed `rgb()`/`rgba()` colour (WCAG 2). */
function luminance(colour: string): number {
	const parts = colour
		.match(/[\d.]+/g)
		?.slice(0, 3)
		.map(Number);
	if (!parts || parts.length < 3) throw new Error(`unexpected colour ${colour}`);
	const [r, g, b] = parts.map((value) => {
		const c = value / 255;
		return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
	});
	return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

/**
 * The printed recovery-code sheet is dark text on a light surface in the dark theme too (phase E
 * finding: the print reset missed the muted and popover tokens). The print button adds the print
 * class only around window.print(), so the test sets it for the emulated print media itself.
 */
async function expectReadablePrint(page: Page): Promise<void> {
	await page.emulateMedia({ media: 'print' });
	await page.evaluate(() => document.documentElement.classList.add('kxm-print-codes'));
	try {
		const sheet = page.locator('.kxm-codes-sheet');
		const read = () =>
			sheet.evaluate((root) => {
				const list = root.querySelector('ul') as HTMLElement;
				const item = root.querySelector('li') as HTMLElement;
				const heading = root.querySelector('.kxm-print-only') as HTMLElement;
				return {
					surface: getComputedStyle(list).backgroundColor,
					code: getComputedStyle(item).color,
					heading: getComputedStyle(heading).color,
					headingDisplay: getComputedStyle(heading).display
				};
			});
		let colours = await read();
		// The dialog's colour transition runs when the emulated media switches; paper gets the
		// end state, so the contrast is read once the transition has finished.
		await expect
			.poll(
				async () => {
					colours = await read();
					const surface = luminance(colours.surface);
					return Math.min(
						...[colours.code, colours.heading].map(
							(text) => (surface + 0.05) / (luminance(text) + 0.05)
						)
					);
				},
				{ timeout: 3_000, intervals: [100, 200, 400] }
			)
			.toBeGreaterThanOrEqual(7);
		expect(colours.headingDisplay, JSON.stringify(colours)).not.toBe('none');
	} finally {
		await page.evaluate(() => document.documentElement.classList.remove('kxm-print-codes'));
		await page.emulateMedia({ media: 'screen' });
	}
}

test('enable 2FA, sign in with a code, then turn it off with a recovery code', async ({
	createUser,
	openPage
}) => {
	test.setTimeout(150_000);
	const user = await createUser('developer');
	const page = await openPage(user.storageState);
	await page.goto('/settings/security');
	await page.getByRole('button', { name: msg('security.twoFactor.enable'), exact: true }).click();

	const dialog = page.getByRole('dialog');
	await expect(
		dialog.getByRole('heading', { name: msg('security.enroll.password.title') })
	).toBeVisible();
	await dialog
		.getByLabel(msg('security.enroll.password.label'), { exact: true })
		.fill(user.password);
	await dialog
		.getByRole('button', { name: msg('security.enroll.password.submit'), exact: true })
		.click();

	await expect(
		dialog.getByRole('heading', { name: msg('security.enroll.scan.title') })
	).toBeVisible();
	await expect(dialog.getByRole('img', { name: msg('security.enroll.scan.qr') })).toBeVisible();
	const secret = (await dialog.locator('[data-secret]').innerText()).replace(/\s/g, '');
	expect(secret).toMatch(/^[A-Z2-7]{32}$/);
	const enrolCode = totpAt(secret, Date.now());
	await dialog
		.getByLabel(msg('security.enroll.scan.code'), { exact: true })
		.pressSequentially(enrolCode.code);
	await dialog
		.getByRole('button', { name: msg('security.enroll.scan.submit'), exact: true })
		.click();

	await expect(
		dialog.getByRole('heading', { name: msg('security.enroll.codes.title') })
	).toBeVisible();
	const codes = await dialog
		.getByRole('list', { name: msg('security.recovery.listLabel') })
		.getByRole('listitem')
		.allInnerTexts();
	expect(codes).toHaveLength(10);
	for (const code of codes) expect(code.trim()).toMatch(/^[a-z2-9]{5}-[a-z2-9]{5}$/);
	await expectReadablePrint(page);
	const done = dialog.getByRole('button', { name: msg('common.actions.done'), exact: true });
	await expect(done).toBeDisabled();
	await dialog.getByLabel(msg('security.recovery.saved')).check();
	await done.click();
	await expect(toast(page, msg('security.twoFactor.enabled'))).toBeVisible();
	await expect(page.getByText(msg('status.enabled'), { exact: true })).toBeVisible();

	await signOutUi(page);
	await signInUi(page, user.username, user.password);
	await expect(page.getByRole('heading', { level: 1, name: msg('auth.totp.title') })).toBeVisible();
	const loginCode = await freshCode(secret, enrolCode.step);
	await enterTotp(page, loginCode.code);
	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();

	await page.goto('/settings/security');
	await page.getByRole('button', { name: msg('security.twoFactor.disable'), exact: true }).click();
	const confirm = page.getByRole('dialog');
	await confirm.getByLabel(msg('security.confirm.password'), { exact: true }).fill(user.password);
	await confirm
		.getByLabel(msg('security.confirm.disable.code'), { exact: true })
		.fill(codes[0].trim());
	await confirm
		.getByRole('button', { name: msg('security.confirm.disable.submit'), exact: true })
		.click();
	await expect(toast(page, msg('security.twoFactor.disabled'))).toBeVisible();
	await expect(page.getByText(msg('status.disabled'), { exact: true })).toBeVisible();

	// The next sign-in asks for the password only.
	await signOutUi(page);
	await signInUi(page, user.username, user.password);
	await expect(page).toHaveURL(/\/$/);
});

test('a recovery code signs in once and is refused the second time', async ({
	createUser,
	page,
	expectHttp
}) => {
	const user = await createUser('customer');
	const codes = await withApi(user, async (api) => {
		const setup = await api.post('/api/me/2fa/setup', { data: { password: user.password } });
		await expectStatus(setup, 200, '2fa setup');
		const { secret } = (await setup.json()) as { secret: string };
		const enable = await api.post('/api/me/2fa/enable', {
			data: { code: totpAt(secret, Date.now()).code }
		});
		await expectStatus(enable, 200, '2fa enable');
		return ((await enable.json()) as { recoveryCodes: string[] }).recoveryCodes;
	});
	const recovery = codes[3];

	await page.goto('/login');
	await signInUi(page, user.username, user.password);
	await expect(page.getByRole('heading', { level: 1, name: msg('auth.totp.title') })).toBeVisible();
	await page.getByRole('button', { name: msg('auth.totp.useRecovery'), exact: true }).click();
	// Upper case and without the hyphen: the form normalises the code.
	await page
		.getByLabel(msg('auth.recovery.code'), { exact: true })
		.fill(recovery.replace('-', '').toUpperCase());
	await page.getByRole('button', { name: msg('auth.totp.submit'), exact: true }).click();
	await expect(page).toHaveURL(/\/$/);
	await expect(toast(page, msg('auth.recovery.used', { count: 9 }))).toBeVisible();

	await signOutUi(page);
	expectHttp(401, '/api/auth/login/totp', 'POST');
	await signInUi(page, user.username, user.password);
	await page.getByRole('button', { name: msg('auth.totp.useRecovery'), exact: true }).click();
	await page.getByLabel(msg('auth.recovery.code'), { exact: true }).fill(recovery);
	await page.getByRole('button', { name: msg('auth.totp.submit'), exact: true }).click();
	await expect(
		page.getByRole('alert').filter({ hasText: msg('errors.invalid_totp.title') })
	).toBeVisible();
	await expect(page).toHaveURL(/\/login/);
});
