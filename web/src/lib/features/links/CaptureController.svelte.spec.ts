// Quick capture by drag and drop and paste (plan 8.7.3, 5.9.3).
import { render, screen } from '@testing-library/svelte';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import type { LinkItem } from '$lib/api/types.js';
import { errorMessage, fieldMessage, t } from '$lib/i18n/index.js';
import { INTERNAL_DRAG_TYPE, MAX_CAPTURE_ITEMS } from '$lib/links/capture.js';
import { linkStats } from '$lib/state/link-stats.svelte.js';
import { setTestPage } from '$lib/test/app-state.svelte.js';
import { makeLink } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, type MockRequest } from '$lib/test/mock-api.js';
import CaptureController from './CaptureController.svelte';
import { makeDetail, makeStore } from './test-helpers.js';

vi.mock('svelte-sonner', () => ({
	toast: Object.assign(vi.fn(), {
		success: vi.fn(),
		error: vi.fn(),
		info: vi.fn(),
		warning: vi.fn()
	})
}));

type ToastOptions = { action?: { label: string; onClick: () => void } };

function lastOptions(fn: unknown): ToastOptions {
	const calls = vi.mocked(fn as (...args: unknown[]) => unknown).mock.calls;
	return (calls.at(-1)?.[1] ?? {}) as ToastOptions;
}

function transfer(data: Record<string, string>) {
	return {
		types: Object.keys(data),
		getData: (type: string) => data[type] ?? '',
		dropEffect: 'none'
	};
}

function drag(type: string, data: Record<string, string>, target: EventTarget = window): Event {
	const event = new Event(type, { bubbles: true, cancelable: true });
	Object.defineProperty(event, 'dataTransfer', { value: transfer(data) });
	target.dispatchEvent(event);
	return event;
}

function paste(data: Record<string, string>, target: EventTarget = document.body): Event {
	const event = new Event('paste', { bubbles: true, cancelable: true });
	Object.defineProperty(event, 'clipboardData', { value: transfer(data) });
	target.dispatchEvent(event);
	return event;
}

function captured(link: LinkItem, created: boolean) {
	return json({ link: makeDetail(link), created }, created ? 201 : 200);
}

function setup(items: LinkItem[] = [], status: 'ready' | 'idle' = 'ready') {
	const { store } = makeStore();
	store.items = items;
	store.total = items.length;
	store.status = status;
	render(CaptureController, { store });
	return store;
}

beforeEach(() => {
	for (const fn of [toast.success, toast.error, toast.info, toast.warning]) {
		vi.mocked(fn).mockClear();
	}
	vi.mocked(goto).mockClear();
	vi.spyOn(linkStats, 'refresh').mockResolvedValue();
});

afterEach(() => {
	vi.unstubAllGlobals();
	document.body.innerHTML = '';
});

describe('drop overlay', () => {
	test('appears for a link drag and disappears when it leaves', async () => {
		setup();
		const enter = drag('dragenter', { 'text/uri-list': 'https://a.io' });
		expect(enter.defaultPrevented).toBe(true);
		const overlay = await screen.findByTestId('capture-overlay');
		expect(overlay).toHaveAttribute('aria-hidden', 'true');
		expect(overlay).toHaveTextContent(t('capture.overlay.title'));
		drag('dragenter', { 'text/uri-list': 'https://a.io' });
		const over = drag('dragover', { 'text/uri-list': 'https://a.io' });
		expect(over.defaultPrevented).toBe(true);
		drag('dragleave', {});
		expect(screen.queryByTestId('capture-overlay')).not.toBeNull();
		drag('dragleave', {});
		await vi.waitFor(() => expect(screen.queryByTestId('capture-overlay')).toBeNull());
		drag('dragleave', {});
	});

	test('ignores files, own links and drags while a dialog is open', async () => {
		setup();
		expect(drag('dragenter', { Files: '' }).defaultPrevented).toBe(false);
		expect(
			drag('dragover', { [INTERNAL_DRAG_TYPE]: '1', 'text/uri-list': 'x' }).defaultPrevented
		).toBe(false);
		const dialog = document.createElement('div');
		dialog.setAttribute('role', 'dialog');
		document.body.append(dialog);
		expect(drag('dragenter', { 'text/plain': 'https://a.io' }).defaultPrevented).toBe(false);
		expect(screen.queryByTestId('capture-overlay')).toBeNull();
		dialog.remove();
		drag('dragenter', { 'text/plain': 'https://a.io' });
		const overlay = await screen.findByTestId('capture-overlay');
		// The panel keeps its content clear of the chamfered corners.
		const panel = overlay.querySelector('.kx-panel')!;
		expect(panel).toHaveClass('kx-panel-lg', 'p-8', 'max-w-[480px]');
		window.dispatchEvent(new Event('dragend'));
		await vi.waitFor(() => expect(screen.queryByTestId('capture-overlay')).toBeNull());
	});
});

