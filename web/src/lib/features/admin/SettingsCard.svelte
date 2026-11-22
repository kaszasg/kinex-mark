<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cardSurface } from '$lib/components/app/styles.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { cn } from '$lib/utils.js';

	/** One block of the system settings form: card head with title and text (GRAPHICS.md 10.0). */
	type Props = { title: string; text: string; contentClass?: string; children: Snippet };
	let { title, text, contentClass, children }: Props = $props();
</script>

<Card.Root class={cardSurface}>
	<Card.Header class="gap-1 border-b px-4 py-3">
		<Card.Title class="text-[13px] leading-5 font-semibold">{title}</Card.Title>
		<Card.Description class="text-[13px] leading-5 text-muted-foreground">{text}</Card.Description>
	</Card.Header>
	<Card.Content class={cn('flex flex-col gap-5 p-4', contentClass)}>
		{@render children()}
	</Card.Content>
</Card.Root>
