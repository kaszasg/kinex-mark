# Kinex Mark: fejlesztési terv

Ez a dokumentum a Kinex Mark (KINEXIUM Bookmark Manager) teljes, a fejlesztés megkezdése előtt rögzített fejlesztési terve. Önálló: a fejlesztő és ellenőrző ügynökök ebből, a `ai/BASE-SYSTEM-PROMPT.md`, `ai/TECHNOLOGY.md` és `ai/GRAPHICS.md` dokumentumokból dolgoznak, más tervezési anyagot nem kapnak.

Elsőbbségi sorrend ütközés esetén:

1. `ai/BASE-SYSTEM-PROMPT.md` (követelmények),
2. ez a terv (`ai/BASE-SYSTEM-PLAN.md`), ezen belül az 5. fejezet API-szerződése a frontendre nézve kötelező,
3. `ai/TECHNOLOGY.md` (architektúra, váz),
4. `ai/GRAPHICS.md` (design system; minden vizuális döntés forrása).

Ahol a terv a TECHNOLOGY.md vagy a GRAPHICS.md viselkedését változatlanul átveszi, szakaszszámmal hivatkozik rá (például „TECHNOLOGY.md 4.7”). Ahol eltér, az eltérés a 2. fejezetben szerepel.

Írásmód: a dokumentáció magyar, az azonosítók, útvonalak, kódrészletek, API-nevek angolok. Dátum: 2026-11-21.

Frissítés (2026-11-21, a C fázis lezárása után, a lead): a felhasználó döntése alapján a Go-minimum a függőségeket követi, a `go.mod` most `go 1.26.0` (2.2 T22); a terv szövege a megvalósítás indokolt eltéréseihez igazodott (többek között 3.3.8 megszakításbiztos SQLite-kapcsolat, 3.3.11 indítási sorrend, 6.7, 6.10, 8.1.1, 8.1.2, 8.8.4, 9.4.1, 10.1, 10.3, 10.10, 10.11, 11.1 felderítés). A szó szerinti listák (`.gitignore`, `.dockerignore`, `build.ps1`, `ci.yml`, `docker.yml`, `prompts.json`) a repository mostani fájljaival egyeznek.

---

## 1. Cél és hatókör

### 1.1 A termék

A Kinex Mark webalkalmazás linkeket tárol és a gyűjteményt kezeli: gyors rögzítés (ráhúzás, beillesztés, kézi felvétel), letöltött cím és ikon, rövid leírás, színes címkék (csak címke alapú csoportosítás), lista és csempe nézet, szöveges és címkeszűrés, használat szerinti rangsor, kitűzés, automatikus archiválás, másolás (URL és Markdown), részletes adatlap, megosztás, felhasználókezelés három szerepkörrel, adminisztrációs felület, kétlépcsős azonosítás, GDPR-megfelelés, háromnyelvű (magyar, angol, német) felület.

### 1.2 Szállítandó termékek

| # | Termék | Hely |
|---|---|---|
| 1 | Ez a terv | `ai/BASE-SYSTEM-PLAN.md` |
| 2 | Egyetlen statikus Go-bináris beágyazott SvelteKit-felülettel | `cmd/kinex-mark`, `internal/**`, `web/**`; kimenet `dist/kinex-mark[.exe]` |
| 3 | Docker image (multi-arch, distroless nonroot) | `Dockerfile`, `compose.yaml`; `ghcr.io/kaszasg/kinex-mark` |
| 4 | Tesztek: Go unit és HTTP, Vitest unit és komponens, Playwright E2E a valódi binárison, feltáró GUI-teszt a Playwright MCP-vel | `internal/**/*_test.go`, `web/src/**/*.spec.ts`, `web/e2e/**` |
| 5 | Nyitóoldali hero-grafika a `gemini-3.1-flash-image` modellel | `graphics/prompts.json`, `graphics/hero.png`, `web/static/images/hero.webp` |
| 6 | GitHub Actions CI/CD Docker-image előállítással | `.github/workflows/ci.yml`, `docker.yml`, `.github/dependabot.yml` |
| 7 | `SUMMARY.md` (a fejlesztés állapota, AI-folytatáshoz) | repo gyökér |
| 8 | Magyar kézikönyv képernyőképekkel | `manual/MANUAL.md`, `manual/images/*.png` |
| 9 | Rövid magyar futtatási összefoglaló a GitHub-alap helyett | `README.md` |

### 1.3 Azonosítók

| Elem | Érték |
|---|---|
| Projektnév | Kinex Mark |
| Go-modul | `github.com/kaszasg/kinex-mark` (`go.mod`: `go 1.26.0`, nincs `toolchain` sor; a direktíva a függőségek igényét követi, 2.2 T22) |
| Bináris | `kinex-mark` (`kinex-mark.exe` Windowson), belépési pont `./cmd/kinex-mark` |
| Docker image | `ghcr.io/kaszasg/kinex-mark` (helyben `kinex-mark:<verzió>`) |
| Frontend csomag | `web/package.json` `name`: `kinex-mark-web` |
| Süti | `__Host-kxm_session` (biztonságos módban) vagy `kxm_session` |
| Kötelező kérésfejléc | `X-Kinex-Request: 1` |
| Helyi tárolókulcsok | `kxm-locale` (nyelv), `mode-watcher-mode` (téma) |

### 1.4 Hatókörön kívül

- Mappák vagy bármilyen hierarchikus csoportosítás (a BASE-SYSTEM-PROMPT kizárja; importnál a mappanév címkévé lapul).
- E-mail-cím, telefonszám, e-mail-küldés, jelszó-visszaállítás e-mailben (adattakarékosság; a jelszót admin állítja vissza).
- Böngészőbővítmény, nyilvános (bejelentkezés nélküli) linkmegosztás, analitika, harmadik féltől betöltött tartalom.
- Git commit: a fejlesztés során senki nem készít commitot (a felhasználó nem kérte).

### 1.5 Toolchain (a fejlesztőgépen igazolt)

| Eszköz | Verzió | Megjegyzés |
|---|---|---|
| Windows 11, PowerShell 5.1 / 7 | | elsődleges shell; Git Bash elérhető |
| Go | 1.27.4 (`GOTOOLCHAIN=auto`) | `go.mod` minimum `go 1.26.0` (2.2 T22); a minimumot a CI `1.26.x` ága és a `golang:1.26` Docker-image `GOTOOLCHAIN=local`-lal igazolja |
| Node | 26.13.1 | CI és Docker is Node 26 |
| npm | 12.2.2 | |
| Python | 3.14 | `google-genai` 2.16.0 és Pillow a felhasználói site-ban (ezért `python`, nem `python -I`) |
| Docker | 29.9 | Linux-konténer mód |
| `make`, `pnpm` | nincs | helyben a `build.ps1` a buildút; a Makefile Linuxra, macOS-re és CI-re készül |

---

## 2. Döntések és eltérések

### 2.1 A lead rögzített döntései (D1–D14)

| # | Döntés |
|---|---|
| D1 | Azonosítók az 1.3 szerint; a projekt közvetlenül a repo gyökerében él (`cmd/`, `internal/`, `web/`), nincs `webapp/` almappa |
| D2 | A BASE-SYSTEM-PROMPT elsőbbsége: (a) a forráskód kizárólag angol (azonosító, komment, napló, csomagolt Go-hiba, API-hibaüzenet, konfigurációs súgó); (b) minden felhasználónak látható szöveg i18n-en megy át (hu, en, de); (c) a váz todo-doménje nem kerül át, a váz architektúrája igen (egy statikus bináris, ServeMux metódusos minták, `/api/` JSON catch-all, SPA-kezelő a gyorsítótár-szabályokkal és a `fonts/` bővítéssel, `go:embed all:build`, migrációs mechanizmus, kapcsolók + környezeti változók, `-healthcheck`, `-version`, szabályos leállítás, Makefile + `build.ps1`, többlépcsős Dockerfile distroless nonroot futtatóval, `compose.yaml`) |
| D3 | Legfrissebb komponensek (2026-11-21): `@sveltejs/kit` 3.4.0, `svelte` 5.59.1, `vite` 8.3.8, `@sveltejs/vite-plugin-svelte` 7.3.1, `@sveltejs/adapter-static` 4.0.1, `tailwindcss` 4.3.5, shadcn-svelte CLI 1.8.1, `sv` 1.7.0, `bits-ui` 2.21.0, `mode-watcher` 1.1.0, `@lucide/svelte` 1.77.0, `vitest` 5.1.1, `@playwright/test` 1.65.1. TypeScript: lásd 2.4 |
| D4 | Go-függőségek: standard könyvtár + `modernc.org/sqlite` + `golang.org/x/net` (csak `html` és `html/charset`; HTML-t csak a `html.NewTokenizer` olvas, `html.Parse*` és `html.Render` hívás tilos, mert a korábbi x/net-sérülékenységek ezeket érintették). A verziók a legfrissebbek, a `go` direktíva ezekhez igazodik (2.2 T22; jelenleg `modernc.org/sqlite` v1.62.0, `golang.org/x/net` v0.63.0, `go 1.26.0`). Jelszó: PBKDF2-HMAC-SHA256 600 000 iterációval (`crypto/pbkdf2`), önleíró hash. TOTP: RFC 6238 standard könyvtárral. QR-kód kliensoldalon az otpauth URI-ból |
| D5 | Szerepkörök: `admin`, `developer`, `customer`. Mindenki saját linket és címkét kezel. Admin: felhasználókezelés, rendszerbeállítások, audit napló + minden, amit a fejlesztő. Fejlesztő: megoszthat mindenkivel, szerepkörrel, felhasználóval. Ügyfél: saját privát linkek + a vele megosztottak olvasása; nem oszthat meg, más felhasználót nem listázhat. Láthatóság SQL-ben érvényesítve, IDOR-teszttel |
| D6 | Link: URL (csak http/https), letöltött cím, letöltött favicon (méretkorlátos blob a DB-ben, hitelesített végponton kiszolgálva), opcionális rövid leírás, színes címkék. Megjelenítési címke: leírás, különben cím, különben URL; hosszú URL helyett csak a domain. Személyes állapot (felhasználó, link) párra, megosztott linkre is: kitűzés, használatszám, utolsó használat, archiválás. Rendezés: kitűzött elöl, mindkét csoportban használat szerint. Automatikus archiválás (alap 90 nap, 0 = soha), kitűzött soha. Gyors rögzítés, másolás, lista és csempe, szűrők, megjelenítési kapcsolók, részletes adatlap |
| D7 | Szerveroldali metaadat-letöltő: szigorú időkorlát és méretkorlát, átirányítás-korlát, SSRF-védelem dial-szinten, privát hálózat engedélyezése rendszerbeállítással vagy környezeti változóval, kódolásfüggetlen, ikonválasztás `/favicon.ico` tartalékkal; a link mentését soha nem blokkolja |
| D8 | Hitelesítés: szerveroldali munkamenet SQLite-ban (csak a token hash-e), HttpOnly + SameSite=Strict + Secure süti, tétlenségi és abszolút lejárat, munkamenetlista és visszavonás, CSRF-védelem, bejelentkezés-korlátozás, opcionális TOTP 2FA helyreállító kódokkal, kötelező jelszócsere, admin jelszó- és 2FA-visszaállítás, első admin környezeti változóból vagy generált jelszóval |
| D9 | GDPR: adattakarékosság, lokalizált tájékoztató (adatkezelő adatai a rendszerbeállításokban), elfogadás rögzítése, önkiszolgáló adatexport (JSON) és fióktörlés, admin felhasználótörlés, audit napló megőrzési idővel, csak szükséges süti, nincs harmadik féltől jövő kérés, biztonsági fejlécek |
| D10 | Saját, típusos, névteres i18n-réteg; a magyar a típusforrás; az angol és a német kulcskészlete azonos (teszt); futásidejű váltás; `localStorage` + profil; `html lang`; Intl-formázás a GRAPHICS.md 1.8 szerint; nyelvválasztó a bejelentkező oldalon, a felhasználói és az admin felületen; a backend csak stabil hibakódot ad |
| D11 | Felület a GRAPHICS.md szerint; AppShell a GRAPHICS.md 12.2 szerint (240 px oldalsáv, 52 px felső sáv, offcanvas mobilon); útvonalak: `/login`, `/privacy`, `/`, `/tags`, `/settings`, `/admin/users`, `/admin/settings`, `/admin/audit`; sötét alap, világos teljes értékű pár |
| D12 | Tesztek: Go legalább 85% utasításlefedettség az `internal` csomagokon; Vitest + Testing Library a kulcsparitás-teszttel; Playwright E2E a valódi binárison ideiglenes adatkönyvtárral; feltáró GUI-teszt a Playwright MCP-vel, amely a kézikönyv képeit is adja |
| D13 | Szállítandók az 1.2 szerint; a hero-grafika lapos, vékonyvonalas rendszerábra sötét petróleum alapon egyetlen cián kiemeléssel, a gyors rögzítést, a címkéket, a szűrést, a rangsort és az automatikus elévülést mutatja, WebP-ben a `web/static/images` alá |
| D14 | Fejlesztő ügynök: Opus 5.5; kódfelderítő ügynök: Sonnet 5.5; felülettervezés: `frontend-design` skill; GUI-teszt: Playwright plugin; nincs git commit; párhuzamos fejlesztők diszjunkt fájlokon dolgoznak |

### 2.2 Eltérések a TECHNOLOGY.md-től

| # | Téma | TECHNOLOGY.md | Kinex Mark | Indok |
|---|---|---|---|---|
| T1 | Gyökér és modul | `webapp/`, `example.com/webapp` | repo gyökér, `github.com/kaszasg/kinex-mark` | D1 |
| T2 | Belépési pont | `cmd/server` | `cmd/kinex-mark`; a `run` logikája az `internal/app`-ba költözik (tesztelhetőség) | D1 |
| T3 | Csomagok | `api`, `store` | `api`, `app`, `auth`, `bookmarks`, `clock`, `config`, `fetch`, `jobs`, `model`, `store`, `testutil` | domén és párhuzamos fejlesztés |
| T4 | Mintadomén | todo | nincs; linkek, címkék, felhasználók | D2c |
| T5 | Szövegek | magyar komment, napló, hiba (`a szerver elindult`, `belső hiba`) | angol (`server started`, `internal error`) | D2a |
| T6 | Hibatörzs | `{"error":"<magyar üzenet>"}` | `{"error":"<angol>","code":"<stabil kód>","fields":{…},"details":{…}}`; a felület a `code`-ot fordítja | D2b, D10 |
| T7 | Router | middleware nélkül | middleware-lánc: recover, biztonsági fejlécek, hozzáférési napló (Debug), CSRF, hitelesítés útvonalanként | D8, D9 |
| T8 | Go-függőség | csak `modernc.org/sqlite` | + `golang.org/x/net` (`html`, `html/charset`) | D4 |
| T9 | API-kliens | `web/src/lib/api.ts`, egy fájl, nyers hibaüzenet | `web/src/lib/api/` mappa, `ApiError` kóddal, `X-Kinex-Request` fejléc | D10, D8 |
| T10 | `robots.txt` | `Disallow:` | `Disallow: /` | belső alkalmazás |
| T11 | ModeWatcher | `<ModeWatcher defaultMode="dark" lightClassNames={['light']} />` | ugyanez `disableHeadScriptInjection` proppal | a CSP tiltaná a futás közben beszúrt inline szkriptet (6.9) |
| T12 | Fontok | 41 216 B és 38 048 B a dokumentált hash-sel | a dokumentált hash-ű fájl egyik nyilvános forrásból sem állítható elő; a fonttools-szal épített változó WOFF2-k kerülnek be: `HankenGrotesk-Variable.woff2` 34 532 B, SHA-256 `bcd9aa547e0e3e3299c8dc0113407a3d3e47c23b2d89d25122d1d36a9e16f74c`; `GeistMono-Variable.woff2` 23 200 B, SHA-256 `8165ac18be02933145d5a331d11630c6fb6dcefef4284cbae5ada1771c546786` | az elfogadási feltételek (változó WOFF2, wght 100–900, ő/ű/Ő/Ű glifa) teljesülnek; reprodukálás: 10.6 |
| T13 | Node a Dockerben | `node:22-alpine` | `node:26-alpine` | egyezik a helyi és a CI Node-dal |
| T14 | Futtató image | `static-debian12:nonroot` | `static-debian13:nonroot` (tartalék: `static-debian12:nonroot`, ha a címke nem érhető el) | Debian 13 a stabil |
| T15 | Compose port | `8080:8080` | `127.0.0.1:8080:8080`, csak olvasható konténer, `cap_drop: ALL` | biztonság; hálózati elérés TLS reverse proxy mögött |
| T16 | `cn()` | saját `clsx` + `tailwind-merge` | a shadcn-svelte 1.8.1 CLI `export { cn } from "cn"` alakja (npm `cn` 0.5.2) | a CLI felülírja |
| T17 | `tailwind-variants` | `^4.0.1` | `^3.3.1` (a shadcn CLI pinje) | a generált komponensek a 3-as API-t használják |
| T18 | Importalias | `#lib` | a saját kód `$lib`-et használ (a shadcn-kóddal egységesen); a `#lib` a `package.json`-ban megmarad | a generált fájlok 173 helyen `$lib` importot használnak |
| T19 | Ismert figyelmeztetések (6.11) | `config_option_deprecated_alias` | + `tsconfig_invalid` („"paths" was overwritten”), kozmetikai, a `$lib` típusellenőrzés működik | SvelteKit 3.4 |
| T20 | Make/PowerShell célok | `build`, `web`, `run`, `dev`, `cross`, `check`, `image`, `binary-image`, `clean` | + `test`, `test-go`, `test-web`, `test-e2e`, `e2e-install`, `verify`, `graphics`, `tidy` | tesztkapuk |
| T21 | `web/build` placeholder | nincs megoldva | nincs `.gitkeep` (az adapter törölné); minden buildút előbb a felületet építi | `git describe --dirty` tisztasága |
| T23 | SQLite DSN (5.1) | `journal_mode(WAL)`, `synchronous(NORMAL)`, `busy_timeout(5000)`, `foreign_keys(1)` | + `_pragma=secure_delete(1)`; a `DeleteUser` sikeres commitja után best-effort `PRAGMA wal_checkpoint(TRUNCATE)` (6.11) | D fázis, R3: a törölt fiók adatai (felhasználónév, jelszóhash, URL-ek) különben olvashatók maradtak az `app.db`-ben és az `app.db-wal`-ban, így a mentésekben is (GDPR 17. cikk, 6.11, 6.12); `TestDeleteUserLeavesNoRemnants` |
| T22 | Go-minimum (2.1 tábla; 9. fejezet, 3. sor) | `go 1.24`, nincs `toolchain` sor; „a minimum 1.24 marad” | `go 1.26.0`, nincs `toolchain` sor; a `go` direktíva a függőségek aktuális verzióinak igényét követi, nem rögzített szám; CI-mátrix `['1.26.x', '1.27.x']`; a Docker-build `golang:1.27-alpine` változatlan | a felhasználó 2026-11-21-i döntése: a TECHNOLOGY.md `go 1.24` minimuma feloldva, hogy a függőségek aktuális verziói használhatók legyenek. A régi pinek (`golang.org/x/net` v0.43.0, `x/sys` v0.35.0, `x/text` v0.28.0) govulncheck szerint 0 elérhető, de 10 csomag- és 10 modulszintű sérülékenységet hordoztak, és a javított verziók mind 1.24 fölötti direktívát kérnek. A DEP-frissítés után a `modernc.org/sqlite` v1.62.0, a `modernc.org/libc` v1.82.9, a `golang.org/x/net` v0.63.0, a `golang.org/x/sys` v0.50.0 és a `golang.org/x/text` v0.44.0 mind `go 1.26.0`-t deklarál, ez a legkisebb megengedett érték; a govulncheck egyik szinten sem talál semmit (3.3.8, 10.10, 14) |

### 2.3 Eltérések a GRAPHICS.md-től

| # | Téma | GRAPHICS.md | Kinex Mark | Indok |
|---|---|---|---|---|
| G1 | `src/app.css` kommentjei | a 13.2 blokk 10 magyar kommentet tartalmaz | a `scripts/sync-app-css.mjs` a blokkot kimásolja, minden CSS-kommentet eltávolít, és egyetlen angol fejléckommentet tesz elé; az `app-css.spec.ts` a kommentek nélküli, normalizált CSS-t hasonlítja a GRAPHICS.md blokkjához | D2a: a forráskód angol |
| G2 | `chart`, `avatar` komponens | a 13.3 hozzáadó parancsa a `chart`-ot tartalmazza | nem kerül be; nincs idősor és profilkép, a LayerChart-függőség elmarad | nincs rá szükség |
| G3 | `StatusBadge` alapszavai | magyar szöveg a forrásban | a szavak a `status` i18n-névtérből jönnek | D2b |
| G4 | Márkakomponensek kommentjei | magyar | angol fordítás | D2a |
| G5 | AppPage metrikasora (12.2 7. régió) | 4 `Metric` az oldal tetején | a Linkek oldalon nincs; a `Metric` a részletek lapon jelenik meg | a lista az elsődleges adat |
| G6 | Sheet fátyla | a 10.8 csak a dialógus és az alert-dialógus fátylát írja elő | a `sheet-overlay.svelte` is ugyanazt kapja | egységes lebegő rétegek |
| G7 | QR-tároló | komponensben szemantikus token | `bg-mist-50 text-ink-950` mindkét témában (márkafelületi kivétel) | a beolvasók sötét modult várnak világos alapon |
| G8 | Bejelentkező oldal bal oszlopa | skálatoken csak márkafelületen | a bal oszlop márkafelület (`ink-950`, `mist-100`, `signal-300`), témától független | a sötét alapú hero-képhez illeszkedik |
| G9 | Német megszólítás | csak a magyar tegezés rögzített | a német felület tegező („du”), a gombok főnévi alakúak („Link hinzufügen”) | a magyar tegezés párja |
| G10 | Szóhasználat | – | felületen „cím” a letöltött oldalcím, „címke” a tag (angolul tag, németül Label) | a BASE-SYSTEM-PROMPT a címet „címke (title)” néven is említi; így a két fogalom nem keveredik |
| G11 | Fontfájlok mérete és hash-e | 3.2 | a 2.2 T12 sora szerinti fájlok | reprodukálhatóság |
| G12 | Csoportfeliratok a listában | `kx-eyebrow` csak címke-szerepben | a „Kitűzött” / „Többi” `kx-eyebrow` a rangsorolási szabályt mutatja | információ, nem díszítés |
| G13 | Címkeszínek | a `chart-1` … `chart-5` tokenek adatsor-színek, rögzített sorrendben, csak diagramon | a `chart-1` … `chart-5` a címkék színnégyzetei (4.6), kategoriális jelölésként, soha nem állapotként; a négyzet mellett mindig ott a címke neve | a tokenkészlet egyetlen kategoriális palettája; új szín nem kerül be, a szín önmagában nem hordoz jelentést (8.9) |

### 2.4 Eltérések a lead döntéseitől és indokuk

| # | Döntés | Eltérés | Indok és bizonyíték |
|---|---|---|---|
| L1 | D3: TypeScript 7.0.2 | `typescript` `~6.0.3` (6.0.3) | a TypeScript 7.0.2-vel a `svelte-kit sync` és a `vite build` leáll: `typescript_compiler_api_missing: TypeScript 7.0.2 does not provide the JavaScript compiler API required by SvelteKit. Install typescript@^6.0.0`. A `svelte-check` 4.7.8 peer-tartománya `^5 \|\| ^6`. A Dependabot a `>=7` főverziót kizárja |
| L2 | D9: CSP inline szkript csak a témaindító | a `script-src` a SvelteKit fallback `index.html`-jének saját inline indító szkriptjét is hash-sel engedi | az adapter-static fallback `index.html` két inline szkriptet tartalmaz; a SvelteKit-indító szövege minden buildnél változik. A szerver induláskor minden inline szkriptet hash-el (6.9); `'unsafe-inline'` nincs |
| L3 | D11: `/settings` | alútvonalak: `/settings/profile`, `/settings/security`, `/settings/display`, `/settings/privacy` (a `/settings` átirányít) + kapuoldalak: `/change-password`, `/consent` | diszjunkt fájltulajdon (F3, F5, F1), közvetlenül linkelhető fülek |
| L4 | D12: 85% Go | csomagonként legalább 80% is (a kapu mindkettőt nézi), frontend Vitest-küszöb 85/80/85/85 | „lehető legnagyobb lefedettség” |

---

## 3. Architektúra

### 3.1 Áttekintés

```
Böngésző (SvelteKit SPA, beágyazva)
   │  same-origin fetch, credentials: same-origin, X-Kinex-Request: 1
   ▼
Go net/http (egy bináris)
   recoverer → securityHeaders → accessLog → csrf → ServeMux
   ├── GET /healthz
   ├── /api/...  (authenticate(access) → kezelő)  ──► store (SQLite, modernc)
   │                                              ──► auth (PBKDF2, TOTP, kulcstartó)
   │                                              ──► jobs.Enqueue (metaadat-sor)
   ├── /api/ catch-all → 404 endpoint_not_found
   └── /  SPA-kezelő (index.html no-cache, _app/immutable + fonts/ immutable)

Háttér (jobs.Runner): archiválás + purge óránként, letöltési seprő percenként,
APP_FETCH_WORKERS letöltő munkás → fetch.Fetcher (SSRF-védelem) → store.ApplyFetchResult

Adatkönyvtár: app.db (+ -wal, -shm), secret.key
```

A build sorrendje kötött (TECHNOLOGY.md 1.): előbb a `web/build`, utána a Go-bináris. Az adatbázist a program induláskor hozza létre és migrálja.

### 3.2 Repository-szerkezet (végleges)

```
kinex-mark/
├── .github/
│   ├── dependabot.yml                 # gomod, npm (web/), github-actions, docker
│   └── workflows/
│       ├── ci.yml                     # web, go (1.26.x + 1.27.x), go-windows, e2e, binaries, image (a zöld tesztek után)
│       └── docker.yml                 # újrahívható (workflow_call): multi-arch image, ghcr.io push (PR-en build + smoke)
├── ai/
│   ├── BASE-SYSTEM-PROMPT.md          # bemenet, nem módosul
│   ├── BASE-SYSTEM-PLAN.md            # ez a terv
│   ├── TECHNOLOGY.md                  # bemenet, nem módosul
│   └── GRAPHICS.md                    # bemenet, nem módosul
├── cmd/kinex-mark/main.go             # -version, -healthcheck, config.Load, app.Run
├── internal/
│   ├── api/                           # HTTP: middleware, útvonaltábla, kezelők, SPA, CSP
│   │   └── testdata/contract.json     # generált szerződés-golden (9.6)
│   ├── app/                           # összekötés, életciklus, első admin, healthcheck, logger
│   ├── auth/                          # PBKDF2, TOTP, tokenek, kulcstartó, korlátozás
│   ├── bookmarks/                     # Netscape HTML és kinex-mark-links/1 JSON
│   ├── clock/                         # Clock, Real, Fake
│   ├── config/                        # kapcsolók + környezeti változók
│   ├── fetch/                         # metaadat-letöltő, SSRF, URL-normalizálás
│   ├── jobs/                          # archiválás, purge, letöltési sor
│   ├── model/                         # típusok, sentinel hibák, konstansok
│   ├── store/                         # SQLite, migrációk, lekérdezések
│   │   └── migrations/                # 0001_users.sql … 0004_system.sql
│   ├── storetest/                     # NewStore, SeedUser; csak más csomagok _test.go fájljai importálják
│   └── testutil/                      # T0, hamis HTTP-forrás; store-t nem importál; csak _test.go importálja
├── tools/
│   ├── covercheck/main.go             # Go-lefedettségi kapu
│   ├── langcheck/main.go              # „csak angol forráskód” ellenőrzés
│   └── fonts/build_fonts.py           # a két WOFF2 reprodukálása (buildben nem fut)
├── web/
│   ├── embed.go                       # //go:embed all:build, Files()
│   ├── assets_test.go                 # fontok mérete és hash-e, OFL, robots.txt, beágyazott build
│   ├── hero_test.go                   # hero.webp: létezés, WebP, 1600×900, méretkorlát
│   ├── package.json  package-lock.json  .npmrc  .gitignore  .prettierrc  .prettierignore
│   ├── tsconfig.json  vite.config.ts  vitest.config.ts  playwright.config.ts  components.json
│   ├── .vscode/extensions.json
│   ├── scripts/sync-app-css.mjs       # GRAPHICS.md 13.2 → src/app.css, angol kommentekkel
│   ├── static/
│   │   ├── favicon.svg  robots.txt
│   │   ├── fonts/                     # HankenGrotesk-Variable.woff2, GeistMono-Variable.woff2, OFL-*.txt
│   │   └── images/hero.webp           # generált
│   ├── e2e/
│   │   ├── support/                   # start-server.mjs, fixture-origin.mjs, constants.mjs, global-setup.ts,
│   │   │                              # global-teardown.ts, cleanup.mjs, fixtures.ts, api.ts, i18n.ts, totp.ts,
│   │   │                              # helpers.ts, seed-demo.mjs
│   │   ├── tsconfig.json              # az e2e és a konfigurációs fájlok típusellenőrzése (8.1.1)
│   │   ├── fixtures/                  # site/ (helyi metaadat-forrás), demo-data.json
│   │   └── specs/                     # *.spec.ts
│   ├── src/                           # 8.2
│   └── build/                         # generált, nem verziókövetett
├── graphics/
│   ├── generate_graphics.py           # Gemini-generátor (bővítve: aspect_ratio, image_size, max_width, background, 8.8.4)
│   ├── prompts.json                   # stílus + hero prompt
│   └── hero.png                       # mester PNG
├── manual/
│   ├── MANUAL.md
│   └── images/                        # képernyőképek (13.2)
├── dist/                              # generált binárisok, nem verziókövetett
├── data/                              # futásidejű adatkönyvtár, nem verziókövetett
├── .dockerignore  .gitattributes  .gitignore
├── Dockerfile  compose.yaml  Makefile  build.ps1
├── go.mod  go.sum
├── LICENSE                            # MIT (meglévő)
├── README.md                          # rövid magyar futtatási összefoglaló
└── SUMMARY.md                         # a fejlesztés állapota
```

Generált, nem verziókövetett: `dist/`, `data/`, `web/build/`, `web/node_modules/`, `web/.svelte-kit/`, `web/coverage/`, `web/test-results/`, `web/playwright-report/`, `web/blob-report/`, `web/e2e/.auth/`, `coverage.out`, `coverage.html`. A `go.sum` és a `web/package-lock.json` verziókövetett (a munkafában léteznie kell; a commitot a felhasználó végzi).

### 3.3 Go-csomagok és függőségi gráf

```
cmd/kinex-mark            -> internal/app, internal/config
internal/app              -> config, clock, model, auth, fetch, store, jobs, api, web
internal/api              -> model, clock, auth, fetch, bookmarks, config     (store-t NEM importál)
internal/jobs             -> model, clock, fetch                              (store-t NEM importál)
internal/store            -> model, clock, modernc.org/sqlite
internal/auth             -> clock                                            (store-t NEM importál)
internal/fetch            -> golang.org/x/net/html, golang.org/x/net/html/charset
internal/bookmarks        -> golang.org/x/net/html
internal/config           -> csak stdlib
internal/model            -> csak stdlib (levélcsomag)
internal/clock            -> csak stdlib (levélcsomag)
internal/testutil         -> model, clock (csak _test.go importálja; store-t NEM importál)
internal/storetest        -> model, clock, store (csak a store-on kívüli csomagok _test.go fájljai importálják)
web                       -> embed
```

Kulcsdöntés: az `api` és a `jobs` a saját maga által deklarált interfészeken át éri el az adatbázist; a sentinel hibák és az adattípusok a `model` levélcsomagban élnek. Így a `store`, `auth`, `fetch`, `jobs` és `api` párhuzamosan fejleszthető. Az összekötés az `internal/app`-ban, fordítási idejű ellenőrzéssel:

```go
var (
	_ api.Store  = (*store.Store)(nil)
	_ jobs.Store = (*store.Store)(nil)
)
```

Csak az `internal/app` importálja a `web` csomagot, ezért a többi csomag `web/build` nélkül is fordul és tesztelhető. A `go vet ./...` és a `go test ./...` a `web` csomag miatt `web/build`-et igényel (minden buildút előbb a felületet építi, 10.1).

Közös Go-szabályok:

- `gofmt`; `go vet ./...`; csak a `go.mod` minimumában (Go 1.26) meglévő standard API használható. A Go 1.25-ös `http.CrossOriginProtection`, `sync.WaitGroup.Go` és `testing/synctest` így elérhető, de a kód nem használja őket: a CSRF-védelem a 6.3 saját middleware-je marad, a tesztek a `clock.Fake`-re épülnek (9.2.1). A CI a 1.26.x-szel is fordít és tesztel `GOTOOLCHAIN=local` mellett (10.10).
- A `go 1.26.0` direktíva miatt a bináris az alapértelmezett GODEBUG-beállításokban a Go 1.25/1.26 viselkedését kapja (a `go list -f '{{.DefaultGODEBUG}}' ./cmd/kinex-mark` listája 11 elemről 2-re, `tracebacklabels=0, x509sslcertoverrideplatform=0` értékre szűkült): konténerérzékeny és frissülő GOMAXPROCS; a `url.Parse` elutasítja a hostban a hibás kettőspontot (szigorúbb bemenet a `fetch.NormalizeURL`-nek); a letöltő TLS 1.2 kézfogásában nincs SHA-1 aláírás; hibrid ML-KEM kulcscsere; a `crypto` figyelmen kívül hagyja az egyedi véletlenforrást; tartománylimit a `http.ServeContent`-ben. Egyik kódrész sem épít a régi viselkedésre; `//go:debug` felülírás nincs.
- Csomagolt hiba angolul: `fmt.Errorf("open database: %w", err)`. Tranzakciónál `defer tx.Rollback()`.
- Minden időbélyeget a `clock.Clock` ad UTC-ben; az SQL `DEFAULT` csak tartalék.

#### 3.3.1 `cmd/kinex-mark`

`main.go`: `-version`, `-healthcheck`, konfiguráció, logger, `app.Run`.

```go
var version = "dev" // -ldflags "-X main.version=..."

func main() {
	cfg, err := config.Load(flag.CommandLine, os.Args[1:], os.Getenv)
	if err != nil { /* log "invalid configuration", os.Exit(2) */ }
	if cfg.ShowVersion { fmt.Println(version); return }
	if cfg.HealthCheck { os.Exit(app.CheckHealth(cfg)) }
	log := app.NewLogger(cfg, os.Stdout)
	if cfg.AdminCommand != nil { // -unlock-user, -reset-password: offline recovery, then exit
		os.Exit(app.RunAdminCommand(context.Background(), cfg, os.Stdout, log))
	}
	if err := app.Run(context.Background(), cfg, version, log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
```

A `flag` csomag hibás kapcsolónál a súgót írja és 2-es kóddal lép ki (TECHNOLOGY.md 4.2).

#### 3.3.2 `internal/config`

Fájlok: `config.go`, `config_test.go`.

```go
type SecureCookies string // "auto" | "true" | "false"

type Config struct {
	Addr              string
	DataDir           string
	BaseURL           *url.URL      // nil, ha nincs megadva
	SecureCookies     SecureCookies
	TrustedProxies    []netip.Prefix
	AllowPrivateFetch *bool         // nil: a rendszerbeállítás dönt
	AllowLoopbackFetch bool         // csak APP_ALLOW_LOOPBACK_FETCH; rendszerbeállítás nincs (6.8)
	TLSCertFile       string
	TLSKeyFile        string
	LogLevel          slog.Level
	LogFormat         string        // "text" | "json"
	FetchWorkers      int
	FetchTimeout      time.Duration
	AdminUser         string
	AdminPassword     string        // csak környezeti változóból
	SecretKey         string        // csak környezeti változóból (base64)
	SecretKeyFile     string
	HealthCheck       bool
	ShowVersion       bool
	AdminCommand      *AdminCommand // nil: normál indulás
}

type AdminCommand struct{ Kind string; Username string } // Kind: "unlock-user" | "reset-password"

func Load(fs *flag.FlagSet, args []string, getenv func(string) string) (Config, error)
func (c Config) EnvLockedSettings() []string // pl. ["allowPrivateFetch"], ha APP_ALLOW_PRIVATE_FETCH meg van adva
```

Elsőbbség: kapcsoló, környezeti változó, alapérték (TECHNOLOGY.md 4.2 `env()` segédfüggvénye megmarad). A `Load` minden értéket validál; az első hiba alakja `config: APP_FETCH_WORKERS: must be between 1 and 16`. Az `APP_ADMIN_USER` és az adminparancsok felhasználóneve az 5.12 felhasználónév-szabályán megy át (`^[A-Za-z0-9._-]+$`, 3–64 karakter), különben `config: APP_ADMIN_USER: invalid username` és 2-es kilépési kód. A `-unlock-user` és a `-reset-password` egymást kizárja. A változók listája: 3.4.

#### 3.3.3 `internal/clock`

Fájlok: `clock.go`, `fake.go`, `clock_test.go`.

```go
type Clock interface{ Now() time.Time }
type Real struct{}                      // Now() = time.Now().UTC()
type Fake struct{ /* mu, now */ }       // NewFake(t time.Time), Set(t), Advance(d), Now()
```

#### 3.3.4 `internal/model`

Fájlok: `errors.go`, `roles.go`, `time.go`, `user.go`, `session.go`, `link.go`, `tag.go`, `share.go`, `settings.go`, `audit.go`, `export.go`, `fetch.go`, `page.go`. Csak típusok, konstansok, tiszta függvények; nincs I/O.

```go
// errors.go
var (
	ErrNotFound     = errors.New("not found")
	ErrNotOwner     = errors.New("not owner")
	ErrConflict     = errors.New("conflict")
	ErrLastAdmin    = errors.New("last active admin")
	ErrLimitReached = errors.New("limit reached")
)
type DuplicateError struct{ Kind string; ExistingID int64 } // Kind: "link" | "tag" | "share" | "username"

// roles.go
type Role string
const ( RoleAdmin Role = "admin"; RoleDeveloper Role = "developer"; RoleCustomer Role = "customer" )
func (r Role) Valid() bool
func (r Role) CanShare() bool      // admin, developer
func (r Role) CanListUsers() bool  // admin, developer
func (r Role) IsAdmin() bool

// time.go
const TimeLayout = "2006-01-02T15:04:05.000Z07:00" // mindig 3 tizedesjegy, UTC-ben "Z"
func FormatTime(t time.Time) string                // t.UTC().Format(TimeLayout)
func ParseTime(s string) (time.Time, error)

// link.go
type Viewer struct{ UserID int64; Role Role }
type TagColor string    // 4.6
type FetchStatus string // "pending" | "ok" | "failed" | "blocked"
```

A többi típus (`User`, `NewUser`, `UserPatch`, `UserRef`, `UserFilter`, `AdminUser`, `Credentials`, `Preferences`, `PreferencesPatch`, `Session`, `NewSession`, `Challenge`, `ThrottleRow`, `LinkItem`, `LinkDetail`, `LinkState`, `LinkQuery`, `LinkPage`, `NewLink`, `LinkPatch`, `IconBlob`, `Tag`, `TagPatch`, `Share`, `ShareTarget`, `SystemSettings`, `SettingsPatch`, `AuditEntry`, `AuditFilter`, `Consent`, `UserExport`, `ExportedLink`, `FetchJob`, `FetchResult`, `ImportEntry`, `ImportResult`, `PurgeResult`, `PageReq`, `Page`) mezőről mezőre az 5. fejezet JSON-alakjait követi, JSON-címkékkel. A `time.Time` mezők JSON-ban RFC 3339 UTC-ként kódolódnak; a nullázható mezők `*time.Time`, `*int64`, `*string`.

Kivétel a származtatott mezőkre: a `model.LinkItem` (és a `LinkDetail`, `LinkState`) a store-ból nyers `BaselineAt time.Time`, `LastUsedAt *time.Time` és `IconSHA256 string` mezőt hoz (`json:"-"`); az `inactiveSince`, az `archiveDueAt` és az `iconUrl` JSON-mezőt kizárólag az `internal/api/dto.go` számolja (5.5). A `model.ThrottleRow` mezői: `Key`, `UserID *int64`, `Failures`, `WindowStart`, `LockedUntil`, `Lockouts`, `UpdatedAt`; a `model.ThrottleKey{Key string; UserID *int64}` a `ThrottleUpdate` bemenete (a `UserID` csak létező fiók `u:` kulcsánál van kitöltve).

#### 3.3.5 `internal/auth`

| Fájl | Tartalom |
|---|---|
| `password.go` | PBKDF2-hash, ellenőrzés, újrahash-igény, jelszószabály, jelszógenerálás |
| `totp.go` | RFC 6238 TOTP, otpauth URI |
| `token.go` | munkamenet- és kihívástokenek, helyreállító kódok |
| `keyring.go` | szerverkulcs betöltése vagy létrehozása, AES-256-GCM lezárás, HMAC |
| `throttle.go` | a bejelentkezés-korlátozás tiszta számításai |

```go
const DefaultIterations = 600_000

type Hasher struct{ /* iterations int; sem chan struct{} */ }
func NewHasher(iterations, maxConcurrent int) *Hasher            // éles: DefaultIterations, 4
func (h *Hasher) Hash(password string) (string, error)
func (h *Hasher) Verify(encoded, password string) (ok, needsRehash bool, err error)
func (h *Hasher) DummyVerify(password string)                     // időzítés-kiegyenlítés ismeretlen felhasználónál
func CheckPasswordPolicy(password, username, current string) string // "" vagy mezőkód (6.4)
func GeneratePassword(length int) (string, error)                // ábécé: ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789

func NewTOTPSecret() (string, error)                              // 20 bájt, base32 padding nélkül
func TOTPCode(secret string, t time.Time) (string, error)
func VerifyTOTP(secret, code string, now time.Time, lastStep int64) (step int64, ok bool)
func TOTPURI(issuer, account, secret string) string

func NewToken() (token string, hash []byte, err error)            // 32 bájt -> base64url (43 karakter), hash = SHA-256
func HashToken(token string) []byte
func NewRecoveryCodes(n int) ([]string, error)                    // "abcde-23456" alak
func NormalizeRecoveryCode(s string) string                       // kisbetű, kötőjel és szóköz nélkül

type Keyring struct{ /* key [32]byte */ }
func LoadOrCreateKeyring(envKey, filePath string) (kr *Keyring, created bool, err error)
func (k *Keyring) Seal(plaintext []byte, associated string) (string, error)  // "v1:" + base64(nonce||ciphertext)
func (k *Keyring) Open(sealed, associated string) ([]byte, error)
func (k *Keyring) MAC(purpose string, data []byte) string                    // hex HMAC-SHA256(HMAC(key, purpose), data)
func (k *Keyring) CheckValue() string                                         // MAC("key-check", "kinex-mark")

type ThrottleState struct{ Failures int; WindowStart time.Time; LockedUntil *time.Time; Lockouts int }
type ThrottleRule struct{ MaxFailures int; Window, LockFor, MaxLock time.Duration; Escalate bool }
var UserRule = ThrottleRule{5, 15 * time.Minute, 15 * time.Minute, 24 * time.Hour, true}
var IPRule   = ThrottleRule{30, 15 * time.Minute, 15 * time.Minute, 15 * time.Minute, false}
// Check: locked, or the window already holds MaxFailures attempts (reserved or failed).
func (r ThrottleRule) Check(s ThrottleState, now time.Time) (blocked bool, retryAfter time.Duration)
func (r ThrottleRule) Reserve(s ThrottleState, now time.Time) ThrottleState // failures + 1 before verification (window reset first)
func (r ThrottleRule) Confirm(s ThrottleState, now time.Time) ThrottleState // verified failure: lock (and escalate) at MaxFailures
func (r ThrottleRule) Refund(s ThrottleState) ThrottleState                 // verified success: failures - 1, not below 0
func UserThrottleKey(k *Keyring, usernameKey string) string                // "u:" + k.MAC("throttle-username", usernameKey)
func IPThrottleKey(ip netip.Addr) string                                    // ip.Unmap(); IPv4 /32, IPv6 /64: "ip:2001:db8:1:2::/64"
```

A korlátozás olvasás–számítás–írás lépése mindig atomikus: az `api` a `store.ThrottleUpdate` hívásának átadja a fenti tiszta függvények egyikét, és a store egyetlen írótranzakcióban olvas, számol és ír (6.7).

#### 3.3.6 `internal/fetch`

Fájlok: `fetch.go` (Fetcher, Result, Error), `guard.go` (IP-szabályok, dial-ellenőrzés), `html.go` (cím és ikonlinkek kinyerése), `icon.go` (ikonválasztás és -ellenőrzés), `urls.go` (URL-normalizálás, URL kinyerése szövegből).

```go
type Policy struct{ AllowPrivate, AllowLoopback bool }
type Options struct {
	Timeout      time.Duration // teljes letöltés, alap 10 s
	MaxHTMLBytes int64         // 1 MiB
	MaxIconBytes int64         // 64 KiB
	MaxRedirects int           // 5
	UserAgent    string        // "KinexMark/<version> (+metadata fetcher)"
	Policy       func() Policy // futásidőben változhat (rendszerbeállítás); minden tárcsázáskor újra olvasva
	DialAllow    func(netip.AddrPort) bool // csak tesztben: a CheckIP után további szűkítés (például csak a hamis forrás portja); élesben nil
}
type Fetcher struct{ /* ... */ }
func New(opts Options) *Fetcher
func (f *Fetcher) Fetch(ctx context.Context, rawURL, acceptLanguage string) (Result, error)

type Result struct {
	FinalURL string
	Title    string // normalizált (6.8: vezérlő- és bidi-karakterek nélkül), legfeljebb 500 rúna, lehet üres
	Icon     *Icon  // nil, ha nincs érvényes ikon
	PageErr  *Error // nem nil, ha az oldal hibás volt, de az ikon-tartalék lefutott
}
type Icon struct{ Data []byte; ContentType string; SHA256 string }
type Error struct{ Code string; Temporary bool; Err error } // Code: 7.3 táblázat
func (e *Error) Error() string

func CheckIP(p Policy, ip netip.Addr) error // nil vagy *Error{Code:"blocked_address"}
func NormalizeURL(raw string) (normalized string, host string, err error) // err: *URLError
type URLError struct{ Code string } // "required" | "too_long" | "invalid_format" | "not_allowed" | "credentials_in_url"
func ExtractURL(text string) (string, bool)
func CleanText(s string, keepNewlines bool) string // C0/C1 vezérlő- és bidi-karakterek eltávolítása (6.8); a cím, az import címe és leírása is ezen megy át
```

#### 3.3.7 `internal/bookmarks`

Fájlok: `netscape.go`, `jsonfmt.go`, `bookmarks_test.go`.

```go
type Entry struct{ URL, Title, Description string; Tags []string; AddedAt *time.Time }
func ParseNetscape(r io.Reader, maxEntries int) ([]Entry, error)
func WriteNetscape(w io.Writer, entries []Entry) error
func ParseJSON(r io.Reader, maxEntries int) ([]Entry, error)
func WriteJSON(w io.Writer, exportedAt time.Time, entries []Entry) error
```

Netscape-írás: a fájl fejléce és címe nyelvfüggetlen, a szabványos Netscape-alak (`<!DOCTYPE NETSCAPE-Bookmark-file-1>`, `<TITLE>Kinex Mark</TITLE>`, `<H1>Kinex Mark</H1>`); a backend lokalizált szöveget itt sem ír (D10). Netscape-olvasás: minden `<A HREF>` egy bejegyzés; `ADD_DATE` Unix-másodperc; a `TAGS` attribútum vesszővel tagolt elemei és a legközelebbi befoglaló `<H3>` mappa neve címkeként kerül be (a hierarchia lapítva); a `<DD>` szöveg a leírás. JSON-formátum: `{"format":"kinex-mark-links/1","exportedAt":"…","links":[{"url","title","description","tags":[…],"addedAt"}]}`.

#### 3.3.8 `internal/store`

| Fájl | Tartalom |
|---|---|
| `driver.go` | `modernc.org/sqlite` import; a `kxm_lower` Unicode-kisbetűsítő SQL-függvény globális regisztrálása (`sqlite.MustRegisterDeterministicScalarFunction`; a beépített `lower()` csak ASCII-t kezel) |
| `store.go` | `Open`, `Close`, `Ping`, migráció (TECHNOLOGY.md 5.2 változatlanul), `withTx`, `querier` interfész, időformázás |
| `cancelsafe.go` | `cancelSafeConnector`, `cancelSafeConn`, `cancelSafeStmt`: megszakításbiztos kapcsolatburok a driver köré (lent) |
| `migrations/0001_users.sql` … `0005_fold_keys.sql` | 4. fejezet (a `0005` a finisher kiegészítése: a régi sorok kereső- és rendezőkulcsát ékezetmentesíti) |
| `users.go` | felhasználók, hitelesítő adatok, könyvtár, adminlista |
| `totp.go` | TOTP-állapot, helyreállító kódok |
| `sessions.go` | munkamenetek, bejelentkezési kihívások |
| `throttle.go` | korlátozási sorok |
| `prefs.go` | felhasználói beállítások |
| `links.go` | link CRUD, részletek, ikon |
| `link_query.go` | láthatósági CTE és listázó lekérdezés |
| `link_state.go` | kitűzés, archiválás, látogatás, személyes címkék |
| `fetchq.go` | metaadat-letöltés állapotai |
| `shares.go`, `tags.go` | megosztások, címkék |
| `settings.go` | rendszerbeállítások, `app_meta` |
| `audit.go`, `consent.go`, `gdpr.go` | audit, hozzájárulások, export és fióktörlés |
| `archive.go`, `purge.go` | háttérfeladatok SQL-je |
| `importexport.go` | saját linkek exportja, importja |

Nyilvános API (minden metódus első paramétere `ctx context.Context`; a láthatóságot és a tulajdonjogot minden metódus SQL-ben szűri):

```go
type Options struct{ Clock clock.Clock }
func Open(ctx context.Context, dataDir string, opts Options) (*Store, error)
func (s *Store) Close() error
func (s *Store) Ping(ctx) error

// app_meta, settings
func (s *Store) KeyCheck(ctx) (string, error)                       // "" ha még nincs
func (s *Store) SetKeyCheck(ctx, v string) error
func (s *Store) Settings(ctx) (model.SystemSettings, error)        // alapértékekkel összefésülve
func (s *Store) UpdateSettings(ctx, p model.SettingsPatch, by int64) (model.SystemSettings, error)
func (s *Store) BumpPrivacyNotice(ctx, by int64) (model.SystemSettings, error)

// users
func (s *Store) CountUsers(ctx) (int, error)
func (s *Store) CreateUser(ctx, u model.NewUser) (model.User, error)           // DuplicateError{Kind:"username"}
func (s *Store) UserByID(ctx, id int64) (model.User, error)
func (s *Store) Credentials(ctx, usernameKey string) (model.Credentials, error)
func (s *Store) AdminUsers(ctx, f model.UserFilter, p model.PageReq) ([]model.AdminUser, int, error)
func (s *Store) AdminUser(ctx, id int64) (model.AdminUser, error)
func (s *Store) UpdateUser(ctx, id int64, p model.UserPatch) (model.User, error) // ErrLastAdmin
func (s *Store) SetPassword(ctx, id int64, hash string, mustChange bool) error
func (s *Store) RecordLogin(ctx, id int64) error
func (s *Store) Directory(ctx, exclude int64, q string, limit int) ([]model.DirectoryUser, error)
func (s *Store) DeleteUser(ctx, id int64, e model.AuditEntry) error            // ErrLastAdmin; az e auditsort a tranzakcióban, a pszeudonimizálás előtt írja (6.11)
func (s *Store) OwnSharedLinkCount(ctx, userID int64) (int, error)              // saját linkek, amelyeknek van megosztása (törlési dialógusok)

// totp
func (s *Store) SetPendingTOTP(ctx, id int64, sealed string, expires time.Time) error
func (s *Store) PendingTOTP(ctx, id int64) (sealed string, expires time.Time, err error)
func (s *Store) EnableTOTP(ctx, id int64, sealed string, step int64, codeHashes []string) error
func (s *Store) DisableTOTP(ctx, id int64) error
func (s *Store) AdvanceTOTPStep(ctx, id, step int64) (bool, error)  // false = visszajátszás
func (s *Store) ReplaceRecoveryCodes(ctx, id int64, codeHashes []string) error
func (s *Store) UseRecoveryCode(ctx, id int64, codeHash string) (bool, error)
func (s *Store) RecoveryCodesRemaining(ctx, id int64) (int, error)

// sessions, challenges
func (s *Store) CreateSession(ctx, n model.NewSession) (model.Session, error)
func (s *Store) SessionByToken(ctx, hash []byte) (model.Session, model.User, error)
func (s *Store) TouchSession(ctx, id int64) error
func (s *Store) Sessions(ctx, userID int64) ([]model.Session, error)
func (s *Store) DeleteSession(ctx, userID, id int64) error
func (s *Store) DeleteSessionByToken(ctx, hash []byte) error
func (s *Store) DeleteUserSessions(ctx, userID, exceptID int64) (int, error) // exceptID 0: mind
func (s *Store) CreateChallenge(ctx, hash []byte, userID int64, ip string, expires time.Time) error // a felhasználó többi nyitott kihívását törli: felhasználónként egy él
// ChallengeAttempt reserves one attempt atomically before verification:
// UPDATE login_challenges SET attempts = attempts + 1
//  WHERE token_hash = :h AND attempts < :max AND expires_at > :now RETURNING user_id, attempts.
func (s *Store) ChallengeAttempt(ctx, hash []byte, maxAttempts int) (model.Challenge, error) // ErrNotFound: lejárt, ismeretlen vagy elfogyott
func (s *Store) DeleteChallenge(ctx, hash []byte) error

// throttle: read, compute and write in one write transaction (6.7)
func (s *Store) ThrottleUpdate(ctx, keys []model.ThrottleKey,
	fn func(row model.ThrottleRow) (model.ThrottleRow, error)) error // fn hibája: rollback, a hiba visszaadva
func (s *Store) ClearThrottle(ctx, key string) error
func (s *Store) ClearUserThrottle(ctx, userID int64) error // DELETE FROM login_throttle WHERE user_id = :id

// preferences
func (s *Store) Preferences(ctx, userID int64) (model.Preferences, error)
func (s *Store) UpdatePreferences(ctx, userID int64, p model.PreferencesPatch) (model.Preferences, error)

// links
func (s *Store) CreateLink(ctx, n model.NewLink) (model.LinkDetail, error)      // DuplicateError{Kind:"link"}, ErrLimitReached
func (s *Store) Link(ctx, v model.Viewer, id int64) (model.LinkDetail, error)   // ErrNotFound, ha nem látható
func (s *Store) Links(ctx, v model.Viewer, q model.LinkQuery) (model.LinkPage, error) // items, total, counts (4.8)
func (s *Store) LinkItem(ctx, v model.Viewer, id int64) (model.LinkItem, error)
func (s *Store) UpdateLink(ctx, v model.Viewer, id int64, p model.LinkPatch) (model.LinkDetail, error) // ErrNotOwner
func (s *Store) DeleteLink(ctx, v model.Viewer, id int64) error
func (s *Store) SetLinkTags(ctx, v model.Viewer, id int64, tagIDs []int64) error // csak saját címke
func (s *Store) SetPinned(ctx, v model.Viewer, id int64, pinned bool) error
func (s *Store) SetArchived(ctx, v model.Viewer, id int64, archived bool) error
func (s *Store) RecordVisit(ctx, v model.Viewer, id int64) (model.LinkState, error)
func (s *Store) LinkIcon(ctx, v model.Viewer, id int64) (model.IconBlob, error)

// metadata fetch
func (s *Store) DueFetches(ctx, limit int) ([]int64, error)
func (s *Store) BeginFetch(ctx, linkID int64, lease time.Duration) (model.FetchJob, error) // ErrNotFound, ha nem pending, vagy a next_fetch_at a jövőben van (bérlet alatt)
func (s *Store) ApplyFetchResult(ctx, r model.FetchResult) error                          // cím- és ikonszabály: 7.3
func (s *Store) ResetFetch(ctx, v model.Viewer, id int64) error // csak tulajdonos; 5.9.7

// shares, tags
func (s *Store) Shares(ctx, v model.Viewer, linkID int64) ([]model.Share, error)
func (s *Store) CreateShare(ctx, v model.Viewer, linkID int64, t model.ShareTarget) (model.Share, error)
func (s *Store) DeleteShare(ctx, v model.Viewer, linkID, shareID int64) error
func (s *Store) Tags(ctx, userID int64, role model.Role) ([]model.Tag, error)
func (s *Store) CreateTag(ctx, userID int64, name string, color model.TagColor) (model.Tag, error)
func (s *Store) UpdateTag(ctx, userID, id int64, p model.TagPatch) (model.Tag, error)
func (s *Store) DeleteTag(ctx, userID, id int64) error

// audit, consent, gdpr
func (s *Store) AppendAudit(ctx, e model.AuditEntry) error
func (s *Store) Audit(ctx, f model.AuditFilter, p model.PageReq) ([]model.AuditEntry, int, error)
func (s *Store) RecordConsent(ctx, userID int64, kind string, version int) error
func (s *Store) Consents(ctx, userID int64) ([]model.Consent, error)
func (s *Store) ExportUser(ctx, userID int64) (model.UserExport, error)         // sharedWithMe a 4.7 CTE-vel szűrve (5.8.1)
func (s *Store) ExportLinks(ctx, ownerID int64) ([]model.ExportedLink, error)
func (s *Store) ImportLinks(ctx, ownerID int64, entries []model.ImportEntry) (model.ImportResult, []int64, error)

// jobs
func (s *Store) ArchiveInactive(ctx, defaultDays int) (int, error)
func (s *Store) Purge(ctx, auditRetention time.Duration) (model.PurgeResult, error)
```

Megszakításbiztos kapcsolat (`cancelsafe.go`; a C fázis zárásakor került be, és tartósan marad):

- Ok: a `modernc.org/sqlite` v1.39.0 `stmt.query` függvényében a context végén induló megszakítás (`interruptOnDone`) versenyezhetett egy már sorokat adó lekérdezéssel; a driver ekkor a sorokat `Close` nélkül eldobta, a prepared statement nem finalizálódott, a `sqlite3_close_v2` zombi kapcsolatot hagyott, és a `Store.Close` után az `app.db` nyitva maradt (Windowson nem törölhető; a C-zárás próbájában 40 futásból 5-ször). Az upstream javítás a v1.40.1-ben jött (a halasztott blokk a context hibájának visszaadása előtt lezárja a sorokat; a v1.62.0-ban `stmt.go` 302–325. sor); a 2.2 T22 szerinti frissítés után a burok nélküli ellenőrző futás 2600 esetből 0-szor szivárgott, a régi v1.39.0 ugyanott 200-ból 23–30-szor.
- Megoldás: az `Open` a DSN-t egyszer építi, a `sql.Open("sqlite", dsn)` csak a regisztrált drivert adja (így a `kxm_lower` megmarad), a kapcsolatokat a `sql.OpenDB(cancelSafeConnector{drv, dsn})` nyitja. A `cancelSafeConn` mindent továbbít; csak a sort adó `QueryContext` (és a prepared statementek `QueryContext`-je) kap `context.WithoutCancel(ctx)`-et, előtte `ctx.Err()`-ellenőrzéssel, így a driver ezekre nem indítja el a versenyző megszakítást. A `database/sql` az eredeti contextet továbbra is alkalmazza a kapcsolatra várásra, és a context végén lezárja a sorokat (ami finalizálja a statementet). Az Exec, a `BeginTx` és a tranzakciók az eredeti contextet kapják. A burok a v1.62.0 `NewConnector` dokumentációja szerinti minden opcionális interfészt továbbít (`ConnBeginTx`, `ConnPrepareContext`, `ExecerContext`, `QueryerContext`, `Pinger`, `SessionResetter`, `Validator`), és `Unwrap() driver.Conn` metódusa van (a `sqlite.Limit` ezen át lát a burok mögé).
- Ár: egyetlen hosszú `sqlite3_step` az első sor előtt nem szakítható meg; a sorok bejárása megszakításra megáll. Helyi, személyes léptékű olvasásoknál ez elhanyagolható.
- A burok és a regressziós teszt a javított driver mellett is marad (mélységi védelem). Tesztek (`cancelsafe_test.go`): `TestCancelledQueriesReleaseDatabaseFile` (60× megszakítás, `Close`, a fájl törlése; burok nélkül v1.39.0-n elbukott), `TestCancelledQueryFailsWithContextError`, `TestCancelSafeConnQueriesWithoutCancellation` (az `Unwrap` ellenőrzésével), `TestCancelSafeStmtQueriesWithoutCancellation`, `TestCancelSafeConnectorErrors`. Driverfrissítéskor az új verzió `stmt.query`-jét újra kell vizsgálni, mielőtt a burok eltávolítását bárki javasolná (14).
- A B7 enyhítései az `internal/app`-ban (Ping a `Close` előtt, a tesztek megvárják az induló `purge run`-t) ezzel feleslegesek, de ártalmatlanok, ezért megmaradtak (3.3.11).

#### 3.3.9 `internal/jobs`

Fájlok: `runner.go` (Runner, Start, Wait, ütemezés), `archive.go`, `purge.go`, `metaqueue.go` (letöltési sor és munkások), `interfaces.go`.

```go
type Store interface {
	Settings(ctx context.Context) (model.SystemSettings, error)
	ArchiveInactive(ctx context.Context, defaultDays int) (int, error)
	Purge(ctx context.Context, auditRetention time.Duration) (model.PurgeResult, error)
	DueFetches(ctx context.Context, limit int) ([]int64, error)
	BeginFetch(ctx context.Context, linkID int64, lease time.Duration) (model.FetchJob, error)
	ApplyFetchResult(ctx context.Context, r model.FetchResult) error
}
type Fetcher interface {
	Fetch(ctx context.Context, rawURL, acceptLanguage string) (fetch.Result, error)
}
type Options struct{ Workers int; QueueSize int; Interval, SweepInterval time.Duration; Log *slog.Logger }
func New(st Store, f Fetcher, clk clock.Clock, opts Options) *Runner
func (r *Runner) Start(ctx context.Context)        // munkások + ütemező goroutine-ok
func (r *Runner) Enqueue(linkID int64) bool        // nem blokkol; false, ha a sor tele
func (r *Runner) RunArchive(ctx context.Context) (int, error)
func (r *Runner) RunPurge(ctx context.Context) (model.PurgeResult, error)
func (r *Runner) Wait(timeout time.Duration) bool  // a ctx lezárása után
```

#### 3.3.10 `internal/api`

| Fájl | Tartalom | Tulajdonos |
|---|---|---|
| `api.go` | `New(d Deps) (http.Handler, error)`, a globális middleware-lánc | B5 |
| `deps.go` | `Deps`, `Store` interfész (a szükséges store-metódusok), `Jobs` interfész (`Enqueue`, `RunArchive`) | A1 → B5 |
| `routes.go` | útvonaltábla adatszerkezetként (metódus, minta, hozzáférési szint, elutasítási kód, kezelő); a mux-regisztráció és a jogosultsági mátrixteszt ezt járja be | A1 → B5 |
| `errors.go` | hibakód-konstansok, `apiError`, `model`-hibák leképezése (végleges az A1-ben) | A1 → B5 |
| `respond.go` | `writeJSON`, `writeError`, `writeValidation`, `readJSON` (végleges az A1-ben) | A1 → B5 |
| `validate.go` | közös mezőellenőrzők (rúnaszám, vezérlőkarakter, lapozás, azonosító) (végleges az A1-ben) | A1 → B5 |
| `context.go` | a kérés-kontextus elérői (végleges az A1-ben): `withAuth(ctx, a authInfo) context.Context`, `viewerFrom(r *http.Request) model.Viewer`, `userFrom(r) model.User`, `sessionFrom(r) model.Session` | A1 → B5 |
| `middleware.go` | recover, hozzáférési napló, törzskorlát | B5 |
| `security.go` | biztonsági fejlécek, CSP | B5 |
| `csrf.go` | egyedi fejléc és Origin-ellenőrzés; `hostCheck` (6.3) | B5 |
| `session.go` | süti, hitelesítés, `requireReady` kapu, szerepkör-ellenőrzés | B5 |
| `clientip.go` | kliens-IP (`netip.Addr.Unmap`) és séma megbízható proxy mögött; nem megbízott forrás `X-Forwarded-*` fejléceire egyszeri Warn (6.2) | B5 |
| `spa.go` | SPA-kezelő (TECHNOLOGY.md 4.7 + `fonts/`), inline szkript hash-ek | B5 |
| `h_public.go` | `/healthz`, `/api/public/config` | B5 |
| `h_auth.go` | bejelentkezés, TOTP-lépés, kijelentkezés, me, jelszócsere | B5 |
| `h_twofa.go` | 2FA beállítás, bekapcsolás, kikapcsolás, helyreállító kódok | B5 |
| `h_me.go` | profil, beállítások, munkamenetek | B5 |
| `h_privacy.go` | hozzájárulás, GDPR-export, fióktörlés | B5 |
| `h_directory.go` | felhasználókönyvtár megosztáshoz | B5 |
| `h_admin_users.go`, `h_admin_settings.go`, `h_admin_audit.go`, `h_admin_jobs.go` | adminfelület | B5 |
| `h_links.go` | lista, létrehozás, gyors rögzítés, részletek, módosítás, törlés, frissítés | B6 |
| `h_links_state.go` | kitűzés, archiválás, látogatás, személyes címkék | B6 |
| `h_shares.go`, `h_icons.go`, `h_import.go`, `h_tags.go`, `dto.go` | megosztás, ikon, export/import, címkék, DTO-k; a `dto.go` számolja az `inactiveSince`, `archiveDueAt` és `iconUrl` mezőt (5.5) | B6 |
| `contract_test.go`, `testdata/contract.json` | szerződés-golden (9.6) | A1 → B5 |
| `apitest_test.go` | a HTTP-tesztek közös kerete (9.2.3): `newEnv`, `client`, `loginAs`, `apiClient`, `decode`, `expectError` | B5 (0. mérföldkő, 11.5) |

A B6 a B5 0. mérföldköve után indul (11.5): addigra az `api.go` (`New` a middleware-lánccal és a `routes.go` szerinti regisztrációval), a `session.go` (`authenticate`, `requireReady`, szerepkör-ellenőrzés), a `h_public.go` és az `apitest_test.go` végleges. A `loginAs` a munkamenetet közvetlenül a `store.CreateSession`-nel hozza létre és a sütit maga állítja be, így nem függ a bejelentkezési kezelőtől.

```go
type Deps struct {
	Store   Store
	Jobs    Jobs
	Hasher  *auth.Hasher
	Keyring *auth.Keyring
	Clock   clock.Clock
	Log     *slog.Logger
	Config  config.Config
	Version string
	UI      fs.FS
}
func New(d Deps) (http.Handler, error)
```

#### 3.3.11 `internal/app`

Fájlok: `app.go` (`Run`), `bootstrap.go` (első admin), `admincmd.go` (`RunAdminCommand`, 6.12), `server.go` (`http.Server`, TLS, leállítás), `health.go` (`CheckHealth`), `logger.go` (`NewLogger`), `app_test.go`.

`Run` lépései:

1. `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`.
2. `os.MkdirAll(cfg.DataDir, 0o750)`; hiba: `fmt.Errorf("create data dir: %w", err)`.
3. `auth.LoadOrCreateKeyring(cfg.SecretKey, keyFile)`, ahol `keyFile` = `cfg.SecretKeyFile`, alapból `<data>/secret.key`. Ha most jött létre: `log.Info("secret key created", "path", keyFile)`.
4. `store.Open(ctx, cfg.DataDir, store.Options{Clock: clock.Real{}})`; a lezárás a `closeStore` (7.5 5. lépés: előbb `Ping`, ami megvárja az aszinkron visszagörgetett egyetlen kapcsolatot, utána `st.Close()`; hibánál csak Warn).
5. Kulcsellenőrzés: ha `st.KeyCheck` üres, `SetKeyCheck(kr.CheckValue())`; ha eltér, a program nem indul: `errors.New("secret key does not match this database: restore the original key file or APP_SECRET_KEY")`. Ha a kulcsfájl ebben az indításban jött létre, eltérésnél a program törli (`log.Info("secret key removed", "path", keyFile)`), hogy az eredeti kulcs visszaállítását egy haszontalan új fájl ne akadályozza. A 2–5. lépés közös függvénye az `openData`, amelyet a `RunAdminCommand` is használ.
6. `bootstrapAdmin` (6.12).
7. `fetch.New(...)`, `jobs.New(...)` (a runner még nem indul; az `Enqueue` az indítás előtt is biztonságos).
8. `web.Files()`, `api.New(deps)`.
9. `http.Server` a TECHNOLOGY.md 4.1 időkorlátaival (`ReadHeaderTimeout` 5 s, `ReadTimeout` 30 s, `WriteTimeout` 30 s, `IdleTimeout` 2 perc), `MaxHeaderBytes: 64 << 10`, `ErrorLog` a slogba Warn szinten (a TLS-kézfogási és hasonló hibák is a közös naplóformátumban jelennek meg); TLS, ha `TLSCertFile` meg van adva (a tanúsítvány előre betöltve, `MinVersion: tls.VersionTLS12`). Ezután a figyelő megnyitása: `net.Listen("tcp", cfg.Addr)`; foglalt portnál `listen: …` hibával tér vissza (a `main` `server stopped`-ot naplóz), és háttérfeladat nem indul.
10. `runner.Start(jobsCtx)` (a `jobsCtx` a jelzéscontexttől független, így a jobok csak a HTTP-szerver leállása után állnak meg, 7.5); majd `log.Info("server started", "addr", <a figyelő valódi címe>, "data", …, "version", …, "tls", bool)`, végül `srv.Serve(ln)` / `srv.ServeTLS(ln, "", "")` a TECHNOLOGY.md 4.1 `select`-mintájával (`http.ErrServerClosed` nem hiba; `ctx.Done()` ágon `log.Info("shutting down")`). Eltérés a TECHNOLOGY.md 4.1-től (ott a napló a `ListenAndServe` előtt van) és a terv eredeti sorrendjétől (a jobok a 7. lépésben indultak): a figyelő megnyitása megelőzi a jobokat és a naplót, így a napló a valódi címet mutatja (a tesztek `127.0.0.1:0`-t használnak), és foglalt portnál nem keletkezik félrevezető `server started` sor.
11. Leállítás: 7.5.

`CheckHealth(cfg)`: a TECHNOLOGY.md 4.3 logikája (`0.0.0.0` és üres host → `127.0.0.1`; 3 s; `GET /healthz`; 0 vagy 1); TLS-módban `https://` és `InsecureSkipVerify: true` (csak a loopbackre megy a kérés).

`NewLogger(cfg, w)`: `slog` text vagy JSON kezelő a `LogFormat` szerint, `LogLevel` szinttel, stdout.

`RunAdminCommand(ctx, cfg, out, log) int`: az 1–5. lépés (adatkönyvtár, kulcstartó, store, kulcsellenőrzés) után HTTP-szerver és háttérfeladat nélkül végrehajtja a `-unlock-user` vagy a `-reset-password` parancsot (6.12), és kilép (0 siker, 1 hiba, például ismeretlen felhasználó).

#### 3.3.12 `internal/testutil` és `internal/storetest`

`internal/testutil` (store-t nem importál, ezért az `internal/store` saját tesztjei is használhatják): `clock.go` (`T0 = 2026-01-05T10:00:00Z`), `origin.go` (hamis HTTP-forrás a fetcherhez és az `AllowOnly` tárcsázási szűrő, 9.2.4). Az A1 fordítható vázként hozza létre (`doc.go`, `clock.go`), a B3 bővíti az `origin.go`-val.

`internal/storetest` (B1): `store.go` (`NewStore(t) *store.Store` ideiglenes könyvtárban, `t.Cleanup`), `seed.go` (`SeedUser(t, st, role) model.User`). Az `api`, a `jobs` és az `app` tesztjei importálják. Az `internal/store` tesztjei (`package store`, a belső segédfüggvények miatt) a `storetest`-et nem importálják (az `store → storetest → store` import-kör lenne); helyette a `store_test.go` saját `newTestStore(t)` és `seedUser(t, st, role)` segédfüggvényt ad. Mindkét csomagot csak `_test.go` importálja; a lefedettségi kapuból kimaradnak.

### 3.4 Konfiguráció

| Kapcsoló | Környezeti változó | Alapérték | Jelentés és érvényesség |
|---|---|---|---|
| `-addr` | `APP_ADDR` | `:8080` | figyelési cím (súgó: `listen address`) |
| `-data` | `APP_DATA_DIR` | `./data` | adatkönyvtár: `app.db`, `secret.key` |
| `-base-url` | `APP_BASE_URL` | üres | a felület külső címe (`https://mark.example.com`); séma `http`/`https`, út csak `/` vagy üres, query és fragment nélkül; az Origin-ellenőrzés, a Host-ellenőrzés és az `auto` sütimód használja (6.2, 6.3). **Reverse proxy mögött kötelező** |
| `-secure-cookies` | `APP_SECURE_COOKIES` | `auto` | `auto` \| `true` \| `false` (6.2) |
| `-trusted-proxy` | `APP_TRUSTED_PROXY` | üres | vesszővel tagolt IP-k vagy CIDR-ek, lehetőleg egyetlen cím (a proxyé); csak innen érkező kérésnél számít az `X-Forwarded-For` (jobbról az első nem megbízható cím), az `X-Forwarded-Proto` és az `X-Forwarded-Host`. Reverse proxy mögött kötelező, különben minden kliens a proxy IP-címén osztozik (6.7) |
| `-allow-private-fetch` | `APP_ALLOW_PRIVATE_FETCH` | üres | `true` \| `false` \| üres; megadva felülírja és zárolja az `allowPrivateFetch` rendszerbeállítást |
| `-allow-loopback-fetch` | `APP_ALLOW_LOOPBACK_FETCH` | `false` | `true` esetén a letöltő a loopback- és a nem metaadat link-local címeket is elérheti (6.8); rendszerbeállítás nincs hozzá, csak tesztkörnyezetbe (E2E) |
| `-tls-cert` | `APP_TLS_CERT` | üres | PEM-tanúsítvány; csak a `-tls-key`-jel együtt |
| `-tls-key` | `APP_TLS_KEY` | üres | PEM-kulcs |
| `-log-level` | `APP_LOG_LEVEL` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `-log-format` | `APP_LOG_FORMAT` | `text` | `text` \| `json` |
| `-fetch-workers` | `APP_FETCH_WORKERS` | `2` | 1–16 |
| `-fetch-timeout` | `APP_FETCH_TIMEOUT` | `10s` | Go-időtartam, 1s–60s |
| `-admin-user` | `APP_ADMIN_USER` | `admin` | az első admin felhasználóneve (csak üres adatbázisnál); az 5.12 felhasználónév-szabálya szerint validálva |
| nincs | `APP_ADMIN_PASSWORD` | üres | az első admin jelszava (csak üres adatbázisnál); kapcsoló szándékosan nincs (folyamatlista) |
| nincs | `APP_SECRET_KEY` | üres | 32 bájtos szerverkulcs base64-ben; elsőbbséget élvez a kulcsfájllal szemben |
| `-secret-key-file` | `APP_SECRET_KEY_FILE` | `<data>/secret.key` | kulcsfájl; ha nincs, a program létrehozza (0600, base64) |
| `-healthcheck` | nincs | `false` | TECHNOLOGY.md 4.3 |
| `-version` | nincs | `false` | kiírja a verziót, kilép |
| `-unlock-user <név>` | nincs | üres | offline helyreállítás: a felhasználó korlátozási sorainak törlése, audit, kilépés (6.12) |
| `-reset-password <név>` | nincs | üres | offline helyreállítás: generált jelszó egyszer a stdout-ra, `must_change_password = 1`, munkamenetek és korlátozás törlése, audit, kilépés (6.12) |

A súgószövegek angolok. A Dockerfile `ENV APP_ADDR=:8080 APP_DATA_DIR=/data` sora marad.

| Környezet | Cím | Adatkönyvtár |
|---|---|---|
| Helyi futtatás | `:8080` | `./data` |
| Fejlesztői mód (`build.ps1 dev`) | `127.0.0.1:8080` | `./data` |
| Vite fejlesztői szerver | `5173` | – |
| Docker image | `APP_ADDR=:8080` | `APP_DATA_DIR=/data` (kötet) |
| Docker Compose | `127.0.0.1:8080:8080` | `kinex-mark-data:/data` |
| E2E | `127.0.0.1:18080` (fixture-forrás `127.0.0.1:18081`) | ideiglenes könyvtár |


---

## 4. Adatmodell

### 4.1 Általános szabályok

- A DSN, a pragmák és a `SetMaxOpenConns(1)` a TECHNOLOGY.md 5.1 szerint, egy kiegészítéssel (2.2 T23): `file:<path>?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=secure_delete(1)`, minden `_pragma` egyetlen utasítás. A `secure_delete` a törölt tartalmat nullákkal írja felül (6.11). A kapcsolatot viszont nem a `sql.Open` pool nyitja, hanem a `sql.OpenDB(cancelSafeConnector{…})` a regisztrált `sqlite` driverrel (3.3.8). Az adatbázisfájl `<data>/app.db`.
- A migráció a TECHNOLOGY.md 5.2 szerint változatlan: beágyazott `migrations/*.sql`, név szerinti sorrend, fájlonként egy tranzakció, a `schema_migrations.name` a teljes beágyazott útvonal. Csak előre irányú; meglévő fájlt nem módosítunk, minden változás új fájl.
- Időbélyeg: `TEXT`, mindig `model.TimeLayout` szerint (`2026-11-21T09:15:00.123Z`, rögzített szélesség), ezért a szöveges összehasonlítás időrendi. Az értéket a Go írja be a `Clock`-ból.
- Logikai érték: `INTEGER` 0/1 `CHECK`-kel.
- Paraméterek: `sql.Named` (`:uid`) a többször ismétlődő értékekhez. A lekérdezésszöveg statikus, a választható szűrők `(:param IS NULL OR …)` mintát követnek.
- Az egyetlen kapcsolat miatt kötelező: amíg egy `rows` nyitva van, új lekérdezés nem indul; tranzakción belül csak a `tx` használható (az `s.db` hívása holtpontot okoz). A belső segédfüggvények `querier` interfészt kapnak (`ExecContext`, `QueryContext`, `QueryRowContext`), így ugyanaz a kód fut `*sql.DB`-n és `*sql.Tx`-en.
- A JSON-függvények (`json_each`, `json_group_array`, `json_valid`, `json_array_length`) a modernc SQLite-ban beépítettek.
- `INSERT … SELECT … ON CONFLICT` alaknál a `SELECT`-nek mindig legyen `WHERE` záradéka (érdemi feltétel híján `WHERE true`), különben az SQLite elemzője az `ON` kulcsszót join-feltételnek olvashatja.

### 4.2 `migrations/0001_users.sql`

```sql
CREATE TABLE users (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    username                TEXT    NOT NULL CHECK (length(username) BETWEEN 3 AND 64),
    username_key            TEXT    NOT NULL UNIQUE,
    display_name            TEXT    NOT NULL DEFAULT '' CHECK (length(display_name) <= 100),
    role                    TEXT    NOT NULL CHECK (role IN ('admin', 'developer', 'customer')),
    password_hash           TEXT    NOT NULL,
    must_change_password    INTEGER NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    is_active               INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    totp_secret             TEXT,
    totp_enabled_at         TEXT,
    totp_last_step          INTEGER NOT NULL DEFAULT 0,
    totp_pending_secret     TEXT,
    totp_pending_expires_at TEXT,
    password_changed_at     TEXT    NOT NULL,
    last_login_at           TEXT,
    created_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK ((totp_secret IS NULL) = (totp_enabled_at IS NULL)),
    CHECK ((totp_pending_secret IS NULL) = (totp_pending_expires_at IS NULL))
);

CREATE INDEX idx_users_role_active ON users (role, is_active);

CREATE TABLE user_preferences (
    user_id            INTEGER PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    locale             TEXT    NOT NULL DEFAULT 'hu'    CHECK (locale IN ('hu', 'en', 'de')),
    theme              TEXT    NOT NULL DEFAULT 'dark'  CHECK (theme IN ('dark', 'light', 'system')),
    view_mode          TEXT    NOT NULL DEFAULT 'list'  CHECK (view_mode IN ('list', 'tiles')),
    show_icon          INTEGER NOT NULL DEFAULT 1 CHECK (show_icon IN (0, 1)),
    show_url           INTEGER NOT NULL DEFAULT 1 CHECK (show_url IN (0, 1)),
    show_title         INTEGER NOT NULL DEFAULT 1 CHECK (show_title IN (0, 1)),
    show_description   INTEGER NOT NULL DEFAULT 1 CHECK (show_description IN (0, 1)),
    show_archived      INTEGER NOT NULL DEFAULT 0 CHECK (show_archived IN (0, 1)),
    sort               TEXT    NOT NULL DEFAULT 'usage' CHECK (sort IN ('usage', 'created', 'title')),
    tag_match          TEXT    NOT NULL DEFAULT 'any'   CHECK (tag_match IN ('any', 'all')),
    archive_after_days INTEGER CHECK (archive_after_days IS NULL OR archive_after_days BETWEEN 0 AND 3650),
    updated_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
```

- `username_key`: `strings.ToLower(username)`, mert a `COLLATE NOCASE` csak az ASCII-t kezeli. Az egyediség erre vonatkozik.
- Nincs e-mail-cím, telefonszám vagy más személyes adat (adattakarékosság).
- `totp_secret`, `totp_pending_secret`: a `Keyring.Seal` kimenete, a társított adat `"totp:<userID>"`.
- A `user_preferences` sort a `CreateUser` ugyanabban a tranzakcióban hozza létre.

### 4.3 `migrations/0002_auth.sql`

```sql
CREATE TABLE sessions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash   BLOB    NOT NULL UNIQUE CHECK (length(token_hash) = 32),
    user_id      INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at   TEXT    NOT NULL,
    last_seen_at TEXT    NOT NULL,
    expires_at   TEXT    NOT NULL,
    ip           TEXT    NOT NULL DEFAULT '' CHECK (length(ip) <= 45),
    user_agent   TEXT    NOT NULL DEFAULT '' CHECK (length(user_agent) <= 255)
);

CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_expires ON sessions (expires_at);
CREATE INDEX idx_sessions_last_seen ON sessions (last_seen_at);

CREATE TABLE login_challenges (
    token_hash BLOB    PRIMARY KEY CHECK (length(token_hash) = 32),
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    attempts   INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    ip         TEXT    NOT NULL DEFAULT '' CHECK (length(ip) <= 45),
    created_at TEXT    NOT NULL,
    expires_at TEXT    NOT NULL
) WITHOUT ROWID;

CREATE INDEX idx_login_challenges_expires ON login_challenges (expires_at);

CREATE TABLE recovery_codes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash  TEXT    NOT NULL CHECK (length(code_hash) = 64),
    used_at    TEXT,
    created_at TEXT    NOT NULL,
    UNIQUE (user_id, code_hash)
);

CREATE TABLE login_throttle (
    key               TEXT    PRIMARY KEY CHECK (length(key) <= 100),
    user_id           INTEGER REFERENCES users (id) ON DELETE CASCADE,
    failures          INTEGER NOT NULL DEFAULT 0 CHECK (failures >= 0),
    window_started_at TEXT    NOT NULL,
    locked_until      TEXT,
    lockouts          INTEGER NOT NULL DEFAULT 0 CHECK (lockouts >= 0),
    updated_at        TEXT    NOT NULL
) WITHOUT ROWID;

CREATE INDEX idx_login_throttle_updated ON login_throttle (updated_at);
CREATE INDEX idx_login_throttle_user ON login_throttle (user_id) WHERE user_id IS NOT NULL;
```

- A tétlenségi lejáratot nem tároljuk: a `last_seen_at` és az aktuális `sessionIdleMinutes` beállítás adja ki, így a beállítás módosítása azonnal hat.
- `login_throttle.key`: `u:<hex MAC>` (`auth.UserThrottleKey`: `Keyring.MAC("throttle-username", username_key)`, 66 karakter, létező és nem létező névre egyformán; a beírt név nyersen soha nem kerül a táblába) vagy `ip:<cím>` (az IPv4-be ágyazott IPv6-cím előbb kibontva: `netip.Addr.Unmap`; IPv4-nél a teljes cím, IPv6-nál a /64-es előtag, `ip:2001:db8:1:2::/64`).
- `login_throttle.user_id`: létező fiók `u:` sorában a fiók azonosítója (különben `NULL`). Erre épül az `AdminUser.lockedUntil`, a feloldás (`ClearUserThrottle`) és a törléskori kaszkád.
- `recovery_codes.code_hash`: `Keyring.MAC("recovery-code", normalizált kód)`, 64 hexa karakter (6.6).

### 4.4 `migrations/0003_links.sql`

```sql
CREATE TABLE icons (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    sha256       TEXT    NOT NULL UNIQUE CHECK (length(sha256) = 64),
    content_type TEXT    NOT NULL CHECK (content_type IN (
                     'image/png', 'image/x-icon', 'image/gif', 'image/jpeg', 'image/webp', 'image/svg+xml')),
    data         BLOB    NOT NULL CHECK (length(data) BETWEEN 1 AND 65536),
    created_at   TEXT    NOT NULL
);

CREATE TABLE links (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id       INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    url            TEXT    NOT NULL CHECK (length(url) BETWEEN 9 AND 2048),
    host           TEXT    NOT NULL CHECK (length(host) BETWEEN 1 AND 253),
    title          TEXT    NOT NULL DEFAULT '' CHECK (length(title) <= 500),
    title_source   TEXT    NOT NULL DEFAULT 'none' CHECK (title_source IN ('none', 'fetched', 'manual')),
    description    TEXT    NOT NULL DEFAULT '' CHECK (length(description) <= 500),
    search_text    TEXT    NOT NULL DEFAULT '',
    sort_label     TEXT    NOT NULL DEFAULT '',
    icon_id        INTEGER REFERENCES icons (id) ON DELETE SET NULL,
    fetch_status   TEXT    NOT NULL DEFAULT 'pending' CHECK (fetch_status IN ('pending', 'ok', 'failed', 'blocked')),
    fetch_error    TEXT    CHECK (fetch_error IS NULL OR length(fetch_error) <= 40),
    fetch_attempts INTEGER NOT NULL DEFAULT 0 CHECK (fetch_attempts >= 0),
    next_fetch_at  TEXT,
    fetched_at     TEXT,
    created_at     TEXT    NOT NULL,
    updated_at     TEXT    NOT NULL,
    UNIQUE (owner_id, url)
);

CREATE INDEX idx_links_owner_created ON links (owner_id, created_at DESC);
CREATE INDEX idx_links_fetch_due ON links (next_fetch_at) WHERE fetch_status = 'pending';
CREATE INDEX idx_links_icon ON links (icon_id);

CREATE TABLE tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id   INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT    NOT NULL CHECK (length(name) BETWEEN 1 AND 40),
    name_key   TEXT    NOT NULL,
    color      TEXT    NOT NULL DEFAULT 'neutral' CHECK (color IN (
                   'blue', 'ochre', 'green', 'violet', 'rose', 'neutral', 'contrast')),
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL,
    UNIQUE (owner_id, name_key)
);

CREATE TABLE link_tags (
    link_id    INTEGER NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    tag_id     INTEGER NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    created_at TEXT    NOT NULL,
    PRIMARY KEY (link_id, tag_id)
) WITHOUT ROWID;

CREATE INDEX idx_link_tags_tag ON link_tags (tag_id, link_id);

CREATE TABLE link_shares (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    link_id        INTEGER NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    target_type    TEXT    NOT NULL CHECK (target_type IN ('all', 'role', 'user')),
    target_role    TEXT    CHECK (target_role IN ('admin', 'developer', 'customer')),
    target_user_id INTEGER REFERENCES users (id) ON DELETE CASCADE,
    created_at     TEXT    NOT NULL,
    CHECK (
        (target_type = 'all'  AND target_role IS NULL     AND target_user_id IS NULL) OR
        (target_type = 'role' AND target_role IS NOT NULL AND target_user_id IS NULL) OR
        (target_type = 'user' AND target_role IS NULL     AND target_user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX ux_link_shares_target
    ON link_shares (link_id, target_type, coalesce(target_role, ''), coalesce(target_user_id, 0));
CREATE INDEX idx_link_shares_role ON link_shares (target_type, target_role);
CREATE INDEX idx_link_shares_user ON link_shares (target_user_id) WHERE target_user_id IS NOT NULL;

CREATE TABLE link_user_state (
    user_id        INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    link_id        INTEGER NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    pinned         INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1)),
    use_count      INTEGER NOT NULL DEFAULT 0 CHECK (use_count >= 0),
    last_used_at   TEXT,
    baseline_at    TEXT    NOT NULL,
    archived_at    TEXT,
    archive_reason TEXT    CHECK (archive_reason IN ('auto', 'manual')),
    updated_at     TEXT    NOT NULL,
    PRIMARY KEY (user_id, link_id),
    CHECK ((archived_at IS NULL) = (archive_reason IS NULL)),
    CHECK (NOT (pinned = 1 AND archived_at IS NOT NULL))
) WITHOUT ROWID;

CREATE INDEX idx_link_user_state_link ON link_user_state (link_id);
```

Mezőszabályok:

- `url`: a `fetch.NormalizeURL` kimenete (5.9.10). A duplikációt tulajdonosonként erre vizsgáljuk.
- `host`: a normalizált hostnév port nélkül (IDN esetén a felhasználó által megadott Unicode-alak kisbetűsítve). Ez adja a hosszú URL helyett megjelenített domaint.
- `title_source`: a cím eredete. `manual`: a felhasználó adta meg (`POST`/`PATCH` nem üres `title`-lel); a letöltés soha nem írja felül. `fetched`: a letöltés írta; minden későbbi sikeres letöltés (frissítés) felülírja. `none`: nincs cím (a `PATCH` üres `title`-je is ezt állítja, így a következő letöltés újra kitölti). URL-változáskor a `fetched` cím törlődik (`title = ''`, `title_source = 'none'`), a `manual` megmarad.
- `fetch_status`, `fetch_error`, `fetch_attempts`, `next_fetch_at`: minden olyan írás, amely a linket letöltésre jelöli (létrehozás, gyors rögzítés, URL-változás, kézi frissítés, import), ugyanabban az utasításban `fetch_status = 'pending', fetch_error = NULL, fetch_attempts = 0, next_fetch_at = :now` értéket ír, így a seprő (7.3) az elveszett `Enqueue` után is felveszi a linket.
- `search_text`: `foldText(url + "\n" + host + "\n" + title + "\n" + description)`, ahol a `foldText` (`internal/store/driver.go`, SQL-ben `kxm_fold`) kisbetűsít, NFD-re bont, elhagyja a kombináló jeleket, és NFC-re visszaállít (finisher-kiegészítés: az „arviz” keresés megtalálja az „Árvíz” szót). A Go számolja újra minden URL-, cím- és leírásváltozáskor; a keresőszó (`:q`) ugyanígy normalizálódik, a címkenév-egyezés `kxm_fold(t.name_key)`-vel történik. A `0005_fold_keys.sql` a korábbi sorokat alakítja át.
- `sort_label`: a megjelenítési címke (leírás, különben cím, különben host) `foldText`-tel, így az „Árvíz” az a betűs feliratok közé kerül, nem a z után. A `title` szerinti rendezés ezt használja; a kliens `web/src/lib/links/fold.ts` `foldText`-je ugyanez.
- `baseline_at`: az időpont, amelytől a tétlenség számít. Saját linknél a létrehozás, megosztott linknél a legkorábbi érvényes megosztás, visszaállításnál és a kitűzés megszüntetésénél annak ideje.
- A kitűzés és az archivált állapot kizárja egymást (CHECK): kitűzéskor az archiválás megszűnik, archiváláskor a kitűzés.
- A címkék személyesek: a `link_tags` sorában a címke tulajdonosa mindig a linket látó felhasználó. A címzett a megosztott linkre a saját címkéit teszi, a tulajdonos címkéit nem látja. A store ezt tranzakcióban ellenőrzi (`tags.owner_id = :uid`, és a link látható a felhasználónak).
- Az ikonok tartalom szerint deduplikáltak (`sha256`). A hozzáférés mindig a link láthatóságán át történik, ezért a közös ikonsor nem szivárogtat. A gazdátlan ikonokat a purge törli.

### 4.5 `migrations/0004_system.sql`

```sql
CREATE TABLE app_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) WITHOUT ROWID;

CREATE TABLE system_settings (
    key        TEXT    PRIMARY KEY,
    value      TEXT    NOT NULL CHECK (json_valid(value)),
    updated_at TEXT    NOT NULL,
    updated_by INTEGER REFERENCES users (id) ON DELETE SET NULL
) WITHOUT ROWID;

CREATE TABLE audit_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    at          TEXT    NOT NULL,
    event       TEXT    NOT NULL CHECK (length(event) <= 64),
    success     INTEGER NOT NULL DEFAULT 1 CHECK (success IN (0, 1)),
    actor_id    INTEGER REFERENCES users (id) ON DELETE SET NULL,
    actor_name  TEXT    NOT NULL DEFAULT '' CHECK (length(actor_name) <= 64),
    target_type TEXT    CHECK (target_type IN ('user', 'link', 'session', 'settings', 'tag')),
    target_id   INTEGER,
    target_name TEXT    NOT NULL DEFAULT '' CHECK (length(target_name) <= 64),
    ip          TEXT    NOT NULL DEFAULT '' CHECK (length(ip) <= 45),
    details     TEXT    NOT NULL DEFAULT '{}' CHECK (json_valid(details))
);

CREATE INDEX idx_audit_at ON audit_log (at);
CREATE INDEX idx_audit_event ON audit_log (event, id DESC);
CREATE INDEX idx_audit_actor ON audit_log (actor_id, id DESC);

CREATE TABLE consent_records (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind        TEXT    NOT NULL CHECK (kind IN ('privacy_notice')),
    version     INTEGER NOT NULL CHECK (version >= 1),
    accepted_at TEXT    NOT NULL,
    UNIQUE (user_id, kind, version)
);
```

- Az `app_meta` egyetlen kulcsa: `key_check` (a kulcstartó ellenőrzőértéke, 3.3.11 5. lépés).
- `system_settings`: a kulcs a `SystemSettings` JSON-mezőneve (5.12), az érték JSON. Csak az alapértéktől eltérő beállítás kerül a táblába; a `Settings()` a Go-alapértékekkel fésüli össze.
- Az audit napló `actor_name` és `target_name` mezője pillanatkép, mert a bejegyzésnek a felhasználó törlése után is olvashatónak kell maradnia (pszeudonimizálás: 6.11). Felhasználónév és megjelenített név csak ebben a két mezőben lehet; a `details` soha nem tartalmaz nevet (6.10).
- A `consent_records` a tájékoztató megismerésének és tudomásulvételének rögzítése, nem az adatkezelés jogalapja (6.14); ezért visszavonási folyamat nem tartozik hozzá.

### 4.6 Címkeszín-paletta

A címkeszín a hételemű, rögzített palettából választott kulcs. A backend csak a kulcsot tárolja és ellenőrzi, a felület a kulcsot design tokenre képezi. Nyers hex érték nem kerül a kódba; állapotszín (`success`, `warning`, `destructive`, `info`) nem jelöl címkét (GRAPHICS.md 2.1, 2.4), és a `primary` sem, mert az a fő műveleté és az aktív elemé.

| Kulcs | Token | Sötét | Világos | i18n kulcs |
|---|---|---|---|---|
| `blue` | `chart-1` | `#2ba0c2` | `#0e7fa3` | `tagColors.blue` (Kék) |
| `ochre` | `chart-2` | `#b08a2c` | `#9a7420` | `tagColors.ochre` (Okker) |
| `green` | `chart-3` | `#1f9b7c` | `#14876b` | `tagColors.green` (Zöld) |
| `violet` | `chart-4` | `#7a84e0` | `#5b63c7` | `tagColors.violet` (Ibolya) |
| `rose` | `chart-5` | `#cf6a95` | `#c04f7d` | `tagColors.rose` (Rózsa) |
| `neutral` | `muted-foreground` | `#94a9a5` | `#486469` | `tagColors.neutral` (Semleges, alapérték) |
| `contrast` | `foreground` | `#d4dcd5` | `#0b171f` | `tagColors.contrast` (Kontraszt) |

A sorrend egyben a választó sorrendje; a `GET /api/public/config` `tagColors` mezője is ebben a sorrendben adja. A hex értékek csak tájékoztató jellegűek (a GRAPHICS.md 13.2 tokenjei), a kódban csak a token neve szerepel.

### 4.7 Láthatósági lekérdezés

Egy felhasználó (`:uid`, szerepkör `:role`) a következő linkeket látja:

1. a saját linkjeit (`links.owner_id = :uid`);
2. más tulajdonos linkjét, ha van rá érvényes megosztás: mindenkinek (`all`), a szerepkörének (`role` = `:role`), vagy személyesen neki (`user` = `:uid`);
3. a megosztás csak akkor érvényes, ha a tulajdonos jelenlegi szerepköre `admin` vagy `developer`. Lefokozáskor a korábbi megosztások így automatikusan hatástalanná válnak, adatvesztés nélkül. Az inaktivált tulajdonos megosztásai érvényben maradnak (a céges tudás nem tűnik el), a törölt tulajdonos linkjei a kaszkáddal megszűnnek.

A közös CTE (a `link_query.go` `visibleCTE` konstansa; minden láthatóságfüggő lekérdezés ezzel kezdődik):

```sql
WITH vis AS (
    SELECT l.id AS link_id, NULL AS via, NULL AS shared_at
    FROM links l
    WHERE l.owner_id = :uid
    UNION ALL
    SELECT s.link_id, s.target_type, s.created_at
    FROM link_shares s
    JOIN links l ON l.id = s.link_id
    JOIN users o ON o.id = l.owner_id AND o.role IN ('admin', 'developer')
    WHERE l.owner_id <> :uid
      AND (s.target_type = 'all'
           OR (s.target_type = 'role' AND s.target_role = :role)
           OR (s.target_type = 'user' AND s.target_user_id = :uid))
),
v AS (
    SELECT link_id,
           MIN(shared_at) AS shared_at,
           CASE WHEN SUM(via = 'user') > 0 THEN 'user'
                WHEN SUM(via = 'role') > 0 THEN 'role'
                WHEN SUM(via = 'all')  > 0 THEN 'all'
           END AS via
    FROM vis
    GROUP BY link_id
)
```

Egyetlen link lekérése (`Link`, `LinkItem`, `LinkIcon`, `RecordVisit`, `SetPinned`, `SetArchived`, `SetLinkTags`): ugyanez a CTE `WHERE v.link_id = :id` feltétellel. Ha nincs sor, `model.ErrNotFound`, így a nem látható és a nem létező link megkülönböztethetetlen (IDOR-védelem). Tulajdonosi művelet (`UpdateLink`, `DeleteLink`, `ResetFetch`, megosztások): `WHERE id = :id AND owner_id = :uid`; ha a link látható, de nem saját, `model.ErrNotOwner` (403), különben `model.ErrNotFound` (404).

Beágyazott azonosító (szabály minden jelenlegi és jövőbeli `/{id}/…/{childId}` útvonalra): a gyermekazonosítónak a szülő útvonal-azonosítójához és a néző tulajdonához is tartoznia kell, egyetlen utasításban. A megosztás törlése (`DeleteShare`):

```sql
DELETE FROM link_shares
WHERE id = :share_id
  AND link_id = :link_id
  AND link_id IN (SELECT id FROM links WHERE id = :link_id AND owner_id = :uid)
```

0 érintett sor esetén `model.ErrNotFound` (404 `not_found`), kivéve ha a link látható, de nem saját (`model.ErrNotOwner`, 403). Így B a saját linkjének útvonalán sem vonhatja vissza A megosztását.

### 4.8 Listázó és rendező lekérdezés

```sql
-- visibleCTE, majd:
SELECT l.id, l.url, l.host, l.title, l.title_source, l.description, l.fetch_status,
       l.owner_id, o.username, o.display_name,
       ic.sha256,
       coalesce(st.pinned, 0)    AS pinned,
       coalesce(st.use_count, 0) AS use_count,
       st.last_used_at, st.archived_at, st.archive_reason,
       coalesce(st.baseline_at,
                CASE WHEN l.owner_id = :uid THEN l.created_at
                     ELSE max(v.shared_at, (SELECT vu.created_at FROM users vu WHERE vu.id = :uid)) END) AS baseline_at,
       v.via, v.shared_at,
       (SELECT json_group_array(lt.tag_id)
          FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
         WHERE lt.link_id = l.id AND t.owner_id = :uid) AS tag_ids,
       CASE WHEN l.owner_id = :uid
            THEN (SELECT COUNT(*) FROM link_shares s2 WHERE s2.link_id = l.id)
            ELSE 0 END AS share_count,
       l.created_at, l.updated_at,
       COUNT(*) OVER () AS total,
       SUM(coalesce(st.pinned, 0)) OVER () AS pinned_count,
       SUM(st.archived_at IS NOT NULL) OVER () AS archived_count
FROM v
JOIN links l ON l.id = v.link_id
JOIN users o ON o.id = l.owner_id
LEFT JOIN icons ic ON ic.id = l.icon_id
LEFT JOIN link_user_state st ON st.user_id = :uid AND st.link_id = l.id
WHERE (:archived = 'include'
       OR (:archived = 'exclude' AND st.archived_at IS NULL)
       OR (:archived = 'only'    AND st.archived_at IS NOT NULL))
  AND (:scope = 'all'
       OR (:scope = 'own'    AND l.owner_id = :uid)
       OR (:scope = 'shared' AND l.owner_id <> :uid))
  AND (:q IS NULL
       OR instr(l.search_text, :q) > 0
       OR EXISTS (SELECT 1 FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
                   WHERE lt.link_id = l.id AND t.owner_id = :uid AND instr(t.name_key, :q) > 0))
  AND (:tag_ids IS NULL
       OR (:tag_match = 'any' AND EXISTS (
              SELECT 1 FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
               WHERE lt.link_id = l.id AND t.owner_id = :uid
                 AND lt.tag_id IN (SELECT value FROM json_each(:tag_ids))))
       OR (:tag_match = 'all' AND (
              SELECT COUNT(DISTINCT lt.tag_id) FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
               WHERE lt.link_id = l.id AND t.owner_id = :uid
                 AND lt.tag_id IN (SELECT value FROM json_each(:tag_ids)))
              = json_array_length(:tag_ids)))
  AND (:untagged = 0 OR NOT EXISTS (
          SELECT 1 FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
           WHERE lt.link_id = l.id AND t.owner_id = :uid))
ORDER BY coalesce(st.pinned, 0) DESC,
         CASE WHEN :sort = 'usage' THEN coalesce(st.use_count, 0) END DESC,
         CASE WHEN :sort = 'usage' THEN st.last_used_at END DESC NULLS LAST,
         CASE WHEN :sort = 'title' THEN l.sort_label END ASC,
         l.created_at DESC,
         l.id DESC
LIMIT :limit OFFSET :offset
```

- Rendezés: kitűzött elöl; a kitűzött és a nem kitűzött csoporton belül is `usage` szerint (`use_count` csökkenő, `last_used_at` csökkenő, a soha nem használt a végén, majd `created_at` csökkenő). `created` rendezésnél csak a kitűzés és a létrehozás számít, `title` rendezésnél a kitűzés, majd a megjelenítési címke ábécérendben.
- `:q`: a kérés `q` paramétere vágva és `strings.ToLower`-rel kisbetűsítve; üresen `NULL`. Az `instr` miatt nincs LIKE-escape-probléma.
- `:tag_ids`: JSON-tömb szövegként (`"[3,5]"`), üresen `NULL`.
- A címkeazonosítók egy oszlopban érkeznek (`json_group_array`), így nincs N+1 lekérdezés, és nincs beágyazott lekérdezés nyitott `rows` mellett. A frontend a `GET /api/tags` válaszából párosítja a neveket és a színeket.
- `baseline_at`: az `inactiveSince` és az `archiveDueAt` számításához (5.9.1). Hiányzó állapotsornál saját linkre a létrehozás, megosztottra a legkorábbi érvényes megosztás és a néző fiókjának létrehozása (`users.created_at`, a kódban `viewerCreatedAt`) közül a későbbi: aki a megosztáskor még nem létezett, nem hagyhatta használatlanul a linket; ez megegyezik azzal, amit az archiváló feladat beírna (7.2). A szerepkör-változás vagy a megosztó visszatérő megosztójoga miatt újonnan láthatóvá vált linkek állapotsort kapnak a változás idejével (7.2), ezért ott a tartalékérték nem érvényesül.
- `total`: a `COUNT(*) OVER ()` a szűrt halmaz mérete; a `pinned_count` és az `archived_count` ugyanennek a szűrt halmaznak a kitűzött és archivált elemszáma (a válasz `counts` mezője, 5.9.1). Ha az `offset` a halmazon túl van (nincs sor), a store a `rows` lezárása után egy külön, azonos szűrésű `SELECT COUNT(*), SUM(…), SUM(…)` lekérdezéssel adja meg mindhármat.
- A rendezés a kitűzött csoporton belül is ugyanaz: két kitűzött link közül a nagyobb `use_count`, egyenlőségnél a későbbi `last_used_at` áll elöl (9.2.2 tesztje).
- A hiányzó `link_user_state` sor alapértéke: nincs kitűzve, 0 használat, nincs archiválva. A sort az első személyes művelet (látogatás, kitűzés, archiválás) vagy az archiváló feladat hozza létre; saját linknél a `CreateLink` azonnal létrehozza (`baseline_at = created_at`).

### 4.9 Személyes állapot írása

Minden állapotírás upsert, a láthatóság ellenőrzése után ugyanabban a tranzakcióban:

```sql
-- RecordVisit
INSERT INTO link_user_state (user_id, link_id, use_count, last_used_at, baseline_at, updated_at)
VALUES (:uid, :id, 1, :now, :now, :now)
ON CONFLICT (user_id, link_id) DO UPDATE SET
    use_count = use_count + 1, last_used_at = :now,
    archived_at = NULL, archive_reason = NULL, updated_at = :now;

-- SetPinned(true)
INSERT INTO link_user_state (user_id, link_id, pinned, baseline_at, updated_at)
VALUES (:uid, :id, 1, :now, :now)
ON CONFLICT (user_id, link_id) DO UPDATE SET
    pinned = 1, archived_at = NULL, archive_reason = NULL, baseline_at = :now, updated_at = :now;

-- SetPinned(false): ugyanez az alak, pinned = 0, baseline_at = :now
-- (a kitűzés megszűnésétől számít a tétlenség)

-- SetArchived(true)
INSERT INTO link_user_state (user_id, link_id, archived_at, archive_reason, baseline_at, updated_at)
VALUES (:uid, :id, :now, 'manual', :now, :now)
ON CONFLICT (user_id, link_id) DO UPDATE SET
    pinned = 0, archived_at = :now, archive_reason = 'manual', updated_at = :now;

-- SetArchived(false) (visszaállítás)
INSERT INTO link_user_state (user_id, link_id, baseline_at, updated_at)
VALUES (:uid, :id, :now, :now)
ON CONFLICT (user_id, link_id) DO UPDATE SET
    archived_at = NULL, archive_reason = NULL, baseline_at = :now, updated_at = :now;
```

### 4.10 Adatmegőrzés

A megőrzési idők és a törlés módja: 6.13.

---

## 5. API-szerződés

Ez a fejezet a backend és a frontend közötti kötelező szerződés. A frontend `web/src/lib/api/types.ts` fájlja az 5.5 és az 5.8.1 TypeScript-blokkját szó szerint veszi át; minden végpont, mező és hibakód, amelyet a felület használ, itt szerepel. Változás csak ennek a fejezetnek a módosításával együtt történhet; a szerződés-golden (9.6) a Go- és a TypeScript-oldal eltérését build-hibává teszi.

### 5.1 Közös szabályok

- Alap: `/api`, ugyanazon az originen, mint a felület (TECHNOLOGY.md 6.6). CORS nincs.
- Válasz: `Content-Type: application/json; charset=utf-8`, JSON és záró újsor (TECHNOLOGY.md 4.5). Minden `/api/` válasz `Cache-Control: no-store` (kivétel: ikon, 5.9.9).
- Kérés: `readJSON` a TECHNOLOGY.md 4.5 szerint (`http.MaxBytesReader` 64 KiB, `DisallowUnknownFields`, egyetlen `Decode`). Bármilyen dekódolási hiba 400 `invalid_json` (a túl nagy törzs is). Kivétel: `POST /api/links/import` 2 MiB-os korláttal, túllépéskor 413 `payload_too_large`.
- Mezőnevek camelCase-ben. Időbélyeg: RFC 3339 UTC (`2026-11-21T09:15:00.123Z`; a záró nullák elmaradhatnak). Azonosító: JSON-szám (int64, a böngésző számára biztonságos tartományban). Hiányzó érték `null`, üres lista `[]`, soha nem `null`.
- Részleges módosítás (`PATCH`): csak a jelen lévő mezők változnak; Go-oldalon minden mező mutató (`*string`, `*bool`), így a hiány felismerhető. `null` csak ott megengedett, ahol a szerződés jelzi (`archiveAfterDays: null` = a rendszer alapértéke).
- Szöveges bemenet: `strings.TrimSpace`, majd rúnaszám-ellenőrzés (`utf8.RuneCountInString`); vezérlőkarakter (U+0000–U+001F, U+007F és U+0080–U+009F, a leírásban a `\n` kivételével), illetve bidi-jelölő, -felülíró vagy -izoláló karakter (a Unicode Bidi_Control halmaza: U+061C, U+200E, U+200F, U+202A–U+202E, U+2066–U+2069) esetén `invalid` mezőkód.
- Azonosító az útvonalban: `strconv.ParseInt`; hiba vagy `< 1` esetén 400 `invalid_id` (TECHNOLOGY.md 4.5).
- Ismeretlen `/api/` útvonal vagy nem regisztrált metódus: 404 `endpoint_not_found` (TECHNOLOGY.md 4.4: a `/api/` catch-all miatt nem 405).
- Minden állapotváltoztató kérés (`POST`, `PUT`, `PATCH`, `DELETE`) a CSRF-szabály alá esik (6.3), a bejelentkezés is.
- A frontend minden kérésnél küldi: `X-Kinex-Request: 1`, `Accept: application/json`, `credentials: 'same-origin'`.

### 5.2 Hibatörzs

```json
{
  "error": "validation failed",
  "code": "validation_failed",
  "fields": { "url": "not_allowed", "description": "too_long" },
  "details": { "linkId": 42 }
}
```

- `error`: rövid angol üzenet, csak naplózáshoz és fejlesztőknek. A felület soha nem jeleníti meg.
- `code`: stabil kód az 5.3 listából. A felület ezt fordítja (`errors.<code>.title`, `errors.<code>.text`).
- `fields`: opcionális, mezőnév → mezőkód; `validation_failed` esetén kötelező, továbbá `current_password_invalid` (`currentPassword: "incorrect"`) és `invalid_query` (paraméternév → `invalid`) esetén szerepel.
- `details`: opcionális gépi kiegészítés (`linkId`, `tagId`, `retryAfterSeconds`, `keys`).
- 429 esetén `Retry-After` fejléc is (másodperc).
- 500: `{"error":"internal error","code":"internal_error"}`; a részlet csak a naplóba kerül (`log.Error("request failed", "err", err, "route", pattern)`).

### 5.3 Hibakódok

Minden kódhoz tartozik i18n-kulcs (`errors.<code>.title` és `errors.<code>.text`) mindhárom nyelven; a szövegeket a 8.5.3 táblázat adja. A kliens saját kódja a `network_error` (HTTP-státusz 0).

| Kód | HTTP | Jelentés | i18n kulcs |
|---|---|---|---|
| `endpoint_not_found` | 404 | Nincs ilyen API-útvonal vagy metódus | `errors.endpoint_not_found` |
| `invalid_json` | 400 | Hibás, üres, túl nagy vagy ismeretlen mezőt tartalmazó JSON | `errors.invalid_json` |
| `invalid_id` | 400 | Hibás útvonal-azonosító | `errors.invalid_id` |
| `invalid_query` | 400 | Hibás query-paraméter (`fields` a paraméternévvel) | `errors.invalid_query` |
| `validation_failed` | 400 | Mezőszintű hiba, `fields` kötelező | `errors.validation_failed` |
| `payload_too_large` | 413 | Importfájl 2 MiB fölött | `errors.payload_too_large` |
| `not_found` | 404 | Az erőforrás nem létezik vagy nem látható | `errors.not_found` |
| `unauthenticated` | 401 | Nincs érvényes munkamenet | `errors.unauthenticated` |
| `forbidden` | 403 | A szerepkör nem jogosult | `errors.forbidden` |
| `csrf_rejected` | 403 | Hiányzó `X-Kinex-Request` fejléc vagy idegen origin | `errors.csrf_rejected` |
| `host_mismatch` | 421 | A kérés `Host` fejléce nem a beállított külső cím, vagy loopbackre kötött szerveren nem loopback név (6.3) | `errors.host_mismatch` |
| `password_change_required` | 403 | Kötelező jelszócsere van folyamatban | `errors.password_change_required` |
| `consent_required` | 403 | Az adatkezelési tájékoztató aktuális verzióját el kell fogadni | `errors.consent_required` |
| `invalid_credentials` | 401 | Hibás felhasználónév vagy jelszó | `errors.invalid_credentials` |
| `account_disabled` | 403 | Helyes jelszó, de a fiók inaktív | `errors.account_disabled` |
| `too_many_attempts` | 429 | Bejelentkezés-korlátozás, `details.retryAfterSeconds` | `errors.too_many_attempts` |
| `challenge_invalid` | 401 | A 2FA-kihívás lejárt, ismeretlen, vagy elfogytak a próbálkozások | `errors.challenge_invalid` |
| `invalid_totp` | 401 | Hibás vagy már felhasznált TOTP-kód, illetve helyreállító kód | `errors.invalid_totp` |
| `current_password_invalid` | 400 | Hibás jelenlegi jelszó (`fields.currentPassword = "incorrect"`); az admin érzékeny műveleteinek megerősítésénél is (5.12) | `errors.current_password_invalid` |
| `totp_already_enabled` | 409 | A 2FA már be van kapcsolva | `errors.totp_already_enabled` |
| `totp_not_enabled` | 409 | A 2FA nincs bekapcsolva | `errors.totp_not_enabled` |
| `totp_setup_required` | 409 | Nincs (vagy lejárt) a függő 2FA-beállítás | `errors.totp_setup_required` |
| `link_duplicate` | 409 | A tulajdonosnak már van ilyen URL-je, `details.linkId` | `errors.link_duplicate` |
| `link_limit_reached` | 409 | Elérte a 10 000 saját linket | `errors.link_limit_reached` |
| `not_owner` | 403 | A link látható, de a művelet csak a tulajdonosé | `errors.not_owner` |
| `share_not_allowed` | 403 | A szerepkör nem oszthat meg (ügyfél) | `errors.share_not_allowed` |
| `share_duplicate` | 409 | Ugyanez a megosztás már létezik | `errors.share_duplicate` |
| `no_url_found` | 400 | A gyors rögzítés szövegében nincs URL | `errors.no_url_found` |
| `import_invalid` | 400 | Az importfájl nem értelmezhető | `errors.import_invalid` |
| `import_too_many_entries` | 400 | Az importfájl 5000-nél több bejegyzést tartalmaz (finisher-kiegészítés) | `errors.import_too_many_entries` |
| `tag_duplicate` | 409 | Ilyen nevű címke már van, `details.tagId` | `errors.tag_duplicate` |
| `tag_limit_reached` | 409 | Elérte az 500 címkét | `errors.tag_limit_reached` |
| `username_taken` | 409 | A felhasználónév foglalt | `errors.username_taken` |
| `last_admin` | 409 | Az utolsó aktív admin nem fokozható le, nem inaktiválható, nem törölhető | `errors.last_admin` |
| `self_modification` | 409 | Az admin a saját szerepkörét és aktív állapotát nem módosíthatja, magát nem törölheti az adminfelületen | `errors.self_modification` |
| `setting_locked` | 409 | Környezeti változó rögzíti a beállítást, `details.keys` | `errors.setting_locked` |
| `consent_version_mismatch` | 409 | Az elfogadott verzió nem az aktuális | `errors.consent_version_mismatch` |
| `confirmation_mismatch` | 400 | A fióktörlés megerősítő szövege nem a felhasználónév | `errors.confirmation_mismatch` |
| `not_implemented` | 501 | Csak az A1 fázis csonkjaiban (a B5/B6 után egyetlen útvonal sem adja) | `errors.not_implemented` |
| `internal_error` | 500 | Váratlan hiba | `errors.internal_error` |
| `network_error` | 0 (kliens) | A `fetch` kivételt dobott | `errors.network_error` |

Mezőkódok (a `fields` értékei; i18n: `fields.<code>`, mezőnkénti felülírás `fields.byField.<field>.<code>`): `required`, `too_short`, `too_long`, `invalid`, `invalid_format`, `not_allowed`, `out_of_range`, `unknown` (ismeretlen vagy idegen azonosító, például címke vagy felhasználó), `duplicate`, `credentials_in_url`, `incorrect`, `password_too_short`, `password_too_long`, `password_contains_username`, `password_too_simple`, `password_reused`.

A kliensoldali tartalék, ha a választörzsben nincs `code`: 401 → `unauthenticated`, 403 → `forbidden`, 404 → `not_found`, 413 → `payload_too_large`, 429 → `too_many_attempts`, egyéb → `internal_error`.

### 5.4 Lapozás

- Query: `limit` (egész, 1 és a végpont maximuma között), `offset` (egész, 0–1 000 000). Hibás érték: 400 `invalid_query`.
- Válasz: `{ "items": [], "total": 0, "limit": 50, "offset": 0 }`.

| Végpont | Alap `limit` | Max `limit` |
|---|---|---|
| `GET /api/links` | 200 | 500 |
| `GET /api/admin/users` | 50 | 200 |
| `GET /api/admin/audit` | 50 | 200 |

A címkék, a megosztások, a munkamenetek és a felhasználókönyvtár lapozás nélküli `{"items": [...]}` választ ad (a felső korlátjuk kicsi). A felület oldalszáma (`?page=` az admin táblázatokban) `offset = (page - 1) × limit` alakban képződik.

### 5.5 Közös JSON-alakok

```ts
export type Role = 'admin' | 'developer' | 'customer';
export type Locale = 'hu' | 'en' | 'de';
export type TagColor = 'blue' | 'ochre' | 'green' | 'violet' | 'rose' | 'neutral' | 'contrast';
export type FetchStatus = 'pending' | 'ok' | 'failed' | 'blocked';
export type FetchError =
  | 'timeout' | 'dns' | 'connection' | 'tls' | 'http_status' | 'too_large'
  | 'not_html' | 'too_many_redirects' | 'blocked_address' | 'blocked_port';
export type ShareTargetType = 'all' | 'role' | 'user';

export type UserRef = { id: number; username: string; displayName: string };
export type DirectoryUser = UserRef & { role: Role };

export type Preferences = {
  locale: Locale;
  theme: 'dark' | 'light' | 'system';
  viewMode: 'list' | 'tiles';
  showIcon: boolean;
  showUrl: boolean;
  showTitle: boolean;
  showDescription: boolean;
  showArchived: boolean;
  sort: 'usage' | 'created' | 'title';
  tagMatch: 'any' | 'all';
  archiveAfterDays: number | null;       // null: the system default
  effectiveArchiveAfterDays: number;     // read-only; 0 = never
  systemArchiveAfterDays: number;        // read-only; the current defaultArchiveDays system setting
};

export type Me = {
  id: number;
  username: string;
  displayName: string;
  role: Role;
  mustChangePassword: boolean;
  totpEnabled: boolean;
  recoveryCodesRemaining: number;
  privacyConsentRequired: boolean;
  privacyNoticeVersion: number;
  permissions: { canShare: boolean; canListUsers: boolean; canAdmin: boolean };
  preferences: Preferences;
  sessionId: number;
  createdAt: string;
  lastLoginAt: string | null;
};

export type LinkItem = {
  id: number;
  url: string;
  host: string;
  title: string;                       // fetched or manually entered title, may be ''
  description: string;                 // short description, may be ''
  iconUrl: string | null;              // "/api/links/12/icon?v=3fa2c1d0" when an icon exists
  fetchStatus: FetchStatus;
  tagIds: number[];                    // the viewer's own tags only
  pinned: boolean;
  useCount: number;
  lastUsedAt: string | null;
  archivedAt: string | null;
  archiveReason: 'auto' | 'manual' | null;
  inactiveSince: string;               // max(baselineAt, lastUsedAt): inactivity counts from here
  archiveDueAt: string | null;         // inactiveSince + effective period; null when never, pinned or archived
  isOwner: boolean;
  owner: UserRef;
  shareCount: number;                  // non-zero only for the owner
  sharedVia: ShareTargetType | null;   // recipients only; priority user > role > all
  sharedAt: string | null;
  createdAt: string;
  updatedAt: string;
};

export type LinkDetail = LinkItem & {
  titleSource: 'none' | 'fetched' | 'manual'; // section 4.4
  tags: Tag[];                         // tagIds resolved
  fetchError: FetchError | null;
  fetchedAt: string | null;
  fetchAttempts: number;
  shares: Share[];                     // owner with share permission only, otherwise []
};

export type LinkState = Pick<
  LinkItem,
  'pinned' | 'useCount' | 'lastUsedAt' | 'archivedAt' | 'archiveReason' | 'inactiveSince' | 'archiveDueAt'
>;

export type Tag = {
  id: number;
  name: string;
  color: TagColor;
  linkCount: number;                   // visible links carrying this tag
  activeLinkCount: number;             // of which not archived
  createdAt: string;
  updatedAt: string;
};

export type Share = {
  id: number;
  targetType: ShareTargetType;
  targetRole: Role | null;
  targetUser: UserRef | null;
  createdAt: string;
};

export type Session = {
  id: number;
  createdAt: string;
  lastSeenAt: string;
  expiresAt: string;
  ip: string;
  userAgent: string;
  current: boolean;
};

export type AdminUser = {
  id: number;
  username: string;
  displayName: string;
  role: Role;
  active: boolean;
  mustChangePassword: boolean;
  totpEnabled: boolean;
  lockedUntil: string | null;
  lastLoginAt: string | null;
  createdAt: string;
  updatedAt: string;
  linkCount: number;
  sharedLinkCount: number;             // own links with at least one share (shown in the delete dialog)
  sessionCount: number;
};

export type SystemSettings = {
  defaultArchiveDays: number;          // 0–3650, default 90; 0 = never
  auditRetentionDays: number;          // 30–3650, default 180
  sessionIdleMinutes: number;          // 15–10080, default 720
  sessionAbsoluteHours: number;        // 1–2160, default 168
  allowPrivateFetch: boolean;          // default false
  privacyControllerName: string;       // 0–200 runes
  privacyControllerAddress: string;    // 0–300 runes
  privacyControllerEmail: string;      // 0–254, empty or e-mail shaped
  privacyDpoContact: string;           // 0–300 runes
  privacySupervisoryAuthority: string; // 0–300 runes; empty: the notice names the authority generically
  privacyNoticeVersion: number;        // read-only, bumped by the bump endpoint
  privacyNoticeUpdatedAt: string | null; // read-only
};

export type AuditTargetType = 'user' | 'link' | 'session' | 'settings' | 'tag';
export type AuditEntry = {
  id: number;
  at: string;
  event: string;
  success: boolean;
  actor: { id: number | null; username: string };
  target: { type: AuditTargetType; id: number | null; name: string } | null;
  ip: string;
  details: Record<string, unknown>;
};

export type Page<T> = { items: T[]; total: number; limit: number; offset: number };

export type PublicConfig = {
  appName: 'Kinex Mark';
  version: string;
  locales: Locale[];                   // ["hu","en","de"]
  defaultLocale: 'hu';
  tagColors: TagColor[];               // in the order of section 4.6
  passwordPolicy: { minLength: 12; maxLength: 128; minDistinct: 6 };
  privacy: {
    controllerName: string;
    controllerAddress: string;
    controllerEmail: string;
    dpoContact: string;
    supervisoryAuthority: string;
    noticeVersion: number;
    updatedAt: string | null;
    auditRetentionDays: number;
    sessionIdleMinutes: number;
    sessionAbsoluteHours: number;
  };
};

export type LoginResponse =
  | { status: 'ok'; user: Me }
  | { status: 'totp_required'; challenge: string; expiresAt: string };
export type TwoFactorSetup = { secret: string; otpauthUri: string; expiresAt: string };
export type RecoveryCodes = { recoveryCodes: string[] };
export type PrivacyStatus = {
  currentVersion: number;
  consentRequired: boolean;              // acknowledgement of the current notice is missing
  consents: { kind: 'privacy_notice'; version: number; acceptedAt: string }[];
  ownSharedLinkCount: number;            // own links with at least one share (account deletion dialog)
};
export type LinkPage = Page<LinkItem> & {
  counts: { pinned: number; archived: number }; // within the filtered set (section 4.8)
};
export type LinkQuery = {
  q?: string;
  tags?: number[];
  tagMatch?: 'any' | 'all';
  untagged?: boolean;
  scope?: 'all' | 'own' | 'shared';
  archived?: 'exclude' | 'include' | 'only';
  sort?: 'usage' | 'created' | 'title';
  limit?: number;
  offset?: number;
};
export type NewLink = { url: string; title?: string; description?: string; tagIds?: number[]; pinned?: boolean };
export type LinkPatch = { url?: string; title?: string; description?: string };
export type QuickCaptureResult = { link: LinkDetail; created: boolean };
export type ShareTarget = { targetType: ShareTargetType; targetRole?: Role; targetUserId?: number };
export type ImportResult = { total: number; created: number; duplicates: number; invalid: number };
export type NewAdminUser = { username: string; displayName?: string; role: Role; password?: string; locale?: Locale };
export type AdminUserPatch = {
  displayName?: string; role?: Role; active?: boolean;
  currentPassword?: string;              // the admin's own password; required when role or active is present
};
export type AdminStepUp = { currentPassword: string }; // reset-password, reset-2fa, delete
export type AdminUserCreated = { user: AdminUser; temporaryPassword: string | null };
export type AdminSettingsResponse = { settings: SystemSettings; defaults: SystemSettings; lockedByEnv: string[] };
export type AuditQuery = {
  event?: string; actorId?: number; success?: boolean; from?: string; to?: string; limit?: number; offset?: number;
};
```

A `LinkItem.inactiveSince` = `max(baseline_at, last_used_at)` a 4.8 lekérdezés `baseline_at` oszlopából; az `archiveDueAt` = `inactiveSince + effektív napok`, ha a link nincs kitűzve, nincs archiválva, és a néző effektív időszaka (`effectiveArchiveAfterDays`) nagyobb nullánál, különben `null`; az `iconUrl` = `/api/links/<id>/icon?v=<sha256 első 8 hexa karaktere>`, ha van ikon, különben `null`.

Felelősség: a store a nyers `BaselineAt`, `LastUsedAt` és `IconSHA256` értéket adja (3.3.4), a három származtatott mezőt kizárólag az `internal/api/dto.go` (B6) számolja minden `LinkItem`-, `LinkDetail`- és `LinkState`-válaszban: `toLinkItem(m model.LinkItem, archiveDays int) LinkItem`, hasonlóan `toLinkDetail(m model.LinkDetail, archiveDays int)`, `toLinkState(m model.LinkState, archiveDays int)` és `toLinkPage(p model.LinkPage, req model.LinkQuery, archiveDays int)`. Aktuális időt egyik sem kap, mert egyik származtatott mező sem függ tőle (a `now` paraméter holt kód lenne). Az `archiveDays` a néző `Preferences.EffectiveArchiveAfterDays` értéke, amelyet a linkkezelő kérésenként egyszer olvas (`Store.Preferences`). Teszt: `dto_test.go` (kitűzött, archivált, 0 nap, ikon nélkül, `lastUsedAt` későbbi a `baselineAt`-nél).

### 5.6 Nyilvános végpontok

| Metódus és út | Hozzáférés | Kérés | Siker | Hibák |
|---|---|---|---|---|
| `GET /healthz` (és `HEAD`) | nyilvános | – | 200 `{"status":"ok"}` | 500 `internal_error` (DB-ping 2 s) |
| `GET /api/public/config` | nyilvános | – | 200 `PublicConfig` | 500 |

A `/healthz` a TECHNOLOGY.md 4.3–4.5 szerint működik. A nem `/api/` útvonalak (SPA, statikus fájlok, `/login`, `/privacy`, `/images/...`) nyilvánosak; a felület a `GET /api/auth/me` 401-es válaszából dönti el, hogy a bejelentkező oldalra irányít.

### 5.7 Hitelesítés

| Metódus és út | Hozzáférés | Kérés | Siker | Hibák |
|---|---|---|---|---|
| `POST /api/auth/login` | nyilvános | `{username: string 1–64, password: string 1–1024}` | 200 `{status:"ok", user: Me}` + süti; vagy 200 `{status:"totp_required", challenge, expiresAt}` | 400 `validation_failed`; 401 `invalid_credentials`; 403 `account_disabled`; 429 `too_many_attempts` |
| `POST /api/auth/login/totp` | nyilvános | `{challenge: string, code?: string (6 számjegy), recoveryCode?: string}`, pontosan az egyik | 200 `{status:"ok", user: Me}` + süti | 400 `validation_failed`; 401 `challenge_invalid`; 401 `invalid_totp`; 429 `too_many_attempts` |
| `POST /api/auth/logout` | bárki | – | 204; a bemutatott token munkamenet-sora törölve (ha van), a süti törölve | – |
| `GET /api/auth/me` | K | – | 200 `Me` | 401 `unauthenticated` |
| `POST /api/auth/password` | K | `{currentPassword: string, newPassword: string}` | 204 | 400 `current_password_invalid`; 400 `validation_failed` (`newPassword`: jelszókódok); 429 |

A bejelentkezés lépései (`h_auth.go`):

1. Validálás; `usernameKey = strings.ToLower(strings.TrimSpace(username))`; korlátozási kulcsok: `uKey = auth.UserThrottleKey(keyring, usernameKey)` és `ipKey = auth.IPThrottleKey(kliens-IP)` (4.3). A `Credentials(usernameKey)` itt fut, hogy létező fióknál a `uKey` sora megkapja a `user_id`-t.
2. Foglalás egyetlen `ThrottleUpdate` hívással mindkét kulcsra (6.7): ha bármelyik `Check` szerint blokkolt, a tranzakció visszagördül, a válasz 429 (`Retry-After` a hosszabbik hátralévő idő), audit `auth.login.locked`; különben mindkét kulcson `Reserve` (a próbálkozás az ellenőrzés előtt hibaként számít). A zárolás ismeretlen felhasználónévre is ugyanígy működik, így a válasz nem árulja el a fiók létezését.
3. Ha nincs ilyen fiók: `Hasher.DummyVerify`, mindkét kulcson `Confirm`, 401 `invalid_credentials`, audit `auth.login.failure` `actor_id = NULL`, `actor_name = ''`, `details = {reason: "unknown_user"}`. A beírt név (amely gyakran elgépelt jelszó) sem az auditba, sem a korlátozási táblába nem kerül.
4. Jelszó-ellenőrzés; hibánál mindkét kulcson `Confirm`, 401, audit `auth.login.failure` `details.reason = "credentials"`, az `actor_id` és az `actor_name` a fiókrekordból (nem a bemenetből).
5. Helyes jelszó, inaktív fiók: mindkét kulcson `Refund`, 403 `account_disabled`, audit `auth.login.failure` `details.reason = "disabled"`.
6. Helyes jelszó: az IP-kulcson `Refund`. `needsRehash` esetén az újrahash-elt jelszó mentődik. Ha a TOTP nincs bekapcsolva, a felhasználói kulcs sora törlődik (`ClearThrottle(uKey)`); bekapcsolt TOTP mellett a felhasználói kulcs **nem** törlődik és nem kap `Refund`-ot, csak a második lépés sikere nullázza.
7. Ha a TOTP be van kapcsolva: kihívás (`NewToken`, 5 perc, legfeljebb 5 próbálkozás; a `CreateChallenge` a felhasználó többi nyitott kihívását törli, így felhasználónként egyetlen kihívás él), válasz `totp_required`. Audit nincs, amíg a második lépés el nem dől.
8. Különben munkamenet létrehozása (6.2; ha a kérés érvényes munkamenet-sütit hozott, annak sora előbb törlődik), `RecordLogin`, audit `auth.login.success`, válasz `ok` + `Me`.

TOTP-lépés (`POST /api/auth/login/totp`):

1. Foglalás a `ChallengeAttempt(hash, 5)` hívással (egyetlen atomikus `UPDATE … attempts = attempts + 1 … RETURNING`); ha nincs sor (lejárt, ismeretlen, elfogyott): 401 `challenge_invalid`.
2. A kihívás felhasználójának `uKey`-e és a kérés `ipKey`-e: `ThrottleUpdate` `Check` + `Reserve`; zárolt kulcsnál 429 `Retry-After`-rel (a kihívás megmarad, de a próbálkozás elhasználódott).
3. Ellenőrzés: TOTP-kód (6.5) vagy helyreállító kód (`UseRecoveryCode`).
4. Hiba: mindkét kulcson `Confirm`, audit `auth.login.failure` `details.reason = "totp"`, 401 `invalid_totp`; ha a kihívás próbálkozásai elfogytak, a kihívás törlődik, és a válasz 401 `challenge_invalid`.
5. Siker: a kihívás törlődik, a felhasználói kulcs sora törlődik (`ClearThrottle(uKey)`), az IP-kulcson `Refund`; helyreállító kódnál audit `auth.recovery_code.used`; munkamenet (a 8. lépés szerint), audit `auth.login.success` (`method`).

Ezzel a második tényező nem törhető: a jelszólépés a felhasználói számlálót nem nullázza, a TOTP-hibák a felhasználói és az IP-kulcson is számítanak, a zárolt kulcs a TOTP-lépést is tiltja, és párhuzamos kihívás nincs.

Kijelentkezés: a bemutatott süti tokenjének sora törlődik (`DeleteSessionByToken`; ha nincs ilyen sor, nem hiba), majd a süti törlődik (`Max-Age=-1`); audit `auth.logout`, ha volt érvényes munkamenet.

Jelszócsere: a jelenlegi jelszó ellenőrzése a felhasználói kulcson ugyanígy foglal (`Reserve`, hibánál `Confirm`, sikernél `Refund`). Siker: új hash, `must_change_password = 0`, `password_changed_at`, a többi munkamenet törlése (`DeleteUserSessions(uid, currentID)`), audit `auth.password.changed`. Minden más jelszóval vagy kóddal megerősített végpont (2FA-beállítás, -kikapcsolás, helyreállító kódok, fióktörlés, az admin érzékeny műveletei, 5.12) ugyanezt a foglalás–megerősítés–visszatérítés mintát követi a hívó felhasználói kulcsán.

### 5.8 Saját fiók: profil, beállítások, munkamenetek, 2FA, adatvédelem

| Metódus és út | Szint | Kérés | Siker | Hibák |
|---|---|---|---|---|
| `PATCH /api/me/profile` | B | `{displayName?: string 0–100}` | 200 `Me` | 400 |
| `GET /api/me/preferences` | K | – | 200 `Preferences` | |
| `PATCH /api/me/preferences` | K | `Partial<Preferences>` az `effectiveArchiveAfterDays` nélkül; `archiveAfterDays: number 0–3650 \| null` | 200 `Preferences` | 400 `validation_failed` (`out_of_range`, `invalid`) |
| `GET /api/me/sessions` | B | – | 200 `{items: Session[]}` (legújabb elöl) | |
| `DELETE /api/me/sessions/{id}` | B | – | 204; ha a saját, a süti is törlődik | 404 `not_found` (idegen vagy nem létező) |
| `POST /api/me/sessions/revoke-others` | B | – | 200 `{revoked: number}` | |
| `POST /api/me/2fa/setup` | B | `{password: string}` | 200 `TwoFactorSetup` | 400 `current_password_invalid`; 409 `totp_already_enabled`; 429 |
| `POST /api/me/2fa/enable` | B | `{code: string 6 számjegy}` | 200 `RecoveryCodes` (10 db, egyszer látható) | 401 `invalid_totp`; 409 `totp_setup_required`, `totp_already_enabled` |
| `POST /api/me/2fa/disable` | B | `{password: string, code: string}` (TOTP vagy helyreállító kód) | 204 | 400 `current_password_invalid`; 401 `invalid_totp`; 409 `totp_not_enabled`; 429 |
| `POST /api/me/2fa/recovery-codes` | B | `{password: string, code: string}` (csak TOTP) | 200 `RecoveryCodes` | mint fent |
| `GET /api/me/privacy` | K | – | 200 `PrivacyStatus` | |
| `POST /api/me/privacy/consent` | K | `{version: number}` | 204 | 409 `consent_version_mismatch` |
| `GET /api/me/export` | K | – | 200, `Content-Disposition: attachment; filename="kinex-mark-export-<username>-<YYYYMMDD>.json"`, törzs `UserExport` | |
| `POST /api/me/delete` | K | `{password: string, confirm: string, code?: string}` (`confirm` = a felhasználónév; `code` TOTP vagy helyreállító kód, bekapcsolt 2FA mellett kötelező) | 204, süti törölve | 400 `current_password_invalid`, `confirmation_mismatch`, `validation_failed` (`code`: `required`); 401 `invalid_totp`; 409 `last_admin`; 429 |

A szintek jelölése az 5.13 szerint (K: a kapuk alól mentes; B: kapukkal). A nyelvváltás így a kötelező lépések alatt is működik.

2FA-beállítás: a `setup` új titkot generál, lezárva a `totp_pending_secret` mezőbe teszi 10 perces lejárattal; a `secret` és az `otpauthUri` csak ebben a válaszban jelenik meg. A QR-kódot a felület rajzolja az URI-ból. Az `enable` a függő titokkal ellenőriz (`lastStep` itt 0); siker: `EnableTOTP` (a titok átmozgatása, `totp_last_step` = az elfogadott lépés, 10 helyreállító kód hash-e), a többi munkamenet törlése, audit `auth.2fa.enabled`. A `disable`: `DisableTOTP` (titkok és helyreállító kódok törlése), a többi munkamenet törlése, audit `auth.2fa.disabled`. Az újragenerálás auditja `auth.2fa.recovery_regenerated`.

#### 5.8.1 `UserExport`

```ts
export type UserExport = {
  format: 'kinex-mark-export/1';
  exportedAt: string;
  user: {
    id: number; username: string; displayName: string; role: Role; active: boolean;
    totpEnabled: boolean; createdAt: string; lastLoginAt: string | null; passwordChangedAt: string;
  };
  preferences: Preferences;
  consents: { kind: string; version: number; acceptedAt: string }[];
  sessions: Omit<Session, 'current'>[];
  tags: { id: number; name: string; color: TagColor; createdAt: string }[];
  links: {                                   // own links
    id: number; url: string; title: string; description: string; createdAt: string; updatedAt: string;
    tags: string[]; pinned: boolean; useCount: number; lastUsedAt: string | null; archivedAt: string | null;
    shares: { targetType: ShareTargetType; targetRole: Role | null; targetUsername: string | null; createdAt: string }[];
  }[];
  sharedWithMe: {                            // personal state on links owned by others
    linkId: number;
    url: string | null;                      // null: the link is no longer visible to the user (share revoked, owner demoted)
    tags: string[]; pinned: boolean; useCount: number;
    lastUsedAt: string | null; archivedAt: string | null;
  }[];
  auditEvents: {                             // the user is the actor, or the target user
    at: string; event: string; success: boolean; ip: string; role: 'actor' | 'target';
  }[];
  security: { lockedUntil: string | null };  // current sign-in lock of the account
};
```

- `sharedWithMe`: a felhasználó `link_user_state` sorai és saját címkéi más tulajdonos linkjein. Az URL csak a 4.7 láthatósági CTE szerint jelenleg látható linknél szerepel (`LEFT JOIN` a CTE-re); a már nem látható link sora `url: null`-lal, csak azonosítóval és a felhasználó saját állapotával kerül be. Így az export nem fedi fel a visszavont megosztás URL-jét (store-teszt: megosztás, személyes állapot, visszavonás, export → az URL nincs benne). A sorok maguk megmaradnak (a megosztás visszaállításakor vagy a tulajdonos visszaléptetésekor adatvesztés nélkül élednek újra), a fióktörlés kaszkádja törli őket.
- `auditEvents`: `actor_id = :uid OR (target_type = 'user' AND target_id = :uid)`; a saját fiókra érkező sikertelen bejelentkezések (5.7 4. lépés: `actor_id` a fiók) és az admin műveletei a fiókon (`user.password_reset`, `user.2fa_reset`, `user.deactivated`, `user.updated`, `user.sessions_revoked`, `user.unlocked`) is benne vannak. Az `ip` csak a felhasználó saját kérésének IP-címe (`CASE WHEN actor_id = :uid THEN ip ELSE '' END`): ahol a felhasználó csak célpont, ott `""`, mert a cselekvő admin IP-címe más személyes adata (15. cikk (4) bekezdés; D fázis, R3). A fiókra érkező sikertelen bejelentkezések `actor_id`-ja a fiók, ezért ezek megtartják az IP-címet. A `security.lockedUntil` a fiók `u:` sorából jön (`login_throttle.user_id`).
- Titkos adat (jelszóhash, TOTP-titok, helyreállító kód, munkamenet-token) nem kerül az exportba, az ikonok bináris tartalma sem. Audit: `gdpr.export`.

Fióktörlés: jelszó és megerősítés ellenőrzése, bekapcsolt 2FA mellett a `code` ellenőrzése is (TOTP a 6.5 szerint vagy helyreállító kód), `last_admin` védelem, majd `DeleteUser(uid, audit user.self_deleted)` (6.11: az auditsor a tranzakcióban, a pszeudonimizálás előtt íródik), süti törlése.

### 5.9 Linkek

Hozzáférés: minden bejelentkezett szerepkör (B); a megosztási végpontok csak `admin` és `developer` (D).

#### 5.9.1 `GET /api/links`

| Paraméter | Típus | Alap | Szabály |
|---|---|---|---|
| `q` | string | – | 0–200 rúna |
| `tags` | string | – | vesszővel tagolt azonosítók, legfeljebb 20, mind pozitív egész |
| `tagMatch` | `any` \| `all` | `any` | |
| `untagged` | `true` \| `false` | `false` | `tags` mellett nem adható meg (`invalid_query`) |
| `scope` | `all` \| `own` \| `shared` | `all` | |
| `archived` | `exclude` \| `include` \| `only` | `exclude` | |
| `sort` | `usage` \| `created` \| `title` | `usage` | |
| `limit`, `offset` | 5.4 | 200, 0 | |

Válasz: 200 `LinkPage` (`Page<LinkItem>` + `counts: {pinned, archived}` a szűrt halmazra, 4.8). Hiba: 400 `invalid_query`. Az oldalsáv „Linkek” számlálója ugyanezt a végpontot hívja `limit=1` és `archived=exclude` mellett (a `total` a nem archivált linkek száma, 8.6 `linkStats`). Idegen vagy nem létező címkeazonosító nem hiba, egyszerűen nincs találat. A lekérdezés: 4.8.

#### 5.9.2 `POST /api/links`

Kérés: `NewLink`.

- `url`: kötelező, `fetch.NormalizeURL` (5.9.10); mezőkódok `required`, `too_long`, `invalid_format`, `not_allowed` (nem http/https), `credentials_in_url`.
- `title`: 0–500 rúna; nem üres érték esetén `title_source = 'manual'`, és a letöltés soha nem írja felül; üresen `title_source = 'none'` (4.4, 7.3).
- `description`: 0–500 rúna, sortörés megengedett.
- `tagIds`: legfeljebb 20, egyedi, mind a felhasználó saját címkéje (`unknown`).
- `pinned`: alap `false`.

Siker: 201 `LinkDetail` (`fetchStatus: "pending"`; az `INSERT` `fetch_attempts = 0`, `fetch_error = NULL`, `next_fetch_at = now` értékkel, 4.4), majd a szerver a letöltési sorba teszi (`Enqueue`; ha ez elveszik, a seprő veszi fel). Hibák: 400 `validation_failed`; 409 `link_duplicate` (`details.linkId`); 409 `link_limit_reached`.

#### 5.9.3 `POST /api/links/quick`

Gyors rögzítés ráhúzásból vagy beillesztésből. Kérés: `{text: string 1–8192, title?: string 0–500}`. A felület a `DataTransfer`-ből a `text/uri-list`, ennek hiányában a `text/plain` tartalmat küldi; a `title` a ráhúzott HTML-horgony szövege, ha van.

`fetch.ExtractURL`: soronként halad, a `#`-tel kezdődő sorokat (uri-list megjegyzés) kihagyja; az első `https?://` kezdetű, szóközig tartó tokent veszi; a záró `.,;:!?)]}>'"` karaktereket levágja (a zárójelet csak akkor, ha a tokenben nincs nyitó párja). Ha nincs ilyen token, és az első token domain-alakú (`^[\p{L}\p{N}.-]+\.[\p{L}]{2,63}(:\d+)?(/\S*)?$`), `https://` előtagot kap.

Siker: 201 `QuickCaptureResult` `created: true`-val (a létrehozás az 5.9.2 szerint: `pending`, `next_fetch_at = now`, sorba tétel; a `title` megadva `manual`); ha a tulajdonosnak már van ilyen URL-je, 200 `created: false` (a felület kiemeli a meglévőt). Hibák: 400 `no_url_found`; 400 `validation_failed` (a kinyert URL-re); 409 `link_limit_reached`.

#### 5.9.4 Egy link

| | Hozzáférés | Kérés | Siker | Hibák |
|---|---|---|---|---|
| `GET /api/links/{id}` | látható link | – | 200 `LinkDetail` | 404 `not_found` |
| `PATCH /api/links/{id}` | tulajdonos | `LinkPatch` | 200 `LinkDetail` | 400; 403 `not_owner`; 404; 409 `link_duplicate` |
| `DELETE /api/links/{id}` | tulajdonos | – | 204 | 403 `not_owner`; 404 |

A `PATCH` `title` mezője: nem üres érték `title_source = 'manual'`, üres érték `title = ''`, `title_source = 'none'`, és ha előtte `manual` volt, a link letöltésre jelölődik (`pending`, `next_fetch_at = now`, `Enqueue`), így a letöltött cím jelenik meg. URL-módosításkor ugyanabban az utasításban: a `host`, a `search_text` és a `sort_label` újraszámolódik; az `icon_id = NULL`; `fetched` cím esetén `title = ''` és `title_source = 'none'` (a `manual` cím megmarad); `fetch_status = 'pending'`, `fetch_error = NULL`, `fetch_attempts = 0`, `next_fetch_at = now`, `fetched_at = NULL`; utána `Enqueue`. Így a régi webhely címe és ikonja nem marad a linken. A törlés kaszkádol (címkekapcsolatok, megosztások, minden felhasználó személyes állapota).

#### 5.9.5 Személyes állapot

| Metódus és út | Hozzáférés | Kérés | Siker |
|---|---|---|---|
| `PUT /api/links/{id}/tags` | látható link | `{tagIds: number[]}` (0–20, saját címkék) | 200 `LinkItem` |
| `POST /api/links/{id}/pin` | látható link | – | 200 `LinkItem` |
| `DELETE /api/links/{id}/pin` | látható link | – | 200 `LinkItem` |
| `POST /api/links/{id}/archive` | látható link | – | 200 `LinkItem` |
| `DELETE /api/links/{id}/archive` | látható link | – | 200 `LinkItem` (visszaállítás) |
| `POST /api/links/{id}/visit` | látható link | – | 200 `LinkState` |

Hibák: 400 `validation_failed` (`tagIds`: `unknown`, `too_long`); 404 `not_found`. A `pin`, `archive`, `tags` műveletek idempotensek; a `visit` minden hívása egy használatot rögzít. Archivált link kitűzése megszünteti az archiválást (4.9), ezért a kitűzés archivált linken is elérhető.

Használatnak számít (a `visit` végpontot hívja, így a rangsor és az automatikus archiválás is erre épül, 7.2):

- a megnyitás: kizárólag a horgony `click` eseménye (bal kattintás és a fókuszált horgonyon lenyomott Enter, mert a böngésző a billentyűs aktiválásra is `click`-et küld) és az `auxclick` `button === 1` esetén (középső kattintás). A horgonyon külön Enter-kezelő nincs, így egy megnyitás pontosan egy látogatás;
- a másolás: „URL másolása”, „Másolás Markdownként”, a `c` és `m` billentyű, valamint a részletek lap URL melletti `CopyButton`-ja. A másolt link beillesztése egy dokumentumba ugyanúgy a link használata, mint a megnyitása; enélkül a csak másolással használt link a lista végére kerülne és archiválódna.

A felület a látogatást `fetch(..., {method: 'POST', keepalive: true})` hívással küldi, mert a `sendBeacon` nem küld egyedi fejlécet. A link `<a href target="_blank" rel="noopener noreferrer">`, így a megnyitott oldal nem kap hivatkozót.

#### 5.9.6 Megosztás

| Metódus és út | Hozzáférés | Kérés | Siker | Hibák |
|---|---|---|---|---|
| `GET /api/links/{id}/shares` | D, tulajdonos | – | 200 `{items: Share[]}` | 403 `share_not_allowed`, `not_owner`; 404 |
| `POST /api/links/{id}/shares` | D, tulajdonos | `ShareTarget` | 201 `Share` | 400 `validation_failed` (`targetRole`/`targetUserId`: `required`, `not_allowed`, `unknown`); 403; 404; 409 `share_duplicate` |
| `DELETE /api/links/{id}/shares/{shareId}` | D, tulajdonos | – | 204 | 403; 404 (a `shareId` nem ehhez a `{id}`-hez tartozik, vagy a link nem a hívóé: 4.7 `DeleteShare`) |

- `role` esetén a `targetRole` kötelező, a `targetUserId` tilos; `user` esetén fordítva; `all` esetén mindkettő tilos.
- `targetUserId`: létező, aktív felhasználó, nem maga a tulajdonos.
- A megosztás után a címzett a linket a saját listájában látja (`sharedVia`, `sharedAt`), személyes állapottal és saját címkékkel.
- Audit: `link.shared`, `link.unshared` (`target_type = 'link'`, `details: {targetType, targetRole, targetUserId}`).

#### 5.9.7 Metaadat-frissítés

`POST /api/links/{id}/refresh` (tulajdonos): `fetch_attempts = 0`, `fetch_status = 'pending'`, `fetch_error = NULL`, `next_fetch_at = now`, sorba tétel. A sikeres újraletöltés a `fetched` vagy `none` eredetű címet és az ikont felülírja (a `manual` címet nem, 7.3), így a webhely megváltozott címe és ikonja frissül. Válasz: 202 `LinkDetail`. Folyamatban lévő letöltésnél ugyanez a válasz (idempotens). Hibák: 403 `not_owner`; 404.

#### 5.9.8 Aszinkron metaadat és a felület

A link mindig azonnal mentődik, a cím és az ikon később érkezik. `fetchStatus: "pending"` esetén a felület 1,5 másodpercenként lekéri a `GET /api/links/{id}`-t legfeljebb 20 másodpercig; utána a lista következő frissítésekor látszik az eredmény. A cím írásának szabálya a `title_source` (4.4, 7.3): a felhasználó által megadott címet és leírást a letöltés soha nem írja.

#### 5.9.9 `GET /api/links/{id}/icon`

Hozzáférés: látható link. Siker: 200 a tárolt `Content-Type`-pal és a fejlécekkel: `Cache-Control: private, max-age=604800`, `ETag: "<sha256>"` (`If-None-Match` esetén 304), `X-Content-Type-Options: nosniff`, `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; sandbox`, `Content-Disposition: inline; filename="icon"`. A `?v=` paraméter csak gyorsítótár-törő (az `iconUrl` az ikon SHA-256-jának első 8 hexa karakterét tartalmazza), a szerver nem vizsgálja. Hiba: 404 `not_found` (nincs ikon, vagy nem látható a link).

#### 5.9.10 URL-normalizálás (`fetch.NormalizeURL`)

1. `strings.TrimSpace`; üres: `required`; 2048 bájt fölött: `too_long`.
2. `url.Parse`; hiba vagy üres host: `invalid_format`.
3. A séma kisbetűsítve; nem `http`/`https`: `not_allowed`.
4. Nem üres `User`: `credentials_in_url`.
5. A host kisbetűsítve; az alapértelmezett port (`:80` http, `:443` https) elhagyva; üres út helyett `/`.
6. A query és a fragment változatlan.
7. Eredmény: `u.String()`; a `host` a port nélküli hostnév.
8. A végső, normalizált szövegre a 2048 bájtos korlát újra érvényes (az újraszerializálás százalékos kódolással hosszabbíthat): fölötte `too_long`, így a `links.url` `CHECK`-je soha nem sérül (táblateszt: 2040 bájtos, szóközöket és ékezetes útrészt tartalmazó bemenet).

#### 5.9.11 Export és import

| Metódus és út | Hozzáférés | Kérés | Siker | Hibák |
|---|---|---|---|---|
| `GET /api/links/export?format=json\|html` | B | – | 200, `Content-Disposition: attachment; filename="kinex-mark-links-<YYYYMMDD>.json"` vagy `.html`; csak a saját linkek a saját címkékkel | 400 `invalid_query` |
| `POST /api/links/import` | B | nyers törzs, `Content-Type: application/json` (`kinex-mark-links/1`) vagy `text/html` (Netscape), legfeljebb 2 MiB és 5000 bejegyzés | 200 `ImportResult` | 400 `import_invalid`; 400 `import_too_many_entries` (5000 bejegyzés fölött); 413 `payload_too_large`; 409 `link_limit_reached` (ha az import átlépné a korlátot, semmi nem jön létre) |

Az import egy tranzakcióban fut: az érvénytelen URL-ű bejegyzés kimarad (`invalid`), a meglévő URL kimarad (`duplicates`). Az importált cím és leírás a `fetch.CleanText`-en megy át (vezérlő- és bidi-karakterek nélkül, 6.8) és a mezőkorlátra vágódik; a nem üres importált cím `title_source = 'manual'`. A hiányzó címkék `neutral` színnel jönnek létre (az 500-as korláton belül; a korlát fölötti címkék elmaradnak). Címkenevek bejegyzésenként (D fázis, R2 és R6):

- az elemző legfeljebb 100 egyedi nevet tart meg (`bookmarks.MaxTagsPerEntry`, a `MaxTagsPerLink` ötszöröse, mert a tisztítás után nevek összeolvadhatnak vagy kiürülhetnek); a többi szó nélkül elmarad, az első nevek nyernek (a `TAGS` attribútum a mappanév előtt);
- a duplikátumszűrés halmazzal történik (lineáris költség a 2 MiB-os törzsön belül);
- a store linkenként legfeljebb 20 címkét köt (`model.MaxTagsPerLink`).

Létrehozási idő: az 1970-01-01T00:00:00Z előtti `addedAt` (JSON) és a nem pozitív `ADD_DATE` (Netscape) hiányzónak számít, ahogy a jövőbeli is; ilyenkor `created_at` = az import ideje. A store (`importCreatedAt`) is csak a [1970, most) tartományt fogadja el. Az új linkek `pending` állapotúak, `next_fetch_at` = most; közvetlenül nem kerülnek a sorba, a seprő (7.3) fokozatosan dolgozza fel őket.

### 5.10 Címkék

| Metódus és út | Kérés | Siker | Hibák |
|---|---|---|---|
| `GET /api/tags` | – | 200 `{items: Tag[]}`, név szerint (`name_key`) | |
| `POST /api/tags` | `{name: string 1–40, color?: TagColor}` (alap `neutral`) | 201 `Tag` | 400; 409 `tag_duplicate` (`details.tagId`), `tag_limit_reached` |
| `PATCH /api/tags/{id}` | `{name?: string, color?: TagColor}` | 200 `Tag` | 400; 404; 409 `tag_duplicate` |
| `DELETE /api/tags/{id}` | – | 204 (a linkekről is lekerül) | 404 |

Hozzáférés: minden bejelentkezett szerepkör (B), csak saját címkék. A `linkCount` és az `activeLinkCount` a 4.7 CTE-vel csak a jelenleg látható linkeket számolja.

### 5.11 Felhasználókönyvtár

`GET /api/users/directory?q=<0–64 rúna>`. Hozzáférés: `admin`, `developer` (ügyfélnek 403 `forbidden`). Válasz: 200 `{items: DirectoryUser[]}`, csak aktív felhasználók, a kérő nélkül, felhasználónév szerint, legfeljebb 50. A `q` a `username_key`-ben és a kisbetűsített `display_name`-ben keres.

### 5.12 Adminfelület

Hozzáférés: csak `admin` (A).

| Metódus és út | Kérés | Siker | Hibák |
|---|---|---|---|
| `GET /api/admin/users?q=&role=&status=active\|inactive\|all&limit=&offset=` | – | 200 `Page<AdminUser>` (felhasználónév szerint) | 400 `invalid_query` |
| `POST /api/admin/users` | `NewAdminUser` | 201 `AdminUserCreated` | 400; 409 `username_taken` |
| `GET /api/admin/users/{id}` | – | 200 `AdminUser` | 404 |
| `PATCH /api/admin/users/{id}` | `AdminUserPatch` (`role` vagy `active` mellett `currentPassword` kötelező) | 200 `AdminUser` | 400 `validation_failed` (`currentPassword`: `required`), `current_password_invalid`; 404; 409 `self_modification`, `last_admin`; 429 |
| `POST /api/admin/users/{id}/reset-password` | `{currentPassword: string, password?: string}` | 200 `{temporaryPassword: string \| null}` | 400 (`current_password_invalid` is); 404; 429 |
| `POST /api/admin/users/{id}/reset-2fa` | `AdminStepUp` | 204 (a cél minden munkamenete törölve) | 400 `current_password_invalid`; 404; 409 `totp_not_enabled`; 429 |
| `POST /api/admin/users/{id}/unlock` | – | 204 | 404 |
| `DELETE /api/admin/users/{id}/sessions` | – | 200 `{revoked: number}` | 404 |
| `DELETE /api/admin/users/{id}` | `AdminStepUp` (JSON-törzs) | 204 | 400 `current_password_invalid`; 404; 409 `self_modification`, `last_admin`; 429 |
| `GET /api/admin/settings` | – | 200 `AdminSettingsResponse` | |
| `PATCH /api/admin/settings` | `Partial<SystemSettings>` a csak olvasható mezők nélkül | 200 `AdminSettingsResponse` | 400; 409 `setting_locked` |
| `POST /api/admin/settings/privacy-notice/bump` | – | 200 `AdminSettingsResponse` (`privacyNoticeVersion` + 1, `privacyNoticeUpdatedAt` = most) | |
| `GET /api/admin/audit?event=&actorId=&success=&from=&to=&limit=&offset=` | – | 200 `Page<AuditEntry>` (legújabb elöl) | 400 `invalid_query` |
| `POST /api/admin/jobs/archive` | – | 200 `{archived: number}` (az archiváló feladat azonnali futtatása) | |

Szabályok:

- Felhasználónév: 3–64 karakter, `^[A-Za-z0-9._-]+$`, kis- és nagybetűtől függetlenül egyedi.
- `locale` (új felhasználó): a `user_preferences.locale` kezdőértéke; hiányában `hu`.
- Megerősítés (step-up): a felhasználótörlés, a jelszó- és a 2FA-visszaállítás, valamint a szerepkör- és az aktív állapot módosítása az admin saját jelszavát kéri (`currentPassword`). Az ellenőrzés az admin felhasználói korlátozási kulcsán foglal (5.7 mintája); hibás jelszó: 400 `current_password_invalid` (`fields.currentPassword = "incorrect"`).
- Új felhasználó és jelszó-visszaállítás: ha nincs `password`, a szerver 16 karakteres jelszót generál (`auth.GeneratePassword`) és egyszer visszaadja; ha van, a jelszószabály érvényes (`fields.password`). Mindkét esetben `must_change_password = 1`. A visszaállítás törli a felhasználó összes munkamenetét és korlátozási sorát (`ClearUserThrottle`). Az admin jelszó-visszaállítása az egyetlen út, amelyen egy admin más fiókjába beléphet; ezért auditált (`user.password_reset`), a felhasználó a következő belépéskor kötelezően jelszót cserél, és az adatkezelési tájékoztató és a kézikönyv is kimondja (8.7.10, 13.2).
- 2FA-visszaállítás: `DisableTOTP`, a cél összes munkamenete törlődik, audit `user.2fa_reset`.
- Inaktiválás (`active: false`): az összes munkamenet törlődik, audit `user.deactivated`; aktiválás: `user.activated`. Szerepkörváltás: a munkamenetek megmaradnak (a jogosultságot minden kérés frissen olvassa), audit `user.updated` `details: {changed: ["role"], role: {from, to}}`.
- Felhasználótörlés: `DeleteUser(id, audit user.deleted)`; az auditsor a tranzakcióban, a pszeudonimizálás előtt íródik, így a `target_name` is álnevet kap (6.11). Az `AdminUser.sharedLinkCount` megmutatja a dialógusban, hány megosztott link tűnik el a címzetteknél.
- `self_modification`: az admin a saját `role` és `active` mezőjét nem módosíthatja, és magát itt nem törölheti (erre a `POST /api/me/delete` való).
- `last_admin`: az utolsó aktív admin lefokozása, inaktiválása és törlése tilos.
- Az audit `event` szűrője pontos egyezés vagy `*` végű előtag (`auth.*`). `from`/`to`: RFC 3339. `success`: `true`/`false`.
- `PATCH /api/admin/settings`: tartományon kívüli érték `validation_failed` `out_of_range`; a szöveges mezők rúnakorlátja (`privacySupervisoryAuthority` 0–300) `too_long`; a `privacyControllerEmail` hibás alakja `invalid_format`; a `lockedByEnv`-ben szereplő kulcs módosítása 409 `setting_locked` (`details.keys`). Siker: audit `settings.updated`. A bump auditja `privacy.notice_bumped`.
- Az adminfelület nem lát más felhasználó linkjeit, csak a darabszámot (`linkCount`).

### 5.13 Útvonaltábla és jogosultsági mátrix

Jelölés: N = nyilvános; K = bejelentkezett, a jelszócsere- és a hozzájárulás-kapu alól mentes; B = bármely bejelentkezett, kapukkal; D = `admin` és `developer`, kapukkal; A = csak `admin`, kapukkal. A link szintű tulajdonosi szabályokat a store érvényesíti.

| Útvonal | Szint | admin | developer | customer | anonim |
|---|---|---|---|---|---|
| `GET /healthz`, `GET /api/public/config` | N | 200 | 200 | 200 | 200 |
| `POST /api/auth/login`, `POST /api/auth/login/totp` | N | | | | 200/401 |
| `POST /api/auth/logout` | N | 204 | 204 | 204 | 204 |
| `GET /api/auth/me`, `POST /api/auth/password` | K | ✓ | ✓ | ✓ | 401 |
| `GET/PATCH /api/me/preferences`, `GET /api/me/privacy`, `POST /api/me/privacy/consent`, `GET /api/me/export`, `POST /api/me/delete` | K | ✓ | ✓ | ✓ | 401 |
| `PATCH /api/me/profile`, `/api/me/sessions*`, `/api/me/2fa/*` | B | ✓ | ✓ | ✓ | 401 |
| `/api/links*` (a megosztás kivételével), `/api/tags*` | B | saját + neki megosztott | saját + neki megosztott | saját + neki megosztott | 401 |
| `/api/links/{id}/shares*` | D | saját link | saját link | 403 `share_not_allowed` | 401 |
| `GET /api/users/directory` | D | ✓ | ✓ | 403 `forbidden` | 401 |
| `/api/admin/*` | A | ✓ | 403 `forbidden` | 403 `forbidden` | 401 |

A `routes.go` minden sora ezt az adatot hordozza (`access: accessPublic | accessExempt | accessUser | accessSharer | accessAdmin`, valamint a D szint elutasító kódja: `share_not_allowed` vagy `forbidden`), és a `routes_test.go` a táblát bejárva minden útvonalra és minden szerepkörre ellenőrzi a 401/403 határt.

### 5.14 A frontend API-modulok és a végpontok megfeleltetése

Minden modul a `client.ts` `request` függvényének vékony burkolója; mindegyik az F0 tulajdona (11.4).

| Modul | Függvény | Végpont |
|---|---|---|
| `public.ts` | `getPublicConfig()` | `GET /api/public/config` |
| | `getHealth()` | `GET /healthz` |
| `auth.ts` | `login(username, password)` | `POST /api/auth/login` |
| | `loginTotp({challenge, code} \| {challenge, recoveryCode})` | `POST /api/auth/login/totp` |
| | `logout()` | `POST /api/auth/logout` |
| | `getMe()` | `GET /api/auth/me` |
| | `changePassword(current, next)` | `POST /api/auth/password` |
| `me.ts` | `updateProfile(patch)` | `PATCH /api/me/profile` |
| | `getPreferences()`, `updatePreferences(patch)` | `GET`, `PATCH /api/me/preferences` |
| | `listSessions()`, `revokeSession(id)`, `revokeOtherSessions()` | `GET /api/me/sessions`, `DELETE /api/me/sessions/{id}`, `POST /api/me/sessions/revoke-others` |
| | `setupTwoFactor(password)`, `enableTwoFactor(code)`, `disableTwoFactor(password, code)`, `regenerateRecoveryCodes(password, code)` | `POST /api/me/2fa/setup`, `/enable`, `/disable`, `/recovery-codes` |
| | `getPrivacy()`, `acceptPrivacy(version)` | `GET /api/me/privacy`, `POST /api/me/privacy/consent` |
| | `exportMyData()` | `GET /api/me/export` (`download`) |
| | `deleteAccount(password, confirm, code?)` | `POST /api/me/delete` |
| `links.ts` | `listLinks(query, signal)` → `LinkPage` | `GET /api/links` |
| | `createLink(input)`, `quickCapture(text, title?)` | `POST /api/links`, `POST /api/links/quick` |
| | `getLink(id)`, `updateLink(id, patch)`, `deleteLink(id)` | `GET`, `PATCH`, `DELETE /api/links/{id}` |
| | `setLinkTags(id, tagIds)` | `PUT /api/links/{id}/tags` |
| | `pin(id)`, `unpin(id)`, `archive(id)`, `restore(id)` | `POST`/`DELETE /api/links/{id}/pin`, `/archive` |
| | `recordVisit(id)` (`keepalive: true`) | `POST /api/links/{id}/visit` |
| | `refreshMetadata(id)` | `POST /api/links/{id}/refresh` |
| | `listShares(id)`, `createShare(id, target)`, `deleteShare(id, shareId)` | `/api/links/{id}/shares*` |
| | `exportLinks(format)`, `importLinks(file)` | `GET /api/links/export`, `POST /api/links/import` |
| | `iconUrl` mező | `GET /api/links/{id}/icon` (`<img src>`) |
| `tags.ts` | `listTags()`, `createTag(name, color)`, `updateTag(id, patch)`, `deleteTag(id)` | `/api/tags*` |
| `directory.ts` | `searchDirectory(q, signal)` | `GET /api/users/directory` |
| `admin.ts` | `listUsers(query)`, `createUser(input)`, `getUser(id)`, `updateUser(id, patch)` (a `patch` a `currentPassword`-del) | `/api/admin/users*` |
| | `resetPassword(id, currentPassword, password?)`, `resetTwoFactor(id, currentPassword)`, `unlockUser(id)`, `revokeUserSessions(id)`, `deleteUser(id, currentPassword)` | `/api/admin/users/{id}/…` |
| | `getSettings()`, `updateSettings(patch)`, `bumpPrivacyNotice()` | `/api/admin/settings*` |
| | `listAudit(query)` | `GET /api/admin/audit` |
| | `runArchiveJob()` | `POST /api/admin/jobs/archive` |

A `listLinks` a `LinkQuery`-t az 5.9.1 paramétereire képezi: `tags` vesszővel tagolva, üres értékek kihagyva, `untagged` csak `true`-ként.

---

## 6. Biztonság és GDPR

### 6.1 Middleware-lánc

Globális sorrend kívülről befelé: `recoverer` → `securityHeaders` → `accessLog` → `hostCheck` (csak `/api/`, 6.3) → `csrf` (csak `/api/` és nem biztonságos metódus) → `ServeMux`. Útvonalanként: `authenticate(access)` → kezelő. A ServeMux-minták a TECHNOLOGY.md 4.4 szerintiek: metódusos minták, `/api/` catch-all, `/` SPA-kezelő.

- `recoverer`: elkapja a pánikot, `log.Error("panic", "err", v, "stack", ...)`, 500 `internal_error`.
- `accessLog`: Debug szinten `method`, a ServeMux által illesztett minta (`r.Pattern`), státusz, időtartam. Query-string nem kerül naplóba (a `q` személyes lehet).

### 6.2 Munkamenet és süti

- Token: 32 véletlen bájt (`crypto/rand`), base64url padding nélkül (43 karakter). Az adatbázisban csak a SHA-256 hash van.
- Süti neve: biztonságos módban `__Host-kxm_session`, egyébként `kxm_session`. Attribútumok: `Path=/`, `HttpOnly`, `SameSite=Strict`, biztonságos módban `Secure`, `Max-Age` = az abszolút lejáratig hátralévő másodperc. `Domain` nincs.
- Biztonságos mód (`APP_SECURE_COOKIES`): `true` mindig; `false` soha; `auto` (alap) esetén akkor, ha a szerver maga TLS-sel fut, vagy az `APP_BASE_URL` `https`, vagy a kérés megbízható proxyn át `X-Forwarded-Proto: https` fejléccel érkezett.
- Reverse proxy mögött az `APP_BASE_URL` és az `APP_TRUSTED_PROXY` kötelező (3.4, 10.9, README). Ha egy kérés nem megbízható forrásból `X-Forwarded-For`, `X-Forwarded-Proto` vagy `X-Forwarded-Host` fejlécet hoz, a fejléceket a szerver figyelmen kívül hagyja, és futásonként egyszer Warn naplót ír: `request has X-Forwarded-* headers from an untrusted peer; set APP_TRUSTED_PROXY and APP_BASE_URL` (`peer` mezővel, a fejlécek értéke nélkül).
- Lejárat: tétlenségi (`sessionIdleMinutes`, alap 720 perc) és abszolút (`sessionAbsoluteHours`, alap 168 óra). Az `authenticate` mindkettőt ellenőrzi; a `last_seen_at` legfeljebb percenként egyszer frissül.
- Az `authenticate` egy lekérdezéssel tölti a munkamenetet és a felhasználót. Érvénytelen vagy lejárt munkamenet, illetve inaktív felhasználó esetén a sor és a süti törlődik (`Max-Age=-1`), a válasz 401 `unauthenticated`. A szerepkör és a jogosultság így minden kérésnél friss.
- Rotáció: minden sikeres bejelentkezés új tokent kap, és ha a kérés érvényes munkamenet-sütit hozott, annak sora előbb törlődik. Kijelentkezés: a bemutatott token sora törlődik (`DeleteSessionByToken`), utána a süti; a kimásolt token így azonnal érvénytelen. Jelszócsere, 2FA be- és kikapcsolás: a többi munkamenet törlődik. Admin jelszó-visszaállítás, 2FA-visszaállítás és inaktiválás: mind törlődik.
- Kapuk (`requireReady`, minden B, D és A szintű útvonalon): ha `must_change_password`, 403 `password_change_required`; különben, ha az aktuális tájékoztató-verzióhoz nincs hozzájárulás, 403 `consent_required`. A K szintű útvonalak mentesek.
- A `SameSite=Strict` a SPA-t nem zavarja: külső hivatkozásról érkezve a felső szintű navigáció süti nélkül tölti az `index.html`-t (nyilvános), a felület ezt követő, azonos originű API-hívásai már viszik a sütit.

### 6.3 CSRF

Minden `/api/` alatti `POST`, `PUT`, `PATCH`, `DELETE` kérésre, a bejelentkezést is beleértve:

1. Kötelező fejléc: `X-Kinex-Request: 1`, hiányában 403 `csrf_rejected`. Egyszerű HTML-űrlap és `sendBeacon` ilyen fejlécet nem tud küldeni, más origin pedig CORS-preflight nélkül nem; preflightra a szerver nem ad engedélyt.
2. Ha van `Origin` fejléc, egyeznie kell az elvárt originnel; ha nincs, de van `Sec-Fetch-Site`, annak értéke `same-origin` vagy `none` lehet. Eltérés: 403 `csrf_rejected`. Ha mindkettő hiányzik (nem böngésző kliens), az 1. pont elég.
3. Elvárt origin: az `APP_BASE_URL` originje, ha meg van adva; különben `<séma>://<Host>`, ahol a séma TLS esetén `https`, megbízható proxynál az `X-Forwarded-Proto`, egyébként `http`; a host az `r.Host` (megbízható proxynál az `X-Forwarded-Host`, ha van).

Host-ellenőrzés (`hostCheck`, minden `/api/` kérésre, metódustól függetlenül; a `/healthz` és a statikus fájlok kivételek, mert a `-healthcheck` a `127.0.0.1`-et hívja):

- Ha az `APP_BASE_URL` meg van adva: a kérés hostja (`r.Host`, megbízható proxynál az `X-Forwarded-Host`) portostul egyezzen az `APP_BASE_URL` hostjával (az alapértelmezett port elhagyható), különben 421 `host_mismatch`. Ez a DNS-rebinding ellen is véd, mert az idegen név nem egyezik.
- Ha nincs megadva, és a figyelési cím loopback (`127.0.0.1`, `::1`, `localhost`): csak `localhost`, `127.0.0.1` és `[::1]` host fogadható el (tetszőleges porttal), különben 421 `host_mismatch`. A Vite fejlesztői proxyja (`localhost:5173`) és az E2E (`127.0.0.1:18080`) így működik.
- Ha nincs megadva, és a cím nem loopback (például konténerben `:8080`): nincs Host-ellenőrzés; ennek kockázatát (DNS-rebinding a belső hálózatból) a README és a `compose.yaml` kommentje kimondja, és az `APP_BASE_URL` beállítását kéri.

A Vite fejlesztői proxyja (`'/api'` és `'/healthz'` → `http://127.0.0.1:8080`, `changeOrigin` nélkül) a `Host: localhost:5173` fejlécet továbbítja, így az Origin-ellenőrzés fejlesztés közben is egyezik. A `changeOrigin: true` tilos.

### 6.4 Jelszavak

- Hash: PBKDF2-HMAC-SHA256 (`crypto/pbkdf2.Key(sha256.New, password, salt, iter, 32)`), 600 000 iteráció, felhasználónként 16 bájt véletlen só.
- Tárolt alak: `pbkdf2-sha256$600000$<base64 raw std só>$<base64 raw std kulcs>`. A `Verify` az alakból olvassa a paramétereket; ha az iteráció kisebb a jelenlegi alapértéknél, `needsRehash = true`, és a sikeres bejelentkezés újrahash-el. Összehasonlítás: `subtle.ConstantTimeCompare`.
- A hash-elés párhuzamossága legfeljebb 4 (szemafor a `Hasher`-ben).
- Ismeretlen felhasználónál a `DummyVerify` egy induláskor kiszámolt hash-sel fut, így a válaszidő nem árulja el a fiók létezését.
- Tesztben a `Hasher` 1000 iterációval készül (`NewHasher(1000, 4)`), a formátum és a `needsRehash` viselkedése ugyanaz.
- Jelszószabály (`CheckPasswordPolicy`, sorrendben, az első hiba mezőkódja):
  1. legalább 12 rúna: `password_too_short`;
  2. legfeljebb 128 rúna: `password_too_long`;
  3. nem tartalmazza a felhasználónevet (kisbetűsítve, ha a név legalább 3 karakter): `password_contains_username`;
  4. legalább 6 különböző rúna: `password_too_simple`;
  5. nem egyezik a jelenlegi jelszóval (csak jelszócserénél): `password_reused`.

  Összetételi szabály (kötelező szám, írásjel) nincs.

### 6.5 TOTP (kétlépcsős azonosítás)

- RFC 6238: HMAC-SHA1, 6 számjegy, 30 s lépés, T0 = 0. Titok: 20 véletlen bájt, base32 (RFC 4648, padding nélkül, nagybetűs).
- Elfogadási ablak: az aktuális lépés ±1. A `VerifyTOTP` a lehetséges lépéseket növekvő sorrendben vizsgálja, és csak a `lastStep`-nél nagyobbat fogadja el; összehasonlítás `subtle.ConstantTimeCompare`.
- Visszajátszás elleni védelem: siker után `AdvanceTOTPStep(uid, step)` = `UPDATE users SET totp_last_step = :step WHERE id = :id AND totp_last_step < :step`; ha 0 sort érint (párhuzamos kérés már felhasználta), a kód `invalid_totp`.
- URI: `otpauth://totp/Kinex%20Mark:<username>?secret=<SECRET>&issuer=Kinex%20Mark&algorithm=SHA1&digits=6&period=30` (a felhasználónév URL-kódolva).
- Tárolás: `Keyring.Seal`, AES-256-GCM, 12 bájtos véletlen nonce, társított adat `totp:<userID>`.
- A bejelentkezés második lépése (5.7): felhasználónként egyetlen élő kihívás, kihívásonként legfeljebb 5 atomikusan foglalt próbálkozás; minden hiba a felhasználói és az IP-kulcson is számít; zárolt kulcsnál a második lépés is 429; a felhasználói számlálót csak a második tényező sikere nullázza, a helyes jelszó nem. Így a jelszót ismerő támadó sem próbálgathat korlátlanul kódot.
- A QR-kódot a felület rajzolja (`uqr`), a szerver csak az URI-t adja.

### 6.6 Helyreállító kódok

- 10 kód, mindegyik 10 karakter az `abcdefghjkmnpqrstuvwxyz23456789` ábécéből (kb. 49 bit), megjelenítve `xxxxx-xxxxx` alakban.
- Tárolás: `Keyring.MAC("recovery-code", normalizált kód)` hexában (64 karakter); a szerverkulcs nélkül az adatbázisból offline sem törhető.
- Felhasználás: `UPDATE recovery_codes SET used_at = :now WHERE user_id = :uid AND code_hash = :h AND used_at IS NULL`; egyszer használható.
- Újragenerálás és 2FA-kikapcsolás: a régi kódok törlődnek.

### 6.7 Bejelentkezés-korlátozás

| Kulcs | Küszöb | Ablak | Zárolás | Fokozás |
|---|---|---|---|---|
| `u:<hex MAC>` (4.3) | 5 hiba | 15 perc | 15 perc | minden újabb zárolás duplázza (15, 30, 60 … perc), legfeljebb 24 óra; 24 óra hiba nélkül nullázza a `lockouts` számlálót |
| `ip:<cím>` (kibontott IPv4: /32; IPv6: /64) | 30 hiba | 15 perc | 15 perc | nincs |

- Hibának számít: hibás jelszó (bejelentkezés, jelszócsere, 2FA-beállítás és -kikapcsolás, helyreállító kódok újragenerálása, fióktörlés, az admin érzékeny műveleteinek megerősítése), hibás TOTP- vagy helyreállító kód (a második bejelentkezési lépésben a felhasználói és az IP-kulcson is).
- Az ablak a `window_started_at`-tól számít; lejárta után a számláló nullázódik.
- A „24 óra hiba nélkül” horgonya (`auth.LockoutResetAfter`): a 24 óra az utolsó számláló ablak vége (`window_started_at` + ablak, az utolsó hiba felső becslése) és az utolsó zárolás vége (`locked_until`) közül a későbbitől számít; a `Reserve` ennyi csend után nullázza a `lockouts`-ot. A zárolás végétől mérve egy 24 órás zárolás lejárta után a fokozás nem indul újra azonnal. A küszöböt elérő `Confirm` zárol, növeli a `lockouts`-ot, és új, üres ablakot nyit (`failures = 0`, `window_started_at = now`), ezért a zárolás utáni párhuzamos megerősítések nem halmozódnak és nem fokoznak kétszer; a `Reserve` lejárt zárolásnál a `locked_until`-t NULL-ra állítja.
- Atomikusság és előfoglalás: minden próbálkozás a jelszó- vagy kódellenőrzés **előtt** foglal (`ThrottleUpdate` + `ThrottleRule.Check`, majd `Reserve`, egyetlen írótranzakcióban; az egyetlen adatbázis-kapcsolat miatt a tranzakciók sorban futnak, 4.1). Ellenőrzött hibánál `Confirm` (ez zárol a küszöbnél), sikernél `Refund`. A párhuzamos kérések így egymás foglalását látják: 20 egyidejű hibás jelszóból legfeljebb 5 jut el a hash-ellenőrzésig, a többi 429 (GoH-teszt, 9.2.3). A kihívások próbálkozása ugyanígy egyetlen `UPDATE … RETURNING` utasításban foglal (`ChallengeAttempt`).
- Feloldás: a második tényezővel is teljes sikeres bejelentkezés (csak a `u:` kulcs), admin `unlock` és admin jelszó-visszaállítás (`ClearUserThrottle`), offline `-unlock-user` és `-reset-password` (6.12).
- A számítás az `auth.ThrottleRule` tiszta függvényeiben él; a store csak a tranzakciót adja.
- Az `AdminUser.lockedUntil` a felhasználó `login_throttle` sorának (`user_id`) `locked_until` értéke, ha a jövőben van.
- Ismert korlát: a felhasználói kulcs globális, ezért egy támadó a felhasználónév ismeretében egy fiókot ismételten zárolhat (legfeljebb 24 óráig zárolásonként). A helyreállítás útjai (admin feloldás, offline parancs) ezért mindig rendelkezésre állnak, az utolsó admin számára is; a lépcsőzetes, (felhasználó, IP-tartomány) párra szűkített zárolás a SUMMARY.md következő lépései közé kerül.

### 6.8 SSRF-védelem és letöltési szabályok

- Csak `http` és `https`; port: 80, 443, vagy 1024–65535 (a többi 1024 alatti port tiltott: `blocked_port`).
- `http.Transport.Proxy = nil` (a környezeti proxyváltozókat figyelmen kívül hagyja, különben a védelem megkerülhető); `http.Transport.DisableKeepAlives = true`, így minden kérés új tárcsázás, és a `Control`-ellenőrzés a házirend minden változása után azonnal érvényes (nincs újrahasznosított kapcsolat egy közben tiltottá vált címhez).
- A `net.Dialer.Control` minden kapcsolódás előtt a ténylegesen tárcsázott IP-címet ellenőrzi (`CheckIP` a pillanatnyi `Policy()`-val, tesztben utána a `DialAllow` is), így a DNS-rebinding és az átirányítás is le van fedve. Az IPv4-be ágyazott IPv6-címet (`::ffff:a.b.c.d`) előbb kibontja (`Unmap`); a NAT64 (`64:ff9b::/96`, `64:ff9b:1::/48`), a 6to4 (`2002::/16`) és a Teredo (`2001::/32`, a kliens-IPv4 bitenként negálva) címből és az IPv4-kompatibilis (`::/96`) címből a beágyazott IPv4-et is ellenőrzi.
- Mindig tiltott, `allowPrivateFetch` mellett is: `0.0.0.0/8`, `::/128`, `224.0.0.0/4`, `ff00::/8`, `240.0.0.0/4`, `255.255.255.255/32`, a felhőmetaadat-címek (`169.254.169.254`, `169.254.170.2`, `100.100.100.200`, `168.63.129.16`, `fd00:ec2::254`, `fd20:ce::254`), valamint a dokumentációs tartományok (`192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`, `2001:db8::/32`).
- Tiltott, csak `APP_ALLOW_LOOPBACK_FETCH=true` mellett engedett (rendszerbeállítás nincs hozzá, az `allowPrivateFetch` nem nyitja): `127.0.0.0/8`, `::1/128`, `169.254.0.0/16` (a metaadat-címek nélkül), `fe80::/10`. Így az intranetes linkek engedélyezése után sem szkennelheti egyetlen szerepkör sem a szerver saját szolgáltatásait, és a `fetchError` nem válik port-szkenner orákulummá a helyi gépen.
- Alapból tiltott, `allowPrivateFetch` mellett engedett: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `100.64.0.0/10`, `fc00::/7`, `192.0.0.0/24`, `198.18.0.0/15`.
- Effektív `allowPrivateFetch`: ha az `APP_ALLOW_PRIVATE_FETCH` meg van adva, az dönt, és az adminfelületen a beállítás zárolt (`lockedByEnv`); különben a rendszerbeállítás. Az E2E-környezet `APP_ALLOW_PRIVATE_FETCH=true` és `APP_ALLOW_LOOPBACK_FETCH=true` mellett fut, mert a fixture-forrás a `127.0.0.1`-en él.
- Időkorlátok: dial 5 s, TLS-kézfogás 5 s, válaszfejléc 5 s; a teljes letöltés (oldal és ikon) `APP_FETCH_TIMEOUT` (alap 10 s) egy `context.WithTimeout`-tal.
- Átirányítás: legfeljebb 5 (`CheckRedirect`), minden lépésnél újra séma- és portellenőrzés.
- Méret: HTML legfeljebb 1 MiB (`io.LimitReader`; a feldolgozás a `</head>` vagy az első `<body>` elemnél megáll); ikon legfeljebb 64 KiB (`Content-Length` alapján előre, olvasás közben `LimitReader` + 1 bájt próbával).
- Kérésfejlécek: `User-Agent: KinexMark/<version> (+metadata fetcher)`, `Accept: text/html,application/xhtml+xml;q=0.9,*/*;q=0.5`, `Accept-Language`: a tulajdonos nyelve (`hu`, `en`, `de`), utána `en;q=0.5`. Süti és hitelesítő adat nem megy ki.
- Kódolás: `charset.NewReader(body, contentType)` (`golang.org/x/net/html/charset`), így a `windows-1250`, az `iso-8859-2` és a `<meta charset>` is helyesen dekódol.
- Cím: az első `<title>` szövege; ha üres, `<meta property="og:title">`. Normalizálás (`CleanText`): a C0 (U+0000–U+001F) és C1 (U+0080–U+009F) vezérlőkarakterek, az U+007F, valamint a bidi-jelölő, -felülíró és -izoláló karakterek (a Unicode Bidi_Control halmaza: U+061C, U+200E, U+200F, U+202A–U+202E, U+2066–U+2069; a webes keresőmező-tisztítás is ugyanezt dobja el) eltávolítása, szóközök összevonása, vágás, legfeljebb 500 rúna. Ugyanez a szabály vonatkozik az importált címre és leírásra (a leírásban a `\n` megmarad), mint az 5.1 a felhasználói bevitelre; így a megjelenítési címke nem hamisítható.
- Ikonjelöltek sorrendje: `rel` tokenjei között `icon` és `sizes` 32×32 → `icon` `sizes` nélkül → `icon` SVG típussal → bármely más `icon` a 32-höz legközelebbi méret szerint → `apple-touch-icon` → `/favicon.ico`. A `<base href>` és a végső URL alapján old fel. `data:image/...;base64` ikont dekódol és ugyanúgy ellenőriz. Legfeljebb 3 jelöltet tölt le.
- Ikonellenőrzés: nem üres, legfeljebb 64 KiB; a típus a tartalomból (`http.DetectContentType`): `image/png`, `image/gif`, `image/jpeg`, `image/webp`, `image/x-icon` (a `image/vnd.microsoft.icon` erre normalizálva). SVG: a tartalom `<svg` vagy `<?xml … <svg` kezdetű, és nem tartalmaz `<script`, `javascript:` és `on…=` eseményattribútumot (kisbetűsítve keresve), és átmegy a szerkezeti ellenőrzésen (`fetch/svg.go` `svgStructureOK`, E fázis, a D-close 1. ismert korlátjának javítása): szigorú `encoding/xml` tokenizálás (csak UTF-8, egyedi entitás nincs, így a karakterhivatkozások az ellenőrzés előtt dekódolódnak); `<?xml?>` csak első tokenként, `DOCTYPE`, `ENTITY` és más feldolgozási utasítás tilos; pontosan egy `<svg>` gyökér az SVG-névtérben; csak SVG-névtérbeli engedélyezett elemek (`script`, `foreignObject`, `a`, `image`, `feImage` és az animációs elemek nélkül); csak engedélyezett attribútumok, a `href`/`xlink:href` csak `#id`; az attribútumértékek és a `<style>` szövege CSS-escape, `@import`, `javascript:`, `data:`, `image-set(` és társaik nélkül, `url(…)` csak `#id`-re; ilyenkor `image/svg+xml`. Elutasított SVG után a következő jelölt jön (Inkscape/Sodipodi-metaadatos vagy névtér nélküli SVG-ikon helyett tehát a raszteres tartalék). Minden más elutasítva. Az SVG-t a böngésző `<img>`-ként tölti be (szkript ott nem fut), a végpont CSP-je `sandbox`.
- Ha az oldal letöltése hibás (nem 2xx, nem HTML), a `/favicon.ico` tartalék akkor is lefut; a `PageErr` jelzi az oldalhibát.

### 6.9 Biztonsági fejlécek és CSP

Minden válaszon (`securityHeaders`):

| Fejléc | Érték |
|---|---|
| `Content-Security-Policy` | lent |
| `X-Content-Type-Options` | `nosniff` |
| `Referrer-Policy` | `no-referrer` |
| `X-Frame-Options` | `DENY` |
| `Cross-Origin-Opener-Policy` | `same-origin` |
| `Cross-Origin-Resource-Policy` | `same-origin` |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()` |
| `X-Robots-Tag` | `noindex, nofollow` |
| `Strict-Transport-Security` | `max-age=31536000` (csak biztonságos módban) |

CSP (az ikonvégpont felülírja, 5.9.9):

```
default-src 'self'; script-src 'self' <hash-ek>; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; manifest-src 'self'; worker-src 'self'
```

Biztonságos módban a végére `; upgrade-insecure-requests` kerül.

Inline szkriptek: a GRAPHICS.md 2.6 témaindító szkriptje szó szerint az `app.html`-ben marad. Az adapter-static fallback `index.html` két inline szkriptet tartalmaz (a témaindítót és a SvelteKit indítóját, amelynek szövege minden buildnél változik). A `spa.go` induláskor az `index.html`-ből (amelyet a TECHNOLOGY.md 4.7 szerint amúgy is memóriába olvas) minden `src` nélküli `<script>` elem nyers tartalmát kinyeri (`golang.org/x/net/html` tokenizáló, `Raw()` a szöveges tokenre), SHA-256-ot számol, és `'sha256-<base64>'` alakban a `script-src`-be teszi. Így minden build magától lefedett, `'unsafe-inline'` nélkül. Következmények:

- A frontend nem állít be `kit.csp`-t (a CSP egyetlen forrása a szerver).
- `<ModeWatcher defaultMode="dark" lightClassNames={['light']} disableHeadScriptInjection />`, mert a futás közben beszúrt inline szkriptet a CSP tiltaná; a villanásmentességet az `app.html` `class="dark"` attribútuma és a hash-elt indító szkript adja.
- A `style-src 'unsafe-inline'` indoka: a bits-ui lebegő elemei és a Svelte-átmenetek `style` attribútumot írnak, és az `app.html` `<div style="display: contents">` eleme is inline stílus. Stílusból szkript nem fut.
- Teszt: a Go `spa_test.go` a hash-kinyerést ellenőrzi; az E2E-csomag minden oldalon figyeli a konzolt, CSP-sértés esetén a teszt elbukik.

SPA-kiszolgálás: a TECHNOLOGY.md 4.7 kezelője a `fonts/` bővítéssel: `_app/immutable/` és `fonts/` alatt `public, max-age=31536000, immutable`; hiányzó `_app/` fájl 404; minden más útvonal `index.html` `no-cache`-sel. A `web/static/robots.txt` tartalma `User-agent: *` és `Disallow: /`.

### 6.10 Audit események

| Esemény | Mikor | `details` |
|---|---|---|
| `bootstrap.admin_created` | Első admin létrehozása | `{generatedPassword: bool}` |
| `auth.login.success` | Sikeres bejelentkezés | `{method: "password" \| "totp" \| "recovery_code"}` |
| `auth.login.failure` | Hibás jelszó, ismeretlen felhasználónév, inaktív fiók, hibás TOTP | `{reason: "credentials" \| "unknown_user" \| "disabled" \| "totp"}`; ismeretlen névnél `actor_id = NULL`, `actor_name = ''` (5.7) |
| `auth.login.locked` | Zárolt kulcsra érkező kísérlet (kulcsonként 15 percenként legfeljebb egyszer) | `{scope: "user" \| "ip"}` |
| `auth.logout` | Kijelentkezés | |
| `auth.recovery_code.used` | Helyreállító kód felhasználása | `{remaining: n}` |
| `auth.password.changed` | Saját jelszócsere | |
| `auth.2fa.enabled`, `auth.2fa.disabled`, `auth.2fa.recovery_regenerated` | 2FA-műveletek | |
| `session.revoked` | Egy munkamenet visszavonása | `{sessionId}` |
| `session.revoked_others` | A többi munkamenet visszavonása | `{count}` |
| `user.created`, `user.updated`, `user.deactivated`, `user.activated`, `user.deleted` | Adminműveletek | `user.created`: `{role}`; `user.updated`: `{changed: ["displayName", "role"]}` (csak a ténylegesen változott mezők), és csak a nem személyes mezőnél az érték is (`role: {from, to}`); az aktív állapot változása nem `user.updated`, hanem `user.activated` / `user.deactivated` `details` nélkül (5.12); megjelenített név, felhasználónév és jelszó értéke soha |
| `user.password_reset`, `user.2fa_reset`, `user.unlocked`, `user.sessions_revoked` | Adminműveletek | |
| `user.self_deleted` | Saját fiók törlése | |
| `settings.updated` | Rendszerbeállítás módosítása | `{changed: ["defaultArchiveDays", ...]}` |
| `privacy.notice_bumped` | Új tájékoztató-verzió | `{version}` |
| `privacy.consent_accepted` | Hozzájárulás | `{version}` |
| `gdpr.export` | Adatexport | |
| `link.shared`, `link.unshared` | Megosztás változása | `{targetType, targetRole, targetUserId}` |

Az eseménykódok listája a Go `model/audit.go` `AuditEvents` szeleteként él (28 kód); a szerződés-golden (9.6) ebből generálódik, és a frontend `audit.events.<event>` kulcsainak teljességét ez ellenőrzi.

Az audit naplóba soha nem kerül jelszó, token, TOTP-kód, helyreállító kód, URL, linkcím vagy a bejelentkezéskor beírt, nem létező felhasználónév (adattakarékosság; az elgépelt jelszó gyakran a névmezőbe kerül). A `details` kulcsai és felsorolt értékei zárt készletből jönnek (`reason`, `method`, `scope`, `changed`, `role`, `active`, `remaining`, `sessionId`, `count`, `version`, `targetType`, `targetRole`, `targetUserId`, `generatedPassword`, `via`), a felület ezeket fordítja (8.5.8). Az `active` kulcs fenntartott: a zárt készletben (`model.AuditDetailKeys`, szerződés-golden, frontendfordítás) szerepel, de jelenleg egyetlen esemény sem írja, mert az aktiválást a saját esemény jelzi; a backend a `TestAdminUsersUpdateAuditSplit` teszttel rögzíti ezt a viselkedést. A készletből nem kerül ki, hogy egy későbbi esemény a fordítás változtatása nélkül használhassa. A linkek létrehozása, módosítása és megnyitása nem auditált esemény.

### 6.11 Felhasználó törlése

`DeleteUser(ctx, id, e)` egy tranzakcióban:

1. `last_admin` ellenőrzés.
2. Az átadott auditsor (`user.self_deleted` vagy `user.deleted`) beírása, így a 3. lépés ezt is pszeudonimizálja.
3. Pszeudonimizálás: `UPDATE audit_log SET actor_name = 'deleted-user-' || :id WHERE actor_id = :id OR (actor_id IS NULL AND lower(actor_name) = :username_key)`, és ugyanígy a `target_name`, ahol `target_type = 'user' AND target_id = :id`. Az `actor_id` a kaszkád miatt `NULL` lesz. A `details` nevet nem tartalmaz (6.10), ezért nem kell átírni; a `store`-teszt ezt is ellenőrzi (egy `user.updated` megjelenítettnév-változás után a törölt felhasználó egyetlen auditsora sem tartalmazza sem a felhasználónevét, sem a megjelenített nevét).
4. `DELETE FROM users WHERE id = :id`: a kaszkád törli a beállításokat, munkameneteket, kihívásokat, korlátozási sorokat (`login_throttle.user_id`), helyreállító kódokat, hozzájárulásokat, címkéket, saját linkeket (velük a megosztásokat és más felhasználók rájuk vonatkozó személyes állapotát), a személyes állapotokat és a felhasználónak szóló megosztásokat.
5. A commit után best-effort `PRAGMA wal_checkpoint(TRUNCATE)` (`truncateWAL`; hibája nem hiba, mert a törlés már véglegesült, és az ellenőrzőpont később magától lefut). A `secure_delete(1)` pragma (4.1, 2.2 T23) miatt a törölt sorok tartalma az új lapváltozatokban nullázva van, a csonkolás pedig a régi lapképeket tartalmazó WAL-keretektől szabadít meg, így a törölt adat nem marad olvasható az adatkönyvtár fájljaiban és a későbbi mentésekben (store-teszt: `TestDeleteUserLeavesNoRemnants`, a nyers `app.db` és `app.db-wal` bájtjait vizsgálja nyitott és lezárt store mellett).

A címzettek a törölt felhasználó megosztott linkjeit ezután nem látják; ezt a törlési dialógus előre kimondja a darabszámmal (`PrivacyStatus.ownSharedLinkCount`, `AdminUser.sharedLinkCount`; 8.7.6, 8.7.7).

### 6.12 Első admin és titkok kezelése

- Ha a `users` tábla üres: felhasználó `APP_ADMIN_USER` néven (a `config.Load` már validálta, 3.3.2), `admin` szerepkörrel. Ha az `APP_ADMIN_PASSWORD` meg van adva, át kell mennie a jelszószabályon (különben a program nem indul: `bootstrap admin: password does not meet the policy`), és `must_change_password = 0`; utána Warn: `bootstrap admin created from APP_ADMIN_PASSWORD; remove the variable from the environment`. Ha nincs megadva: 20 karakteres generált jelszó, `must_change_password = 1`, és egyetlen Warn szintű napló: `log.Warn("initial admin password generated", "username", u, "password", p)`. Audit: `bootstrap.admin_created`.
- Ha a `users` tábla nem üres, de az `APP_ADMIN_PASSWORD` be van állítva, minden induláskor Warn: `APP_ADMIN_PASSWORD is set but ignored on a non-empty database; remove it` (a titok különben a konténer környezetében, például a `docker inspect` kimenetében marad).
- Offline helyreállítás (a zárolt vagy elfelejtett jelszavú utolsó admin esetére is): `kinex-mark -data <dir> -unlock-user <név>` törli a felhasználó korlátozási sorait (`ClearUserThrottle`); `kinex-mark -data <dir> -reset-password <név>` 20 karakteres jelszót generál, `must_change_password = 1`, törli a munkameneteit és korlátozási sorait, és a jelszót egyszer a stdout-ra írja (a naplóba nem). Mindkettő auditál (`user.unlocked`, illetve `user.password_reset`, `actor_id = NULL`, `actor_name = ''`, `details = {via: "cli"}`), HTTP-szervert nem indít, és kilép. Ajánlott a szerver leállítása mellett futtatni (Docker: `docker compose stop`, majd `docker compose run --rm kinex-mark -reset-password admin`); a WAL mód és a `busy_timeout` miatt futó szerver mellett is biztonságos. A README és a kézikönyv hibaelhárítási fejezete leírja.
- A naplóba a generált kezdő jelszón kívül soha nem kerül jelszó, token, süti, TOTP-titok vagy -kód, helyreállító kód, query-string vagy URL.
- A `secret.key` az adatkönyvtárban él, a mentés része. Ha elveszik, a TOTP-titkok és a helyreállító kódok használhatatlanok; az indulási kulcsellenőrzés (3.3.11 5. lépés) ilyenkor megállítja a programot. Helyreállítás: az eredeti kulcs visszaállítása.
- Az adatkönyvtár jogosultsága 0750, a kulcsfájlé 0600 (Windowson az NTFS ACL dönt, a program nem állítja).

### 6.13 Adatmegőrzés

| Adat | Megőrzés | Törlés módja |
|---|---|---|
| Munkamenet | az abszolút vagy a tétlenségi lejáratig | elutasítás lejáratkor, purge |
| Bejelentkezési kihívás | 5 perc | purge |
| Korlátozási sor | utolsó módosítás + 24 óra, ha nincs aktív zárolás | purge |
| Függő TOTP-titok | 10 perc | purge (`NULL`-ra állítja) |
| Audit napló | `auditRetentionDays` (alap 180 nap) | purge |
| Gazdátlan ikon | a következő purge-ig | purge |
| Felhasznált helyreállító kód | a következő újragenerálásig vagy a 2FA kikapcsolásáig | felhasználói művelet |
| Hozzájárulás | a fiók élettartama | kaszkád |
| Fiók és minden linkadat | a fiók törléséig | kaszkád (6.11) |
| Audit-bejegyzés törölt felhasználóról | a megőrzési idő végéig, pszeudonimizálva | purge |

### 6.14 GDPR-megfelelés összefoglalása

| Elv vagy jog | Megvalósítás |
|---|---|
| Adattakarékosság | nincs e-mail, telefonszám, analitika; az audit nem tárol URL-t, linkcímet és a bejelentkezéskor beírt nem létező nevet; a korlátozási kulcs a felhasználónév MAC-je; a napló nem tartalmaz query-stringet és URL-t |
| Jogalap (6. cikk) | fiók, linkek, címkék, megosztások, beállítások: a szolgáltatás nyújtása, 6. cikk (1) b); biztonsági napló, bejelentkezés-korlátozás, munkamenetek IP-címe: az üzemeltető jogos érdeke a rendszer biztonságához, 6. cikk (1) f). Hozzájárulás nem jogalap |
| Tájékoztatás (13. cikk) | lokalizált `/privacy` oldal (bejelentkezés nélkül is) a 8.7.10 szakaszaival: adatkezelő és adatvédelmi kapcsolattartó, kezelt adatok, célok és jogalapok, címzettek (más felhasználók a nekik megosztott linkeket és a tulajdonos felhasználónevét és megjelenített nevét látják; a fejlesztők és az adminok a megosztás címzettjének kiválasztásakor minden aktív felhasználót megtalálnak felhasználónév, megjelenített név és szerepkör szerint (5.11); az üzemeltető és tárhelyszolgáltatója), harmadik országba továbbítás nincs, automatizált döntéshozatal és profilalkotás nincs, megőrzési idők a `PublicConfig.privacy` mezőiből, jogok (15–21. cikk), panasz a felügyeleti hatóságnál (`privacySupervisoryAuthority`), az admin jelszó-visszaállításának ténye |
| A tájékoztató tudomásulvétele | `consent_records`, verziózott tájékoztató; új verziónál a `consent_required` kapu. Ez a tájékoztató megismerésének rögzítése (nyugtázás), nem hozzájárulás mint jogalap, ezért visszavonási folyamat nem kell hozzá; a felület szövege is „elolvastam és tudomásul veszem” |
| Hozzáférés és hordozhatóság (15., 20. cikk) | `GET /api/me/export` (JSON: a fiókra vonatkozó audit események szereplőként és célként, a zárolási állapot, a megosztott linkek személyes állapota a láthatóság szerint, 5.8.1) és a saját linkek exportja (JSON, Netscape HTML) |
| Korlátozás és tiltakozás (18., 21. cikk) | kérésre az adatkezelőnél (a tájékoztató elérhetőségein); technikailag: fiók inaktiválása az adminfelületen (a fiók és az adatok megmaradnak, kezelésük szünetel) |
| Helyesbítés | profil, linkek és címkék szerkesztése |
| Törlés (17. cikk) | `POST /api/me/delete` önkiszolgálóan, admin felhasználótörlés; kaszkád és pszeudonimizálás (a `details` nevet nem tartalmaz, 6.10–6.11); `secure_delete` és WAL-csonkolás, így a törölt adat az adatbázisfájlokban sem marad olvasható (6.11 5. lépés) |
| Korlátozott megőrzés | 6.13, óránkénti purge |
| Sütik | egyetlen, feltétlenül szükséges munkamenet-süti; a téma és a nyelv a böngésző helyi tárolójában (`mode-watcher-mode`, `mode-watcher-theme`, `kxm-locale`); ezért sütisáv nem kell. A generált shadcn `Sidebar.Provider` `sidebar_state` sütit írna, ezért az app a saját `SidebarFrame`-jét használja (8.1.5, 9.8) |
| Harmadik fél | a felület semmit nem tölt harmadik féltől (fontok, ikonok a szerverről; a metaadatot a szerver tölti le, a böngésző nem kapcsolódik a link webhelyéhez a megnyitásig) |
| Biztonság (32. cikk) | 6.1–6.9: titkosított TOTP-titok, hash-elt jelszó és token, CSRF, CSP, korlátozás, audit |

---

## 7. Háttérfolyamatok

### 7.1 Ütemezés

A `jobs.Runner.Start` három goroutine-csoportot indít:

| Feladat | Első futás | Ismétlés |
|---|---|---|
| Archiválás, majd purge (egy goroutine-ban, sorban) | az indulás után azonnal | óránként |
| Letöltési seprő | az indulás után azonnal | percenként |
| Letöltési munkások | folyamatosan | `APP_FETCH_WORKERS` darab |

Minden futás eredménye Info szintű napló (`archive run`, `archived`, `n`; `purge run` és a törölt sorok száma); hibánál Error, a következő ciklus újra próbálja. Az időzítés a `clock.Clock`-ot és egy injektálható `time.Ticker`-gyárat használ, így a teszt a ciklusokat kézzel lépteti.

### 7.2 Automatikus archiválás

Szabály: egy link egy felhasználónál archiválódik, ha nincs archiválva, nincs kitűzve, és az utolsó aktivitás (`max(baseline_at, last_used_at)`) régebbi, mint a felhasználó effektív időszaka. Effektív időszak: a `user_preferences.archive_after_days`, ha nem `NULL`, különben a `defaultArchiveDays` rendszerbeállítás; 0 = soha. Inaktív felhasználónál a feladat nem fut.

`ArchiveInactive(ctx, defaultDays)`:

1. A felhasználók listája egy lekérdezéssel, a `rows` lezárásáig memóriába gyűjtve:

   ```sql
   SELECT u.id, u.role, p.archive_after_days
   FROM users u LEFT JOIN user_preferences p ON p.user_id = u.id
   WHERE u.is_active = 1
   ```

2. Felhasználónként, ha az effektív időszak > 0, egy tranzakcióban:

   ```sql
   -- missing state rows for shared links, baseline = earliest effective share,
   -- but not before the recipient's account existed (viewerCreatedAt)
   INSERT INTO link_user_state (user_id, link_id, baseline_at, updated_at)
   SELECT :uid, s.link_id,
          max(MIN(s.created_at), (SELECT vu.created_at FROM users vu WHERE vu.id = :uid)), :now
   FROM link_shares s
   JOIN links l ON l.id = s.link_id
   JOIN users o ON o.id = l.owner_id AND o.role IN ('admin', 'developer')
   WHERE l.owner_id <> :uid
     AND (s.target_type = 'all'
          OR (s.target_type = 'role' AND s.target_role = :role)
          OR (s.target_type = 'user' AND s.target_user_id = :uid))
   GROUP BY s.link_id
   ON CONFLICT (user_id, link_id) DO NOTHING;

   -- own links get their state row on creation; fallback for older rows
   INSERT INTO link_user_state (user_id, link_id, baseline_at, updated_at)
   SELECT :uid, l.id, l.created_at, :now FROM links l WHERE l.owner_id = :uid
   ON CONFLICT (user_id, link_id) DO NOTHING;

   UPDATE link_user_state
   SET archived_at = :now, archive_reason = 'auto', updated_at = :now
   WHERE user_id = :uid
     AND archived_at IS NULL
     AND pinned = 0
     AND max(baseline_at, coalesce(last_used_at, baseline_at)) < :cutoff
     AND link_id IN (
         SELECT id FROM links WHERE owner_id = :uid
         UNION
         SELECT s.link_id FROM link_shares s
         JOIN links l ON l.id = s.link_id
         JOIN users o ON o.id = l.owner_id AND o.role IN ('admin', 'developer')
         WHERE l.owner_id <> :uid
           AND (s.target_type = 'all'
                OR (s.target_type = 'role' AND s.target_role = :role)
                OR (s.target_type = 'user' AND s.target_user_id = :uid)));
   ```

   ahol `:cutoff = FormatTime(now - days × 24h)`. (Az SQL-kommentek angolok, mert a migrációs és a store-forrás része.) Mindkét `INSERT … SELECT … ON CONFLICT` utasításnak van `WHERE` záradéka (4.1).

3. Visszatérési érték: az archivált sorok száma összesen.

Láthatóvá válás szerepkör-változáskor (`UpdateUser`, ugyanabban a tranzakcióban, a `users` sor frissítése után; D fázis, R1):

- `roleVisibilityUpsert`: ha a felhasználó szerepköre `:old_role`-ról `:role`-ra változik, az új szerepkörrel megosztott, eddig nem látható linkek (nincs `all`, régi szerepkörös vagy neki szóló megosztásuk) állapotsort kapnak `baseline_at = :now` értékkel.
- `ownerSharesReturnUpsert`: ha a megosztó visszakapja a megosztójogot (ügyfélről fejlesztőre vagy adminra), a nála ügyfélként alvó megosztások (4.7) újra láthatók; minden címzett (a megosztó kivételével minden felhasználó, akire az `all`, a szerepkörös vagy a neki szóló megosztás illik) `baseline_at = :now` állapotsort kap.
- Mindkettő `INSERT … SELECT … WHERE … GROUP BY s.link_id … ON CONFLICT (user_id, link_id) DO UPDATE SET baseline_at = max(baseline_at, excluded.baseline_at), updated_at = excluded.updated_at`: a meglévő sor kitűzése, használata és archiválása megmarad, a `baseline_at` csak előre mozdul.

Következmények: a soha nem használt saját link a létrehozástól, a soha nem használt megosztott link a megosztástól számít, de legkorábban a címzett fiókjának létrehozásától, illetve attól, hogy a link számára láthatóvá vált; a használat (megnyitás vagy másolás, 5.9.5), a kitűzés, a kitűzés megszüntetése és a visszaállítás újraindítja a számlálást (4.9); a kitűzött link soha nem archiválódik automatikusan. Az admin `POST /api/admin/jobs/archive` végpontja azonnal lefuttatja a feladatot.

### 7.3 Metaadat-letöltési sor

- Sor: pufferelt csatorna, kapacitás 256; az `Enqueue` nem blokkol (`select` `default` ággal). A folyamatban lévő azonosítók halmaza (`map[int64]struct{}` mutexszel) megakadályozza a kettős feldolgozást.
- Belépési pontok: link létrehozása, gyors rögzítés, URL-módosítás, kézi frissítés (`Enqueue` a kezelőből); a seprő percenként `DueFetches(50)` (`fetch_status = 'pending' AND next_fetch_at <= :now`, `next_fetch_at` szerint), és mindegyiket sorba teszi. Minden letöltésre jelölő írás `next_fetch_at = now`-t ír (4.4), ezért `NULL` `next_fetch_at`-tal függő link nem létezik. Ha a sor tele, vagy a folyamat a `BeginFetch` előtt leáll, a link `pending` marad, a következő seprés felveszi; így az indításkor függőben maradt és az importált linkek is feldolgozódnak (store-teszt: `Enqueue` nélkül létrehozott linket a `DueFetches` visszaad).
- A munkás lépései:
  1. `BeginFetch(id, lease = 2 perc)`: `UPDATE links SET next_fetch_at = :now + lease WHERE id = :id AND fetch_status = 'pending' AND next_fetch_at <= :now RETURNING url, owner_id`, valamint a tulajdonos nyelve. Ha nincs sor (már nem függő, vagy egy másik munkás bérli), kihagyja.
  2. `fetcher.Fetch(ctx, url, lang)` a 6.8 szabályaival, adatbázis-kapcsolat tartása nélkül.
  3. `ApplyFetchResult` egy tranzakcióban, `WHERE id = :id AND url = :url` feltétellel (ha közben az URL megváltozott, az eredmény eldobódik): ikon upsert (`INSERT … ON CONFLICT (sha256) DO NOTHING`, majd `SELECT id`), állapot, `fetch_error`, `fetched_at`, majd a cím és az ikon az alábbi szabály szerint, végül `search_text` és `sort_label` újraszámolása.
- Cím és ikon írása (`title_source`, 4.4), csak végleges eredménynél (`ok`, `failed`, `blocked`; az újrapróbálandó átmeneti hiba semmit nem ír át):
  - `icon_id` = az eredmény ikonja, ha van érvényes ikon, különben `NULL` (a régi ikon nem marad meg).
  - `ok` és `title_source <> 'manual'`: nem üres letöltött címnél `title` = a letöltött cím, `title_source = 'fetched'`; üres letöltött címnél `title = ''`, `title_source = 'none'`.
  - `failed`, `blocked`: a cím nem változik.
  - `manual` címet a letöltés soha nem ír.
- Naplózás: a munkás és a seprő csak Debug szinten naplóz, és csak a `link_id`-t, a hibakódot (7.3 táblázat) és a hostot: `log.Debug("metadata fetch failed", "link_id", id, "code", e.Code, "host", host)`. Az URL, a query-string és az `err.Error()` (a `*url.Error` üzenete a teljes URL-t tartalmazza) soha nem kerül naplóba.

| `fetch_error` | Mikor | Átmeneti | i18n |
|---|---|---|---|
| `timeout` | Időtúllépés | igen | `fetch.errors.timeout` |
| `dns` | Névfeloldási hiba | igen | `fetch.errors.dns` |
| `connection` | Kapcsolódási hiba, megszakadt kapcsolat | igen | `fetch.errors.connection` |
| `tls` | Tanúsítványhiba | nem | `fetch.errors.tls` |
| `http_status` | Nem 2xx; az 5xx és a 429 átmeneti | 5xx/429 igen, egyéb nem | `fetch.errors.http_status` |
| `too_large` | HTML 1 MiB fölött, és a csonkolt feldolgozás sem adott címet | nem | `fetch.errors.too_large` |
| `not_html` | Nem HTML tartalom (a link ettől `ok`, a cím üres, a mező kitöltve tájékoztatásul) | – | `fetch.errors.not_html` |
| `too_many_redirects` | 5-nél több átirányítás | nem | `fetch.errors.too_many_redirects` |
| `blocked_address` | SSRF-szabály tiltotta | nem, állapot `blocked` | `fetch.errors.blocked_address` |
| `blocked_port` | Tiltott port | nem, állapot `blocked` | `fetch.errors.blocked_port` |

Átmeneti hibánál `fetch_attempts + 1`; ha ez kisebb 3-nál, `next_fetch_at = now + backoff` (1 perc, majd 10 perc), az állapot `pending` marad; a harmadik hiba után `failed`. Végleges hibánál azonnal `failed` (vagy `blocked`), `next_fetch_at = NULL`. Siker: `ok`, `fetch_error = NULL` (kivéve a `not_html`), `next_fetch_at = NULL`.

### 7.4 Purge

`Purge(ctx, auditRetention)` utasításonként (nem egy nagy tranzakcióban, hogy ne tartsa sokáig az egyetlen kapcsolatot):

```sql
DELETE FROM sessions WHERE expires_at < :now OR last_seen_at < :idle_cutoff;
DELETE FROM login_challenges WHERE expires_at < :now;
DELETE FROM login_throttle WHERE (locked_until IS NULL OR locked_until < :now) AND updated_at < :day_ago;
UPDATE users SET totp_pending_secret = NULL, totp_pending_expires_at = NULL
 WHERE totp_pending_expires_at < :now;
DELETE FROM audit_log WHERE at < :audit_cutoff;
DELETE FROM icons WHERE NOT EXISTS (SELECT 1 FROM links WHERE links.icon_id = icons.id);
PRAGMA optimize;
```

### 7.5 Szabályos leállítás

1. SIGTERM vagy Ctrl+C: `log.Info("shutting down")`.
2. `srv.Shutdown` 10 s időkorláttal (TECHNOLOGY.md 4.1): új kérés nem jön, a futók befejeződnek.
3. A feladatok kontextusának lezárása: az ütemező és a seprő kilép, a munkások a folyamatban lévő letöltést megszakítják és nem írnak eredményt (a link `pending` marad, a bérlet lejárta után a következő indításkor újra sorra kerül).
4. `runner.Wait(5 * time.Second)`; túllépéskor Warn napló, a leállás folytatódik.
5. `closeStore` (defer): előbb `st.Ping` (legfeljebb a drain-időkorlátig), mert egy tranzakció közben megszakított feladat a kapcsolatát aszinkron adja vissza, és a `DB.Close` nem várja meg a használatban lévő kapcsolatot; utána `st.Close()`. Hiba esetén csak Warn napló.

---

## 8. Frontend

### 8.0 Alapszabályok és tervezési koncepció

Alapszabályok minden frontendfájlra:

- A forráskód kizárólag angol: azonosító, komment, `console`-üzenet, tesztnév. Magyar, angol és német szöveg csak a `src/lib/i18n/messages/**` fájlokban van.
- Minden felhasználónak látható szöveg a `t()` függvényen megy át (`aria-label`, `title`, `alt`, toast, `<title>` is). A backend soha nem ad lokalizált szöveget; a hibákat a stabil `code` alapján a felület fordítja.
- Svelte 5 runes mód, TypeScript `strict`, Lucide-ikon egyenként importálva `strokeWidth={1.5}`-tel, nyers hex érték nincs a kódban, komponensben szemantikus token.
- Emoji és felkiáltójel nincs, sem a felületen, sem a fordításban.
- Felülettervezéskor a fejlesztő ügynök a `frontend-design` skillt tölti be (D14), a GRAPHICS.md keretein belül: a tokenek, betűk, formák, mozgás és hangnem rögzítettek, a szabadság a kompozícióban, a hierarchiában és a termékspecifikus részletekben van.

Tárgy, közönség, fő feladat: belső könyvjelzőkezelő egy mérnöki cég és ügyfelei számára; a link munkaeszköz (dokumentáció, admin felület, repository, ügyféloldali rendszer). A felhasználók naponta sokszor nyitják meg ugyanazt a néhány tucat linket. Fő feladat: a ma kellő link legyen az első sorban, és az új link rögzítése ne szakítsa meg a munkát.

Token-összefoglaló (a GRAPHICS.md-ből, változtatás nélkül):

| Szerep | Token | Sötét | Világos |
|---|---|---|---|
| Alap | `background` | `#071016` | `#f2f5f3` |
| Felület | `card` | `#0b171f` | `#ffffff` |
| Szöveg | `foreground` | `#d4dcd5` | `#0b171f` |
| Halk szöveg | `muted-foreground` | `#94a9a5` | `#486469` |
| Jel (egyetlen kiemelés) | `primary` | `#77dbed` | `#16637e` |
| Márkafelület (témától független) | `ink-950`, `ink-900`, `signal-300` | `#040a0f`, `#071016`, `#77dbed` | ugyanaz |

Betű: Hanken Grotesk minden emberi szövegre (oldalcím `heading-1` 32/40 súly 300, linkcímke `body` 15/24 súly 500), Geist Mono minden gépi értékre (URL, host, darabszám, időpont, azonosító, IP, kód).

Elvek:

1. A sorrend maga az információ. A kitűzött és a többi csoportot egy-egy `kx-eyebrow` sorcímke választja el („Kitűzött”, „Többi”), mert ez a rangsorolási szabályt teszi láthatóvá.
2. A gépi érték gépi betűt kap: URL, host, megnyitásszám, időpont mono; a címke és a leírás sans.
3. Az elévülés műszer, nem riasztás: az `ExpiryMeter` öt hajszálvékony szegmense mutatja a hátralévő időt az automatikus archiválásig; csak az utolsó hét napban kap `warning` ikont és szót.
4. Egy merész pont van: a bejelentkező oldal sötét márkafelülete a levágott sarkú hero-táblával. Minden más csendes: hajszálvonal, sík kártya, árnyék csak lebegő elemen.
5. A rögzítés nem dialógus: ráhúzás és beillesztés azonnal ment; a visszajelzés toast és a sor rövid kiemelése.

Sablonos megoldások elleni önellenőrzés:

| Gyanús alapértelmezés | Döntés |
|---|---|
| Közel fekete alap egyetlen élénk kiemeléssel | A GRAPHICS.md előírja (petróleum alap, cián jel); a petróleum árnyalat és a teljes értékű világos téma különbözteti meg |
| Mono betű kis adatfeliratokra | Csak valódi gépi értéken (URL, szám, idő) |
| Csupa nagybetűs szemöldökfelirat minden cím felett | Csak három helyen: oldalsáv-csoportnév, listacsoport (Kitűzött/Többi), részletek lap szakaszcíme |
| Egyforma kártyák mindenre | A lista hajszálvonalas sorok egyetlen kártyában; kártya csak a csempe nézetben és a beállítási blokkokban |
| Középső pontos meta-sor („A · B · C”) | Nincs; meta-adatok külön oszlopban, felirattal vagy tooltipben |
| Számozott jelölők (01, 02) | Csak valódi sorrendnél: a 2FA bekapcsolásának három lépése |
| `→` a gombfeliraton | Nincs; ige és tárgy (GRAPHICS.md 1.6) |

Vázlatok:

```
Bejelentkezés, asztal (>= 1024 px)
+------------------------------------------+------------------------------+
| ink-950 márkafelület (mindkét témában)    | background (témát követ)      |
|  [K-jel] Kinex Mark                       |                              |
|  +-------------------------------------+  |   Bejelentkezés              |
|  | Panel lg (chamfer 24) hero.webp 16:9 |  |   Felhasználónév [________]  |
|  +-------------------------------------+  |   Jelszó         [______][o] |
|  Linkek, használat szerint rendezve.      |   [  Bejelentkezés  ]        |
|  bevezető (body-lg, mist-300)             |                              |
|  ||||||||||||||||| (TickRule)             |                              |
|  [ikon] Gyors rögzítés   [ikon] Címkék    |   HU EN DE   [nap/hold]      |
|  [ikon] Szűrés           [ikon] Rangsor   |   Adatkezelési tájékoztató   |
|  [ikon] Automatikus archiválás            |   v1.0.0                     |
+------------------------------------------+------------------------------+

Linkek oldal, asztal
+----------------+---------------------------------------------------------+
| [K] Kinex Mark | Gyűjtemény > Linkek     [Keresés a linkek között] [HU][☼]|
| GYŰJTEMÉNY     +---------------------------------------------------------+
|  Linkek   128  | Linkek                       [Import/export] [+ Link felvétele]
|  Címkék    14  | 128 link, ebből 3 kitűzött                               |
| GYAKORI CÍMKÉK | [Mind|Saját|Megosztott] [Használat v] [=|#] [Nézet v] (o) Archiváltak
|  ■ docs    41  | ■ docs  ■ infra  ■ ügyfél-A  ■ ...   [További címkék]    |
| FIÓK           | KITŰZÖTT                                                  |
|  Beállítások   | [ico] Grafana éles  ■infra  pin        42  ma 09:12  ▮▮▮▮▯ [⋯]|
| ADMINISZTRÁCIÓ | TÖBBI                                                     |
|  Felhasználók  | [ico] SvelteKit docs ■docs            17  tegnap    ▮▮▮▯▯ [⋯]|
|  Rendszer...   | [ico] example.com                       0  –      ⌛ 5 nap [⋯]|
|  Audit napló   |                                                           |
| [Kovács Anna v]|                                                           |
+----------------+---------------------------------------------------------+
```

### 8.1 Eszközök és projektváz

#### 8.1.1 Telepítési sor (Git Bashben kipróbált)

A `web/` a repository gyökerében jön létre. A sorrend kötött; a 7. lépés pipe-ja csak Git Bashben igazolt, ezért ezt Git Bashben kell futtatni (PowerShellből `bash -lc '...'`).

```sh
cd /c/Project/@KINEXIUM/kinex-mark

# 1. Skeleton
npx --yes sv@1.7.0 create web --template minimal --types ts --no-add-ons --install npm --no-dir-check
cd web

# 2. Tailwind v4 (non-interactive form; --no-git-check: the repo working tree is dirty,
#    and without it sv asks for confirmation and aborts on closed stdin)
npx sv@1.7.0 add tailwindcss="plugins:none" --no-install --no-git-check

# 3. Adapter, animation, theme, icons
npm i -D @sveltejs/adapter-static tw-animate-css
npm i mode-watcher @lucide/svelte
npm rm @sveltejs/adapter-auto

# 4. The global stylesheet must live at src/app.css BEFORE shadcn init (init stops otherwise)
mv src/routes/layout.css src/app.css
#    src/routes/+layout.svelte imports '../app.css'

# 5. vite.config.ts: written with the Write tool (8.1.4), NOT with a heredoc (the /[/\\]/ regex breaks)

# 6.
npx svelte-kit sync

# 7. shadcn-svelte init (npx --yes installs the CLI without asking; the pipe answers the init prompts)
printf 'y\n\n\n\n' | npx --yes shadcn-svelte@1.8.1 init --preset nova --css src/app.css \
  --components-alias '$lib/components' --lib-alias '$lib' --utils-alias '$lib/utils' \
  --hooks-alias '$lib/hooks' --ui-alias '$lib/components/ui'
npm rm @fontsource-variable/geist

# 8. Components (8.1.5)
npx shadcn-svelte@1.8.1 add -y -o button input label select checkbox switch table tabs \
  dialog alert card badge sidebar alert-dialog dropdown-menu tooltip sonner popover command \
  separator sheet skeleton scroll-area toggle-group textarea breadcrumb pagination input-otp \
  empty field kbd spinner < /dev/null

# 9. Brand stylesheet (8.1.6) and components.json (GRAPHICS.md 13.3)
node scripts/sync-app-css.mjs

# 10. Testing, formatting, QR
npm i -D vitest @vitest/coverage-v8 @testing-library/svelte @testing-library/jest-dom \
  @testing-library/user-event jsdom @playwright/test @axe-core/playwright \
  prettier prettier-plugin-svelte @types/node
npm i -D typescript@~6.0.3
npm i uqr
npx playwright install chromium          # CI: npx playwright install --with-deps chromium

# 11. Manual steps
#   - src/app.html per GRAPHICS.md 2.6 (full replacement)
#   - src/routes/+layout.svelte and +layout.ts (8.1.7)
#   - rm src/lib/assets/favicon.svg
#   - static/favicon.svg (GRAPHICS.md 7.7), static/robots.txt ("User-agent: *" / "Disallow: /")
#   - static/fonts/: the two woff2 files (10.6) + OFL-Hanken-Grotesk.txt, OFL-Geist-Mono.txt
#   - overlay class of dialog-overlay.svelte, alert-dialog-overlay.svelte, sheet-overlay.svelte (GRAPHICS.md 10.8)
#   - tsconfig.json include stays as verified: "src", "vite.config.ts" (no e2e, no other config files)
#   - e2e/tsconfig.json for e2e/**, vitest.config.ts and playwright.config.ts (see below)
#   - src/lib/utils.ts stays as generated by the CLI: export { cn } from "cn"

npm run check && npm run test && npm run build
```

Típusellenőrzés két részben: az alkalmazás `tsconfig.json`-ja a kipróbált `include`-dal marad (`src`, `vite.config.ts`), a `svelte-check` ezt nézi. Az E2E-kód és a két további konfigurációs fájl külön `web/e2e/tsconfig.json`-t kap, amelyet a `check` szkript `tsc --noEmit -p e2e/tsconfig.json` lépése ellenőriz:

```json
{
  "compilerOptions": {
    "target": "ES2023", "module": "NodeNext", "moduleResolution": "NodeNext",
    "strict": true, "noEmit": true, "skipLibCheck": true,
    "allowImportingTsExtensions": true, "allowJs": true, "checkJs": false,
    "types": ["node"]
  },
  "include": ["**/*.ts", "../playwright.config.ts", "../vitest.config.ts"]
}
```

Az A2 kész-feltétele: a `npm run check` (benne a `tsc -p e2e` lépés) 0 hibával és 0 figyelmeztetéssel fut; ha a `vitest.config.ts` `./vite.config.ts` importja a NodeNext-feloldással mégsem fordul, a `vitest.config.ts` kimarad az `include`-ból, és ez az átadó jegyzetbe kerül.

Rögzített verziók (a `package-lock.json` tartja, `lockfileVersion` 3): `@sveltejs/kit` 3.4.0, `svelte` 5.59.1, `vite` 8.3.8, `@sveltejs/vite-plugin-svelte` 7.3.1, `@sveltejs/adapter-static` 4.0.1, `tailwindcss` és `@tailwindcss/vite` 4.3.5, `bits-ui` 2.21.0, `mode-watcher` 1.1.0, `@lucide/svelte` 1.77.0, `shadcn-svelte` 1.8.1, `tw-animate-css` 1.4.0, `tailwind-variants` 3.3.1, `typescript` 6.0.3, `vitest` és `@vitest/coverage-v8` 5.1.1, `@testing-library/svelte` 5.4.2, `@testing-library/jest-dom` 8.0.0, `@testing-library/user-event` 14.6.13, `jsdom` 31.0.1, `@playwright/test` 1.65.1, `@axe-core/playwright` (aktuális), `prettier` és `prettier-plugin-svelte` (aktuális), `uqr` 0.1.2.

Új függőségek indoka:

| Csomag | Indok |
|---|---|
| `uqr` | A 2FA QR-kódja kliensoldalon az `otpauthUri`-ból (D4). Függőség nélküli, kb. 10 KB; csak a mátrixot adja, a rajzolás saját SVG (`QrCode.svelte`) |
| `@axe-core/playwright` | Akadálymentességi ellenőrzés az E2E-csomagban |
| `prettier`, `prettier-plugin-svelte` | Formátumkapu (`npm run lint`) |
| `@testing-library/*`, `jsdom`, `vitest`, `@playwright/test` | Tesztelés (9. fejezet) |
| `@types/node` | Az E2E-keret és a konfigurációs fájlok típusai (`process.env`, `node:fs`, `node:crypto`) az `e2e/tsconfig.json` alatt |

A `chart` és az `avatar` komponens nem kerül be (2.3 G2).

#### 8.1.2 `package.json`

```json
{
  "name": "kinex-mark-web",
  "version": "0.0.1",
  "private": true,
  "type": "module",
  "engines": { "node": "^24 || >=26" },
  "imports": { "#lib": "./src/lib/index.js", "#lib/*": "./src/lib/*" },
  "scripts": {
    "dev": "vite dev",
    "build": "vite build",
    "preview": "vite preview",
    "prepare": "svelte-kit sync || echo ''",
    "check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json --fail-on-warnings && tsc --noEmit -p e2e/tsconfig.json",
    "check:watch": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json --watch",
    "lint": "prettier --check .",
    "format": "prettier --write .",
    "test": "vitest run",
    "test:unit": "vitest",
    "test:coverage": "vitest run --coverage",
    "test:e2e": "playwright test",
    "sync:css": "node scripts/sync-app-css.mjs",
    "seed:demo": "node e2e/support/seed-demo.mjs"
  }
}
```

- A `devDependencies` és `dependencies` listáját az npm tartja (8.1.1); a `typescript` tartománya `~6.0.3`, a `tailwind-variants` tartománya `^3.3.1` (a shadcn CLI pinje).
- A CLI a shadcn-függőségeket (`bits-ui`, `svelte-sonner`, `tailwind-variants`, `cn`, `@internationalized/date`) a `devDependencies`-be teszi; statikus SPA-nál ez helyes, nem mozgatjuk.
- `.npmrc`: `engine-strict=true` (a `sv create` írja). Node mindenhol 26.
- `.prettierrc`: `useTabs: true`, `singleQuote: true`, `trailingComma: "none"`, `printWidth: 100`, `plugins: ["prettier-plugin-svelte"]`, Svelte-fájlokra `parser: "svelte"`. `.prettierignore`: `build`, `.svelte-kit`, `coverage`, `test-results`, `playwright-report`, `blob-report`, `e2e/.auth`, `package-lock.json`, `src/lib/components/ui`, `src/app.css`, `src/app.html` (a prettier különben újratördelné a témaindító szkriptet, amelynek a GRAPHICS.md 2.6 szerint szó szerint kell maradnia; 6.9), `.playwright-mcp` (a Playwright MCP kimeneti mappája, ha egy MCP-szervert a `web/` gyökérrel indítanak; 9.5, 10.1).

#### 8.1.3 Importaliasok

- A shadcn-svelte generált fájljai `$lib/...js` útvonalat használnak (173 előfordulás), ezért a `$lib` alias kötelező; a saját kód is `$lib/...` importot használ. A `#lib` alias megmarad a `package.json`-ban, de nem használjuk.
- A márkakomponensek egymást relatív úton importálják (`./Panel.svelte`).

#### 8.1.4 `vite.config.ts` (teljes, a Write eszközzel írva)

```ts
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

// The Go server listens here during development (build.ps1 dev, go run ./cmd/kinex-mark).
const backend = 'http://127.0.0.1:8080';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			// index.html is the shared entry point of every client-side route.
			adapter: adapter({ fallback: 'index.html' }),
			// shadcn-svelte generates $lib imports.
			alias: { $lib: 'src/lib' }
		})
	],
	server: {
		// No changeOrigin: the Go CSRF check compares Origin with the forwarded Host.
		proxy: { '/api': backend, '/healthz': backend }
	}
});
```

`kit.csp` nincs beállítva (6.9). Ismert figyelmeztetések minden `sync`, `check`, `build`, `vitest` és `playwright` futásnál: `config_option_deprecated_alias` és `tsconfig_invalid` (10.7).

#### 8.1.5 shadcn-svelte komponensek

Hozzáadva: `button input label select checkbox switch table tabs dialog alert card badge sidebar alert-dialog dropdown-menu tooltip sonner popover command separator sheet skeleton scroll-area toggle-group textarea breadcrumb pagination input-otp empty field kbd spinner`. Függőségként jön: `input-group`, `toggle`, `hooks/is-mobile.svelte.ts`.

| Komponens | Felhasználás |
|---|---|
| `sidebar`, `breadcrumb`, `separator`, `scroll-area` | AppShell |
| `button`, `input`, `textarea`, `label`, `field`, `select`, `checkbox`, `switch`, `input-otp`, `input-group` | Űrlapok |
| `dialog`, `alert-dialog`, `sheet` | Felvétel/szerkesztés, megerősítés, részletek lap |
| `dropdown-menu`, `popover`, `command`, `tooltip` | Sorműveletek, címkeválasztó, felhasználóválasztó |
| `toggle-group` | Nézetváltó, hatókör, címkeegyezés, téma, színválasztó |
| `table`, `pagination`, `card`, `badge`, `alert`, `empty`, `skeleton`, `spinner`, `kbd`, `sonner`, `tabs` | Adatnézetek és állapotok |

A generált fájlokat nem szerkesztjük, egy kivétellel: a `dialog-overlay.svelte`, az `alert-dialog-overlay.svelte` és a `sheet-overlay.svelte` `bg-black/…` osztálya helyére `bg-[color-mix(in_oklab,var(--ink-950)_62%,transparent)]` kerül, és ugyanebből az osztálylistából a `supports-backdrop-filter:backdrop-blur-xs` elmarad: a fátyol sima színréteg, elmosás nélkül (GRAPHICS.md 10.8; 2.3 G6; E fázis; 9.8 12. szabály). A shadcn `Tabs` vonalas megjelenését (GRAPHICS.md 10.18), az `Alert` `variant="default"` használatát (10.1) és a `TableHead` osztálylistáját (10.7) a `components/app/` burkolói (`LineTabs`, `StateAlert`, `DataTableHead`) rögzítik.

A nova-alapértékek felülírása az alkalmazásrétegben (D fázis, R3 és R5; a generált fájlok változatlanok):

- `components/app/Button.svelte`: a generált `Button` burkolója változatonkénti `buttonTone` osztálylistával (radius-md, primary hover `color-mix`-szel, accent hover, sötét témában átlátszó `outline`). A saját kód minden `Button` importja ezt használja, a generáltat közvetlenül nem; a `Pager` linkjei a `buttonTone`-t kapják.
- `components/app/SidebarFrame.svelte`: a generált `Sidebar.Provider` helyett adja a sidebar-kontextust rögzített nyitott állapottal. A generált provider `sidebar_state` sütit írt, és egy ablakszintű Ctrl/Cmd+B kezelővel a szövegmezőkből is elvette a billentyűt (6.14 „Sütik”).
- `components/app/styles.ts`: `typeHeading3`, `typeBodySm`, `typeEyebrow` utility-listák a generált komponenseken (dialógus-, alert-dialógus- és lapcím, leírás, címke, sidebar-csoportcímke), mert a komponensrétegbeli `heading-3`, `body-sm`, `kx-eyebrow` osztály alulmarad a nova utility-jeivel szemben; `toggleItem` (a kiválasztott ToggleGroup-elem accent felület, előtérszín és 2 px-es primary vonal, GRAPHICS.md 6.2, WCAG 1.4.11); `inputSize` (radius-md, átlátszó mező); `dialogBox` (200 ms ease-out); `dialogFooter` (sáv nélkül, GRAPHICS.md 10.8); `tooltipSurface` (popover felület kerettel és árnyékkal).
- A sötét téma generált `dark:bg-input/30` kitöltését csak `!important` utility (`bg-transparent!`, `hover:bg-accent!`) írja felül, mert a 8.0 8. szabálya szerint saját kódban `dark:` nincs.
- E fázis (GUI-teszt) kiegészítései a `styles.ts`-ben: a `dialogBox` a `grid-cols-[minmax(0,1fr)] *:min-w-0 wrap-anywhere max-h-[calc(100dvh-32px)] overflow-y-auto` osztályokkal bővül (hosszú címke és törhetetlen szó sem szélesíti a dialógust, a lábléc gombjai elérhetők maradnak; minden `Dialog.Content` és `AlertDialog.Content` ezt használja, 9.8 13. szabály); a `toggleItem` része a `toggleRadius` (radius-md a szegmentált csoport külső sarkain is, a nova `-lg` helyett); `checkboxBox` (`rounded-sm data-unchecked:bg-transparent!`, minden `Checkbox`-on, 9.8 16. szabály); `invalidField` (`aria-invalid:ring-0 aria-invalid:border-destructive!`: hibás mezőn a destruktív keret a nova gyűrűudvara nélkül; az `inputSize`, a `buttonTone.outline` és a LinkDialog szövegmezője használja).
- E fázis új alkalmazásrétegbeli elemei: `components/app/AccountUsername.svelte` (rejtett, csak olvasható `autocomplete="username"` mező a fiók felhasználónevével, `tabindex=-1`, `aria-hidden`; minden `current-password` mezős űrlapban, kivéve a saját felhasználónév-mezős LoginForm és a másik fiókról szóló UserDialog, 9.8 17. szabály) és `utils/toast-error.ts` `toastError(err, message?)`: a kérés hibájának toastja, amely kihagyja a munkamenet-tároló által már kezelt hibát (`ApiError.authHandled`: `unauthenticated`, `password_change_required`, `consent_required`), így lejárt munkamenetnél csak a munkamenet üzenete jelenik meg (9.8 15. szabály).

#### 8.1.6 Stíluslapok

- `src/app.css`: a GRAPHICS.md 13.2 kódblokkja, kézzel soha nem szerkesztve. A `scripts/sync-app-css.mjs` a `../ai/GRAPHICS.md`-ből a „### 13.2” fejléc utáni első ```` ```css ```` blokkot kivágja (a kerítések alapján, nem sorszámmal), minden `/* … */` kommentet eltávolít (a forrás magyar kommentjei a D2a miatt nem kerülhetnek a kódba), és egyetlen angol fejlécet tesz elé: `/* KINEXIUM brand stylesheet. Generated from ai/GRAPHICS.md section 13.2 by scripts/sync-app-css.mjs; do not edit by hand. */`. A `src/lib/test/app-css.spec.ts` a fájlt és a GRAPHICS.md blokkját kommentek nélkül, szóközökben normalizálva hasonlítja össze; ha a `../ai/GRAPHICS.md` nem elérhető (Docker-build), a teszt kihagyja magát.
- `src/kinex-mark.css` (a `+layout.svelte` az `app.css` után importálja). Szabályai: sima CSS, csak `var(--token)` hivatkozással; nyers hex, új szín, új token, `@theme`, `@apply`, új betűcsalád tilos; minden szabály előtt egysoros angol komment. Zárt tartalomlista (bővítés csak e terv módosításával):
  1. `@keyframes kxm-flash` és `.kxm-flash`: a frissen rögzített vagy megtalált sor kiemelése (`background-color` `var(--accent)` → átlátszó, 1200 ms `ease-out`); csökkentett mozgásnál a `.kxm-flash-static` 2 másodpercig `outline: 1px solid var(--ring)`.
  2. `.kxm-drop-scrim`: a ráhúzási fátyol háttere, `color-mix(in oklab, var(--background) 88%, transparent)`.
  3. `.kxm-list-row`: `content-visibility: auto; contain-intrinsic-size: auto 52px;`.
  4. `@media print`: a helyreállító kódok nyomtatása (`.kxm-print-only`, `.kxm-no-print`), sötét szöveg világos alapon mindkét témában: `color-scheme: light`, minden semleges felület (`--background`, `--card`, `--popover`, `--secondary`, `--muted`, `--accent`) `var(--mist-50)`, minden előtérszín és a `--border`, `--input` `var(--ink-950)` (E fázis: a csak `--foreground`/`--background` felülírás mellett a kódlap sötét témában olvashatatlan volt). A `src/lib/test/kinex-mark-css.spec.ts` az `app.css` minden semleges tokenjére ellenőrzi a felülírást.
- Tailwind arbitrary érték csak tokenre mutathat: `max-w-[var(--layout-prose)]`, `w-[var(--layout-sidebar)]`, `h-[var(--layout-topbar)]`.

#### 8.1.7 Gyökérfájlok

`src/app.html`: a GRAPHICS.md 2.6 teljes változata (`<html lang="hu" class="dark">`, témaindító inline szkript, favicon, font preload, `theme-color`). A témaindító szövegét a szerver futásidőben hash-eli, ezért változtatása nem töri a CSP-t, de csak a GRAPHICS.md-vel együtt változhat.

`src/routes/+layout.ts`:

```ts
// Client-side application: no SSR and no prerendering.
export const ssr = false;
export const prerender = false;
```

`src/routes/+layout.svelte`:

```svelte
<script lang="ts">
	import '../app.css';
	import '../kinex-mark.css';
	import { ModeWatcher } from 'mode-watcher';
	import { Toaster } from '$lib/components/ui/sonner/index.js';
	import { i18n } from '$lib/i18n/index.js';
	let { children } = $props();

	// Keep <html lang> in sync with the active locale (GRAPHICS.md 1.9).
	$effect(() => {
		document.documentElement.lang = i18n.locale;
	});
</script>

<ModeWatcher defaultMode="dark" lightClassNames={['light']} disableHeadScriptInjection />
<Toaster position="bottom-right" />
{@render children()}
```

`web/embed.go` (a TECHNOLOGY.md 4.8 mintája angol kommentekkel):

```go
// Package web embeds the static SvelteKit build into the Go binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var build embed.FS

// Files returns the static UI rooted at the build directory.
func Files() (fs.FS, error) {
	return fs.Sub(build, "build")
}
```

### 8.2 Forrásfa és fájltulajdon

A tulajdonos a 11. fejezet munkacsomagja; a fa a B fázisbeli tulajdonost mutatja. Egy fájlnak egy időben egy tulajdonosa van. Az A2 által létrehozott vázfájlok (konfigurációk, `ui/`, `hooks/`, `utils.ts`, `index.ts`, `app.*`, gyökér-layout, `test/setup.ts`, `scripts/`) az A fázis végén az F0-hoz kerülnek; a `playwright.config.ts` váza a DX-hez. Az F0 által előre létrehozott csonk (üres névtérmodul, üres feature-komponens, `EmptyState`-es útvonal) az F0 befejezésekor a megnevezett tulajdonosé lesz; utána az F0 nem módosítja.

```
web/
├── embed.go  assets_test.go                                                 (A2)
├── hero_test.go                                                             (G)
├── components.json  vite.config.ts  vitest.config.ts  tsconfig.json  .vscode/  (F0; váz: A2)
├── playwright.config.ts                                                     (DX; váz: A2)
├── package.json  package-lock.json  .npmrc  .prettierrc  .prettierignore  .gitignore  (F0; váz: A2)
├── scripts/sync-app-css.mjs                                                 (F0; váz: A2)
├── static/
│   ├── favicon.svg  robots.txt                                              (A2)
│   ├── fonts/  HankenGrotesk-Variable.woff2  GeistMono-Variable.woff2  OFL-*.txt  (A2)
│   └── images/hero.webp                                                     (G)
├── e2e/
│   ├── tsconfig.json                                                        (DX; váz: A2)
│   ├── support/  start-server.mjs  fixture-origin.mjs  constants.mjs  global-setup.ts  global-teardown.ts
│   │             cleanup.mjs  fixtures.ts  api.ts  i18n.ts  totp.ts
│   │             helpers.ts  seed-demo.mjs                                  (DX)
│   ├── fixtures/site/  fixtures/demo-data.json                              (DX)
│   ├── specs/smoke.spec.ts                                                  (DX, a B fázis végén E1-nek átadva)
│   └── specs/*.spec.ts                                                      (E1)
└── src/
    ├── app.html  app.css  kinex-mark.css  app.d.ts                          (F0)
    ├── lib/
    │   ├── utils.ts  index.ts  hooks/  components/ui/                       (F0, generált)
    │   ├── api/  client.ts  errors.ts  types.ts  index.ts
    │   │         public.ts  auth.ts  me.ts  links.ts  tags.ts  directory.ts  admin.ts   (F0)
    │   ├── i18n/
    │   │   ├── index.ts  types.ts  plural.ts  format.ts  detect.ts  errors.ts       (F0)
    │   │   ├── contract.spec.ts  parity.spec.ts  format.spec.ts  index.spec.ts      (F0)
    │   │   └── messages/{hu,en,de}/
    │   │       ├── index.ts                                                 (F0, minden névtérrel)
    │   │       ├── common.ts  nav.ts  errors.ts  fields.ts  roles.ts  status.ts     (F0)
    │   │       ├── auth.ts  landing.ts  privacy.ts                          (F1)
    │   │       ├── links.ts  capture.ts  fetch.ts                           (F2)
    │   │       ├── tags.ts  tagColors.ts  settings.ts                       (F3)
    │   │       ├── admin.ts  audit.ts                                       (F4)
    │   │       └── security.ts                                              (F5: settings › biztonság és adatvédelem)
    │   ├── state/  session.svelte.ts  preferences.svelte.ts  public-config.svelte.ts  tags.svelte.ts
    │   │           link-stats.svelte.ts                                     (F0)
    │   │           links.svelte.ts                                          (F2)
    │   ├── links/  presentation.ts  order.ts  markdown.ts  expiry.ts  query.ts  capture.ts  (F2)
    │   ├── tags/colors.ts                                                   (F0)
    │   ├── utils/  clipboard.ts  download.ts  debounce.ts  keyboard.ts  user-agent.ts  qr.ts  navigation.ts  (F0)
    │   ├── components/
    │   │   ├── brand/  Logo  Panel  Metric  StatusBadge  TickRule           (F0)
    │   │   └── app/    (8.4; D fázis: + Button  SidebarFrame, 8.1.5)        (F0)
    │   ├── features/
    │   │   ├── auth/       LoginForm  TotpStep  PasswordChangeForm          (F1; a PasswordChangeForm az F0 típusos csonkja)
    │   │   ├── privacy/    PrivacyNotice  ConsentForm                       (F1; a PrivacyNotice az F0 típusos csonkja)
    │   │   ├── links/      LinkList  LinkRow  LinkTile  LinkToolbar  TagFilterBar  ExpiryMeter
    │   │   │               LinkDialog  LinkDetailsSheet  ShareDialog  ImportExportMenu
    │   │   │               CaptureController  ShortcutsDialog  actions.ts   (F2; a CaptureController
    │   │   │                                                                 és a ShortcutsDialog az F0 csonkja)
    │   │   ├── tags/       TagsTable  TagDialog                             (F3)
    │   │   ├── settings/profile/   ProfileForm                              (F3)
    │   │   ├── settings/display/   DisplaySettings  ArchivePeriodSelect     (F3)
    │   │   ├── settings/security/  PasswordCard  TwoFactorCard  TwoFactorEnrollDialog
    │   │   │                       TwoFactorConfirmDialog  RecoveryCodesView  SessionsCard  (F5)
    │   │   ├── settings/privacy/   ConsentHistory  DataExportCard  DeleteAccountDialog      (F5)
    │   │   └── admin/      UsersTable  UserDialog  TempPasswordDialog  SettingsForm  AuditTable  (F4)
    │   └── test/  setup.ts  fixtures.ts  render.ts  mock-api.ts  conventions.spec.ts  app-css.spec.ts
    │              text.ts (D fázis: `normalizeSpace` a U+00A0-s szövegek összevetéséhez)  (F0)
    └── routes/
        ├── +layout.svelte  +layout.ts  +error.svelte                        (F0)
        ├── (public)/+layout.svelte                                          (F0)
        ├── (public)/login/+page.svelte  +page.ts                            (F1)
        ├── (public)/privacy/+page.svelte                                    (F1)
        ├── (gate)/+layout.svelte  +layout.ts                                (F0)
        ├── (gate)/change-password/+page.svelte                              (F1)
        ├── (gate)/consent/+page.svelte                                      (F1)
        ├── (app)/+layout.svelte  +layout.ts                                 (F0)
        ├── (app)/+page.svelte                                               (F2)  útvonal: /
        ├── (app)/[...rest]/+page.ts  +page.svelte                           (D fázis)  ismeretlen út: 404 a keretben (8.7.11)
        ├── (app)/tags/+page.svelte                                          (F3)
        ├── (app)/settings/+layout.svelte  +page.ts                          (F0)  LineTabs; /settings → /settings/profile
        ├── (app)/settings/profile/+page.svelte                              (F3)
        ├── (app)/settings/display/+page.svelte                              (F3)
        ├── (app)/settings/security/+page.svelte                             (F5)
        ├── (app)/settings/privacy/+page.svelte                              (F5)
        ├── (app)/admin/+layout.ts  +page.ts                                 (F0)  szerepkör-kapu; /admin → /admin/users
        ├── (app)/admin/users/+page.svelte                                   (F4)
        ├── (app)/admin/settings/+page.svelte                                (F4)
        └── (app)/admin/audit/+page.svelte                                   (F4)
```

Névtér-szabály: egy kulcs annak a munkacsomagnak a névterében él, amelyik a kulcsot megjelenítő fájlt birtokolja, mert a `MessageKey` a magyar katalógusból származik, és a hiányzó kulcs `svelte-check`-hiba. Ezért a beállítás-fülek felirata `nav.settingsTabs.*` (F0, a `(app)/settings/+layout.svelte` rajzolja), a `TagPicker` szövegei `common.tagPicker.*` (F0), a címkeszűrő sor szövegei `links.filters.*` (F2). A `settings` névtér (F3) a profil és a megjelenítés kulcsait tartalmazza; a biztonsági és az adatvédelmi fül kulcsai a `security` névtérben (F5) vannak, így két fejlesztő nem szerkeszti ugyanazt a fájlt.

Hullámon belüli függőségek, csonkkal feloldva: az F5 az F1 `PasswordChangeForm` és `PrivacyNotice` komponensét használja. Az F0 mindkettőt végleges propokkal, típusos csonkként hozza létre (`PasswordChangeForm`: `{ mode: 'gate' | 'settings'; onSuccess?: () => void }`; `PrivacyNotice`: `{ config: PublicConfig['privacy']; compact?: boolean }`), így az F5 az F1-gyel párhuzamosan fordul; a tartalmat az F1 írja. A `tags.svelte.ts` (F0) nem importálja a links store-t (F2): `onTagsChanged(listener)` feliratkozást ad, amelyre az F2 `links.svelte.ts` iratkozik fel.

A tesztfájl a forrásfájl tulajdonosáé (`presentation.ts` → `presentation.spec.ts`; komponensteszt: `LinkRow.svelte.spec.ts`).

### 8.3 `lib/api`: típusos kliens

`client.ts`:

```ts
export type Query = Record<string, string | number | boolean | undefined | null>;
export type RequestOptions = {
	query?: Query;
	body?: unknown;                 // JSON-encoded
	raw?: BodyInit;                 // import: raw file body, with contentType
	contentType?: string;
	signal?: AbortSignal;
	keepalive?: boolean;            // visit tracking
	skipAuthRedirect?: boolean;     // login, me probe
};

export async function request<T>(method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE', path: string, opts?: RequestOptions): Promise<T>;
export async function download(path: string, query?: Query): Promise<{ blob: Blob; filename: string }>;
export function onAuthFailure(handler: (code: 'unauthenticated' | 'password_change_required' | 'consent_required') => void): void;
```

Működés:

- URL: `'/api' + path` (a `getHealth` kivétel: `/healthz`) + a `query` nem üres értékeiből `URLSearchParams`.
- Fejlécek minden kérésen: `X-Kinex-Request: 1`, `Accept: application/json`; `Content-Type: application/json` csak törzzsel (importnál a megadott típus). `credentials: 'same-origin'`.
- 204 vagy üres törzs: `undefined`; különben `res.json()`.
- Nem 2xx: a törzset `res.json().catch(() => null)` olvassa; `ApiError` keletkezik `status`, `code`, `fields`, `details` mezőkkel. Kód nélküli hibánál a tartalék az 5.3 szerint. 429-nél a `Retry-After` fejléc a `details.retryAfterSeconds`-be kerül, ha a törzsben nincs.
- `fetch`-kivétel (hálózat): `ApiError(0, 'network_error')`. Az `AbortError` változatlanul továbbdobódik (a hívó elnyeli).
- Ha `!skipAuthRedirect`, és a kód `unauthenticated`, `password_change_required` vagy `consent_required`, a regisztrált `onAuthFailure` kezelő fut (a session store regisztrálja, így nincs körkörös import). Ez az egyetlen hely, ahol a 401 irányítást vált ki.
- `download`: a `Content-Disposition` `filename=` értékét adja vissza; a `utils/download.ts` `saveBlob(blob, filename)` menti (ideiglenes `<a download>` és `URL.createObjectURL`).

`errors.ts`:

```ts
export type ErrorCode = /* every code of section 5.3 */ | 'network_error';
export type FieldCode = /* every field code of section 5.3 */;
export class ApiError extends Error {
	constructor(
		readonly status: number,
		readonly code: ErrorCode,
		readonly fields: Record<string, FieldCode> = {},
		readonly details: Record<string, unknown> = {}
	) { super(code); }
	field(name: string): FieldCode | undefined;
	is(...codes: ErrorCode[]): boolean;
}
```

`types.ts`: az 5.5 és az 5.8.1 TypeScript-blokkja szó szerint. A végpontmodulok és függvényeik: 5.14.

### 8.4 Komponensek

`brand/` (F0): a GRAPHICS.md 10.13 `Logo`, 10.15 `Panel`, 10.14 `Metric` (a `tracking-normal` osztállyal), 10.17 `StatusBadge`, 10.19 `TickRule` forrása szó szerint, két változtatással: a kommentek angolok, és a `StatusBadge` alapszavai a `status.*` kulcsokból jönnek (`t('status.online')` stb.).

`app/` (F0):

| Komponens | Feladat |
|---|---|
| `AppShell.svelte` | `Sidebar.Provider` + `AppSidebar` + `Topbar` + `<main id="main" tabindex="-1">`; ugrólink „Ugrás a tartalomra”; benne a `CaptureController` és a `ShortcutsDialog` (F2) |
| `AppSidebar.svelte` | Oldalsáv (8.7.1) |
| `Topbar.svelte` | Felső sáv (8.7.1) |
| `PageHeader.svelte` | `heading-1` cím, `body-sm` alcím, jobbra igazított `actions` snippet |
| `LocaleSwitcher.svelte` | `globe` ikongomb + `DropdownMenu.RadioGroup`, a nyelvek saját nyelvükön (Magyar, English, Deutsch, `lang` attribútummal). `variant="inline"`: `ToggleGroup` HU/EN/DE felirattal és teljes nevű `aria-label`-lel (bejelentkező és kapuoldal) |
| `ThemeToggle.svelte` | `sun`/`moon` ikongomb, `aria-label` „Téma váltása”; bejelentkezve a preferenciába is ment |
| `StateAlert.svelte` | GRAPHICS.md 10.1 Alert: `kind: 'critical' \| 'warning' \| 'success' \| 'info'`, cím, szöveg, opcionális gomb; kritikusnál `role="alert"` |
| `ErrorAlert.svelte` | `ApiError` → kritikus `StateAlert`: cím `errors.<code>.title`, szöveg `errors.<code>.text` |
| `FieldError.svelte` | Mező alatti hiba: `circle-alert` 14 px `destructive` + szöveg (`fields.byField.<field>.<code>`, különben `fields.<code>`); `id` az `aria-describedby`-hoz |
| `FormField.svelte` | Címke + mező snippet + súgó + `FieldError`; `aria-invalid` és `aria-describedby` |
| `PasswordInput.svelte` | Jelszómező megjelenítés-kapcsolóval (`eye`/`eye-off` ghost ikongomb, `aria-pressed`) |
| `ConfirmDialog.svelte` | `AlertDialog` (GRAPHICS.md 10.8): kérdő cím, következményt kimondó leírás, „Mégse” `outline`, a tárgyat megnevező `destructive` gomb; `busy` állapotban `Spinner`; opcionális kiegészítő mezők snippetben |
| `EmptyState.svelte` | shadcn `Empty`: 24 px `muted-foreground` ikon, `heading-3` cím, `body-sm` szöveg, opcionális művelet |
| `LoadingRows.svelte` | `Skeleton` sorok |
| `TagChip.svelte` | 22 px magas, 1 px `border`, `rounded-sm`, belső margó `0 6px`, rés 6 px, 8×8 px lekerekítés nélküli színnégyzet a `tagSwatchClass` szerint, `body-sm` `foreground` szöveg; `as="button"` esetén `aria-pressed`, aktívan `bg-accent` |
| `TagPicker.svelte` | `Popover` + `Command` többválasztós címkeválasztó kereséssel, „„{name}” címke létrehozása” sorral (a `tags` store `create`), legfeljebb 20 kijelölés; minden szövege a `common.tagPicker.*` kulcsokból (`search`, `create`, `empty`, `limit`) |
| `Favicon.svelte` | `iconUrl` → 16/20/32 px `<img loading="lazy" decoding="async" alt="">` `rounded-sm`; hiány vagy hiba: Lucide `link` ikon `muted-foreground` |
| `CopyButton.svelte` | `copy` ikon, siker után 1,5 s-ig `check`, toast; `onCopied` visszahívás (a részletek lap URL-jénél `links.visit`, 5.9.5) |
| `RelativeTime.svelte` | `<time datetime>` relatív szöveggel, `title`-ben a teljes dátum és idő |
| `QrCode.svelte` | `uqr` mátrixból egyetlen `<path>` SVG, `fill="currentColor"`, 4 modul csendes zóna; tároló `bg-mist-50 text-ink-950` (2.3 G7), `role="img"` + `aria-label` |
| `LineTabs.svelte`, `DataTableHead.svelte`, `DataTableShell.svelte` | A GRAPHICS.md 10.18 és 10.7 osztálylistái; a táblázat héja `Card` + `overflow-x-auto` |
| `KeyboardShortcut.svelte` | `Kbd` csoport a platformnak megfelelő módosítóval (`Ctrl` vagy `⌘`) |

`tags/colors.ts` (F0), az egyetlen színleképezés:

```ts
export const TAG_COLORS = ['blue', 'ochre', 'green', 'violet', 'rose', 'neutral', 'contrast'] as const;
export const tagSwatchClass: Record<TagColor, string> = {
	blue: 'bg-chart-1',
	ochre: 'bg-chart-2',
	green: 'bg-chart-3',
	violet: 'bg-chart-4',
	rose: 'bg-chart-5',
	neutral: 'bg-muted-foreground',
	contrast: 'bg-foreground'
};
```

### 8.5 i18n

#### 8.5.1 Típusok és futásidő

Fájlok: `messages/<locale>/<namespace>.ts` (`export default { … }`; a magyar a típusforrás), `messages/<locale>/index.ts` (aggregátor), `types.ts`, `index.ts`, `plural.ts`, `format.ts`, `detect.ts`, `errors.ts` (ApiError → szöveg).

```ts
import type hu from './messages/hu/index.js';

/** A plural message: selected by Intl.PluralRules; `zero` overrides count === 0. */
export type Plural = { zero?: string; one: string; other: string };

/** Same shape as the Hungarian catalogue, with every leaf widened to string / Plural. */
export type Shape<T> = { [K in keyof T]: T[K] extends string ? string : T[K] extends Plural ? Plural : Shape<T[K]> };
export type Catalogue = Shape<typeof hu>;

/** Dot paths to every leaf ("links.toolbar.search"); a Plural object counts as a leaf. */
type Leaves<T, P extends string = ''> = {
	[K in keyof T & string]: T[K] extends string | Plural ? `${P}${K}` : Leaves<T[K], `${P}${K}.`>;
}[keyof T & string];
export type MessageKey = Leaves<typeof hu>;
export type Locale = 'hu' | 'en' | 'de';
export type Params = Record<string, string | number | Date>;
```

Az angol és a német modul: `export default { … } satisfies Shape<typeof huLinks>;` — hiányzó vagy fölösleges kulcs fordítási hiba.

```ts
class I18n {
	locale = $state<Locale>('hu');
	setLocale(next: Locale, opts?: { persist?: boolean }): void; // localStorage 'kxm-locale', <html lang>
}
export const i18n: I18n;
export function t(key: MessageKey, params?: Params): string;
/** Runtime check for keys built from server codes (errors, fields.byField, audit, fetch.errors). */
export function hasMessage(key: string): key is MessageKey;
/** t() for a dynamic key: the fallback (default: the key itself, shown in mono by the caller) when missing. */
export function tDynamic(key: string, params?: Params, fallback?: string): string;
export const fmt: Formatters; // format.ts, reads i18n.locale
```

Dinamikus kulcs csak a `hasMessage`/`tDynamic` páron át képződik: `FieldError` (`fields.byField.<field>.<code>`, különben `fields.<code>`), `ErrorAlert` (`errors.<code>.title`/`text`, ismeretlen kódnál `errors.internal_error`), `AuditTable` (`audit.events.<event>` és `audit.details.*`, ismeretlennél a nyers kód mono betűvel), `fetch.errors.<code>`. Sablonliterállal összerakott kulcs a `t()`-nek nem adható át (a típus nem engedi).

- A `t` a `i18n.locale` `$state`-et olvassa, ezért a sablonban hívva a nyelvváltás oldalújratöltés nélkül újrarajzol.
- Helyőrző: `{name}`. A `number` paramétert a `fmt.number`, a `Date` paramétert a `fmt.date` formázza; a `string` változatlan. Ismeretlen helyőrző fejlesztői módban figyelmeztet.
- Plural: `count` kötelező; `count === 0` és van `zero` → `zero`; különben `Intl.PluralRules(locale).select(count)` (`one`/`other`), nyelvenként gyorsítótárazva.
- Tartalék: hiányzó kulcs → magyar szöveg, végül maga a kulcs; fejlesztői módban `console.warn`.
- Mindhárom katalógus statikusan importált; a váltás szinkron.
- Nyelvválasztás sorrendje: bejelentkezve a `Me.preferences.locale`; előtte a `localStorage` `kxm-locale`, különben a `navigator.languages` első támogatott eleme (`hu`, `en`, `de` előtag), különben `hu`. Ha a felhasználó a bejelentkező oldalon kifejezetten nyelvet választott, és a sikeres bejelentkezés után a szerveroldali preferencia eltér, a felület `PATCH /api/me/preferences {locale}`-lal a választott nyelvet menti (a kifejezett döntés nyer).
- Nyelvváltás bejelentkezve: `i18n.setLocale` azonnal, majd a preferences store ment; hibánál toast, a nyelv marad.
- A nyelvválasztó mindhárom felületen elérhető: bejelentkező és kapuoldal (`LocaleSwitcher variant="inline"`), felhasználói és admin felület (felső sáv), valamint a Beállítások › Megjelenítés fül.

#### 8.5.2 Formázók (`format.ts`, GRAPHICS.md 1.8)

| Függvény | hu | en | de |
|---|---|---|---|
| `fmt.number(n, {fraction?})` | `Intl.NumberFormat('hu', {useGrouping: 'min2'})`, csoportelválasztó U+00A0: `1284`, `12 480`, `24,8` | `Intl.NumberFormat('en-GB')`: `1,284`, `24.8` | `Intl.NumberFormat('de-DE')`: `1.284`, `24,8` |
| `fmt.date(d)` | `2026. 11. 05.` | `5 Nov 2026` | `05.11.2026` |
| `fmt.time(d)` | `14:32:08` | `14:32:08` | `14:32:08` |
| `fmt.dateTime(d)` | `2026. 11. 05. 14:32` | `5 Nov 2026, 14:32` | `05.11.2026, 14:32` |
| `fmt.relative(d)` | `Intl.RelativeTimeFormat` `numeric: 'auto'`: `ma`, `tegnap`, `3 napja`; 1 órán belül `perce`, `órája` | `today`, `yesterday`, `3 days ago` | `heute`, `gestern`, `vor 3 Tagen` |
| `fmt.duration(days)` | `{count} nap` (`common.units.days` plural kulcsból) | `{count} days` | `{count} Tage` |
| `fmt.missing` | `–` (U+2013) minden hiányzó értékre | | |

- A `fmt.number` minden kimenetben a U+0020 és U+202F csoportelválasztót U+00A0-ra cseréli. Érték és mértékegység között a fordítás ` ` escape-et tartalmaz.
- Az időpontok a böngésző időzónájában jelennek meg; a szerver UTC-t ad.
- Oszlopba rendezett szám mindig `data` vagy `font-mono` osztályú (GRAPHICS.md 3.6).

#### 8.5.3 Névterek

| Névtér | Tulajdonos | Tartalom |
|---|---|---|
| `common` | F0 | `actions.{save,cancel,close,delete,edit,copy,copied,retry,back,done,confirm,download,upload,clear,more,open}`, `states.{loading,loadingMore,empty,saving}`, `time.{today,yesterday,never}`, `units.{days,minutes,hours}` (plural), `missing`, `skipToContent`, `appName`, `pageTitleSuffix`, `tagPicker.{search,create,empty,limit}`, `stepUp.{label,help}` (az admin megerősítő jelszómezője) |
| `nav` | F0 | `groups.{collection,frequentTags,account,admin}`, `links`, `tags`, `settings`, `settingsTabs.{profile,security,display,privacy}`, `adminUsers`, `adminSettings`, `adminAudit`, `logout`, `toggleSidebar`, `breadcrumb.{collection,admin,settings}`, `search.{label,placeholder}`, `theme.toggle`, `locale.{label,hu,en,de}`, `userMenu`, `privacyNotice`, `errorPage.{notFound,forbidden}.{title,text,action}` |
| `errors` | F0 | minden 5.3 kód (`network_error`-ral együtt), kódonként `title` és `text` (8.5.5) |
| `fields` | F0 | mezőkódok általánosan és `byField.<field>.<code>` felülírások (8.5.6) |
| `roles` | F0 | `admin`, `developer`, `customer` egyes számban, `plural.{admin,developer,customer}` többes számban |
| `status` | F0 | `online`, `warning`, `critical`, `offline`, `info` alapszavai; `active`, `inactive`, `locked`, `enabled`, `disabled` |
| `auth` | F1 | `login.*`, `totp.*`, `recovery.*`, `logout.*`, `expired`, `accountDeleted`, `passwordChange.*`, `passwordPolicy` |
| `landing` | F1 | 8.8.3 |
| `privacy` | F1 | `title`, `version`, `sections.{controller,data,purposes,legalBasis,recipients,transfers,automatedDecisions,retention,storage,fetching,adminAccess,rights}.{title,body}`, `rights.{access,rectification,erasure,restriction,portability,objection,complaint}`, `consent.*`, `backToApp` |
| `links` | F2 | `pageTitle`, `summary` (plural), `summaryArchived` (plural), `toolbar.*`, `scope.*`, `sort.*`, `view.*`, `fields.*`, `archivedToggle`, `filters.{untagged,more,match.any,match.all,clear,label}`, `groups.{pinned,others}`, `row.*`, `actions.*`, `dialog.*`, `details.*`, `share.*`, `delete.*`, `importExport.*`, `expiry.*`, `empty.*`, `toast.*`, `shortcuts.*` |
| `capture` | F2 | `overlay.{title,text}`, `toast.{saved,exists,multiple,hidden,tooMany,openAction,editAction,showAction}` |
| `fetch` | F2 | `status.{pending,ok,failed,blocked}`, `errors.<fetch_error>` (8.5.7) |
| `tags` | F3 | `pageTitle`, `summary`, `table.*`, `dialog.*`, `delete.*`, `empty.*` |
| `tagColors` | F3 | a hét szín neve (4.6) |
| `settings` | F3 | `pageTitle`, `profile.*`, `display.*` |
| `security` | F5 | `password.*`, `twoFactor.*`, `recovery.*` (benne `recovery.file.{title,intro,generatedAt,warning}` a letölthető .txt sorai), `sessions.*`, `privacy.*`, `deleteAccount.*` |
| `admin` | F4 | `users.*`, `userDialog.*`, `tempPassword.*`, `settings.*` |
| `audit` | F4 | `pageTitle`, `subtitle`, `filters.*`, `table.*`, `events.<event>` (8.5.8), `targets.<type>` (`user`, `link`, `session`, `settings`, `tag`), `details.keys.<key>` és `details.values.<key>.<value>` (8.5.8) |

Mintamodul (`messages/hu/links.ts`, részlet):

```ts
export default {
	pageTitle: 'Linkek',
	summary: { zero: 'Nincs link', one: '{count} link, ebből {pinned} kitűzött', other: '{count} link, ebből {pinned} kitűzött' },
	actions: {
		add: 'Link felvétele',
		open: 'Megnyitás',
		copyUrl: 'URL másolása',
		copyMarkdown: 'Másolás Markdownként',
		details: 'Részletek',
		edit: 'Szerkesztés',
		pin: 'Kitűzés',
		unpin: 'Kitűzés megszüntetése',
		archive: 'Archiválás',
		restore: 'Visszaállítás',
		share: 'Megosztás',
		refresh: 'Cím és ikon frissítése',
		delete: 'Törlés'
	},
	expiry: {
		label: 'Archiválás {date} körül, ha addig nem nyitod meg és nem másolod',
		daysLeft: { one: '{count} nap', other: '{count} nap' },
		today: 'ma'
	}
};
```

#### 8.5.4 Írásszabályok

- Mondatkezdő nagybetű; csupa nagybetűt a fordítás nem tartalmaz (a `kx-eyebrow` és a `th` alakít).
- Gomb: ige és tárgy („Link felvétele”, „Add link”, „Link hinzufügen”). Az ige a folyamatban azonos: „Link mentése” gomb → „Link mentve” toast.
- Hibaüzenet: mi történt, és mit tegyen a felhasználó (`title` és `text`).
- Visszafordíthatatlan műveletnél: kérdő cím, következményt kimondó leírás, a tárgyat megnevező gomb.
- Emoji, felkiáltójel és a GRAPHICS.md 1.5 tiltott szavai nincsenek.
- Szám és mértékegység között, valamint a magyar ezres tagolásban U+00A0. Számot a fordítás nem formáz: a `{count}` helyőrzőt a `t()` formázza. Dátum soha nincs beégetve; `{date}` helyőrző.
- Szakkifejezések: magyarul „link”, „címke” (tag), „cím” (a letöltött oldalcím), „leírás”, „kitűzés”, „archiválás”, „megosztás”, „munkamenet”, „kétlépcsős azonosítás”, „helyreállító kód”; angolul „tag”, „pin”; németül „Label” (a „Tag” a „nap” jelentéssel ütközne), „anheften”, „archivieren”, „Sitzung”, „Zwei-Faktor-Anmeldung”, „Wiederherstellungscode”.
- Szerepkörök: Adminisztrátor / Administrator / Administrator; Fejlesztő / Developer / Entwickler; Ügyfél / Customer / Kunde.
- Címkeszínek: Kék, Okker, Zöld, Ibolya, Rózsa, Semleges, Kontraszt / Blue, Ochre, Green, Violet, Rose, Neutral, Contrast / Blau, Ocker, Grün, Violett, Rosa, Neutral, Kontrast.
- A felület tegező; a német is („du”, 2.3 G9).

#### 8.5.5 Hibakódok fordítása (`errors.<code>.title` / `errors.<code>.text`)

Az `ErrorAlert` mindkettőt mutatja; a toast és a mezőközeli üzenet a kettőt egy mondatként (`title` + szóköz + `text`).

| Kód | hu | en | de |
|---|---|---|---|
| `endpoint_not_found` | A funkció nem érhető el. / Frissítsd az oldalt. | This function is not available. / Reload the page. | Diese Funktion ist nicht verfügbar. / Lade die Seite neu. |
| `invalid_json` | A kérés hibás formátumú. / Frissítsd az oldalt, és próbáld újra. | The request was malformed. / Reload the page and try again. | Die Anfrage war fehlerhaft. / Lade die Seite neu und versuche es erneut. |
| `invalid_id` | Hibás azonosító. / Nyisd meg újra a listából. | Invalid identifier. / Open the item again from the list. | Ungültige Kennung. / Öffne den Eintrag erneut aus der Liste. |
| `invalid_query` | Hibás szűrőbeállítás. / Töröld a szűrőket, és próbáld újra. | Invalid filter. / Clear the filters and try again. | Ungültiger Filter. / Setze die Filter zurück und versuche es erneut. |
| `validation_failed` | Néhány mező hibás. / Javítsd a jelölt mezőket. | Some fields are invalid. / Correct the marked fields. | Einige Felder sind ungültig. / Korrigiere die markierten Felder. |
| `payload_too_large` | A fájl nagyobb 2 MB-nál. / Válassz kisebb fájlt. | The file is larger than 2 MB. / Choose a smaller file. | Die Datei ist größer als 2 MB. / Wähle eine kleinere Datei. |
| `not_found` | Az elem nem található. / Lehet, hogy törölték, vagy már nem látod. Frissítsd a listát. | The item was not found. / It may have been deleted or is no longer visible to you. Refresh the list. | Der Eintrag wurde nicht gefunden. / Er wurde vielleicht gelöscht oder ist für dich nicht mehr sichtbar. Aktualisiere die Liste. |
| `unauthenticated` | A munkamenet lejárt. / Jelentkezz be újra. | Your session has expired. / Sign in again. | Deine Sitzung ist abgelaufen. / Melde dich erneut an. |
| `forbidden` | Ehhez nincs jogosultságod. / Fordulj a rendszergazdához. | You do not have permission for this. / Contact an administrator. | Dafür fehlt dir die Berechtigung. / Wende dich an einen Administrator. |
| `csrf_rejected` | A biztonsági ellenőrzés elutasította a kérést. / Frissítsd az oldalt, és próbáld újra. | A security check rejected the request. / Reload the page and try again. | Eine Sicherheitsprüfung hat die Anfrage abgelehnt. / Lade die Seite neu und versuche es erneut. |
| `host_mismatch` | A kérés nem a Kinex Mark beállított címére érkezett. / Nyisd meg az alkalmazást a megszokott címén, vagy szólj a rendszergazdának. | The request did not reach the configured Kinex Mark address. / Open the app at its usual address or contact an administrator. | Die Anfrage hat nicht die eingestellte Kinex-Mark-Adresse erreicht. / Öffne die App unter ihrer gewohnten Adresse oder wende dich an einen Administrator. |
| `password_change_required` | Jelszócsere szükséges. / A folytatáshoz állíts be új jelszót. | Password change required. / Set a new password to continue. | Passwortänderung erforderlich. / Lege ein neues Passwort fest, um fortzufahren. |
| `consent_required` | Az adatkezelési tájékoztató elfogadása szükséges. / A folytatáshoz fogadd el. | The privacy notice must be accepted. / Accept it to continue. | Die Datenschutzhinweise müssen bestätigt werden. / Bestätige sie, um fortzufahren. |
| `invalid_credentials` | Hibás felhasználónév vagy jelszó. / Ellenőrizd, és próbáld újra. | Incorrect username or password. / Check them and try again. | Benutzername oder Passwort ist falsch. / Prüfe die Eingaben und versuche es erneut. |
| `account_disabled` | A fiók inaktív. / Fordulj a rendszergazdához. | This account is disabled. / Contact an administrator. | Dieses Konto ist deaktiviert. / Wende dich an einen Administrator. |
| `too_many_attempts` | Túl sok sikertelen próbálkozás. / Próbáld újra {minutes} perc múlva. | Too many failed attempts. / Try again in {minutes} min. | Zu viele fehlgeschlagene Versuche. / Versuche es in {minutes} Min. erneut. |
| `challenge_invalid` | A belépési kérés lejárt. / Jelentkezz be újra. | The sign-in request has expired. / Sign in again. | Die Anmeldeanfrage ist abgelaufen. / Melde dich erneut an. |
| `invalid_totp` | Hibás vagy már felhasznált kód. / Írd be az alkalmazásban látható új kódot. | The code is wrong or already used. / Enter the new code shown in your app. | Der Code ist falsch oder wurde schon verwendet. / Gib den neuen Code aus deiner App ein. |
| `current_password_invalid` | A jelenlegi jelszó hibás. / Írd be újra. | The current password is incorrect. / Enter it again. | Das aktuelle Passwort ist falsch. / Gib es erneut ein. |
| `totp_already_enabled` | A kétlépcsős azonosítás már be van kapcsolva. / Frissítsd az oldalt. | Two-factor sign-in is already on. / Reload the page. | Die Zwei-Faktor-Anmeldung ist bereits aktiv. / Lade die Seite neu. |
| `totp_not_enabled` | A kétlépcsős azonosítás nincs bekapcsolva. / Frissítsd az oldalt. | Two-factor sign-in is off. / Reload the page. | Die Zwei-Faktor-Anmeldung ist nicht aktiv. / Lade die Seite neu. |
| `totp_setup_required` | A beállítás lejárt. / Kezdd újra a bekapcsolást. | The setup has expired. / Start the setup again. | Die Einrichtung ist abgelaufen. / Starte sie erneut. |
| `link_duplicate` | Ez a link már megvan a gyűjteményedben. / Nyisd meg a meglévőt. | This link is already in your collection. / Open the existing one. | Dieser Link ist bereits in deiner Sammlung. / Öffne den vorhandenen. |
| `link_limit_reached` | Elérted a 10 000 linkes korlátot. / Törölj régi linkeket, mielőtt újat veszel fel. | You have reached the limit of 10,000 links. / Delete old links before adding new ones. | Du hast das Limit von 10.000 Links erreicht. / Lösche alte Links, bevor du neue hinzufügst. |
| `not_owner` | Ezt csak a link tulajdonosa módosíthatja. / A saját címkéidet és kitűzésedet szabadon állíthatod. | Only the owner can change this link. / You can still set your own tags and pin. | Nur der Eigentümer kann diesen Link ändern. / Eigene Labels und Anheften kannst du trotzdem setzen. |
| `share_not_allowed` | A szerepköröd nem oszthat meg linket. / Fordulj egy fejlesztőhöz vagy a rendszergazdához. | Your role cannot share links. / Ask a developer or an administrator. | Deine Rolle darf keine Links teilen. / Wende dich an einen Entwickler oder Administrator. |
| `share_duplicate` | Ezzel a címzettel már meg van osztva. / Válassz másik címzettet. | Already shared with this recipient. / Choose another recipient. | Bereits mit diesem Empfänger geteilt. / Wähle einen anderen Empfänger. |
| `no_url_found` | Nem található URL. / Húzz vagy illessz be egy http vagy https kezdetű linket. | No URL found. / Drop or paste a link starting with http or https. | Keine URL gefunden. / Zieh oder füge einen Link ein, der mit http oder https beginnt. |
| `import_invalid` | A fájl nem értelmezhető. / Kinex Mark JSON-exportot vagy böngészős HTML-könyvjelzőfájlt válassz. | The file could not be read. / Choose a Kinex Mark JSON export or a browser HTML bookmark file. | Die Datei konnte nicht gelesen werden. / Wähle einen Kinex-Mark-JSON-Export oder eine HTML-Lesezeichendatei des Browsers. |
| `import_too_many_entries` | A fájlban 5000-nél több bejegyzés van. / Bontsd több fájlra, és importáld őket egymás után. | The file has more than 5,000 entries. / Split it into several files and import them one after another. | Die Datei enthält mehr als 5.000 Einträge. / Teile sie auf mehrere Dateien auf und importiere sie nacheinander. |
| `tag_duplicate` | Ilyen nevű címke már van. / Válassz másik nevet. | A tag with this name already exists. / Choose another name. | Ein Label mit diesem Namen gibt es bereits. / Wähle einen anderen Namen. |
| `tag_limit_reached` | Elérted az 500 címkés korlátot. / Törölj nem használt címkéket. | You have reached the limit of 500 tags. / Delete unused tags. | Du hast das Limit von 500 Labels erreicht. / Lösche ungenutzte Labels. |
| `username_taken` | A felhasználónév foglalt. / Válassz másikat. | This username is taken. / Choose another one. | Dieser Benutzername ist vergeben. / Wähle einen anderen. |
| `last_admin` | Ez az utolsó aktív adminisztrátor. / Előbb nevezz ki egy másik adminisztrátort. | This is the last active administrator. / Appoint another administrator first. | Das ist der letzte aktive Administrator. / Ernenne zuerst einen weiteren Administrator. |
| `self_modification` | A saját fiókodat itt nem módosíthatod. / A szerepkörödet egy másik adminisztrátor állíthatja, a fiókod törlése a Beállítások oldalon érhető el. | You cannot change your own account here. / Another administrator can change your role; delete your account under Settings. | Dein eigenes Konto kannst du hier nicht ändern. / Deine Rolle kann ein anderer Administrator ändern, dein Konto löschst du in den Einstellungen. |
| `setting_locked` | A beállítást környezeti változó rögzíti. / A módosításhoz a szerver konfigurációját kell változtatni: {keys}. | This setting is fixed by an environment variable. / Change the server configuration to modify it: {keys}. | Diese Einstellung ist durch eine Umgebungsvariable festgelegt. / Ändere dafür die Serverkonfiguration: {keys}. |
| `consent_version_mismatch` | Közben új tájékoztató jelent meg. / Olvasd el, és fogadd el újra. | A new notice has been published. / Read it and accept it again. | Inzwischen gibt es neue Hinweise. / Lies sie und bestätige sie erneut. |
| `confirmation_mismatch` | A megerősítés nem egyezik a felhasználóneveddel. / Írd be pontosan. | The confirmation does not match your username. / Type it exactly. | Die Bestätigung stimmt nicht mit deinem Benutzernamen überein. / Gib ihn genau ein. |
| `not_implemented` | A funkció még nem érhető el. / Próbáld újra egy későbbi verzióban. | This function is not available yet. / Try again in a later version. | Diese Funktion ist noch nicht verfügbar. / Versuche es in einer späteren Version. |
| `internal_error` | Váratlan szerverhiba történt. / Próbáld újra később. | An unexpected server error occurred. / Try again later. | Ein unerwarteter Serverfehler ist aufgetreten. / Versuche es später erneut. |
| `network_error` | A szerver nem érhető el. / Ellenőrizd a hálózati kapcsolatot. | The server cannot be reached. / Check your network connection. | Der Server ist nicht erreichbar. / Prüfe deine Netzwerkverbindung. |

A `too_many_attempts` `{minutes}` értéke `ceil(retryAfterSeconds / 60)`, legalább 1. A `setting_locked` `{keys}` a `details.keys` vesszővel összefűzve.

#### 8.5.6 Mezőkódok (`fields.<code>`)

| Kód | hu | en | de |
|---|---|---|---|
| `required` | Kötelező mező. | This field is required. | Pflichtfeld. |
| `too_short` | Túl rövid. | Too short. | Zu kurz. |
| `too_long` | Túl hosszú. | Too long. | Zu lang. |
| `invalid` | Érvénytelen érték. | Invalid value. | Ungültiger Wert. |
| `invalid_format` | Hibás formátum. | Invalid format. | Ungültiges Format. |
| `not_allowed` | Ez az érték nem megengedett. | This value is not allowed. | Dieser Wert ist nicht erlaubt. |
| `out_of_range` | Az érték a megengedett tartományon kívül esik. | The value is out of range. | Der Wert liegt außerhalb des zulässigen Bereichs. |
| `unknown` | Ismeretlen elem. Frissítsd a listát. | Unknown item. Refresh the list. | Unbekannter Eintrag. Aktualisiere die Liste. |
| `duplicate` | Ez már létezik. | This already exists. | Das gibt es bereits. |
| `credentials_in_url` | Az URL nem tartalmazhat felhasználónevet vagy jelszót. | The URL must not contain a username or password. | Die URL darf keinen Benutzernamen und kein Passwort enthalten. |
| `incorrect` | Hibás érték. | Incorrect value. | Falscher Wert. |
| `password_too_short` | Legalább 12 karakter kell. | Use at least 12 characters. | Mindestens 12 Zeichen. |
| `password_too_long` | Legfeljebb 128 karakter lehet. | Use at most 128 characters. | Höchstens 128 Zeichen. |
| `password_contains_username` | A jelszó nem tartalmazhatja a felhasználónevet. | The password must not contain the username. | Das Passwort darf den Benutzernamen nicht enthalten. |
| `password_too_simple` | Legalább 6 különböző karakter kell. | Use at least 6 different characters. | Mindestens 6 verschiedene Zeichen. |
| `password_reused` | Az új jelszó nem egyezhet a jelenlegivel. | The new password must differ from the current one. | Das neue Passwort muss sich vom aktuellen unterscheiden. |

Mezőnkénti felülírások (`fields.byField.<field>.<code>`, mindhárom nyelven): `url.invalid_format` („Írj be teljes URL-t, például https://example.com.” / „Enter a full address, for example https://example.com.” / „Gib eine vollständige Adresse ein, zum Beispiel https://example.com.”); `url.not_allowed` („Csak http és https URL menthető.” / „Only http and https addresses can be saved.” / „Nur http- und https-Adressen können gespeichert werden.”); `username.invalid_format` („Csak betű, szám, pont, kötőjel és aláhúzás.” / „Letters, digits, dot, hyphen and underscore only.” / „Nur Buchstaben, Ziffern, Punkt, Bindestrich und Unterstrich.”); `tagIds.too_long` („Legfeljebb 20 címke.” / „At most 20 tags.” / „Höchstens 20 Labels.”); `privacyControllerEmail.invalid_format` („Érvényes e-mail-címet adj meg.” / „Enter a valid e-mail address.” / „Gib eine gültige E-Mail-Adresse ein.”); az `out_of_range` a rendszerbeállításokban `{min}` és `{max}` helyőrzős változatot kap („Az érték {min} és {max} közé essen.”), a határokat a felület az 5.5 `SystemSettings` megjegyzéseiből rögzíti.

#### 8.5.7 Letöltési hibák (`fetch.errors.<code>`)

| Kód | hu | en | de |
|---|---|---|---|
| `timeout` | Időtúllépés | Timed out | Zeitüberschreitung |
| `dns` | A domain nem található | Domain not found | Domain nicht gefunden |
| `connection` | Kapcsolódási hiba | Connection error | Verbindungsfehler |
| `tls` | Tanúsítványhiba | Certificate error | Zertifikatsfehler |
| `http_status` | A webhely hibát adott vissza | The site returned an error | Die Website hat einen Fehler gemeldet |
| `too_large` | Az oldal túl nagy | The page is too large | Die Seite ist zu groß |
| `not_html` | Nem weboldal (nincs cím) | Not a web page (no title) | Keine Webseite (kein Titel) |
| `too_many_redirects` | Túl sok átirányítás | Too many redirects | Zu viele Weiterleitungen |
| `blocked_address` | Belső hálózati cím, a letöltés tiltott | Internal network address, fetching is blocked | Interne Netzwerkadresse, Abruf gesperrt |
| `blocked_port` | Tiltott port | Port not allowed | Port nicht erlaubt |

#### 8.5.8 Audit események (`audit.events.<event>`)

| Esemény | hu | en | de |
|---|---|---|---|
| `bootstrap.admin_created` | Első adminisztrátor létrehozva | Initial administrator created | Erster Administrator angelegt |
| `auth.login.success` | Sikeres bejelentkezés | Signed in | Angemeldet |
| `auth.login.failure` | Sikertelen bejelentkezés | Sign-in failed | Anmeldung fehlgeschlagen |
| `auth.login.locked` | Bejelentkezés zárolva | Sign-in locked | Anmeldung gesperrt |
| `auth.logout` | Kijelentkezés | Signed out | Abgemeldet |
| `auth.recovery_code.used` | Helyreállító kód felhasználva | Recovery code used | Wiederherstellungscode verwendet |
| `auth.password.changed` | Jelszó módosítva | Password changed | Passwort geändert |
| `auth.2fa.enabled` | Kétlépcsős azonosítás bekapcsolva | Two-factor sign-in turned on | Zwei-Faktor-Anmeldung aktiviert |
| `auth.2fa.disabled` | Kétlépcsős azonosítás kikapcsolva | Two-factor sign-in turned off | Zwei-Faktor-Anmeldung deaktiviert |
| `auth.2fa.recovery_regenerated` | Új helyreállító kódok | New recovery codes | Neue Wiederherstellungscodes |
| `session.revoked` | Munkamenet lezárva | Session revoked | Sitzung beendet |
| `session.revoked_others` | Többi munkamenet lezárva | Other sessions revoked | Andere Sitzungen beendet |
| `user.created` | Felhasználó létrehozva | User created | Benutzer angelegt |
| `user.updated` | Felhasználó módosítva | User updated | Benutzer geändert |
| `user.deactivated` | Felhasználó inaktiválva | User deactivated | Benutzer deaktiviert |
| `user.activated` | Felhasználó aktiválva | User activated | Benutzer aktiviert |
| `user.deleted` | Felhasználó törölve | User deleted | Benutzer gelöscht |
| `user.password_reset` | Jelszó visszaállítva | Password reset | Passwort zurückgesetzt |
| `user.2fa_reset` | Kétlépcsős azonosítás visszaállítva | Two-factor sign-in reset | Zwei-Faktor-Anmeldung zurückgesetzt |
| `user.unlocked` | Zárolás feloldva | Unlocked | Entsperrt |
| `user.sessions_revoked` | Munkamenetek lezárva | Sessions revoked | Sitzungen beendet |
| `user.self_deleted` | Saját fiók törölve | Own account deleted | Eigenes Konto gelöscht |
| `settings.updated` | Rendszerbeállítás módosítva | Settings updated | Einstellungen geändert |
| `privacy.notice_bumped` | Új tájékoztató-verzió | New privacy notice version | Neue Version der Datenschutzhinweise |
| `privacy.consent_accepted` | Tájékoztató elfogadva | Privacy notice accepted | Datenschutzhinweise bestätigt |
| `gdpr.export` | Adatexport | Data export | Datenexport |
| `link.shared` | Link megosztva | Link shared | Link geteilt |
| `link.unshared` | Megosztás visszavonva | Sharing revoked | Freigabe entfernt |

Ismeretlen esemény: maga a kód mono betűvel.

A „Részletek” (`details`) megjelenítése: minden kulcs felirata `audit.details.keys.<key>` (a 6.10 zárt kulcskészlete: `reason` Ok, `method` Módszer, `scope` Hatókör, `changed` Módosított mezők, `role` Szerepkör, `active` Aktív, `remaining` Megmaradt kódok, `sessionId` Munkamenet, `count` Darab, `version` Verzió, `targetType` Célpont típusa, `targetRole` Célszerepkör, `targetUserId` Célfelhasználó, `generatedPassword` Generált jelszó, `via` Forrás). A felsorolt értékek fordítása `audit.details.values.<key>.<value>`: `reason` (`credentials` Hibás jelszó, `unknown_user` Ismeretlen felhasználónév, `disabled` Inaktív fiók, `totp` Hibás kód), `method` (`password` Jelszó, `totp` Hitelesítő kód, `recovery_code` Helyreállító kód), `scope` (`user` Felhasználó, `ip` IP-cím), `via` (`cli` Parancssor), `targetType` (`all`, `role`, `user` a megosztás szavaival). A `changed` elemei a mezők meglévő feliratait használják (`admin.settings.fields.<key>`, illetve `admin.userDialog.fields.<key>`), a `role` értékei a `roles.*`, a logikai értékek `common`-szavak. Ismeretlen kulcs vagy érték: a nyers kód mono betűvel (`tDynamic`). A `contract.spec.ts` a golden `auditDetailKeys` listájára ellenőrzi a `keys` teljességét (9.6).

#### 8.5.9 Szerződés-ellenőrzés

A `src/lib/i18n/contract.spec.ts` beolvassa az `../internal/api/testdata/contract.json` fájlt (9.6), és mindhárom nyelven ellenőrzi, hogy létezik: minden hibakódra `errors.<code>.title` és `errors.<code>.text`; minden mezőkódra `fields.<code>`; minden auditeseményre `audit.events.<event>`; minden audit-részletkulcsra `audit.details.keys.<key>`; minden letöltési hibára `fetch.errors.<code>`; a 8.7.10 tizenkét tájékoztató-szakaszára `privacy.sections.<s>.title` és `.body`, a hét jogra `privacy.rights.<r>`; minden címkeszínre `tagColors.<color>`; minden szerepkörre `roles.<role>`; minden nyelvre `nav.locale.<locale>`. Így a backend új kódja a frontend fordítása nélkül piros CI-t ad. A `parity.spec.ts` a három katalógus kulcshalmazának azonosságát, a plural-kategóriákat és a helyőrzők egyezését ellenőrzi, és hogy nincs üres szöveg.

### 8.6 `lib/state`: runes store-ok

Minden store egy osztálypéldány `$state` mezőkkel, modulszinten exportálva (`export const session = new SessionStore()`); tesztben a konstruktor külön példányt ad.

`session.svelte.ts` (F0):

```ts
class SessionStore {
	me = $state<Me | null>(null);
	status = $state<'unknown' | 'anonymous' | 'authenticated'>('unknown');
	readonly gate = $derived<'password' | 'consent' | null>(…); // mustChangePassword first, then privacyConsentRequired
	ensure(): Promise<Me | null>;        // single-flight GET /api/auth/me (skipAuthRedirect); 401 -> anonymous
	login(username: string, password: string): Promise<LoginResponse>;
	verifyTotp(input: { challenge: string; code?: string; recoveryCode?: string }): Promise<Me>;
	logout(): Promise<void>;             // POST logout, clear, goto('/login')
	refresh(): Promise<Me>;              // after password change, consent, 2FA change
	setMe(me: Me): void;                 // applies locale + theme from preferences
	clear(reason?: 'expired' | 'logout' | 'deleted'): void;
	can(p: 'share' | 'listUsers' | 'admin'): boolean;
}
```

- A `setMe` alkalmazza a nyelvet (`i18n.setLocale(me.preferences.locale)`, a kifejezett-választás szabályával) és a témát (`setMode(me.preferences.theme)`).
- Az `onAuthFailure` kezelő: `unauthenticated` → `clear('expired')`, `goto('/login?next=…&reason=expired')`; `password_change_required` → `refresh()`, `goto('/change-password')`; `consent_required` → `refresh()`, `goto('/consent')`.

`preferences.svelte.ts` (F0): a `session.me.preferences` nézete. `update(patch)`: optimista beírás, a módosítások 400 ms-os összevonása egyetlen `PATCH /api/me/preferences` kérésbe; a válasz felülírja a helyi állapotot; hibánál visszaállítás és toast. A `theme` és a `locale` mellékhatását a store végzi.

`public-config.svelte.ts` (F0): `GET /api/public/config` egyszer, gyorsítótárazva; `config`, `ensure()`.

`tags.svelte.ts` (F0): `items`, `byId` (`$derived` Map), `load()`, `create(name, color)`, `update`, `remove`, `onTagsChanged(listener): () => void`. A store a links store-t nem importálja (az az F2 fájlja); minden módosítás után értesíti a feliratkozókat, és a `links.svelte.ts` (F2) feliratkozva `reload`-ot hív, mert a `linkCount` és a címkeszűrő változik.

`link-stats.svelte.ts` (F0): az oldalsáv „Linkek” számlálója. `active = $state<number | null>(null)`, `refresh()`: `GET /api/links?limit=1&archived=exclude` (`total`), az AppShell betöltésekor és 60 s-onként fókuszált ablaknál. Az F2 a rögzítés, az archiválás, a visszaállítás, a törlés és az import után hívja a `refresh()`-t.

`links.svelte.ts` (F2):

```ts
class LinksStore {
	items = $state<LinkItem[]>([]);
	total = $state(0);
	counts = $state<{ pinned: number; archived: number }>({ pinned: 0, archived: 0 }); // LinkPage.counts
	status = $state<'idle' | 'loading' | 'loading-more' | 'ready' | 'error'>('idle');
	error = $state<ApiError | null>(null);
	query = $state<LinkQuery>(defaultQuery);      // q, tags, tagMatch, untagged, scope, archived, sort
	highlightId = $state<number | null>(null);
	readonly pinned = $derived(this.items.filter((l) => l.pinned));
	readonly others = $derived(this.items.filter((l) => !l.pinned));
	setQuery(patch: Partial<LinkQuery>): void;    // URL sync + debounced reload
	reload(): Promise<void>;                      // offset 0, limit 200, aborts the previous request
	loadMore(): Promise<void>;                    // next 200 (IntersectionObserver sentinel)
	upsert(link: LinkItem | LinkDetail, opts?: { highlight?: boolean }): void;
	remove(id: number): void;
	pin(id: number, pinned: boolean): Promise<void>;     // optimistic + reorder
	archive(id: number, archived: boolean): Promise<void>;
	visit(id: number): void;                      // one use (open or copy): optimistic useCount + 1, lastUsedAt now, keepalive POST
	watchPending(id: number): void;               // 1.5 s polling, max 20 s (5.9.8)
}
```

- A `query` egyetlen igazságforrása az URL (`/?q=grafana&tags=3,5&match=all&scope=own&archived=include&sort=title&link=12`): a store a `page.url`-ből olvas, és `goto(url, { replaceState: true, keepFocus: true, noScroll: true })`-val ír. A `showArchived` és a `sort` alapértékét a preferencia adja; az URL csak az alapértéktől eltérőt tartalmazza.
- Keresés: 200 ms debounce; címkeváltás, rendezés, hatókör azonnal.
- Optimista műveletek után a betöltött ablak a `compareLinks` (8.7.3) szerint újrarendeződik; a szerver válasza (`LinkItem` vagy `LinkState`) felülírja az elemet. Hibánál visszaállítás és hiba-toast.
- Archiválás `archived=exclude` szűrő mellett: az elem 300 ms múlva kikerül a listából, a toast „Visszavonás” művelettel visszaállítja.
- `watchPending`: egyetlen közös időzítő minden függő azonosítóra, egyszerre legfeljebb 5 `GET /api/links/{id}`; 20 s után abbahagyja.
- `links/query.ts`: `LinkQuery` ↔ URL-paraméterek oda-vissza (`match` ↔ `tagMatch`), alapértékek elhagyása, hibás értékek eldobása.

### 8.7 Útvonalak, kapuk és képernyők

#### 8.7.0 Útvonaltábla és kapuk

| Útvonal | Csoport | Hozzáférés | Elsődleges művelet |
|---|---|---|---|
| `/login` | `(public)` | nyilvános | Bejelentkezés |
| `/privacy` | `(public)` | nyilvános | nincs |
| `/change-password` | `(gate)` | bejelentkezett, nyitott jelszókapu | Jelszó módosítása |
| `/consent` | `(gate)` | bejelentkezett, nyitott hozzájárulás-kapu | Elfogadás és tovább |
| `/` | `(app)` | bejelentkezett | Link felvétele |
| `/tags` | `(app)` | bejelentkezett | Címke felvétele |
| `/settings` | `(app)` | bejelentkezett | átirányít: `/settings/profile` |
| `/settings/profile` | `(app)` | bejelentkezett | Profil mentése |
| `/settings/security` | `(app)` | bejelentkezett | Jelszó módosítása |
| `/settings/display` | `(app)` | bejelentkezett | nincs (azonnali mentés) |
| `/settings/privacy` | `(app)` | bejelentkezett | nincs |
| `/admin` | `(app)/admin` | admin | átirányít: `/admin/users` |
| `/admin/users` | `(app)/admin` | admin | Felhasználó felvétele |
| `/admin/settings` | `(app)/admin` | admin | Beállítások mentése |
| `/admin/audit` | `(app)/admin` | admin | nincs |

Kapuk (`ssr = false`, ezért minden `load` a böngészőben fut; a `redirect` és az `error` a `@sveltejs/kit`-ből):

- `(app)/+layout.ts`: `const me = await session.ensure()`. Ha `null`: `redirect(307, '/login?next=' + encodeURIComponent(url.pathname + url.search))`. Ha `session.gate === 'password'`: `redirect(307, '/change-password')`; ha `'consent'`: `redirect(307, '/consent')`. Visszaadja `{ me }`-t. A `tags` és a `publicConfig` store betöltése itt indul (nem blokkol).
- `(app)/admin/+layout.ts`: `const { me } = await parent()`; ha `!me.permissions.canAdmin`: `error(403, 'forbidden')`. A szerver a 403-at úgyis adja; a kliensoldali kapu a fölösleges kérést és a villanást spórolja meg. `(app)/admin/+page.ts`: `redirect(307, '/admin/users')`.
- `(app)/settings/+page.ts`: `redirect(307, '/settings/profile')`. A `(app)/settings/+layout.svelte` a `LineTabs` négy fülét linkekként rajzolja (Profil, Biztonság, Megjelenítés, Adatvédelem; kulcsok: `nav.settingsTabs.*`), az aktív fül az útvonalból jön.
- `(gate)/+layout.ts`: `ensure()`; `null` → `/login`; ha nincs nyitott kapu → `redirect(307, '/')`. A `/change-password` és a `/consent` a nem neki szóló kapunál a megfelelőre irányít (sorrend: előbb jelszó, utána hozzájárulás, mint a szerveren).
- `(public)/login/+page.ts`: ha az `ensure()` felhasználót ad, a `next` vagy `/` felé irányít, illetve a nyitott kapura; sikeres bejelentkezés után ugyanígy. A `next` ellenőrzése a `utils/navigation.ts` `safeNext(next, origin)` függvényével: `const u = new URL(next, origin)`; csak akkor fogadható el, ha `u.origin === origin`, és a nyers `next` nem tartalmaz `\` karaktert és vezérlőkaraktert (U+0000–U+001F); az eredmény `u.pathname + u.search + u.hash`, különben `/`. (A böngésző a `/\evil.example` alakot `//evil.example`-re normalizálná.)
- Navigáció közbeni lejárat: a kliens `onAuthFailure` kezelője irányít; a bejelentkező oldal `reason=expired` esetén info `StateAlert`-et mutat („A munkamenet lejárt. Jelentkezz be újra.”), `reason=deleted` esetén „A fiókod törlődött.” (GRAPHICS.md 1.2: a felület nem beszél „mi” nevében).
- A szerepkör szerinti elemrejtés (megosztás menüpont, admin menücsoport) a `session.can()`-ból jön; a jogosultság igazi határa a szerver.

Közös állapotszabályok minden adatnézetre:

| Állapot | Megjelenés |
|---|---|
| Betöltés | `LoadingRows` vagy `Skeleton` blokkok; 150 ms alatt nem jelenik meg |
| Üres | `EmptyState`: mi hiányzik, és mi a következő lépés |
| Hiba | `ErrorAlert` a tartalom helyén, „Újrapróbálás” `outline` `sm` gombbal |
| Feltöltött | a tartalom |
| Művelet folyamatban | a gomb `disabled` + `Spinner` a felirat előtt; a felirat nem változik |

Toast (svelte-sonner): jobb alul, 4 s, Lucide-ikon az állapotszínnel, szöveg `foreground`; az ige ugyanaz, mint a gombé.

#### 8.7.1 AppShell (F0)

| Régió | Tartalom | Méretek |
|---|---|---|
| Keret | `Sidebar.Provider`, rács: oldalsáv + fő terület | `grid-template-columns: var(--layout-sidebar) 1fr` asztalon; a fő terület `flex min-w-0 flex-col` |
| Oldalsáv fej | `Logo variant="mark" height={28} class="text-foreground"` + „Kinex Mark” (15/24, súly 500) | 40 px magas sor, belső margó `0 space-2`, alsó margó `space-3` |
| Csoport „Gyűjtemény” | Linkek (`bookmark`, számláló: nem archivált linkek a `linkStats` store-ból, 8.6), Címkék (`tags`, számláló: címkék száma) | menüpont 32 px, 13/20; aktív: `sidebar-accent`, súly 500, ikon `sidebar-primary`; számláló mono 11 px `muted-foreground` |
| Csoport „Gyakori címkék” | a `tags` store első 8 eleme `activeLinkCount` szerint: 8×8 px színnégyzet + név + mono számláló; kattintás: `/?tags=<id>` | ha nincs címke, a csoport nem jelenik meg |
| Csoport „Fiók” | Beállítások (`settings`) | |
| Csoport „Adminisztráció” (csak admin) | Felhasználók (`users`), Rendszerbeállítások (`sliders-horizontal`), Audit napló (`scroll-text`) | |
| Oldalsáv lába | felhasználó-gomb: megjelenített név (vagy felhasználónév) + szerepkör szóval, `chevron-up`; menü: Beállítások, Adatkezelési tájékoztató, Kijelentkezés (`log-out`) | felső keret 1 px `sidebar-border` |
| Felső sáv | mobilon és tableten `Sidebar.Trigger`; morzsamenü („Gyűjtemény › Linkek”, „Adminisztráció › Felhasználók”, „Beállítások › Biztonság”); jobbra keresőmező, `LocaleSwitcher`, `ThemeToggle` | 52 px, belső margó `0 space-6`, alsó keret 1 px `border`; kereső legfeljebb 240 px |
| Tartalom | `<main id="main" tabindex="-1">` | belső margó `space-6` (mobilon `space-4`), rés `space-4` |

- A keresőmező `aria-label` „Keresés a linkek között”, `search` ikon, `Kbd` „/” asztalon. A Linkek oldalon élőben szűr; más oldalon Enterre `/?q=…`-ra navigál.
- A morzsamenü utolsó eleme `aria-current="page"`.
- A `CaptureController` és a `ShortcutsDialog` az AppShellben él, így a rögzítés minden bejelentkezett oldalon működik.

#### 8.7.2 Bejelentkezés (`/login`, F1)

Kompozíció és hero: 8.8. Az űrlaposzlop:

| Elem | Tartalom |
|---|---|
| Cím | `heading-2` „Bejelentkezés” |
| Alcím | `body-sm` `muted-foreground`: „Kinex Mark, a Kinexium linkgyűjteménye.” |
| Info sáv | `reason=expired`, `reason=deleted` vagy kijelentkezés után info `StateAlert` |
| Felhasználónév | `Input`, `autocomplete="username"`, `autocapitalize="off"`, `spellcheck="false"`, automatikus fókusz |
| Jelszó | `PasswordInput`, `autocomplete="current-password"` |
| Hiba | `ErrorAlert` a gomb felett: `invalid_credentials`, `account_disabled`, `too_many_attempts` (hátralévő perc, a gomb a lejáratig tiltva, percenként frissül), `network_error` |
| Elsődleges gomb | „Bejelentkezés”, teljes szélesség |
| Súgó | „Fiókot a rendszergazda hoz létre.” |
| Láb | `LocaleSwitcher variant="inline"`, `ThemeToggle`, „Adatkezelési tájékoztató” szöveges link, verzió mono (`publicConfig.version`) |

Folyamat:

1. Beküldés: `session.login`; mezőhiány esetén a kliens kérés nélkül `required` hibát mutat.
2. `status: 'ok'` → `session.setMe`, majd irányítás: kapu, `safeNext(next, location.origin)` (8.7.0) vagy `/`.
3. `status: 'totp_required'` → ugyanazon a kártyán a 2FA-lépés (`TotpStep`; az útvonal nem változik, a fókusz az első kódmezőre kerül, az élő régió bemondja a címet):
   - Cím „Kétlépcsős azonosítás”; szöveg: „Írd be a hitelesítő alkalmazásod 6 jegyű kódját.”
   - `InputOTP` 6 mezővel, mono, `inputmode="numeric"`, `autocomplete="one-time-code"`; a hatodik számjegynél automatikus beküldés.
   - Elsődleges gomb: „Kód ellenőrzése”.
   - `link` gomb: „Helyreállító kódot használok” → mono `Input` (`xxxxx-xxxxx` minta, kötőjel és szóköz tűrve, kisbetűsítve küldve); vissza: „Hitelesítő kódot használok”.
   - `ghost` gomb: „Vissza a bejelentkezéshez” (a kihívás eldobva).
   - Hibák: `invalid_totp` (a mező kiürül, fókusz vissza), `challenge_invalid` (vissza az 1. lépésre, info alert), `too_many_attempts`.
   - Helyreállító kóddal belépve toast: „Bejelentkeztél egy helyreállító kóddal. Még {count} kódod maradt.” (`Me.recoveryCodesRemaining`); 3 vagy kevesebb esetén `warning` toast: „Generálj új kódokat a Beállítások › Biztonság fülön.”

#### 8.7.3 Linkek (`/`, F2)

Elrendezés:

| Régió | Tartalom |
|---|---|
| Oldalfej | `heading-1` „Linkek”; alcím: „{total} link, ebből {pinned} kitűzött” (`links.summary`, plural; „Archiváltak is” mellett `links.summaryArchived`: „{total} link, ebből {pinned} kitűzött, {archived} archivált”); a `total` a `LinkPage.total`, a `pinned` és az `archived` a `LinkPage.counts` (5.9.1); jobbra `outline` ikongomb-menü „Import és export” (`arrow-down-up`) és az elsődleges gomb „Link felvétele” (`plus`) |
| Eszközsor | `ToggleGroup` hatókör (Mind / Saját / Megosztott velem), rendezés `Select` (Használat szerint / Legújabb elöl / Név szerint), nézetváltó `ToggleGroup` (Lista `list`, Csempe `layout-grid`, csak ikon `aria-label`-lel), „Megjelenítés” `DropdownMenu` (`sliders-horizontal`), `Switch` „Archiváltak is”; asztalon egy sor, `flex-wrap`, rés `space-3` |
| Címkesor | `TagChip as="button"` gombok, legfeljebb 12 (`activeLinkCount` szerint), utána „Címke nélkül” chip és „További címkék ({n})” `ghost` `sm` gomb (`Popover` + `Command`). Legalább 2 aktív címkénél `ToggleGroup` „Bármelyik / Mind” (`tagMatch`). Aktív szűrőnél `link` gomb „Szűrők törlése”. Szövegei: `links.filters.*` |
| Lista | egyetlen `Card` `overflow-hidden`, benne két csoport (Kitűzött, Többi) `kx-eyebrow` csoportcímmel; csoportcím csak `usage` rendezésnél, a kitűzött a többi rendezésnél is elöl |
| Lista vége | láthatatlan figyelő elem a `loadMore`-hoz; töltés közben `Spinner` + „Továbbiak betöltése…”; a végén „Minden link betöltve.” csak 200 fölötti listánál |
| Élő régió | `aria-live="polite"`: találatszám („12 találat”), rögzítés eredménye |

A csoportosítás eszköze a címke (BASE-SYSTEM-PROMPT: „csak a tag-ek alapján”): egy csoport = egy címke linkjei. A csoportot a címkesor chipje, az oldalsáv „Gyakori címkék” eleme és a `/tags` oldal linkszám-hivatkozása nyitja meg (`/?tags=<id>`); a „Címke nélkül” chip a csoporton kívüli linkeket mutatja. Egy link több címkével több csoportban is szerepel. Mappa vagy más hierarchia nincs (1.4).

Megjelenítés menü (`DropdownMenu.CheckboxItem`, azonnal hat, a preferenciába ment): Ikon, URL, Cím, Leírás; elválasztó; „Billentyűparancsok”. Ugyanezek a Beállítások › Megjelenítés fülön is állíthatók.

Import és export menü (`ImportExportMenu`): „Exportálás JSON-ként”, „Exportálás böngészős HTML-ként”, elválasztó, „Importálás fájlból…”. Az import rejtett `<input type="file" accept=".json,.html,.htm">`-et nyit; a típus a kiterjesztésből (`application/json` vagy `text/html`); 2 MiB fölött a kliens kérés nélkül `payload_too_large` hibát ad. Eredmény-dialógus: „Importálás kész”, egyszerű `dl`: Összesen, Új, Már megvolt, Hibás (mono számok jobbra); gomb „Bezárás”.

Állapotok: üres gyűjtemény (`bookmark-plus`: „Még nincs link.” / „Húzz ide egyet, illeszd be a vágólapról (Ctrl V), vagy vedd fel kézzel.”); szűrőre nincs találat (`search-x`: „Nincs találat.” / „Módosítsd a keresést, vagy töröld a szűrőket.” + „Szűrők törlése”); üres archivált nézet („Nincs archivált link.”); hiba (`ErrorAlert` + „Újrapróbálás”); betöltés (8 `Skeleton` sor vagy 6 csempe).

Listasor (`LinkRow`):

```
[pin] [ico] Címke (body 15/24, súly 500)              [■ chip][■ chip][+2]  [12] [3 napja] [▮▮▮▯▯] [📌][⋯]
            másodlagos cím (body-sm muted)
            leírás (body-sm muted, 1 sor)
            grafana.example.net/d/abc (code 13/20 mono muted)
```

| Rész | Szabály |
|---|---|
| Sor | `li.kxm-list-row`, rács `grid-cols-[20px_1fr_auto_auto]` (mobilon `20px 1fr auto`), `gap-x-3`, `py-2 px-4`, legalább 44 px; elválasztó 1 px `border`; hover `bg-accent` 120 ms |
| Ikon | `Favicon` 20 px, csak `showIcon` esetén |
| Címke | `<a href={url} target="_blank" rel="noopener noreferrer">`, `body` súly 500, `truncate`, `title` = teljes URL; megnyitáskor `visit`; archivált sornál `muted-foreground` |
| Másodlagos sorok | a `linkPresentation` `secondary` mezői: cím, leírás (`line-clamp-1`), URL (`code` mono, `truncate`); mobilon legfeljebb kettő |
| Jelölők | kitűzött: `pin` 14 px `text-primary` + rejtett szó; megosztott velem: `users` 14 px, tooltip „Megosztotta: {owner}”; archivált: `Badge variant="outline"` `archive` ikonnal és „Archivált” szóval; függő metaadat: `Spinner` 12 px + tooltip „Cím és ikon letöltése folyamatban” |
| Címkék | `md`-től legfeljebb 3 `TagChip`, szélesség szerint: 1280 px alatt 1, 1280-tól 2, 1536-tól 3; utána `+{n}` chip, tooltipben a többi név. `md`-től a címkeoszlop legalább 12rem (`minmax(12rem,1fr)`), a meta oszlop `minmax(0,auto)` rögzített, nem zsugorodó résekkel, így szűk szélességen csak a chipek engednek (D fázis, R5: 1024 px-en a címke 114 px-re szorult) |
| Meta | `lg`-től: használatszám `data` mono jobbra, tooltip „{count} használat (megnyitás vagy másolás)”; utolsó használat `RelativeTime` (soha: `–`); `ExpiryMeter` |
| Műveletek | `pin` ikongomb (`ghost` `sm`, `pointer-coarse:size-9`), `aria-pressed`, `aria-label` „Kitűzés” / „Kitűzés megszüntetése”; `ellipsis` menügomb; egérrel csak `hover`/`focus-within` alatt látszik, érintőképernyőn mindig |
| Kiemelés | rögzítés vagy duplikátum után `kxm-flash` és `scrollIntoView({block: 'nearest'})` |

Megjelenítési címke (`src/lib/links/presentation.ts`, tiszta függvény):

```ts
export type DisplayFields = { showIcon: boolean; showUrl: boolean; showTitle: boolean; showDescription: boolean };
export type Presentation = {
	label: string;
	labelSource: 'description' | 'title' | 'url';
	secondary: { title?: string; description?: string; url?: string };
};
export const LABEL_URL_MAX = 40;   // longer URLs collapse to the host in the label
export const LINE_URL_MAX = 64;    // longer URLs collapse to the host in the URL line
/** "https://Example.com/a/b/" -> "example.com/a/b": no scheme, no "www.", no trailing slash. */
export function prettyUrl(url: string): string;
/** prettyUrl when it fits in max characters, otherwise the host only. */
export function compactUrl(url: string, host: string, max: number): string;
export function linkPresentation(link: Pick<LinkItem, 'url' | 'host' | 'title' | 'description'>, f: DisplayFields): Presentation;
```

1. `title` és `description` vágva; a leírásból a címkéhez csak az első nem üres sor kerül.
2. Címke: a bekapcsolt mezők közül az első nem üres a leírás (`showDescription`), cím (`showTitle`) sorrendben; ha egyik sincs, `compactUrl(url, host, LABEL_URL_MAX)`, `labelSource: 'url'`. Az URL-tartalék mindig érvényes, a címke soha nem üres. Ez a BASE-SYSTEM-PROMPT precedenciája (leírás, különben cím, különben URL, hosszú URL helyett a domain) a felhasználó kapcsolóival szűkítve.
3. Másodlagos sorok: `title`, ha `showTitle`, nem üres, nem a címke forrása, és nem egyezik a címkével (kis-nagybetű nélkül); `description`, ha `showDescription`, nem üres és nem a címke forrása; `url` = `compactUrl(url, host, LINE_URL_MAX)`, ha `showUrl` és a címke nem az URL.
4. A Markdown-másolás és a törlés megerősítő szövege a teljes precedenciát használja a kapcsolóktól függetlenül: `linkPresentation(link, ALL_FIELDS).label`.
5. A hossz Unicode-kódpontban (`[...s].length`).

| description | title | url | kapcsolók | label | secondary |
|---|---|---|---|---|---|
| `Éles műszerfal` | `Grafana` | `https://grafana.example.net/d/abc` | mind | `Éles műszerfal` | title `Grafana`, url `grafana.example.net/d/abc` |
| `''` | `SvelteKit docs` | `https://svelte.dev/docs/kit` | mind | `SvelteKit docs` | url `svelte.dev/docs/kit` |
| `''` | `''` | `https://example.com/` | mind | `example.com` | (nincs) |
| `''` | `''` | `https://docs.example.com/very/long/path/to/a/page?id=12345&x=1` | mind | `docs.example.com` | (nincs) |
| `Leírás` | `Cím` | `https://a.io/x` | leírás ki | `Cím` | url `a.io/x` |
| `Leírás` | `Cím` | `https://a.io/x` | leírás és cím ki | `a.io/x` | (nincs) |

Kliensoldali rendezés (`src/lib/links/order.ts`): `compareLinks(a, b, sort)` és `sortLinks(items, sort)`, a 4.8 `ORDER BY` pontos mása, csak az optimista frissítés utáni helyi újrarendezéshez: 1. `pinned` csökkenő; 2. `usage`: `useCount` csökkenő, majd `lastUsedAt` csökkenő (`null` a végén); 3. `title`: `sortLabel = (description || title || host).toLowerCase()` növekvő, kódpont szerinti összehasonlítással (nem `localeCompare`, hogy a szerver bájtsorrendjével egyezzen); 4. `createdAt` csökkenő, végül `id` csökkenő.

`ExpiryMeter` (az automatikus archiválásig hátralévő idő):

- Bemenet: `archiveDueAt`, `inactiveSince` (5.5 `LinkItem`), `pinned`, `archivedAt`, `effectiveArchiveAfterDays`. Nem jelenik meg kitűzött, archivált vagy `effectiveArchiveAfterDays === 0` esetén.
- Rajz: 5 szegmens, egyenként 6×3 px, rés 2 px, `rounded-none`; kitöltött `bg-muted-foreground`, üres `bg-border`; a kitöltött szegmensek száma `ceil(5 × hátralévő / teljes)`, legalább 1, amíg nem járt le.
- 7 nap vagy kevesebb: a szegmensek helyett `hourglass` 12 px `text-warning` ikon és „{count} nap” (`count` legalább 1; ma esedékesnél „ma”).
- `role="img"` + `aria-label` és tooltip: „Archiválás {date} körül, ha addig nem nyitod meg és nem másolod”.
- `src/lib/links/expiry.ts`: `expiryInfo({archiveDueAt, inactiveSince, now}) → { segments: 0..5, daysLeft: number, soon: boolean } | null`.

Sorműveletek: a sor menüje, a csempe menüje és a részletek lap ugyanazt a `linkActions(link, me)` listát használja (`features/links/actions.ts`):

| Művelet | Ikon | Feltétel | Viselkedés |
|---|---|---|---|
| Megnyitás | `external-link` | mindig | új lapon + `visit` |
| URL másolása | `copy` | mindig | `copyText(url)`, sikeres másolás után `visit` (használat, 5.9.5), toast „URL másolva” |
| Másolás Markdownként | `code` | mindig | `toMarkdownLink(label, url)`, sikeres másolás után `visit`, toast „Markdown-link másolva” |
| Részletek | `info` | mindig | részletek lap |
| Szerkesztés | `pencil` | `isOwner` | szerkesztő dialógus |
| Címkék | `tags` | mindig (a címkék személyesek) | `TagPicker`; mentés `PUT /api/links/{id}/tags` |
| Megosztás | `share-2` | `isOwner && me.permissions.canShare` | megosztás dialógus |
| Cím és ikon frissítése | `refresh-cw` | `isOwner` | `POST …/refresh`, `watchPending` |
| Kitűzés / megszüntetés | `pin` / `pin-off` | mindig | archivált linken a kitűzés vissza is állítja (4.9), toast „Link kitűzve és visszaállítva” |
| Archiválás / Visszaállítás | `archive` / `archive-restore` | mindig | archiválásnál toast „Link archiválva” + „Visszavonás” |
| Törlés | `trash-2` | `isOwner` | `ConfirmDialog` |

Másolás: `src/lib/links/markdown.ts` `toMarkdownLink(label, url)`: a címkében a `\`, `[`, `]` `\`-sel escape-elve, a sortörés szóközzé; az URL-ben a szóköz `%20`, a `(` `%28`, a `)` `%29`. Eredmény: `[Éles műszerfal](https://grafana.example.net/d/abc)`. `utils/clipboard.ts` `copyText(text)`: `navigator.clipboard.writeText`; ha nem elérhető (nem biztonságos környezet), rejtett `textarea` + `document.execCommand('copy')`; hiba esetén toast „A másolás nem sikerült. Jelöld ki és másold kézzel.”

Megnyitás, másolás és látogatás (5.9.5): a horgony `click` eseménye (bal kattintás, és a fókuszált horgonyon az Enter, amelyre a böngésző maga küld `click`-et) és az `auxclick` `button === 1` esetén `links.visit(id)`-t hív (`keepalive: true`); külön Enter-kezelő nincs, így egy megnyitás egy látogatás (komponensteszt: Enter a horgonyon pontosan egy `visit`). A sikeres másolás (menü, `c`, `m`, a részletek lap gombjai és `CopyButton`-ja) ugyanígy `visit`-et hív. Az optimista frissítés (`useCount + 1`, `lastUsedAt = now`, archiválás megszűnése) azonnal átrendez; a szerver `LinkState` válasza felülírja. A saját linkek `dragstart`-ja `application/x-kinex-mark` típust is beállít, hogy a saját linkek ráhúzása ne indítson rögzítést.

Gyors rögzítés (`CaptureController.svelte` + `src/lib/links/capture.ts`):

```ts
/** URL candidates from a DataTransfer: text/uri-list lines (no "#" comments), else text/plain. */
export function extractCaptureItems(dt: Pick<DataTransfer, 'types' | 'getData'>): { text: string; title?: string }[];
/** Anchor text of the first <a> in text/html, trimmed, max 500 characters. */
export function anchorTitle(html: string): string | undefined;
export function isEditableTarget(target: EventTarget | null): boolean; // input, textarea, select, contenteditable
```

- Ráhúzás: `dragenter`/`dragover` a `window`-on; ha a `types` tartalmaz `text/uri-list`-et vagy `text/plain`-t, nem tartalmaz `Files`-t és `application/x-kinex-mark`-ot, és nincs nyitott dialógus, a `preventDefault` engedi a dobást, és megjelenik a fátyol (enter/leave számlálóval). Fátyol: `fixed inset-0 z-50 kxm-drop-scrim`, középen `Panel size="lg" class="[--kx-edge:var(--primary)]"` legfeljebb 480 px, `space-8` margó: 24 px `clipboard-paste` ikon `text-primary`, `heading-3` „Engedd el a link mentéséhez”, `body-sm` „URL, böngészős link vagy URL-t tartalmazó szöveg. A cím és az ikon a háttérben érkezik.”; `aria-hidden="true"`; 120 ms áttűnés. `drop`: `extractCaptureItems`; a `text/uri-list` több sora több link (legfeljebb 20, toast jelzi a többit); a `title` a `text/html` horgonyszövegéből, csak egyetlen elemnél.
- Beillesztés: `paste` a `document`-en, ha `!isEditableTarget(e.target)` és nincs nyitott dialógus vagy lap; a `clipboardData` `text/plain` (és `text/html` a címhez) tartalma. Ha a szövegben nincs `http(s)://` és nem domain-alakú, a kérés el sem indul, toast `errors.no_url_found`.
- Mentés elemenként sorban `POST /api/links/quick`:

| Eredmény | Visszajelzés |
|---|---|
| 201 `created: true` | `links.upsert(link, {highlight: true})`, `watchPending`; toast „Link mentve” + „Szerkesztés” |
| 200 `created: false` | a meglévő sor kiemelése; toast „Ez a link már megvan” + „Megjelenítés” |
| Több elem | egy összesítő toast: „{created} link mentve, {existing} már megvolt” |
| A szűrők elrejtik | „Link mentve. A jelenlegi szűrők elrejtik.” + „Szűrők törlése” |
| Nem a Linkek oldalon | toast + „Megnyitás” (`/` és kiemelés) |
| `no_url_found`, `validation_failed`, `link_limit_reached` | hiba-toast |

Felvétel és szerkesztés (`LinkDialog`, `Dialog` legfeljebb 440 px):

| Mező | Komponens | Szabály |
|---|---|---|
| URL | `Input` mono, `type="url"`, `inputmode="url"`, automatikus fókusz | kötelező; a kliens csak az ürességet nézi; szerkesztéskor csak tulajdonosnak |
| Cím | `Input` | nem kötelező; súgó „Üresen hagyva a letöltött cím jelenik meg.”; 500 karakter |
| Rövid leírás | `Textarea` 3 sor | nem kötelező; mono számláló „{n} / 500”; súgó „A listában ez jelenik meg elsőként.” |
| Címkék | `TagPicker` | legfeljebb 20 |
| Kitűzés | `Checkbox` „Kitűzés a lista elejére” | csak felvételkor |

- Cím: „Link felvétele” / „Link szerkesztése”; gombok „Mégse” `outline`, „Link mentése” `default`.
- Felvétel: `POST /api/links` (`tagIds`-szel); siker: `upsert` + kiemelés + `watchPending`, toast „Link mentve”. Szerkesztés: `PATCH` a változott mezőkkel, a címkék külön `PUT …/tags`.
- `link_duplicate`: `warning` `StateAlert` „Ez a link már megvan a gyűjteményedben.” + „Megjelenítés a listában” (dialógus zárva, sor kiemelve a `details.linkId` alapján).
- Mezőhibák: `FieldError`, `aria-invalid`, fókusz az első hibás mezőre.
- Nyitott dialógusnál a ráhúzás és a beillesztés nem rögzít.

Részletek lap (`LinkDetailsSheet`, `Sheet side="right"`, `sm:max-w-[440px]`, mobilon teljes szélesség; nyitáskor `GET /api/links/{id}`; az URL `?link=<id>` paramétert kap):

| Blokk | Tartalom |
|---|---|
| Fej | `Favicon` 32 px egy 40 px-es `muted` keretes dobozban; címke `heading-3` (2 sor), alatta host mono |
| Gombsor | `outline` `sm`: „Megnyitás”, „URL másolása”, „Másolás Markdownként”; `ghost` ikongomb kitűzés. Primary gomb nincs |
| „Adatok” | `dl` (`grid-cols-[120px_1fr]`): URL (mono, `break-all`, `CopyButton`), Cím, Leírás (sortörés megtartva), Címkék (`TagChip`-ek + „Szerkesztés”), Tulajdonos (megosztott linknél) |
| „Használat” | 3 `Metric`: „Használat” (`fmt.number(useCount)`, megnyitás és másolás együtt; megjegyzés: utolsó használat relatív ideje vagy „Még nem használtad”); „Archiválás” (`fmt.date(archiveDueAt)` vagy „Soha” kitűzött vagy 0 napos időszaknál; megjegyzés „{count} nap múlva”); „Rögzítve” (`fmt.date(createdAt)`) |
| „Időpontok” | Rögzítve, Módosítva, Utolsó használat, Megosztva veled (címzettnél), Tétlen ettől (`inactiveSince`), Archiválva (okkal: „automatikusan” / „kézzel”); `fmt.dateTime`, mono |
| „Metaadat” | `StatusBadge`: `ok` → `online` „Letöltve”; `pending` → `info` „Letöltés folyamatban”; `failed` → `critical` „Sikertelen”; `blocked` → `warning` „Tiltott hálózati cím” (a „cím” önmagában a lapcím, 2.3 G10); mellette `fetch.errors.<fetchError>`, letöltés ideje és a próbálkozások száma mono; tulajdonosnak „Cím és ikon frissítése” |
| „Megosztás” | tulajdonosnak `canShare` esetén: a megosztások listája, „Megosztás kezelése” |
| Láb | „Szerkesztés” (tulajdonos), „Archiválás” / „Visszaállítás”, „Link törlése” `ghost` `text-destructive` (tulajdonos) |

Ez a lap a BASE-SYSTEM-PROMPT teljes információs nézete: ikon, URL, cím, leírás, használati statisztika, létrehozás és utolsó használat ideje.

Megosztás (`ShareDialog`, csak `isOwner && canShare`):

- Leírás: „A címzettek a saját listájukban látják a linket, saját címkékkel és saját kitűzéssel. Módosítani csak te tudod.”
- Jelenlegi megosztások: `globe` „Mindenki”, `users` „Szerepkör: Fejlesztők”, `user` „Kovács Anna” + mono `kanna`; `ghost` `x` ikongomb, `aria-label` „Megosztás visszavonása: {target}”. Üresen: „Még senkivel nincs megosztva.”
- Új megosztás: `ToggleGroup` „Mindenki / Szerepkör / Felhasználó”; szerepkörnél `Select` (Adminisztrátorok, Fejlesztők, Ügyfelek); felhasználónál `Popover` + `Command` a `GET /api/users/directory?q=` 200 ms debounce-szal.
- Gombok: „Megosztás hozzáadása” (`default`), „Kész” (`outline`). `share_duplicate` és a `validation_failed` mezőkódjai a választó alatt.

Törlés: `ConfirmDialog` „Törlöd ezt a linket?” / „A következő link végleg törlődik: „{label}”. A megosztásai és minden felhasználó használati adatai is törlődnek. A művelet nem vonható vissza.” / „Mégse”, „Link törlése”; siker: toast „Link törölve”.

Csempe (`LinkTile`): rács `grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-3`, csoportcímek a rácsok fölött; `Card` (levágott sarok nélkül, mert menüt tartalmaz), `p-4`, legalább 148 px; felső sor: 40 px-es ikondoboz (24 px `Favicon`), jobbra jelölők és menügomb; címke `line-clamp-2` `<a>`, alatta a másodlagos sorok (legfeljebb 2 sor leírás); láb: saját sorban legfeljebb 2 `TagChip` + `+{n}`, alatta rögzített magasságú (`h-5`) meta sor a megnyitásszámmal és az `ExpiryMeter`-rel, így egy rácssor csempéinek lábai egy vonalban vannak (D fázis, R5). A csempe egésze nem kattintható.

Billentyűparancsok (`utils/keyboard.ts` `shouldHandleShortcut`: nem szerkeszthető fókusz, nincs nyitott dialógus, nincs `Ctrl`/`Alt`/`Meta`, a beillesztés kivételével):

| Billentyű | Hatás |
|---|---|
| `/` | fókusz a keresőmezőre |
| `n` | Link felvétele dialógus |
| `Ctrl V` / `⌘ V` | gyors rögzítés a vágólapról |
| `j` / `k` vagy `↓` / `↑` | következő / előző link címkéje |
| `Enter` | a fókuszált link megnyitása: ha a fókusz a horgonyon van, a böngésző natív `click`-je nyit (külön kezelő nincs); ha a soron, a kezelő `anchor.click()`-et hív; mindkét esetben pontosan egy látogatás |
| `p` | kitűzés váltása |
| `c` | URL másolása (használatnak számít) |
| `m` | másolás Markdownként (használatnak számít) |
| `i` | részletek |
| `e` | szerkesztés (tulajdonos) |
| `a` | archiválás / visszaállítás |
| `Delete` | törlés megerősítéssel (tulajdonos) |
| `v` | lista és csempe váltása |
| `?` | Billentyűparancsok dialógus (`KeyboardShortcut`-tal) |
| `Esc` | keresés törlése a keresőmezőben |

Kiegészítések (E fázis, GUI-teszt):

- A parancsok akkor is működnek, ha maga a `<main>` kapta a fókuszt (kattintás üres oldalterületre): a `LinksPage` `inPage` feltétele az oldalgyökér őseit is elfogadja.
- `Esc` a keresőmezőben minden szélességen két lépésben hat: szöveg esetén törli a szöveget, megszakítja a függő élő keresést, és a Linkek oldalon a `?q` paramétert is elhagyja (`goto(…, { replace: true, reset: false })`, a fókusz a mezőben marad); üres mezőnél a mobil keresőréteget zárja. Asztali szélességen a `/` nem nyitja meg a mobil keresőréteget (csak 768 px alatt).
- A billentyűnevek a felület nyelvét követik: `common.keys.{ctrl,enter,delete,esc}` (hu, en: `Ctrl`, `Enter`, `Delete`, `Esc`; de: `Strg`, `Enter`, `Entf`, `Esc`); a `utils/keyboard.ts` `modifierKeyLabel()` (nem Apple platformon) és `keyLabel()` adja, a `KeyboardShortcut` minden billentyűt ezeken át ír ki (a táblázat, az üres állapot és minden más használat).
- Olvasható URL: a részletek lap URL-szövege (`data-testid="details-url"`) és a szerkesztődialógus URL-mezőjének kezdőértéke a `links/presentation.ts` `readableUrl()` kimenete: csak a jól formált, nem ASCII UTF-8 százalékos sorozatokat dekódolja (`%C3%A1` → `á`); az ASCII-escape-ek (`%20`, `%2F`, `%25`) és a láthatatlan vagy formázó karakterek kódolva maradnak, így a szerkesztés a szerver `NormalizeURL`-jén változatlanul megy át. A horgony `href`-je és a másolt szöveg a tárolt URL; a szerkesztődialógus a változatlan URL-t nem küldi el.
- A címkeválasztó, a „további címkék” popover és a megosztás felhasználóválasztója minden nyitáskor üres kereséssel indul. Az egyválasztós `ToggleGroup`-ok (hatókör, nézet, címkeegyezés, megosztási cél, nyelv, jelszómód) őrzött `bind:value`-val nem üríthetők ki újrakattintással (9.8 14. szabály).
- A toastok ikonja az állapot színét kapja (`AppToaster` `toastOptions.classes`: success, info, warning, destructive).

#### 8.7.4 Kapuoldalak (`/change-password`, `/consent`, F1; elrendezés F0)

`(gate)/+layout.svelte`: középre igazított oszlop, legfeljebb 440 px, felül `Logo variant="mark" height={28}` + „Kinex Mark”, jobb felső sarokban `LocaleSwitcher variant="inline"` és `ThemeToggle`, alul „Kijelentkezés” `ghost` gomb.

Kötelező jelszócsere (`PasswordChangeForm`, F1; a Beállítások › Biztonság is ezt használja):

- „Állíts be új jelszót” / „A fiókod ideiglenes jelszóval készült, vagy a rendszergazda visszaállította. A folytatáshoz adj meg egy újat.”
- Mezők: Jelenlegi jelszó, Új jelszó (súgó a `publicConfig.passwordPolicy`-ből: „Legalább 12 karakter, legalább 6 különböző karakter, a felhasználóneved nélkül.”), Új jelszó még egyszer (kliensoldali egyezés: „A két jelszó nem egyezik.”).
- Elsődleges gomb: „Jelszó módosítása”; siker: `session.refresh()`, toast „Jelszó módosítva”, irányítás a következő kapura vagy `/`.
- Hibák: `current_password_invalid` a jelenlegi mező alatt; `validation_failed` `newPassword` kódjai az új mező alatt; `too_many_attempts` alert.

Hozzájárulás (`ConsentForm`):

- „Adatkezelési tájékoztató” / „Olvasd el, hogyan kezeli a Kinex Mark az adataidat. A folytatáshoz erősítsd meg, hogy megismerted.”; verzió és dátum mono.
- `PrivacyNotice` legfeljebb 50vh magas `ScrollArea`-ban; alatta „Megnyitás teljes oldalon” (`/privacy`, új lapon).
- `Checkbox` „Elolvastam az adatkezelési tájékoztatót, és tudomásul veszem.”; a gomb addig `disabled`. A szöveg nyugtázás, nem hozzájárulás (6.14).
- „Tudomásul vettem, tovább” → `POST /api/me/privacy/consent {version}`; `consent_version_mismatch` esetén a tartalom újratöltődik, `info` alert jelzi az új verziót.

#### 8.7.5 Címkék (`/tags`, F3)

- Oldalfej: „Címkék”; alcím: „{count} címke. A címkék személyesek: a veled megosztott linkekre is a saját címkéidet teheted.”; elsődleges gomb „Címke felvétele” (`plus`).
- Táblázat: Név (`TagChip`), Szín (a szín neve), Linkek (`linkCount` mono, szöveges link: `/?tags=<id>&archived=include`), Aktív (`activeLinkCount`), Módosítva (`fmt.date`), műveletek (`pencil`, `trash-2`; „Szerkesztés: {name}”, „Törlés: {name}”). Név szerint rendezve.
- Üres: `tags` ikon, „Még nincs címke.” / „Vegyél fel egyet, és csoportosítsd vele a linkjeidet.”
- `TagDialog`: Név (1–40 karakter, számláló), Szín: `ToggleGroup type="single"` hét elemmel (8×8 px négyzet + szín neve) a 4.6 sorrendjében; előnézet `TagChip`; gomb „Címke mentése”. `tag_duplicate` a név alatt, `tag_limit_reached` alert.
- Törlés: „Törlöd ezt a címkét?” / „A következő címke törlődik: „{name}”. {count} linkről kerül le, a linkek megmaradnak.” / „Címke törlése”.

A csoportosítás kizárólag címkékkel történik; mappák nincsenek.

#### 8.7.6 Beállítások (`/settings/*`)

A `LineTabs` négy füle külön útvonal (8.7.0); a tartalom `Card`-okban, legfeljebb `layout-prose` szélesen.

Profil (`/settings/profile`, F3; elsődleges gomb „Profil mentése”): Felhasználónév (csak olvasható, mono), Megjelenített név (0–100), Szerepkör (szóval), Fiók létrehozva, Utolsó bejelentkezés.

Biztonság (`/settings/security`, F5; elsődleges gomb „Jelszó módosítása”, a többi `outline`):

- „Jelszó” kártya (`PasswordCard`): a `PasswordChangeForm`; siker után toast „Jelszó módosítva. A többi eszközön kijelentkeztél.”
- „Kétlépcsős azonosítás” kártya (`TwoFactorCard`): `StatusBadge` (`online` „Bekapcsolva” / `offline` „Kikapcsolva”); „Fel nem használt helyreállító kód: {count}” (3 vagy kevesebb: `warning` `StateAlert` „Kevés helyreállító kódod maradt” / „Generálj újakat, mielőtt elfogynak.”). Gombok: „Bekapcsolás”; bekapcsolt állapotban „Új helyreállító kódok” és „Kikapcsolás”.
- Bekapcsolás (`TwoFactorEnrollDialog`, „1 / 3” lépésjelző):
  1. Jelszó megerősítése → `POST /api/me/2fa/setup`.
  2. „Olvasd be a QR-kódot”: `QrCode` 176 px az `otpauthUri`-ból; „Kézi beíráshoz:” és a titok mono, négyes csoportokban, `CopyButton`-nal; `InputOTP` 6 mező → `POST /api/me/2fa/enable`. Lejárat (`expiresAt`, 10 perc) után `totp_setup_required`: vissza az 1. lépésre.
  3. „Mentsd el a helyreállító kódokat” (`RecoveryCodesView`): 10 kód két oszlopban mono, `kxm-print-only` fejléccel; „Másolás”, „Letöltés .txt-ként” (`kinex-mark-recovery-codes-<username>.txt`; a fájl sorai a `security.recovery.file.*` kulcsokból a felhasználó nyelvén: cím, bevezető, generálás ideje, figyelmeztetés, majd a kódok), „Nyomtatás”; `Checkbox` „Elmentettem a kódokat, és tudom, hogy csak most láthatók.”; a „Kész” addig tiltott. Bezáráskor `session.refresh()`, toast „Kétlépcsős azonosítás bekapcsolva”.
- Kikapcsolás és új kódok (`TwoFactorConfirmDialog`): jelszó és kód (kikapcsolásnál TOTP vagy helyreállító kód, új kódoknál csak TOTP); az új kódok a `RecoveryCodesView`-ban.
- „Munkamenetek” kártya (`SessionsCard`): Eszköz (`describeUserAgent`: „Chrome, Windows”; ismeretlennél a nyers UA első 40 karaktere mono), IP, Létrehozva, Utolsó aktivitás, Lejár; az aktuális sor `Badge outline` „Ez az eszköz”; soronként `log-out` „Kijelentkeztetés: {device}”; a kártya fejében „Kijelentkezés a többi eszközön” → toast „{count} munkamenet lezárva”. A saját munkamenet visszavonása megerősítést kér, utána `/login`.

Megjelenítés (`/settings/display`, F3; minden vezérlő azonnal ment):

- Nyelv: `Select` (Magyar, English, Deutsch).
- Téma: `ToggleGroup` Sötét / Világos / Rendszer szerint.
- Alapnézet: Lista / Csempe.
- Megjelenített adatok: `Switch` Ikon, URL, Cím, Leírás; súgó: „A link címkéje a leírás, ennek hiányában a cím, végül az URL. Kikapcsolt mezőt a címke sem használ.”
- Archivált linkek alapból: `Switch` „Archiváltak megjelenítése”.
- Rendezés alapból: Használat szerint / Legújabb elöl / Név szerint; címkeegyezés alapból: Bármelyik / Mind.
- Automatikus archiválás (`ArchivePeriodSelect`): „Rendszer-alapérték ({days} nap)” (`null`; a `days` a `Preferences.systemArchiveAfterDays`, így nem adminnak és egyéni érték mellett is ismert), 30, 60, 90, 180, 365 nap, „Soha” (0), „Egyéni…” (mono szám 1–3650, Enterre vagy fókuszvesztésre ment). Súgó: „A kitűzött linkek soha nem archiválódnak. Megnyitás, másolás, kitűzés vagy visszaállítás után a számlálás újraindul.”; „Jelenleg: {effective}”.

Adatvédelem (`/settings/privacy`, F5):

- „Adatkezelési tájékoztató” (`ConsentHistory`): elfogadott verzió és időpont (`GET /api/me/privacy`), link a `/privacy`-re.
- „Adataid letöltése” (`DataExportCard`): mit tartalmaz (fiók, beállítások, munkamenetek, címkék, saját linkek a megosztásokkal, a veled megosztott linkek személyes állapota, a fiókodhoz tartozó audit események szereplőként és célként, a bejelentkezési zárolás állapota; jelszó és titok nélkül); „Adatok letöltése (JSON)”.
- „Fiók törlése” (`DeleteAccountDialog`): „Törlöd a fiókodat?” / „Minden linked, címkéd, megosztásod és beállításod végleg törlődik. A művelet nem vonható vissza.”; ha a `PrivacyStatus.ownSharedLinkCount` nagyobb 0-nál, `warning` `StateAlert`: „{count} megosztott linked a címzetteknél is eltűnik. Ha meg akarják tartani, kérd meg őket, hogy vegyék fel a saját gyűjteményükbe.” (plural). Mezők: Jelszó, bekapcsolt 2FA mellett „Hitelesítő vagy helyreállító kód” (mono), „Írd be a felhasználóneved a megerősítéshez: {username}”; a „Fiók végleges törlése” az egyezésig tiltott. Hibák: `current_password_invalid`, `invalid_totp`, `confirmation_mismatch`, `last_admin`, `too_many_attempts`. Siker: `session.clear('deleted')`, `/login?reason=deleted`.

#### 8.7.7 Admin: felhasználók (`/admin/users`, F4)

- Oldalfej: „Felhasználók”; alcím „{total} felhasználó”; elsődleges gomb „Felhasználó felvétele” (`user-plus`).
- Szűrők: kereső (200 ms), szerepkör (Mind, Adminisztrátor, Fejlesztő, Ügyfél), állapot (Aktív, Inaktív, Mind; alap Aktív). Az állapot az URL-ben (`?q=&role=&status=&page=`; `offset = (page - 1) × 50`).
- Táblázat: Felhasználónév (mono), Megjelenített név, Szerepkör, Állapot (`StatusBadge`: aktív → `online` „Aktív”; inaktív → `offline` „Inaktív”; jövőbeli `lockedUntil` → `warning` „Zárolva”), 2FA (`circle-check` `text-success` + „Be” vagy `–`), Linkek, Munkamenetek, Utolsó belépés, műveletmenü.
- Műveletek: Szerkesztés, Jelszó visszaállítása, 2FA visszaállítása (ha be van kapcsolva), Feloldás (ha zárolt), Munkamenetek lezárása, Inaktiválás / Aktiválás, Törlés. A saját sornál a szerepkör, az inaktiválás és a törlés tiltott, tooltip: „A saját fiókodat a Beállítások oldalon kezeled.”
- Lapozás: shadcn `Pagination`, 50 sor, „{from}–{to} / {total}” mono.
- `UserDialog` (felvétel): Felhasználónév (súgó „3–64 karakter: betű, szám, pont, kötőjel, aláhúzás.”), Megjelenített név, Szerepkör, Nyelv (alap: az admin aktuális nyelve), Jelszó: „Generált jelszó” (alap) / „Megadom”. Gomb „Felhasználó létrehozása”. Siker: `TempPasswordDialog`, ha `temporaryPassword` nem `null`: „Ideiglenes jelszó”, a jelszó mono nagy méretben, `CopyButton`, `warning` „A jelszó csak most látható” / „Add át biztonságos csatornán. Első bejelentkezéskor a felhasználónak le kell cserélnie.”; gomb „Kész”.
- Szerkesztés: Megjelenített név, Szerepkör, Aktív; ha a szerepkör vagy az aktív állapot változik, megjelenik a „Saját jelszavad a megerősítéshez” mező (`common.stepUp`, `PasswordInput`, kötelező); gomb „Változások mentése”; `self_modification`, `last_admin` alertként, `current_password_invalid` a mező alatt.
- Megerősítést kérő műveletek (`ConfirmDialog` a step-up jelszómezővel, 5.12; a megerősítő gomb a mező kitöltéséig tiltott): Jelszó visszaállítása („Visszaállítod {username} jelszavát?” / „Minden munkamenete lezárul, és új, ideiglenes jelszót kap. A művelet a biztonsági naplóba kerül, és a felhasználó a következő belépéskor jelszót cserél.” / „Jelszó visszaállítása”, `default`), 2FA visszaállítása („Minden munkamenete lezárul, a kétlépcsős azonosítást újra be kell kapcsolnia.”), Inaktiválás („Minden munkamenete lezárul. A megosztásai érvényben maradnak.”), Törlés („Törlöd {username} fiókját?” / „A linkjei, címkéi és megosztásai végleg törlődnek; az audit naplóban a neve álnévre cserélődik. A művelet nem vonható vissza.”; `sharedLinkCount > 0` esetén `warning`: „{count} megosztott linkje a címzetteknél is eltűnik.” / „Felhasználó törlése”). A Feloldás és a Munkamenetek lezárása nem kér jelszót (nem ad hozzáférést).

#### 8.7.8 Admin: rendszerbeállítások (`/admin/settings`, F4)

- Egyetlen űrlap kártyákra bontva (`SettingsForm`); „Beállítások mentése” a lap alján tapadó sávban (`sticky bottom-0`), csak változáskor aktív; mellette „Elvetés”. Elhagyáskor változás esetén a `beforeNavigate` megerősítést kér.
- Mezőnként súgó a tartománnyal és az alapértékkel (`defaults`): „0–3650 nap. 0: soha. Alapérték: 90.”

| Kártya | Mezők |
|---|---|
| Archiválás | `defaultArchiveDays`; „Archiválás futtatása most” → `POST /api/admin/jobs/archive`, toast „{count} link archiválva” |
| Munkamenetek | `sessionIdleMinutes`, `sessionAbsoluteHours` (mértékegység: „perc”, „óra”) |
| Audit napló | `auditRetentionDays` |
| Metaadat-letöltés | `allowPrivateFetch` „Belső hálózati címek engedélyezése”; súgó: „Intranetes linkekhez. Bekapcsolva a szerver a helyi hálózat címeit is letöltheti; a felhős metaadat-címek továbbra is tiltottak.” `lockedByEnv` esetén tiltva + `info` „Környezeti változó rögzíti” / „Az APP_ALLOW_PRIVATE_FETCH értéke dönt.” |
| Adatkezelési tájékoztató | adatkezelő neve, címe, e-mail-címe, adatvédelmi kapcsolattartó, illetékes felügyeleti hatóság (`privacySupervisoryAuthority`, 0–300 karakter, súgó: „Például: Nemzeti Adatvédelmi és Információszabadság Hatóság, naih.hu”); jelenlegi verzió és dátuma; „Új verzió kiadása” → megerősítés („Kiadod a tájékoztató új verzióját?” / „Minden felhasználónak újra el kell fogadnia a következő bejelentkezéskor vagy kéréskor.”) |

Hibák: `validation_failed` mezőnként, `setting_locked`.

#### 8.7.9 Admin: audit napló (`/admin/audit`, F4)

- Oldalfej: „Audit napló”; alcím „A biztonsági események {days} napig maradnak meg.”; elsődleges gomb nincs.
- Szűrők (URL-ben): Esemény (Mind; Bejelentkezés `auth.*`; Munkamenetek `session.*`; Felhasználók `user.*`; Beállítások `settings.*`; Adatvédelem `privacy.*` és `gdpr.*`; Megosztás `link.*`; utána minden konkrét esemény), Eredmény (Mind, Sikeres, Sikertelen), Szereplő (`Popover` + `Command` a `GET /api/admin/users` listából, `actorId`), Időszak (két `<input type="date">`; a nap eleje és vége helyi időben, UTC-re alakítva). A két `*` csoport (`privacy.*`, `gdpr.*`) két kérésből fésülődik, mert a szerver egy előtagot fogad.
- Táblázat: Időpont (`fmt.dateTime` mono), Esemény (`audit.events.<event>`, alatta a kód mono 11 px), Eredmény (`circle-check` „Sikeres” / `circle-alert` „Sikertelen”), Szereplő (felhasználónév mono; törölt felhasználónál `deleted-user-<id>`), Cél (`audit.targets.<type>` + név), IP, Részletek (`Popover` mono `dl`-lel; üresnél `–`).
- Lapozás 50 soronként. Üres: „Nincs a szűrőknek megfelelő esemény.”

#### 8.7.10 Adatkezelési tájékoztató (`/privacy`, F1)

- `(public)` elrendezés: felső sáv logóval, jobbra `LocaleSwitcher` és `ThemeToggle`; bejelentkezve „Vissza az alkalmazásba”.
- Tartalom `max-w-[var(--layout-prose)]`, „Adatkezelési tájékoztató”, verzió és dátum mono (`publicConfig.privacy`).
- `PrivacyNotice` szakaszai (`privacy.sections.*`, a GDPR 13. cikke szerint, 6.14):
  1. `controller` Adatkezelő: név, cím, e-mail, adatvédelmi kapcsolattartó (üresnél `–`).
  2. `data` Kezelt adatok: felhasználónév, megjelenített név, jelszó-hash, titkosított 2FA-titok, munkamenetek IP-címmel és böngészőazonosítóval, linkek, címkék, megosztások, használati statisztika, beállítások, biztonsági napló IP-címmel. E-mail-cím, telefonszám, követőadat nincs.
  3. `purposes` Célok: a linkgyűjtemény szolgáltatás nyújtása; a fiók és a rendszer biztonsága (bejelentkezés-korlátozás, biztonsági napló).
  4. `legalBasis` Jogalapok: a szolgáltatás nyújtása (6. cikk (1) b) a fiók, a linkek, a címkék, a megosztások és a beállítások esetén; az üzemeltető jogos érdeke a rendszer biztonságához (6. cikk (1) f) a biztonsági napló, a bejelentkezés-korlátozás és a munkamenetek IP-címe esetén. A tájékoztató elfogadása a megismerés nyugtázása, nem hozzájárulás.
  5. `recipients` Címzettek: azok a felhasználók, akikkel linket megosztasz, látják a linket, a felhasználónevedet és a megjelenített nevedet; a fejlesztők és a rendszergazdák a megosztás címzettjének kiválasztásakor minden aktív felhasználót megtalálnak felhasználónév, megjelenített név és szerepkör szerint (5.11); az üzemeltető rendszergazdái a felhasználókezeléshez szükséges adatokat és a biztonsági naplót látják; az üzemeltető tárhelyszolgáltatója adatfeldolgozóként.
  6. `transfers` Harmadik országba továbbítás nincs.
  7. `automatedDecisions` Automatizált döntéshozatal és profilalkotás nincs; az automatikus archiválás csak a lista rendezését érinti.
  8. `retention` Megőrzés: munkamenetek legfeljebb {sessionAbsoluteHours} óráig ({sessionIdleMinutes} perc tétlenség után lejárnak), biztonsági napló {auditRetentionDays} napig, minden más a fiók törléséig (a számok a `PublicConfig.privacy`-ből).
  9. `storage` Sütik és helyi tárolás: egyetlen, a bejelentkezéshez szükséges süti (`kxm_session` vagy `__Host-kxm_session`); a böngésző helyi tárolójában a téma (`mode-watcher-mode`, `mode-watcher-theme`) és a nyelv (`kxm-locale`). Követés, analitika, harmadik féltől betöltött tartalom nincs.
  10. `fetching` Metaadat-letöltés: a cím és az ikon a szerverről töltődik le; a böngésző nem kapcsolódik harmadik félhez.
  11. `adminAccess` Rendszergazdai hozzáférés: a rendszergazda visszaállíthatja a jelszavadat, és ezzel beléphet a fiókodba; ez a biztonsági naplóba kerül (a saját adatexportodban is látod), és a következő belépéskor új jelszót kell megadnod.
  12. `rights` Jogaid (`privacy.rights.*`): hozzáférés és hordozhatóság (Beállítások › Adatvédelem › Adatok letöltése), helyesbítés (Beállítások › Profil, a linkek szerkesztése), törlés (fiók törlése), az adatkezelés korlátozása és tiltakozás (az adatkezelő elérhetőségein; a rendszergazda a fiókot inaktiválhatja), panasz a felügyeleti hatóságnál: {supervisoryAuthority} (üresnél: „a lakóhelyed szerinti adatvédelmi hatóság”).

#### 8.7.11 Hibaoldal (`+error.svelte`, F0)

- 404: „Az oldal nem található.” / „Ellenőrizd a címet, vagy térj vissza a linkekhez.” + „Ugrás a linkekhez”.
- 403: „Ehhez az oldalhoz nincs jogosultságod.” / „Az adminisztrációs oldalakat csak adminisztrátor látja.” + ugyanaz a gomb.
- Egyéb: `errors.internal_error`. Bejelentkezett felhasználónál az AppShellen belül jelenik meg. Ismeretlen útnál ezt a `(app)/[...rest]/+page.ts` catch-all biztosítja: 404-et dob, így az `(app)` layout lefut, és a `(app)/+error.svelte` a keretben rajzol; kijelentkezett látogató a layout átirányításával előbb a bejelentkezéshez kerül (D fázis, R5).

### 8.8 Nyitóoldal és hero-grafika

#### 8.8.1 Kompozíció (asztal, 1024 px-től)

- Rács: `grid-cols-[minmax(0,7fr)_minmax(400px,5fr)]`, `min-h-dvh`.
- Bal oszlop: márkafelület, témától független (`bg-ink-950 text-mist-100`, helyi `[--primary:var(--signal-300)]`). Margó `space-12` (1280 px fölött `space-16`), flex oszlop, rés `space-8`, legfeljebb 760 px:
  1. `Logo variant="mark" height={32} class="text-mist-100"` + „Kinex Mark” (18/26, súly 500), alatta „Kinexium” `body-sm` `text-mist-300`.
  2. Hero-tábla: `Panel size="lg"` `[--kx-fill:var(--ink-900)] [--kx-edge:var(--ink-600)]`, benne a kép `p-0`, 16:9.
  3. Cím: `display` (48/54, súly 250), legfeljebb 640 px: `landing.headline`.
  4. Bevezető: `body-lg` `text-mist-300`.
  5. `TickRule` (a márkafelületen `ink-300`/`ink-600` helyi felülírással).
  6. Funkciók: `ul` két oszlopban, elemenként 20 px Lucide-ikon `text-signal-300` (`clipboard-paste`, `tags`, `list-filter`, `list-ordered`, `archive`), cím 15/24 súly 500, alatta `body-sm` szöveg.
- Jobb oszlop: `bg-background`, témát követ; űrlap középen, legfeljebb 400 px; a láb alul.
- Sötét témában a két oszlop között 1 px `border` vonal.
- Mozgás: a hero-kép `opacity` 0 → 1, 200 ms `ease-out` a `load` eseményen (csökkentett mozgásnál azonnal).
- Elsődleges gomb: csak a „Bejelentkezés”.

| Szélesség | Elrendezés |
|---|---|
| 1024 px-től | a fenti |
| 768–1023 px | egy oszlop, legfeljebb 640 px: márkafelület kártyaként (`Panel size="lg"`, `ink-950`) a hero-képpel és a címmel; alatta az űrlap; alatta a funkciók |
| 768 px alatt | felül keskeny márkasáv (logó, termékcím); azonnal az űrlap; utána a márkafelület: kép teljes szélességben, cím `heading-1` méretben, bevezető, funkciók egy oszlopban; margó `space-4` |

#### 8.8.2 A kép beágyazása

```svelte
<img
	src="/images/hero.webp"
	width="1600"
	height="900"
	alt={t('landing.hero.alt')}
	decoding="async"
	fetchpriority="high"
	class="block h-auto w-full"
/>
```

A fájl hiánya nem töri az oldalt: `onerror` esetén a tábla üres `ink-900` kitöltéssel marad. Az E2E-teszt ellenőrzi, hogy a kép betölt (200, `naturalWidth > 0`).

#### 8.8.3 Szövegek

| Kulcs | hu | en | de |
|---|---|---|---|
| `landing.headline` | Linkek, használat szerint rendezve. | Links, ranked by use. | Links, nach Nutzung geordnet. |
| `landing.intro` | A Kinex Mark rögzíti, címkézi és használat szerint rangsorolja a linkjeidet. Amit régóta nem nyitottál meg, archívumba kerül. | Kinex Mark stores, tags and ranks your links by use. What you have not opened for a long time moves to the archive. | Kinex Mark speichert, beschriftet und ordnet deine Links nach Nutzung. Was du lange nicht geöffnet hast, wandert ins Archiv. |
| `landing.features.capture.title` | Gyors rögzítés | Quick capture | Schnell erfassen |
| `landing.features.capture.text` | Húzd az ablakra vagy illeszd be. A link azonnal mentődik, a cím és az ikon a háttérben érkezik. | Drop it on the window or paste it. The link is saved at once; title and icon follow in the background. | Zieh ihn ins Fenster oder füge ihn ein. Der Link wird sofort gespeichert, Titel und Symbol folgen im Hintergrund. |
| `landing.features.tags.title` | Színes címkék | Coloured tags | Farbige Labels |
| `landing.features.tags.text` | Csoportosítás mappák nélkül: egy link több címkét is kaphat. | Grouping without folders: one link can carry several tags. | Ordnung ohne Ordner: Ein Link kann mehrere Labels tragen. |
| `landing.features.filter.title` | Szűrés | Filtering | Filtern |
| `landing.features.filter.text` | Keresés címben, leírásban és URL-ben, szűrés egy vagy több címkére. | Search titles, descriptions and URLs; filter by one or more tags. | Suche in Titel, Beschreibung und URL, filtere nach einem oder mehreren Labels. |
| `landing.features.ranking.title` | Rangsor | Ranking | Rangfolge |
| `landing.features.ranking.text` | A gyakran megnyitott linkek előre kerülnek. A kitűzöttek mindig elöl állnak, köztük is a használat dönt. | Frequently opened links move up. Pinned links stay on top, ordered by use among themselves. | Häufig geöffnete Links rücken nach oben. Angeheftete stehen immer vorn, untereinander nach Nutzung geordnet. |
| `landing.features.expiry.title` | Automatikus archiválás | Automatic archiving | Automatisch archivieren |
| `landing.features.expiry.text` | A beállított ideig nem használt linkek archívumba kerülnek. Egy kattintással visszaállíthatók. | Links unused for the set period move to the archive. One click restores them. | Links, die im eingestellten Zeitraum ungenutzt bleiben, wandern ins Archiv. Ein Klick stellt sie wieder her. |
| `landing.hero.alt` | Vonalas rendszerábra: egy ráhúzott link bekerül a listába, színes címkék és szűrő rendezik, a gyakran használt és a kitűzött linkek elöl állnak, a régóta nem használtak archívumba kerülnek. | Line diagram: a dropped link joins the list, coloured tags and a filter sort it, frequently used and pinned links stay on top, long-unused links move to the archive. | Liniendiagramm: Ein abgelegter Link kommt in die Liste, farbige Labels und ein Filter ordnen sie, häufig genutzte und angeheftete Links stehen vorn, lange ungenutzte wandern ins Archiv. |
| `auth.login.title` | Bejelentkezés | Sign in | Anmelden |
| `auth.login.subtitle` | Kinex Mark, a Kinexium linkgyűjteménye. | Kinex Mark, the Kinexium link collection. | Kinex Mark, die Linksammlung von Kinexium. |
| `auth.login.submit` | Bejelentkezés | Sign in | Anmelden |
| `auth.login.noAccount` | Fiókot a rendszergazda hoz létre. | Accounts are created by an administrator. | Konten legt ein Administrator an. |

#### 8.8.4 Hero-kép brief (G munkacsomag)

Cél: a bejelentkező oldal márkafelületén ülő, high-tech, minimal, elegáns, sík vonalas rendszerábra, amely egy pillantásra mutatja a gyors rögzítést, a címkéket, a szűrést, a rangsort és az automatikus elévülést. Stílus: GRAPHICS.md 9.1, 9.4, 7.6 és 2.2–2.5.

| Paraméter | Érték |
|---|---|
| Modell | `gemini-3.1-flash-image` (a `generate_graphics.py` `DEFAULT_MODEL`-je) |
| Kulcs | `GEMINI_API_KEY` környezeti változó (a felhasználó adja; a tervben és a kódban nem szerepel) |
| Képarány | `aspect_ratio: "16:9"` |
| Felbontás | `image_size: "2K"` |
| Mester | `graphics/hero.png` (verziókövetett) |
| Kiszállítás | WebP (VP8, veszteséges), minőség 88, `method` 6, RGB: előbb a háttér `#071016`-ra normalizálva, majd pontos 16:9-re középre vágva, végül 1600 px szélesre kicsinyítve (Lanczos), 1600×900: `web/static/images/hero.webp` |
| Háttér | `background: "#071016"` a képleírásban (validált `#rrggbb`); csak a kiszállított másolatot érinti, a mester PNG-t a szkript nem módosítja (a jelenlegi mester a 4. jelölt, kézi feliratretussal: lásd az „Eredmény” bekezdést) |
| Vágás | `trim: false` (sötét alapon a sarokpixeles vágás nem alkalmazható) |
| Méretkorlát | legfeljebb 400 KB (a `web/hero_test.go` ellenőrzi) |
| Költségkeret | legfeljebb 4 fizetős generálás; minden újragenerálás `--force hero`-val, kézzel |

A `generate_graphics.py` szükséges bővítései:

1. `from google.genai import types`.
2. A `load_specs` a képleírás `aspect_ratio` és `image_size` mezőjét átveszi, és ellenőrzi az engedélyezett listával (`1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`, `21:9`; `1K`, `2K`, `4K`); a régi `size` szöveges mező megmarad.
3. A `generate_image` `config=types.GenerateContentConfig(response_modalities=["IMAGE"], image_config=types.ImageConfig(aspect_ratio=…, image_size=…))` paramétert ad (a `None` mezők elhagyva).
4. A `delivery` új `max_width` mezője a `deliver()`-ben arányos kicsinyítést végez `Image.LANCZOS`-szal; nagyobb képet nem nagyít.
5. A szkript sima `python`-nal fut (nem `python -I`), mert a `google-genai` a felhasználói site-ban van.
6. Minden új komment és üzenet angol. A `generate_image` kiírja a válasz `model_version`-jét, a `--list` a modellt, a képarányt és a felbontást is.
7. Pontos arányú vágás (`crop_to_ratio`): ha a `max_width` be van állítva és a képnek van `aspect_ratio`-ja, a másolatot kicsinyítés előtt középre vágja a pontos arányra. A modell 2K 16:9 kimenete 2752×1536 (1,7917:1), amelyből puszta kicsinyítés 1600×893-at adna; a vágás (2731×1536, oldalanként kb. 10 px üres háttér) után az eredmény pontosan 1600×900.
8. Háttér-normalizálás (`normalize_background`): a modell soha nem fest pontosan `#071016`-ot (mért értékek: `#0e1920`, `#0f1e26`, `#0c1921`). A külső szegélysáv mediánjával mért hátteret csatornánként monoton görbe viszi a célszínre (a vonalszínek kontrasztja megmarad), majd a háttértől legfeljebb ±4-re eső pixeleket pontosan a célszínre állítja. Determinisztikus.
9. A `deliver()` a `quality`-t csak veszteséges formátumnak (`webp`, `jpeg`) adja át; a PNG-kiszállítás `max_width` vagy `background` mellett a PIL-en át íródik, egyébként másolat.

`graphics/prompts.json`:

```json
{
  "model": "gemini-3.1-flash-image",
  "styles": {
    "kinexium-diagram": "Flat vector system diagram, high-tech, minimal and elegant. Background a single flat dark petrol colour #071016 filling the whole canvas edge to edge. Hairline strokes of 1 to 2 px only, no fills except small flat squares and slightly lighter flat panels #0b171f and #10212a. Line colours #253d46, #345058 and #5f7b7d. Text-like content is drawn as short rounded bars in #94a9a5 and #d4dcd5, never as readable letters. The only accent colour is cyan #77dbed, used sparingly for the active path, the pinned item and the nodes at the end of active lines. Lines end in small round nodes, like a network graph. The connecting lines and nodes echo the KINEXIUM helix language: two hairline strands joined by short straight steps, each step ending in a small round node, cyan only on the highlighted steps. Small square tag markers may use #2ba0c2, #b08a2c, #1f9b7c, #7a84e0 and #cf6a95, each covering only a few pixels. Orthographic, no perspective. "
  },
  "delivery": { "format": "webp", "quality": 88, "max_width": 1600 },
  "images": [
    {
      "name": "hero",
      "style": "kinexium-diagram",
      "aspect_ratio": "16:9",
      "image_size": "2K",
      "trim": false,
      "background": "#071016",
      "prompt": "The picture contains no words at all: no captions, labels, titles or headings anywhere; the area names used in this description (capture, list, tags, expiry) only explain the layout and must never be written into the image. Composition from left to right with at least 8% empty margin on every edge and generous empty petrol space, the visual weight slightly left of centre. Left third, quick capture: a small link card with a cut top-right and bottom-left corner and a mouse pointer floats along a thin dashed cyan path towards the top of a list; the path ends in a cyan node where the card arrives. Centre, list and ranking: six horizontal list rows with thin outlines, each with a small square icon slot and two short text bars; the top row is pinned, marked by a tiny pin glyph and a cyan left marker line; at the right end of every row a horizontal usage bar, the bars getting shorter from top to bottom, each ending in a small node. Above and beside the list, tags and filtering: a narrow search field outline with a magnifier, below it exactly five tiny coloured square tag markers, one in each of the five tag colours and none of them grey; two of them are selected with a thin cyan outline and thin lines connect them to the list rows they let through, while non-matching rows are dimmer (#253d46). Right third, automatic expiry: an instrument-like horizontal scale with fine and major tick marks as a timeline; the two bottom rows of the list are drawn in the dim line colour #253d46 and shifted step by step towards a low open-top archive box in the bottom right corner, with no transparency ramps; exactly one thin curved arrow from the box back to the list indicates restoring. The archive box is a flat front-view outline: a plain rectangle open at the top with a small handle slot, no depth, no flaps, never an isometric or 3D drawing. The timeline in the right third carries the KINEXIUM helix motif as a clean, regular, flat drawing of a double helix seen from the side, like a twisted rope ladder: exactly two smooth hairline sine curves of equal amplitude that are mirror images of each other across an invisible horizontal axis, so that wherever one curve is at its highest point the other is at its lowest point; the two curves cross each other at regular intervals and enclose a row of four or five almond-shaped cells. Both curves run along the full length of the motif, one drawn in #5f7b7d and the other in #253d46. Inside every almond-shaped cell three or four straight vertical rungs at equal spacing run from the upper curve down to the lower curve, and every rung ends in a small round node on the upper curve and another small round node on the lower curve; only two or three of those rungs and their nodes are cyan, the rest #253d46 and #5f7b7d. The helix has no straight horizontal line through its middle and is never a single lone wave. The fine instrument tick scale sits directly below the helix. The two bottom rows inside the list itself are drawn dim in #253d46. Keep every element inside the central 84% of the width and height. Outlines are crisp hairlines without any glow. Do not use gradients, opacity gradients or fading, shadows, glow (at most one faint glow around the active node), 3D, perspective, photos, stock elements, robot hands, blue brains, floating holograms, DNA renders, people, hands, readable text, letters, numbers, logos, watermarks, emoji, a frame around the image, a second accent colour or large saturated colour areas."
    }
  ]
}
```

A fenti `prompts.json` a G munkacsomag után érvényes állapot. Az eredeti, szó szerinti prompttal készült első jelölt izometrikus 3D archívumdobozt rajzolt, és a hélix-nyelvet (7. elfogadási pont) nem mutatta, ezért a hero prompt a „Do not use …” mondat elé kiegészítést kapott: az archívumdoboz sík elölnézet; a jobb harmad idővonala a hélixmotívum (két ellentétes fázisú szinuszhullám, csomópontban végződő függőleges lépcsők, csak két-három cián); a két alsó, halvány sor a listán belül marad; minden elem a középső 84%-on belül; hajszálvonalak derengés nélkül. Az eredeti prompt a munkakönyvtárban (`graphics\prompts-1.json`) megmaradt.

Frissítés (2026-11-22, a felhasználó engedélyével, 4. hívás): mivel a 3. jelölt hélixe csak közelítés volt, a hélix leírása egyértelműbb lett (oldalnézeti kettős hélix, „csavart kötélhágcsó”: két, egymás tükörképét adó szinuszgörbe, amelyek mandula alakú cellák sorát zárják közre, a cellákban függőleges lépcsők csomópontokkal; nincs egyenes középvonal, nem egyetlen hullám), a címkék száma pontosan öt (szürke nélkül), a visszaállító nyíl pontosan egy. Ezzel a prompttal készült a 4. jelölt (`graphics\prompts-4.json`). A modell a kompozíció mellé négy angol feliratot is rajzolt (a prompt területnevei), ezért a generálás **után** a prompt elejére felirattiltó mondat került; a fenti `prompts.json` ezt a végső állapotot mutatja, amellyel fizetős hívás még nem készült.

Tilos (a promptban negatív utasításként is): színátmenet, átlátszósági átmenet és elhalványulás, árnyék, derengés (legfeljebb egyetlen halvány derengés az aktív csomópont körül), 3D, perspektíva, fotó, stock-elem, robotkéz, kék agy, lebegő hologram, DNS-render, emberek, kezek, olvasható szöveg, betű, szám, logó, vízjel, emoji, keret, második kiemelőszín, telített nagy színfelület.

Elfogadási ellenőrzőlista:

1. A kép 16:9, a kiszállított WebP 1600×900, legfeljebb 400 KB.
2. Az alap egyszínű `#071016` a széleken is (a táblán belül nem látszik eltérő sáv).
3. Az öt funkció (rögzítés, címkék, szűrés, rangsor, archiválás) a `landing.hero.alt` szövege alapján azonosítható.
4. Nincs olvasható szöveg, betű, szám, logó.
5. Egyetlen kiemelőszín (cián); a címkeszínek csak apró négyzetek.
6. A GRAPHICS.md tiltólistájának egyetlen eleme sem látszik; átlátszósági átmenet (opacity gradient) sincs, az archivált sorok egyszínű, halvány vonallal rajzoltak.
7. A vonalak és csomópontok a KINEXIUM hélix-nyelvét idézik (GRAPHICS.md 9.1: két hajszálvékony szál, csomópontban végződő lépcsők).
8. A kép a bejelentkező oldalon sötét és világos témában is a márkafelülettel egybeolvad (képernyőkép a kézikönyvhöz).

Ha a kép nem felel meg, a `python graphics/generate_graphics.py --force hero` újragenerálja (a keret szerint legfeljebb 4 hívás); a döntés és a hívások száma a SUMMARY.md-be kerül.

Eredmény (G, a B1 hullám zárásakor elfogadva): 3 fizetős hívás a 4-ből, a 3. jelölt került be (mester 2752×1536, `hero.webp` 1600×900, 23 996 B). Az 1–6. pont teljesül (a 2. pont a háttér-normalizálással; a kiszállított képen a szegélysávok egyszínűek, a lossy WebP YUV-kerekítése a vörös csatornán 1 egységet ad hozzá, ami nem látható). A 7. pont közelítés: a hélix egy szinuszszál és egy egyenes középszál lépcsőkkel, nem szó szerinti kétszálas, ellentétes fázisú hélix; elfogadva, újragenerálás nincs. A 8. pontot a valódi `/login` oldalon az E2/E3 erősíti meg (`01-login.png`).

Eredmény (2026-11-22, a felhasználó engedélyével): a 4. fizetős hívás (a keret ezzel kimerült) jelöltje váltotta a 3.-at (mester 2752×1536, `hero.webp` 1600×900, 25 592 B). A 7. pont most teljesül: valódi kétszálú hélix lépcsőkkel és csomópontokkal, két-három cián lépcsővel; a címkék száma öt. A 4. pont a nyers kimeneten nem teljesült (négy angol felirat: „quick capture”, „list and ranking”, „tags and filtering”, „automatic expiry”), ezért a mester PNG-ről a feliratokat utólagos retus távolította el: mind a négy tiszta háttéren állt, a kitöltés soronként a környező háttérből interpolál, más képelem nem változott. A `graphics/hero.png` tehát a retusált mester; a nyers kimenet (`hero-4.png`), a retus szkriptje (`retouch-4.py`) és a korábbi jelöltek (`hero-1…3.png`) a munkakönyvtár `graphics\` mappájában vannak. Maradék apróságok: a hélix a jobb szélen a kért 8% helyett kb. 6% margóig ér, az archívumdoboz teteje enyhén trapéz, a listától balra halvány függőleges sáv látszik. A 8. pont a valódi `/login` oldalon sötét és világos témában ellenőrizve; a kézikönyv `01-login.png` és `02-login-2fa.png` képe az új grafikával újra elkészült. További újragenerálás csak a felhasználó újabb engedélyével.

### 8.9 Akadálymentesség, reszponzivitás, téma

| Szélesség | AppShell | Linkek | Táblázatok |
|---|---|---|---|
| 768 px alatt | oldalsáv `offcanvas` (`Sheet`), `Sidebar.Trigger` a felső sáv bal szélén; a keresőmező ikongombbá válik, amely teljes szélességű keresőt nyit; margó `space-4` | eszközsor két sorban; címkesor vízszintesen görgethető `ScrollArea`; sorban nincs címke- és meta-oszlop (a kitűzés jelölő és az `ExpiryMeter` marad); műveletek csak a menüben; csempe 1 oszlop; az oldalfej gombja `sm` | `overflow-x-auto` tároló; az oldal nem görget vízszintesen |
| 768–1023 px | oldalsáv `offcanvas`, alapból zárva; margó `space-6` | címkék a sorban, meta-oszlop nélkül | ugyanaz |
| 1024 px-től | 240 px-es oldalsáv | teljes sor | |

Dialógusok szélessége `min(440px, 100vw − 32px)`; mobilon a `Sheet` teljes szélességű.

Akadálymentességi szabályok:

- Fókusz: az app.css 2 px `ring`, 2 px eltolással; `Panel`-ben interaktív elem nincs. Dialógus nyitásakor az első mezőre, zárásakor a nyitó elemre kerül a fókusz.
- Ugrólink „Ugrás a tartalomra”; útvonalváltáskor a fókusz a `main`-re kerül, és az élő régió bemondja az oldal címét (`afterNavigate`).
- Fókuszmegőrzés (E fázis): kliensoldali navigáció más útvonalra után az `AppShell` a `main`-re teszi a fókuszt (`preventScroll`), ha az a `body`-ra esett (az első betöltéskor és csak lekérdezés-változásnál nem). Billentyűvel archivált sor után a fókusz a következő sorra kerül (a lista végén az előzőre); a toast „Visszavonás” gombja után a visszaállított sorra, ha a fókusz elveszett. Az importösszesítő bezárása után az „Import és export” menügombra tér vissza (az eredeti menüpont addigra eltűnt).
- Mobil címkesor: 768 px alatt a címkeszűrő chipjei vízszintes `ScrollArea`-ban vannak (vékony, témázott görgetősáv), 768 px-től tördelődő sorban; a `role="group"` és a neve változatlan.
- Jelszómezős űrlapok: rejtett `AccountUsername` mező (8.1.5), hogy a jelszókezelő a fiókhoz kösse a jelszót.
- Csak ikonos gomb: fordított `aria-label`; szöveg melletti ikon `aria-hidden="true"`.
- Állapot soha nem csak szín: ikon + szó minden jelölőn; a címkeszín mellett mindig a név.
- Kapcsoló jellegű gomb `aria-pressed`; `ToggleGroup` `aria-label`-lel.
- Űrlap: látható `Label`; hibánál `aria-invalid="true"` és `aria-describedby`; fókusz az első hibás mezőre.
- Élő régiók: találatszám és rögzítési eredmény `polite`; kritikus `StateAlert` `role="alert"`.
- Lista: `ul` + `li`; egy fő link soronként és nevesített gombok.
- Kontraszt: csak a GRAPHICS.md tokenjei (szöveg 4,5:1, vonal és fókusz 3:1).
- Érintés: ikongombok `pointer-coarse:size-9`; a sorműveletek érintőképernyőn mindig láthatók.
- Csökkentett mozgás: az app.css globális szabálya; JS-görgetés `behavior: 'auto'`; `kxm-flash-static`.
- Nyelv: `<html lang>` mindig az aktív nyelv; a nyelvválasztó elemei saját nyelvükön `lang` attribútummal.
- Oldalcím: `<svelte:head><title>{t('…pageTitle')} – Kinex Mark</title>` minden útvonalon.
- Ellenőrzés: `@axe-core/playwright`, `serious` és `critical` találat 0 (9.4).

Téma:

- A sötét az alap (`app.html` `class="dark"`, témaindító szkript, `ModeWatcher … disableHeadScriptInjection`).
- Bejelentkezés előtt a `ThemeToggle` csak a mode-watcher `localStorage` kulcsát állítja; bejelentkezve a `Preferences.theme` a forrás (`dark` | `light` | `system` → `setMode`), a váltás a preferenciába is ment.
- Márkafelület (bejelentkező oldal bal oszlopa, ráhúzási tábla kerete, QR-tároló) témától független; minden más szemantikus tokennel vált.
- A `dark:` Tailwind-variánst a saját kód nem használja.

---

## 9. Tesztstratégia

### 9.1 Tesztpiramis és területi mátrix

| Szint | Eszköz | Hol | Cél-mennyiség | Mit bizonyít |
|---|---|---|---|---|
| Statikus | `go vet`, gofmt, `tools/langcheck`, govulncheck, `svelte-check --fail-on-warnings`, prettier, `conventions.spec.ts` | CI `go`, `web` | minden fájl | típushelyesség, formátum, angol forráskód, ismert sérülékenység hiánya, design-system szabályok |
| Go unit | `testing`, táblavezérelt | `internal/*/…_test.go` | kb. 500 eset | csomaglogika (9.2.2) |
| Go HTTP | `httptest`, valódi store, `fstest.MapFS` UI | `internal/api/*_test.go` | kb. 250 eset | API-szerződés, jogosultsági mátrix, IDOR, CSRF, sütik, fejlécek, CSP, GDPR (9.2.3) |
| Go integráció | `app.Run` `127.0.0.1:0` címen | `internal/app/app_test.go` | kb. 10 eset | indulás, első admin, healthcheck, leállás, kulcsellenőrzés |
| Frontend unit | Vitest (jsdom) | `web/src/**/*.spec.ts` | kb. 150 eset | i18n, formázók, megjelenítési címke, rendezés, Markdown, DataTransfer, API-kliens, store-ok (9.3.2) |
| Frontend komponens | Vitest + `@testing-library/svelte` + `user-event` | `web/src/**/*.svelte.spec.ts` | kb. 80 eset | űrlapok, dialógusok, nézetek, nyelvválasztó, 2FA-lépés (9.3.3) |
| E2E | `@playwright/test`, valódi bináris | `web/e2e/specs` | kb. 70 eset | teljes folyamatok szerepkörönként, CSP-sértés nélkül, axe-ellenőrzéssel (9.4) |
| Feltáró GUI | Playwright MCP plugin | E fázis | kb. 25 forgatókönyv | vizuális minőség, GRAPHICS.md-megfelelés, reszponzivitás, kézikönyv-képek (9.5) |
| Konténer | `docker.yml` smoke (PR-en és a `ci.yml` `image` jobjában), FV-ben helyi `buildx` amd64 + arm64 | CI, FV | 1 | csak olvasható, jogok nélküli konténer `healthy`, 401 az API-n; a többplatformos build lefordul |

| Terület | Go unit | Go HTTP | Vitest | E2E | GUI (MCP) |
|---|---|---|---|---|---|
| Jelszó-hash, TOTP, helyreállító kód, korlátozás | `auth` | 3. csoport | `LoginForm`, `TwoFactorEnrollDialog` | `twofa`, `auth` | 2FA-képernyők |
| Munkamenet, süti, CSRF, fejlécek, CSP | | 3–5. csoport | `api/client` fejlécei | `security`, CSP-figyelő minden tesztben | |
| Láthatóság, IDOR, szerepkörök | `store` mátrix | 1–2. csoport | `LinkRow` menüje szerepkörönként | `sharing`, `security` | szerepkörönkénti menük |
| Metaadat-letöltés, SSRF | `fetch` | 6. csoport | `watchPending` | `links-detail` (cím és ikon a fixture-oldalról) | |
| Rendezés, kitűzés, használat (megnyitás és másolás) | `store` | 6. csoport | `order.ts`, `LinkRow` (egy megnyitás = egy `visit`) | `links-ranking`, `links-copy` | |
| Automatikus archiválás | `store`, `jobs` hamis órával (89/90/91 nap) | 7. csoport (`jobs/archive`) | `expiry.ts`, `ExpiryMeter` | `links-archive` | |
| Címkék, szűrés | `store` szűrők | 6. csoport | `query.ts`, `TagChip` | `tags`, `links-filter` | |
| Megjelenítés (címke, domain, kapcsolók) | | | `presentation.ts`, `LinkRow`, `LinkTile` | `links-view` | lista és csempe |
| Gyors rögzítés | `fetch.ExtractURL` | `links/quick` | `capture.ts`, `CaptureController` | `links-capture` | ráhúzási fátyol |
| Másolás | | | `markdown.ts`, `clipboard.ts` | `links-copy` | |
| Import, export | `bookmarks` | 6. csoport | `ImportExportMenu` | `links-import-export` | |
| i18n | | 9. csoport (kódok) | `parity`, `contract`, `format`, `index` | `i18n` | három nyelv |
| GDPR | `store` export, törlés | 8. csoport | | `privacy` | adatvédelmi oldalak |
| Admin | `store` | 7. csoport | `UsersTable`, `AuditTable` | `admin-users`, `admin-settings`, `admin-audit` | adminfelület |
| Design system | | | `conventions.spec`, `app-css.spec` | `a11y`, `responsive`, `theme` | teljes vizuális bejárás |

### 9.2 Go-tesztterv

#### 9.2.1 Közös elvek

- Lefedettség: `go test -race -covermode=atomic -coverprofile=coverage.out ./...` (Windowson `-race` nélkül, mert a race detectorhoz cgo és C-fordító kell), majd `go run ./tools/covercheck -profile coverage.out -total 85 -package 80`. Hatókör: `github.com/kaszasg/kinex-mark/internal/...`, kivéve `internal/testutil` és `internal/storetest`; a `cmd/kinex-mark` és a `tools/` nincs a kapuban (a `cmd` logikája az `internal/app`-ban él). Csomagcélok: `auth` legalább 90%, a többi legalább 85%, a kapu 80%-nál bukik.
- A CI a teszteket a `go.mod` minimumával (1.26.x, a `go 1.26.0` direktíva szerint) és a Docker-build verziójával (1.27.x) is futtatja `GOTOOLCHAIN=local` mellett. Helyben, cgo híján, a `-race` futás Dockerben megy: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27 sh -c "go vet ./internal/... && go test -race -count=1 ./internal/..."`, a minimum ellenőrzésére ugyanez `golang:1.26` image-dzsel és `GOTOOLCHAIN=local`-lal.
- Táblavezérelt tesztek, `t.Parallel()` ahol nincs közös állapot; minden adatbázisos teszt saját `t.TempDir()`-t kap.
- Idő: mindenhol `clock.Fake` (`testutil.T0 = 2026-01-05T10:00:00Z`); valódi alvás nincs, a háttérfeladat-tesztek a `RunArchive`/`RunPurge` közvetlen hívásával és csatornás szinkronizációval dolgoznak.
- Jelszó-hash a tesztekben `auth.NewHasher(1000, 4)`; egy külön teszt ellenőrzi a `DefaultIterations = 600000` kódolását.
- A tesztek nevei, kommentjei és üzenetei angolok; tesztadatként ékezetes szöveg megengedett.

#### 9.2.2 Csomagonként

| Csomag | Lefedett viselkedés |
|---|---|
| `config` | alapértékek; kapcsoló > környezeti változó > alap; minden érvénytelen érték angol hibája (`APP_BASE_URL` úttal, hibás CIDR, `APP_FETCH_WORKERS` 0 és 17, `APP_FETCH_TIMEOUT` 500ms és 61s, tanúsítvány kulcs nélkül, ismeretlen `APP_SECURE_COOKIES`, `APP_ADMIN_USER` szóközzel, `/`-rel és 2 karakterrel → `invalid username`); `APP_ALLOW_LOOPBACK_FETCH`; `-unlock-user` és `-reset-password` → `AdminCommand`, egymást kizárják, a `-healthcheck`-kel és a `-version`-nel sem kombinálhatók; `EnvLockedSettings` |
| `clock` | `Fake.Set`, `Advance`, párhuzamos olvasás |
| `model` | `FormatTime`/`ParseTime` oda-vissza és fix szélesség; `Role` jogosultságai; `TagColor` érvényessége; `AuditEvents` egyedisége |
| `auth` | PBKDF2: hash/verify, hibás alak, rossz jelszó, `needsRehash` kisebb iterációnál, egyedi só; jelszószabály minden kódja határértékekkel (11/12, 128/129 rúna, ékezetes rúnák, 5/6 különböző karakter, felhasználónév kis-nagybetű nélkül); TOTP: az RFC 6238 SHA-1 tesztvektorai 6 jegyre vágva (T = 59 → `287082`, 1111111109 → `081804`, 1111111111 → `050471`, 1234567890 → `005924`, 2000000000 → `279037`, 20000000000 → `353130`, titok `12345678901234567890` ASCII), ±1 lépés elfogadva, ±2 elutasítva, `lastStep` miatti elutasítás, URI-formátum; token hossza és hash-e; helyreállító kód formátuma és normalizálása; kulcstartó: létrehozás fájlba, betöltés, környezeti kulcs elsőbbsége, hibás base64, rossz hossz, `Seal`/`Open` oda-vissza, idegen társított adat elutasítása, `MAC` célonként eltér, `CheckValue` stabil; korlátozás: `Check`/`Reserve`/`Confirm`/`Refund` táblateszt (küszöb a foglalással együtt, ablak lejárta, fokozás, 24 órás plafon, a `Refund` nem megy 0 alá és nem old zárolást); `UserThrottleKey` determinisztikus, `u:` + 64 hex, a felhasználónév nem olvasható ki belőle; `IPThrottleKey`: `1.2.3.4` és `::ffff:1.2.3.4` ugyanaz a `/32` kulcs, két különböző IPv4 két kulcs, `2001:db8::1` és `2001:db8::2` ugyanaz a `/64` |
| `fetch` | hamis forrással (9.2.4), `Policy{AllowPrivate: true, AllowLoopback: true}` és a forrás portjára szűkítő `DialAllow` mellett: cím `<title>`-ből és `og:title`-ből, `windows-1250` és `iso-8859-2` kódolás fejlécből és `<meta charset>`-ből, entitások, 500 rúnás csonkolás; ikonválasztás sorrendje, `<base href>`, relatív URL, `data:` ikon, `/favicon.ico` tartalék, túl nagy ikon, hamis típus, SVG `<script>`-tel; átirányítás 5 és 6 lépéssel; időtúllépés; 1 MiB-os HTML; nem HTML (`not_html`); 404, 500, 429 és az átmenetiség; SSRF: `CheckIP` táblateszt minden tartománnyal a négy házirend-kombináció mellett (a loopback és a link-local csak `AllowLoopback`-kel nyílik, az `AllowPrivate` nem nyitja), `::ffff:127.0.0.1`, NAT64 mindkét előtaggal, 6to4 (`2002:7f00:1::`), Teredo (`2001:0:…` negált `127.0.0.1`), `::7f00:1`, `168.63.129.16`, `fd20:ce::254`; `AllowLoopback: false` mellett a 127.0.0.1-es hamis forrás `blocked_address` (dial-szintű tiltás bizonyítéka); átirányítás a 127.0.0.1 egy másik portjára → `blocked_address` (a `DialAllow` és a valódi házirend is); a `Transport.DisableKeepAlives` be van kapcsolva, és házirend-váltás után a következő kérés már tilt; a `Control` függvény közvetlen hívása tiltott címmel (DNS-rebinding); tiltott port; `CleanText` táblateszt (C0, C1, U+007F, bidi-felülíró és -izoláló karakterek, `keepNewlines`); `NormalizeURL` (a 2048 bájtot az újraszerializálás után túllépő bemenet `too_long`) és `ExtractURL` táblatesztjei |
| `bookmarks` | Netscape-fájl mappákkal, `TAGS` attribútummal, `ADD_DATE`-tel, `<DD>` leírással; mappanév címkévé lapítva; hibás HTML; bejegyzéskorlát; JSON oda-vissza; írás után visszaolvasás |
| `store` | migráció üres adatbázison és ismételten (idempotens); minden metódus sikeres és hibaága; `CHECK`- és egyediségi korlátok; kaszkádok (felhasználó, link, címke törlése); láthatósági mátrix: tulajdonos, `all`, `role` mindhárom szerepkörre, `user`, lefokozott tulajdonos, inaktív tulajdonos, törölt megosztás; rendezés (kitűzött + használat, döntetlen `last_used_at`, soha nem használt, `created`, `title`; a kitűzött csoporton belül is: két kitűzött link eltérő `use_count`-tal, illetve azonos `use_count` és eltérő `last_used_at` mellett a nagyobb elöl); szűrők (`q` ékezetes kisbetűsítéssel, címkenévre is, `any`/`all`, `untagged`, `scope`, `archived`); lapozás, `total` és `counts.pinned`/`counts.archived` az utolsó lap után is; személyes állapot upsertjei; a `BaselineAt`, `LastUsedAt`, `IconSHA256` nyers mezők (a származtatott mezőket a `dto_test.go` fedi); `ArchiveInactive` hamis órával (89/90/91 nap, 0 = soha, felhasználói felülírás, kitűzött kimarad, megosztott link a megosztás idejétől, visszaállítás újraindítja); `AdvanceTOTPStep` visszajátszás; `UseRecoveryCode` egyszeri; korlátozás: `ThrottleUpdate` 20 párhuzamos goroutine-nal → pontosan 20 foglalás, egyik sem vész el; `ChallengeAttempt` 10 párhuzamos hívással → legfeljebb 5 sikeres foglalás; `CreateChallenge` törli a felhasználó többi kihívását; `ClearUserThrottle`; `DeleteShare` idegen link azonosítójával 0 sor (`ErrNotFound`), a megosztás megmarad; címeredet: létrehozás címmel `manual`, cím nélkül `none`; URL-változás `fetched` címet és ikont töröl, `manual` címet megtart, a letöltési mezőket nullázza és `next_fetch_at = now`; `ApplyFetchResult`: `none`/`fetched` cím felülíródik és `fetched` lesz, `manual` megmarad, ikon nélküli eredmény `icon_id = NULL`; `UpdateLink` utáni régi URL-es eredmény eldobódik (a jobs-csomagtól ide költözött integráció); `Enqueue` nélkül létrehozott link a `DueFetches`-ben; bérelt link második `BeginFetch`-e `ErrNotFound`; `DeleteUser` pszeudonimizálás (az átadott auditsor is, az `actor_id IS NULL` sorok a felhasználónévvel, a `details`-ben nincs név) és `last_admin`; `Purge` minden ága; `ImportLinks` duplikációval, korláttal és `CleanText`-tel; `ExportUser` teljessége, titok nélkül; visszavont megosztás után a `sharedWithMe` sorában nincs URL; az `auditEvents` a célként érintett eseményeket is tartalmazza |
| `jobs` | hamis store-ral és hamis fetcherrel: sor tele, folyamatban lévő duplikáció kiszűrése, átmeneti hiba backoffja (1 perc, 10 perc, `failed`), végleges hiba, `blocked`, `not_html` → `ok` a `fetch_error` megtartásával; az `ApplyFetchResult` a munkás által várt argumentumokkal hívódik (az URL-változás közbeni eldobást a store-teszt bizonyítja, így a B4 nem függ a B1-től); a hibák naplójában nincs URL (pufferelt `slog` kimenet: csak `link_id`, `code`, `host`); leállítás: a munkások a `Wait` időkorlátján belül kilépnek; `RunArchive` a beállított alapértékkel |
| `api` | 9.2.3; `dto_test.go`: `toLinkItem` táblateszt (`inactiveSince` = `max(BaselineAt, LastUsedAt)`, `archiveDueAt` = `inactiveSince + archiveDays`, kitűzött, archivált és 0 napos esetben `null`; `iconUrl` csak `IconSHA256` mellett) |
| `app` | `Run` `127.0.0.1:0` címen ideiglenes adatkönyvtárral: indulás, első admin generált jelszóval (a napló pufferéből), első admin környezeti jelszóval (utána Warn a változó eltávolításáról), nem üres adatbázison beállított `APP_ADMIN_PASSWORD` → Warn, gyenge környezeti jelszóra indulási hiba; `RunAdminCommand`: `-unlock-user` törli a korlátozási sort, `-reset-password` új jelszót ír a kimenetre (a naplóba nem), `must_change_password = 1`, munkamenetek törölve, audit `via: "cli"`, ismeretlen névre 1-es kilépési kód, `/healthz`, leállítás kontextuslezárással; kulcseltérésre megállás; `CheckHealth` élő és leállított szerverrel |
| `web` | `assets_test.go`: a két font mérete és SHA-256-ja (T12), az OFL-fájlok léte és első sora, a `robots.txt` tartalma, a `Files()` tartalmazza az `index.html`-t és a `fonts/` mappát; `hero_test.go`: `static/images/hero.webp` létezik, `RIFF….WEBP` fejléc, 1600×900 (a VP8/VP8L/VP8X fejlécből), legfeljebb 400 KB |
| `tools/covercheck`, `tools/langcheck` | profil-egyesítés, küszöbök, kizárás, Markdown-kimenet; langcheck: tiltott rúna, kizárt útvonalak, kimeneti formátum |

#### 9.2.3 HTTP-tesztek

Felépítés (`internal/api/apitest_test.go`):

```go
type testEnv struct {
	t     *testing.T
	clock *clock.Fake
	store *store.Store     // real store in t.TempDir()
	jobs  *fakeJobs        // records Enqueue calls
	srv   *httptest.Server // api.New(Deps{...}) with an fstest.MapFS UI
}

func newEnv(t *testing.T, opts ...envOption) *testEnv
func (e *testEnv) client(t *testing.T) *apiClient // own cookie jar, X-Kinex-Request: 1, Origin
func (e *testEnv) loginAs(t *testing.T, role model.Role) (*apiClient, model.User)
func (c *apiClient) do(method, path string, body any) *http.Response
func decode[T any](t *testing.T, res *http.Response) T
func expectError(t *testing.T, res *http.Response, status int, code string)
```

Az `api` teszt a valódi `store`-t a tesztben importálja (a `store` nem függ az `api`-tól); a UI `fstest.MapFS` két inline szkripttel és egy `_app/immutable/x.js` fájllal. A `loginAs` a felhasználót `storetest.SeedUser`-rel, a munkamenetet közvetlenül `store.CreateSession`-nel hozza létre, és a sütit maga állítja be, így nem függ a B5 bejelentkezési kezelőjétől (B5 0. mérföldköve, 11.5). A `newEnv` opciói: `withBaseURL`, `withTrustedProxy`, `withAddr` (a Host-ellenőrzéshez), `withLogBuffer` (a Warn-naplók ellenőrzéséhez).

Kötelező tesztcsoportok:

1. **Útvonaltábla-mátrix**: a `routes.go` táblájának minden sorára anonim, ügyfél, fejlesztő és admin klienssel; a várt 401/403 az 5.13 szerint; minden regisztrált útvonal szerepel a táblában.
2. **IDOR**: A felhasználó linket, címkét és megosztást hoz létre; B (minden szerepkörrel) a link minden végpontját (`GET`, `PATCH`, `DELETE`, `tags`, `pin`, `archive`, `visit`, `refresh`, `icon`, `shares`), A címkéjét (`PATCH`, `DELETE`, `tagIds`-ben) és A munkamenetét (`DELETE /api/me/sessions/{id}`) hívja: mindig 404 `not_found` (illetve `validation_failed` + `unknown` a `tagIds`-ben), és A adatai változatlanok. Megosztás után B olvashat, módosításra 403 `not_owner`. Beágyazott azonosító: B (fejlesztő) saját linkje azonosítójával A megosztását törölné (`DELETE /api/links/<B linkje>/shares/<A shareId>`): 404, A megosztása változatlan.
3. **Hitelesítés**: bejelentkezés sikeres és hibás ágai, korlátozás (5 hiba után 429 `Retry-After`-rel, ismeretlen névre is), inaktív fiók, 2FA kétlépéses folyamat, visszajátszott kód, helyreállító kód egyszer, kihívás lejárata és 5 próbálkozás, tétlenségi és abszolút lejárat hamis órával, munkamenet-visszavonás, jelszócsere a többi munkamenet törlésével, kötelező jelszócsere- és hozzájárulás-kapu a mentes útvonalakkal. Továbbá:
   - második tényező: jelszó, 4 hibás kód, újra jelszó → 429 (a helyes jelszó nem nullázta a számlálót, a jelszólépés foglalása 2FA mellett nem térül vissza, így 1 + 4 = 5 hiba zárol); két párhuzamos kihívásból a régebbi a második bejelentkezés után `challenge_invalid`; zárolt felhasználói vagy IP-kulcs mellett a TOTP-lépés 429; a TOTP-hiba az IP-kulcson is számít;
   - párhuzamosság: 20 egyidejű hibás jelszó egy fiókra → legfeljebb 5 válasz 401, a többi 429, és a `login_throttle` sor `failures` értéke legalább 5;
   - adattakarékosság: jelszó alakú, nem létező felhasználónévvel (`Tavasz2026!jelszo`) való próbálkozás után a szöveg sem az `audit_log`, sem a `login_throttle` egyetlen oszlopában nem szerepel; létező fiók hibájánál az `actor_name` a fiókrekordból jön;
   - kijelentkezés: a munkamenet sora törlődik, a korábban kimásolt token sütiként visszaküldve 401; bejelentkezés érvényes régi sütivel → a régi sor törlődik;
   - IPv4-ként leképezett IPv6 kliens (`::ffff:203.0.113.5` megbízható proxy `X-Forwarded-For`-jából) saját `/32` kulcsot kap, nem közös `::/64`-et.
4. **CSRF és Host**: hiányzó fejléc, idegen `Origin`, `Sec-Fetch-Site: cross-site`, `APP_BASE_URL` origin, megbízható proxy fejlécei és nem megbízható forrásból érkező hamis `X-Forwarded-*` (a fejlécek hatástalanok, és egyetlen Warn-sor kerül a naplóba); `APP_BASE_URL` mellett idegen `Host` → 421 `host_mismatch` `GET`-re is; `APP_BASE_URL` nélkül loopback címen `Host: evil.example` → 421, `localhost:<port>` és `127.0.0.1:<port>` elfogadva; a `/healthz` Host-ellenőrzés nélkül 200.
5. **Sütik és fejlécek**: `auto` mód http-n és `X-Forwarded-Proto: https`-sel megbízható proxyról; `https` `APP_BASE_URL` mellett `Secure`, `__Host-` előtag és HSTS proxy-bizalom nélkül is; `__Host-` előtag; minden biztonsági fejléc; a CSP-ben a két inline szkript hash-e pontosan, `'unsafe-inline'` nélkül; az ikonvégpont saját CSP-je és ETag/304; SPA-gyorsítótár szabályai (TECHNOLOGY.md 4.7 + `fonts/`).
6. **Linkek**: létrehozás minden mezőhibával, duplikáció `details.linkId`-vel, gyors rögzítés szövegváltozatokkal (uri-list, sima szöveg, domain előtag nélkül, nincs URL), sorba tétel a `fakeJobs`-ban, lista minden query-paraméterrel és hibás paraméterrel, `LinkPage.counts`, `LinkItem` `inactiveSince`/`archiveDueAt` mezői, személyes állapot (archivált link kitűzése visszaállít), `visit` hívásonként pontosan egy használat, export/import mindkét formátumban (vezérlő- és bidi-karakteres importált cím tisztítva), 2 MiB fölött 413, címke-CRUD; címeredet: `POST` címmel → `titleSource: "manual"`, `PATCH` URL-változás → `fetchStatus: "pending"`, `fetchError: null`, `title` üres (ha `fetched` volt), `iconUrl: null` és `Enqueue`; `refresh` → `Enqueue` és nullázott letöltési mezők.
7. **Admin**: felhasználó-CRUD, generált jelszó, `self_modification`, `last_admin`, jelszó- és 2FA-visszaállítás (a 2FA-visszaállítás a cél minden munkamenetét törli), feloldás, munkamenetek visszavonása, beállítások határértékei (`privacySupervisoryAuthority` 301 rúnával `too_long`), `setting_locked` környezeti zárolásnál, tájékoztató-verzió emelése után minden felhasználónál `consent_required`, audit szűrés és lapozás, `jobs/archive`; megerősítés: törlés, jelszó- és 2FA-visszaállítás, szerepkör- és aktív állapot-váltás `currentPassword` nélkül 400 `validation_failed` (`required`), hibás jelszóval 400 `current_password_invalid` és a művelet nem történik meg, öt hibás megerősítés után 429; `AdminUser.sharedLinkCount`; a `user.updated` `details`-e megjelenítettnév-változásnál csak `{changed: ["displayName"]}`.
8. **GDPR**: export tartalma (titok nélkül, az 5.8.1 alak szerint; az admin `user.password_reset` eseménye és a fiókra érkező sikertelen belépés `role: "target"`, illetve `role: "actor"` sorként benne van; `security.lockedUntil` zárolás után kitöltött; visszavont megosztás után a `sharedWithMe` sorában `url: null`); fióktörlés bekapcsolt 2FA mellett `code` nélkül 400 `validation_failed`, hibás kóddal 401 `invalid_totp`; fióktörlés (önkiszolgáló és admin) után a felhasználó minden sora törölve, a `user.self_deleted`, illetve `user.deleted` sor is álnévvel, egyetlen auditsor `actor_name`, `target_name` és `details` mezője sem tartalmazza a felhasználónevet és a megjelenített nevet, az `actor_id IS NULL` hibasorok is pszeudonimizálva; a megosztott linkjei a címzetteknél eltűnnek; a `PublicConfig.privacy` megőrzési mezői és `supervisoryAuthority`-je.
9. **Hibatörzs**: minden 5.3-as hibakód legalább egyszer előfordul egy tesztben (az `errors_test.go` a kódlistát bejárva ellenőrzi, hogy mindegyikre van teszthivatkozás); a `contract_test.go` a golden-fájlt (9.6).

#### 9.2.4 Hamis HTTP-forrás

`testutil.NewOrigin(t, routes map[string]http.HandlerFunc) *httptest.Server`; előre gyártott kezelők: `HTMLPage(title, charset, iconLinks...)`, `Redirect(n)`, `RedirectTo(url)`, `Slow()` (a kérés kontextusáig blokkol), `Big(size)`, `Icon(contentType, bytes)`, `Status(code)`. A `testutil.AllowOnly(srv) func(netip.AddrPort) bool` csak a forrás saját címét és portját engedi. A fetcher `Policy{AllowPrivate: true, AllowLoopback: true}` és `Options.DialAllow = testutil.AllowOnly(srv)` beállítással éri el (a forrás loopbacken fut, de más loopback-port így sem érhető el); a tiltási tesztek `AllowLoopback: false`-szal hívják, az átirányítási teszt a forrásból a `127.0.0.1` egy másik portjára irányít.

### 9.3 Vitest: frontend unit- és komponenstesztek

#### 9.3.1 Konfiguráció

`web/vitest.config.ts`:

```ts
import { svelteTesting } from '@testing-library/svelte/vite';
import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config.ts';

export default mergeConfig(
	viteConfig,
	defineConfig({
		plugins: [svelteTesting()],
		test: {
			environment: 'jsdom',
			include: ['src/**/*.{test,spec}.{js,ts}'],
			setupFiles: ['./src/lib/test/setup.ts'],
			expect: { requireAssertions: true },
			restoreMocks: true,
			coverage: {
				provider: 'v8',
				include: ['src/lib/**/*.{ts,svelte}'],
				exclude: [
					'src/lib/components/ui/**',
					'src/lib/hooks/**',
					'src/lib/test/**',
					'src/lib/i18n/messages/**',
					'src/lib/api/types.ts',
					'**/*.{test,spec}.ts',
					'**/*.d.ts'
				],
				reporter: ['text-summary', 'json-summary', 'lcov', 'html'],
				reportsDirectory: './coverage',
				thresholds: { statements: 85, branches: 80, functions: 85, lines: 85 }
			}
		}
	})
);
```

- A `src/routes/**` oldalai vékony kompozíciók; viselkedésüket az E2E fedi, ezért a Vitest-hatókörön kívül vannak. A logika a `src/lib/**` alatt él.
- `src/lib/test/setup.ts`: `@testing-library/jest-dom/vitest`, `matchMedia` és `IntersectionObserver` jsdom-pótlék, `navigator.clipboard` mock, i18n `hu`-ra állítva minden teszt előtt.
- `src/lib/test/render.ts`: `renderWithApp(Component, props, { me?, locale? })` a session és az i18n előkészítésével.
- `src/lib/test/fixtures.ts`: `makeLink(partial)`, `makeMe(role, partial)`, `makeTag(partial)`, `apiErrorBody(code, fields?)`.
- `src/lib/test/mock-api.ts`: `mockApi(routes)` (metódus + útvonal → válasz) a `vi.stubGlobal('fetch', …)`-hoz.

#### 9.3.2 Egységtesztek

| Célterület | Esetek |
|---|---|
| `links/presentation.ts` | a 8.7.3 példatáblája; vágás; többsoros leírás; 40/41 kódpontos URL-határ; `www.` és záró `/` levágása; IDN host; mind a 16 kapcsolókombináció |
| `links/order.ts` | kitűzött elöl; a kitűzött csoporton belül is használat szerint (két kitűzött eltérő `useCount`-tal; azonos `useCount` és eltérő `lastUsedAt`); azonos `useCount` mellett `lastUsedAt`; `null` a végén; `created`, `title` rendezés; stabil `id` döntetlen; ugyanaz a sorrend, mint a store-teszt táblájában |
| `links/markdown.ts` | escape: `[]`, `\`, sortörés; URL-ben szóköz és zárójel |
| `links/expiry.ts` | kitűzött, archivált, 0 nap → `null`; 5/4/…/1 szegmens határai; 7 és 8 nap; ma esedékes; lejárt |
| `links/capture.ts` | uri-list kommenttel és több sorral; `text/plain` URL-lel és anélkül; `text/html` horgonyszöveg; `Files` és `application/x-kinex-mark` kizárása; 20-as korlát; `isEditableTarget` |
| `links/query.ts` | `LinkQuery` ↔ URL oda-vissza; alapértékek elhagyása; `tags` és `untagged` kizárása; hibás értékek eldobása |
| `i18n/index.spec.ts` | `t` helyőrző, szám- és dátumparaméter; plural `zero`/`one`/`other` mindhárom nyelven; hiányzó kulcs tartaléka; `hasMessage` létező és nem létező (dinamikusan képzett) kulcsra; `tDynamic` a fordítással, illetve a `fallback`-kel és a nyers kulccsal; `setLocale` (`localStorage` `kxm-locale`, `<html lang>`); `detect` sorrendje |
| `i18n/format.spec.ts` | `1284`, `12 480` U+00A0-val (hu), `24,8`; en `1,284`, de `1.284`; dátum `2026. 11. 05.`, `5 Nov 2026`, `05.11.2026`; idő `14:32:08`; relatív idő; `–` |
| `i18n/parity.spec.ts` | a három katalógus kulcshalmaza azonos; a plural-objektumok kulcsai azonosak és lefedik a nyelv `Intl.PluralRules` kategóriáit; a helyőrzők halmaza kulcsonként azonos; nincs üres szöveg |
| `i18n/contract.spec.ts` | 8.5.9 és 9.6 |
| `api/client.ts` | fejlécek (`X-Kinex-Request`, `Content-Type` csak törzzsel), query-építés, 204, hibatörzs → `ApiError` mezőkkel, kód nélküli hiba státuszból, `Retry-After`, `network_error`, `AbortError` továbbengedése, `onAuthFailure` a három kódra és a `skipAuthRedirect` kivétel, `download` fájlnév |
| `api/*.ts` | minden függvény a helyes metódussal, útvonallal és törzzsel hív (táblateszt az 5.14 sorai szerint) |
| `state/session` | `ensure` egyszeri kérés párhuzamos hívásnál; 401 → `anonymous`; `gate` sorrend; `setMe` nyelv- és témaalkalmazás a kifejezett választás szabályával; `logout` |
| `state/preferences` | 400 ms-os összevonás egy kérésbe; hiba → visszaállítás |
| `state/links` | URL ↔ query; keresés debounce; előző kérés megszakítása; optimista kitűzés átrendez és hibánál visszaáll; archivált link kitűzése visszaállít; archiválás kivétele a listából; `counts` a `LinkPage`-ből; `visit` számláló; a `tags` store `onTagsChanged` értesítése `reload`-ot vált ki; `watchPending` 1,5 s-os lekérdezés és 20 s-os leállás (`vi.useFakeTimers`); `loadMore` offset |
| `state/tags` | `byId`; létrehozás után a lista rendezett; minden módosítás után az `onTagsChanged` feliratkozói hívódnak, a leiratkozás működik |
| `state/link-stats` | `refresh` a `limit=1&archived=exclude` kéréssel, `active` a `total`-ból; hibánál az előző érték marad |
| `tags/colors.ts` | a hét kulcs mind leképezve, `primary` és állapotszín nincs köztük |
| `utils/*` | `copyText` tartalék; `shouldHandleShortcut`; `describeUserAgent` táblateszt; `qrMatrix` determinisztikus mérete; `safeNext` táblateszt: `/tags?x=1#a` megmarad, `/\evil.example`, `//evil.example`, `https://evil.example`, `/%5Cevil.example` (a dekódolt alak sem vezet idegen originre), `javascript:alert(1)`, vezérlőkarakteres és üres bemenet → `/` |
| `test/app-css.spec.ts` | `src/app.css` kommentek nélkül = a GRAPHICS.md 13.2 blokkja kommentek nélkül, normalizálva; `../ai/GRAPHICS.md` hiányában kihagyva |
| `test/conventions.spec.ts` | 9.8 |

#### 9.3.3 Komponenstesztek

| Komponens | Ellenőrzés |
|---|---|
| `TagChip` | név és színosztály; gombként `aria-pressed` váltás |
| `LinkRow` | címke a precedencia szerint; kapcsolók hatása a másodlagos sorokra; kitűzött, megosztott, archivált, függő jelölők szóval; a menü elemei tulajdonos/címzett és admin/fejlesztő/ügyfél szerint (megosztás csak `canShare && isOwner`); a címkére kattintás `visit`-et hív; Enter a fókuszált horgonyon pontosan egy `visit` (a jsdom-ban a natív aktiválást `user-event` `keyboard('{Enter}')` adja); középső kattintás (`auxclick`, `button: 1`) egy `visit`, jobb kattintás egy sem; „URL másolása” és „Másolás Markdownként” után egy `visit`; archivált soron is van kitűzés gomb |
| `LinkTile` | ugyanez csempében; `line-clamp`; `+n` chip |
| `ExpiryMeter` | szegmensszám; 7 napon belül ikon és szó; `aria-label` |
| `LinkDialog` | kötelező URL kliensoldalon; `ApiError` mezőhibák a mezők alatt, `aria-invalid`; `link_duplicate` alert a „Megjelenítés a listában” gombbal |
| `LinkDetailsSheet` | minden adat megjelenik (ikon, URL, cím, leírás, címkék, megnyitásszám, rögzítés, utolsó használat, archiválás ideje, metaadat-állapot `fetch.errors.*` szöveggel); tulajdonos nélkül nincs szerkesztés és törlés |
| `ShareDialog` | célponttípus-váltás mezői; könyvtárkeresés; visszavonás |
| `CaptureController` | `drop` → `quickCapture`; `created: false` → kiemelés; szerkeszthető elemen és nyitott dialógusnál a `paste` nem rögzít |
| `ImportExportMenu` | 2 MiB fölött kérés nélküli hiba; eredmény-dialógus számai |
| `LoginForm` | 1. lépés hibái; `totp_required` → 2. lépés fókusszal; helyreállító kódra váltás; `challenge_invalid` vissza az 1. lépésre; `too_many_attempts` a gomb tiltásával |
| `PasswordChangeForm` | egyezés-ellenőrzés; jelszókódok a mező alatt |
| `TwoFactorEnrollDialog` | három lépés; a „Kész” a jelölőnégyzetig tiltott; QR SVG megjelenik |
| `ConfirmDialog` | kérdő cím, következmény, tárgyat nevező gomb; `busy` állapot |
| `LocaleSwitcher` | váltás után a szövegek és a `<html lang>` változnak |
| `ErrorAlert`, `FieldError`, `EmptyState`, `PasswordInput` | szöveg a kódból; `role="alert"` csak kritikusnál; megjelenítés-kapcsoló |
| `UsersTable` | saját sor műveletei tiltva; állapot-jelvények; a törlés, a jelszó- és 2FA-visszaállítás dialógusa a megerősítő jelszó nélkül nem küldhető, `current_password_invalid` a mező alatt; a törlési dialógus a `sharedLinkCount`-ot mutatja |
| `DeleteAccountDialog` | bekapcsolt 2FA mellett kódmező; `ownSharedLinkCount > 0` esetén figyelmeztetés a darabszámmal |
| `AuditTable` | eseményfordítás, ismeretlen esemény kódként; `details` kulcsai és felsorolt értékei fordítva (`reason: unknown_user` → „Ismeretlen felhasználónév”), ismeretlen kulcs nyers kódként |

### 9.4 Playwright E2E a valódi binárison

#### 9.4.1 Futtatókeret

`web/playwright.config.ts`:

```ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
	testDir: 'e2e/specs',
	globalSetup: './e2e/support/global-setup.ts',
	globalTeardown: './e2e/support/global-teardown.ts',
	timeout: 90_000,
	fullyParallel: false,
	workers: 1,
	retries: process.env.CI ? 1 : 0,
	reporter: [['list'], ['html', { open: 'never' }]],
	use: {
		baseURL: 'http://127.0.0.1:18080',
		locale: 'hu-HU',
		timezoneId: 'Europe/Budapest',
		colorScheme: 'dark',
		viewport: { width: 1440, height: 900 },
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure'
	},
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } },
			testIgnore: [/responsive\.spec\.ts/, /\.last\.spec\.ts/]
		},
		{ name: 'mobile', use: { ...devices['Pixel 7'] }, testMatch: /responsive\.spec\.ts/, dependencies: ['chromium'] },
		// Specs that bump the privacy notice version run last, after every other project.
		{
			name: 'chromium-last',
			use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } },
			testMatch: /\.last\.spec\.ts/,
			dependencies: ['chromium', 'mobile']
		}
	],
	webServer: {
		command: 'node e2e/support/start-server.mjs',
		url: 'http://127.0.0.1:18080/healthz',
		reuseExistingServer: false,
		timeout: 30_000,
		stdout: 'pipe',
		gracefulShutdown: { signal: 'SIGTERM', timeout: 5_000 }
	}
});
```

`e2e/support/start-server.mjs` (Node, függőség nélkül):

1. Bináris: `KINEX_MARK_BIN`, alapból `../dist/kinex-mark` (`.exe` Windowson). Ha nincs, `Run ".\build.ps1" or "make build" first.` üzenettel kilép.
2. Adatkönyvtár: `fs.mkdtemp(path.join(os.tmpdir(), 'kinex-mark-e2e-'))`.
3. Fixture-forrás (`fixture-origin.mjs`, `node:http`, `127.0.0.1:18081`, az `e2e/fixtures/site/` tartalmával):

| Út | Válasz |
|---|---|
| `/alpha` | UTF-8 oldal, `<title>Alpha docs</title>`, `<link rel="icon" href="/alpha.png">` |
| `/beta` | `windows-1250` kódolású oldal, cím „Árvíztűrő tükörfúrógép”, ikon nélkül (a `/favicon.ico` tartalék) |
| `/svg` | oldal SVG-ikonnal |
| `/plain` | `text/plain` (`not_html`) |
| `/slow` | 10 s-ig nem válaszol (időtúllépés) |
| `/p/<id>` | egyedi oldal `<title>Fixture <id></title>`-vel és PNG-ikonnal; tesztenként egyedi URL |
| `/favicon.ico`, `/alpha.png` | ikonok |

4. A bináris indítása (az örökölt `APP_*` környezeti változókat a szkript elhagyja, így egy helyi `APP_BASE_URL` vagy `APP_SECRET_KEY` nem szivároghat a futásba; a gyerekfolyamat PID-jét a `web/test-results/.e2e-server-pid` fájlba írja): `APP_ADDR=127.0.0.1:18080`, `APP_DATA_DIR=<temp>`, `APP_ADMIN_USER=admin`, `APP_ADMIN_PASSWORD` a `support/constants.mjs` állandója (megfelel a jelszószabálynak, ezért `must_change_password = 0`), `APP_ALLOW_PRIVATE_FETCH=true` (egyben a `lockedByEnv` ellenőrzése) és `APP_ALLOW_LOOPBACK_FETCH=true` (a fixture-forrás a `127.0.0.1`-en fut, 6.8), `APP_SECURE_COOKIES=false`, `APP_FETCH_TIMEOUT=5s`, `APP_LOG_LEVEL=warn`. Az `APP_BASE_URL` nincs beállítva; a loopback-cím miatt a Host-ellenőrzés a `127.0.0.1:18080`-at elfogadja (6.3).
5. Az indulás előtt törli a `kinex-mark-e2e-*` mintájú, 1 óránál régebbi könyvtárakat a `os.tmpdir()` alatt (a korábbi, erőszakkal leállított futások maradéka), az új könyvtár útját pedig a `web/test-results/.e2e-data-dir` fájlba írja.
6. SIGTERM/SIGINT esetén a gyereket leállítja, megvárja az `exit` eseményt, csak utána törli az adatkönyvtárat (Windowson a nyitott `app.db` zárolná); a törlés hibáját csak naplózza. A Playwright Windowson a `gracefulShutdown`-t figyelmen kívül hagyja és a folyamatfát erőszakkal állítja le, ezért ott ez a lépés nem fut; a takarítást az 5. lépés és a `global-teardown.ts` végzi.

`e2e/support/global-teardown.ts`: a Playwright 1.65 a globális teardownt a webszerver leállítása **előtt** futtatja (a `webServer` pluginként a feladatsor elején áll, a teardownok fordított sorrendben futnak, így a plugin leállítása az utolsó; `node_modules/playwright/lib/runner/index.js`), Windowson pedig a futó szerver zárolja az `app.db`-t. Ezért a teardown előbb beolvassa a `.e2e-server-pid` fájlt, és a szervert maga állítja le (`stopProcess`: SIGTERM, ami Windowson TerminateProcess; legfeljebb 10 s várakozás, utána SIGKILL), törli a PID-jelölőt; ezután beolvassa a `.e2e-data-dir` fájlt, legfeljebb 10 másodpercig 500 ms-onként újrapróbálva törli a könyvtárat (`removeDirWithRetry` a `support/cleanup.mjs`-ből, amíg a fájlzár fel nem szabadul), majd a jelölőfájlt; hibánál csak figyelmeztet. A `support/cleanup.mjs` közös segédei: `removeStaleDataDirs`, `removeDirWithRetry`, `isProcessAlive`, `stopProcess`.

`e2e/support/global-setup.ts`: API-n át (`X-Kinex-Request: 1`, `Origin`) belép adminként, elfogadja a tájékoztatót, létrehozza a `dev1` (developer), `cust1` és `cust2` (customer) felhasználót, mindegyikkel belép, lecseréli az ideiglenes jelszót és elfogadja a tájékoztatót, majd a böngészőállapotot az `e2e/.auth/{admin,developer,customer,customer2}.json` fájlba menti.

`e2e/support/fixtures.ts` (`test.extend`):

- Automatikus `consoleGuard`: a `pageerror`, a `securitypolicyviolation` és minden `console.error` elbuktatja a tesztet, **kivéve** a hálózati hibaüzeneteket: az a `console` üzenet, amelynek szövege `Failed to load resource` kezdetű, és a `location().url` a saját origin `/api/` útvonala, nem hiba (a Chromium minden nem 2xx `fetch`-et így jelent, a felület pedig a várt 401, 403, 409, 429 válaszokra épül). A váratlan HTTP-hibákat a `page.on('response')` figyeli: az `/api/` 5xx válasza mindig elbuktatja a tesztet, a 4xx csak akkor, ha a teszt nem jelölte várhatónak (`expectHttp(status, pathPattern)` engedélylista tesztenként; az anonim `GET /api/auth/me` 401 alapból engedélyezett).
- Állapotizolációs szabály: amely teszt egy felhasználó jelszavát, munkameneteit, 2FA-ját, nyelvét, témáját vagy zárolási állapotát módosítja, az `createUser(role)`-lal saját felhasználót hoz létre, és soha nem a seedelt `admin`, `dev1`, `cust1`, `cust2` felhasználót használja (érintett: `auth`, `twofa`, `i18n`, `theme`, `settings`, `privacy` fióktörlése, `admin-users` jelszó- és 2FA-visszaállítása). Egy automatikus `beforeEach` fixtúra a seedelt felhasználók megjelenítési preferenciáit az API-n át (`PATCH /api/me/preferences`) alapállapotba állítja (`hu`, `dark`, lista, minden mező látszik). A tájékoztató verzióját emelő tesztek (`gates.last.spec.ts`, `admin-settings.last.spec.ts`) a `chromium-last` projektben, minden más után futnak, és `afterAll`-ban minden seedelt felhasználóval újra nyugtázzák a tájékoztatót az API-n át.
- Szerepkör-fixtúrák (`asAdmin`, `asDeveloper`, `asCustomer`, `asCustomer2`); `adminApi` (`APIRequestContext` adminként); `createUser(role)` (egyedi név, API-n át jelszócsere és hozzájárulás, visszaadja a `storageState`-et); `uniqueUrl(testInfo)` (`http://127.0.0.1:18081/p/<testId>`). `e2e/support/totp.ts`: RFC 6238 SHA-1, 6 jegy, 30 s, `node:crypto` HMAC, base32-dekódolás; `freshCode(secret, usedStep?)` megvárja a következő lépést, ha a kód lépése már fel lett használva. `e2e/support/helpers.ts`: `addLink`, `openMenu`, `dropUrl` (szintetikus `DataTransfer`), `pasteText` (`ClipboardEvent`). A segédfüggvények a hozzáférhető nevet paraméterként kapják; a specifikációk a neveket a magyar katalógusból oldják fel az `e2e/support/i18n.ts` `msg(key, params)` függvényével (`msg('links.toolbar.add')`). Az `i18n.ts` a `src/lib/i18n/messages/hu/index.ts` alapértelmezett exportját dinamikus importtal tölti be (statikus import esetén az e2e `tsc` az alkalmazás forrásait is fordítaná), a helyőrzőket kitölti, a többes számot `Intl.PluralRules('hu')`-val választja, ismeretlen kulcsra kivételt dob. Így a `support/` alatt nincs ékezetes literál (a `langcheck` vizsgálja). A specifikációk (`e2e/specs/**`) ékezetes szöveget tartalmazhatnak, mint a Go-tesztadat (9.8, 10.5).

#### 9.4.2 Specifikációk

| Fájl | Forgatókönyvek |
|---|---|
| `smoke.spec.ts` | a bejelentkező oldal betölt, `<title>` tartalmazza a „Kinex Mark”-ot, `html.dark`, `lang="hu"`, nincs konzolhiba, `/api/auth/me` 401 |
| `auth.spec.ts` | belépés adminnal, kijelentkezés; hibás jelszó üzenete; védett útvonal (`/tags`) bejelentkezés nélkül `/login?next=%2Ftags`-re visz, belépés után vissza; `/login?next=/%5Cevil.example` belépés után `/`-re visz; 5 hibás próbálkozás után `too_many_attempts` és tiltott gomb (`createUser`-rel létrehozott felhasználón); lejárt munkamenet (süti törlése) → `reason=expired` üzenet; lejárt munkamenetben indított művelet (kitűzés) után csak a munkamenet üzenete, hibatoast nincs (E fázis) |
| `gates.last.spec.ts` (`chromium-last`) | új felhasználó generált jelszóval (admin UI-ból, a jelszót a dialógusból olvasva) → első belépés → kötelező jelszócsere → tájékoztató nyugtázása → Linkek oldal; tájékoztató-verzió emelése után egy másik felhasználó következő kérése a nyugtázási kapura visz, a kapu az új verziót mutatja, és egyetlen nyugtázás megnyitja az alkalmazást (E fázis); a „Megnyitás teljes oldalon” link; `afterAll`: a seedelt felhasználók újranyugtázása |
| `twofa.spec.ts` | `createUser`-rel: Beállítások › Biztonság: 2FA bekapcsolása (titok a kézi beírási mezőből, kód a `totp.ts`-ből), helyreállító kódok mentése; a kódlap nyomtatási nézete sötét témában is sötét szöveg világos alapon, legalább 7:1 (E fázis); kijelentkezés; belépés TOTP-vel (`freshCode`); belépés helyreállító kóddal; ugyanazzal a kóddal második próbálkozás elbukik; 2FA kikapcsolása |
| `i18n.spec.ts` | (a szerveroldali nyelvmentést `createUser`-rel) bejelentkező oldal magyarul (`html[lang=hu]`, „Bejelentkezés”), váltás angolra és németre újratöltés nélkül (cím, gomb, `lang`), a választás újratöltés után megmarad; belépés után a választott nyelv a szerverre mentődik (új kontextusban is az él); nyelvváltás a felhasználói felület felső sávjában és az admin felületen; német billentyűnevek (`Strg`, `Entf`) a Billentyűparancsok dialógusban (E fázis); dátum- és számformátum nyelvenként |
| `theme.spec.ts` | alapból `html.dark`; váltás után `html.light`, újratöltés után is; bejelentkezve (`createUser`) a preferencia mentődik, új kontextusban is világos |
| `links-capture.spec.ts` | felvétel dialógusból címkékkel; ráhúzás (`text/uri-list` + `text/html`, `dragenter`/`dragover`/`drop` a `body`-ra) → fátyol, mentés, toast; beillesztés URL-t tartalmazó szöveggel; URL nélküli szöveg → `no_url_found`; beviteli mezőben a beillesztés nem rögzít; duplikátum → figyelmeztetés és kiemelés |
| `links-view.spec.ts` | lista/csempe váltás és megmaradása; az Ikon, URL, Cím, Leírás kapcsolók hatása a címkére és a másodlagos sorokra, újratöltés után is; címke-elsőbbség (leírás > cím > URL); hosszú URL helyett a domain |
| `links-filter.spec.ts` | 6 link 3 címkével; keresés cím, leírás, URL és címkenév szerint; címkeszűrő `any`/`all`; „Címke nélkül”; hatókör; az URL-paraméterek újratöltés után is érvényesek; „Szűrők törlése”; csoportosítás címkével: az oldalsáv „Gyakori címkék” eleme és a `/tags` linkszám-hivatkozása pontosan a címke linkjeit mutatja, a két címkés link mindkét csoportban szerepel; a fejléc alcíme a szűrt `total`-t és `pinned`-et mutatja, az oldalsáv számlálója a szűrőtől független |
| `links-ranking.spec.ts` | A, B, C felvétele; C kétszer, B egyszer megnyitva (`context.waitForEvent('page')`, majd a lap bezárása) → C, B, A; A és C kitűzése, C további megnyitása → a „Kitűzött” csoportban C, A, utána B; B további háromszori megnyitása → C, A, B; egy billentyűs megnyitás (fókusz a horgonyon `j`-vel, majd Enter) a részletek lap használatszámát pontosan 1-gyel növeli; újratöltés után ugyanez |
| `links-archive.spec.ts` | kézi archiválás → a link eltűnik, „Archiváltak is” mellett „Archivált” jelöléssel látszik, az alcím `{archived}` száma nő; megnyitás visszaállítja; „Visszaállítás” menüpont; archivált link kitűzése megszünteti az archiválást; admin „Archiválás futtatása most” → „0 link archiválva”; a részletek lapon az `archiveDueAt` dátuma |
| `links-copy.spec.ts` | a vágólap tartalma (`clipboard-read` és `clipboard-write` engedéllyel) `http://127.0.0.1:18081/alpha`, illetve `[Alpha docs](http://127.0.0.1:18081/alpha)`; a másolás használatnak számít: mindkét másolás után a részletek lap használatszáma 1-gyel nő, és a másolt link a rangsorban egy soha nem használt elé kerül |
| `links-detail.spec.ts` | a fixture `/alpha` linkjének címe és ikonja 20 s-on belül megjelenik (`img` a `/api/links/{id}/icon` forrással, `naturalWidth > 0`); `/beta` címe helyes ékezetekkel; `/plain` → „Nem weboldal (nincs cím)”; részletek lap minden blokkja; szerkesztés (leírás) → a címke a leírásra vált; az URL szerkesztése `/alpha`-ról `uniqueUrl`-re (`/p/<id>`) → a cím „Fixture <id>”-re vált, az ikon az újé; kézzel megadott cím a „Cím és ikon frissítése” után megmarad; „Cím és ikon frissítése”; törlés megerősítéssel |
| `links-import-export.spec.ts` | export JSON és HTML letöltése (fájlnév, `format`); a JSON visszaimportálása egy másik felhasználónál → összesítő (Új, Már megvolt); hibás fájl → `import_invalid` |
| `sharing.spec.ts` | fejlesztő megoszt egy linket `cust1`-gyel: az ügyfél látja „Megosztotta” jelöléssel, kitűzi és saját címkével címkézi; a fejlesztőnél a kitűzés és a címke nem látszik; az ügyfélnél nincs Szerkesztés, Megosztás, Törlés; szerepkör-célpont („Ügyfelek”) `cust2`-nek is megjelenik; visszavonás után eltűnik |
| `tags.spec.ts` | címke felvétele mind a hét színnel, átnevezés, `tag_duplicate`, törlés (a linkek megmaradnak), linkszám-hivatkozás a szűrt listára |
| `settings.spec.ts` | `createUser`-rel: profil (megjelenített név), megjelenítési preferenciák (nyelv, téma, alapnézet, kapcsolók, archiválási idő „Egyéni…” értékkel), munkamenetlista és visszavonás (második kontextus kijelentkeztetve), jelszócsere |
| `privacy.spec.ts` | `/privacy` bejelentkezés nélkül, az adatkezelő adatai a beállításokból; adatok letöltése: `format` `kinex-mark-export/1`, saját linkek és címkék, nincs `passwordHash`, `totp`, `token` kulcs; a tájékoztató szakaszai (jogalap, címzettek, továbbítás, automatizált döntés, rendszergazdai hozzáférés, jogok, felügyeleti hatóság); fióktörlés `createUser`-rel létrehozott fejlesztővel (hibás felhasználónévre `confirmation_mismatch`; a dialógus kiírja, hány megosztott link tűnik el) → kijelentkeztetve, a belépés `invalid_credentials`; az audit naplóban `user.self_deleted` álnévvel; a törölt fejlesztő megosztott linkje a címzettnél eltűnik |
| `admin-users.spec.ts` | `createUser`-rel létrehozott célfelhasználókon: létrehozás, szerkesztés (szerepkörváltás megerősítő jelszóval), jelszó-visszaállítás megerősítő jelszóval (hibás jelszóra mezőhiba; utána a régi munkamenet megszűnik), inaktiválás (`account_disabled`), aktiválás, 2FA-visszaállítás (a cél munkamenetei megszűnnek), feloldás, törlés; a saját sor műveletei tiltva; `last_admin` |
| `admin-settings.last.spec.ts` (`chromium-last`) | határérték-hiba és sikeres mentés; `allowPrivateFetch` zárolva (`lockedByEnv`); adatkezelő adatai és a felügyeleti hatóság megjelennek a `/privacy`-n; tájékoztató-verzió emelése; `afterAll`: a seedelt felhasználók újranyugtázása |
| `admin-audit.spec.ts` | szűrés eseménycsoportra (`auth.*`), eredményre, szereplőre, időszakra; lapozás; lokalizált eseménynév |
| `security.spec.ts` | minden oldalválaszon CSP `'unsafe-inline'` nélkül a `script-src`-ben, és nincs CSP-sértés; kijelentkezve az API 401; ügyfél: nincs Adminisztráció menücsoport, `/admin/users` 403-as hibaoldal, `GET /api/users/directory` 403; fejlesztő: nincs admin menü, van megosztás; IDOR a böngészőből (más felhasználó linkje → 404); minden kérés a saját originre megy (`page.on('request')` csak `127.0.0.1:18080`) |
| `a11y.spec.ts` | `@axe-core/playwright` a `/login`, `/`, `/tags`, mind a négy beállítás-fül, `/admin/users`, `/admin/settings`, `/admin/audit`, `/privacy` oldalon sötét és világos témában: 0 `serious` és `critical` találat; billentyűzettel bejárható bejelentkezés és link felvétele; `/`, `n`, `j`/`k`, `p` parancsok (az `Esc` asztali szélességen is törli a keresést és a `?q`-t; a `j`/`k` akkor is működik, ha üres oldalterületre kattintva a `main` kapta a fókuszt; E fázis); a hero-kép betölt |
| `responsive.spec.ts` (`mobile` projekt) | offcanvas oldalsáv nyitása és zárása; nincs vízszintes oldalgörgetés (`scrollWidth <= innerWidth`) a fő oldalakon; a sorműveletek menüből elérhetők; 360 px széles képernyőn a hosszú, törhetetlen címkéjű link Címkék dialógusa a képernyőn belül marad, a gombjai láthatók (E fázis) |

Az idő alapú automatikus archiválást a Go store- és jobs-teszt bizonyítja hamis órával; éles kódba tesztkampó nem kerül, ezért az E2E csak a kézi archiválást, az archivált nézetet, a visszaállítást és az `archiveDueAt` megjelenítését fedi.

Futtatás: helyben `.\build.ps1 test-e2e` (előtte egyszer `.\build.ps1 e2e-install`), Linuxon `make test-e2e`; a CI `e2e` jobja a friss binárissal (10.10). Hibánál a trace és a HTML-riport artefaktumként feltöltődik. Kész-feltétel: három egymás utáni futás flaky nélkül, helyben és `CI=true` mellett is.

### 9.5 Feltáró GUI-teszt a Playwright MCP-vel

- Eszközök: a Playwright plugin MCP-eszközei (`browser_navigate`, `browser_snapshot`, `browser_click`, `browser_type`, `browser_fill_form`, `browser_drop`, `browser_press_key`, `browser_resize`, `browser_emulate_media`, `browser_take_screenshot`, `browser_console_messages`, `browser_close`). A böngésző egypéldányos: egyszerre egy GUI-ügynök fut, a végén `browser_close`.
- Környezet: a bináris másolata külön könyvtárban (a `dist\kinex-mark.exe` így nem zárolódik), `127.0.0.1:8080`, friss adatkönyvtár, ismert admin-jelszó, majd `npm.cmd run seed:demo -- --base http://127.0.0.1:8080 --password <admin>` (11.8). A `127.0.0.1:18080` és `18081` az E2E-futtatókereté (3.4): GUI-ügynök nem köti le, mert a futtatókeret idegen szervert nem használ újra (`reuseExistingServer: false`), így a párhuzamos E1-futás elbukik (E fázis). Indítás előtt a GUI-ügynök ellenőrzi, hogy a port szabad (`Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue` üres). Kivétel csak az, ha az ütemezés kizárja a párhuzamos E2E-futást (például az E-close újraellenőrzése, amely maga futtatja az E2E-t is, de nem egyszerre).
- Forgatókönyvek: minden útvonal mindhárom szerepkörrel, mindhárom nyelven, sötét és világos témában, 1440×900, 1024×768 és 390×844 méretben; billentyűzetes bejárás (fókuszgyűrű látható); `prefers-reduced-motion`; üres állapotok; hibaállapotok (hálózati hiba, 429, duplikáció); hosszú URL, hosszú cím, hosszú német szavak tördelése.
- Ellenőrzőlista: nincs csonkolt fordítás; nincs vízszintes oldalgörgetés; a fókuszgyűrű mindenhol látszik; egy primary gomb képernyőnként; nincs angol szöveg magyar felületen; a toastok igéje egyezik a gombéval; a konzol tiszta; a GRAPHICS.md tokenjei, betűi, chamferje és mozgása; a hero a márkafelülettel egybeolvad.
- Kimenet: `findings.md` a munkakönyvtárban (11.1) tételenként súlyossággal, lépésekkel, képernyőképpel; a javítást a frontend-javító végzi, utána újraellenőrzés.
- Takarítás: az MCP a pillanatképeit, letöltéseit és képernyőképeit az MCP-szerver munkakönyvtára (rendszerint a repo gyökere) alatti `.playwright-mcp/` mappába írja (a hely a repón kívüli bővítménykonfigurációból jön, a repóból nem irányítható át). A GUI-ügynök a munkája végén ezt a mappát a munkakönyvtárba mozgatja (például `$env:TEMP\kinex-mark-work\gui-<id>\playwright-mcp`) és a repóból törli; a `.gitignore`, a `.dockerignore` és a `web/.prettierignore` bejegyzése csak a véletlen commitot, a Docker-kontextusba kerülést és a `npm run lint` hibáját előzi meg (10.1, 8.1.2). A Chromium MCP-ben a `navigator.clipboard.readText()` engedéllyel is üres szöveget adhat; a másolás ellenőrzése ezért a felület által írt szövegre (vagy az E1 saját böngészőjének vágólap-engedélyére) épül.

### 9.6 Szerződés-golden a backend és a frontend között

`internal/api/contract_test.go` (`TestContractGolden`) a Go-konstansokból írja (`go test ./internal/api -run TestContractGolden -update`) és ellenőrzi az `internal/api/testdata/contract.json` fájlt:

```json
{
  "errorCodes": ["endpoint_not_found", "invalid_json", "..."],
  "fieldCodes": ["required", "too_short", "..."],
  "auditEvents": ["bootstrap.admin_created", "auth.login.success", "..."],
  "auditDetailKeys": ["reason", "method", "scope", "changed", "role", "active", "..."],
  "auditDetailValues": { "reason": ["credentials", "unknown_user", "disabled", "totp"], "method": ["password", "totp", "recovery_code"], "...": [] },
  "fetchErrors": ["timeout", "dns", "..."],
  "tagColors": ["blue", "ochre", "green", "violet", "rose", "neutral", "contrast"],
  "roles": ["admin", "developer", "customer"],
  "locales": ["hu", "en", "de"]
}
```

A `web/src/lib/i18n/contract.spec.ts` (munkakönyvtár: `web`) beolvassa, és mindhárom nyelven ellenőrzi a kulcsok létezését:

| Golden-lista | Kötelező kulcs |
|---|---|
| `errorCodes` (+ a csak frontend `network_error`) | `errors.<code>.title`, `errors.<code>.text` |
| `fieldCodes` | `fields.<code>` |
| `auditEvents` | `audit.events.<event>` |
| `auditDetailKeys` | `audit.details.keys.<key>` |
| `auditDetailValues` | `audit.details.values.<key>.<value>` |
| `fetchErrors` | `fetch.errors.<code>` |
| (frontend-lista: a 8.7.10 tizenkét szakasza, a hét jog) | `privacy.sections.<s>.title`, `.body`; `privacy.rights.<r>` |
| `tagColors` | `tagColors.<color>` |
| `roles` | `roles.<role>` |
| `locales` | `nav.locale.<locale>` |

Ezen felül a `web/src/lib/api/errors.ts` `ErrorCode` uniója és a golden `errorCodes` listája egyezik (a teszt a forrásfájlból olvassa a literálokat). Új backend-kód így a frontend fordítása nélkül piros CI-t ad.

### 9.7 Tesztadatok

| Szint | Forrás |
|---|---|
| Go | `storetest.NewStore(t)`, `storetest.SeedUser(t, st, role)` (az `internal/store`-on kívüli tesztekhez; a `store` saját tesztjei a csomagon belüli `newTestStore`/`seedUser` segédet használják, így nincs `store → storetest → store` importkör), `clock.Fake` `testutil.T0`, `auth.NewHasher(1000, 4)`, `testutil.NewOrigin`, `testutil.AllowOnly` |
| Vitest | `src/lib/test/fixtures.ts` típusos gyártói, `mockApi` |
| E2E | friss ideiglenes adatkönyvtár futásonként; felhasználók a global setupban; linkek tesztenként egyedi URL-lel (`uniqueUrl`); metaadat a helyi fixture-forrásról |
| Kézikönyv és GUI | `npm run seed:demo -- --base <url> --password <admin>` (`e2e/support/seed-demo.mjs`): az `e2e/fixtures/demo-data.json` betöltése API-n át: 3 felhasználó (admin, fejlesztő, ügyfél kitalált nevekkel), 30 link valódi nyilvános dokumentációs URL-ekre (hogy legyen ikon), az admin 8 címkéje mind a hét színnel (a fejlesztőnek és az ügyfélnek 3, illetve 2 saját címkéje is van, hogy az ő linkjeik is címkézettek legyenek), magyar leírások, megosztások (mindenkinek, szerepkörnek, felhasználónak), kitűzések (4), látogatások, 3 archivált link. A szkript függőség nélküli, saját sütikezelésű és újrafuttatható: a meglévő felhasználókat, címkéket és linkeket újrahasznosítja, az állapotot csak az adott futásban létrehozott linkekre alkalmazza (kapcsolók: `--admin-user`, `--data`). Valódi személyes adat nincs benne |

### 9.8 Konvenciótesztek (`web/src/lib/test/conventions.spec.ts`)

Forrásbejárás a `src/` alatt (kivéve `components/ui/**`, `app.css`, `i18n/messages/**`, ahol nincs másképp jelölve):

1. Nincs nyers hex szín (`#[0-9a-fA-F]{3,8}\b`) a `.svelte` és `.ts` fájlokban; kivétel nincs (a QR-tároló tokennel színez).
2. Minden Lucide-ikonhasználaton `strokeWidth={1.5}`.
3. A fordításokban (mindhárom nyelv) nincs `!` és nincs `\p{Extended_Pictographic}`.
4. Nincs színátmenet (`gradient` osztály vagy CSS).
5. Fordítás nélküli felületi szöveg nincs (kivételek mindenhol: `Kinex Mark`, `Kinexium`, `Markdown`, `URL`, `JSON`, `HTML`, `2FA`):
   - a `.svelte` sablonokban nincs betűs szövegcsomópont (a Svelte-fordító `parse` AST-jét bejárva);
   - ugyanebben az AST-ben az `aria-label`, `aria-description`, `title`, `placeholder` és `alt` attribútum statikus, betűt tartalmazó értéke hiba (a `{t('…')}` kifejezés megengedett);
   - a `.ts` fájlokban és a `.svelte` `<script>` blokkjaiban a `toast(…)`, `toast.success|error|warning|info|message(…)` hívások első argumentuma, illetve a `description` mezője nem lehet betűs sztringliterál (TypeScript-AST, `typescript` `createSourceFile`).
6. Oldalanként (route-fájl és közvetlen feature-komponensei) legfeljebb egy `variant="default"` vagy variáns nélküli `Button`; a szándékos kivétel (külön dialógus) `// primary-exception: <reason>` kommenttel jelölt.
7. A `messages/**`-on kívül nincs `[áéíóöőúüűÁÉÍÓÖŐÚÜŰßäÄ]` karakter (a `langcheck` párja a webes forrásban). Kivétel, a `langcheck`-kel és a 9.2.1 Go-szabályával egyezően: a `**/*.spec.ts`, `**/*.test.ts` és a `src/lib/test/fixtures.ts` ékezetes tesztadatot tartalmazhat (például a 8.7.3 példatáblája, `[Éles műszerfal](…)`); a tesztek azonosítói, kommentjei és tesztnevei ott is angolok (R4). A komponenstesztek a felületi szöveget lehetőleg `t('…')`-vel keresik, nem literállal.
8. A `dark:` Tailwind-variáns nem szerepel a saját kódban.
9. Nincs süti a munkamenet-sütin kívül: a saját kód nem ír `document.cookie`-t, és nem használja a generált `Sidebar.Provider`-t (helyette `SidebarFrame`, 8.1.5; 6.14 „Sütik”). (D fázis, R3)
10. Minden `ToggleGroup`-elem jelöli a kiválasztott állapotát (`toggleItem`, 8.1.5). (D fázis, R5)
11. A generált komponenseken a típusskála utility-listaként jelenik meg (`typeHeading3`, `typeBodySm`, `typeEyebrow`), nem a komponensrétegbeli `heading-3`, `body-sm`, `kx-eyebrow` osztállyal, mert azt a nova utility-jei felülírják. (D fázis, R5)
12. A dialógus-, alert-dialógus- és lapfátyol sima színréteg, `backdrop-blur` nélkül (GRAPHICS.md 10.8; a generált overlay-fájlokban is, 8.1.5). (E fázis)
13. Minden `Dialog.Content` és `AlertDialog.Content` a `dialogBox`-ot használja (8.1.5). (E fázis)
14. Egyválasztós `ToggleGroup` őrzött `bind:value`-val, `onValueChange` nélkül, így nem üríthető ki (8.7.3). (E fázis)
15. A `utils/toast-error.ts`-en kívül nincs `toast.error(errorMessage(err|toApiError…))` vagy `toast.error(captureError(…))`: a kérés hibájának toastja a `toastError`, amely a munkamenet-tároló által kezelt hibát kihagyja (8.1.5). A kliensoldali ellenőrzések (például „nincs URL”) nem kéréshibák, közvetlenek maradnak. (E fázis)
16. Minden `<Checkbox>` a `checkboxBox` osztályt kapja (8.1.5). (E fázis)
17. Minden `autocomplete="current-password"` mezős űrlap tartalmaz `<AccountUsername>`-t; kivétel a saját felhasználónév-mezős `LoginForm` és a másik fiókról szóló `UserDialog` (8.1.5). (E fázis)

---

## 10. Build, Docker, CI/CD

Közös szabályok: minden szkript, konfiguráció, komment, naplóüzenet és hibaüzenet angol; PowerShell-parancsokban `npm.cmd` és `npx.cmd`, mert a végrehajtási házirend a `npm.ps1` shimet tilthatja (TECHNOLOGY.md 7.3); Node mindenhol 26; a `go.sum` és a `web/package-lock.json` verziókövetett, a munkafában léteznie kell.

### 10.1 Verziókövetési fájlok és a buildút sorrendje

`.gitignore` (gyökér):

```
/dist/
/data/
/coverage.out
/coverage.html
web/node_modules/
web/build/
web/.svelte-kit/
web/coverage/
web/test-results/
web/playwright-report/
web/blob-report/
web/e2e/.auth/
graphics/__pycache__/
# Playwright MCP output (GUI test snapshots, downloads, screenshots; may hold test secrets)
.playwright-mcp/
.env
.env.*
```

`.gitattributes`:

```
* text=auto eol=lf
*.ps1 text eol=crlf
*.png binary
*.webp binary
*.ico binary
*.woff2 binary
*.db binary
```

A `build.ps1` CRLF-fel és UTF-8 BOM-mal mentett (TECHNOLOGY.md 7.3, 7.7); a BOM csak ASCII tartalom mellett is marad.

`.dockerignore`:

```
.git
.github
dist
data
ai
manual
graphics
tools
coverage.out
coverage.html
web/node_modules
web/build
web/.svelte-kit
web/coverage
web/test-results
web/playwright-report
web/blob-report
web/e2e
**/.playwright-mcp
*.md
```

A `.playwright-mcp` bejegyzések (a C fázis infra-javítójának pótlása) a Playwright MCP kimeneti mappáját tartják távol a committól és a Docker-build kontextusától. Horgonyzatlanok, mert az MCP-szerver gyökere a repo gyökere vagy a `web/` is lehet; a `tools/langcheck/repoignore_test.go` regressziós tesztje mindkét fájlt ellenőrzi. A mappát magát az MCP-bővítmény konfigurációja (a repón kívül) helyezi el, ezért a GUI-ügynök a munkája végén a `.playwright-mcp/` tartalmát a munkakönyvtárba (`$env:TEMP\kinex-mark-work\`) mozgatja, és a mappát törli (9.5).

Sorrend: a `web/build` nem kap helyőrzőt (az adapter-static build előtt törli a mappát, egy `.gitkeep` minden build után „törölt” lenne, és a `git describe --dirty` hamisan piszkos verziót adna). Ezért minden buildút előbb a felületet építi: Makefile `$(WEB_INDEX)`, `build.ps1` `Assert-WebBuild`, CI `web-build` artefakt.

A verzió a `git describe --tags --always --dirty` kimenete (hiányában `dev`, felülírható: `VERSION=` / `-Version`); `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w -X main.version=<verzió>"`.

### 10.2 Makefile (teljes)

```make
APP         := kinex-mark
CMD         := ./cmd/kinex-mark
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X main.version=$(VERSION)
GOBUILD     := CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)"
TARGETS     := linux/amd64 linux/arm64 windows/amd64 darwin/arm64
RACE        ?= -race
COVER_TOTAL ?= 85
COVER_PKG   ?= 80
NODE_DEPS   := web/node_modules/.package-lock.json
WEB_INDEX   := web/build/index.html
GOFILES     := cmd internal tools $(wildcard web/*.go)

.PHONY: all tidy web build run cross check test test-go test-web test-e2e e2e-install verify graphics image binary-image clean

all: build

tidy:
	go mod tidy

web:
	cd web && npm ci && npm run build

# Installs the web dependencies only when the lockfile changed.
$(NODE_DEPS): web/package-lock.json
	cd web && npm ci

# go:embed needs web/build; build the UI once if it is missing.
$(WEB_INDEX):
	$(MAKE) web

build: web
	$(GOBUILD) -o dist/$(APP) $(CMD)

run: build
	./dist/$(APP)

cross: web
	@for target in $(TARGETS); do \
		os=$${target%/*}; arch=$${target#*/}; ext=""; \
		[ "$$os" = "windows" ] && ext=".exe"; \
		echo "$$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD) -o dist/$(APP)-$$os-$$arch$$ext $(CMD) || exit 1; \
	done

check: $(NODE_DEPS) $(WEB_INDEX)
	cd web && npm run check && npm run lint
	@unformatted="$$(gofmt -l $(GOFILES))"; \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	go run ./tools/langcheck

test: test-go test-web

test-go: $(WEB_INDEX)
	go test $(RACE) -covermode=atomic -coverprofile=coverage.out ./...
	go run ./tools/covercheck -profile coverage.out -total $(COVER_TOTAL) -package $(COVER_PKG)

test-web: $(NODE_DEPS)
	cd web && npm run test:coverage

e2e-install: $(NODE_DEPS)
	cd web && npx playwright install chromium

test-e2e: build
	cd web && KINEX_MARK_BIN="$(CURDIR)/dist/$(APP)" npm run test:e2e

verify: check test test-e2e

# Generates only the images that are still missing (a regeneration is a paid call: use --force by hand).
graphics:
	python3 graphics/generate_graphics.py

image:
	docker build --target runtime --build-arg VERSION=$(VERSION) -t $(APP):$(VERSION) .

# Builds the Linux binary with Docker only, without a local Go or Node.
binary-image:
	docker build --target binary --build-arg VERSION=$(VERSION) --output type=local,dest=dist .

clean:
	rm -rf dist web/build web/.svelte-kit web/coverage web/test-results web/playwright-report web/blob-report coverage.out coverage.html
```

`RACE` felülírható (`make test-go RACE=`). A Makefile-nak nincs `dev` célja (két terminál, TECHNOLOGY.md 7.4).

### 10.3 `build.ps1` (teljes; UTF-8 BOM, CRLF)

```powershell
#Requires -Version 5.1
<#
.SYNOPSIS
    PowerShell counterpart of the Makefile: build, test, development and Docker on Windows.
.EXAMPLE
    .\build.ps1
    .\build.ps1 dev
    .\build.ps1 verify
    .\build.ps1 cross -Version 1.0.0
#>
[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('build', 'web', 'run', 'dev', 'tidy', 'cross', 'check', 'test', 'test-go', 'test-web',
        'test-e2e', 'e2e-install', 'verify', 'graphics', 'image', 'binary-image', 'clean')]
    [string] $Task = 'build',

    [string] $Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$App = 'kinex-mark'
$Cmd = './cmd/kinex-mark'
$Targets = 'linux/amd64', 'linux/arm64', 'windows/amd64', 'darwin/arm64'
$DevAddr = '127.0.0.1:8080'
$CoverTotal = '85'
$CoverPackage = '80'
$OnWindows = $env:OS -eq 'Windows_NT'
$Exe = if ($OnWindows) { '.exe' } else { '' }
# npm.cmd and npx.cmd start regardless of the execution policy.
$Npm = if ($OnWindows) { 'npm.cmd' } else { 'npm' }
$Npx = if ($OnWindows) { 'npx.cmd' } else { 'npx' }
$Python = if ($OnWindows) { 'python' } else { 'python3' }

Set-Location -LiteralPath $PSScriptRoot

function Invoke-Native {
    $file = $args[0]
    # An array is required: a single string would be split into characters when splatted.
    [string[]] $rest = @($args | Select-Object -Skip 1)
    & $file @rest
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed ($LASTEXITCODE): $file $rest"
    }
}

function Get-AppVersion {
    if ($Version) { return $Version }
    if (Get-Command git -ErrorAction SilentlyContinue) {
        try {
            $previous = $ErrorActionPreference
            $ErrorActionPreference = 'Continue'
            $described = & git describe --tags --always --dirty 2>$null
            if ($LASTEXITCODE -eq 0 -and $described) { return "$described" }
        }
        catch { }
        finally { $ErrorActionPreference = $previous }
    }
    return 'dev'
}

function Invoke-WebBuild {
    Push-Location web
    try {
        Invoke-Native $Npm ci
        Invoke-Native $Npm run build
    }
    finally { Pop-Location }
}

function Assert-WebBuild {
    if (-not (Test-Path -LiteralPath 'web/build/index.html')) { Invoke-WebBuild }
}

function Assert-WebDeps {
    if (-not (Test-Path -LiteralPath 'web/node_modules')) {
        Push-Location web
        try { Invoke-Native $Npm ci }
        finally { Pop-Location }
    }
}

function Invoke-WebScript {
    param([Parameter(Mandatory)] [string] $Name)
    Assert-WebDeps
    Push-Location web
    try { Invoke-Native $Npm run $Name }
    finally { Pop-Location }
}

# Environment variables would survive for the whole session, so they are restored.
function Invoke-GoBuild {
    param(
        [Parameter(Mandatory)] [string] $Output,
        [string] $TargetOS,
        [string] $TargetArch
    )
    $saved = @{ CGO_ENABLED = $env:CGO_ENABLED; GOOS = $env:GOOS; GOARCH = $env:GOARCH }
    try {
        $env:CGO_ENABLED = '0'
        if ($TargetOS) { $env:GOOS = $TargetOS }
        if ($TargetArch) { $env:GOARCH = $TargetArch }
        Invoke-Native go build -trimpath -ldflags "-s -w -X main.version=$(Get-AppVersion)" -o $Output $Cmd
    }
    finally {
        $env:CGO_ENABLED = $saved.CGO_ENABLED
        $env:GOOS = $saved.GOOS
        $env:GOARCH = $saved.GOARCH
    }
}

function Invoke-Check {
    Assert-WebBuild
    Invoke-WebScript -Name 'check'
    Invoke-WebScript -Name 'lint'
    [string[]] $goFiles = @('cmd', 'internal', 'tools') + @(Get-ChildItem -LiteralPath web -Filter '*.go' | ForEach-Object { $_.FullName })
    $unformatted = & gofmt -l @goFiles
    if ($LASTEXITCODE -ne 0) { throw "Command failed ($LASTEXITCODE): gofmt -l" }
    if ($unformatted) { throw "gofmt needed: $($unformatted -join ', ')" }
    Invoke-Native go vet ./...
    Invoke-Native go run ./tools/langcheck
}

function Invoke-GoTest {
    Assert-WebBuild
    # The race detector needs cgo and a C compiler; it runs in CI on Linux, not here.
    # Quoted: Windows PowerShell 5.1 splits an unquoted -flag=a.b argument at the dot.
    Invoke-Native go test '-covermode=atomic' '-coverprofile=coverage.out' ./...
    Invoke-Native go run ./tools/covercheck -profile coverage.out -total $CoverTotal -package $CoverPackage
}

function Invoke-E2E {
    $binary = (Resolve-Path -LiteralPath "dist/$App$Exe").Path
    $saved = $env:KINEX_MARK_BIN
    try {
        $env:KINEX_MARK_BIN = $binary
        Invoke-WebScript -Name 'test:e2e'
    }
    finally { $env:KINEX_MARK_BIN = $saved }
}

switch ($Task) {
    'tidy' {
        Invoke-Native go mod tidy
    }
    'web' {
        Invoke-WebBuild
    }
    'build' {
        Invoke-WebBuild
        Invoke-GoBuild -Output "dist/$App$Exe"
    }
    'run' {
        Invoke-WebBuild
        Invoke-GoBuild -Output "dist/$App$Exe"
        Invoke-Native "./dist/$App$Exe"
    }
    'dev' {
        Assert-WebBuild
        Assert-WebDeps
        # A fixed binary path: the firewall does not ask again, and only one process has to be stopped.
        $server = "dist/$App-dev$Exe"
        Invoke-GoBuild -Output $server
        $process = Start-Process -FilePath (Resolve-Path -LiteralPath $server).Path `
            -ArgumentList '-addr', $DevAddr -NoNewWindow -PassThru
        Push-Location web
        try {
            Invoke-Native $Npm run dev
        }
        finally {
            Pop-Location
            Stop-Process -Id $process.Id -ErrorAction SilentlyContinue
        }
    }
    'cross' {
        Invoke-WebBuild
        foreach ($target in $Targets) {
            $os, $arch = $target -split '/'
            $extension = if ($os -eq 'windows') { '.exe' } else { '' }
            Write-Host "$os/$arch"
            Invoke-GoBuild -Output "dist/$App-$os-$arch$extension" -TargetOS $os -TargetArch $arch
        }
    }
    'check' {
        Invoke-Check
    }
    'test' {
        Invoke-GoTest
        Invoke-WebScript -Name 'test:coverage'
    }
    'test-go' {
        Invoke-GoTest
    }
    'test-web' {
        Invoke-WebScript -Name 'test:coverage'
    }
    'e2e-install' {
        Assert-WebDeps
        Push-Location web
        try { Invoke-Native $Npx playwright install chromium }
        finally { Pop-Location }
    }
    'test-e2e' {
        Invoke-WebBuild
        Invoke-GoBuild -Output "dist/$App$Exe"
        Invoke-E2E
    }
    'verify' {
        Invoke-WebBuild
        Invoke-Check
        Invoke-GoTest
        Invoke-WebScript -Name 'test:coverage'
        Invoke-GoBuild -Output "dist/$App$Exe"
        Invoke-E2E
    }
    'graphics' {
        # Plain "python": the user site that holds google-genai is ignored under -I.
        Invoke-Native $Python graphics/generate_graphics.py
    }
    'image' {
        $tag = Get-AppVersion
        Invoke-Native docker build --target runtime --build-arg "VERSION=$tag" -t "${App}:$tag" .
    }
    'binary-image' {
        Invoke-Native docker build --target binary --build-arg "VERSION=$(Get-AppVersion)" --output type=local,dest=dist .
    }
    'clean' {
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue dist, web/build, web/.svelte-kit, web/coverage,
            web/test-results, web/playwright-report, web/blob-report, coverage.out, coverage.html
    }
}
```

| Makefile | PowerShell | Eredmény |
|---|---|---|
| `make tidy` | `.\build.ps1 tidy` | `go.mod`, `go.sum` |
| `make web` | `.\build.ps1 web` | `web/build` |
| `make build` | `.\build.ps1` | `dist\kinex-mark.exe` |
| `make run` | `.\build.ps1 run` | fordít és indít (`:8080`, `./data`) |
| nincs | `.\build.ps1 dev` | `dist\kinex-mark-dev.exe` `127.0.0.1:8080`-on + Vite `5173` |
| `make cross` | `.\build.ps1 cross -Version 1.0.0` | `dist/kinex-mark-{linux-amd64,linux-arm64,windows-amd64.exe,darwin-arm64}` |
| `make check` | `.\build.ps1 check` | svelte-check (0 figyelmeztetés), prettier, gofmt, go vet, langcheck |
| `make test` | `.\build.ps1 test` | Go- és webtesztek lefedettséggel és kapukkal (Windowson `-race` nélkül) |
| `make test-go` / `test-web` | `.\build.ps1 test-go` / `test-web` | |
| `make e2e-install` | `.\build.ps1 e2e-install` | Chromium a Playwrighthoz |
| `make test-e2e` | `.\build.ps1 test-e2e` | E2E a friss binárissal |
| `make verify` | `.\build.ps1 verify` | a teljes helyi kapu |
| `make graphics` | `.\build.ps1 graphics` | csak a hiányzó képek generálása |
| `make image` | `.\build.ps1 image` | `kinex-mark:<verzió>` |
| `make binary-image` | `.\build.ps1 binary-image` | Linuxos `dist/kinex-mark` Dockerrel |
| `make clean` | `.\build.ps1 clean` | a generált kimenetek törlése, a `data` marad |

### 10.4 Webes szkriptek és formázás

A `web/package.json` szkriptjei, a `.prettierrc` és a `.prettierignore`: 8.1.2. A `.vscode/extensions.json` a `svelte.svelte-vscode` bővítményt ajánlja.

### 10.5 Kiegészítő Go-eszközök

`tools/covercheck/main.go` (csak standard könyvtár):

- Kapcsolók: `-profile` (alap `coverage.out`), `-total` (alap 85), `-package` (alap 80), `-prefix` (alap `github.com/kaszasg/kinex-mark/internal/`), `-exclude` (vesszővel tagolt csomagutak, alap `github.com/kaszasg/kinex-mark/internal/testutil,github.com/kaszasg/kinex-mark/internal/storetest`), `-markdown` (Markdown-tábla a stdout-ra).
- A profil blokkjait (`file:line.col,line.col statements count`) kulcs szerint egyesíti (a nagyobb darabszám nyer), csomagonként (`path.Dir`) és összesítve számol utasításlefedettséget, csak a `-prefix` alatti, nem kizárt csomagokra.
- Kimenet: csomagonként egy sor, a végén `total: 87.3% (threshold 85%)`. Ha az összesítő vagy bármely csomag a küszöb alatt van, 1-es kóddal kilép, és megnevezi a hibás csomagokat.

`tools/langcheck/main.go` (csak standard könyvtár):

- Bejárja: `cmd`, `internal`, `tools`, `web/src`, `web/e2e/support`, `web/scripts`, `web/*.ts`, `web/*.go`, `graphics/*.py`, `Makefile`, `build.ps1`, `Dockerfile`, `compose.yaml`, `.github`.
- Kiterjesztések: `.go`, `.ts`, `.svelte`, `.js`, `.mjs`, `.css`, `.html`, `.yml`, `.yaml`, `.ps1`, `.sql`, `.py`, valamint a `Makefile` és a `Dockerfile`.
- Kihagyja: `web/src/lib/i18n/messages/**`, `**/testdata/**`, `web/e2e/fixtures/**`, `*_test.go`, `*.spec.ts`, `*.test.ts`, `web/src/lib/test/fixtures.ts`, `node_modules`, `build`, `.svelte-kit`. Az `src/app.css` nincs kihagyva (a szinkron szkript eltávolítja a magyar kommenteket, 8.1.6).
- Hiba: bármely `áéíóöőúüűÁÉÍÓÖŐÚÜŰäöüßÄÖÜ` rúna; kimenet `<file>:<line>: non-English character "ő"`, kilépési kód 1. A tesztfájlok ékezetes tesztadata megengedett; a tesztkommentek nyelvét az R4 felülvizsgálat nézi.

### 10.6 Fontok reprodukálása

A két WOFF2 az A2 első lépésében kerül a `web/static/fonts/` alá (T12 méret és hash; a `web/assets_test.go` ellenőrzi). A `tools/fonts/build_fonts.py` buildben nem fut, a reprodukálhatóságot dokumentálja:

| Paraméter | Érték |
|---|---|
| Források | `google/fonts` repó: `ofl/hankengrotesk/HankenGrotesk[wght].ttf`, `ofl/geistmono/GeistMono[wght].ttf` |
| Eszköz | `fontTools.subset` (`pip install fonttools brotli`) |
| Unicode-tartomány | `U+0020-007E,U+00A0-00FF,U+0100-017F,U+2010-2015,U+2018-201E,U+2020-2022,U+2026,U+2030,U+2032-2033,U+2039-203A,U+20AC,U+2122,U+2190-2193,U+2212,U+00D7,U+2260,U+2264-2265,U+2713` (latin alap, Latin-1, Latin Extended-A az ő/ű miatt, kötő- és gondolatjelek, idézőjelek, nyilak, néhány matematikai jel) |
| Layout-jellemzők | Hanken: `kern,liga,calt,ccmp,locl,mark,mkmk,tnum,lnum,case,ss01`; Geist Mono: `kern,liga,calt,ccmp,locl,mark,mkmk,tnum,lnum,case,zero,ss01` |
| Kapcsolók | `--flavor=woff2 --no-hinting --desubroutinize --layout-features=<lista>` |
| Kimenet | `HankenGrotesk-Variable.woff2`, `GeistMono-Variable.woff2`; a wght tengely 100–900 marad |

A szkript futtatása után a méret és a hash a forrásverziótól függően eltérhet; eltérésnél a T12 értékei és a `web/assets_test.go` együtt frissül, és ez a SUMMARY.md-be kerül.

### 10.7 Ismert figyelmeztetések

| Figyelmeztetés | Mikor | Teendő |
|---|---|---|
| `config_option_deprecated_alias` | minden `svelte-kit sync` (check, build, dev, vitest, playwright) | várható, nincs teendő |
| `tsconfig_invalid` („"paths" was overwritten. Imports from "$lib" may not typecheck”) | ugyanott | kozmetikai: a `$lib` importok típusellenőrzése működik |
| `npm notice` sorok a stderr-en | `npx` npm 12 alatt | ártalmatlan |
| `[PLUGIN_TIMINGS] JavaScript callbacks ran for …` (rolldown infósor) | `vite build` | tájékoztató, nincs teendő |
| `[WebServer] [start-server] server exited (code 1, signal none)` | az E2E futás végén Windowson | várható: a `global-teardown.ts` állítja le a szervert a webServer előtt (9.4.1) |
| `state_referenced_locally` | `$derived` olvasása closure-ön kívül | alkalmazáskódban a `--fail-on-warnings` megfogja, javítandó |

### 10.8 Dockerfile (teljes)

```dockerfile
# syntax=docker/dockerfile:1

# 1. Static web build
FROM --platform=$BUILDPLATFORM node:26-alpine AS web
WORKDIR /src/web
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# 2. Go binary with the embedded web build
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS server
ARG TARGETOS TARGETARCH
ARG VERSION=dev
ENV GOTOOLCHAIN=local
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/embed.go
COPY --from=web /src/web/build web/build
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=$VERSION" \
    -o /out/kinex-mark ./cmd/kinex-mark
RUN mkdir -p /out/data

# 3. Binary export only: docker build --target binary --output type=local,dest=dist .
FROM scratch AS binary
COPY --from=server /out/kinex-mark /kinex-mark

# 4. Runtime image: no shell, runs as the nonroot user (uid 65532)
FROM gcr.io/distroless/static-debian13:nonroot AS runtime
ARG VERSION=dev
LABEL org.opencontainers.image.source="https://github.com/kaszasg/kinex-mark" \
      org.opencontainers.image.title="Kinex Mark" \
      org.opencontainers.image.description="KINEXIUM Bookmark Manager" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="$VERSION"
COPY --from=server /out/kinex-mark /app/kinex-mark
COPY --from=server --chown=65532:65532 /out/data /data
ENV APP_ADDR=:8080 APP_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --start-interval=2s --retries=3 \
    CMD ["/app/kinex-mark", "-healthcheck"]
ENTRYPOINT ["/app/kinex-mark"]
```

| Stage | Alap | Feladat |
|---|---|---|
| `web` | `node:26-alpine`, build platformon | `npm ci` (cache mounttal), `npm run build`; a `.npmrc` miatt az `engine-strict` itt is érvényes |
| `server` | `golang:1.27-alpine`, build platformon | keresztfordítás a célplatformra, modul- és buildcache, üres `/out/data` |
| `binary` | `scratch` | csak a bináris, exporthoz |
| `runtime` | `gcr.io/distroless/static-debian13:nonroot` | uid 65532, shell nélkül, `/data` kötet, HEALTHCHECK |

- A runtime stage-ben nincs `RUN`, ezért a multi-arch build QEMU nélkül készül.
- `GOTOOLCHAIN=local`: a `go.mod` nem tölthet le csendben más toolchaint. A `golang:1.27-alpine` a `go 1.26.0` minimumnál (2.2 T22) újabb, ezért a direktíva emelése a Dockerfile-t nem érintette; ha egy későbbi direktíva a 1.27-et meghaladná, a build `GOTOOLCHAIN=local` mellett hibával áll le, és az image-címkét is emelni kell.
- `--start-interval=2s` (Docker Engine 25+): a konténer pár másodperc alatt `healthy`; a `-healthcheck` mód az `APP_ADDR`-t és a TLS-módot követi.
- Explicit `USER` nincs: a `:nonroot` image alapértelmezett felhasználója érvényes.
- Ha a `static-debian13:nonroot` címke nem érhető el (A3 ellenőrzi), a `static-debian12:nonroot` marad, és ez a SUMMARY.md-be kerül.

### 10.9 `compose.yaml` (teljes)

```yaml
services:
  kinex-mark:
    build:
      context: .
      target: runtime
      args:
        VERSION: ${VERSION:-dev}
    image: kinex-mark:latest
    # Use the published image instead of a local build:
    # image: ghcr.io/kaszasg/kinex-mark:latest
    restart: unless-stopped
    ports:
      # Loopback only: put a TLS reverse proxy in front of it for network access.
      - "127.0.0.1:8080:8080"
    environment:
      APP_LOG_FORMAT: json
      APP_SECURE_COOKIES: auto
      # Required behind a reverse proxy: the public origin. Without it the CSRF origin
      # check fails over TLS, cookies lose Secure, and no Host check protects the API.
      # APP_BASE_URL: https://mark.example.com
      # Required behind a reverse proxy: only the proxy's single address, never a whole range.
      # Pin that address in the proxy's own service, Docker assigns addresses dynamically:
      #   networks: { proxy: { ipv4_address: 172.30.0.2 } }
      # APP_TRUSTED_PROXY: 172.30.0.2/32
      # APP_ADMIN_USER: admin
      # Remove APP_ADMIN_PASSWORD after the first start (it stays visible in docker inspect).
      # APP_ADMIN_PASSWORD: ${APP_ADMIN_PASSWORD}
      # APP_ALLOW_PRIVATE_FETCH: "true"
      # APP_FETCH_WORKERS: "2"
    volumes:
      - kinex-mark-data:/data
    read_only: true
    tmpfs:
      - /tmp
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    # Behind a reverse proxy container, attach both to a dedicated network:
    # networks: [proxy]

volumes:
  kinex-mark-data:

# networks:
#   proxy:
#     ipam:
#       config:
#         - subnet: 172.30.0.0/29
```

- Reverse proxy mögött az `APP_BASE_URL` és az `APP_TRUSTED_PROXY` kötelező (6.2, 6.3): az előbbi nélkül TLS mögött minden módosító kérés 403 `csrf_rejected`, a süti nem `Secure`, HSTS nincs, és nincs Host-ellenőrzés; az utóbbi nélkül minden kliens a proxy IP-jén osztozik, így 30 hiba bárkitől mindenki bejelentkezését 15 percre zárolja. A megbízott cím mindig egyetlen proxy-cím egy dedikált hálózatban (a példában `172.30.0.2/32`), soha nem egy teljes magánhálózati tartomány, mert abból bármely konténer hamis `X-Forwarded-For` fejlécet küldhetne. A szerver Warn naplóval jelzi, ha nem megbízott forrásból `X-Forwarded-*` fejléc érkezik.
- Közvetlen, proxy nélküli hozzáférésnél (csak a `127.0.0.1` porton) egyik sem kell; ekkor a Host-ellenőrzés csak `localhost`, `127.0.0.1` és `[::1]` hostot enged, ha a figyelési cím loopback; a konténerben a cím `:8080`, ezért ott az `APP_BASE_URL` megadása ajánlott (DNS-rebinding elleni védelem, 6.3).
- Offline helyreállítás (zárolt vagy elfelejtett jelszavú admin, 6.12): `docker compose stop`, majd `docker compose run --rm kinex-mark -unlock-user admin` vagy `docker compose run --rm kinex-mark -reset-password admin`, végül `docker compose start`.
- A HEALTHCHECK a Dockerfile-ból jön.
- A titkok (`APP_ADMIN_PASSWORD`, `APP_SECRET_KEY`) a nem verziókövetett `.env` fájlból vagy a shell környezetéből jönnek; a fájlba titok nem kerül.
- A `secret.key` a kötetben jön létre (`/data/secret.key`), a mentés része. Mentés: a TECHNOLOGY.md 7.9 parancsai a `kinex-mark` szolgáltatás- és `kinex-mark_kinex-mark-data` kötetnévvel.

Ellenőrzés helyben (PowerShell):

```powershell
docker compose config --quiet
.\build.ps1 image
docker compose up -d --build
docker compose ps                                  # STATUS: healthy
docker compose logs kinex-mark | Select-String "initial admin password generated"
curl.exe -fsS http://127.0.0.1:8080/healthz
docker compose down
```

### 10.10 CI: `.github/workflows/ci.yml` (teljes)

Elvek:

- Minden workflow `permissions: contents: read`-del indul; csak a Docker-push job kap `packages: write`-ot.
- Az actionök major verzióra pinneltek. A DX a megírás előtt minden actionre lekérdezte a legfrissebb kiadást (a `gh` CLI hiányában a `https://api.github.com/repos/<owner>/<repo>/releases/latest` REST-végponttal, 2026-11-21), és a legújabb majort írta be; a 10.10 és a 10.11 listája ezt az állapotot rögzíti: `actions/checkout@v7`, `actions/setup-node@v7`, `actions/setup-go@v7`, `actions/upload-artifact@v7`, `actions/download-artifact@v8`, `actions/cache@v6`, `docker/setup-buildx-action@v4`, `docker/metadata-action@v6`, `docker/build-push-action@v7`, `docker/login-action@v4` (a kiadási jegyzetekben a használt inputokat érintő változás nincs). A Dependabot `github-actions` frissítése tartja naprakészen.
- `GOTOOLCHAIN=local` minden Go-lépésben; a mátrix alsó ága (`1.26.x`) a `go.mod` `go 1.26.0` direktívájával egyezik, így jelzi, ha egy függőség a direktívát tovább emelné (a direktíva a függőségek igényét követi, 2.2 T22); a felső ág (`1.27.x`) a Docker-build verziója.
- `concurrency`: PR-en az újabb futás megszakítja a régit; main-en és tagen nem.
- Cache: `setup-go` (`go.sum`), `setup-node` (`web/package-lock.json`), Playwright-böngészők, BuildKit `type=gha`.

```yaml
name: CI

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}

env:
  GO_VERSION: '1.27.x'
  NODE_VERSION: '26'
  GOTOOLCHAIN: local

jobs:
  web:
    name: Web (lint, check, unit tests, build)
    runs-on: ubuntu-latest
    timeout-minutes: 20
    defaults:
      run:
        working-directory: web
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-node@v7
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: npm
          cache-dependency-path: web/package-lock.json
      - run: npm ci
      - run: npm run lint
      - run: npm run check
      - run: npm run test:coverage
      - name: Coverage summary
        if: always()
        run: |
          node -e "
          const t = require('./coverage/coverage-summary.json').total;
          console.log('### Web unit test coverage');
          console.log('| Metric | Percent |');
          console.log('|---|---|');
          for (const k of ['statements','branches','functions','lines']) console.log('| ' + k + ' | ' + t[k].pct + '% |');
          " >> "$GITHUB_STEP_SUMMARY"
      - run: npm run build
      - uses: actions/upload-artifact@v7
        with:
          name: web-build
          path: web/build
          retention-days: 3
          if-no-files-found: error
      - uses: actions/upload-artifact@v7
        if: always()
        with:
          name: web-coverage
          path: web/coverage
          retention-days: 14

  go:
    name: Go ${{ matrix.go }} (fmt, vet, test)
    needs: web
    runs-on: ubuntu-latest
    timeout-minutes: 30
    strategy:
      fail-fast: false
      matrix:
        go: ['1.26.x', '1.27.x']
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version: ${{ matrix.go }}
          cache-dependency-path: go.sum
      - uses: actions/download-artifact@v8
        with:
          name: web-build
          path: web/build
      - name: gofmt
        run: |
          unformatted="$(gofmt -l cmd internal tools web/*.go)"
          if [ -n "$unformatted" ]; then echo "$unformatted"; exit 1; fi
      - run: go mod verify
      - run: go vet ./...
      - run: go run ./tools/langcheck
      - name: Test (race, coverage)
        run: go test -race -timeout 15m -covermode=atomic -coverprofile=coverage.out ./...
      - name: Coverage gate
        if: matrix.go == '1.27.x'
        run: go run ./tools/covercheck -profile coverage.out -total 85 -package 80 -markdown >> "$GITHUB_STEP_SUMMARY"
      - name: Vulnerability check
        if: matrix.go == '1.27.x'
        run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
      - name: Coverage report
        if: always() && matrix.go == '1.27.x'
        run: go tool cover -html=coverage.out -o coverage.html
      - uses: actions/upload-artifact@v7
        if: always() && matrix.go == '1.27.x'
        with:
          name: go-coverage
          path: |
            coverage.out
            coverage.html
          retention-days: 14

  go-windows:
    name: Go tests on Windows
    needs: web
    runs-on: windows-latest
    timeout-minutes: 30
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version: ${{ env.GO_VERSION }}
          cache-dependency-path: go.sum
      - uses: actions/download-artifact@v8
        with:
          name: web-build
          path: web/build
      - run: go test -timeout 15m ./...

  e2e:
    name: End-to-end (Playwright, real binary)
    needs: web
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version: ${{ env.GO_VERSION }}
          cache-dependency-path: go.sum
      - uses: actions/setup-node@v7
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: npm
          cache-dependency-path: web/package-lock.json
      - uses: actions/download-artifact@v8
        with:
          name: web-build
          path: web/build
      - name: Build binary
        run: CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=ci-${GITHUB_SHA::7}" -o dist/kinex-mark ./cmd/kinex-mark
      - run: npm ci
        working-directory: web
      - uses: actions/cache@v6
        with:
          path: ~/.cache/ms-playwright
          key: playwright-${{ runner.os }}-${{ hashFiles('web/package-lock.json') }}
      - run: npx playwright install --with-deps chromium
        working-directory: web
      - run: npm run test:e2e
        working-directory: web
        env:
          CI: 'true'
          KINEX_MARK_BIN: ${{ github.workspace }}/dist/kinex-mark
      - uses: actions/upload-artifact@v7
        if: failure()
        with:
          name: playwright-report
          path: |
            web/playwright-report
            web/test-results
          retention-days: 7

  binaries:
    name: Cross-compiled binaries
    needs: [web, go]
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v7
        with:
          go-version: ${{ env.GO_VERSION }}
          cache-dependency-path: go.sum
      - uses: actions/download-artifact@v8
        with:
          name: web-build
          path: web/build
      - name: Build
        run: |
          version="$(git describe --tags --always)"
          for target in linux/amd64 linux/arm64 windows/amd64 darwin/arm64; do
            os="${target%/*}"; arch="${target#*/}"; ext=""
            [ "$os" = windows ] && ext=".exe"
            CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
              -ldflags="-s -w -X main.version=$version" \
              -o "dist/kinex-mark-$os-$arch$ext" ./cmd/kinex-mark
          done
      - uses: actions/upload-artifact@v7
        with:
          name: binaries
          path: dist/
          retention-days: 14

  image:
    name: Docker image (build, smoke, push)
    # Push events only (main, v* tags); runs after every test job is green.
    if: github.event_name == 'push'
    needs: [web, go, go-windows, e2e]
    permissions:
      contents: read
      packages: write
    uses: ./.github/workflows/docker.yml
    with:
      push: true
```

| Job | Mit bizonyít |
|---|---|
| `web` | prettier, svelte-check 0 hiba és 0 figyelmeztetés, Vitest a küszöbökkel (köztük `app-css.spec` a checkoutban lévő GRAPHICS.md-vel és a `contract.spec`), statikus build; a `web-build` artefakt a többi job bemenete |
| `go` (1.26.x, 1.27.x) | gofmt, `go mod verify`, `go vet`, langcheck, `go test -race`, lefedettségi kapu és govulncheck az 1.27-es ágon; az 1.26-os ág a `go.mod` `go 1.26.0` minimumát bizonyítja (2.2 T22) |
| `go-windows` | a Go-tesztek Windowson is zöldek (útvonalak, fájlzárolás, sorvégek) |
| `e2e` | a valódi bináris és a beágyazott felület együtt; hibánál riport és trace |
| `binaries` | a négy platform fordul; a binárisok letölthetők |
| `image` | csak `push` eseményen (main, `v*` tag), és csak ha a `web`, mindkét `go`, a `go-windows` és az `e2e` job zöld: a `docker.yml` újrahívása `push: true`-val (build, smoke, többplatformos push). Így piros teszt mellett image nem kerül a ghcr.io-ra (D13) |

### 10.11 CD: `.github/workflows/docker.yml` (teljes)

A workflow két módon fut: PR-en önállóan (csak amd64 build és smoke, push nélkül, a CI-vel párhuzamosan), illetve a `ci.yml` `image` jobjából újrahívva (`workflow_call`, `push: true`) a tesztek után. Saját `push` triggere nincs, ezért tesztkapu nélkül soha nem publikál.

```yaml
name: Docker

on:
  pull_request:
  workflow_call:
    inputs:
      push:
        description: Push the multi-platform image to ghcr.io after the smoke test
        type: boolean
        required: false
        default: false

permissions:
  contents: read

concurrency:
  group: docker-${{ github.ref }}-${{ github.event_name }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}

env:
  IMAGE: ghcr.io/kaszasg/kinex-mark

jobs:
  image:
    name: Build, smoke test and push
    runs-on: ubuntu-latest
    timeout-minutes: 45
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0
      - name: Version
        id: version
        run: echo "version=$(git describe --tags --always)" >> "$GITHUB_OUTPUT"
      - uses: docker/setup-buildx-action@v4
      - name: Metadata
        id: meta
        uses: docker/metadata-action@v6
        with:
          images: ${{ env.IMAGE }}
          flavor: |
            latest=false
          tags: |
            type=ref,event=branch
            type=ref,event=pr
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=sha,format=short
            type=raw,value=latest,enable={{is_default_branch}}
          labels: |
            org.opencontainers.image.title=Kinex Mark
            org.opencontainers.image.description=KINEXIUM Bookmark Manager
            org.opencontainers.image.licenses=MIT
      - name: Build amd64 image for the smoke test
        uses: docker/build-push-action@v7
        with:
          context: .
          target: runtime
          platforms: linux/amd64
          load: true
          tags: kinex-mark:smoke
          build-args: VERSION=${{ steps.version.outputs.version }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
      - name: Smoke test
        run: |
          docker run -d --name km --read-only --tmpfs /tmp --cap-drop ALL \
            --security-opt no-new-privileges:true -p 127.0.0.1:8080:8080 -v km-data:/data kinex-mark:smoke
          for _ in $(seq 1 30); do
            status="$(docker inspect -f '{{.State.Health.Status}}' km)"
            [ "$status" = healthy ] && break
            sleep 2
          done
          docker inspect -f '{{.State.Health.Status}}' km | grep -qx healthy
          curl -fsS http://127.0.0.1:8080/healthz
          curl -fsS http://127.0.0.1:8080/ | grep -qi '<html'
          test "$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/api/links)" = 401
          docker logs km 2>&1 | grep -q 'initial admin password generated'
          test "$(docker run --rm kinex-mark:smoke -version)" = "${{ steps.version.outputs.version }}"
          docker rm -f km
      - name: Log in to ghcr.io
        if: inputs.push
        uses: docker/login-action@v4
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push (linux/amd64, linux/arm64)
        if: inputs.push
        uses: docker/build-push-action@v7
        with:
          context: .
          target: runtime
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: VERSION=${{ steps.version.outputs.version }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          provenance: mode=max
          sbom: true
```

A smoke-ciklus változója `_` (`for _ in $(seq 1 30)`): a ciklusváltozót a törzs nem használja, és `i` névvel az actionlint shellcheck-lépése SC2034-gyel elbukik.

Címkézés: `main` ágon `main`, `latest`, `sha-<rövid>`; `v1.2.3` tagen `1.2.3`, `1.2`, `sha-<rövid>`; PR-en nincs push (csak amd64 build és smoke; az `inputs.push` a `pull_request` eseménynél üres, tehát hamis). Az újrahívott workflow a hívó eseményét látja (`github.event_name == 'push'`, `github.ref`), ezért a `metadata-action` címkéi és a `git describe` verziója ugyanazok, mint önálló futásnál; a `packages: write` jogot a hívó `image` job adja át, a job-szintű `permissions` ennél többet nem kérhet. A provenance és az SBOM a push lépésben készül a `GITHUB_TOKEN`-nel. Az első push után a ghcr.io csomag privát; a láthatóságot a felhasználó állítja. A workflow-k csak a felhasználó általi push után futnak.

### 10.12 `.github/dependabot.yml`

```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule: { interval: weekly }
    groups:
      go-dependencies: { patterns: ['*'] }
  - package-ecosystem: npm
    directory: /web
    schedule: { interval: weekly }
    groups:
      web-dependencies: { patterns: ['*'], update-types: [minor, patch] }
    ignore:
      - dependency-name: typescript
        versions: ['>=7']
      - dependency-name: tailwind-variants
        versions: ['>=4']
  - package-ecosystem: github-actions
    directory: /
    schedule: { interval: weekly }
  - package-ecosystem: docker
    directory: /
    schedule: { interval: weekly }
    ignore:
      - dependency-name: node
        update-types: ['version-update:semver-major']
```

A Go-image frissítésekor a `ci.yml` `GO_VERSION` értékét kézzel kell követni; ha egy `gomod`-frissítés a `go.mod` `go` direktíváját emeli, a `ci.yml` mátrixának alsó ága vele együtt emelkedik (2.2 T22, 10.10). A Node-főverzió váltása tudatos döntés.

### 10.13 Helyi workflow-ellenőrzés

```powershell
docker run --rm -v "${PWD}:/repo" -w /repo rhysd/actionlint:latest -color
docker compose config --quiet
.\build.ps1 verify
```

---

## 11. Ütemterv

### 11.1 Szabályok minden fázisra

- Fejlesztő ügynök: Opus 5.5 (`claude-opus-5-5`). Kódfelderítő ügynök (keresés, összevetés, leltár): Sonnet 5.5. Felülvizsgáló: Opus 5.5, amely a bizonyítékgyűjtést Sonnet 5.5 felderítőkre bízhatja (a felderítő itt is a workflow külön lépése, lásd a következő pontot). Koordinátor: Opus 5.5, amely a munkacsomagokat indítja, a hullámokat lezárja és a tulajdonváltásokat vezeti.
- Felderítési szabály minden munkacsomagra (a fejlesztési hullámokban és a javítóknál is): más csomag kódjának felderítését (keresés, API-leltár, hívási helyek, például az F2 számára az F0 komponensei és store-jai, a B5 és a B6 számára a store API-ja) Sonnet 5.5 felderítő végzi. A fejlesztő és a javító ügynökök nem tudnak subagentet indítani (a munkamenetükben nincs Agent eszköz), ezért a felderítők a workflow külön lépéseiként futnak, és leltárt írnak a munkakönyvtárba: a második hullám előtt `scout\backend-inventory.md` (a Go-csomagok exportált API-ja `go doc -all` alapján, az api, a migrációk és a buktatók kézzel olvasott szakaszaival) és `scout\frontend-inventory.md` (a `web/` komponensei, store-jai, tesztsegédei, i18n-névterei, a hiányzó nevek „NOT PRESENT” listájával); a C fázis 3. lépésében az `integration\mismatches.md`. A fejlesztő és a javító ügynök ezekből a leltárakból dolgozik; ha egy konkrét kérdésre a leltár nem felel, a más tulajdonában lévő fájlt csak olvasásra, célzottan nyitja meg. Az Opus 5.5 csak a saját tulajdonú fájlokat és a terv megnevezett szakaszait írja. Az átadó jegyzet „Felderítés” sora rögzíti, mely leltárt használta, és mely idegen fájlokat olvasta közvetlenül, milyen kérdéshez (KV17 bizonyítéka). Az A fázis és az első hullám idején még nem volt leltár; az ottani csomagok idegen kódot nem, vagy csak a sajátjukhoz szükséges minimumot olvasták, és ezt a jegyzetükben rögzítették.
- Felülettervezés előtt (A2, F0–F5, R5 és a frontend-javító) a `frontend-design` skill betöltése kötelező; ütközésnél a GRAPHICS.md az irányadó, a skill a GRAPHICS.md keretein belül ad irányt (8.0). A komponensek a D3 verziói, a shadcn-svelte CLI 1.8.1 regisztrije.
- GUI-tesztelés kizárólag a Playwright plugin (MCP) eszközeivel; a böngésző egypéldányos, ezért egyszerre egy GUI-ügynök fut (C 4. lépés, E2, E3 sorban). Az E1 a saját Playwright-Chromiumát használja, ezért az MCP-ügynökkel párhuzamosan futhat.
- Git commit tilos (D14). A `git status` és a `git diff` olvasásra használható; a koordinátor hullámonként `git status`-szal ellenőrzi, hogy csak a tulajdonosok fájljai változtak.
- Fájltulajdon: egy fájlt egy időben egyetlen munkacsomag szerkeszt (11.10). A csonk (stub) fájlt a létrehozója a fázis végén átadja a megnevezett tulajdonosnak. Más tulajdonában lévő fájl változtatását a koordinátoron keresztül kell kérni.
- Közös, egy tulajdonossal változó fájlok: `internal/api/{routes.go,deps.go,errors.go}`, `contract_test.go`, `testdata/**` (B5); `internal/model/**` (A1, utána a B1 a koordinátor jóváhagyásával); `web/package.json` és `package-lock.json` (F0; új npm-függőség csak az F0-n keresztül kerül be, a többi csomag `npm install`-t nem futtat); `web/src/lib/api/**` (F0); `web/src/lib/i18n/messages/{hu,en,de}/index.ts` (F0); `go.mod`, `go.sum` (A1, utána a koordinátor).
- Munkakönyvtár: a verziókövetésen kívüli munkafájlok helye `$env:TEMP\kinex-mark-work\` (`handoff\`, `scout\`, `integration\`, `review\`, `gui\`, `graphics\`, `dep\`). A tartós eredmény a SUMMARY.md.
- Kész-feltétel minden munkacsomagra: a saját ellenőrző parancsai 0 kilépési kóddal futnak; az érintett fájlok gofmt- vagy prettier-formázottak; a langcheck tiszta; a csomag rövid átadó jegyzetet ír a `handoff\<id>.md`-be (mi kész, eltérések a tervtől, nyitott kérdések), amelyből a SUMMARY.md készül.
- A PowerShell-ellenőrzés a repo gyökeréből indul (`C:\Project\@KINEXIUM\kinex-mark`), webes parancsoknál `Set-Location web` után. A `go vet ./...` és a `go test ./...` a `web` csomag miatt `web/build`-et igényel; a csomagszintű parancsok (`./internal/...`) nélküle is futnak.

### 11.2 Áttekintés

```
A  (részben párhuzamos)  A1 Go-váz (elsőként a go.mod) | A2 web-váz (Go-ellenőrzése az A1 után) | A3 repo- és buildfájlok (build.ps1 check a fázis zárásakor)
B1 (párhuzamos)  B1 store | B2 auth | B3 fetch+bookmarks | B4 config+jobs | F0 frontend-alap | G grafika | DX Docker+CI+E2E-keret
B2 (párhuzamos)  B5 api-mag és fiók (0. mérföldkő -> B6 indul) | B6 api-linkek | B7 app+cmd | F1 belépés, nyilvános, kapuk | F2 linkek | F3 címkék, profil, megjelenítés | F4 admin | F5 biztonság és adatvédelem
C  (soros)       INT integráció -> javítók területenként
D  (párhuzamos)  R1–R6 felülvizsgálat -> V ellenséges ellenőrzés -> javítók területenként
E  (részben)     E1 E2E-specifikációk || E2 feltáró GUI-teszt (MCP) -> E3 képernyőképek (MCP)
F  (soros)       DOC1 kézikönyv || DOC2 README + SUMMARY -> FV végső ellenőrzés
```

A hullám akkor zárul, ha minden munkacsomagja kész; a következő hullám csak ezután indul. A B5 és a B6 a második hullámban indul, hogy a HTTP-tesztek a kész store-ral fussanak; a B6 ezen belül a B5 0. mérföldköve után kezd (11.5). A B7 a második hullámban az `app` és a `cmd` kódját írja; az `app_test.go` `/healthz` 200 állítása a B5 `h_public.go`-ján múlik, ezért a B7 kész-feltételében csak a B5 0. mérföldköve utáni állapotra vonatkozik, és a C fázis első lépése ellenőrzi újra.

Az A fázis sorrendje: az A1 első lépése a `go.mod` létrehozása (`go mod init`), csak utána indulnak az A2 Go-ellenőrzései (`go vet ./web`, `go test ./web`); az A2 webes lépései és az A3 kódja ezzel párhuzamosan fut. Az A3 `.\build.ps1 check` kész-feltételét a koordinátor ellenőrzi a fázis zárásakor, az A1 és az A2 elkészülte után.

### 11.3 A fázis: alapozás

**A1 – Go-váz**

- Cél: fordítható Go-modul a rögzített típusokkal, hibakódokkal, útvonaltáblával és interfészekkel, hogy a B fázis párhuzamosan indulhasson.
- Tulajdon: `go.mod`, `go.sum`, `internal/model/**`, `internal/clock/**`, `internal/api/{routes.go,deps.go,errors.go,respond.go,validate.go,context.go,contract_test.go,testdata/contract.json}` (az `errors.go`, `respond.go`, `validate.go` és `context.go` véglegesen, 3.3.10), csonkként `internal/api/api.go` és `internal/api/h_*.go` (`501 not_implemented`), `internal/testutil/{doc.go,clock.go}` fordítható vázként (3.3.12), `internal/store/{store.go,driver.go}`, `internal/store/migrations/0001–0004` (4. fejezet szó szerint), csonkként `internal/config/config.go`, `internal/jobs/interfaces.go`, `internal/app/app.go` (a `web` importja nélkül), `cmd/kinex-mark/main.go`.
- Bemenet: 3–5. fejezet, 6.10 (auditesemények), D1–D4.
- Lépések: `go mod init github.com/kaszasg/kinex-mark`; nincs `toolchain` sor; `go get modernc.org/sqlite@latest golang.org/x/net@latest`; `go mod tidy`. A `go` direktíva az lesz, amit a függőségek legfrissebb verziói megkövetelnek (a `go get` emeli; 2.2 T22), kézzel nem emeljük feljebb.
- Történet: az A1 2026-11-21-én még a TECHNOLOGY.md `go 1.24` minimumával dolgozott (`modernc.org/sqlite` v1.39.0, `golang.org/x/net` v0.43.0, amelyekkel a `go 1.24` sor változatlan maradt). A felhasználó ugyanaznap feloldotta a minimumot; a DEP-lépés (függőségfrissítés, a C fázis után) `go get -u ./... modernc.org/sqlite@latest golang.org/x/net@latest` és `go mod tidy` után `go 1.26.0`-ra emelte a direktívát (a `go get` kimenete: `go: upgraded go 1.24 => 1.26.0`), toolchain sor nélkül. A `go.mod` állapota: közvetlen `golang.org/x/net` v0.63.0, `modernc.org/sqlite` v1.62.0; közvetett `github.com/dustin/go-humanize` v1.1.0, `github.com/google/uuid` v1.6.0, `github.com/mattn/go-isatty` v0.0.24, `github.com/ncruces/go-strftime` v1.1.0, `github.com/remyoudompheng/bigfft` v0.0.0-20230129092748-24d4a6f8daec, `golang.org/x/sys` v0.50.0, `golang.org/x/text` v0.44.0, `modernc.org/libc` v1.82.9, `modernc.org/mathutil` v1.7.1, `modernc.org/memory` v1.13.0 (a `golang.org/x/exp` kikerült, a libc már nem igényli).
- Sorrend: a `go.mod` az A1 legelső lépése; az elkészültét az A1 jelzi a koordinátornak, aki ekkor engedi az A2 Go-lépéseit.
- Kész: `go build ./...` (a `web` nélkül) és `go vet ./internal/... ./cmd/...` hibátlan; a `contract.json` generált; a migrációk üres adatbázison lefutnak (store-csonk teszttel); a `respond.go`, `validate.go`, `context.go` egységtesztjei zöldek.
- Ellenőrzés:
  ```powershell
  go build ./cmd/... ./internal/...
  go vet ./cmd/... ./internal/...
  go test ./internal/model/... ./internal/clock/... ./internal/api/... ./internal/store/...
  Select-String -Path go.mod -Pattern '^go 1\.26\.0$'   # a 2.2 T22 szerinti mostani érték; toolchain sor nincs
  go mod tidy -diff
  go mod verify
  ```

**A2 – Web-váz**

- Cél: a 8.1 szerinti SvelteKit-projekt a design system bekötésével, a tesztkeretekkel és minden futásidejű függőséggel.
- Tulajdon az A fázisban: a `web/` váza (`package.json`, `package-lock.json`, `.npmrc`, `.gitignore`, `.prettierrc`, `.prettierignore`, `tsconfig.json`, `e2e/tsconfig.json`, `vite.config.ts`, `vitest.config.ts`, `playwright.config.ts` váza, `components.json`, `.vscode/`, `scripts/sync-app-css.mjs`, `static/**` az `images/` kivételével, `src/app.css`, `src/app.html`, `src/app.d.ts`, `src/routes/+layout.svelte`, `src/routes/+layout.ts`, `src/lib/components/ui/**`, `src/lib/hooks/**`, `src/lib/utils.ts`, `src/lib/index.ts`, `src/lib/test/{setup.ts,app-css.spec.ts}`), valamint `web/embed.go`, `web/assets_test.go`, `tools/fonts/build_fonts.py`. Az A fázis végén a `static/**`, az `embed.go`, az `assets_test.go` és a `build_fonts.py` az A2-nél marad (később a frontend-javítóé), a `playwright.config.ts` és az `e2e/tsconfig.json` a DX-hez, minden más az F0-hoz kerül.
- Lépések: a 8.1.1 telepítési sora Git Bashben; a fontok azonnali bemásolása a `static/fonts/` alá (a T12 méretével és hash-ével; a forrásfájlokat a koordinátor adja át, ezek a 10.6 szkriptjével is előállíthatók), az OFL-fájlok a TECHNOLOGY.md 6.10 szövegével; `favicon.svg`, `robots.txt`; `web/embed.go` (8.1.7); `web/assets_test.go` (9.2.2 `web` sora); `tools/fonts/build_fonts.py` (10.6); egy próbakomponens-teszt.
- Kész: `npm run check` 0 hiba és 0 figyelmeztetés, benne a `tsc --noEmit -p e2e/tsconfig.json` lépés a `playwright.config.ts`, a `vitest.config.ts` és egy próba `e2e/support` fájl típusellenőrzésével (`@types/node` telepítve; ha a `vitest.config.ts` nem fordul, a 8.1.1 tartalék szabálya és az átadó jegyzet); `npm run lint` és `npm run build` sikeres; a `web/build/index.html` két inline szkriptet tartalmaz (témaindító, SvelteKit-indító); `npm run test` zöld (köztük `app-css.spec.ts`); `go vet ./web` és `go test ./web` zöld az A1 `go.mod`-ja után (a `hero_test.go` ekkor még nincs).
- Ellenőrzés:
  ```powershell
  Set-Location web
  npm.cmd ci
  npm.cmd run check
  npm.cmd run lint
  npm.cmd run build
  npm.cmd run test
  Set-Location ..
  go vet ./web
  go test ./web
  ```

**A3 – Repo- és buildfájlok**

- Cél: a közös build- és ellenőrző infrastruktúra.
- Tulajdon: `.gitignore`, `.gitattributes`, `Makefile`, `build.ps1`, `tools/covercheck/**`, `tools/langcheck/**`.
- Bemenet: 10.1–10.5.
- Kész: a `build.ps1` UTF-8 BOM-mal és CRLF-fel mentve; a `covercheck` (alapértelmezett `-exclude`: `testutil`, `storetest`) és a `langcheck` saját egységtesztekkel; a `.\build.ps1 check` az A1 és az A2 után zöld (a koordinátor a fázis zárásakor futtatja); a `static-debian13:nonroot` címke elérhetősége ellenőrizve (az eredmény a DX bemenete).
- Ellenőrzés:
  ```powershell
  go test ./tools/...
  go vet ./tools/...
  (Get-Content -Path build.ps1 -Encoding Byte -TotalCount 3) -join ','   # 239,187,191 (Windows PowerShell 5.1)
  $errs = $null; [System.Management.Automation.Language.Parser]::ParseFile((Resolve-Path build.ps1).Path, [ref]$null, [ref]$errs) | Out-Null; $errs.Count   # 0
  .\build.ps1 check
  docker pull gcr.io/distroless/static-debian13:nonroot
  ```

### 11.4 B fázis, első hullám

**B1 – store**

- Tulajdon: `internal/store/**` (a csonkok átvétele A1-től), `internal/storetest/**`.
- Bemenet: 3.3.8, 3.3.12, 4. fejezet, 5.8.1, 6.7, 6.11, 6.13, 7.2, 7.3, 7.4.
- Kész: minden 3.3.8-as metódus implementálva; a 9.2.2 `store` sora teljesen lefedve (benne a `ThrottleUpdate` és a `ChallengeAttempt` párhuzamossági tesztje, a címeredet szabályai és az URL-változás közbeni letöltési eredmény eldobása); csomaglefedettség legalább 85%.
- Ellenőrzés: `go vet ./internal/store/... ./internal/storetest/...`; `go test -cover ./internal/store/...`

**B2 – auth**

- Tulajdon: `internal/auth/**`.
- Bemenet: 3.3.5, 6.4–6.7, 6.12.
- Kész: RFC 6238 tesztvektorok, PBKDF2-formátum, kulcstartó, korlátozás (`Check`/`Reserve`/`Confirm`/`Refund`, `UserThrottleKey`, `IPThrottleKey`); csomaglefedettség legalább 90%.
- Ellenőrzés: `go vet ./internal/auth/...`; `go test -cover ./internal/auth/...`

**B3 – fetch és bookmarks**

- Tulajdon: `internal/fetch/**`, `internal/bookmarks/**`, `internal/testutil/**` (az A1 vázának átvétele; az `origin.go` új).
- Bemenet: 3.3.6, 3.3.7, 3.3.12, 5.9.10, 5.9.11, 6.8, 7.3.
- Kész: SSRF-táblateszt a négy házirend-kombinációval, dial-szintű tiltás és átirányítás más loopback-portra bizonyítva, `CleanText`, kódolások, ikonválasztás, Netscape és JSON oda-vissza; csomaglefedettség legalább 85%.
- Ellenőrzés: `go vet ./internal/fetch/... ./internal/bookmarks/...`; `go test -cover ./internal/fetch/... ./internal/bookmarks/...`

**B4 – config és jobs**

- Tulajdon: `internal/config/**`, `internal/jobs/**`.
- Bemenet: 3.3.2, 3.3.9, 3.4, 7. fejezet.
- Kész: minden 3.4-es változó és kapcsoló validálva angol hibaüzenettel (`APP_ADMIN_USER` a 5.12 szabályával, `AdminCommand`); a runner sor-, backoff-, naplózási és leállítási tesztjei hamis órával és hamis store-ral, valódi alvás nélkül (a valódi store-os integráció a B1-é, 9.2.2).
- Ellenőrzés: `go vet ./internal/config/... ./internal/jobs/...`; `go test -cover ./internal/config/... ./internal/jobs/...` (a `-race` a CI Linux-jobjában fut).

**F0 – frontend-alap**

- Cél: minden felületi csomag közös alapja: API-kliens, i18n-réteg, store-ok, alkalmazáskeret, közös komponensek, kapuk, csonkok a második hullámnak.
- Tulajdon (átvétel A2-től: konfigurációk, `package.json`/`package-lock.json`, `ui/`, `hooks/`, `utils.ts`, `index.ts`, `app.*`, gyökér-layout, `test/`, `scripts/`):
  - `web/src/lib/api/**` (5.14, 8.3; a `types.ts` az 5.5 és az 5.8.1 szó szerint);
  - `web/src/lib/i18n/**` mag (`index.ts` a `hasMessage`/`tDynamic` párral, `types.ts`, `plural.ts`, `format.ts`, `detect.ts`, `errors.ts`, a négy spec), a `messages/{hu,en,de}/index.ts` és a `common` (benne `tagPicker`, `stepUp`), `nav` (benne `settingsTabs`), `errors`, `fields`, `roles`, `status` névtér mindhárom nyelven (8.5.3, 8.5.5–8.5.6 teljes táblái);
  - `web/src/lib/state/{session,preferences,public-config,tags,link-stats}.svelte.ts` (a `tags` az `onTagsChanged` feliratkozással, links-import nélkül), `web/src/lib/tags/colors.ts`, `web/src/lib/utils/**` (benne `navigation.ts` `safeNext`);
  - `web/src/lib/components/brand/**`, `web/src/lib/components/app/**`, `web/src/lib/test/**`;
  - `web/src/routes/{+layout.svelte,+layout.ts,+error.svelte}`, `(public)/+layout.svelte`, `(gate)/+layout.{svelte,ts}`, `(app)/+layout.{svelte,ts}`, `(app)/settings/{+layout.svelte,+page.ts}`, `(app)/admin/{+layout.ts,+page.ts}`.
- Csonkok átadásra (a hullám végén): a `messages/{hu,en,de}/{auth,landing,privacy}.ts` (F1), `{links,capture,fetch}.ts` (F2), `{tags,tagColors,settings}.ts` (F3), `{admin,audit}.ts` (F4), `security.ts` (F5) üres névtérként; `features/links/{CaptureController,ShortcutsDialog}.svelte` (F2); `features/auth/PasswordChangeForm.svelte` és `features/privacy/PrivacyNotice.svelte` (F1) a 8.2 végleges propjaival, típusos csonkként, hogy az F5 az F1-gyel párhuzamosan fordulhasson; minden második hullámbeli `+page.svelte` egy `EmptyState`-tel (8.7.0 útvonaltábla). Az F0 saját fájljai csak F0-névtérbeli kulcsot használnak (8.2 névtér-szabály), így az F0 `npm run check`-je a csonk névterek mellett is zöld.
- Kész: `npm run check`, `npm run lint`, `npm run test` zöld; a saját fájlok lefedettsége legalább 85%; a `frontend-design` skill betöltve; a keret mindkét témában és mobilon használható (üres oldalakkal); a `contract.spec.ts` az A1 golden-fájljával zöld (az F0 névterei miatt az `errors`, `fields`, `roles`, `nav.locale` kulcsok teljesek; az `audit`, `fetch`, `tagColors` csoportot a teszt az F2, F3, F4 kész-feltételéig `test.todo`-ként jelöli, az INT előtt mind aktív).
- Ellenőrzés:
  ```powershell
  Set-Location web
  npm.cmd run check
  npm.cmd run lint
  npx.cmd vitest run src/lib/api src/lib/i18n src/lib/state src/lib/tags src/lib/utils src/lib/components/app src/lib/components/brand src/lib/test
  npm.cmd run build
  ```

**G – nyitóoldali grafika**

- Cél: a `hero` kép a `gemini-3.1-flash-image` modellel, a GRAPHICS.md-hez illesztve, WebP-ben a `web/static/images/` alá (D13).
- Tulajdon: `graphics/generate_graphics.py`, `graphics/prompts.json`, `graphics/hero.png`, `web/static/images/**`, `web/hero_test.go`.
- Lépések: 8.8.4 (a szkript bővítése, a `prompts.json`, a generálás). Minden `--force` előtt a jelenlegi `hero.png` másolata a munkakönyvtár `graphics\hero-<n>.png` fájljába kerül; a végén a legjobb jelölt marad a `graphics/hero.png`-ben, és csak az szállítódik. Az ügynök a PNG-t a Read eszközzel megnézi, és az elfogadási listát tételesen kipipálja az átadó jegyzetben.
- Költségkeret: legfeljebb 4 fizetős hívás. Ha a 4. után sincs elfogadható kép, a legjobbat szállítja, a hiányokat rögzíti, további hívás csak a koordinátor és a felhasználó engedélyével.
- Kész: az elfogadási lista minden pontja teljesül; `hero_test.go` zöld; a hívások száma az átadó jegyzetben.
- Ellenőrzés:
  ```powershell
  python graphics/generate_graphics.py --list
  python graphics/generate_graphics.py hero
  go test ./web -run Hero
  ```

**DX – Docker, CI és E2E-keret**

- Tulajdon: `Dockerfile`, `.dockerignore`, `compose.yaml`, `.github/**`, `web/playwright.config.ts` és `web/e2e/tsconfig.json` (átvétel A2-től), `web/e2e/support/**` (benne a `global-teardown.ts` és a `seed-demo.mjs`), `web/e2e/fixtures/**` (benne a `site/` és a `demo-data.json`), `web/e2e/specs/smoke.spec.ts`.
- Bemenet: 9.4.1, 9.7, 10.1, 10.8–10.13; az A3 distroless-eredménye.
- Kész: actionlint tiszta (a `ci.yml` `image` jobja és a `docker.yml` `workflow_call` bemenete együtt); `docker compose config` érvényes; a `consoleGuard` és az `expectHttp` engedélylista a 9.4.1 szerint; a `start-server.mjs` régi adatkönyvtár-takarítása Windowson kipróbálva; a `smoke.spec.ts` lefut, amint a B7 kész (addig a futtatás a C fázis első lépése); a `docker build --target runtime` sikeres, amint a backend fordul; a `seed-demo.mjs` a 9.7 adataival.
- Ellenőrzés:
  ```powershell
  docker run --rm -v "${PWD}:/repo" -w /repo rhysd/actionlint:latest -color
  docker compose config --quiet
  Set-Location web
  npm.cmd run check
  ```

### 11.5 B fázis, második hullám

**B5 – api-mag és fiók**

- Tulajdon: `internal/api/{api.go,middleware.go,security.go,csrf.go,session.go,clientip.go,spa.go,h_public.go,h_auth.go,h_twofa.go,h_me.go,h_privacy.go,h_directory.go,h_admin_users.go,h_admin_settings.go,h_admin_audit.go,h_admin_jobs.go}`, az A1-től átvett `routes.go`, `deps.go`, `errors.go`, `respond.go`, `validate.go`, `context.go`, `contract_test.go`, `testdata/**`, valamint `apitest_test.go`, `errors_test.go` és a saját fájlok tesztjei.
- Bemenet: 3.3.10, 5. fejezet, 6.1–6.10.
- 0. mérföldkő (a hullám elején, elsőként): az `api.go` (`New` a middleware-lánccal és a `routes.go` szerinti regisztrációval, a még meg nem írt kezelők az A1 `501`-es csonkjai), a `session.go` (`authenticate`, `requireReady`, szerepkör-ellenőrzés, a `context.go` elérőivel), a `h_public.go` és az `apitest_test.go` (a `loginAs` a `store.CreateSession`-nel) végleges és fordul; a `go test ./internal/api/...` zöld. A B5 ezt jelzi a koordinátornak, aki ekkor indítja a B6-ot. Ezután a B5 csak olyan fájlt ment, amely fordul (félkész munka a saját fájlban sem törheti a csomag fordítását).
- Kész: a 9.2.3 1., 3., 4., 5., 7., 8., 9. csoportja; a CSP a valódi build két inline szkriptjének hash-ét adja.
- Ellenőrzés: `go vet ./internal/api/...`; `go test -cover ./internal/api/...`

**B6 – api-linkek**

- Tulajdon: `internal/api/{h_links.go,h_links_state.go,h_shares.go,h_icons.go,h_import.go,h_tags.go,dto.go}` és tesztjeik (`dto_test.go` is).
- Bemenet: 5.5, 5.9–5.11.
- Indulás: a B5 0. mérföldköve után; a közös segédeket (`respond.go`, `validate.go`, `context.go`, `apitest_test.go`) csak használja, nem másolja és nem módosítja (módosítási igény a koordinátoron át a B5-höz).
- Kész: a 9.2.3 2. (IDOR) és 6. csoportja; a `dto_test.go`.
- Ellenőrzés: `go test -cover -run 'Link|Share|Icon|Import|Tag|IDOR' ./internal/api/...`

**B7 – app és cmd**

- Tulajdon: `internal/app/**`, `cmd/kinex-mark/**`.
- Bemenet: 3.3.1, 3.3.11, 6.12, 7.5.
- Kész: az `app.Run` lépései és a `RunAdminCommand`; `app_test.go` (9.2.2; a `/healthz` 200 állítás a B5 0. mérföldköve utáni `h_public.go`-val zöld, és a C fázis 1. lépése újraellenőrzi); a bináris helyben indul.
- Ellenőrzés:
  ```powershell
  go test -cover ./internal/app/...
  .\build.ps1
  .\dist\kinex-mark.exe -version
  ```

**F1 – belépés, nyilvános oldalak és kapuk**

- Tulajdon: `web/src/routes/(public)/login/**`, `(public)/privacy/**`, `(gate)/change-password/**`, `(gate)/consent/**`, `web/src/lib/features/auth/**`, `web/src/lib/features/privacy/**` (a `PasswordChangeForm` és a `PrivacyNotice` F0-csonkjának átvétele; a propok nem változnak), `messages/{hu,en,de}/{auth,landing,privacy}.ts`.
- Tartalom: 8.7.2, 8.7.4, 8.7.10, 8.8.1–8.8.3.
- Kész: komponenstesztek a belépés minden hibakódjára és a 2FA-lépésre; mindhárom nyelv teljes.
- Ellenőrzés: `npx.cmd vitest run src/lib/features/auth src/lib/features/privacy`; `npm.cmd run check`

**F2 – linkek**

- Tulajdon: `web/src/routes/(app)/+page.svelte`, `web/src/lib/features/links/**` (a csonkok átvétele F0-tól), `web/src/lib/links/**`, `web/src/lib/state/links.svelte.ts`, `messages/{hu,en,de}/{links,capture,fetch}.ts`.
- Tartalom: 8.6 (`LinksStore`, feliratkozás a `tags` store `onTagsChanged`-jére, a `linkStats.refresh()` hívása), 8.7.3, 8.5.7.
- Kész: a 9.3.2 és 9.3.3 link-sorai; lefedettség a saját fájlokon legalább 85%.
- Ellenőrzés: `npx.cmd vitest run src/lib/features/links src/lib/links src/lib/state/links`; `npm.cmd run check`

**F3 – címkék, profil, megjelenítés**

- Tulajdon: `web/src/routes/(app)/tags/**`, `(app)/settings/profile/**`, `(app)/settings/display/**`, `web/src/lib/features/tags/**`, `web/src/lib/features/settings/{profile,display}/**`, `messages/{hu,en,de}/{tags,tagColors,settings}.ts`.
- Tartalom: 8.7.5, 8.7.6 Profil és Megjelenítés.
- Ellenőrzés: `npx.cmd vitest run src/lib/features/tags src/lib/features/settings/profile src/lib/features/settings/display`; `npm.cmd run check`

**F4 – adminfelület**

- Tulajdon: `web/src/routes/(app)/admin/{users,settings,audit}/**`, `web/src/lib/features/admin/**`, `messages/{hu,en,de}/{admin,audit}.ts`.
- Tartalom: 8.7.7–8.7.9, 8.5.8.
- Ellenőrzés: `npx.cmd vitest run src/lib/features/admin`; `npm.cmd run check`

**F5 – biztonság és adatvédelem a beállításokban**

- Tulajdon: `web/src/routes/(app)/settings/{security,privacy}/**`, `web/src/lib/features/settings/{security,privacy}/**`, `messages/{hu,en,de}/security.ts`.
- Tartalom: 8.7.6 Biztonság és Adatvédelem (a `PasswordChangeForm` és a `PrivacyNotice` a `features/auth` és `features/privacy` útvonaláról importálva; az F0 típusos csonkja miatt az F1-gyel párhuzamosan fordul, a komponenstesztek a csonkot `vi.mock`-kal helyettesítik).
- Ellenőrzés: `npx.cmd vitest run src/lib/features/settings/security src/lib/features/settings/privacy`; `npm.cmd run check`

Az F1–F5 kész-feltétele közös: a `frontend-design` skill betöltve; mindhárom nyelv teljes (`parity.spec.ts`, `contract.spec.ts` zöld); a saját fájlok lefedettsége legalább 85%; a képernyő a 8.7 leírásának minden állapotát (betöltés, üres, hiba, feltöltött) megvalósítja.

### 11.6 C fázis: integráció

**INT – integrátor (Opus 5.5)**

1. `.\build.ps1 verify` (az E2E-ben ekkor a `smoke.spec.ts` fut).
2. Füstpróba ideiglenes adatkönyvtárral:
   ```powershell
   $data = Join-Path $env:TEMP "kinex-mark-int"
   Remove-Item -Recurse -Force $data -ErrorAction SilentlyContinue
   $p = Start-Process -FilePath .\dist\kinex-mark.exe -ArgumentList '-addr','127.0.0.1:8090','-data',$data -PassThru -NoNewWindow -RedirectStandardOutput "$data.log"
   curl.exe -fsS http://127.0.0.1:8090/healthz
   curl.exe -s -o NUL -w "%{http_code}" http://127.0.0.1:8090/api/links
   curl.exe -sI http://127.0.0.1:8090/ | Select-String "Content-Security-Policy"
   Stop-Process -Id $p.Id
   ```
3. Egy Sonnet 5.5 felderítő összeveti: az `internal/model` és az `internal/api/dto.go` JSON-címkéi ↔ `web/src/lib/api/types.ts`; a `routes.go` ↔ `web/src/lib/api/*.ts` végpontjai és metódusai (5.14); a `contract.json` ↔ i18n-kulcsok. Az eltérések listája a munkakönyvtár `integration\mismatches.md` fájljába kerül.
4. Kézi bejárás a Playwright MCP-vel (egy ügynök): belépés, link felvétele, címke, megosztás, admin.

Javítás: a tulajdon a C fázistól területenként él: backend-javító (`internal/**`, `cmd/**`, `go.mod`, `go.sum`), frontend-javító (`web/src/**`, `web/static/**` az `images/` kivételével, `web/*.go` a `hero_test.go` kivételével, `web/` konfigurációk, `tools/fonts`), infra-javító (gyökérfájlok, `.github/**`, `tools/covercheck`, `tools/langcheck`, `web/playwright.config.ts`, `web/e2e/support/**`, `web/e2e/fixtures/**`). A három párhuzamosan dolgozhat, egymás területére nem nyúl.

Kész: `.\build.ps1 verify` zöld; a `mismatches.md` minden tétele lezárva; a lefedettségi kapuk teljesülnek.

### 11.7 D fázis: felülvizsgálat és ellenséges ellenőrzés

Felülvizsgálók (párhuzamosan, csak olvasnak; Opus 5.5, felderítéshez Sonnet 5.5), mindegyik a munkakönyvtár `review\<id>.md` fájljába ír, tételenként: súlyosság, `fájl:sor`, bizonyíték, javaslat.

| Id | Terület | Bemenet | Fő kérdések |
|---|---|---|---|
| R1 | Követelmény-lefedettség | BASE-SYSTEM-PROMPT.md, 12. fejezet | minden sor teljesül-e, van-e bizonyító teszt |
| R2 | Biztonság és jogosultság | 6. fejezet, OWASP ASVS L2 releváns pontjai | IDOR (beágyazott azonosítók is), CSRF és Host-ellenőrzés, munkamenet (kijelentkezéskor a sor törlése), korlátozás atomikussága és a második tényező védelme, SSRF (DNS-rebinding, átirányítás, loopback, beágyazott IPv4), CSP, fejlécek, bevitel-ellenőrzés (bidi), naplóba szivárgó titok vagy URL, ikon-kiszolgálás (SVG), admin megerősítés (step-up), proxy-konfiguráció |
| R3 | GDPR | D9, 6.10–6.14, 5.8.1, 8.7.10 | adattakarékosság (nincs beírt nem létező név az auditban és a korlátozási táblában, nincs név a `details`-ben, nincs URL a naplóban); a tájékoztató 13. cikk szerinti tartalma (jogalap 6. cikk (1) b) és f), címzettek, harmadik országba továbbítás, automatizált döntés, a 15–21. cikk jogai, felügyeleti hatóság, rendszergazdai jelszó-visszaállítás); a nyugtázás nem hozzájárulás-jogalap; export teljessége (célként érintett audit események, zárolási állapot, `sharedWithMe` csak látható URL-lel); törlés kaszkádja és pszeudonimizálás (`details`, `actor_id IS NULL` sorok, a törlési auditsor maga); audit-megőrzés; harmadik féltől jövő kérés hiánya |
| R4 | i18n és angol forráskód | D2, D10, 8.5 | mindhárom nyelv teljes és természetes, hangnem (GRAPHICS.md 1.), `html lang`, Intl-formátumok, nincs lokalizálatlan szöveg, angol kommentek és naplók (tesztfájlokban is) |
| R5 | Design-system megfelelés | GRAPHICS.md, `frontend-design` skill, 8.0, 8.9 | tokenek, tipográfia, chamfer, egy primary gomb, hangnem, ikonvonal 1,5, fókuszgyűrű, csökkentett mozgás, sötét és világos téma, oldalsáv és felső sáv mérete, hero |
| R6 | Tesztrések | 9. fejezet, lefedettségi riportok | lefedetlen ágak, hiányzó hibakód-tesztek, gyenge assertek, flaky minták |

**V – ellenséges ellenőrzés (Opus 5.5):** minden R-tételt megpróbál cáfolni vagy reprodukálni a futó binárison (IDOR- és CSRF-próbák `curl.exe`-vel ügyfél-munkamenettel, SSRF-próba `http://127.0.0.1`, `http://169.254.169.254` és átirányító URL-lel, lejárt munkamenet, 2FA-visszajátszás). Eredmény tételenként: megerősítve, cáfolva, javítva. Csak megerősített tétel kerül javításra.

**Javítás:** a C fázis három területi javítója, a megerősített tételek területe szerint; minden javításhoz regressziós teszt tartozik; utána a V újraellenőriz.

Kész: nincs nyitott magas vagy közepes súlyosságú tétel; az alacsonyak javítva vagy a SUMMARY.md „Ismert korlátok” szakaszában.

### 11.8 E fázis: GUI-tesztelés és képernyőképek

**E1 – E2E-specifikációk (Opus 5.5, párhuzamosan az E2-vel)**

- Tulajdon: `web/e2e/specs/**` (a `smoke.spec.ts` átvétele a DX-től).
- Tartalom: 9.4.2.
- Kész: minden specifikáció zöld helyben és `CI=true` mellett, három egymás utáni futásban flaky nélkül.
- Ellenőrzés:
  ```powershell
  .\build.ps1 e2e-install
  .\build.ps1 test-e2e
  $env:CI = 'true'; .\build.ps1 test-e2e; Remove-Item Env:CI
  ```

**E2 – feltáró GUI-teszt (Opus 5.5, Playwright MCP, egyedül)**

- Előkészítés: a bináris másolata külön helyre (a `dist\kinex-mark.exe` nem zárolódik), indítás friss adatkönyvtárral és ismert admin-jelszóval, majd a demóadatok betöltése:
  ```powershell
  $gui = Join-Path $env:TEMP "kinex-mark-gui"
  New-Item -ItemType Directory -Force $gui | Out-Null
  Copy-Item .\dist\kinex-mark.exe $gui -Force
  $env:APP_ADMIN_PASSWORD = '<demo-password>'
  $server = Start-Process -FilePath "$gui\kinex-mark.exe" -ArgumentList '-addr','127.0.0.1:8080','-data',"$gui\data" -PassThru
  Set-Location web
  npm.cmd run seed:demo -- --base http://127.0.0.1:8080 --password '<demo-password>'
  ```
- Forgatókönyvek és ellenőrzőlista: 9.5. Kimenet: a munkakönyvtár `gui\findings.md` fájlja. A javítást a frontend-javító végzi, utána az E2 újraellenőriz.
- A végén `browser_close`, a szerver leállítása (`Stop-Process -Id $server.Id`).

**E3 – képernyőképek a kézikönyvhöz (Opus 5.5, Playwright MCP, az E2 után)**

- Tulajdon: `manual/images/**`.
- Szabály: magyar nyelv, sötét téma (kivéve a világos témát és az angol nyelvet bemutató képet), 1440×900 (mobilnál 390×844), demóadatokkal; valódi személyes adat nincs a képeken; a 2FA-képernyő egy eldobható demófiók titkát mutatja. Fájlnevek: 13.2.
- Kész: a 13.2 lista minden képe megvan, PNG, legfeljebb 600 KB képenként.

### 11.9 F fázis: dokumentáció és végső ellenőrzés

**DOC1 – kézikönyv (Opus 5.5):** `manual/MANUAL.md` a 13.2 szerkezettel, relatív képhivatkozásokkal (`images/05-links-list.png`), tegező felszólító móddal a lépésekben (GRAPHICS.md 1.3), a felületi feliratokat szó szerint idézve a `messages/hu/*.ts`-ből.

**DOC2 – README és SUMMARY (Opus 5.5):** `README.md` a 13.3 szerint, `SUMMARY.md` a 13.1 szerint. Forrás: az átadó jegyzetek, a lefedettségi riportok, a review-jegyzetek.

**FV – végső ellenőrzés (Opus 5.5, utolsóként):**

```powershell
.\build.ps1 clean
$before = Get-FileHash go.mod, go.sum
.\build.ps1 tidy
Compare-Object $before (Get-FileHash go.mod, go.sum) -Property Hash   # empty: tidy changed nothing
.\build.ps1 verify                               # check, go test + gate, vitest + gate, e2e
.\build.ps1 cross -Version 0.1.0
.\build.ps1 image
docker buildx build --platform linux/amd64,linux/arm64 --target runtime --build-arg VERSION=0.1.0 .   # no push: proves the docker.yml multi-arch path
docker compose up -d --build
docker compose ps                                 # healthy
curl.exe -s -o NUL -w "%{http_code}" http://127.0.0.1:8080/api/links   # 401
docker compose run --rm kinex-mark -version       # 0.1.0 or the dev version
docker compose down
docker run --rm -v "${PWD}:/repo" -w /repo rhysd/actionlint:latest -color
go run ./tools/langcheck
```

A többplatformos `buildx` futás és a helyi smoke (csak olvasható konténer `healthy`, 401 az API-n, `-version`) eredménye a SUMMARY.md-be kerül a CI/CD bizonyítékaként, mert commit nélkül a GitHub-workflow-k nem futnak (D14). Ha a `buildx` alapértelmezett buildere nem tud többplatformos képet építeni, előtte `docker buildx create --use --name kinex-multi` (QEMU nem kell: a Dockerfile a `$BUILDPLATFORM`-on keresztfordít).

Ezen felül: minden `manual/images/*` hivatkozás létező fájlra mutat; a README parancsai lefutnak; egy friss Sonnet 5.5 ügynök a SUMMARY.md alapján, forráskód nélkül helyesen megválaszol 10 ellenőrző kérdést (például: hol él a láthatósági szabály, hogyan kerül be új hibakód, melyik fájl a jelszócsere-kapu).

Kész: minden parancs 0 kilépési kóddal; a SUMMARY.md tesztállapot-táblája a tényleges számokkal; a 12. fejezet mátrixának minden sora teljesül, vagy a SUMMARY.md ismert korlátai között indokolt.

### 11.10 Fájltulajdon összesítve

| Útvonal | A | B első hullám | B második hullám | C–F |
|---|---|---|---|---|
| `go.mod`, `go.sum` | A1 | koordinátor (kérésre) | koordinátor | backend-javító |
| `internal/model/**`, `internal/clock/**` | A1 | B1 (egyeztetéssel) | | backend-javító |
| `internal/store/**`, `internal/storetest/**` | A1 (csonk) | B1 | | backend-javító |
| `internal/auth/**` | | B2 | | backend-javító |
| `internal/testutil/**` | A1 (váz) | B3 | | backend-javító |
| `internal/fetch/**`, `internal/bookmarks/**` | | B3 | | backend-javító |
| `internal/config/**`, `internal/jobs/**` | A1 (csonk) | B4 | | backend-javító |
| `internal/api/` mag és fiók, `routes.go`, `deps.go`, `errors.go`, `respond.go`, `validate.go`, `context.go`, `contract_test.go`, `testdata/`, `apitest_test.go` | A1 (az `apitest_test.go` kivételével) | | B5 | backend-javító |
| `internal/api/` linkfájlok (`h_links*.go`, `h_shares.go`, `h_icons.go`, `h_import.go`, `h_tags.go`, `dto.go`) | A1 (csonk) | | B6 | backend-javító |
| `internal/app/**`, `cmd/**` | A1 (csonk) | | B7 | backend-javító |
| `.gitignore`, `.gitattributes`, `Makefile`, `build.ps1`, `tools/covercheck/**`, `tools/langcheck/**` | A3 | | | infra-javító |
| `Dockerfile`, `.dockerignore`, `compose.yaml`, `.github/**` | | DX | | infra-javító |
| `web/playwright.config.ts`, `web/e2e/tsconfig.json`, `web/e2e/support/**`, `web/e2e/fixtures/**` | A2 (váz) | DX | | infra-javító |
| `web/e2e/specs/smoke.spec.ts` | | DX | | E1 |
| `web/e2e/specs/**` (többi) | | | | E1 |
| `web/static/**` (az `images/` kivételével), `web/embed.go`, `web/assets_test.go`, `tools/fonts/**` | A2 | A2 (változatlan) | | frontend-javító |
| `web/` konfigurációk, `package.json`, `package-lock.json`, `scripts/`, `src/app.*`, `src/kinex-mark.css`, `components/ui/**`, `hooks/`, `utils.ts`, `index.ts` | A2 | F0 | F0 | frontend-javító |
| `lib/api/**`, `lib/i18n/**` mag és F0-névterek, `lib/state/{session,preferences,public-config,tags,link-stats}`, `lib/tags/**`, `lib/utils/**`, `components/{brand,app}/**`, `lib/test/**`, a 8.2 F0-útvonalfájljai | | F0 | F0 | frontend-javító |
| `features/{auth,privacy}` (az F0 a `PasswordChangeForm` és a `PrivacyNotice` típusos csonkját adja), `routes/(public)/{login,privacy}`, `routes/(gate)/{change-password,consent}`, `messages/*/{auth,landing,privacy}.ts` | | F0 (csonk) | F1 | frontend-javító |
| `features/links`, `lib/links`, `lib/state/links.svelte.ts`, `routes/(app)/+page.svelte`, `messages/*/{links,capture,fetch}.ts` | | F0 (csonk) | F2 | frontend-javító |
| `features/tags`, `features/settings/{profile,display}`, `routes/(app)/tags`, `routes/(app)/settings/{profile,display}`, `messages/*/{tags,tagColors,settings}.ts` | | F0 (csonk) | F3 | frontend-javító |
| `features/admin`, `routes/(app)/admin/{users,settings,audit}`, `messages/*/{admin,audit}.ts` | | F0 (csonk) | F4 | frontend-javító |
| `features/settings/{security,privacy}`, `routes/(app)/settings/{security,privacy}`, `messages/*/security.ts` | | F0 (csonk) | F5 | frontend-javító |
| `graphics/**`, `web/static/images/**`, `web/hero_test.go` | | G | | G (újragenerálás csak koordinátori engedéllyel) |
| `manual/images/**` | | | | E3 |
| `manual/MANUAL.md` | | | | DOC1 |
| `README.md`, `SUMMARY.md` | | | | DOC2 |
| `ai/BASE-SYSTEM-PLAN.md` | lead (a fejlesztés előtt kész) | | | |

A táblázat sorai diszjunktak: minden fájl pontosan egy sorba esik, és egy oszlopban pontosan egy tulajdonosa van. Az `ai/BASE-SYSTEM-PROMPT.md`, `ai/TECHNOLOGY.md`, `ai/GRAPHICS.md` és a `LICENSE` nem módosul.

---

## 12. Követelmény-megfeleltetési mátrix

Jelölés a bizonyításnál: `GoU` Go unit, `GoH` Go HTTP-teszt, `Vi` Vitest, `E2E` Playwright-specifikáció (fájlnév előtag nélkül), `GUI` feltáró MCP-teszt, `Rx` felülvizsgálat, `CI` workflow-lépés. Az azonosítók előtagja (FT, TE, FN, GR, KV, ZT) a BASE-SYSTEM-PROMPT szakaszát jelöli.

### 12.1 Feladat és technológia

| # | Követelmény | Tervelem | Bizonyítás |
|---|---|---|---|
| FT1 | Webes alkalmazás linkek tárolására és a gyűjtemény kezelésére | Go-backend + SvelteKit SPA; linkek, címkék, személyes állapot (3–5., 8.) | E2E `links-*`, R1 |
| FT2 | Projektnév: Kinex Mark | modul, bináris, image, `PublicConfig.appName`, oldalcímek (1.3) | E2E `smoke` (title), `docker.yml` smoke |
| TE1 | SvelteKit, Svelte 5, TypeScript, kliensoldali statikus build | adapter-static `fallback: 'index.html'`, `ssr = false`, runes kényszerítve (8.1) | CI `web` (`check`, `build`), A2 |
| TE2 | Tailwind CSS v4, shadcn-svelte saját kódbázisban | `@tailwindcss/vite` 4.3.5, `src/lib/components/ui/**` a CLI 1.8.1-ből (8.1.5) | `npm run check`, R5 |
| TE3 | Go `net/http` API és felületkiszolgálás | ServeMux metódusos mintákkal, `/api/` catch-all, SPA-kezelő (3.3.10) | GoH útvonaltábla-mátrix, E2E |
| TE4 | SQLite `modernc.org/sqlite`, CGO nélkül | `internal/store` (`modernc.org/sqlite` v1.62.0 a megszakításbiztos connectorral, 3.3.8), `CGO_ENABLED=0` minden buildben (4.) | GoU store (köztük `cancelsafe_test.go`), CI `binaries`, Docker build |
| TE5 | `go:embed`, egy bináris, minimális konténer | `web/embed.go` `all:build`, distroless nonroot (10.8) | `web/assets_test.go`, `docker.yml` smoke |
| TE6 | TECHNOLOGY.md szerinti technológia | váz-architektúra megtartva, eltérések: 2.2 (a Go-minimum a felhasználó döntésével `go 1.26.0`, T22) | R1, CI `go` mátrix (`1.26.x`, `1.27.x`, `GOTOOLCHAIN=local`), govulncheck |
| TE7 | GRAPHICS.md szerinti dizájn | `app.css` 13.2, tokenek, komponensszabályok, eltérések: 2.3 | `conventions.spec`, `app-css.spec`, R5, GUI |

### 12.2 Funkciók

| # | Követelmény | Tervelem | Bizonyítás |
|---|---|---|---|
| FN1 | Adminfelület a törzsadatok szerkesztésére | `/admin/users`, `/admin/settings` (`SystemSettings`), `/admin/audit`; `api/admin/*` (5.12, 8.7.7–8.7.9) | GoH 7. csoport, E2E `admin-*`, GUI |
| FN2 | Felhasználókezelés: admin, fejlesztő, ügyfél | `users.role`, jogosultsági mátrix (5.13), admin felhasználó-CRUD | GoH mátrix, GoU `model.Role`, E2E `admin-users`, `sharing`, `security` |
| FN3 | Gyors rögzítés drag-and-drop vagy kézi felvétel | `POST /api/links/quick`, ablakra ráhúzás fátyollal, beillesztés, `LinkDialog` (5.9.3, 8.7.3) | GoU `ExtractURL`, GoH quick, Vi `capture.ts`, E2E `links-capture` |
| FN4 | Favicon letöltése | ikonjelöltek + `/favicon.ico`, blob a DB-ben, `GET /api/links/{id}/icon` (5.9.9, 7.3) | GoU fetch, GoH ikon (CSP, ETag), E2E `links-detail` |
| FN5 | Cím letöltése, ha elérhető | `<title>`/`og:title`, kódolásfüggetlen, aszinkron sor; `not_html` magyarázata; `title_source` (`none`/`fetched`/`manual`): az URL-változás és a kézi frissítés a letöltött címet és az ikont frissíti, a kézi címet nem (4.4, 5.9.4, 5.9.7, 7.3) | GoU fetch, GoU store (címeredet), GoU jobs, GoH 6. csoport, E2E `links-detail` (URL-csere után új cím) |
| FN6 | Rövid leírás | `description` 0–500 rúna (5.9.2) | GoH validáció, Vi `LinkDialog`, E2E `links-detail` |
| FN7 | Színes címkék | `tags.color` hét elemű palettája (4.6), `TagChip` | GoU store, Vi `tags/colors`, E2E `tags` |
| FN8 | Csoportosítás csak címke alapján | a csoport egy címke linkjei: címkesor, oldalsáv „Gyakori címkék”, `/tags` linkszám-hivatkozás (`/?tags=<id>`), „Címke nélkül” (8.7.3); nincs mappa-entitás; Netscape-import a mappát címkévé lapítja (1.4) | GoU bookmarks, GoU store szűrők, E2E `links-filter` (csoport megnyitása oldalsávból és `/tags`-ről), R1 |
| FN9 | Lista és csempe nézet | `viewMode` preferencia, `LinkRow` és `LinkTile` | Vi, E2E `links-view`, GUI |
| FN10 | Szűrés szöveg vagy címke alapján | `q`, `tags`, `tagMatch`, `untagged`, `scope` (4.8, 5.9.1) | GoU store szűrők, GoH lista, Vi `query.ts`, E2E `links-filter` |
| FN11 | Megjelenített adatok ki/be (ikon, URL, cím, leírás) | `showIcon`, `showUrl`, `showTitle`, `showDescription` preferenciák; Megjelenítés menü és fül | Vi `presentation.ts`, `LinkRow`, E2E `links-view`, `settings` |
| FN12 | Használat szerinti sorrend | `use_count` csökkenő, `last_used_at`, `created_at`; `visit` végpont; használat = megnyitás (egy megnyitás egy látogatás) vagy másolás (4.8, 5.9.5) | GoU store rendezés, GoH `visit`, Vi `order.ts`, `LinkRow` (Enter egy `visit`), E2E `links-ranking`, `links-copy` |
| FN13 | Kitűzés, a kitűzöttek elöl, köztük is használat szerint | `pinned DESC`, utána ugyanaz a rangsor; „Kitűzött”/„Többi” csoport; archivált link kitűzése visszaállít | GoU store (két kitűzött link `use_count` és `last_used_at` szerint), Vi `order.ts` (ugyanez), E2E `links-ranking` (két kitűzött: C, A, utána B) |
| FN14 | Automatikus archiválás, archivált csak kapcsolóval látható | `link_user_state.archived_at`, `archived=exclude` alapból, „Archiváltak is” kapcsoló (7.2) | GoU store, jobs, E2E `links-archive` |
| FN15 | Archiválás, ha az időszakon belül nem használták (például 3 hónap) | `defaultArchiveDays` 90, felhasználói felülírás, alapidőpont és utolsó használat, kitűzött kimarad; `ExpiryMeter` (7.2, 8.7.3) | GoU `ArchiveInactive` 89/90/91 nap hamis órával, Vi `expiry.ts` |
| FN16 | Gyors másolás: URL vagy Markdown `[Leírás](URL)` | sor-, csempe- és részletek-menü, `c`/`m` billentyű, `toMarkdownLink`; a másolás használatnak számít (rangsor, archiválás, 5.9.5) | Vi `markdown.ts`, `LinkRow` (másolás után `visit`), E2E `links-copy` (vágólap és használatszám) |
| FN17 | Megjelenítés: leírás, különben cím, különben URL; hosszú URL helyett domain | `linkPresentation()`, `compactUrl`, `host` mező (8.7.3) | Vi `presentation.ts`, E2E `links-view` |
| FN18 | Minden információ elérhető (ikon, URL, cím, leírás, használati statisztika, rögzítés és utolsó használat ideje) | `LinkDetailsSheet` és `LinkDetail` (5.5, 8.7.3) | Vi `LinkDetailsSheet`, E2E `links-detail`, GUI |

### 12.3 Grafika

| # | Követelmény | Tervelem | Bizonyítás |
|---|---|---|---|
| GR1 | `gemini-3.1-flash-image` modell | `prompts.json` `model`, a szkript `DEFAULT_MODEL`-je (8.8.4) | G átadó jegyzet (modellnév a futás kimenetéből) |
| GR2 | A `graphics/generate_graphics.py` használata, `GEMINI_API_KEY` | a szkript bővítése `aspect_ratio`, `image_size`, `max_width`, `background` mezőkkel, pontos arányú vágással és háttér-normalizálással (8.8.4) | `--list`, futási napló, `web/hero_test.go` |
| GR3 | Stílus és színek a GRAPHICS.md szerint | `kinexium-diagram` stílus, elfogadási lista (8.8.4) | G ellenőrzés, R5 |

### 12.4 Követelmények

| # | Követelmény | Tervelem | Bizonyítás |
|---|---|---|---|
| KV1 | Minden elem lokalizálható, hu/en/de | saját i18n (8.5), névterek, Intl; a backend csak kódot ad (5.2); audit-részletek fordítva (8.5.8), nyelvsemleges Netscape-export (3.3.7), a helyreállító kódok fájlja a `security.recovery.file.*` kulcsokból | Vi `parity`, `contract` (benne `auditDetailKeys`), `conventions` (szövegnód, attribútum, toast-literál), E2E `i18n`, R4 |
| KV2 | A forráskódban csak angol | angol azonosítók, kommentek, naplók, hibák; `sync-app-css.mjs` | `tools/langcheck` (CI), `conventions.spec` 7. pont, R4 |
| KV3 | Nyelvválasztás az admin- és a felhasználói felületen | `LocaleSwitcher` a felső sávban (minden app- és admin-oldal), a bejelentkező és a kapuoldalon, a Megjelenítés fülön | E2E `i18n` (login, felhasználói és admin felület) |
| KV4 | Igényes, jól kezelhető GUI | GRAPHICS.md, `frontend-design` skill, 8.0 és 8.9 | E2E `a11y`, `responsive`, GUI E2, R5 |
| KV5 | GDPR | adattakarékos séma és audit (nincs beírt név, nincs név a `details`-ben), a 13. cikk szerinti tájékoztató jogalappal, címzettekkel, továbbítással, automatizált döntéssel, jogokkal és felügyeleti hatósággal; a tájékoztató nyugtázása (nem hozzájárulás); teljes export (célként érintett események, zárolás, látható megosztások); törlés teljes pszeudonimizálással; audit-megőrzés; csak szükséges süti; nincs harmadik fél (6.14) | GoH 3. és 8. csoport, GoU store export és purge, E2E `privacy`, `gates.last`, R3 |
| KV6 | A webfelület csak bejelentkezés után érhető el | minden `/api/` (a nyilvánosak kivételével) munkamenetet kér; a SPA kapui (8.7.0) | GoH mátrix (anonim 401), E2E `auth`, `security`, `docker.yml` smoke (401) |
| KV7 | Opcionális 2FA | TOTP RFC 6238, helyreállító kódok, admin-visszaállítás a cél munkameneteinek törlésével; a második tényező nem törhető (egy élő kihívás, atomikus próbálkozásszám, a jelszó nem nulláz, hiba mindkét kulcson) (5.7, 6.5–6.7) | GoU auth, GoU store (`ChallengeAttempt`), GoH 3. csoport (jelszó–4 hibás kód–jelszó → 429), E2E `twofa` |
| KV8 | Csak saját adat és megosztott link látható | SQL-láthatósági CTE (4.7), IDOR = 404 (beágyazott azonosítónál is: `DeleteShare`), admin sem lát idegen linket, az export is csak látható URL-t ad | GoU store mátrix és export, GoH IDOR (idegen `shareId` saját link alatt), E2E `sharing`, `security`, R2, V |
| KV9 | Tesztek a lehető legnagyobb lefedettséggel | 9. fejezet; Go ≥85% (csomag ≥80%), web 85/80/85/85 | CI kapuk, R6, SUMMARY tesztállapot |
| KV10 | Nyitóoldali grafika: high-tech, minimal, elegáns, a funkciókat bemutatja | `hero.webp` a bejelentkező oldalon; kompozíció: gyors rögzítés, címkék, szűrés, rangsor, elévülés (8.8) | G elfogadási lista, `hero_test.go`, E2E `a11y` (betölt), GUI, R5 |
| KV11 | Részletes fejlesztési terv a fejlesztés előtt: `ai/BASE-SYSTEM-PLAN.md` | ez a dokumentum | a fájl léte az A fázis előtt |
| KV12 | SUMMARY.md a fejlesztés után | DOC2, 13.1 | FV |
| KV13 | A SUMMARY.md alapján a fejlesztés folytatható | 13.1 szerkezet | FV: a friss Sonnet 5.5 ügynök 10 kérdése |
| KV14 | `frontend-design` plugin, legfrissebb komponensek | skill kötelező (11.1); D3 verziók; shadcn-svelte 1.8.1 | átadó jegyzetek, `package-lock.json`, R5 |
| KV15 | GUI-tesztelés a Playwright pluginnal | C 4. lépés, E2, E3 (MCP) | `gui\findings.md`, `manual/images` |
| KV16 | Fejlesztés Opus 5.5-tel | minden fejlesztő, javító és felülvizsgáló ügynök (11.1) | ütemterv |
| KV17 | Kódfelderítés Sonnet 5.5-tel | Sonnet 5.5 felderítők a workflow külön lépéseiként (a fejlesztő ügynök nem indíthat subagentet): a második hullám előtti leltárak (`scout\backend-inventory.md`, `scout\frontend-inventory.md`), amelyekből a fejlesztők és a javítók más csomag kódját megismerik (11.1), továbbá C 3. lépés (`integration\mismatches.md`), D, FV | ütemterv, a leltárfájlok, az átadó jegyzetek „Felderítés” sora |

### 12.5 Záró tételek

| # | Követelmény | Tervelem | Bizonyítás |
|---|---|---|---|
| ZT1 | Magyar kézikönyv képernyőképekkel: `manual/MANUAL.md` | DOC1, E3, 13.2 | FV: minden képhivatkozás létező fájlra mutat |
| ZT2 | README.md helyett rövid futtatási összefoglaló | DOC2, 13.3 | FV: a README parancsai lefutnak |
| ZT3 | GitHub CI/CD Docker image előállítással | `ci.yml` (tesztek, majd az `image` job `needs: [web, go, go-windows, e2e]`), `docker.yml` (`workflow_call`: build, smoke, többplatformos push; PR-en csak build és smoke), `dependabot.yml` (10.10–10.12) | actionlint, FV: helyi `docker buildx` amd64 + arm64 és smoke (eredmény a SUMMARY.md-ben) |

---

## 13. Dokumentációs termékek

### 13.1 `SUMMARY.md`

Cél: egy új AI-ügynök a forráskód átolvasása nélkül folytathassa a fejlesztést. Tömör, táblázatos, magyar, a kódelemek angolul. Minden szakasz a tényleges állapotot rögzíti, nem a tervet; ahol a megvalósítás eltér e tervtől, az eltérés és az oka is benne van.

| # | Szakasz | Kötelező tartalom |
|---|---|---|
| 1 | Állapot egy bekezdésben | verzió, dátum, mi kész, mi nincs; a `.\build.ps1 verify` utolsó eredménye |
| 2 | Architektúra-térkép | rétegábra (SPA ↔ `/api` ↔ api → store/auth/fetch/jobs ↔ SQLite), a Go-függőségi gráf, a kérés útja a middleware-ekkel, a háttérfolyamatok |
| 3 | Csomagleltár | Go-csomagonként: felelősség, fő típusok és függvények, fájlok, tesztfájlok; frontend-mappánként ugyanez (`lib/api`, `lib/i18n`, `lib/state`, `lib/links`, `features/*`, `components/*`) |
| 4 | Útvonalleltár | SPA-útvonalak (fájl, hozzáférés, fő komponens, kapu); API-végpontok (metódus, út, szint, kezelőfájl); a jogosultsági mátrix |
| 5 | API-összefoglaló | közös szabályok (fejlécek, hibatörzs, lapozás), a hibakódok listája, hivatkozás a `contract.json`-ra és a `types.ts`-re |
| 6 | Sémaösszefoglaló | táblák oszlopokkal egy sorban, kulcsok és korlátok, migrációk listája; a láthatósági CTE és a rendezés szabálya |
| 7 | Konfiguráció | minden kapcsoló és környezeti változó alapértékkel, a rendszerbeállítások és alapértékeik, a titkok helye (`secret.key`), megőrzési idők |
| 8 | Parancsok | build, dev, tesztek, E2E, image, grafika, demóadatok; Windows és Linux alak |
| 9 | Tesztállapot és lefedettség | tesztszámok szintenként, Go-lefedettség csomagonként, web-lefedettség, E2E-specifikációk listája, utolsó futás eredménye |
| 10 | Függőségek és verziók | Go-modulok, npm-csomagok a feloldott verzióval, toolchain, base image-ek; tudatos pinek (TypeScript ~6, tailwind-variants ^3) és okuk |
| 11 | Eltérések a TECHNOLOGY.md-től és a GRAPHICS.md-től | táblázat: téma, eltérés, döntés (a 2.2–2.4 végleges állapota) |
| 12 | Ismert korlátok | ami nem készült el vagy szándékosan kimaradt, a nyitott alacsony súlyosságú review-tételek, a hero-generálás eredménye |
| 13 | Következő lépések | priorizált lista, mindegyiknél az érintett fájlok és a bizonyítás módja; benne a (felhasználó, IP-tartomány) párra szűkített zárolás (6.7 ismert korlátja) |
| 13a | CI/CD bizonyíték | az FV helyi `docker buildx` (amd64 + arm64) és smoke eredménye, mert a workflow-k commit nélkül nem futottak (11.9) |
| 14 | Konvenciók röviden | fájltulajdon-modell; i18n-kulcsok elnevezése; hibakód-folyamat (új kód → `errors.go` → `contract.json` → `errors.ts` → fordítás mindhárom nyelven); új migráció; új útvonal felvétele (`routes.go` → mátrixteszt → API-modul → típus) |

### 13.2 `manual/MANUAL.md`

Magyar, tegező, felszólító módú lépések; a felületi feliratok szó szerint a `messages/hu/*.ts`-ből; képek relatív hivatkozással.

Fejezetek:

1. Bevezetés: mire való a Kinex Mark, szerepkörök röviden, böngésző-követelmény.
2. Belépés: bejelentkezés, kétlépcsős azonosítás, első belépés (kötelező jelszócsere, az adatkezelési tájékoztató megismerésének nyugtázása), nyelv választása belépés előtt, kijelentkezés.
3. A felület: oldalsáv, felső sáv, keresés, nyelvválasztó, témaváltás, mobil nézet, billentyűparancsok.
4. Linkek rögzítése: kézi felvétel, beillesztés, ráhúzás, a cím és az ikon letöltése, kézi frissítés, duplikáció.
5. Linkek használata: lista és csempe nézet, megjelenített adatok, szűrés szöveggel és címkével, rendezés és használat (a megnyitás és a másolás is használat), kitűzés, megnyitás, másolás (URL, Markdown), részletek, importálás és exportálás.
6. Archiválás: az automatikus archiválás szabálya (az a link archiválódik, amelyet az időszakon belül nem nyitottál meg és nem másoltál), az elévülés jelzője, archivált linkek megjelenítése, visszaállítás (megnyitással, másolással, kitűzéssel vagy a menüből), saját archiválási idő.
7. Címkék: létrehozás, színek, átnevezés, törlés, címkék a megosztott linkeken; csoportosítás címkével (a címke linkjeinek megnyitása az oldalsávból vagy a Címkék oldalról).
8. Megosztás (fejlesztő és admin): mindenkivel, szerepkörrel, felhasználóval; a megosztott linkek a címzettnél.
9. Beállítások: profil, jelszó, kétlépcsős azonosítás és helyreállító kódok, munkamenetek, megjelenítés.
10. Adatvédelem: tájékoztató (jogalapok, címzettek, jogaid, panasz), adataim letöltése, fiók törlése (bekapcsolt 2FA mellett kóddal; a megosztott linkjeid a címzetteknél is eltűnnek).
11. Adminisztráció: felhasználók (a törlés, a jelszó- és 2FA-visszaállítás, a szerepkör- és állapotváltás a saját jelszavaddal megerősítve; a jelszó-visszaállítás auditált, és ez az egyetlen út, amelyen admin egy fiókba beléphet), rendszerbeállítások (felügyeleti hatóság is), audit napló, archiválás azonnali futtatása.
12. Hibaelhárítás és gyakori kérdések: zárolt fiók (admin feloldás; ha az egyetlen admin zárolt vagy elfelejtette a jelszavát: `-unlock-user`, `-reset-password` parancs a szerveren, 6.12), elveszett 2FA-eszköz, nem jelenik meg az ikon vagy a cím, belső hálózati linkek, a másolás nem működik nem biztonságos (HTTP) kapcsolaton, „A kérés nem a Kinex Mark beállított címére érkezett” (`host_mismatch`).

- A függelék: szerepkörök és jogosultságok táblázata (az 5.13 felhasználói nyelven).
- B függelék: üzemeltetés röviden (indítás, Docker, mentés), hivatkozással a README.md-re.

Képernyőképek (`manual/images/`, PNG, 1440×900, ha nincs más jelölve; E3 készíti):

| Fájl | Tartalom | Fejezet |
|---|---|---|
| `01-login.png` | bejelentkező oldal a hero-grafikával | 2 |
| `02-login-2fa.png` | kétlépcsős azonosítás kódmezője | 2 |
| `03-password-change.png` | kötelező jelszócsere | 2 |
| `04-privacy-consent.png` | adatkezelési tájékoztató nyugtázása | 2 |
| `05-links-list.png` | linklista demóadatokkal, kitűzött linkek elöl | 3, 5 |
| `06-links-tiles.png` | csempe nézet | 5 |
| `07-quick-add.png` | „Link felvétele” dialógus | 4 |
| `08-drop-overlay.png` | ráhúzás közbeni fátyol | 4 |
| `09-fetch-pending.png` | frissen felvett link letöltés alatt | 4 |
| `10-filter.png` | szöveges és címkeszűrés aktív szűrőkkel | 5 |
| `11-display-toggles.png` | „Megjelenítés” menü kapcsolói | 5 |
| `12-copy-menu.png` | sorművelet-menü a másolással | 5 |
| `13-link-detail.png` | részletek lap a használati adatokkal | 5 |
| `14-archived.png` | archivált linkek megjelenítve | 6 |
| `15-tags.png` | címkék oldala | 7 |
| `16-tag-edit.png` | címke szerkesztése színválasztóval | 7 |
| `17-share-dialog.png` | megosztás dialógus | 8 |
| `18-shared-customer.png` | ügyfél nézete megosztott linkkel | 8 |
| `19-settings-profile.png` | Beállítások › Profil | 9 |
| `20-settings-2fa.png` | 2FA bekapcsolása QR-kóddal (demófiók) | 9 |
| `21-recovery-codes.png` | helyreállító kódok | 9 |
| `22-sessions.png` | munkamenetek | 9 |
| `23-settings-display.png` | Beállítások › Megjelenítés | 9 |
| `24-import-export.png` | az importálás eredménye („Importálás kész”) | 5 |
| `24-import-export-menu.png` | a nyitott „Import és export” menü (finisher-kiegészítés: a menü és az eredmény egy képen nem fér el) | 5 |
| `25-privacy.png` | Beállítások › Adatvédelem: letöltés és fióktörlés | 10 |
| `26-admin-users.png` | felhasználók listája | 11 |
| `27-admin-user-create.png` | új felhasználó ideiglenes jelszóval | 11 |
| `28-admin-settings.png` | rendszerbeállítások | 11 |
| `29-admin-audit.png` | audit napló szűrővel | 11 |
| `30-language-en.png` | angol nyelvű linklista | 3 |
| `31-light-theme.png` | világos téma | 3 |
| `32-mobile.png` | mobil nézet nyitott oldalsávval (390×844) | 3 |

### 13.3 `README.md`

Rövid magyar futtatási összefoglaló a GitHub-alap README helyett, legfeljebb 120 sor:

| # | Szakasz | Tartalom |
|---|---|---|
| 1 | Mi ez | egy bekezdés: Kinex Mark, a KINEXIUM linkgyűjteménye; egy bináris, beágyazott felület, SQLite |
| 2 | Követelmények | futtatáshoz semmi (bináris) vagy Docker; fejlesztéshez Go 1.26+ (a `go.mod` `go 1.26.0`; ajánlott 1.27), Node 26, npm |
| 3 | Gyors indítás Windowson | `.\build.ps1`, `.\dist\kinex-mark.exe`, `http://localhost:8080` |
| 4 | Gyors indítás Linuxon és macOS-en | `make build`, `./dist/kinex-mark` |
| 5 | Docker | `docker compose up -d --build` vagy `docker run` a `ghcr.io/kaszasg/kinex-mark` image-dzsel, kötet, loopback port; reverse proxy mögött **kötelező** az `APP_BASE_URL` (nyilvános origin) és az `APP_TRUSTED_PROXY` (a proxy egyetlen címe egy dedikált hálózatban, nem tartomány), különben a bejelentkezés 403, a süti nem `Secure`, és a korlátozás minden felhasználót együtt zárol; `APP_BASE_URL` nélkül nem loopback címen nincs Host-ellenőrzés (DNS-rebinding kockázat) |
| 6 | Első belépés és helyreállítás | `APP_ADMIN_USER`/`APP_ADMIN_PASSWORD` (az első indulás után a jelszót el kell távolítani a környezetből), vagy a naplóban a generált jelszó (`initial admin password generated`), kötelező jelszócsere; zárolt vagy elfelejtett jelszavú admin: `kinex-mark -unlock-user <név>` / `-reset-password <név>` (Dockerben `docker compose run --rm kinex-mark …`) |
| 7 | Konfiguráció | a legfontosabb kapcsolók és környezeti változók táblája (köztük `APP_ALLOW_PRIVATE_FETCH` és `APP_ALLOW_LOOPBACK_FETCH`; a teljes lista a SUMMARY.md-ben) |
| 8 | Fejlesztői mód | `.\build.ps1 dev` (Go `127.0.0.1:8080` + Vite `5173`), Linuxon két terminál |
| 9 | Tesztek | `.\build.ps1 verify` / `make verify`, `e2e-install` |
| 10 | CI és image | a két workflow röviden (az image csak zöld tesztek után kerül a ghcr.io-ra), címkézés |
| 11 | Mentés | az adatkönyvtár (`app.db`, `secret.key`) mentése |
| 12 | Hivatkozások | `manual/MANUAL.md`, `SUMMARY.md`, `ai/BASE-SYSTEM-PLAN.md`, licenc (MIT) |

---

## 14. Kockázatok

| # | Kockázat | Hatás | Kezelés |
|---|---|---|---|
| 1 | Új főverziók: SvelteKit 3, Vite 8, Svelte 5.59, Tailwind 4.3, bits-ui 2, Vitest 5 | dokumentálatlan viselkedés, ritkán használt API-k hibái | az alapfolyamat (check, build, vitest, E2E) próbaprojekttel igazolt; a lockfile rögzít; `npm ci` mindenhol; Dependabot csak minor és patch csoportban; ha a SvelteKit a `kit.alias`-t eltávolítja, a `$lib` → `#lib` váltás külön feladat |
| 2 | TypeScript 7 nem működik a SvelteKittel | a sync és a build leáll | `typescript ~6.0.3` pin (L1), Dependabot `>=7` tiltás, a SUMMARY.md-ben indokolva |
| 3 | A shadcn-svelte CLI interaktív és felülír | lefagyó vagy hiányos init, felülírt `app.css` | pinnelt `shadcn-svelte@1.8.1`, `--preset nova` és pipe-olt válaszok, `add -y -o`; az `app.css` a `sync-app-css.mjs`-sel az init és az add után; az `ui/**` kézzel nem szerkesztett (az overlay-kivétellel) |
| 4 | Fonthash-eltérés a TECHNOLOGY.md-től | a dokumentált hash-re épülő teszt bukna | a beépített fájlok tényleges mérete és hash-e a teszt alapja (T12); a `tools/fonts/build_fonts.py` reprodukál; a fájlok az A2 első lépésében a repóba kerülnek |
| 5 | Generált kép minősége és fizetős újragenerálás | gyenge vagy márkaidegen hero, költség | részletes stílusleírás, elfogadási lista, legfeljebb 4 hívás, jelöltek megőrzése, további hívás csak engedéllyel |
| 6 | Egy Go-függőség frissítése tovább emeli a `go` direktívát | a CI alsó mátrixága és a régebbi helyi toolchainek nem fordítanak; a Docker-image (`golang:1.27-alpine`) túlléphető | a minimum a felhasználó 2026-11-21-i döntése óta a függőségeket követi (2.2 T22): a direktíva most `go 1.26.0`, a legkisebb érték, amelyet a `modernc.org/sqlite` v1.62.0, a `modernc.org/libc` v1.82.9 és a `golang.org/x/{net,sys,text}` aktuális verziói megengednek. Emeléskor ugyanabban a módosításban emelkedik a `ci.yml` mátrixának alsó ága (`GOTOOLCHAIN=local` miatt az ág jelez), szükség esetén a Dockerfile `golang` címkéje, és a 1.3, 1.5, 13.3 sorai; a változás és az új alapértelmezett GODEBUG-értékek (3.3) a SUMMARY.md-be kerülnek |
| 6a | Elmaradt függőségfrissítés: sérülékeny modulverzió | ismert sérülékenység a buildben | a 2026-11-21-i frissítés után a govulncheck (v1.10.0) egyik szinten sem talál semmit (0 elérhető, 0 csomag-, 0 modulszintű; előtte 0 / 10 / 10 az x/net, x/sys, x/text régi verzióiban); a CI `go` jobja minden futáskor futtatja; Dependabot `gomod` heti csoportos frissítéssel; HTML-olvasás továbbra is csak `html.NewTokenizer`-rel, `html.Parse*`/`html.Render` hívás nélkül (D4) |
| 6b | A `modernc.org/sqlite` megszakítási versenyhelyzete (v1.39.0: megszakított lekérdezés után finalizálatlan statement, nyitva maradó `app.db`) | Windowson a `Store.Close` után nem törölhető adatbázisfájl, instabil tesztek | upstream javítva a v1.40.1-ben, a v1.62.0-n burok nélkül 2600 futásból 0 szivárgás; a `cancelSafeConnector` (3.3.8) mélységi védelemként és a `TestCancelledQueriesReleaseDatabaseFile` regressziós tesztként marad; driverfrissítéskor az új `stmt.query` újravizsgálandó, mielőtt bárki a burok eltávolítását javasolná |
| 7 | Windows: zárolt `.exe` | `Access is denied` buildkor | a dev-szerver `kinex-mark-dev.exe`, a GUI-szerver külön másolatból fut, az E2E-szerver a teszt végén leáll; az ügynökök futó példányt nem hagynak hátra |
| 8 | Windows: CRLF és BOM | gofmt-eltérés, hibás olvasás PowerShell 5.1-ben | `.gitattributes`; a `build.ps1` BOM-ját az A3 ellenőrzi; a CI gofmt-lépése Linuxon jelez |
| 9 | Windows: Defender vagy szinkronizált mappa zárolja a `node_modules`-t | lassú vagy hibás `npm ci` | a repo helyi meghajtón (`C:\Project`); egy ismétlés, utána a fájlzárat tartó folyamat azonosítása |
| 10 | Az E2E ideiglenes adatkönyvtára Windowson megmarad | szemét a `%TEMP%`-ben | a Playwright Windowson a `gracefulShutdown`-t nem támogatja és erőszakkal állít le, ezért a `start-server.mjs` saját takarítása ott nem fut; helyette a `global-teardown.ts` (amely a Playwright 1.65-ben a webServer leállítása előtt fut) előbb a PID-jelölő alapján maga állítja le a szervert, majd újrapróbálva törli a jelölőfájlban rögzített könyvtárat, és a `start-server.mjs` induláskor törli az 1 óránál régebbi `kinex-mark-e2e-*` maradékokat (9.4.1); Linuxon a SIGTERM-ágon is törlődik |
| 11 | A Playwright MCP böngészője egypéldányos, és a kimenetét a repóba írja | párhuzamos GUI-ügynökök összeakadnak; a `.playwright-mcp/` pillanatképei (benne tesztjelszavak, helyreállító kódok) a repóban maradnak | egyszerre egy GUI-ügynök (C 4., E2, E3 sorban); a végén `browser_close`, és a `.playwright-mcp/` a munkakönyvtárba kerül (9.5); `.gitignore`, `.dockerignore` és `web/.prettierignore` bejegyzés (10.1, 8.1.2); az E1 a saját Chromiumát használja |
| 12 | CSP és inline szkriptek | a build változó indítószkriptje miatt fehér oldal | induláskori hash-számítás (6.9, L2), GoH-teszt a hash-ekre, E2E CSP-figyelő minden oldalon, `ModeWatcher disableHeadScriptInjection` |
| 13 | Vágólap nem biztonságos kontextusban (HTTP LAN-címen) | a másolás nem működik | rejtett `textarea` + `execCommand('copy')` tartalék, sikertelenségnél teendőt mondó üzenet; a README TLS-t ajánl |
| 14 | Böngészőnként eltérő drag-and-drop adattípusok | a ráhúzás nem ismeri fel az URL-t | több típus sorrendben (`text/uri-list`, `text/plain`, `text/html` horgony), Vitest-táblateszt, a szerver `ExtractURL`-je a végső döntő |
| 15 | `-race` és a modernc SQLite lassú | a CI Go-jobja időtúllépéssel bukik | `-timeout 15m`, `t.Parallel()`, gyors hasher a tesztekben; ha mégis lassú, a race csak a nem store-csomagokon fut, és ezt a SUMMARY rögzíti |
| 16 | GitHub Actions és distroless címkék változnak | hibás workflow vagy build | a DX a megírás előtt lekérdezi a legfrissebb majorokat; Dependabot `github-actions` és `docker`; az A3 ellenőrzi a `static-debian13` címkét, tartalék `static-debian12` |
| 17 | Párhuzamos ügynökök ugyanazt a fájlt szerkesztik | elveszett változás | 11.10 tulajdontábla, csonk-átadás, közös fájlok egy tulajdonossal, hullámonkénti `git status` |
| 18 | A frontend és a backend szerződése elcsúszik | futásidejű hibák | `types.ts` az 5.5-ből szó szerint, `contract.json` + `contract.spec.ts`, az INT összevetése |
| 19 | A Gemini-hívás nem ad képet vagy a modell nem elérhető | nincs hero | a szkript 1-es kóddal jelez; egy ismétlés; utána a koordinátor dönt (a bejelentkező oldal kép nélkül is működik, a `hero_test.go` jelzi a hiányt) |
| 20 | A Netscape-import nagy fájllal terheli a letöltőt | lassú metaadat | az import nem tesz a sorba, a seprő percenként legfeljebb 50 függő linket vesz fel (7.3) |
| 21 | A GRAPHICS.md 13.2 blokkja és az `app.css` elcsúszik | eltérő tokenek | az `app.css` kizárólag a `sync-app-css.mjs`-ből; az `app-css.spec.ts` a CI-ban összeveti |
| 22 | Az i18n-kulcsok nyelvenként eltérnek vagy hiányoznak | lokalizálatlan szöveg a felületen | `Shape<typeof hu>` + `satisfies` fordítási időben, `parity.spec.ts`, `contract.spec.ts`, `conventions.spec.ts` szövegliterál-szabálya |
| 23 | Az egyetlen admin fiókját egy támadó ismételten zárolja (5 hibás jelszó, legfeljebb 24 óra) | az adminfelület elérhetetlen | admin feloldás és offline `-unlock-user` / `-reset-password` (6.12), README és kézikönyv; a (felhasználó, IP-tartomány) párra szűkített zárolás a következő lépések között (6.7) |
| 24 | Reverse proxy mögött hiányzó `APP_BASE_URL` / `APP_TRUSTED_PROXY` | 403 minden bejelentkezésre, nem `Secure` süti, közös IP-zárolás | kötelezőként dokumentálva (3.4, 10.9, 13.3), Warn napló nem megbízott `X-Forwarded-*` fejlécre (6.2), GoH 4–5. csoport |
