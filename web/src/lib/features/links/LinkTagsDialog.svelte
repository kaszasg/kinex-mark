<script lang="ts">
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import type { ApiError } from '$lib/api/errors.js';
	import * as linksApi from '$lib/api/links.js';
	import type { LinkItem } from '$lib/api/types.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FieldError from '$lib/components/app/FieldError.svelte';
	import {
		buttonSize,
		dialogBox,
		dialogFooter,
		typeBodySm,
		typeHeading3
	} from '$lib/components/app/styles.js';
	import TagPicker from '$lib/components/app/TagPicker.svelte';
	import Button from '$lib/components/app/Button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { t } from '$lib/i18n/index.js';
	import { fullLabel } from '$lib/links/presentation.js';
	import { links as linksStore, toApiError, type LinksStore } from '$lib/state/links.svelte.js';
	import { tags as tagsStore } from '$lib/state/tags.svelte.js';
	import { cn } from '$lib/utils.js';

	/**
	 * The viewer's own tags on a link (PUT /api/links/{id}/tags). Works on shared links too,
	 * because tags are personal.
	 */
	type Props = { open?: boolean; link: LinkItem | null; store?: LinksStore };
	let { open = $bindable(false), link, store = linksStore }: Props = $props();

	let tagIds = $state<number[]>([]);
	let busy = $state(false);
	let error = $state<ApiError | null>(null);
	let fieldCode = $state<string | null>(null);

	$effect(() => {
		if (!open) return;
		untrack(() => {
			tagIds = [...(link?.tagIds ?? [])];
			busy = false;
			error = null;
			fieldCode = null;
		});
	});

	async function save(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (!link || busy) return;
		busy = true;
		error = null;
		fieldCode = null;
		try {
			store.upsert(await linksApi.setLinkTags(link.id, tagIds));
			// The counts behind the sidebar's "Frequent tags" change with the assignment.
			void tagsStore.refresh();
			toast.success(t('links.toast.tagsSaved'));
			open = false;
		} catch (err) {
			const apiError = toApiError(err);
			if (apiError.is('validation_failed') && apiError.field('tagIds')) {
				fieldCode = apiError.field('tagIds') ?? null;
			} else {
				error = apiError;
			}
		} finally {
			busy = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class={dialogBox} showCloseButton={false}>
		<Dialog.Header class="min-w-0 gap-1.5 text-left">
			<Dialog.Title class={typeHeading3}>{t('links.tagsDialog.title')}</Dialog.Title>
			<Dialog.Description class={cn(typeBodySm, 'min-w-0 text-muted-foreground')}>
				{#if link}
					<span
						class="block min-w-0 truncate font-medium text-foreground"
						data-testid="tags-dialog-label"
					>
						{fullLabel(link)}
					</span>
				{/if}
				{t('links.tagsDialog.text')}
			</Dialog.Description>
		</Dialog.Header>
		<form class="flex flex-col gap-4" onsubmit={save}>
			<TagPicker id="link-tags-picker" bind:value={tagIds} disabled={busy} />
			<FieldError id="link-tags-error" field="tagIds" code={fieldCode} />
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
					{t('links.tagsDialog.save')}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
