<script lang="ts">
	import ClipboardPaste from '@lucide/svelte/icons/clipboard-paste';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import * as linksApi from '$lib/api/links.js';
	import type { LinkDetail } from '$lib/api/types.js';
	import Panel from '$lib/components/brand/Panel.svelte';
	import { errorMessage, fieldMessage, t } from '$lib/i18n/index.js';
	import {
		containsUrl,
		extractCaptureItems,
		isCaptureDrag,
		isEditableTarget,
		limitCaptureItems,
		MAX_CAPTURE_ITEMS,
		type CaptureItem
	} from '$lib/links/capture.js';
	import { linkStats } from '$lib/state/link-stats.svelte.js';
	import { links as linksStore, toApiError, type LinksStore } from '$lib/state/links.svelte.js';
	import { isDialogOpen } from '$lib/utils/keyboard.js';
	import { toastError } from '$lib/utils/toast-error.js';

	/**
	 * Quick capture (plan 8.7.3, 5.9.3): a link dragged onto the window or pasted outside a text
	 * field is saved with POST /api/links/quick, one request after the other. Nothing is captured
	 * while a dialog or sheet is open.
	 */
	type Props = { store?: LinksStore };
	let { store = linksStore }: Props = $props();

	let dragging = $state(false);
	let depth = 0;
	/** Captures run one after the other, also when the next drop comes early. */
	let queue: Promise<void> = Promise.resolve();

	function accepts(event: DragEvent): boolean {
		return isCaptureDrag(event.dataTransfer?.types) && !isDialogOpen();
	}

	function ondragenter(event: DragEvent): void {
		if (!accepts(event)) return;
		event.preventDefault();
		depth += 1;
		dragging = true;
	}

	function ondragover(event: DragEvent): void {
		if (!accepts(event)) return;
		event.preventDefault();
		if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy';
		dragging = true;
	}

	function ondragleave(): void {
		if (!dragging) return;
		depth = Math.max(0, depth - 1);
		if (depth === 0) dragging = false;
	}

	function stopDragging(): void {
		depth = 0;
		dragging = false;
	}

	function ondrop(event: DragEvent): void {
		const wasDragging = dragging;
		stopDragging();
		if (!event.dataTransfer || (!wasDragging && !accepts(event))) return;
		const items = extractCaptureItems(event.dataTransfer);
		if (items.length === 0) return;
		event.preventDefault();
		enqueue(items);
	}

	function onpaste(event: ClipboardEvent): void {
		if (event.defaultPrevented || !event.clipboardData) return;
		if (isEditableTarget(event.target) || isDialogOpen()) return;
		const items = extractCaptureItems(event.clipboardData);
		if (items.length === 0) return;
		event.preventDefault();
		enqueue(items);
	}

	function enqueue(items: CaptureItem[]): void {
		queue = queue.then(() => capture(items));
	}

	/**
	 * The toast of a failed capture. validation_failed concerns the URL found in the text
	 * (plan 5.9.3): there is no field to mark, so the toast names the URL problem itself.
	 */
	function captureError(err: unknown): string {
		const apiError = toApiError(err);
		const code = apiError.is('validation_failed') ? apiError.field('url') : undefined;
		if (code) return `${t('capture.toast.invalidUrl')} ${fieldMessage('url', code)}`;
		return errorMessage(apiError);
	}

	function onLinksPage(): boolean {
		return page.url.pathname === '/';
	}

	/** Goes to the links page and highlights the link there. */
	function showOnLinksPage(id: number): void {
		store.highlight(id);
		if (!onLinksPage()) void goto('/');
	}

	async function capture(raw: CaptureItem[]): Promise<void> {
		const { items: limited, skipped } = limitCaptureItems(raw);
		const items = limited.filter((item) => containsUrl(item.text));
		if (items.length === 0) {
			toast.error(errorMessage({ code: 'no_url_found', details: {} }));
			return;
		}
		if (skipped > 0) toast.warning(t('capture.toast.tooMany', { max: MAX_CAPTURE_ITEMS }));

		const here = onLinksPage();
		let created = 0;
		let existing = 0;
		let last: { link: LinkDetail; created: boolean; visible: boolean } | null = null;
		for (const item of items) {
			try {
				const result = await linksApi.quickCapture(item.text, item.title);
				let visible: boolean;
				if (result.created) {
					created += 1;
					visible = store.upsert(result.link, { highlight: here });
					if (result.link.fetchStatus === 'pending') store.watchPending(result.link.id);
				} else {
					existing += 1;
					visible = store.find(result.link.id) !== undefined;
					if (visible && here) store.highlight(result.link.id);
				}
				last = { ...result, visible };
			} catch (err) {
				toastError(err, captureError(err));
			}
		}
		if (created > 0) void linkStats.refresh();
		if (!last) return;

		const id = last.link.id;
		const openAction = { label: t('capture.toast.openAction'), onClick: () => showOnLinksPage(id) };
		if (items.length > 1) {
			toast.success(t('capture.toast.multiple', { count: created, existing }), {
				action: here ? undefined : openAction
			});
			return;
		}
		if (!here) {
			toast.success(last.created ? t('capture.toast.saved') : t('capture.toast.exists'), {
				action: openAction
			});
		} else if (last.created && !last.visible) {
			toast.success(t('capture.toast.hidden'), {
				action: {
					label: t('links.filters.clear'),
					onClick: () => {
						store.highlight(id);
						store.clearFilters();
					}
				}
			});
		} else if (last.created) {
			const link = last.link;
			toast.success(t('capture.toast.saved'), {
				action: { label: t('capture.toast.editAction'), onClick: () => store.requestEdit(link) }
			});
		} else {
			const visible = last.visible;
			toast.info(t('capture.toast.exists'), {
				action: {
					label: t('capture.toast.showAction'),
					onClick: () => (visible ? store.highlight(id) : store.openDetails(id))
				}
			});
		}
	}
</script>

<svelte:window {ondragenter} {ondragover} {ondragleave} {ondrop} ondragend={stopDragging} />
<svelte:document {onpaste} />

{#if dragging}
	<div
		class="kxm-drop-scrim pointer-events-none fixed inset-0 z-50 flex animate-in items-center justify-center p-8 duration-[120ms] fade-in-0 motion-reduce:animate-none"
		aria-hidden="true"
		data-testid="capture-overlay"
	>
		<!-- p-8 keeps the text clear of the chamfered corners (plan 8.7.3: space-8). -->
		<Panel
			size="lg"
			class="flex w-full max-w-[480px] flex-col gap-3 p-8 [--kx-edge:var(--primary)]"
		>
			<ClipboardPaste class="size-6 text-primary" strokeWidth={1.5} aria-hidden="true" />
			<p class="heading-3">{t('capture.overlay.title')}</p>
			<p class="body-sm text-muted-foreground">{t('capture.overlay.text')}</p>
		</Panel>
	</div>
{/if}
