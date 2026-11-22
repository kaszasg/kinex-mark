<script lang="ts">
	import LoginScreen from '$lib/features/auth/LoginScreen.svelte';
	import type { PageProps } from './$types';

	/** Sign-in page (plan 8.7.2, 8.8); the composition lives in LoginScreen. */
	let { data }: PageProps = $props();
</script>

<LoginScreen next={data.next} reason={data.reason} />
