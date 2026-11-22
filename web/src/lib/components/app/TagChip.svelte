<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { TagColor } from '$lib/api/types.js';
	import { tagSwatchClass } from '$lib/tags/colors.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Tag chip of plan 8.4: 22 px high, 1 px border, rounded-sm, 8x8 px square swatch.
	 * As a button it is a toggle (aria-pressed, bg-accent when active); as a link it navigates.
	 */
	type Props = {
		name: string;
		color: TagColor;
		as?: 'span' | 'button' | 'a';
		href?: string;
		pressed?: boolean;
		onclick?: (event: MouseEvent) => void;
		trailing?: Snippet;
		class?: string;
	};
	let {
		name,
		color,
		as = 'span',
		href,
		pressed = false,
		onclick,
		trailing,
		class: className
	}: Props = $props();

	const base = $derived(
		cn(
			'inline-flex h-[22px] max-w-full min-w-0 items-center gap-1.5 rounded-sm border px-1.5 text-[13px] leading-5 text-foreground',
			as !== 'span' &&
				'transition-colors hover:bg-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring',
			as === 'button' && pressed && 'border-ring bg-accent',
			className
		)
	);
</script>

{#snippet body()}
	<span class="size-2 shrink-0 {tagSwatchClass[color]}" aria-hidden="true"></span>
	<span class="truncate">{name}</span>
	{#if trailing}{@render trailing()}{/if}
{/snippet}

{#if as === 'button'}
	<button type="button" class={base} aria-pressed={pressed} {onclick}>{@render body()}</button>
{:else if as === 'a'}
	<a {href} class={base} {onclick}>{@render body()}</a>
{:else}
	<span class={base}>{@render body()}</span>
{/if}
