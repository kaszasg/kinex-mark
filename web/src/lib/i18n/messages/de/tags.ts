import type hu from '../hu/tags.js';
import type { Shape } from '../../types.js';

// Tags page, tag dialog and tag deletion (plan 8.7.5, F3).
export default {
	pageTitle: 'Labels',
	summary: {
		one: '{count} Label. Labels sind persönlich: Auch auf Links, die mit dir geteilt wurden, setzt du deine eigenen Labels.',
		other:
			'{count} Labels. Labels sind persönlich: Auch auf Links, die mit dir geteilt wurden, setzt du deine eigenen Labels.'
	},
	add: 'Label hinzufügen',
	table: {
		caption: 'Labels nach Name',
		name: 'Name',
		color: 'Farbe',
		links: 'Links',
		active: 'Aktiv',
		updated: 'Geändert',
		actions: 'Aktionen',
		showLinks: {
			one: '{count} Link mit dem Label {name}',
			other: '{count} Links mit dem Label {name}'
		},
		edit: 'Bearbeiten: {name}',
		delete: 'Löschen: {name}'
	},
	dialog: {
		createTitle: 'Label hinzufügen',
		editTitle: 'Label bearbeiten',
		description: 'Das Label ist nur in deiner eigenen Sammlung sichtbar.',
		name: 'Name',
		nameHelp: 'Höchstens {max} Zeichen.',
		color: 'Farbe',
		preview: 'Vorschau',
		save: 'Label speichern',
		saved: 'Label gespeichert'
	},
	delete: {
		title: 'Löschst du dieses Label?',
		text: {
			zero: 'Das Label „{name}“ wird gelöscht. Keiner deiner Links trägt es.',
			one: 'Das Label „{name}“ wird gelöscht. Es wird von {count} Link entfernt, der Link bleibt erhalten.',
			other:
				'Das Label „{name}“ wird gelöscht. Es wird von {count} Links entfernt, die Links bleiben erhalten.'
		},
		confirm: 'Label löschen',
		deleted: 'Label gelöscht'
	},
	empty: {
		title: 'Noch kein Label.',
		text: 'Lege eines an und gruppiere damit deine Links.'
	}
} satisfies Shape<typeof hu>;
