import type hu from '../hu/settings.js';
import type { Shape } from '../../types.js';

// Settings: profile and display tabs (plan 8.7.6, F3). Security and privacy live in `security`.
export default {
	pageTitle: '{section} – Einstellungen',
	profile: {
		title: 'Profil',
		description: 'Wer von dir geteilte Links erhält, sieht deinen Anzeigenamen.',
		username: 'Benutzername',
		usernameHelp: 'Der Benutzername kann nicht geändert werden.',
		displayName: 'Anzeigename',
		displayNameHelp:
			'Höchstens {max} Zeichen. Bleibt das Feld leer, wird dein Benutzername angezeigt.',
		role: 'Rolle',
		createdAt: 'Konto erstellt',
		lastLoginAt: 'Letzte Anmeldung',
		save: 'Profil speichern',
		saved: 'Profil gespeichert'
	},
	display: {
		intro: 'Jede Änderung wird sofort gespeichert und gilt auf allen deinen Geräten.',
		appearance: {
			title: 'Sprache und Design'
		},
		language: {
			label: 'Sprache'
		},
		theme: {
			label: 'Design',
			dark: 'Dunkel',
			light: 'Hell',
			system: 'Wie das System'
		},
		list: {
			title: 'Linkliste',
			description: 'Die Seite Links öffnet sich mit diesen Standardwerten.'
		},
		viewMode: {
			label: 'Standardansicht',
			list: 'Liste',
			tiles: 'Kacheln'
		},
		fields: {
			label: 'Angezeigte Angaben',
			icon: 'Symbol',
			url: 'URL',
			title: 'Titel',
			description: 'Beschreibung',
			help: 'Ein Link wird mit seiner Beschreibung beschriftet, sonst mit dem Titel, zuletzt mit der URL. Ausgeschaltete Felder werden auch für die Beschriftung nicht verwendet.'
		},
		preview: {
			label: 'Vorschau',
			title: 'Grafana',
			description: 'Produktiv-Dashboard',
			url: 'grafana.example.net/d/abc'
		},
		archived: {
			label: 'Archivierte Links standardmäßig',
			toggle: 'Archivierte Links anzeigen'
		},
		sort: {
			label: 'Standardsortierung',
			usage: 'Nach Nutzung',
			created: 'Neueste zuerst',
			title: 'Nach Name'
		},
		tagMatch: {
			label: 'Labelabgleich standardmäßig',
			any: 'Beliebiges',
			all: 'Alle',
			help: 'Bei mehreren gewählten Labels genügt eines, oder alle sind nötig.'
		},
		archive: {
			title: 'Automatisches Archivieren',
			label: 'Archivieren nach Inaktivität',
			system: {
				zero: 'Systemstandard (nie)',
				one: 'Systemstandard ({count} Tag)',
				other: 'Systemstandard ({count} Tage)'
			},
			never: 'Nie',
			custom: 'Eigener Wert…',
			customLabel: 'Anzahl der Tage',
			customHelp: '{min}–{max} Tage. Mit Enter oder beim Verlassen des Felds speichern.',
			help: 'Angeheftete Links werden nie archiviert. Öffnen, Kopieren, Anheften oder Wiederherstellen startet die Zählung neu.',
			current: {
				zero: 'Aktuell: kein automatisches Archivieren.',
				one: 'Aktuell: nach {count} Tag Inaktivität.',
				other: 'Aktuell: nach {count} Tagen Inaktivität.'
			}
		}
	}
} satisfies Shape<typeof hu>;
