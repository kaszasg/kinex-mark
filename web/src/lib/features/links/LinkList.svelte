<script lang="ts">
	import type { LinkItem, Me } from '$lib/api/types.js';
	import { cardSurface } from '$lib/components/app/styles.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { t, type MessageKey } from '$lib/i18n/index.js';
	import type { DisplayFields } from '$lib/links/presentation.js';
	import { links as linksStore, PAGE_SIZE, type LinksStore } from '$lib/state/links.svelte.js';
	import { cn } from '$lib/utils.js';
	import type { LinkActionHandler } from './actions.js';
	import LinkRow from './LinkRow.svelte';
	import LinkTile from './LinkTile.svelte';

	/**
	 * The loaded links as a list (one card of hairline rows) or as tiles. With usage order the
	 * pinned and the other links get a group heading, which makes the ranking rule visible; the
	 * pinned links come first in every order. A sentinel loads the next page.
	 */
	type Props = {
		me: Me | null;
		fields: DisplayFields;
		view: 'list' | 'tiles';
		effectiveArchiveAfterDays: number;
		onaction: LinkActionHandler;
		store?: LinksStore;
	};
	let {
		me,
		fields,
		view,
		effectiveArchiveAfterDays,
		onaction,
		store = linksStore
	}: Props = $props();

	type Group = { key: string; heading: MessageKey | null; items: LinkItem[] };

	const groups = $derived.by((): Group[] => {
		if (store.query.sort === 'usage' && store.pinned.length > 0) {
			const split: Group[] = [
				{ key: 'pinned', heading: 'links.groups.pinned', items: store.pinned },
				{ key: 'others', heading: 'links.groups.others', items: store.others }
			];
			return split.filter((group) => group.items.length > 0);
		}
		return [{ key: 'all', heading: null, items: store.items }];
	});
	const hasMore = $derived(store.items.length < store.total);
	const allLoaded = $derived(!hasMore && store.total > PAGE_SIZE);

	let sentinel = $state<HTMLElement | null>(null);

	$effect(() => {
		if (!sentinel || typeof IntersectionObserver === 'undefined') return;
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries.some((entry) => entry.isIntersecting)) void store.loadMore();
			},
			{ rootMargin: '400px 0px' }
		);
		observer.observe(sentinel);
		return () => observer.disconnect();
	});
</script>

<div class="flex flex-col gap-4" aria-busy={store.status === 'loading'}>
	{#if view === 'list'}
		<Card.Root class={cn(cardSurface, 'overflow-hidden')}>
			{#each groups as group, index (group.key)}
				<section
					aria-labelledby={group.heading ? `links-group-${group.key}` : undefined}
					aria-label={group.heading ? undefined : t('links.list.label')}
					class={cn(index > 0 && 'border-t')}
				>
					{#if group.heading}
						<h2
							id="links-group-{group.key}"
							class="kx-eyebrow border-b px-4 pt-3 pb-2 text-muted-foreground"
						>
							{t(group.heading)}
						</h2>
					{/if}
					<ul>
						{#each group.items as link (link.id)}
							<LinkRow {link} {me} {fields} {effectiveArchiveAfterDays} {onaction} {store} />
						{/each}
					</ul>
				</section>
			{/each}
		</Card.Root>
	{:else}
		{#each groups as group (group.key)}
			<section
				aria-labelledby={group.heading ? `links-group-${group.key}` : undefined}
				aria-label={group.heading ? undefined : t('links.list.label')}
				class="flex flex-col gap-2"
			>
				{#if group.heading}
					<h2 id="links-group-{group.key}" class="kx-eyebrow text-muted-foreground">
						{t(group.heading)}
					</h2>
				{/if}
				<ul class="grid grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-3">
					{#each group.items as link (link.id)}
						<LinkTile {link} {me} {fields} {effectiveArchiveAfterDays} {onaction} {store} />
					{/each}
				</ul>
			</section>
		{/each}
	{/if}

	{#if hasMore}
		<div bind:this={sentinel} class="h-px" aria-hidden="true" data-testid="links-sentinel"></div>
	{/if}
	{#if store.status === 'loading-more'}
		<div role="status" class="body-sm flex items-center gap-2 text-muted-foreground">
			<Spinner class="size-4" aria-label={t('common.states.loading')} />
			{t('common.states.loadingMore')}
		</div>
	{:else if allLoaded}
		<p class="body-sm text-muted-foreground">{t('links.list.allLoaded')}</p>
	{/if}
</div>
