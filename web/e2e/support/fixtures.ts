// Test fixtures of the E2E suite (plan 9.4.1). Specs import test and expect from here:
//
//   import { test, expect } from '../support/fixtures.ts';
//
// - consoleGuard (automatic): pageerror, securitypolicyviolation and console.error fail the
//   test, except Chromium's "Failed to load resource" report of a same-origin /api/ response.
//   Every same-origin /api/ 5xx response fails the test; a 4xx response fails it unless the
//   test announced it with expectHttp(status, path). The anonymous GET /api/auth/me 401 is
//   always allowed.
// - preference reset (automatic, before each test): the seeded users get the default display
//   preferences back through PATCH /api/me/preferences.
// - asAdmin, asDeveloper, asCustomer, asCustomer2: guarded pages signed in as the seeded users.
// - adminApi: API request context signed in as admin.
// - createUser(role): a fresh, activated user for tests that change account state.
// - openPage(state): a guarded page in a new context with the given storage state.
import {
	test as base,
	expect,
	type APIRequestContext,
	type BrowserContext,
	type Page,
	type TestInfo
} from '@playwright/test';
import {
	acceptPrivacyNotice,
	activateAccount,
	createAccount,
	resetPreferences,
	type Locale,
	type Role,
	type StorageState
} from './api.ts';
import {
	API_HEADERS,
	BASE_URL,
	FIXTURE_ORIGIN,
	SEEDED_USERS,
	USER_PASSWORD
} from './constants.mjs';
import { stateFile, type SeededUserKey } from './global-setup.ts';

export { expect };
export { stateFile, type SeededUserKey };
export type { Locale, Role, StorageState };

/** An HTTP status a test expects from a same-origin /api/ path. */
export interface ExpectedHttp {
	status: number;
	/** Exact pathname ("/api/auth/me") or a pattern tested against the pathname. */
	path: string | RegExp;
	/** HTTP method; any method when omitted. */
	method?: string;
}

export interface CreatedUser {
	id: number;
	username: string;
	displayName: string;
	role: Role;
	/** Current password: the final one, or the temporary one when activate is false. */
	password: string;
	/** Signed-in storage state; empty when activate is false. */
	storageState: StorageState;
}

export interface CreateUserOptions {
	/** Change the temporary password and acknowledge the privacy notice (default true). */
	activate?: boolean;
	locale?: Locale;
	displayName?: string;
}

const API_PREFIX = '/api/';
const CSP_BINDING = '__kxmReportCspViolation';
const ALWAYS_ALLOWED: ExpectedHttp[] = [{ status: 401, path: '/api/auth/me', method: 'GET' }];
const OWN_ORIGIN = new URL(BASE_URL).origin;

function ownApiPath(url: string): string | undefined {
	try {
		const parsed = new URL(url);
		return parsed.origin === OWN_ORIGIN && parsed.pathname.startsWith(API_PREFIX)
			? parsed.pathname
			: undefined;
	} catch {
		return undefined;
	}
}

function matches(rule: ExpectedHttp, status: number, method: string, path: string): boolean {
	if (rule.status !== status) return false;
	if (rule.method !== undefined && rule.method.toUpperCase() !== method) return false;
	return typeof rule.path === 'string' ? rule.path === path : rule.path.test(path);
}

/** Collects console, page, CSP and HTTP problems of every guarded context of one test. */
export class ConsoleGuard {
	readonly problems: string[] = [];
	private readonly allowed: ExpectedHttp[] = [...ALWAYS_ALLOWED];
	private readonly contexts = new WeakSet<BrowserContext>();

	/** Allows a 4xx status for a same-origin /api/ path in the current test. */
	expectHttp(status: number, path: string | RegExp, method?: string): void {
		this.allowed.push({ status, path, method });
	}

	async attach(context: BrowserContext): Promise<void> {
		if (this.contexts.has(context)) return;
		this.contexts.add(context);
		await context.exposeBinding(CSP_BINDING, ({ page }, report: string) => {
			this.problems.push(`CSP violation on ${page?.url() ?? 'unknown page'}: ${report}`);
		});
		await context.addInitScript((binding: string) => {
			document.addEventListener(
				'securitypolicyviolation',
				(event) => {
					const report = `${event.effectiveDirective} blocked ${event.blockedURI || 'inline'} (${event.sourceFile}:${event.lineNumber})`;
					const reporter = (window as unknown as Record<string, (text: string) => void>)[binding];
					reporter?.(report);
				},
				true
			);
		}, CSP_BINDING);
		for (const page of context.pages()) this.watchPage(page);
		context.on('page', (page) => this.watchPage(page));
	}

	private watchPage(page: Page): void {
		page.on('pageerror', (error) => {
			this.problems.push(`page error on ${page.url()}: ${error.message}`);
		});
		page.on('console', (message) => {
			if (message.type() !== 'error') return;
			const text = message.text();
			if (text.startsWith('Failed to load resource') && ownApiPath(message.location().url)) {
				// Chromium reports every non-2xx fetch; the response listener judges those.
				return;
			}
			this.problems.push(`console error on ${page.url()}: ${text}`);
		});
		page.on('response', (response) => {
			const path = ownApiPath(response.url());
			if (path === undefined) return;
			const status = response.status();
			if (status < 400) return;
			const method = response.request().method();
			if (status < 500 && this.allowed.some((rule) => matches(rule, status, method, path))) return;
			this.problems.push(`unexpected HTTP ${status} for ${method} ${path}`);
		});
	}

