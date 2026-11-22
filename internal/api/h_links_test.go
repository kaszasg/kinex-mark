package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

// linkTestLoginAs is e.loginAs with a client of its own. The harness
// clients share the *http.Client of httptest.Server.Client(), whose Jar each
// new client replaces; without the copy every earlier client would send the
// cookies of the last one. Reported to B5; remove once client() copies.
func linkTestLoginAs(t *testing.T, e *testEnv, role model.Role) (*apiClient, model.User) {
	t.Helper()
	c, u := e.loginAs(t, role)
	return linkTestIsolate(c), u
}

// linkTestIsolate gives c a private copy of its http.Client (same jar).
func linkTestIsolate(c *apiClient) *apiClient {
	hc := *c.http
	c.http = &hc
	return c
}

// linkTestPath returns /api/links/<id> plus an optional suffix.
func linkTestPath(id int64, suffix ...string) string {
	return "/api/links/" + strconv.FormatInt(id, 10) + strings.Join(suffix, "")
}

// linkTestCreate creates a link through the API and expects 201.
func linkTestCreate(t *testing.T, c *apiClient, body map[string]any) LinkDetail {
	t.Helper()
	return decodeStatus[LinkDetail](t, c.post("/api/links", body), http.StatusCreated)
}

// linkTestCreateURL creates a link with only a URL.
func linkTestCreateURL(t *testing.T, c *apiClient, url string) LinkDetail {
	t.Helper()
	return linkTestCreate(t, c, map[string]any{"url": url})
}

// linkTestCreateTag creates a tag through the API and expects 201.
func linkTestCreateTag(t *testing.T, c *apiClient, name string) model.Tag {
	t.Helper()
	return decodeStatus[model.Tag](t, c.post("/api/tags", map[string]any{"name": name}), http.StatusCreated)
}

// linkTestIconBlob builds a stored icon with its hash.
func linkTestIconBlob(contentType string, data []byte) *model.IconBlob {
	sum := sha256.Sum256(data)
	return &model.IconBlob{Data: data, ContentType: contentType, SHA256: hex.EncodeToString(sum[:])}
}

// linkTestFetched completes the pending fetch of a link as the metadata
// worker would: status ok with the given title and icon.
func linkTestFetched(t *testing.T, e *testEnv, linkID int64, title string, icon *model.IconBlob) {
	t.Helper()
	linkTestFetchResult(t, e, linkID, model.FetchResult{Status: model.FetchOK, Title: title, Icon: icon})
}

// linkTestFetchResult leases the link and applies r to it.
func linkTestFetchResult(t *testing.T, e *testEnv, linkID int64, r model.FetchResult) {
	t.Helper()
	job, err := e.store.BeginFetch(e.ctx(), linkID, model.FetchLease)
	if err != nil {
		t.Fatalf("BeginFetch(%d): %v", linkID, err)
	}
	r.LinkID, r.URL = linkID, job.URL
	if err := e.store.ApplyFetchResult(e.ctx(), r); err != nil {
		t.Fatalf("ApplyFetchResult(%d): %v", linkID, err)
	}
}

// linkTestCountID counts how often id occurs in ids.
func linkTestCountID(ids []int64, id int64) int {
	n := 0
	for _, v := range ids {
		if v == id {
			n++
		}
	}
	return n
}

// linkTestFaults holds the errors a linkTestFaultStore injects, by method.
type linkTestFaults struct {
	mu   sync.Mutex
	errs map[string]error
}

// set makes method fail with err (nil removes the fault).
func (f *linkTestFaults) set(method string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err == nil {
		delete(f.errs, method)
		return
	}
	f.errs[method] = err
}

// err returns the injected error of method.
func (f *linkTestFaults) err(method string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.errs[method]
}

// linkTestFaultStore wraps the real store and fails the link, share, tag and
// import methods on demand.
type linkTestFaultStore struct {
	Store
	f *linkTestFaults
}

func (s linkTestFaultStore) Preferences(ctx context.Context, id int64) (model.Preferences, error) {
	if err := s.f.err("Preferences"); err != nil {
		return model.Preferences{}, err
	}
	return s.Store.Preferences(ctx, id)
}

func (s linkTestFaultStore) Links(ctx context.Context, v model.Viewer, q model.LinkQuery) (model.LinkPage, error) {
	if err := s.f.err("Links"); err != nil {
		return model.LinkPage{}, err
	}
	return s.Store.Links(ctx, v, q)
}

func (s linkTestFaultStore) Link(ctx context.Context, v model.Viewer, id int64) (model.LinkDetail, error) {
	if err := s.f.err("Link"); err != nil {
		return model.LinkDetail{}, err
	}
	return s.Store.Link(ctx, v, id)
}

func (s linkTestFaultStore) LinkItem(ctx context.Context, v model.Viewer, id int64) (model.LinkItem, error) {
	if err := s.f.err("LinkItem"); err != nil {
		return model.LinkItem{}, err
	}
	return s.Store.LinkItem(ctx, v, id)
}

func (s linkTestFaultStore) CreateLink(ctx context.Context, n model.NewLink) (model.LinkDetail, error) {
	if err := s.f.err("CreateLink"); err != nil {
		return model.LinkDetail{}, err
	}
	return s.Store.CreateLink(ctx, n)
}

func (s linkTestFaultStore) RecordVisit(ctx context.Context, v model.Viewer, id int64) (model.LinkState, error) {
	if err := s.f.err("RecordVisit"); err != nil {
		return model.LinkState{}, err
	}
	return s.Store.RecordVisit(ctx, v, id)
}

func (s linkTestFaultStore) SetLinkTags(ctx context.Context, v model.Viewer, id int64, tagIDs []int64) error {
	if err := s.f.err("SetLinkTags"); err != nil {
		return err
	}
	return s.Store.SetLinkTags(ctx, v, id, tagIDs)
}

func (s linkTestFaultStore) LinkIcon(ctx context.Context, v model.Viewer, id int64) (model.IconBlob, error) {
	if err := s.f.err("LinkIcon"); err != nil {
		return model.IconBlob{}, err
	}
	return s.Store.LinkIcon(ctx, v, id)
}

