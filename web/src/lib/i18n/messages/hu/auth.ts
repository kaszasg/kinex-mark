export default {
	login: {
		title: 'Bejelentkezés',
		subtitle: 'Kinex Mark, a Kinexium linkgyűjteménye.',
		username: 'Felhasználónév',
		password: 'Jelszó',
		submit: 'Bejelentkezés',
		noAccount: 'Fiókot a rendszergazda hoz létre.',
		version: 'Verzió {version}',
		footer: 'Nyelv, téma és tájékoztató'
	},
	totp: {
		title: 'Kétlépcsős azonosítás',
		text: 'Írd be a hitelesítő alkalmazásod 6 jegyű kódját.',
		code: 'Hitelesítő kód',
		submit: 'Kód ellenőrzése',
		useRecovery: 'Helyreállító kódot használok',
		useCode: 'Hitelesítő kódot használok',
		back: 'Vissza a bejelentkezéshez'
	},
	recovery: {
		text: 'Írd be az egyik elmentett helyreállító kódodat.',
		code: 'Helyreállító kód',
		placeholder: 'xxxxx-xxxxx',
		help: 'A kötőjel és a szóköz elhagyható.',
		used: {
			one: 'Bejelentkeztél egy helyreállító kóddal. Még {count} kódod maradt.',
			other: 'Bejelentkeztél egy helyreállító kóddal. Még {count} kódod maradt.'
		},
		low: 'Generálj új kódokat a Beállítások › Biztonság fülön.'
	},
	logout: {
		title: 'Kijelentkeztél.',
		text: 'A folytatáshoz jelentkezz be újra.'
	},
	expired: {
		title: 'A munkamenet lejárt.',
		text: 'Jelentkezz be újra.'
	},
	accountDeleted: {
		title: 'A fiókod törlődött.',
		text: 'A linkjeid, címkéid, megosztásaid és beállításaid végleg törlődtek.'
	},
	passwordChange: {
		pageTitle: 'Jelszócsere',
		title: 'Állíts be új jelszót',
		text: 'A fiókod ideiglenes jelszóval készült, vagy a rendszergazda visszaállította. A folytatáshoz adj meg egy újat.',
		current: 'Jelenlegi jelszó',
		new: 'Új jelszó',
		confirm: 'Új jelszó még egyszer',
		mismatch: 'A két jelszó nem egyezik.',
		submit: 'Jelszó módosítása',
		saved: 'Jelszó módosítva',
		savedOtherSessions: 'Jelszó módosítva. A többi eszközön kijelentkeztél.'
	},
	passwordPolicy:
		'Legalább {min} karakter, legalább {distinct} különböző karakter, a felhasználóneved nélkül.'
};
