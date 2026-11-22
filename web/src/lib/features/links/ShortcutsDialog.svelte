<script lang="ts" module>
	import type { MessageKey } from '$lib/i18n/index.js';

	/** One row: alternative key combinations ("or") and the effect. */
	export type ShortcutRow = {
		id: string;
		label: MessageKey;
		combos: { keys: string[]; mod?: boolean }[];
	};

	/** The keyboard shortcuts of the app (plan 8.7.3), in the order of the plan table. */
	export const SHORTCUTS: ShortcutRow[] = [
		{ id: 'search', label: 'links.shortcuts.items.search', combos: [{ keys: ['/'] }] },
		{ id: 'add', label: 'links.shortcuts.items.add', combos: [{ keys: ['N'] }] },
		{ id: 'paste', label: 'links.shortcuts.items.paste', combos: [{ keys: ['V'], mod: true }] },
		{ id: 'next', label: 'links.shortcuts.items.next', combos: [{ keys: ['J'] }, { keys: ['↓'] }] },
		{
			id: 'previous',
			label: 'links.shortcuts.items.previous',
			combos: [{ keys: ['K'] }, { keys: ['↑'] }]
		},
		{ id: 'open', label: 'links.shortcuts.items.open', combos: [{ keys: ['Enter'] }] },
		{ id: 'pin', label: 'links.shortcuts.items.pin', combos: [{ keys: ['P'] }] },
		{ id: 'copyUrl', label: 'links.shortcuts.items.copyUrl', combos: [{ keys: ['C'] }] },
		{ id: 'copyMarkdown', label: 'links.shortcuts.items.copyMarkdown', combos: [{ keys: ['M'] }] },
		{ id: 'details', label: 'links.shortcuts.items.details', combos: [{ keys: ['I'] }] },
		{ id: 'edit', label: 'links.shortcuts.items.edit', combos: [{ keys: ['E'] }] },
		{ id: 'archive', label: 'links.shortcuts.items.archive', combos: [{ keys: ['A'] }] },
		{ id: 'delete', label: 'links.shortcuts.items.delete', combos: [{ keys: ['Delete'] }] },
		{ id: 'view', label: 'links.shortcuts.items.view', combos: [{ keys: ['V'] }] },
		{ id: 'help', label: 'links.shortcuts.items.help', combos: [{ keys: ['?'] }] },
		{ id: 'clear', label: 'links.shortcuts.items.clear', combos: [{ keys: ['Esc'] }] }
	];
</script>

<script lang="ts">
	import X from '@lucide/svelte/icons/x';
	import KeyboardShortcut from '$lib/components/app/KeyboardShortcut.svelte';
	import { buttonSize, dialogBox, typeBodySm, typeHeading3 } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';

	/** The keyboard shortcuts table; the AppShell opens it with "?" and from the display menu. */
	let { open = $bindable(false) }: { open?: boolean } = $props();
</script>

<Dialog.Root bind:open>
	<Dialog.Content class={cn(dialogBox, 'sm:max-w-[520px]')} showCloseButton={false}>
		<Dialog.Header class="flex-row items-start justify-between gap-3 text-left">
			<div class="flex flex-col gap-1.5">
				<Dialog.Title class={typeHeading3}>{t('links.shortcuts.title')}</Dialog.Title>
				<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
					{t('links.shortcuts.text')}
				</Dialog.Description>
			</div>
			<Button
				variant="ghost"
				size="icon"
				class={cn(buttonSize.iconSm, '-mt-1 -mr-2 shrink-0')}
				aria-label={t('common.actions.close')}
				onclick={() => (open = false)}
			>
				<X class="size-4" strokeWidth={1.5} aria-hidden="true" />
			</Button>
		</Dialog.Header>
		<table class="w-full text-[13px] leading-5">
			<caption class="sr-only">{t('links.shortcuts.title')}</caption>
			<tbody>
				{#each SHORTCUTS as row (row.id)}
					<tr class="border-b last:border-b-0" data-shortcut={row.id}>
						<th scope="row" class="py-1.5 pr-4 text-left font-normal">{t(row.label)}</th>
						<td class="py-1.5 text-right whitespace-nowrap">
							{#each row.combos as combo, index (index)}
								{#if index > 0}
									<span class="mx-1 text-muted-foreground">{t('links.shortcuts.or')}</span>
								{/if}
								<KeyboardShortcut keys={combo.keys} mod={combo.mod} class="inline-flex" />
							{/each}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</Dialog.Content>
</Dialog.Root>
