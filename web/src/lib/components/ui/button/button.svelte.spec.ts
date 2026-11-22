// Scaffold probe: a generated shadcn-svelte component renders and reacts in the jsdom test setup.
import { render, screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { createRawSnippet } from 'svelte';
import { describe, expect, it, vi } from 'vitest';
import { Button } from './index.js';

const label = createRawSnippet(() => ({ render: () => '<span>Probe</span>' }));

describe('Button (generated)', () => {
	it('renders a button with its data-slot and handles a click', async () => {
		const onclick = vi.fn();
		render(Button, { props: { onclick, children: label } });

		const button = screen.getByRole('button', { name: 'Probe' });
		expect(button).toHaveAttribute('data-slot', 'button');
		expect(button).toHaveAttribute('type', 'button');

		await userEvent.setup().click(button);
		expect(onclick).toHaveBeenCalledTimes(1);
	});

	it('renders a link when href is set and blocks it when disabled', () => {
		render(Button, { props: { href: '/tags', disabled: true, children: label } });

		const link = screen.getByRole('link', { name: 'Probe' });
		expect(link).not.toHaveAttribute('href');
		expect(link).toHaveAttribute('aria-disabled', 'true');
	});
});
