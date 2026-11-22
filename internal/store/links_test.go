package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestCreateLinkAndTitleSource(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	dev := seedUser(t, st, model.RoleDeveloper)
	other := seedUser(t, st, model.RoleDeveloper)
	tag := seedTag(t, st, dev, "Go")
	foreign := seedTag(t, st, other, "Foreign")

	d, err := st.CreateLink(ctx, model.NewLink{OwnerID: dev.ID, URL: "https://go.dev/doc/", Host: "go.dev",
		Title: "Go docs", Description: "Leírás", TagIDs: []int64{tag.ID, tag.ID}, Pinned: true})
	mustNoErr(t, err)
	if d.TitleSource != model.TitleManual || d.FetchStatus != model.FetchPending || d.FetchError != nil ||
		d.FetchAttempts != 0 || d.FetchedAt != nil || !d.IsOwner || !d.Pinned || d.Owner.ID != dev.ID ||
		len(d.TagIDs) != 1 || len(d.Tags) != 1 || d.Tags[0].Name != "Go" || d.Tags[0].LinkCount != 1 ||
		d.SharedVia != nil || d.ShareCount != 0 || len(d.Shares) != 0 || d.IconSHA256 != "" {
		t.Fatalf("detail = %+v", d)
	}
	if !d.BaselineAt.Equal(clk.Now()) || !d.CreatedAt.Equal(clk.Now()) {
		t.Errorf("baseline %v created %v", d.BaselineAt, d.CreatedAt)
	}
	var next, search, label string
	mustNoErr(t, st.db.QueryRow(`SELECT next_fetch_at, search_text, sort_label FROM links WHERE id = ?`, d.ID).
		Scan(&next, &search, &label))
	if next != formatTime(clk.Now()) || search != "https://go.dev/doc/\ngo.dev\ngo docs\nleiras" ||
		label != "leiras" {
		t.Errorf("next %q search %q label %q", next, search, label)
	}

	noTitle, err := st.CreateLink(ctx, model.NewLink{OwnerID: dev.ID, URL: "https://example.org/", Host: "example.org"})
	mustNoErr(t, err)
	if noTitle.TitleSource != model.TitleNone || noTitle.Pinned || noTitle.Tags == nil {
		t.Errorf("untitled = %+v", noTitle)
	}

	_, err = st.CreateLink(ctx, model.NewLink{OwnerID: dev.ID, URL: "https://go.dev/doc/", Host: "go.dev"})
	wantDuplicate(t, err, model.DuplicateLink, d.ID)
	// The same URL for another owner is fine.
	_, err = st.CreateLink(ctx, model.NewLink{OwnerID: other.ID, URL: "https://go.dev/doc/", Host: "go.dev"})
	mustNoErr(t, err)

	_, err = st.CreateLink(ctx, model.NewLink{OwnerID: dev.ID, URL: "https://a.example/", Host: "a.example",
		TagIDs: []int64{foreign.ID}})
	wantErr(t, err, model.ErrConflict)
	_, err = st.CreateLink(ctx, model.NewLink{OwnerID: dev.ID, URL: "https://b.example/", Host: "b.example",
		TagIDs: []int64{424242}})
	wantErr(t, err, model.ErrConflict)
	many := make([]int64, 0, 21)
	for i := 1; i <= 21; i++ {
		many = append(many, int64(i))
	}
	_, err = st.CreateLink(ctx, model.NewLink{OwnerID: dev.ID, URL: "https://c.example/", Host: "c.example",
		TagIDs: many})
	wantErr(t, err, model.ErrConflict)
	_, err = st.CreateLink(ctx, model.NewLink{OwnerID: 9999, URL: "https://d.example/", Host: "d.example"})
	wantErr(t, err, model.ErrNotFound)
	_, err = st.CreateLink(ctx, model.NewLink{OwnerID: dev.ID, URL: "http://x", Host: "x"})
	if err == nil {
		t.Error("url length CHECK not enforced")
	}
}

