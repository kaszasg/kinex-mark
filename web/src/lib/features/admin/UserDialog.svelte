<script lang="ts">
	// primary-exception: the create/save button belongs to its own dialog screen.
	import { untrack } from 'svelte';
	import { adminApi } from '$lib/api/index.js';
	import { ApiError, type FieldCode } from '$lib/api/errors.js';
	import type {
		AdminUser,
		AdminUserCreated,
		AdminUserPatch,
		Locale,
		Role
	} from '$lib/api/types.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import PasswordInput from '$lib/components/app/PasswordInput.svelte';
	import {
		buttonSize,
		dialogBox,
		dialogFooter,
		inputSize,
		toggleItem,
		typeBodySm,
		typeHeading3
	} from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import * as ToggleGroup from '$lib/components/ui/toggle-group/index.js';
	import { errorMessage, i18n, LOCALES, t } from '$lib/i18n/index.js';
	import { publicConfig } from '$lib/state/public-config.svelte.js';
	import { cn } from '$lib/utils.js';
	import AdminSelect from './AdminSelect.svelte';
	import { ROLES } from './users-query.js';

	/**
	 * Create and edit dialog of plan 8.7.7. Create: username, display name, role, language and a
	 * generated or typed password. Edit: display name, role, active; a change of role or active
	 * asks for the admin's own password (step-up). On the admin's own row role and active are
	 * locked (self_modification).
	 */
	type Props = {
		open?: boolean;
		/** The user to edit; null opens the create form. */
		user?: AdminUser | null;
		/** The signed-in admin's id, to lock role and active on the own account. */
		selfId?: number | null;
		onCreated?: (result: AdminUserCreated) => void;
		onSaved?: (user: AdminUser) => void;
	};
	let { open = $bindable(false), user = null, selfId = null, onCreated, onSaved }: Props = $props();

	const USERNAME_PATTERN = /^[A-Za-z0-9._-]+$/;
	const formId = $props.id();

	let username = $state('');
	let displayName = $state('');
	let role = $state<Role>('customer');
	let locale = $state<Locale>('hu');
	let active = $state(true);
	let passwordMode = $state<'generated' | 'manual'>('generated');
	let password = $state('');
	let currentPassword = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let fieldTexts = $state<Record<string, string>>({});
	let error = $state<unknown>(null);
	let busy = $state(false);
	let usernameInput = $state<HTMLInputElement | null>(null);
	let displayNameInput = $state<HTMLInputElement | null>(null);

	/** The server policy (plan 6.4); the (app) layout loads the public configuration. */
	const DEFAULT_POLICY = { minLength: 12, minDistinct: 6 };
	const policy = $derived(publicConfig.config?.passwordPolicy ?? DEFAULT_POLICY);
	const editing = $derived(user !== null);
	const isSelf = $derived(user !== null && user.id === selfId);
	const stepUp = $derived(user !== null && (role !== user.role || active !== user.active));
	const changed = $derived(
		user !== null &&
			(displayName.trim() !== user.displayName || role !== user.role || active !== user.active)
	);
	const canSubmit = $derived(
		!busy && (!editing || (changed && (!stepUp || currentPassword.length > 0)))
	);

	const roleOptions = $derived([
		{ options: ROLES.map((value) => ({ value, label: t(`roles.${value}`) })) }
	]);
	const localeOptions = $derived([
		{
			options: LOCALES.map((value) => ({ value, label: t(`nav.locale.${value}`), lang: value }))
		}
	]);

	/** Resets the form whenever the dialog opens. */
	$effect(() => {
		if (open) untrack(reset);
	});

	function reset(): void {
		username = '';
		displayName = user?.displayName ?? '';
		role = user?.role ?? 'customer';
		active = user?.active ?? true;
		locale = i18n.locale;
		passwordMode = 'generated';
		password = '';
		currentPassword = '';
		fieldErrors = {};
		fieldTexts = {};
		error = null;
		busy = false;
	}

	function validate(): Record<string, FieldCode> {
		const issues: Record<string, FieldCode> = {};
		if (!editing) {
			const name = username.trim();
			if (!name) issues.username = 'required';
			else if (name.length < 3) issues.username = 'too_short';
			else if (name.length > 64) issues.username = 'too_long';
			else if (!USERNAME_PATTERN.test(name)) issues.username = 'invalid_format';
			if (passwordMode === 'manual' && !password) issues.password = 'required';
		}
		if ([...displayName.trim()].length > 100) issues.displayName = 'too_long';
		if (stepUp && !currentPassword) issues.currentPassword = 'required';
		return issues;
	}

	function focusFirstError(): void {
		queueMicrotask(() => {
			const invalid = document.querySelector<HTMLElement>(
				`[data-form="${formId}"] [aria-invalid="true"]`
			);
			invalid?.focus();
		});
	}

	function showError(err: unknown): void {
		if (!(err instanceof ApiError)) {
			error = err;
			return;
		}
		if (err.is('username_taken')) {
			fieldTexts = { username: errorMessage(err) };
		} else if (err.is('current_password_invalid')) {
			fieldTexts = { currentPassword: errorMessage(err) };
		} else if (err.is('validation_failed') && Object.keys(err.fields).length > 0) {
			fieldErrors = { ...err.fields };
		} else {
			error = err;
			return;
		}
		focusFirstError();
	}

	async function submit(event: Event): Promise<void> {
		event.preventDefault();
		if (busy) return;
		error = null;
		fieldTexts = {};
		fieldErrors = validate();
		if (Object.keys(fieldErrors).length > 0) {
			focusFirstError();
			return;
		}
		busy = true;
		try {
			if (user) {
				const patch: AdminUserPatch = {};
				if (displayName.trim() !== user.displayName) patch.displayName = displayName.trim();
				if (role !== user.role) patch.role = role;
				if (active !== user.active) patch.active = active;
				if (stepUp) patch.currentPassword = currentPassword;
				const saved = await adminApi.updateUser(user.id, patch);
				open = false;
				onSaved?.(saved);
			} else {
				const created = await adminApi.createUser({
					username: username.trim(),
					displayName: displayName.trim() || undefined,
					role,
					locale,
					password: passwordMode === 'manual' ? password : undefined
				});
				open = false;
				onCreated?.(created);
			}
		} catch (err) {
			showError(err);
		} finally {
			busy = false;
		}
	}

	const fieldError = (name: string): string | null => fieldErrors[name] ?? null;
	const fieldText = (name: string): string | null => fieldTexts[name] ?? null;
