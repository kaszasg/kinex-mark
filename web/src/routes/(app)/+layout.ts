// Signed-in application (plan 8.7.0): session probe, gates, background loads.
import { redirect } from '@sveltejs/kit';
import { publicConfig } from '$lib/state/public-config.svelte.js';
import { session } from '$lib/state/session.svelte.js';
import { tags } from '$lib/state/tags.svelte.js';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ url }) => {
	const me = await session.ensure();
	if (!me) redirect(307, '/login?next=' + encodeURIComponent(url.pathname + url.search));
	if (session.gate === 'password') redirect(307, '/change-password');
	if (session.gate === 'consent') redirect(307, '/consent');
	// Not awaited: the shell renders while the tags and the public configuration load.
	void tags.ensure(me.id);
	void publicConfig.ensure();
	return { me };
};
