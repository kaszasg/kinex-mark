// Application frame: sidebar, top bar, theme and language switches, tag picker (plan 8.7.1).
import { fireEvent, render, screen, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { createRawSnippet } from 'svelte';
import { setMode } from 'mode-watcher';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { afterNavigate, goto } from '$app/navigation';
import { i18n, t } from '$lib/i18n/index.js';
import { MAX_QUERY_LENGTH } from '$lib/links/query.js';
import { linkStats } from '$lib/state/link-stats.svelte.js';
import { preferences } from '$lib/state/preferences.svelte.js';
import { session } from '$lib/state/session.svelte.js';
import { tags, TagsStore } from '$lib/state/tags.svelte.js';
import { setTestPage } from '$lib/test/app-state.svelte.js';
import { makeMe, makePreferences, makeTag } from '$lib/test/fixtures.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import AppShell from './AppShell.svelte';
import AppSidebar from './AppSidebar.svelte';
import LocaleSwitcher from './LocaleSwitcher.svelte';
import { getShellContext } from './shell-context.js';
import ContextProbe from '$lib/test/ContextProbe.svelte';
import TagPicker from './TagPicker.svelte';
import ThemeToggle from './ThemeToggle.svelte';
import Topbar from './Topbar.svelte';

vi.mock('mode-watcher', async (importOriginal) => ({
	...(await importOriginal<typeof import('mode-watcher')>()),
	setMode: vi.fn()
}));

const user = () => userEvent.setup({ pointerEventsCheck: 0 });

/** The user menu trigger's name: its visible name and role, then what it is. */
function userMenuName(me: ReturnType<typeof makeMe>): string {
	return t('nav.userMenu', { name: me.displayName || me.username, role: t(`roles.${me.role}`) });
}

// Floating content is visibility:hidden in jsdom (no layout), so its text does not count as an
// accessible name: floating items are found by role and text instead.
function queryFloating(role: string, text: string): HTMLElement | null {
	return (
		screen.queryAllByRole(role, { hidden: true }).find((el) => el.textContent?.trim() === text) ??
		null
	);
}
function getFloating(role: string, text: string): HTMLElement {
	const el = queryFloating(role, text);
	if (!el) throw new Error(`no ${role} with text "${text}"`);
	return el;
}
function findFloating(role: string, text: string): Promise<HTMLElement> {
	return vi.waitFor(() => getFloating(role, text));
}

beforeEach(() => {
	vi.mocked(setMode).mockClear();
	vi.mocked(goto).mockClear();
	session.clear();
	tags.reset();
	linkStats.reset();
	mockApi({ 'PATCH /api/me/preferences': json(makePreferences()) });
});

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
});

describe('ThemeToggle', () => {
	test('signed out it only switches the mode', async () => {
		render(ThemeToggle);
		await user().click(screen.getByRole('button', { name: t('nav.theme.toggle') }));
		expect(setMode).toHaveBeenCalledWith('light');
	});

	test('signed in it saves the preference', async () => {
		const update = vi.spyOn(preferences, 'update').mockImplementation(() => {});
		renderWithApp(ThemeToggle, {}, { me: makeMe() });
		await user().click(screen.getByRole('button', { name: t('nav.theme.toggle') }));
		expect(update).toHaveBeenCalledWith({ theme: 'light' });
	});
});