describe('saving', () => {
	test('a dropped link is saved, highlighted and can be edited', async () => {
		const link = makeLink({ id: 5, fetchStatus: 'pending' });
		const api = mockApi({ 'POST /api/links/quick': captured(link, true) });
		const store = setup();
		const watch = vi.spyOn(store, 'watchPending').mockImplementation(() => {});
		drag('dragenter', { 'text/uri-list': 'https://a.io' });
		const drop = drag('drop', {
			'text/uri-list': 'https://a.io',
			'text/html': '<a href="https://a.io">Docs</a>'
		});
		expect(drop.defaultPrevented).toBe(true);
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(t('capture.toast.saved'), expect.anything())
		);
		expect(api.calls('POST /api/links/quick')[0].body).toEqual({
			text: 'https://a.io',
			title: 'Docs'
		});
		expect(store.find(5)).toBeDefined();
		expect(store.highlightId).toBe(5);
		expect(watch).toHaveBeenCalledWith(5);
		expect(linkStats.refresh).toHaveBeenCalled();
		const action = lastOptions(toast.success).action!;
		expect(action.label).toBe(t('capture.toast.editAction'));
		action.onClick();
		expect(store.editRequest?.id).toBe(5);
		expect(screen.queryByTestId('capture-overlay')).toBeNull();
	});

	test('an existing link is highlighted with a show action', async () => {
		const link = makeLink({ id: 6 });
		mockApi({ 'POST /api/links/quick': captured(link, false) });
		const store = setup([link]);
		paste({ 'text/plain': 'https://a.io' });
		await vi.waitFor(() =>
			expect(toast.info).toHaveBeenCalledWith(t('capture.toast.exists'), expect.anything())
		);
		expect(store.highlightId).toBe(6);
		store.clearHighlight(6);
		const action = lastOptions(toast.info).action!;
		expect(action.label).toBe(t('capture.toast.showAction'));
		action.onClick();
		expect(store.highlightId).toBe(6);
	});

	test('an existing link outside the list opens its details', async () => {
		const link = makeLink({ id: 6 });
		mockApi({ 'POST /api/links/quick': captured(link, false) });
		const store = setup([]);
		paste({ 'text/plain': 'https://a.io' });
		await vi.waitFor(() => expect(toast.info).toHaveBeenCalled());
		lastOptions(toast.info).action!.onClick();
		expect(store.detailsId).toBe(6);
	});

	test('a link hidden by the filters offers to clear them', async () => {
		const link = makeLink({ id: 7, tagIds: [] });
		mockApi({
			'POST /api/links/quick': captured(link, true),
			'GET /api/links': json({ items: [], total: 0, counts: { pinned: 0, archived: 0 } })
		});
		const store = setup();
		store.query = { ...store.query, tags: [3] };
		paste({ 'text/plain': 'https://a.io' });
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(t('capture.toast.hidden'), expect.anything())
		);
		expect(store.find(7)).toBeUndefined();
		const action = lastOptions(toast.success).action!;
		expect(action.label).toBe(t('links.filters.clear'));
		action.onClick();
		expect(store.query.tags).toEqual([]);
		expect(store.highlightId).toBe(7);
	});

	test('off the links page the toast opens the links page', async () => {
		setTestPage('/tags');
		const link = makeLink({ id: 8 });
		mockApi({ 'POST /api/links/quick': captured(link, true) });
		const store = setup([], 'idle');
		paste({ 'text/plain': 'https://a.io' });
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(t('capture.toast.saved'), expect.anything())
		);
		expect(store.highlightId).toBeNull();
		const action = lastOptions(toast.success).action!;
		expect(action.label).toBe(t('capture.toast.openAction'));
		action.onClick();
		expect(goto).toHaveBeenCalledWith('/');
		expect(store.highlightId).toBe(8);
	});

	test('several links are saved one after the other with one summary', async () => {
		let n = 0;
		const api = mockApi({
			'POST /api/links/quick': (req: MockRequest) => {
				n += 1;
				const body = req.body as { text: string };
				if (body.text.endsWith('/bad')) return apiError(422, 'validation_failed');
				return captured(makeLink({ id: 100 + n }), n !== 2);
			}
		});
		setup();
		const lines = ['https://a.io/1', 'https://a.io/2', 'https://a.io/bad', 'https://a.io/4'];
		drag('drop', { 'text/uri-list': lines.join('\n') });
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(
				t('capture.toast.multiple', { count: 2, existing: 1 }),
				expect.anything()
			)
		);
		expect(
			api.calls('POST /api/links/quick').map((req) => (req.body as { text: string }).text)
		).toEqual(lines);
		expect(toast.error).toHaveBeenCalledTimes(1);
	});

	test('more than twenty links: the rest is skipped with a warning', async () => {
		const api = mockApi({ 'POST /api/links/quick': captured(makeLink(), true) });
		setTestPage('/settings');
		setup();
		const lines = Array.from({ length: 22 }, (_, index) => `https://a.io/${index}`);
		drag('drop', { 'text/uri-list': lines.join('\n') });
		await vi.waitFor(() =>
			expect(api.calls('POST /api/links/quick')).toHaveLength(MAX_CAPTURE_ITEMS)
		);
		expect(toast.warning).toHaveBeenCalledWith(
			t('capture.toast.tooMany', { max: MAX_CAPTURE_ITEMS })
		);
		await vi.waitFor(() =>
			expect(lastOptions(toast.success).action?.label).toBe(t('capture.toast.openAction'))
		);
	});

	test('text without a URL is refused without a request', async () => {
		const api = mockApi({});
		setup();
		const event = paste({ 'text/plain': 'just some words' });
		expect(event.defaultPrevented).toBe(true);
		await vi.waitFor(() =>
			expect(toast.error).toHaveBeenCalledWith(errorMessage({ code: 'no_url_found', details: {} }))
		);
		expect(api.requests).toHaveLength(0);
	});

	test.each([['not_allowed'], ['credentials_in_url'], ['invalid_format'], ['too_long']] as const)(
		'a %s URL names the URL problem instead of "fix the marked fields"',
		async (code) => {
			mockApi({ 'POST /api/links/quick': apiError(400, 'validation_failed', { url: code }) });
			setup();
			paste({ 'text/plain': 'https://a.io/x' });
			await vi.waitFor(() => expect(toast.error).toHaveBeenCalled());
			const message = vi.mocked(toast.error).mock.calls[0][0];
			expect(message).toBe(`${t('capture.toast.invalidUrl')} ${fieldMessage('url', code)}`);
			expect(message).not.toContain(errorMessage({ code: 'validation_failed', details: {} }));
		}
	);

	test('a failed single capture shows only the error', async () => {
		mockApi({ 'POST /api/links/quick': apiError(403, 'link_limit_reached') });
		setup();
		paste({ 'text/plain': 'https://a.io' });
		await vi.waitFor(() => expect(toast.error).toHaveBeenCalled());
		expect(toast.success).not.toHaveBeenCalled();
	});
});

