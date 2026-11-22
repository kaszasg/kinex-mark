import type hu from '../hu/capture.js';
import type { Shape } from '../../types.js';

export default {
	overlay: {
		title: 'Release to save the link',
		text: 'A URL, a browser link or text containing a URL. The title and icon arrive in the background.'
	},
	toast: {
		saved: 'Link saved',
		exists: 'You already have this link',
		multiple: {
			one: '{count} link saved, {existing} already present',
			other: '{count} links saved, {existing} already present'
		},
		hidden: 'Link saved. The current filters hide it.',
		tooMany: 'You can save at most {max} links at once; the rest were skipped.',
		invalidUrl: 'The URL found cannot be saved.',
		openAction: 'Open',
		editAction: 'Edit',
		showAction: 'Show'
	}
} satisfies Shape<typeof hu>;
