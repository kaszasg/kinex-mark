// /settings opens the first tab (plan 8.7.0).
import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = () => {
	redirect(307, '/settings/profile');
};