func TestCreateLinkLimit(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	now := formatTime(clk.Now())
	// Fill up to the limit with direct inserts (fast), then hit the limit.
	tx, err := st.db.Begin()
	mustNoErr(t, err)
	for i := 0; i < model.MaxLinksPerUser; i++ {
		_, err := tx.Exec(`INSERT INTO links (owner_id, url, host, created_at, updated_at)
			VALUES (?, ?, 'example.com', ?, ?)`, u.ID, fmt.Sprintf("https://example.com/%d", i), now, now)
		mustNoErr(t, err)
	}
	mustNoErr(t, tx.Commit())
	_, err = st.CreateLink(ctx, model.NewLink{OwnerID: u.ID, URL: "https://example.com/new", Host: "example.com"})
	wantErr(t, err, model.ErrLimitReached)
	// A duplicate is reported as such even at the limit.
	_, err = st.CreateLink(ctx, model.NewLink{OwnerID: u.ID, URL: "https://example.com/1", Host: "example.com"})
	wantDuplicate(t, err, model.DuplicateLink, 2)

	// Import beyond the limit creates nothing.
	_, _, err = st.ImportLinks(ctx, u.ID, []model.ImportEntry{{URL: "https://example.com/import", Host: "example.com"}})
	wantErr(t, err, model.ErrLimitReached)
	if n := countRows(t, st, `SELECT COUNT(*) FROM links WHERE owner_id = ?`, u.ID); n != model.MaxLinksPerUser {
		t.Errorf("links = %d", n)
	}
}

func TestUpdateLinkRules(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	dev := seedUser(t, st, model.RoleDeveloper)
	cust := seedUser(t, st, model.RoleCustomer)
	v := dev.Viewer()

	// A fetched title and an icon, then a URL change drops both.
	d := seedLink(t, st, dev, "")
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: d.ID, URL: d.URL, Status: model.FetchOK,
		Title: "Fetched", Icon: &model.IconBlob{Data: []byte("png"), ContentType: "image/png"}}))
	d, _ = st.Link(ctx, v, d.ID)
	if d.TitleSource != model.TitleFetched || d.Title != "Fetched" || d.IconSHA256 == "" || d.FetchStatus != model.FetchOK {
		t.Fatalf("after fetch = %+v", d)
	}
	clk.Advance(time.Minute)
	newURL := "https://other.example/x"
	u, err := st.UpdateLink(ctx, v, d.ID, model.LinkPatch{URL: &newURL, Host: ptr("other.example")})
	mustNoErr(t, err)
	if u.URL != newURL || u.Host != "other.example" || u.Title != "" || u.TitleSource != model.TitleNone ||
		u.IconSHA256 != "" || u.FetchStatus != model.FetchPending || u.FetchError != nil ||
		u.FetchAttempts != 0 || u.FetchedAt != nil || !u.UpdatedAt.Equal(clk.Now()) {
		t.Fatalf("after url change = %+v", u)
	}
	due, _ := st.DueFetches(ctx, 10)
	if len(due) != 1 || due[0] != d.ID {
		t.Errorf("due after url change = %v", due)
	}

	// A manual title survives a URL change.
	m := seedLink(t, st, dev, "Manual")
	u, err = st.UpdateLink(ctx, v, m.ID, model.LinkPatch{URL: ptr("https://manual.example/"), Host: ptr("manual.example")})
	mustNoErr(t, err)
	if u.Title != "Manual" || u.TitleSource != model.TitleManual {
		t.Errorf("manual after url change = %+v", u)
	}
	// Emptying a manual title marks the link for fetching.
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: m.ID, URL: u.URL, Status: model.FetchOK, Title: "Site"}))
	u, err = st.UpdateLink(ctx, v, m.ID, model.LinkPatch{Title: ptr("")})
	mustNoErr(t, err)
	if u.Title != "" || u.TitleSource != model.TitleNone || u.FetchStatus != model.FetchPending {
		t.Errorf("emptied manual title = %+v", u)
	}
	// Setting a title and a description; same URL is not a change.
	u, err = st.UpdateLink(ctx, v, m.ID, model.LinkPatch{URL: ptr(u.URL), Title: ptr("New"), Description: ptr("Desc")})
	mustNoErr(t, err)
	if u.Title != "New" || u.TitleSource != model.TitleManual || u.Description != "Desc" {
		t.Errorf("title+description = %+v", u)
	}
	// Emptying a fetched title does not refetch.
	f := seedLink(t, st, dev, "")
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: f.ID, URL: f.URL, Status: model.FetchOK, Title: "T"}))
	u, _ = st.UpdateLink(ctx, v, f.ID, model.LinkPatch{Title: ptr("")})
	if u.FetchStatus != model.FetchOK || u.TitleSource != model.TitleNone {
		t.Errorf("emptied fetched title = %+v", u)
	}

	// Duplicate URL.
	_, err = st.UpdateLink(ctx, v, f.ID, model.LinkPatch{URL: ptr(newURL), Host: ptr("other.example")})
	wantDuplicate(t, err, model.DuplicateLink, d.ID)

	// Ownership: visible foreign link is ErrNotOwner, invisible is ErrNotFound.
	_, err = st.UpdateLink(ctx, cust.Viewer(), f.ID, model.LinkPatch{Title: ptr("x")})
	wantErr(t, err, model.ErrNotFound)
	share(t, st, dev, f.ID, shareAll())
	_, err = st.UpdateLink(ctx, cust.Viewer(), f.ID, model.LinkPatch{Title: ptr("x")})
	wantErr(t, err, model.ErrNotOwner)
	_, err = st.UpdateLink(ctx, v, 9999, model.LinkPatch{Title: ptr("x")})
	wantErr(t, err, model.ErrNotFound)

	// Delete: owner only, cascades.
	wantErr(t, st.DeleteLink(ctx, cust.Viewer(), f.ID), model.ErrNotOwner)
	wantErr(t, st.DeleteLink(ctx, cust.Viewer(), m.ID), model.ErrNotFound)
	mustNoErr(t, st.SetPinned(ctx, cust.Viewer(), f.ID, true))
	mustNoErr(t, st.DeleteLink(ctx, v, f.ID))
	_, err = st.Link(ctx, v, f.ID)
	wantErr(t, err, model.ErrNotFound)
	for _, q := range []string{
		`SELECT COUNT(*) FROM link_shares WHERE link_id = ?`,
		`SELECT COUNT(*) FROM link_user_state WHERE link_id = ?`,
		`SELECT COUNT(*) FROM link_tags WHERE link_id = ?`,
	} {
		if n := countRows(t, st, q, f.ID); n != 0 {
			t.Errorf("%s = %d after delete", q, n)
		}
	}
	wantErr(t, st.DeleteLink(ctx, v, f.ID), model.ErrNotFound)
}

