// The error toast of a failed action, and the expired session (plan 8.7.0, E2).
import { toast } from 'svelte-sonner';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { onAuthFailure } from '$lib/api/client.js';
import { ApiError } from '$lib/api/errors.js';
import { errorMessage } from '$lib/i18n/index.js';
import { LinksStore } from '$lib/state/links.svelte.js';
import { makeLink } from '$lib/test/fixtures.js';
import { apiError, mockApi } from '$lib/test/mock-api.js';
import { toastError } from './toast-error.js';

vi.mock('svelte-sonner', () => ({
	toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn(), info: vi.fn() })
}));

afterEach(() => {
	vi.mocked(toast.error).mockClear();
	vi.unstubAllGlobals();
});

describe('toastError', () => {
	test('shows the localized message of an API error, or a given message', () => {
		const err = new ApiError(409, 'link_duplicate');
		toastError(err);
		expect(toast.error).toHaveBeenLastCalledWith(errorMessage(err));
		toastError(err, 'custom text');
		expect(toast.error).toHaveBeenLastCalledWith('custom text');
	});

	test('a non-API exception counts as a network error', () => {
		toastError(new Error('offline'));
		expect(toast.error).toHaveBeenLastCalledWith(errorMessage(new ApiError(0, 'network_error')));
	});

	test('an error the session handler took over raises no toast', () => {
		const err = new ApiError(401, 'unauthenticated');
		err.authHandled = true;
		toastError(err);
		expect(toast.error).not.toHaveBeenCalled();
	});
});

describe('an action during an expired session', () => {
	test('pinning shows only the sign-in page message, not an error toast as well', async () => {
		const handler = vi.fn();
		onAuthFailure(handler);
		try {
			mockApi({ 'POST /api/links/1/pin': apiError(401, 'unauthenticated') });
			const store = new LinksStore();
			store.items = [makeLink({ id: 1 })];
			await expect(store.pin(1, true)).resolves.toBeNull();
			expect(handler).toHaveBeenCalledWith('unauthenticated');
			expect(toast.error).not.toHaveBeenCalled();
			// The optimistic change is rolled back.
			expect(store.find(1)?.pinned).toBe(false);
		} finally {
			onAuthFailure(null);
		}
	});

	test('the same failure without a session handler still shows the toast', async () => {
		onAuthFailure(null);
		mockApi({ 'POST /api/links/1/pin': apiError(500, 'internal_error') });
		const store = new LinksStore();
		store.items = [makeLink({ id: 1 })];
		await store.pin(1, true);
		expect(toast.error).toHaveBeenCalledOnce();
	});
});
