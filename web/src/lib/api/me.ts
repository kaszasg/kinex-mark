// Own account endpoints: profile, preferences, sessions, 2FA, privacy (plan 5.8, 5.14).
import { download, request } from './client.js';
import type {
	Me,
	Preferences,
	PrivacyStatus,
	RecoveryCodes,
	Session,
	TwoFactorSetup
} from './types.js';

export type PreferencesPatch = Partial<
	Omit<Preferences, 'effectiveArchiveAfterDays' | 'systemArchiveAfterDays'>
>;

/** PATCH /api/me/profile */
export function updateProfile(patch: { displayName?: string }): Promise<Me> {
	return request<Me>('PATCH', '/me/profile', { body: patch });
}

/** GET /api/me/preferences */
export function getPreferences(): Promise<Preferences> {
	return request<Preferences>('GET', '/me/preferences');
}

/** PATCH /api/me/preferences */
export function updatePreferences(patch: PreferencesPatch): Promise<Preferences> {
	return request<Preferences>('PATCH', '/me/preferences', { body: patch });
}

/** GET /api/me/sessions */
export function listSessions(): Promise<{ items: Session[] }> {
	return request<{ items: Session[] }>('GET', '/me/sessions');
}

/** DELETE /api/me/sessions/{id} */
export function revokeSession(id: number): Promise<void> {
	return request<void>('DELETE', `/me/sessions/${id}`);
}

/** POST /api/me/sessions/revoke-others */
export function revokeOtherSessions(): Promise<{ revoked: number }> {
	return request<{ revoked: number }>('POST', '/me/sessions/revoke-others');
}

/** POST /api/me/2fa/setup */
export function setupTwoFactor(password: string): Promise<TwoFactorSetup> {
	return request<TwoFactorSetup>('POST', '/me/2fa/setup', { body: { password } });
}

/** POST /api/me/2fa/enable */
export function enableTwoFactor(code: string): Promise<RecoveryCodes> {
	return request<RecoveryCodes>('POST', '/me/2fa/enable', { body: { code } });
}

/** POST /api/me/2fa/disable */
export function disableTwoFactor(password: string, code: string): Promise<void> {
	return request<void>('POST', '/me/2fa/disable', { body: { password, code } });
}

/** POST /api/me/2fa/recovery-codes */
export function regenerateRecoveryCodes(password: string, code: string): Promise<RecoveryCodes> {
	return request<RecoveryCodes>('POST', '/me/2fa/recovery-codes', { body: { password, code } });
}

/** GET /api/me/privacy */
export function getPrivacy(): Promise<PrivacyStatus> {
	return request<PrivacyStatus>('GET', '/me/privacy');
}

/** POST /api/me/privacy/consent */
export function acceptPrivacy(version: number): Promise<void> {
	return request<void>('POST', '/me/privacy/consent', { body: { version } });
}

/** GET /api/me/export (file download) */
export function exportMyData(): Promise<{ blob: Blob; filename: string }> {
	return download('/me/export');
}

/** POST /api/me/delete */
export function deleteAccount(password: string, confirm: string, code?: string): Promise<void> {
	const body: { password: string; confirm: string; code?: string } = { password, confirm };
	if (code !== undefined && code !== '') body.code = code;
	return request<void>('POST', '/me/delete', { body });
}
