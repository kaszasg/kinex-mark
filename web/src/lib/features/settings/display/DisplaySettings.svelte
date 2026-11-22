<script lang="ts">
	import LayoutGrid from '@lucide/svelte/icons/layout-grid';
	import List from '@lucide/svelte/icons/list';
	import Monitor from '@lucide/svelte/icons/monitor';
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';
	import type { Component } from 'svelte';
	import type { PreferencesPatch } from '$lib/api/me.js';
	import type { Preferences } from '$lib/api/types.js';
	import Favicon from '$lib/components/app/Favicon.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import { inputSize, toggleItem } from '$lib/components/app/styles.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import * as ToggleGroup from '$lib/components/ui/toggle-group/index.js';
	import { isLocale, LOCALES, t, type MessageKey } from '$lib/i18n/index.js';
	import { preferences, type PreferencesStore } from '$lib/state/preferences.svelte.js';
	import { cn } from '$lib/utils.js';
	import ArchivePeriodSelect from './ArchivePeriodSelect.svelte';
	import { previewLines, type DisplayFields } from './preview.js';
	import SettingRow from './SettingRow.svelte';
	import SettingsCard from './SettingsCard.svelte';

	/**
	 * Settings > Display (plan 8.7.6). Every control saves at once through the preferences store
	 * (optimistic, merged PATCH, revert and toast on failure); there is no save button.
	 */
	type Props = { store?: PreferencesStore };
	let { store = preferences }: Props = $props();

	const id = $props.id();
	const prefs = $derived(store.current);

	type ChoiceKey = 'theme' | 'viewMode' | 'sort' | 'tagMatch';
	type Choice<V extends string = string> = {
		value: V;
		key: MessageKey;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		icon?: Component<any>;
	};

	const themes: Choice<Preferences['theme']>[] = [
		{ value: 'dark', key: 'settings.display.theme.dark', icon: Moon },
		{ value: 'light', key: 'settings.display.theme.light', icon: Sun },
		{ value: 'system', key: 'settings.display.theme.system', icon: Monitor }
	];
	const viewModes: Choice<Preferences['viewMode']>[] = [
		{ value: 'list', key: 'settings.display.viewMode.list', icon: List },
		{ value: 'tiles', key: 'settings.display.viewMode.tiles', icon: LayoutGrid }
	];
	const sorts: Choice<Preferences['sort']>[] = [
		{ value: 'usage', key: 'settings.display.sort.usage' },
		{ value: 'created', key: 'settings.display.sort.created' },
		{ value: 'title', key: 'settings.display.sort.title' }
	];
	const tagMatches: Choice<Preferences['tagMatch']>[] = [
		{ value: 'any', key: 'settings.display.tagMatch.any' },
		{ value: 'all', key: 'settings.display.tagMatch.all' }
	];
	const fieldSwitches: { field: keyof DisplayFields; key: MessageKey }[] = [
		{ field: 'showIcon', key: 'settings.display.fields.icon' },
		{ field: 'showUrl', key: 'settings.display.fields.url' },
		{ field: 'showTitle', key: 'settings.display.fields.title' },
		{ field: 'showDescription', key: 'settings.display.fields.description' }
	];

	const preview = $derived(
		prefs
			? previewLines(
					{
						title: t('settings.display.preview.title'),
						description: t('settings.display.preview.description'),
						url: t('settings.display.preview.url')
					},
					prefs
				)
			: null
	);

	function save(patch: PreferencesPatch): void {
		store.update(patch);
	}

	/** A single-choice change; the toggle group reports '' when the chosen item is clicked again. */
	function pick(key: ChoiceKey, choices: Choice[], next: string): void {
		const choice = choices.find((item) => item.value === next);
		if (!choice || !prefs || prefs[key] === choice.value) return;
		save({ [key]: choice.value } as PreferencesPatch);
	}

	function pickLocale(next: string): void {
		if (!isLocale(next) || !prefs || prefs.locale === next) return;
		save({ locale: next });
	}

	const segmentToggleItem = cn(toggleItem, 'h-8 gap-1.5 px-3 text-[13px] leading-5 font-normal');
</script>

