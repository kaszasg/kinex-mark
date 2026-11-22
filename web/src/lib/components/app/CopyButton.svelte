<script lang="ts" module>
	/** How long the check icon stays after a successful copy. */
	export const COPIED_MS = 1500;
</script>

<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import Copy from '@lucide/svelte/icons/copy';
	import { onDestroy } from 'svelte';
	import { toast } from 'svelte-sonner';
	import Button from './Button.svelte';
	import { t } from '$lib/i18n/index.js';
	import { copyText } from '$lib/utils/clipboard.js';
	import { cn } from '$lib/utils.js';
	import { buttonSize } from './styles.js';

	/**
	 * Copy icon button of plan 8.4: check icon for 1.5 s and a toast after success. `label` is
	 * the accessible name (default "Copy"); `onCopied` runs after a successful copy.
	 */
	type Props = {
		text: string;
		label?: string;
		toastText?: string;
		size?: 'icon' | 'iconSm';
		onCopied?: () => void;
		class?: string;
	};
	let { text, label, toastText, size = 'iconSm', onCopied, class: className }: Props = $props();

	let copied = $state(false);
	let timer: ReturnType<typeof setTimeout> | null = null;

	async function copy(): Promise<void> {
		if (!(await copyText(text))) return;
		copied = true;
		toast.success(toastText ?? t('common.actions.copied'));
		onCopied?.();
		if (timer) clearTimeout(timer);
		timer = setTimeout(() => (copied = false), COPIED_MS);
	}

	onDestroy(() => {
		if (timer) clearTimeout(timer);
	});
</script>

<Button
	variant="ghost"
	size="icon"
	class={cn(buttonSize[size], className)}
	aria-label={label ?? t('common.actions.copy')}
	onclick={copy}
>
	{#if copied}
		<Check class="size-4 text-success" strokeWidth={1.5} aria-hidden="true" />
	{:else}
		<Copy class="size-4" strokeWidth={1.5} aria-hidden="true" />
	{/if}
</Button>
