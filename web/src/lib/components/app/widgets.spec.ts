// Small building blocks of plan 8.4.
import { fireEvent, render, screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { createRawSnippet } from 'svelte';
import { toast } from 'svelte-sonner';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { ApiError } from '$lib/api/errors.js';
import { fmt, i18n, t } from '$lib/i18n/index.js';
import Hint from '$lib/features/links/Hint.svelte';
import { breadcrumbsFor, pageTitleKey } from './breadcrumbs.js';
import Button from './Button.svelte';
import ConfirmDialog from './ConfirmDialog.svelte';
import CopyButton from './CopyButton.svelte';
import DataTableHead from './DataTableHead.svelte';
import DataTableShell from './DataTableShell.svelte';
import ErrorView from './ErrorView.svelte';
import Favicon from './Favicon.svelte';
import KeyboardShortcut from './KeyboardShortcut.svelte';
import LineTabs from './LineTabs.svelte';
import PageHeader from './PageHeader.svelte';
import PageTitle from './PageTitle.svelte';
import QrCode from './QrCode.svelte';
import RelativeTime from './RelativeTime.svelte';
import TagChip from './TagChip.svelte';
import { Checkbox } from '$lib/components/ui/checkbox/index.js';
import { Input } from '$lib/components/ui/input/index.js';
import ToggleGroupProbe from '$lib/test/ToggleGroupProbe.svelte';
import { buttonTone, checkboxBox, inputSize } from './styles.js';

afterEach(() => {
	vi.useRealTimers();
});

const html = (markup: string) => createRawSnippet(() => ({ render: () => markup }));

describe('TagChip', () => {
	test('plain chip with the colour swatch', () => {
		const { container } = render(TagChip, { name: 'infra', color: 'ochre' });
		expect(screen.getByText('infra')).toBeInTheDocument();
		expect(container.querySelector('.bg-chart-2')).not.toBeNull();
	});

	test('toggle button with aria-pressed', async () => {
		const onclick = vi.fn();
		render(TagChip, { name: 'infra', color: 'blue', as: 'button', pressed: true, onclick });
		const chip = screen.getByRole('button', { name: 'infra' });
		expect(chip).toHaveAttribute('aria-pressed', 'true');
		expect(chip).toHaveClass('bg-accent');
		await userEvent.click(chip);
		expect(onclick).toHaveBeenCalledOnce();
	});

	test('link chip with a trailing snippet', () => {
		render(TagChip, {
			name: 'infra',
			color: 'rose',
			as: 'a',
			href: '/?tags=1',
			trailing: html('<em>3</em>')
		});
		expect(screen.getByRole('link')).toHaveAttribute('href', '/?tags=1');
		expect(screen.getByText('3')).toBeInTheDocument();
	});
});

describe('Favicon', () => {
	test('lazy image, falling back to the link icon on error', async () => {
		const { container } = render(Favicon, { src: '/api/icons/1', size: 20 });
		const img = container.querySelector('img')!;
		expect(img).toHaveAttribute('loading', 'lazy');
		expect(img).toHaveAttribute('alt', '');
		expect(img).toHaveClass('size-5');
		await fireEvent.error(img);
		expect(container.querySelector('img')).toBeNull();
		expect(container.querySelector('svg')).toHaveAttribute('aria-hidden', 'true');
	});

	test('no source shows the icon', () => {
		const { container } = render(Favicon, { src: null });
		expect(container.querySelector('svg')).toHaveClass('size-4');
	});
});

describe('CopyButton', () => {
	test('copies, toasts, calls back and shows the check for a while', async () => {
		vi.useFakeTimers();
		const success = vi.spyOn(toast, 'success').mockImplementation(() => 1);
		const onCopied = vi.fn();
		const { container } = render(CopyButton, {
			text: 'https://a.example',
			onCopied,
			label: 'Copy URL'
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Copy URL' }));
		await vi.waitFor(() => expect(onCopied).toHaveBeenCalledOnce());
		await expect(navigator.clipboard.readText()).resolves.toBe('https://a.example');
		expect(success).toHaveBeenCalledWith(t('common.actions.copied'));
		expect(container.querySelector('.text-success')).not.toBeNull();
		await fireEvent.click(screen.getByRole('button'));
		await vi.advanceTimersByTimeAsync(1500);
		expect(container.querySelector('.text-success')).toBeNull();
	});

	test('a failed copy does not confirm', async () => {
		vi.spyOn(toast, 'error').mockImplementation(() => 1);
		vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValueOnce(new Error('denied'));
		Object.assign(document, { execCommand: () => false });
		const onCopied = vi.fn();
		const { unmount } = render(CopyButton, { text: 'x', onCopied, size: 'icon' });
		expect(screen.getByRole('button', { name: t('common.actions.copy') })).toBeInTheDocument();
		await fireEvent.click(screen.getByRole('button'));
		await Promise.resolve();
		expect(onCopied).not.toHaveBeenCalled();
		unmount();
	});
});

describe('RelativeTime', () => {
	test('relative text with the full date in the title', () => {
		const now = new Date(2026, 10, 21, 12, 0);
		const value = new Date(2026, 10, 18, 9, 0);
		const { container } = render(RelativeTime, { value: value.toISOString(), now });
		const time = container.querySelector('time')!;
		expect(time.textContent).toBe(fmt.relative(value, now));
		expect(time).toHaveAttribute('datetime', value.toISOString());
		expect(time).toHaveAttribute('title', fmt.dateTime(value));
	});

	test.each([null, 'nonsense'])('%s renders the missing marker', (value) => {
		render(RelativeTime, { value });
		expect(screen.getByText(t('common.missing'))).toBeInTheDocument();
	});
});

describe('QrCode', () => {
	test('one path in currentColor with an accessible name', () => {
		const { container } = render(QrCode, {
			value: 'otpauth://totp/x?secret=ABC',
			label: 'QR',
			size: 160
		});
		const svg = screen.getByRole('img', { name: 'QR' });
		expect(svg).toHaveAttribute('width', '160');
		expect(container.querySelectorAll('path')).toHaveLength(1);
		expect(container.querySelector('path')).toHaveAttribute('fill', 'currentColor');
		expect(container.firstElementChild).toHaveClass('bg-mist-50', 'text-ink-950');
	});
});

describe('LineTabs', () => {
	test('marks the current tab, also for nested paths', () => {
		const items = [
			{ href: '/settings/profile', label: 'Profile' },
			{ href: '/settings/security', label: 'Security' }
		];
		render(LineTabs, { items, current: '/settings/security/x', label: 'Tabs' });
		expect(screen.getByRole('navigation', { name: 'Tabs' })).toBeInTheDocument();
		expect(screen.getByRole('link', { name: 'Security' })).toHaveAttribute('aria-current', 'page');
		expect(screen.getByRole('link', { name: 'Profile' })).not.toHaveAttribute('aria-current');
	});
});

describe('data table', () => {
	test('shell with caption and header cells', () => {
		render(DataTableShell, {
			caption: 'Users',
			children: html('<tbody><tr><td>1</td></tr></tbody>')
		});
		expect(screen.getByRole('table', { name: 'Users' })).toBeInTheDocument();
	});

	test('shell without caption', () => {
		const { container } = render(DataTableShell, { children: html('<tbody></tbody>') });
		expect(container.querySelector('caption')).toBeNull();
	});

	test('numeric header aligns right and carries the sort state', () => {
		const table = document.createElement('table');
		const row = table
			.appendChild(document.createElement('thead'))
			.appendChild(document.createElement('tr'));
		document.body.appendChild(table);
		render(DataTableHead, {
			target: row,
			props: { numeric: true, sort: 'descending', children: html('<span>Count</span>') }
		});
		render(DataTableHead, { target: row, props: {} });
		const [numeric, plain] = screen.getAllByRole('columnheader');
		expect(numeric).toHaveClass('text-right');
		expect(numeric).toHaveAttribute('aria-sort', 'descending');
		expect(plain).toHaveClass('text-left');
		table.remove();
	});
});

describe('KeyboardShortcut', () => {
	test('prefixes the platform modifier', () => {
		vi.spyOn(navigator, 'platform', 'get').mockReturnValue('Win32');
		render(KeyboardShortcut, { keys: ['K'], mod: true });
		expect(screen.getByText('Ctrl')).toBeInTheDocument();
		expect(screen.getByText('K')).toBeInTheDocument();
	});

	test('plain keys', () => {
		const { container } = render(KeyboardShortcut, { keys: ['?'] });
		expect(container.querySelectorAll('kbd kbd')).toHaveLength(1);
	});
});

describe('page frame pieces', () => {
	test('PageHeader with subtitle and actions', () => {
		render(PageHeader, { title: 'Tags', subtitle: '12', actions: html('<button>New</button>') });
		expect(screen.getByRole('heading', { level: 1, name: 'Tags' })).toBeInTheDocument();
		expect(screen.getByText('12')).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'New' })).toBeInTheDocument();
	});

	test('PageHeader title only', () => {
		const { container } = render(PageHeader, { title: 'X' });
		expect(container.querySelector('p')).toBeNull();
	});

	test('PageTitle sets the document title', async () => {
		render(PageTitle, { title: 'Tags' });
		await vi.waitFor(() => expect(document.title).toBe(`Tags – ${t('common.pageTitleSuffix')}`));
		render(PageTitle, {});
		await vi.waitFor(() => expect(document.title).toBe(t('common.pageTitleSuffix')));
	});

	test.each([
		[404, 'nav.errorPage.notFound.title'],
		[403, 'nav.errorPage.forbidden.title'],
		[500, 'errors.internal_error.title']
	] as const)('ErrorView %d', (status, key) => {
		render(ErrorView, { status });
		expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(t(key));
		expect(screen.getByText(String(status))).toBeInTheDocument();
		expect(screen.getByRole('link', { name: t('nav.errorPage.notFound.action') })).toHaveAttribute(
			'href',
			'/'
		);
	});
});

