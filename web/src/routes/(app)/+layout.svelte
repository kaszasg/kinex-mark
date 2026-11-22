<script lang="ts">
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/app/AppShell.svelte';
	import { linkStats } from '$lib/state/link-stats.svelte.js';

	let { children } = $props();

	// The sidebar's link counter refreshes while the signed-in frame is mounted.
	onMount(() => linkStats.start());
</script>

<AppShell>
	{@render children()}
</AppShell>
