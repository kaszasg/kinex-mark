# KINEXIUM grafikai követelmények

Ez a dokumentum a KINEXIUM weboldalának és webalkalmazásának teljes arculati előírása: hangnem, színek, betűk, tér, forma, mozgás, logó, ikonok, képi világ, komponensek, diagramok, oldalszerkezetek és a Tailwind-megfeleltetés. Minden érték, osztály és kódrészlet itt szerepel, amire a felület elkészítéséhez szükség van.

A csomagok telepítését, a Vite- és SvelteKit-konfigurációt és a build lépéseit a TECHNOLOGY.md írja le.

Írásmód a dokumentumban: folyószövegben tizedesvessző és szóközzel írt mértékegység („1,4 px”, „2,0–4,4”); CSS-értékben, SVG-koordinátában és -attribútumban, valamint a backtickes és táblázati értékekben pont és szóköz nélküli egység (`1.414px`, `76.8`, `72px`).

Célkörnyezet: SvelteKit + Svelte 5 + TypeScript, Tailwind CSS v4, shadcn-svelte komponensek, Lucide ikonok (`@lucide/svelte`). A stíluslap helye `src/app.css` (13. fejezet), a márkakomponensek helye `src/lib/components/brand/` (10. fejezet).

---

## 1. Márka és hangnem

### 1.1 A cég és a rendszer karaktere

- A Kinexium szoftverfejlesztéssel, konzultációval és rendszertervezéssel foglalkozik, IoT és AI fókusszal.
- A vizuális rendszer high-tech, átlátható és minimalista: sötét petróleum alap, egyetlen izzó ciánkék jel, vékony betűk, hajszálvonalak.
- Ugyanez a rendszer érvényes a weboldalra és a webalkalmazásra.

### 1.2 Hang

- Mérnök beszél mérnökkel és döntéshozóval: közvetlen, tényszerű, rövid.
- Előbb a kimenet, utána a módszer.
- A cég egy ember, ezért egyes szám első személyben írj: „dolgozom”, „vállalom”. A céget jelentő „mi” hamis lenne. Az olvasót is magába foglaló, közös cselekvésre hívó alak („Beszéljünk a projektről”, 12.1) nem a cégről szól, ezért megengedett.
- Emoji nincs. Felkiáltójel nincs.

### 1.3 Megszólítás

A kettősség szándékos.

| Helyzet | Forma | Példa |
| --- | --- | --- |
| Ügyfélkommunikáció | Magázás | – |
| Weboldal | Közvetlen megszólítás kerülése, vagy magázás | „Van egy rendszerötlete?” |
| Alkalmazás felületi szövege (gomb, hibaüzenet, dialógus) | Tegező felszólító mód | „Ellenőrizd a kapcsolatot”, „Törlöd a GW-04 eszközt?” |

### 1.4 Írásmód és nagybetű

- Címben, gombon, menüben mondatkezdő nagybetű.
- Csupa nagybetű csak a logóban, a `kx-eyebrow` osztályon át és a táblázatfejlécben (`th`, 10.7) jelenik meg. Kézzel nagybetűs szöveget ne gépelj.
- Szövegben: Kinexium. Logóban: KINEXIUM.
- A ™ jel az oldal láblécébe és a szóvédjegybe kerül, folyószövegbe nem. A lábléc védjegysora („KINEXIUM™ · kinexium.net”, 12.1) a szóvédjegy szöveges alakja, ezért nagybetűs: ez az egyik kivétel a kézzel gépelt nagybetű tilalma alól. A másik a borító neve (10.6), amely a márkanév rögzített, logószerű alakja. Rajtuk kívül kézzel gépelt nagybetű nincs.

### 1.5 Tiltott szavak

Ne használd: forradalmi, innovatív, kulcsrakész, szinergia, „a jövő technológiája”. Helyettük írd le, mit csinál a rendszer.

### 1.6 Gomb- és hibaüzenet-minták

- Gomb: ige és tárgy. „Eszköz felvétele”, nem „OK”.
- Törlésnél nevezd meg, mi törlődik: „Eszköz törlése”.
- Hibaüzenet: mi történt, és mit tegyen a felhasználó. Példa: „A GW-04 gateway nem elérhető. Ellenőrizd a hálózati kapcsolatot.”
- Visszafordíthatatlan műveletnél a dialógus címe kérdés, a leírás kimondja a következményt, a gomb megnevezi a tárgyat (10.8).
- Riasztásnál a cím mondja meg, mi történt, a szöveg, hogy mi a teendő (10.1).

### 1.7 Példatáblázat

| Helyzet | Így | Ne így |
| --- | --- | --- |
| Hero | Az érzékelőtől a döntésig. | Forradalmasítjuk az ipari IoT-t! |
| Bemutatkozás | Egy mérnök, közvetlen kapcsolat, felelősség a teljes rendszerért. | Tapasztalt csapatunk komplex megoldásokat kínál. |
| Üres állapot | Még nincs eszköz. Vedd fel az elsőt. | Hoppá, itt nincs semmi! |
| EN hero | From sensor to decision. | Empowering the future of IoT. |

### 1.8 Számformátum

| Szabály | Példa |
| --- | --- |
| Magyar felületen tizedesvessző | `24,8 °C` |
| Ezres tagolás nem törő szóközzel (U+00A0), ötjegytől | `1284`, `12 480` |
| Érték és mértékegység között nem törő szóköz (U+00A0) | `42 ms` |
| Idő 24 órás | `14:32:08` |
| Dátum | `2026. 11. 05.` |
| Angol felületen pont a tizedesjel | `24.8 °C` |
| Hiányzó érték `–`, soha nem nulla | `–` |

Oszlopba rendezett szám mindig `data` stílusú, mono betűvel (3.6).

A fenti példákban az ezres csoportok, illetve a szám és a mértékegység között U+00A0 karakter áll. A dokumentum többi példájában (mintaszövegek, mintaadatok, táblázatok, kódrészletek) a sima szóközzel írt ilyen elválasztás is U+00A0-ként értendő. Kódban `\u00a0`, HTML-ben `&nbsp;` írja.

### 1.9 Nyelv

- A magyar az alap, az angol egyenrangú.
- Felületi szövegben maradjon meg a szakmai angol szó, ha nincs bevett magyar: gateway, firmware, API-kulcs.
- A HTML `lang` attribútuma magyar felületen `hu`.

---

## 2. Színrendszer

### 2.1 Alapelvek

- A felület 90%-a semleges: `background`, `card`, `foreground`.
- A `primary` az egyetlen kiemelőszín, és jelentése van: fő művelet, aktív elem, adatjel, link.
- Képernyőnként egy `primary` gomb. Egy kivétel: a weboldal `signal-700` kapcsolatblokkjának `primary` gombja, mert ugyanazt a műveletet (kapcsolatfelvétel) ismétli meg, mint a hero fő gombja. Felirata ott az e-mail-cím (12.1).
- A sötét téma az elsődleges. A világos téma teljes értékű párja, sűrű irodai munkához.
- Skálák: `ink-*` az alapok, `mist-*` a szövegek, `signal-*` a kiemelés.
- Komponensben szemantikus tokent használj, ne skálatokent (`card`, nem `ink-850`). Skálatoken csak márkafelületen szerepel: borító, logó-előnézet, a weboldal `signal-700` kapcsolatblokkja.
- Színátmenet nincs.
- Nyers hex értéket ne írj a kódba. Minden szín tokenből jön.

### 2.2 Skálák

A skálák a két témában azonosak, a világos téma nem írja felül őket.

#### ink (alapok)

| Token | Hex | Használat |
| --- | --- | --- |
| `ink-950` | `#040a0f` | A legmélyebb tónus. Oldalsáv alapja, kitöltött gomb felirata sötét témában. |
| `ink-900` | `#071016` | Sötét téma oldalalapja (`background`). |
| `ink-850` | `#0b171f` | Sötét téma kártyafelülete (`card`), világos téma szövegszíne. |
| `ink-800` | `#10212a` | Sötét téma lebegő és másodlagos felülete (`popover`, `muted`). |
| `ink-700` | `#192d37` | Sötét téma hover-felülete (`accent`). |
| `ink-600` | `#253d46` | Sötét téma hajszálvonala (`border`). Csak díszítő vonalhoz, 3:1 kontraszt alatt van. |
| `ink-500` | `#345058` | Másodlagos vonal sötét alapon, ha a `border` túl halk, például nagy ábrán. |
| `ink-400` | `#486469` | Világos téma halk szövege (`muted-foreground`). |
| `ink-300` | `#5f7b7d` | Sötét téma űrlapkerete (`input`). |

#### mist (szövegek)

| Token | Hex | Használat |
| --- | --- | --- |
| `mist-400` | `#799290` | Letiltott ikon sötét alapon. |
| `mist-300` | `#94a9a5` | Sötét téma halk szövege (`muted-foreground`). |
| `mist-200` | `#b3c1bb` | Másodlagos szöveg sötét alapon, ha a `muted-foreground` túl halk. |
| `mist-100` | `#d4dcd5` | Sötét téma szövegszíne (`foreground`). |
| `mist-50` | `#f2f5f3` | Világos téma oldalalapja (`background`). |

#### signal (kiemelés)

| Token | Hex | Használat |
| --- | --- | --- |
| `signal-300` | `#77dbed` | A márka fő színe: sötét téma `primary` és `ring`. |
| `signal-400` | `#52aec8` | Grafikai kiemelés, a borító lépcsővonala. Szövegre világos alapon nem elég. |
| `signal-500` | `#36879f` | Grafikai kiemelés világos alapon, minta és ikon. |
| `signal-600` | `#16637e` | Világos téma `primary` és `ring`, link világos alapon. |
| `signal-700` | `#0c4962` | Mély kiemelés: hero blokk, borító, világos téma lenyomott gombja. |

### 2.3 Szemantikus tokenek

Zárójelben a skálatoken, amelyre a szemantikus token mutat.

| Token | Sötét | Világos | Használat |
| --- | --- | --- | --- |
| `background` | `#071016` (ink-900) | `#f2f5f3` (mist-50) | Oldalalap. `foreground` és `muted-foreground` szöveget hordoz. |
| `foreground` | `#d4dcd5` (mist-100) | `#0b171f` (ink-850) | Alapszöveg `background`, `card`, `popover`, `muted`, `accent` felületen. |
| `card` | `#0b171f` (ink-850) | `#ffffff` | Kártya, panel, táblázat alapja. Mindig `border` kerettel, árnyék nélkül. |
| `card-foreground` | `#d4dcd5` (mist-100) | `#0b171f` (ink-850) | Szöveg `card` felületen. |
| `popover` | `#10212a` (ink-800) | `#ffffff` | Legördülő, tooltip, dialógus alapja, `shadow-popover` árnyékkal. |
| `popover-foreground` | `#d4dcd5` (mist-100) | `#0b171f` (ink-850) | Szöveg `popover` felületen. |
| `primary` | `#77dbed` (signal-300) | `#16637e` (signal-600) | Az egyetlen kiemelőszín: fő gomb, aktív állapot, link. |
| `primary-foreground` | `#040a0f` (ink-950) | `#ffffff` | Szöveg és ikon `primary` kitöltésen. |
| `secondary` | `#10212a` (ink-800) | `#e4eae7` | Másodlagos gomb kitöltése. |
| `secondary-foreground` | `#d4dcd5` (mist-100) | `#0b171f` (ink-850) | Szöveg `secondary` felületen. |
| `muted` | `#10212a` (ink-800) | `#e4eae7` | Csendes sáv: táblázatfejléc, kódblokk, kikapcsolt kapcsoló alapja. |
| `muted-foreground` | `#94a9a5` (mist-300) | `#486469` (ink-400) | Másodlagos szöveg, felirat `background`, `card`, `popover`, `muted` felületen. |
| `accent` | `#192d37` (ink-700) | `#d9e2df` | Hover és kijelölt sor felülete. Nem márkaszín, a nevét a shadcn-svelte adja. |
| `accent-foreground` | `#d4dcd5` (mist-100) | `#0b171f` (ink-850) | Szöveg `accent` felületen. |
| `destructive` | `#ff8a66` | `#b3401e` | Veszély és hiba: törlés gomb, hibaüzenet, kritikus állapot. |
| `destructive-foreground` | `#040a0f` (ink-950) | `#ffffff` | Szöveg `destructive` kitöltésen. |
| `success` | `#8be8c0` | `#146c4a` | Online, sikeres, rendben. |
| `warning` | `#f4c95d` | `#865600` | Figyelmeztetés, közelgő küszöb. |
| `info` | `#77dbed` (signal-300) | `#16637e` (signal-600) | Semleges tájékoztató állapot. |
| `border` | `#253d46` (ink-600) | `#c9d4d0` | Hajszálvonal: kártya, elválasztó, táblázatsor. Díszítő, jelentést nem hordoz. |
| `input` | `#5f7b7d` (ink-300) | `#6b8482` | Űrlapmezők kerete. Legalább 3:1 `background`, `card`, `popover` felületen. |
| `ring` | `#77dbed` (signal-300) | `#16637e` (signal-600) | Fókuszgyűrű: 2 px tömör, 2 px eltolással. Legalább 3:1 minden felületen. |

#### Oldalsáv tokenek

| Token | Sötét | Világos | Használat |
| --- | --- | --- | --- |
| `sidebar` | `#040a0f` (ink-950) | `#e9eeeb` | Alkalmazás-oldalsáv alapja. |
| `sidebar-foreground` | `#b3c1bb` (mist-200) | `#0b171f` (ink-850) | Szöveg az oldalsávon. |
| `sidebar-primary` | `#77dbed` (signal-300) | `#16637e` (signal-600) | Aktív menüpont jelölője. |
| `sidebar-primary-foreground` | `#040a0f` (ink-950) | `#ffffff` | Szöveg `sidebar-primary` kitöltésen. |
| `sidebar-accent` | `#10212a` (ink-800) | `#d9e2df` | Menüpont hover és aktív felülete. |
| `sidebar-accent-foreground` | `#d4dcd5` (mist-100) | `#0b171f` (ink-850) | Szöveg `sidebar-accent` felületen. |
| `sidebar-border` | `#192d37` (ink-700) | `#c9d4d0` | Oldalsáv elválasztó vonala. |
| `sidebar-ring` | `#77dbed` (signal-300) | `#16637e` (signal-600) | Fókuszgyűrű az oldalsávban. |

#### Áttetsző tokenek és árnyék

| Token | Sötét | Világos | Használat |
| --- | --- | --- | --- |
| `selection` | `rgba(119, 219, 237, 0.26)` | `rgba(22, 99, 126, 0.18)` | Szövegkijelölés háttere. |
| `glow` | `rgba(119, 219, 237, 0.14)` | `rgba(22, 99, 126, 0.08)` | Halvány derengés az aktív csomópont körül. Csak díszítés, szöveg mögé nem. |
| `shadow-popover` | `0 12px 32px rgba(0, 0, 0, 0.56)` | `0 12px 32px rgba(11, 23, 31, 0.14)` | Legördülő, dialógus, tooltip. |

### 2.4 Állapotszínek

- A `success`, `warning`, `destructive`, `info` kizárólag állapotot jelöl, és mindig ikonnal és szóval együtt jelenik meg.
- Szín önmagában nem hordoz információt.
- Az állapotszín az ikonon van, a szöveg `foreground` marad.
- Kitöltött, színes jelvény nincs. A `success`, `warning`, `info` tokennek nincs `-foreground` párja, ezért kitöltésként nem használható.
- Szövegként a `destructive`, `success`, `warning` a `background`, `card` és `popover` felületen olvasható.
- Állapotszín nem kerül diagram-adatsorra, és `chart-*` szín nem jelöl állapotot.

### 2.5 Diagramszínek

| Token | Sötét | Világos | Használat |
| --- | --- | --- | --- |
| `chart-1` | `#2ba0c2` | `#0e7fa3` | 1. adatsor. A sorrend rögzített, állapotot nem jelöl. |
| `chart-2` | `#b08a2c` | `#9a7420` | 2. adatsor. |
| `chart-3` | `#1f9b7c` | `#14876b` | 3. adatsor. |
| `chart-4` | `#7a84e0` | `#5b63c7` | 4. adatsor. |
| `chart-5` | `#cf6a95` | `#c04f7d` | 5. adatsor. Ötnél több sor helyett bontsd kis diagramokra. |

### 2.6 Témaváltás

- Az alap téma a sötét. A sötét változók a `:root`, `.dark` és `[data-theme="dark"]` szelektoron élnek, a `.light` vagy `[data-theme="light"]` kapcsol át világosra.
- A `color-scheme` sötétben `dark`, világosban `light`.
- A témát a `mode-watcher` kezeli, sötét alapértelmezéssel és `light` osztállyal a világos témához.
- A `dark:` Tailwind-variáns definíciója `@custom-variant dark (&:is(.dark *));`. Ez a `.dark` elem leszármazottaira illeszkedik, magára a `.dark` elemre nem.
- A böngésző-felület színe: `theme-color` `#071016` (ink-900).

A témához tartozó jelölés (`src/routes/+layout.svelte` és a teljes `src/app.html`):

```svelte
<script lang="ts">
	import '../app.css';
	import { ModeWatcher } from 'mode-watcher';
	let { children } = $props();
</script>

<ModeWatcher defaultMode="dark" lightClassNames={['light']} />
{@render children()}
```

```html
<!doctype html>
<html lang="hu" class="dark">
	<head>
		<meta charset="utf-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		<script>
			try {
				localStorage.getItem('mode-watcher-mode') || localStorage.setItem('mode-watcher-mode', 'dark');
			} catch {}
		</script>
		<link rel="icon" href="%sveltekit.assets%/favicon.svg" type="image/svg+xml" />
		<link rel="preload" href="%sveltekit.assets%/fonts/HankenGrotesk-Variable.woff2" as="font" type="font/woff2" crossorigin />
		<meta name="theme-color" content="#071016" />
		%sveltekit.head%
	</head>
	<body data-sveltekit-preload-data="hover">
		<div style="display: contents">%sveltekit.body%</div>
	</body>
</html>
```

A `dark` osztály megakadályozza a villanást. A szkript kliensoldali (`ssr = false`) módban is sötétre állítja az első indítást, mert ilyenkor a `defaultMode` magában nem hat. A projektváz `<meta name="text-scale" content="scale" />` sora a célállapotban nem szerepel. A váz `+layout.svelte`-jéből a `favicon` import és a `<link rel="icon">` kikerül, mert a favicont az `app.html` és a `static/favicon.svg` adja; az oldalcímet `<svelte:head><title>…</title></svelte:head>` blokk adja.

---

## 3. Tipográfia

### 3.1 Két család

