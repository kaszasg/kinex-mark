// Administration (plan 8.7.0): the client-side gate spares the needless request and the flash;
// the server answers 403 anyway.
import { error } from '@sveltejs/kit';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ parent }) => {
	const { me } = await parent();
	if (!me.permissions.canAdmin) error(403, 'forbidden');
	return {};
};
