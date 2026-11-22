<script lang="ts" module>
	/** Class set on <html> while the browser prints, so only the code sheet reaches the paper. */
	export const PRINT_CLASS = 'kxm-print-codes';

	/** File name of the downloaded codes (plan 8.7.6). */
	export function recoveryCodesFilename(username: string): string {
		return `kinex-mark-recovery-codes-${username}.txt`;
	}
</script>

<script lang="ts">
	import Copy from '@lucide/svelte/icons/copy';
	import Download from '@lucide/svelte/icons/download';
	import Printer from '@lucide/svelte/icons/printer';
	import { toast } from 'svelte-sonner';
	import { buttonSize, checkboxBox, typeBodySm } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Checkbox } from '$lib/components/ui/checkbox/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { copyText } from '$lib/utils/clipboard.js';
	import { saveText } from '$lib/utils/download.js';
	import { cn } from '$lib/utils.js';

	/**
	 * One-time view of fresh recovery codes (plan 8.7.6): two mono columns, copy, .txt download
	 * in the user's language, print, and the "I have saved them" checkbox that gates "Done".
	 */
	type Props = {
		codes: string[];
		username: string;
		generatedAt: Date;
		saved?: boolean;
	};
	let { codes, username, generatedAt, saved = $bindable(false) }: Props = $props();

	const checkboxId = $props.id();

	function fileText(): string {
		return [
			t('security.recovery.file.title'),
			'',
			t('security.recovery.file.intro', { username }),
			t('security.recovery.file.generatedAt', { time: fmt.dateTime(generatedAt) }),
			t('security.recovery.file.warning'),
			'',
			...codes,
			''
		].join('\n');
	}

	async function copy(): Promise<void> {
		if (await copyText(codes.join('\n'))) toast.success(t('security.recovery.copied'));
	}

	function download(): void {
		saveText(fileText(), recoveryCodesFilename(username));
	}

	function print(): void {
		const root = document.documentElement;
		root.classList.add(PRINT_CLASS);
		try {
			window.print();
		} finally {
			root.classList.remove(PRINT_CLASS);
		}
	}
</script>

<div class="flex flex-col gap-4">
	<p class="body-sm text-muted-foreground">{t('security.recovery.text')}</p>
	<div class="kxm-codes-sheet flex flex-col gap-3">
		<p class="kxm-print-only heading-3">{t('security.recovery.printHeading', { username })}</p>
		<ul
			aria-label={t('security.recovery.listLabel')}
			class="grid grid-cols-2 gap-x-6 gap-y-1.5 rounded-md border bg-muted px-4 py-3 font-mono text-[13px] leading-5 tracking-wide"
		>
			{#each codes as code, index (index)}
				<li>{code}</li>
			{/each}
		</ul>
	</div>
	<div class="kxm-no-print flex flex-wrap gap-2">
		<Button type="button" variant="outline" class={buttonSize.sm} onclick={copy}>
			<Copy class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('security.recovery.copy')}
		</Button>
		<Button type="button" variant="outline" class={buttonSize.sm} onclick={download}>
			<Download class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('security.recovery.download')}
		</Button>
		<Button type="button" variant="outline" class={buttonSize.sm} onclick={print}>
			<Printer class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('security.recovery.print')}
		</Button>
	</div>
	<div class="kxm-no-print flex items-start gap-2">
		<Checkbox id={checkboxId} bind:checked={saved} class={cn(checkboxBox, 'mt-0.5')} />
		<Label for={checkboxId} class={cn(typeBodySm, 'leading-5 font-normal')}>
			{t('security.recovery.saved')}
		</Label>
	</div>
</div>

<style>
	/* While printing the codes, everything but the code sheet is hidden. */
	@media print {
		:global(html.kxm-print-codes body *) {
			visibility: hidden;
		}
		:global(html.kxm-print-codes .kxm-codes-sheet),
		:global(html.kxm-print-codes .kxm-codes-sheet *) {
			visibility: visible;
		}
	}
</style>
