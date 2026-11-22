// Link actions shared by the row menu, the tile menu, the details sheet and the keyboard
// shortcuts (plan 8.7.3).
import Archive from '@lucide/svelte/icons/archive';
import ArchiveRestore from '@lucide/svelte/icons/archive-restore';
import Code from '@lucide/svelte/icons/code';
import Copy from '@lucide/svelte/icons/copy';
import ExternalLink from '@lucide/svelte/icons/external-link';
import Info from '@lucide/svelte/icons/info';
import Pencil from '@lucide/svelte/icons/pencil';
import Pin from '@lucide/svelte/icons/pin';
import PinOff from '@lucide/svelte/icons/pin-off';
import RefreshCw from '@lucide/svelte/icons/refresh-cw';
import Share2 from '@lucide/svelte/icons/share-2';
import Tags from '@lucide/svelte/icons/tags';
import Trash2 from '@lucide/svelte/icons/trash-2';
import type { Component } from 'svelte';
import { toast } from 'svelte-sonner';
import * as linksApi from '$lib/api/links.js';
import type { LinkItem, Me } from '$lib/api/types.js';
import { t, type MessageKey } from '$lib/i18n/index.js';
import { toMarkdownLink } from '$lib/links/markdown.js';
import { fullLabel } from '$lib/links/presentation.js';
import { linkStats } from '$lib/state/link-stats.svelte.js';
import { links as linksStore, type LinksStore } from '$lib/state/links.svelte.js';
import { tags as tagsStore } from '$lib/state/tags.svelte.js';
import { copyText } from '$lib/utils/clipboard.js';
import { toastError } from '$lib/utils/toast-error.js';

export type LinkActionId =
	| 'open'
	| 'copyUrl'
	| 'copyMarkdown'
	| 'details'
	| 'edit'
	| 'tags'
	| 'share'
	| 'refresh'
	| 'pin'
	| 'archive'
	| 'delete';

export type LinkAction = {
	id: LinkActionId;
	label: MessageKey;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	icon: Component<any>;
	destructive?: boolean;
};

/** Called by the list, the tiles and the details sheet; the links page dispatches it. */
export type LinkActionHandler = (id: LinkActionId, link: LinkItem) => void;

type Viewer = Pick<Me, 'permissions'> | null | undefined;

/** The actions a viewer may run on a link, in menu order (plan 8.7.3 table). */
export function linkActions(link: LinkItem, me: Viewer): LinkAction[] {
	const owner = link.isOwner;
	const canShare = me?.permissions.canShare ?? false;
	const list: LinkAction[] = [
		{ id: 'open', label: 'links.actions.open', icon: ExternalLink },
		{ id: 'copyUrl', label: 'links.actions.copyUrl', icon: Copy },
		{ id: 'copyMarkdown', label: 'links.actions.copyMarkdown', icon: Code },
		{ id: 'details', label: 'links.actions.details', icon: Info }
	];
	if (owner) list.push({ id: 'edit', label: 'links.actions.edit', icon: Pencil });
	list.push({ id: 'tags', label: 'links.actions.tags', icon: Tags });
	if (owner && canShare) list.push({ id: 'share', label: 'links.actions.share', icon: Share2 });
	if (owner) list.push({ id: 'refresh', label: 'links.actions.refresh', icon: RefreshCw });
	list.push(
		link.pinned
			? { id: 'pin', label: 'links.actions.unpin', icon: PinOff }
			: { id: 'pin', label: 'links.actions.pin', icon: Pin },
		link.archivedAt
			? { id: 'archive', label: 'links.actions.restore', icon: ArchiveRestore }
			: { id: 'archive', label: 'links.actions.archive', icon: Archive }
	);
	if (owner) {
		list.push({ id: 'delete', label: 'links.actions.delete', icon: Trash2, destructive: true });
	}
	return list;
}

/** Whether the viewer may run the action on the link. */
export function canRun(id: LinkActionId, link: LinkItem, me: Viewer): boolean {
	return linkActions(link, me).some((action) => action.id === id);
}

/** Opens the link in a new tab and counts one use. */
export function openLink(link: LinkItem, store: LinksStore = linksStore): void {
	window.open(link.url, '_blank', 'noopener,noreferrer');
	store.visit(link.id);
}

