// Brand components of GRAPHICS.md 10.13-10.19.
import { render, screen } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { describe, expect, test } from 'vitest';
import { i18n, t } from '$lib/i18n/index.js';
import Logo from './Logo.svelte';
import Metric from './Metric.svelte';
import Panel from './Panel.svelte';
import StatusBadge from './StatusBadge.svelte';
import TickRule from './TickRule.svelte';

const text = (value: string) => createRawSnippet(() => ({ render: () => `<span>${value}</span>` }));

describe('Logo', () => {
	test.each([
		['mark', '0 0 64 64'],
		['lockup', '0 0 232.6 64'],
		['wordmark', '0 0 513.7 71.7']
	] as const)('%s variant', (variant, box) => {
		render(Logo, { variant, height: 28, class: 'text-foreground' });
		const svg = screen.getByRole('img', { name: 'Kinexium' });
		expect(svg).toHaveAttribute('viewBox', box);
		expect(svg).toHaveAttribute('height', '28');
	});

	test('defaults to the lockup', () => {
		render(Logo);
		expect(screen.getByRole('img')).toHaveAttribute('viewBox', '0 0 232.6 64');
	});
});

describe('Panel and Metric', () => {
	test.each(['sm', 'md', 'lg'] as const)('panel %s', (size) => {
		const { container } = render(Panel, { size, class: 'extra', children: text('inside') });
		const panel = container.querySelector('.kx-panel');
		expect(panel).toHaveClass('extra');
		if (size !== 'md') expect(panel).toHaveClass(`kx-panel-${size}`);
		expect(screen.getByText('inside')).toBeInTheDocument();
	});

	test('panel defaults to md', () => {
		const { container } = render(Panel, { children: text('x') });
		expect(container.querySelector('.kx-panel')?.className).not.toContain('kx-panel-sm');
	});

	test('metric shows label, value, unit and note', () => {
		render(Metric, { label: 'Links', value: '1284', unit: 'db', note: 'active' });
		expect(screen.getByText('Links')).toHaveClass('kx-eyebrow');
		expect(screen.getByText('db')).toHaveClass('tracking-normal');
		expect(screen.getByText('active')).toBeInTheDocument();
	});

	test('metric without unit and note', () => {
		const { container } = render(Metric, { label: 'Links', value: '7' });
		expect(container.querySelector('small')).toBeNull();
		expect(screen.getByText('7')).toBeInTheDocument();
	});
});

describe('StatusBadge', () => {
	test.each(['online', 'warning', 'critical', 'offline', 'info'] as const)('%s word', (status) => {
		render(StatusBadge, { status });
		expect(screen.getByText(t(`status.${status}`))).toBeInTheDocument();
	});

	test('follows the locale and accepts a custom label', () => {
		i18n.setLocale('en', { persist: false });
		render(StatusBadge, { status: 'online', label: 'On' });
		expect(screen.getByText('On')).toBeInTheDocument();
	});
});

describe('TickRule', () => {
	test('is decorative', () => {
		const { container } = render(TickRule, { class: 'my-2' });
		const rule = container.querySelector('.kx-ticks');
		expect(rule).toHaveAttribute('aria-hidden', 'true');
		expect(rule).toHaveClass('my-2');
	});
});
