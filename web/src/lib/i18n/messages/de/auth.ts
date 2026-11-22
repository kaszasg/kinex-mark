import type hu from '../hu/auth.js';
import type { Shape } from '../../types.js';

export default {
	login: {
		title: 'Anmelden',
		subtitle: 'Kinex Mark, die Linksammlung von Kinexium.',
		username: 'Benutzername',
		password: 'Passwort',
		submit: 'Anmelden',
		noAccount: 'Konten legt ein Administrator an.',
		version: 'Version {version}',
		footer: 'Sprache, Design und Datenschutzhinweise'
	},
	totp: {
		title: 'Zwei-Faktor-Anmeldung',
		text: 'Gib den 6-stelligen Code aus deiner Authenticator-App ein.',
		code: 'Bestätigungscode',
		submit: 'Code prüfen',
		useRecovery: 'Wiederherstellungscode verwenden',
		useCode: 'Bestätigungscode verwenden',
		back: 'Zurück zur Anmeldung'
	},
	recovery: {
		text: 'Gib einen deiner gespeicherten Wiederherstellungscodes ein.',
		code: 'Wiederherstellungscode',
		placeholder: 'xxxxx-xxxxx',
		help: 'Bindestrich und Leerzeichen kannst du weglassen.',
		used: {
			one: 'Du hast dich mit einem Wiederherstellungscode angemeldet. Dir bleibt noch {count} Code.',
			other:
				'Du hast dich mit einem Wiederherstellungscode angemeldet. Dir bleiben noch {count} Codes.'
		},
		low: 'Erzeuge neue Codes unter Einstellungen › Sicherheit.'
	},
	logout: {
		title: 'Du hast dich abgemeldet.',
		text: 'Melde dich wieder an, um weiterzumachen.'
	},
	expired: {
		title: 'Die Sitzung ist abgelaufen.',
		text: 'Melde dich erneut an.'
	},
	accountDeleted: {
		title: 'Dein Konto wurde gelöscht.',
		text: 'Deine Links, Labels, Freigaben und Einstellungen sind endgültig entfernt.'
	},
	passwordChange: {
		pageTitle: 'Passwortänderung',
		title: 'Neues Passwort festlegen',
		text: 'Dein Konto wurde mit einem vorläufigen Passwort angelegt, oder ein Administrator hat es zurückgesetzt. Lege ein neues fest, um weiterzumachen.',
		current: 'Aktuelles Passwort',
		new: 'Neues Passwort',
		confirm: 'Neues Passwort wiederholen',
		mismatch: 'Die beiden Passwörter stimmen nicht überein.',
		submit: 'Passwort ändern',
		saved: 'Passwort geändert',
		savedOtherSessions: 'Passwort geändert. Auf deinen anderen Geräten bist du abgemeldet.'
	},
	passwordPolicy:
		'Mindestens {min} Zeichen, mindestens {distinct} verschiedene Zeichen, ohne deinen Benutzernamen.'
} satisfies Shape<typeof hu>;
