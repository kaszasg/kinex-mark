<script lang="ts" module>
	/** How long a captured or located row stays highlighted. */
	export const HIGHLIGHT_MS = 2000;
</script>

<script lang="ts">
	import Pin from '@lucide/svelte/icons/pin';
	import { MediaQuery } from 'svelte/reactivity';
	import type { LinkItem, Me } from '$lib/api/types.js';
	import Favicon from '$lib/components/app/Favicon.svelte';
	import RelativeTime from '$lib/components/app/RelativeTime.svelte';
	import { buttonSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
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
	import LinkTagChips from './LinkTagChips.svelte';

	/**
	 * One list row (plan 8.7.3): icon, label as the only link, secondary lines, markers, tags,
	 * usage meta and the actions. Opening counts exactly one use: the anchor's click (left click
	 * and Enter on the focused anchor) and a middle-button auxclick; there is no Enter handler.
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
	// Tag chips grow with the width, so the label keeps its room next to the meta (plan 8.7.3:
	// at most 3 chips from md). At 1024 px the sidebar and the meta slots leave about 200 px.
	const xl = new MediaQuery('min-width: 1280px', false);
	const xxl = new MediaQuery('min-width: 1536px', false);
	const tagLimit = $derived(xxl.current ? 3 : xl.current ? 2 : 1);
	let element = $state<HTMLLIElement | null>(null);

	const view = $derived(linkPresentation(link, fields));
	const secondary = $derived(
		[
			view.secondary.title !== undefined
				? { kind: 'title', text: view.secondary.title, cls: 'body-sm truncate' }
				: null,
			view.secondary.description !== undefined
				? { kind: 'description', text: view.secondary.description, cls: 'body-sm line-clamp-1' }
				: null,
			view.secondary.url !== undefined
				? { kind: 'url', text: view.secondary.url, cls: 'code truncate' }
				: null
		].filter((line) => line !== null)
	);
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

	/** Hover-revealed controls on fine pointers; always visible on touch screens. */
	const reveal =
		'pointer-fine:opacity-0 pointer-fine:group-hover:opacity-100 pointer-fine:group-focus-within:opacity-100 data-[state=open]:opacity-100';
</script>

<li
	bind:this={element}
	tabindex="-1"
	data-link-id={link.id}
	class={cn(
		'kxm-list-row group grid min-h-11 grid-cols-[20px_minmax(0,1fr)_auto] items-start gap-x-3 md:grid-cols-[20px_minmax(12rem,1fr)_minmax(0,auto)] border-b px-4 py-2 outline-offset-[-2px] transition-colors duration-[120ms] ease-out last:border-b-0 hover:bg-accent',
		highlighted && (reducedMotion.current ? 'kxm-flash-static' : 'kxm-flash')
	)}
>
	<span class="flex h-6 items-center justify-center">
		{#if fields.showIcon}
			<Favicon src={link.iconUrl} size={20} />
		{/if}
	</span>
	<div class="flex min-w-0 flex-col">
		<!-- Markers follow the label, so every label starts at the same x (plan 8.7.3). -->
		<div class="flex h-6 min-w-0 items-center gap-1.5">
			<a
				href={link.url}
				target="_blank"
				rel="noopener noreferrer"
				title={link.url}
				data-link-anchor
				class={cn(
					'body min-w-0 truncate font-medium hover:underline hover:underline-offset-3',
					link.archivedAt && 'text-muted-foreground'
				)}
				onclick={visit}
				{onauxclick}
				{ondragstart}
			>
				{view.label}
			</a>
			<LinkMarkers {link} />
		</div>
		{#each secondary as line, index (line.kind)}
			<p
				class={cn(line.cls, 'text-muted-foreground', index >= 2 && 'max-md:hidden')}
				data-secondary={line.kind}
			>
				{line.text}
			</p>
		{/each}
	</div>
	<!-- From md the label keeps at least 12rem: the meta column gives way by shrinking the chips. -->
	<div class="flex h-6 min-w-0 items-center gap-3" data-row-meta>
		<LinkTagChips tagIds={link.tagIds} max={tagLimit} class="hidden md:flex" />
		<!--
			Fixed-width meta slots keep the columns aligned from row to row: the use count up to
			"99 999", the longest relative time of hu/en/de ("in dieser Minute", 16 characters of
			the mono face), and the expiry meter's slot also on rows without a meter (pinned).
		-->
		<span class="hidden shrink-0 items-center gap-3 lg:flex">
			<Hint text={t('links.row.uses', { count: link.useCount })}>
				{#snippet children(props)}
					<span {...props} class="data w-[6ch] shrink-0 text-right text-muted-foreground" data-uses>
						<span aria-hidden="true">{fmt.number(link.useCount)}</span>
						<span class="sr-only">{t('links.row.uses', { count: link.useCount })}</span>
					</span>
				{/snippet}
			</Hint>
			<span
				class="w-[16ch] shrink-0 text-right font-mono text-[13px] leading-5 whitespace-nowrap text-muted-foreground"
				data-last-used
			>
				{#if link.lastUsedAt}
					<RelativeTime value={link.lastUsedAt} />
				{:else}
					<span title={t('links.row.neverUsed')}>{fmt.missing}</span>
				{/if}
			</span>
		</span>
		{#if effectiveArchiveAfterDays > 0}
			<span class="flex w-14 shrink-0 justify-end" data-expiry-slot>
				<ExpiryMeter {link} {effectiveArchiveAfterDays} />
			</span>
		{/if}
		<span class="flex shrink-0 items-center gap-0.5">
			<Button
				variant="ghost"
				size="icon"
				class={cn(buttonSize.iconSm, 'hidden md:inline-flex', reveal)}
				aria-pressed={link.pinned}
				aria-label={link.pinned ? t('links.actions.unpin') : t('links.actions.pin')}
				onclick={() => onaction('pin', link)}
			>
				<Pin
					class={cn('size-4', link.pinned && 'text-primary')}
					strokeWidth={1.5}
					aria-hidden="true"
				/>
			</Button>
			<LinkMenu {link} {me} label={view.label} {onaction} class={reveal} />
		</span>
	</div>
</li>
