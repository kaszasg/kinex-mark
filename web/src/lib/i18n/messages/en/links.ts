import type hu from '../hu/links.js';
import type { Shape } from '../../types.js';

export default {
	pageTitle: 'Links',
	summary: {
		zero: 'No links',
		one: '{count} link, {pinned} pinned',
		other: '{count} links, {pinned} pinned'
	},
	summaryArchived: {
		zero: 'No links',
		one: '{count} link, {pinned} pinned, {archived} archived',
		other: '{count} links, {pinned} pinned, {archived} archived'
	},
	results: {
		zero: 'No results',
		one: '{count} result',
		other: '{count} results'
	},
	toolbar: {
		label: 'List options',
		add: 'Add link',
		scope: 'Scope',
		sort: 'Sort order',
		view: 'View',
		display: 'Display',
		shortcuts: 'Keyboard shortcuts'
	},
	scope: {
		all: 'All',
		own: 'Mine',
		shared: 'Shared with me'
	},
	sort: {
		usage: 'By usage',
		created: 'Newest first',
		title: 'By name'
	},
	view: {
		list: 'List',
		tiles: 'Tiles'
	},
	fields: {
		icon: 'Icon',
		url: 'URL',
		title: 'Title',
		description: 'Description'
	},
	archivedToggle: 'Include archived',
	filters: {
		label: 'Tag filter',
		untagged: 'Untagged',
		more: 'More tags ({count})',
		search: 'Search tags…',
		empty: 'No such tag.',
		clear: 'Clear filters',
		match: {
			label: 'Tag matching',
			any: 'Any',
			all: 'All'
		}
	},
	groups: {
		pinned: 'Pinned',
		others: 'Others'
	},
	list: {
		label: 'Links',
		allLoaded: 'All links loaded.'
	},
	row: {
		pinned: 'Pinned',
		sharedBy: 'Shared by {owner}',
		archived: 'Archived',
		pending: 'Fetching title and icon',
		uses: {
			one: '{count} use (opened or copied)',
			other: '{count} uses (opened or copied)'
		},
		lastUsed: 'Last used: {time}',
		neverUsed: 'Not used yet',
		moreTags: 'More tags: {names}',
		menu: 'Actions: {label}'
	},
	actions: {
		add: 'Add link',
		open: 'Open',
		copyUrl: 'Copy URL',
		copyMarkdown: 'Copy as Markdown',
		details: 'Details',
		edit: 'Edit',
		tags: 'Tags',
		share: 'Share',
		refresh: 'Refresh title and icon',
		pin: 'Pin',
		unpin: 'Unpin',
		archive: 'Archive',
		restore: 'Restore',
		delete: 'Delete'
	},
	dialog: {
		addTitle: 'Add link',
		editTitle: 'Edit link',
		addText: 'The URL is enough; the title and icon arrive in the background.',
		editText: 'Changes are visible to everyone you shared the link with.',
		url: 'URL',
		title: 'Title',
		titleHelp: 'Leave it empty to show the fetched title.',
		description: 'Short description',
		descriptionHelp: 'The list shows this first.',
		counter: '{count} / 500',
		tags: 'Tags',
		pin: 'Pin to the top of the list',
		save: 'Save link',
		showExisting: 'Show in the list'
	},
	tagsDialog: {
		title: 'Edit tags',
		text: 'Tags are personal: only you see them, on shared links too.',
		save: 'Save tags'
	},
	details: {
		title: 'Link details',
		sections: {
			data: 'Data',
			usage: 'Usage',
			times: 'Timestamps',
			metadata: 'Metadata',
			sharing: 'Sharing'
		},
		fields: {
			url: 'URL',
			title: 'Title',
			description: 'Description',
			tags: 'Tags',
			owner: 'Owner'
		},
		noTags: 'No tags',
		editTags: 'Edit tags',
		metrics: {
			usage: 'Usage',
			archive: 'Archiving',
			created: 'Added'
		},
		neverUsed: 'Not used yet',
		never: 'Never',
		archivedNote: 'Already archived',
		dueIn: {
			one: 'in {count} day',
			other: 'in {count} days'
		},
		dueToday: 'due today',
		times: {
			created: 'Added',
			updated: 'Modified',
			lastUsed: 'Last used',
			sharedAt: 'Shared with you',
			inactiveSince: 'Inactive since',
			archivedAt: 'Archived'
		},
		archiveReason: {
			auto: 'automatically',
			manual: 'manually'
		},
		archivedWith: '{date}, {reason}',
		fetchedAt: 'Fetched at',
		attempts: 'Attempts',
		manageShares: 'Manage sharing',
		deleteLink: 'Delete link'
	},
	share: {
		title: 'Share link',
		text: 'Recipients see the link in their own list, with their own tags and pin. Only you can change it.',
		current: 'Current shares',
		empty: 'Not shared with anyone yet.',
		everyone: 'Everyone',
		roleTarget: 'Role: {role}',
		revoke: 'Revoke share: {target}',
		newShare: 'New share',
		type: {
			label: 'Recipient',
			all: 'Everyone',
			role: 'Role',
			user: 'User'
		},
		role: 'Role',
		rolePlaceholder: 'Choose a role',
		user: 'User',
		userPlaceholder: 'Choose a user',
		userSearch: 'Search users…',
		userEmpty: 'No such user.',
		add: 'Add share'
	},
	delete: {
		title: 'Delete this link?',
		text: 'This link will be deleted permanently: “{label}”. Its shares and every user’s usage data are deleted too. This cannot be undone.',
		confirm: 'Delete link'
	},
	importExport: {
		label: 'Import and export',
		exportJson: 'Export as JSON',
		exportHtml: 'Export as browser HTML',
		import: 'Import from file…',
		resultTitle: 'Import finished',
		resultText: 'Titles and icons of the new links arrive in the background.',
		total: 'Total',
		created: 'New',
		duplicates: 'Already present',
		invalid: 'Invalid'
	},
	expiry: {
		label: 'Archived around {date} unless you open or copy it before then',
		daysLeft: { one: '{count} day', other: '{count} days' },
		today: 'today'
	},
	empty: {
		collection: {
			title: 'No links yet.',
			text: 'Drop one here, paste it from the clipboard ({shortcut}) or add it by hand.'
		},
		filtered: {
			title: 'No results.',
			text: 'Change the search or clear the filters.'
		},
		archived: {
			title: 'No archived links.',
			text: 'Links you archive and links unused for a long time appear here.'
		}
	},
	toast: {
		saved: 'Link saved',
		deleted: 'Link deleted',
		pinned: 'Link pinned',
		unpinned: 'Link unpinned',
		pinnedRestored: 'Link pinned and restored',
		archived: 'Link archived',
		restored: 'Link restored',
		undo: 'Undo',
		urlCopied: 'URL copied',
		markdownCopied: 'Markdown link copied',
		refreshing: 'Refreshing title and icon',
		tagsSaved: 'Tags saved',
		shared: 'Link shared',
		unshared: 'Share revoked',
		exported: 'Export downloaded'
	},
	shortcuts: {
		title: 'Keyboard shortcuts',
		text: 'Shortcuts are off while you type in a field or a dialog is open.',
		or: 'or',
		items: {
			search: 'Focus the search field',
			add: 'Add link',
			paste: 'Quick capture from the clipboard',
			next: 'Next link',
			previous: 'Previous link',
			open: 'Open the selected link',
			pin: 'Toggle pin',
			copyUrl: 'Copy URL',
			copyMarkdown: 'Copy as Markdown',
			details: 'Details',
			edit: 'Edit',
			archive: 'Archive or restore',
			delete: 'Delete with confirmation',
			view: 'Switch between list and tiles',
			help: 'Keyboard shortcuts',
			clear: 'Clear the search field'
		}
	}
} satisfies Shape<typeof hu>;