func (s linkTestFaultStore) Shares(ctx context.Context, v model.Viewer, linkID int64) ([]model.Share, error) {
	if err := s.f.err("Shares"); err != nil {
		return nil, err
	}
	return s.Store.Shares(ctx, v, linkID)
}

func (s linkTestFaultStore) DeleteShare(ctx context.Context, v model.Viewer, linkID, shareID int64) error {
	if err := s.f.err("DeleteShare"); err != nil {
		return err
	}
	return s.Store.DeleteShare(ctx, v, linkID, shareID)
}

func (s linkTestFaultStore) AppendAudit(ctx context.Context, e model.AuditEntry) error {
	if err := s.f.err("AppendAudit"); err != nil {
		return err
	}
	return s.Store.AppendAudit(ctx, e)
}

func (s linkTestFaultStore) Tags(ctx context.Context, userID int64, role model.Role) ([]model.Tag, error) {
	if err := s.f.err("Tags"); err != nil {
		return nil, err
	}
	return s.Store.Tags(ctx, userID, role)
}

func (s linkTestFaultStore) CreateTag(ctx context.Context, userID int64, name string, color model.TagColor) (model.Tag, error) {
	if err := s.f.err("CreateTag"); err != nil {
		return model.Tag{}, err
	}
	return s.Store.CreateTag(ctx, userID, name, color)
}

func (s linkTestFaultStore) ExportLinks(ctx context.Context, ownerID int64) ([]model.ExportedLink, error) {
	if err := s.f.err("ExportLinks"); err != nil {
		return nil, err
	}
	return s.Store.ExportLinks(ctx, ownerID)
}

func (s linkTestFaultStore) ImportLinks(ctx context.Context, ownerID int64, entries []model.ImportEntry) (model.ImportResult, []int64, error) {
	if err := s.f.err("ImportLinks"); err != nil {
		return model.ImportResult{}, nil, err
	}
	return s.Store.ImportLinks(ctx, ownerID, entries)
}

// linkTestFaultEnv starts an environment whose store fails on demand.
func linkTestFaultEnv(t *testing.T) (*testEnv, *linkTestFaults) {
	t.Helper()
	faults := &linkTestFaults{errs: map[string]error{}}
	e := newEnv(t, withDeps(func(d *Deps) {
		d.Store = linkTestFaultStore{Store: d.Store, f: faults}
	}))
	return e, faults
}

func TestLinkCreate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := linkTestLoginAs(t, e, model.RoleCustomer)
	tag := linkTestCreateTag(t, c, "Munka")

	d := linkTestCreate(t, c, map[string]any{
		"url":         "  HTTPS://Example.COM:443?q=1#top ",
		"title":       "  Kézi cím  ",
		"description": "Első sor\r\nmásodik sor",
		"tagIds":      []int64{tag.ID},
		"pinned":      true,
	})
	if d.URL != "https://example.com/?q=1#top" || d.Host != "example.com" {
		t.Errorf("url/host = %q %q", d.URL, d.Host)
	}
	if d.Title != "Kézi cím" || d.TitleSource != model.TitleManual {
		t.Errorf("title = %q (%s), want the trimmed manual title", d.Title, d.TitleSource)
	}
	if d.Description != "Első sor\nmásodik sor" {
		t.Errorf("description = %q", d.Description)
	}
	if d.FetchStatus != model.FetchPending || d.FetchAttempts != 0 || d.FetchError != nil || d.FetchedAt != nil {
		t.Errorf("fetch fields = %s %d %v %v", d.FetchStatus, d.FetchAttempts, d.FetchError, d.FetchedAt)
	}
	if !d.Pinned || d.ArchiveDueAt != nil || !d.IsOwner || d.Owner.ID != u.ID || d.IconURL != nil {
		t.Errorf("state = pinned %v due %v owner %v/%d icon %v", d.Pinned, d.ArchiveDueAt, d.IsOwner, d.Owner.ID, d.IconURL)
	}
	if len(d.Tags) != 1 || d.Tags[0].ID != tag.ID || !slices.Equal(d.TagIDs, []int64{tag.ID}) {
		t.Errorf("tags = %+v / %v", d.Tags, d.TagIDs)
	}
	if d.Shares == nil || len(d.Shares) != 0 {
		t.Errorf("shares = %v, want []", d.Shares)
	}
	if got := e.jobs.enqueued(); !slices.Equal(got, []int64{d.ID}) {
		t.Errorf("enqueued = %v, want [%d]", got, d.ID)
	}

	plain := linkTestCreateURL(t, c, "http://example.org")
	if plain.URL != "http://example.org/" || plain.Title != "" || plain.TitleSource != model.TitleNone || plain.Pinned {
		t.Errorf("plain link = %+v", plain)
	}
	if !plain.InactiveSince.Equal(testutil.T0) || plain.ArchiveDueAt == nil ||
		!plain.ArchiveDueAt.Equal(testutil.T0.AddDate(0, 0, 90)) {
		t.Errorf("inactiveSince %v archiveDueAt %v, want T0 and T0+90d", plain.InactiveSince, plain.ArchiveDueAt)
	}

	// A full queue is not an error: the sweeper picks the link up later.
	e.jobs.setFull(true)
	linkTestCreateURL(t, c, "https://queue.example/")
}

