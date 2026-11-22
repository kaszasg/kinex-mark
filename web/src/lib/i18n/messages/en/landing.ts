import type hu from '../hu/landing.js';
import type { Shape } from '../../types.js';

export default {
	headline: 'Links, ranked by use.',
	intro:
		'Kinex Mark stores, tags and ranks your links by use. What you have not opened for a long time moves to the archive.',
	features: {
		label: 'Features',
		capture: {
			title: 'Quick capture',
			text: 'Drop it on the window or paste it. The link is saved at once; title and icon follow in the background.'
		},
		tags: {
			title: 'Coloured tags',
			text: 'Grouping without folders: one link can carry several tags.'
		},
		filter: {
			title: 'Filtering',
			text: 'Search titles, descriptions and URLs; filter by one or more tags.'
		},
		ranking: {
			title: 'Ranking',
			text: 'Frequently opened links move up. Pinned links stay on top, ordered by use among themselves.'
		},
		expiry: {
			title: 'Automatic archiving',
			text: 'Links unused for the set period move to the archive. One click restores them.'
		}
	},
	hero: {
		alt: 'Line diagram: a dropped link joins the list, coloured tags and a filter sort it, frequently used and pinned links stay on top, long-unused links move to the archive.'
	}
} satisfies Shape<typeof hu>;
