export default {
	actions: {
		save: 'Mentés',
		cancel: 'Mégse',
		close: 'Bezárás',
		delete: 'Törlés',
		edit: 'Szerkesztés',
		copy: 'Másolás',
		copied: 'Másolva',
		retry: 'Újrapróbálás',
		back: 'Vissza',
		done: 'Kész',
		confirm: 'Megerősítés',
		download: 'Letöltés',
		upload: 'Feltöltés',
		clear: 'Törlés',
		more: 'Továbbiak',
		open: 'Megnyitás'
	},
	states: {
		loading: 'Betöltés…',
		loadingMore: 'Továbbiak betöltése…',
		empty: 'Nincs megjeleníthető elem.',
		saving: 'Mentés…'
	},
	time: {
		today: 'ma',
		yesterday: 'tegnap',
		never: 'soha'
	},
	units: {
		days: { one: '{count} nap', other: '{count} nap' },
		minutes: { one: '{count} perc', other: '{count} perc' },
		hours: { one: '{count} óra', other: '{count} óra' }
	},
	missing: '–',
	notifications: 'Értesítések',
	skipToContent: 'Ugrás a tartalomra',
	appName: 'Kinex Mark',
	pageTitleSuffix: 'Kinex Mark',
	copyFailed: 'A másolás nem sikerült. Jelöld ki és másold kézzel.',
	password: {
		show: 'Jelszó megjelenítése',
		hide: 'Jelszó elrejtése'
	},
	keys: {
		ctrl: 'Ctrl',
		enter: 'Enter',
		delete: 'Delete',
		esc: 'Esc'
	},
	tagPicker: {
		label: 'Címkék',
		search: 'Címke keresése…',
		create: '„{name}” címke létrehozása',
		empty: 'Nincs ilyen címke.',
		limit: 'Legfeljebb {max} címke választható.',
		remove: 'Eltávolítás: {name}'
	},
	stepUp: {
		label: 'Saját jelszavad a megerősítéshez',
		help: 'A művelet a biztonsági naplóba kerül.'
	}
};
