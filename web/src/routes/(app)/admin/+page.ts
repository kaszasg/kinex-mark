// /admin opens the users page (plan 8.7.0).
import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = () => {
	redirect(307, '/admin/users');
};
