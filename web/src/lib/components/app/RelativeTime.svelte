<script lang="ts">
	import { fmt, t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';

	/** <time> with a relative text and the full date and time in the title (plan 8.4). */
	type Props = { value: string | Date | null | undefined; now?: Date; class?: string };
	let { value, now, class: className }: Props = $props();

	const date = $derived(value ? new Date(value) : null);
	const valid = $derived(date !== null && !Number.isNaN(date.getTime()));
</script>

{#if valid && date}
	<time
		datetime={date.toISOString()}
		title={fmt.dateTime(date)}
		class={cn('whitespace-nowrap', className)}
	>
		{fmt.relative(date, now)}
	</time>
{:else}
	<span class={className}>{t('common.missing')}</span>
{/if}
