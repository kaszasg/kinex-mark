<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Table from '$lib/components/ui/table/index.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Table shell of GRAPHICS.md 10.7: card surface without chamfer, horizontal scroll on narrow
	 * screens, 13/20 text, hover accent, no border under the last row. `caption` names the table.
	 */
	type Props = { caption?: string; class?: string; children: Snippet };
	let { caption, class: className, children }: Props = $props();
</script>

<div
	class={cn(
		'w-full overflow-x-auto rounded-lg border bg-card text-card-foreground [&_[data-slot=table-container]]:overflow-visible',
		className
	)}
>
	<Table.Root
		class="border-collapse text-[13px] leading-5 [&_tbody_tr:hover]:bg-accent [&_tbody_tr:last-child]:border-0"
	>
		{#if caption}<caption class="sr-only">{caption}</caption>{/if}
		{@render children()}
	</Table.Root>
</div>