{#snippet segmented(
	key: ChoiceKey,
	choices: Choice[],
	current: string,
	labelId: string,
	helpId: string | undefined
)}
	<ToggleGroup.Root
		type="single"
		variant="outline"
		aria-labelledby={labelId}
		aria-describedby={helpId}
		class="flex-wrap"
		bind:value={() => current, (next: string) => pick(key, choices, next)}
	>
		{#each choices as choice (choice.value)}
			<ToggleGroup.Item value={choice.value} class={segmentToggleItem}>
				{#if choice.icon}
					{@const Icon = choice.icon}
					<Icon class="size-4" strokeWidth={1.5} aria-hidden="true" />
				{/if}
				{t(choice.key)}
			</ToggleGroup.Item>
		{/each}
	</ToggleGroup.Root>
{/snippet}

{#if !prefs || !preview}
	<LoadingRows rows={6} rowClass="h-12" />
{:else}
	<p class="body-sm text-muted-foreground">{t('settings.display.intro')}</p>

	<SettingsCard title={t('settings.display.appearance.title')}>
		<SettingRow label={t('settings.display.language.label')} controlId={`${id}-locale`}>
			{#snippet children()}
				<Select.Root type="single" bind:value={() => prefs.locale, pickLocale}>
					<Select.Trigger id={`${id}-locale`} class={cn(inputSize, 'w-full sm:w-44')}>
						<span lang={prefs.locale}>{t(`nav.locale.${prefs.locale}`)}</span>
					</Select.Trigger>
					<Select.Content>
						{#each LOCALES as locale (locale)}
							<Select.Item value={locale} label={t(`nav.locale.${locale}`)} lang={locale} />
						{/each}
					</Select.Content>
				</Select.Root>
			{/snippet}
		</SettingRow>
		<SettingRow label={t('settings.display.theme.label')}>
			{#snippet children({ labelId, helpId })}
				{@render segmented('theme', themes, prefs.theme, labelId, helpId)}
			{/snippet}
		</SettingRow>
	</SettingsCard>

	<SettingsCard
		title={t('settings.display.list.title')}
		description={t('settings.display.list.description')}
	>
		<SettingRow label={t('settings.display.viewMode.label')}>
			{#snippet children({ labelId, helpId })}
				{@render segmented('viewMode', viewModes, prefs.viewMode, labelId, helpId)}
			{/snippet}
		</SettingRow>

		<SettingRow
			label={t('settings.display.fields.label')}
			help={t('settings.display.fields.help')}
			stacked
		>
			{#snippet children({ labelId, helpId })}
				<div
					role="group"
					aria-labelledby={labelId}
					aria-describedby={helpId}
					class="grid grid-cols-2 gap-x-6 gap-y-2 sm:grid-cols-4"
				>
					{#each fieldSwitches as item (item.field)}
						<div class="flex items-center gap-2">
							<Switch
								id={`${id}-${item.field}`}
								bind:checked={
									() => prefs[item.field], (next: boolean) => save({ [item.field]: next })
								}
							/>
							<Label for={`${id}-${item.field}`} class="text-[13px] leading-5 font-normal">
								{t(item.key)}
							</Label>
						</div>
					{/each}
				</div>
				<figure
					class="flex min-w-0 flex-col gap-1.5"
					aria-labelledby={`${id}-preview`}
					data-testid="display-preview"
				>
					<figcaption id={`${id}-preview`} class="text-[13px] leading-5 text-muted-foreground">
						{t('settings.display.preview.label')}
					</figcaption>
					<div
						class="grid min-h-11 grid-cols-[20px_minmax(0,1fr)] items-start gap-x-3 rounded-md border bg-background px-4 py-2"
					>
						<span class="flex h-6 items-center">
							{#if preview.icon}<Favicon src={null} size={20} />{/if}
						</span>
						<div class="flex min-w-0 flex-col">
							<span class="truncate text-[15px] leading-6 font-medium" data-testid="preview-label">
								{preview.label}
							</span>
							{#if preview.title}
								<span class="body-sm truncate text-muted-foreground" data-testid="preview-title">
									{preview.title}
								</span>
							{/if}
							{#if preview.url}
								<span
									class="truncate font-mono text-[13px] leading-5 text-muted-foreground"
									data-testid="preview-url"
								>
									{preview.url}
								</span>
							{/if}
						</div>
					</div>
				</figure>
			{/snippet}
		</SettingRow>

		<SettingRow label={t('settings.display.archived.label')}>
			{#snippet children()}
				<div class="flex items-center gap-2">
					<Switch
						id={`${id}-showArchived`}
						bind:checked={() => prefs.showArchived, (next: boolean) => save({ showArchived: next })}
					/>
					<Label for={`${id}-showArchived`} class="text-[13px] leading-5 font-normal">
						{t('settings.display.archived.toggle')}
					</Label>
				</div>
			{/snippet}
		</SettingRow>

		<SettingRow label={t('settings.display.sort.label')}>
			{#snippet children({ labelId, helpId })}
				{@render segmented('sort', sorts, prefs.sort, labelId, helpId)}
			{/snippet}
		</SettingRow>

		<SettingRow
			label={t('settings.display.tagMatch.label')}
			help={t('settings.display.tagMatch.help')}
		>
			{#snippet children({ labelId, helpId })}
				{@render segmented('tagMatch', tagMatches, prefs.tagMatch, labelId, helpId)}
			{/snippet}
		</SettingRow>
	</SettingsCard>

	<SettingsCard title={t('settings.display.archive.title')}>
		<SettingRow
			label={t('settings.display.archive.label')}
			help={t('settings.display.archive.help')}
			controlId={`${id}-archive`}
			align="start"
		>
			{#snippet children({ helpId })}
				<ArchivePeriodSelect
					id={`${id}-archive`}
					describedBy={helpId}
					value={prefs.archiveAfterDays}
					systemDays={prefs.systemArchiveAfterDays}
					onChange={(next) => save({ archiveAfterDays: next })}
				/>
			{/snippet}
		</SettingRow>
	</SettingsCard>
{/if}
