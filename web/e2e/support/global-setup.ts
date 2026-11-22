// Global setup of the E2E suite (plan 9.4.1): through the API, signs in as the bootstrap admin,
// acknowledges the privacy notice, creates dev1 (developer), cust1 and cust2 (customer),
// activates each (password change, privacy notice) and saves the signed-in states to
// e2e/.auth/{admin,developer,customer,customer2}.json.
import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import {
	acceptPrivacyNotice,
	activateAccount,
	createAccount,
	expectStatus,
	newApiContext,
	signIn
} from './api.ts';
import {
	ADMIN_PASSWORD,
	ADMIN_USERNAME,
	AUTH_DIR,
	SEEDED_USERS,
	USER_PASSWORD
} from './constants.mjs';

export type SeededUserKey = keyof typeof SEEDED_USERS;

/** Saved browser state of a seeded user. */
export function stateFile(key: SeededUserKey): string {
	return path.join(AUTH_DIR, `${key}.json`);
}

export default async function globalSetup(): Promise<void> {
	await mkdir(AUTH_DIR, { recursive: true });
	const admin = await newApiContext();
	try {
		const me = await signIn(admin, ADMIN_USERNAME, ADMIN_PASSWORD);
		if (me.mustChangePassword) {
			throw new Error(
				'the bootstrap admin must change its password: APP_ADMIN_PASSWORD has to satisfy the password policy'
			);
		}
		await acceptPrivacyNotice(admin);
		const profile = await admin.patch('/api/me/profile', {
			data: { displayName: SEEDED_USERS.admin.displayName }
		});
		await expectStatus(profile, 200, 'set the admin display name');
		await admin.storageState({ path: stateFile('admin') });

		for (const key of ['developer', 'customer', 'customer2'] as const) {
			const user = SEEDED_USERS[key];
			const account = await createAccount(admin, {
				username: user.username,
				role: user.role,
				displayName: user.displayName
			});
			const api = await activateAccount(account.username, account.temporaryPassword, USER_PASSWORD);
			try {
				await api.storageState({ path: stateFile(key) });
			} finally {
				await api.dispose();
			}
		}
	} finally {
		await admin.dispose();
	}
}
