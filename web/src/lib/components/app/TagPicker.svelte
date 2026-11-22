<script lang="ts" module>
	/** The server limit of tags on one link (fields.byField.tagIds.too_long). */
	export const MAX_TAGS_PER_LINK = 20;
</script>

<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import Tags from '@lucide/svelte/icons/tags';
	import X from '@lucide/svelte/icons/x';
	import { toast } from 'svelte-sonner';
	import type { Tag } from '$lib/api/types.js';
	import Button from './Button.svelte';
	import * as Command from '$lib/components/ui/command/index.js';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import { t } from '$lib/i18n/index.js';
	import { tags as tagsStore, type TagsStore } from '$lib/state/tags.svelte.js';
	import { cn } from '$lib/utils.js';
	import TagChip from './TagChip.svelte';
	import { buttonSize } from './styles.js';
	import { toastError } from '$lib/utils/toast-error.js';

	/**
	 * Multi-select tag picker of plan 8.4: Popover + Command with search and a create row.
	 * `value` holds the selected tag ids in selection order.
	 */
	type Props = {
		value?: number[];
		max?: number;
		id?: string;
		disabled?: boolean;
		/** Allows creating a missing tag from the search text. */
		allowCreate?: boolean;
		store?: TagsStore;
		onchange?: (ids: number[]) => void;
		class?: string;
	};
	let {
		value = $bindable([]),
		max = MAX_TAGS_PER_LINK,
		id,
		disabled = false,
		allowCreate = true,
		store = tagsStore,
		onchange,
		class: className
	}: Props = $props();

	let open = $state(false);
	let search = $state('');
	let creating = $state(false);

	const selectedTags = $derived(
		value.map((tagId) => store.byId.get(tagId)).filter((tag): tag is Tag => tag !== undefined)
	);
	const needle = $derived(search.trim().toLowerCase());
	const matches = $derived(
		needle ? store.items.filter((tag) => tag.name.toLowerCase().includes(needle)) : store.items
	);
	const exact = $derived(store.items.some((tag) => tag.name.toLowerCase() === needle));
	const atLimit = $derived(value.length >= max);
	const canCreate = $derived(allowCreate && needle.length > 0 && !exact && !atLimit);

	function setValue(next: number[]): void {
		value = next;
		onchange?.(next);
	}

	function toggle(tagId: number): void {
		if (value.includes(tagId)) setValue(value.filter((item) => item !== tagId));
		else if (!atLimit) setValue([...value, tagId]);
	}

	async function create(): Promise<void> {
		if (!canCreate || creating) return;
		creating = true;
		try {
			const tag = await store.create(search.trim());
			setValue([...value, tag.id]);
			search = '';
		} catch (err) {
			toastError(err);
		} finally {
			creating = false;
		}
	}
</script>

<div class={cn('flex flex-wrap items-center gap-1.5', className)}>
	{#each selectedTags as tag (tag.id)}
		<TagChip name={tag.name} color={tag.color}>
			{#snippet trailing()}
				<button
					type="button"
					class="-mr-1 inline-flex size-4 items-center justify-center rounded-sm text-muted-foreground hover:text-foreground disabled:opacity-50"
					aria-label={t('common.tagPicker.remove', { name: tag.name })}
					{disabled}
					onclick={() => toggle(tag.id)}
				>
					<X class="size-3" strokeWidth={1.5} aria-hidden="true" />
				</button>
			{/snippet}
		</TagChip>
	{/each}
	<!-- Every opening starts with an empty search. -->
	<Popover.Root
		bind:open={
			() => open,
			(next: boolean) => {
				if (next && !open) search = '';
				open = next;
			}
		}
	>
		<Popover.Trigger {id} {disabled}>
			{#snippet child({ props })}
				<Button {...props} variant="outline" size="sm" class={buttonSize.sm}>
					<Tags class="size-4" strokeWidth={1.5} aria-hidden="true" />
					{t('common.tagPicker.label')}
				</Button>
			{/snippet}
		</Popover.Trigger>
		<Popover.Content class="w-72 p-0" align="start">
			<Command.Root shouldFilter={false}>
				<Command.Input
					bind:value={search}
					placeholder={t('common.tagPicker.search')}
					aria-label={t('common.tagPicker.search')}
					onkeydown={(event: KeyboardEvent) => {
						if (event.key === 'Enter' && canCreate && matches.length === 0) {
							event.preventDefault();
							void create();
						}
					}}
				/>
				<Command.List aria-label={t('common.tagPicker.label')}>
					{#if atLimit}
						<p class="px-3 py-2 text-[13px] leading-5 text-muted-foreground" role="status">
							{t('common.tagPicker.limit', { max })}
						</p>
					{/if}
					{#if matches.length === 0 && !canCreate}
						<p class="px-3 py-6 text-center text-[13px] leading-5 text-muted-foreground">
							{t('common.tagPicker.empty')}
						</p>
					{/if}
					{#if matches.length > 0}
						<Command.Group>
							{#each matches as tag (tag.id)}
								{@const checked = value.includes(tag.id)}
								<Command.Item
									value={String(tag.id)}
									data-checked={checked}
									aria-selected={checked}
									disabled={!checked && atLimit}
									onSelect={() => toggle(tag.id)}
								>
									<TagChip name={tag.name} color={tag.color} class="border-0 px-0" />
								</Command.Item>
							{/each}
						</Command.Group>
					{/if}
					{#if canCreate}
						<Command.Group>
							<Command.Item value="__create" disabled={creating} onSelect={() => void create()}>
								<Plus class="size-4" strokeWidth={1.5} aria-hidden="true" />
								<span class="truncate">
									{t('common.tagPicker.create', { name: search.trim() })}
								</span>
							</Command.Item>
						</Command.Group>
					{/if}
				</Command.List>
			</Command.Root>
		</Popover.Content>
	</Popover.Root>
</div>
