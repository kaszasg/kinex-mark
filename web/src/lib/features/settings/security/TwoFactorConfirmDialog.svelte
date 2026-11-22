<script lang="ts" module>
	export type TwoFactorConfirmMode = 'disable' | 'regenerate';
</script>

<script lang="ts">
	import { toast } from 'svelte-sonner';
	import * as meApi from '$lib/api/me.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import AccountUsername from '$lib/components/app/AccountUsername.svelte';
	import PasswordInput from '$lib/components/app/PasswordInput.svelte';
	import {
		buttonSize,
		destructiveButton,
		dialogBox,
		inputSize,
		typeBodySm,
		typeHeading3
	} from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { fieldMessage, t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';
	import { focusFirstInvalid, formErrors, NO_ERRORS, type FormErrors } from './form-errors.js';
	import RecoveryCodesView from './RecoveryCodesView.svelte';

	/**
	 * Step-up dialog of plan 8.7.6 for turning two-factor sign-in off (password plus an
	 * authentication or recovery code) and for new recovery codes (password plus an
	 * authentication code only). New codes are shown in the RecoveryCodesView of the same dialog.
	 */
	type Props = { open?: boolean; mode: TwoFactorConfirmMode; onDone?: () => void };
	let { open = $bindable(false), mode, onDone }: Props = $props();

	const uid = $props.id();

	let password = $state('');
	let code = $state('');
	let busy = $state(false);
	let errors = $state<FormErrors>(NO_ERRORS);
	let codes = $state<string[] | null>(null);
	let generatedAt = $state(new Date());
	let saved = $state(false);
	let form = $state<HTMLFormElement | null>(null);

	const keys = $derived(
		mode === 'disable'
			? ({
					title: 'security.confirm.disable.title',
					text: 'security.confirm.disable.text',
					code: 'security.confirm.disable.code',
					codeHelp: 'security.confirm.disable.codeHelp',
					submit: 'security.confirm.disable.submit'
				} as const)
			: ({
					title: 'security.confirm.regenerate.title',
					text: 'security.confirm.regenerate.text',
					code: 'security.confirm.regenerate.code',
					codeHelp: 'security.confirm.regenerate.codeHelp',
					submit: 'security.confirm.regenerate.submit'
				} as const)
	);

	let wasOpen = false;
	$effect(() => {
		if (open && !wasOpen) reset();
		wasOpen = open;
	});

	function reset(): void {
		password = '';
		code = '';
		busy = false;
		errors = NO_ERRORS;
		codes = null;
		saved = false;
	}

	function requestClose(): void {
		if (busy) return;
		if (codes) {
			if (saved) finishCodes();
			return;
		}
		password = '';
		code = '';
		open = false;
	}

	function finishCodes(): void {
		open = false;
		codes = null;
		onDone?.();
		void session.refresh().catch(() => undefined);
	}

	function validate(): FormErrors | null {
		const fields: Record<string, string> = {};
		if (!password) fields.password = fieldMessage('password', 'required');
		const value = code.trim();
		if (!value) fields.code = fieldMessage('code', 'required');
		else if (mode === 'regenerate' && !/^\d{6}$/.test(value))
			fields.code = fieldMessage('code', 'invalid_format');
		return Object.keys(fields).length > 0 ? { fields, alert: null } : null;
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (busy) return;
		if (codes) {
			if (saved) finishCodes();
			return;
		}
		const invalid = validate();
		if (invalid) {
			errors = invalid;
			void focusFirstInvalid(form);
			return;
		}
		busy = true;
		errors = NO_ERRORS;
		try {
			if (mode === 'disable') {
				await meApi.disableTwoFactor(password, code.trim());
				open = false;
				toast.success(t('security.twoFactor.disabled'));
				onDone?.();
				void session.refresh().catch(() => undefined);
			} else {
				const result = await meApi.regenerateRecoveryCodes(password, code.trim());
				codes = result.recoveryCodes;
				generatedAt = new Date();
				saved = false;
				toast.success(t('security.twoFactor.regenerated'));
			}
			password = '';
			code = '';
		} catch (err) {
			code = '';
			errors = formErrors(err, { current_password_invalid: 'password', invalid_totp: 'code' }, [
				'password',
				'code'
			]);
			void focusFirstInvalid(form);
		} finally {
			busy = false;
		}
	}
</script>

<Dialog.Root
	bind:open={
		() => open,
		(value) => {
			if (value) open = true;
			else requestClose();
		}
	}
>
	<Dialog.Content class={dialogBox} showCloseButton={false}>
		<form bind:this={form} class="flex min-w-0 flex-col gap-4" onsubmit={submit} novalidate>
			<AccountUsername />
			<Dialog.Header class="gap-1.5 text-left">
				{#if codes}
					<Dialog.Title class={typeHeading3}>{t('security.enroll.codes.title')}</Dialog.Title>
				{:else}
					<Dialog.Title class={typeHeading3}>{t(keys.title)}</Dialog.Title>
					<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
						{t(keys.text)}
					</Dialog.Description>
				{/if}
			</Dialog.Header>

			{#if codes}
				<RecoveryCodesView {codes} username={session.me?.username ?? ''} {generatedAt} bind:saved />
			{:else}
				<FormField
					id="{uid}-password"
					label={t('security.confirm.password')}
					field="password"
					errorMessage={errors.fields.password}
				>
					{#snippet children(control)}
						<PasswordInput
							{...control}
							bind:value={password}
							autocomplete="current-password"
							disabled={busy}
						/>
					{/snippet}
				</FormField>
				<FormField
					id="{uid}-code"
					label={t(keys.code)}
					help={t(keys.codeHelp)}
					field="code"
					errorMessage={errors.fields.code}
				>
					{#snippet children(control)}
						<Input
							{...control}
							bind:value={code}
							class={cn(inputSize, 'font-mono')}
							autocomplete="one-time-code"
							inputmode={mode === 'regenerate' ? 'numeric' : 'text'}
							maxlength={mode === 'regenerate' ? 6 : 16}
							spellcheck={false}
							disabled={busy}
						/>
					{/snippet}
				</FormField>
			{/if}

			{#if errors.alert}
				<ErrorAlert error={errors.alert} />
			{/if}

			<div class="flex flex-row flex-wrap justify-end gap-2">
				{#if codes}
					<Button type="submit" class={buttonSize.default} disabled={!saved}>
						{t('common.actions.done')}
					</Button>
				{:else}
					<Button
						type="button"
						variant="outline"
						class={buttonSize.default}
						disabled={busy}
						onclick={requestClose}
					>
						{t('common.actions.cancel')}
					</Button>
					<Button
						type="submit"
						class={cn(buttonSize.default, mode === 'disable' && destructiveButton)}
						disabled={busy}
						aria-busy={busy}
					>
						{#if busy}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
						{t(keys.submit)}
					</Button>
				{/if}
			</div>
		</form>
	</Dialog.Content>
</Dialog.Root>
