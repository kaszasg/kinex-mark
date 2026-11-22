<script lang="ts">
	import Inbox from '@lucide/svelte/icons/inbox';
	import type { Component, Snippet } from 'svelte';
	import * as Empty from '$lib/components/ui/empty/index.js';
	import { cn } from '$lib/utils.js';
	import { typeBodySm, typeHeading3 } from './styles.js';

	/** Empty state of plan 8.4: 24 px muted icon, heading-3 title, body-sm text, optional action. */
	type Props = {
		title: string;
		text?: string;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		icon?: Component<any>;
		action?: Snippet;
		class?: string;
	};
	let { title, text, icon: Icon = Inbox, action, class: className }: Props = $props();
</script>

<Empty.Root class={cn('flex-none rounded-lg border border-dashed px-6 py-12', className)}>
	<Empty.Header class="max-w-[var(--layout-prose)] gap-2">
		<Empty.Media class="mb-1 text-muted-foreground">
			<Icon class="size-6" strokeWidth={1.5} aria-hidden="true" />
		</Empty.Media>
		<Empty.Title class={typeHeading3}>{title}</Empty.Title>
		{#if text}
			<Empty.Description class={cn(typeBodySm, 'text-muted-foreground')}>{text}</Empty.Description>
		{/if}
	</Empty.Header>
	{#if action}
		<Empty.Content>{@render action()}</Empty.Content>
	{/if}
</Empty.Root>
