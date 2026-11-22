# KINEXIUM webapp: technológiai leírás

Ez a dokumentum a KINEXIUM webalkalmazás műszaki felépítését írja le. Minden érték, parancs és konfiguráció benne van, ami az induláshoz kell. A grafikai követelményeket (színek, tipográfia, komponensek megjelenése, logó, ikonok) a társdokumentum, a GRAPHICS.md tartalmazza.

A dokumentum két állapotot különböztet meg:

- **Váz:** a jelenlegi mintaprojekt. Működő Go + SQLite + SvelteKit alkalmazás egy teendőlistával, semleges, ideiglenes arculattal.
- **Célállapot:** a váz a KINEXIUM-arculat bekötése után (mode-watcher, shadcn-svelte CLI-s komponensek, saját fontok, favicon, márkakomponensek).

Ahol a kettő eltér, a dokumentum mindkettőt megadja, és jelzi, melyik a cél.

---

## 1. Cél és áttekintés

A webapp egyetlen statikus Go-bináris. Ez tartalmazza:

- a lefordított SvelteKit-felületet (kliensoldali SPA, statikus buildként),
- a HTTP API-t,
- az SQLite-motort (tiszta Go-s driver, CGO nélkül),
- az adatbázis-migrációkat.

Futás közben csak az adatkönyvtár marad külön, ebbe kerül az `app.db` adatbázisfájl. A felület futtatásához nem kell Node.

| Réteg | Választás | Szerep |
|---|---|---|
| Felület | SvelteKit 3, Svelte 5, TypeScript | Kliensoldali alkalmazás, statikus buildként |
| Megjelenés | Tailwind CSS v4, shadcn-svelte | A saját kódbázisba másolt komponensek |
| Backend | Go, `net/http` | Az API és a felület kiszolgálása |
| Adat | SQLite, `modernc.org/sqlite` | Tiszta Go-s driver, CGO nélkül |
| Csomagolás | `go:embed`, Docker | Egy bináris, illetve minimális konténer |

A build sorrendje kötött. A `web/build` mappa a Go-fordítás bemenete, ezért előbb a felületnek kell elkészülnie, utána a Go-binárisnak. Az adatbázist a program futás közben hozza létre, és induláskor maga futtatja a migrációkat.

A váz egyetlen oldala egy teendőlista („Teendők”): hozzáadás, kész állapot váltása, törlés. Hitelesítés, felhasználókezelés, szerkesztés és szűrés nincs benne.

---

## 2. Technológiai stack

### 2.1 Backend és futtatókörnyezet

| Elem | Verzió / érték | Megjegyzés |
|---|---|---|
| Go modul | `example.com/webapp` | A `go.mod` teljes tartalma: `module example.com/webapp` és `go 1.24` |
| Go (minimum) | `1.24` | A `go.mod` szerint. Nincs `toolchain` sor |
| Go (próbák) | `1.24.7` | Ezzel a verzióval futottak a próbák |
| Go (Docker build image) | `golang:1.27-alpine` | A Docker-buildben, ezzel nem volt próba |
| SQLite driver | `modernc.org/sqlite` | Tiszta Go, CGO nélkül. A `go mod tidy` a verziót a `go.mod` `require` sorába, az ellenőrző összegeket a `go.sum`-ba írja |
| Driver neve | `sqlite` | `sql.Open("sqlite", ...)` |
| Egyéb Go-függőség | nincs | Minden más import a standard könyvtárból jön |
| CGO | `CGO_ENABLED=0` | Mindenhol (Makefile, PowerShell-szkript, Dockerfile) |
| Futtató image | `gcr.io/distroless/static-debian12:nonroot` | Nincs benne shell, uid 65532 |
| Node (Docker build image) | `node:22-alpine` | Csak a felület buildjéhez |

A használt standard csomagok: `context`, `errors`, `flag`, `fmt`, `log/slog`, `net`, `net/http`, `os`, `os/signal`, `syscall`, `time`, `encoding/json`, `io/fs`, `path`, `strconv`, `strings`, `unicode/utf8`, `database/sql`, `embed`, `net/url`, `path/filepath`, `sort`.

### 2.2 Frontend (váz)

| Csomag | Tartomány (`package.json`) | Feloldott verzió (`package-lock.json`) | Típus |
|---|---|---|---|
| Node | nincs (`engines` nincs megadva) | 22 (a próbákon) | futtatókörnyezet (build); 22.17 vagy újabb 22.x, illetve 24 vagy újabb (a `@sveltejs/kit` és a `@sveltejs/vite-plugin-svelte` `engines` mezőjéből), az `engine-strict=true` miatt kötelező; a Node 23 nem támogatott |
| `@sveltejs/kit` | `^3.0.0` | 3.1.1 | dev |
| `svelte` | `^5.57.2` | 5.57.3 | dev |
| `vite` | `^8.3.0` | 8.3.6 | dev |
| `@sveltejs/vite-plugin-svelte` | `^7.2.0` | 7.3.1 | dev |
| `@sveltejs/adapter-static` | `^4.0.1` | 4.0.1 | dev |
| `tailwindcss` | `^4.3.4` | 4.3.4 | dev |
| `@tailwindcss/vite` | `^4.3.4` | 4.3.4 | dev |
| `typescript` | `^6.0.3` | 6.0.3 | dev |
| `svelte-check` | `^4.6.0` | 4.7.8 | dev |
| `@lucide/svelte` | `^1.69.0` | 1.69.0 | futásidejű |
| `clsx` | `^2.1.1` | 2.1.1 | futásidejű |
| `tailwind-merge` | `^3.7.0` | 3.7.0 | futásidejű |
| `tailwind-variants` | `^4.0.1` | 4.0.1 | futásidejű |

- A „Típus” oszlopban a `dev` a `devDependencies`, a `futásidejű` a `dependencies` mezőt jelenti.
- A pontos feloldott verziók a `web/package-lock.json`-ban vannak (`lockfileVersion` 3), és megegyeznek a táblázat „Feloldott verzió” oszlopával. A lockfile kötelező, mert minden buildút (Makefile `web` cél, `build.ps1`, Dockerfile) `npm ci`-t futtat, és az `npm ci` lockfile nélkül leáll.
- Ha a `web/package-lock.json` nincs meg, a `web/` mappában egyszer `npm install` hozza létre a fenti tartományokból (Node 22.17 vagy újabb 22.x, illetve Node 24 vagy újabb). Utána a feloldott verziókat a táblázattal kell egyeztetni, a lockfile verziókövetésbe kerül, és a build `npm ci`-vel fut.
- `web/.npmrc`: `engine-strict=true`. A saját `package.json`-ban nincs `engines` mező, de a beállítás a függőségek `engines` mezőjére is érvényes: a `@sveltejs/kit` 3.1.1 `engines.node` értéke `>=22.17` (a `vite` 8.3.6-é `^20.19.0 || >=22.12.0`, a `@sveltejs/vite-plugin-svelte` 7.3.1-é `^20.19 || ^22.12 || >=24`), ezért Node 22.17 alatt és Node 23.x alatt az `npm ci` `EBADENGINE` hibával leáll; a beállítás nélkül csak figyelmeztetne. A `.npmrc`-t meg kell tartani.

### 2.3 Frontend (célállapotban hozzájön)

| Csomag | Telepítés | Szerep |
|---|---|---|
| `mode-watcher` | `npm i mode-watcher` | Téma (sötét alap, világos pár), `localStorage` kulcs: `mode-watcher-mode` |
| `tw-animate-css` | `npm i -D tw-animate-css` | Animációs utility-k, a shadcn-svelte komponensekhez |
| `shadcn-svelte` | az `npx shadcn-svelte@latest init` telepíti | A shadcn-svelte stílusai és CLI-je |
| Hanken Grotesk | `static/fonts/HankenGrotesk-Variable.woff2` | Változó font, WOFF2, súly 100–900, 41 216 bájt (kb. 40 KB), SIL OFL 1.1 |
| Geist Mono | `static/fonts/GeistMono-Variable.woff2` | Változó font, WOFF2, súly 100–900, 38 048 bájt (kb. 37 KB), SIL OFL 1.1 |

