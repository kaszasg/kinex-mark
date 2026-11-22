<script lang="ts">
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import { onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { ApiError } from '$lib/api/errors.js';
	import * as meApi from '$lib/api/me.js';
	import * as publicApi from '$lib/api/public.js';
	import type { PublicConfig } from '$lib/api/types.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import StateAlert from '$lib/components/app/StateAlert.svelte';
	import { buttonSize, checkboxBox } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Checkbox } from '$lib/components/ui/checkbox/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { errorText, errorTitle, fmt, t } from '$lib/i18n/index.js';
	import { signedInTarget } from '$lib/features/auth/redirect.js';
	import { publicConfig } from '$lib/state/public-config.svelte.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';
	import PrivacyNotice from './PrivacyNotice.svelte';

	/**
	 * Privacy notice gate of plan 8.7.4. The user confirms having read the current notice; this
	 * is an acknowledgement, not consent as a legal basis (plan 6.14).
	 */

	const mismatchError = new ApiError(409, 'consent_version_mismatch');

	/**
	 * The cached configuration is outdated when it names another notice version than the signed-in
	 * user's (the notice was bumped after the tab loaded it): acknowledging it would fail with 409.
	 */
	function isStale(cached: PublicConfig | null): boolean {
		if (!cached) return true;
		const current = session.me?.privacyNoticeVersion;
		return current !== undefined && cached.privacy.noticeVersion !== current;
	}

	let status = $state<'loading' | 'ready' | 'error'>(
		isStale(publicConfig.config) ? 'loading' : 'ready'
	);
	let checked = $state(false);
	let busy = $state(false);
	let error = $state<unknown>(null);
	let newVersion = $state(false);
	let viewport = $state<HTMLElement | null>(null);
	let checkboxRef = $state<HTMLButtonElement | null>(null);

	const config = $derived(publicConfig.config);

	onMount(() => {
		if (status !== 'ready') void load();
	});

	// The scrolling notice must be reachable by keyboard even when it holds no link.
	$effect(() => {
		if (!viewport) return;
		viewport.tabIndex = 0;
		viewport.setAttribute('role', 'region');
		viewport.setAttribute('aria-label', t('privacy.consent.title'));
	});

	async function load(): Promise<void> {
		status = 'loading';
		const loaded = publicConfig.config ? await publicConfig.refresh() : await publicConfig.ensure();
		status = loaded ? 'ready' : 'error';
	}

	/** Fetches the notice again after the server reported a newer version. */
	async function reload(): Promise<void> {
		try {
			publicConfig.config = await publicApi.getPublicConfig();
			status = 'ready';
		} catch (err) {
			error = err;
		}
	}

	async function continueAfterConsent(): Promise<void> {
		let me = session.me;
		try {
			me = await session.refresh();
		} catch {
			// The acknowledgement is saved; the consent gate is known to be closed.
			if (me) me = { ...me, privacyConsentRequired: false };
			if (me) session.setMe(me);
		}
		await goto(me ? signedInTarget(me, null, location.origin) : '/', { replaceState: true });
	}

	async function submit(): Promise<void> {
		if (busy || !checked || !config) return;
		busy = true;
		error = null;
		newVersion = false;
		try {
			await meApi.acceptPrivacy(config.privacy.noticeVersion);
		} catch (err) {
			if (err instanceof ApiError && err.is('consent_version_mismatch')) {
				checked = false;
				newVersion = true;
				await reload();
			} else {
				error = err;
			}
			busy = false;
			await tick();
			checkboxRef?.focus();
			return;
		}
		await continueAfterConsent();
	}

	function onsubmit(event: SubmitEvent): void {
		event.preventDefault();
		void submit();
	}
</script>

<form class="flex flex-col gap-5" novalidate {onsubmit}>
	<div class="flex flex-col gap-1">
		<h1 class="heading-2 text-foreground">{t('privacy.consent.title')}</h1>
		<p class="text-[13px] leading-5 text-muted-foreground">{t('privacy.consent.text')}</p>
		{#if config && status === 'ready'}
			<p class="font-mono text-[11px] leading-4 text-muted-foreground" data-notice-version>
				<span>{t('privacy.version', { version: config.privacy.noticeVersion })}</span>
				{#if config.privacy.updatedAt}
					<span class="ml-3">{fmt.date(config.privacy.updatedAt)}</span>
				{/if}
			</p>
		{/if}
	</div>

	{#if newVersion}
		<StateAlert kind="info" title={errorTitle(mismatchError)} text={errorText(mismatchError)} />
	{/if}

	{#if status === 'loading'}
		<LoadingRows rows={6} rowClass="h-5" />
	{:else if status === 'error' || !config}
		<ErrorAlert error={new ApiError(0, 'network_error')} onRetry={() => void load()} />
	{:else}
		<div class="flex flex-col gap-2">
			<ScrollArea bind:viewportRef={viewport} class="h-[50vh] rounded-lg border bg-card">
				<div class="p-4">
					<PrivacyNotice config={config.privacy} compact />
				</div>
			</ScrollArea>
			<a
				href="/privacy"
				target="_blank"
				rel="noopener"
				class="inline-flex items-center gap-1.5 self-start rounded-sm text-[13px] leading-5 text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
			>
				<ExternalLink class="size-3.5" strokeWidth={1.5} aria-hidden="true" />
				{t('privacy.consent.openFull')}
				<span class="sr-only">({t('privacy.consent.newTab')})</span>
			</a>
		</div>

		<div class="flex items-start gap-2">
			<Checkbox
				id="consent-ack"
				bind:ref={checkboxRef}
				bind:checked
				disabled={busy}
				class={cn(checkboxBox, 'mt-0.5')}
			/>
			<Label for="consent-ack" class="text-[13px] leading-5 font-normal text-foreground">
				{t('privacy.consent.checkbox')}
			</Label>
		</div>

		{#if error}
			<ErrorAlert {error} />
		{/if}

		<Button type="submit" class={cn(buttonSize.default, 'w-full')} disabled={busy || !checked}>
			{#if busy}
				<Spinner aria-label={t('common.states.loading')} />
			{/if}
			{t('privacy.consent.submit')}
		</Button>
	{/if}
</form>