	assertClean(): void {
		expect(
			this.problems,
			'console errors, page errors, CSP violations or unexpected API errors'
		).toEqual([]);
	}
}

/**
 * A fixture-origin URL unique to the test (plan 9.4.1): http://127.0.0.1:18081/p/<testId>.
 * The retry number keeps retried tests from hitting their own earlier link; the suffix tells
 * several URLs of one test apart. The fixture page title is "Fixture <last path segment>".
 */
export function uniqueUrl(testInfo: TestInfo, suffix?: string): string {
	const id = [testInfo.testId, `r${testInfo.retry}`, suffix]
		.filter((part): part is string => !!part)
		.join('-')
		.replace(/[^A-Za-z0-9._-]/g, '-');
	return `${FIXTURE_ORIGIN}/p/${id}`;
}

let userCounter = 0;

/** A user name that is unique across runs and workers (plan 5.12: 3-64 of [A-Za-z0-9._-]). */
function uniqueUsername(role: Role, workerIndex: number): string {
	userCounter += 1;
	return `e2e-${role.slice(0, 4)}-${Date.now().toString(36)}-${workerIndex}-${userCounter}`;
}

/** Acknowledges the current privacy notice for every seeded user (afterAll of *.last specs). */
export async function acceptPrivacyNoticeForSeededUsers(
	newContext: (state: string) => Promise<APIRequestContext>
): Promise<void> {
	for (const key of Object.keys(SEEDED_USERS) as SeededUserKey[]) {
		const api = await newContext(stateFile(key));
		try {
			await acceptPrivacyNotice(api);
		} finally {
			await api.dispose();
		}
	}
}

type TestFixtures = {
	consoleGuard: ConsoleGuard;
	expectHttp: (status: number, path: string | RegExp, method?: string) => void;
	resetSeededPreferences: void;
	openPage: (state?: string | StorageState) => Promise<Page>;
	asAdmin: Page;
	asDeveloper: Page;
	asCustomer: Page;
	asCustomer2: Page;
	adminApi: APIRequestContext;
	createUser: (role: Role, options?: CreateUserOptions) => Promise<CreatedUser>;
};

type WorkerFixtures = {
	seededApis: Record<SeededUserKey, APIRequestContext>;
};

export const test = base.extend<TestFixtures, WorkerFixtures>({
	seededApis: [
		async ({ playwright }, use) => {
			const apis = {} as Record<SeededUserKey, APIRequestContext>;
			for (const key of Object.keys(SEEDED_USERS) as SeededUserKey[]) {
				apis[key] = await playwright.request.newContext({
					baseURL: BASE_URL,
					extraHTTPHeaders: { ...API_HEADERS },
					storageState: stateFile(key)
				});
			}
			await use(apis);
			for (const api of Object.values(apis)) await api.dispose();
		},
		{ scope: 'worker' }
	],

	consoleGuard: [
		async ({ context }, use) => {
			const guard = new ConsoleGuard();
			await guard.attach(context);
			await use(guard);
			guard.assertClean();
		},
		{ auto: true }
	],

	expectHttp: async ({ consoleGuard }, use) => {
		await use((status, path, method) => consoleGuard.expectHttp(status, path, method));
	},

	resetSeededPreferences: [
		async ({ seededApis }, use) => {
			for (const api of Object.values(seededApis)) await resetPreferences(api);
			await use();
		},
		{ auto: true }
	],

	openPage: async ({ browser, consoleGuard }, use) => {
		const contexts: BrowserContext[] = [];
		await use(async (state) => {
			const context = await browser.newContext({ storageState: state });
			contexts.push(context);
			await consoleGuard.attach(context);
			return context.newPage();
		});
		for (const context of contexts) await context.close();
	},

	asAdmin: async ({ openPage }, use) => {
		await use(await openPage(stateFile('admin')));
	},
	asDeveloper: async ({ openPage }, use) => {
		await use(await openPage(stateFile('developer')));
	},
	asCustomer: async ({ openPage }, use) => {
		await use(await openPage(stateFile('customer')));
	},
	asCustomer2: async ({ openPage }, use) => {
		await use(await openPage(stateFile('customer2')));
	},

	adminApi: async ({ playwright }, use) => {
		const api = await playwright.request.newContext({
			baseURL: BASE_URL,
			extraHTTPHeaders: { ...API_HEADERS },
			storageState: stateFile('admin')
		});
		await use(api);
		await api.dispose();
	},

	createUser: async ({ adminApi }, use, testInfo) => {
		await use(async (role, options = {}) => {
			const username = uniqueUsername(role, testInfo.workerIndex);
			const displayName = options.displayName ?? `E2E ${role} ${userCounter}`;
			const account = await createAccount(adminApi, {
				username,
				role,
				displayName,
				locale: options.locale
			});
			if (options.activate === false) {
				return {
					id: account.id,
					username,
					displayName,
					role,
					password: account.temporaryPassword,
					storageState: { cookies: [], origins: [] }
				};
			}
			const api = await activateAccount(username, account.temporaryPassword, USER_PASSWORD);
			try {
				return {
					id: account.id,
					username,
					displayName,
					role,
					password: USER_PASSWORD,
					storageState: await api.storageState()
				};
			} finally {
				await api.dispose();
			}
		});
	}
});
