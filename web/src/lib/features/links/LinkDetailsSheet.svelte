<script lang="ts">
	import Archive from '@lucide/svelte/icons/archive';
	import ArchiveRestore from '@lucide/svelte/icons/archive-restore';
	import Code from '@lucide/svelte/icons/code';
	import Copy from '@lucide/svelte/icons/copy';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Globe from '@lucide/svelte/icons/globe';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Pin from '@lucide/svelte/icons/pin';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Share2 from '@lucide/svelte/icons/share-2';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import UserIcon from '@lucide/svelte/icons/user';
	import Users from '@lucide/svelte/icons/users';
	import X from '@lucide/svelte/icons/x';
	import { untrack } from 'svelte';
	import type { ApiError } from '$lib/api/errors.js';
	import * as linksApi from '$lib/api/links.js';
	import type { LinkDetail, LinkItem, Me, Share, Tag } from '$lib/api/types.js';
	import CopyButton from '$lib/components/app/CopyButton.svelte';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import Favicon from '$lib/components/app/Favicon.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import { buttonSize, typeHeading3 } from '$lib/components/app/styles.js';
	import TagChip from '$lib/components/app/TagChip.svelte';
	import Metric from '$lib/components/brand/Metric.svelte';
	import StatusBadge, { type Status } from '$lib/components/brand/StatusBadge.svelte';
	import Button from '$lib/components/app/Button.svelte';
	import * as Sheet from '$lib/components/ui/sheet/index.js';
	import { fmt, t, tDynamic, type MessageKey } from '$lib/i18n/index.js';
	import { expiryInfo } from '$lib/links/expiry.js';
	import { fullLabel, prettyHost, readableUrl } from '$lib/links/presentation.js';
	import {
		links as linksStore,
		toApiError,
		toLinkItem,
		type LinksStore
	} from '$lib/state/links.svelte.js';
	import { tags as tagsStore } from '$lib/state/tags.svelte.js';
	import { cn } from '$lib/utils.js';
	import {
		canRun,
		copyLinkMarkdown,
		copyLinkUrl,
		refreshLink,
		toggleArchive,
		togglePin,
		type LinkActionHandler
	} from './actions.js';

	/**
	 * The full view of one link (plan 8.7.3): every datum, usage statistics, timestamps,
	 * metadata state and sharing. Opens with ?link=<id>; loads GET /api/links/{id}. Personal
	 * state follows the list's copy of the link, so optimistic changes show here too.
	 */
	type Props = {
		/** The link to show; null closes the sheet. */
		id: number | null;
		me: Me | null;
		effectiveArchiveAfterDays: number;
		onaction: LinkActionHandler;
		onclose: () => void;
		/** Bumped by the page after a change that the sheet must reload (sharing). */
		version?: number;
		store?: LinksStore;
	};
	let {
		id,
		me,
		effectiveArchiveAfterDays,
		onaction,
		onclose,
		version = 0,
		store = linksStore
	}: Props = $props();

	let detail = $state<LinkDetail | null>(null);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let error = $state<ApiError | null>(null);
	let abort: AbortController | null = null;

	async function load(linkId: number): Promise<void> {
		abort?.abort();
		const controller = new AbortController();
		abort = controller;
		if (detail?.id !== linkId) detail = null;
		status = detail ? 'ready' : 'loading';
		error = null;
		try {
			const result = await linksApi.getLink(linkId, controller.signal);
			if (controller !== abort) return;
			detail = result;
			status = 'ready';
		} catch (err) {
			if (controller !== abort || (err as { name?: string })?.name === 'AbortError') return;
			error = toApiError(err);
			status = 'error';
		}
	}

	$effect(() => {
		void version;
		const linkId = id;
		if (linkId === null) return;
		untrack(() => void load(linkId));
		return () => abort?.abort();
	});

	// Polling results (pending metadata) update an open sheet.
	$effect(() => {
		const latest = store.lastDetail;
		if (latest && latest.id === id) detail = latest;
	});

	/** The detail with the list's (possibly optimistic) personal state on top. */
	const view = $derived.by((): LinkDetail | null => {
		if (!detail) return null;
		const item = store.find(detail.id);
		return item ? { ...detail, ...item } : detail;
	});
	const label = $derived(view ? fullLabel(view) : '');
	const viewTags = $derived(
		view
			? view.tagIds
					.map((tagId) => tagsStore.byId.get(tagId) ?? view.tags.find((tag) => tag.id === tagId))
					.filter((tag): tag is Tag => tag !== undefined)
			: []
	);
	const canShare = $derived(Boolean(view?.isOwner && me?.permissions.canShare));

	const fetchStatus: Record<LinkDetail['fetchStatus'], { status: Status; label: MessageKey }> = {
		ok: { status: 'online', label: 'fetch.status.ok' },
		pending: { status: 'info', label: 'fetch.status.pending' },
		failed: { status: 'critical', label: 'fetch.status.failed' },
		blocked: { status: 'warning', label: 'fetch.status.blocked' }
	};

	const archiveMetric = $derived.by(() => {
		if (!view) return { value: '', note: '' };
		if (view.archivedAt) {
			return { value: fmt.date(view.archivedAt), note: t('links.details.archivedNote') };
		}
		if (view.pinned || effectiveArchiveAfterDays === 0 || !view.archiveDueAt) {
			return { value: t('links.details.never'), note: '' };
		}
		const info = expiryInfo({
			archiveDueAt: view.archiveDueAt,
			inactiveSince: view.inactiveSince,
			now: new Date()
		});
		const days = info?.daysLeft ?? 0;
		return {
			value: fmt.date(view.archiveDueAt),
			note: days > 0 ? t('links.details.dueIn', { count: days }) : t('links.details.dueToday')
		};
	});

	/** Applies the server's answer to the shown detail (when the list does not hold the link). */
	function apply(result: LinkItem | null): void {
		if (result && detail && result.id === detail.id) detail = { ...detail, ...toLinkItem(result) };
	}

	/** Without a list copy the sheet counts the use itself. */
	function countUse(): void {
		if (detail && !store.find(detail.id)) {
			detail = { ...detail, useCount: detail.useCount + 1, lastUsedAt: new Date().toISOString() };
		}
	}

	function open(): void {
		if (!view) return;
		countUse();
		store.visit(view.id);
	}

	function onauxclick(event: MouseEvent): void {
		if (event.button === 1) open();
	}

	async function copyUrl(): Promise<void> {
		if (view && (await copyLinkUrl(view, store))) countUse();
	}

	async function copyMarkdown(): Promise<void> {
		if (view && (await copyLinkMarkdown(view, store))) countUse();
	}

	function shareText(share: Share): string {
		if (share.targetType === 'all') return t('links.share.everyone');
		if (share.targetType === 'role' && share.targetRole) {
			return t('links.share.roleTarget', { role: t(`roles.plural.${share.targetRole}`) });
		}
		return share.targetUser?.displayName || share.targetUser?.username || fmt.missing;
	}

	function time(value: string | null): string {
		return value ? fmt.dateTime(value) : fmt.missing;
	}
