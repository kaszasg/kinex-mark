# Kinex Mark

A Kinex Mark a KINEXIUM belső linkgyűjteménye: gyors rögzítés, letöltött cím és ikon, színes címkék, használat szerinti rangsor, kitűzés, automatikus archiválás, megosztás, háromnyelvű (hu, en, de) felület, 2FA és GDPR-funkciók.
Egyetlen statikus Go-bináris, amely a beágyazott SvelteKit-felületet és a JSON API-t is kiszolgálja, az adatokat pedig SQLite-ban tárolja (`app.db` és `secret.key` az adatkönyvtárban).

## Követelmények

- Futtatáshoz: semmi (kész bináris) vagy Docker (Compose v2).
- Fejlesztéshez: Go 1.26+ (`go.mod`: `go 1.26.0`, ajánlott 1.27), Node 26 és npm (`web/package.json` `engines`: `^24 || >=26`). Windowson a `build.ps1`, Linuxon és macOS-en a `Makefile` a buildút.

## Gyors indítás Docker Compose-zal

```sh
docker compose up -d --build
docker compose logs kinex-mark      # az első admin generált jelszava: "initial admin password generated"
```

A felület: `http://127.0.0.1:8080` (a port csak loopbacken nyílik; a konténer csak olvasható, `cap_drop: ALL`, kötet: `kinex-mark-data:/data`). A Compose a helyi image-et `kinex-mark:latest` néven építi, és ez `docker run`-nal is indítható (a `.\build.ps1 image -Version 0.1.0` / `make image VERSION=0.1.0` ezzel szemben `kinex-mark:0.1.0` címkét ad, verzió nélkül a `git describe` értékét):

```sh
docker run -d --name kinex-mark -p 127.0.0.1:8080:8080 -v kinex-mark-data:/data \
  --read-only --tmpfs /tmp --cap-drop ALL --security-opt no-new-privileges:true kinex-mark:latest
```

A CI a sikeres tesztek után a `ghcr.io/kaszasg/kinex-mark` image-et teszi közzé (`latest`, verzió- és SHA-címkék); amíg a repó nincs a GitHubon, csak a helyi image létezik.

**Reverse proxy mögött kötelező** az `APP_BASE_URL` (a nyilvános origin, például `https://mark.example.com`) és az `APP_TRUSTED_PROXY` (a proxy címe; ajánlott egyetlen cím egy dedikált hálózatban, CIDR-tartomány is megadható, de a tartomány bármely gépe hamis kliens-IP-címet küldhet). Nélkülük a bejelentkezés 403 (`csrf_rejected`), a süti nem `Secure`, és a bejelentkezés-korlátozás minden felhasználót a proxy IP-címén együtt zárol. `APP_BASE_URL` nélkül nem loopback figyelési címen nincs Host-ellenőrzés (DNS-rebinding kockázat). Hálózati eléréshez TLS-t végző reverse proxyt ajánlunk (a vágólapra másolás is csak biztonságos kapcsolaton megbízható).

## Gyors indítás binárissal

Windows (PowerShell, a repó gyökerében):

```powershell
.\build.ps1                 # npm ci + vite build, majd go build -> dist\kinex-mark.exe
.\dist\kinex-mark.exe       # http://localhost:8080, adatok: .\data
```

Linux és macOS:

```sh
make build                  # dist/kinex-mark
./dist/kinex-mark           # http://localhost:8080, adatok: ./data
```

Kapcsolók: `-addr`, `-data`, `-version`, `-healthcheck`, a teljes lista: `kinex-mark -h`. Keresztfordítás: `.\build.ps1 cross -Version 1.0.0` / `make cross VERSION=1.0.0` (linux/amd64, linux/arm64, windows/amd64, darwin/arm64).

## Első belépés és helyreállítás

- Üres adatbázisnál a program létrehozza az első admint (`APP_ADMIN_USER`, alap `admin`). Ha az `APP_ADMIN_PASSWORD` nincs megadva, generált jelszót ír a naplóba (`initial admin password generated`), és az első belépéskor kötelező a jelszócsere. Ha meg van adva, át kell mennie a jelszószabályon (legalább 12 karakter, legalább 6 különböző, a felhasználónév nélkül); az első indulás után vedd ki a környezetből.
- Utána minden felhasználónak nyugtáznia kell az adatkezelési tájékoztatót. Az adatkezelő adatait az admin a Rendszerbeállítások oldalon adja meg.
- Zárolt vagy elfelejtett jelszavú admin (a szerver leállítása mellett ajánlott):
  - `kinex-mark -data ./data -unlock-user admin` (a zárolás törlése);
  - `kinex-mark -data ./data -reset-password admin` (új jelszó egyszer a kimenetre, kötelező csere);
  - Dockerben: `docker compose stop`, majd `docker compose run --rm kinex-mark -reset-password admin` (vagy `-unlock-user admin`), végül `docker compose start`.

