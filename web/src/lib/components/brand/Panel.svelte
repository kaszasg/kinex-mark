<script lang="ts">
	import type { Snippet } from 'svelte';

	/** Chamfered panel. The clip-path cuts overflowing content, so no popover or focus ring may sit on it. */
	type Props = { size?: 'sm' | 'md' | 'lg'; class?: string; children: Snippet };
	let { size = 'md', class: className = '', children }: Props = $props();
	const sizes = { sm: 'kx-panel-sm', md: '', lg: 'kx-panel-lg' };
</script>

<div class="kx-panel {sizes[size]} {className}">
	{@render children()}
</div>
