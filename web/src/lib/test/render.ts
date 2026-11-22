// Component rendering with the session and the locale prepared (plan 9.3.1).
import { render } from '@testing-library/svelte';
import type { Component, ComponentProps } from 'svelte';
import type { Locale, Me } from '$lib/api/types.js';
import { i18n } from '$lib/i18n/index.js';
import { session } from '$lib/state/session.svelte.js';

export type AppRenderOptions = { me?: Me | null; locale?: Locale };

/** Renders a component after signing in the given user (or none) and selecting the locale. */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function renderWithApp<C extends Component<any>>(
	component: C,
	props: ComponentProps<C> = {} as ComponentProps<C>,
	options: AppRenderOptions = {}
) {
	if (options.me) session.setMe(options.me);
	else session.clear();
	i18n.setLocale(options.locale ?? options.me?.preferences.locale ?? 'hu', { persist: false });
	// The generic Component type does not narrow to the testing-library import type.
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	return render(component as any, { props });
}
