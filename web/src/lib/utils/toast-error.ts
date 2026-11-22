// The error toast of a failed action (plan 8.7.0).
import { toast } from 'svelte-sonner';
import { ApiError } from '$lib/api/errors.js';
import { errorMessage } from '$lib/i18n/index.js';

/**
 * Shows a failed request as an error toast. A non-API exception counts as a network error. An
 * error the session handler took over (expired session, open gate) raises no toast: the sign-in
 * or gate page already says what happened (plan 8.7.0), and a second message would repeat it.
 */
export function toastError(err: unknown, message?: string): void {
	const apiError = err instanceof ApiError ? err : new ApiError(0, 'network_error');
	if (apiError.authHandled) return;
	toast.error(message ?? errorMessage(apiError));
}