func TestLinkCreateValidation(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	other, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	foreign := linkTestCreateTag(t, other, "Idegen")
	own := linkTestCreateTag(t, c, "Saját")
	long := "https://example.com/" + strings.Repeat("a", 2040)
	many := make([]int64, 21)
	for i := range many {
		many[i] = int64(i + 1)
	}

	cases := []struct {
		name string
		body map[string]any
		want map[string]string
	}{
		{"missing url", map[string]any{}, map[string]string{"url": "required"}},
		{"blank url", map[string]any{"url": "   "}, map[string]string{"url": "required"}},
		{"too long url", map[string]any{"url": long}, map[string]string{"url": "too_long"}},
		{"no host", map[string]any{"url": "https://"}, map[string]string{"url": "invalid_format"}},
		{"ftp", map[string]any{"url": "ftp://example.com/"}, map[string]string{"url": "not_allowed"}},
		{"javascript", map[string]any{"url": "javascript:alert(1)"}, map[string]string{"url": "invalid_format"}},
		{"credentials", map[string]any{"url": "https://user:pw@example.com/"}, map[string]string{"url": "credentials_in_url"}},
		{"long title", map[string]any{"url": "https://a.example/", "title": strings.Repeat("é", 501)}, map[string]string{"title": "too_long"}},
		{"title newline", map[string]any{"url": "https://a.example/", "title": "a\nb"}, map[string]string{"title": "invalid"}},
		{"title bidi", map[string]any{"url": "https://a.example/", "title": "abc‮def"}, map[string]string{"title": "invalid"}},
		{"long description", map[string]any{"url": "https://a.example/", "description": strings.Repeat("x", 501)}, map[string]string{"description": "too_long"}},
		{"description control", map[string]any{"url": "https://a.example/", "description": "a\u0000b"}, map[string]string{"description": "invalid"}},
		{"description isolate", map[string]any{"url": "https://a.example/", "description": "a⁦b"}, map[string]string{"description": "invalid"}},
		{"too many tags", map[string]any{"url": "https://a.example/", "tagIds": many}, map[string]string{"tagIds": "too_long"}},
		{"tag id zero", map[string]any{"url": "https://a.example/", "tagIds": []int64{0}}, map[string]string{"tagIds": "invalid"}},
		{"repeated tag", map[string]any{"url": "https://a.example/", "tagIds": []int64{own.ID, own.ID}}, map[string]string{"tagIds": "duplicate"}},
		{"several fields", map[string]any{"url": "gopher://example.com/", "title": "\u0007"}, map[string]string{"url": "not_allowed", "title": "invalid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expectFields(t, c.post("/api/links", tc.body), tc.want)
		})
	}
	t.Run("foreign tag", func(t *testing.T) {
		expectFields(t, c.post("/api/links", map[string]any{"url": "https://b.example/", "tagIds": []int64{foreign.ID}}),
			map[string]string{"tagIds": "unknown"})
	})
	t.Run("missing tag", func(t *testing.T) {
		expectFields(t, c.post("/api/links", map[string]any{"url": "https://b.example/", "tagIds": []int64{own.ID, 99999}}),
			map[string]string{"tagIds": "unknown"})
	})
	t.Run("invalid json", func(t *testing.T) {
		expectError(t, c.post("/api/links", "{"), http.StatusBadRequest, codeInvalidJSON)
		expectError(t, c.post("/api/links", map[string]any{"url": "https://c.example/", "owner": 1}), http.StatusBadRequest, codeInvalidJSON)
		expectError(t, c.post("/api/links", map[string]any{"url": 5}), http.StatusBadRequest, codeInvalidJSON)
	})
	if got := e.jobs.enqueued(); len(got) != 0 {
		t.Errorf("rejected requests enqueued %v", got)
	}
	// Nothing was stored.
	page := decode[LinkPage](t, c.get("/api/links?archived=include"))
	if page.Total != 0 {
		t.Errorf("total = %d after rejected creates", page.Total)
	}
}

func TestLinkCreateDuplicateAndLimit(t *testing.T) {
	t.Parallel()
	e, faults := linkTestFaultEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	first := linkTestCreateURL(t, c, "https://example.com/")
	body := expectError(t, c.post("/api/links", map[string]any{"url": "HTTPS://EXAMPLE.com:443"}), http.StatusConflict, codeLinkDuplicate)
	if id, _ := body.Details["linkId"].(float64); int64(id) != first.ID {
		t.Errorf("details.linkId = %v, want %d", body.Details["linkId"], first.ID)
	}
	// Another user may store the same URL.
	other, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	linkTestCreateURL(t, other, "https://example.com/")

	faults.set("CreateLink", fmt.Errorf("insert: %w", model.ErrLimitReached))
	expectError(t, c.post("/api/links", map[string]any{"url": "https://new.example/"}), http.StatusConflict, codeLinkLimitReached)
	expectError(t, c.post("/api/links/quick", map[string]any{"text": "https://new.example/"}), http.StatusConflict, codeLinkLimitReached)
}

func TestLinkQuickCapture(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)

	created := []struct {
		name, text, wantURL string
	}{
		{"uri list with comment", "# dragged from a browser\r\nhttps://example.org/page?x=1\r\nhttps://ignored.example/\r\n", "https://example.org/page?x=1"},
		{"plain text with punctuation", "Nézd meg: https://example.net/cikk. Jó!", "https://example.net/cikk"},
		{"bare domain", "example.com/docs", "https://example.com/docs"},
		{"bare host", "www.example.com", "https://www.example.com/"},
	}
	for _, tc := range created {
		t.Run(tc.name, func(t *testing.T) {
			res := decodeStatus[QuickCaptureResult](t, c.post("/api/links/quick", map[string]any{"text": tc.text}), http.StatusCreated)
			if !res.Created || res.Link.URL != tc.wantURL || res.Link.FetchStatus != model.FetchPending ||
				res.Link.TitleSource != model.TitleNone {
				t.Errorf("result = created %v url %q status %s source %s", res.Created, res.Link.URL, res.Link.FetchStatus, res.Link.TitleSource)
			}
			if linkTestCountID(e.jobs.enqueued(), res.Link.ID) != 1 {
				t.Errorf("link %d not enqueued once: %v", res.Link.ID, e.jobs.enqueued())
			}
		})
	}

	t.Run("dropped anchor title is cleaned", func(t *testing.T) {
		res := decodeStatus[QuickCaptureResult](t, c.post("/api/links/quick", map[string]any{
			"text": "https://title.example/", "title": " Cím‮\n  szöveg ",
		}), http.StatusCreated)
		if res.Link.Title != "Cím szöveg" || res.Link.TitleSource != model.TitleManual {
			t.Errorf("title = %q (%s)", res.Link.Title, res.Link.TitleSource)
		}
	})

	t.Run("existing url answers 200 with the existing link", func(t *testing.T) {
		before := len(e.jobs.enqueued())
		existing := linkTestCreateURL(t, c, "https://existing.example/a")
		res := decodeStatus[QuickCaptureResult](t, c.post("/api/links/quick", map[string]any{"text": "HTTPS://existing.example/a"}), http.StatusOK)
		if res.Created || res.Link.ID != existing.ID {
			t.Errorf("result = created %v id %d, want false %d", res.Created, res.Link.ID, existing.ID)
		}
		if got := len(e.jobs.enqueued()); got != before+1 {
			t.Errorf("enqueued %d times, want only the create", got-before)
		}
	})

	t.Run("errors", func(t *testing.T) {
		expectError(t, c.post("/api/links/quick", map[string]any{"text": "nincs itt link"}), http.StatusBadRequest, codeNoURLFound)
		expectError(t, c.post("/api/links/quick", map[string]any{"text": "# only a comment\nftp://x"}), http.StatusBadRequest, codeNoURLFound)
		expectFields(t, c.post("/api/links/quick", map[string]any{"text": "  \n "}), map[string]string{"text": "required"})
		expectFields(t, c.post("/api/links/quick", map[string]any{}), map[string]string{"text": "required"})
		expectFields(t, c.post("/api/links/quick", map[string]any{"text": strings.Repeat("ű", 8193)}), map[string]string{"text": "too_long"})
		expectFields(t, c.post("/api/links/quick", map[string]any{"text": "https://user:pw@example.com/"}),
			map[string]string{"url": "credentials_in_url"})
		expectFields(t, c.post("/api/links/quick", map[string]any{"text": "https://example.com/" + strings.Repeat("a", 2100)}),
			map[string]string{"url": "too_long"})
		expectError(t, c.post("/api/links/quick", map[string]any{"text": "https://x.example/", "url": "y"}), http.StatusBadRequest, codeInvalidJSON)
	})
}

