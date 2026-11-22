// Gate pages (plan 8.7.0): only for a signed-in user with an open gate, password gate first.
import { redirect } from '@sveltejs/kit';
import { session } from '$lib/state/session.svelte.js';
import type { LayoutLoad } from './$types';

const GATE_PATHS = { password: '/change-password', consent: '/consent' } as const;

export const load: LayoutLoad = async ({ url }) => {
	const me = await session.ensure();
	if (!me) redirect(307, '/login');
	const gate = session.gate;
	if (!gate) redirect(307, '/');
	const target = GATE_PATHS[gate];
	if (url.pathname !== target) redirect(307, target);
	return { me };
};
