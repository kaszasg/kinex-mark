<script lang="ts" module>
	/** Short loads finish without flicker: the skeleton appears only after this delay. */
	export const LOADING_DELAY_MS = 150;
</script>

<script lang="ts">
	import { onMount } from 'svelte';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';

	/** Skeleton rows while a list loads (plan 8.4); announced once through a polite status. */
	type Props = { rows?: number; rowClass?: string; delay?: number; class?: string };
	let { rows = 5, rowClass = 'h-10', delay = LOADING_DELAY_MS, class: className }: Props = $props();

	let visible = $state(false);

	onMount(() => {
		if (delay <= 0) {
			visible = true;
			return;
		}
		const timer = setTimeout(() => (visible = true), delay);
		return () => clearTimeout(timer);
	});
</script>

<div role="status" aria-live="polite" class={cn('flex flex-col gap-2', className)}>
	<span class="sr-only">{t('common.states.loading')}</span>
	{#if visible}
		{#each { length: rows }, index (index)}
			<Skeleton data-testid="loading-row" class={cn('w-full rounded-md', rowClass)} />
		{/each}
	{/if}
</div>
