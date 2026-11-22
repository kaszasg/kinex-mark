<script lang="ts" module>
	import type { PrivacyStatus } from '$lib/api/types.js';

	/** Acceptances, newest version first. */
	export function sortConsents(consents: PrivacyStatus['consents']): PrivacyStatus['consents'] {
		return [...consents].sort(
			(a, b) => b.version - a.version || b.acceptedAt.localeCompare(a.acceptedAt)
		);
	}
</script>

<script lang="ts">
	import FileCheck from '@lucide/svelte/icons/file-check';
	import { onMount } from 'svelte';
	import * as meApi from '$lib/api/me.js';
	import EmptyState from '$lib/components/app/EmptyState.svelte';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
	import SectionCard from '$lib/features/settings/security/SectionCard.svelte';
	import PrivacyNotice from '$lib/features/privacy/PrivacyNotice.svelte';
	import { fmt, t } from '$lib/i18n/index.js';
	import { publicConfig } from '$lib/state/public-config.svelte.js';

	/**
	 * "Privacy notice" card of plan 8.7.6: the current notice version, the accepted versions
	 * with their times (GET /me/privacy), the notice text in a scroll box and a link to /privacy.
	 */
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let privacy = $state<PrivacyStatus | null>(null);
	let error = $state<unknown>(null);
	let viewport = $state<HTMLElement | null>(null);

	// The scrolling viewport (not the ScrollArea root) is the keyboard-reachable named region:
	// the notice holds no focusable element (WCAG 2.1.1, axe scrollable-region-focusable).
	$effect(() => {
		if (!viewport) return;
		viewport.tabIndex = 0;
		viewport.setAttribute('role', 'region');
		viewport.setAttribute('aria-label', t('security.privacy.notice.summary'));
	});

	const consents = $derived(privacy ? sortConsents(privacy.consents) : []);
	const notice = $derived(publicConfig.config?.privacy ?? null);

	async function load(): Promise<void> {
		status = 'loading';
		error = null;
		try {
			const [result] = await Promise.all([meApi.getPrivacy(), publicConfig.ensure()]);
			privacy = result;
			status = 'ready';
		} catch (err) {
			error = err;
			status = 'error';
		}
	}

	onMount(() => {
		void load();
	});
</script>

<SectionCard
	id="privacy-notice"
	title={t('security.privacy.notice.title')}
	description={t('security.privacy.notice.text')}
>
	{#if status === 'loading'}
		<LoadingRows rows={3} rowClass="h-8" />
	{:else if status === 'error'}
		<ErrorAlert {error} onRetry={load} />
	{:else if privacy}
		<dl class="grid grid-cols-2 gap-x-4 gap-y-1.5 border-b pb-3">
			<div class="flex min-w-0 flex-col">
				<dt class="text-[11px] leading-4 text-muted-foreground">
					{t('security.privacy.notice.current')}
				</dt>
				<dd class="font-mono text-[13px] leading-5" data-testid="notice-version">
					{fmt.number(privacy.currentVersion)}
				</dd>
			</div>
			<div class="flex min-w-0 flex-col">
				<dt class="text-[11px] leading-4 text-muted-foreground">
					{t('security.privacy.notice.updatedAt')}
				</dt>
				<dd class="font-mono text-[13px] leading-5">
					{notice?.updatedAt ? fmt.dateTime(notice.updatedAt) : fmt.missing}
				</dd>
			</div>
		</dl>

		<div class="flex flex-col gap-2">
			<h3 class="text-[13px] leading-5 font-medium">{t('security.privacy.notice.history')}</h3>
			{#if consents.length === 0}
				<EmptyState
					icon={FileCheck}
					title={t('security.privacy.notice.none.title')}
					text={t('security.privacy.notice.none.text')}
				/>
			{:else}
				<ul class="flex flex-col">
					{#each consents as consent (consent.version + consent.acceptedAt)}
						<li
							class="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-b py-2 last:border-b-0"
						>
							<span class="flex items-center gap-2 text-[13px] leading-5">
								{t('security.privacy.notice.version', { version: consent.version })}
								{#if consent.version === privacy.currentVersion}
									<Badge variant="outline" class="h-5 rounded-sm px-1.5 text-[11px] leading-4">
										{t('security.privacy.notice.currentBadge')}
									</Badge>
								{/if}
							</span>
							<span class="font-mono text-[12px] leading-5 text-muted-foreground">
								{t('security.privacy.notice.acceptedAt', {
									time: fmt.dateTime(consent.acceptedAt)
								})}
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		</div>

		{#if notice}
			<ScrollArea bind:viewportRef={viewport} class="h-72 rounded-md border">
				<div class="p-4">
					<PrivacyNotice config={notice} compact />
				</div>
			</ScrollArea>
		{/if}

		<a href="/privacy" class="body-sm self-start text-primary underline underline-offset-3">
			{t('security.privacy.notice.open')}
		</a>
	{/if}
</SectionCard>
