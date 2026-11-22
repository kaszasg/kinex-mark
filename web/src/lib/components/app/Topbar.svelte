<script lang="ts">
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Menu from '@lucide/svelte/icons/menu';
	import Search from '@lucide/svelte/icons/search';
	import X from '@lucide/svelte/icons/x';
	import { onDestroy, tick } from 'svelte';
	import { MediaQuery } from 'svelte/reactivity';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Button from './Button.svelte';
	import * as Kbd from '$lib/components/ui/kbd/index.js';
	import { t } from '$lib/i18n/index.js';
	import { MAX_QUERY_LENGTH } from '$lib/links/query.js';
	import { debounce } from '$lib/utils/debounce.js';
	import { shouldHandleShortcut } from '$lib/utils/keyboard.js';
	import { cn } from '$lib/utils.js';
	import { breadcrumbsFor } from './breadcrumbs.js';
	import LocaleSwitcher from './LocaleSwitcher.svelte';
	import { getShellContext } from './shell-context.js';
	import { buttonSize } from './styles.js';
	import ThemeToggle from './ThemeToggle.svelte';

	/** Live search on the links page waits this long after the last keystroke. */
	const SEARCH_DEBOUNCE_MS = 250;

	/**
	 * Top bar of plan 8.7.1: sidebar trigger below 1024 px, breadcrumb, search, language and
	 * theme. On the links page the search filters live through ?q=; elsewhere Enter opens /?q=.
	 */
	const shell = getShellContext();

	const crumbs = $derived(breadcrumbsFor(page.url.pathname));
	const onLinksPage = $derived(page.url.pathname === '/');

	let query = $state(page.url.searchParams.get('q') ?? '');
	let input = $state<HTMLInputElement | null>(null);
	/** Below 768 px the search is an icon button that opens a full-width field (plan 8.9). */
	let searchOpen = $state(false);
	let opener = $state<HTMLButtonElement | null>(null);

	/** From 768 px the field always shows; below it the icon button opens it (plan 8.9). */
	const wide = new MediaQuery('min-width: 768px', true);

	async function openSearch(): Promise<void> {
		if (!wide.current) searchOpen = true;
		await tick();
		input?.focus();
	}

	async function closeSearch(): Promise<void> {
		searchOpen = false;
		await tick();
		opener?.focus();
	}

	/** The opened mobile field folds back when the focus leaves it. */
	function onSearchFocusout(event: FocusEvent): void {
		const form = event.currentTarget as HTMLFormElement;
		if (!form.contains(event.relatedTarget as Node | null)) searchOpen = false;
	}

	/**
	 * Escape clears the search and ?q= (plan 8.7.3) however the field got the focus, in every
	 * browser (the native type=search clearing is not universal and preventDefault skips it).
	 * On an empty field it folds the mobile search back.
	 */
	function onSearchKeydown(event: KeyboardEvent): void {
		if (event.key !== 'Escape') return;
		event.preventDefault();
		if (query !== '') {
			query = '';
			liveSearch.cancel();
			if (onLinksPage && page.url.searchParams.has('q')) {
				void goto(searchUrl(''), { replace: true, reset: false });
			}
			return;
		}
		if (searchOpen) void closeSearch();
	}

	// Follow the URL when it changes from elsewhere (back button, a tag link, clearing a filter).
	$effect(() => {
		const fromUrl = page.url.searchParams.get('q') ?? '';
		if (document.activeElement !== input) query = fromUrl;
	});

	function searchUrl(value: string): string {
		const params = new URLSearchParams(onLinksPage ? page.url.search : '');
		if (value.trim()) params.set('q', value.trim());
		else params.delete('q');
		const qs = params.toString();
		return qs ? `/?${qs}` : '/';
	}

	const liveSearch = debounce((value: string) => {
		void goto(searchUrl(value), { replace: true, reset: false });
	}, SEARCH_DEBOUNCE_MS);

	onDestroy(() => liveSearch.cancel());

	function onInput(): void {
		if (onLinksPage) liveSearch(query);
	}

	function onSubmit(event: SubmitEvent): void {
		event.preventDefault();
		liveSearch.cancel();
		void goto(searchUrl(query), { reset: !onLinksPage });
	}

	function onKeydown(event: KeyboardEvent): void {
		if (event.key === '/' && shouldHandleShortcut(event)) {
			event.preventDefault();
			void openSearch();
		}
	}
