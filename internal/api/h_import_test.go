package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// importTestNetscape is a browser bookmark file: a folder with an invalid
// entry, tags, a description and an entity in a title.
const importTestNetscape = `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
    <DT><H3>Munka</H3>
    <DL><p>
        <DT><A HREF="HTTPS://Work.Example:443" ADD_DATE="1700000000" TAGS="alfa,béta">Work &amp; co</A>
        <DD>Első leírás
        <DT><A HREF="ftp://bad.example/">Bad scheme</A>
    </DL><p>
    <DT><A HREF="https://plain.example/page">Plain</A>
</DL><p>
`

// importTestJSON builds a kinex-mark-links/1 file.
func importTestJSON(t *testing.T, links ...map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{"format": "kinex-mark-links/1", "exportedAt": "2026-01-01T00:00:00Z", "links": links})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestImportNetscapeAndJSON(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	html := c.withHeader("Content-Type", "text/html; charset=utf-8")

	res := decode[model.ImportResult](t, html.post("/api/links/import", importTestNetscape))
	if res != (model.ImportResult{Total: 3, Created: 2, Duplicates: 0, Invalid: 1}) {
		t.Errorf("netscape result = %+v", res)
	}
	page := decode[LinkPage](t, c.get("/api/links?q=work"))
	if len(page.Items) != 1 {
		t.Fatalf("imported work link not found: %+v", page)
	}
	work := decode[LinkDetail](t, c.get(linkTestPath(page.Items[0].ID)))
	if work.URL != "https://work.example/" || work.Title != "Work & co" || work.TitleSource != model.TitleManual ||
		work.Description != "Első leírás" || work.FetchStatus != model.FetchPending {
		t.Errorf("imported link = %+v", work)
	}
	var names []string
	for _, tg := range work.Tags {
		names = append(names, tg.Name)
		if tg.Color != model.TagColorNeutral {
			t.Errorf("created tag color = %s", tg.Color)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"Munka", "alfa", "béta"}) {
		t.Errorf("tags = %v", names)
	}
	if got := e.jobs.enqueued(); len(got) != 0 {
		t.Errorf("import enqueued %v; the sweeper handles imported links", got)
	}

	again := decode[model.ImportResult](t, html.post("/api/links/import", importTestNetscape))
	if again != (model.ImportResult{Total: 3, Created: 0, Duplicates: 2, Invalid: 1}) {
		t.Errorf("second import = %+v", again)
	}

	file := importTestJSON(t,
		map[string]any{"url": "https://json.example/a", "title": "Cím‮\u0007 vége", "description": "Sor⁦ egy\nSor kettő", "tags": []string{"json"}, "addedAt": "2025-06-01T12:00:00Z"},
		map[string]any{"url": "https://json.example/a", "title": "same url again"},
		map[string]any{"url": "https://user:pw@json.example/", "title": "credentials"},
		map[string]any{"url": "", "title": "empty"},
		map[string]any{"url": "https://plain.example/page", "title": "existing"},
	)
	res = decode[model.ImportResult](t, c.post("/api/links/import", file))
	if res != (model.ImportResult{Total: 5, Created: 1, Duplicates: 2, Invalid: 2}) {
		t.Errorf("json result = %+v", res)
	}
	j := decode[LinkPage](t, c.get("/api/links?q=json.example")).Items
	if len(j) != 1 || j[0].Title != "Cím vége" || j[0].Description != "Sor egy\nSor kettő" {
		t.Errorf("cleaned json link = %+v", j)
	}
	if len(j) == 1 && !j[0].CreatedAt.Equal(importTestTime(t, "2025-06-01T12:00:00Z")) {
		t.Errorf("createdAt = %v, want the addedAt", j[0].CreatedAt)
	}
	empty := decode[model.ImportResult](t, c.post("/api/links/import", importTestJSON(t)))
	if empty != (model.ImportResult{}) {
		t.Errorf("empty import = %+v", empty)
	}
}

