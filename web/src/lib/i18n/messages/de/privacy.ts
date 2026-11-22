import type hu from '../hu/privacy.js';
import type { Shape } from '../../types.js';

export default {
	title: 'Datenschutzhinweise',
	version: 'Version {version}',
	updatedAt: 'Aktualisiert: {date}',
	backToApp: 'Zurück zur Anwendung',
	controller: {
		name: 'Name',
		address: 'Anschrift',
		email: 'E-Mail',
		dpo: 'Ansprechpartner für Datenschutz'
	},
	sections: {
		controller: {
			title: 'Verantwortlicher',
			body: 'Verantwortlich ist die Organisation, die Kinex Mark betreibt. Mit Fragen zum Datenschutz erreichst du sie über die folgenden Kontaktdaten.'
		},
		data: {
			title: 'Verarbeitete Daten',
			body: 'Benutzername, Anzeigename, Passwort-Hash, verschlüsseltes Geheimnis der Zwei-Faktor-Anmeldung, Sitzungen mit IP-Adresse und Browserkennung, Links, Labels, Freigaben, Nutzungsstatistik, Einstellungen sowie ein Sicherheitsprotokoll mit IP-Adressen. E-Mail-Adressen, Telefonnummern und Tracking-Daten verarbeitet das System nicht.'
		},
		purposes: {
			title: 'Zwecke',
			body: 'Bereitstellung der Linksammlung sowie Sicherheit des Kontos und des Systems (Anmeldebegrenzung, Sicherheitsprotokoll).'
		},
		legalBasis: {
			title: 'Rechtsgrundlagen',
			body: 'Die Verarbeitung von Konto, Links, Labels, Freigaben und Einstellungen ist für die Bereitstellung des Dienstes erforderlich (Art. 6 Abs. 1 lit. b DSGVO). Sicherheitsprotokoll, Anmeldebegrenzung und die IP-Adresse der Sitzungen beruhen auf dem berechtigten Interesse des Betreibers an einem sicheren System (Art. 6 Abs. 1 lit. f DSGVO). Mit der Bestätigung dieser Hinweise bestätigst du, dass du sie gelesen hast; sie ist keine Einwilligung.'
		},
		recipients: {
			title: 'Empfänger',
			body: 'Benutzer, mit denen du einen Link teilst, sehen den Link, deinen Benutzernamen und deinen Anzeigenamen. Entwickler und Administratoren finden bei der Wahl eines Freigabeempfängers jeden aktiven Benutzer nach Benutzername, Anzeigename und Rolle. Die Administratoren des Betreibers sehen die für die Benutzerverwaltung nötigen Daten und das Sicherheitsprotokoll. Der Hosting-Anbieter des Betreibers greift als Auftragsverarbeiter auf die Daten zu.'
		},
		transfers: {
			title: 'Übermittlung in Drittländer',
			body: 'Es werden keine Daten in ein Drittland oder an eine internationale Organisation übermittelt.'
		},
		automatedDecisions: {
			title: 'Automatisierte Entscheidungen',
			body: 'Es gibt keine automatisierte Entscheidungsfindung und kein Profiling. Die automatische Archivierung betrifft nur die Reihenfolge der Linkliste.'
		},
		retention: {
			title: 'Speicherdauer',
			body: 'Sitzungen enden spätestens nach {hours}, bei Inaktivität schon nach {minutes}. Einträge im Sicherheitsprotokoll bleiben {days} lang gespeichert. Alle anderen Daten bleiben bis zur Löschung des Kontos erhalten.'
		},
		storage: {
			title: 'Cookies und lokaler Speicher',
			body: 'Das System verwendet ein einziges, für die Anmeldung nötiges Cookie (kxm_session oder __Host-kxm_session). Der lokale Speicher deines Browsers behält das gewählte Design (mode-watcher-mode, mode-watcher-theme) und die Sprache (kxm-locale). Es gibt kein Tracking, keine Analyse und keine Inhalte von Dritten.'
		},
		fetching: {
			title: 'Abruf von Metadaten',
			body: 'Titel und Symbol deiner Links lädt der Server. Dein Browser verbindet sich deshalb erst mit einem Dritten, wenn du einen Link öffnest.'
		},
		adminAccess: {
			title: 'Zugriff durch Administratoren',
			body: 'Ein Administrator kann dein Passwort zurücksetzen und sich damit in dein Konto anmelden. Das wird im Sicherheitsprotokoll vermerkt, erscheint in deinem eigenen Datenexport, und bei der nächsten Anmeldung musst du ein neues Passwort festlegen.'
		},
		rights: {
			title: 'Deine Rechte',
			body: 'Nach Art. 15 bis 21 DSGVO hast du folgende Rechte:'
		}
	},
	rights: {
		access:
			'Auskunft: Unter Einstellungen › Datenschutz erhältst du mit „Daten herunterladen“ alle deine Daten.',
		portability:
			'Datenübertragbarkeit: Derselbe Export ist eine maschinenlesbare JSON-Datei; deine Links kannst du auf der Seite Links auch als Lesezeichendatei für den Browser exportieren.',
		rectification:
			'Berichtigung: Deinen Anzeigenamen änderst du unter Einstellungen › Profil, deine Links und Labels direkt.',
		erasure:
			'Löschung: Unter Einstellungen › Datenschutz kannst du dein Konto löschen; alle Links, Labels und Freigaben werden mitgelöscht.',
		restriction:
			'Einschränkung der Verarbeitung: Wende dich über die obigen Kontaktdaten an den Verantwortlichen. Ein Administrator kann dein Konto deaktivieren; deine Daten bleiben dann erhalten, ihre Verarbeitung ruht.',
		objection:
			'Widerspruch: Gegen die Verarbeitung auf Grundlage berechtigter Interessen kannst du über die obigen Kontaktdaten widersprechen.',
		complaint: 'Beschwerde: Du kannst dich bei der Aufsichtsbehörde beschweren: {authority}.',
		defaultAuthority: 'die Datenschutzbehörde an deinem Wohnort'
	},
	consent: {
		title: 'Datenschutzhinweise',
		text: 'Lies, wie Kinex Mark mit deinen Daten umgeht. Bestätige zum Weitermachen, dass du die Hinweise gelesen hast.',
		openFull: 'Als ganze Seite öffnen',
		newTab: 'öffnet in einem neuen Tab',
		checkbox: 'Ich habe die Datenschutzhinweise gelesen und nehme sie zur Kenntnis.',
		submit: 'Zur Kenntnis genommen, weiter'
	}
} satisfies Shape<typeof hu>;