// TestVisibilityMatrix covers 4.7: owner, all, role for each role, user,
// demoted owner, inactive owner and a deleted share.
func TestVisibilityMatrix(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	owner := seedUser(t, st, model.RoleDeveloper)
	admin := seedUser(t, st, model.RoleAdmin)
	dev := seedUser(t, st, model.RoleDeveloper)
	cust := seedUser(t, st, model.RoleCustomer)
	cust2 := seedUser(t, st, model.RoleCustomer)

	private := seedLink(t, st, owner, "private")
	toAll := seedLink(t, st, owner, "all")
	toCustomers := seedLink(t, st, owner, "customers")
	toAdmins := seedLink(t, st, owner, "admins")
	toCust2 := seedLink(t, st, owner, "cust2")
	share(t, st, owner, toAll.ID, shareAll())
	share(t, st, owner, toCustomers.ID, shareRole(model.RoleCustomer))
	share(t, st, owner, toAdmins.ID, shareRole(model.RoleAdmin))
	personal := share(t, st, owner, toCust2.ID, shareUser(cust2.ID))
	// A link shared both to all and personally: the personal route wins.
	share(t, st, owner, toAll.ID, shareUser(cust2.ID))

	visible := func(u model.User) map[string]bool {
		page, err := st.Links(ctx, u.Viewer(), model.LinkQuery{Archived: model.ArchivedInclude, Limit: 100})
		mustNoErr(t, err)
		out := map[string]bool{}
		for _, it := range page.Items {
			out[it.Title] = true
		}
		return out
	}
	check := func(label string, u model.User, want ...string) {
		t.Helper()
		got := visible(u)
		if len(got) != len(want) {
			t.Errorf("%s sees %v, want %v", label, got, want)
			return
		}
		for _, w := range want {
			if !got[w] {
				t.Errorf("%s sees %v, want %v", label, got, want)
				return
			}
		}
	}
	check("owner", owner, "private", "all", "customers", "admins", "cust2")
	check("admin", admin, "all", "admins")
	check("developer", dev, "all")
	check("customer", cust, "all", "customers")
	check("customer2", cust2, "all", "customers", "cust2")

	it, err := st.LinkItem(ctx, cust2.Viewer(), toAll.ID)
	mustNoErr(t, err)
	if it.SharedVia == nil || *it.SharedVia != model.ShareUser || it.SharedAt == nil || it.IsOwner || it.ShareCount != 0 {
		t.Errorf("recipient item = %+v", it)
	}
	ownerItem, _ := st.LinkItem(ctx, owner.Viewer(), toAll.ID)
	if ownerItem.ShareCount != 2 || ownerItem.SharedVia != nil || !ownerItem.IsOwner {
		t.Errorf("owner item = %+v", ownerItem)
	}
	_, err = st.LinkItem(ctx, dev.Viewer(), private.ID)
	wantErr(t, err, model.ErrNotFound)
	_, err = st.Link(ctx, dev.Viewer(), private.ID)
	wantErr(t, err, model.ErrNotFound)
	recipientDetail, err := st.Link(ctx, cust2.Viewer(), toCust2.ID)
	mustNoErr(t, err)
	if len(recipientDetail.Shares) != 0 {
		t.Errorf("recipient sees shares: %+v", recipientDetail.Shares)
	}
	ownerDetail, _ := st.Link(ctx, owner.Viewer(), toCust2.ID)
	if len(ownerDetail.Shares) != 1 || ownerDetail.Shares[0].TargetUser == nil ||
		ownerDetail.Shares[0].TargetUser.Username != cust2.Username {
		t.Errorf("owner shares = %+v", ownerDetail.Shares)
	}

	// Inactive owner: shares stay effective.
	_, err = st.UpdateUser(ctx, owner.ID, model.UserPatch{Active: ptr(false)})
	mustNoErr(t, err)
	check("customer, inactive owner", cust, "all", "customers")
	_, err = st.UpdateUser(ctx, owner.ID, model.UserPatch{Active: ptr(true)})
	mustNoErr(t, err)

	// Demoted owner: shares lose effect without data loss.
	_, err = st.UpdateUser(ctx, owner.ID, model.UserPatch{Role: ptr(model.RoleCustomer)})
	mustNoErr(t, err)
	check("customer, demoted owner", cust)
	check("admin, demoted owner", admin)
	// The demoted owner still sees own links, but not as a sharer.
	d, _ := st.Link(ctx, model.Viewer{UserID: owner.ID, Role: model.RoleCustomer}, toCust2.ID)
	if len(d.Shares) != 0 {
		t.Errorf("demoted owner detail lists shares: %+v", d.Shares)
	}
	_, err = st.UpdateUser(ctx, owner.ID, model.UserPatch{Role: ptr(model.RoleDeveloper)})
	mustNoErr(t, err)
	check("customer2, restored owner", cust2, "all", "customers", "cust2")

	// Deleted share.
	mustNoErr(t, st.DeleteShare(ctx, owner.Viewer(), toCust2.ID, personal.ID))
	check("customer2, share deleted", cust2, "all", "customers")
}

