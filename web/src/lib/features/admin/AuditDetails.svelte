<script lang="ts">
	import Info from '@lucide/svelte/icons/info';
	import type { AuditEntry } from '$lib/api/types.js';
	import { buttonSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';
	import { detailRows } from './audit-format.js';

	/** The details of one audit entry in a popover (mono definition list), or a dash when empty. */
	type Props = { entry: Pick<AuditEntry, 'event' | 'details'> };
	let { entry }: Props = $props();

	const rows = $derived(detailRows(entry));
</script>

{#if rows.length === 0}
	<span class="text-muted-foreground">{fmt.missing}</span>
{:else}
	<Popover.Root>
		<Popover.Trigger>
			{#snippet child({ props })}
				<Button
					{...props}
					variant="ghost"
					size="icon"
					class={buttonSize.iconSm}
					aria-label={t('audit.table.showDetails')}
				>
					<Info class="size-4" strokeWidth={1.5} aria-hidden="true" />
				</Button>
			{/snippet}
		</Popover.Trigger>
		<Popover.Content align="end" class="w-auto max-w-80 p-3">
			<dl
				class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 font-mono text-[12px] leading-4"
			>
				{#each rows as row (row.key)}
					<dt class="text-muted-foreground">{row.label.text}</dt>
					<dd class={cn('break-words', row.value.raw && 'text-muted-foreground')}>
						{row.value.text}
					</dd>
				{/each}
			</dl>
		</Popover.Content>
	</Popover.Root>
{/if}
