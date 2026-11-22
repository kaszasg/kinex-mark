<script lang="ts">
	import { page } from '$app/state';
	import LocaleSwitcher from '$lib/components/app/LocaleSwitcher.svelte';
	import ThemeToggle from '$lib/components/app/ThemeToggle.svelte';
	import Logo from '$lib/components/brand/Logo.svelte';
	import { t } from '$lib/i18n/index.js';

	/**
	 * Public pages. The sign-in page draws its own split layout (F1); the other public pages
	 * (privacy notice) get a slim top bar with the logo, language and theme.
	 */
	let { children } = $props();

	const bare = $derived(page.url.pathname === '/login');
</script>

{#if bare}
	{@render children()}
{:else}
	<div class="flex min-h-dvh flex-col bg-background">
		<header class="flex h-[var(--layout-topbar)] shrink-0 items-center gap-3 border-b px-4 md:px-6">
			<a
				href="/"
				class="flex items-center gap-2 rounded-md text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
			>
				<Logo variant="mark" height={24} />
				<span class="text-[15px] leading-6 font-medium">{t('common.appName')}</span>
			</a>
			<div class="ml-auto flex items-center gap-1">
				<LocaleSwitcher />
				<ThemeToggle />
			</div>
		</header>
		<main
			id="main"
			tabindex="-1"
			class="mx-auto flex w-full max-w-[var(--layout-prose)] flex-1 flex-col gap-4 px-4 py-6 outline-none md:py-10"
		>
			{@render children()}
		</main>
	</div>
{/if}
