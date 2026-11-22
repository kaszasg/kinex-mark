// Any other path of a signed-in user is a 404 inside the AppShell (plan 8.7.11): matching it here
// runs the (app) layout, so (app)/+error.svelte renders within the shell instead of the bare root
// error page. A signed-out visitor is sent to the sign-in by the layout first.
import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = () => {
	error(404, 'Not Found');
};