A két font együtt kb. 80 KB, a binárisba ágyazva sem számít. Licencszövegek: Hanken Grotesk: „Copyright 2021 The Hanken Grotesk Project Authors (https://github.com/marcologous/hanken-grotesk)”; Geist Mono: „Copyright 2024 The Geist Project Authors (https://github.com/vercel/geist-font.git)”. Mindkettő SIL Open Font License, Version 1.1. A licencszövegek fájlnevei: `OFL-Hanken-Grotesk.txt`, `OFL-Geist-Mono.txt`; ezek a fontokkal együtt maradnak (6.10).

### 2.4 Fejlesztői eszközök

| Eszköz | Megjegyzés |
|---|---|
| GNU Make + bash | Linux, macOS |
| Windows PowerShell 5.1 vagy PowerShell 7 | Windows (`build.ps1`). A próbák PowerShell 7.6.0-val, Linuxon futottak; valódi Windowson nincs kipróbálva (7.10) |
| Docker (Linux-konténer mód) | Csak a konténeres buildhez és futtatáshoz |
| Git | A verziószámot a `git describe` adja |
| C-fordító | Nem kell (nincs CGO) |
| VS Code ajánlott bővítmény | `svelte.svelte-vscode` (`web/.vscode/extensions.json`) |

---

## 3. Repository-struktúra

```
webapp/
├── cmd/
│   └── server/
│       └── main.go              # belépési pont, kapcsolók, leállítás, healthcheck
├── internal/
│   ├── api/
│   │   └── api.go               # HTTP-végpontok, a felület kiszolgálása
│   └── store/
│       ├── driver.go            # az SQLite-driver importálása
│       ├── store.go             # adatelérés, migrációk futtatása, Todo modell
│       └── migrations/
│           └── 0001_create_todos.sql
├── web/
│   ├── embed.go                 # a build mappa beágyazása (Go csomag)
│   ├── vite.config.ts           # Vite, Tailwind, SvelteKit, adapter, fejlesztői proxy
│   ├── package.json
│   ├── package-lock.json
│   ├── tsconfig.json
│   ├── .npmrc
│   ├── .gitignore
│   ├── .vscode/extensions.json
│   ├── components.json          # célállapot: shadcn-svelte konfiguráció
│   ├── static/                  # változatlanul másolt fájlok
│   │   ├── robots.txt
│   │   ├── favicon.svg          # célállapot
│   │   └── fonts/               # célállapot: a két .woff2
│   ├── src/
│   │   ├── app.css              # Tailwind és a design tokenek
│   │   ├── app.html
│   │   ├── app.d.ts
│   │   ├── lib/
│   │   │   ├── api.ts           # típusos API-kliens
│   │   │   ├── utils.ts         # cn() segédfüggvény
│   │   │   ├── index.ts
│   │   │   ├── assets/          # váz: favicon.svg (Svelte-logó)
│   │   │   ├── hooks/           # célállapot: a shadcn-svelte CLI hozza létre
│   │   │   └── components/
│   │   │       ├── ui/          # váz: kézzel írt komponensek (célállapotban a shadcn-svelte komponensei)
│   │   │       │   ├── button/
│   │   │       │   │   ├── button.svelte
│   │   │       │   │   └── index.ts
│   │   │       │   ├── card/
│   │   │       │   │   ├── card.svelte
│   │   │       │   │   └── index.ts
│   │   │       │   └── input/
│   │   │       │       ├── input.svelte
│   │   │       │       └── index.ts
│   │   │       └── brand/       # célállapot: Logo, Metric, Panel, StatusBadge, TickRule
│   │   └── routes/
│   │       ├── +layout.svelte
│   │       ├── +layout.ts       # ssr = false
│   │       └── +page.svelte
│   └── build/                   # generált, nincs verziókövetésben
├── dist/                        # generált binárisok
├── data/                        # futás közben jön létre, itt az app.db
├── Dockerfile
├── .dockerignore
├── compose.yaml
├── Makefile
├── build.ps1                    # a Makefile párja Windowsra
├── README.md
├── go.mod
├── go.sum                       # a go mod tidy hozza létre
├── .gitattributes
└── .gitignore
```

Generált, nem verziókövetett mappák: `web/build`, `web/node_modules`, `web/.svelte-kit`, `dist`, `data`. A `go.sum` a `go mod tidy` első futtatásakor jön létre.

A `web/static/robots.txt` a sablonból jön, és minden útvonalat engedélyez:

```
# allow crawling everything by default
User-agent: *
Disallow:
```

Hitelesítés nélküli belső alkalmazásnál a `Disallow: /` zárja ki a keresőrobotokat.

A gyökér `README.md` tartalma (teljes):

````markdown
# webapp

SvelteKit, Tailwind CSS v4 és Go + SQLite egyetlen futtatható programban.

## Első indítás

```bash
go mod tidy    # letölti a modernc.org/sqlite modult és létrehozza a go.sum fájlt
make build
./dist/webapp  # http://localhost:8080
```

Docker esetén: `go mod tidy`, majd `docker compose up -d --build`.

A `go.sum` nincs a csomagban, ezért a `go mod tidy` lépés a Docker-buildhez is kötelező.

## Windows és PowerShell

```powershell
go mod tidy
.\build.ps1
.\dist\webapp.exe  # http://localhost:8080
```

Fejlesztéshez: `.\build.ps1 dev`, a felület a http://localhost:5173 címen érhető el.
````

### 3.1 Csomagfelelősségek

| Csomag | Útvonal | Felelősség |
|---|---|---|
| `main` | `cmd/server` | Kapcsolók és környezeti változók, logger, jelzéskezelés, adatkönyvtár létrehozása, store megnyitása, HTTP-szerver, szabályos leállítás, healthcheck mód. A webapp egyetlen binárisban szolgálja ki a felületet és az API-t |
| `api` | `internal/api` | A HTTP-végpontok és a felület kiszolgálása |
| `store` | `internal/store` | Az SQLite-adatbázis, a migrációk és az adatelérés |
| `web` | `web` | A lefordított SvelteKit-felület beágyazása a binárisba |

Belső importútvonalak: `example.com/webapp/internal/api`, `example.com/webapp/internal/store`, `example.com/webapp/web`.

A `web/embed.go` azért van a `web/build` mellett, mert a `go:embed` nem hivatkozhat `..` útvonalra.

---

## 4. Backend

### 4.1 Indítás

A `cmd/server/main.go` lépései:

1. `var version = "dev"`. A build a `-ldflags "-X main.version=..."` kapcsolóval állítja be.
2. Kapcsolók feldolgozása. `-version`: kiírja a verziót (`fmt.Println(version)`) és kilép. `-healthcheck`: `os.Exit(checkHealth(*addr))`.
3. Logger: `slog.New(slog.NewTextHandler(os.Stdout, nil))`. Szöveges formátum, stdout, alapszint Info.
4. `run(addr, dataDir, log)`. Ha hibát ad: `log.Error("a szerver leállt", "err", err)` és `os.Exit(1)`.

A `run` lépései:

1. `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`.
2. `os.MkdirAll(dataDir, 0o750)`. Hiba: `fmt.Errorf("adatkönyvtár: %w", err)`.
3. `store.Open(ctx, dataDir)`, majd `defer st.Close()`.
4. `web.Files()`: a beágyazott felület.
5. `api.New(st, ui, log)`: a HTTP-kezelő.
6. `http.Server` a lenti időkorlátokkal.
7. Goroutine: `log.Info("a szerver elindult", "addr", addr, "data", dataDir, "version", version)`, majd `errs <- srv.ListenAndServe()` (`errs` egy 1 elemű pufferelt `chan error`). A napló a `ListenAndServe` előtt íródik ki, ezért foglalt portnál is megjelenik; ilyenkor a hiba az `errs` csatornán jön, a program 1-es kóddal kilép.
8. `select`: az `errs` ágon a `http.ErrServerClosed` nem hiba, minden más hiba visszatér. A `ctx.Done()` ágon `log.Info("leállítás")`.
9. Szabályos leállítás: `context.WithTimeout(context.Background(), 10*time.Second)`, `srv.Shutdown(...)`. SIGTERM vagy Ctrl+C után a szerver legfeljebb 10 másodpercig várja a futó kéréseket, majd lezárja az adatbázist.

| `http.Server` mező | Érték |
|---|---|
| `Addr` | a `-addr` értéke |
| `ReadHeaderTimeout` | `5 * time.Second` |
| `ReadTimeout` | `30 * time.Second` |
| `WriteTimeout` | `30 * time.Second` |
| `IdleTimeout` | `2 * time.Minute` |
| `Handler` | az `api.New` által adott mux, middleware nélkül |

### 4.2 Konfiguráció

| Kapcsoló | Környezeti változó | Alapérték | Súgószöveg (`-h`) | Jelentés |
|---|---|---|---|---|
| `-addr` | `APP_ADDR` | `:8080` | `figyelési cím` | Figyelési cím |
| `-data` | `APP_DATA_DIR` | `./data` | `adatkönyvtár` | Adatkönyvtár, itt jön létre az `app.db` |
| `-healthcheck` | nincs | `false` | `az élő szerver ellenőrzése, majd kilépés` | Lekérdezi a futó szervert, majd 0 vagy 1 kóddal kilép |
| `-version` | nincs | `false` | `verzió kiírása, majd kilépés` | Kiírja a verziót, majd kilép |

- A kapcsolókat a standard `flag` csomag dolgozza fel: ismeretlen vagy hibás kapcsolónál kiírja a hibát és a súgót, majd 2-es kóddal kilép; a `-h` a súgót írja ki.
- Elsőbbség: kapcsoló, majd környezeti változó, majd alapérték. Az `env()` segédfüggvény a változó értékét adja, ha az nem üres szöveg, különben az alapértéket.
- Példa: `./dist/webapp -addr :9000 -data /var/lib/webapp`
- Windows: `.\dist\webapp.exe -addr 127.0.0.1:9000 -data C:\ProgramData\webapp`, vagy `$env:APP_DATA_DIR = 'D:\webapp-adat'` és `.\dist\webapp.exe`.

| Környezet | Cím | Adatkönyvtár |
|---|---|---|
| Helyi futtatás | `:8080` (http://localhost:8080) | `./data` |
| Fejlesztői mód (`build.ps1 dev`) | `127.0.0.1:8080`, csak loopback | `./data` |
| Docker image | `APP_ADDR=:8080`, `EXPOSE 8080` | `APP_DATA_DIR=/data`, volume |
| Docker Compose | `8080:8080` | `webapp-data:/data` |
| Vite fejlesztői szerver | `5173` (a Vite alapértelmezése, nincs beállítva) | |

### 4.3 Healthcheck

A `-healthcheck` mód a konténer `HEALTHCHECK`-jéhez készült, mert a futtató image-ben nincs `curl`. A `checkHealth(addr)` működése:

1. `net.SplitHostPort(addr)`. Hiba esetén 1.
2. Ha a host `""` vagy `"0.0.0.0"`, `127.0.0.1` lesz. A `::` címet nem írja át.
3. `http.Client{Timeout: 3 * time.Second}`, `GET http://<host:port>/healthz` (`net.JoinHostPort`).
4. Hiba vagy nem 200-as státusz esetén 1, különben 0.

A `-healthcheck` a `-addr` értékét használja, tehát az `APP_ADDR` változót is figyelembe veszi.

### 4.4 Router

`api.New(st *store.Store, ui fs.FS, log *slog.Logger) (http.Handler, error)`. A router `http.NewServeMux()` (Go 1.22 óta elérhető metódusos minták). Kéréstörzs-korlát: `maxBodyBytes = 1 << 16` (65 536 bájt).

| Minta | Kezelő | Megjegyzés |
|---|---|---|
| `GET /healthz` | `health` | Életjel, az adatbázist is pingeli |
| `GET /api/todos` | `listTodos` | Lista |
| `POST /api/todos` | `createTodo` | Létrehozás |
| `PATCH /api/todos/{id}` | `updateTodo` | Kész állapot módosítása |
| `DELETE /api/todos/{id}` | `deleteTodo` | Törlés |
| `/api/` | catch-all | Minden metódusra 404 `{"error":"nincs ilyen végpont"}` |
| `/` | SPA-kezelő | Beágyazott felület |

Az ismeretlen API-útvonal JSON-hibát kap, nem az `index.html`-t, mert az `/api/` minta specifikusabb a `/` mintánál. A ServeMux a regisztrálás sorrendjétől függetlenül a specifikusabb mintát választja; a kódban az `/api/` a `/` előtt áll.

Viselkedés a minták miatt:

- `PUT /api/todos` és minden más, nem regisztrált metódus az `/api/` alatt 404-et ad, nem 405-öt.
- `POST /healthz` és minden nem GET kérés a `/healthz`-re a `/` mintára esik, és 200-as választ kap az `index.html`-lel.
- A `/api` (záró perjel nélkül) kérést a ServeMux az `/api/` részfa-minta (záró perjeles minta) miatt átirányítja `/api/`-ra (Go 1.24.7-tel 301; a Docker-build Go 1.27-es fordítójával a státuszkód ellenőrizendő), utána a válasz 404 `{"error":"nincs ilyen végpont"}`.
- A `GET /healthz` és a `GET /api/todos` minta a `HEAD` kérést is kiszolgálja.
- Nincs middleware: nincs hozzáférési napló, recover, CORS, hitelesítés, rate limit, tömörítés.

### 4.5 API-végpontok

Közös szabályok:

- Válasz: `Content-Type: application/json; charset=utf-8`. A JSON után záró újsor áll (`json.Encoder`).
- Hibaformátum mindig: `{"error":"<üzenet>"}`. Az üzenetek magyarok.
- Kéréstörzs: `json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))` és `DisallowUnknownFields()`. Bármilyen dekódolási hiba 400 `{"error":"érvénytelen JSON"}`: hibás JSON, ismeretlen mező, 64 KiB-nál nagyobb törzs (nem 413), üres törzs. A `Content-Type` fejlécet nem ellenőrzi. A szerver egyetlen `Decode` hívással csak az első JSON-értéket olvassa, az utána következő maradékot nem vizsgálja (a `{"title":"a"} xyz` törzs elfogadott).
- Váratlan hiba: a részlet a naplóba kerül (`kérés sikertelen`), a kliens 500 `{"error":"belső hiba"}` választ kap.
- Azonosító: `strconv.ParseInt(id, 10, 64)`. Parse-hiba vagy `id < 1`: 400 `{"error":"érvénytelen azonosító"}`.

| Metódus és út | Kérés | Siker | Hibák |
|---|---|---|---|
| `GET /healthz` | nincs | 200 `{"status":"ok"}` | 500 `belső hiba` (adatbázis-ping 2 s időkorláttal) |
| `GET /api/todos` | nincs | 200, `Todo` tömb, legújabb elöl; üresen `[]` | 500 |
| `POST /api/todos` | `{"title": string}` | 201, a létrehozott `Todo` | 400 `érvénytelen JSON`; 400 `a cím 1-200 karakter lehet`; 500 |
| `PATCH /api/todos/{id}` | `{"done": bool}` | 200, a frissített `Todo` | 400 `érvénytelen azonosító`; 400 `érvénytelen JSON`; 400 `a done mező kötelező`; 404 `nincs ilyen teendő`; 500 |
| `DELETE /api/todos/{id}` | nincs | 204, törzs nélkül | 400 `érvénytelen azonosító`; 404 `nincs ilyen teendő`; 500 |

A `title` értékét a szerver `strings.TrimSpace`-szel vágja, majd a karakterszámot (`utf8.RuneCountInString`) ellenőrzi: 1–200. A `done` mező típusa `*bool`, ezért a hiánya felismerhető.

Példák (az értékek szemléltetők):

```http
GET /api/todos

HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

[{"id":2,"title":"Gateway firmware frissítése","done":false,"createdAt":"2026-11-06T09:15:00.123Z"},{"id":1,"title":"Mérések exportja","done":true,"createdAt":"2026-11-05T14:32:08Z"}]
```

```http
POST /api/todos
Content-Type: application/json

{"title":"Gateway firmware frissítése"}

HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8

{"id":2,"title":"Gateway firmware frissítése","done":false,"createdAt":"2026-11-06T09:15:00.123Z"}
```

```http
POST /api/todos
Content-Type: application/json

{"title":"   "}

HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8

{"error":"a cím 1-200 karakter lehet"}
```

```http
PATCH /api/todos/2
Content-Type: application/json

{"done":true}

HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

{"id":2,"title":"Gateway firmware frissítése","done":true,"createdAt":"2026-11-06T09:15:00.123Z"}
```

```http
PATCH /api/todos/99
Content-Type: application/json

{"done":true}

HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8

{"error":"nincs ilyen teendő"}
```

```http
DELETE /api/todos/2

HTTP/1.1 204 No Content
```

```http
GET /api/nemletezik

HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8

{"error":"nincs ilyen végpont"}
```

Windows PowerShell 5.1 alatt a `curl` az `Invoke-WebRequest` aliasa. Próbához a `curl.exe` vagy az `Invoke-RestMethod http://localhost:8080/api/todos` használható.

### 4.6 Hibakezelés és naplózás

- A hibák kontextussal csomagolva: `fmt.Errorf("<magyar kontextus>: %w", err)`.
- A store `ErrNotFound` sentinel hibát ad (`errors.New("not found")`). Az api `errors.Is`-szel 404-re képezi. Minden más hiba 500 `belső hiba`, a részlet csak a naplóba kerül.
- Tranzakcióknál `defer tx.Rollback()`.
- A válaszírás (`Encode`, `Write`) hibáit a kód szándékosan eldobja (`_ =`).
- A `run` hibát ad vissza, a `main` naplóz és `os.Exit(1)`.

| Napló | Szint | Mezők |
|---|---|---|
| `a szerver elindult` | Info | `addr`, `data`, `version` |
| `leállítás` | Info | |
| `kérés sikertelen` | Error | `err` |
| `a szerver leállt` | Error | `err` |

Kérésszintű hozzáférési napló nincs.

### 4.7 SPA-kiszolgálás és gyorsítótár

Induláskor a kezelő memóriába olvassa az `index.html`-t (`fs.ReadFile(ui, "index.html")`). Ha ez nem sikerül, az `api.New` hibát ad, és a szerver nem indul el. A fájlkiszolgáló: `files := http.FileServerFS(ui)`.

A kérés feldolgozása:

```go
name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
if info, err := fs.Stat(ui, name); err == nil && !info.IsDir() && name != "index.html" {
	if strings.HasPrefix(name, "_app/immutable/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	files.ServeHTTP(w, r)
	return
}
// Hiányzó verziózott fájlnál a 404 jobb, mint a HTML válasz.
if strings.HasPrefix(name, "_app/") {
	http.NotFound(w, r)
	return
}
w.Header().Set("Content-Type", "text/html; charset=utf-8")
w.Header().Set("Cache-Control", "no-cache")
_, _ = w.Write(index)
```

| Kérés | Válasz | `Cache-Control` |
|---|---|---|
| Létező fájl az `_app/immutable/` alatt | a fájl | `public, max-age=31536000, immutable` |
| Egyéb létező fájl (pl. `robots.txt`, `favicon.svg`) | a fájl | nincs explicit fejléc, a `FileServer` alapviselkedése |
| Hiányzó fájl az `_app/` alatt | 404, `text/plain` | |
| `/`, `/index.html`, bármely más útvonal, bármely metódus | 200, `index.html` | `no-cache` |

Az `_app/immutable/` fájlok nevében hash van, ezért egy évig gyorsítótárazhatók. Az `index.html` mindig újraellenőrzött.

**Célállapotban** a fontok a `static/fonts/` alól a `build/fonts/` alá kerülnek. Hogy ők is egyéves gyorsítótárat kapjanak, a feltételt bővíteni kell:

```go
if strings.HasPrefix(name, "_app/immutable/") || strings.HasPrefix(name, "fonts/") {
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
}
```

Ez a bővítés a mostani kódban nincs benne. A fontfájlok nevében nincs hash, ezért fontcsere után a fájlnevet is változtatni kell.

### 4.8 Beágyazás

`web/embed.go`:

```go
// Package web a lefordított SvelteKit-felületet ágyazza a binárisba.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var files embed.FS

// Files a build mappa tartalmát adja vissza, a gyökérben az index.html-lel.
func Files() (fs.FS, error) {
	return fs.Sub(files, "build")
}
```

- Az `all:` előtag kötelező. A SvelteKit az `_app` mappába ír, a `go:embed` pedig alapból kihagyja az aláhúzással és ponttal kezdődő neveket.
- A `web/build` mappának fordításkor léteznie kell. Ha nincs, a `go build`, a `go run` és a `go vet` is hibát ad. Első alkalommal ezért előbb `npm ci` és `npm run build` kell a `web/` mappában.
- A migrációk is be vannak ágyazva: `internal/store/store.go`, `//go:embed migrations/*.sql`, `var migrations embed.FS`.

### 4.9 A backend teljes forrása

A 4.1–4.8 és az 5. fejezet leírásai ezekre a fájlokra vonatkoznak. A `web/embed.go` a 4.8-ban, a migráció az 5.3-ban teljes.

`cmd/server/main.go` (teljes):

```go
// A webapp egyetlen binárisban szolgálja ki a felületet és az API-t.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/webapp/internal/api"
	"example.com/webapp/internal/store"
	"example.com/webapp/web"
)

// version értékét a build állítja be: -ldflags "-X main.version=...".
var version = "dev"

func main() {
	addr := flag.String("addr", env("APP_ADDR", ":8080"), "figyelési cím")
	dataDir := flag.String("data", env("APP_DATA_DIR", "./data"), "adatkönyvtár")
	healthcheck := flag.Bool("healthcheck", false, "az élő szerver ellenőrzése, majd kilépés")
	showVersion := flag.Bool("version", false, "verzió kiírása, majd kilépés")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}
	if *healthcheck {
		os.Exit(checkHealth(*addr))
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(*addr, *dataDir, log); err != nil {
		log.Error("a szerver leállt", "err", err)
		os.Exit(1)
	}
}

func run(addr, dataDir string, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return fmt.Errorf("adatkönyvtár: %w", err)
	}
	st, err := store.Open(ctx, dataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	ui, err := web.Files()
	if err != nil {
		return err
	}
	handler, err := api.New(st, ui, log)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errs := make(chan error, 1)
	go func() {
		log.Info("a szerver elindult", "addr", addr, "data", dataDir, "version", version)
		errs <- srv.ListenAndServe()
	}()

	select {
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Info("leállítás")
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdown)
}

// checkHealth a konténer HEALTHCHECK-jéhez kell, mert a futtató image-ben nincs curl.
func checkHealth(addr string) int {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 1
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	client := http.Client{Timeout: 3 * time.Second}
	res, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return 1
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
```

`internal/api/api.go` (teljes):

```go
// Package api a HTTP-végpontokat és a felület kiszolgálását tartalmazza.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/webapp/internal/store"
)

const maxBodyBytes = 1 << 16

// New összeállítja a teljes HTTP-kezelőt.
func New(st *store.Store, ui fs.FS, log *slog.Logger) (http.Handler, error) {
	spa, err := spaHandler(ui)
	if err != nil {
		return nil, err
	}
	h := &handlers{store: st, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /api/todos", h.listTodos)
	mux.HandleFunc("POST /api/todos", h.createTodo)
	mux.HandleFunc("PATCH /api/todos/{id}", h.updateTodo)
	mux.HandleFunc("DELETE /api/todos/{id}", h.deleteTodo)
	// Az ismeretlen API-útvonal JSON-hibát kap, nem az index.html-t.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "nincs ilyen végpont")
	})
	mux.Handle("/", spa)
	return mux, nil
}

type handlers struct {
	store *store.Store
	log   *slog.Logger
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.store.Ping(ctx); err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) listTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := h.store.ListTodos(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, todos)
}

func (h *handlers) createTodo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	title := strings.TrimSpace(body.Title)
	if n := utf8.RuneCountInString(title); n < 1 || n > 200 {
		writeError(w, http.StatusBadRequest, "a cím 1-200 karakter lehet")
		return
	}
	todo, err := h.store.CreateTodo(r.Context(), title)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, todo)
}

func (h *handlers) updateTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Done *bool `json:"done"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if body.Done == nil {
		writeError(w, http.StatusBadRequest, "a done mező kötelező")
		return
	}
	todo, err := h.store.SetTodoDone(r.Context(), id, *body.Done)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "nincs ilyen teendő")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (h *handlers) deleteTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	err := h.store.DeleteTodo(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "nincs ilyen teendő")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) fail(w http.ResponseWriter, err error) {
	h.log.Error("kérés sikertelen", "err", err)
	writeError(w, http.StatusInternalServerError, "belső hiba")
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "érvénytelen azonosító")
		return 0, false
	}
	return id, true
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "érvénytelen JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// spaHandler a statikus fájlokat adja, minden más útvonalra az index.html-t.
func spaHandler(ui fs.FS) (http.Handler, error) {
	index, err := fs.ReadFile(ui, "index.html")
	if err != nil {
		return nil, err
	}
	files := http.FileServerFS(ui)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if info, err := fs.Stat(ui, name); err == nil && !info.IsDir() && name != "index.html" {
			if strings.HasPrefix(name, "_app/immutable/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			files.ServeHTTP(w, r)
			return
		}
		// Hiányzó verziózott fájlnál a 404 jobb, mint a HTML válasz.
		if strings.HasPrefix(name, "_app/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	}), nil
}
```

`internal/store/store.go` (teljes):

```go
// Package store az SQLite-adatbázist és az adatelérést kezeli.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"sort"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

// ErrNotFound jelzi, ha a kért sor nem létezik.
var ErrNotFound = errors.New("not found")

// Todo egy teendő.
type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}

// Store az adatbázis-kapcsolatot tartja.
type Store struct {
	db *sql.DB
}

// Open megnyitja vagy létrehozza az adatbázist, majd lefuttatja a migrációkat.
func Open(ctx context.Context, dataDir string) (*Store, error) {
	path := filepath.Join(dataDir, "app.db")
	query := url.Values{"_pragma": {
		"journal_mode(WAL)",
		"synchronous(NORMAL)",
		"busy_timeout(5000)",
		"foreign_keys(1)",
	}}
	db, err := sql.Open("sqlite", "file:"+path+"?"+query.Encode())
	if err != nil {
		return nil, fmt.Errorf("adatbázis megnyitása: %w", err)
	}
	// Az SQLite egyszerre egy írót enged: egy kapcsolattal elkerülhető a SQLITE_BUSY.
	db.SetMaxOpenConns(1)

	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close lezárja az adatbázist.
func (s *Store) Close() error { return s.db.Close() }

// Ping ellenőrzi, hogy az adatbázis elérhető-e.
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// migrate a még le nem futott fájlokat alkalmazza, sorrendben, tranzakcióban.
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name       TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`); err != nil {
		return fmt.Errorf("migrációs tábla: %w", err)
	}

	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	for _, name := range names {
		var applied int
		err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, name).Scan(&applied)
		if err != nil {
			return fmt.Errorf("migráció ellenőrzése: %w", err)
		}
		if applied > 0 {
			continue
		}
		if err := applyMigration(ctx, db, name); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, name string) error {
	body, err := migrations.ReadFile(name)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (name) VALUES (?)`, name); err != nil {
		return err
	}
	return tx.Commit()
}