| Család | Szerep | CSS-verem | Tailwind |
| --- | --- | --- | --- |
| Hanken Grotesk | Minden emberi szöveg | `"Hanken Grotesk", ui-sans-serif, system-ui, sans-serif` | `font-sans` |
| Geist Mono | Minden, ami gépi: mért érték, azonosító, idő, kód, szemöldökfelirat | `"Geist Mono", ui-monospace, "SF Mono", Menlo, monospace` | `font-mono` |

Más család nincs. Csak a `font-sans` és a `font-mono` utility létezik.

### 3.2 Fontfájlok

| Fájl a projektben | URL | Formátum | Súlytartomány | Méret |
| --- | --- | --- | --- | --- |
| `static/fonts/HankenGrotesk-Variable.woff2` | `/fonts/HankenGrotesk-Variable.woff2` | WOFF2, változó | 100–900 | 41 216 bájt (kb. 40 KB) |
| `static/fonts/GeistMono-Variable.woff2` | `/fonts/GeistMono-Variable.woff2` | WOFF2, változó | 100–900 | 38 048 bájt (kb. 37 KB) |

- A két font együtt kb. 80 KB, a binárisba ágyazva sem számít.
- Mindkét fájl tartalmazza az ő és ű betűt.
- A Hanken Grotesk előtöltése az `app.html`-ben (2.6): `<link rel="preload" href="%sveltekit.assets%/fonts/HankenGrotesk-Variable.woff2" as="font" type="font/woff2" crossorigin />`.
- A 250-es súly csak a változó fonttal érhető el, ezért statikus vágatra nem cserélhető.
- Beszerzés: a két font projektje (3.3) adja ki őket. A projektbe a változó WOFF2-változat kerül, 100–900 súlytartománnyal, ő és ű glifával, a fenti fájlnéven. A fájlok projektbe másolását a TECHNOLOGY.md írja le.
- A használt fájlt a bájtmérete és az SHA-256 ellenőrző összege azonosítja; más buildből származó fájl más méretű vagy más glifakészletű lehet, ezért csak egyező összegű fájl kerül a projektbe:

| Fájl | SHA-256 |
| --- | --- |
| `HankenGrotesk-Variable.woff2` | `c38c7a29913400c75678bcb9b6bd016f21cc120f2680db35a37fda6c9f38fa0b` |
| `GeistMono-Variable.woff2` | `4017c5ee04cfc2d6dd089f6faceda312e627514aa083d202742f29b3f9866f29` |

### 3.3 Licenc

Mindkét font SIL Open Font License, Version 1.1 (`SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007`).

