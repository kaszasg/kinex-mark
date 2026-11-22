// Tags page, tag dialog and tag deletion (plan 8.7.5, F3).
export default {
	pageTitle: 'Címkék',
	summary: {
		one: '{count} címke. A címkék személyesek: a veled megosztott linkekre is a saját címkéidet teheted.',
		other:
			'{count} címke. A címkék személyesek: a veled megosztott linkekre is a saját címkéidet teheted.'
	},
	add: 'Címke felvétele',
	table: {
		caption: 'Címkék név szerint',
		name: 'Név',
		color: 'Szín',
		links: 'Linkek',
		active: 'Aktív',
		updated: 'Módosítva',
		actions: 'Műveletek',
		showLinks: {
			one: '{count} link ezzel a címkével: {name}',
			other: '{count} link ezzel a címkével: {name}'
		},
		edit: 'Szerkesztés: {name}',
		delete: 'Törlés: {name}'
	},
	dialog: {
		createTitle: 'Címke felvétele',
		editTitle: 'Címke szerkesztése',
		description: 'A címke csak a te gyűjteményedben látszik.',
		name: 'Név',
		nameHelp: 'Legfeljebb {max} karakter.',
		color: 'Szín',
		preview: 'Előnézet',
		save: 'Címke mentése',
		saved: 'Címke mentve'
	},
	delete: {
		title: 'Törlöd ezt a címkét?',
		text: {
			zero: 'A következő címke törlődik: „{name}”. Egyik linkeden sincs rajta.',
			one: 'A következő címke törlődik: „{name}”. {count} linkről kerül le, a linkek megmaradnak.',
			other: 'A következő címke törlődik: „{name}”. {count} linkről kerül le, a linkek megmaradnak.'
		},
		confirm: 'Címke törlése',
		deleted: 'Címke törölve'
	},
	empty: {
		title: 'Még nincs címke.',
		text: 'Vegyél fel egyet, és csoportosítsd vele a linkjeidet.'
	}
};
