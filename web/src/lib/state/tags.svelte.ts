// The user's own tags (plan 8.6). It does not import the links store: listeners subscribe instead.
import * as tagsApi from '$lib/api/tags.js';
import type { ApiError } from '$lib/api/errors.js';
import type { Tag, TagColor } from '$lib/api/types.js';
import { session } from './session.svelte.js';

export type TagsListener = () => void;

/** Name order of the server (name_key): lower-cased, compared by code point. */
export function compareTagNames(a: Pick<Tag, 'name' | 'id'>, b: Pick<Tag, 'name' | 'id'>): number {
	const ka = a.name.toLowerCase();
	const kb = b.name.toLowerCase();
	if (ka < kb) return -1;
	if (ka > kb) return 1;
	return a.id - b.id;
}

/** Number of tags the sidebar lists under "Frequent tags". */
export const FREQUENT_TAG_COUNT = 8;

export class TagsStore {
	items = $state<Tag[]>([]);
	status = $state<'idle' | 'loading' | 'ready' | 'error'>('idle');
	error = $state<ApiError | null>(null);
	readonly byId = $derived(new Map(this.items.map((tag) => [tag.id, tag])));
	/** The most used tags (active links first), for the sidebar. */
	readonly frequent = $derived(
		[...this.items]
			.filter((tag) => tag.activeLinkCount > 0)
			.sort((a, b) => b.activeLinkCount - a.activeLinkCount || compareTagNames(a, b))
			.slice(0, FREQUENT_TAG_COUNT)
	);

	#listeners = new Set<TagsListener>();
	#loadedFor: number | null = null;
	#pending: Promise<void> | null = null;
	#generation = 0;

	/** Subscribes to every change of the tag set; returns the unsubscribe function. */
	onTagsChanged(listener: TagsListener): () => void {
		this.#listeners.add(listener);
		return () => {
			this.#listeners.delete(listener);
		};
	}

	#notify(): void {
		for (const listener of [...this.#listeners]) listener();
	}

	/** Loads GET /api/tags; concurrent calls share one request. */
	load(): Promise<void> {
		if (this.#pending) return this.#pending;
		// A reset (sign-out) in the meantime makes the answer stale: it is dropped.
		const generation = this.#generation;
		const current = () => generation === this.#generation;
		// A refresh keeps the shown tags; only the first load shows the loading state.
		if (this.status !== 'ready') this.status = 'loading';
		this.error = null;
		const pending = tagsApi
			.listTags()
			.then((res) => {
				if (!current()) return;
				this.items = [...res.items].sort(compareTagNames);
				this.status = 'ready';
			})
			.catch((err: ApiError) => {
				if (!current()) return;
				this.error = err;
				this.status = 'error';
			})
			.finally(() => {
				if (this.#pending === pending) this.#pending = null;
			});
		this.#pending = pending;
		return pending;
	}

	/**
	 * Reloads the counts (linkCount, activeLinkCount) after a link change: creation, tag
	 * assignment, archiving, restoring or deletion. Does nothing before the first load.
	 */
	refresh(): Promise<void> {
		if (this.status === 'idle') return Promise.resolve();
		// A request already running may predate the change: one more load follows it.
		const running = this.#pending;
		return running ? running.then(() => this.load()) : this.load();
	}

	/** Loads the tags once per signed-in user; a new user id reloads. */
	ensure(userId: number): Promise<void> {
		if (this.#loadedFor === userId && this.status !== 'error' && this.status !== 'idle') {
			return this.#pending ?? Promise.resolve();
		}
		this.#loadedFor = userId;
		return this.load();
	}

	/** Creates a tag and keeps the list in name order. */
	async create(name: string, color: TagColor = 'neutral'): Promise<Tag> {
		const tag = await tagsApi.createTag(name, color);
		this.items = [...this.items.filter((item) => item.id !== tag.id), tag].sort(compareTagNames);
		this.#notify();
		return tag;
	}

	/** Renames or recolours a tag. */
	async update(id: number, patch: { name?: string; color?: TagColor }): Promise<Tag> {
		const tag = await tagsApi.updateTag(id, patch);
		this.items = this.items.map((item) => (item.id === id ? tag : item)).sort(compareTagNames);
		this.#notify();
		return tag;
	}

	/** Deletes a tag; it also disappears from every link. */
	async remove(id: number): Promise<void> {
		await tagsApi.deleteTag(id);
		this.items = this.items.filter((item) => item.id !== id);
		this.#notify();
	}

	/** Forgets the loaded tags (sign-out). */
	reset(): void {
		this.#generation += 1;
		this.#pending = null;
		this.items = [];
		this.status = 'idle';
		this.error = null;
		this.#loadedFor = null;
	}
}

export const tags = new TagsStore();

// The tags are personal: nothing of the previous user stays after a sign-out or a user switch.
session.onUserChange(() => tags.reset());
