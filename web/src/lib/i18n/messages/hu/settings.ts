// Settings: profile and display tabs (plan 8.7.6, F3). Security and privacy live in `security`.
export default {
	pageTitle: '{section} – Beállítások',
	profile: {
		title: 'Profil',
		description: 'A megjelenített neved látják azok, akikkel linket osztasz meg.',
		username: 'Felhasználónév',
		usernameHelp: 'A felhasználónév nem módosítható.',
		displayName: 'Megjelenített név',
		displayNameHelp: 'Legfeljebb {max} karakter. Üresen hagyva a felhasználóneved látszik.',
		role: 'Szerepkör',
		createdAt: 'Fiók létrehozva',
		lastLoginAt: 'Utolsó bejelentkezés',
		save: 'Profil mentése',
		saved: 'Profil mentve'
	},
	display: {
		intro: 'Minden módosítás azonnal mentődik, és minden eszközödön érvényes.',
		appearance: {
			title: 'Nyelv és téma'
		},
		language: {
			label: 'Nyelv'
		},
		theme: {
			label: 'Téma',
			dark: 'Sötét',
			light: 'Világos',
			system: 'Rendszer szerint'
		},
		list: {
			title: 'Linklista',
			description: 'A Linkek oldal ezekkel az alapértékekkel nyílik meg.'
		},
		viewMode: {
			label: 'Alapnézet',
			list: 'Lista',
			tiles: 'Csempe'
		},
		fields: {
			label: 'Megjelenített adatok',
			icon: 'Ikon',
			url: 'URL',
			title: 'Cím',
			description: 'Leírás',
			help: 'A link felirata a leírás, ennek hiányában a cím, végül az URL. Kikapcsolt mezőt a felirat sem használ.'
		},
		preview: {
			label: 'Előnézet',
			title: 'Grafana',
			description: 'Éles műszerfal',
			url: 'grafana.example.net/d/abc'
		},
		archived: {
			label: 'Archivált linkek alapból',
			toggle: 'Archiváltak megjelenítése'
		},
		sort: {
			label: 'Rendezés alapból',
			usage: 'Használat szerint',
			created: 'Legújabb elöl',
			title: 'Név szerint'
		},
		tagMatch: {
			label: 'Címkeegyezés alapból',
			any: 'Bármelyik',
			all: 'Mind',
			help: 'Több kijelölt címkénél elég az egyik, vagy mindegyik kell.'
		},
		archive: {
			title: 'Automatikus archiválás',
			label: 'Archiválás tétlenség után',
			system: {
				zero: 'Rendszer-alapérték (soha)',
				one: 'Rendszer-alapérték ({count} nap)',
				other: 'Rendszer-alapérték ({count} nap)'
			},
			never: 'Soha',
			custom: 'Egyéni…',
			customLabel: 'Napok száma',
			customHelp: '{min}–{max} nap. Enterrel vagy a mező elhagyásával mented.',
			help: 'A kitűzött linkek soha nem archiválódnak. Megnyitás, másolás, kitűzés vagy visszaállítás után a számlálás újraindul.',
			current: {
				zero: 'Jelenleg: nincs automatikus archiválás.',
				one: 'Jelenleg: {count} nap tétlenség után.',
				other: 'Jelenleg: {count} nap tétlenség után.'
			}
		}
	}
};
