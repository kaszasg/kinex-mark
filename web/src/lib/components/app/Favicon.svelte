<script lang="ts">
	import LinkIcon from '@lucide/svelte/icons/link';
	import { cn } from '$lib/utils.js';

	/** Site icon of plan 8.4; a missing or broken icon falls back to the Lucide link icon. */
	type Props = { src: string | null | undefined; size?: 16 | 20 | 32; class?: string };
	let { src, size = 16, class: className }: Props = $props();

	let failedFor = $state<string | null>(null);
	const show = $derived(Boolean(src) && failedFor !== src);
	const sizes = { 16: 'size-4', 20: 'size-5', 32: 'size-8' } as const;
</script>

{#if show}
	<img
		{src}
		alt=""
		width={size}
		height={size}
		loading="lazy"
		decoding="async"
		referrerpolicy="no-referrer"
		class={cn('shrink-0 rounded-sm object-contain', sizes[size], className)}
		onerror={() => (failedFor = src ?? null)}
	/>
{:else}
	<LinkIcon
		class={cn('shrink-0 text-muted-foreground', sizes[size], className)}
		strokeWidth={1.5}
		aria-hidden="true"
	/>
{/if}
