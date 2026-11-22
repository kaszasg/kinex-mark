<script lang="ts">
	import type { Snippet } from 'svelte';
	import { tooltipSurface } from '$lib/components/app/styles.js';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';
	import { cn } from '$lib/utils.js';

	/**
	 * A tooltip around one element: `children` renders the trigger element and spreads the
	 * given props on it. Carries its own provider so rows and tiles render standalone.
	 */
	type Props = { text: string; children: Snippet<[Record<string, unknown>]> };
	let { text, children }: Props = $props();
</script>

<Tooltip.Provider delayDuration={300}>
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				{@render children(props)}
			{/snippet}
		</Tooltip.Trigger>
		<!-- GRAPHICS.md 2.3/5.4: a popover surface, not nova's inverted fill; no arrow. -->
		<Tooltip.Content
			class={cn(tooltipSurface, 'max-w-xs text-[13px] leading-5')}
			arrowClasses="hidden"
		>
			{text}
		</Tooltip.Content>
	</Tooltip.Root>
</Tooltip.Provider>
