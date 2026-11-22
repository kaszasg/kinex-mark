// Page navigation of the admin tables: range text, translated labels, page change.
import { render, screen } from '@testing-library/svelte';
import { describe, expect, test, vi } from 'vitest';
import { i18n, t } from '$lib/i18n/index.js';
import Pager from './Pager.svelte';
import { user } from './test-helpers.js';

describe('Pager', () => {
	test('one page: only the range, no navigation', () => {
		render(Pager, { page: 1, total: 12, onchange: vi.fn() });
		expect(screen.getByText('1–12 / 12')).toBeInTheDocument();
		expect(screen.queryByRole('navigation')).toBeNull();
	});

	test('empty result shows a zero range', () => {
		render(Pager, { page: 1, total: 0, onchange: vi.fn() });
		expect(screen.getByText('0–0 / 0')).toBeInTheDocument();
	});

	test('several pages: translated labels and page changes', async () => {
		const onchange = vi.fn();
		render(Pager, { page: 2, total: 132, onchange });
		expect(screen.getByText('51–100 / 132')).toBeInTheDocument();
		const nav = screen.getByRole('navigation', { name: t('admin.pager.label') });
		expect(nav).toBeInTheDocument();
		const current = screen.getByRole('button', { name: t('admin.pager.page', { page: 2 }) });
		expect(current).toHaveAttribute('aria-current', 'page');
		const u = user();
		await u.click(screen.getByRole('button', { name: t('admin.pager.next') }));
		expect(onchange).toHaveBeenLastCalledWith(3);
		await u.click(screen.getByRole('button', { name: t('admin.pager.previous') }));
		await u.click(screen.getByRole('button', { name: t('admin.pager.page', { page: 1 }) }));
		expect(onchange).toHaveBeenCalledWith(1);
	});

	test('many pages collapse into an ellipsis and numbers use the locale', () => {
		i18n.setLocale('en', { persist: false });
		render(Pager, { page: 1, total: 12480, size: 50, onchange: vi.fn() });
		expect(screen.getByText('1–50 / 12,480')).toBeInTheDocument();
		expect(screen.getByText('…')).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'Page 250' })).toBeInTheDocument();
	});
});
