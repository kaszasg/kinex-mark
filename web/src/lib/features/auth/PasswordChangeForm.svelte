<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import * as authApi from '$lib/api/auth.js';
	import { ApiError } from '$lib/api/errors.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import AccountUsername from '$lib/components/app/AccountUsername.svelte';
	import PasswordInput from '$lib/components/app/PasswordInput.svelte';
	import { buttonSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { errorMessage, t } from '$lib/i18n/index.js';
	import { publicConfig } from '$lib/state/public-config.svelte.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';
	import { signedInTarget } from './redirect.js';

	/**
	 * Password change of plan 8.7.4, shared by the password gate (`gate`) and Settings > Security
	 * (`settings`). The form shows its own success toast in both modes; the gate then continues
	 * to the next gate or "/", the settings mode clears the fields and calls `onSuccess`.
	 */
	type Props = { mode: 'gate' | 'settings'; onSuccess?: () => void };
	let { mode, onSuccess }: Props = $props();

	/** The server policy (plan 6.4) until the public configuration has loaded. */
	const DEFAULT_POLICY = { minLength: 12, minDistinct: 6 };

	let current = $state('');
	let password = $state('');
	let confirm = $state('');
	let busy = $state(false);
	let error = $state<unknown>(null);
	let currentError = $state<string | null>(null);
	let currentMessage = $state<string | null>(null);
	let newError = $state<string | null>(null);
	let confirmError = $state<string | null>(null);
	let confirmMismatch = $state(false);
	let currentInput = $state<HTMLInputElement | null>(null);
	let newInput = $state<HTMLInputElement | null>(null);
	let confirmInput = $state<HTMLInputElement | null>(null);

	const policy = $derived(publicConfig.config?.passwordPolicy ?? DEFAULT_POLICY);

	onMount(() => {
		void publicConfig.ensure();
	});

	function resetErrors(): void {
		error = null;
		currentError = null;
		currentMessage = null;
		newError = null;
		confirmError = null;
		confirmMismatch = false;
	}

	/** Client-side checks: every field is required and the repetition must match. */
	function validate(): HTMLInputElement | null {
		currentError = current ? null : 'required';
		newError = password ? null : 'required';
		confirmError = confirm ? null : 'required';
		confirmMismatch = Boolean(password && confirm && password !== confirm);
		if (currentError) return currentInput;
		if (newError) return newInput;
		if (confirmError || confirmMismatch) return confirmInput;
		return null;
	}

	async function succeed(): Promise<void> {
		toast.success(
			mode === 'gate' ? t('auth.passwordChange.saved') : t('auth.passwordChange.savedOtherSessions')
		);
		if (mode === 'settings') {
			current = '';
			password = '';
			confirm = '';
			busy = false;
			onSuccess?.();
			return;
		}
		let me = session.me;
		try {
			me = await session.refresh();
		} catch {
			// The change is saved; without a fresh user only the password gate is known to be closed.
			if (me) me = { ...me, mustChangePassword: false };
			if (me) session.setMe(me);
		}
		onSuccess?.();
		const target = me ? signedInTarget(me, null, location.origin) : '/';
		await goto(target, { replaceState: true });
	}

	async function submit(): Promise<void> {
		if (busy) return;
		resetErrors();
		const invalid = validate();
		if (invalid) {
			invalid.focus();
			return;
		}
		busy = true;
		try {
			await authApi.changePassword(current, password);
		} catch (err) {
			let focusTarget: HTMLInputElement | null = null;
			if (err instanceof ApiError && err.is('current_password_invalid')) {
				currentMessage = errorMessage(err);
				current = '';
				focusTarget = currentInput;
			} else if (err instanceof ApiError && err.is('validation_failed')) {
				currentError = err.field('currentPassword') ?? null;
				newError = err.field('newPassword') ?? null;
				focusTarget = currentError ? currentInput : newInput;
				if (!currentError && !newError) error = err;
			} else {
				error = err;
			}
			busy = false;
			await tick();
			focusTarget?.focus();
			return;
		}
		await succeed();
	}

	function onsubmit(event: SubmitEvent): void {
		event.preventDefault();
		void submit();
	}
</script>

<form
	class="flex flex-col gap-5"
	novalidate
	data-mode={mode}
	data-has-success={onSuccess ? 'true' : undefined}
	{onsubmit}
>
	<AccountUsername />
	<FormField
		id="password-current"
		label={t('auth.passwordChange.current')}
		field="currentPassword"
		errorCode={currentError}
		errorMessage={currentMessage}
	>
		{#snippet children(props)}
			<PasswordInput
				{...props}
				bind:ref={currentInput}
				bind:value={current}
				autocomplete="current-password"
				disabled={busy}
			/>
		{/snippet}
	</FormField>

	<FormField
		id="password-new"
		label={t('auth.passwordChange.new')}
		field="newPassword"
		help={t('auth.passwordPolicy', { min: policy.minLength, distinct: policy.minDistinct })}
		errorCode={newError}
	>
		{#snippet children(props)}
			<PasswordInput
				{...props}
				bind:ref={newInput}
				bind:value={password}
				autocomplete="new-password"
				disabled={busy}
			/>
		{/snippet}
	</FormField>

	<FormField
		id="password-confirm"
		label={t('auth.passwordChange.confirm')}
		field="confirmPassword"
		errorCode={confirmError}
		errorMessage={confirmMismatch ? t('auth.passwordChange.mismatch') : null}
	>
		{#snippet children(props)}
			<PasswordInput
				{...props}
				bind:ref={confirmInput}
				bind:value={confirm}
				autocomplete="new-password"
				disabled={busy}
			/>
		{/snippet}
	</FormField>

	{#if error}
		<ErrorAlert {error} />
	{/if}

	<Button
		type="submit"
		class={cn(buttonSize.default, mode === 'gate' ? 'w-full' : 'self-start')}
		disabled={busy}
	>
		{#if busy}
			<Spinner aria-label={t('common.states.loading')} />
		{/if}
		{t('auth.passwordChange.submit')}
	</Button>
</form>
