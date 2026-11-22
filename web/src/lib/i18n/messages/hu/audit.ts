export default {
	pageTitle: 'Audit napló',
	subtitle: 'A biztonsági események {days} napig maradnak meg.',
	filters: {
		label: 'Audit napló szűrése',
		event: 'Esemény',
		all: 'Mind',
		groupsHeading: 'Eseménycsoportok',
		eventsHeading: 'Egyes események',
		groups: {
			auth: 'Bejelentkezés',
			session: 'Munkamenetek',
			user: 'Felhasználók',
			settings: 'Beállítások',
			privacy: 'Adatvédelem',
			link: 'Megosztás'
		},
		result: 'Eredmény',
		actor: 'Szereplő',
		actorSearch: 'Felhasználó keresése…',
		actorEmpty: 'Nincs ilyen felhasználó.',
		actorUnavailable: 'A felhasználók listája nem tölthető be.',
		actorUnknown: 'Felhasználó #{id}',
		from: 'Ettől',
		to: 'Eddig',
		clear: 'Szűrők törlése'
	},
	table: {
		caption: 'Audit napló bejegyzései',
		at: 'Időpont',
		event: 'Esemény',
		result: 'Eredmény',
		actor: 'Szereplő',
		target: 'Cél',
		ip: 'IP',
		details: 'Részletek',
		showDetails: 'Részletek megjelenítése'
	},
	result: {
		success: 'Sikeres',
		failure: 'Sikertelen'
	},
	empty: {
		title: 'Nincs a szűrőknek megfelelő esemény.',
		text: 'Módosítsd vagy töröld a szűrőket.'
	},
	events: {
		bootstrap: {
			admin_created: 'Első adminisztrátor létrehozva'
		},
		auth: {
			login: {
				success: 'Sikeres bejelentkezés',
				failure: 'Sikertelen bejelentkezés',
				locked: 'Bejelentkezés zárolva'
			},
			logout: 'Kijelentkezés',
			recovery_code: {
				used: 'Helyreállító kód felhasználva'
			},
			password: {
				changed: 'Jelszó módosítva'
			},
			'2fa': {
				enabled: 'Kétlépcsős azonosítás bekapcsolva',
				disabled: 'Kétlépcsős azonosítás kikapcsolva',
				recovery_regenerated: 'Új helyreállító kódok'
			}
		},
		session: {
			revoked: 'Munkamenet lezárva',
			revoked_others: 'Többi munkamenet lezárva'
		},
		user: {
			created: 'Felhasználó létrehozva',
			updated: 'Felhasználó módosítva',
			deactivated: 'Felhasználó inaktiválva',
			activated: 'Felhasználó aktiválva',
			deleted: 'Felhasználó törölve',
			password_reset: 'Jelszó visszaállítva',
			'2fa_reset': 'Kétlépcsős azonosítás visszaállítva',
			unlocked: 'Zárolás feloldva',
			sessions_revoked: 'Munkamenetek lezárva',
			self_deleted: 'Saját fiók törölve'
		},
		settings: {
			updated: 'Rendszerbeállítás módosítva'
		},
		privacy: {
			notice_bumped: 'Új tájékoztató-verzió',
			consent_accepted: 'Tájékoztató elfogadva'
		},
		gdpr: {
			export: 'Adatexport'
		},
		link: {
			shared: 'Link megosztva',
			unshared: 'Megosztás visszavonva'
		}
	},
	targets: {
		user: 'Felhasználó',
		link: 'Link',
		session: 'Munkamenet',
		settings: 'Beállítások',
		tag: 'Címke'
	},
	details: {
		keys: {
			reason: 'Ok',
			method: 'Módszer',
			scope: 'Hatókör',
			changed: 'Módosított mezők',
			role: 'Szerepkör',
			active: 'Aktív',
			remaining: 'Megmaradt kódok',
			sessionId: 'Munkamenet',
			count: 'Darab',
			version: 'Verzió',
			targetType: 'Célpont típusa',
			targetRole: 'Célszerepkör',
			targetUserId: 'Célfelhasználó',
			generatedPassword: 'Generált jelszó',
			via: 'Forrás'
		},
		values: {
			reason: {
				credentials: 'Hibás jelszó',
				unknown_user: 'Ismeretlen felhasználónév',
				disabled: 'Inaktív fiók',
				totp: 'Hibás kód'
			},
			method: {
				password: 'Jelszó',
				totp: 'Hitelesítő kód',
				recovery_code: 'Helyreállító kód'
			},
			scope: {
				user: 'Felhasználó',
				ip: 'IP-cím'
			},
			targetType: {
				all: 'Mindenki',
				role: 'Szerepkör',
				user: 'Felhasználó'
			},
			via: {
				cli: 'Parancssor'
			}
		},
		yes: 'Igen',
		no: 'Nem',
		change: '{from} → {to}'
	}
};
