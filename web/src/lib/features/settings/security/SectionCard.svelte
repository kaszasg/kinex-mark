<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import { cardSurface } from '$lib/components/app/styles.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Settings block of plan 8.7.6: a flat card with a hairline head (title, optional status or
	 * action on the right) and the content below. Shared by the Security and Privacy tabs.
	 */
	type Props = {
		id: string;
		title: string;
		description?: string;
		aside?: Snippet;
		class?: string;
		children: Snippet;
	};
	let { id, title, description, aside, class: className, children }: Props = $props();

	const titleId = $derived(`${id}-title`);
</script>

<section {id} aria-labelledby={titleId} class={cn('min-w-0', className)}>
	<Card.Root class={cardSurface}>
		<header class="flex flex-wrap items-start justify-between gap-3 border-b px-4 py-3">
			<div class="flex min-w-0 flex-col gap-0.5">
				<h2 id={titleId} class="text-[13px] leading-5 font-semibold">{title}</h2>
				{#if description}
					<p class="body-sm text-muted-foreground">{description}</p>
				{/if}
			</div>
			{#if aside}
				<div class="flex shrink-0 items-center gap-2">{@render aside()}</div>
			{/if}
		</header>
		<div class="flex flex-col gap-4 p-4">
			{@render children()}
		</div>
	</Card.Root>
</section>
