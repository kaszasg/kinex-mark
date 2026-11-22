// List rows, tiles, markers and the expiry meter (plan 8.7.3).
import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import type { LinkItem, Me } from '$lib/api/types.js';
import { fmt, t } from '$lib/i18n/index.js';
import { INTERNAL_DRAG_TYPE } from '$lib/links/capture.js';
import { ALL_FIELDS, type DisplayFields } from '$lib/links/presentation.js';
import { tags } from '$lib/state/tags.svelte.js';
import { makeLink, makeMe, makeTag } from '$lib/test/fixtures.js';
import { json, mockApi } from '$lib/test/mock-api.js';
import ExpiryMeter from './ExpiryMeter.svelte';
import LinkRow, { HIGHLIGHT_MS } from './LinkRow.svelte';
import LinkTile from './LinkTile.svelte';
import { getFloating, makeStore, queryFloating, user } from './test-helpers.js';

const DAY = 86_400_000;

type Options = {
	link?: Partial<LinkItem>;
	me?: Me | null;
	fields?: DisplayFields;
	days?: number;
};

function setup(component: typeof LinkRow | typeof LinkTile, options: Options = {}) {
	const { store } = makeStore();
	const link = makeLink({ id: 7, ...options.link });
	store.items = [link];
	store.total = 1;
	const onaction = vi.fn();
	const visit = vi.spyOn(store, 'visit').mockImplementation(() => {});
	const result = render(component, {
		link,
		me: options.me === undefined ? makeMe() : options.me,
		fields: options.fields ?? ALL_FIELDS,
		effectiveArchiveAfterDays: options.days ?? 90,
		onaction,
		store
	});
	const anchor = result.container.querySelector<HTMLAnchorElement>('[data-link-anchor]')!;
	return { ...result, store, link, onaction, visit, anchor };
}

beforeEach(() => {
	tags.reset();
});

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
	tags.reset();
});

