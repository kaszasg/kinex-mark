<script lang="ts">
	import Archive from '@lucide/svelte/icons/archive';
	import BookmarkPlus from '@lucide/svelte/icons/bookmark-plus';
	import Plus from '@lucide/svelte/icons/plus';
	import SearchX from '@lucide/svelte/icons/search-x';
	import { tick, untrack } from 'svelte';
	import { page } from '$app/state';
	import type { LinkItem } from '$lib/api/types.js';
	import ConfirmDialog from '$lib/components/app/ConfirmDialog.svelte';
	import EmptyState from '$lib/components/app/EmptyState.svelte';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import { buttonSize, cardSurface } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { t } from '$lib/i18n/index.js';
	import { ALL_FIELDS, fullLabel, type DisplayFields } from '$lib/links/presentation.js';
	import { defaultsFromPreferences, hasActiveFilters } from '$lib/links/query.js';
	import { links as linksStore, type LinksStore } from '$lib/state/links.svelte.js';
	import { preferences } from '$lib/state/preferences.svelte.js';
	import { session } from '$lib/state/session.svelte.js';
	import { modifierKeyLabel, shouldHandleShortcut } from '$lib/utils/keyboard.js';
	import { cn } from '$lib/utils.js';
	import {
		canRun,
		copyLinkMarkdown,
		copyLinkUrl,
		deleteLink,
		runLinkAction,
		toggleArchive,
		togglePin,
		type LinkActionId,
		type LinkUi
	} from './actions.js';
	import ImportExportMenu from './ImportExportMenu.svelte';
	import LinkDetailsSheet from './LinkDetailsSheet.svelte';
	import LinkDialog from './LinkDialog.svelte';
	import LinkList from './LinkList.svelte';
	import LinkTagsDialog from './LinkTagsDialog.svelte';
	import LinkToolbar from './LinkToolbar.svelte';
	import ShareDialog from './ShareDialog.svelte';
	import TagFilterBar from './TagFilterBar.svelte';

	/**
	 * The links page (plan 8.7.3): the core screen. The URL holds the query (search, tags,
	 * scope, deviations from the preferred sort and archived mode) and the open details sheet.
	 */
	type Props = { store?: LinksStore };
	let { store = linksStore }: Props = $props();

	const me = $derived(session.me);
	const prefs = $derived(preferences.current);
	const view = $derived(prefs?.viewMode ?? 'list');
	const fields = $derived<DisplayFields>(
		prefs
			? {
					showIcon: prefs.showIcon,
					showUrl: prefs.showUrl,
					showTitle: prefs.showTitle,
					showDescription: prefs.showDescription
				}
			: ALL_FIELDS
	);
	const archiveDays = $derived(prefs?.effectiveArchiveAfterDays ?? 0);

	// The URL is the source of truth of the query. Only a URL change syncs: preference changes
	// made here update store.defaults together with the query, while the URL catches up later.
	$effect(() => {
		const params = page.url.searchParams;
		untrack(() => store.sync(params, defaultsFromPreferences(preferences.current)));
	});

	let editOpen = $state(false);
	let editLink = $state<LinkItem | null>(null);
	let tagsOpen = $state(false);
	let tagsLink = $state<LinkItem | null>(null);
	let shareOpen = $state(false);
	let shareLink = $state<LinkItem | null>(null);
	let deleteOpen = $state(false);
	let deleteTarget = $state<LinkItem | null>(null);
	let detailsVersion = $state(0);
	let root = $state<HTMLElement | null>(null);

	// Quick capture's "Edit" toast action.
	$effect(() => {
		const request = store.editRequest;
		if (!request) return;
		store.editRequest = null;
		openEditor(request);
	});

	// Sharing changes the details sheet's share list: reload it when the share dialog closes.
	let shareWasOpen = false;
	$effect(() => {
		const isOpen = shareOpen;
		if (shareWasOpen && !isOpen) untrack(() => (detailsVersion += 1));
		shareWasOpen = isOpen;
	});

	const ui: LinkUi = {
		details: (link) => store.openDetails(link.id),
		edit: (link) => openEditor(link),
		tags: (link) => {
			tagsLink = link;
			tagsOpen = true;
		},
		share: (link) => {
			shareLink = link;
			shareOpen = true;
		},
		confirmDelete: (link) => {
			deleteTarget = link;
			deleteOpen = true;
		}
	};

	function openEditor(link: LinkItem | null): void {
		editLink = link;
		editOpen = true;
	}

	function onaction(id: LinkActionId, link: LinkItem): void {
		if (id === 'archive') void toggleArchive(link, store, { onUndo: focusRestored });
		else runLinkAction(id, link, ui, store);
	}

	function showExisting(id: number): void {
		if (store.find(id)) store.highlight(id);
		else store.openDetails(id);
	}

	const shortcut = $derived(`${modifierKeyLabel()} V`);
	const filtered = $derived(hasActiveFilters(store.query));
	const onlyArchived = $derived(
		store.query.archived === 'only' && !hasActiveFilters({ ...store.query, archived: 'exclude' })
	);
	const summary = $derived.by(() => {
		if (store.status !== 'ready' && store.status !== 'loading-more') return undefined;
		// An empty filtered result says "no results", not "no links": the collection is not empty.
		if (filtered && store.total === 0) return t('links.results', { count: 0 });
		const params = {
			count: store.total,
			pinned: store.counts.pinned,
			archived: store.counts.archived
		};
		return store.query.archived === 'exclude'
			? t('links.summary', params)
			: t('links.summaryArchived', params);
	});
	const liveText = $derived.by(() => {
		if (store.announcement) return store.announcement;
		if (store.status === 'ready' && filtered) return t('links.results', { count: store.total });
		return '';
	});

	// Keyboard shortcuts (plan 8.7.3); "/" and "?" belong to the shell.
	function linkElements(): HTMLElement[] {
		return root ? Array.from(root.querySelectorAll<HTMLElement>('[data-link-id]')) : [];
	}

	function currentElement(): HTMLElement | null {
		const active = document.activeElement;
		if (!(active instanceof HTMLElement) || !root?.contains(active)) return null;
		return active.closest<HTMLElement>('[data-link-id]');
	}

	function currentLink(): LinkItem | undefined {
		const id = Number(currentElement()?.dataset.linkId);
		return Number.isFinite(id) ? store.find(id) : undefined;
	}

	function move(step: 1 | -1): void {
		const elements = linkElements();
		if (elements.length === 0) return;
		const current = currentElement();
		const index = current ? elements.indexOf(current) : -1;
		const next =
			index < 0
				? elements[step === 1 ? 0 : elements.length - 1]
				: elements[Math.min(elements.length - 1, Math.max(0, index + step))];
		(next.querySelector<HTMLElement>('[data-link-anchor]') ?? next).focus();
	}

	function focusRow(id: number): boolean {
		const anchor = root?.querySelector<HTMLElement>(`[data-link-id="${id}"] [data-link-anchor]`);
		anchor?.focus();
		return Boolean(anchor);
	}

	/** Focus fell back to <body>, or sits on a toast that is about to close (Undo). */
	function focusLost(): boolean {
		const active = document.activeElement;
		return !active || active === document.body || Boolean(active.closest('[data-sonner-toaster]'));
	}

	/** After an Undo the restored row takes the keyboard position back (plan 8.9). */
	async function focusRestored(link: LinkItem): Promise<void> {
		await tick();
		if (focusLost()) focusRow(link.id);
	}

	// A row that leaves the list while it holds the focus (archive, a filter it no longer matches)
	// hands the focus to its neighbour, so the keyboard position is kept (plan 8.9). The check
	// runs before the DOM update, while the row still exists and has the focus.
	$effect.pre(() => {
		void store.items;
		untrack(() => {
			const active = document.activeElement;
			if (!(active instanceof HTMLElement) || !root?.contains(active)) return;
			const row = active.closest<HTMLElement>('[data-link-id]');
			if (!row) return;
			const id = Number(row.dataset.linkId);
			const order = linkElements().map((element) => Number(element.dataset.linkId));
			void tick().then(() => {
				if (!focusLost() || store.find(id)) return;
				const index = order.indexOf(id);
				const candidates = [...order.slice(index + 1), ...order.slice(0, index).reverse()];
				for (const candidate of candidates) if (focusRow(candidate)) return;
			});
		});
	});

	function onkeydown(event: KeyboardEvent): void {
		if (!shouldHandleShortcut(event)) return;
		const active = document.activeElement;
		// The page owns the keys while focus is inside it or on one of its containers: <body>, or
		// <main> itself after the skip link or a click on a blank part of the content area.
		const inPage = !active || Boolean(root && (root.contains(active) || active.contains(root)));
		const run = (id: LinkActionId): void => {
			const link = currentLink();
			if (!link || !canRun(id, link, me)) return;
			event.preventDefault();
			if (id === 'pin') void togglePin(link, store);
			else if (id === 'archive') void toggleArchive(link, store, { onUndo: focusRestored });
			else if (id === 'copyUrl') void copyLinkUrl(link, store);
			else if (id === 'copyMarkdown') void copyLinkMarkdown(link, store);
			else runLinkAction(id, link, ui, store);
		};
		switch (event.key) {
			case 'n':
				event.preventDefault();
				openEditor(null);
				break;
			case 'j':
			case 'k':
			case 'ArrowDown':
			case 'ArrowUp':
				if (!inPage) return;
				event.preventDefault();
				move(event.key === 'j' || event.key === 'ArrowDown' ? 1 : -1);
				break;
			case 'Enter': {
				const element = currentElement();
				// On the anchor itself the browser's own click opens the link (one visit).
				if (!element || active !== element) return;
				event.preventDefault();
				element.querySelector<HTMLAnchorElement>('[data-link-anchor]')?.click();
				break;
			}
			case 'p':
				run('pin');
				break;
			case 'c':
				run('copyUrl');
				break;
			case 'm':
				run('copyMarkdown');
				break;
			case 'i':
				run('details');
				break;
			case 'e':
				run('edit');
				break;
			case 'a':
				run('archive');
				break;
			case 'Delete':
				run('delete');
				break;
			case 'v':
				event.preventDefault();
				preferences.update({ viewMode: view === 'list' ? 'tiles' : 'list' });
				break;
		}
	}

	async function closeDetails(): Promise<void> {
		const id = store.detailsId;
		store.closeDetails();
		await tick();
		if (id !== null) {
			root
				?.querySelector<HTMLElement>(`[data-link-id="${id}"] [data-link-anchor]`)
				?.focus({ preventScroll: true });
		}
	}
