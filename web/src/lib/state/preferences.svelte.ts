// Preferences of the signed-in user: optimistic updates merged into one PATCH (plan 8.6).
import { toast } from 'svelte-sonner';
import * as meApi from '$lib/api/me.js';
import type { PreferencesPatch } from '$lib/api/me.js';
import type { Preferences } from '$lib/api/types.js';
import { i18n } from '$lib/i18n/index.js';
import { applyTheme, session, type SessionStore } from './session.svelte.js';
import { toastError } from '$lib/utils/toast-error.js';

/** Changes made within this window travel in one request. */
export const PREFERENCES_DEBOUNCE_MS = 400;

export class PreferencesStore {
	#session: SessionStore;
	#pending: PreferencesPatch = {};
	#before: Preferences | null = null;
	#timer: ReturnType<typeof setTimeout> | null = null;
	#inflight: Promise<void> | null = null;

	constructor(sessionStore: SessionStore = session) {
		this.#session = sessionStore;
	}

	/** The current preferences (null before sign-in). */
	get current(): Preferences | null {
		return this.#session.me?.preferences ?? null;
	}

	/** Reports whether a save is waiting or running. */
	get saving(): boolean {
		return this.#timer !== null || this.#inflight !== null;
	}

	/** Applies the change at once (theme and locale included) and schedules the merged PATCH. */
	update(patch: PreferencesPatch): void {
		const me = this.#session.me;
		if (!me) return;
		if (!this.#before) this.#before = { ...me.preferences };
		this.#session.me = { ...me, preferences: { ...me.preferences, ...patch } };
		this.#pending = { ...this.#pending, ...patch };
		if (patch.theme !== undefined) applyTheme(patch.theme);
		if (patch.locale !== undefined) i18n.setLocale(patch.locale);
		if (this.#timer !== null) clearTimeout(this.#timer);
		this.#timer = setTimeout(() => {
			this.#timer = null;
			void this.flush();
		}, PREFERENCES_DEBOUNCE_MS);
	}

	/** Drops changes that wait for the debounce: they belong to a user who signed out. */
	reset(): void {
		if (this.#timer !== null) clearTimeout(this.#timer);
		this.#timer = null;
		this.#pending = {};
		this.#before = null;
	}

	/** Sends the merged pending changes now; resolves when the server answered. */
	async flush(): Promise<void> {
		if (this.#timer !== null) {
			clearTimeout(this.#timer);
			this.#timer = null;
		}
		const patch = this.#pending;
		const before = this.#before;
		this.#pending = {};
		this.#before = null;
		if (Object.keys(patch).length === 0) return;
		const run = (async () => {
			try {
				const saved = await meApi.updatePreferences(patch);
				const me = this.#session.me;
				// Changes made while the request was running stay on top of the server answer.
				if (me) {
					this.#session.me = { ...me, preferences: { ...saved, ...this.#pending } };
				}
			} catch (err) {
				const me = this.#session.me;
				if (me && before) {
					// Revert every field except the language, which stays as chosen (plan 8.5.1).
					const reverted = { ...before, locale: me.preferences.locale };
					this.#session.me = { ...me, preferences: reverted };
					if (patch.theme !== undefined) applyTheme(reverted.theme);
				}
				toastError(err);
			}
		})();
		this.#inflight = run;
		try {
			await run;
		} finally {
			if (this.#inflight === run) this.#inflight = null;
		}
	}
}

export const preferences = new PreferencesStore();

session.onUserChange(() => preferences.reset());
