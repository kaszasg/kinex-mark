<script lang="ts" module>
	export type EnrollStep = 'password' | 'scan' | 'codes';

	const STEPS: EnrollStep[] = ['password', 'scan', 'codes'];

	/** The base32 secret in groups of four for manual entry. */
	export function groupSecret(secret: string): string {
		return (secret.match(/.{1,4}/g) ?? []).join(' ');
	}
</script>

<script lang="ts">
	import { REGEXP_ONLY_DIGITS } from 'bits-ui';
	import { tick } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { ApiError } from '$lib/api/errors.js';
	import * as meApi from '$lib/api/me.js';
	import type { TwoFactorSetup } from '$lib/api/types.js';
	import CopyButton from '$lib/components/app/CopyButton.svelte';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import AccountUsername from '$lib/components/app/AccountUsername.svelte';
	import PasswordInput from '$lib/components/app/PasswordInput.svelte';
	import QrCode from '$lib/components/app/QrCode.svelte';
	import { buttonSize, dialogBox, typeBodySm, typeHeading3 } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as InputOTP from '$lib/components/ui/input-otp/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { fieldMessage, fmt, t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';
	import { focusFirstInvalid, formErrors, NO_ERRORS, type FormErrors } from './form-errors.js';
	import RecoveryCodesView from './RecoveryCodesView.svelte';

	/**
	 * Two-factor enrolment of plan 8.7.6 in three numbered steps: confirm the password
	 * (POST /me/2fa/setup), scan the QR code and enter a code (POST /me/2fa/enable), save the
	 * recovery codes. The dialog cannot be dismissed on the last step until the codes are saved;
	 * closing it then refreshes the session and confirms with a toast.
	 */
	type Props = { open?: boolean; onEnabled?: () => void };
	let { open = $bindable(false), onEnabled }: Props = $props();

	const uid = $props.id();

	let step = $state<EnrollStep>('password');
	let password = $state('');
	let code = $state('');
	let setup = $state<TwoFactorSetup | null>(null);
	let codes = $state<string[]>([]);
	let generatedAt = $state(new Date());
	let saved = $state(false);
	let busy = $state(false);
	let errors = $state<FormErrors>(NO_ERRORS);
	let form = $state<HTMLFormElement | null>(null);

	const stepNumber = $derived(STEPS.indexOf(step) + 1);
	const username = $derived(session.me?.username ?? '');

	let wasOpen = false;
	$effect(() => {
		if (open && !wasOpen) reset();
		wasOpen = open;
	});

	function reset(): void {
		step = 'password';
		password = '';
		code = '';
		setup = null;
		codes = [];
		saved = false;
		busy = false;
		errors = NO_ERRORS;
	}

	async function focusFirstField(): Promise<void> {
		await tick();
		form?.querySelector<HTMLElement>('input:not([type="hidden"])')?.focus();
	}

	function requestClose(): void {
		if (busy) return;
		if (step === 'codes') {
			if (saved) finish();
			return;
		}
		password = '';
		open = false;
	}

	function finish(): void {
		open = false;
		toast.success(t('security.twoFactor.enabled'));
		onEnabled?.();
		void session.refresh().catch(() => undefined);
	}

	function fail(next: FormErrors): void {
		errors = next;
		void focusFirstInvalid(form);
	}

	async function submitPassword(): Promise<void> {
		if (!password)
			return fail({ fields: { password: fieldMessage('password', 'required') }, alert: null });
		busy = true;
		errors = NO_ERRORS;
		try {
			setup = await meApi.setupTwoFactor(password);
			password = '';
			code = '';
			step = 'scan';
			void focusFirstField();
		} catch (err) {
			fail(formErrors(err, { current_password_invalid: 'password' }, ['password']));
		} finally {
			busy = false;
		}
	}

	async function submitCode(): Promise<void> {
		if (code.length !== 6) {
			const fieldCode = code ? 'invalid_format' : 'required';
			return fail({ fields: { code: fieldMessage('code', fieldCode) }, alert: null });
		}
		busy = true;
		errors = NO_ERRORS;
		try {
			const result = await meApi.enableTwoFactor(code);
			codes = result.recoveryCodes;
			generatedAt = new Date();
			setup = null;
			saved = false;
			step = 'codes';
		} catch (err) {
			code = '';
			if (err instanceof ApiError && err.code === 'totp_setup_required') {
				setup = null;
				step = 'password';
				errors = { fields: {}, alert: err };
				void focusFirstField();
			} else {
				fail(formErrors(err, { invalid_totp: 'code' }, ['code']));
			}
		} finally {
			busy = false;
		}
	}

	function submit(event: SubmitEvent): void {
		event.preventDefault();
		if (busy) return;
		if (step === 'password') void submitPassword();
		else if (step === 'scan') void submitCode();
		else if (saved) finish();
	}

	const titles = {
		password: 'security.enroll.password.title',
		scan: 'security.enroll.scan.title',
		codes: 'security.enroll.codes.title'
	} as const;
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
			<div class="flex items-center gap-3">
				<span class="font-mono text-[11px] leading-4 text-muted-foreground" aria-hidden="true">
					{t('security.enroll.step', { current: String(stepNumber), total: String(STEPS.length) })}
				</span>
				<span class="sr-only">
					{t('security.enroll.stepLabel', {
						current: String(stepNumber),
						total: String(STEPS.length)
					})}
				</span>
				<span class="flex flex-1 gap-1" aria-hidden="true">
					{#each STEPS as item, index (item)}
						<span
							class={cn('h-px flex-1', index < stepNumber ? 'bg-primary' : 'bg-border')}
							data-step-done={index < stepNumber ? 'true' : undefined}
						></span>
					{/each}
				</span>
			</div>

			<Dialog.Header class="gap-1.5 text-left">
				<Dialog.Title class={typeHeading3}>{t(titles[step])}</Dialog.Title>
				{#if step === 'password'}
					<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
						{t('security.enroll.password.text')}
					</Dialog.Description>
				{:else if step === 'scan'}
					<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
						{t('security.enroll.scan.text')}
					</Dialog.Description>
				{/if}
			</Dialog.Header>

			{#if step === 'password'}
				<FormField
					id="{uid}-password"
					label={t('security.enroll.password.label')}
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
			{:else if step === 'scan' && setup}
				<div class="flex flex-col items-center gap-4 sm:flex-row sm:items-start">
					<QrCode
						value={setup.otpauthUri}
						label={t('security.enroll.scan.qr')}
						size={176}
						class="shrink-0"
					/>
					<div class="flex min-w-0 flex-col gap-1.5 self-stretch">
						<p class="body-sm text-muted-foreground">{t('security.enroll.scan.manual')}</p>
						<div class="flex items-start gap-1">
							<code class="min-w-0 pt-1 font-mono text-[13px] leading-5 break-normal" data-secret>
								{groupSecret(setup.secret)}
							</code>
							<CopyButton
								text={setup.secret}
								label={t('security.enroll.scan.copySecret')}
								toastText={t('security.enroll.scan.secretCopied')}
							/>
						</div>
						<p class="body-sm text-muted-foreground">
							{t('security.enroll.scan.expires', { time: fmt.time(setup.expiresAt) })}
						</p>
					</div>
				</div>
				<FormField
					id="{uid}-code"
					label={t('security.enroll.scan.code')}
					help={t('security.enroll.scan.codeHelp')}
					field="code"
					errorMessage={errors.fields.code}
				>
					{#snippet children(control)}
						<InputOTP.Root
							maxlength={6}
							pattern={REGEXP_ONLY_DIGITS}
							pushPasswordManagerStrategy="none"
							bind:value={code}
							inputId={control.id}
							aria-invalid={control['aria-invalid']}
							aria-describedby={control['aria-describedby']}
							disabled={busy}
						>
							{#snippet children({ cells })}
								<InputOTP.Group>
									{#each cells as cell, index (index)}
										<InputOTP.Slot {cell} class="size-10 font-mono text-[15px]" />
									{/each}
								</InputOTP.Group>
							{/snippet}
						</InputOTP.Root>
					{/snippet}
				</FormField>
			{:else if step === 'codes'}
				<RecoveryCodesView {codes} {username} {generatedAt} bind:saved />
			{/if}

			{#if errors.alert}
				<ErrorAlert error={errors.alert} />
			{/if}

			<div class="flex flex-row justify-end gap-2">
				{#if step !== 'codes'}
					<Button
						type="button"
						variant="outline"
						class={buttonSize.default}
						disabled={busy}
						onclick={requestClose}
					>
						{t('common.actions.cancel')}
					</Button>
				{/if}
				<Button
					type="submit"
					class={buttonSize.default}
					disabled={busy || (step === 'codes' && !saved)}
					aria-busy={busy}
				>
					{#if busy}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
					{#if step === 'password'}
						{t('security.enroll.password.submit')}
					{:else if step === 'scan'}
						{t('security.enroll.scan.submit')}
					{:else}
						{t('common.actions.done')}
					{/if}
				</Button>
			</div>
		</form>
	</Dialog.Content>
</Dialog.Root>
