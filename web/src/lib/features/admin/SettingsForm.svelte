<script lang="ts">
	import Archive from '@lucide/svelte/icons/archive';
	import FileText from '@lucide/svelte/icons/file-text';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { beforeNavigate, goto } from '$app/navigation';
	import { adminApi } from '$lib/api/index.js';
	import { ApiError } from '$lib/api/errors.js';
	import type { AdminSettingsResponse, SystemSettings } from '$lib/api/types.js';
	import ConfirmDialog from '$lib/components/app/ConfirmDialog.svelte';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import StateAlert from '$lib/components/app/StateAlert.svelte';
	import { buttonSize, inputSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import { fmt, t, type MessageKey } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';
	import SettingsCard from './SettingsCard.svelte';
	import {
		draftPatch,
		isDirty,
		NUMBER_SPECS,
		rangeParams,
		TEXT_LIMITS,
		toDraft,
		validateDraft,
		type EditableKey,
		type FieldIssue,
		type NumberKey,
		type SettingsDraft,
		type TextKey
	} from './settings-model.js';
	import { toastError } from '$lib/utils/toast-error.js';

	/**
	 * The /admin/settings screen of plan 8.7.8: one form split into cards, a sticky save bar that
	 * is active only with changes, a leave confirmation while changes are unsaved, the archive job
	 * and the privacy notice version bump.
	 */

	let response = $state<AdminSettingsResponse | null>(null);
	let draft = $state<SettingsDraft | null>(null);
	let loadError = $state<unknown>(null);
	let loading = $state(false);
	let saving = $state(false);
	let saveError = $state<unknown>(null);
	let issues = $state<Partial<Record<EditableKey, FieldIssue>>>({});
	let archiving = $state(false);
	let bumpOpen = $state(false);
	let leaveOpen = $state(false);
	let leaveTarget: URL | null = null;
	let leaving = false;

	const saved = $derived(response?.settings ?? null);
	const dirty = $derived(draft !== null && saved !== null && isDirty(draft, saved));
	const locked = $derived(new Set(response?.lockedByEnv ?? []));

	async function load(): Promise<void> {
		loading = true;
		loadError = null;
		try {
			apply(await adminApi.getSettings());
		} catch (err) {
			loadError = err;
		} finally {
			loading = false;
		}
	}

	function apply(next: AdminSettingsResponse): void {
		response = next;
		draft = toDraft(next.settings);
		issues = {};
		saveError = null;
	}

	onMount(() => {
		void load();
	});

	beforeNavigate((navigation) => {
		if (!dirty || leaving) return;
		navigation.cancel();
		if (navigation.type === 'leave' || !navigation.to) return;
		leaveTarget = navigation.to.url;
		leaveOpen = true;
	});

	async function leave(): Promise<void> {
		if (!leaveTarget) return;
		leaving = true;
		await goto(leaveTarget);
	}

	function edit<K extends keyof SettingsDraft>(key: K, value: SettingsDraft[K]): void {
		if (!draft) return;
		draft[key] = value;
		if (issues[key]) {
			const { [key]: _removed, ...rest } = issues;
			issues = rest;
		}
	}

	function discard(): void {
		if (response) apply(response);
	}

	function focusFirstIssue(): void {
		queueMicrotask(() => {
			document.querySelector<HTMLElement>('[data-settings-form] [aria-invalid="true"]')?.focus();
		});
	}

	async function save(event: Event): Promise<void> {
		event.preventDefault();
		if (!draft || !saved || saving || !dirty) return;
		saveError = null;
		issues = validateDraft(draft);
		if (Object.keys(issues).length > 0) {
			focusFirstIssue();
			return;
		}
		saving = true;
		try {
			apply(await adminApi.updateSettings(draftPatch(draft, saved)));
			toast.success(t('admin.settings.saved'));
		} catch (err) {
			if (
				err instanceof ApiError &&
				err.is('validation_failed') &&
				Object.keys(err.fields).length
			) {
				const next: Partial<Record<EditableKey, FieldIssue>> = {};
				for (const [field, code] of Object.entries(err.fields)) {
					next[field as EditableKey] = {
						code,
						params: code === 'out_of_range' ? rangeParams(field) : undefined
					};
				}
				issues = next;
				focusFirstIssue();
			} else {
				saveError = err;
			}
		} finally {
			saving = false;
		}
	}

	async function runArchive(): Promise<void> {
		archiving = true;
		try {
			const { archived } = await adminApi.runArchiveJob();
			toast.success(t('admin.settings.archived', { count: archived }));
		} catch (err) {
			toastError(err);
		} finally {
			archiving = false;
		}
	}

	async function bump(): Promise<void> {
		const next = await adminApi.bumpPrivacyNotice();
		if (response && draft) {
			// Keep unsaved edits: only the read-only notice fields come from the answer.
			response = {
				...response,
				settings: {
					...response.settings,
					privacyNoticeVersion: next.settings.privacyNoticeVersion,
					privacyNoticeUpdatedAt: next.settings.privacyNoticeUpdatedAt
				}
			};
		}
		toast.success(
			t('admin.settings.notice.bumped', { version: next.settings.privacyNoticeVersion })
		);
	}

	type NumberField = { key: NumberKey; help: MessageKey };
	const archiveFields: NumberField[] = [
		{ key: 'defaultArchiveDays', help: 'admin.settings.help.archiveDays' }
	];
	const sessionFields: NumberField[] = [
		{ key: 'sessionIdleMinutes', help: 'admin.settings.help.minutes' },
		{ key: 'sessionAbsoluteHours', help: 'admin.settings.help.hours' }
	];
	const auditFields: NumberField[] = [
		{ key: 'auditRetentionDays', help: 'admin.settings.help.days' }
	];
	const textFields: { key: TextKey; help: MessageKey }[] = [
		{ key: 'privacyControllerName', help: 'admin.settings.help.text' },
		{ key: 'privacyControllerAddress', help: 'admin.settings.help.text' },
		{ key: 'privacyControllerEmail', help: 'admin.settings.help.email' },
		{ key: 'privacyDpoContact', help: 'admin.settings.help.text' },
		{ key: 'privacySupervisoryAuthority', help: 'admin.settings.help.supervisoryAuthority' }
	];

	const unitKeys = {
		days: 'admin.settings.units.days',
		minutes: 'admin.settings.units.minutes',
		hours: 'admin.settings.units.hours'
	} as const satisfies Record<string, MessageKey>;

	function numberHelp(field: NumberField, defaults: SystemSettings): string {
		const spec = NUMBER_SPECS[field.key];
		const parts = t(field.help, { min: spec.min, max: spec.max, default: defaults[field.key] });
		return locked.has(field.key) ? `${parts} ${t('admin.settings.locked.generic')}` : parts;
	}

	function textHelp(key: TextKey, help: MessageKey): string {
		const text = t(help, { max: TEXT_LIMITS[key] });
		return locked.has(key) ? `${text} ${t('admin.settings.locked.generic')}` : text;
	}
</script>

{#snippet numberInputs(fields: NumberField[], current: SettingsDraft, defaults: SystemSettings)}
	{#each fields as field (field.key)}
		{@const spec = NUMBER_SPECS[field.key]}
		<FormField
			id="setting-{field.key}"
			label={t(`admin.settings.fields.${field.key}`)}
			help={numberHelp(field, defaults)}
			field={field.key}
			errorCode={issues[field.key]?.code}
			errorParams={issues[field.key]?.params}
		>
			{#snippet children(control)}
				<div class="flex items-center gap-2">
					<Input
						{...control}
						type="number"
						inputmode="numeric"
						min={spec.min}
						max={spec.max}
						step={1}
						name={field.key}
						disabled={locked.has(field.key)}
						value={current[field.key]}
						oninput={(event: Event) =>
							edit(field.key, (event.currentTarget as HTMLInputElement).value)}
						class="{inputSize} w-32 text-right font-mono"
					/>
					<span class="text-[13px] leading-5 text-muted-foreground">
						{t(unitKeys[spec.unit])}
					</span>
				</div>
			{/snippet}
		</FormField>
	{/each}
{/snippet}

<PageHeader title={t('admin.settings.pageTitle')} subtitle={t('admin.settings.subtitle')} />

{#if loadError && !response}
	<ErrorAlert error={loadError} onRetry={() => void load()} retryDisabled={loading} />
{:else if !response || !draft}
	<LoadingRows rows={8} rowClass="h-12" />
{:else}
	{@const current = draft}
	{@const defaults = response.defaults}
	{@const settings = response.settings}
	<form class="flex w-full max-w-3xl flex-col gap-4" data-settings-form novalidate onsubmit={save}>
		<SettingsCard
			title={t('admin.settings.cards.archive.title')}
			text={t('admin.settings.cards.archive.text')}
		>
			{@render numberInputs(archiveFields, current, defaults)}
			<div>
				<Button
					type="button"
					variant="outline"
					class={buttonSize.sm}
					disabled={archiving}
					aria-busy={archiving}
					onclick={() => void runArchive()}
				>
					{#if archiving}
						<Spinner class="size-4" aria-label={t('common.states.loading')} />
					{:else}
						<Archive class="size-4" strokeWidth={1.5} aria-hidden="true" />
					{/if}
					{t('admin.settings.runArchive')}
				</Button>
			</div>
		</SettingsCard>

		<SettingsCard
			title={t('admin.settings.cards.sessions.title')}
			text={t('admin.settings.cards.sessions.text')}
			contentClass="grid gap-5 p-4 sm:grid-cols-2"
		>
			{@render numberInputs(sessionFields, current, defaults)}
		</SettingsCard>

		<SettingsCard
			title={t('admin.settings.cards.audit.title')}
			text={t('admin.settings.cards.audit.text')}
		>
			{@render numberInputs(auditFields, current, defaults)}
		</SettingsCard>

		<SettingsCard
			title={t('admin.settings.cards.fetch.title')}
			text={t('admin.settings.cards.fetch.text')}
			contentClass="flex flex-col gap-4 p-4"
		>
			<div class="flex items-start justify-between gap-4">
				<div class="flex min-w-0 flex-col gap-0.5">
					<label for="setting-allowPrivateFetch" class="text-[13px] leading-5 font-medium">
						{t('admin.settings.fields.allowPrivateFetch')}
					</label>
					<p
						id="setting-allowPrivateFetch-help"
						class="text-[13px] leading-5 text-muted-foreground"
					>
						{t('admin.settings.help.allowPrivateFetch')}
					</p>
				</div>
				<Switch
					id="setting-allowPrivateFetch"
					checked={current.allowPrivateFetch}
					onCheckedChange={(value: boolean) => edit('allowPrivateFetch', value)}
					disabled={locked.has('allowPrivateFetch')}
					aria-describedby="setting-allowPrivateFetch-help"
					class="mt-0.5"
				/>
			</div>
			{#if locked.has('allowPrivateFetch')}
				<StateAlert
					kind="info"
					title={t('admin.settings.locked.title')}
					text={t('admin.settings.locked.privateFetch')}
				/>
			{/if}
		</SettingsCard>

		<SettingsCard
			title={t('admin.settings.cards.privacy.title')}
			text={t('admin.settings.cards.privacy.text')}
		>
			{#each textFields as field (field.key)}
				<FormField
					id="setting-{field.key}"
					label={t(`admin.settings.fields.${field.key}`)}
					help={textHelp(field.key, field.help)}
					field={field.key}
					errorCode={issues[field.key]?.code}
					errorParams={issues[field.key]?.params}
				>
					{#snippet children(control)}
						<Input
							{...control}
							type={field.key === 'privacyControllerEmail' ? 'email' : 'text'}
							name={field.key}
							autocomplete="off"
							disabled={locked.has(field.key)}
							value={current[field.key]}
							oninput={(event: Event) =>
								edit(field.key, (event.currentTarget as HTMLInputElement).value)}
							class={cn(inputSize, field.key === 'privacyControllerEmail' && 'font-mono')}
						/>
					{/snippet}
				</FormField>
			{/each}
			<div class="flex flex-wrap items-center justify-between gap-x-6 gap-y-3 border-t pt-4">
				<dl class="flex gap-6 text-[13px] leading-5">
					<div class="flex flex-col gap-0.5">
						<dt class="text-muted-foreground">{t('admin.settings.notice.version')}</dt>
						<dd class="font-mono" data-testid="notice-version">
							{settings.privacyNoticeVersion}
						</dd>
					</div>
					<div class="flex flex-col gap-0.5">
						<dt class="text-muted-foreground">{t('admin.settings.notice.updatedAt')}</dt>
						<dd class="font-mono">
							{settings.privacyNoticeUpdatedAt
								? fmt.dateTime(settings.privacyNoticeUpdatedAt)
								: fmt.missing}
						</dd>
					</div>
				</dl>
				<Button
					type="button"
					variant="outline"
					class={buttonSize.sm}
					onclick={() => (bumpOpen = true)}
				>
					<FileText class="size-4" strokeWidth={1.5} aria-hidden="true" />
					{t('admin.settings.notice.bump')}
				</Button>
			</div>
		</SettingsCard>

		{#if saveError}
			<ErrorAlert error={saveError} />
		{/if}

		<div
			class="sticky bottom-0 z-10 -mx-4 flex flex-wrap items-center justify-end gap-x-4 gap-y-2 border-t bg-background px-4 py-3 md:-mx-6 md:px-6"
		>
			<p class="mr-auto text-[13px] leading-5 text-muted-foreground" aria-live="polite">
				{dirty ? t('admin.settings.unsaved') : t('admin.settings.noChanges')}
			</p>
			<Button
				type="button"
				variant="outline"
				class={buttonSize.default}
				disabled={!dirty || saving}
				onclick={discard}
			>
				{t('admin.settings.discard')}
			</Button>
			<Button
				type="submit"
				class={buttonSize.default}
				disabled={!dirty || saving}
				aria-busy={saving}
			>
				{#if saving}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
				{t('admin.settings.save')}
			</Button>
		</div>
	</form>
{/if}

<ConfirmDialog
	bind:open={bumpOpen}
	title={t('admin.settings.notice.bumpTitle')}
	text={t('admin.settings.notice.bumpText')}
	confirmLabel={t('admin.settings.notice.bump')}
	onConfirm={bump}
/>

<ConfirmDialog
	bind:open={leaveOpen}
	title={t('admin.settings.leave.title')}
	text={t('admin.settings.leave.text')}
	confirmLabel={t('admin.settings.leave.action')}
	destructive
	onConfirm={leave}
/>
