<script lang="ts">
	import { onDestroy, tick } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { ApiError } from '$lib/api/errors.js';
	import type { Me } from '$lib/api/types.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import PasswordInput from '$lib/components/app/PasswordInput.svelte';
	import StateAlert from '$lib/components/app/StateAlert.svelte';
	import { buttonSize, inputSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { errorText, errorTitle, t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';
	import { Lockout, retryAfterSeconds } from './lockout.svelte.js';
	import { signedInTarget, type LoginReason } from './redirect.js';
	import TotpStep from './TotpStep.svelte';

	/**
	 * Sign-in card of plan 8.7.2: username and password, then (with 2FA) the code step on the
	 * same card. Success continues to the open gate, the validated `next` target or "/".
	 */
	type Props = { next?: string | null; reason?: LoginReason | null };
	let { next = null, reason = null }: Props = $props();

	/** Recovery codes at or below this count trigger the "generate new codes" warning. */
	const LOW_RECOVERY_CODES = 3;

	type Notice = 'expired' | 'deleted' | 'logout' | 'challenge';
	const noticeKeys = {
		expired: { title: 'auth.expired.title', text: 'auth.expired.text' },
		deleted: { title: 'auth.accountDeleted.title', text: 'auth.accountDeleted.text' },
		logout: { title: 'auth.logout.title', text: 'auth.logout.text' }
	} as const;
	const challengeError = new ApiError(401, 'challenge_invalid');

	let step = $state<'credentials' | 'totp'>('credentials');
	let challenge = $state('');
	let username = $state('');
	let password = $state('');
	let busy = $state(false);
	let error = $state<unknown>(null);
	let usernameError = $state<string | null>(null);
	let passwordError = $state<string | null>(null);
	// svelte-ignore state_referenced_locally
	let notice = $state<Notice | null>(reason ?? session.lastClearReason);
	let announceTotp = $state(false);
	let usernameInput = $state<HTMLInputElement | null>(null);
	let passwordInput = $state<HTMLInputElement | null>(null);
	const lockout = new Lockout();

	const locked = $derived(lockout.active);
	const shownError = $derived(lockout.error ?? error);

	onDestroy(() => lockout.stop());

	// The username field takes the focus on arrival and when the card returns to the first step.
	$effect(() => {
		if (step === 'credentials') usernameInput?.focus();
	});

	function finish(me: Me): void {
		void goto(signedInTarget(me, next, location.origin), { replaceState: true });
	}

	function onTotpSuccess(me: Me, recovery: boolean): void {
		if (recovery) {
			const count = me.recoveryCodesRemaining;
			const message = t('auth.recovery.used', { count });
			if (count <= LOW_RECOVERY_CODES) {
				toast.warning(message, { description: t('auth.recovery.low') });
			} else {
				toast.info(message);
			}
		}
		finish(me);
	}

	function backToCredentials(nextNotice: Notice | null): void {
		challenge = '';
		password = '';
		error = null;
		announceTotp = false;
		notice = nextNotice;
		step = 'credentials';
	}

	async function submit(): Promise<void> {
		if (busy || locked) return;
		const name = username.trim();
		usernameError = name ? null : 'required';
		passwordError = password ? null : 'required';
		if (usernameError || passwordError) {
			(usernameError ? usernameInput : passwordInput)?.focus();
			return;
		}
		error = null;
		notice = null;
		busy = true;
		let focusTarget: HTMLInputElement | null = usernameInput;
		try {
			const res = await session.login(name, password);
			if (res.status === 'ok') {
				// The button stays disabled while the page navigates away.
				finish(res.user);
				return;
			}
			challenge = res.challenge;
			password = '';
			announceTotp = true;
			step = 'totp';
			busy = false;
			return;
		} catch (err) {
			if (err instanceof ApiError && err.is('too_many_attempts')) {
				lockout.start(retryAfterSeconds(err));
			} else {
				error = err;
			}
			if (err instanceof ApiError && err.is('validation_failed')) {
				usernameError = err.field('username') ?? null;
				passwordError = err.field('password') ?? null;
				if (!usernameError && passwordError) focusTarget = passwordInput;
			}
			if (err instanceof ApiError && err.is('invalid_credentials')) {
				password = '';
				focusTarget = passwordInput;
			}
		}
		busy = false;
		// The fields are enabled again only after the DOM update.
		await tick();
		focusTarget?.focus();
	}

	function onsubmit(event: SubmitEvent): void {
		event.preventDefault();
		void submit();
	}
</script>

<div class="flex flex-col gap-6">
	<p class="sr-only" aria-live="polite">{announceTotp ? t('auth.totp.title') : ''}</p>

	{#if step === 'totp'}
		<TotpStep
			{challenge}
			onSuccess={onTotpSuccess}
			onChallengeInvalid={() => backToCredentials('challenge')}
			onBack={() => backToCredentials(null)}
		/>
	{:else}
		<form class="flex flex-col gap-5" novalidate {onsubmit}>
			<div class="flex flex-col gap-1">
				<h1 class="heading-2 text-foreground">{t('auth.login.title')}</h1>
				<p class="text-[13px] leading-5 text-muted-foreground">{t('auth.login.subtitle')}</p>
			</div>

			<!-- Plan 8.7.2 order: title, subtitle, then the info alert of the previous session. -->
			{#if notice === 'challenge'}
				<StateAlert
					kind="info"
					title={errorTitle(challengeError)}
					text={errorText(challengeError)}
				/>
			{:else if notice}
				<StateAlert
					kind="info"
					title={t(noticeKeys[notice].title)}
					text={t(noticeKeys[notice].text)}
				/>
			{/if}

			<FormField
				id="login-username"
				label={t('auth.login.username')}
				field="username"
				errorCode={usernameError}
			>
				{#snippet children(props)}
					<Input
						{...props}
						bind:ref={usernameInput}
						bind:value={username}
						name="username"
						autocomplete="username"
						autocapitalize="off"
						spellcheck={false}
						disabled={busy}
						class={inputSize}
					/>
				{/snippet}
			</FormField>

			<FormField
				id="login-password"
				label={t('auth.login.password')}
				field="password"
				errorCode={passwordError}
			>
				{#snippet children(props)}
					<PasswordInput
						{...props}
						bind:ref={passwordInput}
						bind:value={password}
						name="password"
						autocomplete="current-password"
						disabled={busy}
					/>
				{/snippet}
			</FormField>

			{#if shownError}
				<ErrorAlert error={shownError} />
			{/if}

			<div class="flex flex-col gap-3">
				<Button type="submit" class={cn(buttonSize.default, 'w-full')} disabled={busy || locked}>
					{#if busy}
						<Spinner aria-label={t('common.states.loading')} />
					{/if}
					{t('auth.login.submit')}
				</Button>
				<p class="text-[13px] leading-5 text-muted-foreground">{t('auth.login.noAccount')}</p>
			</div>
		</form>
	{/if}
</div>