describe('LocaleSwitcher', () => {
	test('inline variant: an explicit choice before sign-in', async () => {
		render(LocaleSwitcher, { variant: 'inline' });
		const group = screen.getByRole('group', { name: t('nav.locale.label') });
		const english = within(group).getByRole('radio', { name: 'English' });
		expect(english).toHaveAttribute('lang', 'en');
		await user().click(english);
		expect(i18n.locale).toBe('en');
		expect(i18n.explicit).toBe('en');
	});

	test('dropdown variant: saves the preference when signed in', async () => {
		const update = vi.spyOn(preferences, 'update').mockImplementation(() => {});
		renderWithApp(LocaleSwitcher, {}, { me: makeMe() });
		const u = user();
		await u.click(screen.getByRole('button', { name: t('nav.locale.label') }));
		const deutsch = await findFloating('menuitemradio', 'Deutsch');
		expect(deutsch).toHaveAttribute('lang', 'de');
		await u.click(deutsch);
		expect(update).toHaveBeenCalledWith({ locale: 'de' });
		// The choice closes the menu.
		await vi.waitFor(() => expect(screen.queryByRole('menu', { hidden: true })).toBeNull());
		expect(screen.getByRole('button', { name: t('nav.locale.label') })).toHaveAttribute(
			'aria-expanded',
			'false'
		);
	});

	test('choosing the active locale changes nothing', async () => {
		render(LocaleSwitcher, { variant: 'inline' });
		await user().click(screen.getByRole('radio', { name: 'Magyar' }));
		expect(i18n.explicit).toBeNull();
	});
});

describe('TagPicker', () => {
	function storeWith(...items: ReturnType<typeof makeTag>[]) {
		const store = new TagsStore();
		store.items = items;
		store.status = 'ready';
		return store;
	}

	test('selects and removes tags', async () => {
		const store = storeWith(makeTag({ id: 1, name: 'infra' }), makeTag({ id: 2, name: 'docs' }));
		const onchange = vi.fn();
		render(TagPicker, { store, value: [2], onchange });
		expect(screen.getByText('docs')).toBeInTheDocument();
		const u = user();
		await u.click(screen.getByRole('button', { name: t('common.tagPicker.label') }));
		await u.click(await findFloating('option', 'infra'));
		expect(onchange).toHaveBeenLastCalledWith([2, 1]);
		await u.click(getFloating('option', 'docs'));
		expect(onchange).toHaveBeenLastCalledWith([1]);
	});

	test('filters by the search text and creates a missing tag', async () => {
		const store = storeWith(makeTag({ id: 1, name: 'infra' }));
		const create = vi.spyOn(store, 'create').mockResolvedValue(makeTag({ id: 5, name: 'ops' }));
		const onchange = vi.fn();
		render(TagPicker, { store, onchange });
		const u = user();
		await u.click(screen.getByRole('button', { name: t('common.tagPicker.label') }));
		await u.type(await screen.findByRole('combobox', { hidden: true }), 'zz');
		expect(queryFloating('option', 'infra')).toBeNull();
		await u.clear(screen.getByRole('combobox', { hidden: true }));
		await u.type(screen.getByRole('combobox', { hidden: true }), 'ops');
		await u.click(getFloating('option', t('common.tagPicker.create', { name: 'ops' })));
		expect(create).toHaveBeenCalledWith('ops');
		await vi.waitFor(() => expect(onchange).toHaveBeenLastCalledWith([5]));
	});

	test('every opening starts with an empty search (E2)', async () => {
		render(TagPicker, { store: storeWith(makeTag({ id: 1, name: 'infra' })) });
		const u = user();
		const trigger = screen.getByRole('button', { name: t('common.tagPicker.label') });
		await u.click(trigger);
		await u.type(await screen.findByRole('combobox', { hidden: true }), 'nagyon');
		await u.keyboard('{Escape}');
		await vi.waitFor(() => expect(screen.queryByRole('combobox', { hidden: true })).toBeNull());
		await u.click(trigger);
		expect(await screen.findByRole('combobox', { hidden: true })).toHaveValue('');
	});

	test('Enter creates when nothing matches; a failure shows a toast', async () => {
		const error = vi.spyOn(toast, 'error').mockImplementation(() => 1);
		const store = storeWith();
		vi.spyOn(store, 'create').mockRejectedValue(new Error('offline'));
		render(TagPicker, { store });
		const u = user();
		await u.click(screen.getByRole('button', { name: t('common.tagPicker.label') }));
		await u.type(await screen.findByRole('combobox', { hidden: true }), 'new{Enter}');
		await vi.waitFor(() => expect(error).toHaveBeenCalledOnce());
	});

	test('at the limit unselected tags are disabled and the limit is explained', async () => {
		const store = storeWith(makeTag({ id: 1, name: 'a' }), makeTag({ id: 2, name: 'b' }));
		render(TagPicker, { store, value: [1], max: 1, allowCreate: false });
		const u = user();
		await u.click(screen.getByRole('button', { name: t('common.tagPicker.label') }));
		expect(await screen.findByText(t('common.tagPicker.limit', { max: 1 }))).toBeInTheDocument();
		expect(getFloating('option', 'b')).toHaveAttribute('aria-disabled', 'true');
		await u.type(screen.getByRole('combobox', { hidden: true }), 'zzz');
		expect(screen.getByText(t('common.tagPicker.empty'))).toBeInTheDocument();
	});

	test('the chip remove button deselects; it says remove, not delete', async () => {
		const store = storeWith(makeTag({ id: 1, name: 'infra' }));
		const onchange = vi.fn();
		render(TagPicker, { store, value: [1, 99], onchange });
		expect(
			screen.queryByRole('button', { name: `${t('common.actions.delete')}: infra` })
		).toBeNull();
		await user().click(
			screen.getByRole('button', { name: t('common.tagPicker.remove', { name: 'infra' }) })
		);
		expect(onchange).toHaveBeenCalledWith([99]);
	});

	test('the suggestion list has a translated name', async () => {
		render(TagPicker, { store: storeWith(makeTag({ id: 1, name: 'infra' })) });
		await user().click(screen.getByRole('button', { name: t('common.tagPicker.label') }));
		const list = await vi.waitFor(() => {
			const found = document.querySelector('[data-slot="command-list"]');
			if (!found) throw new Error('no list');
			return found;
		});
		expect(list).toHaveAttribute('aria-label', t('common.tagPicker.label'));
		expect(list.getAttribute('aria-label')).not.toMatch(/Suggestions/);
	});
});

