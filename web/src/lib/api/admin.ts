// Administration endpoints (plan 5.12, 5.14).
import { request } from './client.js';
import type {
	AdminSettingsResponse,
	AdminUser,
	AdminUserCreated,
	AdminUserPatch,
	AuditEntry,
	AuditQuery,
	NewAdminUser,
	Page,
	Role,
	SystemSettings
} from './types.js';

export type AdminUserQuery = {
	q?: string;
	role?: Role;
	status?: 'active' | 'inactive' | 'all';
	limit?: number;
	offset?: number;
};

export type SystemSettingsPatch = Partial<
	Omit<SystemSettings, 'privacyNoticeVersion' | 'privacyNoticeUpdatedAt'>
>;

/** GET /api/admin/users */
export function listUsers(query: AdminUserQuery = {}): Promise<Page<AdminUser>> {
	return request<Page<AdminUser>>('GET', '/admin/users', { query });
}

/** POST /api/admin/users */
export function createUser(input: NewAdminUser): Promise<AdminUserCreated> {
	return request<AdminUserCreated>('POST', '/admin/users', { body: input });
}

/** GET /api/admin/users/{id} */
export function getUser(id: number): Promise<AdminUser> {
	return request<AdminUser>('GET', `/admin/users/${id}`);
}

/** PATCH /api/admin/users/{id}; role and active changes carry currentPassword. */
export function updateUser(id: number, patch: AdminUserPatch): Promise<AdminUser> {
	return request<AdminUser>('PATCH', `/admin/users/${id}`, { body: patch });
}

/** POST /api/admin/users/{id}/reset-password */
export function resetPassword(
	id: number,
	currentPassword: string,
	password?: string
): Promise<{ temporaryPassword: string | null }> {
	const body: { currentPassword: string; password?: string } = { currentPassword };
	if (password !== undefined && password !== '') body.password = password;
	return request<{ temporaryPassword: string | null }>(
		'POST',
		`/admin/users/${id}/reset-password`,
		{ body }
	);
}

/** POST /api/admin/users/{id}/reset-2fa */
export function resetTwoFactor(id: number, currentPassword: string): Promise<void> {
	return request<void>('POST', `/admin/users/${id}/reset-2fa`, { body: { currentPassword } });
}

/** POST /api/admin/users/{id}/unlock */
export function unlockUser(id: number): Promise<void> {
	return request<void>('POST', `/admin/users/${id}/unlock`);
}

/** DELETE /api/admin/users/{id}/sessions */
export function revokeUserSessions(id: number): Promise<{ revoked: number }> {
	return request<{ revoked: number }>('DELETE', `/admin/users/${id}/sessions`);
}

/** DELETE /api/admin/users/{id} with the step-up body */
export function deleteUser(id: number, currentPassword: string): Promise<void> {
	return request<void>('DELETE', `/admin/users/${id}`, { body: { currentPassword } });
}

/** GET /api/admin/settings */
export function getSettings(): Promise<AdminSettingsResponse> {
	return request<AdminSettingsResponse>('GET', '/admin/settings');
}

/** PATCH /api/admin/settings */
export function updateSettings(patch: SystemSettingsPatch): Promise<AdminSettingsResponse> {
	return request<AdminSettingsResponse>('PATCH', '/admin/settings', { body: patch });
}

/** POST /api/admin/settings/privacy-notice/bump */
export function bumpPrivacyNotice(): Promise<AdminSettingsResponse> {
	return request<AdminSettingsResponse>('POST', '/admin/settings/privacy-notice/bump');
}

/** GET /api/admin/audit */
export function listAudit(query: AuditQuery = {}): Promise<Page<AuditEntry>> {
	return request<Page<AuditEntry>>('GET', '/admin/audit', { query });
}

/** POST /api/admin/jobs/archive */
export function runArchiveJob(): Promise<{ archived: number }> {
	return request<{ archived: number }>('POST', '/admin/jobs/archive');
}
