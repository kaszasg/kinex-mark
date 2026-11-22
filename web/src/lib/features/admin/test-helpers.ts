// Test helpers of the admin feature specs (floating content, user events, admin fixtures).
import { screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import type {
	AdminSettingsResponse,
	AdminUser,
	AuditEntry,
	SystemSettings
} from '$lib/api/types.js';

const T0 = '2026-01-05T10:00:00.000Z';

export const user = () => userEvent.setup({ pointerEventsCheck: 0 });

type FloatingOptions = { exact?: boolean };

/**
 * Floating content is visibility:hidden in jsdom (no layout), so its text does not count as an
 * accessible name: floating items are found by role and text instead.
 */
export function queryFloating(
	role: string,
	text: string,
	{ exact = true }: FloatingOptions = {}
): HTMLElement | null {
	return (
		screen.queryAllByRole(role, { hidden: true }).find((el) => {
			const content = el.textContent?.trim() ?? '';
			return exact ? content === text : content.includes(text);
		}) ?? null
	);
}

export function getFloating(role: string, text: string, options?: FloatingOptions): HTMLElement {
	const el = queryFloating(role, text, options);
	if (!el) throw new Error(`no ${role} with text "${text}"`);
	return el;
}

export function findFloating(
	role: string,
	text: string,
	options?: FloatingOptions
): Promise<HTMLElement> {
	return vi.waitFor(() => getFloating(role, text, options));
}

let nextId = 500;

export function makeAdminUser(partial: Partial<AdminUser> = {}): AdminUser {
	nextId += 1;
	return {
		id: nextId,
		username: `user${nextId}`,
		displayName: 'Szabo Peter',
		role: 'developer',
		active: true,
		mustChangePassword: false,
		totpEnabled: false,
		lockedUntil: null,
		lastLoginAt: T0,
		createdAt: T0,
		updatedAt: T0,
		linkCount: 12,
		sharedLinkCount: 0,
		sessionCount: 1,
		...partial
	};
}

export function makeSettings(partial: Partial<SystemSettings> = {}): SystemSettings {
	return {
		defaultArchiveDays: 90,
		auditRetentionDays: 180,
		sessionIdleMinutes: 720,
		sessionAbsoluteHours: 168,
		allowPrivateFetch: false,
		privacyControllerName: 'Kinexium',
		privacyControllerAddress: 'Budapest',
		privacyControllerEmail: 'privacy@example.com',
		privacyDpoContact: '',
		privacySupervisoryAuthority: '',
		privacyNoticeVersion: 3,
		privacyNoticeUpdatedAt: T0,
		...partial
	};
}

export function makeSettingsResponse(
	settings: Partial<SystemSettings> = {},
	lockedByEnv: string[] = []
): AdminSettingsResponse {
	return {
		settings: makeSettings(settings),
		defaults: makeSettings({ privacyControllerName: '', privacyControllerAddress: '' }),
		lockedByEnv
	};
}

let nextEntry = 9000;

export function makeAuditEntry(partial: Partial<AuditEntry> = {}): AuditEntry {
	nextEntry += 1;
	return {
		id: nextEntry,
		at: T0,
		event: 'auth.login.success',
		success: true,
		actor: { id: 1, username: 'kanna' },
		target: null,
		ip: '192.0.2.10',
		details: {},
		...partial
	};
}

export function page<T>(items: T[], total = items.length, offset = 0) {
	return { items, total, limit: 50, offset };
}