describe.each([
	['LinkRow', LinkRow],
	['LinkTile', LinkTile]
] as const)('%s', (_name, component) => {
	test('shows the label as the only link with the secondary lines', () => {
		const { anchor, container } = setup(component, {
			link: {
				description: 'Éles műszerfal',
				title: 'Grafana',
				url: 'https://grafana.example.net/d/abc'
			}
		});
		expect(anchor).toHaveTextContent('Éles műszerfal');
		expect(anchor).toHaveAttribute('href', 'https://grafana.example.net/d/abc');
		expect(anchor).toHaveAttribute('target', '_blank');
		expect(anchor).toHaveAttribute('rel', 'noopener noreferrer');
		expect(anchor).toHaveAttribute('title', 'https://grafana.example.net/d/abc');
		expect(container.querySelector('[data-secondary="title"]')).toHaveTextContent('Grafana');
		expect(container.querySelector('[data-secondary="url"]')).toHaveTextContent(
			'grafana.example.net/d/abc'
		);
		expect(container.querySelectorAll('a')).toHaveLength(1);
	});

	test('follows the display switches', () => {
		const { anchor, container } = setup(component, {
			link: { description: 'Leírás', title: 'Cím', url: 'https://a.io/x' },
			fields: { showIcon: false, showUrl: false, showTitle: true, showDescription: false }
		});
		expect(anchor).toHaveTextContent('Cím');
		expect(container.querySelector('[data-secondary]')).toBeNull();
		expect(container.querySelector('img')).toBeNull();
	});

	test('a left click and the browser click of Enter count exactly one visit each', async () => {
		const { anchor, visit } = setup(component);
		anchor.addEventListener('click', (event) => event.preventDefault());
		await fireEvent.click(anchor);
		expect(visit).toHaveBeenCalledTimes(1);
		anchor.focus();
		const u = user();
		await u.keyboard('{Enter}');
		expect(visit).toHaveBeenCalledTimes(2);
		expect(visit).toHaveBeenCalledWith(7);
	});

	test('a middle click counts a visit, a right click does not', async () => {
		const { anchor, visit } = setup(component);
		await fireEvent(anchor, new MouseEvent('auxclick', { bubbles: true, button: 1 }));
		expect(visit).toHaveBeenCalledTimes(1);
		await fireEvent(anchor, new MouseEvent('auxclick', { bubbles: true, button: 2 }));
		await fireEvent.contextMenu(anchor);
		expect(visit).toHaveBeenCalledTimes(1);
	});

	test('dragging an own link marks it as internal', async () => {
		const { anchor } = setup(component);
		const setData = vi.fn();
		const event = new Event('dragstart', { bubbles: true }) as DragEvent;
		Object.defineProperty(event, 'dataTransfer', { value: { setData } });
		await fireEvent(anchor, event);
		expect(setData).toHaveBeenCalledWith(INTERNAL_DRAG_TYPE, '7');
	});

	test('the owner menu lists every action and runs the chosen one', async () => {
		const { onaction, link } = setup(component, { link: { title: 'Grafana', description: '' } });
		const u = user();
		await u.click(screen.getByRole('button', { name: t('links.row.menu', { label: 'Grafana' }) }));
		const labels = screen
			.queryAllByRole('menuitem', { hidden: true })
			.map((item) => item.textContent?.trim());
		expect(labels).toEqual([
			t('links.actions.open'),
			t('links.actions.copyUrl'),
			t('links.actions.copyMarkdown'),
			t('links.actions.details'),
			t('links.actions.edit'),
			t('links.actions.tags'),
			t('links.actions.share'),
			t('links.actions.refresh'),
			t('links.actions.pin'),
			t('links.actions.archive'),
			t('links.actions.delete')
		]);
		await u.click(getFloating('menuitem', t('links.actions.details')));
		expect(onaction).toHaveBeenCalledWith('details', link);
	});

	test('a shared link offers no owner actions; a customer cannot share', async () => {
		setup(component, {
			link: { isOwner: false, owner: { id: 2, username: 'peter', displayName: 'Nagy Péter' } },
			me: makeMe('customer')
		});
		const u = user();
		await u.click(document.querySelector<HTMLElement>('[aria-haspopup="menu"]')!);
		expect(queryFloating('menuitem', t('links.actions.edit'))).toBeNull();
		expect(queryFloating('menuitem', t('links.actions.share'))).toBeNull();
		expect(queryFloating('menuitem', t('links.actions.delete'))).toBeNull();
		expect(queryFloating('menuitem', t('links.actions.tags'))).not.toBeNull();
		expect(document.querySelector('[data-marker="shared"]')?.textContent?.trim()).toBe(
			t('links.row.sharedBy', { owner: 'Nagy Péter' })
		);
	});

	test('an owner without the share permission has no share action', async () => {
		setup(component, { me: makeMe('customer') });
		const u = user();
		await u.click(document.querySelector<HTMLElement>('[aria-haspopup="menu"]')!);
		expect(queryFloating('menuitem', t('links.actions.share'))).toBeNull();
		expect(queryFloating('menuitem', t('links.actions.edit'))).not.toBeNull();
	});

	test('shows the markers of a pinned, archived and pending link', () => {
		const { container } = setup(component, {
			link: {
				pinned: true,
				archivedAt: '2026-01-01T00:00:00Z',
				fetchStatus: 'pending'
			}
		});
		expect(container.querySelector('[data-marker="pinned"]')).toHaveTextContent(
			t('links.row.pinned')
		);
		expect(container.querySelector('[data-marker="archived"]')).toHaveTextContent(
			t('links.row.archived')
		);
		expect(
			within(container.querySelector<HTMLElement>('[data-marker="pending"]')!).getByLabelText(
				t('links.row.pending')
			)
		).toBeInTheDocument();
	});

	test('shows tag chips and names the hidden rest', () => {
		tags.items = [
			makeTag({ id: 1, name: 'a1' }),
			makeTag({ id: 2, name: 'b2' }),
			makeTag({ id: 3, name: 'c3' }),
			makeTag({ id: 4, name: 'd4' })
		];
		const { container } = setup(component, { link: { tagIds: [1, 2, 3, 4, 99] } });
		const more = container.querySelector('[data-more-tags]')!;
		// Below 1280 px a row shows one chip (see 'LinkRow width'), a tile always two.
		const max = component === LinkRow ? 1 : 2;
		expect(more).toHaveAttribute('data-more-tags', String(4 - max));
		expect(more).toHaveTextContent(`+${4 - max}`);
		expect(container).toHaveTextContent('a1');
	});

	test('shows the use count', () => {
		const { container } = setup(component, { link: { useCount: 1234 } });
		expect(container.querySelector('[data-uses]')).toHaveTextContent(fmt.number(1234));
		expect(container.querySelector('[data-uses]')).toHaveTextContent(
			t('links.row.uses', { count: 1234 })
		);
	});

	test('a highlight scrolls the item into view and ends after two seconds', async () => {
		vi.useFakeTimers();
		const scroll = vi.spyOn(Element.prototype, 'scrollIntoView');
		const { store, container } = setup(component);
		store.highlight(7);
		await vi.advanceTimersByTimeAsync(0);
		expect(scroll).toHaveBeenCalledWith({ block: 'nearest', behavior: 'auto' });
		expect(container.querySelector('.kxm-flash')).not.toBeNull();
		await vi.advanceTimersByTimeAsync(HIGHLIGHT_MS);
		expect(store.highlightId).toBeNull();
	});
});

