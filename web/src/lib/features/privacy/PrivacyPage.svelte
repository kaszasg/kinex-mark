<script lang="ts">
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import { onMount } from 'svelte';
	import { ApiError } from '$lib/api/errors.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import { fmt, t } from '$lib/i18n/index.js';
	import { publicConfig } from '$lib/state/public-config.svelte.js';
	import { session } from '$lib/state/session.svelte.js';
	import PrivacyNotice from './PrivacyNotice.svelte';

	/**
	 * Public privacy notice page of plan 8.7.10, reachable without signing in. A signed-in user
	 * also gets a link back to the application.
	 */

	let status = $state<'loading' | 'ready' | 'error'>(publicConfig.config ? 'ready' : 'loading');

	const config = $derived(publicConfig.config);
	const signedIn = $derived(session.me !== null);

	onMount(() => {
		// Only decides whether the "back to the app" link shows; a failure keeps it hidden.
		session.ensure().catch(() => null);
		if (status !== 'ready') void load();
	});

	async function load(): Promise<void> {
		status = 'loading';
		const loaded = await publicConfig.ensure();
		status = loaded ? 'ready' : 'error';
	}
</script>

<PageTitle title={t('privacy.title')} />

<article class="flex flex-col gap-8">
	<header class="flex flex-col gap-3">
		{#if signedIn}
			<a
				href="/"
				class="inline-flex items-center gap-1.5 self-start rounded-sm text-[13px] leading-5 text-muted-foreground transition-colors hover:text-foreground"
			>
				<ArrowLeft class="size-4" strokeWidth={1.5} aria-hidden="true" />
				{t('privacy.backToApp')}
			</a>
		{/if}
		<h1 class="heading-1 text-foreground">{t('privacy.title')}</h1>
		{#if config && status === 'ready'}
			<p class="flex flex-wrap gap-x-4 font-mono text-[13px] leading-5 text-muted-foreground">
				<span data-notice-version>
					{t('privacy.version', { version: config.privacy.noticeVersion })}
				</span>
				{#if config.privacy.updatedAt}
					<span data-notice-updated>
						{t('privacy.updatedAt', { date: fmt.date(config.privacy.updatedAt) })}
					</span>
				{/if}
			</p>
		{/if}
	</header>

	{#if status === 'loading'}
		<LoadingRows rows={8} rowClass="h-6" />
	{:else if status === 'error' || !config}
		<ErrorAlert error={new ApiError(0, 'network_error')} onRetry={() => void load()} />
	{:else}
		<PrivacyNotice config={config.privacy} />
	{/if}
</article>
