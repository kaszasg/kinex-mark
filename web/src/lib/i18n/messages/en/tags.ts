import type hu from '../hu/tags.js';
import type { Shape } from '../../types.js';

// Tags page, tag dialog and tag deletion (plan 8.7.5, F3).
export default {
	pageTitle: 'Tags',
	summary: {
		one: '{count} tag. Tags are personal: you can put your own tags on links shared with you too.',
		other:
			'{count} tags. Tags are personal: you can put your own tags on links shared with you too.'
	},
	add: 'Add tag',
	table: {
		caption: 'Tags by name',
		name: 'Name',
		color: 'Colour',
		links: 'Links',
		active: 'Active',
		updated: 'Modified',
		actions: 'Actions',
		showLinks: {
			one: '{count} link with the tag {name}',
			other: '{count} links with the tag {name}'
		},
		edit: 'Edit: {name}',
		delete: 'Delete: {name}'
	},
	dialog: {
		createTitle: 'Add tag',
		editTitle: 'Edit tag',
		description: 'The tag is visible only in your own collection.',
		name: 'Name',
		nameHelp: 'At most {max} characters.',
		color: 'Colour',
		preview: 'Preview',
		save: 'Save tag',
		saved: 'Tag saved'
	},
	delete: {
		title: 'Delete this tag?',
		text: {
			zero: 'The tag “{name}” will be deleted. None of your links carry it.',
			one: 'The tag “{name}” will be deleted. It is removed from {count} link; the link stays.',
			other: 'The tag “{name}” will be deleted. It is removed from {count} links; the links stay.'
		},
		confirm: 'Delete tag',
		deleted: 'Tag deleted'
	},
	empty: {
		title: 'No tags yet.',
		text: 'Add one and use it to group your links.'
	}
} satisfies Shape<typeof hu>;