describe('breadcrumbs', () => {
	test.each([
		['/', ['nav.breadcrumb.collection', 'nav.links']],
		['/tags', ['nav.breadcrumb.collection', 'nav.tags']],
		['/settings/security', ['nav.breadcrumb.settings', 'nav.settingsTabs.security']],
		['/settings/profile/', ['nav.breadcrumb.settings', 'nav.settingsTabs.profile']],
		['/settings/display', ['nav.breadcrumb.settings', 'nav.settingsTabs.display']],
		['/settings/privacy', ['nav.breadcrumb.settings', 'nav.settingsTabs.privacy']],
		['/settings', ['nav.breadcrumb.settings']],
		['/admin/users', ['nav.breadcrumb.admin', 'nav.adminUsers']],
		['/admin/settings', ['nav.breadcrumb.admin', 'nav.adminSettings']],
		['/admin/audit', ['nav.breadcrumb.admin', 'nav.adminAudit']],
		['/admin', ['nav.breadcrumb.admin']],
		['/unknown', []]
	])('%s', (path, keys) => {
		expect(breadcrumbsFor(path).map((crumb) => crumb.key)).toEqual(keys);
	});

	test('pageTitleKey is the last crumb', () => {
		expect(pageTitleKey('/tags')).toBe('nav.tags');
		expect(pageTitleKey('/nope')).toBeNull();
	});
});

