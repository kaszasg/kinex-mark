import type hu from '../hu/admin.js';
import type { Shape } from '../../types.js';

export default {
	users: {
		pageTitle: 'Benutzer',
		summary: { one: '{count} Benutzer', other: '{count} Benutzer' },
		add: 'Benutzer hinzufügen',
		filters: {
			label: 'Benutzer filtern',
			search: 'Suche',
			searchPlaceholder: 'Benutzername oder Name',
			role: 'Rolle',
			allRoles: 'Alle',
			status: 'Status',
			statusActive: 'Aktiv',
			statusInactive: 'Inaktiv',
			statusAll: 'Alle',
			clear: 'Filter zurücksetzen'
		},
		table: {
			caption: 'Liste der Benutzer',
			username: 'Benutzername',
			displayName: 'Anzeigename',
			role: 'Rolle',
			status: 'Status',
			twoFactor: '2FA',
			twoFactorOn: 'An',
			links: 'Links',
			sessions: 'Sitzungen',
			lastLogin: 'Letzte Anmeldung',
			actions: 'Aktionen'
		},
		actions: {
			menu: 'Aktionen für {username}',
			edit: 'Bearbeiten',
			resetPassword: 'Passwort zurücksetzen',
			resetTwoFactor: '2FA zurücksetzen',
			unlock: 'Entsperren',
			revokeSessions: 'Sitzungen beenden',
			deactivate: 'Deaktivieren',
			activate: 'Aktivieren',
			delete: 'Löschen',
			selfHint: 'Dein eigenes Konto verwaltest du in den Einstellungen.'
		},
		confirm: {
			resetPassword: {
				title: 'Passwort von {username} zurücksetzen?',
				text: 'Alle Sitzungen enden, und es gibt ein neues temporäres Passwort. Die Aktion wird im Sicherheitsprotokoll erfasst, und bei der nächsten Anmeldung muss das Passwort geändert werden.',
				action: 'Passwort zurücksetzen'
			},
			resetTwoFactor: {
				title: 'Zwei-Faktor-Anmeldung von {username} zurücksetzen?',
				text: 'Alle Sitzungen enden, und die Zwei-Faktor-Anmeldung muss neu aktiviert werden.',
				action: '2FA zurücksetzen'
			},
			deactivate: {
				title: 'Konto von {username} deaktivieren?',
				text: 'Alle Sitzungen enden. Die Freigaben bleiben bestehen.',
				action: 'Konto deaktivieren'
			},
			activate: {
				title: 'Konto von {username} aktivieren?',
				text: 'Die Anmeldung mit dem bisherigen Passwort ist wieder möglich.',
				action: 'Konto aktivieren'
			},
			delete: {
				title: 'Konto von {username} löschen?',
				text: 'Links, Labels und Freigaben werden endgültig gelöscht; im Audit-Protokoll wird der Name durch ein Pseudonym ersetzt. Das kann nicht rückgängig gemacht werden.',
				sharedTitle: {
					one: '{count} geteilter Link verschwindet auch bei den Empfängern.',
					other: '{count} geteilte Links verschwinden auch bei den Empfängern.'
				},
				sharedText:
					'Wenn die Empfänger sie noch brauchen, sollten sie sie vorher selbst speichern.',
				action: 'Benutzer löschen'
			}
		},
		toast: {
			created: '{username} angelegt',
			updated: '{username} geändert',
			passwordReset: 'Passwort von {username} zurückgesetzt',
			twoFactorReset: 'Zwei-Faktor-Anmeldung von {username} zurückgesetzt',
			unlocked: '{username} entsperrt',
			sessionsRevoked: {
				zero: 'Es gab keine Sitzungen zum Beenden',
				one: '{count} Sitzung beendet',
				other: '{count} Sitzungen beendet'
			},
			deactivated: '{username} deaktiviert',
			activated: '{username} aktiviert',
			deleted: '{username} gelöscht'
		},
		empty: {
			title: 'Kein Benutzer entspricht den Filtern.',
			text: 'Ändere die Filter oder setze sie zurück.'
		}
	},
	pager: {
		label: 'Seitennavigation',
		previous: 'Vorherige Seite',
		next: 'Nächste Seite',
		page: 'Seite {page}',
		range: '{from}–{to} / {total}'
	},
	userDialog: {
		createTitle: 'Benutzer hinzufügen',
		createText: 'Der neue Benutzer legt bei der ersten Anmeldung ein eigenes Passwort fest.',
		editTitle: '{username} bearbeiten',
		editText: 'Änderungen an Rolle und Aktivstatus werden im Sicherheitsprotokoll erfasst.',
		fields: {
			username: 'Benutzername',
			displayName: 'Anzeigename',
			role: 'Rolle',
			locale: 'Sprache',
			password: 'Passwort',
			active: 'Aktiv'
		},
		usernameHelp: '3–64 Zeichen: Buchstaben, Ziffern, Punkt, Bindestrich, Unterstrich.',
		displayNameHelp: 'Diesen Namen sehen andere bei geteilten Links. Höchstens 100 Zeichen.',
		localeHelp:
			'Die Sprache der Oberfläche bei der ersten Anmeldung. Sie lässt sich später ändern.',
		passwordMode: {
			label: 'Passwortvergabe',
			generated: 'Generiertes Passwort',
			manual: 'Selbst festlegen'
		},
		generatedHelp: 'Der Server erzeugt ein Passwort mit 16 Zeichen und zeigt es nur einmal an.',
		passwordHelp:
			'Mindestens {min} Zeichen, mindestens {distinct} verschiedene Zeichen, ohne den Benutzernamen.',
		activeHelp: 'Mit einem inaktiven Konto ist keine Anmeldung möglich.',
		selfHint: 'Dein eigenes Konto verwaltest du in den Einstellungen.',
		create: 'Benutzer anlegen',
		save: 'Änderungen speichern'
	},
	tempPassword: {
		title: 'Temporäres Passwort',
		text: 'Temporäres Passwort von {username}:',
		copy: 'Passwort kopieren',
		copied: 'Passwort kopiert',
		warningTitle: 'Das Passwort ist nur jetzt sichtbar',
		warningText:
			'Gib es über einen sicheren Kanal weiter. Bei der ersten Anmeldung muss es ersetzt werden.',
		done: 'Fertig'
	},
	settings: {
		pageTitle: 'Systemeinstellungen',
		subtitle: 'Einstellungen, die für alle Benutzer gelten.',
		cards: {
			archive: {
				title: 'Archivierung',
				text: 'Nicht geöffnete Links werden nach diesem Zeitraum automatisch archiviert, wenn der Benutzer keinen eigenen festgelegt hat.'
			},
			sessions: {
				title: 'Sitzungen',
				text: 'Eine Anmeldung läuft nach der Leerlaufzeit oder nach der Höchstdauer ab.'
			},
			audit: {
				title: 'Audit-Protokoll',
				text: 'Sicherheitsereignisse werden so lange aufbewahrt und danach gelöscht.'
			},
			fetch: {
				title: 'Abruf von Metadaten',
				text: 'Der Server ruft Titel und Symbol der Links ab.'
			},
			privacy: {
				title: 'Datenschutzhinweise',
				text: 'Diese Angaben erscheinen in den Datenschutzhinweisen. Veröffentliche nach einer wesentlichen Änderung eine neue Version.'
			}
		},
		fields: {
			defaultArchiveDays: 'Archivierungszeitraum',
			sessionIdleMinutes: 'Leerlaufzeit',
			sessionAbsoluteHours: 'Höchstdauer einer Sitzung',
			auditRetentionDays: 'Aufbewahrungsdauer',
			allowPrivateFetch: 'Interne Netzwerkadressen erlauben',
			privacyControllerName: 'Name des Verantwortlichen',
			privacyControllerAddress: 'Anschrift des Verantwortlichen',
			privacyControllerEmail: 'E-Mail-Adresse des Verantwortlichen',
			privacyDpoContact: 'Datenschutzkontakt',
			privacySupervisoryAuthority: 'Zuständige Aufsichtsbehörde'
		},
		help: {
			archiveDays: '{min}–{max} Tage. 0: nie. Standardwert: {default}.',
			days: '{min}–{max} Tage. Standardwert: {default}.',
			minutes: '{min}–{max} Minuten. Standardwert: {default}.',
			hours: '{min}–{max} Stunden. Standardwert: {default}.',
			text: 'Höchstens {max} Zeichen.',
			email: 'Darf leer bleiben. Höchstens {max} Zeichen.',
			supervisoryAuthority:
				'Höchstens {max} Zeichen. Zum Beispiel: Der Bundesbeauftragte für den Datenschutz und die Informationsfreiheit, bfdi.bund.de',
			allowPrivateFetch:
				'Für Intranet-Links. Eingeschaltet darf der Server auch Adressen im lokalen Netzwerk abrufen; Cloud-Metadatenadressen bleiben gesperrt.'
		},
		units: {
			days: 'Tage',
			minutes: 'Minuten',
			hours: 'Stunden'
		},
		runArchive: 'Archivierung jetzt ausführen',
		archived: {
			zero: 'Es gab keine Links zum Archivieren',
			one: '{count} Link archiviert',
			other: '{count} Links archiviert'
		},
		locked: {
			title: 'Durch eine Umgebungsvariable festgelegt',
			privateFetch: 'Der Wert von APP_ALLOW_PRIVATE_FETCH entscheidet.',
			generic: 'Die Serverkonfiguration entscheidet.'
		},
		notice: {
			version: 'Aktuelle Version',
			updatedAt: 'Veröffentlicht',
			bump: 'Neue Version veröffentlichen',
			bumpTitle: 'Neue Version der Hinweise veröffentlichen?',
			bumpText:
				'Alle Benutzer müssen sie bei der nächsten Anmeldung oder Anfrage erneut bestätigen.',
			bumped: 'Version {version} der Hinweise veröffentlicht'
		},
		unsaved: 'Ungespeicherte Änderungen',
		noChanges: 'Keine ungespeicherten Änderungen',
		save: 'Einstellungen speichern',
		discard: 'Verwerfen',
		saved: 'Einstellungen gespeichert',
		leave: {
			title: 'Änderungen verwerfen?',
			text: 'Die nicht gespeicherten Einstellungen gehen verloren.',
			action: 'Änderungen verwerfen'
		}
	}
} satisfies Shape<typeof hu>;
