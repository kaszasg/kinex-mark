// Settings > Security and Settings > Privacy (plan 8.7.6, owner F5).
export default {
	password: {
		title: 'Jelszó',
		text: 'Jelszócsere után a többi eszközön kijelentkezel.'
	},
	twoFactor: {
		title: 'Kétlépcsős azonosítás',
		text: 'Belépéskor a jelszó mellett a hitelesítő alkalmazás hatjegyű kódja is kell.',
		remaining: 'Fel nem használt helyreállító kód',
		lowCodes: {
			title: 'Kevés helyreállító kódod maradt',
			text: 'Generálj újakat, mielőtt elfogynak.'
		},
		enable: 'Bekapcsolás',
		regenerate: 'Új helyreállító kódok',
		disable: 'Kikapcsolás',
		enabled: 'Kétlépcsős azonosítás bekapcsolva',
		disabled: 'Kétlépcsős azonosítás kikapcsolva',
		regenerated: 'Új helyreállító kódok létrehozva'
	},
	enroll: {
		step: '{current} / {total}',
		stepLabel: 'Lépés: {current} / {total}',
		password: {
			title: 'Jelszó megerősítése',
			text: 'A bekapcsoláshoz add meg a jelenlegi jelszavadat.',
			label: 'Jelenlegi jelszó',
			submit: 'Jelszó megerősítése'
		},
		scan: {
			title: 'Olvasd be a QR-kódot',
			text: 'Nyisd meg a hitelesítő alkalmazásodat, és olvasd be a kódot. Utána írd be az alkalmazásban megjelenő kódot.',
			qr: 'QR-kód a hitelesítő alkalmazáshoz',
			manual: 'Kézi beíráshoz:',
			copySecret: 'Titkos kulcs másolása',
			secretCopied: 'Titkos kulcs másolva',
			expires: 'A beállítás érvényes eddig: {time}',
			code: 'Hitelesítő kód',
			codeHelp: 'Az alkalmazásban látható hatjegyű kód.',
			submit: 'Kód ellenőrzése'
		},
		codes: {
			title: 'Mentsd el a helyreállító kódokat'
		}
	},
	recovery: {
		text: 'Ha nem éred el a hitelesítő alkalmazást, ezekkel léphetsz be. Mindegyik kód egyszer használható.',
		listLabel: 'Helyreállító kódok',
		printHeading: 'Kinex Mark helyreállító kódok: {username}',
		copy: 'Kódok másolása',
		copied: 'Kódok másolva',
		download: 'Letöltés .txt-ként',
		print: 'Nyomtatás',
		saved: 'Elmentettem a kódokat, és tudom, hogy csak most láthatók.',
		file: {
			title: 'Kinex Mark helyreállító kódok',
			intro: 'Fiók: {username}. Mindegyik kód egyszer használható a hitelesítő kód helyett.',
			generatedAt: 'Létrehozva: {time}',
			warning:
				'Tárold biztonságos helyen. Aki a jelszavadat és egy kódot ismer, beléphet a fiókodba.'
		}
	},
	confirm: {
		password: 'Jelenlegi jelszó',
		disable: {
			title: 'Kikapcsolod a kétlépcsős azonosítást?',
			text: 'Belépéskor ezután csak a jelszó kell. A helyreállító kódok törlődnek, és a többi eszközön kijelentkezel.',
			code: 'Hitelesítő vagy helyreállító kód',
			codeHelp: 'Az alkalmazás hatjegyű kódja vagy egy fel nem használt helyreállító kód.',
			submit: 'Kétlépcsős azonosítás kikapcsolása'
		},
		regenerate: {
			title: 'Új helyreállító kódokat hozol létre?',
			text: 'A régi kódok érvényüket vesztik. Az újak csak egyszer láthatók.',
			code: 'Hitelesítő kód',
			codeHelp: 'Az alkalmazásban látható hatjegyű kód.',
			submit: 'Új kódok létrehozása'
		}
	},
	sessions: {
		title: 'Munkamenetek',
		text: 'Az eszközök, amelyeken be vagy jelentkezve.',
		listLabel: 'Aktív munkamenetek',
		current: 'Ez az eszköz',
		unknownDevice: 'Ismeretlen eszköz',
		ip: 'IP-cím',
		created: 'Létrehozva',
		lastSeen: 'Utolsó aktivitás',
		expires: 'Lejár',
		revoke: 'Kijelentkeztetés: {device}',
		revoked: 'Munkamenet lezárva: {device}',
		revokeOthers: 'Kijelentkezés a többi eszközön',
		revokedOthers: {
			zero: 'Nem volt másik munkamenet',
			one: '{count} munkamenet lezárva',
			other: '{count} munkamenet lezárva'
		},
		revokeCurrent: {
			title: 'Kijelentkezel ezen az eszközön?',
			text: 'A munkamenet lezárul, és újra be kell jelentkezned.',
			submit: 'Kijelentkezés ezen az eszközön'
		},
		empty: {
			title: 'Nincs aktív munkamenet',
			text: 'Frissítsd a listát, vagy jelentkezz be újra.'
		}
	},
	privacy: {
		notice: {
			title: 'Adatkezelési tájékoztató',
			text: 'A tájékoztató elfogadásával azt nyugtázod, hogy megismerted.',
			current: 'Aktuális verzió',
			updatedAt: 'Frissítve',
			version: '{version}. verzió',
			history: 'Elfogadások',
			acceptedAt: 'Elfogadva: {time}',
			currentBadge: 'Aktuális',
			none: {
				title: 'Még nincs elfogadott tájékoztató',
				text: 'Nyisd meg a tájékoztatót, és fogadd el.'
			},
			open: 'Tájékoztató megnyitása',
			summary: 'A tájékoztató szövege'
		},
		export: {
			title: 'Adataid letöltése',
			text: 'A JSON-fájl a fiókodhoz tartozó minden adatot tartalmaz:',
			items: {
				account: 'fiókadatok',
				preferences: 'beállítások',
				sessions: 'munkamenetek',
				tags: 'címkék',
				links: 'saját linkek a megosztásaikkal',
				shared: 'a veled megosztott linkek személyes állapota',
				audit: 'a fiókodhoz tartozó biztonsági napló bejegyzései, szereplőként és célként',
				lock: 'a bejelentkezési zárolás állapota'
			},
			excluded: 'A jelszó, a titkos kulcsok és a helyreállító kódok nem kerülnek bele.',
			submit: 'Adatok letöltése (JSON)',
			done: 'Adatok letöltve'
		}
	},
	deleteAccount: {
		title: 'Fiók törlése',
		text: 'A fiókod és minden hozzá tartozó adat végleg törlődik. Ha meg akarod őrizni a linkjeidet, előbb töltsd le az adataidat.',
		open: 'Fiók törlése',
		dialogTitle: 'Törlöd a fiókodat?',
		dialogText:
			'Minden linked, címkéd, megosztásod és beállításod végleg törlődik. A művelet nem vonható vissza.',
		shared: {
			title: {
				one: '{count} megosztott linked a címzetteknél is eltűnik.',
				other: '{count} megosztott linked a címzetteknél is eltűnik.'
			},
			text: 'Ha meg akarják tartani, kérd meg őket, hogy vegyék fel a saját gyűjteményükbe.'
		},
		password: 'Jelszó',
		code: 'Hitelesítő vagy helyreállító kód',
		confirm: 'Írd be a felhasználóneved a megerősítéshez: {username}',
		submit: 'Fiók végleges törlése'
	}
};