</script>

<svelte:window onkeydown={onKeydown} />

<header
	class="sticky top-0 z-20 flex h-[var(--layout-topbar)] shrink-0 items-center gap-3 border-b bg-background px-4 md:px-6"
>
	<Button
		variant="ghost"
		size="icon"
		class="{buttonSize.icon} -ml-2 lg:hidden"
		aria-label={t('nav.toggleSidebar')}
		onclick={() => shell.toggleSidebar()}
	>
		<Menu class="size-4" strokeWidth={1.5} aria-hidden="true" />
	</Button>
	{#if crumbs.length > 0}
		<nav aria-label={t('nav.breadcrumb.label')} class="hidden min-w-0 sm:block">
			<ol class="flex min-w-0 items-center gap-1.5 text-[13px] leading-5 text-muted-foreground">
				{#each crumbs as crumb, index (crumb.key)}
					{@const last = index === crumbs.length - 1}
					<li class="flex min-w-0 items-center gap-1.5">
						{#if last}
							<span aria-current="page" class="truncate font-medium text-foreground">
								{t(crumb.key)}
							</span>
						{:else if crumb.href}
							<a href={crumb.href} class="truncate hover:text-foreground">{t(crumb.key)}</a>
						{:else}
							<span class="truncate">{t(crumb.key)}</span>
						{/if}
						{#if !last}
							<ChevronRight class="size-3.5 shrink-0" strokeWidth={1.5} aria-hidden="true" />
						{/if}
					</li>
				{/each}
			</ol>
		</nav>
	{/if}
	<div class="ml-auto flex items-center gap-1">
		<Button
			bind:ref={opener}
			variant="ghost"
			size="icon"
			class={cn(buttonSize.icon, 'md:hidden')}
			aria-label={t('nav.search.open')}
			aria-expanded={searchOpen}
			onclick={() => void openSearch()}
		>
			<Search class="size-4" strokeWidth={1.5} aria-hidden="true" />
		</Button>
		<form
			role="search"
			class={cn(
				'relative mr-1 w-full max-w-[240px]',
				searchOpen
					? 'max-md:absolute max-md:inset-0 max-md:z-10 max-md:mr-0 max-md:flex max-md:max-w-none max-md:items-center max-md:gap-2 max-md:bg-background max-md:px-4'
					: 'max-md:hidden'
			)}
			onsubmit={onSubmit}
			onfocusout={onSearchFocusout}
			data-search-open={searchOpen}
		>
			<div class="relative min-w-0 flex-1">
				<Search
					class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
					strokeWidth={1.5}
					aria-hidden="true"
				/>
				<input
					bind:this={input}
					bind:value={query}
					oninput={onInput}
					onkeydown={onSearchKeydown}
					type="search"
					enterkeyhint="search"
					autocomplete="off"
					maxlength={MAX_QUERY_LENGTH}
					aria-label={t('nav.search.label')}
					placeholder={t('nav.search.placeholder')}
					class="h-9 w-full min-w-0 rounded-md border border-input bg-transparent pr-9 pl-8 text-[13px] text-foreground placeholder:text-muted-foreground focus-visible:border-ring [&::-webkit-search-cancel-button]:hidden"
				/>
				<Kbd.Root
					class="pointer-events-none absolute top-1/2 right-2 hidden -translate-y-1/2 font-mono lg:inline-flex"
					aria-hidden="true"
				>
					/
				</Kbd.Root>
			</div>
			{#if searchOpen}
				<Button
					type="button"
					variant="ghost"
					size="icon"
					class={cn(buttonSize.icon, 'md:hidden')}
					aria-label={t('nav.search.close')}
					onclick={() => void closeSearch()}
				>
					<X class="size-4" strokeWidth={1.5} aria-hidden="true" />
				</Button>
			{/if}
		</form>
		<LocaleSwitcher />
		<ThemeToggle />
	</div>
</header>
