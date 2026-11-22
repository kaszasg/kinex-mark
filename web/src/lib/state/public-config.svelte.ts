// Public configuration: loaded once and cached (plan 8.6).
import * as publicApi from '$lib/api/public.js';
import type { PublicConfig } from '$lib/api/types.js';

export class PublicConfigStore {
	config = $state<PublicConfig | null>(null);
	#pending: Promise<PublicConfig | null> | null = null;
	#refreshing: Promise<PublicConfig | null> | null = null;

	/** Loads GET /api/public/config once; a failure leaves config null and is retried next time. */
	ensure(): Promise<PublicConfig | null> {
		if (this.config) return Promise.resolve(this.config);
		if (!this.#pending) {
			this.#pending = publicApi
				.getPublicConfig()
				.then((config) => {
					this.config = config;
					return config;
				})
				.catch(() => null)
				.finally(() => {
					this.#pending = null;
				});
		}
		return this.#pending;
	}

	/**
	 * Loads the configuration again, past the cache (single flight). Used when the server reports
	 * that the cached copy is outdated, for example a bumped privacy notice. A failure keeps the
	 * cached copy and resolves null.
	 */
	refresh(): Promise<PublicConfig | null> {
		if (!this.#refreshing) {
			this.#refreshing = publicApi
				.getPublicConfig()
				.then((config) => {
					this.config = config;
					return config;
				})
				.catch(() => null)
				.finally(() => {
					this.#refreshing = null;
				});
		}
		return this.#refreshing;
	}
}

export const publicConfig = new PublicConfigStore();
