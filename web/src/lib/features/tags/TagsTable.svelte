<script lang="ts">
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import type { Tag } from '$lib/api/types.js';
	import DataTableHead from '$lib/components/app/DataTableHead.svelte';
	import DataTableShell from '$lib/components/app/DataTableShell.svelte';
	import TagChip from '$lib/components/app/TagChip.svelte';
	import { buttonSize, tableCell } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Table from '$lib/components/ui/table/index.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { tagSwatchClass } from '$lib/tags/colors.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Tag table of plan 8.7.5 in name order: chip, colour name, link count as a link to the
	 * links page (archived links included), active link count, modification date, edit and delete.
	 * Narrow screens drop the colour, date and active columns so the actions stay in view.
	 */
	type Props = {
		items: Tag[];
		onEdit: (tag: Tag) => void;
		onDelete: (tag: Tag) => void;
	};
	let { items, onEdit, onDelete }: Props = $props();

	/** The links page filtered to one tag, archived links included (plan 8.7.5). */
	const linksHref = (tag: Tag): string => `/?tags=${tag.id}&archived=include`;
</script>

<DataTableShell caption={t('tags.table.caption')}>
	<Table.Header>
		<Table.Row class="hover:bg-transparent">
			<DataTableHead sort="ascending">{t('tags.table.name')}</DataTableHead>
			<DataTableHead class="hidden md:table-cell">{t('tags.table.color')}</DataTableHead>
			<DataTableHead numeric>{t('tags.table.links')}</DataTableHead>
			<DataTableHead numeric class="hidden sm:table-cell">{t('tags.table.active')}</DataTableHead>
			<DataTableHead numeric class="hidden lg:table-cell">{t('tags.table.updated')}</DataTableHead>
			<DataTableHead class="w-0">
				<span class="sr-only">{t('tags.table.actions')}</span>
			</DataTableHead>
		</Table.Row>
	</Table.Header>
	<Table.Body>
		{#each items as tag (tag.id)}
			<Table.Row data-testid="tag-row">
				<Table.Cell class={cn(tableCell, 'max-w-64')}>
					<TagChip name={tag.name} color={tag.color} />
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'hidden md:table-cell')}>
					<span class="inline-flex items-center gap-2 text-muted-foreground">
						<span class="size-2 shrink-0 {tagSwatchClass[tag.color]}" aria-hidden="true"></span>
						{t(`tagColors.${tag.color}`)}
					</span>
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'text-right font-mono')}>
					<a
						href={linksHref(tag)}
						class="text-primary underline underline-offset-3"
						aria-label={t('tags.table.showLinks', { count: tag.linkCount, name: tag.name })}
					>
						{fmt.number(tag.linkCount)}
					</a>
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'hidden text-right font-mono sm:table-cell')}>
					{fmt.number(tag.activeLinkCount)}
				</Table.Cell>
				<Table.Cell
					class={cn(tableCell, 'hidden text-right font-mono text-muted-foreground lg:table-cell')}
				>
					<time datetime={tag.updatedAt} title={fmt.dateTime(tag.updatedAt)}>
						{fmt.date(tag.updatedAt)}
					</time>
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'py-1 text-right')}>
					<div class="flex justify-end gap-1">
						<Button
							variant="ghost"
							size="icon"
							class={buttonSize.iconSm}
							aria-label={t('tags.table.edit', { name: tag.name })}
							onclick={() => onEdit(tag)}
						>
							<Pencil class="size-4" strokeWidth={1.5} aria-hidden="true" />
						</Button>
						<Button
							variant="ghost"
							size="icon"
							class={cn(buttonSize.iconSm, 'hover:text-destructive')}
							aria-label={t('tags.table.delete', { name: tag.name })}
							onclick={() => onDelete(tag)}
						>
							<Trash2 class="size-4" strokeWidth={1.5} aria-hidden="true" />
						</Button>
					</div>
				</Table.Cell>
			</Table.Row>
		{/each}
	</Table.Body>
</DataTableShell>
