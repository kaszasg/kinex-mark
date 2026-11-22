import type hu from '../hu/settings.js';
import type { Shape } from '../../types.js';

// Settings: profile and display tabs (plan 8.7.6, F3). Security and privacy live in `security`.
export default {
	pageTitle: '{section} – Settings',
	profile: {
		title: 'Profile',
		description: 'People you share links with see your display name.',
		username: 'Username',
		usernameHelp: 'The username cannot be changed.',
		displayName: 'Display name',
		displayNameHelp: 'At most {max} characters. When empty, your username is shown.',
		role: 'Role',
		createdAt: 'Account created',
		lastLoginAt: 'Last sign-in',
		save: 'Save profile',
		saved: 'Profile saved'
	},
	display: {
		intro: 'Every change is saved at once and applies on all your devices.',
		appearance: {
			title: 'Language and theme'
		},
		language: {
			label: 'Language'
		},
		theme: {
			label: 'Theme',
			dark: 'Dark',
			light: 'Light',
			system: 'Match system'
		},
		list: {
			title: 'Link list',
			description: 'The Links page opens with these defaults.'
		},
		viewMode: {
			label: 'Default view',
			list: 'List',
			tiles: 'Tiles'
		},
		fields: {
			label: 'Shown details',
			icon: 'Icon',
			url: 'URL',
			title: 'Title',
			description: 'Description',
			help: 'A link is labelled with its description, otherwise its title, finally its URL. A field you turn off is not used for the label either.'
		},
		preview: {
			label: 'Preview',
			title: 'Grafana',
			description: 'Production dashboard',
			url: 'grafana.example.net/d/abc'
		},
		archived: {
			label: 'Archived links by default',
			toggle: 'Show archived links'
		},
		sort: {
			label: 'Default sort order',
			usage: 'By usage',
			created: 'Newest first',
			title: 'By name'
		},
		tagMatch: {
			label: 'Default tag match',
			any: 'Any',
			all: 'All',
			help: 'With several tags selected, one is enough or all are required.'
		},
		archive: {
			title: 'Automatic archiving',
			label: 'Archive after inactivity',
			system: {
				zero: 'System default (never)',
				one: 'System default ({count} day)',
				other: 'System default ({count} days)'
			},
			never: 'Never',
			custom: 'Custom…',
			customLabel: 'Number of days',
			customHelp: '{min}–{max} days. Press Enter or leave the field to save.',
			help: 'Pinned links are never archived. Opening, copying, pinning or restoring a link restarts the count.',
			current: {
				zero: 'Currently: no automatic archiving.',
				one: 'Currently: after {count} day of inactivity.',
				other: 'Currently: after {count} days of inactivity.'
			}
		}
	}
} satisfies Shape<typeof hu>;
