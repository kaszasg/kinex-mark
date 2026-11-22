<script lang="ts">
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import { onDestroy, tick } from 'svelte';
	import { ApiError } from '$lib/api/errors.js';
	import type { Me } from '$lib/api/types.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import { buttonSize, inputSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import * as InputOTP from '$lib/components/ui/input-otp/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';
	import { Lockout, retryAfterSeconds } from './lockout.svelte.js';

	/**
	 * Second sign-in step (plan 8.7.2): a 6-digit authenticator code that submits itself on the
	 * sixth digit, or a recovery code. The challenge comes from the first step.
	 */
	type Props = {
		challenge: string;
		/** Signed in; `recovery` tells whether a recovery code was used. */
		onSuccess: (me: Me, recovery: boolean) => void;
		/** The challenge expired or ran out of attempts: back to the first step. */
		onChallengeInvalid: (err: ApiError) => void;
		/** "Back to sign-in": the challenge is dropped. */
		onBack: () => void;
	};
	let { challenge, onSuccess, onChallengeInvalid, onBack }: Props = $props();

	const CODE_LENGTH = 6;
	/**
	 * bits-ui's PinInput re-syncs its input 0, 10 and 50 ms after every value change (each sync
	 * dispatches an `input` event) and does not cancel those timers on teardown. A sign-in answered
	 * faster than that would unmount the field (navigation) first, and the late timers read its
	 * destroyed state (Svelte derived_inert). The hand-over therefore waits until the field has
	 * been quiet for this long.
	 */
	const PIN_SYNC_MS = 60;

	let mode = $state<'code' | 'recovery'>('code');
	let code = $state('');
	let recoveryCode = $state('');
	let busy = $state(false);
	let error = $state<unknown>(null);
	let fieldCode = $state<string | null>(null);
	let codeInput = $state<HTMLInputElement | null>(null);
	let recoveryInput = $state<HTMLInputElement | null>(null);
	let codeActivityAt = 0;
	const lockout = new Lockout();

	const locked = $derived(lockout.active);
	const shownError = $derived(lockout.error ?? error);

	onDestroy(() => lockout.stop());

	$effect(() => {
		void code;
		codeActivityAt = performance.now();
	});

	$effect(() => {
		const input = codeInput;
		if (!input) return;
		const mark = () => (codeActivityAt = performance.now());
		input.addEventListener('input', mark, true);
		return () => input.removeEventListener('input', mark, true);
	});

	/** Resolves once the code field has had no change and no PinInput re-sync for PIN_SYNC_MS. */
	async function pinSettled(): Promise<void> {
		for (;;) {
			const wait = codeActivityAt + PIN_SYNC_MS - performance.now();
			if (wait <= 0) return;
			await new Promise((resolve) => setTimeout(resolve, wait));
		}
	}

	// The focus follows the visible field: on mount, after switching and after a wrong code.
	$effect(() => {
		const target = mode === 'code' ? codeInput : recoveryInput;
		target?.focus();
	});

	function switchMode(next: 'code' | 'recovery'): void {
		mode = next;
		error = null;
		fieldCode = null;
	}

	function focusField(): void {
		(mode === 'code' ? codeInput : recoveryInput)?.focus();
	}

	async function submit(): Promise<void> {
		if (busy || locked) return;
		const value = mode === 'code' ? code.trim() : recoveryCode.trim().toLowerCase();
		if (!value) {
			fieldCode = 'required';
			focusField();
			return;
		}
		if (mode === 'code' && !/^\d{6}$/.test(value)) {
			fieldCode = 'invalid_format';
			focusField();
			return;
		}
		fieldCode = null;
		error = null;
		busy = true;
		try {
			const me =
				mode === 'code'
					? await session.verifyTotp({ challenge, code: value })
					: await session.verifyTotp({ challenge, recoveryCode: value });
			// The button stays disabled while the page navigates away.
			await pinSettled();
			onSuccess(me, mode === 'recovery');
			return;
		} catch (err) {
			if (err instanceof ApiError && err.is('challenge_invalid')) {
				onChallengeInvalid(err);
				return;
			}
			if (err instanceof ApiError && err.is('too_many_attempts')) {
				lockout.start(retryAfterSeconds(err));
			} else {
				error = err;
			}
			code = '';
			recoveryCode = '';
		}
		busy = false;
		// The field is enabled again only after the DOM update.
		await tick();
		focusField();
	}

	function onsubmit(event: SubmitEvent): void {
		event.preventDefault();
		void submit();
	}
</script>

<form class="flex flex-col gap-5" novalidate {onsubmit}>
	<div class="flex flex-col gap-1">
		<h1 class="heading-2 text-foreground">{t('auth.totp.title')}</h1>
		<p class="text-[13px] leading-5 text-muted-foreground">
			{mode === 'code' ? t('auth.totp.text') : t('auth.recovery.text')}
		</p>
	</div>

	{#if mode === 'code'}
		<FormField id="login-totp-code" label={t('auth.totp.code')} errorCode={fieldCode}>
			{#snippet children(props)}
				<InputOTP.Root
					maxlength={CODE_LENGTH}
					pattern={'^\\d+$'}
					pushPasswordManagerStrategy="none"
					bind:value={code}
					bind:inputRef={codeInput}
					inputId={props.id}
					aria-invalid={props['aria-invalid']}
					aria-describedby={props['aria-describedby']}
					disabled={busy || locked}
					onComplete={() => void submit()}
				>
					{#snippet children({ cells })}
						<InputOTP.Group>
							{#each cells as cell, index (index)}
								<InputOTP.Slot {cell} class="size-10 font-mono text-base text-foreground" />
							{/each}
						</InputOTP.Group>
					{/snippet}
				</InputOTP.Root>
			{/snippet}
		</FormField>
	{:else}
		<FormField
			id="login-recovery-code"
			label={t('auth.recovery.code')}
			help={t('auth.recovery.help')}
			errorCode={fieldCode}
		>
			{#snippet children(props)}
				<Input
					{...props}
					bind:ref={recoveryInput}
					bind:value={recoveryCode}
					autocomplete="off"
					autocapitalize="off"
					spellcheck={false}
					placeholder={t('auth.recovery.placeholder')}
					disabled={busy || locked}
					class={cn(inputSize, 'font-mono')}
				/>
			{/snippet}
		</FormField>
	{/if}

	{#if shownError}
		<ErrorAlert error={shownError} />
	{/if}

	<div class="flex flex-col gap-2">
		<Button type="submit" class={cn(buttonSize.default, 'w-full')} disabled={busy || locked}>
			{#if busy}
				<Spinner aria-label={t('common.states.loading')} />
			{/if}
			{t('auth.totp.submit')}
		</Button>
		<Button
			type="button"
			variant="link"
			class={cn(buttonSize.sm, 'self-center')}
			disabled={busy}
			onclick={() => switchMode(mode === 'code' ? 'recovery' : 'code')}
		>
			{mode === 'code' ? t('auth.totp.useRecovery') : t('auth.totp.useCode')}
		</Button>
		<Button
			type="button"
			variant="ghost"
			class={cn(buttonSize.default, 'self-start')}
			disabled={busy}
			onclick={onBack}
		>
			<ArrowLeft class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('auth.totp.back')}
		</Button>
	</div>
</form>