describe('paste guards', () => {
	test('a paste into a text field, with a dialog open or without text stays a normal paste', () => {
		const api = mockApi({});
		setup();
		const input = document.createElement('input');
		document.body.append(input);
		expect(paste({ 'text/plain': 'https://a.io' }, input).defaultPrevented).toBe(false);
		const dialog = document.createElement('div');
		dialog.setAttribute('role', 'dialog');
		document.body.append(dialog);
		expect(paste({ 'text/plain': 'https://a.io' }).defaultPrevented).toBe(false);
		dialog.remove();
		expect(paste({ 'text/plain': '  ' }).defaultPrevented).toBe(false);
		expect(paste({ 'image/png': 'x' }).defaultPrevented).toBe(false);
		const bare = new Event('paste', { bubbles: true, cancelable: true });
		document.body.dispatchEvent(bare);
		expect(bare.defaultPrevented).toBe(false);
		expect(api.requests).toHaveLength(0);
	});

	test('a drop without a prior accepted drag and without data is ignored', () => {
		setup();
		const event = new Event('drop', { bubbles: true, cancelable: true });
		window.dispatchEvent(event);
		expect(event.defaultPrevented).toBe(false);
		expect(drag('drop', { Files: '' }).defaultPrevented).toBe(false);
		expect(drag('drop', { 'text/plain': '   ' }).defaultPrevented).toBe(false);
	});
});
