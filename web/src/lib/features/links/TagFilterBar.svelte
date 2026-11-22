<script lang="ts" module>
	/** At most this many tag chips stand in the bar; the rest are in "More tags". */
	export const VISIBLE_TAG_CHIPS = 12;
</script>

<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import Tags from '@lucide/svelte/icons/tags';
	import { MediaQuery } from 'svelte/reactivity';
	import type { Tag } from '$lib/api/types.js';
	import TagChip from '$lib/components/app/TagChip.svelte';
	import { buttonSize, toggleItem } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Command from '$lib/components/ui/command/index.js';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
	import * as ToggleGroup from '$lib/components/ui/toggle-group/index.js';
	import { t } from '$lib/i18n/index.js';
	import { defaultsFromPreferences, hasActiveFilters, MAX_FILTER_TAGS } from '$lib/links/query.js';
	import { links as linksStore, type LinksStore } from '$lib/state/links.svelte.js';
	import { preferences } from '$lib/state/preferences.svelte.js';
	import { tags as tagsStore, type TagsStore } from '$lib/state/tags.svelte.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Tag filter row (plan 8.7.3): grouping happens only by tags. Chips for the most used tags,
	 * "Untagged", the rest in a searchable popover, Any/All with two or more tags, and "Clear
	 * filters" while any filter is active.
	 */
	type Props = { store?: LinksStore; tagStore?: TagsStore; class?: string };
	let { store = linksStore, tagStore = tagsStore, class: className }: Props = $props();

	/** From 768 px the chips wrap; below it the row scrolls sideways (plan 8.9). */
	const wide = new MediaQuery('min-width: 768px', true);

	let moreOpen = $state(false);
	let search = $state('');

	const selected = $derived(store.query.tags);
	const ranked = $derived(
		[...tagStore.items].sort(
			(a, b) => b.activeLinkCount - a.activeLinkCount || a.name.localeCompare(b.name)
		)
	);
	/** The most used tags, plus every selected tag so an active filter always shows. */
	const chips = $derived.by(() => {
		const top = ranked.slice(0, VISIBLE_TAG_CHIPS);
		const extra = ranked.slice(VISIBLE_TAG_CHIPS).filter((tag) => selected.includes(tag.id));
		return [...top, ...extra];
	});
	const rest = $derived(ranked.filter((tag) => !chips.includes(tag)));
	const needle = $derived(search.trim().toLowerCase());
	const restMatches = $derived(
		needle ? rest.filter((tag) => tag.name.toLowerCase().includes(needle)) : rest
	);
	const active = $derived(hasActiveFilters(store.query));

	function toggleTag(tag: Tag): void {
		const next = selected.includes(tag.id)
			? selected.filter((id) => id !== tag.id)
			: [...selected, tag.id].slice(0, MAX_FILTER_TAGS);
		store.setQuery({ tags: next, untagged: false });
	}

	function toggleUntagged(): void {
		store.setQuery({ untagged: !store.query.untagged, tags: [] });
	}

	function setMatch(value: string): void {
		if (value !== 'any' && value !== 'all') return;
		preferences.update({ tagMatch: value });
		store.defaults = defaultsFromPreferences(preferences.current);
		store.setQuery({ tagMatch: value });
	}
</script>

{#snippet items()}
	{#each chips as tag (tag.id)}
		<TagChip
			as="button"
			name={tag.name}
			color={tag.color}
			pressed={selected.includes(tag.id)}
			class="shrink-0"
			onclick={() => toggleTag(tag)}
		/>
	{/each}
	{#if tagStore.items.length > 0}
		<button
			type="button"
			aria-pressed={store.query.untagged}
			class={cn(
				'inline-flex h-[22px] shrink-0 items-center rounded-sm border border-dashed px-1.5 text-[13px] leading-5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground',
				store.query.untagged && 'border-solid border-ring bg-accent text-foreground'
			)}
			onclick={toggleUntagged}
		>
			{t('links.filters.untagged')}
		</button>
	{/if}
	{#if rest.length > 0}
		<!-- Every opening starts with an empty search. -->
		<Popover.Root
			bind:open={
				() => moreOpen,
				(next: boolean) => {
					if (next && !moreOpen) search = '';
					moreOpen = next;
				}
			}
		>
			<Popover.Trigger>
				{#snippet child({ props })}
					<Button {...props} variant="ghost" class={cn(buttonSize.sm, 'shrink-0')}>
						<Tags class="size-4" strokeWidth={1.5} aria-hidden="true" />
						{t('links.filters.more', { count: rest.length })}
					</Button>
				{/snippet}
			</Popover.Trigger>
			<Popover.Content class="w-64 p-0" align="start">
				<Command.Root shouldFilter={false}>
					<Command.Input
						bind:value={search}
						placeholder={t('links.filters.search')}
						aria-label={t('links.filters.search')}
					/>
					<Command.List aria-label={t('links.filters.label')}>
						{#if restMatches.length === 0}
							<p class="px-3 py-6 text-center text-[13px] leading-5 text-muted-foreground">
								{t('links.filters.empty')}
							</p>
						{:else}
							<Command.Group>
								{#each restMatches as tag (tag.id)}
									{@const checked = selected.includes(tag.id)}
									<Command.Item
										value={String(tag.id)}
										aria-selected={checked}
										onSelect={() => toggleTag(tag)}
									>
										<TagChip name={tag.name} color={tag.color} class="border-0 px-0" />
										{#if checked}
											<Check class="ml-auto size-4" strokeWidth={1.5} aria-hidden="true" />
										{/if}
									</Command.Item>
								{/each}
							</Command.Group>
						{/if}
					</Command.List>
				</Command.Root>
			</Popover.Content>
		</Popover.Root>
	{/if}
	{#if selected.length >= 2}
		<ToggleGroup.Root
			type="single"
			variant="outline"
			bind:value={() => store.query.tagMatch, setMatch}
			aria-label={t('links.filters.match.label')}
			class="shrink-0"
		>
			<ToggleGroup.Item value="any" class={cn(toggleItem, 'h-7 px-2.5 text-[13px]')}>
				{t('links.filters.match.any')}
			</ToggleGroup.Item>
			<ToggleGroup.Item value="all" class={cn(toggleItem, 'h-7 px-2.5 text-[13px]')}>
				{t('links.filters.match.all')}
			</ToggleGroup.Item>
		</ToggleGroup.Root>
	{/if}
	{#if active}
		<Button
			variant="link"
			class="h-7 shrink-0 px-1 text-[13px]"
			onclick={() => store.clearFilters()}
		>
			{t('links.filters.clear')}
		</Button>
	{/if}
{/snippet}

{#if tagStore.items.length > 0 || active}
	{#if wide.current}
		<div
			role="group"
			aria-label={t('links.filters.label')}
			class={cn('flex flex-wrap items-center gap-2', className)}
		>
			{@render items()}
		</div>
	{:else}
		<!--
			Below 768 px the row scrolls sideways inside a ScrollArea (plan 8.9): a thin themed
			scrollbar instead of the native one. The padding keeps the 2 px focus ring and the
			scrollbar clear of the chips; the row bleeds to the screen edges.
		-->
		<ScrollArea orientation="horizontal" class={cn('-mx-4', className)} data-tag-filter-scroll>
			<div
				role="group"
				aria-label={t('links.filters.label')}
				class="flex w-max items-center gap-2 px-4 pt-1 pb-3.5"
			>
				{@render items()}
			</div>
		</ScrollArea>
	{/if}
{/if}
