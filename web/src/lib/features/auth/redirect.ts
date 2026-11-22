// Where the sign-in flow continues (plan 8.7.0, 8.7.2).
import type { Me } from '$lib/api/types.js';
import { safeNext } from '$lib/utils/navigation.js';

export type LoginReason = 'expired' | 'deleted';

/** The paths of the two gates, in the order the server enforces them. */
export const GATE_PATHS = { password: '/change-password', consent: '/consent' } as const;

/**
 * The page a signed-in user goes to: an open gate first (password, then the privacy notice),
 * otherwise the validated `next` target, otherwise the links page. A `next` pointing back to
 * the login page would loop, so it becomes "/".
 */
export function signedInTarget(
	me: Pick<Me, 'mustChangePassword' | 'privacyConsentRequired'>,
	next: string | null | undefined,
	origin: string
): string {
	if (me.mustChangePassword) return GATE_PATHS.password;
	if (me.privacyConsentRequired) return GATE_PATHS.consent;
	const target = safeNext(next, origin);
	const path = target.split(/[?#]/, 1)[0];
	return path === '/login' ? '/' : target;
}

/** The `reason` query parameter of the login URL; unknown values are ignored. */
export function parseLoginReason(value: string | null | undefined): LoginReason | null {
	return value === 'expired' || value === 'deleted' ? value : null;
}
