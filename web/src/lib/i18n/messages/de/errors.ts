import type hu from '../hu/errors.js';
import type { Shape } from '../../types.js';

export default {
	endpoint_not_found: { title: 'Diese Funktion ist nicht verfügbar.', text: 'Lade die Seite neu.' },
	invalid_json: {
		title: 'Die Anfrage war fehlerhaft.',
		text: 'Lade die Seite neu und versuche es erneut.'
	},
	invalid_id: { title: 'Ungültige Kennung.', text: 'Öffne den Eintrag erneut aus der Liste.' },
	invalid_query: {
		title: 'Ungültiger Filter.',
		text: 'Setze die Filter zurück und versuche es erneut.'
	},
	validation_failed: {
		title: 'Einige Felder sind ungültig.',
		text: 'Korrigiere die markierten Felder.'
	},
	payload_too_large: {
		title: 'Die Datei ist größer als 2 MB.',
		text: 'Wähle eine kleinere Datei.'
	},
	not_found: {
		title: 'Der Eintrag wurde nicht gefunden.',
		text: 'Er wurde vielleicht gelöscht oder ist für dich nicht mehr sichtbar. Aktualisiere die Liste.'
	},
	unauthenticated: { title: 'Deine Sitzung ist abgelaufen.', text: 'Melde dich erneut an.' },
	forbidden: {
		title: 'Dafür fehlt dir die Berechtigung.',
		text: 'Wende dich an einen Administrator.'
	},
	csrf_rejected: {
		title: 'Eine Sicherheitsprüfung hat die Anfrage abgelehnt.',
		text: 'Lade die Seite neu und versuche es erneut.'
	},
	host_mismatch: {
		title: 'Die Anfrage hat nicht die eingestellte Kinex-Mark-Adresse erreicht.',
		text: 'Öffne die App unter ihrer gewohnten Adresse oder wende dich an einen Administrator.'
	},
	password_change_required: {
		title: 'Passwortänderung erforderlich.',
		text: 'Lege ein neues Passwort fest, um fortzufahren.'
	},
	consent_required: {
		title: 'Die Datenschutzhinweise müssen bestätigt werden.',
		text: 'Bestätige sie, um fortzufahren.'
	},
	invalid_credentials: {
		title: 'Benutzername oder Passwort ist falsch.',
		text: 'Prüfe die Eingaben und versuche es erneut.'
	},
	account_disabled: {
		title: 'Dieses Konto ist deaktiviert.',
		text: 'Wende dich an einen Administrator.'
	},
	too_many_attempts: {
		title: 'Zu viele fehlgeschlagene Versuche.',
		text: 'Versuche es in {minutes} Min. erneut.'
	},
	challenge_invalid: {
		title: 'Die Anmeldeanfrage ist abgelaufen.',
		text: 'Melde dich erneut an.'
	},
	invalid_totp: {
		title: 'Der Code ist falsch oder wurde schon verwendet.',
		text: 'Gib den neuen Code aus deiner App ein.'
	},
	current_password_invalid: {
		title: 'Das aktuelle Passwort ist falsch.',
		text: 'Gib es erneut ein.'
	},
	totp_already_enabled: {
		title: 'Die Zwei-Faktor-Anmeldung ist bereits aktiv.',
		text: 'Lade die Seite neu.'
	},
	totp_not_enabled: {
		title: 'Die Zwei-Faktor-Anmeldung ist nicht aktiv.',
		text: 'Lade die Seite neu.'
	},
	totp_setup_required: { title: 'Die Einrichtung ist abgelaufen.', text: 'Starte sie erneut.' },
	link_duplicate: {
		title: 'Dieser Link ist bereits in deiner Sammlung.',
		text: 'Öffne den vorhandenen.'
	},
	link_limit_reached: {
		title: 'Du hast das Limit von 10.000 Links erreicht.',
		text: 'Lösche alte Links, bevor du neue hinzufügst.'
	},
	not_owner: {
		title: 'Nur der Eigentümer kann diesen Link ändern.',
		text: 'Eigene Labels und Anheften kannst du trotzdem setzen.'
	},
	share_not_allowed: {
		title: 'Deine Rolle darf keine Links teilen.',
		text: 'Wende dich an einen Entwickler oder Administrator.'
	},
	share_duplicate: {
		title: 'Bereits mit diesem Empfänger geteilt.',
		text: 'Wähle einen anderen Empfänger.'
	},
	no_url_found: {
		title: 'Keine URL gefunden.',
		text: 'Zieh oder füge einen Link ein, der mit http oder https beginnt.'
	},
	import_invalid: {
		title: 'Die Datei konnte nicht gelesen werden.',
		text: 'Wähle einen Kinex-Mark-JSON-Export oder eine HTML-Lesezeichendatei des Browsers.'
	},
	import_too_many_entries: {
		title: 'Die Datei enthält mehr als 5.000 Einträge.',
		text: 'Teile sie auf mehrere Dateien auf und importiere sie nacheinander.'
	},
	tag_duplicate: {
		title: 'Ein Label mit diesem Namen gibt es bereits.',
		text: 'Wähle einen anderen Namen.'
	},
	tag_limit_reached: {
		title: 'Du hast das Limit von 500 Labels erreicht.',
		text: 'Lösche ungenutzte Labels.'
	},
	username_taken: { title: 'Dieser Benutzername ist vergeben.', text: 'Wähle einen anderen.' },
	last_admin: {
		title: 'Das ist der letzte aktive Administrator.',
		text: 'Ernenne zuerst einen weiteren Administrator.'
	},
	self_modification: {
		title: 'Dein eigenes Konto kannst du hier nicht ändern.',
		text: 'Deine Rolle kann ein anderer Administrator ändern, dein Konto löschst du in den Einstellungen.'
	},
	setting_locked: {
		title: 'Diese Einstellung ist durch eine Umgebungsvariable festgelegt.',
		text: 'Ändere dafür die Serverkonfiguration: {keys}.'
	},
	consent_version_mismatch: {
		title: 'Inzwischen gibt es neue Hinweise.',
		text: 'Lies sie und bestätige sie erneut.'
	},
	confirmation_mismatch: {
		title: 'Die Bestätigung stimmt nicht mit deinem Benutzernamen überein.',
		text: 'Gib ihn genau ein.'
	},
	not_implemented: {
		title: 'Diese Funktion ist noch nicht verfügbar.',
		text: 'Versuche es in einer späteren Version.'
	},
	internal_error: {
		title: 'Ein unerwarteter Serverfehler ist aufgetreten.',
		text: 'Versuche es später erneut.'
	},
	network_error: {
		title: 'Der Server ist nicht erreichbar.',
		text: 'Prüfe deine Netzwerkverbindung.'
	}
} satisfies Shape<typeof hu>;
