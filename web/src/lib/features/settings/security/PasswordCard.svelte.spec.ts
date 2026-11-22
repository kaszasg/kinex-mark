// Password card of Settings > Security (plan 8.7.6). The F1 form is replaced by a double.
import { screen, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { toast } from 'svelte-sonner';
import { describe, expect, test, vi } from 'vitest';
import { t } from '$lib/i18n/index.js';
import { makeMe } from '$lib/test/fixtures.js';
import { renderWithApp } from '$lib/test/render.js';
import PasswordCard from './PasswordCard.svelte';

vi.mock('$lib/features/auth/PasswordChangeForm.svelte', async () => ({
	default: (await import('./test-doubles/PasswordChangeFormDouble.svelte')).default
}));

describe('PasswordCard', () => {
	test('renders the shared form in the settings mode inside the card', async () => {
		const success = vi.spyOn(toast, 'success');
		renderWithApp(PasswordCard, {}, { me: makeMe() });
		const card = screen.getByRole('region', { name: t('security.password.title') });
		expect(within(card).getByText(t('security.password.text'))).toBeInTheDocument();
		const form = within(card).getByTestId('password-change-form');
		expect(form).toHaveAttribute('data-mode', 'settings');
		// The form confirms a change itself; the card adds no second toast.
		await userEvent.click(form);
		expect(success).not.toHaveBeenCalled();
	});
});
