<script lang="ts">
	import type { Snippet } from 'svelte';
	import { MediaQuery } from 'svelte/reactivity';
	import { afterNavigate } from '$app/navigation';
	import * as Sheet from '$lib/components/ui/sheet/index.js';
	import CaptureController from '$lib/features/links/CaptureController.svelte';
	import ShortcutsDialog from '$lib/features/links/ShortcutsDialog.svelte';
	import { t } from '$lib/i18n/index.js';
	import { shouldHandleShortcut } from '$lib/utils/keyboard.js';
	import AppSidebar from './AppSidebar.svelte';
	import SidebarFrame from './SidebarFrame.svelte';
	import { setShellContext } from './shell-context.js';
	import Topbar from './Topbar.svelte';

	/**
	 * Application frame of plan 8.7.1: fixed 240 px sidebar from 1024 px, an off-canvas sheet
	 * below it, the top bar and <main id="main">. The sheet and the trigger are our own so their
	 * labels are translated (the generated mobile sidebar has fixed English texts).
	 */
	let { children }: { children: Snippet } = $props();

	const desktop = new MediaQuery('min-width: 1024px', true);

	let sheetOpen = $state(false);
	let shortcutsOpen = $state(false);

	setShellContext({
		toggleSidebar: () => (sheetOpen = !sheetOpen),
		closeSidebar: () => (sheetOpen = false),
		openShortcuts: () => (shortcutsOpen = true)
	});

	// A desktop-sized window never keeps the sheet open.
	$effect(() => {
		if (desktop.current) sheetOpen = false;
	});

	let main = $state<HTMLElement | null>(null);

	// A route change moves the focus to <main> (plan 8.9); the router itself leaves it on <body>.
	// A query change on the same page (search, filters, ?link=) keeps the focus where it is.
	afterNavigate(({ from, to, type }) => {
		sheetOpen = false;
		if (type === 'enter' || !from || !to || from.url.pathname === to.url.pathname) return;
		const active = document.activeElement;
		if (active && active !== document.body) return;
		main?.focus({ preventScroll: true });
	});

	function onKeydown(event: KeyboardEvent): void {
		if (event.key === '?' && shouldHandleShortcut(event)) {
			event.preventDefault();
			shortcutsOpen = true;
		}
	}
</script>

<svelte:window onkeydown={onKeydown} />

<SidebarFrame style="--sidebar-width: var(--layout-sidebar);" class="min-h-dvh">
	<a
		href="#main"
		class="sr-only z-50 rounded-md bg-background px-3 py-2 text-[13px] focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:outline-2 focus:outline-ring"
	>
		{t('common.skipToContent')}
	</a>
	<div class="grid min-h-dvh w-full grid-cols-1 lg:grid-cols-[var(--layout-sidebar)_minmax(0,1fr)]">
		<aside class="sticky top-0 hidden h-dvh lg:block">
			<AppSidebar />
		</aside>
		<div class="flex min-w-0 flex-col">
			<Topbar />
			<main
				bind:this={main}
				id="main"
				tabindex="-1"
				class="flex min-w-0 flex-1 flex-col gap-4 p-4 outline-none md:p-6"
			>
				{@render children()}
			</main>
		</div>
	</div>
	{#if !desktop.current}
		<Sheet.Root bind:open={sheetOpen}>
			<Sheet.Content
				side="left"
				showCloseButton={false}
				class="w-[var(--layout-sidebar)] max-w-[85vw] gap-0 border-sidebar-border bg-sidebar p-0 text-sidebar-foreground"
			>
				<Sheet.Header class="sr-only">
					<Sheet.Title>{t('nav.sidebarTitle')}</Sheet.Title>
				</Sheet.Header>
				<AppSidebar onNavigate={() => (sheetOpen = false)} />
			</Sheet.Content>
		</Sheet.Root>
	{/if}
	<CaptureController />
	<ShortcutsDialog bind:open={shortcutsOpen} />
</SidebarFrame>
