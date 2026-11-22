<script lang="ts">
	import type { Tag } from '$lib/api/types.js';
	import TagChip from '$lib/components/app/TagChip.svelte';
	import { t } from '$lib/i18n/index.js';
	import { tags as tagsStore } from '$lib/state/tags.svelte.js';
	import { cn } from '$lib/utils.js';
	import Hint from './Hint.svelte';

	/** The viewer's tags on a link: the first `max` chips, then a "+n" chip naming the rest. */
	type Props = {
		tagIds: number[];
		max: number;
		byId?: Map<number, Tag>;
		class?: string;
		/** Width limit of one chip: list rows cap it; tiles let a chip use the whole line. */
		chipClass?: string;
	};
	let { tagIds, max, byId, class: className, chipClass = 'max-w-32' }: Props = $props();

	const resolved = $derived(
		tagIds
			.map((id) => (byId ?? tagsStore.byId).get(id))
			.filter((tag): tag is Tag => tag !== undefined)
	);
	const shown = $derived(resolved.slice(0, max));
	const rest = $derived(resolved.slice(max));
	const restNames = $derived(rest.map((tag) => tag.name).join(', '));
</script>

{#if resolved.length > 0}
	<span class={cn('flex min-w-0 items-center gap-1', className)}>
		{#each shown as tag (tag.id)}
			<TagChip name={tag.name} color={tag.color} class={chipClass} />
		{/each}
		{#if rest.length > 0}
			<Hint text={t('links.row.moreTags', { names: restNames })}>
				{#snippet children(props)}
					<span
						{...props}
						class="inline-flex h-[22px] shrink-0 items-center rounded-sm border px-1.5 font-mono text-[11px] text-muted-foreground"
						data-more-tags={rest.length}
					>
						+{rest.length}
						<span class="sr-only">{t('links.row.moreTags', { names: restNames })}</span>
					</span>
				{/snippet}
			</Hint>
		{/if}
	</span>
{/if}