## Konfiguráció

Elsőbbség: kapcsoló, környezeti változó, alapérték. A teljes lista a [SUMMARY.md](SUMMARY.md) 12. fejezetében.

| Környezeti változó | Alapérték | Jelentés |
|---|---|---|
| `APP_ADDR` | `:8080` | figyelési cím |
| `APP_DATA_DIR` | `./data` (Dockerben `/data`) | adatkönyvtár (`app.db`, `secret.key`) |
| `APP_BASE_URL` | üres | a felület külső originje; proxy mögött kötelező |
| `APP_TRUSTED_PROXY` | üres | megbízható proxy IP-je vagy CIDR-je (vesszővel több is); proxy mögött kötelező |
| `APP_SECURE_COOKIES` | `auto` | `auto`, `true`, `false` |
| `APP_ADMIN_USER`, `APP_ADMIN_PASSWORD` | `admin`, üres | első admin (csak üres adatbázisnál) |
| `APP_SECRET_KEY`, `APP_SECRET_KEY_FILE` | üres, `<data>/secret.key` | szerverkulcs (TOTP-titkok, helyreállító kódok) |
| `APP_ALLOW_PRIVATE_FETCH` | üres | `true`: a metaadat-letöltő privát hálózatot is elér (zárolja a rendszerbeállítást) |
| `APP_ALLOW_LOOPBACK_FETCH` | `false` | loopback és link-local címek letöltése; csak tesztkörnyezetbe |
| `APP_TLS_CERT`, `APP_TLS_KEY` | üres | saját TLS (PEM) |
| `APP_LOG_LEVEL`, `APP_LOG_FORMAT` | `info`, `text` | napló |
| `APP_FETCH_WORKERS`, `APP_FETCH_TIMEOUT` | `2`, `10s` | letöltő munkások (1-16), időkorlát (1s-60s) |

## Fejlesztés és tesztek

- Fejlesztői mód Windowson: `.\build.ps1 dev` (Go `127.0.0.1:8080` + Vite, a felület `http://localhost:5173`). Linuxon két terminálban: `make build && ./dist/kinex-mark -addr 127.0.0.1:8080`, illetve `cd web && npm run dev`.
- Első alkalommal: `.\build.ps1 e2e-install` / `make e2e-install` (Playwright Chromium).
- Teljes kapu: `.\build.ps1 verify` / `make verify` (svelte-check, prettier, gofmt, go vet, langcheck, Go-tesztek lefedettségi kapuval, Vitest, E2E a valódi binárison).
- Részcélok: `check`, `test-go`, `test-web`, `test-e2e`, `image`, `clean`.

## CI és image

- `.github/workflows/ci.yml`: web, Go (1.26.x és 1.27.x, race, govulncheck), Go Windowson, E2E, keresztfordított binárisok; push után, ha minden teszt zöld, az `image` job meghívja a `docker.yml`-t.
- `.github/workflows/docker.yml`: amd64 build és smoke, majd többplatformos (amd64, arm64) push a ghcr.io-ra; PR-en csak build és smoke.
- Címkék: ág, `vX.Y.Z` és `X.Y`, rövid SHA, `latest` a `main` ágon. A workflow-k első futását a 2026-11-22-i első push indítja; az eredményt a GitHub Actions oldalán találod.

## Mentés és frissítés

- Mentsd az adatkönyvtárat (`app.db`, `app.db-wal`, `app.db-shm`, `secret.key`), lehetőleg a szerver leállítása után. A `secret.key` nélkül a TOTP-titkok és a helyreállító kódok használhatatlanok, és a program nem indul.
- Frissítési megjegyzés: az adatkezelési tájékoztató szövege a fejlesztés D fázisában bővült. Ha egy meglévő példány felhasználói a régi szöveget nyugtázták, az admin adjon ki új verziót (Rendszerbeállítások, „Új verzió kiadása”, vagy `POST /api/admin/settings/privacy-notice/bump`). Friss telepítésnél nincs teendő.

## Dokumentáció

- [manual/MANUAL.md](manual/MANUAL.md): magyar felhasználói kézikönyv képernyőképekkel.
- [SUMMARY.md](SUMMARY.md): a fejlesztés állapota és térképe (AI-folytatáshoz).
- [ai/BASE-SYSTEM-PLAN.md](ai/BASE-SYSTEM-PLAN.md): a fejlesztési terv; követelmények: [ai/BASE-SYSTEM-PROMPT.md](ai/BASE-SYSTEM-PROMPT.md), [ai/TECHNOLOGY.md](ai/TECHNOLOGY.md), [ai/GRAPHICS.md](ai/GRAPHICS.md).
- Licenc: MIT ([LICENSE](LICENSE)).
