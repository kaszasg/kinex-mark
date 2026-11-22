import type hu from '../hu/nav.js';
import type { Shape } from '../../types.js';

export default {
	groups: {
		collection: 'Sammlung',
		frequentTags: 'Häufige Labels',
		account: 'Konto',
		admin: 'Verwaltung'
	},
	links: 'Links',
	tags: 'Labels',
	settings: 'Einstellungen',
	settingsTabs: {
		label: 'Reiter der Einstellungen',
		profile: 'Profil',
		security: 'Sicherheit',
		display: 'Darstellung',
		privacy: 'Datenschutz'
	},
	adminUsers: 'Benutzer',
	adminSettings: 'Systemeinstellungen',
	adminAudit: 'Audit-Protokoll',
	logout: 'Abmelden',
	toggleSidebar: 'Seitenleiste öffnen',
	sidebarTitle: 'Navigation',
	breadcrumb: {
		label: 'Brotkrümelnavigation',
		collection: 'Sammlung',
		admin: 'Verwaltung',
		settings: 'Einstellungen'
	},
	search: {
		label: 'Links durchsuchen',
		placeholder: 'Suchen…',
		open: 'Suche öffnen',
		close: 'Suche schließen'
	},
	theme: {
		toggle: 'Design wechseln'
	},
	locale: {
		label: 'Sprache',
		hu: 'Magyar',
		en: 'English',
		de: 'Deutsch'
	},
	userMenu: '{name} {role}, Benutzermenü',
	privacyNotice: 'Datenschutzhinweise',
	errorPage: {
		notFound: {
			title: 'Seite nicht gefunden.',
			text: 'Prüfe die Adresse oder kehre zu deinen Links zurück.',
			action: 'Zu den Links'
		},
		forbidden: {
			title: 'Für diese Seite fehlt dir die Berechtigung.',
			text: 'Die Verwaltungsseiten sieht nur ein Administrator.',
			action: 'Zu den Links'
		}
	}
} satisfies Shape<typeof hu>;
