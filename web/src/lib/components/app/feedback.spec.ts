// Alerts, field errors, form fields, empty and loading states (plan 8.4).
import { render, screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { createRawSnippet } from 'svelte';
import { toast } from 'svelte-sonner';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { ApiError } from '$lib/api/errors.js';
import { i18n, t } from '$lib/i18n/index.js';
import AppToaster from './AppToaster.svelte';
import EmptyState from './EmptyState.svelte';
import ErrorAlert from './ErrorAlert.svelte';
import FieldError from './FieldError.svelte';
import FormField, { type FieldControlProps } from './FormField.svelte';
import LoadingRows from './LoadingRows.svelte';
import PasswordInput from './PasswordInput.svelte';
import StateAlert from './StateAlert.svelte';

afterEach(() => {
	vi.useRealTimers();
});

const button = (label: string) =>
	createRawSnippet(() => ({ render: () => `<button type="button">${label}</button>` }));

describe('StateAlert', () => {
	test('only a critical alert is announced as an alert', () => {
		const { container } = render(StateAlert, {
			kind: 'critical',
			title: 'Broken',
			text: 'Retry later'
		});
		expect(screen.getByRole('alert')).toHaveTextContent('Broken');
		expect(screen.getByText('Retry later')).toBeInTheDocument();
		expect(container.querySelector('[data-kind="critical"]')).toHaveClass('grid-cols-[16px_1fr]');
	});

	test.each(['warning', 'success', 'info'] as const)('%s has no alert role', (kind) => {
		render(StateAlert, { kind, title: 'Note' });
		expect(screen.queryByRole('alert')).toBeNull();
		expect(screen.getByText('Note')).toBeInTheDocument();
	});

	test('action button calls back and can be disabled', async () => {
		const onAction = vi.fn();
		const { rerender } = render(StateAlert, {
			kind: 'warning',
			title: 'Few codes',
			actionLabel: 'Generate',
			onAction
		});
		await userEvent.click(screen.getByRole('button', { name: 'Generate' }));
		expect(onAction).toHaveBeenCalledOnce();
		await rerender({ actionDisabled: true });
		expect(screen.getByRole('button', { name: 'Generate' })).toBeDisabled();
	});

	test('an action snippet replaces the button', () => {
		const { container } = render(StateAlert, {
			kind: 'info',
			title: 'x',
			action: button('Custom')
		});
		expect(screen.getByRole('button', { name: 'Custom' })).toBeInTheDocument();
		expect(container.firstElementChild).toHaveClass('grid-cols-[16px_1fr_auto]');
	});
});

describe('ErrorAlert', () => {
	test('shows the error texts and a retry button', async () => {
		const onRetry = vi.fn();
		render(ErrorAlert, { error: new ApiError(404, 'not_found'), onRetry });
		expect(screen.getByRole('alert')).toHaveTextContent(t('errors.not_found.title'));
		await userEvent.click(screen.getByRole('button', { name: t('common.actions.retry') }));
		expect(onRetry).toHaveBeenCalledOnce();
	});

	test('a non-API error reads as internal_error without retry', () => {
		render(ErrorAlert, { error: new Error('boom') });
		expect(screen.getByRole('alert')).toHaveTextContent(t('errors.internal_error.title'));
		expect(screen.queryByRole('button')).toBeNull();
	});
});

describe('FieldError', () => {
	test('per-field text with the id for aria-describedby', () => {
		render(FieldError, { id: 'url-error', field: 'url', code: 'invalid_format' });
		expect(screen.getByText(t('fields.byField.url.invalid_format')).parentElement).toHaveAttribute(
			'id',
			'url-error'
		);
	});

	test('a ready message wins; nothing renders without code or message', () => {
		const { container, rerender } = render(FieldError, {
			id: 'e',
			code: 'required',
			message: 'Custom'
		});
		expect(screen.getByText('Custom')).toBeInTheDocument();
		rerender({ message: null, code: null });
		return Promise.resolve().then(() => expect(container.querySelector('p')).toBeNull());
	});
});

describe('FormField', () => {
	const control = createRawSnippet((props: () => FieldControlProps) => ({
		render: () => {
			const p = props();
			return `<input id="${p.id}" aria-invalid="${p['aria-invalid'] ?? ''}" aria-describedby="${p['aria-describedby'] ?? ''}" />`;
		}
	}));

	test('joins label, help and error', () => {
		render(FormField, {
			id: 'name',
			label: 'Name',
			help: 'Shown to others',
			field: 'displayName',
			errorCode: 'too_long',
			children: control
		});
		const input = screen.getByLabelText('Name');
		expect(input).toHaveAttribute('aria-invalid', 'true');
		expect(input).toHaveAttribute('aria-describedby', 'name-help name-error');
		expect(screen.getByText(t('fields.too_long'))).toBeInTheDocument();
	});

	test('a valid field has no description', () => {
		render(FormField, { id: 'plain', label: 'Plain', children: control });
		const input = screen.getByLabelText('Plain');
		expect(input).toHaveAttribute('aria-invalid', '');
		expect(input).toHaveAttribute('aria-describedby', '');
	});

	test('an error message also marks the field invalid', () => {
		render(FormField, { id: 'm', label: 'M', errorMessage: 'Passwords differ', children: control });
		expect(screen.getByLabelText('M')).toHaveAttribute('aria-describedby', 'm-error');
	});
});

describe('PasswordInput', () => {
	test('toggles the visibility with aria-pressed', async () => {
		render(PasswordInput, { id: 'pw', 'aria-label': 'Password', value: 'secret' });
		const input = screen.getByLabelText('Password');
		expect(input).toHaveAttribute('type', 'password');
		const toggle = screen.getByRole('button', { name: t('common.password.show') });
		expect(toggle).toHaveAttribute('aria-pressed', 'false');
		await userEvent.click(toggle);
		expect(input).toHaveAttribute('type', 'text');
		expect(screen.getByRole('button', { name: t('common.password.hide') })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
	});

	test('disabled disables both parts', () => {
		render(PasswordInput, { 'aria-label': 'Password', disabled: true });
		expect(screen.getByLabelText('Password')).toBeDisabled();
		expect(screen.getByRole('button')).toBeDisabled();
	});
});

describe('EmptyState', () => {
	test('title, text and action', () => {
		render(EmptyState, { title: 'No links yet', text: 'Paste a URL', action: button('Add link') });
		expect(screen.getByText('No links yet')).toBeInTheDocument();
		expect(screen.getByText('Paste a URL')).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'Add link' })).toBeInTheDocument();
	});

	test('title only', () => {
		render(EmptyState, { title: 'Nothing' });
		expect(screen.queryByRole('button')).toBeNull();
	});
});

