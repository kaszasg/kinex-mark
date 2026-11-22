import type hu from '../hu/fetch.js';
import type { Shape } from '../../types.js';

export default {
	status: {
		pending: 'Wird abgerufen',
		ok: 'Abgerufen',
		failed: 'Fehlgeschlagen',
		blocked: 'Gesperrte Adresse'
	},
	errors: {
		timeout: 'Zeitüberschreitung',
		dns: 'Domain nicht gefunden',
		connection: 'Verbindungsfehler',
		tls: 'Zertifikatsfehler',
		http_status: 'Die Website hat einen Fehler gemeldet',
		too_large: 'Die Seite ist zu groß',
		not_html: 'Keine Webseite (kein Titel)',
		too_many_redirects: 'Zu viele Weiterleitungen',
		blocked_address: 'Interne Netzwerkadresse, Abruf gesperrt',
		blocked_port: 'Port nicht erlaubt'
	}
} satisfies Shape<typeof hu>;