// An addedAt that is a negative year in UTC used to be stored as
// "-0001-…", after which every read of the link (and of each list that held
// it, also for share recipients) failed with 500. Such a value now counts
// as missing: the link is created at the import time.
func TestImportAddedAtBeforeEpoch(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	dev, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	file := importTestJSON(t,
		map[string]any{"url": "https://year-zero.example/", "title": "Year zero", "addedAt": "0000-01-01T00:30:00+01:00"},
		map[string]any{"url": "https://before-epoch.example/", "title": "Before epoch", "addedAt": "1969-07-20T20:17:00Z"})
	res := decode[model.ImportResult](t, dev.post("/api/links/import", file))
	if res != (model.ImportResult{Total: 2, Created: 2}) {
		t.Fatalf("import = %+v", res)
	}
	page := decode[LinkPage](t, dev.get("/api/links"))
	if len(page.Items) != 2 {
		t.Fatalf("list = %+v", page)
	}
	for _, it := range page.Items {
		if !it.CreatedAt.Equal(e.clock.Now()) {
			t.Errorf("%s: createdAt = %v, want the import time", it.URL, it.CreatedAt)
		}
		expectStatus(t, dev.get(linkTestPath(it.ID)), http.StatusOK)
		expectStatus(t, dev.post(linkTestPath(it.ID, "/shares"), map[string]any{"targetType": "all"}), http.StatusCreated)
	}
	expectStatus(t, dev.get("/api/links/export?format=json"), http.StatusOK)
	expectStatus(t, dev.get("/api/links/export?format=html"), http.StatusOK)
	expectStatus(t, dev.get("/api/me/export"), http.StatusOK)

	cust, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	if got := decode[LinkPage](t, cust.get("/api/links")); got.Total != 2 {
		t.Errorf("recipient list total = %d", got.Total)
	}
}

