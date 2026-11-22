# KINEXIUM Bookmark Manager

## Feladat összefoglaló

- A feladat egy WEB alapú alkalmazás fejlesztése ami link-ket tárol és a gyűjteményt kezeli.
- Projekt név: Kinex Mark

## Technológia

- Felület: SvelteKit, Svelte 5, TypeScript (kliensoldali alkalmazás, statikus buildként).
- Megjelenés: Tailwind CSS v4, shadcn-svelte (saját kódbázisba másolt komponensek).
- Backend: Go, `net/http` (API és a felület kiszolgálása).
- Adat: SQLite `modernc.org/sqlite`  (Tiszta Go-s driver, CGO nélkül).
- Csomagolás: `go:embed` (Docker, egy bináris, illetve minimális konténer).

- A technológia részletes leírása a ./ai/TECHNOLOGY.md fájlban.
- A dizájn részletes leírása a ./ai/GRAPHICS.md fájlban.

# Funkciók

- Az alkalmazásnak megfelelő adminisztrációs felülettel rendelkezzen ahol a törzsadatokat lehet szerkeszteni.
- Felhasználók kezelése (adminisztrátorok, fejlesztők és ügyfelek).
- Link-ek gyors rögzítése drag-and-drop vagy kézi hozzáadás.
- Ikon (favicon) letöltése a link-hez.
- Cimke (title) letöltlése a link-hez (ha elérhető).
- Rövid leírás hozzáadásának lehetősége.
- Színes tag-ek hozzáadásának lehetősége.
- A link-ek csoportosítása csak a tag-ek alapján történjen, más hierarchikus csoportosítás nem szükséges.
- Link-ek lista és csempe alapú megjelenítése.
- Szűrés szöveg vagy tag alapján.
- Megjelenített adatok ki/be kapcsolhatósága (ikon, URL, címke, leírás).
- A link-ek használat alapján történő sorbarakása (gyakran használt link-ek előre).
- A link-ek kitűzésének lehetősége (pin-ed címke mindig elöl, de ebben az esetben is figyelni kell a rangsorolást, azaz legelöl a gyakran használt és pin-ed).
- Automatikus archiválás (beállított idő után a link-ek archivált állapotba kerülnek és csak akkor jelennek meg ha ezt bekapcsoljuk).
- Az archiválás akkor történik meg ha a megadott időn belül nem használtuk a link-t (pl. 3 hónap).
- A linkek gyors másolásának a lehetősége (URL másolás vagy URL másolás markdown formában - [Leírás](URL)).
- A megjelenítésnél a rövid leírás az elsődleges, ha az nincs akkor a letöltött címke, ha az sincs akkor az URL. Ha az URL hosszú csak a domain jelenjen meg.
- Az összes információ is elérhető legyen a link-ről (ikon, URL, címke, leírás, használati statisztika, rögzítés és utolsó használat időpontja).

## Grafikák

- Az grafikák létrehozásához gemini-3.1-flash-image modellt használd.
- A modell használatára példa a graphics/generate_graphics.py fájlban, GEMINI_API_KEY változó be van állítva.
- A grafika stilusát és szineit illeszteni kell a ./ai/GRAPHICS.md fájlban meghatározott értékekhez.

## Követelmények

- A program minden eleme legyen lokalizálható (legalább magyar, angol és német nyelv támogatása).
- A forráskódban csak angol nyelv legyen használva.
- A GUI-n legyen nyelv választási lehetőség az adminisztrációs és a felhasználói felületen is.
- A GUI igényes és jól kezelhető legyen.
- GDPR megfelelőség.
- Az WEB felület csak megfelelő autentikáció után legyen elérhető (bejelentkezés).
- A bejelentkezéshez opcionálisan két-faktoros autentikáció is legyen bekapcsolható.
- A felhasználók csak saját adatokat lássák és a megosztott link-ket.
- A fejlesztés tartalmazzon teszteket is a lehető legnagyobb lefedettséggel.
- A fejlesztés során egy grafika is készüljön a nyitó oldalra, megfelelő minőségben, high-tech, minimal, elegant stílusban.
  A grafika látványosan mutassa be a program funkcióit (gyors link tárolás, tag-ek kezelése, szűrés, rangsorolás, automatikus elévülés).
- A fejlesztés megkezdése előtt készüljön egy részletes fejlesztési terv az ./ai/BASE-SYSTEM-PLAN.md fájlba.
- A fejlesztés befejezése után készüljön egy SUMMARY.md fájl ami minden releváns információt tartalmaz.
- A SUMMARY.md legfontosabb feladata a fejlesztés aktuális állapotának rögzítése, hogy az AI fejlesztés folytatható
  legyen ennek a fájlnak az ismeretében, azaz ne kelljen a teljes forráskódot átnézni a folytatáshoz.
- A frontend tervezéshez használd a frontend-design plugin-t, de mindenképpen a legfrissebb komponenseket használd.
- A GUI teszteléshez használd a playwright plugin-t.
- A fejlesztéshez használj Opus 5.5 modellt.
- A kód felderítéséhez használj Sonnet 5.5 modellt.

## Záró tételek

- A fejlesztés befejezése után készüljön egy magyar kézikönyv a ./manual/MANUAL.md fájlban képernyőképekkel.
- Az alap GitHub README.md fájl helyett készüljön egy rövid összefoglaló a futtatásról.
- Készüljön megfelelő GitHub CI/CD szkript is ami képes docker image-t előállítani 