describe('AppSidebar', () => {
	test('collection, frequent tags, account and admin groups for an admin', () => {
		setTestPage('/tags');
		linkStats.active = 1284;
		tags.items = [
			makeTag({ id: 1, name: 'infra', activeLinkCount: 4 }),
			makeTag({ id: 2, name: 'idle', activeLinkCount: 0 })
		];
		tags.status = 'ready';
		renderWithApp(AppSidebar, {}, { me: makeMe('admin', { displayName: 'Kovacs Anna' }) });
		expect(screen.getByRole('link', { name: t('nav.tags') })).toHaveAttribute(
			'aria-current',
			'page'
		);
		expect(screen.getByRole('link', { name: t('nav.links') })).not.toHaveAttribute('aria-current');
		expect(screen.getByText('1284')).toBeInTheDocument();
		expect(screen.getByText(t('nav.groups.frequentTags'))).toBeInTheDocument();
		expect(screen.getByRole('link', { name: 'infra' })).toHaveAttribute('href', '/?tags=1');
		expect(screen.queryByRole('link', { name: 'idle' })).toBeNull();
		expect(screen.getByRole('link', { name: t('nav.adminAudit') })).toHaveAttribute(
			'href',
			'/admin/audit'
		);
		expect(screen.getByText('Kovacs Anna')).toBeInTheDocument();
		expect(screen.getByText(t('roles.admin'))).toBeInTheDocument();
	});

	test('a customer without tags sees no admin and no frequent tags', () => {
		setTestPage('/');
		renderWithApp(AppSidebar, {}, { me: makeMe('customer', { displayName: '' }) });
		expect(screen.getByRole('link', { name: t('nav.links') })).toHaveAttribute(
			'aria-current',
			'page'
		);
		expect(screen.queryByText(t('nav.groups.admin'))).toBeNull();
		expect(screen.queryByText(t('nav.groups.frequentTags'))).toBeNull();
		expect(screen.getByText('kanna')).toBeInTheDocument();
	});

	test('the user menu signs out and links close the sheet', async () => {
		const logout = vi.spyOn(session, 'logout').mockResolvedValue();
		const onNavigate = vi.fn();
		setTestPage('/settings/profile');
		renderWithApp(AppSidebar, { onNavigate }, { me: makeMe() });
		expect(screen.getByRole('link', { name: t('nav.settings') })).toHaveAttribute(
			'aria-current',
			'page'
		);
		const u = user();
		await u.click(screen.getByRole('button', { name: userMenuName(makeMe()) }));
		const privacy = await findFloating('menuitem', t('nav.privacyNotice'));
		expect(privacy).toHaveAttribute('href', '/privacy');
		expect(getFloating('menuitem', t('nav.settings'))).toHaveAttribute('href', '/settings');
		await u.click(getFloating('menuitem', t('nav.logout')));
		expect(logout).toHaveBeenCalledOnce();
		const link = screen.getByRole('link', { name: t('nav.links') });
		link.addEventListener('click', (event) => event.preventDefault());
		await fireEvent.click(link);
		expect(onNavigate).toHaveBeenCalled();
	});

	test('menu links and the logo also close the sheet', async () => {
		const onNavigate = vi.fn();
		document.addEventListener('click', (event) => event.preventDefault(), { once: false });
		renderWithApp(AppSidebar, { onNavigate }, { me: makeMe() });
		const u = user();
		await u.click(screen.getByRole('link', { name: /Kinex Mark/ }));
		tags.items = [makeTag({ id: 3, name: 'ops', activeLinkCount: 1 })];
		tags.status = 'ready';
		await u.click(await screen.findByRole('link', { name: 'ops' }));
		await u.click(screen.getByRole('button', { name: userMenuName(makeMe()) }));
		await u.click(await findFloating('menuitem', t('nav.settings')));
		await u.click(screen.getByRole('button', { name: userMenuName(makeMe()) }));
		await u.click(await findFloating('menuitem', t('nav.privacyNotice')));
		expect(onNavigate).toHaveBeenCalledTimes(4);
	});

	test('the user menu is named after its visible text, inside a navigation landmark', () => {
		const me = makeMe('admin', { displayName: 'Kovacs Anna' });
		renderWithApp(AppSidebar, {}, { me });
		const trigger = screen.getByRole('button', { name: userMenuName(me) });
		// WCAG 2.5.3: the accessible name starts with the visible "name role" text.
		const visible = `${me.displayName} ${t('roles.admin')}`;
		expect(trigger.getAttribute('aria-label')?.startsWith(visible)).toBe(true);
		const nav = screen.getByRole('navigation', { name: t('nav.sidebarTitle') });
		expect(within(nav).getByRole('link', { name: t('nav.links') })).toBeInTheDocument();
		expect(within(nav).getByRole('link', { name: t('nav.adminAudit') })).toBeInTheDocument();
	});

	test('group labels use the eyebrow scale and full muted contrast (GRAPHICS.md 3.5, WCAG 1.4.3)', () => {
		renderWithApp(AppSidebar, {}, { me: makeMe() });
		const label = screen.getByText(t('nav.groups.collection'));
		expect(label).toHaveClass('text-[11px]', 'leading-4', 'uppercase', 'text-muted-foreground');
		// The generated text-xs and an alpha-modified sidebar colour (4.35:1) must not survive.
		for (const cls of label.classList) {
			expect(cls).not.toMatch(/^text-xs$|^text-sidebar-foreground\//);
		}
	});

	test('signed out renders no user menu', () => {
		render(AppSidebar);
		const suffix = t('nav.userMenu', { name: '', role: '' }).trim();
		expect(screen.queryByRole('button', { name: (name) => name.endsWith(suffix) })).toBeNull();
	});
});

describe('Topbar', () => {
	test('breadcrumb with the current page and a link to the section', () => {
		setTestPage('/settings/security');
		render(Topbar);
		const nav = screen.getByRole('navigation', { name: t('nav.breadcrumb.label') });
		expect(within(nav).getByRole('link', { name: t('nav.breadcrumb.settings') })).toHaveAttribute(
			'href',
			'/settings'
		);
		expect(within(nav).getByText(t('nav.settingsTabs.security'))).toHaveAttribute(
			'aria-current',
			'page'
		);
	});

	test('section crumbs without a link and unknown pages without breadcrumb', () => {
		setTestPage('/admin/users');
		const { unmount } = render(Topbar);
		expect(screen.getByText(t('nav.breadcrumb.admin')).tagName).toBe('SPAN');
		unmount();
		setTestPage('/elsewhere');
		render(Topbar);
		expect(screen.queryByRole('navigation')).toBeNull();
	});

	test('Enter on another page opens the links with the query', async () => {
		setTestPage('/tags');
		render(Topbar);
		const u = user();
		await u.type(screen.getByRole('searchbox', { name: t('nav.search.label') }), 'grafana{Enter}');
		expect(goto).toHaveBeenCalledWith('/?q=grafana', { reset: true });
		expect(goto).toHaveBeenCalledTimes(1);
	});

	test('on the links page the search filters live and keeps other parameters', async () => {
		vi.useFakeTimers();
		setTestPage('/?tags=1&q=old');
		render(Topbar);
		const input = screen.getByRole('searchbox') as HTMLInputElement;
		expect(input.value).toBe('old');
		input.focus();
		await fireEvent.input(input, { target: { value: 'new' } });
		await vi.advanceTimersByTimeAsync(250);
		expect(goto).toHaveBeenCalledWith('/?tags=1&q=new', { replace: true, reset: false });
		await fireEvent.input(input, { target: { value: '' } });
		await fireEvent.submit(input.form!);
		expect(goto).toHaveBeenLastCalledWith('/?tags=1', { reset: false });
	});

	test('the URL query follows navigation when the field is not focused', async () => {
		setTestPage('/?q=first');
		render(Topbar);
		const input = screen.getByRole('searchbox') as HTMLInputElement;
		expect(input.value).toBe('first');
		setTestPage('/?q=second');
		await vi.waitFor(() => expect(input.value).toBe('second'));
		setTestPage('/');
		await vi.waitFor(() => expect(input.value).toBe(''));
	});

	test('"/" focuses the search; typing in a field does not', async () => {
		render(Topbar);
		const input = screen.getByRole('searchbox');
		await fireEvent.keyDown(document.body, { key: '/' });
		await vi.waitFor(() => expect(document.activeElement).toBe(input));
		input.blur();
		await fireEvent.keyDown(document.body, { key: 'x' });
		expect(document.activeElement).not.toBe(input);
	});

	test('below 768 px the search is an icon button that opens a full-width field', async () => {
		render(Topbar);
		const form = screen.getByRole('search');
		expect(form).toHaveClass('max-md:hidden');
		const open = screen.getByRole('button', { name: t('nav.search.open') });
		expect(open).toHaveClass('md:hidden');
		expect(open).toHaveAttribute('aria-expanded', 'false');
		const u = user();
		await u.click(open);
		const input = screen.getByRole('searchbox', { name: t('nav.search.label') });
		await vi.waitFor(() => expect(document.activeElement).toBe(input));
		expect(form).toHaveAttribute('data-search-open', 'true');
		expect(form).toHaveClass('max-md:absolute', 'max-md:inset-0', 'max-md:max-w-none');
		expect(form).not.toHaveClass('max-md:hidden');
		expect(input).toHaveAttribute('maxlength', String(MAX_QUERY_LENGTH));
		// Escape folds it back and returns the focus to the icon button.
		await u.keyboard('{Escape}');
		await vi.waitFor(() => expect(document.activeElement).toBe(open));
		expect(form).toHaveClass('max-md:hidden');
		await u.click(open);
		await u.click(screen.getByRole('button', { name: t('nav.search.close') }));
		await vi.waitFor(() => expect(form).toHaveAttribute('data-search-open', 'false'));
		// Leaving the field also folds it.
		await u.click(open);
		await vi.waitFor(() => expect(document.activeElement).toBe(input));
		input.blur();
		await vi.waitFor(() => expect(form).toHaveAttribute('data-search-open', 'false'));
	});

	test('Escape clears the query and ?q= on the links page, also after "/" on a desktop (E2)', async () => {
		const real = window.matchMedia;
		vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
			...real(query),
			matches: query.includes('min-width: 768px')
		}));
		setTestPage('/?tags=1&q=docker');
		render(Topbar);
		const input = screen.getByRole('searchbox', {
			name: t('nav.search.label')
		}) as HTMLInputElement;
		expect(input.value).toBe('docker');
		await fireEvent.keyDown(document.body, { key: '/' });
		await vi.waitFor(() => expect(document.activeElement).toBe(input));
		// On a desktop the field is always shown: "/" does not open the mobile overlay.
		expect(screen.getByRole('search')).toHaveAttribute('data-search-open', 'false');
		const escape = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true });
		input.dispatchEvent(escape);
		expect(escape.defaultPrevented).toBe(true);
		await vi.waitFor(() => expect(input.value).toBe(''));
		expect(goto).toHaveBeenCalledWith('/?tags=1', { replace: true, reset: false });
		expect(document.activeElement).toBe(input);
		// A second Escape on the empty field changes nothing on a desktop.
		vi.mocked(goto).mockClear();
		await fireEvent.keyDown(input, { key: 'Escape' });
		expect(goto).not.toHaveBeenCalled();
		expect(document.activeElement).toBe(input);
	});

	test('Escape clears a typed query before the live search ran; elsewhere it never navigates', async () => {
		vi.useFakeTimers();
		setTestPage('/tags');
		render(Topbar);
		const input = screen.getByRole('searchbox') as HTMLInputElement;
		input.focus();
		await fireEvent.input(input, { target: { value: 'grafana' } });
		await fireEvent.keyDown(input, { key: 'Escape' });
		expect(input.value).toBe('');
		await vi.advanceTimersByTimeAsync(500);
		expect(goto).not.toHaveBeenCalled();
	});

	test('the search field takes the global 2 px ring with its 2 px offset (GRAPHICS.md 6.2)', () => {
		render(Topbar);
		const input = screen.getByRole('searchbox');
		const classes = [...input.classList];
		expect(classes.filter((name) => /outline/.test(name))).toEqual([]);
	});

	test('the menu button toggles the sidebar through the shell context', async () => {
		render(Topbar);
		await user().click(screen.getByRole('button', { name: t('nav.toggleSidebar') }));
		expect(screen.getByRole('button', { name: t('nav.toggleSidebar') })).toBeInTheDocument();
	});
});