func TestLinkList(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	owner, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	work := linkTestCreateTag(t, c, "Munka")
	home := linkTestCreateTag(t, c, "Otthon")

	alpha := linkTestCreate(t, c, map[string]any{"url": "https://alpha.example/", "title": "Alfa projekt", "tagIds": []int64{work.ID, home.ID}})
	e.clock.Advance(time.Minute)
	beta := linkTestCreate(t, c, map[string]any{"url": "https://beta.example/", "title": "Béta", "tagIds": []int64{work.ID}})
	e.clock.Advance(time.Minute)
	gamma := linkTestCreate(t, c, map[string]any{"url": "https://gamma.example/", "description": "Ünnepi recept"})
	e.clock.Advance(time.Minute)
	archived := linkTestCreateURL(t, c, "https://archived.example/")
	expectStatus(t, c.post(linkTestPath(archived.ID, "/archive"), nil), http.StatusOK)
	expectStatus(t, c.post(linkTestPath(beta.ID, "/pin"), nil), http.StatusOK)
	expectStatus(t, c.post(linkTestPath(gamma.ID, "/visit"), nil), http.StatusOK)
	shared := linkTestCreateURL(t, owner, "https://shared.example/")
	expectStatus(t, owner.post(linkTestPath(shared.ID, "/shares"), map[string]any{"targetType": "all"}), http.StatusCreated)

	ids := func(p LinkPage) []int64 {
		out := make([]int64, 0, len(p.Items))
		for _, it := range p.Items {
			out = append(out, it.ID)
		}
		return out
	}
	cases := []struct {
		query        string
		want         []int64
		total        int
		pinned, arch int
	}{
		{"", []int64{beta.ID, gamma.ID, shared.ID, alpha.ID}, 4, 1, 0},
		{"?archived=include", []int64{beta.ID, gamma.ID, shared.ID, archived.ID, alpha.ID}, 5, 1, 1},
		{"?archived=only", []int64{archived.ID}, 1, 0, 1},
		{"?scope=own", []int64{beta.ID, gamma.ID, alpha.ID}, 3, 1, 0},
		{"?scope=shared", []int64{shared.ID}, 1, 0, 0},
		{"?sort=created", []int64{beta.ID, shared.ID, gamma.ID, alpha.ID}, 4, 1, 0},
		{"?sort=title", []int64{beta.ID, alpha.ID, shared.ID, gamma.ID}, 4, 1, 0},
		{"?q=%C3%9CNNEPI", []int64{gamma.ID}, 1, 0, 0},
		{"?q=munka", []int64{beta.ID, alpha.ID}, 2, 1, 0},
		{"?tags=" + strconv.FormatInt(home.ID, 10), []int64{alpha.ID}, 1, 0, 0},
		{fmt.Sprintf("?tags=%d,%d&tagMatch=any", work.ID, home.ID), []int64{beta.ID, alpha.ID}, 2, 1, 0},
		{fmt.Sprintf("?tags=%d,%d&tagMatch=all", work.ID, home.ID), []int64{alpha.ID}, 1, 0, 0},
		{"?tags=99999", []int64{}, 0, 0, 0},
		{"?untagged=true", []int64{gamma.ID, shared.ID}, 2, 0, 0},
		{"?untagged=false&limit=2", []int64{beta.ID, gamma.ID}, 4, 1, 0},
		{"?limit=1&offset=1", []int64{gamma.ID}, 4, 1, 0},
		{"?offset=50", []int64{}, 4, 1, 0},
	}
	for _, tc := range cases {
		t.Run("query "+tc.query, func(t *testing.T) {
			p := decode[LinkPage](t, c.get("/api/links"+tc.query))
			if got := ids(p); !slices.Equal(got, tc.want) {
				t.Errorf("ids = %v, want %v", got, tc.want)
			}
			if p.Total != tc.total || p.Counts.Pinned != tc.pinned || p.Counts.Archived != tc.arch {
				t.Errorf("total %d counts %+v, want %d {%d %d}", p.Total, p.Counts, tc.total, tc.pinned, tc.arch)
			}
		})
	}

	t.Run("shape and derived fields", func(t *testing.T) {
		res := c.get("/api/links?limit=1")
		body := expectStatus(t, res, http.StatusOK)
		if keys := strings.Join(jsonKeys(t, body), ","); keys != "counts,items,limit,offset,total" {
			t.Errorf("keys = %s", keys)
		}
		if res.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("Cache-Control = %q", res.Header.Get("Cache-Control"))
		}
		p := decode[LinkPage](t, c.get("/api/links?scope=shared"))
		item := p.Items[0]
		if item.IsOwner || item.SharedVia == nil || *item.SharedVia != model.ShareAll || item.ShareCount != 0 ||
			item.SharedAt == nil || !item.InactiveSince.Equal(*item.SharedAt) {
			t.Errorf("shared item = %+v", item)
		}
		g := decode[LinkPage](t, c.get("/api/links?q=recept")).Items[0]
		if g.UseCount != 1 || g.LastUsedAt == nil || !g.InactiveSince.Equal(*g.LastUsedAt) ||
			g.ArchiveDueAt == nil || !g.ArchiveDueAt.Equal(g.LastUsedAt.AddDate(0, 0, 90)) {
			t.Errorf("visited item = use %d last %v inactive %v due %v", g.UseCount, g.LastUsedAt, g.InactiveSince, g.ArchiveDueAt)
		}
		p = decode[LinkPage](t, c.get("/api/links?archived=only"))
		if p.Items[0].ArchiveDueAt != nil || p.Items[0].ArchivedAt == nil {
			t.Errorf("archived item due %v archivedAt %v", p.Items[0].ArchiveDueAt, p.Items[0].ArchivedAt)
		}
		owned := decode[LinkPage](t, owner.get("/api/links")).Items[0]
		if !owned.IsOwner || owned.ShareCount != 1 || owned.SharedVia != nil {
			t.Errorf("owner item = %+v", owned)
		}
	})

	t.Run("user archive period", func(t *testing.T) {
		days := 7
		if _, err := e.store.UpdatePreferences(e.ctx(), alpha.Owner.ID, model.PreferencesPatch{
			ArchiveAfterDays: model.NullableInt{Set: true, Value: &days},
		}); err != nil {
			t.Fatal(err)
		}
		item := decode[LinkPage](t, c.get("/api/links?q=alfa")).Items[0]
		if item.ArchiveDueAt == nil || !item.ArchiveDueAt.Equal(item.InactiveSince.AddDate(0, 0, 7)) {
			t.Errorf("due = %v, want inactiveSince + 7 days", item.ArchiveDueAt)
		}
		never := 0
		if _, err := e.store.UpdatePreferences(e.ctx(), alpha.Owner.ID, model.PreferencesPatch{
			ArchiveAfterDays: model.NullableInt{Set: true, Value: &never},
		}); err != nil {
			t.Fatal(err)
		}
		if item := decode[LinkPage](t, c.get("/api/links?q=alfa")).Items[0]; item.ArchiveDueAt != nil {
			t.Errorf("due = %v with archiving off", item.ArchiveDueAt)
		}
	})

	t.Run("invalid parameters", func(t *testing.T) {
		bad := []struct{ query, param string }{
			{"?q=" + strings.Repeat("a", 201), "q"},
			{"?q=a%00b", "q"},
			{"?tags=1,x", "tags"},
			{"?tags=0", "tags"},
			{"?tags=" + strings.TrimSuffix(strings.Repeat("1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,", 1), ","), "tags"},
			{"?tagMatch=some", "tagMatch"},
			{"?untagged=yes", "untagged"},
			{"?untagged=true&tags=1", "untagged"},
			{"?scope=mine", "scope"},
			{"?archived=all", "archived"},
			{"?sort=random", "sort"},
			{"?limit=0", "limit"},
			{"?limit=501", "limit"},
			{"?limit=x", "limit"},
			{"?offset=-1", "offset"},
			{"?offset=1000001", "offset"},
		}
		for _, b := range bad {
			body := expectError(t, c.get("/api/links"+b.query), http.StatusBadRequest, codeInvalidQuery)
			if body.Fields[b.param] != "invalid" || len(body.Fields) != 1 {
				t.Errorf("%s: fields = %v, want %s", b.query, body.Fields, b.param)
			}
		}
		if p := decode[LinkPage](t, c.get("/api/links?limit=500")); p.Limit != 500 {
			t.Errorf("limit = %d", p.Limit)
		}
		if p := decode[LinkPage](t, c.get("/api/links")); p.Limit != 200 || p.Offset != 0 {
			t.Errorf("default limit/offset = %d/%d", p.Limit, p.Offset)
		}
	})
}

