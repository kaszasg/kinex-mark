import type hu from '../hu/common.js';
import type { Shape } from '../../types.js';

export default {
	actions: {
		save: 'Save',
		cancel: 'Cancel',
		close: 'Close',
		delete: 'Delete',
		edit: 'Edit',
		copy: 'Copy',
		copied: 'Copied',
		retry: 'Retry',
		back: 'Back',
		done: 'Done',
		confirm: 'Confirm',
		download: 'Download',
		upload: 'Upload',
		clear: 'Clear',
		more: 'More',
		open: 'Open'
	},
	states: {
		loading: 'Loading…',
		loadingMore: 'Loading more…',
		empty: 'Nothing to show.',
		saving: 'Saving…'
	},
	time: {
		today: 'today',
		yesterday: 'yesterday',
		never: 'never'
	},
	units: {
		days: { one: '{count} day', other: '{count} days' },
		minutes: { one: '{count} minute', other: '{count} minutes' },
		hours: { one: '{count} hour', other: '{count} hours' }
	},
	missing: '–',
	notifications: 'Notifications',
	skipToContent: 'Skip to content',
	appName: 'Kinex Mark',
	pageTitleSuffix: 'Kinex Mark',
	copyFailed: 'Copying failed. Select the text and copy it by hand.',
	password: {
		show: 'Show password',
		hide: 'Hide password'
	},
	keys: {
		ctrl: 'Ctrl',
		enter: 'Enter',
		delete: 'Delete',
		esc: 'Esc'
	},
	tagPicker: {
		label: 'Tags',
		search: 'Search tags…',
		create: 'Create tag “{name}”',
		empty: 'No matching tag.',
		limit: 'You can choose at most {max} tags.',
		remove: 'Remove: {name}'
	},
	stepUp: {
		label: 'Your own password to confirm',
		help: 'This action is recorded in the security log.'
	}
} satisfies Shape<typeof hu>;
