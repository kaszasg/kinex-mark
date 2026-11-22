<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import TagsIcon from '@lucide/svelte/icons/tags';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import type { Tag } from '$lib/api/types.js';
	import ConfirmDialog from '$lib/components/app/ConfirmDialog.svelte';
	import EmptyState from '$lib/components/app/EmptyState.svelte';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import { buttonSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import { tags, type TagsStore } from '$lib/state/tags.svelte.js';
	import TagDialog from './TagDialog.svelte';
	import TagsTable from './TagsTable.svelte';

	/**
	 * The tags screen of plan 8.7.5: heading with the count and "Add tag", then the loading,
	 * error, empty or table state of the tags store; create, edit and delete in dialogs.
	 */
	type Props = { store?: TagsStore };
	let { store = tags }: Props = $props();

	let dialogOpen = $state(false);
	let editing = $state<Tag | null>(null);
	let deleteOpen = $state(false);
	let deleting = $state<Tag | null>(null);

	const items = $derived(store.items);
	const ready = $derived(store.status === 'ready');

	onMount(() => {
		// The app layout starts the load; a direct visit before it finished shares that request.
		const me = session.me;
		if (me) void store.ensure(me.id);
		else if (store.status === 'idle') void store.load();
	});

	function openCreate(): void {
		editing = null;
		dialogOpen = true;
	}

	function openEdit(tag: Tag): void {
		editing = tag;
		dialogOpen = true;
	}

	function askDelete(tag: Tag): void {
		deleting = tag;
		deleteOpen = true;
	}

	async function confirmDelete(): Promise<void> {
		if (!deleting) return;
		await store.remove(deleting.id);
		toast.success(t('tags.delete.deleted'));
	}
</script>

<PageTitle title={t('tags.pageTitle')} />

<PageHeader
	title={t('tags.pageTitle')}
	subtitle={ready ? t('tags.summary', { count: items.length }) : undefined}
>
	{#snippet actions()}
		<Button class={buttonSize.default} onclick={openCreate}>
			<Plus class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('tags.add')}
		</Button>
	{/snippet}
</PageHeader>

{#if store.status === 'error'}
	<ErrorAlert error={store.error} onRetry={() => void store.load()} />
{:else if ready && items.length === 0}
	<EmptyState icon={TagsIcon} title={t('tags.empty.title')} text={t('tags.empty.text')} />
{:else if ready || items.length > 0}
	<TagsTable {items} onEdit={openEdit} onDelete={askDelete} />
{:else}
	<LoadingRows rows={6} />
{/if}

<TagDialog bind:open={dialogOpen} tag={editing} {store} />

<ConfirmDialog
	bind:open={deleteOpen}
	title={t('tags.delete.title')}
	text={deleting
		? t('tags.delete.text', { name: deleting.name, count: deleting.linkCount })
		: undefined}
	confirmLabel={t('tags.delete.confirm')}
	destructive
	onConfirm={confirmDelete}
/>
