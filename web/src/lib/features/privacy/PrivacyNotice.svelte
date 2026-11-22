<script lang="ts">
	import type { PublicConfig } from '$lib/api/types.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';

	/**
	 * The privacy notice text (GDPR Article 13, plan 8.7.10 and 6.14), localised, with the
	 * controller data and retention periods of the public configuration. `compact` is the
	 * denser variant inside the consent gate and Settings > Privacy (h3 headings, 13 px text).
	 */
	type Props = { config: PublicConfig['privacy']; compact?: boolean };
	let { config, compact = false }: Props = $props();

	/** The sections of plan 8.7.10 in order; controller, retention and rights carry extra data. */
	const SECTIONS = [
		'controller',
		'data',
		'purposes',
		'legalBasis',
		'recipients',
		'transfers',
		'automatedDecisions',
		'retention',
		'storage',
		'fetching',
		'adminAccess',
		'rights'
	] as const;
	const RIGHTS = [
		'access',
		'portability',
		'rectification',
		'erasure',
		'restriction',
		'objection'
	] as const;

	const heading = $derived(compact ? 'h3' : 'h2');
	const value = (text: string): string => text.trim() || fmt.missing;
	const controllerRows = $derived([
		{ key: 'name', label: t('privacy.controller.name'), value: config.controllerName },
		{ key: 'address', label: t('privacy.controller.address'), value: config.controllerAddress },
		{ key: 'email', label: t('privacy.controller.email'), value: config.controllerEmail },
		{ key: 'dpo', label: t('privacy.controller.dpo'), value: config.dpoContact }
	]);
	const retention = $derived(
		t('privacy.sections.retention.body', {
			hours: t('common.units.hours', { count: config.sessionAbsoluteHours }),
			minutes: t('common.units.minutes', { count: config.sessionIdleMinutes }),
			days: t('common.units.days', { count: config.auditRetentionDays })
		})
	);
	const authority = $derived(
		config.supervisoryAuthority.trim() || t('privacy.rights.defaultAuthority')
	);
</script>

<div
	class={cn('flex flex-col', compact ? 'gap-5' : 'gap-8')}
	data-compact={compact ? 'true' : undefined}
	data-version={config.noticeVersion}
>
	{#each SECTIONS as section (section)}
		<section class="flex flex-col gap-2" data-section={section}>
			<svelte:element
				this={heading}
				class={compact
					? 'text-[15px] leading-6 font-medium text-foreground'
					: 'heading-3 text-foreground'}
			>
				{t(`privacy.sections.${section}.title`)}
			</svelte:element>
			<p class={cn('text-foreground', compact ? 'text-[13px] leading-5' : 'body')}>
				{section === 'retention' ? retention : t(`privacy.sections.${section}.body`)}
			</p>
			{#if section === 'controller'}
				<dl
					class={cn(
						'grid grid-cols-1 gap-x-6 gap-y-1 sm:grid-cols-[max-content_minmax(0,1fr)]',
						compact ? 'text-[13px] leading-5' : 'text-[15px] leading-6'
					)}
				>
					{#each controllerRows as row (row.key)}
						<dt class="text-muted-foreground">{row.label}</dt>
						<dd class="mb-2 break-words text-foreground sm:mb-0">
							{#if row.key === 'email' && row.value.trim()}
								<a
									href="mailto:{row.value.trim()}"
									class="underline underline-offset-4 hover:text-primary"
								>
									{row.value.trim()}
								</a>
							{:else}
								{value(row.value)}
							{/if}
						</dd>
					{/each}
				</dl>
			{:else if section === 'rights'}
				<ul
					class={cn(
						'flex list-disc flex-col gap-1.5 pl-5 text-foreground marker:text-muted-foreground',
						compact ? 'text-[13px] leading-5' : 'body'
					)}
				>
					{#each RIGHTS as right (right)}
						<li>{t(`privacy.rights.${right}`)}</li>
					{/each}
					<li>{t('privacy.rights.complaint', { authority })}</li>
				</ul>
			{/if}
		</section>
	{/each}
</div>