func TestLinkListOrderFiltersAndPaging(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleDeveloper)
	other := seedUser(t, st, model.RoleDeveloper)
	v := u.Viewer()

	mk := func(title, description string) int64 {
		clk.Advance(time.Minute)
		d, err := st.CreateLink(ctx, model.NewLink{OwnerID: u.ID,
			URL: fmt.Sprintf("https://site.example/%s", title), Host: "site.example",
			Title: title, Description: description})
		mustNoErr(t, err)
		return d.ID
	}
	a := mk("Alma", "")       // used twice, the last earlier
	b := mk("Barack", "")     // used twice, the last later
	c := mk("Citrom", "")     // never used
	d := mk("Dinnye", "Édes") // pinned, used once
	e := mk("Eper", "")       // pinned, used three times
	f := mk("Füge", "")       // archived

	visit := func(id int64) {
		clk.Advance(time.Minute)
		_, err := st.RecordVisit(ctx, v, id)
		mustNoErr(t, err)
	}
	visit(a)
	visit(b)
	visit(a)
	visit(b)
	visit(d)
	visit(e)
	visit(e)
	visit(e)
	mustNoErr(t, st.SetPinned(ctx, v, d, true))
	mustNoErr(t, st.SetPinned(ctx, v, e, true))
	mustNoErr(t, st.SetArchived(ctx, v, f, true))

	// A link shared to u by someone else.
	sharedLink := seedLink(t, st, other, "Shared gránát")
	share(t, st, other, sharedLink.ID, shareUser(u.ID))

	ids := func(q model.LinkQuery) []int64 {
		t.Helper()
		if q.Limit == 0 {
			q.Limit = 100
		}
		page, err := st.Links(ctx, v, q)
		mustNoErr(t, err)
		out := []int64{}
		for _, it := range page.Items {
			out = append(out, it.ID)
		}
		return out
	}
	eq := func(label string, got []int64, want ...int64) {
		t.Helper()
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s = %v, want %v", label, got, want)
		}
	}
	// usage: pinned first (more uses first), then use count, then last use,
	// then never used by creation time (newest first).
	eq("usage", ids(model.LinkQuery{}), e, d, b, a, sharedLink.ID, c)
	eq("created", ids(model.LinkQuery{Sort: model.SortCreated}), e, d, sharedLink.ID, c, b, a)
	eq("title", ids(model.LinkQuery{Sort: model.SortTitle}), d, e, a, b, c, sharedLink.ID) // "Édes" folds to "edes", before "eper"
	eq("archived only", ids(model.LinkQuery{Archived: model.ArchivedOnly}), f)
	eq("include archived", ids(model.LinkQuery{Archived: model.ArchivedInclude, Sort: model.SortCreated}),
		e, d, sharedLink.ID, f, c, b, a)
	eq("own", ids(model.LinkQuery{Scope: model.ScopeOwn, Sort: model.SortCreated}), e, d, c, b, a)
	eq("shared", ids(model.LinkQuery{Scope: model.ScopeShared}), sharedLink.ID)
	eq("q accented, case folded", ids(model.LinkQuery{Q: "édes"}), d)
	eq("q url", ids(model.LinkQuery{Q: "citrom"}), c)
	eq("q shared title", ids(model.LinkQuery{Q: "gránát"}), sharedLink.ID)

	// Tag filters, with u's own tags (also on the shared link).
	red := seedTag(t, st, u, "Piros")
	sweet := seedTag(t, st, u, "Édes gyümölcs")
	foreignTag := seedTag(t, st, other, "Other")
	mustNoErr(t, st.SetLinkTags(ctx, v, a, []int64{red.ID, sweet.ID}))
	mustNoErr(t, st.SetLinkTags(ctx, v, b, []int64{red.ID}))
	mustNoErr(t, st.SetLinkTags(ctx, v, sharedLink.ID, []int64{sweet.ID}))
	eq("tag any", ids(model.LinkQuery{TagIDs: []int64{red.ID, sweet.ID}, TagMatch: model.TagMatchAny}),
		b, a, sharedLink.ID)
	eq("tag all", ids(model.LinkQuery{TagIDs: []int64{red.ID, sweet.ID}, TagMatch: model.TagMatchAll}), a)
	eq("foreign tag id", ids(model.LinkQuery{TagIDs: []int64{foreignTag.ID}}))
	eq("untagged", ids(model.LinkQuery{Untagged: true}), e, d, c)
	eq("q tag name", ids(model.LinkQuery{Q: "gyümölcs"}), a, sharedLink.ID)

	// Paging, total and counts, also past the last page.
	page, err := st.Links(ctx, v, model.LinkQuery{Archived: model.ArchivedInclude, Limit: 2, Offset: 2})
	mustNoErr(t, err)
	if page.Total != 7 || len(page.Items) != 2 || page.Counts.Pinned != 2 || page.Counts.Archived != 1 ||
		page.Limit != 2 || page.Offset != 2 {
		t.Errorf("page = total %d items %d counts %+v", page.Total, len(page.Items), page.Counts)
	}
	page, err = st.Links(ctx, v, model.LinkQuery{Archived: model.ArchivedInclude, Limit: 2, Offset: 50})
	mustNoErr(t, err)
	if page.Total != 7 || len(page.Items) != 0 || page.Items == nil || page.Counts.Pinned != 2 || page.Counts.Archived != 1 {
		t.Errorf("past last page = %+v", page)
	}
	page, _ = st.Links(ctx, v, model.LinkQuery{Q: "nothing-matches", Limit: 10})
	if page.Total != 0 || page.Counts.Pinned != 0 || len(page.Items) != 0 {
		t.Errorf("empty result = %+v", page)
	}

	// Raw fields of the list rows.
	page, _ = st.Links(ctx, v, model.LinkQuery{Q: "alma", Limit: 10})
	it := page.Items[0]
	if it.UseCount != 2 || it.LastUsedAt == nil || it.BaselineAt.IsZero() || fmt.Sprint(it.TagIDs) !=
		fmt.Sprint([]int64{red.ID, sweet.ID}) {
		t.Errorf("alma item = %+v", it)
	}
	page, _ = st.Links(ctx, v, model.LinkQuery{Scope: model.ScopeShared, Limit: 10})
	if it := page.Items[0]; it.SharedAt == nil || !it.BaselineAt.Equal(*it.SharedAt) || it.UseCount != 0 {
		t.Errorf("shared item baseline = %+v", it)
	}
}