const todoColumns = `id, title, done, created_at`

func scanTodo(row interface{ Scan(...any) error }) (Todo, error) {
	var t Todo
	var created string
	if err := row.Scan(&t.ID, &t.Title, &t.Done, &created); err != nil {
		return Todo{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Todo{}, fmt.Errorf("created_at: %w", err)
	}
	t.CreatedAt = parsed
	return t, nil
}

// ListTodos a teendőket adja vissza, a legújabbal kezdve.
func (s *Store) ListTodos(ctx context.Context) ([]Todo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+todoColumns+` FROM todos ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := []Todo{}
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, rows.Err()
}

// CreateTodo új teendőt ment.
func (s *Store) CreateTodo(ctx context.Context, title string) (Todo, error) {
	row := s.db.QueryRowContext(ctx,
		`INSERT INTO todos (title) VALUES (?) RETURNING `+todoColumns, title)
	return scanTodo(row)
}

// SetTodoDone a teendő állapotát módosítja.
func (s *Store) SetTodoDone(ctx context.Context, id int64, done bool) (Todo, error) {
	row := s.db.QueryRowContext(ctx,
		`UPDATE todos SET done = ? WHERE id = ? RETURNING `+todoColumns, done, id)
	t, err := scanTodo(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Todo{}, ErrNotFound
	}
	return t, err
}

// DeleteTodo törli a teendőt.
func (s *Store) DeleteTodo(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM todos WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
```

`internal/store/driver.go` (teljes):

```go
package store

// Tiszta Go-s SQLite-driver: "sqlite" néven regisztrál, nem igényel CGO-t.
import _ "modernc.org/sqlite"
```

---

## 5. Adatréteg

### 5.1 Driver és kapcsolat

- Driver: `modernc.org/sqlite`, tiszta Go. Regisztráció az `internal/store/driver.go`-ban: `import _ "modernc.org/sqlite"`. Regisztrált név: `sqlite`.
- Adatbázisfájl: `filepath.Join(dataDir, "app.db")`.
- WAL módban a fájl mellett `app.db-wal` és `app.db-shm` is létrejön.

```go
query := url.Values{"_pragma": {
	"journal_mode(WAL)",
	"synchronous(NORMAL)",
	"busy_timeout(5000)",
	"foreign_keys(1)",
}}
db, err := sql.Open("sqlite", "file:"+path+"?"+query.Encode())
if err != nil {
	return nil, fmt.Errorf("adatbázis megnyitása: %w", err)
}
// Az SQLite egyszerre egy írót enged: egy kapcsolattal elkerülhető a SQLITE_BUSY.
db.SetMaxOpenConns(1)
```

A létrejövő DSN:

```
file:<dataDir>/app.db?_pragma=journal_mode%28WAL%29&_pragma=synchronous%28NORMAL%29&_pragma=busy_timeout%285000%29&_pragma=foreign_keys%281%29
```

| Pragma | Érték | Hatás |
|---|---|---|
| `journal_mode` | `WAL` | Írás előtti napló |
| `synchronous` | `NORMAL` | WAL mellett szokásos tartóssági szint |
| `busy_timeout` | `5000` | 5 s várakozás zárolt adatbázisnál |
| `foreign_keys` | `1` | Idegen kulcsok ellenőrzése |

Tudnivalók:

- Az `sql.Open` lusta: a kapcsolódási és DSN-hibák a migráció első lekérdezésénél derülnek ki, nem az `Open` hívásnál. Ismeretlen drivernévnél (hiányzó `driver.go` import) maga az `sql.Open` ad hibát.
- A `SetMaxOpenConns(1)` miatt az olvasások is sorba rendeződnek. Egy még nyitott `rows` bejárása közben nem szabad új lekérdezést indítani, mert a program megakad. Nagyobb olvasási párhuzamossághoz külön olvasó- és írókapcsolat-készlet kell.
- Az adatkönyvtár helyi lemezen legyen. Hálózati fájlrendszeren, OneDrive-ban vagy megosztáson a WAL nem megbízható.

A store API-ja: `Open(ctx, dataDir) (*Store, error)`, `Close() error`, `Ping(ctx) error` (`db.PingContext`), `ListTodos`, `CreateTodo`, `SetTodoDone`, `DeleteTodo`. Ha a migráció hibát ad, az `Open` lezárja az adatbázist, és a hibát adja vissza.

### 5.2 Migráció

Működés induláskor (`migrate`):

1. A nyilvántartó tábla létrehozása:
   ```sql
   CREATE TABLE IF NOT EXISTS schema_migrations (
       name TEXT PRIMARY KEY,
       applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
   )
   ```
   Hiba: `migrációs tábla: %w`.
2. `fs.Glob(migrations, "migrations/*.sql")`, majd `sort.Strings`: név szerinti sorrend.
3. Minden fájlra: `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`. Ha már szerepel, kimarad. Hiba: `migráció ellenőrzése: %w`.
4. Alkalmazás egy tranzakcióban: a teljes fájl egyetlen `ExecContext` hívásban, majd `INSERT INTO schema_migrations (name) VALUES (?)`, végül `Commit`. Hiba: `<név>: %w`.

- A `name` oszlop a teljes beágyazott útvonal, például `migrations/0001_create_todos.sql`.
- Csak előre irányú migráció van, down migráció nincs. Visszalépéshez a frissítés előtti mentés kell.
- Új sémaváltozáshoz elég egy új, sorszámozott fájl az `internal/store/migrations/` alá, például `0002_add_due_date.sql`.

### 5.3 Séma

`internal/store/migrations/0001_create_todos.sql`:

```sql
CREATE TABLE todos (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT    NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    done       INTEGER NOT NULL DEFAULT 0 CHECK (done IN (0, 1)),
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_todos_created_at ON todos (created_at DESC);
```

A cím hosszát az SQL `CHECK` is ellenőrzi, az API karakterszámlálása mellett.

### 5.4 Lekérdezések

Oszloplista: `todoColumns = "id, title, done, created_at"`.

| Művelet | SQL | Nem létező sor |
|---|---|---|
| `ListTodos` | `SELECT id, title, done, created_at FROM todos ORDER BY created_at DESC, id DESC` | |
| `CreateTodo` | `INSERT INTO todos (title) VALUES (?) RETURNING id, title, done, created_at` | |
| `SetTodoDone` | `UPDATE todos SET done = ? WHERE id = ? RETURNING id, title, done, created_at` | `sql.ErrNoRows` → `ErrNotFound` |
| `DeleteTodo` | `DELETE FROM todos WHERE id = ?` | `RowsAffected == 0` → `ErrNotFound` |

A lista üres esetben is `[]Todo{}`, így a JSON `[]`, nem `null`. A kód sima `database/sql`; sok statikus lekérdezésnél az sqlc bevezethető.

### 5.5 Modellek

```go
type Store struct {
	db *sql.DB
}

var ErrNotFound = errors.New("not found")

type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}
```

- A `scanTodo` a `created_at` szöveget `time.Parse(time.RFC3339Nano, created)`-del olvassa. Hiba: `created_at: %w`. A `done` SQLite INTEGER, `bool`-ba olvasva.
- A JSON-ban a `createdAt` RFC 3339 időbélyeg, UTC (a Go `time.Time` RFC 3339 Nano kódolása). A tört másodperc 0–3 jegyű, mert az adatbázis ezredmásodpercig tárol, és a záró nullák elmaradnak: `2026-11-05T14:32:08Z`, `2026-11-06T09:15:00.123Z`.
- Kérésstruktúrák (névtelenek az api-ban): `` struct{ Title string `json:"title"` } `` és `` struct{ Done *bool `json:"done"` } ``.

---

## 6. Frontend

### 6.1 SvelteKit-konfiguráció

A SvelteKit 3 konfigurációja a `vite.config.ts`-ben él, a `sveltekit()` plugin opcióiként. `svelte.config.js` nincs. Az alap importalias a `#lib`, amely a `package.json` `imports` mezőjéből jön (Node subpath import):

```json
"imports": {
	"#lib": "./src/lib/index.js",
	"#lib/*": "./src/lib/*"
}
```

A kód kiterjesztéssel importál: `#lib/utils.ts`, `#lib/api.ts`, `#lib/components/ui/button/index.ts`, `#lib/assets/favicon.svg`.

**Váz, `web/vite.config.ts` (teljes):**

```ts
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			// Az index.html a kliensoldali útvonalak közös belépési pontja.
			adapter: adapter({ fallback: 'index.html' })
		})
	],
	server: {
		// Fejlesztés közben a Vite az API-hívásokat a Go-szervernek adja tovább.
		proxy: { '/api': 'http://127.0.0.1:8080' }
	}
});
```

**Célállapot, `web/vite.config.ts` (teljes):** a shadcn-svelte CLI `$lib` útvonalakat generál, ezért a `$lib` aliast vissza kell kapcsolni. A runes-kényszerítés és a fejlesztői proxy megmarad.

```ts
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			// Az index.html a kliensoldali útvonalak közös belépési pontja.
			adapter: adapter({ fallback: 'index.html' }),
			// A shadcn-svelte $lib útvonalakat generál.
			alias: { $lib: 'src/lib' }
		})
	],
	server: {
		// Fejlesztés közben a Vite az API-hívásokat a Go-szervernek adja tovább.
		proxy: { '/api': 'http://127.0.0.1:8080' }
	}
});
```

A `#lib` importok ezután is működnek, a két alias együtt él.

| Beállítás | Érték | Indok |
|---|---|---|
| Adapter | `@sveltejs/adapter-static`, `fallback: 'index.html'` | SPA: minden kliensoldali útvonal az `index.html`-re esik |
| Build kimenet | `web/build` | Ezt ágyazza be a Go |
| Runes mód | a saját kódra kényszerítve, a `node_modules` kivétel | Svelte 5 szintaxis kötelező |
| Fejlesztői proxy | `/api` → `http://127.0.0.1:8080` | A proxy IPv4 loopbackre mutat, így nem számít, hogy a `localhost` IPv4-re vagy IPv6-ra oldódik |
| Vite port | nincs megadva, alapértelmezett 5173 | |
| Weboldal (nem webapp) esetén | teljes prerender, `fallback` nélkül | |

`web/tsconfig.json`:

```json
{ "extends": "$app/tsconfig", "compilerOptions": { "strict": true }, "include": ["src", "vite.config.ts"] }
```

`web/package.json` mezői: `"name": "web"`, `"version": "0.0.1"`, `"private": true`, `"type": "module"` (ES-modul csomag, az ESM-konfigurációhoz kell), a fenti `imports`, a 2.2 függőségei és a szkriptek:

| Szkript | Parancs |
|---|---|
| `dev` | `vite dev` |
| `build` | `vite build` |
| `preview` | `vite preview` |
| `prepare` | `svelte-kit sync \|\| echo ''` (az `npm ci` után automatikusan lefut) |
| `check` | `svelte-kit sync && svelte-check --tsconfig ./tsconfig.json` |
| `check:watch` | `svelte-kit sync && svelte-check --tsconfig ./tsconfig.json --watch` |

**Váz, `web/package.json` (teljes):**

```json
{
	"name": "web",
	"version": "0.0.1",
	"private": true,
	"type": "module",
	"imports": {
		"#lib": "./src/lib/index.js",
		"#lib/*": "./src/lib/*"
	},
	"scripts": {
		"dev": "vite dev",
		"build": "vite build",
		"preview": "vite preview",
		"prepare": "svelte-kit sync || echo ''",
		"check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json",
		"check:watch": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json --watch"
	},
	"devDependencies": {
		"@sveltejs/adapter-static": "^4.0.1",
		"@sveltejs/kit": "^3.0.0",
		"@sveltejs/vite-plugin-svelte": "^7.2.0",
		"@tailwindcss/vite": "^4.3.4",
		"svelte": "^5.57.2",
		"svelte-check": "^4.6.0",
		"tailwindcss": "^4.3.4",
		"typescript": "^6.0.3",
		"vite": "^8.3.0"
	},
	"dependencies": {
		"@lucide/svelte": "^1.69.0",
		"clsx": "^2.1.1",
		"tailwind-merge": "^3.7.0",
		"tailwind-variants": "^4.0.1"
	}
}
```

### 6.2 SSR és prerender

`web/src/routes/+layout.ts` (teljes, váz és célállapot):

```ts
// Tisztán kliensoldali alkalmazás: nincs SSR és előrenderelés.
export const ssr = false;
export const prerender = false;
```

Weboldalnál (nem webalkalmazásnál) ehelyett `export const prerender = true;` kell, és az adapter `fallback` nélkül fut.

### 6.3 app.html

**Váz, `web/src/app.html` (teljes):**

```html
<!doctype html>
<html lang="hu">
	<head>
		<meta charset="utf-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		<meta name="text-scale" content="scale" />
		%sveltekit.head%
	</head>
	<body data-sveltekit-preload-data="hover">
		<div style="display: contents">%sveltekit.body%</div>
	</body>
</html>
```

**Célállapot, `web/src/app.html` (teljes csere):**

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

| Meta | Érték | Jelentés |
|---|---|---|
| `theme-color` | `#071016` | A sötét téma alapszíne a böngésző kereténél |

### 6.4 Téma és villanásmentes sötét indítás

**Váz:** nincs mode-watcher és nincs témaváltó. A sötét mód csak a rendszerbeállítást követi: `@media (prefers-color-scheme: dark)` írja felül a `:root` változókat. Az alap a világos. A `@custom-variant dark (&:is(.dark *))` definiálva van, de `.dark` osztályt semmi nem állít be.

**Célállapot:** a sötét az alap téma, a világos a teljes értékű párja. A működés három részből áll:

1. A `<html class="dark">` az első rajzolástól sötét, így nincs villanás.
2. Az inline szkript az első indításkor `dark` értékre állítja a `mode-watcher-mode` kulcsot a `localStorage`-ban. Erre azért van szükség, mert `ssr = false` mellett a `defaultMode` magában nem hat. A `try`/`catch` a tiltott tárhelyet kezeli.
3. A `ModeWatcher` a `+layout.svelte`-ben kezeli a váltást. Világos módban a `light` osztályt teszi a gyökérre.

A stíluslap a sötét változókat a `:root, .dark, [data-theme="dark"]` szelektoron, a világosakat a `.light, [data-theme="light"]` szelektoron adja. A `dark:` Tailwind-variáns (`&:is(.dark *)`) csak a `.dark` elem leszármazottaira illeszkedik, magára a `.dark` elemre nem.

### 6.5 +layout.svelte

**Váz:** importálja a `#lib/assets/favicon.svg`-t (az alapértelmezett Svelte-logó) és a `../app.css`-t, a `LayoutProps` típust használja. A `<svelte:head>` blokkban `<link rel="icon" href={favicon} />` és `<title>Teendők</title>` áll, a tartalom `{@render children()}`.

**Célállapot, `web/src/routes/+layout.svelte` (teljes):**

```svelte
<script lang="ts">
	import '../app.css';
	import { ModeWatcher } from 'mode-watcher';
	let { children } = $props();
</script>

<ModeWatcher defaultMode="dark" lightClassNames={['light']} />
{@render children()}
```

A favicon a célállapotban az `app.html`-ből és a `static/favicon.svg`-ből jön, ezért a `src/lib/assets/favicon.svg` és az importja törölhető. A célállapotú layoutban nincs `<title>`. Az oldalcímet az egyes `+page.svelte` fájlok `<svelte:head><title>…</title></svelte:head>` blokkja adja; a Teendők oldalon: `<svelte:head><title>Teendők</title></svelte:head>`.

### 6.6 API-kliens

Az alapútvonal relatív: `/api`. Fejlesztés közben a Vite proxyja továbbítja a 8080-as portra, élesben ugyanaz a Go-szerver szolgálja ki ugyanazon az originen, ezért CORS nem kell.

`web/src/lib/api.ts` (teljes):

```ts
export type Todo = {
	id: number;
	title: string;
	done: boolean;
	createdAt: string;
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`/api${path}`, {
		...init,
		headers: { 'Content-Type': 'application/json', ...init?.headers }
	});
	if (!res.ok) {
		const body = await res.json().catch(() => null);
		throw new Error(body?.error ?? `HTTP ${res.status}`);
	}
	return res.status === 204 ? (undefined as T) : res.json();
}

export const api = {
	list: () => request<Todo[]>('/todos'),
	create: (title: string) =>
		request<Todo>('/todos', { method: 'POST', body: JSON.stringify({ title }) }),
	setDone: (id: number, done: boolean) =>
		request<Todo>(`/todos/${id}`, { method: 'PATCH', body: JSON.stringify({ done }) }),
	remove: (id: number) => request<void>(`/todos/${id}`, { method: 'DELETE' })
};
```

A `request<T>(path, init?)` működése:

- `fetch('/api' + path)` `Content-Type: application/json` fejléccel; az `init.headers` felülírhatja.
- Ha `!res.ok`: a törzset `res.json().catch(() => null)`-lal olvassa, és `` new Error(body?.error ?? `HTTP ${res.status}`) `` hibát dob. A szerver magyar hibaüzenete így közvetlenül a felületre kerül.
- 204 esetén `undefined`, egyébként `res.json()`.
- Az `init.headers` csak sima objektumként adható meg: `Headers` példány vagy tömb tartalmát a spread elveszíti. A `Content-Type: application/json` minden kérésre rákerül, a törzs nélküliekre (GET, DELETE) is.

### 6.7 Komponensstruktúra

**`src/lib/utils.ts`:**

```ts
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}
```

A `src/lib/index.ts` csak egy megjegyzést tartalmaz, exportja nincs. A `src/app.d.ts` a sablonból jön, minden interfész kikommentezve (váz és célállapot, teljes):

```ts
// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
```

**Váz, `src/lib/components/ui/`:** kézzel írt, shadcn-stílusú komponensek, `tailwind-variants` (`tv`) és `cn()` használatával. `components.json` és Bits UI nincs.

| Komponens | Export (`index.ts`) | Propok | Megjegyzés |
|---|---|---|---|
| `Button` | `Button`, `buttonVariants`. A `ButtonVariant` és `ButtonSize` típust a `button.svelte` `<script module>` blokkja exportálja | `HTMLButtonAttributes & { variant?, size? }`, alap `type='button'` | Variánsok: `default`, `ghost`. Méretek: `default` (`h-10 px-4`), `icon` (`size-9`) |
| `Card` | `Card` | `HTMLAttributes<HTMLDivElement>` | Egyetlen `div`, alkomponensek nélkül |
| `Input` | `Input` | `HTMLInputAttributes`, `value = $bindable()` | `bind:value` támogatott |

A pontos osztálylisták (`Button` alap és variánsok, `Card`, `Input`) a 6.12 teljes forrásában vannak; a shadcn-svelte CLI-s cserével megszűnnek.

**Váz, `src/routes/+page.svelte`:** egyetlen útvonal (`/`).

- Állapot runes-szal: `todos = $state<Todo[]>([])`, `title = $state('')`, `error = $state('')`, `loading = $state(true)`, `remaining = $derived(todos.filter((t) => !t.done).length)`.
- Az `attempt(action)` törli a hibát, és elkapja a kivételt: `error = e instanceof Error ? e.message : 'Ismeretlen hiba'`.
- `onMount`: `api.list()`, majd `loading = false`; a `loading` a betöltés hibájától függetlenül `false` lesz.
- Űrlap: `<Input bind:value={title} placeholder="Új teendő" aria-label="Új teendő" maxlength={200} />` és `<Button type="submit">` „Hozzáadás” felirattal. A törlőgomb `variant="ghost" size="icon"`, a kész állapot natív `checkbox`.
- Hozzáadás: `preventDefault`, vágás, üres címnél nincs kérés; az új elem a lista elejére kerül, sikeres létrehozás után `title = ''`.
- Elrendezés: `<main class="mx-auto flex w-full max-w-xl flex-col gap-6 px-4 py-12">`; `h1`: `text-2xl font-semibold tracking-tight`; a lista `ul.divide-y`, a sorok `flex items-center gap-3 px-4 py-2`; a checkbox `size-4 accent-primary`; a kész elem `text-muted-foreground line-through`.
- Kész állapot: `api.setDone(todo.id, !todo.done)`, csere azonosító szerint. Törlés: `api.remove`, szűrés azonosító szerint.
- Nincs optimista frissítés.
- Ikonok: `import { Plus, Trash2 } from '@lucide/svelte'` gyűjtőimporttal, `class="size-4"`, `strokeWidth` nélkül (alapérték 2). Ez eltér a 8.2 konvenciójától; a célállapotban `import Plus from '@lucide/svelte/icons/plus'`, `import Trash2 from '@lucide/svelte/icons/trash-2'` és `strokeWidth={1.5}`.
- Szövegek: „Teendők”, „{remaining} nyitott”, „Új teendő”, „Hozzáadás”, „Betöltés…”, „Még nincs teendő.”, `aria-label`: „Kész: {cím}”, „Törlés: {cím}”. A hiba `<p role="alert">` elemben jelenik meg.

**Célállapot:** az `ui/` mappát a shadcn-svelte CLI tölti fel (6.8), a márkakomponensek a `src/lib/components/brand/` alá kerülnek (6.10).

### 6.8 shadcn-svelte

A shadcn-svelte változónevei és a design tokenek nevei azonosak, ezért a CLI-vel hozzáadott komponensek módosítás nélkül átveszik az arculatot. Amit a shadcn alapból másképp csinál, azt az új `src/app.css` írja felül:

- a kártya és a dialógus `rounded-xl` sugara `radius-lg` (6 px) lesz,
- a gomb és a mező `shadow-xs` árnyéka eltűnik,
- a 3 px-es félátlátszó fókuszgyűrű helyett 2 px-es tömör, 2 px-re eltolt gyűrű jelenik meg.

Hozzáadandó komponensek: `button input label select checkbox switch table tabs dialog alert card badge sidebar`. A diagramhoz a shadcn-svelte `Chart` (LayerChart) a tervezett komponens, a sűrű adattáblához igény szerint TanStack Table. Ezek nincsenek a fenti listában.

`web/components.json` (teljes):

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

- A `style`, `iconLibrary`, `menuColor` és `menuAccent` mező az `init` kérdéseire adott válasz.
- A `baseColor: "mist"` a shadcn-svelte saját, kékesszürke alapszíne. Csak a kiindulási változókat adja, amelyeket az új `src/app.css` úgyis felülír. Nem azonos az arculat `mist-*` szövegszín-skálájával.
- A globális CSS útvonala `src/app.css`, egyezzen a `+layout.svelte` importjával.

### 6.9 Tailwind CSS v4

- CSS-first konfiguráció. Nincs `tailwind.config.js` és nincs PostCSS-konfiguráció. A Tailwind a `@tailwindcss/vite` pluginon keresztül fut, a `src/app.css` importálja (`@import 'tailwindcss'`), a tokeneket `@theme inline` blokk képezi le utility-kre.
- Minden színtoken utility: `bg-card`, `text-muted-foreground`, `border-border`, `text-success`. A skálák is: `bg-signal-700`.
- Típusstílus osztályként: `class="display-xl"`, `class="data-lg"`. Márkaosztályok: `kx-eyebrow`, `kx-panel`, `kx-ticks`.
- Csak `font-sans` (Hanken Grotesk) és `font-mono` (Geist Mono) használható. A stíluslap csak ezt a kettőt írja felül (`--font-*: initial` nincs benne), ezért a Tailwind alap `font-serif` utility-je technikailag megmarad, de tilos.
- Sarok: `rounded-sm` 3 px, `rounded-md` 4 px, `rounded-lg` 6 px. Az `xl` és nagyobb fokozat is 6 px, hogy a shadcn-komponensek ne kapjanak nagyobb sugarat.
- Árnyék: `shadow-2xs`, `shadow-xs`, `shadow-sm` üres (`0 0 #0000`), `shadow-md` és `shadow-lg` a `shadow-popover` értéke.
- Térköz: a Tailwind alapskála pontosan a `space-*` tokenek (4 px-es rács, `p-4` = `space-4`).
- Nyers hex értéket nem írunk a kódba.

**Az új `src/app.css` szerkezete (célállapot).** A fájlt nem kézzel írjuk: a teljes tartalma a GRAPHICS.md 13.2 (`src/app.css`) kódblokkja, ezt kell változatlanul a `src/app.css` helyére írni, a shadcn-svelte `init` és `add` után. Első sora: `/* KINEXIUM arculat. Helye: src/app.css, teljes cserével, a shadcn-svelte init után. */`. A szerkezete:

| Rész | Tartalom |
|---|---|
| Importok | `@import "tailwindcss";`, `@import "tw-animate-css";`, `@import "shadcn-svelte/tailwind.css";` |
| `@custom-variant` | `dark (&:is(.dark *))` |
| `@font-face` | `Hanken Grotesk` → `/fonts/HankenGrotesk-Variable.woff2`; `Geist Mono` → `/fonts/GeistMono-Variable.woff2`; mindkettő `font-weight: 100 900`, `font-style: normal`, `font-display: swap` |
| Sötét téma | `:root, .dark, [data-theme="dark"]`: `color-scheme: dark`; a nyers `ink-*`, `mist-*`, `signal-*` skálák, a szemantikus, állapot-, diagram- és oldalsávtokenek, valamint a `--selection`, `--glow` és `--shadow-popover` |
| Világos téma | `.light, [data-theme="light"]`: `color-scheme: light`; a szemantikus, állapot-, diagram- és oldalsávtokeneket, valamint a `--selection`, `--glow` és `--shadow-popover` értékét írja felül; a nyers `ink-*`, `mist-*`, `signal-*` skálákat a sötét blokkból örökli |
| Témafüggetlen tokenek | külön `:root` blokk: `--space-1` … `--space-24` (4–96 px), `--radius: 6px`, `--chamfer-sm: 8px`, `--chamfer-md: 14px`, `--chamfer-lg: 24px`, `--layout-site: 1200px`, `--layout-prose: 640px`, `--layout-sidebar: 240px`, `--layout-topbar: 52px`. A `chamfer-*` és `layout-*` tokenek csak CSS-változók, Tailwind-utility nem készül belőlük; használat: `w-[var(--layout-sidebar)]` |
| `@theme inline` | `--color-*` leképezés minden színre (a skálákra is, továbbá `--color-selection`, `--color-glow`), `--font-sans`, `--font-mono`, `--radius-sm/md/lg/xl/2xl/3xl/4xl/full`, `--shadow-*` |
| `@layer base` | lásd lent |
| `@layer components` | típusosztályok (`.display-xl`, `.display`, `.heading-1..3`, `.body-lg`, `.body`, `.body-sm`, `.label`, `.code`, `.data`, `.data-lg`) és márkaosztályok (`.kx-eyebrow`, `.kx-num`, `.kx-panel*`, `.kx-ticks`, `.kx-dot*`, `.kx-helix` a `.b`, `.f`, `.h`, `.nb`, `.nf`, `.nh` alosztályokkal) |
| Fókusz-felülírás | `[data-slot]:focus-visible`, a shadcn 3 px-es ring utility-jeinek kikapcsolására |

A `@layer base` és a fókusz-felülírás tartalma:

```css
@layer base {
	* {
		@apply border-border;
	}
	body {
		@apply bg-background text-foreground font-sans;
		font-size: 15px;
		line-height: 24px;
		-webkit-font-smoothing: antialiased;
	}
	::selection {
		background: var(--selection);
	}
	:focus-visible {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
	@media (prefers-reduced-motion: reduce) {
		*, ::before, ::after {
			animation-duration: .01ms !important;
			transition-duration: .01ms !important;
		}
	}
}

[data-slot]:focus-visible {
	outline: 2px solid var(--ring);
	outline-offset: 2px;
	--tw-ring-color: transparent;
}
```

A tokenek konkrét értékei, a típusskála és a márkaosztályok megjelenése a GRAPHICS.md-ben vannak.

A váz `src/app.css`-e ideiglenes, semleges oklch-paletta (világos alap, `prefers-color-scheme` sötét változattal, `--radius: 0.625rem`). Ezek az értékek nem KINEXIUM-tokenek, a csere után megszűnnek. A célállapothoz nem kell, a 6.10 szerinti teljes csere felülírja.

**Váz, `web/src/app.css` (teljes):**

```css
@import 'tailwindcss';

@custom-variant dark (&:is(.dark *));

/* shadcn-svelte stílusú tokenek: a design rendszer egy helyen módosítható. */
:root {
	--radius: 0.625rem;
	--background: oklch(0.985 0.002 247);
	--foreground: oklch(0.21 0.03 265);
	--card: oklch(1 0 0);
	--card-foreground: oklch(0.21 0.03 265);
	--primary: oklch(0.45 0.16 265);
	--primary-foreground: oklch(0.98 0.01 265);
	--muted: oklch(0.96 0.005 265);
	--muted-foreground: oklch(0.5 0.02 265);
	--accent: oklch(0.95 0.01 265);
	--accent-foreground: oklch(0.21 0.03 265);
	--destructive: oklch(0.55 0.21 27);
	--border: oklch(0.92 0.006 265);
	--input: oklch(0.92 0.006 265);
	--ring: oklch(0.6 0.13 265);
}

@media (prefers-color-scheme: dark) {
	:root {
		--background: oklch(0.17 0.02 265);
		--foreground: oklch(0.97 0.005 265);
		--card: oklch(0.21 0.025 265);
		--card-foreground: oklch(0.97 0.005 265);
		--primary: oklch(0.72 0.13 265);
		--primary-foreground: oklch(0.17 0.02 265);
		--muted: oklch(0.27 0.025 265);
		--muted-foreground: oklch(0.72 0.02 265);
		--accent: oklch(0.27 0.025 265);
		--accent-foreground: oklch(0.97 0.005 265);
		--destructive: oklch(0.7 0.19 22);
		--border: oklch(1 0 0 / 12%);
		--input: oklch(1 0 0 / 16%);
		--ring: oklch(0.6 0.13 265);
	}
}

@theme inline {
	--radius-md: calc(var(--radius) - 2px);
	--radius-lg: var(--radius);
	--color-background: var(--background);
	--color-foreground: var(--foreground);
	--color-card: var(--card);
	--color-card-foreground: var(--card-foreground);
	--color-primary: var(--primary);
	--color-primary-foreground: var(--primary-foreground);
	--color-muted: var(--muted);
	--color-muted-foreground: var(--muted-foreground);
	--color-accent: var(--accent);
	--color-accent-foreground: var(--accent-foreground);
	--color-destructive: var(--destructive);
	--color-border: var(--border);
	--color-input: var(--input);
	--color-ring: var(--ring);
}

@layer base {
	* {
		@apply border-border;
	}
	body {
		@apply bg-background text-foreground antialiased;
	}
}
```

### 6.10 A design system bekötése

**Az arculat fájljai a projektben** (a `web/` mappához képest):

| Cél a projektben | Tartalom forrása | Mikor |
|---|---|---|
| `src/app.css` | a GRAPHICS.md 13.2 teljes stíluslapja, teljes cserével | a shadcn-svelte `init` és `add` után |
| `components.json` (a SvelteKit-projekt gyökere, azaz `web/`) | a 6.8 teljes tartalma | az `init` után ellenőrizve, eltérés esetén felülírva |
| `src/lib/components/brand/Logo.svelte`, `Panel.svelte`, `Metric.svelte`, `StatusBadge.svelte`, `TickRule.svelte` | a lenti „Márkakomponensek” kódblokkjai | bármikor |
| `static/favicon.svg` | a lenti app ikon SVG | bármikor |
| `static/fonts/HankenGrotesk-Variable.woff2` | bináris, 41 216 bájt, SHA-256 `c38c7a29913400c75678bcb9b6bd016f21cc120f2680db35a37fda6c9f38fa0b` | bármikor |
| `static/fonts/GeistMono-Variable.woff2` | bináris, 38 048 bájt, SHA-256 `4017c5ee04cfc2d6dd089f6faceda312e627514aa083d202742f29b3f9866f29` | bármikor |
| `static/fonts/OFL-Hanken-Grotesk.txt`, `static/fonts/OFL-Geist-Mono.txt` | a lenti „Licencfájlok” fejléce és a SIL Open Font License 1.1 teljes szövege | a fontokkal együtt |

- A fontfájlok binárisak, szövegként nem adhatók meg. A fontok projektjei: Hanken Grotesk: https://github.com/marcologous/hanken-grotesk, Geist Mono: https://github.com/vercel/geist-font. Ellenőrzött letöltési parancs nincs; a használt fájlt a fenti bájtméret és SHA-256 azonosítja.
- Beszerzési feltétel: változó WOFF2, súly (`wght`) 100–900, tartalmazza az ő és ű glifát, és a fenti fájlnéven kerül a `static/fonts/` alá. Ha a beszerzett fájl hash-e eltér a fentitől, ezek a feltételek az elfogadás alapja.
- A licencfájlok helye: `static/fonts/OFL-Hanken-Grotesk.txt` és `static/fonts/OFL-Geist-Mono.txt`. A fontok a binárisba ágyazva továbbadásra kerülnek, ezért az OFL 1.1 szerint a licencszövegnek velük kell maradnia; a `static/fonts/` alatt a buildbe is bekerülnek.

**Licencfájlok.** Mindkét fájl egy fejlécből és utána a SIL Open Font License 1.1 alább megadott, közös teljes szövegéből áll.

`static/fonts/OFL-Hanken-Grotesk.txt` fejléce (a szerzői jogi sor után két üres sor áll):

```text
Copyright 2021 The Hanken Grotesk Project Authors (https://github.com/marcologous/hanken-grotesk)


This Font Software is licensed under the SIL Open Font License, Version 1.1.
This license is copied below, and is also available with a FAQ at:
http://scripts.sil.org/OFL
```

`static/fonts/OFL-Geist-Mono.txt` fejléce (a szerzői jogi sor után egy üres sor áll):

```text
Copyright 2024 The Geist Project Authors (https://github.com/vercel/geist-font.git)

This Font Software is licensed under the SIL Open Font License, Version 1.1.
This license is copied below, and is also available with a FAQ at:
https://openfontlicense.org
```

A fejléc utolsó sora után két üres sor következik, majd a közös licencszöveg:

```text
-----------------------------------------------------------
SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007
-----------------------------------------------------------

PREAMBLE
The goals of the Open Font License (OFL) are to stimulate worldwide
development of collaborative font projects, to support the font creation
efforts of academic and linguistic communities, and to provide a free and
open framework in which fonts may be shared and improved in partnership
with others.

The OFL allows the licensed fonts to be used, studied, modified and
redistributed freely as long as they are not sold by themselves. The
fonts, including any derivative works, can be bundled, embedded, 
redistributed and/or sold with any software provided that any reserved
names are not used by derivative works. The fonts and derivatives,
however, cannot be released under any other type of license. The
requirement for fonts to remain under this license does not apply
to any document created using the fonts or their derivatives.

DEFINITIONS
"Font Software" refers to the set of files released by the Copyright
Holder(s) under this license and clearly marked as such. This may
include source files, build scripts and documentation.

"Reserved Font Name" refers to any names specified as such after the
copyright statement(s).

"Original Version" refers to the collection of Font Software components as
distributed by the Copyright Holder(s).

"Modified Version" refers to any derivative made by adding to, deleting,
or substituting -- in part or in whole -- any of the components of the
Original Version, by changing formats or by porting the Font Software to a
new environment.

"Author" refers to any designer, engineer, programmer, technical
writer or other person who contributed to the Font Software.

PERMISSION & CONDITIONS
Permission is hereby granted, free of charge, to any person obtaining
a copy of the Font Software, to use, study, copy, merge, embed, modify,
redistribute, and sell modified and unmodified copies of the Font
Software, subject to the following conditions:

1) Neither the Font Software nor any of its individual components,
in Original or Modified Versions, may be sold by itself.

2) Original or Modified Versions of the Font Software may be bundled,
redistributed and/or sold with any software, provided that each copy
contains the above copyright notice and this license. These can be
included either as stand-alone text files, human-readable headers or
in the appropriate machine-readable metadata fields within text or
binary files as long as those fields can be easily viewed by the user.

3) No Modified Version of the Font Software may use the Reserved Font
Name(s) unless explicit written permission is granted by the corresponding
Copyright Holder. This restriction only applies to the primary font name as
presented to the users.

4) The name(s) of the Copyright Holder(s) or the Author(s) of the Font
Software shall not be used to promote, endorse or advertise any
Modified Version, except to acknowledge the contribution(s) of the
Copyright Holder(s) and the Author(s) or with their explicit written
permission.

5) The Font Software, modified or unmodified, in part or in whole,
must be distributed entirely under this license, and must not be
distributed under any other license. The requirement for fonts to
remain under this license does not apply to any document created
using the Font Software.

TERMINATION
This license becomes null and void if any of the above conditions are
not met.

DISCLAIMER
THE FONT SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO ANY WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT
OF COPYRIGHT, PATENT, TRADEMARK, OR OTHER RIGHT. IN NO EVENT SHALL THE
COPYRIGHT HOLDER BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
INCLUDING ANY GENERAL, SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL
DAMAGES, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF THE USE OR INABILITY TO USE THE FONT SOFTWARE OR FROM
OTHER DEALINGS IN THE FONT SOFTWARE.
```

`static/favicon.svg` (teljes; az arculat app ikonja: négyzet, a jobb felső és a bal alsó sarok 22 egységgel levágva, a jel 6.4-es vonallal és 6.8 sugarú csomópontokkal):

```html
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" role="img" aria-label="Kinexium"><title>Kinexium</title><path d="M0 0H106L128 22V128H22L0 106Z" fill="#10212a"/><g transform="translate(26 26) scale(1.19)"><path d="M16 8V56M16 38 45 13M27.5 28 45 51" stroke="#d4dcd5" stroke-width="6.4" fill="none"/><circle cx="47" cy="11.5" r="6.8" fill="#77dbed"/><circle cx="47" cy="53" r="6.8" fill="#d4dcd5"/></g></svg>
```

**Új projekt esetén a telepítés sorrendje:**

```sh
npx sv create web --template minimal --types ts --no-add-ons --install npm
cd web
npx sv add tailwindcss="plugins:none"
npm i -D @sveltejs/adapter-static tw-animate-css
npm i mode-watcher @lucide/svelte
npm rm @sveltejs/adapter-auto
```

Ezután a `web/.npmrc` létrehozása ezzel a tartalommal: `engine-strict=true`. Majd a `vite.config.ts` a 6.1 célállapotú változata szerint készül (adapter-static, `fallback: 'index.html'`, `alias: { $lib: 'src/lib' }`). Majd:

```sh
npx svelte-kit sync
npx shadcn-svelte@latest init
npx shadcn-svelte@latest add button input label select checkbox switch table tabs dialog alert card badge sidebar alert-dialog chart
```

Az `alert-dialog` a megerősítő dialógushoz, a `chart` (LayerChart-függőséggel) az idősor-diagramhoz kell (GRAPHICS.md 10.8 és 10.12).

Ezután a `src/app.css` teljes tartalmát a GRAPHICS.md 13.2 kódblokkjára kell cserélni. Utána a lenti, meglévő vázra vonatkozó lista 5–9. lépése következik (fontok és licencfájlok, favicon, márkakomponensek, `app.html` és `+layout.svelte`, Go SPA-kezelő a `fonts/` gyorsítótárral, `npm run check` és `npm run build`), valamint a 6.2 szerinti `+layout.ts`.

- Az `init` megkérdezi a globális CSS útvonalát: `src/app.css`-t kell megadni.
- Az `init` ebbe a fájlba írja a `@import "shadcn-svelte/tailwind.css"` sort, a `:root` változókat és a `@layer base` blokkot, és telepíti a `shadcn-svelte` csomagot. Az új `src/app.css` mindezt tartalmazza, ezért utána teljes cserével kerül a helyére. Ezt a sorrendet tartani kell, mert az `init` felülírja a `src/app.css`-t.

**A meglévő váz (`web/`) esetén** a `sv create` és a `sv add tailwindcss` lépés kimarad, mert a Tailwind v4 és az adapter-static már telepítve van. A váz `package.json`-ja a következő paranccsorral készült, ez a meglévő állapot:

```sh
npx sv create web --template minimal --types ts --no-add-ons --install npm
cd web
npm i -D @sveltejs/adapter-static tailwindcss @tailwindcss/vite
npm i clsx tailwind-merge tailwind-variants @lucide/svelte
npm rm @sveltejs/adapter-auto
```

A bekötéshez a vázon a következő lépések kellenek, ebben a sorrendben:

1. `npm i -D tw-animate-css` és `npm i mode-watcher`.
2. A `vite.config.ts` cseréje a 6.1 célállapotú változatára (`$lib` alias).
3. `npx svelte-kit sync`, `npx shadcn-svelte@latest init`, `npx shadcn-svelte@latest add button input label select checkbox switch table tabs dialog alert card badge sidebar alert-dialog chart`. Az `add` a kézzel írt `ui/button`, `ui/card`, `ui/input` helyére a CLI változatait teszi; a `+page.svelte` importjait és használatát ehhez kell igazítani. A váz mostani importjai: `import { Button } from '#lib/components/ui/button/index.ts'`, `import { Card } from '#lib/components/ui/card/index.ts'`, `import { Input } from '#lib/components/ui/input/index.ts'`. A használt propok: `Button`: `type="submit"`, `variant="ghost"`, `size="icon"`, `onclick`, `aria-label`; `Input`: `bind:value`, `placeholder`, `aria-label`, `maxlength={200}`; `Card`: alkomponens nélkül. A CLI-s változat exportjait és propjait az `add` után ellenőrizni kell, ez nincs kipróbálva. Utána a `web/components.json` a 6.8 szerinti tartalomra cserélendő, ha eltér.
4. A `src/app.css` teljes cseréje a GRAPHICS.md 13.2 tartalmára.
5. A fontok és a licencfájlok a `static/fonts/` alá, a favicon `static/favicon.svg` néven (a fenti táblázat szerint).
6. A márkakomponensek a `src/lib/components/brand/` alá.
7. Az `app.html` és a `+layout.svelte` cseréje a 6.3 és 6.5 célállapotú változatára, és a `+page.svelte`-be a `<svelte:head><title>Teendők</title></svelte:head>` blokk; a `src/lib/assets/favicon.svg` törlése.
8. A Go SPA-kezelő bővítése a `fonts/` gyorsítótárával (4.7).
9. `npm run check` és `npm run build`, majd a Go-build.

A shadcn-svelte `init` és a `#lib` alias együttműködése nincs kipróbálva, a 3. lépés után ellenőrizni kell, hogy az importok feloldódnak.

**Márkakomponensek.** Mindegyik Svelte 5 szintaxisú (`$props()`, `$derived`, `{@render children()}`), szemantikus tokeneket használ, Lucide-ikonjai `strokeWidth={1.5}` vonalúak.

| Komponens | Propok | Alapérték | Függőség |
|---|---|---|---|
| `Logo` | `variant: 'mark' \| 'lockup' \| 'wordmark'`, `height: number`, `class` | `lockup`, `32` | nincs. A színt a szülő `text-*` osztálya adja, a felső csomópont `var(--primary)`. `<svg role="img" aria-label="Kinexium">`; viewBox: mark `0 0 64 64`, lockup `0 0 232.6 64`, wordmark `0 0 513.7 71.7`; a szélesség a `height`-ból arányosan adódik |
| `Panel` | `size: 'sm' \| 'md' \| 'lg'`, `class`, `children: Snippet` | `md` | `.kx-panel`, `.kx-panel-sm`, `.kx-panel-lg` osztályok |
| `Metric` | `label: string`, `value: string`, `unit?: string`, `note?: string` | | `./Panel.svelte` |
| `StatusBadge` | `status: Status`, `label?: string` | | `@lucide/svelte/icons/{circle-alert,circle-check,info,triangle-alert,wifi-off}` |
| `TickRule` | `class` | | `.kx-ticks` osztály |

Használat:

```svelte
<script lang="ts">
	import Logo from '$lib/components/brand/Logo.svelte';
	import Metric from '$lib/components/brand/Metric.svelte';
	import StatusBadge from '$lib/components/brand/StatusBadge.svelte';
</script>

<Logo variant="lockup" height={32} class="text-foreground" />
<Metric label="Átlagos késleltetés" value="42" unit="ms" note="Tegnap: 47 ms" />
<StatusBadge status="warning" />
<StatusBadge status="online" label="Csatlakozva" />
```

`Panel.svelte` (teljes):

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

`Metric.svelte` (teljes):

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
			{value}{#if unit}<small class="text-muted-foreground ml-1 text-sm">{unit}</small>{/if}
		</span>
		{#if note}<span class="body-sm text-muted-foreground">{note}</span>{/if}
	</div>
</Panel>
```

`StatusBadge.svelte` (teljes):

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

`TickRule.svelte` (teljes):

```svelte
<script lang="ts">
	/** Mérőléc. Szekció vagy idővonal tagolására, oldalanként egyszer-kétszer. */
	let { class: className = '' }: { class?: string } = $props();
</script>

<span class="kx-ticks {className}" aria-hidden="true"></span>
```

`Logo.svelte` (teljes; az útvonaladatokat nem szabad átírni):

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

Ikonok: a Lucide ikonokat egyenként importáljuk, ez tartja kicsiben a bundle-t:

```svelte
<script lang="ts">
	import Cpu from '@lucide/svelte/icons/cpu';
</script>

<Cpu class="size-4" strokeWidth={1.5} aria-hidden="true" />
```

### 6.11 Ismert figyelmeztetések

| Figyelmeztetés | Mikor | Teendő |
|---|---|---|
| `config_option_deprecated_alias` | Build közben, amíg a shadcn-svelte `$lib` útvonalakat használ és a `vite.config.ts`-ben `alias: { $lib: 'src/lib' }` áll | Várható, nincs teendő |
| `go:embed` hiba (`pattern all:build: no matching files found` jellegű) | `go build`, `go run`, `go vet` a `web/build` nélkül | Előbb `npm run build` a `web/` mappában |
| Hiányzó `go.sum` / hiányzó modul | `go build` vagy Docker-build a `go mod tidy` előtt | `go mod tidy` |
| `EBADENGINE` hiba | `npm ci` Node 22.17 alatt vagy Node 23.x alatt (a `@sveltejs/kit` `engines.node` értéke `>=22.17`, a `@sveltejs/vite-plugin-svelte`-é `^20.19 || ^22.12 || >=24`; az `engine-strict=true` miatt hiba, nem figyelmeztetés) | Node 22.17 vagy újabb 22.x, illetve Node 24 vagy újabb telepítése (a Node 23 nem támogatott) |

### 6.12 A váz felületének teljes forrása

A 6.5 és 6.7 leírásai ezekre a fájlokra vonatkoznak. A célállapotban a `+layout.svelte` a 6.5 szerint, az `ui/` mappa a shadcn-svelte CLI-vel cserélődik.

`web/src/routes/+page.svelte` (teljes):

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { Plus, Trash2 } from '@lucide/svelte';
	import { api, type Todo } from '#lib/api.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Card } from '#lib/components/ui/card/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';

	let todos = $state<Todo[]>([]);
	let title = $state('');
	let error = $state('');
	let loading = $state(true);

	const remaining = $derived(todos.filter((t) => !t.done).length);

	async function attempt(action: () => Promise<void>) {
		error = '';
		try {
			await action();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Ismeretlen hiba';
		}
	}

	onMount(async () => {
		await attempt(async () => {
			todos = await api.list();
		});
		loading = false;
	});

	const add = (event: SubmitEvent) => {
		event.preventDefault();
		const text = title.trim();
		if (!text) return;
		return attempt(async () => {
			todos = [await api.create(text), ...todos];
			title = '';
		});
	};

	const toggle = (todo: Todo) =>
		attempt(async () => {
			const updated = await api.setDone(todo.id, !todo.done);
			todos = todos.map((t) => (t.id === updated.id ? updated : t));
		});

	const remove = (todo: Todo) =>
		attempt(async () => {
			await api.remove(todo.id);
			todos = todos.filter((t) => t.id !== todo.id);
		});
</script>

<main class="mx-auto flex w-full max-w-xl flex-col gap-6 px-4 py-12">
	<header class="flex items-baseline justify-between gap-4">
		<h1 class="text-2xl font-semibold tracking-tight">Teendők</h1>
		<p class="text-sm text-muted-foreground">{remaining} nyitott</p>
	</header>

	<form class="flex gap-2" onsubmit={add}>
		<Input bind:value={title} placeholder="Új teendő" aria-label="Új teendő" maxlength={200} />
		<Button type="submit"><Plus class="size-4" />Hozzáadás</Button>
	</form>

	{#if error}
		<p role="alert" class="text-sm text-destructive">{error}</p>
	{/if}

	<Card>
		{#if loading}
			<p class="p-6 text-center text-sm text-muted-foreground">Betöltés…</p>
		{:else if todos.length === 0}
			<p class="p-6 text-center text-sm text-muted-foreground">Még nincs teendő.</p>
		{:else}
			<ul class="divide-y">
				{#each todos as todo (todo.id)}
					<li class="flex items-center gap-3 px-4 py-2">
						<input
							type="checkbox"
							class="size-4 accent-primary"
							checked={todo.done}
							onchange={() => toggle(todo)}
							aria-label={`Kész: ${todo.title}`}
						/>
						<span class={['min-w-0 flex-1 break-words text-sm', todo.done && 'text-muted-foreground line-through']}>
							{todo.title}
						</span>
						<Button variant="ghost" size="icon" onclick={() => remove(todo)} aria-label={`Törlés: ${todo.title}`}>
							<Trash2 class="size-4" />
						</Button>
					</li>
				{/each}
			</ul>
		{/if}
	</Card>
</main>
```

`web/src/routes/+layout.svelte` (váz, teljes):

```svelte
<script lang="ts">
	import favicon from '#lib/assets/favicon.svg';
	import '../app.css';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Teendők</title>
</svelte:head>

{@render children()}
```

`web/src/lib/components/ui/button/button.svelte` (teljes):

```svelte
<script lang="ts" module>
	import { tv, type VariantProps } from 'tailwind-variants';

	export const buttonVariants = tv({
		base: 'inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 rounded-md text-sm font-medium whitespace-nowrap transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50',
		variants: {
			variant: {
				default: 'bg-primary text-primary-foreground hover:bg-primary/90',
				ghost: 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
			},
			size: {
				default: 'h-10 px-4',
				icon: 'size-9'
			}
		},
		defaultVariants: { variant: 'default', size: 'default' }
	});

	export type ButtonVariant = VariantProps<typeof buttonVariants>['variant'];
	export type ButtonSize = VariantProps<typeof buttonVariants>['size'];
</script>

<script lang="ts">
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { cn } from '#lib/utils.ts';

	type Props = HTMLButtonAttributes & { variant?: ButtonVariant; size?: ButtonSize };

	let {
		class: className,
		variant = 'default',
		size = 'default',
		type = 'button',
		children,
		...rest
	}: Props = $props();
</script>

<button {type} class={cn(buttonVariants({ variant, size }), className)} {...rest}>
	{@render children?.()}
</button>
```

`web/src/lib/components/ui/button/index.ts` (teljes):

```ts
export { default as Button, buttonVariants } from './button.svelte';
```

`web/src/lib/components/ui/card/card.svelte` (teljes):

```svelte
<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '#lib/utils.ts';

	let { class: className, children, ...rest }: HTMLAttributes<HTMLDivElement> = $props();
</script>

<div class={cn('rounded-lg border bg-card text-card-foreground shadow-sm', className)} {...rest}>
	{@render children?.()}
</div>
```

`web/src/lib/components/ui/card/index.ts` (teljes):

```ts
export { default as Card } from './card.svelte';
```

`web/src/lib/components/ui/input/input.svelte` (teljes):

```svelte
<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';
	import { cn } from '#lib/utils.ts';

	let { class: className, value = $bindable(), ...rest }: HTMLInputAttributes = $props();
</script>

<input
	bind:value
	class={cn(
		'h-10 w-full min-w-0 rounded-md border border-input bg-transparent px-3 text-sm outline-none transition-colors placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring',
		className
	)}
	{...rest}
/>
```

`web/src/lib/components/ui/input/index.ts` (teljes):

```ts
export { default as Input } from './input.svelte';
```

`web/src/lib/index.ts` (teljes):

```ts
// place files you want to import through the `#lib` alias in this folder.
```

---

## 7. Build és futtatás

### 7.1 Első indítás

A `go.mod`-ban nincs `require` sor a `modernc.org/sqlite`-ra, és `go.sum` sincs. Az első lépés ezért mindig a `go mod tidy`: letölti a modult és létrehozza a `go.sum`-ot. Docker-buildhez is kötelező, mert a Dockerfile a `go.sum`-ot másolja.

A `web/package-lock.json` is kell, mert a build `npm ci`-t futtat. Ha hiányzik, előbb a `web/` mappában egyszer `npm install` kell (Node 22.17 vagy újabb 22.x, illetve Node 24 vagy újabb), utána a lockfile verziókövetésbe kerül (2.2).

Linux, macOS:

```sh
go mod tidy    # letölti a modernc.org/sqlite modult és létrehozza a go.sum fájlt
make build
./dist/webapp  # http://localhost:8080
```

Windows, PowerShell:

```powershell
go mod tidy
.\build.ps1
.\dist\webapp.exe  # http://localhost:8080
```

Docker:

```sh
go mod tidy
docker compose up -d --build
docker compose ps
```

Go nélkül, konténerben futtatott `go mod tidy`:

```sh
docker run --rm -v "$PWD":/src -w /src golang:1.27-alpine go mod tidy
```

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.27-alpine go mod tidy
```

PowerShellben a `${PWD}` alak kötelező, mert a `"$PWD:/src"`-ben a kettőspontot változóhatókörként értelmezné.

A háttérben futó két lépés:

```sh
cd web && npm ci && npm run build && cd ..
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=1.0.0" \
  -o dist/webapp ./cmd/server
```

### 7.2 Makefile

```make
APP      := webapp
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
GOBUILD  := CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)"
TARGETS  := linux/amd64 linux/arm64 windows/amd64 darwin/arm64

.PHONY: all tidy web build run cross check image binary-image clean

all: build

tidy:
	go mod tidy

web:
	cd web && npm ci && npm run build

build: web
	$(GOBUILD) -o dist/$(APP) ./cmd/server

run: build
	./dist/$(APP)

cross: web
	@for target in $(TARGETS); do \
		os=$${target%/*}; arch=$${target#*/}; ext=""; \
		[ "$$os" = "windows" ] && ext=".exe"; \
		echo "$$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD) -o dist/$(APP)-$$os-$$arch$$ext ./cmd/server || exit 1; \
	done

check:
	cd web && npm run check
	go vet ./cmd/... ./internal/... ./web

image:
	docker build --target runtime --build-arg VERSION=$(VERSION) -t $(APP):$(VERSION) .

# A kész binárist Dockerrel állítja elő, helyi Go és Node nélkül.
binary-image:
	docker build --target binary --build-arg VERSION=$(VERSION) --output type=local,dest=dist .

clean:
	rm -rf dist web/build web/.svelte-kit
```

| Target | Függ | Eredmény |
|---|---|---|
| `all` | `build` | |
| `tidy` | | Frissíti a `go.mod`-ot és a `go.sum`-ot |
| `web` | | `web/build` |
| `build` | `web` | `dist/webapp` |
| `run` | `build` | Fordít és elindítja a programot |
| `cross` | `web` | `dist/webapp-linux-amd64`, `dist/webapp-linux-arm64`, `dist/webapp-windows-amd64.exe`, `dist/webapp-darwin-arm64` |
| `check` | | `svelte-check` és `go vet` |
| `image` | | Docker image `webapp:<verzió>` |
| `binary-image` | | Linuxos `dist/webapp`, helyi Go és Node nélkül |
| `clean` | | Törli: `dist`, `web/build`, `web/.svelte-kit`. A `data` mappát nem törli |

Verzió: `git describe --tags --always --dirty`, ha nincs git, `dev`. Felülírás: `make VERSION=1.0.0 build`. A Makefile-nak nincs `dev` targetje.

### 7.3 build.ps1

A Makefile PowerShell-es párja, `make` nélkül. Windows PowerShell 5.1-hez és PowerShell 7-hez készült (`#Requires -Version 5.1`). UTF-8 BOM-mal van mentve, mert a Windows PowerShell 5.1 enélkül hibásan olvassa az ékezeteket; szerkesztéskor a BOM-ot meg kell őrizni.

Paraméterek: `-Task` (0. pozíció, alapérték `build`), `-Version` (alapérték nincs; ha nincs megadva, a `git describe --tags --always --dirty`, annak hiányában `dev`). A szkript `Set-StrictMode -Version Latest` és `$ErrorActionPreference = 'Stop'` mellett fut, a saját mappájában. A natív parancsok nem nulla kilépési kódja kivételt dob: `Sikertelen parancs (<kód>): <parancs>`. Windowson `npm.cmd`-t hív (ez a végrehajtási házirendtől függetlenül elindul), a binárisok `.exe` kiterjesztést kapnak.

| Makefile | PowerShell | Eredmény |
|---|---|---|
| `make tidy` | `.\build.ps1 tidy` | Frissíti a `go.mod`-ot és a `go.sum`-ot |
| `make web` | `.\build.ps1 web` | `web/build` |
| `make build` | `.\build.ps1` | `dist\webapp.exe` |
| `make run` | `.\build.ps1 run` | Fordít és elindítja a programot |
| nincs | `.\build.ps1 dev` | Go-szerver és Vite egyszerre |
| `make cross` | `.\build.ps1 cross -Version 1.0.0` | Négy platform binárisa a `dist` mappában |
| `make check` | `.\build.ps1 check` | `svelte-check` és `go vet` |
| `make image` | `.\build.ps1 image` | Docker image `webapp:<verzió>` |
| `make binary-image` | `.\build.ps1 binary-image` | Linuxos `dist\webapp`, Windowson nem fut |
| `make clean` | `.\build.ps1 clean` | Törli: `dist`, `web/build`, `web/.svelte-kit` |

A Go-build a szkriptben: `CGO_ENABLED=0`, opcionálisan `GOOS`/`GOARCH`, `go build -trimpath -ldflags "-s -w -X main.version=<verzió>" -o <kimenet> ./cmd/server`. A környezeti változókat utána visszaállítja, különben a munkamenet végéig megmaradnának.

`build.ps1` (teljes; UTF-8 BOM-mal kell menteni, a BOM kötelező; a `.gitattributes` `*.ps1 text eol=crlf` sora checkoutkor CRLF-re alakítja, a git nélkül másolt fájl LF sorvégű is lehet):

```powershell
#Requires -Version 5.1
<#
.SYNOPSIS
    A Makefile PowerShell-es párja: build, fejlesztés és Docker Windowson.
.EXAMPLE
    .\build.ps1
    .\build.ps1 dev
    .\build.ps1 cross -Version 1.0.0
#>
[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('build', 'web', 'run', 'dev', 'tidy', 'cross', 'check', 'image', 'binary-image', 'clean')]
    [string] $Task = 'build',

    [string] $Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$App = 'webapp'
$Targets = 'linux/amd64', 'linux/arm64', 'windows/amd64', 'darwin/arm64'
$DevAddr = '127.0.0.1:8080'
$OnWindows = $env:OS -eq 'Windows_NT'
$Exe = if ($OnWindows) { '.exe' } else { '' }
# Az npm.cmd a végrehajtási házirendtől függetlenül elindul.
$Npm = if ($OnWindows) { 'npm.cmd' } else { 'npm' }

Set-Location -LiteralPath $PSScriptRoot

function Invoke-Native {
    $file = $args[0]
    # Tömb kell: egyetlen szöveg splatkor karakterekre esne szét.
    [string[]] $rest = @($args | Select-Object -Skip 1)
    & $file @rest
    if ($LASTEXITCODE -ne 0) {
        throw "Sikertelen parancs ($LASTEXITCODE): $file $rest"
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

# A környezeti változók az egész munkamenetre megmaradnának, ezért visszaállítjuk őket.
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
        Invoke-Native go build -trimpath -ldflags "-s -w -X main.version=$(Get-AppVersion)" -o $Output ./cmd/server
    }
    finally {
        $env:CGO_ENABLED = $saved.CGO_ENABLED
        $env:GOOS = $saved.GOOS
        $env:GOARCH = $saved.GOARCH
    }
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
        if (-not (Test-Path -LiteralPath 'web/node_modules')) { Invoke-WebBuild }
        # Állandó útvonalú bináris: a tűzfal nem kérdez újra, és egy folyamatot kell leállítani.
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
        Push-Location web
        try { Invoke-Native $Npm run check }
        finally { Pop-Location }
        Invoke-Native go vet ./cmd/... ./internal/... ./web
    }
    'image' {
        $tag = Get-AppVersion
        Invoke-Native docker build --target runtime --build-arg "VERSION=$tag" -t "${App}:$tag" .
    }
    'binary-image' {
        Invoke-Native docker build --target binary --build-arg "VERSION=$(Get-AppVersion)" --output type=local,dest=dist .
    }
    'clean' {
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue dist, web/build, web/.svelte-kit
    }
}
```

### 7.4 Fejlesztői mód

Két terminállal (bármely platformon):

```sh
# egyszer, hogy a go:embed ne adjon hibát
cd web && npm ci && npm run build && cd ..

# 1. terminál
go run ./cmd/server

# 2. terminál
cd web
npm run dev
```

Windowson egy paranccsal:

```powershell
.\build.ps1 dev
```

A `dev` feladat lépései:

1. Ha nincs `web/build/index.html` vagy `web/node_modules`, lefuttatja a felület buildjét (`npm ci`, `npm run build`).
2. Lefordítja a `dist/webapp-dev.exe` binárist. Az állandó útvonal miatt a tűzfal nem kérdez újra, és egy folyamatot kell leállítani.
3. Elindítja: `-addr 127.0.0.1:8080`. A szerver csak loopbacken figyel, a hálózatról nem érhető el.
4. Az előtérben futtatja a Vite-ot (`npm run dev`).
5. Ctrl+C mindkettőt leállítja.

| Szolgáltatás | Cím |
|---|---|
| Felület (Vite) | http://localhost:5173 |
| Go API | http://127.0.0.1:8080 |
| Proxy | `/api` → `http://127.0.0.1:8080` |

A felület változása azonnal látszik. Go-változás után a szervert újra kell indítani.

### 7.5 Kimenetek

| Kimenet | Útvonal |
|---|---|
| Frontend build | `web/build` (benne `index.html`, `_app/`, a `static/` tartalma) |
| Linux / macOS bináris | `dist/webapp` |
| Windows bináris | `dist\webapp.exe` |
| Fejlesztői bináris | `dist/webapp-dev` (`.exe` Windowson) |
| Keresztfordítás | `dist/webapp-<os>-<arch>[.exe]` |
| Docker `binary` stage | `dist/webapp` |
| Docker image | `/app/webapp`, adat: `/data` |

Közös buildbeállítások: `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w -X main.version=<verzió>"`. A `web/build` mérete kb. 260 KB; a JS és a CSS gzippelve kb. 68 KB. Helyettesített driverrel a bináris 7,0 MB volt; a valódi SQLite-driver ezt növeli, a végleges méret nincs lemérve.

### 7.6 Docker

`Dockerfile` (teljes):

```dockerfile
# 1. A felület statikus buildje
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2. A Go-bináris, beágyazott felülettel
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS server
ARG TARGETOS TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/embed.go
COPY --from=web /src/web/build web/build
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=$VERSION" \
    -o /out/webapp ./cmd/server
RUN mkdir -p /out/data

# 3. Csak a bináris kimásolásához: docker build --target binary --output ...
FROM scratch AS binary
COPY --from=server /out/webapp /webapp

# 4. Futtató image: nincs benne shell, és nem root felhasználóval fut
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=server /out/webapp /app/webapp
COPY --from=server --chown=65532:65532 /out/data /data
ENV APP_ADDR=:8080 APP_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/app/webapp", "-healthcheck"]
ENTRYPOINT ["/app/webapp"]
```

| Stage | Alap | Feladat |
|---|---|---|
| `web` | `node:22-alpine`, build platformon | `npm ci`, `npm run build` |
| `server` | `golang:1.27-alpine`, build platformon | Go-build a célplatformra (`TARGETOS`, `TARGETARCH`), üres `/out/data` |
| `binary` | `scratch` | Csak a bináris, kimásoláshoz |
| `runtime` | `gcr.io/distroless/static-debian12:nonroot` | Futtató image, uid 65532, shell nélkül |

- A `--platform=$BUILDPLATFORM` miatt a fordítás natívan fut a build gépen, emuláció nélkül. A célplatformot a `GOOS` és a `GOARCH` adja.
- A `/data` a 65532-es felhasználó tulajdona, ezért a csak olvasható konténerben is írható a kötet.
- Explicit `USER` direktíva nincs, a `:nonroot` image alapértelmezett felhasználója érvényes. `CMD` nincs.
- A `.git` nincs a build-kontextusban, ezért a verziót csak a `VERSION` build-arg adja (alapérték `dev`). A Make és a PowerShell-szkript átadja.

`compose.yaml` (teljes):

```yaml
services:
  webapp:
    build:
      context: .
      target: runtime
    image: webapp:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - webapp-data:/data
    read_only: true
    tmpfs:
      - /tmp
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true

volumes:
  webapp-data:
```

A Compose nem ad meg környezeti változót és saját healthchecket; a Dockerfile `ENV` és `HEALTHCHECK` sora érvényes. A Compose image-címkéje `webapp:latest`, a `make image` címkéje `webapp:<verzió>`. A Compose-build nem ad át `VERSION` build-argot, ezért a benne lévő bináris verziója `dev`.

`.dockerignore`:

```
.git
dist
data
web/node_modules
web/build
web/.svelte-kit
*.md
```

Bináris előállítása csak Dockerrel (a `--build-arg VERSION` nélkül a bináris verziója `dev`):

```sh
docker build --target binary --build-arg VERSION=1.0.0 --output type=local,dest=dist .
./dist/webapp

# ARM-es Linuxra
docker build --platform linux/arm64 --target binary --build-arg VERSION=1.0.0 --output type=local,dest=dist .
```

Windowsos bináris Dockerrel (a Dockerfile `webapp` néven adja ki, át kell nevezni):

```powershell
docker build --platform windows/amd64 --target binary --build-arg VERSION=1.0.0 --output type=local,dest=dist .
Move-Item -Force dist\webapp dist\webapp.exe
```

### 7.7 Verziókövetési beállítások

`.gitignore` (gyökér):

```
/dist/
/data/
web/node_modules/
web/build/
web/.svelte-kit/
```

`web/.gitignore`: `node_modules`; `.output`, `.vercel`, `.netlify`, `.wrangler`, `/.svelte-kit`, `/build`; `.DS_Store`, `Thumbs.db`; `.env`, `.env.*`, `!.env.example`, `!.env.test`; `vite.config.js.timestamp-*`, `vite.config.ts.timestamp-*`.

`.gitattributes`:

```
* text=auto eol=lf
*.ps1 text eol=crlf
*.png binary
*.ico binary
```

Minden szövegfájl LF sorvégű, így a `gofmt` és a Linuxos konténerbuild ugyanazt a tartalmat látja. A `.ps1` fájlokat a `*.ps1 text eol=crlf` sor checkoutkor CRLF-re alakítja; a git nélkül másolt fájl LF sorvégű is lehet, a BOM viszont kötelező.

### 7.8 Windows és Linux különbségek

| Szempont | Linux / macOS | Windows |
|---|---|---|
| Build eszköz | `make` (bash) | `build.ps1` |
| Fejlesztői mód | két terminál | `.\build.ps1 dev` vagy két terminál |
| Bináris | `dist/webapp` | `dist\webapp.exe` |
| npm hívás | `npm` | `npm.cmd` |
| Törlés | `rm -rf` | `Remove-Item -Recurse -Force` |
| Környezeti változók a buildben | a parancs elején, csak arra a parancsra | a `build.ps1` visszaállítja; kézzel állítva a munkamenet végéig megmaradnak |
| Docker | natív | Docker Desktop, Linux-konténer módban |

Windows-telepítés:

```powershell
winget install --id Microsoft.PowerShell --source winget
winget install --id OpenJS.NodeJS.22 --exact
winget install --id GoLang.Go --exact
winget install --id Git.Git --exact
winget install --id Docker.DockerDesktop --exact     # csak konténerhez
```

Utána új terminál kell a PATH frissüléséhez. Ellenőrzés:

```powershell
node --version
go version
docker version
```

Szkriptek engedélyezése. Windows-kliensen az alapértelmezett végrehajtási házirend a `Restricted`, ez a `.ps1` fájlokat is tiltja. A `RemoteSigned` a helyi szkripteket engedi, az internetről letöltötteket blokkolja:

```powershell
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
cd C:\src\webapp
Get-ChildItem -Recurse -File | Unblock-File   # csak a zip kicsomagolása után
```

Kézi fordítás Windowson:

```powershell
$env:CGO_ENABLED = '0'
go build -trimpath -ldflags "-s -w -X main.version=1.0.0" -o dist\webapp.exe .\cmd\server
```

Kézzel állított `GOOS` esetén a következő `go build` is arra a platformra fordít, amíg le nem fut a `Remove-Item Env:GOOS`.

Tipikus Windows-buktatók:

- **Zárolt .exe:** futó program fájlját a Windows nem engedi felülírni. `Access is denied` hibánál előbb le kell állítani a `webapp.exe`-t.
- **curl:** Windows PowerShell 5.1-ben az `Invoke-WebRequest` aliasa. Helyette `curl.exe` vagy `Invoke-RestMethod`.
- **Leállítás:** a Ctrl+C szabályos leállítást indít. A `Stop-Process` azonnal kilövi a folyamatot, a futó kérések megszakadnak.
- **Mappa:** a projekt és az adatkönyvtár helyi meghajtón legyen, ne OneDrive-ban vagy hálózati megosztáson. A szinkronizálás a `node_modules` fájljait is zárolhatja.

### 7.9 Üzemeltetés

- **Mentés tartalma:** az egész adatkönyvtár (`app.db`, `app.db-wal`, `app.db-shm`). A bináris bármikor újra előállítható, az adat nem.
- **Mentés módja:** futó szerver mellett nem fájlmásolással. Le kell állítani a programot, vagy az SQLite `.backup` parancsát, illetve a `VACUUM INTO` utasítást kell használni.
- **Konténeres mentés** (a futó image-ben nincs shell, ezért segédkonténer kell; a `webapp` nevű Compose-projekt kötete):
  ```sh
  docker compose stop webapp
  docker run --rm -v webapp_webapp-data:/data -v "$PWD":/backup alpine \
    tar czf /backup/webapp-data.tgz -C /data .
  docker compose start webapp
  ```
  ```powershell
  docker compose stop webapp
  docker run --rm -v webapp_webapp-data:/data -v "${PWD}:/backup" alpine `
    tar czf /backup/webapp-data.tgz -C /data .
  docker compose start webapp
  ```
- **Frissítés:** előbb mentés, aztán a bináris vagy az image cseréje és újraindítás. A migrációk induláskor lefutnak.
- **Visszalépés:** a program csak előrefelé migrál, a régi verzióhoz a frissítés előtti mentés kellhet.
- **Publikus elérés:** a program sima HTTP-n fut, hitelesítés nincs benne. Internetre csak TLS-t adó reverse proxy mögött és hitelesítéssel tehető ki.

### 7.10 Ellenőrzési állapot

A felület típusellenőrzése és buildje valódi függőségekkel lefutott. A Go-szerver, a keresztfordítás és a Docker-build helyettesített driverrel, illetve helyettesítő alapimage-ekkel futott. Az első teljes build a fejlesztő gépén történik.

| Próba | Eredmény | Környezet |
|---|---|---|
| Felület típusellenőrzése és buildje | 0 hiba; `web/build` 260 KB, a JS és a CSS gzippelve 68 KB | valódi függőségek |
| Kicsomagolt projekt `npm ci` és build | sikeres | valódi függőségek |
| Fejlesztői proxy | a Vite 5173-as portja eléri az API-t | valódi Vite, helyettesített driver |
| HTTP-végpontok, hibakódok, cache-fejlécek | mind a várt választ adta | Go 1.24.7, helyettesített driver |
| Felület Chromiumban | hozzáadás, pipálás, törlés, újratöltés működött, konzolhiba nélkül | helyettesített driver |
| Újraindítás, SIGTERM, `-healthcheck` | az adat megmaradt, a leállás tiszta; kilépési kód 0, leállított szervernél 1 | helyettesített driver |
| Az alkalmazás SQL-utasításai | migráció, `RETURNING`, `CHECK`, WAL rendben | valódi SQLite 3.53.1, Pythonból |
| Keresztfordítás | Linux arm64, Windows amd64, macOS arm64 lefordult | helyettesített driver, futtatás nélkül |
| Változatlan Dockerfile | `runtime` és `binary` cél, amd64 és arm64 export sikeres | helyettesítő alapimage-ek: Ubuntu-alapú Node 22 és Go 1.24.7 |
| Konténer | nem rootként, csak olvasható fájlrendszerrel, jogok nélkül futott, a köteten megmaradt az adat | helyettesítő alapimage-ek |
| `compose.yaml` | érvényes | csak `docker compose config` |
| `build.ps1` | a `build`, `cross`, `check`, `clean` és `dev` lefutott; hibás build 1-es kilépési kódot adott; a `cross` után a környezeti változók visszaálltak; a `dev` leállításakor a Go-szerver is leállt | PowerShell 7.6.0, Linux, helyettesített driver |

- A Windows-bináris helyes PE-fájl lett (`go build` és Dockeres `--platform windows/amd64` fordítással is), de Windowson nem futott.
- A `build.ps1` `image` és `binary-image` feladata nem futott a szkriptből; a mögöttük álló Docker-parancsok sikeresek voltak.
- A `-healthcheck` parancs a helyettesítő környezetben kézzel futott.
- A winget-telepítés, a végrehajtási házirend és a Docker Desktop nincs kipróbálva.

Még ellenőrizendő:

- `go mod tidy` és a valódi `modernc.org/sqlite` driver a `_pragma` beállításokkal;
- build a valódi `node:22-alpine`, `golang:1.27-alpine` és distroless image-ekkel;
- a `/api` kérés átirányításának státuszkódja Go 1.27-tel (Go 1.24.7-tel 301);
- `docker compose up` és a beépített healthcheck állapota;
- a shadcn-svelte `init` és a `#lib` alias együttműködése;
- a `build.ps1` valódi Windowson, Windows PowerShell 5.1-gyel és PowerShell 7-tel, a `webapp.exe` indításával, abszolút `-data` útvonallal is;
- futtatás macOS-en és ARM-en;
- a mentési parancs.

---

## 8. Fejlesztési konvenciók

### 8.1 Go

- Formázás: `gofmt`. Ellenőrzés: `go vet ./cmd/... ./internal/... ./web`.
- Csak standard könyvtár, az egyetlen külső függőség a `modernc.org/sqlite`. Új függőség csak indokolt esetben.
- Hibák csomagolása magyar kontextussal: `fmt.Errorf("adatbázis megnyitása: %w", err)`.
- Sentinel hibák (`ErrNotFound`) és `errors.Is` a HTTP-kódok képzéséhez. A belső hiba részlete csak a naplóba kerül.
- Naplózás `log/slog`-gal, magyar üzenetekkel, kulcs–érték mezőkkel.
- SQL sima `database/sql`-lel, statikus lekérdezésekkel, `RETURNING`-gel. Tranzakciónál `defer tx.Rollback()`.
- Új API-útvonal metódusos mintával (`"GET /api/..."`) az `/api/` catch-all és a `/` SPA-kezelő mellé.
- A sémaváltozás mindig új, sorszámozott migrációs fájl. Meglévő migrációt nem módosítunk.

### 8.2 TypeScript és Svelte

- TypeScript `strict: true`. Ellenőrzés: `npm run check` (`svelte-check`).
- Svelte 5 runes mód kötelező a saját kódra: `$state`, `$derived`, `$props`, `$bindable`, `{@render children()}`, `Snippet`.
- Osztályösszefűzés `cn()`-nel, variánsok `tailwind-variants`-szal (váz) vagy a shadcn-svelte komponenseivel (célállapot).
- API-hívás csak a `src/lib/api.ts`-en keresztül, típusos függvényekkel.
- Ikon: Lucide, egyenként importálva, `strokeWidth={1.5}`. Csak ikonos gombra kell `aria-label`.
- Komponensben szemantikus tokent használunk (`bg-card`), nyers skálatokent csak márkafelületen. Nyers hex érték nem kerül a kódba.

### 8.3 Szövegek és hibaüzenetek

- A felület és az API hibaüzenetei magyarok. Az API mindig `{"error":"..."}` formában válaszol, a kliens ezt az üzenetet jeleníti meg.
- Felületi szövegben (gomb, hibaüzenet) tegező felszólító mód: „Ellenőrizd a kapcsolatot”. Weboldalon kerülni kell a közvetlen megszólítást; ha mégis kell, magázás: „Van egy rendszerötlete?”
- Gomb: ige és tárgy („Eszköz felvétele”, nem „OK”). Törlésnél a gomb megnevezi, mi törlődik („Eszköz törlése”).
- Hibaüzenet: mi történt, és mit tegyen a felhasználó. Példa: „A GW-04 gateway nem elérhető. Ellenőrizd a hálózati kapcsolatot.”
- Mondatkezdő nagybetű címben, gombon, menüben. Emoji és felkiáltójel nincs.
- Számok magyar felületen: tizedesvessző, ötjegytől szóközös ezres tagolás (`24,8 °C`, `1284`, `12 480`), érték és mértékegység között szóköz, idő 24 órás (`14:32:08`), dátum `2026. 11. 05.`. Hiányzó érték: `–`, nulla helyett soha. Angol felületen pont a tizedesjel.
- Szakmai angol szó marad, ha nincs bevett magyar: gateway, firmware, API-kulcs.

### 8.4 OpenAPI-enum és StatusBadge

- Az állapotlista egy helyen él: az API OpenAPI-leírásában. A `StatusBadge` `status` propja az ebből generált enumot kapja.
- A komponens jelenlegi típusa: `'online' | 'warning' | 'critical' | 'offline' | 'info'`. Az OpenAPI-enum értékei ezzel egyezzenek.
- A vázban még nincs OpenAPI-leírás. Amíg nincs, a `Status` típus a `StatusBadge.svelte`-ből exportált típus.

| `status` | Alapszó | Lucide ikon | Ikonszín |
|---|---|---|---|
| `online` | Online | `circle-check` | `text-success` |
| `warning` | Figyelmeztetés | `triangle-alert` | `text-warning` |
| `critical` | Kritikus | `circle-alert` | `text-destructive` |
| `offline` | Offline | `wifi-off` | `text-muted-foreground` |
| `info` | Szinkronizál | `info` | `text-info` |

### 8.5 Tokenfrissítés

- A tokenértékek egyetlen forrása a GRAPHICS.md (2–5. fejezet; a teljes stíluslap a 13.2-ben). Értékváltozáskor a GRAPHICS.md 13.2 stíluslapja frissül, és ezzel kell teljes cserével felülírni a `src/app.css`-t.
- A `src/app.css`-t nem szerkesztjük kézzel, mert a következő csere felülírja. Projektszintű kiegészítés külön fájlba kerül, vagy a GRAPHICS.md 13.2 stíluslapjába.
- Fontcsere után a `fonts/` alatti fájl nevét is változtatni kell, ha a 4.7 szerinti egyéves gyorsítótár be van kapcsolva.

---

## 9. Eltérések és döntések

| # | Téma | Eltérés | Döntés |
|---|---|---|---|
| 1 | Hiányzó Go-függőség | A `go.mod`-ban nincs `require` a `modernc.org/sqlite`-ra, `go.sum` nincs | Első lépés mindig a `go mod tidy`, Docker-build előtt is. A keletkező `go.sum` verziókövetésbe kerül |
| 2 | Hiányzó `web/build` | A `//go:embed all:build` a mappát megköveteli | Go-fordítás előtt mindig `npm run build`. A Make és a `build.ps1` ezt automatikusan elvégzi |
| 3 | Go-verzió | `go.mod`: `go 1.24`; Docker: `golang:1.27-alpine`; próbák: 1.24.7 | A minimum 1.24 marad, a Docker-image 1.27. Az 1.27-es build ellenőrizendő |
| 4 | `vite.config.ts` | A váz `#lib` aliast, runes-kényszerítést és proxyt használ, a bekötés `$lib` aliast kér | A célállapot mindhármat tartalmazza (6.1). A `config_option_deprecated_alias` figyelmeztetés ezzel jár |
| 5 | `app.html` | A vázban nincs `class="dark"`, inline szkript, `theme-color`, font preload és favicon-link; van `text-scale` meta | A célállapotban a teljes 6.3 szerinti változat a helyére kerül |
| 6 | Témakezelés | A váz világos alapú, csak `prefers-color-scheme`-et követ; a cél sötét alapú, mode-watcherrel | A célállapot a mode-watcheres sötét alap (6.4, 6.5) |
| 7 | SPA-kezelő és gyorsítótár | Olyan egyszerűsített kezelő nem használható, amely az `index.html`-re nem ad `no-cache`-t, és a hiányzó `_app/*` fájlra nem ad 404-et | A 4.7 szerinti kezelő érvényes, `fonts/` bővítéssel: a `fonts/` prefix is `public, max-age=31536000, immutable` fejlécet kap |
| 8 | Embed | Az `fs.Sub` hibája nem dobható el (`dist, _ := fs.Sub(...)` tilos) | A `web` csomag `files` változója és a hibát visszaadó `Files() (fs.FS, error)` érvényes (4.8) |
| 9 | shadcn-svelte | A vázban kézzel írt `button`, `card`, `input`, `components.json` nélkül | Célállapotban a CLI-vel hozzáadott komponensek és a 6.8 szerinti `components.json`. Az `init` és a `#lib` együttműködése ellenőrizendő |
| 10 | Telepítési parancsok | A váz és a bekötés más csomaglistával indul | Meglévő vázon csak a hiányzó lépések futnak (6.10); új projekt a bekötés sorrendjével indul |
| 11 | `app.css` | A váz oklch-palettája ideiglenes | Teljes csere az arculati stíluslappal; a váz értékei nem tokenek |
| 12 | Favicon | A váz faviconja a Svelte-logó | `static/favicon.svg` az arculat alkalmazásikonjával, az `app.html` hivatkozik rá |
| 13 | `binary-image` | A `docker build --target binary --output type=local,dest=dist .` build-arg nélkül `dev` verziójú binárist ad | Mindig `--build-arg VERSION=<verzió>`-val, ahogy a Makefile: `docker build --target binary --build-arg VERSION=$(VERSION) --output type=local,dest=dist .` |
| 14 | Image-címke | `make image`: `webapp:<verzió>`; Compose: `webapp:latest` | Szándékos, a két út külön címkét ad |
| 15 | `engine-strict` | A saját `package.json`-ban nincs `engines` mező | A beállítás a függőségek `engines` mezőjére hat: a minimum Node 22.17 a `@sveltejs/kit` `engines` mezőjéből és az `engine-strict=true` beállításból érvényesül; a `@sveltejs/vite-plugin-svelte` miatt a Node 23 nem támogatott. A `.npmrc` marad |
| 16 | Kéréstörzs-hiba | A túl nagy törzs 400 `érvénytelen JSON`, nem 413 | A jelenlegi viselkedés marad |
| 17 | Metódusok | `PUT /api/todos` 404 (nem 405); `POST /healthz` 200 `index.html` | A jelenlegi viselkedés marad, a kliens csak a dokumentált metódusokat használja |
| 18 | Egy kapcsolat | `SetMaxOpenConns(1)`: az olvasások is sorba rendeződnek | Marad. Nagyobb olvasási terhelésnél külön olvasó- és írókészlet kell |
| 19 | Migrációnév | A `schema_migrations.name` a teljes beágyazott útvonal (`migrations/0001_create_todos.sql`) | Marad; a fájlokat nem szabad átnevezni vagy áthelyezni |
| 20 | SvelteKit 3 viselkedése | A SvelteKit 3-as vázban nincs `svelte.config.js`, az alap alias a `#lib`, a konfiguráció és az adapter a `vite.config.ts`-ben van | A váz így épült és fordult le; a `$lib` alias visszakapcsolása nincs kipróbálva |
| 21 | `mist` név | A shadcn `baseColor: "mist"` és az arculat `mist-*` skálája névben egyezik | Két különböző dolog; a shadcn-alapszínt az új `app.css` felülírja |
| 22 | Hiányzó lockfile | A `web/package-lock.json` nélkül az `npm ci` leáll | Egyszer `npm install` a `web/` mappában a 2.2 tartományaiból, a feloldott verziók egyeztetése a 2.2 táblázatával; a lockfile verziókövetésbe kerül, a build utána `npm ci`-vel fut |
