<script lang="ts">
	import { MediaQuery } from 'svelte/reactivity';
	import type { LinkItem, Me } from '$lib/api/types.js';
	import Favicon from '$lib/components/app/Favicon.svelte';
	import { cardSurface } from '$lib/components/app/styles.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { INTERNAL_DRAG_TYPE } from '$lib/links/capture.js';
	import { linkPresentation, type DisplayFields } from '$lib/links/presentation.js';
	import { links as linksStore, type LinksStore } from '$lib/state/links.svelte.js';
	import { cn } from '$lib/utils.js';
	import type { LinkActionHandler } from './actions.js';
	import ExpiryMeter from './ExpiryMeter.svelte';
	import Hint from './Hint.svelte';
	import LinkMarkers from './LinkMarkers.svelte';
	import LinkMenu from './LinkMenu.svelte';
	import { HIGHLIGHT_MS } from './LinkRow.svelte';
	import LinkTagChips from './LinkTagChips.svelte';

	/**
	 * One tile (plan 8.7.3): a flat card without a chamfer (it holds a menu). The label is the
	 * only link; the tile itself is not clickable. Opening counts one use like the row.
	 */
	type Props = {
		link: LinkItem;
		me: Me | null;
		fields: DisplayFields;
		effectiveArchiveAfterDays: number;
		onaction: LinkActionHandler;
		store?: LinksStore;
	};
	let {
		link,
		me,
		fields,
		effectiveArchiveAfterDays,
		onaction,
		store = linksStore
	}: Props = $props();

	const reducedMotion = new MediaQuery('prefers-reduced-motion: reduce', false);
	let element = $state<HTMLLIElement | null>(null);

	const view = $derived(linkPresentation(link, fields));
	const highlighted = $derived(store.highlightId === link.id);

	$effect(() => {
		if (!highlighted || !element) return;
		element.scrollIntoView({ block: 'nearest', behavior: 'auto' });
		const id = link.id;
		const timer = setTimeout(() => store.clearHighlight(id), HIGHLIGHT_MS);
		return () => clearTimeout(timer);
	});

	function visit(): void {
		store.visit(link.id);
	}

	function onauxclick(event: MouseEvent): void {
		if (event.button === 1) visit();
	}

	function ondragstart(event: DragEvent): void {
		event.dataTransfer?.setData(INTERNAL_DRAG_TYPE, String(link.id));
	}

	const reveal =
		'pointer-fine:opacity-0 pointer-fine:group-hover:opacity-100 pointer-fine:group-focus-within:opacity-100 data-[state=open]:opacity-100';
</script>

<li bind:this={element} tabindex="-1" data-link-id={link.id} class="group min-w-0 outline-offset-2">
	<Card.Root
		class={cn(
			cardSurface,
			'flex h-full min-h-[148px] flex-col gap-2 p-4 transition-colors duration-[120ms] ease-out group-hover:bg-accent/40',
			highlighted && (reducedMotion.current ? 'kxm-flash-static' : 'kxm-flash')
		)}
	>
		<div class="flex items-start justify-between gap-2">
			{#if fields.showIcon}
				<span class="flex size-10 shrink-0 items-center justify-center rounded-md border bg-muted">
					<Favicon src={link.iconUrl} size={20} class="size-6" />
				</span>
			{/if}
			<span class="ml-auto flex items-center gap-1.5">
				<LinkMarkers {link} />
				<LinkMenu {link} {me} label={view.label} {onaction} class={reveal} />
			</span>
		</div>
		<a
			href={link.url}
			target="_blank"
			rel="noopener noreferrer"
			title={link.url}
			data-link-anchor
			class={cn(
				'body line-clamp-2 font-medium break-words hover:underline hover:underline-offset-3',
				link.archivedAt && 'text-muted-foreground'
			)}
			onclick={visit}
			{onauxclick}
			{ondragstart}
		>
			{view.label}
		</a>
		{#if view.secondary.title !== undefined}
			<p class="body-sm truncate text-muted-foreground" data-secondary="title">
				{view.secondary.title}
			</p>
		{/if}
		{#if view.secondary.description !== undefined}
			<p class="body-sm line-clamp-2 text-muted-foreground" data-secondary="description">
				{view.secondary.description}
			</p>
		{/if}
		{#if view.secondary.url !== undefined}
			<p class="code truncate text-muted-foreground" data-secondary="url">
				{view.secondary.url}
			</p>
		{/if}
		<!--
			Footer: the tag chips on their own line (they wrap there, so long names stay readable),
			then a fixed-height meta row. Tiles in one grid row are equally high, so their meta rows
			line up at the bottom whatever the tags do.
		-->
		<div class="mt-auto flex flex-col gap-1.5 pt-1" data-tile-footer>
			<LinkTagChips tagIds={link.tagIds} max={2} class="flex-wrap" chipClass="max-w-full" />
			<span class="flex h-5 items-center justify-end gap-3" data-tile-meta>
				<Hint text={t('links.row.uses', { count: link.useCount })}>
					{#snippet children(props)}
						<span {...props} class="data text-muted-foreground" data-uses>
							<span aria-hidden="true">{fmt.number(link.useCount)}</span>
							<span class="sr-only">{t('links.row.uses', { count: link.useCount })}</span>
						</span>
					{/snippet}
				</Hint>
				<ExpiryMeter {link} {effectiveArchiveAfterDays} />
			</span>
		</div>
	</Card.Root>
</li>
