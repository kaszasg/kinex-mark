import type hu from '../hu/nav.js';
import type { Shape } from '../../types.js';

export default {
	groups: {
		collection: 'Collection',
		frequentTags: 'Frequent tags',
		account: 'Account',
		admin: 'Administration'
	},
	links: 'Links',
	tags: 'Tags',
	settings: 'Settings',
	settingsTabs: {
		label: 'Settings tabs',
		profile: 'Profile',
		security: 'Security',
		display: 'Display',
		privacy: 'Privacy'
	},
	adminUsers: 'Users',
	adminSettings: 'System settings',
	adminAudit: 'Audit log',
	logout: 'Sign out',
	toggleSidebar: 'Open the sidebar',
	sidebarTitle: 'Navigation',
	breadcrumb: {
		label: 'Breadcrumb',
		collection: 'Collection',
		admin: 'Administration',
		settings: 'Settings'
	},
	search: {
		label: 'Search links',
		placeholder: 'Search…',
		open: 'Open search',
		close: 'Close search'
	},
	theme: {
		toggle: 'Switch theme'
	},
	locale: {
		label: 'Language',
		hu: 'Magyar',
		en: 'English',
		de: 'Deutsch'
	},
	userMenu: '{name} {role}, user menu',
	privacyNotice: 'Privacy notice',
	errorPage: {
		notFound: {
			title: 'Page not found.',
			text: 'Check the address or go back to your links.',
			action: 'Go to links'
		},
		forbidden: {
			title: 'You do not have access to this page.',
			text: 'Only administrators can see the administration pages.',
			action: 'Go to links'
		}
	}
} satisfies Shape<typeof hu>;