func TestLinkGetUpdateDelete(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	d := linkTestCreateURL(t, c, "https://old.example/path")
	icon := linkTestIconBlob("image/png", []byte("\x89PNG old icon"))
	linkTestFetched(t, e, d.ID, "Letöltött cím", icon)

	got := decode[LinkDetail](t, c.get(linkTestPath(d.ID)))
	if got.Title != "Letöltött cím" || got.TitleSource != model.TitleFetched || got.FetchStatus != model.FetchOK ||
		got.IconURL == nil || *got.IconURL != linkTestPath(d.ID, "/icon?v="+icon.SHA256[:8]) || got.FetchedAt == nil {
		t.Fatalf("fetched detail = %+v", got)
	}

	t.Run("description only keeps the fetch state", func(t *testing.T) {
		before := len(e.jobs.enqueued())
		up := decode[LinkDetail](t, c.patch(linkTestPath(d.ID), map[string]any{"description": " Új\nleírás "}))
		if up.Description != "Új\nleírás" || up.FetchStatus != model.FetchOK || up.Title != "Letöltött cím" {
			t.Errorf("patched = %+v", up)
		}
		if len(e.jobs.enqueued()) != before {
			t.Error("a description change enqueued a fetch")
		}
	})

	t.Run("url change resets the fetched metadata", func(t *testing.T) {
		e.clock.Advance(time.Hour)
		up := decode[LinkDetail](t, c.patch(linkTestPath(d.ID), map[string]any{"url": "https://NEW.example/path"}))
		if up.URL != "https://new.example/path" || up.Host != "new.example" || up.FetchStatus != model.FetchPending ||
			up.FetchError != nil || up.FetchAttempts != 0 || up.FetchedAt != nil || up.Title != "" ||
			up.TitleSource != model.TitleNone || up.IconURL != nil {
			t.Errorf("after url change = %+v", up)
		}
		if linkTestCountID(e.jobs.enqueued(), d.ID) != 2 {
			t.Errorf("enqueued = %v, want the link twice (create, url change)", e.jobs.enqueued())
		}
	})

	t.Run("manual title survives an url change", func(t *testing.T) {
		up := decode[LinkDetail](t, c.patch(linkTestPath(d.ID), map[string]any{"title": "Saját cím"}))
		if up.Title != "Saját cím" || up.TitleSource != model.TitleManual {
			t.Errorf("title = %q (%s)", up.Title, up.TitleSource)
		}
		up = decode[LinkDetail](t, c.patch(linkTestPath(d.ID), map[string]any{"url": "https://third.example/"}))
		if up.Title != "Saját cím" || up.TitleSource != model.TitleManual || up.FetchStatus != model.FetchPending {
			t.Errorf("after url change = %q (%s) %s", up.Title, up.TitleSource, up.FetchStatus)
		}
	})

	t.Run("clearing a manual title fetches again", func(t *testing.T) {
		linkTestFetched(t, e, d.ID, "Ignored, the title is manual", nil)
		before := linkTestCountID(e.jobs.enqueued(), d.ID)
		up := decode[LinkDetail](t, c.patch(linkTestPath(d.ID), map[string]any{"title": "   "}))
		if up.Title != "" || up.TitleSource != model.TitleNone || up.FetchStatus != model.FetchPending {
			t.Errorf("after clearing = %q (%s) %s", up.Title, up.TitleSource, up.FetchStatus)
		}
		if linkTestCountID(e.jobs.enqueued(), d.ID) != before+1 {
			t.Error("clearing a manual title did not enqueue a fetch")
		}
	})

	t.Run("validation and conflicts", func(t *testing.T) {
		other := linkTestCreateURL(t, c, "https://other.example/")
		body := expectError(t, c.patch(linkTestPath(d.ID), map[string]any{"url": "https://other.example"}), http.StatusConflict, codeLinkDuplicate)
		if id, _ := body.Details["linkId"].(float64); int64(id) != other.ID {
			t.Errorf("details.linkId = %v, want %d", body.Details["linkId"], other.ID)
		}
		expectFields(t, c.patch(linkTestPath(d.ID), map[string]any{"url": "ftp://x.example/", "title": strings.Repeat("a", 501), "description": "‪"}),
			map[string]string{"url": "not_allowed", "title": "too_long", "description": "invalid"})
		expectFields(t, c.patch(linkTestPath(d.ID), map[string]any{"url": ""}), map[string]string{"url": "required"})
		expectError(t, c.patch(linkTestPath(d.ID), map[string]any{"tagIds": []int64{}}), http.StatusBadRequest, codeInvalidJSON)
		same := decode[LinkDetail](t, c.patch(linkTestPath(d.ID), map[string]any{}))
		if same.ID != d.ID {
			t.Errorf("empty patch answered %+v", same)
		}
	})

	t.Run("invalid ids", func(t *testing.T) {
		for _, p := range []string{"/api/links/abc", "/api/links/0", "/api/links/-1", "/api/links/99999999999999999999"} {
			expectError(t, c.get(p), http.StatusBadRequest, codeInvalidID)
			expectError(t, c.patch(p, map[string]any{}), http.StatusBadRequest, codeInvalidID)
			expectError(t, c.del(p), http.StatusBadRequest, codeInvalidID)
			expectError(t, c.post(p+"/refresh", nil), http.StatusBadRequest, codeInvalidID)
		}
		expectError(t, c.get("/api/links/424242"), http.StatusNotFound, codeNotFound)
	})

	t.Run("delete", func(t *testing.T) {
		res := c.del(linkTestPath(d.ID))
		expectStatus(t, res, http.StatusNoContent)
		if len(readBody(t, res)) != 0 {
			t.Error("204 with a body")
		}
		expectError(t, c.get(linkTestPath(d.ID)), http.StatusNotFound, codeNotFound)
		expectError(t, c.del(linkTestPath(d.ID)), http.StatusNotFound, codeNotFound)
		expectError(t, c.patch(linkTestPath(d.ID), map[string]any{"title": "x"}), http.StatusNotFound, codeNotFound)
	})
}

