<script lang="ts">
	import { page } from '$app/state';
	import AppShell from '$lib/components/app/AppShell.svelte';
	import ErrorView from '$lib/components/app/ErrorView.svelte';
	import { session } from '$lib/state/session.svelte.js';
</script>

{#if session.me && !session.gate}
	<AppShell>
		<ErrorView status={page.status} />
	</AppShell>
{:else}
	<div class="flex min-h-dvh items-center justify-center bg-background p-4 md:p-6">
		<main id="main" tabindex="-1" class="w-full max-w-[var(--layout-prose)] outline-none">
			<ErrorView status={page.status} />
		</main>
	</div>
{/if}
