// Signed-in user, gates and permissions (plan 8.6).
import { goto } from '$app/navigation';
import { setMode } from 'mode-watcher';
import * as authApi from '$lib/api/auth.js';
import { onAuthFailure } from '$lib/api/client.js';
import { ApiError, type AuthFailureCode } from '$lib/api/errors.js';
import type { LoginResponse, Me, Preferences } from '$lib/api/types.js';
import * as meApi from '$lib/api/me.js';
import { i18n } from '$lib/i18n/index.js';
import { loginUrl } from '$lib/utils/navigation.js';
import { publicConfig } from './public-config.svelte.js';

export type SessionStatus = 'unknown' | 'anonymous' | 'authenticated';
export type Gate = 'password' | 'consent';
export type Permission = 'share' | 'listUsers' | 'admin';
export type ClearReason = 'expired' | 'logout' | 'deleted';

/** Applies the theme preference through mode-watcher (dark, light or the system setting). */
export function applyTheme(theme: Preferences['theme']): void {
	setMode(theme);
}

export class SessionStore {
	me = $state<Me | null>(null);
	status = $state<SessionStatus>('unknown');
	/** The reason of the last clear, read by the login page for its info alert. */
	lastClearReason = $state<ClearReason | null>(null);
	/** The open gate: the password change first, then the privacy notice (as on the server). */
	readonly gate = $derived<Gate | null>(
		this.me?.mustChangePassword ? 'password' : this.me?.privacyConsentRequired ? 'consent' : null
	);

	#pending: Promise<Me | null> | null = null;
	/** The user whose data the per-user stores may hold (kept until the session ends). */
	#userId: number | null = null;
	#userListeners = new Set<() => void>();

	/**
	 * Subscribes a per-user store (links, tags, counters, preferences) to the end of a user's
	 * session: the listener runs on every clear and when a different user signs in, so no data of
	 * the previous user survives on the tab (plan 8.6). Returns the unsubscribe function.
	 */
	onUserChange(listener: () => void): () => void {
		this.#userListeners.add(listener);
		return () => {
			this.#userListeners.delete(listener);
		};
	}

	#switchUser(id: number | null): void {
		const previous = this.#userId;
		this.#userId = id;
		if (previous === null || previous === id) return;
		for (const listener of [...this.#userListeners]) listener();
	}

	/** Single-flight probe of GET /api/auth/me; a 401 marks the session anonymous. */
	ensure(): Promise<Me | null> {
		if (this.status === 'authenticated') return Promise.resolve(this.me);
		if (this.status === 'anonymous') return Promise.resolve(null);
		if (!this.#pending) {
			this.#pending = authApi
				.getMe({ skipAuthRedirect: true })
				.then((me) => {
					this.setMe(me);
					return me;
				})
				.catch((err: unknown) => {
					if (err instanceof ApiError && err.status === 401) {
						this.me = null;
						this.status = 'anonymous';
						return null;
					}
					throw err;
				})
				.finally(() => {
					this.#pending = null;
				});
		}
		return this.#pending;
	}

	/** First sign-in step; an 'ok' answer signs the user in at once. */
	async login(username: string, password: string): Promise<LoginResponse> {
		const res = await authApi.login(username, password);
		if (res.status === 'ok') this.setMe(res.user);
		return res;
	}

	/** Second sign-in step with a TOTP code or a recovery code. */
	async verifyTotp(input: {
		challenge: string;
		code?: string;
		recoveryCode?: string;
	}): Promise<Me> {
		const body =
			input.recoveryCode !== undefined
				? { challenge: input.challenge, recoveryCode: input.recoveryCode }
				: { challenge: input.challenge, code: input.code ?? '' };
		const res = await authApi.loginTotp(body);
		this.setMe(res.user);
		return res.user;
	}

	/** Signs out on the server (errors ignored), clears the state and opens the login page. */
	async logout(): Promise<void> {
		try {
			await authApi.logout();
		} catch {
			// The local session ends regardless of the server answer.
		}
		this.clear('logout');
		await goto('/login');
	}

	/** Reloads the user after a password change, consent or 2FA change. */
	async refresh(): Promise<Me> {
		const me = await authApi.getMe();
		this.setMe(me);
		return me;
	}

	/**
	 * Stores the user and applies its locale and theme. A language picked by hand on the login
	 * page wins: it is kept and saved to the server preference.
	 */
	setMe(me: Me): void {
		const explicit = i18n.explicit;
		if (explicit && explicit !== me.preferences.locale) {
			me = { ...me, preferences: { ...me.preferences, locale: explicit } };
			meApi.updatePreferences({ locale: explicit }).catch(() => {
				// The choice stays active locally; the next change saves it again.
			});
		}
		i18n.clearExplicit();
		this.#switchUser(me.id);
		this.me = me;
		this.status = 'authenticated';
		this.lastClearReason = null;
		i18n.setLocale(me.preferences.locale);
		applyTheme(me.preferences.theme);
	}

	/** Forgets the user (expired session, sign-out, deleted account). */
	clear(reason?: ClearReason): void {
		this.#switchUser(null);
		this.me = null;
		this.status = 'anonymous';
		this.lastClearReason = reason ?? null;
		this.#pending = null;
	}

	/** UI-side permission check; the server remains the real boundary. */
	can(permission: Permission): boolean {
		const permissions = this.me?.permissions;
		if (!permissions) return false;
		if (permission === 'share') return permissions.canShare;
		if (permission === 'listUsers') return permissions.canListUsers;
		return permissions.canAdmin;
	}

	/** Reaction of the API client to a lost session or an open gate (plan 8.6). */
	async handleAuthFailure(code: AuthFailureCode): Promise<void> {
		if (code === 'unauthenticated') {
			const here = typeof location !== 'undefined' ? location.pathname + location.search : '/';
			this.clear('expired');
			if (typeof location !== 'undefined' && location.pathname === '/login') return;
			await goto(loginUrl(here, 'expired'));
			return;
		}
		// consent_required means a newer privacy notice than the one the tab may have cached:
		// the gate must show (and acknowledge) the current version.
		const config = code === 'consent_required' ? publicConfig.refresh() : null;
		try {
			await this.refresh();
		} catch {
			// A failed refresh is reported by its own handler.
		}
		await config;
		await goto(code === 'password_change_required' ? '/change-password' : '/consent');
	}
}

export const session = new SessionStore();

onAuthFailure((code) => {
	void session.handleAuthFailure(code);
});
