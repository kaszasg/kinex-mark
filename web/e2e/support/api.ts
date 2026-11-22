// API helpers of the E2E harness: sign-in, password change, privacy notice and account
// creation through the real HTTP API (plan 5.7, 5.8, 5.12). Used by global-setup.ts,
// fixtures.ts and the specs. Every request carries X-Kinex-Request and Origin (plan 6.3).
import { request, type APIRequestContext, type APIResponse } from '@playwright/test';
import { API_HEADERS, BASE_URL } from './constants.mjs';

export type Role = 'admin' | 'developer' | 'customer';
export type Locale = 'hu' | 'en' | 'de';
export type StorageState = Awaited<ReturnType<APIRequestContext['storageState']>>;

/** The fields of Me (plan 5.5) the harness relies on. */
export interface MeSummary {
	id: number;
	username: string;
	role: Role;
	mustChangePassword: boolean;
	privacyConsentRequired: boolean;
	privacyNoticeVersion: number;
}

export interface CreatedAccount {
	id: number;
	username: string;
	temporaryPassword: string;
}

/** Default display preferences of a user (plan 4.2 user_preferences defaults). */
export const DEFAULT_PREFERENCES = {
	locale: 'hu',
	theme: 'dark',
	viewMode: 'list',
	showIcon: true,
	showUrl: true,
	showTitle: true,
	showDescription: true,
	showArchived: false,
	sort: 'usage',
	tagMatch: 'any',
	archiveAfterDays: null
} as const;

/** Creates an API request context for the server under test, optionally signed in. */
export function newApiContext(storageState?: string | StorageState): Promise<APIRequestContext> {
	return request.newContext({
		baseURL: BASE_URL,
		extraHTTPHeaders: { ...API_HEADERS },
		storageState
	});
}

/** Throws a readable error unless the response has the expected status. */
export async function expectStatus(
	response: APIResponse,
	expected: number | number[],
	action: string
): Promise<void> {
	const allowed = Array.isArray(expected) ? expected : [expected];
	if (allowed.includes(response.status())) return;
	let body = '';
	try {
		body = (await response.text()).slice(0, 500);
	} catch {
		body = '(no body)';
	}
	throw new Error(
		`${action}: expected HTTP ${allowed.join(' or ')}, got ${response.status()} from ` +
			`${new URL(response.url()).pathname}: ${body}`
	);
}

/** Signs in with a password (no second factor) and returns the signed-in user. */
export async function signIn(
	api: APIRequestContext,
	username: string,
	password: string
): Promise<MeSummary> {
	const response = await api.post('/api/auth/login', { data: { username, password } });
	await expectStatus(response, 200, `sign in as ${username}`);
	const body = (await response.json()) as { status: string; user?: MeSummary };
	if (body.status !== 'ok' || !body.user) {
		throw new Error(`sign in as ${username}: unexpected login status "${body.status}"`);
	}
	return body.user;
}

/** Changes the password of the signed-in user; the current session stays valid. */
export async function changePassword(
	api: APIRequestContext,
	currentPassword: string,
	newPassword: string
): Promise<void> {
	const response = await api.post('/api/auth/password', { data: { currentPassword, newPassword } });
	await expectStatus(response, 204, 'change password');
}

/** Acknowledges the current privacy notice version for the signed-in user. */
export async function acceptPrivacyNotice(api: APIRequestContext): Promise<void> {
	const status = await api.get('/api/me/privacy');
	await expectStatus(status, 200, 'read privacy status');
	const { currentVersion, consentRequired } = (await status.json()) as {
		currentVersion: number;
		consentRequired: boolean;
	};
	if (!consentRequired) return;
	const response = await api.post('/api/me/privacy/consent', { data: { version: currentVersion } });
	await expectStatus(response, 204, 'accept privacy notice');
}

/** Resets the display preferences of the signed-in user to the defaults. */
export async function resetPreferences(api: APIRequestContext): Promise<void> {
	const response = await api.patch('/api/me/preferences', { data: DEFAULT_PREFERENCES });
	await expectStatus(response, 200, 'reset preferences');
}

/** Creates a user as admin; without a password the server generates a temporary one. */
export async function createAccount(
	adminApi: APIRequestContext,
	account: { username: string; role: Role; displayName?: string; locale?: Locale }
): Promise<CreatedAccount> {
	const response = await adminApi.post('/api/admin/users', { data: account });
	await expectStatus(response, 201, `create user ${account.username}`);
	const body = (await response.json()) as {
		user: { id: number; username: string };
		temporaryPassword: string | null;
	};
	if (!body.temporaryPassword) {
		throw new Error(`create user ${account.username}: no temporary password in the response`);
	}
	return {
		id: body.user.id,
		username: body.user.username,
		temporaryPassword: body.temporaryPassword
	};
}

/**
 * First sign-in of a new account: signs in with the temporary password, sets the final
 * password and acknowledges the privacy notice. Returns the signed-in context; the caller
 * disposes it.
 */
export async function activateAccount(
	username: string,
	temporaryPassword: string,
	password: string
): Promise<APIRequestContext> {
	const api = await newApiContext();
	try {
		await signIn(api, username, temporaryPassword);
		// must_change_password is always set for accounts an admin creates (plan 5.12).
		await changePassword(api, temporaryPassword, password);
		await acceptPrivacyNotice(api);
		return api;
	} catch (err) {
		await api.dispose();
		throw err;
	}
}
