// Actor filter of the audit log: loading, failure, search, choice and unknown ids.
import { render, screen } from '@testing-library/svelte';
import { describe, expect, test, vi } from 'vitest';
import { t } from '$lib/i18n/index.js';
import ActorPicker, { type Actor } from './ActorPicker.svelte';
import { findFloating, queryFloating, user } from './test-helpers.js';

const actors: Actor[] = [
	{ id: 1, username: 'kanna', displayName: 'Kovacs Anna' },
	{ id: 2, username: 'peter', displayName: '' }
];

const trigger = () => document.getElementById('actor') as HTMLElement;

describe('ActorPicker', () => {
	test('shows "All" and lists the actors; choosing one reports its id', async () => {
		const onchange = vi.fn();
		render(ActorPicker, { id: 'actor', value: null, actors, onchange });
		expect(trigger()).toHaveTextContent(t('audit.filters.all'));
		const u = user();
		await u.click(trigger());
		await u.click(await findFloating('option', 'kanna Kovacs Anna'));
		expect(onchange).toHaveBeenCalledWith(1);
	});

	test('the search narrows by username and display name', async () => {
		render(ActorPicker, { id: 'actor', value: null, actors, onchange: vi.fn() });
		const u = user();
		await u.click(trigger());
		// The search box is named by the command's own label; floating content is hidden in jsdom.
		const search = await vi.waitFor(() => screen.getByRole('combobox', { hidden: true }));
		const labelId = search.getAttribute('aria-labelledby') ?? '';
		expect(document.getElementById(labelId)).toHaveTextContent(t('audit.filters.actorSearch'));
		expect(screen.getByRole('listbox', { hidden: true })).toHaveAttribute(
			'aria-label',
			t('audit.filters.actor')
		);
		await u.type(search, 'kovacs');
		await vi.waitFor(() => expect(queryFloating('option', 'peter')).toBeNull());
		expect(queryFloating('option', t('audit.filters.all'))).toBeNull();
		expect(queryFloating('option', 'kanna Kovacs Anna')).not.toBeNull();
		await u.clear(search);
		await u.type(search, 'zzz');
		await vi.waitFor(() =>
			expect(screen.getByText(t('audit.filters.actorEmpty'))).toBeInTheDocument()
		);
	});

	test('the selected actor is shown and "All" clears it', async () => {
		const onchange = vi.fn();
		render(ActorPicker, { id: 'actor', value: 2, actors, onchange });
		expect(trigger()).toHaveTextContent('peter');
		const u = user();
		await u.click(trigger());
		await u.click(await findFloating('option', 'peter'));
		expect(onchange).not.toHaveBeenCalled();
		await u.click(trigger());
		await u.click(await findFloating('option', t('audit.filters.all')));
		expect(onchange).toHaveBeenCalledWith(null);
	});

	test('an id outside the list is named by its number', () => {
		render(ActorPicker, { id: 'actor', value: 99, actors, onchange: vi.fn() });
		expect(trigger()).toHaveTextContent(t('audit.filters.actorUnknown', { id: '99' }));
	});

	test('loading and failure states inside the list', async () => {
		const { rerender } = render(ActorPicker, {
			id: 'actor',
			value: null,
			actors: null,
			onchange: vi.fn()
		});
		const u = user();
		await u.click(trigger());
		await vi.waitFor(() =>
			expect(
				screen.getByLabelText(t('common.states.loading'), { selector: '[role="status"]' })
			).toBeInTheDocument()
		);
		await rerender({ failed: true });
		expect(screen.getByText(t('audit.filters.actorUnavailable'))).toBeInTheDocument();
	});
});
