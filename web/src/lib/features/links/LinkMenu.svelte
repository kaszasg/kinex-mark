<script lang="ts">
	import Ellipsis from '@lucide/svelte/icons/ellipsis';
	import type { LinkItem, Me } from '$lib/api/types.js';
	import { buttonSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';
	import { linkActions, type LinkActionHandler } from './actions.js';

	/** The ellipsis menu of a row or tile: the linkActions list of the viewer. */
	type Props = {
		link: LinkItem;
		me: Me | null;
		label: string;
		onaction: LinkActionHandler;
		class?: string;
	};
	let { link, me, label, onaction, class: className }: Props = $props();

	const actions = $derived(linkActions(link, me));
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button
				{...props}
				variant="ghost"
				size="icon"
				class={cn(buttonSize.iconSm, className)}
				aria-label={t('links.row.menu', { label })}
			>
				<Ellipsis class="size-4" strokeWidth={1.5} aria-hidden="true" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="min-w-56">
		{#each actions as action (action.id)}
			{#if action.destructive}
				<DropdownMenu.Separator />
			{/if}
			<DropdownMenu.Item
				class={cn(action.destructive && 'text-destructive data-highlighted:text-destructive')}
				onSelect={() => onaction(action.id, link)}
			>
				<action.icon
					class={cn('size-4', action.destructive && 'text-destructive')}
					strokeWidth={1.5}
					aria-hidden="true"
				/>
				{t(action.label)}
			</DropdownMenu.Item>
		{/each}
	</DropdownMenu.Content>
</DropdownMenu.Root>
