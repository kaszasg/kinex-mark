import { redirect } from '@sveltejs/kit';
import { parseLoginReason, signedInTarget } from '$lib/features/auth/redirect.js';
import { session } from '$lib/state/session.svelte.js';
import type { PageLoad } from './$types';

/**
 * A signed-in user does not see the login page: the open gate, the `next` target or "/" follows
 * (plan 8.7.0). An unreachable server still shows the form, which then reports the error.
 */
export const load: PageLoad = async ({ url }) => {
	const next = url.searchParams.get('next');
	const me = await session.ensure().catch(() => null);
	if (me) redirect(307, signedInTarget(me, next, url.origin));
	return { next, reason: parseLoginReason(url.searchParams.get('reason')) };
};
