<script lang="ts">
	// primary-exception: the done button closes its own dialog screen.
	import CopyButton from '$lib/components/app/CopyButton.svelte';
	import StateAlert from '$lib/components/app/StateAlert.svelte';
	import {
		buttonSize,
		dialogBox,
		dialogFooter,
		typeBodySm,
		typeHeading3
	} from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';

	/**
	 * The one-time temporary password after creating a user or resetting a password (plan 8.7.7):
	 * the password in large mono with a copy button and the warning that it is shown only now.
	 */
	type Props = { open?: boolean; username: string; password: string };
	let { open = $bindable(false), username, password }: Props = $props();
</script>

<Dialog.Root bind:open>
	<Dialog.Content class={dialogBox} showCloseButton={false}>
		<Dialog.Header class="gap-1.5 text-left">
			<Dialog.Title class={typeHeading3}>{t('admin.tempPassword.title')}</Dialog.Title>
			<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
				{t('admin.tempPassword.text', { username })}
			</Dialog.Description>
		</Dialog.Header>
		<div class="flex items-center gap-2 rounded-md border bg-muted px-4 py-3">
			<output
				class="min-w-0 flex-1 font-mono text-[20px] leading-7 tracking-[0.04em] break-all select-all"
				data-testid="temporary-password"
			>
				{password}
			</output>
			<CopyButton
				text={password}
				label={t('admin.tempPassword.copy')}
				toastText={t('admin.tempPassword.copied')}
				size="icon"
			/>
		</div>
		<StateAlert
			kind="warning"
			title={t('admin.tempPassword.warningTitle')}
			text={t('admin.tempPassword.warningText')}
		/>
		<Dialog.Footer class={dialogFooter}>
			<Button class={buttonSize.default} onclick={() => (open = false)}>
				{t('admin.tempPassword.done')}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
