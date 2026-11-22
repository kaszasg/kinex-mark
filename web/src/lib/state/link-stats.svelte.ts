// The sidebar's link counter: number of links that are not archived (plan 8.6).
import * as linksApi from '$lib/api/links.js';
import { session } from './session.svelte.js';

/** Refresh period while the window has focus. */
export const LINK_STATS_INTERVAL_MS = 60_000;

export class LinkStatsStore {
	active = $state<number | null>(null);
	#generation = 0;

	/** GET /api/links?limit=1&archived=exclude; on failure the previous value stays. */
	async refresh(): Promise<void> {
		const generation = this.#generation;
		try {
			const page = await linksApi.listLinks({ limit: 1, archived: 'exclude' });
			// An answer that arrives after a sign-out belongs to the previous user.
			if (generation === this.#generation) this.active = page.total;
		} catch {
			// Keep the last known count.
		}
	}

	/** Refreshes now and every minute while the document has focus; returns the stop function. */
	start(): () => void {
		void this.refresh();
		const timer = setInterval(() => {
			if (typeof document === 'undefined' || document.hasFocus()) void this.refresh();
		}, LINK_STATS_INTERVAL_MS);
		return () => clearInterval(timer);
	}

	/** Forgets the count (sign-out). */
	reset(): void {
		this.#generation += 1;
		this.active = null;
	}
}

export const linkStats = new LinkStatsStore();

session.onUserChange(() => linkStats.reset());
