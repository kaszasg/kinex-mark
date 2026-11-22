<script lang="ts" module>
	/** The server's import limit (plan 5.9.7); larger files are refused before any request. */
	export const IMPORT_MAX_BYTES = 2 * 1024 * 1024;
</script>

<script lang="ts">
	import ArrowDownUp from '@lucide/svelte/icons/arrow-down-up';
	import FileCode from '@lucide/svelte/icons/file-code';
	import FileJson from '@lucide/svelte/icons/file-json';
	import Upload from '@lucide/svelte/icons/upload';
	import { toast } from 'svelte-sonner';
	import * as linksApi from '$lib/api/links.js';
	import type { ImportResult } from '$lib/api/types.js';
	import {
		buttonSize,
		dialogBox,
		dialogFooter,
		typeBodySm,
		typeHeading3
	} from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { errorMessage, fmt, t } from '$lib/i18n/index.js';
	import { linkStats } from '$lib/state/link-stats.svelte.js';
	import { links as linksStore, type LinksStore } from '$lib/state/links.svelte.js';
	import { tags } from '$lib/state/tags.svelte.js';
	import { saveBlob } from '$lib/utils/download.js';
	import { cn } from '$lib/utils.js';
	import { toastError } from '$lib/utils/toast-error.js';

	/**
	 * Export (JSON or browser bookmark HTML) and import of the viewer's own links (plan 8.7.3).
	 * The import result shows in a small dialog; the list, the counters and the tags reload.
	 */
	type Props = { store?: LinksStore };
	let { store = linksStore }: Props = $props();

	let input = $state<HTMLInputElement | null>(null);
	let trigger = $state<HTMLButtonElement | null>(null);
	let busy = $state(false);
	let result = $state<ImportResult | null>(null);
	let resultOpen = $state(false);

	async function exportAs(format: 'json' | 'html'): Promise<void> {
		if (busy) return;
		busy = true;
		try {
			const file = await linksApi.exportLinks(format);
			saveBlob(file.blob, file.filename);
			toast.success(t('links.toast.exported'));
		} catch (err) {
			toastError(err);
		} finally {
			busy = false;
		}
	}

	async function onchange(event: Event): Promise<void> {
		const target = event.currentTarget as HTMLInputElement;
		const file = target.files?.[0];
		target.value = '';
		if (!file) return;
		if (file.size > IMPORT_MAX_BYTES) {
			toast.error(errorMessage({ code: 'payload_too_large', details: {} }));
			return;
		}
		busy = true;
		try {
			result = await linksApi.importLinks(file);
			resultOpen = true;
			void store.reload();
			void linkStats.refresh();
			void tags.load();
		} catch (err) {
			toastError(err);
		} finally {
			busy = false;
		}
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger disabled={busy}>
		{#snippet child({ props })}
			<Button
				{...props}
				bind:ref={trigger}
				variant="outline"
				class={buttonSize.default}
				aria-busy={busy}
			>
				{#if busy}
					<Spinner class="size-4" aria-label={t('common.states.loading')} />
				{:else}
					<ArrowDownUp class="size-4" strokeWidth={1.5} aria-hidden="true" />
				{/if}
				<span class="max-sm:sr-only">{t('links.importExport.label')}</span>
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="min-w-60">
		<DropdownMenu.Item onSelect={() => void exportAs('json')}>
			<FileJson class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('links.importExport.exportJson')}
		</DropdownMenu.Item>
		<DropdownMenu.Item onSelect={() => void exportAs('html')}>
			<FileCode class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('links.importExport.exportHtml')}
		</DropdownMenu.Item>
		<DropdownMenu.Separator />
		<DropdownMenu.Item onSelect={() => input?.click()}>
			<Upload class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('links.importExport.import')}
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<input
	bind:this={input}
	type="file"
	accept=".json,.html,.htm,application/json,text/html"
	class="sr-only"
	tabindex="-1"
	aria-hidden="true"
	data-testid="import-input"
	onchange={(event) => void onchange(event)}
/>

<Dialog.Root bind:open={resultOpen}>
	<!-- The dialog opens from the file picker, not from a trigger: closing it returns the focus
	     to the menu button that started the import (plan 8.9). -->
	<Dialog.Content
		class={dialogBox}
		showCloseButton={false}
		onCloseAutoFocus={(event: Event) => {
			event.preventDefault();
			trigger?.focus();
		}}
	>
		<Dialog.Header class="gap-1.5 text-left">
			<Dialog.Title class={typeHeading3}>{t('links.importExport.resultTitle')}</Dialog.Title>
			<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
				{t('links.importExport.resultText')}
			</Dialog.Description>
		</Dialog.Header>
		{#if result}
			<dl class="grid grid-cols-[minmax(0,1fr)_auto] gap-x-6 gap-y-1.5 text-[13px] leading-5">
				<dt class="text-muted-foreground">{t('links.importExport.total')}</dt>
				<dd class="data text-right">{fmt.number(result.total)}</dd>
				<dt class="text-muted-foreground">{t('links.importExport.created')}</dt>
				<dd class="data text-right">{fmt.number(result.created)}</dd>
				<dt class="text-muted-foreground">{t('links.importExport.duplicates')}</dt>
				<dd class="data text-right">{fmt.number(result.duplicates)}</dd>
				<dt class="text-muted-foreground">{t('links.importExport.invalid')}</dt>
				<dd class="data text-right">{fmt.number(result.invalid)}</dd>
			</dl>
		{/if}
		<Dialog.Footer class={dialogFooter}>
			<Button variant="outline" class={buttonSize.default} onclick={() => (resultOpen = false)}>
				{t('common.actions.close')}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
