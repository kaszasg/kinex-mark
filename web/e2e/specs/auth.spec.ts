// Sign-in, sign-out, redirects, throttling and session expiry (plan 9.4.2 auth.spec.ts).
import { test, expect } from '../support/fixtures.ts';
import { ADMIN_PASSWORD, ADMIN_USERNAME, API_HEADERS, BASE_URL } from '../support/constants.mjs';
import {
	apiCreateLink,
	fixtureUrl,
	gotoLinks,
	msg,
	rowAction,
	signInUi,
	signOutUi,
	withApi
} from './lib/app.ts';

const loginHeading = () => ({ level: 1, name: msg('auth.login.title') }) as const;

test('admin signs in and out', async ({ page }) => {
	await page.goto('/login');
	await expect(page.getByRole('heading', loginHeading())).toBeVisible();
	await signInUi(page, ADMIN_USERNAME, ADMIN_PASSWORD);
	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();
	// The administration group is part of the admin's sidebar.
	await expect(page.getByRole('link', { name: msg('nav.adminUsers') })).toBeVisible();

	await signOutUi(page);
	await expect(page.getByText(msg('auth.logout.title'))).toBeVisible();
	// The session is gone on the server as well.
	expect((await page.request.get('/api/auth/me')).status()).toBe(401);
});

test('a wrong password shows the invalid credentials message', async ({
	page,
	createUser,
	expectHttp
}) => {
	const user = await createUser('customer');
	expectHttp(401, '/api/auth/login', 'POST');
	await page.goto('/login');
	await signInUi(page, user.username, 'Wrong-Passphrase-123');
	await expect(
		page.getByRole('alert').filter({ hasText: msg('errors.invalid_credentials.title') })
	).toBeVisible();
	await expect(page).toHaveURL(/\/login/);
	// The password field is emptied and focused for the next try.
	const password = page.getByLabel(msg('auth.login.password'), { exact: true });
	await expect(password).toHaveValue('');
	await expect(password).toBeFocused();
});

test('a protected route sends the visitor to sign-in and back', async ({ page, createUser }) => {
	const user = await createUser('developer');
	await page.goto('/tags');
	await expect(page).toHaveURL(/\/login\?next=%2Ftags$/);
	await signInUi(page, user.username, user.password);
	await expect(page).toHaveURL(/\/tags$/);
	await expect(page.getByRole('heading', { level: 1, name: msg('tags.pageTitle') })).toBeVisible();
});

test('an unsafe next target falls back to the links page', async ({ page, createUser }) => {
	const user = await createUser('developer');
	await page.goto('/login?next=/%5Cevil.example');
	await signInUi(page, user.username, user.password);
	await expect(page).toHaveURL(`${BASE_URL}/`);
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();
});

test('five failed attempts lock the account and disable the button', async ({
	page,
	createUser,
	expectHttp
}) => {
	const user = await createUser('customer');
	expectHttp(401, '/api/auth/login', 'POST');
	expectHttp(429, '/api/auth/login', 'POST');
	await page.goto('/login');
	const submit = page.getByRole('button', { name: msg('auth.login.submit'), exact: true });
	const lockedAlert = page
		.getByRole('alert')
		.filter({ hasText: msg('errors.too_many_attempts.title') });
	const statuses: number[] = [];
	for (let attempt = 1; attempt <= 6; attempt++) {
		const answered = page.waitForResponse(
			(response) => new URL(response.url()).pathname === '/api/auth/login'
		);
		await signInUi(page, user.username, `Wrong-Passphrase-${attempt}x`);
		statuses.push((await answered).status());
		if (statuses.at(-1) === 429) break;
	}
	// Five counted failures, then the lock answers.
	expect(statuses).toEqual([401, 401, 401, 401, 401, 429]);
	await expect(lockedAlert).toBeVisible();
	await expect(submit).toBeDisabled();
	// The correct password is refused as well while the lock lasts.
	const login = await page.request.post('/api/auth/login', {
		data: { username: user.username, password: user.password },
		headers: { ...API_HEADERS }
	});
	expect(login.status()).toBe(429);
	expect(await login.json()).toMatchObject({ code: 'too_many_attempts' });
});

test('an expired session leads to the sign-in page with a notice', async ({
	page,
	createUser,
	expectHttp
}) => {
	const user = await createUser('developer');
	await page.goto('/login');
	await signInUi(page, user.username, user.password);
	await expect(page.getByRole('heading', { level: 1, name: msg('links.pageTitle') })).toBeVisible();

	expectHttp(401, /^\/api\//);
	await page.context().clearCookies();
	// The next API request (a scope change reloads the list) is answered with 401.
	await page
		.getByRole('group', { name: msg('links.toolbar.scope') })
		.getByRole('radio', { name: msg('links.scope.own') })
		.click();
	await expect(page).toHaveURL(/\/login\?.*reason=expired/);
	await expect(page.getByText(msg('auth.expired.title'))).toBeVisible();
});

test('an action in an expired session shows the notice and no error toast', async ({
	createUser,
	openPage,
	expectHttp
}, testInfo) => {
	const user = await createUser('developer');
	await withApi(user, (api) =>
		apiCreateLink(api, { url: fixtureUrl(testInfo.testId), description: 'Expiry pin' })
	);
	const page = await openPage(user.storageState);
	await gotoLinks(page);

	expectHttp(401, /^\/api\//);
	await page.context().clearCookies();
	await rowAction(page, 'Expiry pin', 'links.actions.pin');
	await expect(page).toHaveURL(/\/login\?.*reason=expired/);
	await expect(page.getByText(msg('auth.expired.title'))).toBeVisible();
	// The session notice is the only message (phase E): the failed pin raises no error toast.
	await page.waitForTimeout(500);
	await expect(page.locator('[data-sonner-toast][data-type="error"]')).toHaveCount(0);
});