// importTestTime parses an RFC 3339 test timestamp.
func importTestTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := model.ParseTime(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// importTestErrReader fails every read.
type importTestErrReader struct{}

func (importTestErrReader) Read([]byte) (int, error) { return 0, errors.New("injected read failure") }

func TestImportBodyReadFailure(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	u := e.seedUser(t, model.RoleCustomer)
	token, _ := e.createSession(t, u.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/links/import", importTestErrReader{})
	req.Host = "127.0.0.1"
	req.Header.Set("Content-Type", "text/html")
	req.Header.Set(headerKinexRequest, "1")
	req.AddCookie(&http.Cookie{Name: e.cookieName(), Value: token})
	rec := e.serve(req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500; body %s", rec.Code, rec.Body)
	}
	if e.logs.count("injected read failure") != 1 {
		t.Errorf("read failure not logged:\n%s", e.logs.String())
	}
}

func TestImportErrors(t *testing.T) {
	t.Parallel()
	e, faults := linkTestFaultEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	html := c.withHeader("Content-Type", "text/html")

	expectError(t, c.withHeader("Content-Type", "text/plain").post("/api/links/import", importTestNetscape), http.StatusBadRequest, codeImportInvalid)
	expectError(t, c.withHeader("Content-Type", "multipart/form-data; boundary=x").post("/api/links/import", "x"), http.StatusBadRequest, codeImportInvalid)
	expectError(t, c.withHeader("Content-Type", ";;").post("/api/links/import", "x"), http.StatusBadRequest, codeImportInvalid)
	expectError(t, c.do(http.MethodPost, "/api/links/import", nil), http.StatusBadRequest, codeImportInvalid)
	expectError(t, html.post("/api/links/import", "<html><body>no bookmarks</body></html>"), http.StatusBadRequest, codeImportInvalid)
	expectError(t, c.post("/api/links/import", "{"), http.StatusBadRequest, codeImportInvalid)
	expectError(t, c.post("/api/links/import", `{"format":"other/1","links":[]}`), http.StatusBadRequest, codeImportInvalid)

	many := make([]map[string]any, model.MaxImportEntries+1)
	for i := range many {
		many[i] = map[string]any{"url": fmt.Sprintf("https://e.example/%d", i)}
	}
	expectError(t, c.post("/api/links/import", importTestJSON(t, many...)), http.StatusBadRequest, codeImportTooManyEntries)
	var netscape strings.Builder
	netscape.WriteString("<!DOCTYPE NETSCAPE-Bookmark-file-1>\n<DL><p>\n")
	for i := 0; i <= model.MaxImportEntries; i++ {
		fmt.Fprintf(&netscape, "<DT><A HREF=\"https://n.example/%d\">n</A>\n", i)
	}
	netscape.WriteString("</DL><p>\n")
	expectError(t, html.post("/api/links/import", netscape.String()), http.StatusBadRequest, codeImportTooManyEntries)

	big := "<!DOCTYPE NETSCAPE-Bookmark-file-1>\n<!--" + strings.Repeat("x", model.MaxImportBytes) + "-->"
	expectError(t, html.post("/api/links/import", big), http.StatusRequestEntityTooLarge, codePayloadTooLarge)

	faults.set("ImportLinks", fmt.Errorf("import: %w", model.ErrLimitReached))
	expectError(t, html.post("/api/links/import", importTestNetscape), http.StatusConflict, codeLinkLimitReached)
	faults.set("ImportLinks", errors.New("injected import failure"))
	expectError(t, html.post("/api/links/import", importTestNetscape), http.StatusInternalServerError, codeInternalError)
	faults.set("ImportLinks", nil)

	if page := decode[LinkPage](t, c.get("/api/links?archived=include")); page.Total != 0 {
		t.Errorf("failed imports created %d links", page.Total)
	}
}

func TestImportExport(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	a, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	other, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	tag := linkTestCreateTag(t, a, "Export")
	linkTestCreate(t, a, map[string]any{"url": "https://one.example/", "title": "Egy", "description": "Leírás", "tagIds": []int64{tag.ID}})
	linkTestCreateURL(t, a, "https://two.example/")
	shared := linkTestCreateURL(t, other, "https://foreign.example/")
	expectStatus(t, other.post(linkTestPath(shared.ID, "/shares"), map[string]any{"targetType": "all"}), http.StatusCreated)

	res := a.get("/api/links/export?format=json")
	body := expectStatus(t, res, http.StatusOK)
	if res.Header.Get("Content-Type") != "application/json; charset=utf-8" ||
		res.Header.Get("Content-Disposition") != `attachment; filename="kinex-mark-links-20260105.json"` ||
		res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("json export headers = %v", res.Header)
	}
	var file struct {
		Format string `json:"format"`
		Links  []struct {
			URL         string   `json:"url"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
			AddedAt     *string  `json:"addedAt"`
		} `json:"links"`
	}
	decodeInto(t, body, &file)
	if file.Format != "kinex-mark-links/1" || len(file.Links) != 2 {
		t.Fatalf("export = %s", body)
	}
	if l := file.Links[0]; l.URL != "https://one.example/" || l.Title != "Egy" || l.Description != "Leírás" ||
		!slices.Equal(l.Tags, []string{"Export"}) || l.AddedAt == nil {
		t.Errorf("first exported link = %+v", l)
	}
	if file.Links[1].Tags == nil {
		t.Error("tags of an untagged link encode as null")
	}
	if strings.Contains(string(body), "foreign.example") {
		t.Error("the export contains a link shared with the user")
	}

	res = a.get("/api/links/export?format=html")
	body = expectStatus(t, res, http.StatusOK)
	if res.Header.Get("Content-Type") != "text/html; charset=utf-8" ||
		res.Header.Get("Content-Disposition") != `attachment; filename="kinex-mark-links-20260105.html"` {
		t.Errorf("html export headers = %v", res.Header)
	}
	if !strings.Contains(string(body), "<!DOCTYPE NETSCAPE-Bookmark-file-1>") || !strings.Contains(string(body), "https://one.example/") ||
		strings.Contains(string(body), "foreign.example") {
		t.Errorf("html export = %s", body)
	}

	// Round trip into another account.
	b, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	got := decode[model.ImportResult](t, b.withHeader("Content-Type", "text/html").post("/api/links/import", body))
	if got.Created != 2 || got.Invalid != 0 {
		t.Errorf("html round trip = %+v", got)
	}
	jsonBody := readBody(t, a.get("/api/links/export?format=json"))
	got = decode[model.ImportResult](t, b.post("/api/links/import", jsonBody))
	if got.Duplicates != 2 || got.Created != 0 {
		t.Errorf("json round trip = %+v", got)
	}

	for _, q := range []string{"", "?format=", "?format=xml", "?format=JSON"} {
		body := expectError(t, a.get("/api/links/export"+q), http.StatusBadRequest, codeInvalidQuery)
		if body.Fields["format"] != "invalid" {
			t.Errorf("export%s fields = %v", q, body.Fields)
		}
	}
}
