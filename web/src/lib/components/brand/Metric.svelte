<script lang="ts">
	import Panel from './Panel.svelte';

	/** A single measured value. `value` is already formatted text; rounding is the caller's job. */
	type Props = { label: string; value: string; unit?: string; note?: string };
	let { label, value, unit, note }: Props = $props();
</script>

<Panel size="sm">
	<div class="flex flex-col gap-2 p-4">
		<span class="kx-eyebrow">{label}</span>
		<span class="data-lg">
			{value}{#if unit}<small class="text-muted-foreground ml-1 text-sm tracking-normal">
					{unit}
				</small>{/if}
		</span>
		{#if note}<span class="body-sm text-muted-foreground">{note}</span>{/if}
	</div>
</Panel>