describe('ConfirmDialog', () => {
	test('confirms, closes and reports success', async () => {
		const onConfirm = vi.fn(async () => {});
		const { component } = render(ConfirmDialog, {
			open: true,
			title: 'Delete the link?',
			text: 'It disappears for everyone it was shared with.',
			confirmLabel: 'Delete link',
			destructive: true,
			onConfirm,
			children: html('<p>extra field</p>')
		});
		expect(await screen.findByRole('alertdialog')).toHaveTextContent('Delete the link?');
		expect(screen.getByText('extra field')).toBeInTheDocument();
		const confirm = screen.getByRole('button', { name: 'Delete link' });
		expect(confirm).toHaveClass('bg-destructive');
		await userEvent.click(confirm);
		expect(onConfirm).toHaveBeenCalledOnce();
		await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull());
		expect(component).toBeDefined();
	});

	test('a failing confirm keeps the dialog open with the error', async () => {
		const onConfirm = vi.fn(async () => {
			throw new ApiError(400, 'current_password_invalid');
		});
		render(ConfirmDialog, { open: true, title: 'Reset?', confirmLabel: 'Reset', onConfirm });
		await userEvent.click(await screen.findByRole('button', { name: 'Reset' }));
		expect(await screen.findByRole('alert')).toHaveTextContent(
			t('errors.current_password_invalid.title')
		);
		expect(screen.getByRole('alertdialog')).toBeInTheDocument();
	});

	test('cancel closes and calls back', async () => {
		const onCancel = vi.fn();
		render(ConfirmDialog, {
			open: true,
			title: 'Leave?',
			confirmLabel: 'Leave',
			cancelLabel: 'Stay',
			confirmDisabled: true,
			onConfirm: () => {},
			onCancel
		});
		expect(await screen.findByRole('button', { name: 'Leave' })).toBeDisabled();
		await userEvent.click(screen.getByRole('button', { name: 'Stay' }));
		expect(onCancel).toHaveBeenCalledOnce();
		await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull());
	});

	test('Escape closes through the dialog itself', async () => {
		const onCancel = vi.fn();
		render(ConfirmDialog, {
			open: true,
			title: 'Q',
			confirmLabel: 'OK',
			onConfirm: () => {},
			onCancel
		});
		await screen.findByRole('alertdialog');
		await userEvent.keyboard('{Escape}');
		await vi.waitFor(() => expect(onCancel).toHaveBeenCalled());
	});

	test('the default cancel label is translated', async () => {
		i18n.setLocale('en', { persist: false });
		render(ConfirmDialog, { open: true, title: 'Q', confirmLabel: 'OK', onConfirm: () => {} });
		expect(await screen.findByRole('button', { name: 'Cancel' })).toBeInTheDocument();
	});
});

