export default {
	users: {
		pageTitle: 'Felhasználók',
		summary: { one: '{count} felhasználó', other: '{count} felhasználó' },
		add: 'Felhasználó felvétele',
		filters: {
			label: 'Felhasználók szűrése',
			search: 'Keresés',
			searchPlaceholder: 'Felhasználónév vagy név',
			role: 'Szerepkör',
			allRoles: 'Mind',
			status: 'Állapot',
			statusActive: 'Aktív',
			statusInactive: 'Inaktív',
			statusAll: 'Mind',
			clear: 'Szűrők törlése'
		},
		table: {
			caption: 'Felhasználók listája',
			username: 'Felhasználónév',
			displayName: 'Megjelenített név',
			role: 'Szerepkör',
			status: 'Állapot',
			twoFactor: '2FA',
			twoFactorOn: 'Be',
			links: 'Linkek',
			sessions: 'Munkamenetek',
			lastLogin: 'Utolsó belépés',
			actions: 'Műveletek'
		},
		actions: {
			menu: '{username} műveletei',
			edit: 'Szerkesztés',
			resetPassword: 'Jelszó visszaállítása',
			resetTwoFactor: '2FA visszaállítása',
			unlock: 'Feloldás',
			revokeSessions: 'Munkamenetek lezárása',
			deactivate: 'Inaktiválás',
			activate: 'Aktiválás',
			delete: 'Törlés',
			selfHint: 'A saját fiókodat a Beállítások oldalon kezeled.'
		},
		confirm: {
			resetPassword: {
				title: 'Visszaállítod {username} jelszavát?',
				text: 'Minden munkamenete lezárul, és új, ideiglenes jelszót kap. A művelet a biztonsági naplóba kerül, és a felhasználó a következő belépéskor jelszót cserél.',
				action: 'Jelszó visszaállítása'
			},
			resetTwoFactor: {
				title: 'Visszaállítod {username} kétlépcsős azonosítását?',
				text: 'Minden munkamenete lezárul, a kétlépcsős azonosítást újra be kell kapcsolnia.',
				action: '2FA visszaállítása'
			},
			deactivate: {
				title: 'Inaktiválod {username} fiókját?',
				text: 'Minden munkamenete lezárul. A megosztásai érvényben maradnak.',
				action: 'Fiók inaktiválása'
			},
			activate: {
				title: 'Aktiválod {username} fiókját?',
				text: 'A felhasználó újra bejelentkezhet a meglévő jelszavával.',
				action: 'Fiók aktiválása'
			},
			delete: {
				title: 'Törlöd {username} fiókját?',
				text: 'A linkjei, címkéi és megosztásai végleg törlődnek; az audit naplóban a neve álnévre cserélődik. A művelet nem vonható vissza.',
				sharedTitle: {
					one: '{count} megosztott linkje a címzetteknél is eltűnik.',
					other: '{count} megosztott linkje a címzetteknél is eltűnik.'
				},
				sharedText: 'Ha a címzetteknek szükségük van rájuk, előbb mentsék el maguknak.',
				action: 'Felhasználó törlése'
			}
		},
		toast: {
			created: '{username} létrehozva',
			updated: '{username} módosítva',
			passwordReset: '{username} jelszava visszaállítva',
			twoFactorReset: '{username} kétlépcsős azonosítása visszaállítva',
			unlocked: '{username} zárolása feloldva',
			sessionsRevoked: {
				zero: 'Nem volt lezárható munkamenet',
				one: '{count} munkamenet lezárva',
				other: '{count} munkamenet lezárva'
			},
			deactivated: '{username} inaktiválva',
			activated: '{username} aktiválva',
			deleted: '{username} törölve'
		},
		empty: {
			title: 'Nincs a szűrőknek megfelelő felhasználó.',
			text: 'Módosítsd vagy töröld a szűrőket.'
		}
	},
	pager: {
		label: 'Lapozás',
		previous: 'Előző oldal',
		next: 'Következő oldal',
		page: '{page}. oldal',
		range: '{from}–{to} / {total}'
	},
	userDialog: {
		createTitle: 'Felhasználó felvétele',
		createText: 'Az új felhasználó az első belépéskor saját jelszót állít be.',
		editTitle: '{username} szerkesztése',
		editText: 'A szerepkör és az aktív állapot módosítása a biztonsági naplóba kerül.',
		fields: {
			username: 'Felhasználónév',
			displayName: 'Megjelenített név',
			role: 'Szerepkör',
			locale: 'Nyelv',
			password: 'Jelszó',
			active: 'Aktív'
		},
		usernameHelp: '3–64 karakter: betű, szám, pont, kötőjel, aláhúzás.',
		displayNameHelp: 'Ezt látják a többiek a megosztásoknál. Legfeljebb 100 karakter.',
		localeHelp: 'A felület nyelve az első belépéskor. Később ő maga módosíthatja.',
		passwordMode: {
			label: 'Jelszó megadása',
			generated: 'Generált jelszó',
			manual: 'Megadom'
		},
		generatedHelp: 'A szerver 16 karakteres jelszót készít, és csak egyszer mutatja meg.',
		passwordHelp:
			'Legalább {min} karakter, legalább {distinct} különböző karakter, a felhasználónév nélkül.',
		activeHelp: 'Inaktív fiókkal nem lehet bejelentkezni.',
		selfHint: 'A saját fiókodat a Beállítások oldalon kezeled.',
		create: 'Felhasználó létrehozása',
		save: 'Változások mentése'
	},
	tempPassword: {
		title: 'Ideiglenes jelszó',
		text: '{username} ideiglenes jelszava:',
		copy: 'Jelszó másolása',
		copied: 'Jelszó másolva',
		warningTitle: 'A jelszó csak most látható',
		warningText:
			'Add át biztonságos csatornán. Első bejelentkezéskor a felhasználónak le kell cserélnie.',
		done: 'Kész'
	},
	settings: {
		pageTitle: 'Rendszerbeállítások',
		subtitle: 'Minden felhasználóra érvényes beállítások.',
		cards: {
			archive: {
				title: 'Archiválás',
				text: 'A meg nem nyitott linkek ennyi idő után automatikusan archiválódnak, ha a felhasználó nem állított be saját időszakot.'
			},
			sessions: {
				title: 'Munkamenetek',
				text: 'A bejelentkezés tétlenség után vagy a leghosszabb időtartam elteltével lejár.'
			},
			audit: {
				title: 'Audit napló',
				text: 'A biztonsági események ennyi ideig maradnak meg, utána törlődnek.'
			},
			fetch: {
				title: 'Metaadat-letöltés',
				text: 'A szerver tölti le a linkek címét és ikonját.'
			},
			privacy: {
				title: 'Adatkezelési tájékoztató',
				text: 'Ezek az adatok jelennek meg a tájékoztatóban. Lényeges változásnál adj ki új verziót.'
			}
		},
		fields: {
			defaultArchiveDays: 'Archiválási időszak',
			sessionIdleMinutes: 'Tétlenségi idő',
			sessionAbsoluteHours: 'Leghosszabb munkamenet',
			auditRetentionDays: 'Megőrzési idő',
			allowPrivateFetch: 'Belső hálózati címek engedélyezése',
			privacyControllerName: 'Adatkezelő neve',
			privacyControllerAddress: 'Adatkezelő címe',
			privacyControllerEmail: 'Adatkezelő e-mail-címe',
			privacyDpoContact: 'Adatvédelmi kapcsolattartó',
			privacySupervisoryAuthority: 'Illetékes felügyeleti hatóság'
		},
		help: {
			archiveDays: '{min}–{max} nap. 0: soha. Alapérték: {default}.',
			days: '{min}–{max} nap. Alapérték: {default}.',
			minutes: '{min}–{max} perc. Alapérték: {default}.',
			hours: '{min}–{max} óra. Alapérték: {default}.',
			text: 'Legfeljebb {max} karakter.',
			email: 'Üresen hagyható. Legfeljebb {max} karakter.',
			supervisoryAuthority:
				'Legfeljebb {max} karakter. Például: Nemzeti Adatvédelmi és Információszabadság Hatóság, naih.hu',
			allowPrivateFetch:
				'Intranetes linkekhez. Bekapcsolva a szerver a helyi hálózat címeit is letöltheti; a felhős metaadat-címek továbbra is tiltottak.'
		},
		units: {
			days: 'nap',
			minutes: 'perc',
			hours: 'óra'
		},
		runArchive: 'Archiválás futtatása most',
		archived: {
			zero: 'Nem volt archiválandó link',
			one: '{count} link archiválva',
			other: '{count} link archiválva'
		},
		locked: {
			title: 'Környezeti változó rögzíti',
			privateFetch: 'Az APP_ALLOW_PRIVATE_FETCH értéke dönt.',
			generic: 'A szerver konfigurációja dönt.'
		},
		notice: {
			version: 'Jelenlegi verzió',
			updatedAt: 'Kiadva',
			bump: 'Új verzió kiadása',
			bumpTitle: 'Kiadod a tájékoztató új verzióját?',
			bumpText:
				'Minden felhasználónak újra el kell fogadnia a következő bejelentkezéskor vagy kéréskor.',
			bumped: 'A tájékoztató {version}. verziója kiadva'
		},
		unsaved: 'Mentetlen módosítások',
		noChanges: 'Nincs mentetlen módosítás',
		save: 'Beállítások mentése',
		discard: 'Elvetés',
		saved: 'Beállítások mentve',
		leave: {
			title: 'Elveted a módosításokat?',
			text: 'A nem mentett beállítások elvesznek.',
			action: 'Módosítások elvetése'
		}
	}
};