</script>

<svelte:window {onkeydown} />

<div bind:this={root} class="flex flex-col gap-4">
	<PageHeader title={t('links.pageTitle')} subtitle={summary}>
		{#snippet actions()}
			<ImportExportMenu {store} />
			<Button class={buttonSize.default} onclick={() => openEditor(null)}>
				<Plus class="size-4" strokeWidth={1.5} aria-hidden="true" />
				{t('links.toolbar.add')}
			</Button>
		{/snippet}
	</PageHeader>

	<LinkToolbar {store} />
	<TagFilterBar {store} />

	<div class="sr-only" role="status" aria-live="polite" data-testid="links-live">{liveText}</div>

	{#if store.status === 'error'}
		<ErrorAlert error={store.error} onRetry={() => void store.reload()} />
	{:else if (store.status === 'loading' || store.status === 'idle') && store.items.length === 0}
		{#if view === 'tiles'}
			<LoadingRows
				rows={6}
				class="grid grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-3"
				rowClass="h-[148px] rounded-xl"
			/>
		{:else}
			<LoadingRows rows={8} class={cn(cardSurface, 'p-3')} rowClass="h-12" />
		{/if}
	{:else if store.items.length === 0}
		{#if onlyArchived}
			<EmptyState
				icon={Archive}
				title={t('links.empty.archived.title')}
				text={t('links.empty.archived.text')}
			/>
		{:else if filtered}
			<EmptyState
				icon={SearchX}
				title={t('links.empty.filtered.title')}
				text={t('links.empty.filtered.text')}
			>
				{#snippet action()}
					<Button variant="outline" class={buttonSize.default} onclick={() => store.clearFilters()}>
						{t('links.filters.clear')}
					</Button>
				{/snippet}
			</EmptyState>
		{:else}
			<EmptyState
				icon={BookmarkPlus}
				title={t('links.empty.collection.title')}
				text={t('links.empty.collection.text', { shortcut })}
			/>
		{/if}
	{:else}
		<LinkList {me} {fields} {view} effectiveArchiveAfterDays={archiveDays} {onaction} {store} />
	{/if}
</div>

<!--
	The details sheet comes first: bits-ui mounts every portal when its component mounts, so the
	dialogs that the sheet opens (edit, tags, share, delete) land after it in <body> and stack
	above its overlay at the same z-index.
-->
<LinkDetailsSheet
	id={store.detailsId}
	{me}
	effectiveArchiveAfterDays={archiveDays}
	{onaction}
	onclose={() => void closeDetails()}
	version={detailsVersion}
	{store}
/>
<LinkDialog bind:open={editOpen} link={editLink} {store} onShowExisting={showExisting} />
<LinkTagsDialog bind:open={tagsOpen} link={tagsLink} {store} />
<ShareDialog bind:open={shareOpen} link={shareLink} {store} />
<ConfirmDialog
	bind:open={deleteOpen}
	title={t('links.delete.title')}
	text={deleteTarget ? t('links.delete.text', { label: fullLabel(deleteTarget) }) : undefined}
	confirmLabel={t('links.delete.confirm')}
	destructive
	onConfirm={() => (deleteTarget ? deleteLink(deleteTarget, store) : undefined)}
/>