/** Copies the URL; a successful copy counts as a use (plan 5.9.5). */
export async function copyLinkUrl(
	link: LinkItem,
	store: LinksStore = linksStore
): Promise<boolean> {
	if (!(await copyText(link.url))) return false;
	toast.success(t('links.toast.urlCopied'));
	store.visit(link.id);
	return true;
}

/** Copies "[label](url)" with the full label precedence; counts as a use. */
export async function copyLinkMarkdown(
	link: LinkItem,
	store: LinksStore = linksStore
): Promise<boolean> {
	if (!(await copyText(toMarkdownLink(fullLabel(link), link.url)))) return false;
	toast.success(t('links.toast.markdownCopied'));
	store.visit(link.id);
	return true;
}

/** Pins or unpins; resolves the server's version, or null after a failure (already toasted). */
export async function togglePin(
	link: LinkItem,
	store: LinksStore = linksStore
): Promise<LinkItem | null> {
	const pinning = !link.pinned;
	const result = await store.pin(link.id, pinning);
	if (!result) return null;
	if (!pinning) toast.success(t('links.toast.unpinned'));
	else if (link.archivedAt) toast.success(t('links.toast.pinnedRestored'));
	else toast.success(t('links.toast.pinned'));
	return result;
}

/** Callbacks of toggleArchive: onUndo runs after the toast's Undo restored the link. */
export type ArchiveHooks = { onUndo?: (restored: LinkItem) => void | Promise<void> };

/** Archive with an undo action in the toast, or restore; resolves the server's version. */
export async function toggleArchive(
	link: LinkItem,
	store: LinksStore = linksStore,
	hooks: ArchiveHooks = {}
): Promise<LinkItem | null> {
	if (link.archivedAt) {
		const restored = await store.archive(link.id, false);
		if (restored) toast.success(t('links.toast.restored'));
		return restored;
	}
	const archived = await store.archive(link.id, true);
	if (!archived) return null;
	toast.success(t('links.toast.archived'), {
		action: {
			label: t('links.toast.undo'),
			onClick: () => {
				void store.archive(link.id, false).then((restored) => {
					if (!restored) return;
					toast.success(t('links.toast.restored'));
					void hooks.onUndo?.(restored);
				});
			}
		}
	});
	return archived;
}

/** Starts a new metadata fetch and polls until it finishes. */
export async function refreshLink(link: LinkItem, store: LinksStore = linksStore): Promise<void> {
	try {
		const detail = await linksApi.refreshMetadata(link.id);
		store.upsert(detail);
		store.lastDetail = detail;
		store.watchPending(link.id);
		toast.success(t('links.toast.refreshing'));
	} catch (err) {
		toastError(err);
	}
}

/** The dialogs and the sheet that an action may open; the links page provides them. */
export type LinkUi = {
	details(link: LinkItem): void;
	edit(link: LinkItem): void;
	tags(link: LinkItem): void;
	share(link: LinkItem): void;
	confirmDelete(link: LinkItem): void;
};

/** Runs one action: the direct ones here, the rest through the page's dialogs. */
export function runLinkAction(
	id: LinkActionId,
	link: LinkItem,
	ui: LinkUi,
	store: LinksStore = linksStore
): void {
	switch (id) {
		case 'open':
			openLink(link, store);
			break;
		case 'copyUrl':
			void copyLinkUrl(link, store);
			break;
		case 'copyMarkdown':
			void copyLinkMarkdown(link, store);
			break;
		case 'details':
			ui.details(link);
			break;
		case 'edit':
			ui.edit(link);
			break;
		case 'tags':
			ui.tags(link);
			break;
		case 'share':
			ui.share(link);
			break;
		case 'refresh':
			void refreshLink(link, store);
			break;
		case 'pin':
			void togglePin(link, store);
			break;
		case 'archive':
			void toggleArchive(link, store);
			break;
		case 'delete':
			ui.confirmDelete(link);
			break;
	}
}

/** Deletes the link; errors propagate to the confirmation dialog. */
export async function deleteLink(link: LinkItem, store: LinksStore = linksStore): Promise<void> {
	await linksApi.deleteLink(link.id);
	store.remove(link.id);
	if (store.detailsId === link.id) store.closeDetails();
	toast.success(t('links.toast.deleted'));
	void linkStats.refresh();
	void tagsStore.refresh();
}
