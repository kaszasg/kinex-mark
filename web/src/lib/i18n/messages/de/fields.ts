import type hu from '../hu/fields.js';
import type { Shape } from '../../types.js';

export default {
	required: 'Pflichtfeld.',
	too_short: 'Zu kurz.',
	too_long: 'Zu lang.',
	invalid: 'Ungültiger Wert.',
	invalid_format: 'Ungültiges Format.',
	not_allowed: 'Dieser Wert ist nicht erlaubt.',
	out_of_range: 'Der Wert liegt außerhalb des zulässigen Bereichs.',
	unknown: 'Unbekannter Eintrag. Aktualisiere die Liste.',
	duplicate: 'Das gibt es bereits.',
	credentials_in_url: 'Die URL darf keinen Benutzernamen und kein Passwort enthalten.',
	incorrect: 'Falscher Wert.',
	password_too_short: 'Mindestens 12 Zeichen.',
	password_too_long: 'Höchstens 128 Zeichen.',
	password_contains_username: 'Das Passwort darf den Benutzernamen nicht enthalten.',
	password_too_simple: 'Mindestens 6 verschiedene Zeichen.',
	password_reused: 'Das neue Passwort muss sich vom aktuellen unterscheiden.',
	range: 'Der Wert muss zwischen {min} und {max} liegen.',
	byField: {
		url: {
			invalid_format: 'Gib eine vollständige Adresse ein, zum Beispiel https://example.com.',
			not_allowed: 'Nur http- und https-Adressen können gespeichert werden.',
			too_long: 'Die URL darf höchstens 2048 Byte lang sein.'
		},
		username: {
			invalid_format: 'Nur Buchstaben, Ziffern, Punkt, Bindestrich und Unterstrich.'
		},
		tagIds: {
			too_long: 'Höchstens 20 Labels.'
		},
		privacyControllerEmail: {
			invalid_format: 'Gib eine gültige E-Mail-Adresse ein.'
		}
	}
} satisfies Shape<typeof hu>;
