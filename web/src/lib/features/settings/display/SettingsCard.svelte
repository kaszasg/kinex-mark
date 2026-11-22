<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import { cardSurface } from '$lib/components/app/styles.js';
	import { cn } from '$lib/utils.js';

	/**
	 * A settings block (plan 8.7.6): flat card with a hairline head (title 13/20 semibold and an
	 * optional muted description), hairline-separated rows, and an optional footer for actions.
	 */
	type Props = {
		title: string;
		description?: string;
		class?: string;
		children: Snippet;
		footer?: Snippet;
	};
	let { title, description, class: className, children, footer }: Props = $props();

	const id = $props.id();
</script>

<Card.Root class={cn(cardSurface, 'overflow-hidden')}>
	<section aria-labelledby={`${id}-title`} class={className}>
		<header class="flex flex-col gap-0.5 border-b px-4 py-3">
			<h2 id={`${id}-title`} class="text-[13px] leading-5 font-semibold">{title}</h2>
			{#if description}
				<p class="text-[13px] leading-5 text-muted-foreground">{description}</p>
			{/if}
		</header>
		<div class="flex flex-col divide-y">
			{@render children()}
		</div>
		{#if footer}
			<footer class="flex flex-wrap items-center justify-end gap-2 border-t px-4 py-3">
				{@render footer()}
			</footer>
		{/if}
	</section>
</Card.Root>
