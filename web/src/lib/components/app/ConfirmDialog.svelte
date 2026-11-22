<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import Button from './Button.svelte';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';
	import ErrorAlert from './ErrorAlert.svelte';
	import {
		buttonSize,
		destructiveButton,
		dialogBox,
		dialogFooter,
		typeBodySm,
		typeHeading3
	} from './styles.js';

	/**
	 * Confirmation of GRAPHICS.md 10.8: the title says what happens, the text what follows.
	 * onConfirm may be async; the dialog stays open while it runs and shows its error inline.
	 * A resolved onConfirm closes the dialog. `children` holds extra fields (step-up password).
	 */
	type Props = {
		open?: boolean;
		title: string;
		text?: string;
		confirmLabel: string;
		cancelLabel?: string;
		destructive?: boolean;
		confirmDisabled?: boolean;
		onConfirm: () => unknown;
		onCancel?: () => void;
		children?: Snippet;
	};
	let {
		open = $bindable(false),
		title,
		text,
		confirmLabel,
		cancelLabel,
		destructive = false,
		confirmDisabled = false,
		onConfirm,
		onCancel,
		children
	}: Props = $props();

	let busy = $state(false);
	let error = $state<unknown>(null);

	$effect(() => {
		if (!open) {
			error = null;
			busy = false;
		}
	});

	async function confirm(event: Event): Promise<void> {
		event.preventDefault();
		if (busy) return;
		busy = true;
		error = null;
		try {
			await onConfirm();
			open = false;
		} catch (err) {
			error = err;
		} finally {
			busy = false;
		}
	}

	function cancel(): void {
		open = false;
		onCancel?.();
	}
</script>

<AlertDialog.Root
	bind:open={
		() => open,
		(value) => {
			if (busy) return;
			if (!value) onCancel?.();
			open = value;
		}
	}
>
	<AlertDialog.Content class={dialogBox}>
		<form class="contents" onsubmit={confirm}>
			<AlertDialog.Header class="place-items-start gap-1.5 text-left">
				<AlertDialog.Title class={typeHeading3}>{title}</AlertDialog.Title>
				{#if text}
					<AlertDialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
						{text}
					</AlertDialog.Description>
				{/if}
			</AlertDialog.Header>
			{#if children}
				<div class="flex flex-col gap-4">{@render children()}</div>
			{/if}
			{#if error}
				<ErrorAlert {error} />
			{/if}
			<AlertDialog.Footer class={dialogFooter}>
				<Button
					type="button"
					variant="outline"
					class={buttonSize.default}
					disabled={busy}
					onclick={cancel}
				>
					{cancelLabel ?? t('common.actions.cancel')}
				</Button>
				<Button
					type="submit"
					class={cn(buttonSize.default, destructive && destructiveButton)}
					disabled={busy || confirmDisabled}
					aria-busy={busy}
				>
					{#if busy}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
					{confirmLabel}
				</Button>
			</AlertDialog.Footer>
		</form>
	</AlertDialog.Content>
</AlertDialog.Root>