describe('AppShell', () => {
	const content = createRawSnippet(() => ({ render: () => '<p>page content</p>' }));

	test('skip link, main landmark and content', () => {
		renderWithApp(AppShell, { children: content }, { me: makeMe() });
		expect(screen.getByRole('link', { name: t('common.skipToContent') })).toHaveAttribute(
			'href',
			'#main'
		);
		const main = screen.getByRole('main');
		expect(main).toHaveAttribute('id', 'main');
		expect(main).toHaveAttribute('tabindex', '-1');
		expect(within(main).getByText('page content')).toBeInTheDocument();
	});

	test('the menu button opens the off-canvas sidebar with a translated title', async () => {
		renderWithApp(AppShell, { children: content }, { me: makeMe() });
		await user().click(screen.getByRole('button', { name: t('nav.toggleSidebar') }));
		const sheet = await screen.findByRole('dialog', { name: t('nav.sidebarTitle') });
		await user().click(within(sheet).getByRole('link', { name: t('nav.tags') }));
		// The sheet closes; its exit animation may keep the node mounted for a moment.
		await vi.waitFor(() => {
			expect(!sheet.isConnected || sheet.getAttribute('data-state') === 'closed').toBe(true);
		});
	});

	test('a route change moves the focus to <main>; a query change leaves it alone (plan 8.9)', async () => {
		renderWithApp(AppShell, { children: content }, { me: makeMe() });
		const calls = vi.mocked(afterNavigate).mock.calls;
		const callback = calls[calls.length - 1][0];
		const main = screen.getByRole('main');
		const nav = (from: string, to: string, type = 'link') =>
			callback({
				from: { url: new URL(from, 'http://localhost') },
				to: { url: new URL(to, 'http://localhost') },
				type
			} as unknown as Parameters<typeof callback>[0]);
		(document.activeElement as HTMLElement | null)?.blur();
		nav('/tags', '/');
		expect(document.activeElement).toBe(main);
		main.blur();
		nav('/', '/?q=docker');
		expect(document.activeElement).toBe(document.body);
		nav('/', '/tags', 'enter');
		expect(document.activeElement).toBe(document.body);
		// A focus that the new page set itself (a dialog, a field) is kept.
		const search = screen.getByRole('searchbox', { name: t('nav.search.label') });
		search.focus();
		nav('/tags', '/');
		expect(document.activeElement).toBe(search);
	});

	test('"?" opens the shortcuts dialog', async () => {
		renderWithApp(AppShell, { children: content }, { me: makeMe() });
		const name = t('links.shortcuts.title');
		expect(screen.queryByRole('dialog', { name })).toBeNull();
		await fireEvent.keyDown(document.body, { key: '?' });
		expect(await screen.findByRole('dialog', { name })).toBeInTheDocument();
		await fireEvent.keyDown(document.body, { key: 'a' });
	});

	test('Ctrl/Cmd+B sets no cookie and stays with the focused field (plan D9, 6.14)', async () => {
		document.cookie = 'sidebar_state=; path=/; max-age=0';
		renderWithApp(AppShell, { children: content }, { me: makeMe() });
		const search = screen.getByRole('searchbox', { name: t('nav.search.label') });
		search.focus();
		for (const init of [{ ctrlKey: true }, { metaKey: true }]) {
			const inField = new KeyboardEvent('keydown', {
				key: 'b',
				bubbles: true,
				cancelable: true,
				...init
			});
			search.dispatchEvent(inField);
			expect(inField.defaultPrevented).toBe(false);
			const onPage = new KeyboardEvent('keydown', {
				key: 'b',
				bubbles: true,
				cancelable: true,
				...init
			});
			document.body.dispatchEvent(onPage);
			expect(onPage.defaultPrevented).toBe(false);
		}
		expect(document.cookie).toBe('');
	});

	test('outside the shell the context is a no-op', () => {
		const ctx = getShellContextOutside();
		expect(() => {
			ctx.toggleSidebar();
			ctx.closeSidebar();
			ctx.openShortcuts();
		}).not.toThrow();
	});
});

function getShellContextOutside() {
	let ctx: ReturnType<typeof getShellContext> | undefined;
	render(ContextProbe, {
		run: () => {
			ctx = getShellContext();
		}
	});
	return ctx!;
}

describe('errors from stores', () => {
	test('a failed tag load does not break the sidebar', async () => {
		mockApi({ 'GET /api/tags': apiError(500, 'internal_error') });
		await tags.load();
		renderWithApp(AppSidebar, {}, { me: makeMe() });
		expect(screen.getByRole('link', { name: t('nav.tags') })).toBeInTheDocument();
	});
});