describe('GRAPHICS.md over the nova defaults', () => {
	const label = html('<span>Save link</span>');
	const classes = (el: Element) => [...el.classList];

	test.each(['default', 'outline', 'secondary', 'ghost', 'destructive'] as const)(
		'%s button: radius-md, not the nova rounded-lg (5.1, 10.3)',
		(variant) => {
			render(Button, { variant, children: label });
			const button = screen.getByRole('button', { name: 'Save link' });
			expect(button).toHaveClass('rounded-md');
			expect(button).not.toHaveClass('rounded-lg');
		}
	);

	test('the primary hover shifts toward the foreground instead of fading (6.2)', () => {
		render(Button, { children: label });
		const button = screen.getByRole('button', { name: 'Save link' });
		expect(button).toHaveClass(
			'hover:bg-[color-mix(in_oklab,var(--primary)_88%,var(--foreground))]'
		);
		expect(classes(button)).not.toContain('hover:bg-primary/80');
	});

	test('the outline button stays transparent in the dark theme and hovers on accent (10.3)', () => {
		render(Button, { variant: 'outline', children: label });
		const button = screen.getByRole('button', { name: 'Save link' });
		// nova: bg-background with a dark-theme input/30 fill; only an important utility wins.
		expect(button).toHaveClass('bg-transparent!', 'hover:bg-accent!', 'aria-expanded:bg-accent!');
	});

	test('the tooltip is a popover surface with border and shadow (2.3, 5.4)', async () => {
		render(Hint, {
			text: 'Two uses',
			children: createRawSnippet((props: () => Record<string, unknown>) => ({
				render: () => '<button type="button">count</button>',
				setup: (node: Element) => {
					for (const [key, value] of Object.entries(props())) {
						if (typeof value === 'function') {
							node.addEventListener(key.replace(/^on/, ''), value as EventListener);
						} else if (value !== undefined && value !== null) {
							node.setAttribute(key, String(value));
						}
					}
				}
			}))
		});
		await fireEvent.focus(screen.getByRole('button', { name: 'count' }));
		const content = await vi.waitFor(() => {
			const el = document.querySelector('[data-slot="tooltip-content"]');
			if (!el) throw new Error('tooltip not open');
			return el;
		});
		expect(content).toHaveClass('bg-popover', 'text-popover-foreground', 'border', 'shadow-md');
		expect(classes(content)).not.toContain('bg-foreground');
		expect(classes(content)).not.toContain('text-background');
	});

	test('dialog: 200 ms motion, heading-3 title, body-sm text, a footer without the nova band (10.8)', async () => {
		render(ConfirmDialog, {
			open: true,
			title: 'Delete the link?',
			text: 'It disappears for everyone.',
			confirmLabel: 'Delete link',
			onConfirm: () => {}
		});
		const dialog = await screen.findByRole('alertdialog');
		expect(dialog).toHaveClass('duration-200', 'ease-out');
		expect(classes(dialog)).not.toContain('duration-100');
		const title = screen.getByRole('heading', { name: 'Delete the link?' });
		expect(title).toHaveClass('text-lg', 'leading-[26px]', 'font-medium');
		expect(classes(title)).not.toContain('text-base');
		const text = screen.getByText('It disappears for everyone.');
		expect(text).toHaveClass('text-[13px]', 'leading-5');
		expect(classes(text)).not.toContain('text-sm');
		const footer = dialog.querySelector('[data-slot="alert-dialog-footer"]')!;
		expect(footer).toHaveClass('mx-0', 'mb-0', 'bg-transparent', 'p-0', 'border-t-0');
		for (const nova of ['-mx-4', '-mb-4', 'bg-muted/50', 'border-t', 'p-4']) {
			expect(classes(footer)).not.toContain(nova);
		}
	});

	test('dialog: a long unbreakable word wraps inside the box, the footer stays reachable (E2)', async () => {
		const word = 'Rindfleischetikettierungsueberwachungsaufgabenuebertragungsgesetz'.repeat(4);
		render(ConfirmDialog, {
			open: true,
			title: 'Delete the link?',
			text: `The link "${word}" is deleted.`,
			confirmLabel: 'Delete link',
			onConfirm: () => {}
		});
		const dialog = await screen.findByRole('alertdialog');
		// One column that may shrink to 0 instead of nova's implicit auto column, shrinking
		// children, anywhere wrapping, and scrolling when taller than the window.
		expect(dialog).toHaveClass(
			'grid-cols-[minmax(0,1fr)]',
			'*:min-w-0',
			'wrap-anywhere',
			'max-h-[calc(100dvh-32px)]',
			'overflow-y-auto',
			'w-[min(440px,calc(100vw-32px))]'
		);
		expect(screen.getByRole('button', { name: 'Delete link' })).toBeInTheDocument();
	});

	test('invalid fields: the destructive border without the nova ring halo (6.2, 10.9)', () => {
		expect(inputSize.split(' ')).toEqual(
			expect.arrayContaining(['aria-invalid:ring-0', 'aria-invalid:border-destructive!'])
		);
		expect(buttonTone.outline.split(' ')).toEqual(
			expect.arrayContaining(['aria-invalid:ring-0', 'aria-invalid:border-destructive!'])
		);
		render(Input, { 'aria-invalid': 'true', class: inputSize, 'aria-label': 'Invalid field' });
		const input = screen.getByRole('textbox', { name: 'Invalid field' });
		expect(input).toHaveClass('aria-invalid:ring-0', 'aria-invalid:border-destructive!');
		expect(classes(input)).not.toContain('aria-invalid:ring-3');
	});

	test('toggle items: radius-md, also on the outer corners of a segmented group (5.1)', () => {
		render(ToggleGroupProbe);
		const items = screen.getAllByRole('radio');
		const first = items[0];
		const last = items[items.length - 1];
		expect(first).toHaveClass(
			'rounded-md',
			'group-data-horizontal/toggle-group:data-[spacing=0]:first:rounded-l-md'
		);
		expect(last).toHaveClass(
			'group-data-horizontal/toggle-group:data-[spacing=0]:last:rounded-r-md'
		);
		for (const item of items) {
			expect(classes(item)).not.toContain('rounded-lg');
			expect(classes(item)).not.toContain(
				'group-data-horizontal/toggle-group:data-[spacing=0]:first:rounded-l-lg'
			);
			expect(classes(item)).not.toContain(
				'group-data-horizontal/toggle-group:data-[spacing=0]:last:rounded-r-lg'
			);
		}
	});

	test('checkbox: radius-sm and transparent while unchecked in both themes (10.5)', () => {
		render(Checkbox, { class: checkboxBox, 'aria-label': 'Pin' });
		const box = screen.getByRole('checkbox', { name: 'Pin' });
		expect(box).toHaveClass('rounded-sm', 'data-unchecked:bg-transparent!');
		expect(classes(box)).not.toContain('rounded-[4px]');
	});
});