| Font | Szerzői jogi közlemény |
| --- | --- |
| Geist Mono | Copyright 2024 The Geist Project Authors (https://github.com/vercel/geist-font.git) |
| Hanken Grotesk | Copyright 2021 The Hanken Grotesk Project Authors (https://github.com/marcologous/hanken-grotesk) |

A licencszövegek fájlnevei: `OFL-Geist-Mono.txt`, `OFL-Hanken-Grotesk.txt`, helyük a fontok mellett, `static/fonts/`. Tartalmuk: első sorban a fenti szerzői jogi közlemény, utána a SIL Open Font License 1.1 teljes szövege. A font továbbadásakor a közlemény és a licenc szövege is a csomagban marad.

### 3.4 @font-face

```css
@font-face { font-family: "Hanken Grotesk"; src: url("/fonts/HankenGrotesk-Variable.woff2") format("woff2"); font-weight: 100 900; font-style: normal; font-display: swap; }
@font-face { font-family: "Geist Mono"; src: url("/fonts/GeistMono-Variable.woff2") format("woff2"); font-weight: 100 900; font-style: normal; font-display: swap; }
```

### 3.5 Típusstílusok

A stílusok osztályként érhetők el: `class="display-xl"`, `class="data-lg"`. Az alapszöveg (`body`) a `<body>` elemen is beállított: 15 px / 24 px, Hanken Grotesk.

| Osztály | Család | Méret | Sorköz | Súly | Betűköz | Rendeltetés |
| --- | --- | --- | --- | --- | --- | --- |
| `display-xl` | sans | 72px | 76px | 200 | -0.02em | Webes hero. Csak 40 px felett, csak `foreground` színnel. |
| `display` | sans | 48px | 54px | 250 | -0.015em | Webes szekciócím, mobil hero. |
| `heading-1` | sans | 32px | 40px | 300 | -0.01em | Oldalcím alkalmazásban. |
| `heading-2` | sans | 24px | 32px | 400 | -0.005em | Szekciócím. |
| `heading-3` | sans | 18px | 26px | 500 | – | Kártyacím, dialógus címe. |
| `body-lg` | sans | 18px | 28px | 400 | – | Webes bevezető. Sorhossz legfeljebb `layout-prose`. |
| `body` | sans | 15px | 24px | 400 | – | Alapszöveg weben és alkalmazásban. |
| `body-sm` | sans | 13px | 20px | 400 | – | Sűrű UI: gomb, táblázat, súgószöveg. |
| `label` | mono | 11px | 16px | 500 | 0.08em | Szemöldökfelirat, oszlopfejléc. Mindig `muted-foreground`. Nem nagybetűsít. |
| `code` | mono | 13px | 20px | 400 | – | Kód, azonosító, API-útvonal. |
| `data` | mono | 14px | 20px | 400 | – | Számok táblázatban, mérési adat. |
| `data-lg` | mono | 32px | 36px | 300 | -0.02em | Metrikakártya fő értéke. |

Márka-szövegosztályok:

| Osztály | Érték | Rendeltetés |
| --- | --- | --- |
| `kx-eyebrow` | mono 11px / 16px, súly 500, betűköz .08em, `text-transform: uppercase`, szín `muted-foreground` | Szemöldökfelirat. Az egyetlen nagybetűsítő osztály; rajta kívül csak a táblázatfejléc (`th`, 10.7) nagybetűs. |
| `kx-num` | `font-family: var(--font-mono)` | Szám mono betűvel, szövegen belül. |

Mintaszövegek a stílusokhoz: `display-xl` „Rendszerek, amelyek működnek”, `display` „IoT és AI, egy kézben”, `heading-1` „Rendszerterv és architektúra”, `heading-2` „Eszközök állapota”, `heading-3` „Hőmérséklet, 24 óra”, `body-lg` „Az érzékelőtől a döntésig tervezek.”, `body` „A szenzorok adatait a rendszer helyben dolgozza fel.”, `body-sm` „Utolsó szinkron: ma 14:32”, `label` „01 Rendszertervezés” (csupa nagybetűs szemöldökfeliratként a `kx-eyebrow` osztály alakítja át), `code` „GET /api/v1/devices”, `data` „24,8 °C”, `data-lg` „1284”.

### 3.6 Szabályok

- Nagy méreten vékony, kis méreten normál: `display-xl` 200, `display` 250, `heading-1` 300.
- 18 px alatt 400-nál vékonyabb súly nem megy.
- A Hanken Grotesknak nincs táblázatos számjegye. Oszlopba rendezett szám és mérési adat ezért mindig `data` (vagy `data-lg`) stílusú, mono betűvel.
- Folyószöveg sorhossza legfeljebb `layout-prose` (640 px).
- Gépi értéket tartalmazó beviteli mező (IP-cím, azonosító) mono betűt kap.
- Alkalmazott betűk: kizárólag a két család. Rendszerfontra csak a CSS-verem tartalékaként esik vissza a böngésző.

---

## 4. Tér és rács

### 4.1 Térközskála

4 px-es rács. A Tailwind alapskálája pontosan a `space-*` tokenekkel egyezik (`p-4` = `space-4` = 16 px).

| Token | Érték | Tailwind-fokozat | Használat |
| --- | --- | --- | --- |
| `space-1` | 4px | `1` | Ikon és felirata között. |
| `space-2` | 8px | `2` | Mezőcsoporton belül, jelvény belső margója. |
| `space-3` | 12px | `3` | Kompakt kártya, táblázatcella. |
| `space-4` | 16px | `4` | Alap kártyabelső, mobil oldalmargó. |
| `space-5` | 20px | `5` | Űrlapmezők között. |
| `space-6` | 24px | `6` | Kártyák közti rés, tágas kártyabelső, alkalmazás-tartalom margója. |
| `space-8` | 32px | `8` | Tágas tartalommargó, weboldal oldalsó belső margója. |
| `space-10` | 40px | `10` | Szekción belüli tagolás. |
| `space-12` | 48px | `12` | Kisebb webes szekciótávolság. |
| `space-16` | 64px | `16` | Tablet szekciótávolság. |
| `space-24` | 96px | `24` | Asztali webes szekciótávolság. |

### 4.2 Sűrűség

| Felület | Jelleg | Előírás |
| --- | --- | --- |
| Alkalmazás | Sűrű | `space-3` és `space-4` a kártyákon belül, `space-6` a tartalom margója. |
| Weboldal | Levegős | Asztalon `space-24` a szekciók között, `layout-site` (1200 px) tartalomszélesség, mobilon `space-4` oldalmargó. |

Töréspontok. Az arculat saját töréspontot nem rögzít; a Tailwind alap `md` (768 px) és `lg` (1024 px) töréspontja érvényes:

| Szélesség | Weboldal | Alkalmazás |
| --- | --- | --- |
| 768 px alatt (mobil) | Minden rács 1 oszlop (szolgáltatások, munkamódszer); hero cím `display` a `display-xl` helyett; oldalmargó `space-4`; szekciótávolság `space-12`; a kapcsolatblokk tartalma egymás alá kerül, a gomb a cím alatt áll. | A metrikák rácsa 1 oszlop; az oldalsáv a shadcn-svelte `Sidebar` mobil (offcanvas) módjára vált, a felső sáv bal szélére `Sidebar.Trigger` kerül; tartalommargó `space-4`. |
| 768–1023 px (tablet) | Szolgáltatások és munkamódszer 2 oszlop; szekciótávolság `space-16`; oldalmargó `space-8`. | A metrikák rácsa 2 oszlop; az oldalsáv `offcanvas` módban, alapból zárva; tartalommargó `space-6`. |
| 1024 px-től (asztal) | A 12.1 szerinti teljes elrendezés: 3, illetve 4 oszlop, `space-24` szekciótávolság. | A 12.2 szerinti teljes elrendezés: 240 px-es oldalsáv, 4 oszlopos metrikasor. |

A széles adattábla mobilon vízszintesen görgethető tárolóba kerül (`overflow-x: auto`), az oldal maga nem görget vízszintesen.

### 4.3 Layout tokenek

Rögzített méretek.

| Token | Érték | Használat |
| --- | --- | --- |
| `layout-site` | 1200px | Webes tartalom legnagyobb szélessége. |
| `layout-prose` | 640px | Folyószöveg legnagyobb szélessége. |
| `layout-sidebar` | 240px | Alkalmazás-oldalsáv szélessége. |
| `layout-topbar` | 52px | Alkalmazás felső sávjának magassága. |

A `space-*`, `chamfer-*` és `layout-*` tokenek CSS-változóként érhetők el a `:root`-on, például `max-width: var(--layout-prose)` vagy Tailwindben `max-w-[var(--layout-prose)]`.

---

## 5. Forma

### 5.1 Sugarak

A sarkok szinte szögletesek.

| Token | Érték | Tailwind | Használat |
| --- | --- | --- | --- |
| `radius-sm` | 3px | `rounded-sm` | Jelvény, jelölőnégyzet, kód a sorban. |
| `radius-md` | 4px | `rounded-md` | Gomb, beviteli mező, menüelem. |
| `radius-lg` | 6px | `rounded-lg` | Kártya, dialógus, legördülő. |
| `radius` | 6px | – | A shadcn-svelte alapsugara, ebből ered a `rounded-lg`. |
| `radius-xl` … `radius-4xl` | 6px | `rounded-xl` … `rounded-4xl` | Szándékosan 6 px, hogy a shadcn-komponensek ne kapjanak nagyobb sugarat, és ne legyen pill alak. |
| `radius-full` | 9999px | `rounded-full` | Csak állapotpont és kapcsoló. Gombra nem. |

### 5.2 Levágott sarok (chamfer)

A márka kézjegye. Egy kapszula levágott sarkából ered. Mindig a jobb felső és a bal alsó sarkot vágja. Kivétel a borító moduláris tömbjei, ahol a vágott sarkot a 10.6 poligonjai rögzítik.

| Token | Érték | Használat |
| --- | --- | --- |
| `chamfer-sm` | 8px | Metrikakártya, kisméretű panel (`Panel size="sm"`). |
| `chamfer-md` | 14px | Alap `Panel`. |
| `chamfer-lg` | 24px | Hero panel, borító. Az app ikon 128 egységes rajzán a vágás 22 egység (7.7). |

Hol szabad:

- `Panel` komponensen (metrika, szolgáltatáskártya, hero blokk, weboldal kapcsolatblokk).
- A logón és az app ikonon.
- Márkafelületen (borító).

Hol tilos:

- Űrlap, gomb, táblázat, dialógus.
- Bármi, ami legördülőt vagy tooltipet tartalmaz. A `clip-path` levágja a kilógó tartalmat és a fókuszgyűrűt. Ilyenkor shadcn `Card` kell.

Megvalósítás (`kx-panel`): a keret a külső réteg, a kitöltés a `::before`. A kitöltés 1 px-es belső eltolással ül a kereten, és 1,414 px-szel (√2) nagyobb vágást kap. Így az átlón a keret kb. 2,4 px széles; azonos vágással kb. 1,4 px lenne.

```css
.kx-panel { --kx-c: var(--chamfer-md); --kx-fill: var(--card); --kx-edge: var(--border); position: relative; isolation: isolate; background: var(--kx-edge); clip-path: polygon(0 0,calc(100% - var(--kx-c)) 0,100% var(--kx-c),100% 100%,var(--kx-c) 100%,0 calc(100% - var(--kx-c))); }
.kx-panel::before { --kx-c: calc(var(--kx-cut, var(--chamfer-md)) + 1.414px); content: ""; position: absolute; inset: 1px; z-index: -1; background: var(--kx-fill); clip-path: polygon(0 0,calc(100% - var(--kx-c)) 0,100% var(--kx-c),100% 100%,var(--kx-c) 100%,0 calc(100% - var(--kx-c))); }
.kx-panel-sm { --kx-c: var(--chamfer-sm); --kx-cut: var(--chamfer-sm); }
.kx-panel-lg { --kx-c: var(--chamfer-lg); --kx-cut: var(--chamfer-lg); }
```

A `--kx-fill` a kitöltést, a `--kx-edge` a keretet állítja. Kiemelt kerethez: `[--kx-edge:var(--primary)]`.

### 5.3 Vonal és mélység

- A tagolás eszköze az 1 px-es `border`, nem az árnyék.
- Kártya, panel, táblázat: `border` kerettel, árnyék nélkül.
- Másodlagos, erősebb vonal sötét alapon nagy ábrán: `ink-500`.

### 5.4 Árnyék

- Árnyék csak ténylegesen lebegő elemen van: legördülő, dialógus, tooltip. Értéke `shadow-popover` (2.3).
- Tailwindben: `shadow-2xs`, `shadow-xs`, `shadow-sm` üres (`0 0 #0000`), így a shadcn gomb, mező és kártya nem kap árnyékot. A `shadow-md` és a `shadow-lg` a `shadow-popover` értékét adja.

### 5.5 Derengés (glow)

- A `glow` az egyetlen fénytoken. Csak színérték, kész `box-shadow` nincs hozzá.
- Csak márkagrafikában, az aktív csomópont körül. Szöveg mögé nem kerül.
- Adatfelületen nincs derengés.

### 5.6 Mérőléc

- A `kx-ticks` egy műszeres skála rajzolata: 8 px-es finom és 40 px-es hangsúlyos osztás, 10 px magas.
- Szekció vagy idővonal tagolására való, oldalanként legfeljebb kétszer.
- Nem dekoráció a kártyák szélére. Tartalmat nem fogad, a szülő szélességét kitölti.

```css
.kx-ticks { display: block; height: 10px; background-image: linear-gradient(to right, var(--input) 1px, transparent 1px), linear-gradient(to right, var(--border) 1px, transparent 1px); background-size: 40px 10px, 8px 5px; background-repeat: repeat-x; background-position: 0 0; }
```

A `linear-gradient` itt éles, 1 px-es vonalat rajzol, nem színátmenet.

---

## 6. Mozgás és interakciós állapotok

### 6.1 Mozgás

| Elem | Időtartam | Görbe |
| --- | --- | --- |
| Hover (gomb, menüpont, sor) | 120 ms | `ease-out` |
| Panel és dialógus nyitása, zárása | 200 ms | `ease-out` |

- Rövid és funkcionális.
- Nincs ugrálás, parallaxis, görgetésre induló belépő animáció.
- Az állapotpont nem pulzál folyamatosan.
- A helix nem mozog.
- `prefers-reduced-motion` esetén minden animáció kikapcsol:

```css
@media (prefers-reduced-motion: reduce) {
	*, ::before, ::after { animation-duration: .01ms !important; transition-duration: .01ms !important; }
}
```

### 6.2 Interakciós állapotok

| Állapot | Megjelenés |
| --- | --- |
| Hover | `accent` felület. Kitöltött gombon enyhe eltolás a `foreground` felé (sötét témában világosít, világosban sötétít): `color-mix(in oklab, var(--primary) 88%, var(--foreground))`. Linkszerű elemen `muted-foreground` helyett `foreground`. |
| Fókusz | 2 px tömör `ring`, 2 px eltolással, minden interaktív elemen. |
| Aktív | `primary` jelölő (aktív fül 2 px-es `primary` vonal, aktív menüpont ikonja `sidebar-primary`). |
| Letiltott | 50% átlátszóság, `not-allowed` kurzor. |
| Hibás mező | `aria-invalid="true"`, keret `destructive`. |
| Kijelölt szöveg | `selection` háttér. |

A fókusz CSS-e. A második szabály rétegen kívül áll, így felülírja a shadcn-svelte komponensek `outline-none` és 3 px-es félátlátszó ring párosát:

```css
:focus-visible { outline: 2px solid var(--ring); outline-offset: 2px; }

[data-slot]:focus-visible { outline: 2px solid var(--ring); outline-offset: 2px; --tw-ring-color: transparent; }
```

---

## 7. Logó

### 7.1 Felépítés

- A jel egy K, amelynek két karja csomópontban ér véget: egy hálózati gráf és a helix lépcsője egyszerre.
- A felső csomópont a `primary` színű. Ez a pont nem kap más színt, csak az egyszínű változatban.
- A szöveg körvonalazott, font nem kell hozzá.

### 7.2 Változatok és fájlok

Minden SVG-fájl: `xmlns="http://www.w3.org/2000/svg"`, `role="img"`, `aria-label="Kinexium"`, `<title>Kinexium</title>`, csak `viewBox`, `width`/`height` nélkül.

| Fájl | Változat | viewBox | Alap | Színek |
| --- | --- | --- | --- | --- |
| `kinexium-lockup-on-dark.svg` | Jel + név (elsődleges) | `0 0 232.6 64` | Sötét | tinta `#d4dcd5`, csomópont `#77dbed` |
| `kinexium-lockup-on-light.svg` | Jel + név | `0 0 232.6 64` | Világos | tinta `#0b171f`, csomópont `#16637e` |
| `kinexium-mark-on-dark.svg` | Csak jel | `0 0 64 64` | Sötét | tinta `#d4dcd5`, csomópont `#77dbed` |
| `kinexium-mark-on-light.svg` | Csak jel | `0 0 64 64` | Világos | tinta `#0b171f`, csomópont `#16637e` |
| `kinexium-wordmark-on-dark.svg` | Szóvédjegy ™-mal, jel nélkül | `0 0 513.7 71.7` | Sötét | egyszínű `#d4dcd5` |
| `kinexium-wordmark-on-light.svg` | Szóvédjegy ™-mal, jel nélkül | `0 0 513.7 71.7` | Világos | egyszínű `#0b171f` |
| `kinexium-appicon.svg` → `static/favicon.svg` | Favicon és alkalmazásikon | `0 0 128 128` | Saját `#10212a` alap | tinta `#d4dcd5`, csomópont `#77dbed` |

### 7.3 Hol melyik

| Helyzet | Változat |
| --- | --- |
| Alap, fejléc, általános márkázás | Lockup (sötét alapon `on-dark`, világos alapon `on-light`) |
| Favicon, app ikon, szűk fejléc | Csak jel (favicon: appicon) |
| Lábléc, dokumentum, borító | Szóvédjegy ™-mal |
| Felületen, Svelte-komponensben | A `Logo` komponens (10.13) |
| Oldalon kívül (e-mail, PDF) | A fenti SVG-fájlok |

A lockupban nincs ™, csak a szóvédjegyben.

### 7.4 Védőtér és legkisebb méret

- Védőtér körben: a csomópont átmérőjének kétszerese. A jel 64 egységes rajzán a csomópont átmérője 11,2 egység, a védőtér tehát 22,4 egység, a logó magasságának 35%-a (32 px magas lockupnál 11,2 px).
- Legkisebb méret: lockup 24 px magas, jel 16 px magas.

### 7.5 Színek témánként

| Rész | Sötét alapon | Világos alapon | Egyszínű |
| --- | --- | --- | --- |
| Tinta (szár, ágak, alsó csomópont, szöveg) | `mist-100` `#d4dcd5` | `ink-850` `#0b171f` | a környező szövegszín |
| Felső csomópont | `primary` = `signal-300` `#77dbed` | `primary` = `signal-600` `#16637e` | a tintával azonos |
| App ikon alapja | `ink-800` `#10212a` | – | – |

Márkafelületi változatok:

| Alap | Tinta | Felső csomópont |
| --- | --- | --- |
| `signal-700` | `mist-100` | `signal-300` |
| `signal-300` (egyszínű) | `ink-950` | `ink-950` |

### 7.6 Tiltások

Ne torzítsd, ne forgasd, ne tegyél rá árnyékot, derengést, színátmenetet, és ne rakd fotóra. A jel útvonalait ne írd át.

### 7.7 SVG-szerkezet

A K jel geometriája (a jel és a lockup bal fele):

- Vonal: `M16 8V56M16 38 45 13M27.5 28 45 51`, `fill="none"`, `stroke-width="4.5"`. Függőleges szár x=16-on y 8..56 között, felső ág (16,38) → (45,13), alsó ág (27.5,28) → (45,51).
- Felső csomópont: `<circle cx="47" cy="11.5" r="5.6">`, csomópontszín.
- Alsó csomópont: `<circle cx="47" cy="53" r="5.6">`, tintaszín.

Jel sötét alapra:

```html
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" role="img" aria-label="Kinexium"><title>Kinexium</title><path d="M16 8V56M16 38 45 13M27.5 28 45 51" stroke="#d4dcd5" stroke-width="4.5" fill="none"/><circle cx="47" cy="11.5" r="5.6" fill="#77dbed"/><circle cx="47" cy="53" r="5.6" fill="#d4dcd5"/></svg>
```

Jel világos alapra:

```html
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" role="img" aria-label="Kinexium"><title>Kinexium</title><path d="M16 8V56M16 38 45 13M27.5 28 45 51" stroke="#0b171f" stroke-width="4.5" fill="none"/><circle cx="47" cy="11.5" r="5.6" fill="#16637e"/><circle cx="47" cy="53" r="5.6" fill="#0b171f"/></svg>
```

App ikon (`static/favicon.svg`). Négyzet, a jobb felső és a bal alsó sarok 22 egységgel levágva, a jel vastagabb, 6,4 egységes vonallal és nagyobb, 6,8 egység sugarú csomóponttal a kis méretek miatt:

```html
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" role="img" aria-label="Kinexium"><title>Kinexium</title><path d="M0 0H106L128 22V128H22L0 106Z" fill="#10212a"/><g transform="translate(26 26) scale(1.19)"><path d="M16 8V56M16 38 45 13M27.5 28 45 51" stroke="#d4dcd5" stroke-width="6.4" fill="none"/><circle cx="47" cy="11.5" r="6.8" fill="#77dbed"/><circle cx="47" cy="53" r="6.8" fill="#d4dcd5"/></g></svg>
```

Lockup: a jel mellett egyetlen `<path>` a KINEXIUM betűkkel (x 76.97..228.65, y 20.15..44.16). Szóvédjegy: két `<path>`, az első a KINEXIUM betűk (x 8.74..454.87, y 1..71.62), a második a ™ jel a jobb felső sarokban (x 470.19..510.06, y 1..21.91). A teljes útvonalak a `Logo` komponens forrásában szerepelnek (10.13). A `kinexium-lockup-*` és a `kinexium-wordmark-*` fájl összeállítása: a 7.2 gyökérattribútumai és `<title>Kinexium</title>`, utána a 10.13 megfelelő útvonalai, `currentColor` és `var(--primary)` helyett a 7.2 táblázatának hex értékeivel (tinta `#d4dcd5` / `#0b171f`, csomópont `#77dbed` / `#16637e`; a szóvédjegy mindkét útvonala tintaszínű).

---

## 8. Ikonok

### 8.1 Szabályok

- Készlet: Lucide, a shadcn-svelte alapkészlete, csomag: `@lucide/svelte`.
- Vonalvastagság: `strokeWidth={1.5}`, hogy illeszkedjen a vékony betűkhöz. Az egyetlen kivétel a jelölőnégyzet pipája (`stroke-width="2.5"`, 10.5).
- Az ikon a szövegszínt örökli (`stroke="currentColor"`). `primary` csak aktív elemen és szolgáltatáskártyán.
- Állapotikon az állapotszínt kapja, a mellette álló szó `foreground` marad (2.4).
- Csak ikonos gombra `aria-label` kell. Szöveg melletti ikon `aria-hidden="true"`.
- Emoji és Unicode-szimbólum nem helyettesít ikont.
- Egyenkénti import tartja kicsiben a bundle-t. A listán kívüli Lucide ikon is használható, ugyanazzal a vonalvastagsággal.

### 8.2 Méretek

| Hely | Méret | Tailwind |
| --- | --- | --- |
| UI (gomb, riasztás, mező) | 16 px | `size-4` |
| Navigáció (önálló navigációs ikon) | 20 px | `size-5` |
| Alkalmazás-oldalsáv menüpontja | 16 px | `size-4` |
| Weboldal, szolgáltatáskártya (`primary` színnel) | 24 px | `size-6` |
| Állapotjelvény | 12 px | `size-3` |
| Hibaüzenet a mező alatt | 14 px | `size-3.5` |

### 8.3 Használat

```svelte
<script lang="ts">
	import Cpu from '@lucide/svelte/icons/cpu';
</script>

<Cpu class="size-4" strokeWidth={1.5} aria-hidden="true" />
```

Nyers SVG-attribútumok: `viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"`.

### 8.4 A kiválasztott ikonkészlet

32 ikon. A „Rögzített szerep” oszlop csak ott van kitöltve, ahol egy komponens az ikont meghatározott célra használja.

| # | Lucide név | Import | Rögzített szerep |
| --- | --- | --- | --- |
| 1 | `cpu` | `@lucide/svelte/icons/cpu` | Eszközök menüpont |
| 2 | `radio-tower` | `@lucide/svelte/icons/radio-tower` | – |
| 3 | `network` | `@lucide/svelte/icons/network` | – |
| 4 | `server` | `@lucide/svelte/icons/server` | – |
| 5 | `cable` | `@lucide/svelte/icons/cable` | – |
| 6 | `brain-circuit` | `@lucide/svelte/icons/brain-circuit` | Konzultáció szolgáltatáskártya |
| 7 | `workflow` | `@lucide/svelte/icons/workflow` | Szabályok menüpont |
| 8 | `activity` | `@lucide/svelte/icons/activity` | Mérések menüpont |
| 9 | `gauge` | `@lucide/svelte/icons/gauge` | – |
| 10 | `thermometer` | `@lucide/svelte/icons/thermometer` | – |
| 11 | `database` | `@lucide/svelte/icons/database` | – |
| 12 | `terminal` | `@lucide/svelte/icons/terminal` | API-kulcsok menüpont |
| 13 | `shield-check` | `@lucide/svelte/icons/shield-check` | – |
| 14 | `bell` | `@lucide/svelte/icons/bell` | Értesítések gomb, Riasztások menüpont |
| 15 | `settings` | `@lucide/svelte/icons/settings` | Beállítások gomb és menüpont |
| 16 | `sliders-horizontal` | `@lucide/svelte/icons/sliders-horizontal` | – |
| 17 | `layout-dashboard` | `@lucide/svelte/icons/layout-dashboard` | Áttekintés menüpont |
| 18 | `search` | `@lucide/svelte/icons/search` | Keresés |
| 19 | `plus` | `@lucide/svelte/icons/plus` | Felvétel („Eszköz felvétele”) |
| 20 | `download` | `@lucide/svelte/icons/download` | Export |
| 21 | `copy` | `@lucide/svelte/icons/copy` | Másolás (kódblokk) |
| 22 | `circle-check` | `@lucide/svelte/icons/circle-check` | Online, siker |
| 23 | `triangle-alert` | `@lucide/svelte/icons/triangle-alert` | Figyelmeztetés |
| 24 | `circle-alert` | `@lucide/svelte/icons/circle-alert` | Kritikus, hiba |
| 25 | `wifi-off` | `@lucide/svelte/icons/wifi-off` | Offline |
| 26 | `info` | `@lucide/svelte/icons/info` | Szinkronizál, tájékoztatás |
| 27 | `sun` | `@lucide/svelte/icons/sun` | Téma váltása |
| 28 | `moon` | `@lucide/svelte/icons/moon` | – |
| 29 | `mail` | `@lucide/svelte/icons/mail` | Kapcsolat (e-mail gomb) |
| 30 | `map-pin` | `@lucide/svelte/icons/map-pin` | – |
| 31 | `globe` | `@lucide/svelte/icons/globe` | – |
| 32 | `arrow-right` | `@lucide/svelte/icons/arrow-right` | Hero fő gomb |

Kiegészítő ikon az alkalmazás-felületen: `chevron-down` (`@lucide/svelte/icons/chevron-down`, legördülő mező), `chevron-right` (`@lucide/svelte/icons/chevron-right`, morzsamenü elválasztója, 14 px), `trash-2` (`@lucide/svelte/icons/trash-2`, törlés gomb), `check` (`@lucide/svelte/icons/check`, jelölőnégyzet pipája).

Kiegészítő ikon a weboldalon: `compass` (`@lucide/svelte/icons/compass`, Rendszertervezés szolgáltatáskártya) és `code-xml` (`@lucide/svelte/icons/code-xml`, Szoftverfejlesztés szolgáltatáskártya), 24 px, `primary`.

Az ikonkészlet bemutató rácsa: 8 oszlop, cellánként 20 px-es ikon `foreground` színnel, alatta a név mono 10px/14px `muted-foreground` betűvel, a cellák között 1 px `border` színű rés. Cella: flex oszlop, középre igazítva, rés `space-2`, belső margó `space-3 space-1`, háttér `background`. A rácsnak 1px `border` kerete és `radius-lg` sugara van.

---

## 9. Képi világ

### 9.1 Alapelv

- Az elsődleges grafika a vonalas helix: két szál, hajszálvékony lépcsők, a lépcsők végén csomópont. A K jel nyelve is ez: vonal, amely csomópontban ér véget.
- Fotó helyett rendszerábra és valódi felületkép.
- Nincs stock fotó, robotkéz, kék agy, lebegő hologram.
- 3D-render (például DNS-render) és fotóból vett színminta publikus felületre nem kerül; a palettát a 2. fejezet tokenjei adják.

### 9.2 A helix minta (önálló SVG, oldalon kívüli használatra)

Fájlok: `helix-on-dark.svg` sötét alapra, `helix-on-light.svg` világos alapra. Oldalon kívül (dokumentum, prezentáció) teljes szélességű sávként, 5:2 arányban; a webes projektbe nem kerülnek, felületen a 9.3 szerinti beágyazott, témát váltó Helix szerepel. A fájlok az alábbi képletekből állnak elő (egy tizedesjegyre kerekítve, eltérés 0,1 alatt).

| Paraméter | Érték |
| --- | --- |
| viewBox | `0 0 1200 480` (5:2), `aria-hidden="true"`, háttér nélkül |
| Formátum | gyökér `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 480" aria-hidden="true">`; szál `<path d="M0,240.0 8,224.6 16,209.4 …" fill="none" stroke="…" stroke-width="1"/>` (x,y párok szóközzel elválasztva, az y egy tizedesjegyre; az elülső szálon `stroke-width="1.25"`); lépcső `<path d="M17.6 273.7V206.3" stroke="…" stroke-width="1"/>`; csomópont `<circle cx="17.6" cy="273.7" r="4.4" fill="…"/>` |
| Elemek | 36 `<path>` (2 szál + 34 lépcső), 68 `<circle>` |
| Periódus | P = 1200 / 2,25 ≈ 533,33 px (a rajz szélességén 2,25 periódus) |
| Szálak | 2 db, 151 pont, 8 px-es x-lépés (x = 0..1200), középvonal y = 240, kilengés ±163,2 (y 76.8..403.2), ellentétes fázisban. Mindkettő x=0-nál y=240-ből indul, az első felfelé, a második lefelé. Hátsó (első) szál: y = 240 − 163,2 · sin(2πx / P); elülső (második) szál: y = 240 + 163,2 · sin(2πx / P). |
| Hátsó szál | `stroke-width="1"`, halvány szín |
| Elülső szál | `stroke-width="1.25"`, erősebb szín |
| Lépcsők | 34 db függőleges vonal a két szál között, `stroke-width="1"`, x = 17.6-tól 1182.4-ig, 1200/34 ≈ 35,29 px lépésközzel (az i-edik lépcső x-e, i = 0..33: xᵢ = (i + 0,5) · 1200 / 34); a lépcső az elülső szál xᵢ-beli pontjától a hátsó szál xᵢ-beli pontjáig fut |
| Kiemelt lépcsők | 5 db, minden 7. a 4.-től (i = 3, 10, 17, 24, 31): x = 123.5, 370.6, 617.6, 864.7, 1111.8 |
| Csomópontok | lépcsőnként 2 (a két végén), sugár 2.0..4.4 között a mélység szerint. Az elöl lévő nagyobb, a hátul lévő kisebb. Képlet: az elülső szál végén r = 3,2 + 1,2 · cos(2πxᵢ / P), a hátsó szál végén 6,4 − r; a nagyobb sugarú csomópont az elülső (`input` színű), a kisebb a hátsó (`border` színű). Példa: x=17.6-nál elöl r=4.4 (cy 273.7), hátul r=2.0 (cy 206.3). |
| Darabszámok | 29 elülső csomópont, 29 hátsó csomópont, 10 kiemelt csomópont |
| Rajzolási sorrend | 1. a hátsó szál (1 px); 2. a 34 lépcső x szerint növekvő sorrendben, mindegyik közvetlenül a saját két csomópontjával: előbb a lépcső, utána az elülső szál pontján álló csomópont, végül a hátsó szál pontján álló csomópont (a csomópont színét a sugara adja, nem a szál); 3. legutoljára az elülső szál (`stroke-width="1.25"`), amely így a lépcsők és a csomópontok fölött fut. |

Színek:

| Elem | Sötét fájl | Világos fájl | Token-megfelelő |
| --- | --- | --- | --- |
| Hátsó szál, normál lépcső, hátsó csomópont | `#253d46` | `#c9d4d0` | `border` |
| Elülső szál, elülső csomópont | `#5f7b7d` | `#6b8482` | `input` |
| Kiemelt lépcső és csomópontjai | `#77dbed` | `#16637e` | `primary` |

### 9.3 A Helix komponens (beágyazott SVG)

Felületbe ágyazva a helix osztályokkal színeződik, így témát vált. Az alábbi blokk a `src/app.css` `@layer components` rétegében áll (13.2).

```css
.kx-helix { display: block; width: 100%; height: auto; }
.kx-helix .b { stroke: var(--border); fill: none; }   /* hátsó szál, normál lépcső */
.kx-helix .f { stroke: var(--input); fill: none; }    /* elülső szál */
.kx-helix .h { stroke: var(--primary); }              /* kiemelt lépcső */
.kx-helix .nb { fill: var(--border); }                /* hátsó csomópont */
.kx-helix .nf { fill: var(--input); }                 /* elülső csomópont */
.kx-helix .nh { fill: var(--primary); }               /* kiemelt csomópont */
```

| Paraméter | Érték |
| --- | --- |
| Gyökér | `<svg class="kx-helix" viewBox="0 0 1200 300" aria-hidden="true">`; a weboldal teljes szélességű sávjában `viewBox="0 0 1280 300"` (paraméterei az utolsó sorban) |
| Szálak | `path.b` és `path.f`, formátum `<path class="b" d="M0,150.0 12,134.8 24,119.9 …"/>` (x,y párok szóközzel elválasztva, az y egy tizedesjegyre), pontlista 12 px-es x-lépéssel, középvonal y = 150, kilengés ±108 (y 42..258), ellentétes fázisban; x=0-nál y=150, a `b` szál felfelé indul; x = 0..1200 (101 pont); `b`: y = 150 − 108 · sin(2πx / P), `f`: y = 150 + 108 · sin(2πx / P), P ≈ 533,33 px |
| Lépcsők | 34 db: 29 `path.b` + 5 `path.h`, formátum `<path class="b" d="M{x} {y1}V{y2}"/>`, xᵢ = (i + 0,5) · 1200 / 34, vagyis ≈ 35,29 px-enként 17.6-tól 1182.4-ig; a lépcső az `f` szál pontjától a `b` szál pontjáig fut |
| Kiemelt lépcsők | x=123.5 (y 257.3–42.7), 370.6 (48.4–251.6), 617.6 (240.5–59.5), 864.7 (75.4–224.6), 1111.8 (204.7–95.3) |
| Csomópontok | formátum `<circle class="nf" cx="17.6" cy="172.3" r="4.4"/>`; `nf` r ≈ 3.2–4.4, `nb` r ≈ 2.0–3.2, `nh` r ≈ 2.2–4.2 (a mélység szerint), egy lépcsőn r_f + r_b ≈ 6,4; sugárképlet a 9.2 szerint; példa: `M17.6 172.3V127.7`, `nf` r=4.4 (cy 172.3), `nb` r=2.0 (cy 127.7) |
| Darabszámok | 29 `nf`, 29 `nb`, 10 `nh` kör; 30 `path.b` (1 szál + 29 lépcső), 5 `path.h`, 1 `path.f` |
| Rajzolási sorrend | `path.b` szál → lépcsőnként, x szerint növekvő sorrendben: `<path class="b" d="M{x} {yf}V{yb}"/>` (kiemeltnél `class="h"`), majd a `yf`-beli és a `yb`-beli `<circle>` (a nagyobb sugarú `nf`, a kisebb `nb`, kiemelt lépcsőn mindkettő `nh`) → végül `path.f`, legfelül. Az 1280×300-as sávban ugyanígy. |
| Vonalvastagság | Nincs `stroke-width` sem attribútumként, sem CSS-ben: minden vonal 1 egység. Az elülső szál 1.25-ös vastagsága csak a 9.2 önálló SVG-fájljaira érvényes. |
| Weboldal-sáv (1280×300) | 36 lépcső: 31 `path.b` + 5 `path.h`, xᵢ = (i + 0,5) · 1280 / 36, vagyis ≈ 35,56 px-enként 17.8-tól 1262.2-ig (első: `M17.8 172.5V127.5`); kiemelt lépcsők i = 3, 10, 17, 24, 31: x=124.4 (y 257.4–42.6), 373.3 (47.3–252.7), 622.2 (243.5–56.5), 871.1 (69.7–230.3), 1120.0 (213.5–86.5); szálak 12 px-es lépéssel x = 0..1272 (107 pont), középvonal 150, kilengés ±108, P ≈ 533,33 px; `nf` r 3.3–4.4, `nb` r 2.0–3.1, `nh` r 2.2–4.2; 31 `nf`, 31 `nb`, 10 `nh` kör; 32 `path.b` (1 szál + 31 lépcső), 5 `path.h`, 1 `path.f` |

### 9.4 Mit szabad, mit nem

| Szabad | Nem szabad |
| --- | --- |
| Hero alatt, teljes szélességben, oldalanként egyszer | Szöveg alá tenni |
| A kiemelt lépcsőket `primary` színnel jelölni | Mozgatni, animálni |
| Márkagrafikában `glow` a csomópont körül | Színátmenetet, árnyékot adni |
| Borítón a lépcsősor motívumként (10.6) | Adatfelületen derengést használni |

---

## 10. Komponensek

### 10.0 Közös tudnivalók

- 19 komponens van. Öt saját Svelte márkakomponens (`Logo`, `Metric`, `Panel`, `StatusBadge`, `TickRule`), helyük `src/lib/components/brand/`. Nyolc shadcn-svelte komponensre épül, az arculatot a `src/app.css` adja (13. fejezet): `Alert`, `Button`, `Controls`, `DataTable`, `Dialog`, `Field`, `Tabs`, `LineChart` (`Chart`). A komponensfájlok változatlanok, egy kivétellel: a `Dialog` és az `AlertDialog` háttérfátyla (10.8). Ahol a referencia-megjelenéshez a hívásnál kell prop vagy osztály, az az adott komponensnél szerepel (`Alert` 10.1, `DataTable` 10.7, `Tabs` 10.18). A shadcn-svelte belső fájlnevei és alaposztályai (a 10.8 fátyolfájljai és `bg-black/…` osztálya, a 10.18 `data-[state=active]` osztálylistája, a 10.1 variánsai) a shadcn-svelte szokásos felépítésén alapulnak, telepített komponenseken nem voltak kipróbálva: az `add` futtatása után a generált fájlokon ellenőrizendők. Három oldalminta: `AppPage`, `SitePage`, `Cover`. A `Helix` beágyazott SVG (9.3), az `Iconography` ikonkészlet (8. fejezet), a `CodeBlock` saját jelölés (10.4).
- A shadcn-svelte változónevei és a tokenek nevei azonosak, ezért a komponensek módosítás nélkül átveszik az arculatot.
- A `src/app.css` a következő márkaosztályokat tartalmazza: `kx-eyebrow`, `kx-num`, `kx-panel`, `kx-panel-sm`, `kx-panel-lg`, `kx-ticks`, `kx-dot`, `kx-dot-ring`, `kx-helix` (9.3), valamint a típusstílusokat.
- Az alábbi HTML-minták `kx-*` komponensosztályai (`kx-btn`, `kx-card`, `kx-alert` stb.) referenciajelölések: a pontos méreteket és színeket rögzítik. Nem részei a `src/app.css`-nek. Felületen a megfelelő shadcn-svelte komponens adja ugyanezt a megjelenést. Ha mégis CSS-ként kerülnek a projektbe, csak a `src/app.css` mellett működnek, mert a `--radius-*` és `--font-*` változókat az adja.

Közös referenciaosztályok:

| Osztály | Érték |
| --- | --- |
| `kx-card` | háttér `card`, szín `card-foreground`, keret 1px `border`, sugár `radius-lg` |
| `kx-card-head` | flex, függőlegesen középre, `space-between`, rés `space-3`, belső margó `space-3 space-4`, alsó keret 1px `border` |
| `kx-card-title` | 13px / 20px, súly 600 |
| `kx-pad` | belső margó `space-4` |
| `kx-row` | flex, sortörés, függőleges közép, rés `space-3` |
| `kx-col` | flex oszlop, rés `space-3` |
| `kx-dot` | 8×8 px kör, `currentColor` kitöltés, `radius-full` |
| `kx-dot-ring` | 7×7 px, átlátszó, `box-shadow: inset 0 0 0 1.5px currentColor` |
| link (`a`) | szín `primary`, aláhúzva, `text-underline-offset: 3px`. Felületen a szöveges link osztálya `text-primary underline underline-offset-3`; a `src/app.css` az `a` elemet globálisan nem színezi, mert a Tailwind alapstílusa a linkről leveszi a színt és az aláhúzást, egy globális `a` szabály pedig a linkként renderelt `Button` és oldalsáv-menüpont színét is átírná. |

```css
.kx-dot { display: inline-block; width: 8px; height: 8px; flex: none; border-radius: var(--radius-full); background: currentColor; }
.kx-dot-ring { width: 7px; height: 7px; background: transparent; box-shadow: inset 0 0 0 1.5px currentColor; }
```

### 10.1 Alert

- Rendeltetés: tartós üzenet az oldalon belül. shadcn-svelte `Alert`.
- Tartalom: ikon, cím, szöveg, opcionálisan egy `outline` `sm` gomb. A cím mondja meg, mi történt, a szöveg, hogy mi a teendő.
- Szabályok:
  - Kritikus üzenetre `role="alert"`. Figyelmeztetés és siker nem kap `role`-t.
  - Az alap `card` felület marad, csak az ikon színes.
  - Színes bal szegély nincs.
  - shadcn-svelte jelölés: minden állapotban `variant="default"`, az állapotszín az ikon osztályán (`<CircleAlert class="text-destructive" strokeWidth={1.5} aria-hidden="true" />`). A `destructive` variáns nem használható, mert a címet és a szöveget is színezi.

| Variáns | Ikon | Ikonszín |
| --- | --- | --- |
| `kx-alert-critical` | `circle-alert` | `destructive` |
| `kx-alert-warning` | `triangle-alert` | `warning` |
| `kx-alert-success` | `circle-check` | `success` |
| `kx-alert-info` | `info` | `info` |

Méretek: rács `16px 1fr auto`, elemek felülre igazítva (`align-items: start`), rés `2px space-3`, belső margó `space-3 space-4`, háttér `card`, keret 1px `border`, sugár `radius-lg`. Ikon 16×16, `margin-top: 2px`, két soron át. Cím 13px/20px súly 600. Szöveg a 2. oszlopban, 13px/20px, `muted-foreground`. Gomb a 3. oszlopban, mindkét soron át, függőlegesen középen.

```html
<div class="kx-alert kx-alert-critical" role="alert">
	<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><!-- circle-alert --></svg>
	<div class="kx-alert-title">A GW-04 gateway nem elérhető</div>
	<div class="kx-alert-body">Utolsó üzenet 14:02-kor. Ellenőrizd a hálózati kapcsolatot.</div>
	<button class="kx-btn kx-btn-outline kx-btn-sm">Részletek</button>
</div>
```

További minták: figyelmeztetés „Közeledik a riasztási küszöb” / „A B csarnok hőmérséklete 2 °C-ra van a határértéktől.”; siker „Firmware frissítve” / „12 eszköz fut az 1.8.2-es verzión.”

### 10.2 AppPage

- Rendeltetés: a webalkalmazás oldalmintája: oldalsáv, felső sáv, metrikák, diagram, táblázat. A shadcn-svelte `Sidebar` komponensére épül.
- Szabályok: a sűrűség ennyi legyen, a metrikák mindig az oldal tetején állnak. Teljes szerkezet: 12.2.

### 10.3 Button

- Rendeltetés: művelet indítása. shadcn-svelte `Button`, módosítás nélkül.
- Tartalom: felirat, opcionálisan 16 px-es Lucide ikon.

```svelte
<Button>Eszköz felvétele</Button>
<Button variant="outline" size="sm">Export</Button>
<Button variant="destructive">Eszköz törlése</Button>
```

| Változat (`variant`) | Referenciaosztály | Megjelenés | Hover |
| --- | --- | --- | --- |
| `default` | `kx-btn-primary` | háttér `primary`, szöveg `primary-foreground` | `color-mix(in oklab, var(--primary) 88%, var(--foreground))` |
| `secondary` | `kx-btn-secondary` | háttér `secondary`, szöveg `secondary-foreground` | háttér `accent` |
| `outline` | `kx-btn-outline` | átlátszó, keret `input` | háttér `accent` |
| `ghost` | `kx-btn-ghost` | átlátszó, keret nélkül | háttér `accent` |
| `destructive` | `kx-btn-destructive` | háttér `destructive`, szöveg `destructive-foreground` | – |
| `link` | `kx-btn-link` | szín `primary`, aláhúzva, eltolás 3px, magasság és belső margó nélkül | – |

| Méret (`size`) | Magasság | Vízszintes belső margó | Betű |
| --- | --- | --- | --- |
| `sm` | 28px | `space-3` | 13px |
| `default` | 36px | `space-4` | 13px |
| `lg` | 44px | `space-6` | 15px |
| `icon` | 36×36px | 0 | – |

Alapértékek: inline-flex, középre igazítva, ikon és felirat között `space-2`, 13px/20px súly 500, `white-space: nowrap`, átlátszó háttér, szín `foreground`, 1px átlátszó keret, sugár `radius-md`, ikon 16×16. Letiltott: átlátszóság .5, `cursor: not-allowed`.

Szabályok:

- Képernyőnként egy `default` (primary) gomb.
- Webes hero: `lg`. Alkalmazás: `default` és `sm`.
- Csak ikonos gombra `aria-label` kell.
- Pill alak nincs, a sarok `rounded-md`.
- Árnyék nincs.

### 10.4 CodeBlock

- Rendeltetés: kód és parancs megjelenítése `code` stílussal.
- Megvalósítás: shadcn-svelte megfelelője nincs; felületen saját jelölés a lenti értékekkel (Tailwind-osztályokkal vagy saját CSS-sel).
- Szabályok:
  - Egy kiemelőszín elég: kulcsszó `primary`, komment `muted-foreground`.
  - A fejlécben fájlnév (`kx-eyebrow`) és másolás gomb (`ghost`, `sm`, csak ikon, 28 px, `aria-label="Másolás"`, `copy` ikon).
  - Soron belüli kód és billentyű: `muted` alap, `radius-sm`.

| Rész | Érték |
| --- | --- |
| Doboz | háttér `muted`, keret 1px `border`, sugár `radius-lg`, `overflow: hidden` |
| Fejléc | flex, `space-between`, belső margó `6px 6px 6px space-4`, alsó keret 1px `border` |
| `pre` | margó 0, belső margó `space-3 space-4`, `overflow-x: auto`, mono 13px/20px, `foreground` |
| Kulcsszó (`kx-c-k`) | `primary` |
| Komment (`kx-c-m`) | `muted-foreground` |
| Soron belüli kód (`kx-inline-code`), billentyű (`kx-kbd`) | mono 12px, belső margó `1px 5px`, keret 1px `border`, `radius-sm`, háttér `muted` |

```html
<div class="kx-codeblock">
	<div class="kx-codeblock-head">
		<span class="kx-eyebrow">main.go</span>
		<button class="kx-btn kx-btn-ghost kx-btn-sm kx-btn-icon" aria-label="Másolás" style="width:28px"><!-- copy ikon --></button>
	</div>
	<pre><span class="kx-c-m">// A frontend a binárisba kerül</span>
<span class="kx-c-k">//go:embed</span> all:build
<span class="kx-c-k">var</span> frontend embed.FS

mux.Handle(<span class="kx-c-k">"GET /"</span>, http.FileServerFS(sub))</pre>
</div>
```

### 10.5 Controls (Checkbox, Switch)

- Rendeltetés: kétállapotú vezérlők. shadcn-svelte `Checkbox` és `Switch`.
- Szabályok:
  - Checkbox űrlapon, mentéskor érvényesül.
  - Switch azonnal hat (például eszköz be- és kikapcsolása).
  - Mindig van mellette szöveges címke.
  - A bekapcsolt állapot `primary`.

| Rész | Érték |
| --- | --- |
| Címkés sor (`kx-check`) | inline-flex, rés `space-2`, 13px/20px, `cursor: pointer` |
| Jelölőnégyzet (`kx-box`) | 16×16, keret 1px `input`, `radius-sm`; bejelölve háttér és keret `primary`, pipa `primary-foreground`, 12×12, `stroke-width="2.5"`, útvonal `M20 6 9 17l-5-5` |
| Kapcsoló (`kx-switch`) | 32×18, `radius-full`, háttér `muted`, `box-shadow: inset 0 0 0 1px var(--input)` |
| Kapcsoló gombja | 12×12, `radius-full`, top 3px, left 3px, `muted-foreground` |
| Kapcsoló bekapcsolva | háttér `primary`, árnyék nélkül; a gomb left 17px, `primary-foreground` |

```html
<span class="kx-check"><span class="kx-box kx-box-on"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg></span>Riasztás e-mailben</span>
<span class="kx-check"><span class="kx-box"></span>Heti jelentés</span>
<span class="kx-check"><span class="kx-switch kx-switch-on"></span>Eszköz aktív</span>
<span class="kx-check"><span class="kx-switch"></span>Karbantartási mód</span>
```

### 10.6 Cover

- Rendeltetés: dokumentum címlapja, prezentáció nyitóképe, megosztási előnézet.
- Megvalósítás: nem felületi komponens, hanem önálló, a `src/app.css` nélkül is működő HTML/SVG lap: a lenti teljes dokumentum a fontokat és a felhasznált változókat maga definiálja. Sötét alapú, témaváltás nincs.
- Szerkezet: bal oldalon a név vékony (200-as súlyú) betűvel és rövid tagline. Jobb oldalon a márka moduláris tömbjei, jellemzően `chamfer-lg` (24 px) vágással: `signal-300`, `signal-700`, `signal-600`, `ink-700`, `mist-100`. Köztük a helix lépcsőinek sora `signal-400` vonallal és `signal-300` csomóponttal. Ez márkafelület, ezért a skálatokenek közvetlenül szerepelnek.
- Szabályok:
  - Arány 10:3, legalább 960×288 px. Social előnézethez 1200×630 px, dokumentumcímlaphoz álló A4; mindkettő értékei a lap alatt.
  - A tömbök a jobb és a felső szélen kifutnak, a szöveg a `layout-prose` szélességen belül marad.
  - A tagline cserélhető, a név nem.
  - Fotót, derengést, színátmenetet nem kap.

Tömbök (a 24 px-es, `space-6` lépésű rácson, x ≥ 520):

| Tömb | Méret | Vágott sarok | Szín |
| --- | --- | --- | --- |
| Nagy | 264×144 | jobb alsó, 24 (`chamfer-lg`) | `signal-300` |
| Felső kicsi | 144×72 | bal alsó, 24 | `signal-700` |
| Csík | 144×24 | nincs | `signal-600` |
| Alsó széles | 288×120 | jobb alsó, 24 | `ink-700` |
| Négyzet | 120×120 | bal felső, 24 | `mist-100` |

Lépcsősor: 15 lépcső, 16 px-es (`space-4`) lépésköz, csomópont r=3. A tömbök vágott sarka `chamfer-lg` (24 px), lekerekítés sehol nincs.

```html
<!doctype html>
<html lang="hu">
<head>
<meta charset="utf-8">
<title>Kinexium</title>
<style>
@font-face { font-family: "Hanken Grotesk"; src: url("/fonts/HankenGrotesk-Variable.woff2") format("woff2"); font-weight: 100 900; font-style: normal; font-display: swap; }
@font-face { font-family: "Geist Mono"; src: url("/fonts/GeistMono-Variable.woff2") format("woff2"); font-weight: 100 900; font-style: normal; font-display: swap; }
:root {
	--ink-900: #071016; --ink-700: #192d37;
	--mist-300: #94a9a5; --mist-100: #d4dcd5;
	--signal-300: #77dbed; --signal-400: #52aec8; --signal-600: #16637e; --signal-700: #0c4962;
	--background: var(--ink-900); --foreground: var(--mist-100); --muted-foreground: var(--mist-300);
	--font-sans: "Hanken Grotesk", ui-sans-serif, system-ui, sans-serif;
	--font-mono: "Geist Mono", ui-monospace, "SF Mono", Menlo, monospace;
}
html, body { height: 288px; }
body { position: relative; overflow: hidden; margin: 0; background: var(--background); font-family: var(--font-sans); }
svg.art { position: absolute; inset: 0; width: 960px; height: 288px; }
.s3 { fill: var(--signal-300); } .s7 { fill: var(--signal-700); } .s6 { fill: var(--signal-600); }
.mist { fill: var(--mist-100); } .ink { fill: var(--ink-700); }
.rung { stroke: var(--signal-400); stroke-width: 1; fill: none; } .node { fill: var(--signal-300); }
.name { position: absolute; left: 40px; bottom: 58px; margin: 0; width: 440px; font-family: var(--font-sans); font-weight: 200; font-size: 88px; line-height: .95; letter-spacing: .005em; color: var(--foreground); }
.tag { position: absolute; left: 42px; bottom: 26px; margin: 0; width: 430px; font-size: 14px; line-height: 20px; color: var(--muted-foreground); }
</style>
</head>
<body>
<svg class="art" viewBox="0 0 960 288" aria-hidden="true">
	<polygon class="s7" points="552,0 696,0 696,72 576,72 552,48"/>
	<polygon class="s3" points="696,0 960,0 960,120 936,144 696,144"/>
	<rect class="s6" x="696" y="144" width="144" height="24"/>
	<polygon class="ink" points="552,168 840,168 840,264 816,288 552,288"/>
	<polygon class="mist" points="864,168 960,168 960,288 840,288 840,192"/>
	<!-- lépcsők: <path class="rung" d="M{x} {y1}V{y2}"/><circle class="node" cx="{x}" cy="{y1}" r="3"/><circle class="node" cx="{x}" cy="{y2}" r="3"/> -->
</svg>
<h1 class="name">KINEXIUM</h1>
<p class="tag">IoT- és AI-rendszerek tervezése és fejlesztése.</p>
</body>
</html>
```

A fontok útvonala (`/fonts/…`) a lap helyéhez igazítandó, ha a lap nem a webalkalmazásból töltődik be.

| Lépcső x | y1..y2 | Lépcső x | y1..y2 | Lépcső x | y1..y2 |
| --- | --- | --- | --- | --- | --- |
| 568 | 183..273 | 664 | 196..260 | 760 | 204..252 |
| 584 | 188..268 | 680 | 187..269 | 776 | 217..239 |
| 600 | 197..259 | 696 | 183..273 | 792 | 211..245 |
| 616 | 208..248 | 712 | 182..274 | | |
| 632 | 219..237 | 728 | 186..270 | | |
| 648 | 207..249 | 744 | 194..262 | | |

A lépcsők elemsorrendje: a tömbök után lépcsőnként a `path.rung`, utána az `y1`-beli, majd az `y2`-beli `circle.node`, x szerint növekvő sorrendben.

Social előnézet (1200×630). Ugyanaz a dokumentum, a rajz 1,25-szörös nagyításban a felső szélhez igazítva (a tömbök felül és jobbra kifutnak), a név és a tagline a lap aljához igazítva. A 960×288-as lap stílusát ezek a sorok írják felül:

```css
html, body { height: 630px; }
svg.art { inset: auto; top: 0; left: 0; width: 1200px; height: 360px; }
.name { left: 50px; bottom: 72px; width: 550px; font-size: 110px; }
.tag { left: 52px; bottom: 32px; width: 540px; font-size: 18px; line-height: 24px; }
```

Dokumentumcímlap (álló A4, 210×297 mm, 595×842 pt). Felül a tömbök és a lépcsősor nagyítva, alatta a lockup, a cím, a tagline és egy szemöldökfelirat-sor:

| Elem | Érték |
| --- | --- |
| Háttér | `background` (`ink-900`), kifutóan |
| Felső sáv | a fenti `svg.art` rajz `viewBox="540 0 420 288"` kivágással, teljes lapszélességben: 210×144 mm (595×408 pt), a lap tetejéhez igazítva. A tömbök a felső és a jobb szélen kifutnak, a `signal-700` és az `ink-700` tömb bal széle 17 pt-ra (6 mm) áll; a lépcsősor az `ink-700` tömbön fut. |
| Bal margó | 48 pt (17 mm), a lockup, a cím, a tagline és a szemöldökfelirat közös bal éle |
| Lockup | a sötét alapra szánt lockup (7.7): tinta `#d4dcd5`, felső csomópont `#77dbed`; 37 pt (13 mm) magas, teteje 524 pt-nál (185 mm) |
| Cím | a dokumentum címe, például „Design System”; sans, 46 pt, súly 200, `foreground`, alapvonala 695 pt-nál (245 mm) |
| Tagline | „IoT- és AI-rendszerek tervezése és fejlesztése.”; sans, 12 pt, súly 400, `muted-foreground`, alapvonala 733 pt-nál (259 mm) |
| Szemöldökfelirat | `kx-eyebrow` sor (mono, súly 500, betűköz .08em, nagybetűsítve), 8 pt, `muted-foreground`, alapvonala 776 pt-nál (274 mm); szövege a dokumentum típusa, a domain és az év, például „Áttekintés · kinexium.net · 2026” |

A cím és a szemöldökfelirat dokumentumonként cserélhető, a lockup és a tagline nem.

### 10.7 DataTable

- Rendeltetés: sűrű adattábla. shadcn-svelte `Table`, igény szerint TanStack Table-lel.
- Tartalom: oszlopok és sorok.
- Szabályok:
  - A szöveg balra, a szám jobbra igazodik, a számok `font-mono`.
  - Az azonosító `muted-foreground` mono.
  - Fejléc: `muted` alap, `label` stílus, nagybetűvel.
  - Sormagasság 40 px körül.
  - Hiányzó érték: `–`.
  - Hover: `accent`.
  - Az állapot oszlopban `StatusBadge`.
  - Kártyába ágyazva, `overflow: hidden`, levágott sarok nélkül.

| Rész | Érték |
| --- | --- |
| Tábla | teljes szélesség, `border-collapse: collapse`, 13px/20px |
| `th` | belső margó `space-2 space-4`, balra, mono 11px/16px súly 500, betűköz .08em, nagybetű, `muted-foreground`, háttér `muted`, alsó keret 1px `border`, nem törik |
| shadcn `TableHead` | `h-auto px-4 py-2 font-mono text-[11px] leading-4 font-medium tracking-[0.08em] uppercase text-muted-foreground bg-muted` |
| `td` | belső margó `9px space-4`, alsó keret 1px `border`, nem törik; az utolsó sorban keret nélkül |
| `tbody tr:hover` | háttér `accent` |
| `.num` | mono, jobbra |
| `.id` | mono, `muted-foreground` |

Mintaadat (fejléc: Azonosító, Hely, Állapot, Hőm., Késl., Utolsó üzenet; az utolsó három `.num`):

| Azonosító | Hely | Állapot | Hőm. | Késl. | Utolsó üzenet |
| --- | --- | --- | ---: | ---: | ---: |
| GW-01 | Csarnok A | Online | 24,8 °C | 38 ms | 14:32:08 |
| GW-02 | Csarnok B | Figyelmeztetés | 29,1 °C | 61 ms | 14:32:05 |
| GW-03 | Raktár | Online | 18,3 °C | 44 ms | 14:32:07 |
| GW-04 | Kazánház | Kritikus | – | – | 14:02:41 |
| SN-17 | Hűtőkamra 2 | Offline | – | – | 09:15:20 |
| SN-21 | Tetőszint | Szinkronizál | 12,6 °C | 52 ms | 14:31:58 |

```html
<div class="kx-card" style="overflow:hidden">
	<table class="kx-table">
		<thead><tr><th>Azonosító</th><th>Hely</th><th>Állapot</th><th class="num">Hőm.</th><th class="num">Késl.</th><th class="num">Utolsó üzenet</th></tr></thead>
		<tbody>
			<tr><td class="id">GW-01</td><td>Csarnok A</td><td><span class="kx-badge kx-s-online">…Online</span></td><td class="num">24,8&nbsp;°C</td><td class="num">38&nbsp;ms</td><td class="num">14:32:08</td></tr>
		</tbody>
	</table>
</div>
```

### 10.8 Dialog

- Rendeltetés: megerősítés vagy rövid feladat. shadcn-svelte `Dialog` és `AlertDialog`.
- Tartalom: cím, leírás, két gomb.
- Szabályok:
  - Visszafordíthatatlan műveletnél a cím kérdés, a gomb megnevezi a tárgyat („Eszköz törlése”), a leírás kimondja a következményt.
  - Levágott sarok itt nincs.
  - Nyitás és zárás 200 ms, `ease-out`.

| Rész | Érték |
| --- | --- |
| Háttérfátyol | flex, középre, belső margó `space-6`, `color-mix(in oklab, var(--ink-950) 62%, transparent)`. shadcn-svelte-ben a fátyol színe nem a stíluslapból jön: a `src/lib/components/ui/dialog/dialog-overlay.svelte` és a `src/lib/components/ui/alert-dialog/alert-dialog-overlay.svelte` háttérosztálya (`bg-black/…`) helyére `bg-[color-mix(in_oklab,var(--ink-950)_62%,transparent)]` kerül. |
| Doboz | szélesség 100%, legfeljebb 440px, háttér `popover`, szöveg `popover-foreground`, keret 1px `border`, `radius-lg`, árnyék `shadow-popover`, belső margó `space-6`, flex oszlop, rés `space-4` |
| Cím (`h3`) | 18px/26px, súly 500 |
| Leírás (`p`) | 13px/20px, `muted-foreground` |
| Lábléc | flex, jobbra zárva, rés `space-2` |

```html
<div class="kx-scrim">
	<div class="kx-dialog" role="dialog" aria-modal="true" aria-labelledby="t">
		<h3 id="t">Törlöd a GW-04 eszközt?</h3>
		<p>A mérési előzmények megmaradnak, de az eszköz többé nem küldhet adatot. A művelet nem vonható vissza.</p>
		<div class="kx-dialog-foot">
			<button class="kx-btn kx-btn-outline">Mégse</button>
			<button class="kx-btn kx-btn-destructive">Eszköz törlése</button>
		</div>
	</div>
</div>
```

### 10.9 Field

- Rendeltetés: címkézett beviteli mező. shadcn-svelte `Label`, `Input`, `Select`.
- Tartalom: címke, érték, súgó, hiba.
- Szabályok:
  - A címke mindig látszik, a placeholder nem helyettesíti.
  - Gépi értékhez (IP-cím, azonosító) `font-mono`.
  - Hibánál `aria-invalid="true"`, és a mező alatt ikonos `destructive` üzenet, amely megmondja a teendőt.
  - Levágott sarok nincs.

| Rész | Érték |
| --- | --- |
| Mezőcsoport (`kx-field`) | flex oszlop, rés 6px, `min-width: 0` |
| Címke | 13px/20px, súly 500 |
| Súgó | 13px/20px, `muted-foreground` |
| Hiba | flex, rés 6px, 13px/20px, `destructive`, ikon 14×14 |
| Mező (`kx-input`) | szélesség 100%, magasság 36px, belső margó `0 space-3`, 13px, `foreground`, átlátszó háttér, keret 1px `input`, `radius-md`; placeholder `muted-foreground`; hibásan keret `destructive` |
| Legördülő (`kx-select`) | a mező stílusa + flex, `space-between`, rés `space-2`, `cursor: pointer`, chevron 16×16 `muted-foreground` |
| Mono mező | `font-family: var(--font-mono)` |
| Mezők közti rés | 20px (`space-5`) |

```html
<label class="kx-field"><span class="kx-label">Eszköz neve</span><input class="kx-input" placeholder="pl. GW-01 Csarnok A"><span class="kx-help">Legfeljebb 32 karakter.</span></label>
<div class="kx-field"><span class="kx-label">Protokoll</span><div class="kx-input kx-select" role="combobox" tabindex="0" aria-expanded="false">Modbus RTU <!-- chevron-down --></div><span class="kx-help">A gateway beállításai szerint.</span></div>
<label class="kx-field"><span class="kx-label">IP-cím</span><input class="kx-input kx-mono-input" aria-invalid="true" value="192.168.1.300"><span class="kx-error"><!-- circle-alert --> Érvénytelen IPv4-cím.</span></label>
```

### 10.10 Helix

- Rendeltetés: a márka vonalas grafikája: két szál, lépcsők, csomópontok.
- Szabályok: hero alatt teljes szélességben, oldalanként egyszer. Szöveg alá nem kerül. A kiemelt lépcsők `primary` színűek, a többi `border` és `input`. Nem mozog.
- Paraméterek és CSS: 9.2 és 9.3.

### 10.11 Iconography

- Rendeltetés: a választott Lucide ikonkészlet 1,5 px vonallal.
- Szabályok, méretek és a teljes lista: 8. fejezet.

### 10.12 LineChart

- Rendeltetés: idősor. shadcn-svelte `Chart` (LayerChart), `chart-*` színekkel.
- Tartalom: adatsorok és konfiguráció.

```ts
import * as Chart from '$lib/components/ui/chart/index.js';

const config = {
	a: { label: 'A csarnok', color: 'var(--chart-1)' },
	b: { label: 'B csarnok', color: 'var(--chart-2)' }
} satisfies Chart.ChartConfig;
```

Szabályok: 11. fejezet. Rajzolati értékek:

| Elem | Érték |
| --- | --- |
| SVG | `display: block; width: 100%; height: auto; overflow: visible` (Tailwind: `block w-full h-auto overflow-visible`); a tároló `position: relative`, ehhez igazodik a tooltip |
| Vonal | 2 px, `stroke-linecap="round"`, `stroke-linejoin="round"`, kitöltés nélkül |
| Végpont | r=4, adatsor-színű, 2 px `card` színű gyűrűvel |
| Rács és tengely | 1 px, `border`; vízszintes rácsvonal `x1 = L`, `x2 = W − R + 6` |
| Hover-vonal | függőleges, 1 px, `input`, `y1 = T`, `y2 = H − B` |
| Hover-pont | adatsoronként r=4, adatsor-színű, 2 px `card` színű gyűrűvel, a hover-vonalon |
| Tengelyfelirat | mono 11px, `muted-foreground`; y-címke `x = L − 8`, `y = Y(v) + 4`, `text-anchor="end"`; x-címke `y = H − 6`, `text-anchor="middle"` |
| Közvetlen címke a sor végén | sans 12px, `foreground`, a végpont után 10 px-re (`x = X(n−1) + 10`, `y = Y + 4`) |
| Jelmagyarázat | flex, sortörés, rés `space-4`, 12px/16px, `foreground`; jelölő 12×2 px, sugár 1px, rés 6px |
| Tooltip | `popover` alap, `popover-foreground`, keret 1px `border`, `radius-md`, `shadow-popover`, legalább 148px, belső margó `space-2 space-3`, 12px/18px, `z-index: 2`, nem kattintható; időpont mono súly 400 `muted-foreground`; soronként flex, `space-between`, rés `space-4`: a sor neve sans, az érték mono, tizedesvesszővel és „ °C” egységgel |
| Táblázatnézet gombja | `ghost` `sm`, felirata „Táblázat” / „Diagram” |
| Akadálymentesség | `<svg role="img" aria-label="Hőmérséklet 24 órában, három zóna">` |

Mintageometria: `W = max(420, szélesség)`, `H = 240` (alkalmazásoldalon 190), margók `L=28, R=84, T=8, B=24`, értéktartomány 16..26, `X(i) = L + i·(W−L−R)/(n−1)`, `Y(v) = T + (max−v)·(H−T−B)/(max−min)`. Vízszintes rácsvonalak 16, 18, 20, 22, 24, 26 értéknél, időcímkék minden 5. ponton (`HH:MM`). Hover: a `pointermove` a legközelebbi pontra ugrik, a `pointerleave` elrejti; a tooltip a kiválasztott pont x-ától jobbra 12 px-re áll; ha az x + 160 px túllóg a diagram szélességén, a ponttól balra 164 px-re; `top: 8px`.

Mintaadat (16 pont, 96 percenként, 24 óra):

| Sor | Szín | Értékek (°C) |
| --- | --- | --- |
| A csarnok | `chart-1` | 21.2, 21.0, 20.8, 20.7, 20.9, 21.6, 22.8, 23.9, 24.6, 24.9, 25.2, 25.1, 24.8, 24.4, 23.9, 23.4 |
| B csarnok | `chart-2` | 19.1, 19.0, 18.9, 18.8, 19.0, 19.4, 20.1, 20.9, 21.5, 22.0, 22.3, 22.5, 22.2, 21.8, 21.1, 20.6 |
| Raktár | `chart-3` | 16.4, 16.3, 16.2, 16.2, 16.3, 16.5, 16.9, 17.4, 17.8, 18.1, 18.3, 18.3, 18.1, 17.8, 17.4, 17.0 |

Szerkezet: `kx-card` → fejléc (cím „Hőmérséklet, 24 óra (°C)” + táblázatnézet gomb) → 16 px belső margójú oszlop: jelmagyarázat, diagram, rejtett táblázat (fejléc: Idő + adatsoronként egy jobbra igazított oszlop; sorai a 0., 5., 10. és 15. pont: 00:00, 08:00, 16:00, 24:00; az időcella `.id` stílusú, az értékek tizedesvesszővel és „ °C” egységgel). A gomb a diagram és a táblázat között vált, egyszerre csak az egyik látszik.

### 10.13 Logo

- Rendeltetés: a logó három alakban: jel (`mark`), `lockup`, szóvédjegy (`wordmark`).
- Szabályok: a színt a szülő `text-*` osztálya adja, a felső csomópont `primary`. A jel útvonalait ne írd át. Oldalon kívüli használatra az SVG-fájlok valók (7.2).

| Prop | Érték | Alap |
| --- | --- | --- |
| `variant` | `mark`, `lockup`, `wordmark` | `lockup` |
| `height` | px | `32` |
| `class` | szöveg (`text-*` színosztály) | `''` |

```svelte
<Logo variant="lockup" height={32} class="text-foreground" />
```

Bemutató rács: 2 oszlop, 1 px `border` színű rés, keret 1px `border`, `radius-lg`; cella flex oszlop, `space-between`, rés `space-4`, legalább 148px magas, belső margó `space-4`, háttér `background`. Bemutató cellák: lockup 44 px; szóvédjegy 30 px; lockup 40 px `signal-700` alapon (`mist-100` tinta, `signal-300` csomópont); jel 48 px egyszínűen `signal-300` alapon (`ink-950` tinta és csomópont).

`src/lib/components/brand/Logo.svelte`:

```svelte
<script lang="ts">
	/**
	 * KINEXIUM logó. A színt a környező `text-*` adja, a csomópont a `primary`.
	 * variant: mark (jel), lockup (jel + név), wordmark (szóvédjegy ™-mal).
	 */
	type Props = { variant?: 'mark' | 'lockup' | 'wordmark'; height?: number; class?: string };
	let { variant = 'lockup', height = 32, class: className = '' }: Props = $props();
	const box = { mark: '0 0 64 64', lockup: '0 0 232.6 64', wordmark: '0 0 513.7 71.7' };
</script>

<svg viewBox={box[variant]} {height} class={className} role="img" aria-label="Kinexium">
	{#if variant !== 'wordmark'}
		<path d="M16 8V56M16 38 45 13M27.5 28 45 51" stroke="currentColor" stroke-width="4.5" fill="none" />
		<circle cx="47" cy="11.5" r="5.6" fill="var(--primary)" />
		<circle cx="47" cy="53" r="5.6" fill="currentColor" />
	{/if}
	{#if variant === 'lockup'}
		<path d="M78.44 36V33.52L91.27 20.15H93.86ZM76.97 43.85V20.15H78.88V43.85ZM91.69 43.85 83.24 30.48 84.67 29.15 93.94 43.85ZM98.83 43.85V20.15H100.75V43.85ZM107.71 43.85V20.15H109.84L122.79 41.4H122.42V20.15H124.34V43.85H122.15L109.24 22.58H109.62V43.85ZM131.3 43.85V20.15H146.75V21.88H133.22V30.96H145.31V32.69H133.22V42.12H146.75V43.85ZM150.28 43.85 158.59 31.89 150.44 20.15H152.74L159.91 30.51L167.08 20.15H169.37L161.22 31.89L169.57 43.85H167.27L159.89 33.32L152.54 43.85ZM174.48 43.85V20.15H176.4V43.85ZM191.44 44.16Q187.41 44.16 185.26 41.88Q183.1 39.61 183.1 35.28V20.15H185.02V35.08Q185.02 37.45 185.7 39.08Q186.39 40.71 187.82 41.54Q189.24 42.38 191.44 42.38Q193.66 42.38 195.1 41.52Q196.54 40.66 197.24 39.01Q197.95 37.37 197.95 34.99V20.15H199.84V35.11Q199.84 39.52 197.68 41.84Q195.52 44.16 191.44 44.16ZM206.58 43.85V20.15H209.46L217.83 40.06H217.44L225.81 20.15H228.65V43.85H226.74V22.35L226.85 22.37L218.57 41.91H216.68L208.38 22.37L208.49 22.35V43.85Z" fill="currentColor" />
	{/if}
	{#if variant === 'wordmark'}
		<path d="M13.07 47.61V40.33L50.81 1H58.41ZM8.74 70.7V1H14.37V70.7ZM52.04 70.7 27.17 31.38 31.39 27.47 58.65 70.7ZM73.04 70.7V1H78.67V70.7ZM99.14 70.7V1H105.41L143.49 63.5H142.42V1H148.05V70.7H141.63L103.65 8.15H104.77V70.7ZM168.54 70.7V1H213.97V6.1H174.17V32.79H209.73V37.89H174.17V65.6H213.97V70.7ZM224.35 70.7 248.81 35.51 224.83 1H231.6L252.67 31.46L273.77 1H280.49L256.52 35.51L281.08 70.7H274.31L252.61 39.72L231.01 70.7ZM295.54 70.7V1H301.17V70.7ZM345.41 71.62Q333.57 71.62 327.23 64.92Q320.89 58.22 320.89 45.5V1H326.52V44.92Q326.52 51.87 328.54 56.66Q330.57 61.45 334.76 63.91Q338.95 66.37 345.41 66.37Q351.93 66.37 356.17 63.84Q360.41 61.31 362.48 56.47Q364.55 51.63 364.55 44.63V1H370.13V45Q370.13 57.97 363.76 64.79Q357.4 71.62 345.41 71.62ZM389.94 70.7V1H398.41L423.02 59.55H421.89L446.49 1H454.87V70.7H449.24V7.47L449.57 7.52L425.21 64.99H419.65L395.24 7.52L395.57 7.47V70.7Z" fill="currentColor" />
		<path d="M477.62 21.91V2.53H470.19V1H486.74V2.53H479.33V21.91ZM490.58 21.91V1H493.12L500.51 18.57H500.17L507.55 1H510.06V21.91H508.37V2.94L508.47 2.96L501.16 20.2H499.49L492.17 2.96L492.27 2.94V21.91Z" fill="currentColor" />
	{/if}
</svg>
```

### 10.14 Metric

- Rendeltetés: egyetlen mért érték kiemelve.
- Szabályok:
  - Sorban legfeljebb négy.
  - Trendnyíl csak valódi összehasonlítás mögött.
  - Az érték színe `foreground`, nem `primary`.
  - A `value` már formázott szöveg, a kerekítés és a számformátum (1.8) a hívó dolga.
- Felépítés: `Panel size="sm"` (`chamfer-sm`) → `kx-eyebrow` címke, `data-lg` érték + 14 px-es `muted-foreground` egység 4 px-es bal margóval és 0 betűközzel (`tracking-normal`, mert a `data-lg` -0.02em betűköze különben öröklődik), `body-sm` `muted-foreground` megjegyzés. Belső margó `space-4`, rés `space-2`.

| Prop | Típus | Tartalom |
| --- | --- | --- |
| `label` | `string`, kötelező | rövid név |
| `value` | `string`, kötelező | formázott érték |
| `unit` | `string`, opcionális | mértékegység |
| `note` | `string`, opcionális | összevetés vagy kontextus |

```svelte
<Metric label="Átlagos késleltetés" value="42" unit="ms" note={'Tegnap: 47 ms'} />
```

Mintaértékek (a `/ 132` az egység helyén áll, egység nélküli értéknél a `unit` elmarad):

```svelte
<Metric label="Online eszköz" value="128" unit="/ 132" note="4 offline" />
<Metric label="Aktív riasztás" value="3" note="1 kritikus" />
<Metric label="Átlagos késleltetés" value="42" unit="ms" note={'Tegnap: 47 ms'} />
<Metric label="Üzenet / perc" value="1284" note="Csúcs 14:10-kor" />
```

Referenciajelölés: `.kx-metrics` (rács 4 oszlop, rés `space-3`) → `.kx-panel.kx-panel-sm` → `.kx-metric` (belső margó `space-4`, flex oszlop, rés `space-2`) → `.kx-eyebrow`, `.kx-metric-value` (mono 32px/36px súly 300, betűköz -.02em; `small` 14px, betűköz 0, `muted-foreground`, bal margó 4px), `.kx-metric-delta` (flex, rés 6px, 13px/20px, `muted-foreground`).

`src/lib/components/brand/Metric.svelte`:

```svelte
<script lang="ts">
	import Panel from './Panel.svelte';

	/** Egyetlen mért érték. A `value` már formázott szöveg, a kerekítés a hívó dolga. */
	type Props = { label: string; value: string; unit?: string; note?: string };
	let { label, value, unit, note }: Props = $props();
</script>

<Panel size="sm">
	<div class="flex flex-col gap-2 p-4">
		<span class="kx-eyebrow">{label}</span>
		<span class="data-lg">
			{value}{#if unit}<small class="text-muted-foreground ml-1 text-sm tracking-normal">{unit}</small>{/if}
		</span>
		{#if note}<span class="body-sm text-muted-foreground">{note}</span>{/if}
	</div>
</Panel>
```

### 10.15 Panel

- Rendeltetés: levágott sarkú konténer, a márka kézjegye.
- Szabályok:
  - Használd: metrika, szolgáltatáskártya, hero blokk, weboldal kapcsolatblokk.
  - Ne használd: űrlap, táblázat, dialógus, bármi legördülővel vagy tooltippel, mert a `clip-path` levágja a kilógó tartalmat és a fókuszgyűrűt. Ilyenkor shadcn `Card` kell.
  - A belső margót a fogyasztó adja.
  - A `--kx-fill` a kitöltést, a `--kx-edge` a keretet állítja.
- CSS: 5.2.

| Prop | Érték | Alap |
| --- | --- | --- |
| `size` | `sm` (8 px vágás), `md` (14 px), `lg` (24 px) | `md` |
| `class` | például `[--kx-edge:var(--primary)]` kiemelt kerethez | `''` |
| `children` | `Snippet`, kötelező | – |

```svelte
<Panel size="md"><div class="p-6">…</div></Panel>
<Panel size="lg" class="[--kx-edge:var(--primary)]"><div class="p-6">…</div></Panel>
```

`src/lib/components/brand/Panel.svelte`:

```svelte
<script lang="ts">
	import type { Snippet } from 'svelte';

	/** Levágott sarkú panel. A clip-path levágja a kilógó tartalmat, popover és fókuszgyűrű ne essen rá. */
	type Props = { size?: 'sm' | 'md' | 'lg'; class?: string; children: Snippet };
	let { size = 'md', class: className = '', children }: Props = $props();
	const sizes = { sm: 'kx-panel-sm', md: '', lg: 'kx-panel-lg' };
</script>

<div class="kx-panel {sizes[size]} {className}">
	{@render children()}
</div>
```

### 10.16 SitePage

- Rendeltetés: a céges weboldal oldalmintája: hero, helix, szolgáltatások, munkamódszer, kapcsolat.
- Szabályok: a sorrend tartandó: állítás, bizonyíték, cselekvés. Oldalanként egyetlen `signal-700` blokk lehet. A szövegek javaslatok. Teljes szerkezet: 12.1.

### 10.17 StatusBadge

- Rendeltetés: eszköz vagy folyamat állapota, ikonnal és szóval.
- Szabályok:
  - A szín csak az ikonon van, a szó `foreground`.
  - A `status` az OpenAPI-ból generált enum, így az állapotlista egy helyen él.
  - Kitöltött, színes jelvény nincs.

| Prop | Érték |
| --- | --- |
| `status` | `online`, `warning`, `critical`, `offline`, `info` (kötelező) |
| `label` | opcionális, felülírja az alapszót |

| `status` | Alapszó | Lucide ikon | Ikonszín |
| --- | --- | --- | --- |
| `online` | Online | `circle-check` | `success` |
| `warning` | Figyelmeztetés | `triangle-alert` | `warning` |
| `critical` | Kritikus | `circle-alert` | `destructive` |
| `offline` | Offline | `wifi-off` | `muted-foreground` |
| `info` | Szinkronizál | `info` | `info` |

Méretek: magasság 22px, belső margó `0 space-2`, rés 6px, 12px/16px súly 500, nem törik, `foreground`, keret 1px `border`, `radius-sm`, átlátszó háttér, ikon 12×12.

```svelte
<StatusBadge status="warning" />
<StatusBadge status="online" label="Csatlakozva" />
```

Referenciajelölés: `<span class="kx-badge kx-s-online"><span class="kx-mark"><svg …/></span>Online</span>`; az ikonszínt a `.kx-s-online/-warning/-critical/-info .kx-mark` adja, az offline az alap `muted-foreground`.

`src/lib/components/brand/StatusBadge.svelte`:

```svelte
<script lang="ts">
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Info from '@lucide/svelte/icons/info';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import WifiOff from '@lucide/svelte/icons/wifi-off';

	/** Az állapotot ikon és szó együtt hordozza. A `status` az OpenAPI-ból generált enum. */
	export type Status = 'online' | 'warning' | 'critical' | 'offline' | 'info';
	type Props = { status: Status; label?: string };
	let { status, label }: Props = $props();

	const map = {
		online: { icon: CircleCheck, text: 'Online', color: 'text-success' },
		warning: { icon: TriangleAlert, text: 'Figyelmeztetés', color: 'text-warning' },
		critical: { icon: CircleAlert, text: 'Kritikus', color: 'text-destructive' },
		offline: { icon: WifiOff, text: 'Offline', color: 'text-muted-foreground' },
		info: { icon: Info, text: 'Szinkronizál', color: 'text-info' }
	} as const;
	const item = $derived(map[status]);
</script>

<span class="inline-flex h-[22px] items-center gap-1.5 rounded-sm border px-2 text-xs font-medium whitespace-nowrap">
	<item.icon class="size-3 {item.color}" strokeWidth={1.5} aria-hidden="true" />
	{label ?? item.text}
</span>
```

### 10.18 Tabs

- Rendeltetés: azonos szintű nézetek váltása. shadcn-svelte `Tabs`, vonalas (`line`) megjelenéssel: fül háttér nélkül, az aktív fül alatt 2 px-es vonal.
- Szabályok:
  - Az aktív fület 2 px-es `primary` vonal jelöli.
  - Legfeljebb öt fül.
  - Oldalak közti navigációra az oldalsáv való, nem a fül.

| Rész | Érték |
| --- | --- |
| Fülsor | flex, rés `space-6`, alsó keret 1px `border` |
| Fül | belső margó `space-2 0 10px`, 13px/20px súly 500, `muted-foreground`, háttér és keret nélkül |
| Hover és aktív fül | `foreground` |
| Aktív jelölő | `::after`, bal 0, jobb 0, alul -1px, magasság 2px, `primary` |

shadcn-svelte jelölés. A vonalas megjelenést a hívás osztályai adják, mert az alap `Tabs.List` kitöltött, lekerekített sáv; a `dark:` változatú osztályok a komponens sötét témás aktív hátterét és keretét oltják ki:

```svelte
<Tabs.Root value="attekintes">
	<Tabs.List class="h-auto w-full justify-start gap-6 rounded-none border-b bg-transparent p-0">
		<Tabs.Trigger
			value="attekintes"
			class="relative flex-none rounded-none border-0 bg-transparent px-0 pt-2 pb-2.5 text-[13px] leading-5 font-medium text-muted-foreground shadow-none hover:text-foreground data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none dark:data-[state=active]:bg-transparent dark:data-[state=active]:border-transparent data-[state=active]:after:absolute data-[state=active]:after:inset-x-0 data-[state=active]:after:-bottom-px data-[state=active]:after:h-0.5 data-[state=active]:after:bg-primary"
		>Áttekintés</Tabs.Trigger>
		<!-- Mérések, Riasztások, Konfiguráció: ugyanezzel az osztálylistával -->
	</Tabs.List>
</Tabs.Root>
```

Referenciajelölés:

```html
<div class="kx-tabs" role="tablist">
	<button class="kx-tab" role="tab" aria-selected="true">Áttekintés</button>
	<button class="kx-tab" role="tab" aria-selected="false">Mérések</button>
	<button class="kx-tab" role="tab" aria-selected="false">Riasztások</button>
	<button class="kx-tab" role="tab" aria-selected="false">Konfiguráció</button>
</div>
```

### 10.19 TickRule

- Rendeltetés: mérőléc szekciók és idővonalak tagolásához.
- Szabályok: a szülő szélességét kitölti, tartalmat nem fogad, oldalanként legfeljebb kétszer. CSS: 5.6.
- Prop: `class` (opcionális, alap `''`).

```svelte
<TickRule />
<!-- vagy: <span class="kx-ticks" aria-hidden="true"></span> -->
```

Minta: `kx-eyebrow` „01 Rendszertervezés”, alatta a mérőléc.

`src/lib/components/brand/TickRule.svelte`:

```svelte
<script lang="ts">
	/** Mérőléc. Szekció vagy idővonal tagolására, oldalanként egyszer-kétszer. */
	let { class: className = '' }: { class?: string } = $props();
</script>

<span class="kx-ticks {className}" aria-hidden="true"></span>
```

---

## 11. Diagram-szabályok

- Legfeljebb öt adatsor, rögzített sorrendben `chart-1`-től. Ötnél több sor helyett kis diagramok.
- A szín az entitást követi, nem a rangsort: szűrésnél nem színeződik át.
- Egy tengely van. Két eltérő skálához két diagram kell.
- Két vagy több sornál kötelező a jelmagyarázat, és legyen táblázatnézet is.
- Mindig legyen hover: függőleges vonal és tooltip.
- A kártya címe tartalmazza a mértékegységet: „Hőmérséklet, 24 óra (°C)”.
- Vonal 2 px, végpont 4 px sugarú, `card` színű gyűrűvel.
- Rács és tengely hajszálvékony, `border` színű.
- Állapotszín nem kerül adatsorra, `chart-*` szín nem jelöl állapotot.
- Adatfelületen nincs derengés és színátmenet.
- A tooltip és a táblázatnézet számai mono betűvel, magyar számformátummal (1.8).

---

## 12. Oldalszerkezetek

### 12.1 SitePage (weboldal)

Egyoszlopos ritmus, nagy cím, helixsáv, hajszálvonallal elválasztott szekciók. Sorrend: állítás, bizonyíték, cselekvés.

| # | Régió | Tartalom | Méretek |
| --- | --- | --- | --- |
| 1 | Keret (`kx-wrap`) | – | legfeljebb `layout-site` (1200 px), középre, oldalsó belső margó `space-8` (mobilon `space-4`) |
| 2 | Fejléc | `Logo` lockup 36 px; navigáció: Szolgáltatások, Munkamódszer, Kapcsolat (`muted-foreground`, hover `foreground`), „EN” `outline` `sm` gomb | magasság 72px, alsó keret 1px `border`, navigáció rés `space-6`, 13px/20px |
| 3 | Hero | `kx-eyebrow` „IoT · AI · Rendszertervezés”; `display-xl` cím „Az érzékelőtől a döntésig.”; `body-lg` bevezető; két `lg` gomb: `primary` „Beszéljünk a projektről” (`arrow-right` ikon a felirat után), `outline` „Szolgáltatások” | belső margó `space-24 0 space-12`; cím margó `space-4 0 space-6`, legfeljebb 900px; bevezető legfeljebb `layout-prose`, `muted-foreground`, alsó margó `space-8` |
| 4 | Helixsáv | Helix, teljes szélességben, a kereten kívül | `viewBox="0 0 1280 300"`, `aria-hidden`, 36 lépcső (paraméterei: 9.3) |
| 5 | Szolgáltatások | `kx-eyebrow` „Szolgáltatások”; `display` cím „Három feladat, egy felelős.”; 3 szolgáltatáskártya (`Panel`) | szekció belső margó `space-24 0` (felső keret nélkül, mert a helix választ el); cím margó `space-3 0 space-12`, legfeljebb 720px; rács 3 oszlop, rés `space-6`; kártya belső margó `space-6`, rés `space-3`, legalább 260px magas, ikon 24 px `primary` (01 `compass`, 02 `code-xml`, 03 `brain-circuit`), `kx-eyebrow` „01”, `heading-2` cím (margó `space-2 0 0`), `muted-foreground` szöveg (margó 0) |
| 6 | Munkamódszer | `kx-eyebrow` „Munkamódszer”; `display` cím „Négy lépés, mérhető kimenettel.”; `TickRule`; 4 lépés | szekció belső margó `space-24 0`, felső keret 1px `border`; rács 4 oszlop, rés `space-6`, felső margó `space-6`; lépés: `kx-eyebrow` „0N”, `heading-3` (margó `space-2 0`), `muted-foreground` szöveg (margó 0) |
| 7 | Kapcsolat | `Panel size="lg"` `signal-700` kitöltéssel és kerettel: `kx-eyebrow` „Kapcsolat”, `display` cím „Van egy rendszerötlete?”; `lg` `primary` gomb: `mail` ikon a felirat előtt, felirata „gabor.kaszas@kinexium.net”, `href="mailto:gabor.kaszas@kinexium.net"`. Az 1.6 „ige és tárgy” szabály alól kivétel: a felirat itt maga az e-mail-cím. | szekció alsó belső margó 64px; panel belső margó `space-12`, flex, `space-between`, alulra igazítva, rés `space-8`; szöveg `mist-100`; cím margó `space-3 0 0`; a gomb itt `signal-300` háttér, `ink-950` szöveg; eyebrow `mist-100` |
| 8 | Lábléc | „KINEXIUM™ · kinexium.net” és „6725 Szeged, Teve utca 25.” | felső keret 1px `border`, flex, `space-between`, rés `space-4`, belső margó `space-6 0`, 13px/20px, `muted-foreground` |

Szolgáltatások szövegjavaslata:

| # | Cím | Ikon | Szöveg |
| --- | --- | --- | --- |
| 01 | Rendszertervezés | `compass` | Architektúra az érzékelőtől a felhőig. Komponensek, protokollok, adatutak és költségek, leírva, mielőtt egy sor kód megszületik. |
| 02 | Szoftverfejlesztés | `code-xml` | Embedded szoftver, backend és webes felület. Karbantartható kód, automatizált build, dokumentált átadás. |
| 03 | Konzultáció | `brain-circuit` | Műszaki felülvizsgálat, AI-megvalósíthatóság, technológiaválasztás. Egyértelmű javaslat, a kockázatok megnevezésével. |

Munkamódszer szövegjavaslata:

| # | Cím | Szöveg |
| --- | --- | --- |
| 01 | Felmérés | Mit kell mérni, kinek, milyen döntéshez. |
| 02 | Rendszerterv | Rögzített architektúra és mérföldkövek. |
| 03 | Fejlesztés | Rövid ciklusok, működő köztes verziók. |
| 04 | Átadás | Üzembe helyezés, dokumentáció, támogatás. |

Hero bevezető: „Szoftverfejlesztés, konzultáció és rendszertervezés IoT- és AI-projektekhez. Egy mérnök, közvetlen kapcsolat, felelősség a teljes rendszerért.”

A kapcsolatblokk az oldal egyetlen `signal-700` blokkja, és az egyetlen hely, ahol a hero fő műveletét (kapcsolatfelvétel) egy második `primary` gomb ismétli meg.

A kapcsolatblokk Svelte-ben. A gomb színei mindkét témában a skálatokenekből jönnek, ezért a `default` gomb `primary`-ra épülő hovere helyett saját hovert kap (a 6.2 képlete, `signal-300` és `mist-100` keveréke):

```svelte
<script lang="ts">
	import Mail from '@lucide/svelte/icons/mail';
	import { Button } from '$lib/components/ui/button/index.js';
	import Panel from '$lib/components/brand/Panel.svelte';
</script>

<Panel size="lg" class="[--kx-fill:var(--signal-700)] [--kx-edge:var(--signal-700)] text-mist-100">
	<div class="flex flex-col gap-8 p-12 md:flex-row md:items-end md:justify-between">
		<div>
			<span class="kx-eyebrow text-mist-100">Kapcsolat</span>
			<h2 class="display mt-3">Van egy rendszerötlete?</h2>
		</div>
		<Button
			size="lg"
			href="mailto:gabor.kaszas@kinexium.net"
			class="bg-signal-300 text-ink-950 hover:bg-[color-mix(in_oklab,var(--signal-300)_88%,var(--mist-100))]"
		>
			<Mail class="size-4" strokeWidth={1.5} aria-hidden="true" />gabor.kaszas@kinexium.net
		</Button>
	</div>
</Panel>
```

### 12.2 AppPage (webalkalmazás)

`layout-sidebar` széles oldalsáv, `layout-topbar` magas felső sáv, a tartalom tetején metrikák, alatta riasztás, diagram és táblázat.

| # | Régió | Tartalom | Méretek |
| --- | --- | --- | --- |
| 1 | Keret | Rács: oldalsáv + fő terület | `grid-template-columns: var(--layout-sidebar) 1fr` (240px 1fr), háttér `background`; a fő terület flex oszlop, `min-width: 0` (Tailwind: `flex min-w-0 flex-col`), különben a széles táblázat az `1fr` oszlopot és vele az oldalt kitolja |
| 2 | Oldalsáv | Logó (lockup 30 px, `foreground` színű, nem örökli a `sidebar-foreground`-ot: `<Logo height={30} class="text-foreground" />`), csoportcím `kx-eyebrow` „Telephely”, menüpontok ikonnal: Áttekintés (`layout-dashboard`, aktív), Eszközök (`cpu`, számláló 132), Riasztások (`bell`, számláló 3), Mérések (`activity`), Szabályok (`workflow`); csoportcím „Rendszer”: API-kulcsok (`terminal`), Beállítások (`settings`); alul állapotsor: `success` színű `kx-dot` + „Rendszer rendben” | flex oszlop, rés `space-1`, belső margó `space-3`, háttér `sidebar`, szöveg `sidebar-foreground`, jobb keret 1px `sidebar-border`; logósor 40px magas, belső margó `0 space-2`, alsó margó `space-3`; csoportcím belső margó `space-3 space-2 space-1`; állapotsor alulra tolva, belső margó `space-2`, rés `space-2`, 12px/16px, felső keret 1px `sidebar-border` |
| 3 | Menüpont | 16 px-es ikon (`size-4`) + felirat + opcionális számláló | 32px magas, rés 10px, belső margó `0 space-2`, 13px/20px, `radius-md`; hover `sidebar-accent`; aktív (`aria-current="page"`): háttér `sidebar-accent`, szöveg `sidebar-accent-foreground`, súly 500, ikon `sidebar-primary`; számláló jobbra tolva, mono 11px, `muted-foreground` |
| 4 | Felső sáv | Morzsamenü („Telephely” › **Áttekintés**), keresőmező („Keresés az eszközök között”, `aria-label="Keresés"`), `ghost` ikongombok: Értesítések (`bell`), Téma váltása (`sun`) | magasság `layout-topbar` (52px), belső margó `0 space-6`, rés `space-3`, alsó keret 1px `border`; morzsamenü 13px/20px `muted-foreground`, az aktuális elem `foreground` súly 500, a morzsamenü flex, rés 6px, elválasztó 14 px-es `chevron-right`; kereső legfeljebb 240px, jobbra tolva |
| 5 | Tartalom | – | belső margó `space-6`, flex oszlop, rés `space-4` |
| 6 | Oldalfej | `heading-1` cím „Áttekintés”, alatta `body-sm` `muted-foreground` „Mintaadatok. Utolsó szinkron: 14:32:08”; jobbra `outline` „Export” (`download`) és `default` „Eszköz felvétele” (`plus`) | flex, alulra igazítva, `space-between`, rés `space-4` |
| 7 | Metrikák | 4 `Metric` | rács 4 oszlop, rés `space-3` |
| 8 | Riasztás | Kritikus `Alert` `role="alert"`, „Részletek” gombbal; cím „A GW-04 gateway nem elérhető”, szöveg „Utolsó üzenet 14:02-kor. Ellenőrizd a hálózati kapcsolatot.” (a teendő az 1.6 szerint kötelező része) | – |
| 9 | Diagram | Kártya: fejléc „Hőmérséklet, 24 óra (°C)” + jelmagyarázat + táblázatnézet gomb (`ghost` `sm`); `LineChart`. A gomb azért kötelező, mert a diagram három adatsoros, és két vagy több sornál táblázatnézet is kell (11. fejezet). | kártya belső margó 16px, diagram magassága 190 |
| 10 | Táblázat | Kártya: fejléc „Eszközök” + `body-sm` link „Összes eszköz” (`text-primary underline underline-offset-3`, 10.0); `DataTable` | kártya `overflow: hidden` |

A minta legkisebb magassága 860px. Az oldalsáv ikonjai 16 px-es Lucide ikonok (8.2, 8.4).

---

## 13. Tailwind-megfeleltetés

### 13.1 Token → utility

Minden színtoken Tailwind-színként érhető el, a szokásos előtagokkal (`bg-`, `text-`, `border-`, `ring-`, `fill-`, `stroke-`, `outline-`).

| Csoport | Utility-példák |
| --- | --- |
| Felületek | `bg-background`, `bg-card`, `bg-popover`, `bg-muted`, `bg-accent`, `bg-secondary`, `bg-sidebar` |
| Szöveg | `text-foreground`, `text-card-foreground`, `text-muted-foreground`, `text-primary-foreground`, `text-sidebar-foreground` |
| Kiemelés | `bg-primary`, `text-primary`, `ring-ring` |
| Vonal | `border-border`, `border-input`, `border-sidebar-border` |
| Állapot | `text-success`, `text-warning`, `text-destructive`, `text-info`, `bg-destructive` + `text-destructive-foreground` |
| Diagram | `bg-chart-1` … `bg-chart-5`, `stroke-chart-1` … `stroke-chart-5` |
| Skálák (csak márkafelületen) | `bg-ink-950` … `bg-ink-300`, `text-mist-400` … `text-mist-50`, `bg-signal-300` … `bg-signal-700` |
| Egyéb | `bg-selection`, `bg-glow` |

Az alapértelmezett keret minden elemen `border-border`. A `<body>` `bg-background text-foreground font-sans`, 15px/24px, élsimítással.

Típus és betű:

| Mire | Osztály |
| --- | --- |
| Típusstílus | `display-xl`, `display`, `heading-1`, `heading-2`, `heading-3`, `body-lg`, `body`, `body-sm`, `label`, `code`, `data`, `data-lg` |
| Család | `font-sans`, `font-mono` (más nincs) |

Sugár és árnyék:

| Utility | Érték |
| --- | --- |
| `rounded-sm` | 3px |
| `rounded-md` | 4px |
| `rounded-lg` | 6px |
| `rounded-xl`, `rounded-2xl`, `rounded-3xl`, `rounded-4xl` | 6px |
| `rounded-full` | 9999px (csak állapotpont és kapcsoló) |
| `shadow-2xs`, `shadow-xs`, `shadow-sm` | nincs árnyék (`0 0 #0000`) |
| `shadow-md`, `shadow-lg` | `shadow-popover` (csak lebegő elemen) |

Térköz: a Tailwind alapskála (`p-1` … `p-24`, `gap-*`, `m-*`) pontosan a `space-*` tokenekkel egyezik. A `chamfer-*` és `layout-*` tokenek CSS-változók: `max-w-[var(--layout-prose)]`, `w-[var(--layout-sidebar)]`.

Márkaosztályok:

| Osztály | Szerep |
| --- | --- |
| `kx-eyebrow` | Szemöldökfelirat, nagybetűs, mono |
| `kx-num` | Mono szám szövegben |
| `kx-panel`, `kx-panel-sm`, `kx-panel-lg` | Levágott sarkú panel (a `Panel` komponens használja) |
| `kx-ticks` | Mérőléc (a `TickRule` komponens használja) |
| `kx-dot`, `kx-dot-ring` | Állapotpont, kitöltött és gyűrűs |
| `kx-helix` (és `.b`, `.f`, `.h`, `.nb`, `.nf`, `.nh`) | Beágyazott, témát váltó helix (9.3) |

`data-*`: a shadcn-svelte komponensek `data-slot` attribútumot kapnak, a fókuszgyűrűt erre egy rétegen kívüli szabály egységesíti (6.2). A komponensek `data-*` variánsait és keyframe-jeit a `shadcn-svelte/tailwind.css` import adja, az animációkat a `tw-animate-css`.

### 13.2 `src/app.css`

A teljes stíluslap. A shadcn-svelte `init` után teljes cserével kerül a helyére, mert az `init` által írt `@import "shadcn-svelte/tailwind.css"` sort, `:root` változókat és `@layer base` blokkot ez a fájl mind tartalmazza.

```css
/* KINEXIUM arculat. Helye: src/app.css, teljes cserével, a shadcn-svelte init után. */
@import "tailwindcss";
@import "tw-animate-css";
@import "shadcn-svelte/tailwind.css"; /* a shadcn-svelte komponensek data-* variánsai és keyframe-jei */

@custom-variant dark (&:is(.dark *));

@font-face { font-family: "Hanken Grotesk"; src: url("/fonts/HankenGrotesk-Variable.woff2") format("woff2"); font-weight: 100 900; font-style: normal; font-display: swap; }
@font-face { font-family: "Geist Mono"; src: url("/fonts/GeistMono-Variable.woff2") format("woff2"); font-weight: 100 900; font-style: normal; font-display: swap; }

/* Sötét az alap téma: .light kapcsolja át. */
:root, .dark, [data-theme="dark"] {
	color-scheme: dark;
	--ink-950: #040a0f;
	--ink-900: #071016;
	--ink-850: #0b171f;
	--ink-800: #10212a;
	--ink-700: #192d37;
	--ink-600: #253d46;
	--ink-500: #345058;
	--ink-400: #486469;
	--ink-300: #5f7b7d;
	--mist-400: #799290;
	--mist-300: #94a9a5;
	--mist-200: #b3c1bb;
	--mist-100: #d4dcd5;
	--mist-50: #f2f5f3;
	--signal-300: #77dbed;
	--signal-400: #52aec8;
	--signal-500: #36879f;
	--signal-600: #16637e;
	--signal-700: #0c4962;
	--background: var(--ink-900);
	--foreground: var(--mist-100);
	--card: var(--ink-850);
	--card-foreground: var(--mist-100);
	--popover: var(--ink-800);
	--popover-foreground: var(--mist-100);
	--primary: var(--signal-300);
	--primary-foreground: var(--ink-950);
	--secondary: var(--ink-800);
	--secondary-foreground: var(--mist-100);
	--muted: var(--ink-800);
	--muted-foreground: var(--mist-300);
	--accent: var(--ink-700);
	--accent-foreground: var(--mist-100);
	--destructive: #ff8a66;
	--destructive-foreground: var(--ink-950);
	--success: #8be8c0;
	--warning: #f4c95d;
	--info: var(--signal-300);
	--border: var(--ink-600);
	--input: var(--ink-300);
	--ring: var(--signal-300);
	--selection: rgba(119, 219, 237, 0.26);
	--glow: rgba(119, 219, 237, 0.14);
	--chart-1: #2ba0c2;
	--chart-2: #b08a2c;
	--chart-3: #1f9b7c;
	--chart-4: #7a84e0;
	--chart-5: #cf6a95;
	--sidebar: var(--ink-950);
	--sidebar-foreground: var(--mist-200);
	--sidebar-primary: var(--signal-300);
	--sidebar-primary-foreground: var(--ink-950);
	--sidebar-accent: var(--ink-800);
	--sidebar-accent-foreground: var(--mist-100);
	--sidebar-border: var(--ink-700);
	--sidebar-ring: var(--signal-300);
	--shadow-popover: 0 12px 32px rgba(0, 0, 0, 0.56);
}

.light, [data-theme="light"] {
	color-scheme: light;
	--background: var(--mist-50);
	--foreground: var(--ink-850);
	--card: #ffffff;
	--card-foreground: var(--ink-850);
	--popover: #ffffff;
	--popover-foreground: var(--ink-850);
	--primary: var(--signal-600);
	--primary-foreground: #ffffff;
	--secondary: #e4eae7;
	--secondary-foreground: var(--ink-850);
	--muted: #e4eae7;
	--muted-foreground: var(--ink-400);
	--accent: #d9e2df;
	--accent-foreground: var(--ink-850);
	--destructive: #b3401e;
	--destructive-foreground: #ffffff;
	--success: #146c4a;
	--warning: #865600;
	--info: var(--signal-600);
	--border: #c9d4d0;
	--input: #6b8482;
	--ring: var(--signal-600);
	--selection: rgba(22, 99, 126, 0.18);
	--glow: rgba(22, 99, 126, 0.08);
	--chart-1: #0e7fa3;
	--chart-2: #9a7420;
	--chart-3: #14876b;
	--chart-4: #5b63c7;
	--chart-5: #c04f7d;
	--sidebar: #e9eeeb;
	--sidebar-foreground: var(--ink-850);
	--sidebar-primary: var(--signal-600);
	--sidebar-primary-foreground: #ffffff;
	--sidebar-accent: #d9e2df;
	--sidebar-accent-foreground: var(--ink-850);
	--sidebar-border: #c9d4d0;
	--sidebar-ring: var(--signal-600);
	--shadow-popover: 0 12px 32px rgba(11, 23, 31, 0.14);
}

:root {
	--space-1: 4px;
	--space-2: 8px;
	--space-3: 12px;
	--space-4: 16px;
	--space-5: 20px;
	--space-6: 24px;
	--space-8: 32px;
	--space-10: 40px;
	--space-12: 48px;
	--space-16: 64px;
	--space-24: 96px;
	--radius: 6px;
	--chamfer-sm: 8px;
	--chamfer-md: 14px;
	--chamfer-lg: 24px;
	--layout-site: 1200px;
	--layout-prose: 640px;
	--layout-sidebar: 240px;
	--layout-topbar: 52px;
}

@theme inline {
	--font-sans: "Hanken Grotesk", ui-sans-serif, system-ui, sans-serif;
	--font-mono: "Geist Mono", ui-monospace, "SF Mono", Menlo, monospace;
	--radius-sm: 3px;
	--radius-md: 4px;
	--radius-lg: 6px;
	/* A shadcn-svelte kártya és dialógus rounded-xl, a jelvény rounded-4xl: itt minden a radius-lg-n marad, pill alak nincs. */
	--radius-xl: 6px;
	--radius-2xl: 6px;
	--radius-3xl: 6px;
	--radius-4xl: 6px;
	--radius-full: 9999px;
	/* Árnyék csak lebegő elemen: a shadcn gomb, mező és kártya shadow-xs / shadow-sm értéke üres. */
	--shadow-2xs: 0 0 #0000;
	--shadow-xs: 0 0 #0000;
	--shadow-sm: 0 0 #0000;
	--shadow-md: var(--shadow-popover);
	--shadow-lg: var(--shadow-popover);
	/* Hover-átmenet (6.1): a transition-* utility-k és a shadcn-komponensek alapértéke 120 ms ease-out. */
	--default-transition-duration: 120ms;
	--default-transition-timing-function: ease-out;
	--color-ink-950: var(--ink-950);
	--color-ink-900: var(--ink-900);
	--color-ink-850: var(--ink-850);
	--color-ink-800: var(--ink-800);
	--color-ink-700: var(--ink-700);
	--color-ink-600: var(--ink-600);
	--color-ink-500: var(--ink-500);
	--color-ink-400: var(--ink-400);
	--color-ink-300: var(--ink-300);
	--color-mist-400: var(--mist-400);
	--color-mist-300: var(--mist-300);
	--color-mist-200: var(--mist-200);
	--color-mist-100: var(--mist-100);
	--color-mist-50: var(--mist-50);
	--color-signal-300: var(--signal-300);
	--color-signal-400: var(--signal-400);
	--color-signal-500: var(--signal-500);
	--color-signal-600: var(--signal-600);
	--color-signal-700: var(--signal-700);
	--color-background: var(--background);
	--color-foreground: var(--foreground);
	--color-card: var(--card);
	--color-card-foreground: var(--card-foreground);
	--color-popover: var(--popover);
	--color-popover-foreground: var(--popover-foreground);
	--color-primary: var(--primary);
	--color-primary-foreground: var(--primary-foreground);
	--color-secondary: var(--secondary);
	--color-secondary-foreground: var(--secondary-foreground);
	--color-muted: var(--muted);
	--color-muted-foreground: var(--muted-foreground);
	--color-accent: var(--accent);
	--color-accent-foreground: var(--accent-foreground);
	--color-destructive: var(--destructive);
	--color-destructive-foreground: var(--destructive-foreground);
	--color-success: var(--success);
	--color-warning: var(--warning);
	--color-info: var(--info);
	--color-border: var(--border);
	--color-input: var(--input);
	--color-ring: var(--ring);
	--color-selection: var(--selection);
	--color-glow: var(--glow);
	--color-chart-1: var(--chart-1);
	--color-chart-2: var(--chart-2);
	--color-chart-3: var(--chart-3);
	--color-chart-4: var(--chart-4);
	--color-chart-5: var(--chart-5);
	--color-sidebar: var(--sidebar);
	--color-sidebar-foreground: var(--sidebar-foreground);
	--color-sidebar-primary: var(--sidebar-primary);
	--color-sidebar-primary-foreground: var(--sidebar-primary-foreground);
	--color-sidebar-accent: var(--sidebar-accent);
	--color-sidebar-accent-foreground: var(--sidebar-accent-foreground);
	--color-sidebar-border: var(--sidebar-border);
	--color-sidebar-ring: var(--sidebar-ring);
}

@layer base {
	* { @apply border-border; }
	body {
		@apply bg-background text-foreground font-sans;
		font-size: 15px;
		line-height: 24px;
		-webkit-font-smoothing: antialiased;
	}
	::selection { background: var(--selection); }
	:focus-visible { outline: 2px solid var(--ring); outline-offset: 2px; }
	@media (prefers-reduced-motion: reduce) {
		*, ::before, ::after { animation-duration: .01ms !important; transition-duration: .01ms !important; }
	}
}

@layer components {
	.display-xl { font-family: var(--font-sans); font-size: 72px; line-height: 76px; font-weight: 200; letter-spacing: -0.02em; }
	.display { font-family: var(--font-sans); font-size: 48px; line-height: 54px; font-weight: 250; letter-spacing: -0.015em; }
	.heading-1 { font-family: var(--font-sans); font-size: 32px; line-height: 40px; font-weight: 300; letter-spacing: -0.01em; }
	.heading-2 { font-family: var(--font-sans); font-size: 24px; line-height: 32px; font-weight: 400; letter-spacing: -0.005em; }
	.heading-3 { font-family: var(--font-sans); font-size: 18px; line-height: 26px; font-weight: 500; }
	.body-lg { font-family: var(--font-sans); font-size: 18px; line-height: 28px; font-weight: 400; }
	.body { font-family: var(--font-sans); font-size: 15px; line-height: 24px; font-weight: 400; }
	.body-sm { font-family: var(--font-sans); font-size: 13px; line-height: 20px; font-weight: 400; }
	.label { font-family: var(--font-mono); font-size: 11px; line-height: 16px; font-weight: 500; letter-spacing: 0.08em; }
	.code { font-family: var(--font-mono); font-size: 13px; line-height: 20px; font-weight: 400; }
	.data { font-family: var(--font-mono); font-size: 14px; line-height: 20px; font-weight: 400; }
	.data-lg { font-family: var(--font-mono); font-size: 32px; line-height: 36px; font-weight: 300; letter-spacing: -0.02em; }
	.kx-eyebrow { font-family: var(--font-mono); font-size: 11px; line-height: 16px; font-weight: 500; letter-spacing: .08em; text-transform: uppercase; color: var(--muted-foreground); }
	.kx-num { font-family: var(--font-mono); }
	
	/* Levágott sarkú panel. A keret a külső réteg, a kitöltés a ::before. A ::before vágása 1.414px-szel (√2) nagyobb, így az átlón a keret kb. 2,4 px. */
	.kx-panel { --kx-c: var(--chamfer-md); --kx-fill: var(--card); --kx-edge: var(--border); position: relative; isolation: isolate; background: var(--kx-edge); clip-path: polygon(0 0,calc(100% - var(--kx-c)) 0,100% var(--kx-c),100% 100%,var(--kx-c) 100%,0 calc(100% - var(--kx-c))); }
	.kx-panel::before { --kx-c: calc(var(--kx-cut, var(--chamfer-md)) + 1.414px); content: ""; position: absolute; inset: 1px; z-index: -1; background: var(--kx-fill); clip-path: polygon(0 0,calc(100% - var(--kx-c)) 0,100% var(--kx-c),100% 100%,var(--kx-c) 100%,0 calc(100% - var(--kx-c))); }
	.kx-panel-sm { --kx-c: var(--chamfer-sm); --kx-cut: var(--chamfer-sm); }
	.kx-panel-lg { --kx-c: var(--chamfer-lg); --kx-cut: var(--chamfer-lg); }
	
	/* Mérőléc: 8 px-es finom, 40 px-es hangsúlyos osztás. */
	.kx-ticks { display: block; height: 10px; background-image: linear-gradient(to right, var(--input) 1px, transparent 1px), linear-gradient(to right, var(--border) 1px, transparent 1px); background-size: 40px 10px, 8px 5px; background-repeat: repeat-x; background-position: 0 0; }
	
	.kx-dot { display: inline-block; width: 8px; height: 8px; flex: none; border-radius: var(--radius-full); background: currentColor; }
	.kx-dot-ring { width: 7px; height: 7px; background: transparent; box-shadow: inset 0 0 0 1.5px currentColor; }
	
	/* Beágyazott helix (9.3): a színeket a téma adja. */
	.kx-helix { display: block; width: 100%; height: auto; }
	.kx-helix .b { stroke: var(--border); fill: none; }
	.kx-helix .f { stroke: var(--input); fill: none; }
	.kx-helix .h { stroke: var(--primary); }
	.kx-helix .nb { fill: var(--border); }
	.kx-helix .nf { fill: var(--input); }
	.kx-helix .nh { fill: var(--primary); }
}

/* shadcn-svelte fókusz: a komponensek outline-none és 3 px-es, félátlátszó ring párosát a rendszer 2 px tömör, 2 px-re eltolt gyűrűje váltja. Rétegen kívül áll, ezért felülírja az utility-ket. */
[data-slot]:focus-visible { outline: 2px solid var(--ring); outline-offset: 2px; --tw-ring-color: transparent; }
```

Mit ír felül ez a fájl a shadcn-svelte alapjához képest:

- A kártya és a dialógus `rounded-xl` sugara `radius-lg` (6 px) lesz.
- A gomb és a mező `shadow-xs` árnyéka eltűnik.
- A 3 px-es félátlátszó fókuszgyűrű helyett a 2 px tömör, 2 px-re eltolt gyűrű jelenik meg.
- Az átmenetek alapértéke a Tailwind 150 ms-os `cubic-bezier(0.4, 0, 0.2, 1)` görbéje helyett 120 ms `ease-out` (6.1); ahol a komponens saját `duration-*` osztályt ad (például a dialógus 200 ms-os nyitása), az marad érvényben.

Ha egy token értéke változik, a fenti fájl megfelelő sorát kell módosítani, a komponensek érintetlenek maradnak.

### 13.3 `components.json`

A SvelteKit-projekt gyökerébe kerül (`web/components.json`). A `style`, `iconLibrary`, `menuColor` és `menuAccent` mező a shadcn-svelte `init` kérdéseire adott válasz.

```json
{
	"$schema": "https://www.shadcn-svelte.com/schema.json",
	"tailwind": {
		"css": "src/app.css",
		"baseColor": "mist"
	},
	"aliases": {
		"components": "$lib/components",
		"utils": "$lib/utils",
		"ui": "$lib/components/ui",
		"hooks": "$lib/hooks",
		"lib": "$lib"
	},
	"typescript": true,
	"registry": "https://www.shadcn-svelte.com/registry",
	"style": "nova",
	"iconLibrary": "lucide",
	"menuColor": "default",
	"menuAccent": "subtle"
}
```

A felhasznált shadcn-svelte komponensek: `button`, `input`, `label`, `select`, `checkbox`, `switch`, `table`, `tabs`, `dialog`, `alert`, `card`, `badge`, `sidebar`, valamint `alert-dialog` (10.8) és `chart` (10.12, LayerChart-függőséggel). A sűrű adattáblához igény szerint TanStack Table (10.7).

A teljes hozzáadó parancs:

```bash
npx shadcn-svelte@latest add button input label select checkbox switch table tabs dialog alert card badge sidebar alert-dialog chart
```

Az `alert-dialog` és a `chart` nélkül a 10.8 `AlertDialog`-ja nem létezik, és a 10.12 `$lib/components/ui/chart/index.js` importja nem fordul le. A `chart` hozzáadása a LayerChart függőséget is telepíti.

---

## 14. Eltérések és döntések

Szándékos döntések és a kiinduló projektváz eltérései.

| # | Téma | Eltérés | Döntés |
| --- | --- | --- | --- |
| 1 | Kiinduló projektváz stíluslapja | A Go + SvelteKit projektváz `src/app.css`-e shadcn-stílusú, oklch alapú, világos alapú, indigó-kékes paletta (`--radius: 0.625rem`), fontok, `chart-*`, `success`, `warning`, `info` és `sidebar-*` tokenek nélkül. | Nem KINEXIUM-arculat. Teljes cserével a 13.2 tartalma kerül a helyére. Értékei nem vehetők át tokenként. |
| 2 | Témakezelés | A váz a rendszerbeállítást követi (`prefers-color-scheme`), világos alappal, téma-kapcsoló nélkül; a `dark` variáns definiált, de `.dark` osztályt senki nem állít. | Sötét az alap téma, `mode-watcher` kezeli, `.light` osztály a világos témához, `app.html` `class="dark"` + indító szkript + `theme-color #071016` (2.6). |
| 3 | Favicon és cím | A váz faviconja az alapértelmezett Svelte logó. | `static/favicon.svg` = KINEXIUM app ikon (7.7). |
| 4 | Váz komponensei | A váz kézzel írt `button`, `card`, `input` komponense 40 px magas (`h-10`), a kártya `shadow-sm`, a fókusz `outline-none` + `ring-2` eltolás nélkül, és nincs `data-slot` attribútumuk. | A 10. fejezet méretei érvényesek: gomb és mező 36 px, `radius-md`, árnyék nélkül, 2 px tömör, 2 px-re eltolt fókuszgyűrű. A váz komponenseit shadcn-svelte komponensekre kell cserélni, vagy ezekre az értékekre igazítani. |
| 5 | Fontok gyorsítótárazása | A kiszolgáló a fontokat a `/fonts/` útvonalon adja ki. A Go-kiszolgáló jelenleg csak az `_app/immutable/` fájloknak ad `Cache-Control: public, max-age=31536000, immutable` fejlécet. | A fontokra is egyéves, `immutable` gyorsítótár kell: a feltétel bővül a `strings.HasPrefix(name, "fonts/")` vagy-ággal. A részletek a TECHNOLOGY.md-ben. |
| 6 | Borító betűsúlya | A borító neve 200-as súlyú, a `display` osztály 250-es. | A borító neve a saját `.name` stílusát kapja (88 px, súly 200, line-height .95), nem a `display` osztályt. A `display` osztály 250 marad. |
| 7 | Helix lépcsőszám és periódus | Három változat: önálló minta 1200×480, felületbe ágyazott 1200×300, weboldal-sáv 1280×300. | A periódus mindháromban P = 1200 / 2,25 ≈ 533,33 px (9.2). Az 1200 széles változatok 34, az 1280 széles 36 lépcsősek, mindegyikben 5 kiemelt lépcső (9.2, 9.3). A három eltérés szándékos. |
| 8 | Átmenet görbéje | Gomb, menüpont és sor hover-átmenete. A Tailwind alapértéke 150 ms `cubic-bezier(0.4, 0, 0.2, 1)`. | `background-color`, `border-color` 120 ms `ease-out` (6.1), a panel- és dialógusnyitással egységesen; az `ease` görbe nem használható. A 13.2 `@theme inline` blokkja ezt teszi alapértékké (`--default-transition-duration`, `--default-transition-timing-function`). |
| 9 | StatusBadge két alakja | A Svelte komponens Tailwind-osztályokkal (`h-[22px]`, `rounded-sm`, `border`, `text-xs`, `size-3`) dolgozik, a referenciajelölés `kx-badge` osztállyal. | A két alak értékben azonos (22 px, 3 px sugár, 12 px betű és ikon). Felületen a Svelte komponens használandó. |
| 10 | Referenciajelölések | A `kx-btn`, `kx-card`, `kx-alert` stb. osztályok fix px-értékeket használnak, és csak a 13.2 stíluslap mellett működnek, mert a `--radius-*`, `--radius-full` és `--font-*` változókat az `@theme inline` blokk adja. | Ezek az osztályok mintajelölések, nem részei a stíluslapnak. Felületen shadcn-svelte komponens adja ugyanezt a megjelenést. |
| 11 | `shadow-popover` utility | A `--shadow-popover` csak `:root` változó, nincs Tailwind-témában. | Tailwindben `shadow-md` vagy `shadow-lg` adja, CSS-ben `box-shadow: var(--shadow-popover)`. |
| 12 | `baseColor: "mist"` | A `components.json` `mist` alapszíne a shadcn-svelte saját kékesszürke alapszíne, nem azonos a `mist-*` skálával. | Csak a kiinduló változókat adja, amelyeket a 13.2 stíluslap teljesen felülír. |
| 13 | Skálák a világos témában | A világos téma nem definiálja újra az `ink`, `mist`, `signal` skálát. | Szándékos: a skálák témától függetlenek. A skálát használó márkafelületek (kapcsolatblokk, borító) világos témában is ugyanúgy néznek ki. |
| 14 | Állapotszínek kitöltése | A `success`, `warning`, `info` tokennek nincs `-foreground` párja. | Ezek csak ikon- és szövegszínek, kitöltésként nem használhatók (2.4). |
| 15 | Primary gomb kivétele | „Képernyőnként egy primary gomb” | Egy kivétel: a weboldal `signal-700` kapcsolatblokkjának gombja, amely a hero fő műveletét (kapcsolatfelvétel) ismétli meg, felirata az e-mail-cím (12.1). Ott a gomb `signal-300` hátterű, `ink-950` feliratú. |
| 16 | Jelölőnégyzet pipája | Az ikonok vonala 1.5, a pipa 2.5. | Szándékos kivétel, a 12 px-es pipa olvashatósága miatt. |
| 17 | Skálatoken komponensben | Szabály: komponensben szemantikus token. A borító, a logó-előnézet és a kapcsolatblokk nyers skálatokent használ. | Szándékos: ezek márkafelületek. Másutt csak szemantikus token. |
| 18 | Megszólítás | Magázás és tegezés egyaránt előírt. | Szándékos kettősség: ügyfél és weboldal magáz vagy nem szólít meg, a felületi szöveg tegező felszólító mód (1.3). |
| 19 | Ezres tagoló | Az ezres tagolás és az érték–mértékegység elválasztása szóközzel történik. | Nem törő szóköz (U+00A0) mindkét helyen, hogy a szám ne törjön el (1.8). |
| 20 | `$lib` és `#lib` | A `components.json` aliasai `$lib/...` útvonalak, a projektváz a `#lib` importot használja. | Az alias beállítása a TECHNOLOGY.md hatásköre. A márkakomponensek egymást relatív úton importálják (`./Panel.svelte`), az aliastól függetlenek. |
| 21 | Navigációs ikon mérete | Az általános navigációs ikonméret 20 px, az alkalmazás-oldalsáv menüpontjai 16 px-esek. | Az oldalsáv menüpontja 16 px (`size-4`), a 32 px-es sormagassághoz; a 20 px az önálló navigációs ikonra érvényes (8.2). |
| 22 | Levágott sarok iránya | A szabály a jobb felső és a bal alsó sarkot vágja; a borító tömbjei más sarkokat is vágnak, az app ikon vágása 22/128 egység. | Szándékos: a borító tömbjeinek vágását a 10.6, az app ikonét a 7.7 rögzíti (5.2). |
| 23 | Alkalmazás-tartalom margója | Két szomszédos térközfokozat: `space-6` (24 px) és `space-8` (32 px). | Az alkalmazás tartalommargója `space-6` (24 px), ahogy a 4.1, a 4.2 és a 12.2 rögzíti; a `space-8` (32 px) a tágas tartalommargó és a weboldal oldalsó belső margója. |
| 24 | `body-lg` mintaszövege | A mintaszöveg a cég hangján szól. | „Az érzékelőtől a döntésig tervezek.”, egyes szám első személyben (1.2). |
| 25 | Töréspontok | Az arculat töréspontot nem rögzít. | A Tailwind alap `md` (768 px) és `lg` (1024 px) töréspontja; a rácsok és az oldalsáv viselkedését a 4.2 rögzíti. |
| 26 | Borító formátumai | A borító három méretben készül. | 960×288 px (alap), 1200×630 px (social előnézet) és álló A4 (dokumentumcímlap), mindhárom értékei a 10.6 szerint. |
| 27 | Kapcsolatblokk gombjának hovere | A `default` gomb hovere a `primary`-ból számol, ami világos témában a `signal-300` gombot a `signal-600` felé sötétítené. | `color-mix(in oklab, var(--signal-300) 88%, var(--mist-100))` mindkét témában (12.1). |
| 28 | Metric egységfelirata | A `data-lg` -0.02em betűköze az egységre is öröklődik. | Az egység `tracking-normal` osztályt kap (betűköz 0), a referenciajelölés `small` elemével egyezően (10.14). |
| 29 | Linkstílus | A Tailwind alapstílusa a linkről leveszi a színt és az aláhúzást; globális `a` szabály a linkként renderelt `Button` és oldalsáv-menüpont színét is átírná. | Globális `a` szabály nincs. Szöveges link: `text-primary underline underline-offset-3` (10.0, 12.2). |
| 30 | Helix a stíluslapban | A beágyazott helix színei témától függnek. | A `kx-helix` és alosztályai (`.b`, `.f`, `.h`, `.nb`, `.nf`, `.nh`) a `src/app.css` `@layer components` rétegében állnak (9.3, 13.2). |
| 31 | AppPage diagramkártyája | A diagram három adatsoros. | A kártyafejlécben táblázatnézet gomb (`ghost` `sm`) áll, mert két vagy több sornál táblázatnézet kötelező (11., 12.2). |
| 32 | Kritikus Alert shadcn-variánsa | A shadcn `destructive` variáns a címet és a szöveget is színezi. | `variant="default"`, a szín csak az ikonon (`text-destructive`), 10.1. |
| 33 | Dialógus háttérfátyla | A shadcn-svelte fátyol színe a komponensfájlban rögzített. | A `dialog-overlay.svelte` és az `alert-dialog-overlay.svelte` háttérosztálya `bg-[color-mix(in_oklab,var(--ink-950)_62%,transparent)]` (10.8). |
