<script lang="ts" module>
	/** Longest display name the server accepts, in Unicode code points (plan 5.8). */
	export const DISPLAY_NAME_MAX = 100;
</script>

<script lang="ts">
	import { toast } from 'svelte-sonner';
	import { ApiError } from '$lib/api/errors.js';
	import * as meApi from '$lib/api/me.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import { buttonSize, inputSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import SettingsCard from '../display/SettingsCard.svelte';

	/**
	 * Settings > Profile (plan 8.7.6): username (read-only, mono), display name (0-100), role in
	 * words, account creation and last sign-in. "Save profile" sends PATCH /api/me/profile.
	 */
	const me = $derived(session.me);

	let displayName = $state('');
	let loadedFor: number | null = null;
	let busy = $state(false);
	let fieldCode = $state<string | null>(null);
	let error = $state<unknown>(null);

	// The field starts from the signed-in user's name (and follows a user switch).
	$effect.pre(() => {
		const current = session.me;
		if (current && current.id !== loadedFor) {
			loadedFor = current.id;
			displayName = current.displayName;
		}
	});

	const trimmed = $derived(displayName.trim());
	const dirty = $derived(me !== null && trimmed !== me.displayName);

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (busy || !me || !dirty) return;
		error = null;
		if ([...trimmed].length > DISPLAY_NAME_MAX) {
			fieldCode = 'too_long';
			return;
		}
		fieldCode = null;
		busy = true;
		try {
			const updated = await meApi.updateProfile({ displayName: trimmed });
			// Keep the preferences already in the session: pending display changes stay applied.
			const current = session.me;
			session.setMe(current ? { ...updated, preferences: current.preferences } : updated);
			displayName = updated.displayName;
			toast.success(t('settings.profile.saved'));
		} catch (err) {
			const apiError = err instanceof ApiError ? err : new ApiError(0, 'network_error');
			const code = apiError.is('validation_failed') ? apiError.field('displayName') : undefined;
			if (code) fieldCode = code;
			else error = apiError;
		} finally {
			busy = false;
		}
	}
</script>

{#if !me}
	<LoadingRows rows={4} rowClass="h-12" />
{:else}
	<form novalidate onsubmit={submit}>
		<SettingsCard
			title={t('settings.profile.title')}
			description={t('settings.profile.description')}
		>
			<dl class="grid gap-x-6 gap-y-3 px-4 py-4 sm:grid-cols-[minmax(0,12rem)_minmax(0,1fr)]">
				<dt class="text-[13px] leading-5 text-muted-foreground">
					{t('settings.profile.username')}
				</dt>
				<dd class="flex min-w-0 flex-col">
					<span class="truncate font-mono text-[13px] leading-5" data-testid="profile-username">
						{me.username}
					</span>
					<span class="text-[13px] leading-5 text-muted-foreground">
						{t('settings.profile.usernameHelp')}
					</span>
				</dd>
				<dt class="text-[13px] leading-5 text-muted-foreground">{t('settings.profile.role')}</dt>
				<dd class="text-[13px] leading-5">{t(`roles.${me.role}`)}</dd>
				<dt class="text-[13px] leading-5 text-muted-foreground">
					{t('settings.profile.createdAt')}
				</dt>
				<dd class="font-mono text-[13px] leading-5">
					<time datetime={me.createdAt}>{fmt.dateTime(me.createdAt)}</time>
				</dd>
				<dt class="text-[13px] leading-5 text-muted-foreground">
					{t('settings.profile.lastLoginAt')}
				</dt>
				<dd class="font-mono text-[13px] leading-5" data-testid="profile-last-login">
					{#if me.lastLoginAt}
						<time datetime={me.lastLoginAt}>{fmt.dateTime(me.lastLoginAt)}</time>
					{:else}
						{fmt.missing}
					{/if}
				</dd>
			</dl>
			<div class="flex flex-col gap-4 px-4 py-4">
				<FormField
					id="profile-display-name"
					label={t('settings.profile.displayName')}
					field="displayName"
					help={t('settings.profile.displayNameHelp', { max: DISPLAY_NAME_MAX })}
					errorCode={fieldCode}
				>
					{#snippet children(control)}
						<Input
							{...control}
							bind:value={displayName}
							oninput={() => (fieldCode = null)}
							autocomplete="name"
							disabled={busy}
							class={inputSize}
						/>
					{/snippet}
				</FormField>
				{#if error}
					<ErrorAlert {error} />
				{/if}
			</div>
			{#snippet footer()}
				<Button type="submit" class={buttonSize.default} disabled={busy || !dirty} aria-busy={busy}>
					{#if busy}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
					{t('settings.profile.save')}
				</Button>
			{/snippet}
		</SettingsCard>
	</form>
{/if}