</script>

<Dialog.Root bind:open={() => open, (value: boolean) => (open = busy ? open : value)}>
	<Dialog.Content
		class={dialogBox}
		showCloseButton={false}
		onOpenAutoFocus={(event: Event) => {
			event.preventDefault();
			(editing ? displayNameInput : usernameInput)?.focus();
		}}
	>
		<form class="flex flex-col gap-4" data-form={formId} novalidate onsubmit={submit}>
			<Dialog.Header class="gap-1.5 text-left">
				<Dialog.Title class={typeHeading3}>
					{user
						? t('admin.userDialog.editTitle', { username: user.username })
						: t('admin.userDialog.createTitle')}
				</Dialog.Title>
				<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
					{editing ? t('admin.userDialog.editText') : t('admin.userDialog.createText')}
				</Dialog.Description>
			</Dialog.Header>

			{#if !editing}
				<FormField
					id="{formId}-username"
					label={t('admin.userDialog.fields.username')}
					help={t('admin.userDialog.usernameHelp')}
					field="username"
					errorCode={fieldError('username')}
					errorMessage={fieldText('username')}
				>
					{#snippet children(control)}
						<Input
							{...control}
							bind:ref={usernameInput}
							bind:value={username}
							name="username"
							autocomplete="off"
							autocapitalize="off"
							spellcheck={false}
							maxlength={64}
							class="{inputSize} font-mono"
						/>
					{/snippet}
				</FormField>
			{/if}

			<FormField
				id="{formId}-displayName"
				label={t('admin.userDialog.fields.displayName')}
				help={t('admin.userDialog.displayNameHelp')}
				field="displayName"
				errorCode={fieldError('displayName')}
			>
				{#snippet children(control)}
					<Input
						{...control}
						bind:ref={displayNameInput}
						bind:value={displayName}
						name="displayName"
						autocomplete="off"
						class={inputSize}
					/>
				{/snippet}
			</FormField>

			<FormField
				id="{formId}-role"
				label={t('admin.userDialog.fields.role')}
				help={isSelf ? t('admin.userDialog.selfHint') : undefined}
				field="role"
				errorCode={fieldError('role')}
			>
				{#snippet children(control)}
					<AdminSelect
						{...control}
						value={role}
						groups={roleOptions}
						disabled={isSelf}
						onchange={(value) => (role = value as Role)}
					/>
				{/snippet}
			</FormField>

			{#if editing}
				<div class="flex items-start justify-between gap-4">
					<div class="flex min-w-0 flex-col gap-0.5">
						<label for="{formId}-active" class="text-[13px] leading-5 font-medium">
							{t('admin.userDialog.fields.active')}
						</label>
						<p id="{formId}-active-help" class="text-[13px] leading-5 text-muted-foreground">
							{isSelf ? t('admin.userDialog.selfHint') : t('admin.userDialog.activeHelp')}
						</p>
					</div>
					<Switch
						id="{formId}-active"
						bind:checked={active}
						disabled={isSelf}
						aria-describedby="{formId}-active-help"
						class="mt-0.5"
					/>
				</div>
				{#if stepUp}
					<FormField
						id="{formId}-currentPassword"
						label={t('common.stepUp.label')}
						help={t('common.stepUp.help')}
						field="currentPassword"
						errorCode={fieldError('currentPassword')}
						errorMessage={fieldText('currentPassword')}
					>
						{#snippet children(control)}
							<PasswordInput
								{...control}
								bind:value={currentPassword}
								name="currentPassword"
								autocomplete="current-password"
							/>
						{/snippet}
					</FormField>
				{/if}
			{:else}
				<FormField
					id="{formId}-locale"
					label={t('admin.userDialog.fields.locale')}
					help={t('admin.userDialog.localeHelp')}
					field="locale"
					errorCode={fieldError('locale')}
				>
					{#snippet children(control)}
						<AdminSelect
							{...control}
							value={locale}
							groups={localeOptions}
							onchange={(value) => (locale = value as Locale)}
						/>
					{/snippet}
				</FormField>

				<div class="flex flex-col gap-1.5">
					<span id="{formId}-mode-label" class="text-[13px] leading-5 font-medium">
						{t('admin.userDialog.passwordMode.label')}
					</span>
					<ToggleGroup.Root
						type="single"
						variant="outline"
						bind:value={
							() => passwordMode,
							(value: string) => {
								if (value === 'generated' || value === 'manual') passwordMode = value;
							}
						}
						aria-labelledby="{formId}-mode-label"
						class="w-full"
					>
						<ToggleGroup.Item value="generated" class={cn(toggleItem, 'h-9 flex-1 text-[13px]')}>
							{t('admin.userDialog.passwordMode.generated')}
						</ToggleGroup.Item>
						<ToggleGroup.Item value="manual" class={cn(toggleItem, 'h-9 flex-1 text-[13px]')}>
							{t('admin.userDialog.passwordMode.manual')}
						</ToggleGroup.Item>
					</ToggleGroup.Root>
					{#if passwordMode === 'generated'}
						<p class="text-[13px] leading-5 text-muted-foreground">
							{t('admin.userDialog.generatedHelp')}
						</p>
					{/if}
				</div>

				{#if passwordMode === 'manual'}
					<FormField
						id="{formId}-password"
						label={t('admin.userDialog.fields.password')}
						help={t('admin.userDialog.passwordHelp', {
							min: policy.minLength,
							distinct: policy.minDistinct
						})}
						field="password"
						errorCode={fieldError('password')}
					>
						{#snippet children(control)}
							<PasswordInput
								{...control}
								bind:value={password}
								name="password"
								autocomplete="new-password"
							/>
						{/snippet}
					</FormField>
				{/if}
			{/if}

			{#if error}
				<ErrorAlert {error} />
			{/if}

			<Dialog.Footer class={dialogFooter}>
				<Button
					type="button"
					variant="outline"
					class={buttonSize.default}
					disabled={busy}
					onclick={() => (open = false)}
				>
					{t('common.actions.cancel')}
				</Button>
				<Button type="submit" class={buttonSize.default} disabled={!canSubmit} aria-busy={busy}>
					{#if busy}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
					{editing ? t('admin.userDialog.save') : t('admin.userDialog.create')}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
