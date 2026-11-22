<script lang="ts">
	import LogOut from '@lucide/svelte/icons/log-out';
	import LocaleSwitcher from '$lib/components/app/LocaleSwitcher.svelte';
	import ThemeToggle from '$lib/components/app/ThemeToggle.svelte';
	import { buttonSize } from '$lib/components/app/styles.js';
	import Logo from '$lib/components/brand/Logo.svelte';
	import TickRule from '$lib/components/brand/TickRule.svelte';
	import Button from '$lib/components/app/Button.svelte';
	import { t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';

	/** Gate layout of plan 8.7.4: centred column of at most 440 px with sign-out at the bottom. */
	let { children } = $props();
</script>

<div class="flex min-h-dvh flex-col bg-background">
	<div class="flex items-center justify-end gap-2 p-4">
		<LocaleSwitcher variant="inline" />
		<ThemeToggle />
	</div>
	<main
		id="main"
		tabindex="-1"
		class="mx-auto flex w-full max-w-[440px] flex-1 flex-col gap-6 px-4 pb-10 outline-none"
	>
		<div class="flex flex-col gap-3">
			<div class="flex items-center gap-2 text-foreground">
				<Logo variant="mark" height={28} />
				<span class="text-[15px] leading-6 font-medium">{t('common.appName')}</span>
			</div>
			<TickRule />
		</div>
		{@render children()}
		<div class="mt-auto flex justify-start pt-4">
			<Button variant="ghost" class={buttonSize.default} onclick={() => void session.logout()}>
				<LogOut class="size-4" strokeWidth={1.5} aria-hidden="true" />
				{t('nav.logout')}
			</Button>
		</div>
	</main>
</div>
