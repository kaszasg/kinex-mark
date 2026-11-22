package store

import (
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestFetchQueueLeaseAndResults(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	_, err := st.UpdatePreferences(ctx, u.ID, model.PreferencesPatch{Locale: ptr(model.LocaleEN)})
	mustNoErr(t, err)

	// A link created without Enqueue is due for the sweeper.
	l1 := seedLink(t, st, u, "")
	clk.Advance(time.Second)
	l2 := seedLink(t, st, u, "")
	due, err := st.DueFetches(ctx, 10)
	mustNoErr(t, err)
	if len(due) != 2 || due[0] != l1.ID || due[1] != l2.ID {
		t.Fatalf("due = %v", due)
	}
	if due, _ := st.DueFetches(ctx, 1); len(due) != 1 {
		t.Errorf("limit not applied: %v", due)
	}

	job, err := st.BeginFetch(ctx, l1.ID, model.FetchLease)
	mustNoErr(t, err)
	if job.LinkID != l1.ID || job.URL != l1.URL || job.OwnerID != u.ID || job.Locale != model.LocaleEN {
		t.Errorf("job = %+v", job)
	}
	// Leased: not due, a second BeginFetch fails.
	_, err = st.BeginFetch(ctx, l1.ID, model.FetchLease)
	wantErr(t, err, model.ErrNotFound)
	if due, _ := st.DueFetches(ctx, 10); len(due) != 1 || due[0] != l2.ID {
		t.Errorf("due while leased = %v", due)
	}
	// After the lease runs out the link is due again (crashed worker).
	clk.Advance(model.FetchLease)
	if due, _ := st.DueFetches(ctx, 10); len(due) != 2 {
		t.Errorf("due after lease = %v", due)
	}
	_, err = st.BeginFetch(ctx, 9999, model.FetchLease)
	wantErr(t, err, model.ErrNotFound)

	// Transient failures back off 1 and 10 minutes, the third one fails.
	tmp := model.FetchResult{LinkID: l1.ID, URL: l1.URL, Status: model.FetchFailed,
		Error: model.FetchErrTimeout, Temporary: true, Title: "ignored"}
	mustNoErr(t, st.ApplyFetchResult(ctx, tmp))
	d, _ := st.Link(ctx, u.Viewer(), l1.ID)
	if d.FetchStatus != model.FetchPending || d.FetchAttempts != 1 || d.FetchError == nil ||
		*d.FetchError != model.FetchErrTimeout || d.Title != "" || d.FetchedAt != nil {
		t.Fatalf("after 1st transient = %+v", d)
	}
	assertNextFetch(t, st, l1.ID, clk.Now().Add(time.Minute))
	mustNoErr(t, st.ApplyFetchResult(ctx, tmp))
	assertNextFetch(t, st, l1.ID, clk.Now().Add(10*time.Minute))
	mustNoErr(t, st.ApplyFetchResult(ctx, tmp))
	d, _ = st.Link(ctx, u.Viewer(), l1.ID)
	if d.FetchStatus != model.FetchFailed || d.FetchAttempts != 3 || d.FetchedAt == nil || d.Title != "" {
		t.Fatalf("after 3rd transient = %+v", d)
	}
	assertNextFetch(t, st, l1.ID, time.Time{})

	// Final results.
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: l2.ID, URL: l2.URL,
		Status: model.FetchBlocked, Error: model.FetchErrBlockedAddress}))
	d, _ = st.Link(ctx, u.Viewer(), l2.ID)
	if d.FetchStatus != model.FetchBlocked || *d.FetchError != model.FetchErrBlockedAddress || d.TitleSource != model.TitleNone {
		t.Errorf("blocked = %+v", d)
	}
	mustNoErr(t, st.ResetFetch(ctx, u.Viewer(), l2.ID))
	d, _ = st.Link(ctx, u.Viewer(), l2.ID)
	if d.FetchStatus != model.FetchPending || d.FetchError != nil || d.FetchAttempts != 0 {
		t.Errorf("after ResetFetch = %+v", d)
	}
	assertNextFetch(t, st, l2.ID, clk.Now())
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: l2.ID, URL: l2.URL,
		Status: model.FetchOK, Error: model.FetchErrNotHTML}))
	d, _ = st.Link(ctx, u.Viewer(), l2.ID)
	if d.FetchStatus != model.FetchOK || d.FetchError == nil || *d.FetchError != model.FetchErrNotHTML || d.Title != "" {
		t.Errorf("not_html = %+v", d)
	}

	// An invalid final status is rejected.
	if err := st.ApplyFetchResult(ctx, model.FetchResult{LinkID: l2.ID, URL: l2.URL, Status: model.FetchPending}); err == nil {
		t.Error("pending accepted as a final status")
	}
	// A result for a missing link is discarded.
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: 9999, URL: "https://x.example/", Status: model.FetchOK}))
}

func assertNextFetch(t *testing.T, st *Store, id int64, want time.Time) {
	t.Helper()
	var next *string
	mustNoErr(t, st.db.QueryRow(`SELECT next_fetch_at FROM links WHERE id = ?`, id).Scan(&next))
	switch {
	case want.IsZero() && next != nil:
		t.Errorf("next_fetch_at = %q, want NULL", *next)
	case !want.IsZero() && (next == nil || *next != formatTime(want)):
		t.Errorf("next_fetch_at = %v, want %s", next, formatTime(want))
	}
}

