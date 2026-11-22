<script lang="ts">
	import Archive from '@lucide/svelte/icons/archive';
	import ClipboardPaste from '@lucide/svelte/icons/clipboard-paste';
	import ListFilter from '@lucide/svelte/icons/list-filter';
	import ListOrdered from '@lucide/svelte/icons/list-ordered';
	import Tags from '@lucide/svelte/icons/tags';
	import { onMount } from 'svelte';
	import LocaleSwitcher from '$lib/components/app/LocaleSwitcher.svelte';
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import ThemeToggle from '$lib/components/app/ThemeToggle.svelte';
	import Logo from '$lib/components/brand/Logo.svelte';
	import Panel from '$lib/components/brand/Panel.svelte';
	import TickRule from '$lib/components/brand/TickRule.svelte';
	import { t } from '$lib/i18n/index.js';
	import { publicConfig } from '$lib/state/public-config.svelte.js';
	import LoginForm from './LoginForm.svelte';
	import type { LoginReason } from './redirect.js';

	/**
	 * Sign-in page of plan 8.8.1. From 1024 px: the theme-independent brand surface (ink-950) on
	 * the left with the chamfered hero panel, the form column on the right. 768-1023 px: one
	 * column with the brand surface as a chamfered card, the form, then the features. Below
	 * 768 px: a narrow brand strip, the form at once, then the brand surface.
	 */
	type Props = { next?: string | null; reason?: LoginReason | null };
	let { next = null, reason = null }: Props = $props();

	const HERO_SRC = '/images/hero.webp';

	const features = [
		{ key: 'capture', icon: ClipboardPaste },
		{ key: 'tags', icon: Tags },
		{ key: 'filter', icon: ListFilter },
		{ key: 'ranking', icon: ListOrdered },
		{ key: 'expiry', icon: Archive }
	] as const;

	let hero = $state<'loading' | 'loaded' | 'failed'>('loading');
	let heroImage = $state<HTMLImageElement | null>(null);

	const version = $derived(publicConfig.config?.version ?? null);

	onMount(() => {
		void publicConfig.ensure();
		// A cached image may have finished before the listener ran.
		if (heroImage?.complete && heroImage.naturalWidth > 0) hero = 'loaded';
	});
</script>

<PageTitle title={t('auth.login.title')} />

<div
	class="isolate grid min-h-dvh bg-background text-foreground md:grid-cols-[minmax(0,640px)] md:content-start md:justify-center md:px-6 md:py-10 lg:grid-cols-[minmax(0,7fr)_minmax(400px,5fr)] lg:grid-rows-[auto_auto_auto_1fr] lg:content-stretch lg:p-0"
