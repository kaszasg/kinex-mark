<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils.js';

	/** Page heading of plan 8.4: heading-1 title, body-sm subtitle, right-aligned actions. */
	type Props = { title: string; subtitle?: string; actions?: Snippet; class?: string };
	let { title, subtitle, actions, class: className }: Props = $props();
</script>

<header class={cn('flex flex-wrap items-end justify-between gap-x-6 gap-y-3', className)}>
	<div class="flex min-w-0 flex-col gap-1">
		<h1 class="heading-1 text-balance">{title}</h1>
		{#if subtitle}<p class="body-sm text-muted-foreground">{subtitle}</p>{/if}
	</div>
	{#if actions}
		<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>
	{/if}
</header>
