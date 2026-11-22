<script lang="ts">
	import type { Snippet } from 'svelte';
	import { setSidebar } from '$lib/components/ui/sidebar/context.svelte.js';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';
	import { cn } from '$lib/utils.js';

	/**
	 * The sidebar context for the generated sidebar parts, in place of the generated provider
	 * component. Our sidebar never collapses (collapsible="none", the narrow layout uses the
	 * shell's own sheet), so the open state is fixed. The generated provider would store that state
	 * in a persistent "sidebar_state" cookie and take over Ctrl/Cmd+B in every text field; the
	 * application sets no cookie besides the session cookie (plan D9, 6.14).
	 */
	type Props = { class?: string; style?: string; children: Snippet };
	let { class: className, style, children }: Props = $props();

	setSidebar({ open: () => true, setOpen: () => {} });
</script>

<Tooltip.Provider delayDuration={0}>
	<div
		data-slot="sidebar-wrapper"
		{style}
		class={cn('group/sidebar-wrapper flex min-h-svh w-full', className)}
	>
		{@render children()}
	</div>
</Tooltip.Provider>
