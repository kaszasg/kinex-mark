<script lang="ts" module>
	/** Field order for focusing the first invalid field. */
	const FIELD_ORDER = ['url', 'title', 'description', 'tagIds'] as const;
	/** The server's text limit of title and description (plan 5.9.2). */
	export const TEXT_MAX = 500;
</script>

<script lang="ts">
	import { tick, untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import type { ApiError, FieldCode } from '$lib/api/errors.js';
	import * as linksApi from '$lib/api/links.js';
	import type { LinkItem } from '$lib/api/types.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import StateAlert from '$lib/components/app/StateAlert.svelte';
	import {
		buttonSize,
		checkboxBox,
		dialogBox,
		dialogFooter,
		inputSize,
		invalidField,
		typeBodySm,
		typeHeading3
	} from '$lib/components/app/styles.js';
	import TagPicker from '$lib/components/app/TagPicker.svelte';
	import Button from '$lib/components/app/Button.svelte';
	import { Checkbox } from '$lib/components/ui/checkbox/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import { errorText, errorTitle, t } from '$lib/i18n/index.js';
	import { codePointLength, readableUrl } from '$lib/links/presentation.js';
	import { linkStats } from '$lib/state/link-stats.svelte.js';
	import { links as linksStore, toApiError, type LinksStore } from '$lib/state/links.svelte.js';
	import { tags as tagsStore } from '$lib/state/tags.svelte.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Add or edit a link (plan 8.7.3). Adding sends the tags with the link; editing sends the
	 * changed fields with PATCH and the tags separately (tags are personal). A duplicate URL
	 * shows a warning with a way to the existing link.
	 */
	type Props = {
		open?: boolean;
		/** The link to edit; null or missing: add a new link. */
		link?: LinkItem | null;
		store?: LinksStore;
		/** Shows an existing link (duplicate warning); the dialog is already closed. */
		onShowExisting?: (id: number) => void;
	};
	let {
		open = $bindable(false),
		link = null,
		store = linksStore,
		onShowExisting
	}: Props = $props();

	let url = $state('');
	let title = $state('');
	let description = $state('');
	let tagIds = $state<number[]>([]);
	let pinned = $state(false);
	let fieldErrors = $state<Record<string, FieldCode>>({});
	let error = $state<ApiError | null>(null);
	let duplicateId = $state<number | null>(null);
	let busy = $state(false);

	const editing = $derived(link !== null);
	const descriptionLength = $derived(codePointLength(description));

	// Every opening starts from the link (or from an empty form).
	$effect(() => {
		if (!open) return;
		untrack(() => {
			// Shown as the list shows it; the server encodes it again on save.
			url = link ? readableUrl(link.url) : '';
			title = link?.title ?? '';
			description = link?.description ?? '';
			tagIds = [...(link?.tagIds ?? [])];
			pinned = false;
			fieldErrors = {};
			error = null;
			duplicateId = null;
			busy = false;
		});
	});

	async function focusFirstInvalid(): Promise<void> {
		await tick();
		const first = FIELD_ORDER.find((field) => fieldErrors[field]);
		if (first) document.getElementById(`link-dialog-${first}`)?.focus();
	}

	function sameIds(a: number[], b: number[]): boolean {
		return a.length === b.length && a.every((id) => b.includes(id));
	}

	async function create(): Promise<void> {
		const created = await linksApi.createLink({
			url: url.trim(),
			title: title.trim() || undefined,
			description: description.trim() || undefined,
			tagIds,
			pinned
		});
		store.upsert(created, { highlight: true });
		if (created.fetchStatus === 'pending') store.watchPending(created.id);
		void linkStats.refresh();
		// The new link's tags change their counts (sidebar "Frequent tags").
		if (tagIds.length > 0) void tagsStore.refresh();
	}

	async function update(current: LinkItem): Promise<void> {
		const patch: { url?: string; title?: string; description?: string } = {};
		if (url.trim() !== readableUrl(current.url)) patch.url = url.trim();
		if (title.trim() !== current.title) patch.title = title.trim();
		if (description.trim() !== current.description) patch.description = description.trim();
		if (Object.keys(patch).length > 0) {
			const detail = await linksApi.updateLink(current.id, patch);
			// The server has saved the fields: the list shows them even if the tag request fails.
			store.upsert(detail);
			if (detail.fetchStatus === 'pending') store.watchPending(detail.id);
		}
		if (!sameIds(tagIds, current.tagIds)) {
			store.upsert(await linksApi.setLinkTags(current.id, tagIds));
			void tagsStore.refresh();
		}
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (busy) return;
		fieldErrors = {};
		error = null;
		duplicateId = null;
		if (!url.trim()) {
			fieldErrors = { url: 'required' };
			void focusFirstInvalid();
			return;
		}
		busy = true;
		try {
			if (link) await update(link);
			else await create();
			toast.success(t('links.toast.saved'));
			open = false;
		} catch (err) {
			const apiError = toApiError(err);
			const existing = Number(apiError.details.linkId);
			if (apiError.is('link_duplicate')) {
				duplicateId = Number.isSafeInteger(existing) && existing > 0 ? existing : null;
				error = duplicateId === null ? apiError : null;
			} else if (apiError.is('validation_failed') && Object.keys(apiError.fields).length > 0) {
				fieldErrors = { ...apiError.fields };
				void focusFirstInvalid();
			} else {
				error = apiError;
			}
		} finally {
			busy = false;
		}
	}

	function showExisting(): void {
		const id = duplicateId;
		if (id === null) return;
		open = false;
		onShowExisting?.(id);
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class={dialogBox} showCloseButton={false}>
		<Dialog.Header class="gap-1.5 text-left">
			<Dialog.Title class={typeHeading3}>
				{editing ? t('links.dialog.editTitle') : t('links.dialog.addTitle')}
			</Dialog.Title>
			<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
				{editing ? t('links.dialog.editText') : t('links.dialog.addText')}
			</Dialog.Description>
		</Dialog.Header>
		<form class="flex flex-col gap-5" novalidate onsubmit={submit}>
			<FormField
				id="link-dialog-url"
				label={t('links.dialog.url')}
				field="url"
				errorCode={fieldErrors.url}
			>
				{#snippet children(props)}
					<Input
						{...props}
						bind:value={url}
						type="url"
						inputmode="url"
						autocomplete="off"
						spellcheck="false"
						class={cn(inputSize, 'font-mono')}
					/>
				{/snippet}
			</FormField>
			<FormField
				id="link-dialog-title"
				label={t('links.dialog.title')}
				field="title"
				help={t('links.dialog.titleHelp')}
				errorCode={fieldErrors.title}
			>
				{#snippet children(props)}
					<Input {...props} bind:value={title} maxlength={TEXT_MAX} class={inputSize} />
				{/snippet}
			</FormField>
			<FormField
				id="link-dialog-description"
				label={t('links.dialog.description')}
				field="description"
				help={t('links.dialog.descriptionHelp')}
				errorCode={fieldErrors.description}
			>
				{#snippet children(props)}
					<Textarea
						{...props}
						bind:value={description}
						rows={3}
						maxlength={TEXT_MAX}
						class={cn(
							'min-h-20 resize-y rounded-md bg-transparent! px-3 py-2 text-[13px] leading-5',
							invalidField
						)}
					/>
					<span
						class={cn(
							'self-end font-mono text-[11px] leading-4 text-muted-foreground',
							descriptionLength > TEXT_MAX && 'text-destructive'
						)}
						data-testid="description-counter"
					>
						{t('links.dialog.counter', { count: descriptionLength })}
					</span>
				{/snippet}
			</FormField>
			<FormField
				id="link-dialog-tagIds"
				label={t('links.dialog.tags')}
				field="tagIds"
				errorCode={fieldErrors.tagIds}
			>
				{#snippet children(props)}
					<TagPicker id={props.id} bind:value={tagIds} disabled={busy} />
				{/snippet}
			</FormField>
			{#if !editing}
				<div class="flex items-center gap-2">
					<Checkbox id="link-dialog-pinned" bind:checked={pinned} class={checkboxBox} />
					<Label for="link-dialog-pinned" class="text-[13px] leading-5 font-normal">
						{t('links.dialog.pin')}
					</Label>
				</div>
			{/if}
			{#if duplicateId !== null}
				<StateAlert
					kind="warning"
					title={errorTitle({ code: 'link_duplicate', details: {} })}
					text={errorText({ code: 'link_duplicate', details: {} })}
					actionLabel={t('links.dialog.showExisting')}
					onAction={showExisting}
				/>
			{/if}
			{#if error}
				<ErrorAlert {error} />
			{/if}
			<Dialog.Footer class={dialogFooter}>
				<Button
					type="button"
					variant="outline"
					class={buttonSize.default}
					disabled={busy}
					onclick={() => (open = false)}
				>
					{t('common.actions.cancel')}
				</Button>
				<Button type="submit" class={buttonSize.default} disabled={busy} aria-busy={busy}>
					{#if busy}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
					{t('links.dialog.save')}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