// TestSearchAndNameOrderIgnoreAccents is the regression test of the
// accent-insensitive keys: a search without accents finds accented text (also
// in tag names), and an accented label sorts next to its base letter instead
// of after z.
func TestSearchAndNameOrderIgnoreAccents(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	v := u.Viewer()
	mk := func(path, title, description string) int64 {
		clk.Advance(time.Minute)
		d, err := st.CreateLink(ctx, model.NewLink{OwnerID: u.ID,
			URL: "https://accent.example/" + path, Host: "accent.example",
			Title: title, Description: description})
		mustNoErr(t, err)
		return d.ID
	}
	zebra := mk("z", "Zebra", "")
	flood := mk("a", "", "Árvíztűrő tükörfúrógép")
	apple := mk("b", "alma", "")
	ox := mk("o", "Ökör", "")
	oak := mk("t", "Tölgy", "")
	ids := func(q model.LinkQuery) string {
		t.Helper()
		q.Limit = 100
		page, err := st.Links(ctx, v, q)
		mustNoErr(t, err)
		out := []int64{}
		for _, it := range page.Items {
			out = append(out, it.ID)
		}
		return fmt.Sprint(out)
	}
	want := func(id ...int64) string { return fmt.Sprint(id) }
	if got := ids(model.LinkQuery{Sort: model.SortTitle}); got != want(apple, flood, ox, oak, zebra) {
		t.Errorf("name order = %s, want alma, Árvíz…, Ökör, Tölgy, Zebra", got)
	}
	for _, q := range []string{"arviz", "ÁRVÍZ", "arvizturo", "Árvíztűrő"} {
		if got := ids(model.LinkQuery{Q: q}); got != want(flood) {
			t.Errorf("q %q = %s, want the flood link", q, got)
		}
	}
	if got := ids(model.LinkQuery{Q: "tukorfurogep"}); got != want(flood) {
		t.Errorf("q without accents = %s", got)
	}
	if got := ids(model.LinkQuery{Q: "okor"}); got != want(ox) {
		t.Errorf("q okor = %s", got)
	}
	tag := seedTag(t, st, u, "Építés")
	mustNoErr(t, st.SetLinkTags(ctx, v, zebra, []int64{tag.ID}))
	if got := ids(model.LinkQuery{Q: "epites"}); got != want(zebra) {
		t.Errorf("q tag without accents = %s", got)
	}
}

