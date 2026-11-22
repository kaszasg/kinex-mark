<script lang="ts">
	import ScrollText from '@lucide/svelte/icons/scroll-text';
	import { onMount, untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { adminApi } from '$lib/api/index.js';
	import type { AuditEntry } from '$lib/api/types.js';
	import EmptyState from '$lib/components/app/EmptyState.svelte';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import { buttonSize, inputSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { t } from '$lib/i18n/index.js';
	import { publicConfig } from '$lib/state/public-config.svelte.js';
	import ActorPicker, { type Actor } from './ActorPicker.svelte';
	import AdminSelect from './AdminSelect.svelte';
	import AuditTable from './AuditTable.svelte';
	import Pager from './Pager.svelte';
	import { eventLabel } from './audit-format.js';
	import {
		AUDIT_EVENTS,
		AUDIT_GROUPS,
		AUDIT_MAX_LIMIT,
		auditFilterSearch,
		DEFAULT_AUDIT_FILTER,
		hasAuditFilters,
		isDay,
		loadAuditPage,
		parseAuditFilter,
		type AuditFilter
	} from './audit-query.js';

	/**
	 * The /admin/audit screen of plan 8.7.9: retention in the subtitle, filters kept in the URL
	 * (event or event group, result, actor, period), the audit table and paging.
	 */

	const ALL = 'all';

	const filter = $derived(parseAuditFilter(page.url.searchParams));

	let entries = $state<AuditEntry[] | null>(null);
	let total = $state(0);
	let error = $state<unknown>(null);
	let loading = $state(false);
	let retentionDays = $state<number | null>(null);
	let actors = $state<Actor[] | null>(null);
	let actorsFailed = $state(false);
	let requestSeq = 0;

	const days = $derived(retentionDays ?? publicConfig.config?.privacy.auditRetentionDays ?? null);

	async function load(next: AuditFilter): Promise<void> {
		const seq = ++requestSeq;
		loading = true;
		error = null;
		try {
			const result = await loadAuditPage(adminApi.listAudit, next);
			if (seq !== requestSeq) return;
			entries = result.items;
			total = result.total;
		} catch (err) {
			if (seq === requestSeq) error = err;
		} finally {
			if (seq === requestSeq) loading = false;
		}
	}

	const reload = (): Promise<void> => load(filter);

	$effect(() => {
		const next = filter;
		untrack(() => void load(next));
	});

	onMount(() => {
		void publicConfig.ensure();
		adminApi
			.getSettings()
			.then((response) => (retentionDays = response.settings.auditRetentionDays))
			.catch(() => undefined);
		adminApi
			.listUsers({ status: 'all', limit: AUDIT_MAX_LIMIT })
			.then((result) => (actors = result.items))
			.catch(() => (actorsFailed = true));
	});

	function navigate(next: AuditFilter): void {
		void goto(`${page.url.pathname}${auditFilterSearch(next)}`, { reset: false });
	}

	function setFilter(patch: Partial<AuditFilter>): void {
		navigate({ ...filter, ...patch, page: patch.page ?? 1 });
	}

	function setDay(key: 'from' | 'to', value: string): void {
		if (value === '' || isDay(value)) setFilter({ [key]: value });
	}

	const eventGroups = $derived([
		{ options: [{ value: ALL, label: t('audit.filters.all') }] },
		{
			heading: t('audit.filters.groupsHeading'),
			options: AUDIT_GROUPS.map((group) => ({
				value: group.value,
				label: t(`audit.filters.groups.${group.group}`)
			}))
		},
		{
			heading: t('audit.filters.eventsHeading'),
			options: AUDIT_EVENTS.map((event) => {
				const label = eventLabel(event);
				return { value: event, label: label.text, mono: label.raw };
			})
		}
	]);
	const resultGroups = $derived([
		{
			options: [
				{ value: ALL, label: t('audit.filters.all') },
				{ value: 'success', label: t('audit.result.success') },
				{ value: 'failure', label: t('audit.result.failure') }
			]
		}
	]);
</script>

<PageHeader
	title={t('audit.pageTitle')}
	subtitle={days !== null ? t('audit.subtitle', { days }) : undefined}
/>

<div
	role="search"
	class="grid grid-cols-2 items-end gap-3 sm:flex sm:flex-wrap"
	aria-label={t('audit.filters.label')}
>
	<div class="col-span-2 flex flex-col gap-1.5 sm:w-64">
		<label for="audit-event" class="text-[13px] leading-5 font-medium">
			{t('audit.filters.event')}
		</label>
		<AdminSelect
			id="audit-event"
			value={filter.event || ALL}
			groups={eventGroups}
			onchange={(value) => setFilter({ event: value === ALL ? '' : value })}
		/>
	</div>
	<div class="flex flex-col gap-1.5 sm:w-36">
		<label for="audit-result" class="text-[13px] leading-5 font-medium">
			{t('audit.filters.result')}
		</label>
		<AdminSelect
			id="audit-result"
			value={filter.result || ALL}
			groups={resultGroups}
			onchange={(value) =>
				setFilter({ result: value === ALL ? '' : (value as AuditFilter['result']) })}
		/>
	</div>
	<div class="flex flex-col gap-1.5 sm:w-48">
		<label for="audit-actor" class="text-[13px] leading-5 font-medium">
			{t('audit.filters.actor')}
		</label>
		<ActorPicker
			id="audit-actor"
			value={filter.actorId}
			{actors}
			failed={actorsFailed}
			onchange={(actorId) => setFilter({ actorId })}
		/>
	</div>
	<div class="flex flex-col gap-1.5 sm:w-40">
		<label for="audit-from" class="text-[13px] leading-5 font-medium">
			{t('audit.filters.from')}
		</label>
		<Input
			id="audit-from"
			type="date"
			class="{inputSize} font-mono"
			value={filter.from}
			max={filter.to || undefined}
			onchange={(event: Event) => setDay('from', (event.currentTarget as HTMLInputElement).value)}
		/>
	</div>
	<div class="flex flex-col gap-1.5 sm:w-40">
		<label for="audit-to" class="text-[13px] leading-5 font-medium">
			{t('audit.filters.to')}
		</label>
		<Input
			id="audit-to"
			type="date"
			class="{inputSize} font-mono"
			value={filter.to}
			min={filter.from || undefined}
			onchange={(event: Event) => setDay('to', (event.currentTarget as HTMLInputElement).value)}
		/>
	</div>
	{#if hasAuditFilters(filter)}
		<Button
			variant="ghost"
			class="{buttonSize.default} col-span-2 justify-self-start"
			onclick={() => navigate(DEFAULT_AUDIT_FILTER)}
		>
			{t('audit.filters.clear')}
		</Button>
	{/if}
</div>

{#if error}
	<ErrorAlert {error} onRetry={() => void reload()} retryDisabled={loading} />
{:else if entries === null}
	<LoadingRows rows={8} />
{:else if entries.length === 0}
	<EmptyState icon={ScrollText} title={t('audit.empty.title')} text={t('audit.empty.text')}>
		{#snippet action()}
			{#if hasAuditFilters(filter) || filter.page > 1}
				<Button
					variant="outline"
					class={buttonSize.sm}
					onclick={() => navigate(DEFAULT_AUDIT_FILTER)}
				>
					{t('audit.filters.clear')}
				</Button>
			{/if}
		{/snippet}
	</EmptyState>
{:else}
	<div class="flex flex-col gap-3" aria-busy={loading}>
		<AuditTable {entries} />
		<Pager page={filter.page} {total} onchange={(next) => setFilter({ page: next })} />
	</div>
{/if}
