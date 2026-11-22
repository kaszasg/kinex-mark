<script lang="ts">
	import { qrMatrix, qrPath } from '$lib/utils/qr.js';
	import { cn } from '$lib/utils.js';

	/**
	 * QR code of plan 8.4: one SVG path in currentColor with a 4-module quiet zone. The box is
	 * theme independent (mist-50 on ink-950, GRAPHICS.md 2.3 G7) so every scanner reads it.
	 */
	type Props = { value: string; label: string; size?: number; class?: string };
	let { value, label, size = 192, class: className }: Props = $props();

	const matrix = $derived(qrMatrix(value));
	const path = $derived(qrPath(matrix));
</script>

<div class={cn('inline-block rounded-md bg-mist-50 text-ink-950', className)}>
	<svg
		role="img"
		aria-label={label}
		width={size}
		height={size}
		viewBox="0 0 {matrix.size} {matrix.size}"
		shape-rendering="crispEdges"
	>
		<path d={path} fill="currentColor" />
	</svg>
</div>
