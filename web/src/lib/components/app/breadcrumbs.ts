// Breadcrumb trail of the top bar (plan 8.7.1): "Collection > Links", "Settings > Security".
import type { MessageKey } from '$lib/i18n/index.js';

export type Crumb = { key: MessageKey; href?: string };

const TRAILS: [RegExp, Crumb[]][] = [
	[/^\/$/, [{ key: 'nav.breadcrumb.collection' }, { key: 'nav.links' }]],
	[/^\/tags\/?$/, [{ key: 'nav.breadcrumb.collection' }, { key: 'nav.tags' }]],
	[
		/^\/settings\/profile\/?$/,
		[{ key: 'nav.breadcrumb.settings', href: '/settings' }, { key: 'nav.settingsTabs.profile' }]
	],
	[
		/^\/settings\/security\/?$/,
		[{ key: 'nav.breadcrumb.settings', href: '/settings' }, { key: 'nav.settingsTabs.security' }]
	],
	[
		/^\/settings\/display\/?$/,
		[{ key: 'nav.breadcrumb.settings', href: '/settings' }, { key: 'nav.settingsTabs.display' }]
	],
	[
		/^\/settings\/privacy\/?$/,
		[{ key: 'nav.breadcrumb.settings', href: '/settings' }, { key: 'nav.settingsTabs.privacy' }]
	],
	[/^\/settings\/?$/, [{ key: 'nav.breadcrumb.settings' }]],
	[/^\/admin\/users\/?$/, [{ key: 'nav.breadcrumb.admin' }, { key: 'nav.adminUsers' }]],
	[/^\/admin\/settings\/?$/, [{ key: 'nav.breadcrumb.admin' }, { key: 'nav.adminSettings' }]],
	[/^\/admin\/audit\/?$/, [{ key: 'nav.breadcrumb.admin' }, { key: 'nav.adminAudit' }]],
	[/^\/admin\/?$/, [{ key: 'nav.breadcrumb.admin' }]]
];

/** The crumbs of a path; an unknown path has none. The last crumb is the current page. */
export function breadcrumbsFor(pathname: string): Crumb[] {
	for (const [pattern, trail] of TRAILS) if (pattern.test(pathname)) return trail;
	return [];
}

/** Message key of the page title of a path (the last crumb), or null for an unknown path. */
export function pageTitleKey(pathname: string): MessageKey | null {
	const trail = breadcrumbsFor(pathname);
	return trail.length > 0 ? trail[trail.length - 1].key : null;
}
