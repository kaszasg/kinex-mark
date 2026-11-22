// Admin select wrapper: shows the chosen label and reports a new choice.
import { render, screen } from '@testing-library/svelte';
import { describe, expect, test, vi } from 'vitest';
import AdminSelect from './AdminSelect.svelte';
import { findFloating, user } from './test-helpers.js';

const groups = [
	{ options: [{ value: 'all', label: 'Every role' }] },
	{
		heading: 'Roles',
		options: [
			{ value: 'admin', label: 'Administrator' },
			{ value: 'user.created', label: 'user.created', mono: true }
		]
	}
];

describe('AdminSelect', () => {
	test('shows the selected label and reports a different choice', async () => {
		const onchange = vi.fn();
		render(AdminSelect, { value: 'all', groups, onchange, 'aria-label': 'Role filter' });
		const trigger = screen.getByRole('combobox', { name: 'Role filter' });
		expect(trigger).toHaveTextContent('Every role');
		const u = user();
		await u.click(trigger);
		expect(await findFloating('group', 'Roles', { exact: false })).toBeTruthy();
		await u.click(await findFloating('option', 'Administrator'));
		expect(onchange).toHaveBeenCalledWith('admin');
	});

	test('the trigger is as high as the inputs beside it (36 px)', () => {
		render(AdminSelect, { value: 'all', groups, onchange: vi.fn(), 'aria-label': 'Role filter' });
		const trigger = screen.getByRole('combobox', { name: 'Role filter' });
		// The trigger's default size is data-[size=default]:h-8; the override must use the same variant.
		expect(trigger).toHaveAttribute('data-size', 'default');
		expect(trigger).toHaveClass('data-[size=default]:h-9');
		expect(trigger).not.toHaveClass('data-[size=default]:h-8');
	});

	test('does not report the current value again and renders mono labels', async () => {
		const onchange = vi.fn();
		render(AdminSelect, { id: 'event', value: 'user.created', groups, onchange });
		const trigger = screen.getByRole('combobox');
		expect(trigger).toHaveAttribute('id', 'event');
		expect(trigger.querySelector('.font-mono')).toHaveTextContent('user.created');
		const u = user();
		await u.click(trigger);
		await u.click(await findFloating('option', 'user.created'));
		expect(onchange).not.toHaveBeenCalled();
	});

	test('shows nothing for an unknown value', () => {
		render(AdminSelect, { value: 'nope', groups, onchange: vi.fn(), disabled: true });
		const trigger = screen.getByRole('combobox');
		expect(trigger).toHaveTextContent('');
		expect(trigger).toBeDisabled();
	});
});
