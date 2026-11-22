// Step-up confirmation: the password is required, a wrong one is shown under the field.
import { render, screen } from '@testing-library/svelte';
import { describe, expect, test, vi } from 'vitest';
import { ApiError } from '$lib/api/errors.js';
import { errorMessage, errorTitle, t } from '$lib/i18n/index.js';
import StepUpDialog from './StepUpDialog.svelte';
import { user } from './test-helpers.js';

const base = {
	open: true,
	title: 'Reset the password?',
	text: 'All sessions end.',
	confirmLabel: 'Reset password'
};

const passwordField = () => screen.getByLabelText(t('common.stepUp.label'));
const confirmButton = () => screen.getByRole('button', { name: base.confirmLabel });

describe('StepUpDialog', () => {
	test('the confirm button waits for the password, then runs with it', async () => {
		const onConfirm = vi.fn(async () => undefined);
		render(StepUpDialog, { ...base, onConfirm });
		expect(screen.getByRole('alertdialog', { name: base.title })).toBeInTheDocument();
		expect(screen.getByText(t('common.stepUp.help'))).toBeInTheDocument();
		expect(confirmButton()).toBeDisabled();
		const u = user();
		await u.type(passwordField(), 'Admin-Passphrase-1');
		expect(confirmButton()).toBeEnabled();
		await u.click(confirmButton());
		expect(onConfirm).toHaveBeenCalledWith('Admin-Passphrase-1');
		await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull());
	});

	test('current_password_invalid stays under the field and keeps the dialog open', async () => {
		const err = new ApiError(400, 'current_password_invalid', { currentPassword: 'incorrect' });
		const onConfirm = vi.fn().mockRejectedValueOnce(err).mockResolvedValueOnce(undefined);
		render(StepUpDialog, { ...base, onConfirm });
		const u = user();
		await u.type(passwordField(), 'wrong');
		await u.click(confirmButton());
		await vi.waitFor(() => expect(passwordField()).toHaveAttribute('aria-invalid', 'true'));
		expect(screen.getByText(errorMessage(err))).toBeInTheDocument();
		expect(screen.queryByRole('alert')).toBeNull();
		expect(screen.getByRole('alertdialog')).toBeInTheDocument();
		await u.clear(passwordField());
		await u.type(passwordField(), 'right');
		await u.click(confirmButton());
		await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull());
		expect(onConfirm).toHaveBeenLastCalledWith('right');
	});

	test('a field code on currentPassword is shown with the field text', async () => {
		const onConfirm = vi
			.fn()
			.mockRejectedValue(new ApiError(400, 'validation_failed', { currentPassword: 'required' }));
		render(StepUpDialog, { ...base, onConfirm });
		const u = user();
		await u.type(passwordField(), 'x');
		await u.click(confirmButton());
		expect(await screen.findByText(t('fields.required'))).toBeInTheDocument();
		expect(screen.getByRole('alertdialog')).toBeInTheDocument();
	});

	test('other errors appear as an alert in the dialog', async () => {
		const err = new ApiError(409, 'last_admin');
		render(StepUpDialog, { ...base, destructive: true, onConfirm: vi.fn().mockRejectedValue(err) });
		const u = user();
		await u.type(passwordField(), 'x');
		await u.click(confirmButton());
		expect(await screen.findByRole('alert')).toHaveTextContent(errorTitle(err));
		expect(passwordField()).not.toHaveAttribute('aria-invalid');
	});

	test('a non-API failure also stays an alert', async () => {
		render(StepUpDialog, { ...base, onConfirm: vi.fn().mockRejectedValue(new Error('boom')) });
		const u = user();
		await u.type(passwordField(), 'x');
		await u.click(confirmButton());
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.internal_error.title'));
	});

	test('cancel closes and clears the password', async () => {
		const onConfirm = vi.fn();
		const { rerender } = render(StepUpDialog, { ...base, onConfirm });
		const u = user();
		await u.type(passwordField(), 'secret');
		await u.click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull());
		await rerender({ open: true });
		expect(passwordField()).toHaveValue('');
		expect(onConfirm).not.toHaveBeenCalled();
	});
});
