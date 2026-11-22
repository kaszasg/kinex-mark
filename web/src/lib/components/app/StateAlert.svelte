<script lang="ts" module>
	export type AlertKind = 'critical' | 'warning' | 'success' | 'info';
</script>

<script lang="ts">
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Info from '@lucide/svelte/icons/info';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import type { Snippet } from 'svelte';
	import Button from './Button.svelte';
	import { cn } from '$lib/utils.js';
	import { buttonSize } from './styles.js';

	/**
	 * In-page message of GRAPHICS.md 10.1: the card surface stays, only the icon is coloured.
	 * The title says what happened, the text what to do. Only critical alerts get role="alert".
	 */
	type Props = {
		kind: AlertKind;
		title: string;
		text?: string;
		actionLabel?: string;
		onAction?: () => void;
		actionDisabled?: boolean;
		action?: Snippet;
		class?: string;
	};
	let {
		kind,
		title,
		text,
		actionLabel,
		onAction,
		actionDisabled = false,
		action,
		class: className
	}: Props = $props();

	const icons = {
		critical: { icon: CircleAlert, color: 'text-destructive' },
		warning: { icon: TriangleAlert, color: 'text-warning' },
		success: { icon: CircleCheck, color: 'text-success' },
		info: { icon: Info, color: 'text-info' }
	} as const;
	const item = $derived(icons[kind]);
	const hasAction = $derived(Boolean(action) || Boolean(actionLabel && onAction));
</script>

<div
	data-kind={kind}
	role={kind === 'critical' ? 'alert' : undefined}
	class={cn(
		'grid w-full items-start gap-x-3 gap-y-0.5 rounded-lg border bg-card px-4 py-3 text-card-foreground',
		hasAction ? 'grid-cols-[16px_1fr_auto]' : 'grid-cols-[16px_1fr]',
		className
	)}
>
	<item.icon class="row-span-2 mt-0.5 size-4 {item.color}" strokeWidth={1.5} aria-hidden="true" />
	<p class="col-start-2 text-[13px] leading-5 font-semibold">{title}</p>
	{#if hasAction}
		<div class="col-start-3 row-span-2 row-start-1 self-center">
			{#if action}
				{@render action()}
			{:else}
				<Button
					variant="outline"
					size="sm"
					class={buttonSize.sm}
					disabled={actionDisabled}
					onclick={onAction}
				>
					{actionLabel}
				</Button>
			{/if}
		</div>
	{/if}
	{#if text}
		<p class="col-start-2 text-[13px] leading-5 text-muted-foreground">{text}</p>
	{/if}
</div>