describe('LinkRow specifics', () => {
	test('the pin button runs the pin action and reports its state', async () => {
		const { onaction, link } = setup(LinkRow, { link: { pinned: true } });
		const button = screen.getByRole('button', { name: t('links.actions.unpin') });
		expect(button).toHaveAttribute('aria-pressed', 'true');
		await user().click(button);
		expect(onaction).toHaveBeenCalledWith('pin', link);
	});

	test('shows the last use or a dash', () => {
		const { container, unmount } = setup(LinkRow, {
			link: { lastUsedAt: new Date(Date.now() - 2 * DAY).toISOString() }
		});
		expect(container.querySelector('time')).not.toBeNull();
		unmount();
		const second = setup(LinkRow, { link: { lastUsedAt: null } });
		expect(second.container).toHaveTextContent(fmt.missing);
	});
});

describe('LinkRow alignment', () => {
	test('markers follow the label, so a pinned or pending row starts its label at the same x', () => {
		const { anchor } = setup(LinkRow, { link: { pinned: true, fetchStatus: 'pending' } });
		const line = anchor.parentElement!;
		expect(line.firstElementChild).toBe(anchor);
		for (const marker of ['pinned', 'pending']) {
			const element = line.querySelector(`[data-marker="${marker}"]`)!;
			expect(
				anchor.compareDocumentPosition(element) & Node.DOCUMENT_POSITION_FOLLOWING
			).toBeTruthy();
		}
	});

	test('the meta slots have fixed widths, the expiry slot also on a pinned row', () => {
		const due = new Date(Date.now() + 60 * DAY).toISOString();
		const plain = setup(LinkRow, { link: { archiveDueAt: due } });
		const slot = plain.container.querySelector('[data-expiry-slot]')!;
		expect(slot.querySelector('[role="img"]')).not.toBeNull();
		const widths = (root: HTMLElement) =>
			['[data-uses]', '[data-last-used]', '[data-expiry-slot]'].map((selector) =>
				[...root.querySelector(selector)!.classList].find((c) => c.startsWith('w-'))
			);
		const plainWidths = widths(plain.container);
		plain.unmount();
		const pinned = setup(LinkRow, { link: { pinned: true, archiveDueAt: null } });
		expect(pinned.container.querySelector('[data-expiry-slot] [role="img"]')).toBeNull();
		expect(widths(pinned.container)).toEqual(plainWidths);
		// Wide enough for the longest relative time of hu/en/de ("in dieser Minute").
		expect(plainWidths).toEqual(['w-[6ch]', 'w-[16ch]', 'w-14']);
	});

	test('without automatic archiving there is no empty expiry slot', () => {
		const { container } = setup(LinkRow, { days: 0 });
		expect(container.querySelector('[data-expiry-slot]')).toBeNull();
	});
});

describe('LinkRow width (plan 8.7.3: the label comes first)', () => {
	function atWidth(width: number): void {
		vi.stubGlobal('matchMedia', (query: string) => {
			const min = /min-width:\s*(\d+)px/.exec(query);
			return {
				matches: min ? width >= Number(min[1]) : false,
				media: query,
				onchange: null,
				addListener: () => {},
				removeListener: () => {},
				addEventListener: () => {},
				removeEventListener: () => {},
				dispatchEvent: () => false
			};
		});
	}

	test.each([
		[1024, 1],
		[1280, 2],
		[1536, 3]
	])('at %i px a row shows %i tag chips and names the rest', (width, shown) => {
		atWidth(width);
		tags.items = [1, 2, 3, 4].map((id) => makeTag({ id, name: `tag${id}` }));
		const { container } = setup(LinkRow, { link: { tagIds: [1, 2, 3, 4] } });
		const more = container.querySelector('[data-more-tags]')!;
		expect(more).toHaveAttribute('data-more-tags', String(4 - shown));
		expect(container.querySelectorAll('[data-row-meta] .truncate')).toHaveLength(shown);
	});

	test('from md the label column keeps 12rem and only the chips give way', () => {
		const { container } = setup(LinkRow, { link: { pinned: true } });
		const row = container.querySelector('li')!;
		expect(row).toHaveClass('md:grid-cols-[20px_minmax(12rem,1fr)_minmax(0,auto)]');
		const meta = container.querySelector('[data-row-meta]')!;
		expect(meta).toHaveClass('min-w-0');
		const slots = [...meta.children].filter((child) => !child.matches('.hidden.md\\:flex'));
		for (const slot of slots) expect(slot).toHaveClass('shrink-0');
	});
});

