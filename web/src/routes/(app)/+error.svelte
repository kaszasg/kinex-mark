<script lang="ts">
	// Errors below the signed-in frame (for example the admin 403) render inside the AppShell.
	import { page } from '$app/state';
	import ErrorView from '$lib/components/app/ErrorView.svelte';
</script>

<ErrorView status={page.status} />
