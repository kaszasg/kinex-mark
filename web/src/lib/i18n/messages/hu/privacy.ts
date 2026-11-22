export default {
	title: 'Adatkezelési tájékoztató',
	version: '{version}. verzió',
	updatedAt: 'Frissítve: {date}',
	backToApp: 'Vissza az alkalmazásba',
	controller: {
		name: 'Név',
		address: 'Cím',
		email: 'E-mail',
		dpo: 'Adatvédelmi kapcsolattartó'
	},
	sections: {
		controller: {
			title: 'Adatkezelő',
			body: 'Az adatkezelő a Kinex Markot üzemeltető szervezet. Adatvédelmi kérdéseiddel az alábbi elérhetőségeken fordulhatsz hozzá.'
		},
		data: {
			title: 'Kezelt adatok',
			body: 'Felhasználónév, megjelenített név, jelszó-hash, titkosított kétlépcsős azonosítási titok, munkamenetek IP-címmel és böngészőazonosítóval, linkek, címkék, megosztások, használati statisztika, beállítások, valamint biztonsági napló IP-címmel. E-mail-címet, telefonszámot és követési adatot a rendszer nem kezel.'
		},
		purposes: {
			title: 'Célok',
			body: 'A linkgyűjtemény szolgáltatás nyújtása, valamint a fiók és a rendszer biztonsága (bejelentkezés-korlátozás, biztonsági napló).'
		},
		legalBasis: {
			title: 'Jogalapok',
			body: 'A fiók, a linkek, a címkék, a megosztások és a beállítások kezelése a szolgáltatás nyújtásához szükséges (GDPR 6. cikk (1) bekezdés b) pont). A biztonsági napló, a bejelentkezés-korlátozás és a munkamenetek IP-címe az üzemeltetőnek a rendszer biztonságához fűződő jogos érdekén alapul (6. cikk (1) bekezdés f) pont). A tájékoztató elfogadása a megismerés nyugtázása, nem hozzájárulás.'
		},
		recipients: {
			title: 'Címzettek',
			body: 'Azok a felhasználók, akikkel linket megosztasz, látják a linket, a felhasználónevedet és a megjelenített nevedet. A fejlesztők és a rendszergazdák a megosztás címzettjének kiválasztásakor minden aktív felhasználót megtalálnak felhasználónév, megjelenített név és szerepkör szerint. Az üzemeltető rendszergazdái a felhasználókezeléshez szükséges adatokat és a biztonsági naplót látják. Az üzemeltető tárhelyszolgáltatója adatfeldolgozóként fér hozzá az adatokhoz.'
		},
		transfers: {
			title: 'Továbbítás harmadik országba',
			body: 'Az adatok nem kerülnek harmadik országba vagy nemzetközi szervezethez.'
		},
		automatedDecisions: {
			title: 'Automatizált döntéshozatal',
			body: 'Automatizált döntéshozatal és profilalkotás nincs. Az automatikus archiválás csak a linklista rendezését érinti.'
		},
		retention: {
			title: 'Megőrzési idő',
			body: 'A munkamenetek legfeljebb {hours} után lejárnak, tétlenség esetén már {minutes} után. A biztonsági napló bejegyzései {days} után törlődnek. Minden más adat a fiók törléséig megmarad.'
		},
		storage: {
			title: 'Sütik és helyi tárolás',
			body: 'A rendszer egyetlen, a bejelentkezéshez szükséges sütit használ (kxm_session vagy __Host-kxm_session). A böngésző helyi tárolója a választott témát (mode-watcher-mode, mode-watcher-theme) és nyelvet (kxm-locale) őrzi. Követés, analitika és harmadik féltől betöltött tartalom nincs.'
		},
		fetching: {
			title: 'Metaadat-letöltés',
			body: 'A linkek címét és ikonját a szerver tölti le. A böngésződ ezért nem kapcsolódik harmadik félhez, amíg meg nem nyitsz egy linket.'
		},
		adminAccess: {
			title: 'Rendszergazdai hozzáférés',
			body: 'A rendszergazda visszaállíthatja a jelszavadat, és ezzel beléphet a fiókodba. Ez a biztonsági naplóba kerül, a saját adatexportodban is látod, és a következő belépéskor új jelszót kell megadnod.'
		},
		rights: {
			title: 'Jogaid',
			body: 'A GDPR 15–21. cikke szerint a következő jogok illetnek meg:'
		}
	},
	rights: {
		access:
			'Hozzáférés: a Beállítások › Adatvédelem fülön az Adatok letöltése gombbal minden adatodat megkapod.',
		portability:
			'Adathordozhatóság: ugyanez az export géppel olvasható JSON-fájl, a linkjeidet a Linkek oldalon böngészős könyvjelzőfájlként is exportálhatod.',
		rectification:
			'Helyesbítés: a megjelenített nevedet a Beállítások › Profil fülön, a linkjeidet és a címkéidet közvetlenül szerkesztheted.',
		erasure:
			'Törlés: a fiókodat a Beállítások › Adatvédelem fülön törölheted, minden linked, címkéd és megosztásod vele együtt törlődik.',
		restriction:
			'Az adatkezelés korlátozása: kérd az adatkezelőtől a fenti elérhetőségeken. A rendszergazda inaktiválhatja a fiókodat, ilyenkor az adataid megmaradnak, de a kezelésük szünetel.',
		objection:
			'Tiltakozás: a jogos érdeken alapuló adatkezelés ellen a fenti elérhetőségeken tiltakozhatsz.',
		complaint: 'Panasz: panaszt tehetsz a felügyeleti hatóságnál: {authority}.',
		defaultAuthority: 'a lakóhelyed szerinti adatvédelmi hatóság'
	},
	consent: {
		title: 'Adatkezelési tájékoztató',
		text: 'Olvasd el, hogyan kezeli a Kinex Mark az adataidat. A folytatáshoz erősítsd meg, hogy megismerted.',
		openFull: 'Megnyitás teljes oldalon',
		newTab: 'új lapon nyílik',
		checkbox: 'Elolvastam az adatkezelési tájékoztatót, és tudomásul veszem.',
		submit: 'Tudomásul vettem, tovább'
	}
};