</script>

<Sheet.Root
	open={id !== null}
	onOpenChange={(next: boolean) => {
		if (!next) onclose();
	}}
>
	<Sheet.Content
		side="right"
		showCloseButton={false}
		class="w-full gap-0 overflow-y-auto p-0 data-[side=right]:w-full data-[side=right]:sm:max-w-[440px]"
	>
		<Sheet.Header class="flex-row items-start gap-3 border-b p-6 pb-4">
			<span class="flex size-10 shrink-0 items-center justify-center rounded-md border bg-muted">
				<Favicon src={view?.iconUrl} size={32} />
			</span>
			<div class="flex min-w-0 flex-1 flex-col gap-0.5">
				<Sheet.Title class={cn(typeHeading3, 'line-clamp-2 break-words')}>
					{label || t('links.details.title')}
				</Sheet.Title>
				<Sheet.Description class="truncate font-mono text-[13px] leading-5 text-muted-foreground">
					{view ? prettyHost(view.host) : t('links.details.title')}
				</Sheet.Description>
			</div>
			<Button
				variant="ghost"
				size="icon"
				class={cn(buttonSize.iconSm, '-mt-1 -mr-2')}
				aria-label={t('common.actions.close')}
				onclick={onclose}
			>
				<X class="size-4" strokeWidth={1.5} aria-hidden="true" />
			</Button>
		</Sheet.Header>

		{#if status === 'loading'}
			<LoadingRows rows={6} class="p-6" />
		{:else if status === 'error' || !view}
			<div class="p-6">
				<ErrorAlert {error} onRetry={() => id !== null && void load(id)} />
			</div>
		{:else}
			<div class="flex flex-wrap items-center gap-2 px-6 py-4">
				<Button
					href={view.url}
					target="_blank"
					rel="noopener noreferrer"
					variant="outline"
					class={buttonSize.sm}
					onclick={open}
					{onauxclick}
				>
					<ExternalLink class="size-4" strokeWidth={1.5} aria-hidden="true" />
					{t('links.actions.open')}
				</Button>
				<Button variant="outline" class={buttonSize.sm} onclick={() => void copyUrl()}>
					<Copy class="size-4" strokeWidth={1.5} aria-hidden="true" />
					{t('links.actions.copyUrl')}
				</Button>
				<Button variant="outline" class={buttonSize.sm} onclick={() => void copyMarkdown()}>
					<Code class="size-4" strokeWidth={1.5} aria-hidden="true" />
					{t('links.actions.copyMarkdown')}
				</Button>
				<Button
					variant="ghost"
					size="icon"
					class={buttonSize.iconSm}
					aria-pressed={view.pinned}
					aria-label={view.pinned ? t('links.actions.unpin') : t('links.actions.pin')}
					onclick={async () => view && apply(await togglePin(view, store))}
				>
					<Pin
						class={cn('size-4', view.pinned && 'text-primary')}
						strokeWidth={1.5}
						aria-hidden="true"
					/>
				</Button>
			</div>

			<div class="flex flex-col gap-6 px-6 pb-6">
				<section class="flex flex-col gap-3" aria-labelledby="link-details-data">
					<h3 id="link-details-data" class="kx-eyebrow text-muted-foreground">
						{t('links.details.sections.data')}
					</h3>
					<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-[13px] leading-5">
						<dt class="text-muted-foreground">{t('links.details.fields.url')}</dt>
						<dd class="flex min-w-0 items-start gap-1">
							<span class="min-w-0 font-mono break-all" data-testid="details-url">
								{readableUrl(view.url)}
							</span>
							<CopyButton
								text={view.url}
								label={t('links.actions.copyUrl')}
								toastText={t('links.toast.urlCopied')}
								class="-mt-1 shrink-0"
								onCopied={() => {
									countUse();
									if (view) store.visit(view.id);
								}}
							/>
						</dd>
						<dt class="text-muted-foreground">{t('links.details.fields.title')}</dt>
						<dd class="min-w-0 break-words">{view.title || fmt.missing}</dd>
						<dt class="text-muted-foreground">{t('links.details.fields.description')}</dt>
						<dd class="min-w-0 break-words whitespace-pre-line">
							{view.description || fmt.missing}
						</dd>
						<dt class="text-muted-foreground">{t('links.details.fields.tags')}</dt>
						<dd class="flex min-w-0 flex-wrap items-center gap-1.5">
							{#each viewTags as tag (tag.id)}
								<TagChip name={tag.name} color={tag.color} />
							{:else}
								<span class="text-muted-foreground">{t('links.details.noTags')}</span>
							{/each}
							<Button
								variant="link"
								class="h-auto px-1 text-[13px]"
								onclick={() => view && onaction('tags', view)}
							>
								{t('links.details.editTags')}
							</Button>
						</dd>
						{#if !view.isOwner}
							<dt class="text-muted-foreground">{t('links.details.fields.owner')}</dt>
							<dd class="min-w-0">
								{view.owner.displayName || view.owner.username}
								<span class="ml-1 font-mono text-muted-foreground">{view.owner.username}</span>
							</dd>
						{/if}
					</dl>
				</section>

				<section class="flex flex-col gap-3" aria-labelledby="link-details-usage">
					<h3 id="link-details-usage" class="kx-eyebrow text-muted-foreground">
						{t('links.details.sections.usage')}
					</h3>
					<!--
						One metric per row: a date in data-lg (32 px mono) needs about 250 px, more than a
						third of the 440 px sheet, so side-by-side cards would clip it.
					-->
					<div class="grid min-w-0 grid-cols-1 gap-2" data-testid="details-usage">
						<Metric
							label={t('links.details.metrics.usage')}
							value={fmt.number(view.useCount)}
							note={view.lastUsedAt ? fmt.relative(view.lastUsedAt) : t('links.details.neverUsed')}
						/>
						<Metric
							label={t('links.details.metrics.archive')}
							value={archiveMetric.value}
							note={archiveMetric.note || undefined}
						/>
						<Metric label={t('links.details.metrics.created')} value={fmt.date(view.createdAt)} />
					</div>
				</section>

				<section class="flex flex-col gap-3" aria-labelledby="link-details-times">
					<h3 id="link-details-times" class="kx-eyebrow text-muted-foreground">
						{t('links.details.sections.times')}
					</h3>
					<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-[13px] leading-5">
						<dt class="text-muted-foreground">{t('links.details.times.created')}</dt>
						<dd class="font-mono">{time(view.createdAt)}</dd>
						<dt class="text-muted-foreground">{t('links.details.times.updated')}</dt>
						<dd class="font-mono">{time(view.updatedAt)}</dd>
						<dt class="text-muted-foreground">{t('links.details.times.lastUsed')}</dt>
						<dd class="font-mono">{time(view.lastUsedAt)}</dd>
						{#if !view.isOwner}
							<dt class="text-muted-foreground">{t('links.details.times.sharedAt')}</dt>
							<dd class="font-mono">{time(view.sharedAt)}</dd>
						{/if}
						<dt class="text-muted-foreground">{t('links.details.times.inactiveSince')}</dt>
						<dd class="font-mono">{time(view.inactiveSince)}</dd>
						<dt class="text-muted-foreground">{t('links.details.times.archivedAt')}</dt>
						<dd class="font-mono">
							{#if view.archivedAt}
								{t('links.details.archivedWith', {
									date: fmt.dateTime(view.archivedAt),
									reason: t(
										view.archiveReason === 'auto'
											? 'links.details.archiveReason.auto'
											: 'links.details.archiveReason.manual'
									)
								})}
							{:else}
								{fmt.missing}
							{/if}
						</dd>
					</dl>
				</section>

				<section class="flex flex-col gap-3" aria-labelledby="link-details-metadata">
					<h3 id="link-details-metadata" class="kx-eyebrow text-muted-foreground">
						{t('links.details.sections.metadata')}
					</h3>
					<div class="flex flex-wrap items-center gap-2" data-testid="details-fetch">
						<StatusBadge
							status={fetchStatus[view.fetchStatus].status}
							label={t(fetchStatus[view.fetchStatus].label)}
						/>
						{#if view.fetchError}
							<span class="text-[13px] leading-5 text-muted-foreground">
								{tDynamic(`fetch.errors.${view.fetchError}`, undefined, view.fetchError)}
							</span>
						{/if}
					</div>
					<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-[13px] leading-5">
						<dt class="text-muted-foreground">{t('links.details.fetchedAt')}</dt>
						<dd class="font-mono">{time(view.fetchedAt)}</dd>
						<dt class="text-muted-foreground">{t('links.details.attempts')}</dt>
						<dd class="font-mono">{fmt.number(view.fetchAttempts)}</dd>
					</dl>
					{#if canRun('refresh', view, me)}
						<Button
							variant="outline"
							class={cn(buttonSize.sm, 'self-start')}
							disabled={view.fetchStatus === 'pending'}
							onclick={() => view && void refreshLink(view, store)}
						>
							<RefreshCw class="size-4" strokeWidth={1.5} aria-hidden="true" />
							{t('links.actions.refresh')}
						</Button>
					{/if}
				</section>

				{#if canShare}
					<section class="flex flex-col gap-3" aria-labelledby="link-details-sharing">
						<h3 id="link-details-sharing" class="kx-eyebrow text-muted-foreground">
							{t('links.details.sections.sharing')}
						</h3>
						{#if view.shares.length === 0}
							<p class="body-sm text-muted-foreground">{t('links.share.empty')}</p>
						{:else}
							<ul class="flex flex-col gap-1.5 text-[13px] leading-5">
								{#each view.shares as share (share.id)}
									<li class="flex items-center gap-2">
										{#if share.targetType === 'all'}
											<Globe
												class="size-4 text-muted-foreground"
												strokeWidth={1.5}
												aria-hidden="true"
											/>
										{:else if share.targetType === 'role'}
											<Users
												class="size-4 text-muted-foreground"
												strokeWidth={1.5}
												aria-hidden="true"
											/>
										{:else}
											<UserIcon
												class="size-4 text-muted-foreground"
												strokeWidth={1.5}
												aria-hidden="true"
											/>
										{/if}
										<span class="truncate">{shareText(share)}</span>
										{#if share.targetUser}
											<span class="font-mono text-muted-foreground">
												{share.targetUser.username}
											</span>
										{/if}
									</li>
								{/each}
							</ul>
						{/if}
						<Button
							variant="outline"
							class={cn(buttonSize.sm, 'self-start')}
							onclick={() => view && onaction('share', view)}
						>
							<Share2 class="size-4" strokeWidth={1.5} aria-hidden="true" />
							{t('links.details.manageShares')}
						</Button>
					</section>
				{/if}
			</div>

			<Sheet.Footer class="mt-auto flex-row flex-wrap gap-2 border-t px-6 py-4">
				{#if view.isOwner}
					<Button
						variant="outline"
						class={buttonSize.sm}
						onclick={() => view && onaction('edit', view)}
					>
						<Pencil class="size-4" strokeWidth={1.5} aria-hidden="true" />
						{t('links.actions.edit')}
					</Button>
				{/if}
				<Button
					variant="outline"
					class={buttonSize.sm}
					onclick={async () => view && apply(await toggleArchive(view, store))}
				>
					{#if view.archivedAt}
						<ArchiveRestore class="size-4" strokeWidth={1.5} aria-hidden="true" />
						{t('links.actions.restore')}
					{:else}
						<Archive class="size-4" strokeWidth={1.5} aria-hidden="true" />
						{t('links.actions.archive')}
					{/if}
				</Button>
				{#if view.isOwner}
					<Button
						variant="ghost"
						class={cn(buttonSize.sm, 'ml-auto text-destructive hover:text-destructive')}
						onclick={() => view && onaction('delete', view)}
					>
						<Trash2 class="size-4" strokeWidth={1.5} aria-hidden="true" />
						{t('links.details.deleteLink')}
					</Button>
				{/if}
			</Sheet.Footer>
		{/if}
	</Sheet.Content>
</Sheet.Root>
