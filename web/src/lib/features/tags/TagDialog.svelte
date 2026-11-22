<script lang="ts" module>
	/** Longest tag name the server accepts, in Unicode code points (plan 5.10). */
	export const TAG_NAME_MAX = 40;
</script>

<script lang="ts">
	import { toast } from 'svelte-sonner';
	import { ApiError } from '$lib/api/errors.js';
	import type { Tag, TagColor } from '$lib/api/types.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import TagChip from '$lib/components/app/TagChip.svelte';
	import {
		buttonSize,
		dialogBox,
		inputSize,
		typeBodySm,
		typeHeading3
	} from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import * as ToggleGroup from '$lib/components/ui/toggle-group/index.js';
	import { errorMessage, t } from '$lib/i18n/index.js';
	import { tags, type TagsStore } from '$lib/state/tags.svelte.js';
	import { isTagColor, TAG_COLORS, tagSwatchClass } from '$lib/tags/colors.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Create or edit dialog of plan 8.7.5: name (1-40 characters with a counter), colour as a
	 * single-choice toggle group in the palette order, a live TagChip preview, "Save tag".
	 * `tag_duplicate` shows under the name, `tag_limit_reached` and other errors as an alert.
	 */
	type Props = {
		open?: boolean;
		/** The tag to edit; null creates a new one. */
		tag?: Tag | null;
		store?: TagsStore;
		onSaved?: (tag: Tag) => void;
	};
	let { open = $bindable(false), tag = null, store = tags, onSaved }: Props = $props();

	let name = $state('');
	let color = $state<TagColor>('neutral');
	let busy = $state(false);
	let nameCode = $state<string | null>(null);
	let nameMessage = $state<string | null>(null);
	let error = $state<unknown>(null);
	let wasOpen = false;

	// Each opening starts from the edited tag (or an empty neutral one) without old errors.
	$effect.pre(() => {
		if (open && !wasOpen) {
			name = tag?.name ?? '';
			color = tag?.color ?? 'neutral';
			nameCode = null;
			nameMessage = null;
			error = null;
			busy = false;
		}
		wasOpen = open;
	});

	const length = $derived([...name.trim()].length);
	const previewName = $derived(name.trim() || t(`tagColors.${color}`));

	function setOpen(next: boolean): void {
		if (busy && !next) return;
		open = next;
	}

	function changeName(): void {
		nameCode = null;
		nameMessage = null;
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (busy) return;
		const trimmed = name.trim();
		nameMessage = null;
		error = null;
		if (trimmed === '') {
			nameCode = 'required';
			return;
		}
		if ([...trimmed].length > TAG_NAME_MAX) {
			nameCode = 'too_long';
			return;
		}
		nameCode = null;
		const patch: { name?: string; color?: TagColor } = {};
		if (!tag || trimmed !== tag.name) patch.name = trimmed;
		if (!tag || color !== tag.color) patch.color = color;
		if (tag && Object.keys(patch).length === 0) {
			open = false;
			return;
		}
		busy = true;
		try {
			const saved = tag ? await store.update(tag.id, patch) : await store.create(trimmed, color);
			busy = false;
			open = false;
			toast.success(t('tags.dialog.saved'));
			onSaved?.(saved);
		} catch (err) {
			busy = false;
			const apiError = err instanceof ApiError ? err : new ApiError(0, 'network_error');
			if (apiError.is('tag_duplicate')) {
				nameMessage = errorMessage(apiError);
			} else if (apiError.is('validation_failed') && apiError.field('name')) {
				nameCode = apiError.field('name') ?? null;
			} else {
				error = apiError;
			}
		}
	}
</script>

<Dialog.Root bind:open={() => open, setOpen}>
	<Dialog.Content class={dialogBox} showCloseButton={false}>
		<form class="contents" novalidate onsubmit={submit}>
			<Dialog.Header class="gap-1.5 text-left">
				<Dialog.Title class={typeHeading3}>
					{tag ? t('tags.dialog.editTitle') : t('tags.dialog.createTitle')}
				</Dialog.Title>
				<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
					{t('tags.dialog.description')}
				</Dialog.Description>
			</Dialog.Header>

			<FormField
				id="tag-dialog-name"
				label={t('tags.dialog.name')}
				field="name"
				help={t('tags.dialog.nameHelp', { max: TAG_NAME_MAX })}
				errorCode={nameCode}
				errorMessage={nameMessage}
			>
				{#snippet children(control)}
					<div class="relative">
						<Input
							{...control}
							bind:value={name}
							oninput={changeName}
							autocomplete="off"
							spellcheck={false}
							disabled={busy}
							class={cn(inputSize, 'pr-14')}
						/>
						<span
							class={cn(
								'pointer-events-none absolute inset-y-0 right-3 flex items-center font-mono text-[11px] leading-4',
								length > TAG_NAME_MAX ? 'text-destructive' : 'text-muted-foreground'
							)}
							aria-hidden="true"
							data-testid="tag-name-counter"
						>
							{length}/{TAG_NAME_MAX}
						</span>
					</div>
				{/snippet}
			</FormField>

			<div class="flex flex-col gap-1.5">
				<span id="tag-dialog-color" class="text-[13px] leading-5 font-medium">
					{t('tags.dialog.color')}
				</span>
				<ToggleGroup.Root
					type="single"
					variant="outline"
					size="sm"
					spacing={1}
					aria-labelledby="tag-dialog-color"
					class="w-full flex-wrap"
					disabled={busy}
					bind:value={
						() => color,
						(next: string) => {
							if (isTagColor(next)) color = next;
						}
					}
				>
					{#each TAG_COLORS as option (option)}
						<ToggleGroup.Item
							value={option}
							class="h-7 gap-1.5 rounded-md px-2 text-[13px] leading-5 font-normal data-[state=on]:border-ring data-[state=on]:bg-accent"
						>
							<span class="size-2 shrink-0 {tagSwatchClass[option]}" aria-hidden="true"></span>
							{t(`tagColors.${option}`)}
						</ToggleGroup.Item>
					{/each}
				</ToggleGroup.Root>
			</div>

			<div class="flex min-w-0 items-center gap-3">
				<span class="text-[13px] leading-5 text-muted-foreground">{t('tags.dialog.preview')}</span>
				<TagChip name={previewName} {color} class="min-w-0" />
			</div>

			{#if error}
				<ErrorAlert {error} />
			{/if}

			<div class="flex flex-row justify-end gap-2">
				<Button
					type="button"
					variant="outline"
					class={buttonSize.default}
					disabled={busy}
					onclick={() => setOpen(false)}
				>
					{t('common.actions.cancel')}
				</Button>
				<Button type="submit" class={buttonSize.default} disabled={busy} aria-busy={busy}>
					{#if busy}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
					{t('tags.dialog.save')}
				</Button>
			</div>
		</form>
	</Dialog.Content>
</Dialog.Root>