func TestLinkRefresh(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	d := linkTestCreateURL(t, c, "https://refresh.example/")
	linkTestFetchResult(t, e, d.ID, model.FetchResult{Status: model.FetchFailed, Error: model.FetchErrHTTPStatus})
	failed := decode[LinkDetail](t, c.get(linkTestPath(d.ID)))
	if failed.FetchStatus != model.FetchFailed || failed.FetchError == nil {
		t.Fatalf("failed detail = %+v", failed)
	}
	res := decodeStatus[LinkDetail](t, c.post(linkTestPath(d.ID, "/refresh"), nil), http.StatusAccepted)
	if res.FetchStatus != model.FetchPending || res.FetchError != nil || res.FetchAttempts != 0 {
		t.Errorf("after refresh = %s %v %d", res.FetchStatus, res.FetchError, res.FetchAttempts)
	}
	if linkTestCountID(e.jobs.enqueued(), d.ID) != 2 {
		t.Errorf("enqueued = %v", e.jobs.enqueued())
	}
	// Refreshing a pending link is idempotent.
	again := decodeStatus[LinkDetail](t, c.post(linkTestPath(d.ID, "/refresh"), nil), http.StatusAccepted)
	if again.FetchStatus != model.FetchPending {
		t.Errorf("second refresh = %s", again.FetchStatus)
	}
	expectError(t, c.post(linkTestPath(424242, "/refresh"), nil), http.StatusNotFound, codeNotFound)
}

