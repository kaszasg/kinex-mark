<script lang="ts" module>
	/** Ids a row hands to its control: the label (for aria-labelledby) and the help text. */
	export type SettingRowIds = { labelId: string; helpId: string | undefined };
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Label } from '$lib/components/ui/label/index.js';
	import { cn } from '$lib/utils.js';

	/**
	 * One setting: label and help on the left, the control on the right (stacked on narrow
	 * screens, or always when `stacked`). `controlId` makes the label a <label for>; otherwise
	 * the control refers to the label through aria-labelledby.
	 */
	type Props = {
		label: string;
		help?: string;
		controlId?: string;
		stacked?: boolean;
		/** Vertical alignment of the two columns; `start` suits a control that grows. */
		align?: 'center' | 'start';
		class?: string;
		children: Snippet<[SettingRowIds]>;
	};
	let {
		label,
		help,
		controlId,
		stacked = false,
		align = 'center',
		class: className,
		children
	}: Props = $props();

	const id = $props.id();
	const ids = $derived<SettingRowIds>({
		labelId: `${id}-label`,
		helpId: help ? `${id}-help` : undefined
	});
</script>

<div
	class={cn(
		'grid gap-x-6 gap-y-2 px-4 py-3',
		!stacked && 'sm:grid-cols-[minmax(0,1fr)_auto]',
		!stacked && (align === 'start' ? 'sm:items-start' : 'sm:items-center'),
		className
	)}
>
	<div class="flex min-w-0 flex-col gap-0.5">
		{#if controlId}
			<Label id={ids.labelId} for={controlId} class="text-[13px] leading-5 font-medium">
				{label}
			</Label>
		{:else}
			<span id={ids.labelId} class="text-[13px] leading-5 font-medium">{label}</span>
		{/if}
		{#if help}
			<p id={ids.helpId} class="text-[13px] leading-5 text-muted-foreground">{help}</p>
		{/if}
	</div>
	<div class={cn('flex min-w-0 flex-col gap-2', !stacked && 'sm:items-end')}>
		{@render children(ids)}
	</div>
</div>
