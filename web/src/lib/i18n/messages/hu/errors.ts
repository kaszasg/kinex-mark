export default {
	endpoint_not_found: { title: 'A funkció nem érhető el.', text: 'Frissítsd az oldalt.' },
	invalid_json: {
		title: 'A kérés hibás formátumú.',
		text: 'Frissítsd az oldalt, és próbáld újra.'
	},
	invalid_id: { title: 'Hibás azonosító.', text: 'Nyisd meg újra a listából.' },
	invalid_query: {
		title: 'Hibás szűrőbeállítás.',
		text: 'Töröld a szűrőket, és próbáld újra.'
	},
	validation_failed: { title: 'Néhány mező hibás.', text: 'Javítsd a jelölt mezőket.' },
	payload_too_large: { title: 'A fájl nagyobb 2 MB-nál.', text: 'Válassz kisebb fájlt.' },
	not_found: {
		title: 'Az elem nem található.',
		text: 'Lehet, hogy törölték, vagy már nem látod. Frissítsd a listát.'
	},
	unauthenticated: { title: 'A munkamenet lejárt.', text: 'Jelentkezz be újra.' },
	forbidden: { title: 'Ehhez nincs jogosultságod.', text: 'Fordulj a rendszergazdához.' },
	csrf_rejected: {
		title: 'A biztonsági ellenőrzés elutasította a kérést.',
		text: 'Frissítsd az oldalt, és próbáld újra.'
	},
	host_mismatch: {
		title: 'A kérés nem a Kinex Mark beállított címére érkezett.',
		text: 'Nyisd meg az alkalmazást a megszokott címén, vagy szólj a rendszergazdának.'
	},
	password_change_required: {
		title: 'Jelszócsere szükséges.',
		text: 'A folytatáshoz állíts be új jelszót.'
	},
	consent_required: {
		title: 'Az adatkezelési tájékoztató elfogadása szükséges.',
		text: 'A folytatáshoz fogadd el.'
	},
	invalid_credentials: {
		title: 'Hibás felhasználónév vagy jelszó.',
		text: 'Ellenőrizd, és próbáld újra.'
	},
	account_disabled: { title: 'A fiók inaktív.', text: 'Fordulj a rendszergazdához.' },
	too_many_attempts: {
		title: 'Túl sok sikertelen próbálkozás.',
		text: 'Próbáld újra {minutes} perc múlva.'
	},
	challenge_invalid: { title: 'A belépési kérés lejárt.', text: 'Jelentkezz be újra.' },
	invalid_totp: {
		title: 'Hibás vagy már felhasznált kód.',
		text: 'Írd be az alkalmazásban látható új kódot.'
	},
	current_password_invalid: { title: 'A jelenlegi jelszó hibás.', text: 'Írd be újra.' },
	totp_already_enabled: {
		title: 'A kétlépcsős azonosítás már be van kapcsolva.',
		text: 'Frissítsd az oldalt.'
	},
	totp_not_enabled: {
		title: 'A kétlépcsős azonosítás nincs bekapcsolva.',
		text: 'Frissítsd az oldalt.'
	},
	totp_setup_required: { title: 'A beállítás lejárt.', text: 'Kezdd újra a bekapcsolást.' },
	link_duplicate: {
		title: 'Ez a link már megvan a gyűjteményedben.',
		text: 'Nyisd meg a meglévőt.'
	},
	link_limit_reached: {
		title: 'Elérted a 10 000 linkes korlátot.',
		text: 'Törölj régi linkeket, mielőtt újat veszel fel.'
	},
	not_owner: {
		title: 'Ezt csak a link tulajdonosa módosíthatja.',
		text: 'A saját címkéidet és kitűzésedet szabadon állíthatod.'
	},
	share_not_allowed: {
		title: 'A szerepköröd nem oszthat meg linket.',
		text: 'Fordulj egy fejlesztőhöz vagy a rendszergazdához.'
	},
	share_duplicate: {
		title: 'Ezzel a címzettel már meg van osztva.',
		text: 'Válassz másik címzettet.'
	},
	no_url_found: {
		title: 'Nem található URL.',
		text: 'Húzz vagy illessz be egy http vagy https kezdetű linket.'
	},
	import_invalid: {
		title: 'A fájl nem értelmezhető.',
		text: 'Kinex Mark JSON-exportot vagy böngészős HTML-könyvjelzőfájlt válassz.'
	},
	import_too_many_entries: {
		title: 'A fájlban 5000-nél több bejegyzés van.',
		text: 'Bontsd több fájlra, és importáld őket egymás után.'
	},
	tag_duplicate: { title: 'Ilyen nevű címke már van.', text: 'Válassz másik nevet.' },
	tag_limit_reached: {
		title: 'Elérted az 500 címkés korlátot.',
		text: 'Törölj nem használt címkéket.'
	},
	username_taken: { title: 'A felhasználónév foglalt.', text: 'Válassz másikat.' },
	last_admin: {
		title: 'Ez az utolsó aktív adminisztrátor.',
		text: 'Előbb nevezz ki egy másik adminisztrátort.'
	},
	self_modification: {
		title: 'A saját fiókodat itt nem módosíthatod.',
		text: 'A szerepkörödet egy másik adminisztrátor állíthatja, a fiókod törlése a Beállítások oldalon érhető el.'
	},
	setting_locked: {
		title: 'A beállítást környezeti változó rögzíti.',
		text: 'A módosításhoz a szerver konfigurációját kell változtatni: {keys}.'
	},
	consent_version_mismatch: {
		title: 'Közben új tájékoztató jelent meg.',
		text: 'Olvasd el, és fogadd el újra.'
	},
	confirmation_mismatch: {
		title: 'A megerősítés nem egyezik a felhasználóneveddel.',
		text: 'Írd be pontosan.'
	},
	not_implemented: {
		title: 'A funkció még nem érhető el.',
		text: 'Próbáld újra egy későbbi verzióban.'
	},
	internal_error: { title: 'Váratlan szerverhiba történt.', text: 'Próbáld újra később.' },
	network_error: {
		title: 'A szerver nem érhető el.',
		text: 'Ellenőrizd a hálózati kapcsolatot.'
	}
};
