import type hu from '../hu/landing.js';
import type { Shape } from '../../types.js';

export default {
	headline: 'Links, nach Nutzung geordnet.',
	intro:
		'Kinex Mark speichert, beschriftet und ordnet deine Links nach Nutzung. Was du lange nicht geöffnet hast, wandert ins Archiv.',
	features: {
		label: 'Funktionen',
		capture: {
			title: 'Schnell erfassen',
			text: 'Zieh ihn ins Fenster oder füge ihn ein. Der Link wird sofort gespeichert, Titel und Symbol folgen im Hintergrund.'
		},
		tags: {
			title: 'Farbige Labels',
			text: 'Ordnung ohne Ordner: Ein Link kann mehrere Labels tragen.'
		},
		filter: {
			title: 'Filtern',
			text: 'Suche in Titel, Beschreibung und URL, filtere nach einem oder mehreren Labels.'
		},
		ranking: {
			title: 'Rangfolge',
			text: 'Häufig geöffnete Links rücken nach oben. Angeheftete stehen immer vorn, untereinander nach Nutzung geordnet.'
		},
		expiry: {
			title: 'Automatisch archivieren',
			text: 'Links, die im eingestellten Zeitraum ungenutzt bleiben, wandern ins Archiv. Ein Klick stellt sie wieder her.'
		}
	},
	hero: {
		alt: 'Liniendiagramm: Ein abgelegter Link kommt in die Liste, farbige Labels und ein Filter ordnen sie, häufig genutzte und angeheftete Links stehen vorn, lange ungenutzte wandern ins Archiv.'
	}
} satisfies Shape<typeof hu>;