func TestApplyFetchResultTitleRules(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	v := u.Viewer()
	icon := &model.IconBlob{Data: []byte("GIF89a"), ContentType: "image/gif"}

	// none -> fetched, with icon.
	n := seedLink(t, st, u, "")
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: n.ID, URL: n.URL, Status: model.FetchOK,
		Title: "Fetched title", Icon: icon}))
	d, _ := st.Link(ctx, v, n.ID)
	if d.Title != "Fetched title" || d.TitleSource != model.TitleFetched || d.IconSHA256 == "" || d.FetchError != nil {
		t.Fatalf("none -> fetched = %+v", d)
	}
	var search, label string
	mustNoErr(t, st.db.QueryRow(`SELECT search_text, sort_label FROM links WHERE id = ?`, n.ID).Scan(&search, &label))
	if label != "fetched title" || search != n.URL+"\nexample.com\nfetched title\n" {
		t.Errorf("search %q label %q", search, label)
	}
	// fetched -> overwritten by a refresh; no icon clears the icon.
	mustNoErr(t, st.ResetFetch(ctx, v, n.ID))
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: n.ID, URL: n.URL, Status: model.FetchOK,
		Title: "Renamed site"}))
	d, _ = st.Link(ctx, v, n.ID)
	if d.Title != "Renamed site" || d.TitleSource != model.TitleFetched || d.IconSHA256 != "" {
		t.Fatalf("refresh = %+v", d)
	}
	// An empty fetched title resets to none.
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: n.ID, URL: n.URL, Status: model.FetchOK}))
	d, _ = st.Link(ctx, v, n.ID)
	if d.Title != "" || d.TitleSource != model.TitleNone {
		t.Fatalf("empty fetched title = %+v", d)
	}
	// failed keeps the title but drops the icon.
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: n.ID, URL: n.URL, Status: model.FetchOK,
		Title: "Kept", Icon: icon}))
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: n.ID, URL: n.URL, Status: model.FetchFailed,
		Error: model.FetchErrHTTPStatus, Title: "Error page"}))
	d, _ = st.Link(ctx, v, n.ID)
	if d.Title != "Kept" || d.FetchStatus != model.FetchFailed || d.IconSHA256 != "" {
		t.Fatalf("failed = %+v", d)
	}

	// manual is never overwritten.
	m := seedLink(t, st, u, "My title")
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: m.ID, URL: m.URL, Status: model.FetchOK,
		Title: "Site title", Icon: icon}))
	d, _ = st.Link(ctx, v, m.ID)
	if d.Title != "My title" || d.TitleSource != model.TitleManual || d.IconSHA256 == "" {
		t.Fatalf("manual = %+v", d)
	}

	// A long fetched title is cut to the column limit.
	long := make([]rune, model.TitleMaxRunes+20)
	for i := range long {
		long[i] = 'x'
	}
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: n.ID, URL: n.URL, Status: model.FetchOK,
		Title: string(long)}))
	d, _ = st.Link(ctx, v, n.ID)
	if len([]rune(d.Title)) != model.TitleMaxRunes {
		t.Errorf("title length = %d", len([]rune(d.Title)))
	}
}

// TestApplyFetchResultDiscardedAfterURLChange is the integration moved here
// from the jobs package (9.2.2): a result for the old URL must not be written.
func TestApplyFetchResultDiscardedAfterURLChange(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	v := u.Viewer()
	l := seedLink(t, st, u, "")
	job, err := st.BeginFetch(ctx, l.ID, model.FetchLease)
	mustNoErr(t, err)

	clk.Advance(time.Second)
	_, err = st.UpdateLink(ctx, v, l.ID, model.LinkPatch{URL: ptr("https://new.example/"), Host: ptr("new.example")})
	mustNoErr(t, err)
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: l.ID, URL: job.URL, Status: model.FetchOK,
		Title: "Old site", Icon: &model.IconBlob{Data: []byte("old"), ContentType: "image/png"}}))

	d, _ := st.Link(ctx, v, l.ID)
	if d.Title != "" || d.TitleSource != model.TitleNone || d.IconSHA256 != "" || d.FetchStatus != model.FetchPending ||
		d.FetchAttempts != 0 || d.FetchedAt != nil {
		t.Fatalf("stale result was applied: %+v", d)
	}
	// The new URL is due right away despite the old lease.
	due, _ := st.DueFetches(ctx, 10)
	if len(due) != 1 || due[0] != l.ID {
		t.Errorf("due = %v", due)
	}
	if n := countRows(t, st, `SELECT COUNT(*) FROM icons`); n != 0 {
		t.Errorf("stale icon stored: %d", n)
	}
}

func TestResetFetchOwnership(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	owner := seedUser(t, st, model.RoleDeveloper)
	cust := seedUser(t, st, model.RoleCustomer)
	l := seedLink(t, st, owner, "")
	wantErr(t, st.ResetFetch(ctx, cust.Viewer(), l.ID), model.ErrNotFound)
	share(t, st, owner, l.ID, shareAll())
	wantErr(t, st.ResetFetch(ctx, cust.Viewer(), l.ID), model.ErrNotOwner)
	wantErr(t, st.ResetFetch(ctx, owner.Viewer(), 9999), model.ErrNotFound)
}
