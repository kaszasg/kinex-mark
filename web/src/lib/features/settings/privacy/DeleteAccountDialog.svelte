<script lang="ts">
	import UserRoundX from '@lucide/svelte/icons/user-round-x';
	import { goto } from '$app/navigation';
	import * as meApi from '$lib/api/me.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import AccountUsername from '$lib/components/app/AccountUsername.svelte';
	import PasswordInput from '$lib/components/app/PasswordInput.svelte';
	import StateAlert from '$lib/components/app/StateAlert.svelte';
	import {
		buttonSize,
		destructiveButton,
		dialogBox,
		inputSize,
		typeBodySm,
		typeHeading3
	} from '$lib/components/app/styles.js';
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import {
		focusFirstInvalid,
		formErrors,
		NO_ERRORS,
		type FormErrors
	} from '$lib/features/settings/security/form-errors.js';
	import SectionCard from '$lib/features/settings/security/SectionCard.svelte';
	import { fieldMessage, t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';

	/**
	 * "Delete account" of plan 8.7.6 (GDPR erasure): the card with its trigger and the
	 * confirmation dialog. The dialog asks for the password, the second factor when two-factor
	 * sign-in is on, and the username; the final button stays disabled until the username
	 * matches. It warns with the number of own shared links (GET /me/privacy) that disappear
	 * for their recipients. Success clears the session and goes to /login?reason=deleted.
	 */
	let open = $state(false);
	let password = $state('');
	let code = $state('');
	let confirm = $state('');
	let busy = $state(false);
	let errors = $state<FormErrors>(NO_ERRORS);
	let form = $state<HTMLFormElement | null>(null);
	let passwordInput = $state<HTMLInputElement | null>(null);

	let sharedStatus = $state<'loading' | 'ready' | 'error'>('loading');
	let sharedCount = $state(0);
	let sharedError = $state<unknown>(null);

	const uid = $props.id();
	const username = $derived(session.me?.username ?? '');
	const totpEnabled = $derived(session.me?.totpEnabled ?? false);
	const matches = $derived(username !== '' && confirm === username);

	function start(): void {
		password = '';
		code = '';
		confirm = '';
		busy = false;
		errors = NO_ERRORS;
		open = true;
		void loadShared();
	}

	async function loadShared(): Promise<void> {
		sharedStatus = 'loading';
		sharedError = null;
		try {
			sharedCount = (await meApi.getPrivacy()).ownSharedLinkCount;
			sharedStatus = 'ready';
		} catch (err) {
			sharedError = err;
			sharedStatus = 'error';
		}
	}

	function close(): void {
		if (busy) return;
		password = '';
		code = '';
		open = false;
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (busy || !matches) return;
		const fields: Record<string, string> = {};
		if (!password) fields.password = fieldMessage('password', 'required');
		if (totpEnabled && !code.trim()) fields.code = fieldMessage('code', 'required');
		if (Object.keys(fields).length > 0) {
			errors = { fields, alert: null };
			void focusFirstInvalid(form);
			return;
		}
		busy = true;
		errors = NO_ERRORS;
		try {
			await meApi.deleteAccount(password, confirm, totpEnabled ? code.trim() : undefined);
			busy = false;
			open = false;
			session.clear('deleted');
			await goto('/login?reason=deleted');
		} catch (err) {
			busy = false;
			code = '';
			errors = formErrors(
				err,
				{
					current_password_invalid: 'password',
					invalid_totp: 'code',
					confirmation_mismatch: 'confirm'
				},
				['password', 'code', 'confirm']
			);
			void focusFirstInvalid(form);
		}
	}
</script>

<SectionCard
	id="privacy-delete"
	title={t('security.deleteAccount.title')}
	description={t('security.deleteAccount.text')}
>
	<div>
		<Button variant="outline" class={cn(buttonSize.default, 'text-destructive')} onclick={start}>
			<UserRoundX class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('security.deleteAccount.open')}
		</Button>
	</div>
</SectionCard>

<AlertDialog.Root
	bind:open={
		() => open,
		(value) => {
			if (value) open = true;
			else close();
		}
	}
>
	<!-- Plan 8.9: an opening dialog focuses its first field (the password), not the dialog box. -->
	<AlertDialog.Content
		class={dialogBox}
		onOpenAutoFocus={(event: Event) => {
			event.preventDefault();
			passwordInput?.focus();
		}}
	>
		<form bind:this={form} class="flex min-w-0 flex-col gap-4" onsubmit={submit} novalidate>
			<AccountUsername />
			<AlertDialog.Header class="place-items-start gap-1.5 text-left">
				<AlertDialog.Title class={typeHeading3}>
					{t('security.deleteAccount.dialogTitle')}
				</AlertDialog.Title>
				<AlertDialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
					{t('security.deleteAccount.dialogText')}
				</AlertDialog.Description>
			</AlertDialog.Header>

			{#if sharedStatus === 'loading'}
				<LoadingRows rows={1} rowClass="h-14" />
			{:else if sharedStatus === 'error'}
				<ErrorAlert error={sharedError} onRetry={loadShared} />
			{:else if sharedCount > 0}
				<StateAlert
					kind="warning"
					title={t('security.deleteAccount.shared.title', { count: sharedCount })}
					text={t('security.deleteAccount.shared.text')}
				/>
			{/if}

			<FormField
				id="{uid}-password"
				label={t('security.deleteAccount.password')}
				field="password"
				errorMessage={errors.fields.password}
			>
				{#snippet children(control)}
					<PasswordInput
						{...control}
						bind:ref={passwordInput}
						bind:value={password}
						autocomplete="current-password"
						disabled={busy}
					/>
				{/snippet}
			</FormField>

			{#if totpEnabled}
				<FormField
					id="{uid}-code"
					label={t('security.deleteAccount.code')}
					field="code"
					errorMessage={errors.fields.code}
				>
					{#snippet children(control)}
						<Input
							{...control}
							bind:value={code}
							class={cn(inputSize, 'font-mono')}
							autocomplete="one-time-code"
							maxlength={16}
							spellcheck={false}
							disabled={busy}
						/>
					{/snippet}
				</FormField>
			{/if}

			<FormField
				id="{uid}-confirm"
				label={t('security.deleteAccount.confirm', { username })}
				field="confirm"
				errorMessage={errors.fields.confirm}
			>
				{#snippet children(control)}
					<Input
						{...control}
						bind:value={confirm}
						class={cn(inputSize, 'font-mono')}
						autocomplete="off"
						autocapitalize="off"
						spellcheck={false}
						disabled={busy}
					/>
				{/snippet}
			</FormField>

			{#if errors.alert}
				<ErrorAlert error={errors.alert} />
			{/if}

			<div class="flex flex-row flex-wrap justify-end gap-2">
				<Button
					type="button"
					variant="outline"
					class={buttonSize.default}
					disabled={busy}
					onclick={close}
				>
					{t('common.actions.cancel')}
				</Button>
				<Button
					type="submit"
					class={cn(buttonSize.default, destructiveButton)}
					disabled={busy || !matches}
					aria-busy={busy}
				>
					{#if busy}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
					{t('security.deleteAccount.submit')}
				</Button>
			</div>
		</form>
	</AlertDialog.Content>
</AlertDialog.Root>
