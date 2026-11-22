// Shared constants of the E2E harness (plan 9.4.1). Plain JavaScript, so that both the
// Node scripts (start-server.mjs, seed-demo.mjs) and the TypeScript support files use them.
import { fileURLToPath } from 'node:url';

/** Address of the server under test (plan 3.4: E2E environment). */
export const SERVER_HOST = '127.0.0.1';
export const SERVER_PORT = 18080;
export const BASE_URL = `http://${SERVER_HOST}:${SERVER_PORT}`;

/** Local metadata fixture origin served by fixture-origin.mjs. */
export const FIXTURE_HOST = '127.0.0.1';
export const FIXTURE_PORT = 18081;
export const FIXTURE_ORIGIN = `http://${FIXTURE_HOST}:${FIXTURE_PORT}`;

/**
 * Password of the first admin (APP_ADMIN_PASSWORD). It satisfies the password policy
 * (plan 6.4: at least 12 runes, 6 distinct, no user name), so the server sets
 * must_change_password = 0 for the bootstrap admin.
 */
export const ADMIN_USERNAME = 'admin';
export const ADMIN_PASSWORD = 'E2e-Fixture-Passphrase-7Q';

/** Final password of every user that global setup or createUser() creates. */
export const USER_PASSWORD = 'E2e-Seeded-Passphrase-4K';

/**
 * Seeded accounts. Tests that change a password, sessions, 2FA, locale, theme or the lock
 * state create their own user with createUser() and never touch these (plan 9.4.1).
 */
export const SEEDED_USERS = /** @type {const} */ ({
	admin: { username: ADMIN_USERNAME, role: 'admin', displayName: 'E2E Admin' },
	developer: { username: 'dev1', role: 'developer', displayName: 'E2E Developer' },
	customer: { username: 'cust1', role: 'customer', displayName: 'E2E Customer One' },
	customer2: { username: 'cust2', role: 'customer', displayName: 'E2E Customer Two' }
});

/** Directory of the saved browser states (e2e/.auth/<key>.json, ignored by git). */
export const AUTH_DIR = fileURLToPath(new URL('../.auth/', import.meta.url));

/** Playwright output directory; start-server.mjs leaves its markers here. */
export const TEST_RESULTS_DIR = fileURLToPath(new URL('../../test-results/', import.meta.url));

/** Holds the absolute path of the temporary data directory of the current run. */
export const DATA_DIR_MARKER = fileURLToPath(
	new URL('../../test-results/.e2e-data-dir', import.meta.url)
);

/** Holds the process id of the server binary of the current run. */
export const SERVER_PID_MARKER = fileURLToPath(
	new URL('../../test-results/.e2e-server-pid', import.meta.url)
);

/** Prefix of the temporary data directories below os.tmpdir(). */
export const DATA_DIR_PREFIX = 'kinex-mark-e2e-';

/** Data directories of earlier, forcibly stopped runs older than this are removed. */
export const STALE_DATA_DIR_MS = 60 * 60 * 1000;

/** Headers every state-changing API request needs (plan 6.3). */
export const API_HEADERS = /** @type {const} */ ({
	'X-Kinex-Request': '1',
	Accept: 'application/json',
	Origin: BASE_URL
});
