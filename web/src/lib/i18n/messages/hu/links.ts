export default {
	pageTitle: 'Linkek',
	summary: {
		zero: 'Nincs link',
		one: '{count} link, ebből {pinned} kitűzött',
		other: '{count} link, ebből {pinned} kitűzött'
	},
	summaryArchived: {
		zero: 'Nincs link',
		one: '{count} link, ebből {pinned} kitűzött, {archived} archivált',
		other: '{count} link, ebből {pinned} kitűzött, {archived} archivált'
	},
	results: {
		zero: 'Nincs találat',
		one: '{count} találat',
		other: '{count} találat'
	},
	toolbar: {
		label: 'Lista beállításai',
		add: 'Link felvétele',
		scope: 'Hatókör',
		sort: 'Rendezés',
		view: 'Nézet',
		display: 'Megjelenítés',
		shortcuts: 'Billentyűparancsok'
	},
	scope: {
		all: 'Mind',
		own: 'Saját',
		shared: 'Megosztott velem'
	},
	sort: {
		usage: 'Használat szerint',
		created: 'Legújabb elöl',
		title: 'Név szerint'
	},
	view: {
		list: 'Lista',
		tiles: 'Csempe'
	},
	fields: {
		icon: 'Ikon',
		url: 'URL',
		title: 'Cím',
		description: 'Leírás'
	},
	archivedToggle: 'Archiváltak is',
	filters: {
		label: 'Címkeszűrő',
		untagged: 'Címke nélkül',
		more: 'További címkék ({count})',
		search: 'Címke keresése…',
		empty: 'Nincs ilyen címke.',
		clear: 'Szűrők törlése',
		match: {
			label: 'Címkék egyezése',
			any: 'Bármelyik',
			all: 'Mind'
		}
	},
	groups: {
		pinned: 'Kitűzött',
		others: 'Többi'
	},
	list: {
		label: 'Linkek',
		allLoaded: 'Minden link betöltve.'
	},
	row: {
		pinned: 'Kitűzve',
		sharedBy: 'Megosztotta: {owner}',
		archived: 'Archivált',
		pending: 'Cím és ikon letöltése folyamatban',
		uses: {
			one: '{count} használat (megnyitás vagy másolás)',
			other: '{count} használat (megnyitás vagy másolás)'
		},
		lastUsed: 'Utolsó használat: {time}',
		neverUsed: 'Még nem használtad',
		moreTags: 'További címkék: {names}',
		menu: 'Műveletek: {label}'
	},
	actions: {
		add: 'Link felvétele',
		open: 'Megnyitás',
		copyUrl: 'URL másolása',
		copyMarkdown: 'Másolás Markdownként',
		details: 'Részletek',
		edit: 'Szerkesztés',
		tags: 'Címkék',
		share: 'Megosztás',
		refresh: 'Cím és ikon frissítése',
		pin: 'Kitűzés',
		unpin: 'Kitűzés megszüntetése',
		archive: 'Archiválás',
		restore: 'Visszaállítás',
		delete: 'Törlés'
	},
	dialog: {
		addTitle: 'Link felvétele',
		editTitle: 'Link szerkesztése',
		addText: 'Elég az URL, a cím és az ikon a háttérben érkezik.',
		editText: 'A módosítás mindenkinél látszik, akivel a linket megosztottad.',
		url: 'URL',
		title: 'Cím',
		titleHelp: 'Üresen hagyva a letöltött cím jelenik meg.',
		description: 'Rövid leírás',
		descriptionHelp: 'A listában ez jelenik meg elsőként.',
		counter: '{count} / 500',
		tags: 'Címkék',
		pin: 'Kitűzés a lista elejére',
		save: 'Link mentése',
		showExisting: 'Megjelenítés a listában'
	},
	tagsDialog: {
		title: 'Címkék szerkesztése',
		text: 'A címkék személyesek: csak te látod őket, a megosztott linkeken is.',
		save: 'Címkék mentése'
	},
	details: {
		title: 'Link részletei',
		sections: {
			data: 'Adatok',
			usage: 'Használat',
			times: 'Időpontok',
			metadata: 'Metaadat',
			sharing: 'Megosztás'
		},
		fields: {
			url: 'URL',
			title: 'Cím',
			description: 'Leírás',
			tags: 'Címkék',
			owner: 'Tulajdonos'
		},
		noTags: 'Nincs címke',
		editTags: 'Címkék szerkesztése',
		metrics: {
			usage: 'Használat',
			archive: 'Archiválás',
			created: 'Rögzítve'
		},
		neverUsed: 'Még nem használtad',
		never: 'Soha',
		archivedNote: 'Már archiválva',
		dueIn: {
			one: '{count} nap múlva',
			other: '{count} nap múlva'
		},
		dueToday: 'ma esedékes',
		times: {
			created: 'Rögzítve',
			updated: 'Módosítva',
			lastUsed: 'Utolsó használat',
			sharedAt: 'Megosztva veled',
			inactiveSince: 'Tétlen ettől',
			archivedAt: 'Archiválva'
		},
		archiveReason: {
			auto: 'automatikusan',
			manual: 'kézzel'
		},
		archivedWith: '{date}, {reason}',
		fetchedAt: 'Letöltés ideje',
		attempts: 'Próbálkozások',
		manageShares: 'Megosztás kezelése',
		deleteLink: 'Link törlése'
	},
	share: {
		title: 'Link megosztása',
		text: 'A címzettek a saját listájukban látják a linket, saját címkékkel és saját kitűzéssel. Módosítani csak te tudod.',
		current: 'Jelenlegi megosztások',
		empty: 'Még senkivel nincs megosztva.',
		everyone: 'Mindenki',
		roleTarget: 'Szerepkör: {role}',
		revoke: 'Megosztás visszavonása: {target}',
		newShare: 'Új megosztás',
		type: {
			label: 'Címzett',
			all: 'Mindenki',
			role: 'Szerepkör',
			user: 'Felhasználó'
		},
		role: 'Szerepkör',
		rolePlaceholder: 'Válassz szerepkört',
		user: 'Felhasználó',
		userPlaceholder: 'Válassz felhasználót',
		userSearch: 'Felhasználó keresése…',
		userEmpty: 'Nincs ilyen felhasználó.',
		add: 'Megosztás hozzáadása'
	},
	delete: {
		title: 'Törlöd ezt a linket?',
		text: 'A következő link végleg törlődik: „{label}”. A megosztásai és minden felhasználó használati adatai is törlődnek. A művelet nem vonható vissza.',
		confirm: 'Link törlése'
	},
	importExport: {
		label: 'Import és export',
		exportJson: 'Exportálás JSON-ként',
		exportHtml: 'Exportálás böngészős HTML-ként',
		import: 'Importálás fájlból…',
		resultTitle: 'Importálás kész',
		resultText: 'Az új linkek címe és ikonja a háttérben érkezik.',
		total: 'Összesen',
		created: 'Új',
		duplicates: 'Már megvolt',
		invalid: 'Hibás'
	},
	expiry: {
		label: 'Archiválás {date} körül, ha addig nem nyitod meg és nem másolod',
		daysLeft: { one: '{count} nap', other: '{count} nap' },
		today: 'ma'
	},
	empty: {
		collection: {
			title: 'Még nincs link.',
			text: 'Húzz ide egyet, illeszd be a vágólapról ({shortcut}), vagy vedd fel kézzel.'
		},
		filtered: {
			title: 'Nincs találat.',
			text: 'Módosítsd a keresést, vagy töröld a szűrőket.'
		},
		archived: {
			title: 'Nincs archivált link.',
			text: 'Ide kerülnek a kézzel archivált és a régóta nem használt linkek.'
		}
	},
	toast: {
		saved: 'Link mentve',
		deleted: 'Link törölve',
		pinned: 'Link kitűzve',
		unpinned: 'Kitűzés megszüntetve',
		pinnedRestored: 'Link kitűzve és visszaállítva',
		archived: 'Link archiválva',
		restored: 'Link visszaállítva',
		undo: 'Visszavonás',
		urlCopied: 'URL másolva',
		markdownCopied: 'Markdown-link másolva',
		refreshing: 'A cím és az ikon frissítése elindult',
		tagsSaved: 'Címkék mentve',
		shared: 'Link megosztva',
		unshared: 'Megosztás visszavonva',
		exported: 'Export letöltve'
	},
	shortcuts: {
		title: 'Billentyűparancsok',
		text: 'Szövegmezőben és nyitott dialógusnál a parancsok nem működnek.',
		or: 'vagy',
		items: {
			search: 'Fókusz a keresőmezőre',
			add: 'Link felvétele',
			paste: 'Gyors rögzítés a vágólapról',
			next: 'Következő link',
			previous: 'Előző link',
			open: 'A kijelölt link megnyitása',
			pin: 'Kitűzés váltása',
			copyUrl: 'URL másolása',
			copyMarkdown: 'Másolás Markdownként',
			details: 'Részletek',
			edit: 'Szerkesztés',
			archive: 'Archiválás vagy visszaállítás',
			delete: 'Törlés megerősítéssel',
			view: 'Lista és csempe váltása',
			help: 'Billentyűparancsok',
			clear: 'Keresés törlése a keresőmezőben'
		}
	}
};
