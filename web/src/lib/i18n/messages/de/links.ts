import type hu from '../hu/links.js';
import type { Shape } from '../../types.js';

export default {
	pageTitle: 'Links',
	summary: {
		zero: 'Keine Links',
		one: '{count} Link, davon {pinned} angeheftet',
		other: '{count} Links, davon {pinned} angeheftet'
	},
	summaryArchived: {
		zero: 'Keine Links',
		one: '{count} Link, davon {pinned} angeheftet, {archived} archiviert',
		other: '{count} Links, davon {pinned} angeheftet, {archived} archiviert'
	},
	results: {
		zero: 'Keine Treffer',
		one: '{count} Treffer',
		other: '{count} Treffer'
	},
	toolbar: {
		label: 'Listenoptionen',
		add: 'Link hinzufügen',
		scope: 'Bereich',
		sort: 'Sortierung',
		view: 'Ansicht',
		display: 'Anzeige',
		shortcuts: 'Tastenkürzel'
	},
	scope: {
		all: 'Alle',
		own: 'Eigene',
		shared: 'Mit mir geteilt'
	},
	sort: {
		usage: 'Nach Nutzung',
		created: 'Neueste zuerst',
		title: 'Nach Name'
	},
	view: {
		list: 'Liste',
		tiles: 'Kacheln'
	},
	fields: {
		icon: 'Symbol',
		url: 'URL',
		title: 'Titel',
		description: 'Beschreibung'
	},
	archivedToggle: 'Archivierte anzeigen',
	filters: {
		label: 'Labelfilter',
		untagged: 'Ohne Label',
		more: 'Weitere Labels ({count})',
		search: 'Label suchen…',
		empty: 'Kein solches Label.',
		clear: 'Filter zurücksetzen',
		match: {
			label: 'Labelabgleich',
			any: 'Beliebiges',
			all: 'Alle'
		}
	},
	groups: {
		pinned: 'Angeheftet',
		others: 'Weitere'
	},
	list: {
		label: 'Links',
		allLoaded: 'Alle Links geladen.'
	},
	row: {
		pinned: 'Angeheftet',
		sharedBy: 'Geteilt von {owner}',
		archived: 'Archiviert',
		pending: 'Titel und Symbol werden abgerufen',
		uses: {
			one: '{count} Nutzung (geöffnet oder kopiert)',
			other: '{count} Nutzungen (geöffnet oder kopiert)'
		},
		lastUsed: 'Zuletzt genutzt: {time}',
		neverUsed: 'Noch nicht genutzt',
		moreTags: 'Weitere Labels: {names}',
		menu: 'Aktionen: {label}'
	},
	actions: {
		add: 'Link hinzufügen',
		open: 'Öffnen',
		copyUrl: 'URL kopieren',
		copyMarkdown: 'Als Markdown kopieren',
		details: 'Details',
		edit: 'Bearbeiten',
		tags: 'Labels',
		share: 'Teilen',
		refresh: 'Titel und Symbol aktualisieren',
		pin: 'Anheften',
		unpin: 'Lösen',
		archive: 'Archivieren',
		restore: 'Wiederherstellen',
		delete: 'Löschen'
	},
	dialog: {
		addTitle: 'Link hinzufügen',
		editTitle: 'Link bearbeiten',
		addText: 'Die URL genügt, Titel und Symbol kommen im Hintergrund.',
		editText: 'Änderungen sehen alle, mit denen du den Link geteilt hast.',
		url: 'URL',
		title: 'Titel',
		titleHelp: 'Leer lassen, um den abgerufenen Titel zu zeigen.',
		description: 'Kurzbeschreibung',
		descriptionHelp: 'Die Liste zeigt diesen Text zuerst.',
		counter: '{count} / 500',
		tags: 'Labels',
		pin: 'Oben in der Liste anheften',
		save: 'Link speichern',
		showExisting: 'In der Liste zeigen'
	},
	tagsDialog: {
		title: 'Labels bearbeiten',
		text: 'Labels sind persönlich: Nur du siehst sie, auch auf geteilten Links.',
		save: 'Labels speichern'
	},
	details: {
		title: 'Linkdetails',
		sections: {
			data: 'Daten',
			usage: 'Nutzung',
			times: 'Zeitpunkte',
			metadata: 'Metadaten',
			sharing: 'Freigabe'
		},
		fields: {
			url: 'URL',
			title: 'Titel',
			description: 'Beschreibung',
			tags: 'Labels',
			owner: 'Eigentümer'
		},
		noTags: 'Keine Labels',
		editTags: 'Labels bearbeiten',
		metrics: {
			usage: 'Nutzung',
			archive: 'Archivierung',
			created: 'Hinzugefügt'
		},
		neverUsed: 'Noch nicht genutzt',
		never: 'Nie',
		archivedNote: 'Bereits archiviert',
		dueIn: {
			one: 'in {count} Tag',
			other: 'in {count} Tagen'
		},
		dueToday: 'heute fällig',
		times: {
			created: 'Hinzugefügt',
			updated: 'Geändert',
			lastUsed: 'Zuletzt genutzt',
			sharedAt: 'Mit dir geteilt',
			inactiveSince: 'Inaktiv seit',
			archivedAt: 'Archiviert'
		},
		archiveReason: {
			auto: 'automatisch',
			manual: 'manuell'
		},
		archivedWith: '{date}, {reason}',
		fetchedAt: 'Abgerufen am',
		attempts: 'Versuche',
		manageShares: 'Freigabe verwalten',
		deleteLink: 'Link löschen'
	},
	share: {
		title: 'Link teilen',
		text: 'Empfänger sehen den Link in ihrer eigenen Liste, mit eigenen Labels und eigenem Anheften. Ändern kannst nur du ihn.',
		current: 'Aktuelle Freigaben',
		empty: 'Noch mit niemandem geteilt.',
		everyone: 'Alle',
		roleTarget: 'Rolle: {role}',
		revoke: 'Freigabe entfernen: {target}',
		newShare: 'Neue Freigabe',
		type: {
			label: 'Empfänger',
			all: 'Alle',
			role: 'Rolle',
			user: 'Benutzer'
		},
		role: 'Rolle',
		rolePlaceholder: 'Wähle eine Rolle',
		user: 'Benutzer',
		userPlaceholder: 'Wähle einen Benutzer',
		userSearch: 'Benutzer suchen…',
		userEmpty: 'Kein solcher Benutzer.',
		add: 'Freigabe hinzufügen'
	},
	delete: {
		title: 'Diesen Link löschen?',
		text: 'Dieser Link wird endgültig gelöscht: „{label}“. Seine Freigaben und die Nutzungsdaten aller Benutzer werden ebenfalls gelöscht. Das lässt sich nicht rückgängig machen.',
		confirm: 'Link löschen'
	},
	importExport: {
		label: 'Import und Export',
		exportJson: 'Als JSON exportieren',
		exportHtml: 'Als Browser-HTML exportieren',
		import: 'Aus Datei importieren…',
		resultTitle: 'Import abgeschlossen',
		resultText: 'Titel und Symbole der neuen Links kommen im Hintergrund.',
		total: 'Gesamt',
		created: 'Neu',
		duplicates: 'Schon vorhanden',
		invalid: 'Ungültig'
	},
	expiry: {
		label: 'Wird um den {date} archiviert, wenn du ihn bis dahin nicht öffnest oder kopierst',
		daysLeft: { one: '{count} Tag', other: '{count} Tage' },
		today: 'heute'
	},
	empty: {
		collection: {
			title: 'Noch keine Links.',
			text: 'Zieh einen hierher, füge ihn aus der Zwischenablage ein ({shortcut}) oder lege ihn von Hand an.'
		},
		filtered: {
			title: 'Keine Treffer.',
			text: 'Ändere die Suche oder setze die Filter zurück.'
		},
		archived: {
			title: 'Keine archivierten Links.',
			text: 'Hier erscheinen Links, die du archivierst oder lange nicht genutzt hast.'
		}
	},
	toast: {
		saved: 'Link gespeichert',
		deleted: 'Link gelöscht',
		pinned: 'Link angeheftet',
		unpinned: 'Link gelöst',
		pinnedRestored: 'Link angeheftet und wiederhergestellt',
		archived: 'Link archiviert',
		restored: 'Link wiederhergestellt',
		undo: 'Rückgängig',
		urlCopied: 'URL kopiert',
		markdownCopied: 'Markdown-Link kopiert',
		refreshing: 'Titel und Symbol werden aktualisiert',
		tagsSaved: 'Labels gespeichert',
		shared: 'Link geteilt',
		unshared: 'Freigabe entfernt',
		exported: 'Export heruntergeladen'
	},
	shortcuts: {
		title: 'Tastenkürzel',
		text: 'In Eingabefeldern und bei geöffneten Dialogen sind die Kürzel aus.',
		or: 'oder',
		items: {
			search: 'Suchfeld fokussieren',
			add: 'Link hinzufügen',
			paste: 'Schnellerfassung aus der Zwischenablage',
			next: 'Nächster Link',
			previous: 'Vorheriger Link',
			open: 'Ausgewählten Link öffnen',
			pin: 'Anheften umschalten',
			copyUrl: 'URL kopieren',
			copyMarkdown: 'Als Markdown kopieren',
			details: 'Details',
			edit: 'Bearbeiten',
			archive: 'Archivieren oder wiederherstellen',
			delete: 'Löschen mit Bestätigung',
			view: 'Zwischen Liste und Kacheln wechseln',
			help: 'Tastenkürzel',
			clear: 'Suchfeld leeren'
		}
	}
} satisfies Shape<typeof hu>;
