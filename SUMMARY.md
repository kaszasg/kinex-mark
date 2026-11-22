# Kinex Mark – fejlesztési összefoglaló (SUMMARY.md)

Állapot: **2026-11-22** (finisher). Ez a dokumentum a repó tényleges állapotát írja le (kódból, `go doc`, `go list`, a route-tábla, a migrációk, a `web/src` fa, a `package.json`, a `build.ps1`, a CI-workflow-k és a legutóbbi kapufuttatások alapján), nem a tervet. Célja: egy AI-ügynök ebből az egy fájlból folytathassa a fejlesztést. A kódelemek angolul, a magyarázat magyarul.

## Tartalom

0. [Gyors válaszok](#0-gyors-válaszok)
1. [Cél, hatókör, állapot](#1-cél-hatókör-állapot)
2. [Verziók és függőségek](#2-verziók-és-függőségek)
3. [Architektúra-térkép](#3-architektúra-térkép)
4. [Repository-térkép](#4-repository-térkép)
5. [Go-csomagleltár](#5-go-csomagleltár)
6. [Adatmodell](#6-adatmodell)
7. [API](#7-api)
8. [Biztonsági modell és GDPR](#8-biztonsági-modell-és-gdpr)
9. [Háttérfolyamatok](#9-háttérfolyamatok)
10. [Frontend-leltár](#10-frontend-leltár)
11. [Új funkció végponttól képernyőig (ellenőrzőlista) és konvenciók](#11-új-funkció-végponttól-képernyőig-ellenőrzőlista-és-konvenciók)
12. [Konfiguráció](#12-konfiguráció)
13. [Parancsok (Windows és Linux)](#13-parancsok-windows-és-linux)
14. [Tesztállapot és lefedettség](#14-tesztállapot-és-lefedettség)
15. [CI/CD és bizonyíték](#15-cicd-és-bizonyíték)
16. [Hero-grafika](#16-hero-grafika)
17. [Eltérések a TECHNOLOGY.md-től és a GRAPHICS.md-től](#17-eltérések-a-technologymd-től-és-a-graphicsmd-től)
18. [Workaroundok, amelyeket a karbantartónak ismernie kell](#18-workaroundok-amelyeket-a-karbantartónak-ismernie-kell)
19. [Ismert korlátok](#19-ismert-korlátok)
20. [Nyitott ötletek és következő lépések](#20-nyitott-ötletek-és-következő-lépések)
21. [Dokumentumtérkép](#21-dokumentumtérkép)

---

## 0. Gyors válaszok

| Kérdés | Válasz |
|---|---|
| Hol él a láthatósági szabály? | SQL-ben: `internal/store/link_query.go` `visibleCTE` (és `viewerCreatedAt`); minden láthatóságfüggő lekérdezés ezzel kezdődik. Az `api` a store-t nem importálja. |
| Hol a route-tábla és a jogosultság? | `internal/api/routes.go` `routes()` (59 sor, szintek N/K/B/D/A), a lánc `internal/api/session.go` `guard` → `authenticate` → `requireReady` → `requireRole`. Teszt: `internal/api/routes_test.go` `TestRoutePermissionMatrix`, darabszám-ellenőrzés: `internal/api/contract_test.go` `TestRouteTable` (`want 59`). |
| Hogyan kerül be új hibakód? | 11.1: `internal/api/errors.go` (konstans + `errorSpecs`) → `go test ./internal/api -run TestContractGolden -update` → `web/src/lib/api/errors.ts` `ErrorCode` → `errors.<code>.{title,text}` a `web/src/lib/i18n/messages/{hu,en,de}/errors.ts`-ben. A kódot egy `internal/api` kezelőtesztnek is hivatkoznia kell (`TestErrorCodesExercised`). |
| Hol a linkek rendezése? | Szerver: `internal/store/link_query.go` `linkOrder`. Kliens: `web/src/lib/links/order.ts` `compareLinks`/`sortLinks`. A kettőnek egyeznie kell (6.5). |
| Melyik fájl a jelszócsere-kapu? | Szerver: `internal/api/session.go` `requireReady` (403 `password_change_required`, utána `consent_required`). Kliens: a kapu a `me.mustChangePassword`-ből számolt `session.gate` (`web/src/lib/state/session.svelte.ts`); `web/src/routes/(app)/+layout.ts` (átirányít), `web/src/routes/(gate)/+layout.ts`, oldal: `web/src/routes/(gate)/change-password/+page.svelte` (`PasswordChangeForm`). |
| Hol a CSP? | `internal/api/security.go` `buildCSP`/`securityHeaders`; az inline szkriptek hash-ei induláskor: `internal/api/spa.go` `inlineScriptHashes`. Az ikonvégpont saját CSP-je: `internal/api/h_icons.go`. |
| Hol az archiválási szabály? | `internal/store/archive.go` (`ArchiveInactive`, `roleVisibilityUpsert`, `ownerSharesReturnUpsert`), ütemezés: `internal/jobs/archive.go`, `internal/jobs/runner.go`. |
| Hol az SSRF-védelem? | `internal/fetch/guard.go` `CheckIP`/`Policy`, a tárcsázáskori ellenőrzés `internal/fetch/fetch.go` (`net.Dialer.Control`). |
| Hogyan jön létre az `app.css`? | Kizárólag `web/scripts/sync-app-css.mjs` (`npm run sync:css`) generálja az `ai/GRAPHICS.md` 13.2 blokkjából; kézzel nem szerkeszthető, `web/src/lib/test/app-css.spec.ts` ellenőrzi. |
| Hol kell új felületi szöveget felvenni? | `web/src/lib/i18n/messages/hu/<namespace>.ts` (típusforrás), majd ugyanaz a kulcs `en/` és `de/` alatt (`satisfies Shape<typeof hu>` fordítási hibát ad, ha hiányzik). |
| Mi az utolsó kapueredmény? | Finisher (2026-11-22): `.\build.ps1 verify` exit 0; Go 95,3% (covercheck), Vitest 73 fájl / 1107 teszt, E2E 81/81. Előtte FV (2026-11-21): verify exit 0 (325 s), 95,3%, 1102 teszt, E2E 81/81 kétszer; race (Docker), govulncheck, többplatformos buildx és Compose-smoke zöld (15.3). |

## 1. Cél, hatókör, állapot

**Cél.** A Kinex Mark a KINEXIUM belső könyvjelzőkezelője (projektnév: Kinex Mark, Go-modul `github.com/kaszasg/kinex-mark`, bináris `kinex-mark`, image `ghcr.io/kaszasg/kinex-mark`). Egyetlen statikus Go-bináris (CGO nélkül) a JSON API-val és a beágyazott SvelteKit SPA-val, SQLite-tárolással (`modernc.org/sqlite`).

**Funkciók (mind kész):** gyors rögzítés (ráhúzás, beillesztés, kézi felvétel, `POST /api/links/quick`), szerveroldali cím- és faviconletöltés aszinkron sorral és SSRF-védelemmel, rövid leírás, színes címkék (7 színű paletta), csoportosítás csak címkével, lista és csempe nézet, szöveges és címkeszűrés, megjelenítési kapcsolók (ikon, URL, cím, leírás), használat szerinti rangsor (megnyitás és másolás is használat), kitűzés, automatikus archiválás (alap 90 nap, felhasználónként felülírható, 0 = soha), másolás URL-ként és Markdownként, részletek lap, megosztás (mindenkivel, szerepkörrel, felhasználóval), import/export (JSON `kinex-mark-links/1`, Netscape HTML), három szerepkör (`admin`, `developer`, `customer`), adminfelület (felhasználók, rendszerbeállítások, audit napló, archiválás azonnal), opcionális TOTP 2FA helyreállító kódokkal, kötelező jelszócsere, GDPR (tájékoztató nyugtázása, adatexport, fióktörlés pszeudonimizálással), hu/en/de felület, sötét alap és teljes értékű világos téma, nyitóoldali hero-grafika.

**Hatókörön kívül (szándékosan):** mappák vagy hierarchia (importnál a mappa címkévé lapul), e-mail és telefonszám, jelszó-visszaállítás e-mailben (az admin állítja vissza), böngészőbővítmény, nyilvános (bejelentkezés nélküli) megosztás, analitika, harmadik féltől betöltött tartalom.

**Állapot (2026-11-22).** A terv A–E fázisa kész: A alapozás, B fejlesztés két hullámban, C integráció, D hatszempontú felülvizsgálat ellenséges újraellenőrzéssel és javításokkal, E teljes E2E-csomag, feltáró GUI-teszt és a kézikönyv képernyőképei (`manual/images/01-…32-*.png`, 33 kép: a 24-es sorszámon két fájl osztozik). Az F fázis (dokumentáció és végső ellenőrzés) is kész: ez a fájl és a `README.md` (DOC2), a `manual/MANUAL.md` (DOC1) és az FV végső ellenőrzése (eredménye a 14. fejezetben és a 15.3 pontban). Utána egy finisher-kör három független audit 41 megállapítását dolgozta fel (javítások: ékezetfüggetlen keresés és névrendezés, `import_too_many_entries` hibakód, kézikönyv-pontosítások, négy új vagy újrafelvett kézikönyvkép; részletek: 17.4, átadó: `handoff\FINISH.md`).

- Legutolsó teljes kapu: `.\build.ps1 verify` **exit 0** a finisher módosításai után (2026-11-22; számok: 14. fejezet). Az azt megelőző: `.\build.ps1 verify` **exit 0** (FV, 325 s; előtte E-close, 336 s): svelte-check 1570 fájl 0 hiba 0 figyelmeztetés, prettier tiszta, gofmt/go vet tiszta, langcheck 547 fájl, Go-tesztek mind ok, covercheck 95,3%, Vitest 72 fájl/1102 teszt, E2E 81 passed.
- DOC2 megerősítő futtatás ugyanazon a napon: `go test -count=1 -cover ./internal/... ./tools/... ./web/` minden csomag ok, a csomagonkénti számok azonosak (14. fejezet); `npx vitest run --coverage`: 72 fájl, 1102 teszt passed, 98,69 / 91,1 / 99,06 / 99,62%.
- Git: a kezdeti commit (`e0e1df1`, a `LICENSE`) után a teljes projekt egyetlen commitban került a `main` ágra 2026-11-22-én, a felhasználó kérésére, és ugyanekkor ment az első push; ez indítja a GitHub-workflow-k első futását, amelynek eredménye ebben a fájlban még nincs rögzítve (20. fejezet 1. pont).
- Verzió: a bináris verzióját a `git describe --tags --always --dirty` adja, ennek hiányában `dev` (`-ldflags -X main.version=…`).

## 2. Verziók és függőségek

### 2.1 Toolchain (a fejlesztőgépen mérve)

| Eszköz | Verzió | Megjegyzés |
|---|---|---|
| Go | go1.27.4 windows/amd64 | `go.mod`: `go 1.26.0`, nincs `toolchain` sor |
| Node / npm | v26.13.1 / 12.2.2 | `web/package.json` `engines`: `^24 \|\| >=26`, `web/.npmrc` `engine-strict=true` |
| Python | 3.14.10 | csak a hero-generátorhoz (`google-genai`, Pillow a felhasználói site-ban) |
| Docker | 29.9.3 | Linux-konténer mód, Docker Desktop containerd image store; buildx `desktop-linux` builder (amd64 és arm64) |
| Shell | Windows 11, PowerShell 5.1 (elsődleges), Git Bash | `make` és `pnpm` nincs helyben |

### 2.2 Go (`go.mod`)

- `go 1.26.0`: a legkisebb érték, amelyet a frissített modulok engednek (`modernc.org/sqlite` v1.62.0, `modernc.org/libc` v1.82.9, `golang.org/x/{net,sys,text}` mind `go 1.26.0`-t deklarál). A felhasználó 2026-11-21-i döntése oldotta fel a TECHNOLOGY.md `go 1.24` minimumát (17.1 T22).
- Közvetlen: `golang.org/x/net` v0.63.0 (csak `html`, `html/atom`, `html/charset`; HTML-t csak `html.NewTokenizer` olvas, három helyen: `internal/fetch`, `internal/bookmarks` és `internal/api/spa.go` `inlineScriptHashes`; `html.Parse*`/`html.Render` tilos), `golang.org/x/text` v0.44.0 (csak `unicode/norm`, a `foldText` kulcshoz; a finisher óta közvetlen, korábban közvetett), `modernc.org/sqlite` v1.62.0.
- Közvetett: `github.com/dustin/go-humanize` v1.1.0, `github.com/google/uuid` v1.6.0, `github.com/mattn/go-isatty` v0.0.24, `github.com/ncruces/go-strftime` v1.1.0, `github.com/remyoudompheng/bigfft` v0.0.0-20230129092748-24d4a6f8daec, `golang.org/x/sys` v0.50.0, `modernc.org/libc` v1.82.9, `modernc.org/mathutil` v1.7.1, `modernc.org/memory` v1.13.0.
- govulncheck (DEP-futás): a frissítés előtt 0 elérhető, 10 csomag- és 10 modulszintű találat; utána minden szinten 0.
- govulncheck (FV, 2026-11-21, v1.10.0): `No vulnerabilities found`; `npm audit --omit=dev`: 0 sebezhetőség. A finisher új közvetlen függősége (`golang.org/x/text` v0.44.0) ugyanaz a modulverzió, amelyet az FV már átvizsgált.
- A `go 1.26.0` direktíva miatt az alapértelmezett GODEBUG a Go 1.25/1.26 viselkedését adja: konténerérzékeny GOMAXPROCS, szigorúbb `url.Parse`, nincs SHA-1 a TLS 1.2 kézfogásban, hibrid ML-KEM. `//go:debug` felülírás nincs.
- A Go 1.25+ std API-k (`http.CrossOriginProtection`, `sync.WaitGroup.Go`, `testing/synctest`) elérhetők, de a kód nem használja őket.

### 2.3 npm (`web/package-lock.json`, feloldott verzió)

| Csomag | Verzió | Csomag | Verzió |
|---|---|---|---|
| `@sveltejs/kit` | 3.4.0 | `svelte` | 5.59.1 |
| `vite` | 8.3.8 | `@sveltejs/vite-plugin-svelte` | 7.3.1 |
| `@sveltejs/adapter-static` | 4.0.1 | `typescript` | 6.0.3 (`~6.0.3`, pin) |
| `tailwindcss`, `@tailwindcss/vite` | 4.3.5 | `tailwind-variants` | 3.3.1 (`^3.3.1`, pin) |
| `shadcn-svelte` (CLI) | 1.8.1 | `bits-ui` | 2.21.0 |
| `mode-watcher` | 1.1.0 | `@lucide/svelte` | 1.77.0 |
| `svelte-sonner` | 1.2.3 | `uqr` (QR) | 0.1.2 |
| `cn` | 0.5.2 | `tw-animate-css` | 1.4.0 |
| `@internationalized/date` | 3.12.4 | `svelte-check` | 4.7.8 |
| `vitest`, `@vitest/coverage-v8` | 5.1.1 | `jsdom` | 31.0.1 |
| `@testing-library/svelte` | 5.4.2 | `@testing-library/jest-dom` | 8.0.0 |
| `@testing-library/user-event` | 14.6.13 | `@playwright/test` | 1.65.1 |
| `@axe-core/playwright` | 4.14.0 | `prettier` / `prettier-plugin-svelte` | 3.10.1 / 4.2.0 |
| `@types/node` | 26.6.6 | | |

Tudatos pinek: `typescript ~6.0.3`, mert a TypeScript 7 alatt a `svelte-kit sync` elbukik (hiányzó compiler API), és a `svelte-check` peer-tartománya `^5 || ^6`; `tailwind-variants ^3.3.1`, mert a generált shadcn-komponensek a 3-as API-t használják. A Dependabot mindkettő főverzió-emelését tiltja (`.github/dependabot.yml`).

### 2.4 Base image-ek (`Dockerfile`)

| Szakasz | Image |
|---|---|
| `web` | `node:26-alpine` (`--platform=$BUILDPLATFORM`) |
| `server` | `golang:1.27-alpine` (`GOTOOLCHAIN=local`, keresztfordítás `TARGETOS/TARGETARCH`) |
| `binary` | `scratch` (csak a bináris exportja) |
| `runtime` | `gcr.io/distroless/static-debian13:nonroot` (uid 65532, `ENV APP_ADDR=:8080 APP_DATA_DIR=/data`, `VOLUME /data`, `HEALTHCHECK ["/app/kinex-mark","-healthcheck"]`) |

## 3. Architektúra-térkép

### 3.1 Rétegek

```
Böngésző: SvelteKit SPA (adapter-static, ssr=false), beágyazva a binárisba
   │  same-origin fetch, credentials: same-origin, X-Kinex-Request: 1, Accept: application/json
   ▼
Go net/http (cmd/kinex-mark → internal/app.Run)
   recoverer → securityHeaders → accessLog → hostCheck(/api/) → csrf(/api/, nem biztonságos metódus) → limitBody → ServeMux
   ├── GET /healthz                       (nyilvános)
   ├── /api/...  guard(route): authenticate → requireReady (kapuk) → requireRole → kezelő
   │        ├─► api.Store  ──► internal/store (SQLite, modernc, 1 kapcsolat, WAL)
   │        ├─► auth (PBKDF2, TOTP, Keyring, ThrottleRule)
   │        └─► api.Jobs.Enqueue / RunArchive ──► internal/jobs.Runner
   ├── /api/ catch-all → 404 endpoint_not_found (JSON)
   └── /  SPA-kezelő (index.html no-cache; _app/immutable és fonts/ immutable; hiányzó _app/ fájl 404)

Háttér (jobs.Runner): archiválás → purge óránként; letöltési seprő percenként;
APP_FETCH_WORKERS munkás → fetch.Fetcher (SSRF-guard) → store.ApplyFetchResult
Adatkönyvtár: app.db (+ app.db-wal, app.db-shm), secret.key
```

### 3.2 Go-függőségi gráf (tényleges importok)

```
cmd/kinex-mark      -> internal/app, internal/config
internal/app        -> api, auth, clock, config, fetch, jobs, model, store, web
internal/api        -> auth, bookmarks, clock, config, fetch, model, golang.org/x/net/html   (store-t NEM importál)
internal/jobs       -> clock, fetch, model     (a fetch-ből csak a fetch.Result és fetch.Icon típus; store-t NEM, interfészen át)
internal/store      -> clock, model, modernc.org/sqlite, golang.org/x/text/unicode/norm
internal/auth       -> stdlib
internal/fetch      -> golang.org/x/net/html, html/charset   (belső csomagot NEM; a hibakódjai a model.FetchErr* másolatai)
internal/bookmarks  -> golang.org/x/net/html
internal/config, internal/model, internal/clock -> csak stdlib
internal/testutil   -> clock (csak _test.go importálja)
internal/storetest  -> clock, model, store (csak a store-on kívüli csomagok _test.go fájljai)
web                 -> embed (go:embed all:build)
```

Kulcsdöntés: az `api` és a `jobs` saját interfészen (`api.Store`, `api.Jobs`, `jobs.Store`, `jobs.Fetcher`) éri el a store-t és a letöltőt; az adattípusok és a sentinel hibák a `model` levélcsomagban vannak. Az összekötés és a fordítási idejű ellenőrzés az `internal/app/app.go`-ban (`_ api.Store = (*store.Store)(nil)` stb.). Csak az `internal/app` importálja a `web` csomagot; a `go vet ./...` és a `go test ./...` ezért `web/build`-et igényel (minden buildút előbb a felületet építi).

### 3.3 Életciklus (`internal/app`)

`Run`: adatkönyvtár (0750) → `auth.LoadOrCreateKeyring` (`APP_SECRET_KEY` vagy `secret.key`, 0600) → `store.Open` (migrációk) → kulcsellenőrzés (`KeyCheck`; eltérésnél a program nem indul) → első admin (`bootstrap.go`) → `jobs.Runner.Start` → `api.New` → HTTP(S) szerver (`server.go`; `server started`). Leállás (SIGTERM/Ctrl+C): `srv.Shutdown` 10 s → feladatok kontextusa lezár → `runner.Wait(5 s)` → `closeStore` (előbb `Ping` drain, utána `Close`). `CheckHealth` (`-healthcheck`): `GET http://127.0.0.1:<port>/healthz`. `RunAdminCommand` (`-unlock-user`, `-reset-password`): offline, HTTP nélkül.

## 4. Repository-térkép

```
.github/workflows/ci.yml, docker.yml; .github/dependabot.yml
ai/                        BASE-SYSTEM-PLAN.md (terv), BASE-SYSTEM-PROMPT.md, TECHNOLOGY.md, GRAPHICS.md (követelmények; nem módosulnak)
cmd/kinex-mark/main.go     belépési pont (-version, -healthcheck, config.Load, admin parancs, app.Run)
internal/                  api, app, auth, bookmarks, clock, config, fetch, jobs, model, store (+ migrations/), storetest, testutil
tools/                     covercheck/ (Go lefedettségi kapu), langcheck/ („csak angol forrás”), fonts/build_fonts.py
web/                       SvelteKit-projekt + embed.go, assets_test.go, hero_test.go
  scripts/sync-app-css.mjs  static/{favicon.svg,robots.txt,fonts/,images/hero.webp}
  src/{app.html,app.css,app.d.ts,kinex-mark.css}  src/lib/**  src/routes/**
  e2e/{support/,fixtures/,specs/}  playwright.config.ts  vite.config.ts  vitest.config.ts  components.json
graphics/                  generate_graphics.py, prompts.json, hero.png (mester)
manual/                    MANUAL.md (DOC1), images/01-…32-*.png (33 kép)
Dockerfile  compose.yaml  Makefile  build.ps1  .gitignore  .gitattributes  .dockerignore  go.mod  go.sum  LICENSE (MIT)
README.md  SUMMARY.md
```

Generált, nem verziókövetett (`.gitignore`): `dist/`, `data/`, `coverage.out`, `coverage.html`, `web/node_modules/`, `web/build/`, `web/.svelte-kit/`, `web/coverage/`, `web/test-results/`, `web/playwright-report/`, `web/e2e/.auth/`, `.playwright-mcp/`. A `go.sum` és a `web/package-lock.json` verziókövetendő. Sorvégek: `.gitattributes` (`* text=auto eol=lf`, `*.ps1` CRLF, képek, fontok és `*.db` binárisak).

## 5. Go-csomagleltár

Lefedettség: `go test -cover` (finisher, 2026-11-22). Tesztfájlok összesen: 71 `_test.go`, 476 felső szintű `Test…` függvény.

### `cmd/kinex-mark` (nincs teszt)
`main.go`: `var version = "dev"`; `config.Load(flag.CommandLine, os.Args[1:], os.Getenv)`; hibás konfiguráció: `invalid configuration`, kilépési kód 2; `-version`; `-healthcheck` → `app.CheckHealth`; admin parancs → `app.RunAdminCommand`; különben `app.Run`.

### `internal/config` (100%)
`config.go`: `Config` (Addr, DataDir, BaseURL, SecureCookies, TrustedProxies `[]netip.Prefix`, AllowPrivateFetch `*bool`, AllowLoopbackFetch, TLS, LogLevel/LogFormat, FetchWorkers/FetchTimeout, AdminUser/AdminPassword, SecretKey/SecretKeyFile, HealthCheck, ShowVersion, AdminCommand), `Load(fs, args, getenv)` (elsőbbség: kapcsoló, környezet, alapérték; minden érték validálva), `Config.EnvLockedSettings()`, `Env*` és `Default*` konstansok. Csak stdlib.

### `internal/clock` (100%)
`Clock` interfész (`Now()`), `Real` (UTC), `Fake` (`NewFake`, `Set`, `Advance`). Minden időbélyeg innen jön.

### `internal/model` (100%)
Levélcsomag: típusok, sentinel hibák és zárt listák, amelyekből a szerződés-golden készül. Fájlok: `audit.go` (`AuditEvents` 28 kód, `AuditDetailKeys`, `AuditDetailValues`), `errors.go` (`ErrNotFound`, `ErrNotOwner`, `DuplicateError` …), `export.go` (`MaxImportBytes` 2 MiB, `MaxImportEntries` 5000), `fetch.go` (`FetchErrors`, `FetchLease` 2 perc, `FetchMaxAttempts` 3, `FetchSweepBatch` 50, `FetchBackoff`), `link.go` (`LinkItem`, `LinkDetail`, `LinkQuery`, `LinkPage`, `MaxLinksPerUser` 10 000), `page.go`, `roles.go` (`Role.CanShare/CanListUsers/IsAdmin`, `Roles`, `Locales`), `session.go`, `settings.go` (`SystemSettings`, `DefaultSystemSettings`, tartományok), `share.go`, `tag.go` (`TagColors`, `MaxTagsPerUser` 500), `time.go` (`FormatTime`/`ParseTime`), `user.go` (`Viewer`, `UsernameKey`, `ValidUsername`, `DefaultPreferences`).

### `internal/auth` (97,4%; store-t nem importál)
- `password.go`: `Hasher` (`NewHasher(iterations, maxConcurrent)`, `Hash`, `Verify`, `DummyVerify`), PBKDF2-HMAC-SHA256 600 000 iteráció, párhuzamosság 4; `CheckPasswordPolicy` (12–128 rúna, nincs benne a felhasználónév, ≥ 6 különböző rúna, nem a jelenlegi); `GeneratePassword`.
- `totp.go`: RFC 6238 (SHA1, 6 jegy, 30 s, ±1 lépés, csak `lastStep` utáni lépés), `NewTOTPSecret`, `VerifyTOTP`, `TOTPURI`.
- `token.go`: `NewToken` (32 bájt, base64url), `HashToken` (SHA-256), `NewRecoveryCodes`, `NormalizeRecoveryCode`.
- `keyring.go`: `Keyring` (`LoadOrCreateKeyring`, `Seal`/`Open` AES-256-GCM, `MAC`, `CheckValue`).
- `throttle.go`: `ThrottleRule` (`Check`, `Reserve`, `Confirm`, `Refund`), `UserRule{5, 15m, 15m, 24h, true}` (duplázódó zárolás, max 24 óra), `IPRule{30, 15m, 15m, 15m, false}`, `LockoutResetAfter` 24 óra, `UserThrottleKey`, `IPThrottleKey` (IPv4 /32, IPv6 /64).

### `internal/fetch` (98,0%)
- `fetch.go`: `Fetcher` (`New(Options)`, `Fetch`), `Error` (kód + átmeneti jelző); transport `Proxy = nil`, `DisableKeepAlives`, `net.Dialer.Control` → `CheckIP`, max 5 átirányítás, HTML ≤ 1 MiB, ikon ≤ 64 KiB, `charset.NewReader`.
- `guard.go`: `Policy`, `CheckIP` (mindig tiltott, csak loopback-kapcsolóval és csak privát-kapcsolóval engedett tartományok; beágyazott IPv4-formák kibontása).
- `html.go`: tokenizáló cím- és ikonjelölt-kinyerés, `CleanText`.
- `icon.go`: ikonjelöltek sorrendje és ellenőrzése, `isSafeSVG`; `svg.go`: `svgStructureOK` (szigorú allow-list, E fázis).
- `urls.go`: `NormalizeURL` (csak http/https, port 80, 443 vagy 1024–65535, hitelesítő adat tilos, max 2048 bájt), `ExtractURL` (gyors rögzítés).

### `internal/bookmarks` (99,4%)
`jsonfmt.go`: `ParseJSON`/`WriteJSON` (`kinex-mark-links/1`, `MaxTagsPerEntry` 100); `netscape.go`: `ParseNetscape`/`WriteNetscape` (mappák címkévé lapítva). `Entry`, `ErrInvalid`.

### `internal/store` (88,9%)
`Open(ctx, dataDir, Options)`: DSN `file:<data>/app.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=secure_delete(1)`, a kapcsolat a `cancelSafeConnector`-on át, `SetMaxOpenConns(1)`, majd migráció (`schema_migrations`, `migrations/*.sql` névsorrendben, mindegyik egy tranzakcióban).

| Fájl | Fő metódusok |
|---|---|
| `store.go` | `Open`, `Close`, `Ping`, `withTx`, `migrate`, `querier` |
| `driver.go` | determinisztikus SQL-függvények: `kxm_lower` (Unicode lower) és `kxm_fold` (`foldText`: kisbetű, NFD, kombináló jelek nélkül, NFC; ékezetfüggetlen kereső- és rendezőkulcs) |
| `cancelsafe.go` | `cancelSafeConnector`, `cancelSafeConn`, `cancelSafeStmt` (18.1) |
| `users.go` | `CountUsers`, `CreateUser`, `UserByID`, `Credentials`, `AdminUsers`, `AdminUser`, `UpdateUser` (szerepkör-váltáskor a két láthatósági upsert), `SetPassword`, `RecordLogin`, `Directory`, `DeleteUser` + `truncateWAL`, `OwnSharedLinkCount` |
| `sessions.go` | `CreateSession`, `SessionByToken`, `TouchSession`, `Sessions`, `DeleteSession`, `DeleteUserSessions`, `CreateChallenge`, `ChallengeAttempt`, `DeleteChallenge` |
| `totp.go` | `SetPendingTOTP`, `PendingTOTP`, `EnableTOTP`, `DisableTOTP`, `AdvanceTOTPStep`, `ReplaceRecoveryCodes`, `UseRecoveryCode` |
| `throttle.go` | `ThrottleUpdate`, `ClearThrottle`, `ClearUserThrottle` |
| `link_query.go` | `visibleCTE`, `viewerCreatedAt`, `Links`, `LinkItem`, `Link` |
| `links.go` | `CreateLink`, `UpdateLink`, `DeleteLink`, `LinkIcon` |
| `link_state.go` | `SetPinned`, `SetArchived`, `RecordVisit`, `SetLinkTags` |
| `shares.go` | `Shares`, `CreateShare`, `DeleteShare` |
| `tags.go` | `Tags`, `CreateTag`, `UpdateTag`, `DeleteTag` |
| `fetchq.go` | `DueFetches`, `BeginFetch`, `ApplyFetchResult`, `ResetFetch` |
| `archive.go` | `ArchiveInactive`, `roleVisibilityUpsert`, `ownerSharesReturnUpsert` |
| `purge.go` | `Purge` (végén `PRAGMA optimize`) |
| `settings.go` | `Settings`, `UpdateSettings`, `BumpPrivacyNotice`, `KeyCheck`, `SetKeyCheck` |
| `consent.go`, `audit.go`, `gdpr.go`, `importexport.go`, `prefs.go` | hozzájárulás, audit, GDPR-export, link import/export, preferenciák |

Tesztsegédek: `store_test.go` (`newTestStore`, `seedUser`, `seedLink`, `seedTag`, `share`); kiemelt tesztek `TestDeleteUserLeavesNoRemnants` (`gdpr_test.go`), `TestSearchAndNameOrderIgnoreAccents` (`links_test.go`), `TestFoldKeysMigration` (`migrate_test.go`), `TestCancelledQueriesReleaseDatabaseFile` (`cancelsafe_test.go`), `TestArchiveInactiveOwnerSharesReturn` (`jobs_test.go`). Csomagon kívüli tesztekhez: `internal/storetest` (`seed.go`, `store.go`).

### `internal/jobs` (100%; store-t nem importál)
`interfaces.go` (`Store`, `Fetcher`, `Ticker`), `runner.go` (`Runner`: `New`, `Start`, `Wait`; sor 256, intervallum 1 óra, seprő 1 perc), `metaqueue.go` (`Enqueue`, seprő, munkások), `archive.go` (`RunArchive`), `purge.go` (`RunPurge`).

### `internal/api` (98,4%)
`New(Deps) (http.Handler, error)`; `Deps{Store, Jobs, Hasher, Keyring, Clock, Log, Config, Version, UI fs.FS}` (`deps.go`).

| Fájl | Felelősség |
|---|---|
| `api.go` | `server`, `newServer`, `handler()` (mux + globális lánc) |
| `routes.go` | `access` szintek, `route`, `routes()` (59 sor) |
| `session.go` | süti, `authenticate`, `requireReady`, `requireRole`, `guard` |
| `csrf.go` | `csrf`, `hostCheck` (`host_mismatch` 421) |
| `clientip.go` | `clientIP`, `requestScheme`, `requestHost` (`X-Forwarded-*` csak megbízható proxytól) |
| `security.go` | `buildCSP`, `secureMode`, `securityHeaders` |
| `middleware.go` | `recoverer`, `accessLog`, `limitBody` |
| `respond.go` | `writeJSON`, `writeError`, `writeValidation`, `writeFailure`, `readJSON` (`maxBodyBytes` 64 KiB, `DisallowUnknownFields`), `writeList` |
| `errors.go` | hibakódok, `errorSpecs`, `mapModelError` |
| `validate.go` | `checkText`, `checkUsername`, `parseID`/`pathID`, `newQuery` |
| `context.go`, `dto.go`, `spa.go` | kérés-kontextus, válaszalakok (`inactiveSince`, `archiveDueAt`), SPA-kiszolgálás + `inlineScriptHashes` |
| `h_*.go` | végpontkezelők területenként: `h_public`, `h_auth`, `h_me`, `h_twofa`, `h_privacy`, `h_links`, `h_links_state`, `h_icons`, `h_import`, `h_shares`, `h_tags`, `h_directory`, `h_admin_users`, `h_admin_settings`, `h_admin_audit`, `h_admin_jobs` |

Tesztkeret: `apitest_test.go` (`newEnv(t, opts…)`), `harness_test.go`, `contract_test.go` (`TestContractGolden`, `TestContractListsAreUnique`, `TestRouteTable`), `routes_test.go` (`TestRoutePermissionMatrix`), végpontonként `h_*_test.go`; `TestAdminUsersUpdateAuditSplit`.

### `internal/app` (94,1%)
`app.go` (`Run`, interfész-ellenőrzések, `closeStore`), `server.go`, `bootstrap.go` (első admin), `admincmd.go` (`RunAdminCommand`), `health.go` (`CheckHealth`), `logger.go`.

### `internal/testutil`, `web`, `tools`
- `internal/testutil`: `clock.go`, `origin.go` (hamis HTTP-forrás), `doc.go`; csak tesztek importálják.
- `web` (100%): `embed.go` (`//go:embed all:build`, `Files()`), `assets_test.go` (fontok mérete és hash-e, robots.txt), `hero_test.go` (`hero.webp` WebP, 1600×900).
- `tools/covercheck` (97,8%): Go-lefedettségi kapu (összesen ≥ 85%, csomagonként ≥ 80%, `testutil` és `storetest` kizárva). `tools/langcheck` (88,9%): ékezetes betűt keres a forrásban az i18n-katalógusokon, tesztadaton és fixture-ökön kívül. `tools/fonts/build_fonts.py`: a két WOFF2 reprodukálása.

## 6. Adatmodell

### 6.1 Általános szabályok
SQLite egy kapcsolattal (WAL), minden időbélyeg TEXT RFC 3339 UTC milliszekundummal (`model.FormatTime`), a `clock.Clock`-ból. Kis-nagybetű-független kulcsok: `username_key`, `name_key`. Ékezet- és kisbetű-független kulcsok (`foldText`, `kxm_fold`): `links.search_text` (URL, host, cím, leírás) és `links.sort_label`; a keresőszót is így alakítja a store, a címkenevet `kxm_fold(name_key)` illeszti. Tranzakció: `withTx`.

### 6.2 Migrációk (`internal/store/migrations/`)

| Fájl | Táblák |
|---|---|
| `0001_users.sql` | `users`, `user_preferences` |
| `0002_auth.sql` | `sessions`, `login_challenges`, `recovery_codes`, `login_throttle` |
| `0003_links.sql` | `icons`, `links`, `tags`, `link_tags`, `link_shares`, `link_user_state` |
| `0004_system.sql` | `app_meta`, `system_settings`, `audit_log`, `consent_records` |
| `0005_fold_keys.sql` | nincs új tábla: a meglévő `links.search_text` és `sort_label` átírása `kxm_fold`-dal (finisher) |

A `schema_migrations` táblát a `migrate` hozza létre.

**Új migráció menete:**
- Új fájl: `internal/store/migrations/NNNN_<név>.sql` (a következő szám `0006`); az `embed` automatikusan felveszi, a `schema_migrations.name` értéke `migrations/NNNN_<név>.sql`.
- A migrációk névsorrendben, fájlonként egy tranzakcióban futnak a `store.Open` alatt, csak előre (visszalépés nincs); közben a `foreign_keys(1)` be van kapcsolva.
- Frissítendő teszt: `internal/store/migrate_test.go` (a `want` névlista és a `TestOpenIsIdempotent` `n != 5` darabszáma), különben elbukik.
- SQLite-ban a CHECK-korlát csak táblaátépítéssel módosítható (új tábla, adatmásolás, átnevezés).

### 6.3 Táblák

| Tábla | Kulcsoszlopok és korlátok |
|---|---|
| `users` | `username_key` UNIQUE, `display_name`, `role` CHECK admin/developer/customer, `password_hash`, `must_change_password`, `is_active`, TOTP-oszlopok (`totp_secret`, `totp_enabled_at`, `totp_last_step`, `totp_pending_secret`, `totp_pending_expires_at`) |
| `user_preferences` | PK `user_id`; `locale` hu/en/de, `theme` dark/light/system, `view_mode` list/tiles, `show_*` kapcsolók, `sort` usage/created/title, `tag_match` any/all, `archive_after_days` 0–3650 vagy NULL |
| `sessions` | `token_hash` UNIQUE (SHA-256), `user_id`, `last_seen_at`, `expires_at`, `ip`, `user_agent` |
| `login_challenges`, `recovery_codes`, `login_throttle` | 2FA-kihívás; helyreállító kódok MAC-je; korlátozási kulcsok és zárolások |
| `icons` | `sha256` UNIQUE, tartalomtípus-lista, `data` max 64 KiB |
| `links` | UNIQUE(`owner_id`, `url`), `title_source` none/fetched/manual, `fetch_status` pending/ok/failed/blocked, `fetch_*` oszlopok, `search_text`, `sort_label` |
| `tags` | `name_key` tulajdonosonként UNIQUE, 7 szín |
| `link_tags` | PK(`link_id`, `tag_id`); a címke mindig a nézőé |
| `link_shares` | `target_type` all/role/user CHECK-kel, egyedi index |
| `link_user_state` | PK(`user_id`, `link_id`); `pinned`, `use_count`, `last_used_at`, `baseline_at`, `archived_at`, `archive_reason` auto/manual; CHECK: kitűzött nem lehet archivált |
| `app_meta`, `system_settings` (JSON érték), `audit_log`, `consent_records` | rendszer, beállítások, audit, tájékoztató-nyugtázás |

Korlátok: 10 000 saját link, 500 címke felhasználónként, import max 5000 bejegyzés és 2 MiB, ikon max 64 KiB.

### 6.4 Láthatóság (`visibleCTE`)
Egy felhasználó látja (1) a saját linkjeit, (2) más tulajdonos linkjét, ha van rá `all`, a szerepkörére szóló `role` vagy neki szóló `user` megosztás, és (3) a megosztás csak akkor él, ha a tulajdonos **jelenlegi** szerepköre `admin` vagy `developer`. Nem látható vagy nem létező link: `ErrNotFound` (IDOR = 404). Látható, de idegen link tulajdonosi műveletnél: `ErrNotOwner` (403 `not_owner`). Az admin sem lát idegen linket.

### 6.5 Rendezés
`pinned DESC`, majd `usage` esetén `use_count DESC`, `last_used_at DESC NULLS LAST`; `title` esetén `sort_label` (ékezet- és kisbetű-független; az ö/ő az o-val együtt rendeződik, nem a magyar ábécé szerint); végül `created_at DESC`, `id DESC`. Hiányzó állapotsor: nem kitűzött, 0 használat; a `baseline_at` tartaléka saját linknél `created_at`, megosztottnál `max(shared_at, viewerCreatedAt)`.

Megvalósítás: a szerveren az `internal/store/link_query.go` `linkOrder` konstansa (ORDER BY), a kliensen a `web/src/lib/links/order.ts` (`compareLinks`, `sortLinks`, `sortLabel`; tesztje `order.spec.ts`), amely a helyben módosított listát rendezi újra. A két oldalnak egyeznie kell: ha az egyik változik, a másikat is módosítani kell.

### 6.6 Személyes linkállapot (`link_state.go`)
Minden írás upsert a láthatóság ellenőrzése után: `RecordVisit` (`use_count+1`, `last_used_at`, archiválás törlése), `SetPinned`, `SetArchived`. A megnyitás és a másolás is `visit`. A `SetPinned` (kitűzés és levétel egyaránt, `pinUpsert`) és a visszaállítás (`SetArchived(false)`, `restoreUpsert`) a `baseline_at`-ot a jelen időre állítja, vagyis újraindítja az inaktivitási időt. A kitűzés megszünteti az archivált állapotot, a kézi archiválás leveszi a kitűzést.

### 6.7 Archiválási szabály (`ArchiveInactive`)
Archiválódik (`archive_reason='auto'`), ha nincs kitűzve és `max(baseline_at, coalesce(last_used_at, baseline_at)) < now - days` (szigorú egyenlőtlenség), ahol `days` a felhasználó `archive_after_days` értéke, különben a `defaultArchiveDays` rendszerbeállítás (90); 0 = soha. Óránként fut, csak aktív felhasználókra. Szerepkör-váltáskor (`UpdateUser`) a `roleVisibilityUpsert` és az `ownerSharesReturnUpsert` előre mozdítja a `baseline_at`-ot. Az importált link inaktivitása az import idejétől számít.

## 7. API

### 7.1 Közös szabályok
- Alap `/api`, azonos origin, nincs CORS; JSON, `Cache-Control: no-store` (kivétel az ikon). Mezők camelCase-ben; időbélyeg RFC 3339 UTC.
- Kérés: 64 KiB, `DisallowUnknownFields`; import 2 MiB (413 `payload_too_large`). Nem biztonságos metódusnál kötelező `X-Kinex-Request: 1`.
- Hibatörzs: `{"error", "code", "fields", "details"}`; 429-nél `Retry-After`.
- Lapozás: `limit`, `offset` (max 1 000 000); alap/max: linkek 200/500, admin felhasználók és audit 50/200.

### 7.2 Hibakódok
40 szerverkód (`internal/api/errors.go` `errorSpecs`; a 40. a finisher `import_too_many_entries` 400-as kódja), köztük `not_implemented` 501 (maradvány, egy útvonal sem adja) és `host_mismatch` 421; kliensoldali: `network_error`. 16 mezőkód: `required`, `too_short`, `too_long`, `invalid`, `invalid_format`, `not_allowed`, `out_of_range`, `unknown`, `duplicate`, `credentials_in_url`, `incorrect`, `password_too_short`, `password_too_long`, `password_contains_username`, `password_too_simple`, `password_reused`.

Szerződés-golden: `internal/api/testdata/contract.json` (errorCodes 40, fieldCodes 16, auditEvents 28, auditDetailKeys 15, auditDetailValues 5, fetchErrors 10, tagColors 7, roles 3, locales 3). Frissítés: `go test ./internal/api -run TestContractGolden -update`. Frontend-ellenőrzés: `web/src/lib/i18n/contract.spec.ts`.

### 7.3 Végpontok (`internal/api/routes.go`, 59 sor)
Szintek: **N** nyilvános; **K** bejelentkezett, mentes a kapuktól; **B** bármely bejelentkezett, kapukkal; **D** admin és developer (megosztáson `share_not_allowed`, könyvtáron `forbidden`); **A** csak admin (`forbidden`).

| Csoport | Metódus és út | Szint |
|---|---|---|
| nyilvános | GET `/healthz`, GET `/api/public/config` | N |
| auth | POST `/api/auth/login`, POST `/api/auth/login/totp`, POST `/api/auth/logout` | N |
| auth | GET `/api/auth/me`, POST `/api/auth/password` | K |
| me | PATCH `/api/me/profile` | B |
| me | GET, PATCH `/api/me/preferences` | K |
| me | GET `/api/me/sessions`, DELETE `/api/me/sessions/{id}`, POST `/api/me/sessions/revoke-others` | B |
| me | POST `/api/me/2fa/setup`, `/enable`, `/disable`, `/recovery-codes` | B |
| me | GET `/api/me/privacy`, POST `/api/me/privacy/consent`, GET `/api/me/export`, POST `/api/me/delete` | K |
| linkek | GET, POST `/api/links`; POST `/api/links/quick`; GET `/api/links/export`; POST `/api/links/import` | B |
| linkek | GET, PATCH, DELETE `/api/links/{id}`; POST `/refresh`; PUT `/tags`; POST/DELETE `/pin`, `/archive`; POST `/visit`; GET `/icon` | B |
| megosztás | GET, POST `/api/links/{id}/shares`; DELETE `/api/links/{id}/shares/{shareId}` | D |
| címkék | GET, POST `/api/tags`; PATCH, DELETE `/api/tags/{id}` | B |
| könyvtár | GET `/api/users/directory` | D |
| admin | GET, POST `/api/admin/users`; GET, PATCH, DELETE `/api/admin/users/{id}`; POST `/reset-password`, `/reset-2fa`, `/unlock`; DELETE `/sessions` | A |
| admin | GET, PATCH `/api/admin/settings`; POST `/api/admin/settings/privacy-notice/bump`; GET `/api/admin/audit`; POST `/api/admin/jobs/archive` | A |

Admin lépcsős megerősítés (`currentPassword`): szerepkör- és állapotváltás, törlés, jelszó- és 2FA-visszaállítás.

## 8. Biztonsági modell és GDPR

### 8.1 Munkamenet
- 32 bájtos token; a DB-ben csak a SHA-256-ja van. Süti: `__Host-kxm_session` biztonságos módban, különben `kxm_session`; HttpOnly, SameSite=Strict.
- Lejárat: tétlenség 720 perc, abszolút 168 óra (rendszerbeállítás). `last_seen_at` legfeljebb percenként frissül.
- Kapuk (`requireReady`): előbb `password_change_required`, utána `consent_required`.

### 8.2 CSRF és Host-ellenőrzés
- CSRF: kötelező fejléc, majd `Origin`, ennek hiányában `Sec-Fetch-Site`.
- `hostCheck`:
  - `APP_BASE_URL` mellett a hostnak egyeznie kell;
  - anélkül, loopback figyelésnél csak loopback név fogadható el;
  - nem loopback figyelésnél `APP_BASE_URL` nélkül nincs host-ellenőrzés.

### 8.3 Jelszó, korlátozás, 2FA
- Jelszó: PBKDF2-HMAC-SHA256, 600 000 iteráció.
- Korlátozás:
  - felhasználói kulcs: 5 hiba / 15 perc, utána duplázódó zárolás, legfeljebb 24 óra;
  - IP-kulcs: 30 hiba / 15 perc; IPv6-nál /64;
  - minden ellenőrzés előtt foglalás (`Reserve`).
- TOTP:
  - RFC 6238, ±1 lépés; visszajátszás ellen a `totp_last_step`;
  - a titok AES-256-GCM-mel zárva;
  - belépési kihívás: 5 perc, legfeljebb 5 próbálkozás.
- Helyreállító kódok: 10 darab, 10 karakteresek, MAC-ként tárolva.

### 8.4 SSRF-guard
- `CheckIP` a tárcsázáskor fut. Engedett portok: 80, 443 és 1024 fölött.
- Mindig tiltott: a `0.0.0.0/8`, a multicast (`224.0.0.0/4`, `ff00::/8`), a fenntartott `240.0.0.0/4`, a felhő-metaadat címek és a dokumentációs tartományok.
- A „loopback” osztály (`127.0.0.0/8`, `::1`, valamint a link-local `169.254.0.0/16` és `fe80::/10`) csak `APP_ALLOW_LOOPBACK_FETCH` mellett, a privát tartomány (`10/8`, `172.16/12`, `192.168/16`, `100.64/10`, `fc00::/7`) csak `allowPrivateFetch` mellett érhető el.
- A `fetch.Policy`-t az `internal/app/app.go` `fetchPolicy` építi, és minden tárcsázáskor újraolvassa. Az `APP_ALLOW_PRIVATE_FETCH` (ha meg van adva) felülírja és lezárja az `allowPrivateFetch` rendszerbeállítást; ha a beállítás nem olvasható, a privát címek tiltva maradnak.
- Proxy nincs, keep-alive nincs. Legfeljebb 5 átirányítás; HTML legfeljebb 1 MiB, ikon legfeljebb 64 KiB.
- Az elutasított SVG helyett a raszteres jelölt kerül be.

### 8.5 Fejlécek és CSP
CSP: `default-src 'self'; script-src 'self' <hash-ek>; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; manifest-src 'self'; worker-src 'self'`, biztonságos módban kiegészítve: `upgrade-insecure-requests`.

További fejlécek:
- `nosniff`, `no-referrer`, `DENY`;
- COOP és CORP: `same-origin`;
- Permissions-Policy, X-Robots-Tag;
- HSTS (biztonságos módban).

Az ikonvégpont saját CSP-je: `default-src 'none'; style-src 'unsafe-inline'; sandbox`.

### 8.6 Audit
28 esemény (`model.AuditEvents`), a `details` zárt kulcskészletből (15 kulcs; az `active` fenntartott, nem íródik). Jelszó, token, kód, URL soha nem kerül az auditba.

### 8.7 GDPR
- Tájékoztató: verziózott nyugtázás (`consent_records`); ez nem hozzájárulás.
- Adatexport: `kinex-mark-export/1`.
- Fióktörlés:
  - kaszkád törlés, az audit pszeudonimizálása (`deleted-user-<id>`);
  - `secure_delete(1)`, utána best-effort `wal_checkpoint(TRUNCATE)` (`truncateWAL`);
  - a teszt: `TestDeleteUserLeavesNoRemnants`.
- Sütik és tároló: csak a munkamenet-süti. `localStorage` kulcsok: `kxm-locale`, `mode-watcher-mode`, `mode-watcher-theme`.
- Az audit megőrzési ideje 180 nap.

## 9. Háttérfolyamatok

| Feladat | Futás | Kód |
|---|---|---|
| Archiválás, majd purge | indulás után és óránként | `internal/jobs/archive.go`, `purge.go` |
| Letöltési seprő | indulás után és percenként (`DueFetches(50)`) | `internal/jobs/metaqueue.go` |
| Letöltő munkások | folyamatosan, `APP_FETCH_WORKERS` darab | `metaqueue.go` → `fetch.Fetcher` → `ApplyFetchResult` |

- Sor: kapacitás 256; bérlet 2 perc; `FetchMaxAttempts` 3; visszalépés 1, majd 10 perc.
- A purge törli: munkameneteket, kihívásokat, korlátozási sorokat, függő TOTP-titkokat, régi auditot, gazdátlan ikonokat. Végén `PRAGMA optimize`.
- Admin: a `POST /api/admin/jobs/archive` azonnal futtatja az archiválást.

## 10. Frontend-leltár

Projekt: `web/` (`kinex-mark-web`).
- SvelteKit 3 és Svelte 5 runes; adapter-static `fallback: 'index.html'`, `ssr = false`.
- A konfiguráció a `web/vite.config.ts`-ben van; `svelte.config.js` nincs.
- A saját kód `$lib`-et használ; a `#lib` import-map a `package.json`-ban megmaradt.

### 10.1 Útvonalak és kapuk (`web/src/routes`)

| Csoport | Útvonalak |
|---|---|
| `(public)` | `/login`, `/privacy` |
| `(gate)` | `/change-password`, `/consent` |
| `(app)` | `/`, `/tags`, `/settings/{profile,security,display,privacy}`, `/admin/{users,settings,audit}`, `[...rest]` (404 a shellen belül) |

Kapuk:
- `(app)/+layout.ts`: `session.ensure()`; bejelentkezés nélkül → `/login`; nyitott kapu esetén → `/change-password` vagy `/consent`.
- `(gate)/+layout.ts`: nyitott kapu nélkül → `/`.
- `(app)/admin/+layout.ts`: nem admin → 403.
- `(public)/login/+page.ts`: a bejelentkezett felhasználót átirányítja (`signedInTarget`, a `next` ellenőrzése `safeNext`-tel), a `reason` paramétert a `parseLoginReason` értelmezi; mindhárom a `features/auth/redirect.ts`-ben van.

### 10.2 `lib/api`
- `client.ts`: `request<T>`, `X-Kinex-Request: 1`, `onAuthFailure`.
- `errors.ts`: `ErrorCode` és `FieldCode` unió, `ApiError`.
- `types.ts`: a szerződés típusai.
- Modulok: `public`, `auth`, `me`, `links`, `tags`, `directory`, `admin`.
- A megfeleltetést az `endpoints.spec.ts` ellenőrzi.

### 10.3 i18n (`lib/i18n`)
- Fájlok: `types.ts` (`Locale`, `LOCALES`, `isLocale`), `runtime.svelte.ts` (`i18n`, `t`, `hasMessage`, `tDynamic`, `fmt`, `catalogue`, `catalogues: Record<Locale, Catalogue>`), `format.ts` (`NUMBER_LOCALE`), `detect.ts` (`LOCALE_STORAGE_KEY 'kxm-locale'`), `plural.ts`, `errors.ts`.
- Katalógusok: `messages/{hu,en,de}/`, 18 névtér: `common`, `nav`, `errors`, `fields`, `roles`, `status`, `auth`, `landing`, `privacy`, `links`, `capture`, `fetch`, `tags`, `tagColors`, `settings`, `admin`, `audit`, `security`.
- A típusforrás a `hu`; az `en` és a `de` `satisfies Shape<typeof hu>`.

**Új kulcs:** vedd fel a `hu/<névtér>.ts`-be, majd az `en/` és a `de/` megfelelő fájljába. Hívás `t('névtér.kulcs')`-csal; dinamikus kulcshoz `hasMessage`/`tDynamic`. A `parity.spec.ts`, a `contract.spec.ts` és a `conventions.spec.ts` ellenőrzi.

**Új nyelv:**
- frontend:
  - `Locale` és `LOCALES`, valamint az `isLocale` a `types.ts`-ben (kézzel felsorolt `'hu' || 'en' || 'de'`, nem a `LOCALES`-ből számol; a `detect.ts` erre épül);
  - `messages/<új>/` mind a 18 névtérrel és a névtereket összefogó `messages/<új>/index.ts`-szel, bejegyzés a `catalogues`-ba;
  - `format.ts`: `NUMBER_LOCALE` és a `locale === 'hu'` ágak (csoportosítás, elválasztó, relatív idő stílusa);
  - a `nav.locale.<új>` kulcs minden nyelven, `api/types.ts` `Locale`;
  - `parity.spec.ts`: a kézzel felsorolt `['en', 'de']` listába fel kell venni, különben az új nyelvet nem ellenőrzi;
  - a `features/admin/UserDialog.svelte` alapértelmezett nyelve `'hu'` (nem kell módosítani, de tudni kell róla).
- backend:
  - `model.Locales` (`internal/model/roles.go`);
  - új migráció a `user_preferences.locale` CHECK-korlátjára (`0001_users.sql`: `CHECK (locale IN ('hu','en','de'))`), táblaátépítéssel (6.2);
  - `internal/fetch/fetch.go` `acceptLanguageHeader`: új `case`, különben az új nyelv figyelmeztetés nélkül `en`-re esik vissza;
  - golden `-update` (a `contract.json` `locales` listája).

### 10.4 Állapottárolók (`lib/state`)

| Tároló | Fő tartalom |
|---|---|
| `session` | `ensure`, `login`, `logout`, `refresh`, `setMe`, `clear`, `can`, `handleAuthFailure`, `gate` |
| `links` | `LinksStore`: `PAGE_SIZE` 200, keresési debounce 200 ms, függőség-lekérdezés 1500 ms-onként, legfeljebb 20 s |
| `tags` | `FREQUENT_TAG_COUNT` 8 |
| `preferences` | 400 ms-os debounce |
| `publicConfig` | nyilvános konfiguráció |
| `linkStats` | 60 s-onként frissül |

### 10.5 Komponensek és design-szabályok
- `components/ui/**`: shadcn-svelte által generált komponensek, kézzel nem szerkesztjük. Kivétel a három overlay-fájl (dialog, alert-dialog, sheet).
- `components/app/`: közös réteg, például `Button.svelte` (burkoló), `SidebarFrame.svelte` (`sidebar_state` süti nélkül), `styles.ts`, `AccountUsername.svelte`.
- `styles.ts` exportjai: `buttonSize`, `destructiveButton`, `invalidField`, `inputSize`, `cardSurface`, `lineTabsList`, `lineTabsTrigger`, `tableHead`, `tableCell`, `dialogBox`, `dialogFooter`, `toggleItem`, `checkboxBox`, `typeHeading3`, `typeBodySm`, `typeEyebrow`, `buttonTone`, `tooltipSurface`.
- Kéréshiba-toast csak `utils/toast-error.ts` `toastError`-ral.
- `features/`: `auth`, `privacy`, `links`, `tags`, `settings`, `admin`.
- `app.css`: csak generált (`web/scripts/sync-app-css.mjs`), az `app-css.spec.ts` ellenőrzi.
- `kinex-mark.css`: zárt szabálylista.
- `conventions.spec.ts`: az 1–17. szabályt kényszeríti.

## 11. Új funkció végponttól képernyőig (ellenőrzőlista) és konvenciók

### 11.1 Lépések
1. Model: típus vagy konstans az `internal/model`-ben.
2. Séma: ha kell, új `internal/store/migrations/0006_<név>.sql`, a `migrate_test.go` frissítésével (menete: 6.2).
3. Store: metódus, láthatóságfüggő lekérdezésnél `visibleCTE`-vel.
4. Interfész: `api.Store` (`internal/api/deps.go`).
5. Hibakód:
   - konstans és `errorSpecs` sor (`internal/api/errors.go`);
   - modellhibához `mapModelError`;
   - a `TestErrorSpecsAreConsistent` darabszáma (`internal/api/respond_test.go`, jelenleg 40);
   - a kódot legalább egy `internal/api/*_test.go` fájlnak hivatkoznia kell a konstansával vagy a literáljával (nem számít az `errors_test.go` és a `respond_test.go`), különben a `TestErrorCodesExercised` (`internal/api/errors_test.go`) elbukik; a `TestErrorCodeConstants` a konstansok és a táblasorok egyezését ellenőrzi;
   - a webes oldalon a `contract.spec.ts` megköveteli, hogy az `ErrorCode` unió a golden listája plusz `network_error` legyen, és minden kódhoz legyen `errors.<code>.title` és `.text` mindhárom nyelven.
6. Kezelő: `internal/api/h_<terület>.go`.
7. Útvonal:
   - új sor a `routes.go`-ban;
   - `TestRouteTable` darabszáma (`want 59`);
   - a `TestRoutePermissionMatrix` automatikusan lefedi.
8. Golden: `go test ./internal/api -run TestContractGolden -update`.
9. Go-teszt: `h_<terület>_test.go`, `newEnv`-vel.
10. Kliens:
    - `web/src/lib/api/types.ts`;
    - modul a `lib/api`-ban;
    - `ErrorCode` unió (`errors.ts`);
    - `endpoints.spec.ts`.
11. i18n: `errors.<code>.{title,text}` és az új kulcsok mindhárom nyelven.
12. Képernyő: tároló, komponens, útvonal. Új adminoldal példáján:
    - útvonal: `web/src/routes/(app)/admin/<név>/+page.svelte` `PageTitle`-lel (az `admin/+layout.ts` kapuja öröklődik);
    - képernyő: `web/src/lib/features/admin/<Név>Screen.svelte` és mellette `.svelte.spec.ts`;
    - oldalsáv: az `admin` tömb a `web/src/lib/components/app/AppSidebar.svelte`-ben (`href`, `nav.admin<Név>` felirat, ikon);
    - morzsamenü és oldalcím: `TRAILS` a `web/src/lib/components/app/breadcrumbs.ts`-ben (a `widgets.spec.ts` és a `shell.spec.ts` felsorolja az adminlinkeket és a nyomvonalakat);
    - saját i18n-névtér: `hu/<ns>.ts`, `en/<ns>.ts`, `de/<ns>.ts` (az utóbbi kettő `satisfies Shape<typeof hu>`), bejegyzés mindhárom `messages/*/index.ts`-be, és a `nav.admin<Név>` kulcs mindhárom `nav.ts`-ben;
    - E2E-minta: `web/e2e/specs/admin-audit.spec.ts`.
13. Vitest; küszöb 85/80/85/85.
14. E2E: `web/e2e/specs/`.
15. Kapu: `.\build.ps1 verify` vagy `make verify`.

### 11.2 Konvenciók
- A forráskód csak angol; felületi szöveg csak a `messages/**`-ban lehet.
- Minden idő a `clock.Clock`-ból jön.
- A `components/ui/**` generált, kézzel nem szerkesztjük.
- Portok:
  - fejlesztés: 8080, Vite: 5173;
  - E2E: 18080, fixture: 18081.

## 12. Konfiguráció

Elsőbbség: kapcsoló, környezeti változó, alapérték.

| Kapcsoló | Környezeti változó | Alapérték |
|---|---|---|
| `-addr` | `APP_ADDR` | `:8080` |
| `-data` | `APP_DATA_DIR` | `./data` |
| `-base-url` | `APP_BASE_URL` | üres |
| `-secure-cookies` | `APP_SECURE_COOKIES` | `auto` |
| `-trusted-proxy` | `APP_TRUSTED_PROXY` | üres (IP-cím vagy CIDR, vesszővel több is; ajánlott egyetlen cím egy dedikált hálózatban) |
| `-allow-private-fetch` | `APP_ALLOW_PRIVATE_FETCH` | üres |
| `-allow-loopback-fetch` | `APP_ALLOW_LOOPBACK_FETCH` | `false` |
| `-tls-cert`, `-tls-key` | `APP_TLS_CERT`, `APP_TLS_KEY` | üres |
| `-log-level`, `-log-format` | `APP_LOG_LEVEL`, `APP_LOG_FORMAT` | `info`, `text` |
| `-fetch-workers`, `-fetch-timeout` | `APP_FETCH_WORKERS`, `APP_FETCH_TIMEOUT` | `2` (1–16), `10s` (1s–60s) |
| `-admin-user` | `APP_ADMIN_USER` | `admin` |
| – | `APP_ADMIN_PASSWORD`, `APP_SECRET_KEY` | üres |
| `-secret-key-file` | `APP_SECRET_KEY_FILE` | `<data>/secret.key` |
| `-healthcheck`, `-version`, `-unlock-user`, `-reset-password` | – | egyszeri parancsok |

Rendszerbeállítások:

| Beállítás | Alapérték | Tartomány |
|---|---|---|
| `defaultArchiveDays` | 90 | 0–3650 |
| `auditRetentionDays` | 180 | 30–3650 |
| `sessionIdleMinutes` | 720 | 15–10 080 |
| `sessionAbsoluteHours` | 168 | 1–2160 |
| `allowPrivateFetch` | false | – |
| `privacyNoticeVersion` | 1 | – |

Ezen felül megadhatók az adatkezelő adatai.

## 13. Parancsok (Windows és Linux)

| Feladat | Windows | Linux |
|---|---|---|
| Build | `.\build.ps1` | `make build` |
| Fejlesztői mód | `.\build.ps1 dev` | két terminál |
| Keresztfordítás | `.\build.ps1 cross -Version 0.1.0` | `make cross VERSION=0.1.0` |
| Ellenőrzés | `.\build.ps1 check` | `make check` |
| Tesztek | `.\build.ps1 test-go`, `test-web`, `test-e2e` | `make test-go`, `test-web`, `test-e2e` |
| Teljes kapu | `.\build.ps1 verify` | `make verify` |
| Image | `.\build.ps1 image` | `make image` |
| Golden | `go test ./internal/api -run TestContractGolden -update` | ugyanaz |
| app.css | `cd web; npm.cmd run sync:css` | `cd web && npm run sync:css` |
| Hero (csak a hiányzó képet generálja és újrakézbesít; újragenerálás: 16. fejezet) | `.\build.ps1 graphics` | `make graphics` |
| Demóadatok | `cd web; npm.cmd run seed:demo -- --base http://127.0.0.1:8080 --password <admin jelszó>` | `cd web && npm run seed:demo -- --base http://127.0.0.1:8080 --password <admin jelszó>` |
| E2E-böngésző | `.\build.ps1 e2e-install` | `make e2e-install` |

## 14. Tesztállapot és lefedettség

### 14.1 Számok (finisher, 2026-11-22)

| Szint | Eredmény |
|---|---|
| Go unit + HTTP | minden csomag ok; 71 `_test.go` fájl, 476 felső szintű tesztfüggvény (sok táblázatos altesztel) |
| Go-lefedettség (covercheck, összevont atomic profil; kapu: összesen ≥ 85%, csomagonként ≥ 80%) | **95,3%** összesen |
| Vitest (unit + komponens, jsdom) | **73 fájl, 1107 teszt** passed |
| Vitest-lefedettség (kapu 85/80/85/85) | Statements 98,69% (9247/9369), Branches 91,1% (3134/3440), Functions 99,06% (3070/3099), Lines 99,62% (6176/6199) |
| E2E (Playwright 1.65, valódi bináris, Chromium) | **81/81** passed (chromium 71, mobile 4, chromium-last 6), flaky és retry nélkül; `fixme`/`skip` nincs. Korábban: E-close 4 futás, FV 2 futás, mind 81/81 |
| Statikus | svelte-check 1572 fájl, 0 hiba, 0 figyelmeztetés; prettier tiszta; gofmt és `go vet` tiszta; langcheck 549 fájl, nincs nem angol karakter |
| govulncheck | 0 találat (FV, v1.10.0) |
| npm audit (`--omit=dev`) | 0 sebezhetőség (FV) |
| Race | Dockerben (`golang:1.27`, go1.27.3) `go test -race -count=1 ./internal/...` exit 0 (FV, majd 2026-11-22-én a végleges kódon megismételve: exit 0); helyben nincs cgo, a CI `go` jobja minden pushnál futtatja |

Go-lefedettség csomagonként:

| Csomag | covercheck (a kapu ezt méri) | `go test -cover` |
|---|---|---|
| `internal/api` | 98,3% | 98,4% |
| `internal/app` | 94,1% | 94,1% |
| `internal/auth` | 97,3% | 97,4% |
| `internal/bookmarks` | 99,3% | 99,4% |
| `internal/clock` | 100% | 100% |
| `internal/config` | 100% | 100% |
| `internal/fetch` | 97,9% | 98,0% |
| `internal/jobs` | 100% | 100% |
| `internal/model` | 100% | 100% |
| `internal/store` | 88,9% | 88,9% |
| `internal/storetest` | kizárva | 84,8% |
| `tools/covercheck`, `tools/langcheck`, `web` | – | 97,8%, 88,9%, 100% |
| **összesen** | **95,3%** | – |

`cmd/kinex-mark` és `internal/testutil`: nincs tesztfájl (a kapun kívül); a `storetest` a covercheckből kizárt segédcsomag.

Előzmények: az FV (2026-11-21, a futások 2026-11-22 00:20-kor értek véget) 95,3%, 72 fájl / 1102 Vitest-teszt és kétszer 81/81 E2E eredményt mért (Vitest-lefedettség 98,69 / 91,1 / 99,06 / 99,62%). A finisher két új Go-tesztfüggvényt és több új esetet (store, api), egy új Vitest-fájlt (`fold.spec.ts`) és új eseteket (`order.spec.ts`, `query.spec.ts`, `ImportExportMenu.svelte.spec.ts`) adott hozzá (17.4).

### 14.2 Tesztkészletek
- **Go:** csomagonként (5. fejezet). Kiemelt regressziók: `TestRoutePermissionMatrix` (minden útvonal × anonim + 3 szerepkör), `TestContractGolden`, `TestErrorSpecsAreConsistent` (40 kód), `TestCancelledQueriesReleaseDatabaseFile`, `TestDeleteUserLeavesNoRemnants`, `TestArchiveInactiveOwnerSharesReturn`, `TestSearchAndNameOrderIgnoreAccents` és `TestFoldKeysMigration` (ékezetfüggetlen keresés és rendezés), `TestImportErrors` (`import_too_many_entries` JSON-ra és Netscape-re), `internal/fetch/svg_test.go` (37 megkerülési minta elutasítva, 10 valódi ikon elfogadva), `TestHeroImage`.
- **Vitest:** `web/vitest.config.ts`, beállítás `src/lib/test/setup.ts`; segédek `render.ts`, `mock-api.ts` (`mockApi`, `json`), `fixtures.ts`, `app-state.svelte.ts`, `text.ts`; keresztvizsgálatok `i18n/parity.spec.ts`, `i18n/contract.spec.ts`, `test/conventions.spec.ts`, `test/app-css.spec.ts`, `test/kinex-mark-css.spec.ts`.
- **E2E:** `web/playwright.config.ts` (1 worker, `fullyParallel: false`, CI-ban 1 retry, `hu-HU`, `Europe/Budapest`, sötét séma, 1440×900; projektek: `chromium`, `mobile` = Pixel 7 csak a `responsive.spec.ts`-hez, `chromium-last` a `*.last.spec.ts`-hez). Harness: `e2e/support/start-server.mjs` (bináris `KINEX_MARK_BIN`, ideiglenes adatkönyvtár, fixture-forrás `fixture-origin.mjs` a 18081-en), `global-setup.ts` (admin, dev1, cust1, cust2, `e2e/.auth/*.json`), `global-teardown.ts`, `fixtures.ts` (konzolőr: CSP-sértés, `console.error`, váratlan 4xx/5xx → bukás), `api.ts`, `i18n.ts` (`msg`), `helpers.ts`, `totp.ts`, `cleanup.mjs`, `constants.mjs`, demóadat-szkript `seed-demo.mjs` (`web/e2e/fixtures/demo-data.json`).
- **E2E-specifikációk** (`web/e2e/specs/*.spec.ts`, 81 teszt): `a11y` (axe), `admin-audit`, `admin-settings.last`, `admin-users`, `auth`, `gates.last`, `i18n`, `links-archive`, `links-capture`, `links-copy`, `links-detail`, `links-filter`, `links-import-export`, `links-ranking`, `links-view`, `privacy`, `responsive` (mobile), `security`, `settings`, `sharing`, `smoke`, `tags`, `theme`, `twofa`; közös segéd `e2e/specs/lib/app.ts`.
- **Feltáró GUI-teszt (Playwright MCP, KV15):** a C, D és E fázisban; minden jelentett hiba javítva és böngészőben újraellenőrizve. A terv által megnevezett `gui\findings.md` **nem készült el** (a harness az ügynököknek nem engedett jelentésfájlt írni). A tényleges bizonyítékok: a `handoff\E-close.md` megállapítás-táblája (állapot és böngészős bizonyíték soronként), a `handoff\C-close.md` és `handoff\D-close.md` GUI-szakaszai, a `gui\fix-frontend.md`, `fix-backend.md`, `fix-infra.md` javítójegyzetek, a képernyőképek a `gui-c\`, `gui-cclose\`, `dclose\`, `gui-e2\`, `gui-e3\`, `e-close\shots\` és `finish\playwright-mcp\` mappákban (mind a `C:\Users\kasza\AppData\Local\Temp\kinex-mark-work\` alatt; ideiglenes), valamint a tartós `manual/images/*.png` (33 kép, mind MCP-vel készült).

## 15. CI/CD és bizonyíték

### 15.1 `ci.yml`
Eseményei: push `main`-re és `v*` tagre, pull request, kézi indítás. `GO_VERSION 1.27.x`, `NODE_VERSION 26`, `GOTOOLCHAIN: local`. Jobok:
- `web`: `npm ci`, `lint`, `check`, `test:coverage` (összefoglaló a step summaryba), `build`, artifact `web-build` és `web-coverage`.
- `go` (mátrix `1.26.x`, `1.27.x`): gofmt, `go mod verify`, `go vet`, langcheck, `go test -race -timeout 15m` lefedettséggel; 1.27.x-en covercheck (`-markdown`), govulncheck, `coverage.html`.
- `go-windows`: `go test ./...` Windowson.
- `e2e`: bináris build, `npx playwright install --with-deps chromium`, `npm run test:e2e` (`CI=true`), hibánál riport-artifact.
- `binaries`: 4 platform keresztfordítása, artifact `binaries`.
- `image`: csak push eseményen, `needs: [web, go, go-windows, e2e]`, a `docker.yml` hívása `push: true`-val.

### 15.2 `docker.yml` és `dependabot.yml`
- `docker.yml`: újrahívható (`workflow_call`), PR-en is fut: buildx, metadata (címkék: ág, PR, semver `{{version}}` és `{{major}}.{{minor}}`, rövid SHA, `latest` az alapértelmezett ágon), amd64 build és smoke (csak olvasható konténer `healthy`, `/healthz`, `<html`, `/api/links` 401, `initial admin password generated` a naplóban, `-version`), majd `push` esetén ghcr.io-bejelentkezés és többplatformos (amd64, arm64) push provenance-szel és SBOM-mal.
- `dependabot.yml`: heti gomod (csoport), npm `/web` (minor+patch csoport; `typescript >=7` és `tailwind-variants >=4` tiltva), github-actions, docker (Node főverzió tiltva).
- Action-verziók: checkout v7, setup-node v7, setup-go v7, upload-artifact v7, download-artifact v8, cache v6, setup-buildx v4, metadata v6, build-push v7, login v4. Az actionlint elfogadta őket; a GitHub-on az első futásuk a 2026-11-22-i push (15.3).

### 15.3 Bizonyíték
**A workflow-k eredménye a GitHubon itt még nincs rögzítve:** a teljes projekt 2026-11-22-én került az első commitba és pushba a `main` ágra; ez a push indítja a `ci.yml` első futását, amely zöld tesztek után a `docker.yml`-lel image-et is közzétesz a ghcr.io-n. Az első futás eredményét a GitHub Actions oldalán kell ellenőrizni, és ide visszavezetni (20. fejezet 1. pont). A helyi bizonyíték a terv 11.9 FV-listája szerint készült 2026-11-21-én (a futások 2026-11-22 00:20-kor értek véget). Naplók: `kinex-mark-work\fv\`; átadó: `handoff\FV.md`.

| Lépés | Parancs | Eredmény |
|---|---|---|
| Tisztítás, tidy | `.\build.ps1 clean`, `.\build.ps1 tidy`, `Compare-Object` a `go.mod`/`go.sum` hash-én | exit 0; az összevetés üres |
| Teljes kapu | `.\build.ps1 verify` | exit 0, 325 s (az FV számai: Go 95,3%, Vitest 72 fájl / 1102 teszt, E2E 81) |
| Keresztfordítás | `.\build.ps1 cross -Version 0.1.0` | exit 0; linux/amd64, linux/arm64, windows/amd64, darwin/arm64; `-version` = `0.1.0` |
| Image | `.\build.ps1 image` | exit 0; `kinex-mark:e0e1df1`, 29 MB |
| Többplatformos build | `docker buildx build --platform linux/amd64,linux/arm64 --target runtime --build-arg VERSION=0.1.0 .` (push nélkül) | exit 0; manifest-lista két platformmal, az arm64 bináris keresztfordítva; mindkét image `-version` = `0.1.0` (az arm64 emulációval) |
| Compose-smoke | `docker compose up -d --build`, `docker compose ps` | `Up (healthy)`, `127.0.0.1:8080->8080/tcp`; csak olvasható gyökér, `CapDrop=[ALL]`, uid 65532, `no-new-privileges` |
| API-próba | `curl.exe -s -o NUL -w "%{http_code}" http://127.0.0.1:8080/api/links` | 401; `/healthz`: `{"status":"ok"}` |
| Verzió | `docker compose run --rm kinex-mark -version` | `dev` (a Compose alapértéke `VERSION:-dev`) |
| Helyreállítás | `docker compose stop`, `run --rm kinex-mark -unlock-user admin` és `-reset-password admin`, `start` | exit 0, utána ismét `healthy` |
| Leállítás | `docker compose down` | exit 0 |
| actionlint | `docker run --rm -v "${PWD}:/repo" -w /repo rhysd/actionlint:latest -color` | exit 0 (1.7.12) |
| langcheck | `go run ./tools/langcheck` | 547 fájl, nincs nem angol karakter |
| govulncheck | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` | No vulnerabilities found (v1.10.0) |
| Race | `golang:1.27`: `go test -race -count=1 ./internal/...` | exit 0, minden csomag ok |
| npm audit | `npm audit --omit=dev` (web) | 0 sebezhetőség |
| E2E újra | `.\build.ps1 test-e2e` | 81 passed |

A finisher (2026-11-22) a kód módosítása után a teljes `.\build.ps1 verify` kaput futtatta újra (14. fejezet; átadó: `handoff\FINISH.md`); a Docker-, buildx-, race- és govulncheck-lépéseket nem ismételte. A lead ezután a végleges kódon megismételte a teljes `verify` kaput (exit 0), a Docker race-futást (exit 0) és a govulncheck-et (`No vulnerabilities found`); a Compose-smoke és a többplatformos buildx az FV állapotából származik. Az új közvetlen függőség (`golang.org/x/text` v0.44.0) már korábban is ugyanezzel a verzióval volt a modulgráfban, amelyet a govulncheck átvizsgált.

Dokumentumok:
- a `manual/MANUAL.md` 33 képhivatkozása mind létező fájlra mutat, és mind a 33 kép használt (finisher-ellenőrzés);
- a README Windows-parancsai lefutottak (FV), a Linux-parancsok egyeznek a `Makefile`-lal és a `compose.yaml`-lal;
- a három követelménydokumentum és a `LICENSE` változatlan.

## 16. Hero-grafika

- Fájlok: `graphics/prompts.json` (modell `gemini-3.1-flash-image`, stílus `kinexium-diagram`, kép `hero`: 16:9, 2K, `background #071016`, kézbesítés WebP q88, max 1600 px), `graphics/generate_graphics.py` (a meglévő szkript bővítve: `aspect_ratio`, `image_size`, `max_width`, `background`, pontos arányú középre vágás, háttér-normalizálás, `--list`, `--force`, `--no-deliver`), mester `graphics/hero.png` (2752×1536), kézbesített `web/static/images/hero.webp` (1600×900, 25 592 B; a `web/hero_test.go` ellenőrzi).
- Generálás: mind a 4 engedélyezett fizetős hívás elfogyott; a 4. jelölt került be (2026-11-22, a felhasználó engedélyével, pontosított hélix-leírással). A 8.8.4 elfogadási lista minden pontja teljesül: paletta, hajszálvonalak, a rögzítés–címkék–szűrés–rangsor–elévülés kompozíció, valódi kétszálú hélix lépcsőkkel és csomópontokkal, öt címkenégyzet, egy visszaállító nyíl.
- **Utólagos retus (dokumentált):** a modell a 4. jelöltre négy angol feliratot is rajzolt („quick capture”, „list and ranking”, „tags and filtering”, „automatic expiry”). Ezeket a mester PNG-ről utólagos retus távolította el (mind tiszta háttéren állt; a kitöltés soronként a környező háttérből interpolál, más képelem nem változott). A `graphics/hero.png` tehát a retusált mester; a nyers kimenet (`hero-4.png`), a retus szkriptje (`retouch-4.py`), a használt prompt (`prompts-4.json`) és a korábbi jelöltek a repón kívüli munkakönyvtár `graphics\` mappájában vannak (nem tartós). A `graphics/prompts.json` a generálás **után** felirattiltó mondatot kapott a prompt elejére; ezzel a végső prompttal fizetős hívás még nem készült. Maradék apróságok: a hélix a jobb szélen kb. 6% margóig ér a kért 8% helyett, az archívumdoboz teteje enyhén trapéz, a listától balra halvány függőleges sáv látszik.
- Újragenerálás (**fizetős hívás, csak engedéllyel**): `GEMINI_API_KEY` beállítása, majd `python graphics/generate_graphics.py --force hero` (sima `python`, mert a `google-genai` a felhasználói site-ban van). `--force` nélkül a szkript a meglévő mesterből csak újrakézbesít; a `--list` hívás nélkül listáz. A keret (4 hívás) kimerült, további hívás csak a felhasználó újabb engedélyével. Csere után `go test ./web`, a bejelentkező oldal ellenőrzése a 8.8.4 listája szerint (olvasható szöveg a képen: retus vagy újragenerálás), és a kézikönyv `01-login.png`, `02-login-2fa.png` képének újrafelvétele (`-Version 0.1.0` build, 1440×900, sötét téma, magyar felület).
- Megjelenés: a bejelentkező oldal bal oszlopában, levágott sarkú táblában, témától független `ink-950` márkafelületen.

## 17. Eltérések a TECHNOLOGY.md-től és a GRAPHICS.md-től

### 17.1 TECHNOLOGY.md (terv 2.2, a kóddal egyeztetve)

| # | Téma | TECHNOLOGY.md | Kinex Mark (tényleges) | Indok |
|---|---|---|---|---|
| T1 | Gyökér, modul | `webapp/`, `example.com/webapp` | repó gyökér, `github.com/kaszasg/kinex-mark` | projektazonosítók |
| T2 | Belépési pont | `cmd/server` | `cmd/kinex-mark`; a `run` logika az `internal/app`-ban | tesztelhetőség |
| T3 | Csomagok | `api`, `store` | api, app, auth, bookmarks, clock, config, fetch, jobs, model, store, storetest, testutil | domén, párhuzamos fejlesztés |
| T4 | Mintadomén | todo | linkek, címkék, felhasználók | a feladat |
| T5 | Szövegek | magyar komment/napló/hiba | angol (`server started`, `internal error`) | a BASE-SYSTEM-PROMPT elsőbbsége |
| T6 | Hibatörzs | `{"error":"<magyar>"}` | `{"error","code","fields","details"}`, a felület a kódot fordítja | i18n |
| T7 | Router | middleware nélkül | recoverer, securityHeaders, accessLog, hostCheck, csrf, limitBody + útvonalankénti `guard` | biztonság |
| T8 | Go-függőség | csak sqlite | + `golang.org/x/net` (`html`, `html/charset`) és `golang.org/x/text` (`unicode/norm`, a finisher óta közvetlen) | HTML-tokenizálás, kódolás; ékezetfüggetlen kereső- és rendezőkulcs |
| T9 | API-kliens | `web/src/lib/api.ts` | `web/src/lib/api/` mappa, `ApiError` kóddal, `X-Kinex-Request` | szerződés, CSRF |
| T10 | `robots.txt` | `Disallow:` | `Disallow: /` | belső alkalmazás |
| T11 | ModeWatcher | alap props | + `disableHeadScriptInjection` | CSP |
| T12 | Fontok | 41 216 B / 38 048 B dokumentált hash-sel | `HankenGrotesk-Variable.woff2` 34 532 B, `GeistMono-Variable.woff2` 23 200 B; fonttools-szal reprodukálható (`tools/fonts/build_fonts.py`) | a dokumentált hash nem állítható elő |
| T13 | Node a Dockerben | `node:22-alpine` | `node:26-alpine` | egyezik a helyi és a CI Node-dal |
| T14 | Futtató image | `static-debian12:nonroot` | `static-debian13:nonroot` | Debian 13 a stabil |
| T15 | Compose port | `8080:8080` | `127.0.0.1:8080:8080`, `read_only`, `cap_drop: ALL`, `no-new-privileges` | biztonság |
| T16 | `cn()` | saját clsx + tailwind-merge | `export { cn } from 'cn'` (npm `cn` 0.5.2) | a shadcn CLI 1.8.1 ezt generálja |
| T17 | `tailwind-variants` | `^4.0.1` | `^3.3.1` | a generált komponensek 3-as API-ja |
| T18 | Importalias | `#lib` | saját kód `$lib`; `#lib` a `package.json`-ban marad | a generált fájlok `$lib`-et használnak |
| T19 | Ismert figyelmeztetések | `config_option_deprecated_alias` | + `tsconfig_invalid` („"paths" was overwritten”), kozmetikai | SvelteKit 3.4 |
| T20 | Build-célok | build, web, run, dev, cross, check, image, binary-image, clean | + tidy, test, test-go, test-web, test-e2e, e2e-install, verify, graphics (a Makefile-ban nincs `dev`) | tesztkapuk |
| T21 | `web/build` placeholder | – | nincs `.gitkeep`; minden buildút előbb a felületet építi | `git describe --dirty` |
| **T22** | Go-minimum | `go 1.24` | `go 1.26.0`, nincs `toolchain`; CI-mátrix `1.26.x`, `1.27.x`; Docker `golang:1.27-alpine` | a felhasználó 2026-11-21-i döntése: a régi `x/net`, `x/sys`, `x/text` 10 csomag- és 10 modulszintű sérülékenységet hordozott, a javított verziók 1.26-ot kérnek |
| **T23** | SQLite DSN | WAL, NORMAL, busy_timeout, foreign_keys | + `_pragma=secure_delete(1)`; `DeleteUser` után best-effort `wal_checkpoint(TRUNCATE)` | GDPR 17. cikk: a törölt adat ne maradjon olvasható a fájlokban és a későbbi mentésekben |
| – | SQLite-driver csomagolása | `sql.Open("sqlite", …)` | `sql.OpenDB(cancelSafeConnector{…})` a regisztrált driverrel (`kxm_lower`, `kxm_fold`) | a v1.39.0 megszakítási versenyhelyzete (18. fejezet 1.); mélységi védelemként megtartva |

### 17.2 GRAPHICS.md (terv 2.3)

| # | Téma | Eltérés | Indok |
|---|---|---|---|
| G1 | `app.css` kommentjei | a generátor minden kommentet eltávolít, egy angol fejlécet tesz | a forrás angol |
| G2 | `chart`, `avatar` | nincs telepítve | nincs idősor és profilkép |
| G3 | `StatusBadge` szavai | a `status` i18n-névtérből | i18n |
| G4 | Márkakomponensek kommentjei | angol | a forrás angol |
| G5 | AppPage metrikasor | a Linkek oldalon nincs; `Metric` a részletek lapon | a lista az elsődleges adat |
| G6 | Sheet fátyla | ugyanaz, mint a dialógusé, és egyik sem mos el (blur nélkül, E fázis) | egységes lebegő rétegek |
| G7 | QR-tároló | `bg-mist-50 text-ink-950` mindkét témában | a beolvasók sötét modult várnak világos alapon |
| G8 | Bejelentkező bal oszlop | témától független márkafelület (`ink-950`, `mist-100`, `signal-300`) | a sötét hero-képhez illeszkedik |
| G9 | Német megszólítás | tegező („du”), főnévi gombok | a magyar tegezés párja |
| G10 | Szóhasználat | „cím” = letöltött oldalcím, „címke” = tag (de: Label) | a két fogalom ne keveredjen |
| G11 | Fontfájlok | a T12 szerinti fájlok | reprodukálhatóság |
| G12 | Listacsoport-feliratok | „Kitűzött”/„Többi” `kx-eyebrow` | a rangsorolási szabályt mutatja |
| G13 | Címkeszínek | `chart-1…5` kategoriális címkeszínként, mindig névvel | egyetlen kategoriális paletta, új szín nélkül |
| G14 | Hero-feliratok | a generált képről négy angol felirat utólagos retussal lett eltávolítva (16. fejezet); a hélix a 4. jelölt óta valódi kétszálú | a GRAPHICS.md tiltja az olvasható szöveget a grafikán; a retus fizetős hívás nélkül javít |
| – | nova-alapértékek | alkalmazásrétegbeli felülírás (`Button.svelte`, `SidebarFrame.svelte`, `styles.ts`, `!important` utility a `dark:bg-input/30` ellen) | a GRAPHICS.md formái a generált nova stílusban mások |

### 17.3 Eltérések a lead döntéseitől (terv 2.4)
- L1: TypeScript `~6.0.3` a 7.0.2 helyett (SvelteKit compiler API).
- L2: a CSP a SvelteKit-indító inline szkriptjét is hash-sel engedi (induláskori számítás).
- L3: `/settings` négy alútvonal + kapuoldalak (`/change-password`, `/consent`).
- L4: a Go-kapu csomagonként is ≥ 80%, a Vitest-küszöb 85/80/85/85.

### 17.4 Értelmezések és finisher-kiegészítések
- **Törzsadatok (FN1, „Adminfelület a törzsadatok szerkesztésére”):** a megvalósítás a törzsadatokat így értelmezi: a felhasználói fiókok és szerepkörük (`/admin/users`), valamint a rendszerbeállítások (`/admin/settings`: archiválási időszak, munkamenet-időkorlátok, audit-megőrzés, belső hálózati letöltés, az adatkezelő adatai, tájékoztató-verzió); az audit napló csak olvasható. A címkék és színeik szándékosan személyesek (a követelmény szerint a csoportosítás címkével történik, és mindenki csak a saját adatát látja), ezért nincs globális címke- vagy kategóriakezelés; a három szerepkör rögzített (SQL `CHECK`). Ez a kézikönyv 11. fejezetének bevezetőjében is szerepel. Ha az ügyfél közös törzsadatot (globális címkék, kategóriák, szerkeszthető szerepkörök) vár, az új funkció (20. fejezet).
- **Ékezetfüggetlen keresés és „Név szerint” rendezés (finisher):** a `links.search_text`, a `links.sort_label` és a keresőszó a `foldText` kulcsot használja (`internal/store/driver.go`: kisbetű, NFD, kombináló jelek nélkül, NFC; SQL-ben `kxm_fold`), a címkenév-egyezés `kxm_fold(t.name_key)`. Migráció: `0005_fold_keys.sql`. Kliens-iker: `web/src/lib/links/fold.ts` (`order.ts` `sortLabel`, `query.ts` `matchesQuery`). Tesztek: `TestSearchAndNameOrderIgnoreAccents`, `TestFoldKeysMigration`, `fold.spec.ts`, az `order.spec.ts` és a `query.spec.ts` új esetei.
- **`import_too_many_entries` (finisher):** 5000-nél több bejegyzés esetén a szerver 400 `import_too_many_entries` kódot ad (korábban `import_invalid`, „A fájl nem értelmezhető.”, ami félrevezető volt). Golden, `errors.ts`, mindhárom nyelv, `TestImportErrors`, `TestErrorSpecsAreConsistent` (40), `ImportExportMenu.svelte.spec.ts`.
- **Kézikönyvképek (finisher):** az `01-login.png` és a `02-login-2fa.png` a `0.1.0` verziójú buildről készült újra („Verzió 0.1.0” a láblécben), a `22-sessions.png` a Munkamenetek kártya elemképe dokumentációs IP-címekkel (192.0.2.10, 203.0.113.25, 198.51.100.7), új kép a `24-import-export-menu.png` (a nyitott „Import és export” menü; a terv 13.2 listája kiegészítve).
- **Dokumentáció (finisher):** a README `APP_TRUSTED_PROXY` leírása tanácsként (egyetlen cím ajánlott, CIDR is megadható), a `compose.yaml` proxy-kommentje (a 172.30.0.2 címet a proxy szolgáltatásban `ipv4_address`-szel kell rögzíteni), a kézikönyv szóhasználata (adminisztrátor, üzemeltető, rendszergazda) és számos pontosítás.

## 18. Workaroundok, amelyeket a karbantartónak ismernie kell

1. **Megszakításbiztos SQLite-connector** (`internal/store/cancelsafe.go`): a sor-visszaadó `QueryContext` a `context.WithoutCancel(ctx)`-tel fut (előtte `ctx.Err()` ellenőrzés), így a driver nem indítja a versenyhelyzetes interrupt-goroutine-t. Ok: a `modernc.org/sqlite` v1.39.0 megszakításkor finalizálatlan statementet hagyott, az `app.db` nyitva maradt (Windowson nem törölhető). Az upstream a v1.40.1-ben javította; a burok és a `TestCancelledQueriesReleaseDatabaseFile` mélységi védelemként marad. Ára: egy hosszú `sqlite3_step` az első sor előtt nem szakítható meg. Driverfrissítéskor az új `stmt.query`-t újra kell vizsgálni.
2. **`kxm_lower` és `kxm_fold` SQL-függvények** (`internal/store/driver.go`): a beépített `lower()` csak ASCII-t kisbetűsít; a felhasználókönyvtár keresése a `kxm_lower`-t, a linkkeresés és a névrendezés a `kxm_fold`-ot használja. A `foldText` Go- (`driver.go`) és TS-változatának (`web/src/lib/links/fold.ts`) egyeznie kell.
3. **60 ms-os PinInput-várakozás** (`web/src/lib/features/auth/TotpStep.svelte` `pinSettled`): a bits-ui PinInput időzítő-szivárgása miatt (`derived_inert` hiba TOTP-belépés után).
4. **`SidebarFrame` a `Sidebar.Provider` helyett**: a generált provider `sidebar_state` sütit írt, és globális Ctrl/Cmd+B kezelővel a szövegmezőkből is elvette a billentyűt.
5. **nova-felülírások az alkalmazásrétegben** (`components/app/Button.svelte`, `styles.ts`, `!important` utilityk): a generált komponensek nem módosíthatók; az egyetlen kézi módosítás a három overlay-fájl fátyla.
6. **CSP-hash induláskor** (`internal/api/spa.go`): a SvelteKit-indító szövege buildenként változik, ezért a szerver az `index.html` inline szkriptjeit hash-eli; a `ModeWatcher` `disableHeadScriptInjection`-nel fut.
7. **E2E-leállítás sorrendje** (`web/e2e/support/global-teardown.ts`): a Playwright 1.65 a global teardownt a webServer leállítása előtt futtatja, Windowson a futó szerver zárolja az `app.db`-t; ezért a teardown a `test-results/.e2e-server-pid` alapján maga állítja le a szervert. A futás végi `[start-server] server exited (code 1…)` sor várt. A `start-server.mjs` eldobja az örökölt `APP_*` változókat.
8. **`reuseExistingServer: false`** (`web/playwright.config.ts`): a 18080/18081 a harnessé.
9. **`msg()` dinamikus importtal** (`web/e2e/support/i18n.ts`): hogy az e2e `tsc` ne fordítsa az app-forrást.
10. **`build.ps1` sajátosságai**: idézett `'-covermode=atomic'`, `npm.cmd`/`npx.cmd`, a fejlesztői szerver külön `kinex-mark-dev.exe`, a környezeti változók visszaállítása a Go-build után; a `-race` helyben nem fut. Az `npm notice` stderr-sora PowerShell 5.1-ben `*>` átirányítással `NativeCommandError`-t és hamis hibakódot ad: a buildet `powershell -File .\build.ps1 …` alakban vagy Git Bashből futtasd.
11. **`secure_delete(1)` + WAL-csonkolás** (`store.go`, `users.go` `truncateWAL`).
12. **Leállási drain** (`internal/app/app.go` `closeStore`): `Ping` a `Close` előtt.
13. **`toastError` és `ApiError.authHandled`** (`web/src/lib/utils/toast-error.ts`): a munkamenet-tároló által kezelt hibához nem jelenik meg hibatoast.
14. **Tájékoztató-kapu verziófrissítés**: `consent_required` esetén a `session.handleAuthFailure` a nyilvános konfigurációt is frissíti (E1 javítás).

## 19. Ismert korlátok

A D-close és az E-close listái alapján, a finisher változásaival frissítve.

1. **Toaster-felirat utótagja (alacsony):** a svelte-sonner hozzáfűzi a gyorsbillentyű-tippet, a régió neve „Értesítések alt+T”.
2. **Elutasított akadálymentességi alpontok (R5):** a nyelvváltók HU/EN/DE feliratúak, de a nevük Magyar, English, Deutsch (a terv így írja elő); a bejelentkező oldal lábléce a `main`-en belül van (terv 8.8.1). Nem WCAG-hiba.
3. **Az R6 javítása előtt tárolt, negatív évszámú `created_at` sorok** nincsenek javítva (nincs migráció); friss telepítésen ilyen sor nem keletkezhet.
4. **Az importált linkek inaktivitása az import idejétől számít** (szándékos).
5. **`secure_delete(1)` írási költsége** (T23); a fióktörlés utáni WAL-csonkolás best-effort.
6. **Bent maradó workaroundok:** megszakításbiztos connector, 60 ms-os PinInput-várakozás (18. fejezet).
7. **Frissítési megjegyzés – adatkezelési tájékoztató:** a tájékoztató szövege a D fázisban bővült (címzettek, tárolókulcsok). Meglévő telepítésen, ahol a felhasználók a régi szöveget nyugtázták, egy adminnak új verziót kell kiadnia (Rendszerbeállítások, „Új verzió kiadása”, vagy `POST /api/admin/settings/privacy-notice/bump`). Friss telepítés: nincs teendő. A README és a kézikönyv B.9 pontja is kimondja.
8. **SVG-faviconok:** szerkesztői metaadatos, idegen névterű vagy `xmlns` nélküli SVG nem tárolódik, ilyenkor a raszteres tartalék jön.
9. **Bejelentkezés-korlátozás globális felhasználói kulccsal:** a felhasználónevet ismerő támadó egy fiókot ismételten zárolhat (zárolásonként legfeljebb 24 óra). Helyreállítás: admin feloldás, offline `-unlock-user`/`-reset-password`.
10. **Névrendezés:** ékezet- és kisbetű-független, de nem a magyar ábécé szerinti: az ö/ő az o-val, az ü/ű az u-val együtt rendeződik (`foldText`, 17.4).
11. **Nem ellenőrzött ebben az állapotban:** a Compose-smoke és a többplatformos buildx a finisher módosításai után (az FV-ben a korábbi kódon zöld); böngészőben a 200 linken túli lapozás az E-close óta (Vitest fedi); a valódi `window.print()` párbeszéd (csak print-media emulációval).
12. **CI/CD:** a workflow-k első GitHub-futásának (2026-11-22-i push) eredménye és a ghcr.io image megléte itt nincs rögzítve (15.3).
13. **Ismert build-figyelmeztetések** (terv 10.7): `config_option_deprecated_alias` és `tsconfig_invalid` minden `svelte-kit sync`-nél, `[PLUGIN_TIMINGS]`, `npm notice`, és a Windows E2E végi `server exited (code 1…)` sor; ártalmatlanok.
14. **Vágólap nem biztonságos kontextusban** (HTTP LAN-címen): az `execCommand('copy')` tartalék működik, de nem garantált; TLS ajánlott.
15. **Hatókörön kívül:** mappák, e-mail, jelszó-visszaállítás e-mailben, böngészőbővítmény, nyilvános megosztás, analitika (1. fejezet), globális címkék és szerkeszthető szerepkörök (17.4). A szerepkörök és nyelvek halmazát SQL `CHECK`-korlátok rögzítik.
16. **Hero:** a mester PNG kézzel retusált (feliratok eltávolítva, 16. fejezet), ezért a `--force` újragenerálás után a képet újra át kell nézni; apró eltérések: jobb oldali margó kb. 6%, enyhén trapéz dobozfedél.
17. **Kézikönyvképek:** a demóadatok különböző pillanataiban készültek (a link- és címkeszám képenként eltér, sok sor utolsó használata azonos); a bejelentkező oldal 1440×900-on görgethető, a bal oszlop funkciólistája a kép alján levágódik. A kézikönyv bevezetője ezt kimondja.
18. **KV13 ellenőrzés:** az FV és a finisher nem tudta futtatni (nem volt ügynökindító eszközük); a lead 2026-11-22-én pótolta, a talált hiányok javítva (20. fejezet 2. pont).

## 20. Nyitott ötletek és következő lépések

Prioritás szerint; mindegyiknél az érintett fájlok és a bizonyítás módja.

1. **Az első CI-futás ellenőrzése** (az első commit és push 2026-11-22-én megtörtént a `main` ágra): a `ci.yml` és a `docker.yml` első futásának figyelése zöldig; bizonyíték: zöld `web`, `go` (mindkét mátrixág, race, govulncheck), `go-windows`, `e2e`, `binaries`, `image` job és a ghcr.io-n megjelenő `latest`/SHA címke; utána a README és a 15.3/19. pont „nem futott” megjegyzésének frissítése.
2. **KV13-ellenőrzés: kész (2026-11-22).** Egy friss Sonnet 5.5 ügynök csak ebből a fájlból válaszolt 12 fejlesztői kérdésre, majd a forráskóddal vetette össze a válaszait, és több mint 100 útvonalat és azonosítót ellenőrzött. Hét kérdésre elegendő volt a dokumentum, ötnél hiányt talált (hibakód-teszt, a rendezés helye, új nyelv, új adminoldal, migrációs menet), továbbá két hibás sort a 3.2 függőségi gráfban; mindet javítottuk ebben a fájlban (0., 2.2, 3.2, 6.2, 6.5–6.7, 8.4, 10.1, 10.3, 11.1, 13.).
3. **Race-teszt a végleges kódon: kész (2026-11-22).** A finisher módosításai után `go test -race -count=1 ./internal/...` Dockerben (`golang:1.27`, `GOTOOLCHAIN=local`): exit 0, minden csomag ok (`internal/store` 17,3 s). Ugyanezen a kódon a lead saját `.\build.ps1 verify` futása is exit 0 (95,3%, 73 fájl / 1107 Vitest-teszt, E2E 81 passed).
4. **(Felhasználó, IP-tartomány) párra szűkített, lépcsőzetes zárolás** a globális felhasználói kulcs mellé (19/9): `internal/auth/throttle.go`, `internal/api/h_auth.go` (`signInThrottleKeys`), `internal/store/throttle.go`; bizonyíték: Go HTTP-teszt, hogy idegen IP-ről érkező támadás nem zárja ki a tulajdonost, és a meglévő párhuzamossági teszt zöld marad.
5. **Hero: kész (2026-11-22).** A felhasználó engedélyével a 4. hívás kétszálú hélixet adott; a feliratok retusálva, a kézikönyv két bejelentkező képe újrafelvéve (16. fejezet). További finomítás (margó, dobozfedél) csak újabb fizetős hívással és engedéllyel.
6. **Magyar ábécérend** a névrendezéshez, ha az ügyfél kéri (19/10): külön rendezőkulcs nyelvi tailoringgal (például `golang.org/x/text/collate` kulcs a `sort_label` mellé, a kliensen `Intl.Collator`), migrációval; bizonyíték: `TestSearchAndNameOrderIgnoreAccents` kiterjesztése (o < ö < p).
7. **Toaster-régió neve hotkey-utótag nélkül**: `web/src/lib/components/app/AppToaster.svelte`; bizonyíték: Vitest + axe az `a11y.spec.ts`-ben.
8. **Driverfrissítéskor** a `cancelSafeConnector` újraértékelése (18/1).
9. **Böngészős ellenőrzés** 200 linken túli lapozásra és a valódi nyomtatásra.
10. **Közös törzsadatok** (globális címkék, kategóriák), ha az ügyfél igényli (17.4): új táblák és admin-oldal, a láthatósági modell bővítésével.

## 21. Dokumentumtérkép

| Dokumentum | Hely | Tartós? | Tartalom |
|---|---|---|---|
| README | `README.md` | igen | rövid futtatási összefoglaló |
| Ez az összefoglaló | `SUMMARY.md` | igen | állapot, térkép, folytatás |
| Kézikönyv | `manual/MANUAL.md`, `manual/images/*.png` (33 kép) | igen | magyar felhasználói és üzemeltetői kézikönyv |
| Fejlesztési terv | `ai/BASE-SYSTEM-PLAN.md` (~5600 sor, magyar) | igen | a kötelező terv; a megvalósítás eltéréseit visszavezették |
| Követelmények | `ai/BASE-SYSTEM-PROMPT.md` (legmagasabb rang), `ai/TECHNOLOGY.md`, `ai/GRAPHICS.md` | igen | nem módosíthatók |
| Szerződés-golden | `internal/api/testdata/contract.json` | igen | hibakódok, mezőkódok, audit, színek, szerepkörök, nyelvek |
| Licenc | `LICENSE` | igen | MIT |
| Átadó jegyzetek | `C:\Users\kasza\AppData\Local\Temp\kinex-mark-work\handoff\` (fázisonkénti close- és átadófájlok, FV, FINISH) | **nem** (ideiglenes könyvtár) | fázisonkénti eredmények, kapulogok |
| Felülvizsgálati jegyzetek | `C:\Users\kasza\AppData\Local\Temp\kinex-mark-work\review\` | **nem** | a D fázis megállapításai és javításai |
| Egyéb munkafájlok | `C:\Users\kasza\AppData\Local\Temp\kinex-mark-work\` (fv, e-close, dclose, gui-*, finish, graphics) | **nem** | logok, képernyőképek, a hero jelöltjei |

Ami a repón kívüli jegyzetekből tartósan fontos, az ebben a fájlban szerepel (különösen a 15–20. fejezet).
