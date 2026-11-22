// Users table: status badges, 2FA column, row menu per user, the own row's locked actions.
import { render, screen, within } from '@testing-library/svelte';
import { describe, expect, test, vi } from 'vitest';
import { t } from '$lib/i18n/index.js';
import UsersTable from './UsersTable.svelte';
import { findFloating, getFloating, makeAdminUser, queryFloating, user } from './test-helpers.js';

const future = new Date(Date.now() + 15 * 60_000).toISOString();

function row(username: string): HTMLElement {
	const cell = screen.getByText(username);
	return cell.closest('tr') as HTMLElement;
}

async function openMenu(username: string) {
	const u = user();
	await u.click(screen.getByRole('button', { name: t('admin.users.actions.menu', { username }) }));
	await findFloating('menuitem', t('admin.users.actions.edit'));
	return u;
}

describe('UsersTable', () => {
	test('shows every column with the state words', () => {
		const users = [
			makeAdminUser({ username: 'anna', role: 'admin', totpEnabled: true, linkCount: 12480 }),
			makeAdminUser({ username: 'bela', active: false, displayName: '', lastLoginAt: null }),
			makeAdminUser({ username: 'cecil', role: 'customer', lockedUntil: future })
		];
		render(UsersTable, { users, selfId: null, onaction: vi.fn() });
		expect(screen.getByRole('table', { name: t('admin.users.table.caption') })).toBeInTheDocument();
		const anna = within(row('anna'));
		expect(anna.getByText(t('roles.admin'))).toBeInTheDocument();
		expect(anna.getByText(t('status.active'))).toBeInTheDocument();
		expect(anna.getByText(t('admin.users.table.twoFactorOn'))).toBeInTheDocument();
		expect(row('anna').textContent).toContain('12 480');
		const bela = within(row('bela'));
		expect(bela.getByText(t('status.inactive'))).toBeInTheDocument();
		expect(bela.getAllByText('–').length).toBeGreaterThanOrEqual(3);
		const cecil = within(row('cecil'));
		expect(cecil.getByText(t('status.locked'))).toBeInTheDocument();
		expect(cecil.getByText(t('roles.customer'))).toBeInTheDocument();
	});

	test('the menu offers 2FA reset and unlock only when they apply', async () => {
		const onaction = vi.fn();
		const plain = makeAdminUser({ username: 'plain' });
		const both = makeAdminUser({ username: 'both', totpEnabled: true, lockedUntil: future });
		render(UsersTable, { users: [plain, both], selfId: null, onaction });
		let u = await openMenu('plain');
		expect(queryFloating('menuitem', t('admin.users.actions.resetTwoFactor'))).toBeNull();
		expect(queryFloating('menuitem', t('admin.users.actions.unlock'))).toBeNull();
		await u.click(getFloating('menuitem', t('admin.users.actions.revokeSessions')));
		expect(onaction).toHaveBeenLastCalledWith('revokeSessions', plain);

		u = await openMenu('both');
		await u.click(getFloating('menuitem', t('admin.users.actions.resetTwoFactor')));
		expect(onaction).toHaveBeenLastCalledWith('resetTwoFactor', both);
		u = await openMenu('both');
		await u.click(getFloating('menuitem', t('admin.users.actions.unlock')));
		expect(onaction).toHaveBeenLastCalledWith('unlock', both);
	});

	test('every action reports the user', async () => {
		const onaction = vi.fn();
		const active = makeAdminUser({ username: 'active' });
		const inactive = makeAdminUser({ username: 'inactive', active: false });
		render(UsersTable, { users: [active, inactive], selfId: 1, onaction });
		for (const [key, action] of [
			['edit', 'edit'],
			['resetPassword', 'resetPassword'],
			['deactivate', 'deactivate'],
			['delete', 'delete']
		] as const) {
			const u = await openMenu('active');
			await u.click(getFloating('menuitem', t(`admin.users.actions.${key}`)));
			expect(onaction).toHaveBeenLastCalledWith(action, active);
		}
		const u = await openMenu('inactive');
		expect(queryFloating('menuitem', t('admin.users.actions.deactivate'))).toBeNull();
		await u.click(getFloating('menuitem', t('admin.users.actions.activate')));
		expect(onaction).toHaveBeenLastCalledWith('activate', inactive);
	});

	test('the own row cannot be deactivated or deleted and says why', async () => {
		const onaction = vi.fn();
		const me = makeAdminUser({ id: 1, username: 'me', role: 'admin' });
		render(UsersTable, { users: [me], selfId: 1, onaction });
		const u = await openMenu('me');
		const deactivate = getFloating('menuitem', t('admin.users.actions.deactivate'));
		const remove = getFloating('menuitem', t('admin.users.actions.delete'));
		expect(deactivate).toHaveAttribute('aria-disabled', 'true');
		expect(remove).toHaveAttribute('aria-disabled', 'true');
		expect(remove).toHaveAttribute('title', t('admin.users.actions.selfHint'));
		expect(
			screen.getAllByText(t('admin.users.actions.selfHint'), { exact: true }).length
		).toBeGreaterThan(0);
		await u.click(remove);
		expect(onaction).not.toHaveBeenCalled();
	});

	test('the menu is disabled while a direct action runs on the row', () => {
		const busy = makeAdminUser({ username: 'busy' });
		render(UsersTable, { users: [busy], selfId: null, busyId: busy.id, onaction: vi.fn() });
		expect(
			screen.getByRole('button', { name: t('admin.users.actions.menu', { username: 'busy' }) })
		).toBeDisabled();
	});
});