func TestLinkStoreFailures(t *testing.T) {
	t.Parallel()
	e, faults := linkTestFaultEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	tag := linkTestCreateTag(t, c, "Hiba")
	d := linkTestCreate(t, c, map[string]any{"url": "https://fail.example/", "tagIds": []int64{tag.ID}})
	linkTestFetched(t, e, d.ID, "", linkTestIconBlob("image/png", []byte("png")))
	share := decodeStatus[model.Share](t, c.post(linkTestPath(d.ID, "/shares"), map[string]any{"targetType": "all"}), http.StatusCreated)
	boom := errors.New("injected store failure")

	type call struct {
		method, path string
		body         any
	}
	cases := []struct {
		fail  string
		calls []call
	}{
		{"Preferences", []call{
			{"GET", "/api/links", nil},
			{"POST", "/api/links", map[string]any{"url": "https://p1.example/"}},
			{"POST", "/api/links/quick", map[string]any{"text": "https://p2.example/"}},
			{"GET", linkTestPath(d.ID), nil},
			{"POST", linkTestPath(d.ID, "/pin"), nil},
			{"POST", linkTestPath(d.ID, "/visit"), nil},
		}},
		{"Links", []call{{"GET", "/api/links", nil}}},
		{"LinkItem", []call{{"DELETE", linkTestPath(d.ID, "/archive"), nil}, {"PUT", linkTestPath(d.ID, "/tags"), map[string]any{"tagIds": []int64{}}}}},
		{"Link", []call{
			{"POST", linkTestPath(d.ID, "/refresh"), nil},
			{"POST", "/api/links/quick", map[string]any{"text": "https://fail.example/"}},
		}},
		{"CreateLink", []call{{"POST", "/api/links", map[string]any{"url": "https://c.example/"}}, {"POST", "/api/links/quick", map[string]any{"text": "https://c.example/"}}}},
		{"RecordVisit", []call{{"POST", linkTestPath(d.ID, "/visit"), nil}}},
		{"SetLinkTags", []call{{"PUT", linkTestPath(d.ID, "/tags"), map[string]any{"tagIds": []int64{}}}}},
		{"LinkIcon", []call{{"GET", linkTestPath(d.ID, "/icon"), nil}}},
		{"Shares", []call{{"GET", linkTestPath(d.ID, "/shares"), nil}, {"DELETE", linkTestPath(d.ID, "/shares/", strconv.FormatInt(share.ID, 10)), nil}}},
		{"DeleteShare", []call{{"DELETE", linkTestPath(d.ID, "/shares/", strconv.FormatInt(share.ID, 10)), nil}}},
		{"Tags", []call{{"GET", "/api/tags", nil}}},
		{"CreateTag", []call{{"POST", "/api/tags", map[string]any{"name": "Új"}}}},
		{"ExportLinks", []call{{"GET", "/api/links/export?format=json", nil}}},
	}
	for _, tc := range cases {
		t.Run(tc.fail, func(t *testing.T) {
			faults.set(tc.fail, boom)
			defer faults.set(tc.fail, nil)
			for _, cl := range tc.calls {
				expectError(t, c.do(cl.method, cl.path, cl.body), http.StatusInternalServerError, codeInternalError)
			}
		})
	}
	if n := e.logs.count("injected store failure"); n == 0 {
		t.Error("failures were not logged")
	}
	// The listing after the faults still works.
	expectStatus(t, c.get("/api/links"), http.StatusOK)
}

func TestLinkIDOR(t *testing.T) {
	t.Parallel()
	for _, role := range model.Roles {
		t.Run(string(role), func(t *testing.T) {
			t.Parallel()
			linkTestIDOR(t, role)
		})
	}
}

// linkTestIDOR runs the 9.2.3 group 2 matrix with B in the given role: A
// (developer) owns a link, a tag and a share; B calls every endpoint that
// takes one of these ids and always gets 404 (validation_failed unknown
// for tag ids in bodies); A's data stays unchanged.
func linkTestIDOR(t *testing.T, role model.Role) {
	e := newEnv(t)
	a, aUser := linkTestLoginAs(t, e, model.RoleDeveloper)
	recipient := e.seedUser(t, model.RoleCustomer)
	aTag := linkTestCreateTag(t, a, "A címkéje")
	aLink := linkTestCreate(t, a, map[string]any{"url": "https://a.example/secret?token=1", "title": "A titka", "tagIds": []int64{aTag.ID}})
	linkTestFetched(t, e, aLink.ID, "", linkTestIconBlob("image/png", []byte("a icon")))
	aShare := decodeStatus[model.Share](t, a.post(linkTestPath(aLink.ID, "/shares"),
		map[string]any{"targetType": "user", "targetUserId": recipient.ID}), http.StatusCreated)
	_, aSession := e.createSession(t, aUser.ID)
	enqueuedBefore := len(e.jobs.enqueued())

	snapshot := func() string {
		return string(readBody(t, a.get(linkTestPath(aLink.ID)))) + string(readBody(t, a.get("/api/tags"))) +
			string(readBody(t, a.get(linkTestPath(aLink.ID, "/shares"))))
	}
	before := snapshot()

	b, bUser := linkTestLoginAs(t, e, role)
	bTag := linkTestCreateTag(t, b, "B címkéje")
	bLink := linkTestCreateURL(t, b, "https://b.example/")
	enqueuedBefore++

	p := linkTestPath(aLink.ID)
	sharePath := linkTestPath(aLink.ID, "/shares/", strconv.FormatInt(aShare.ID, 10))
	calls := []struct {
		method, path string
		body         any
		shares       bool
	}{
		{"GET", p, nil, false},
		{"HEAD", p, nil, false},
		{"PATCH", p, map[string]any{"title": "B was here", "url": "https://evil.example/"}, false},
		{"DELETE", p, nil, false},
		{"PUT", p + "/tags", map[string]any{"tagIds": []int64{bTag.ID}}, false},
		{"PUT", p + "/tags", map[string]any{"tagIds": []int64{}}, false},
		{"POST", p + "/pin", nil, false},
		{"DELETE", p + "/pin", nil, false},
		{"POST", p + "/archive", nil, false},
		{"DELETE", p + "/archive", nil, false},
		{"POST", p + "/visit", nil, false},
		{"POST", p + "/refresh", nil, false},
		{"GET", p + "/icon", nil, false},
		{"GET", p + "/icon?v=deadbeef", nil, false},
		{"GET", p + "/shares", nil, true},
		{"POST", p + "/shares", map[string]any{"targetType": "all"}, true},
		{"DELETE", sharePath, nil, true},
	}
	for _, cl := range calls {
		res := b.do(cl.method, cl.path, cl.body)
		switch {
		case cl.shares && !role.CanShare():
			expectError(t, res, http.StatusForbidden, codeShareNotAllowed)
		case cl.method == "HEAD":
			expectStatus(t, res, http.StatusNotFound)
		default:
			body := expectError(t, res, http.StatusNotFound, codeNotFound)
			if strings.Contains(fmt.Sprint(body), "a.example") {
				t.Errorf("%s %s leaks the URL: %+v", cl.method, cl.path, body)
			}
		}
	}

	// A's tag: by path and inside tagIds bodies.
	tagPath := "/api/tags/" + strconv.FormatInt(aTag.ID, 10)
	expectError(t, b.patch(tagPath, map[string]any{"name": "Átnevezve", "color": "rose"}), http.StatusNotFound, codeNotFound)
	expectError(t, b.del(tagPath), http.StatusNotFound, codeNotFound)
	expectFields(t, b.post("/api/links", map[string]any{"url": "https://c.example/", "tagIds": []int64{aTag.ID}}),
		map[string]string{"tagIds": "unknown"})
	expectFields(t, b.put(linkTestPath(bLink.ID, "/tags"), map[string]any{"tagIds": []int64{bTag.ID, aTag.ID}}),
		map[string]string{"tagIds": "unknown"})
	if got := decode[LinkDetail](t, b.get(linkTestPath(bLink.ID))); len(got.TagIDs) != 0 {
		t.Errorf("B's link got tags %v from a rejected request", got.TagIDs)
	}

	// A's session.
	expectError(t, b.del("/api/me/sessions/"+strconv.FormatInt(aSession.ID, 10)), http.StatusNotFound, codeNotFound)
	if sessions, err := e.store.Sessions(e.ctx(), aUser.ID); err != nil || !slices.ContainsFunc(sessions, func(s model.Session) bool {
		return s.ID == aSession.ID
	}) {
		t.Errorf("A's session %d is gone (%v)", aSession.ID, err)
	}

	// Nested ids: B's own link path with A's share id, and A's own link path
	// with B's share id. Both 404; neither share changes.
	if role.CanShare() {
		bShare := decodeStatus[model.Share](t, b.post(linkTestPath(bLink.ID, "/shares"), map[string]any{"targetType": "all"}), http.StatusCreated)
		expectError(t, b.del(linkTestPath(bLink.ID, "/shares/", strconv.FormatInt(aShare.ID, 10))), http.StatusNotFound, codeNotFound)
		expectError(t, a.del(linkTestPath(aLink.ID, "/shares/", strconv.FormatInt(bShare.ID, 10))), http.StatusNotFound, codeNotFound)
		shares := decode[listResponse[model.Share]](t, b.get(linkTestPath(bLink.ID, "/shares"))).Items
		if len(shares) != 1 || shares[0].ID != bShare.ID {
			t.Errorf("B's shares = %+v", shares)
		}
	}

	if after := snapshot(); after != before {
		t.Errorf("A's data changed:\nbefore %s\nafter  %s", before, after)
	}
	if got := e.jobs.enqueued(); len(got) != enqueuedBefore {
		t.Errorf("B's requests enqueued fetches: %v", got)
	}
	if _, err := e.store.LinkItem(e.ctx(), bUser.Viewer(), aLink.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("store sees A's link for B: %v", err)
	}
	if rows, _, err := e.store.Audit(e.ctx(), model.AuditFilter{Event: model.EventLinkUnshared}, model.PageReq{Limit: 10}); err != nil || len(rows) != 0 {
		t.Errorf("unshare audit rows = %v (%v)", rows, err)
	}
}

