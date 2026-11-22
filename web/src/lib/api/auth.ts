// Authentication endpoints (plan 5.7, 5.14).
import { request } from './client.js';
import type { LoginResponse, Me } from './types.js';

export type TotpInput =
	{ challenge: string; code: string } | { challenge: string; recoveryCode: string };

/** POST /api/auth/login */
export function login(username: string, password: string): Promise<LoginResponse> {
	return request<LoginResponse>('POST', '/auth/login', {
		body: { username, password },
		skipAuthRedirect: true
	});
}

/** POST /api/auth/login/totp */
export function loginTotp(input: TotpInput): Promise<{ status: 'ok'; user: Me }> {
	return request<{ status: 'ok'; user: Me }>('POST', '/auth/login/totp', {
		body: input,
		skipAuthRedirect: true
	});
}

/** POST /api/auth/logout */
export function logout(): Promise<void> {
	return request<void>('POST', '/auth/logout', { skipAuthRedirect: true });
}

/** GET /api/auth/me; the session probe passes skipAuthRedirect so a 401 does not redirect. */
export function getMe(opts: { skipAuthRedirect?: boolean } = {}): Promise<Me> {
	return request<Me>('GET', '/auth/me', { skipAuthRedirect: opts.skipAuthRedirect });
}

/** POST /api/auth/password */
export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
	return request<void>('POST', '/auth/password', { body: { currentPassword, newPassword } });
}
