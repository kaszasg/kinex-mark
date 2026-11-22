<script lang="ts">
	import { toggleItem } from '$lib/components/app/styles.js';
	import * as ToggleGroup from '$lib/components/ui/toggle-group/index.js';
	import { cn } from '$lib/utils.js';

	/** A segmented single-select group with three items, as the toolbar renders it (tests only). */
	let value = $state('a');
</script>

<ToggleGroup.Root
	type="single"
	variant="outline"
	bind:value={() => value, (next) => next && (value = next)}
>
	{#each ['a', 'b', 'c'] as item (item)}
		<ToggleGroup.Item value={item} class={cn(toggleItem, 'h-9 px-3')}>{item}</ToggleGroup.Item>
	{/each}
</ToggleGroup.Root>
