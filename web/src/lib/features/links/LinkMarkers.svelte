<script lang="ts">
	import Archive from '@lucide/svelte/icons/archive';
	import Pin from '@lucide/svelte/icons/pin';
	import Users from '@lucide/svelte/icons/users';
	import type { LinkItem } from '$lib/api/types.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { t } from '$lib/i18n/index.js';
	import Hint from './Hint.svelte';

	/**
	 * State markers of a link (plan 8.7.3): pinned, shared with me, archived, metadata pending.
	 * Every marker carries a word for assistive technology; colour is never the only signal.
	 */
	type Props = {
		link: Pick<LinkItem, 'pinned' | 'isOwner' | 'owner' | 'archivedAt' | 'fetchStatus'>;
		/** The pin is shown by the caller elsewhere (the tile's top row). */
		hidePin?: boolean;
	};
	let { link, hidePin = false }: Props = $props();

	const ownerName = $derived(link.owner.displayName || link.owner.username);
</script>

{#if link.pinned && !hidePin}
	<span class="inline-flex shrink-0 items-center" data-marker="pinned">
		<Pin class="size-3.5 text-primary" strokeWidth={1.5} aria-hidden="true" />
		<span class="sr-only">{t('links.row.pinned')}</span>
	</span>
{/if}
{#if !link.isOwner}
	<Hint text={t('links.row.sharedBy', { owner: ownerName })}>
		{#snippet children(props)}
			<span {...props} class="inline-flex shrink-0 items-center" data-marker="shared">
				<Users class="size-3.5 text-muted-foreground" strokeWidth={1.5} aria-hidden="true" />
				<span class="sr-only">{t('links.row.sharedBy', { owner: ownerName })}</span>
			</span>
		{/snippet}
	</Hint>
{/if}
{#if link.archivedAt}
	<Badge
		variant="outline"
		class="h-5 shrink-0 gap-1 rounded-sm px-1.5 text-[11px] font-normal text-muted-foreground"
		data-marker="archived"
	>
		<Archive class="size-3" strokeWidth={1.5} aria-hidden="true" />
		{t('links.row.archived')}
	</Badge>
{/if}
{#if link.fetchStatus === 'pending'}
	<Hint text={t('links.row.pending')}>
		{#snippet children(props)}
			<span {...props} class="inline-flex shrink-0 items-center" data-marker="pending">
				<Spinner class="size-3 text-muted-foreground" aria-label={t('links.row.pending')} />
			</span>
		{/snippet}
	</Hint>
{/if}
