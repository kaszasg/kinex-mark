<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ApiError } from '$lib/api/errors.js';
	import ConfirmDialog from '$lib/components/app/ConfirmDialog.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import AccountUsername from '$lib/components/app/AccountUsername.svelte';
	import PasswordInput from '$lib/components/app/PasswordInput.svelte';
	import { errorMessage, t } from '$lib/i18n/index.js';

	/**
	 * Confirmation with the admin's own password (step-up, plan 5.12 and 8.7.7). The confirm button
	 * stays disabled until the password is typed. A wrong password (current_password_invalid) or a
	 * field error on currentPassword is shown under the field and keeps the dialog open; every
	 * other error is shown by ConfirmDialog as an alert.
	 */
	type Props = {
		open?: boolean;
		title: string;
		text: string;
		confirmLabel: string;
		destructive?: boolean;
		/** Runs the action with the typed password; a resolved promise closes the dialog. */
		onConfirm: (currentPassword: string) => Promise<unknown>;
		/** Extra content between the text and the password field (for example a warning). */
		children?: Snippet;
	};
	let {
		open = $bindable(false),
		title,
		text,
		confirmLabel,
		destructive = false,
		onConfirm,
		children
	}: Props = $props();

	const fieldId = $props.id();
	let password = $state('');
	let fieldCode = $state<string | null>(null);
	let fieldText = $state<string | null>(null);
	let input = $state<HTMLInputElement | null>(null);
	/** Set when a field error must keep the dialog open although onConfirm resolved. */
	let hold = false;

	$effect(() => {
		if (!open) {
			password = '';
			fieldCode = null;
			fieldText = null;
		}
	});

	async function confirm(): Promise<void> {
		fieldCode = null;
		fieldText = null;
		try {
			await onConfirm(password);
		} catch (err) {
			if (!(err instanceof ApiError)) throw err;
			if (err.is('current_password_invalid')) fieldText = errorMessage(err);
			else if (err.field('currentPassword')) fieldCode = err.field('currentPassword') ?? null;
			else throw err;
			hold = true;
			queueMicrotask(() => input?.select());
		}
	}
</script>

<ConfirmDialog
	bind:open={
		() => open,
		(value: boolean) => {
			if (!value && hold) {
				hold = false;
				return;
			}
			open = value;
		}
	}
	{title}
	{text}
	{confirmLabel}
	{destructive}
	confirmDisabled={password.length === 0}
	onConfirm={confirm}
	onCancel={() => (hold = false)}
>
	<AccountUsername />
	{@render children?.()}
	<FormField
		id="{fieldId}-password"
		label={t('common.stepUp.label')}
		help={t('common.stepUp.help')}
		field="currentPassword"
		errorCode={fieldCode}
		errorMessage={fieldText}
	>
		{#snippet children(control)}
			<PasswordInput
				{...control}
				bind:value={password}
				bind:ref={input}
				name="currentPassword"
				autocomplete="current-password"
				required
			/>
		{/snippet}
	</FormField>
</ConfirmDialog>
