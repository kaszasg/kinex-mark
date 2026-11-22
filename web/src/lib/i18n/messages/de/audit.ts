import type hu from '../hu/audit.js';
import type { Shape } from '../../types.js';

export default {
	pageTitle: 'Audit-Protokoll',
	subtitle: 'Sicherheitsereignisse werden {days} Tage aufbewahrt.',
	filters: {
		label: 'Audit-Protokoll filtern',
		event: 'Ereignis',
		all: 'Alle',
		groupsHeading: 'Ereignisgruppen',
		eventsHeading: 'Einzelne Ereignisse',
		groups: {
			auth: 'Anmeldung',
			session: 'Sitzungen',
			user: 'Benutzer',
			settings: 'Einstellungen',
			privacy: 'Datenschutz',
			link: 'Freigabe'
		},
		result: 'Ergebnis',
		actor: 'Akteur',
		actorSearch: 'Benutzer suchen…',
		actorEmpty: 'Kein solcher Benutzer.',
		actorUnavailable: 'Die Benutzerliste kann nicht geladen werden.',
		actorUnknown: 'Benutzer #{id}',
		from: 'Von',
		to: 'Bis',
		clear: 'Filter zurücksetzen'
	},
	table: {
		caption: 'Einträge des Audit-Protokolls',
		at: 'Zeitpunkt',
		event: 'Ereignis',
		result: 'Ergebnis',
		actor: 'Akteur',
		target: 'Ziel',
		ip: 'IP',
		details: 'Details',
		showDetails: 'Details anzeigen'
	},
	result: {
		success: 'Erfolgreich',
		failure: 'Fehlgeschlagen'
	},
	empty: {
		title: 'Kein Ereignis entspricht den Filtern.',
		text: 'Ändere die Filter oder setze sie zurück.'
	},
	events: {
		bootstrap: {
			admin_created: 'Erster Administrator angelegt'
		},
		auth: {
			login: {
				success: 'Angemeldet',
				failure: 'Anmeldung fehlgeschlagen',
				locked: 'Anmeldung gesperrt'
			},
			logout: 'Abgemeldet',
			recovery_code: {
				used: 'Wiederherstellungscode verwendet'
			},
			password: {
				changed: 'Passwort geändert'
			},
			'2fa': {
				enabled: 'Zwei-Faktor-Anmeldung aktiviert',
				disabled: 'Zwei-Faktor-Anmeldung deaktiviert',
				recovery_regenerated: 'Neue Wiederherstellungscodes'
			}
		},
		session: {
			revoked: 'Sitzung beendet',
			revoked_others: 'Andere Sitzungen beendet'
		},
		user: {
			created: 'Benutzer angelegt',
			updated: 'Benutzer geändert',
			deactivated: 'Benutzer deaktiviert',
			activated: 'Benutzer aktiviert',
			deleted: 'Benutzer gelöscht',
			password_reset: 'Passwort zurückgesetzt',
			'2fa_reset': 'Zwei-Faktor-Anmeldung zurückgesetzt',
			unlocked: 'Entsperrt',
			sessions_revoked: 'Sitzungen beendet',
			self_deleted: 'Eigenes Konto gelöscht'
		},
		settings: {
			updated: 'Einstellungen geändert'
		},
		privacy: {
			notice_bumped: 'Neue Version der Datenschutzhinweise',
			consent_accepted: 'Datenschutzhinweise bestätigt'
		},
		gdpr: {
			export: 'Datenexport'
		},
		link: {
			shared: 'Link geteilt',
			unshared: 'Freigabe entfernt'
		}
	},
	targets: {
		user: 'Benutzer',
		link: 'Link',
		session: 'Sitzung',
		settings: 'Einstellungen',
		tag: 'Label'
	},
	details: {
		keys: {
			reason: 'Grund',
			method: 'Methode',
			scope: 'Bereich',
			changed: 'Geänderte Felder',
			role: 'Rolle',
			active: 'Aktiv',
			remaining: 'Verbleibende Codes',
			sessionId: 'Sitzung',
			count: 'Anzahl',
			version: 'Version',
			targetType: 'Zieltyp',
			targetRole: 'Zielrolle',
			targetUserId: 'Zielbenutzer',
			generatedPassword: 'Generiertes Passwort',
			via: 'Quelle'
		},
		values: {
			reason: {
				credentials: 'Falsches Passwort',
				unknown_user: 'Unbekannter Benutzername',
				disabled: 'Inaktives Konto',
				totp: 'Falscher Code'
			},
			method: {
				password: 'Passwort',
				totp: 'Bestätigungscode',
				recovery_code: 'Wiederherstellungscode'
			},
			scope: {
				user: 'Benutzer',
				ip: 'IP-Adresse'
			},
			targetType: {
				all: 'Alle',
				role: 'Rolle',
				user: 'Benutzer'
			},
			via: {
				cli: 'Kommandozeile'
			}
		},
		yes: 'Ja',
		no: 'Nein',
		change: '{from} → {to}'
	}
} satisfies Shape<typeof hu>;