func TestLinkIDORSharedReadOnly(t *testing.T) {
	t.Parallel()
	for _, role := range model.Roles {
		t.Run(string(role), func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			a, _ := linkTestLoginAs(t, e, model.RoleAdmin)
			aTag := linkTestCreateTag(t, a, "A")
			aLink := linkTestCreate(t, a, map[string]any{"url": "https://shared.example/", "title": "Közös", "tagIds": []int64{aTag.ID}})
			icon := linkTestIconBlob("image/gif", []byte("GIF89a"))
			linkTestFetched(t, e, aLink.ID, "", icon)
			b, _ := linkTestLoginAs(t, e, role)
			aShare := decodeStatus[model.Share](t, a.post(linkTestPath(aLink.ID, "/shares"),
				map[string]any{"targetType": "role", "targetRole": string(role)}), http.StatusCreated)
			before := string(readBody(t, a.get(linkTestPath(aLink.ID))))
			enqueued := len(e.jobs.enqueued())
			p := linkTestPath(aLink.ID)

			got := decode[LinkDetail](t, b.get(p))
			if got.IsOwner || got.SharedVia == nil || *got.SharedVia != model.ShareRole || len(got.Shares) != 0 ||
				len(got.TagIDs) != 0 || got.ShareCount != 0 {
				t.Errorf("recipient view = %+v", got)
			}
			expectError(t, b.patch(p, map[string]any{"title": "B"}), http.StatusForbidden, codeNotOwner)
			expectError(t, b.del(p), http.StatusForbidden, codeNotOwner)
			expectError(t, b.post(p+"/refresh", nil), http.StatusForbidden, codeNotOwner)
			shareCode := codeShareNotAllowed
			if role.CanShare() {
				shareCode = codeNotOwner
			}
			expectError(t, b.get(p+"/shares"), http.StatusForbidden, shareCode)
			expectError(t, b.post(p+"/shares", map[string]any{"targetType": "all"}), http.StatusForbidden, shareCode)
			expectError(t, b.del(p+"/shares/"+strconv.FormatInt(aShare.ID, 10)), http.StatusForbidden, shareCode)

			// Personal state is allowed and stays personal.
			bTag := linkTestCreateTag(t, b, "B")
			item := decode[LinkItem](t, b.put(p+"/tags", map[string]any{"tagIds": []int64{bTag.ID}}))
			if !slices.Equal(item.TagIDs, []int64{bTag.ID}) {
				t.Errorf("recipient tags = %v", item.TagIDs)
			}
			expectFields(t, b.put(p+"/tags", map[string]any{"tagIds": []int64{aTag.ID}}), map[string]string{"tagIds": "unknown"})
			if item := decode[LinkItem](t, b.post(p+"/pin", nil)); !item.Pinned {
				t.Error("recipient pin failed")
			}
			if st := decode[LinkState](t, b.post(p+"/visit", nil)); st.UseCount != 1 {
				t.Errorf("recipient use count = %d", st.UseCount)
			}
			res := b.get(p + "/icon")
			expectStatus(t, res, http.StatusOK)
			if string(readBody(t, res)) != "GIF89a" {
				t.Error("recipient icon body differs")
			}

			if after := string(readBody(t, a.get(linkTestPath(aLink.ID)))); after != before {
				t.Errorf("owner's view changed:\nbefore %s\nafter  %s", before, after)
			}
			if len(e.jobs.enqueued()) != enqueued {
				t.Error("recipient enqueued a fetch")
			}
		})
	}
}