describe('LoadingRows', () => {
	test('the skeleton appears only after the delay', async () => {
		vi.useFakeTimers();
		render(LoadingRows, { rows: 3 });
		expect(screen.getByRole('status')).toHaveTextContent(t('common.states.loading'));
		expect(screen.queryAllByTestId('loading-row')).toHaveLength(0);
		await vi.advanceTimersByTimeAsync(150);
		expect(screen.getAllByTestId('loading-row')).toHaveLength(3);
	});

	test('zero delay shows the rows at once', async () => {
		render(LoadingRows, { delay: 0 });
		await Promise.resolve();
		expect(screen.getAllByTestId('loading-row')).toHaveLength(5);
	});
});

describe('AppToaster', () => {
	afterEach(() => {
		i18n.setLocale('hu', { persist: false });
	});

	test('the toast region is named in the active language', async () => {
		i18n.setLocale('de', { persist: false });
		const { container } = render(AppToaster);
		const region = container.ownerDocument.querySelector('section[aria-live="polite"]')!;
		expect(region.getAttribute('aria-label')).toMatch(/^Benachrichtigungen /);
		i18n.setLocale('hu', { persist: false });
		await vi.waitFor(() =>
			expect(region.getAttribute('aria-label')).toMatch(
				new RegExp(`^${t('common.notifications')} `)
			)
		);
		expect(region.getAttribute('aria-label')).not.toMatch(/Notifications/);
	});

	test.each([
		['success', 'text-success'],
		['info', 'text-info'],
		['warning', 'text-warning'],
		['error', 'text-destructive']
	] as const)(
		'a %s toast shows its icon in the status colour (plan 8.7.0)',
		async (type, colour) => {
			render(AppToaster);
			toast[type](`${type} message`);
			const item = await vi.waitFor(() => {
				const found = document.querySelector<HTMLElement>(
					`[data-sonner-toast][data-type="${type}"]`
				);
				if (!found) throw new Error('no toast yet');
				return found;
			});
			expect(item.querySelector('[data-icon] svg')).not.toBeNull();
			expect(item).toHaveClass(`[&_[data-icon]]:${colour}`, 'cn-toast');
			toast.dismiss();
		}
	);
});