func TestPinnedGroupOrder(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	v := u.Viewer()
	x := seedLink(t, st, u, "x").ID
	y := seedLink(t, st, u, "y").ID
	z := seedLink(t, st, u, "z").ID
	for _, id := range []int64{x, y, z} {
		mustNoErr(t, st.SetPinned(ctx, v, id, true))
	}
	visit := func(id int64) {
		clk.Advance(time.Second)
		_, err := st.RecordVisit(ctx, v, id)
		mustNoErr(t, err)
	}
	// x: 2 uses; y: 1 use (late); z: 1 use (early). Pinning again is idempotent.
	visit(z)
	visit(x)
	visit(x)
	visit(y)
	mustNoErr(t, st.SetPinned(ctx, v, y, true))
	page, err := st.Links(ctx, v, model.LinkQuery{Limit: 10})
	mustNoErr(t, err)
	got := []int64{}
	for _, it := range page.Items {
		got = append(got, it.ID)
	}
	if fmt.Sprint(got) != fmt.Sprint([]int64{x, y, z}) {
		t.Errorf("pinned order = %v, want %v", got, []int64{x, y, z})
	}
}

func TestPersonalStateUpserts(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	owner := seedUser(t, st, model.RoleAdmin)
	cust := seedUser(t, st, model.RoleCustomer)
	stranger := seedUser(t, st, model.RoleCustomer)
	l := seedLink(t, st, owner, "Shared")
	share(t, st, owner, l.ID, shareUser(cust.ID))
	v := cust.Viewer()

	clk.Advance(time.Hour)
	mustNoErr(t, st.SetArchived(ctx, v, l.ID, true))
	it, _ := st.LinkItem(ctx, v, l.ID)
	if it.ArchivedAt == nil || it.ArchiveReason == nil || *it.ArchiveReason != model.ArchiveManual || it.Pinned {
		t.Fatalf("archived = %+v", it)
	}
	// Pinning an archived link restores it.
	clk.Advance(time.Hour)
	mustNoErr(t, st.SetPinned(ctx, v, l.ID, true))
	it, _ = st.LinkItem(ctx, v, l.ID)
	if !it.Pinned || it.ArchivedAt != nil || !it.BaselineAt.Equal(clk.Now()) {
		t.Fatalf("pinned = %+v", it)
	}
	// Archiving unpins.
	mustNoErr(t, st.SetArchived(ctx, v, l.ID, true))
	it, _ = st.LinkItem(ctx, v, l.ID)
	if it.Pinned || it.ArchivedAt == nil {
		t.Fatalf("archived again = %+v", it)
	}
	clk.Advance(time.Hour)
	mustNoErr(t, st.SetArchived(ctx, v, l.ID, false))
	it, _ = st.LinkItem(ctx, v, l.ID)
	if it.ArchivedAt != nil || it.ArchiveReason != nil || !it.BaselineAt.Equal(clk.Now()) {
		t.Fatalf("restored = %+v", it)
	}
	mustNoErr(t, st.SetPinned(ctx, v, l.ID, true))
	clk.Advance(time.Hour)
	mustNoErr(t, st.SetPinned(ctx, v, l.ID, false))
	it, _ = st.LinkItem(ctx, v, l.ID)
	if it.Pinned || !it.BaselineAt.Equal(clk.Now()) {
		t.Fatalf("unpinned = %+v", it)
	}

	// Every visit counts and restores an archived link.
	mustNoErr(t, st.SetArchived(ctx, v, l.ID, true))
	for i := 1; i <= 3; i++ {
		clk.Advance(time.Minute)
		state, err := st.RecordVisit(ctx, v, l.ID)
		mustNoErr(t, err)
		if state.UseCount != int64(i) || state.LastUsedAt == nil || !state.LastUsedAt.Equal(clk.Now()) ||
			state.ArchivedAt != nil {
			t.Fatalf("visit %d = %+v", i, state)
		}
	}
	// A fresh state row from a first visit.
	fresh := seedLink(t, st, stranger, "fresh")
	state, err := st.RecordVisit(ctx, stranger.Viewer(), fresh.ID)
	mustNoErr(t, err)
	if state.UseCount != 1 || state.BaselineAt.IsZero() {
		t.Errorf("first visit = %+v", state)
	}

	// The owner's state is separate.
	ownerItem, _ := st.LinkItem(ctx, owner.Viewer(), l.ID)
	if ownerItem.UseCount != 0 || ownerItem.ArchivedAt != nil {
		t.Errorf("owner state touched: %+v", ownerItem)
	}

	// Invisible links: every state write is ErrNotFound.
	sv := stranger.Viewer()
	wantErr(t, st.SetPinned(ctx, sv, l.ID, true), model.ErrNotFound)
	wantErr(t, st.SetArchived(ctx, sv, l.ID, true), model.ErrNotFound)
	_, err = st.RecordVisit(ctx, sv, l.ID)
	wantErr(t, err, model.ErrNotFound)
	wantErr(t, st.SetLinkTags(ctx, sv, l.ID, nil), model.ErrNotFound)
	_, err = st.LinkIcon(ctx, sv, l.ID)
	wantErr(t, err, model.ErrNotFound)
	_, err = linkStateQ(ctx, st.db, stranger.ID, l.ID)
	wantErr(t, err, model.ErrNotFound)
}

