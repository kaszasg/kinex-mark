<script lang="ts">
	import Keyboard from '@lucide/svelte/icons/keyboard';
	import LayoutGrid from '@lucide/svelte/icons/layout-grid';
	import List from '@lucide/svelte/icons/list';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import type { Preferences } from '$lib/api/types.js';
	import { getShellContext } from '$lib/components/app/shell-context.js';
	import { buttonSize, inputSize, toggleItem } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import * as ToggleGroup from '$lib/components/ui/toggle-group/index.js';
	import { t, type MessageKey } from '$lib/i18n/index.js';
	import {
		defaultsFromPreferences,
		SCOPES,
		SORTS,
		type ListQuery,
		type Scope,
		type SortMode
	} from '$lib/links/query.js';
	import { preferences } from '$lib/state/preferences.svelte.js';
	import { links as linksStore, type LinksStore } from '$lib/state/links.svelte.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Toolbar of the links page (plan 8.7.3): scope, sort order, list or tiles, the display
	 * fields and "Include archived". Sort, view, fields and the archived switch are preferences;
	 * the scope lives only in the URL.
	 */
	type Props = { store?: LinksStore; class?: string };
	let { store = linksStore, class: className }: Props = $props();

	const shell = getShellContext();
	const prefs = $derived(preferences.current);
	const viewMode = $derived(prefs?.viewMode ?? 'list');

	const scopeLabels: Record<Scope, MessageKey> = {
		all: 'links.scope.all',
		own: 'links.scope.own',
		shared: 'links.scope.shared'
	};
	const sortLabels: Record<SortMode, MessageKey> = {
		usage: 'links.sort.usage',
		created: 'links.sort.created',
		title: 'links.sort.title'
	};
	const fieldItems: {
		key: 'showIcon' | 'showUrl' | 'showTitle' | 'showDescription';
		label: MessageKey;
	}[] = [
		{ key: 'showIcon', label: 'links.fields.icon' },
		{ key: 'showUrl', label: 'links.fields.url' },
		{ key: 'showTitle', label: 'links.fields.title' },
		{ key: 'showDescription', label: 'links.fields.description' }
	];

	/** Saves a preference that is also a query default, so the URL keeps only deviations. */
	function setPreferred(
		patch: Partial<ListQuery>,
		prefPatch: Partial<Pick<Preferences, 'sort' | 'showArchived'>>
	): void {
		preferences.update(prefPatch);
		store.defaults = defaultsFromPreferences(preferences.current);
		store.setQuery(patch);
	}

	function setScope(value: string): void {
		if ((SCOPES as readonly string[]).includes(value)) store.setQuery({ scope: value as Scope });
	}

	function setSort(value: string): void {
		if ((SORTS as readonly string[]).includes(value)) {
			const sort = value as SortMode;
			setPreferred({ sort }, { sort });
		}
	}

	function setArchived(checked: boolean): void {
		setPreferred({ archived: checked ? 'include' : 'exclude' }, { showArchived: checked });
	}

	function setView(value: string): void {
		if (value === 'list' || value === 'tiles') preferences.update({ viewMode: value });
	}
</script>

<!--
	Below 768 px the toolbar takes two rows (plan 8.9): scope and view first, then sort, the
	display menu (icon only) and the archived switch. A zero-height break item forces the row
	change; the halved row gap makes up for the extra line it creates.
-->
<div
	role="group"
	aria-label={t('links.toolbar.label')}
	class={cn('flex flex-wrap items-center gap-x-3 gap-y-3 max-md:gap-y-1.5', className)}
	data-toolbar
>
	<ToggleGroup.Root
		type="single"
		variant="outline"
		bind:value={() => store.query.scope, setScope}
		aria-label={t('links.toolbar.scope')}
	>
		{#each SCOPES as scope (scope)}
			<ToggleGroup.Item value={scope} class={cn(toggleItem, 'h-9 px-3 text-[13px]')}>
				{t(scopeLabels[scope])}
			</ToggleGroup.Item>
		{/each}
	</ToggleGroup.Root>

	<span class="hidden h-0 basis-full max-md:order-2 max-md:block" aria-hidden="true"></span>

	<Select.Root type="single" value={store.query.sort} onValueChange={setSort}>
		<Select.Trigger
			class={cn(inputSize, 'w-auto max-md:order-3 md:min-w-44')}
			aria-label={t('links.toolbar.sort')}
		>
			{t(sortLabels[store.query.sort])}
		</Select.Trigger>
		<Select.Content>
			{#each SORTS as sort (sort)}
				<Select.Item value={sort} label={t(sortLabels[sort])} />
			{/each}
		</Select.Content>
	</Select.Root>

	<ToggleGroup.Root
		type="single"
		variant="outline"
		bind:value={() => viewMode, setView}
		aria-label={t('links.toolbar.view')}
		class="max-md:order-1"
	>
		<ToggleGroup.Item
			value="list"
			class={cn(toggleItem, 'size-9 px-0')}
			aria-label={t('links.view.list')}
		>
			<List class="size-4" strokeWidth={1.5} aria-hidden="true" />
		</ToggleGroup.Item>
		<ToggleGroup.Item
			value="tiles"
			class={cn(toggleItem, 'size-9 px-0')}
			aria-label={t('links.view.tiles')}
		>
			<LayoutGrid class="size-4" strokeWidth={1.5} aria-hidden="true" />
		</ToggleGroup.Item>
	</ToggleGroup.Root>

	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<Button
					{...props}
					variant="outline"
					class={cn(buttonSize.default, 'max-md:order-3 max-md:size-9 max-md:px-0')}
				>
					<SlidersHorizontal class="size-4" strokeWidth={1.5} aria-hidden="true" />
					<span class="max-md:sr-only">{t('links.toolbar.display')}</span>
				</Button>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="start" class="min-w-52">
			{#each fieldItems as item (item.key)}
				<DropdownMenu.CheckboxItem
					checked={prefs?.[item.key] ?? true}
					onCheckedChange={(checked: boolean) => preferences.update({ [item.key]: checked })}
				>
					{t(item.label)}
				</DropdownMenu.CheckboxItem>
			{/each}
			<DropdownMenu.Separator />
			<DropdownMenu.Item onSelect={() => shell.openShortcuts()}>
				<Keyboard class="size-4" strokeWidth={1.5} aria-hidden="true" />
				{t('links.toolbar.shortcuts')}
			</DropdownMenu.Item>
		</DropdownMenu.Content>
	</DropdownMenu.Root>

	<div class="flex items-center gap-2 max-md:order-3">
		<Switch
			id="links-show-archived"
			checked={store.query.archived !== 'exclude'}
			onCheckedChange={setArchived}
		/>
		<Label for="links-show-archived" class="text-[13px] leading-5 font-normal">
			{t('links.archivedToggle')}
		</Label>
	</div>
</div>
