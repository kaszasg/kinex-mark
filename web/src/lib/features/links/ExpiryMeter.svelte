<script lang="ts">
	import Hourglass from '@lucide/svelte/icons/hourglass';
	import type { LinkItem } from '$lib/api/types.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { EXPIRY_SEGMENTS, expiryInfo, showsExpiry } from '$lib/links/expiry.js';
	import { cn } from '$lib/utils.js';
	import Hint from './Hint.svelte';

	/**
	 * Time left until automatic archiving (plan 8.7.3): five hairline segments, and in the last
	 * seven days an hourglass with the day count. An instrument, not an alarm.
	 */
	type Props = {
		link: Pick<LinkItem, 'archiveDueAt' | 'inactiveSince' | 'pinned' | 'archivedAt'>;
		effectiveArchiveAfterDays: number;
		now?: Date;
		class?: string;
	};
	let { link, effectiveArchiveAfterDays, now, class: className }: Props = $props();

	const info = $derived(
		showsExpiry(link, effectiveArchiveAfterDays)
			? expiryInfo({
					archiveDueAt: link.archiveDueAt,
					inactiveSince: link.inactiveSince,
					now: now ?? new Date()
				})
			: null
	);
	const label = $derived(
		link.archiveDueAt ? t('links.expiry.label', { date: fmt.date(link.archiveDueAt) }) : ''
	);
</script>

{#if info}
	<Hint text={label}>
		{#snippet children(props)}
			<span
				{...props}
				role="img"
				aria-label={label}
				data-segments={info.segments}
				class={cn('inline-flex h-5 items-center gap-0.5', className)}
			>
				{#if info.soon}
					<Hourglass class="size-3 text-warning" strokeWidth={1.5} aria-hidden="true" />
					<span class="ml-0.5 font-mono text-[11px] leading-4 text-muted-foreground">
						{info.daysLeft > 0
							? t('links.expiry.daysLeft', { count: info.daysLeft })
							: t('links.expiry.today')}
					</span>
				{:else}
					{#each { length: EXPIRY_SEGMENTS }, index (index)}
						<span
							class={cn(
								'h-[3px] w-1.5 rounded-none',
								index < info.segments ? 'bg-muted-foreground' : 'bg-border'
							)}
						></span>
					{/each}
				{/if}
			</span>
		{/snippet}
	</Hint>
{/if}