func TestLinkTagsArePersonal(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	owner := seedUser(t, st, model.RoleDeveloper)
	rcpt := seedUser(t, st, model.RoleCustomer)
	l := seedLink(t, st, owner, "Link")
	share(t, st, owner, l.ID, shareAll())
	ownerTag := seedTag(t, st, owner, "Mine")
	rcptTag := seedTag(t, st, rcpt, "Theirs")
	rcptTag2 := seedTag(t, st, rcpt, "Theirs 2")

	mustNoErr(t, st.SetLinkTags(ctx, owner.Viewer(), l.ID, []int64{ownerTag.ID}))
	mustNoErr(t, st.SetLinkTags(ctx, rcpt.Viewer(), l.ID, []int64{rcptTag.ID, rcptTag2.ID}))
	// The owner's tag is not usable by the recipient.
	wantErr(t, st.SetLinkTags(ctx, rcpt.Viewer(), l.ID, []int64{ownerTag.ID}), model.ErrConflict)

	ownerItem, _ := st.LinkItem(ctx, owner.Viewer(), l.ID)
	rcptItem, _ := st.LinkItem(ctx, rcpt.Viewer(), l.ID)
	if fmt.Sprint(ownerItem.TagIDs) != fmt.Sprint([]int64{ownerTag.ID}) ||
		fmt.Sprint(rcptItem.TagIDs) != fmt.Sprint([]int64{rcptTag.ID, rcptTag2.ID}) {
		t.Fatalf("tag ids owner %v recipient %v", ownerItem.TagIDs, rcptItem.TagIDs)
	}
	// Replacing the recipient's tags leaves the owner's.
	mustNoErr(t, st.SetLinkTags(ctx, rcpt.Viewer(), l.ID, []int64{rcptTag2.ID}))
	mustNoErr(t, st.SetLinkTags(ctx, rcpt.Viewer(), l.ID, []int64{rcptTag2.ID})) // idempotent
	ownerItem, _ = st.LinkItem(ctx, owner.Viewer(), l.ID)
	rcptItem, _ = st.LinkItem(ctx, rcpt.Viewer(), l.ID)
	if len(ownerItem.TagIDs) != 1 || fmt.Sprint(rcptItem.TagIDs) != fmt.Sprint([]int64{rcptTag2.ID}) {
		t.Fatalf("after replace owner %v recipient %v", ownerItem.TagIDs, rcptItem.TagIDs)
	}
	d, _ := st.Link(ctx, rcpt.Viewer(), l.ID)
	if len(d.Tags) != 1 || d.Tags[0].Name != "Theirs 2" || d.Tags[0].LinkCount != 1 {
		t.Errorf("recipient detail tags = %+v", d.Tags)
	}

	// Tag counts follow visibility: unsharing hides the link from the count.
	tags, err := st.Tags(ctx, rcpt.ID, rcpt.Role)
	mustNoErr(t, err)
	if len(tags) != 2 || tags[1].LinkCount != 1 || tags[1].ActiveLinkCount != 1 {
		t.Fatalf("recipient tags = %+v", tags)
	}
	mustNoErr(t, st.SetArchived(ctx, rcpt.Viewer(), l.ID, true))
	tags, _ = st.Tags(ctx, rcpt.ID, rcpt.Role)
	if tags[1].LinkCount != 1 || tags[1].ActiveLinkCount != 0 {
		t.Errorf("archived counts = %+v", tags[1])
	}
	shares, _ := st.Shares(ctx, owner.Viewer(), l.ID)
	mustNoErr(t, st.DeleteShare(ctx, owner.Viewer(), l.ID, shares[0].ID))
	tags, _ = st.Tags(ctx, rcpt.ID, rcpt.Role)
	if tags[1].LinkCount != 0 {
		t.Errorf("invisible link still counted: %+v", tags[1])
	}

	// Deleting a tag removes it from links.
	mustNoErr(t, st.DeleteTag(ctx, owner.ID, ownerTag.ID))
	ownerItem, _ = st.LinkItem(ctx, owner.Viewer(), l.ID)
	if len(ownerItem.TagIDs) != 0 {
		t.Errorf("deleted tag still on link: %v", ownerItem.TagIDs)
	}
}