describe('LinkTile footer', () => {
	test('long tag names wrap onto their own line instead of being cut to a few letters', () => {
		tags.items = [
			makeTag({ id: 1, name: 'Dokumentáció' }),
			makeTag({ id: 2, name: 'Belső eszközök' })
		];
		const { container } = setup(LinkTile, { link: { tagIds: [1, 2] } });
		const footer = container.querySelector<HTMLElement>('[data-tile-footer]')!;
		const chips = within(footer).getByText('Dokumentáció').closest('span.inline-flex')!;
		expect(chips).not.toHaveClass('max-w-32');
		expect(chips.parentElement).toHaveClass('flex-wrap');
		expect(footer).toHaveTextContent('Belső eszközök');
	});

	test('the meta sits on its own fixed row, so the footers of one grid row line up', () => {
		tags.items = [makeTag({ id: 1, name: 'Dokumentáció' })];
		const due = new Date(Date.now() + 60 * DAY).toISOString();
		const { container } = setup(LinkTile, { link: { tagIds: [1], archiveDueAt: due } });
		const footer = container.querySelector<HTMLElement>('[data-tile-footer]')!;
		expect(footer).toHaveClass('flex-col');
		expect(footer).not.toHaveClass('flex-wrap');
		const meta = footer.querySelector<HTMLElement>('[data-tile-meta]')!;
		expect(footer.lastElementChild).toBe(meta);
		expect(meta).toHaveClass('h-5');
		expect(meta.querySelector('[data-uses]')).not.toBeNull();
		expect(meta.querySelector('[role="img"]')).not.toBeNull();
		expect(within(meta).queryByText('Dokumentáció')).toBeNull();
	});
});

describe('ExpiryMeter', () => {
	const now = new Date('2026-03-01T00:00:00Z');
	const since = new Date(now.getTime() - 30 * DAY).toISOString();

	function meter(link: Partial<LinkItem>, days = 90) {
		return render(ExpiryMeter, {
			link: makeLink({ inactiveSince: since, ...link }),
			effectiveArchiveAfterDays: days,
			now
		});
	}

	test('shows the filled segments with a label', () => {
		const due = new Date(now.getTime() + 60 * DAY).toISOString();
		meter({ archiveDueAt: due });
		const img = screen.getByRole('img');
		expect(img).toHaveAttribute('data-segments', '4');
		expect(img).toHaveAccessibleName(t('links.expiry.label', { date: fmt.date(due) }));
		expect(img.querySelectorAll('.bg-muted-foreground')).toHaveLength(4);
		expect(img.querySelectorAll('.bg-border')).toHaveLength(1);
	});

	test('turns into a day count in the last week and says today when due', () => {
		const soon = meter({ archiveDueAt: new Date(now.getTime() + 3 * DAY).toISOString() });
		// The day count keeps U+00A0 before its unit (plan 8.5.4), so whitespace is compared as is.
		expect(screen.getByRole('img').textContent).toContain(t('links.expiry.daysLeft', { count: 3 }));
		soon.unmount();
		meter({ archiveDueAt: now.toISOString() });
		expect(screen.getByRole('img')).toHaveTextContent(t('links.expiry.today'));
	});

	test('is hidden on pinned links and when archiving is off', () => {
		const due = new Date(now.getTime() + 60 * DAY).toISOString();
		const pinned = meter({ archiveDueAt: due, pinned: true });
		expect(screen.queryByRole('img')).toBeNull();
		pinned.unmount();
		meter({ archiveDueAt: due }, 0);
		expect(screen.queryByRole('img')).toBeNull();
	});
});

describe('visits through the real store', () => {
	test('a click on the label posts one visit', async () => {
		const api = mockApi({ 'POST /api/links/7/visit': json({ useCount: 1 }) });
		const { store } = makeStore();
		const link = makeLink({ id: 7 });
		store.items = [link];
		render(LinkRow, {
			link,
			me: makeMe(),
			fields: ALL_FIELDS,
			effectiveArchiveAfterDays: 90,
			onaction: vi.fn(),
			store
		});
		const anchor = document.querySelector<HTMLAnchorElement>('[data-link-anchor]')!;
		anchor.addEventListener('click', (event) => event.preventDefault());
		await fireEvent.click(anchor);
		await vi.waitFor(() => expect(api.calls('POST /api/links/7/visit')).toHaveLength(1));
		expect(store.find(7)?.useCount).toBe(1);
	});
});
