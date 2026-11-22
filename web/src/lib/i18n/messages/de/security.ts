import type hu from '../hu/security.js';
import type { Shape } from '../../types.js';

// Settings > Security and Settings > Privacy (plan 8.7.6, owner F5).
export default {
	password: {
		title: 'Passwort',
		text: 'Nach einer Passwortänderung wirst du auf deinen anderen Geräten abgemeldet.'
	},
	twoFactor: {
		title: 'Zwei-Faktor-Anmeldung',
		text: 'Bei der Anmeldung brauchst du zusätzlich den sechsstelligen Code deiner Authenticator-App.',
		remaining: 'Unbenutzte Wiederherstellungscodes',
		lowCodes: {
			title: 'Nur noch wenige Wiederherstellungscodes',
			text: 'Erstelle neue, bevor sie aufgebraucht sind.'
		},
		enable: 'Aktivieren',
		regenerate: 'Neue Wiederherstellungscodes',
		disable: 'Deaktivieren',
		enabled: 'Zwei-Faktor-Anmeldung aktiviert',
		disabled: 'Zwei-Faktor-Anmeldung deaktiviert',
		regenerated: 'Neue Wiederherstellungscodes erstellt'
	},
	enroll: {
		step: '{current} / {total}',
		stepLabel: 'Schritt {current} von {total}',
		password: {
			title: 'Passwort bestätigen',
			text: 'Gib dein aktuelles Passwort ein, um die Zwei-Faktor-Anmeldung zu aktivieren.',
			label: 'Aktuelles Passwort',
			submit: 'Passwort bestätigen'
		},
		scan: {
			title: 'Scanne den QR-Code',
			text: 'Öffne deine Authenticator-App und scanne den Code. Gib danach den Code ein, den die App anzeigt.',
			qr: 'QR-Code für deine Authenticator-App',
			manual: 'Zur manuellen Eingabe:',
			copySecret: 'Geheimen Schlüssel kopieren',
			secretCopied: 'Geheimer Schlüssel kopiert',
			expires: 'Diese Einrichtung gilt bis {time}',
			code: 'Bestätigungscode',
			codeHelp: 'Der sechsstellige Code aus deiner App.',
			submit: 'Code prüfen'
		},
		codes: {
			title: 'Speichere deine Wiederherstellungscodes'
		}
	},
	recovery: {
		text: 'Wenn du deine Authenticator-App nicht erreichst, meldest du dich mit einem dieser Codes an. Jeder Code gilt einmal.',
		listLabel: 'Wiederherstellungscodes',
		printHeading: 'Kinex Mark Wiederherstellungscodes: {username}',
		copy: 'Codes kopieren',
		copied: 'Codes kopiert',
		download: 'Als .txt herunterladen',
		print: 'Drucken',
		saved: 'Ich habe die Codes gespeichert und weiß, dass sie nur jetzt angezeigt werden.',
		file: {
			title: 'Kinex Mark Wiederherstellungscodes',
			intro:
				'Konto: {username}. Jeder Code kann einmal anstelle eines Bestätigungscodes verwendet werden.',
			generatedAt: 'Erstellt: {time}',
			warning:
				'Bewahre sie sicher auf. Wer dein Passwort und einen Code kennt, kann sich bei deinem Konto anmelden.'
		}
	},
	confirm: {
		password: 'Aktuelles Passwort',
		disable: {
			title: 'Zwei-Faktor-Anmeldung deaktivieren?',
			text: 'Danach brauchst du zur Anmeldung nur noch dein Passwort. Deine Wiederherstellungscodes werden gelöscht, und auf deinen anderen Geräten wirst du abgemeldet.',
			code: 'Bestätigungs- oder Wiederherstellungscode',
			codeHelp:
				'Der sechsstellige Code aus deiner App oder ein unbenutzter Wiederherstellungscode.',
			submit: 'Zwei-Faktor-Anmeldung deaktivieren'
		},
		regenerate: {
			title: 'Neue Wiederherstellungscodes erstellen?',
			text: 'Deine alten Codes werden ungültig. Die neuen werden nur einmal angezeigt.',
			code: 'Bestätigungscode',
			codeHelp: 'Der sechsstellige Code aus deiner App.',
			submit: 'Neue Codes erstellen'
		}
	},
	sessions: {
		title: 'Sitzungen',
		text: 'Die Geräte, auf denen du angemeldet bist.',
		listLabel: 'Aktive Sitzungen',
		current: 'Dieses Gerät',
		unknownDevice: 'Unbekanntes Gerät',
		ip: 'IP-Adresse',
		created: 'Erstellt',
		lastSeen: 'Letzte Aktivität',
		expires: 'Läuft ab',
		revoke: 'Abmelden: {device}',
		revoked: 'Sitzung beendet: {device}',
		revokeOthers: 'Auf anderen Geräten abmelden',
		revokedOthers: {
			zero: 'Es gab keine anderen Sitzungen',
			one: '{count} Sitzung beendet',
			other: '{count} Sitzungen beendet'
		},
		revokeCurrent: {
			title: 'Auf diesem Gerät abmelden?',
			text: 'Diese Sitzung endet, und du musst dich erneut anmelden.',
			submit: 'Auf diesem Gerät abmelden'
		},
		empty: {
			title: 'Keine aktiven Sitzungen',
			text: 'Aktualisiere die Liste oder melde dich erneut an.'
		}
	},
	privacy: {
		notice: {
			title: 'Datenschutzhinweise',
			text: 'Mit der Bestätigung hältst du fest, dass du die Hinweise gelesen hast.',
			current: 'Aktuelle Version',
			updatedAt: 'Aktualisiert',
			version: 'Version {version}',
			history: 'Bestätigungen',
			acceptedAt: 'Bestätigt: {time}',
			currentBadge: 'Aktuell',
			none: {
				title: 'Noch keine Hinweise bestätigt',
				text: 'Öffne die Hinweise und bestätige sie.'
			},
			open: 'Hinweise öffnen',
			summary: 'Text der Hinweise'
		},
		export: {
			title: 'Deine Daten herunterladen',
			text: 'Die JSON-Datei enthält alle Daten, die zu deinem Konto gehören:',
			items: {
				account: 'Kontodaten',
				preferences: 'Einstellungen',
				sessions: 'Sitzungen',
				tags: 'Labels',
				links: 'deine eigenen Links mit ihren Freigaben',
				shared: 'dein persönlicher Status bei Links, die mit dir geteilt wurden',
				audit: 'Einträge des Sicherheitsprotokolls zu deinem Konto, als Handelnder und als Ziel',
				lock: 'der Status der Anmeldesperre'
			},
			excluded: 'Passwort, geheime Schlüssel und Wiederherstellungscodes sind nicht enthalten.',
			submit: 'Daten herunterladen (JSON)',
			done: 'Daten heruntergeladen'
		}
	},
	deleteAccount: {
		title: 'Konto löschen',
		text: 'Dein Konto und alle zugehörigen Daten werden endgültig gelöscht. Wenn du deine Links behalten willst, lade zuerst deine Daten herunter.',
		open: 'Konto löschen',
		dialogTitle: 'Dein Konto löschen?',
		dialogText:
			'Alle deine Links, Labels, Freigaben und Einstellungen werden endgültig gelöscht. Das lässt sich nicht rückgängig machen.',
		shared: {
			title: {
				one: '{count} geteilter Link verschwindet auch bei den Empfängern.',
				other: '{count} geteilte Links verschwinden auch bei den Empfängern.'
			},
			text: 'Wenn sie sie behalten wollen, bitte sie, die Links in ihre eigene Sammlung aufzunehmen.'
		},
		password: 'Passwort',
		code: 'Bestätigungs- oder Wiederherstellungscode',
		confirm: 'Gib zur Bestätigung deinen Benutzernamen ein: {username}',
		submit: 'Konto endgültig löschen'
	}
} satisfies Shape<typeof hu>;