func TestLinkIcon(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	other := seedUser(t, st, model.RoleCustomer)
	l := seedLink(t, st, u, "")
	_, err := st.LinkIcon(ctx, u.Viewer(), l.ID)
	wantErr(t, err, model.ErrNotFound)

	icon := &model.IconBlob{Data: []byte("\x89PNG"), ContentType: "image/png"}
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: l.ID, URL: l.URL, Status: model.FetchOK, Icon: icon}))
	got, err := st.LinkIcon(ctx, u.Viewer(), l.ID)
	mustNoErr(t, err)
	if string(got.Data) != "\x89PNG" || got.ContentType != "image/png" || len(got.SHA256) != 64 {
		t.Errorf("icon = %+v", got)
	}
	it, _ := st.LinkItem(ctx, u.Viewer(), l.ID)
	if it.IconSHA256 != got.SHA256 {
		t.Errorf("item sha %q, icon sha %q", it.IconSHA256, got.SHA256)
	}
	_, err = st.LinkIcon(ctx, other.Viewer(), l.ID)
	wantErr(t, err, model.ErrNotFound)

	// Icons are deduplicated by content.
	l2 := seedLink(t, st, other, "")
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: l2.ID, URL: l2.URL, Status: model.FetchOK,
		Icon: &model.IconBlob{Data: []byte("\x89PNG"), ContentType: "image/png"}}))
	if n := countRows(t, st, `SELECT COUNT(*) FROM icons`); n != 1 {
		t.Errorf("icons = %d, want 1", n)
	}
	// Invalid icons are ignored.
	for _, bad := range []*model.IconBlob{
		{Data: []byte("x"), ContentType: "text/html"},
		{Data: nil, ContentType: "image/png"},
		{Data: make([]byte, model.MaxIconBytes+1), ContentType: "image/png"},
	} {
		id, err := upsertIconQ(ctx, st.db, bad, formatTime(time.Now()))
		if err != nil || id != 0 {
			t.Errorf("upsertIconQ(%s, %d bytes) = %d, %v", bad.ContentType, len(bad.Data), id, err)
		}
	}
}
