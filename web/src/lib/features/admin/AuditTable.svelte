<script lang="ts">
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import type { AuditEntry } from '$lib/api/types.js';
	import DataTableHead from '$lib/components/app/DataTableHead.svelte';
	import DataTableShell from '$lib/components/app/DataTableShell.svelte';
	import { tableCell } from '$lib/components/app/styles.js';
	import * as Table from '$lib/components/ui/table/index.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';
	import AuditDetails from './AuditDetails.svelte';
	import { eventLabel, targetLabel } from './audit-format.js';

	/**
	 * Audit log table of plan 8.7.9: time, translated event with its code, result (icon and word),
	 * actor, target, IP and the details popover. Unknown codes are shown raw in mono.
	 */
	type Props = { entries: AuditEntry[]; class?: string };
	let { entries, class: className }: Props = $props();
</script>

<DataTableShell caption={t('audit.table.caption')} class={className}>
	<Table.Header>
		<Table.Row class="hover:bg-transparent">
			<DataTableHead>{t('audit.table.at')}</DataTableHead>
			<DataTableHead>{t('audit.table.event')}</DataTableHead>
			<DataTableHead>{t('audit.table.result')}</DataTableHead>
			<DataTableHead>{t('audit.table.actor')}</DataTableHead>
			<DataTableHead>{t('audit.table.target')}</DataTableHead>
			<DataTableHead>{t('audit.table.ip')}</DataTableHead>
			<DataTableHead class="w-16">{t('audit.table.details')}</DataTableHead>
		</Table.Row>
	</Table.Header>
	<Table.Body>
		{#each entries as entry (entry.id)}
			{@const event = eventLabel(entry.event)}
			<Table.Row data-entry-id={entry.id}>
				<Table.Cell class={cn(tableCell, 'align-top font-mono')}>
					<time datetime={entry.at}>{fmt.dateTime(entry.at)}</time>
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'align-top')}>
					<div class="flex flex-col">
						<span class={cn(event.raw && 'font-mono')}>{event.text}</span>
						{#if !event.raw}
							<span class="font-mono text-[11px] leading-4 text-muted-foreground">
								{entry.event}
							</span>
						{/if}
					</div>
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'align-top')}>
					{#if entry.success}
						<span class="inline-flex items-center gap-1.5">
							<CircleCheck class="size-4 text-success" strokeWidth={1.5} aria-hidden="true" />
							{t('audit.result.success')}
						</span>
					{:else}
						<span class="inline-flex items-center gap-1.5">
							<CircleAlert class="size-4 text-destructive" strokeWidth={1.5} aria-hidden="true" />
							{t('audit.result.failure')}
						</span>
					{/if}
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'align-top font-mono')}>
					{entry.actor.username || fmt.missing}
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'align-top')}>
					{#if entry.target}
						{@const target = targetLabel(entry.target.type)}
						<span class="inline-flex items-baseline gap-1.5">
							<span class={cn('text-muted-foreground', target.raw && 'font-mono')}>
								{target.text}
							</span>
							{#if entry.target.name}
								<span class="font-mono">{entry.target.name}</span>
							{/if}
						</span>
					{:else}
						<span class="text-muted-foreground">{fmt.missing}</span>
					{/if}
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'align-top font-mono text-muted-foreground')}>
					{entry.ip || fmt.missing}
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'py-1.5 align-top')}>
					<AuditDetails {entry} />
				</Table.Cell>
			</Table.Row>
		{/each}
	</Table.Body>
</DataTableShell>
