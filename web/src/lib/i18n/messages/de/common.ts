import type hu from '../hu/common.js';
import type { Shape } from '../../types.js';

export default {
	actions: {
		save: 'Speichern',
		cancel: 'Abbrechen',
		close: 'Schließen',
		delete: 'Löschen',
		edit: 'Bearbeiten',
		copy: 'Kopieren',
		copied: 'Kopiert',
		retry: 'Erneut versuchen',
		back: 'Zurück',
		done: 'Fertig',
		confirm: 'Bestätigen',
		download: 'Herunterladen',
		upload: 'Hochladen',
		clear: 'Leeren',
		more: 'Mehr',
		open: 'Öffnen'
	},
	states: {
		loading: 'Wird geladen…',
		loadingMore: 'Weitere werden geladen…',
		empty: 'Nichts anzuzeigen.',
		saving: 'Wird gespeichert…'
	},
	time: {
		today: 'heute',
		yesterday: 'gestern',
		never: 'nie'
	},
	units: {
		days: { one: '{count} Tag', other: '{count} Tage' },
		minutes: { one: '{count} Minute', other: '{count} Minuten' },
		hours: { one: '{count} Stunde', other: '{count} Stunden' }
	},
	missing: '–',
	notifications: 'Benachrichtigungen',
	skipToContent: 'Zum Inhalt springen',
	appName: 'Kinex Mark',
	pageTitleSuffix: 'Kinex Mark',
	copyFailed: 'Kopieren fehlgeschlagen. Markiere den Text und kopiere ihn von Hand.',
	password: {
		show: 'Passwort anzeigen',
		hide: 'Passwort verbergen'
	},
	keys: {
		ctrl: 'Strg',
		enter: 'Enter',
		delete: 'Entf',
		esc: 'Esc'
	},
	tagPicker: {
		label: 'Labels',
		search: 'Labels suchen…',
		create: 'Label „{name}“ anlegen',
		empty: 'Kein passendes Label.',
		limit: 'Du kannst höchstens {max} Labels wählen.',
		remove: 'Entfernen: {name}'
	},
	stepUp: {
		label: 'Dein eigenes Passwort zur Bestätigung',
		help: 'Die Aktion wird im Sicherheitsprotokoll vermerkt.'
	}
} satisfies Shape<typeof hu>;