>
	<!-- Brand column from 1024 px; the 1 px line separates it from the dark form column. -->
	<div
		aria-hidden="true"
		class="hidden border-r border-border bg-ink-950 lg:col-start-1 lg:row-start-1 lg:row-end-5 lg:block"
	></div>
	<!-- Brand card between 768 and 1023 px, behind the header and the hero: kx-panel is
	     positioned, so without a negative z-index it would paint over its in-flow siblings
	     (the grid is isolated, so the card stays above the page background). -->
	<div
		aria-hidden="true"
		class="kx-panel kx-panel-lg -z-10 hidden [--kx-edge:var(--ink-700)] [--kx-fill:var(--ink-950)] md:col-start-1 md:row-start-1 md:row-end-3 md:block lg:hidden"
	></div>

	<header
		class="row-start-1 flex items-center gap-3 bg-ink-950 px-4 py-3 text-mist-100 [--primary:var(--signal-300)] md:col-start-1 md:bg-transparent md:px-8 md:pt-8 md:pb-0 lg:px-12 lg:pt-12 xl:px-16 xl:pt-16"
	>
		<Logo variant="mark" height={32} class="shrink-0 text-mist-100" />
		<div class="flex flex-col">
			<span class="text-[18px] leading-[26px] font-medium">{t('common.appName')}</span>
			<span class="text-[13px] leading-5 text-mist-300">Kinexium</span>
		</div>
	</header>

	<main
		id="main"
		tabindex="-1"
		class="row-start-2 flex flex-col bg-background px-4 py-10 outline-none md:row-start-3 md:px-0 md:pt-10 md:pb-0 lg:sticky lg:top-0 lg:col-start-2 lg:row-start-1 lg:row-end-5 lg:h-dvh lg:self-start lg:overflow-y-auto lg:px-12 lg:py-12"
		data-form-column
	>
		<!-- From 1024 px the form column is one viewport high and stays put while the taller brand
		     column scrolls: the form sits in its middle and the footer at the bottom of the view
		     (plan 8.8.1). -->
		<div class="flex flex-1 flex-col justify-center">
			<div class="mx-auto w-full max-w-[400px]">
				<LoginForm {next} {reason} />
			</div>
		</div>
		<footer
			aria-label={t('auth.login.footer')}
			class="mx-auto mt-10 flex w-full max-w-[400px] flex-wrap items-center gap-x-4 gap-y-2"
		>
			<div class="flex items-center gap-1">
				<LocaleSwitcher variant="inline" />
				<ThemeToggle />
			</div>
			<a
				href="/privacy"
				class="rounded-sm text-[13px] leading-5 text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
			>
				{t('nav.privacyNotice')}
			</a>
			{#if version}
				<span class="ml-auto font-mono text-[11px] leading-4 text-muted-foreground" data-version>
					{t('auth.login.version', { version })}
				</span>
			{/if}
		</footer>
	</main>

	<section
		aria-labelledby="landing-headline"
		class="row-start-3 bg-ink-950 px-4 pt-8 pb-8 text-mist-100 md:col-start-1 md:row-start-2 md:bg-transparent md:px-8 lg:px-12 lg:pb-0 xl:px-16"
	>
		<div class="flex max-w-[760px] flex-col gap-8">
			<Panel size="lg" class="p-px [--kx-edge:var(--ink-600)] [--kx-fill:var(--ink-900)]">
				<div
					class="aspect-video w-full overflow-hidden"
					style="clip-path: polygon(0 0, calc(100% - var(--chamfer-lg) - 1.414px) 0, 100% calc(var(--chamfer-lg) + 1.414px), 100% 100%, calc(var(--chamfer-lg) + 1.414px) 100%, 0 calc(100% - var(--chamfer-lg) - 1.414px))"
					data-hero={hero}
				>
					{#if hero !== 'failed'}
						<img
							bind:this={heroImage}
							src={HERO_SRC}
							width="1600"
							height="900"
							alt={t('landing.hero.alt')}
							decoding="async"
							fetchpriority="high"
							class="block h-auto w-full transition-opacity duration-200 ease-out {hero === 'loaded'
								? 'opacity-100'
								: 'opacity-0'}"
							onload={() => (hero = 'loaded')}
							onerror={() => (hero = 'failed')}
						/>
					{/if}
				</div>
			</Panel>
			<div class="flex max-w-[640px] flex-col gap-4">
				<h2
					id="landing-headline"
					class="text-[32px] leading-[40px] font-light tracking-[-0.01em] text-mist-100 md:text-[48px] md:leading-[54px] md:font-[250] md:tracking-[-0.015em]"
				>
					{t('landing.headline')}
				</h2>
				<p class="body-lg text-mist-300">{t('landing.intro')}</p>
			</div>
		</div>
	</section>

	<section
		aria-labelledby="landing-features"
		class="row-start-4 bg-ink-950 px-4 pb-10 text-mist-100 md:col-start-1 md:bg-transparent md:px-0 md:pt-12 md:pb-0 md:text-foreground lg:row-start-3 lg:px-12 lg:pt-8 lg:pb-12 lg:text-mist-100 xl:px-16 xl:pb-16"
	>
		<h2 id="landing-features" class="sr-only">{t('landing.features.label')}</h2>
		<div
			class="flex max-w-[760px] flex-col gap-8 max-md:[--border:var(--ink-600)] max-md:[--input:var(--ink-300)] lg:[--border:var(--ink-600)] lg:[--input:var(--ink-300)]"
		>
			<TickRule />
			<ul class="grid gap-x-8 gap-y-6 md:grid-cols-2">
				{#each features as feature (feature.key)}
					<li class="flex gap-3">
						<feature.icon
							class="mt-0.5 size-5 shrink-0 text-signal-300 md:max-lg:text-primary"
							strokeWidth={1.5}
							aria-hidden="true"
						/>
						<div class="flex flex-col gap-1">
							<h3 class="text-[15px] leading-6 font-medium">
								{t(`landing.features.${feature.key}.title`)}
							</h3>
							<p class="text-[13px] leading-5 text-mist-300 md:max-lg:text-muted-foreground">
								{t(`landing.features.${feature.key}.text`)}
							</p>
						</div>
					</li>
				{/each}
			</ul>
		</div>
	</section>
</div>
