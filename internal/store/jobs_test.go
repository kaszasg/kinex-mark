package store

import (
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

const day = 24 * time.Hour

func archivedAt(t *testing.T, st *Store, u model.User, id int64) *time.Time {
	t.Helper()
	it, err := st.LinkItem(ctx, u.Viewer(), id)
	mustNoErr(t, err)
	if it.ArchivedAt != nil && (it.ArchiveReason == nil || *it.ArchiveReason != model.ArchiveAuto) {
		t.Fatalf("archive reason = %v", it.ArchiveReason)
	}
	return it.ArchivedAt
}

func TestArchiveInactiveBoundaries(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	l := seedLink(t, st, u, "")
	pinned := seedLink(t, st, u, "")
	mustNoErr(t, st.SetPinned(ctx, u.Viewer(), pinned.ID, true))

	clk.Set(testutil.T0.Add(89 * day))
	n, err := st.ArchiveInactive(ctx, 90)
	if err != nil || n != 0 {
		t.Fatalf("day 89: %d, %v", n, err)
	}
	clk.Set(testutil.T0.Add(90 * day))
	if n, _ := st.ArchiveInactive(ctx, 90); n != 0 {
		t.Fatalf("day 90 exactly: %d archived (cutoff is strict)", n)
	}
	clk.Set(testutil.T0.Add(91 * day))
	if n, _ := st.ArchiveInactive(ctx, 90); n != 1 {
		t.Fatalf("day 91: %d archived, want 1", n)
	}
	if archivedAt(t, st, u, l.ID) == nil || archivedAt(t, st, u, pinned.ID) != nil {
		t.Fatal("wrong links archived (pinned must stay)")
	}
	// Already archived links are not counted again.
	if n, _ := st.ArchiveInactive(ctx, 90); n != 0 {
		t.Errorf("second run archived %d", n)
	}
	// Restoring restarts the period.
	mustNoErr(t, st.SetArchived(ctx, u.Viewer(), l.ID, false))
	clk.Advance(89 * day)
	if n, _ := st.ArchiveInactive(ctx, 90); n != 0 {
		t.Errorf("89 days after restore: %d", n)
	}
	// A visit restarts it too.
	_, err = st.RecordVisit(ctx, u.Viewer(), l.ID)
	mustNoErr(t, err)
	clk.Advance(2 * day)
	if n, _ := st.ArchiveInactive(ctx, 90); n != 0 {
		t.Errorf("after visit: %d", n)
	}
	// 0 means never.
	clk.Advance(1000 * day)
	if n, _ := st.ArchiveInactive(ctx, 0); n != 0 {
		t.Errorf("defaultDays 0 archived %d", n)
	}
}

func TestArchiveInactiveOverridesAndSharing(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	owner := seedUser(t, st, model.RoleDeveloper)
	short := seedUser(t, st, model.RoleCustomer)
	never := seedUser(t, st, model.RoleCustomer)
	inactive := seedUser(t, st, model.RoleCustomer)
	_, err := st.UpdatePreferences(ctx, short.ID, model.PreferencesPatch{ArchiveAfterDays: model.NullableInt{Set: true, Value: ptr(10)}})
	mustNoErr(t, err)
	_, err = st.UpdatePreferences(ctx, never.ID, model.PreferencesPatch{ArchiveAfterDays: model.NullableInt{Set: true, Value: ptr(0)}})
	mustNoErr(t, err)

	own := seedLink(t, st, owner, "")
	shortOwn := seedLink(t, st, short, "")
	inactiveOwn := seedLink(t, st, inactive, "")
	_, err = st.UpdateUser(ctx, inactive.ID, model.UserPatch{Active: ptr(false)})
	mustNoErr(t, err)

	// Shared 20 days after creation: the recipient's period starts at the share.
	clk.Set(testutil.T0.Add(20 * day))
	shared := seedLink(t, st, owner, "")
	clk.Set(testutil.T0.Add(25 * day))
	share(t, st, owner, shared.ID, shareRole(model.RoleCustomer))
	share(t, st, owner, own.ID, shareUser(never.ID))

	// Day 34: the share is 9 days old for "short" (10-day period).
	clk.Set(testutil.T0.Add(34 * day))
	n, err := st.ArchiveInactive(ctx, 90)
	mustNoErr(t, err)
	if n != 1 || archivedAt(t, st, short, shortOwn.ID) == nil || archivedAt(t, st, short, shared.ID) != nil {
		t.Fatalf("day 34: %d archived", n)
	}
	// The job created the recipient's state row with the share time as baseline.
	var baseline string
	mustNoErr(t, st.db.QueryRow(`SELECT baseline_at FROM link_user_state WHERE user_id = ? AND link_id = ?`,
		short.ID, shared.ID).Scan(&baseline))
	if baseline != formatTime(testutil.T0.Add(25*day)) {
		t.Errorf("recipient baseline = %s", baseline)
	}
	clk.Set(testutil.T0.Add(36 * day))
	if n, _ := st.ArchiveInactive(ctx, 90); n != 1 || archivedAt(t, st, short, shared.ID) == nil {
		t.Fatalf("day 36: %d archived", n)
	}

	// Far in the future: never-user keeps everything, inactive user is skipped,
	// the owner's own links follow the default.
	clk.Set(testutil.T0.Add(400 * day))
	n, _ = st.ArchiveInactive(ctx, 90)
	if n != 2 || archivedAt(t, st, owner, own.ID) == nil || archivedAt(t, st, owner, shared.ID) == nil {
		t.Errorf("owner: %d archived", n)
	}
	if archivedAt(t, st, never, own.ID) != nil || archivedAt(t, st, never, shared.ID) != nil {
		t.Error("archive_after_days 0 did not prevent archiving")
	}
	if archivedAt(t, st, inactive, inactiveOwn.ID) != nil {
		t.Error("inactive user's links archived")
	}

	// A demoted owner's shares are not visible, so they are not archived.
	late := seedLink(t, st, owner, "")
	share(t, st, owner, late.ID, shareAll())
	_, err = st.UpdateUser(ctx, owner.ID, model.UserPatch{Role: ptr(model.RoleCustomer)})
	mustNoErr(t, err)
	clk.Advance(100 * day)
	_, err = st.ArchiveInactive(ctx, 90)
	mustNoErr(t, err)
	if n := countRows(t, st, `SELECT COUNT(*) FROM link_user_state WHERE user_id = ? AND link_id = ?`,
		short.ID, late.ID); n != 0 {
		t.Errorf("state row created for an invisible link: %d", n)
	}
}

// A recipient cannot leave a link unused before the account exists or before
// the link becomes visible to it: the period starts at the later of the
// share and the account creation, and at a role change for links shared
// with the new role.
func TestArchiveInactiveLateRecipients(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	owner := seedUser(t, st, model.RoleDeveloper)
	promoted := seedUser(t, st, model.RoleCustomer)
	returning := seedUser(t, st, model.RoleDeveloper)
	toAll := seedLink(t, st, owner, "")
	toDevs := seedLink(t, st, owner, "")
	share(t, st, owner, toAll.ID, shareAll())
	share(t, st, owner, toDevs.ID, shareRole(model.RoleDeveloper))

	// Day 10: the job gives "returning" its state rows (baseline day 0).
	clk.Set(testutil.T0.Add(10 * day))
	_, err := st.ArchiveInactive(ctx, 90)
	mustNoErr(t, err)
	_, err = st.UpdateUser(ctx, returning.ID, model.UserPatch{Role: ptr(model.RoleCustomer)})
	mustNoErr(t, err)

	// Day 100: a new user is created; "promoted" and "returning" become developers.
	clk.Set(testutil.T0.Add(100 * day))
	created := testutil.T0.Add(100 * day)
	newbie := seedUser(t, st, model.RoleDeveloper)
	for _, u := range []*model.User{&promoted, &returning} {
		*u, err = st.UpdateUser(ctx, u.ID, model.UserPatch{Role: ptr(model.RoleDeveloper)})
		mustNoErr(t, err)
	}
	baseline := func(u model.User, id int64) time.Time {
		t.Helper()
		it, err := st.LinkItem(ctx, u.Viewer(), id)
		mustNoErr(t, err)
		return it.BaselineAt
	}
	// Before any job run the list already reports the new baseline.
	for _, id := range []int64{toAll.ID, toDevs.ID} {
		if got := baseline(newbie, id); !got.Equal(created) {
			t.Errorf("newbie baseline of link %d = %v, want the account creation %v", id, got, created)
		}
	}
	if got := baseline(promoted, toDevs.ID); !got.Equal(created) {
		t.Errorf("promoted baseline = %v, want the promotion %v", got, created)
	}
	if got := baseline(returning, toDevs.ID); !got.Equal(created) {
		t.Errorf("returning baseline = %v, want the promotion %v", got, created)
	}
	// The link shared with everyone was visible to "promoted" all along.
	if got := baseline(promoted, toAll.ID); !got.Equal(testutil.T0) {
		t.Errorf("promoted baseline of the link shared with all = %v, want the share", got)
	}

	// Day 101: only the owner's links and the old all-share of "promoted"
	// and "returning" are due.
	clk.Set(testutil.T0.Add(101 * day))
	n, err := st.ArchiveInactive(ctx, 90)
	mustNoErr(t, err)
	if n != 4 {
		t.Errorf("day 101: %d archived, want 4 (owner 2, promoted 1, returning 1)", n)
	}
	for _, c := range []struct {
		u    model.User
		id   int64
		want bool
	}{
		{newbie, toAll.ID, false}, {newbie, toDevs.ID, false},
		{promoted, toAll.ID, true}, {promoted, toDevs.ID, false},
		{returning, toAll.ID, true}, {returning, toDevs.ID, false},
	} {
		if got := archivedAt(t, st, c.u, c.id) != nil; got != c.want {
			t.Errorf("user %d link %d archived = %v, want %v", c.u.ID, c.id, got, c.want)
		}
	}
	if got := baseline(newbie, toAll.ID); !got.Equal(created) {
		t.Errorf("job baseline = %v, want %v", got, created)
	}

	// 90 days after creation and promotion the period ends.
	clk.Set(created.Add(91 * day))
	if n, _ := st.ArchiveInactive(ctx, 90); n != 4 {
		t.Errorf("day 191: %d archived, want 4 (newbie 2, promoted 1, returning 1)", n)
	}
}

// The shares of an owner who was demoted to customer are dormant (4.7). When
// the owner gets a sharing role back, they become visible again, so every
// recipient's period of those links starts at the promotion; the personal
// state (use count) survives.
func TestArchiveInactiveOwnerSharesReturn(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	owner := seedUser(t, st, model.RoleDeveloper)
	cust := seedUser(t, st, model.RoleCustomer)
	dev := seedUser(t, st, model.RoleDeveloper)
	target := seedUser(t, st, model.RoleCustomer)
	toAll := seedLink(t, st, owner, "")
	toCustomers := seedLink(t, st, owner, "")
	toTarget := seedLink(t, st, owner, "")
	share(t, st, owner, toAll.ID, shareAll())
	share(t, st, owner, toCustomers.ID, shareRole(model.RoleCustomer))
	share(t, st, owner, toTarget.ID, shareUser(target.ID))

	clk.Set(testutil.T0.Add(5 * day))
	_, err := st.RecordVisit(ctx, cust.Viewer(), toAll.ID)
	mustNoErr(t, err)
	// Day 10: the job gives every recipient its state rows (baseline day 0).
	clk.Set(testutil.T0.Add(10 * day))
	_, err = st.ArchiveInactive(ctx, 90)
	mustNoErr(t, err)
	// Day 20 to day 200 the owner is a customer: the shares are dormant.
	clk.Set(testutil.T0.Add(20 * day))
	_, err = st.UpdateUser(ctx, owner.ID, model.UserPatch{Role: ptr(model.RoleCustomer)})
	mustNoErr(t, err)
	promotion := testutil.T0.Add(200 * day)
	clk.Set(promotion)
	_, err = st.UpdateUser(ctx, owner.ID, model.UserPatch{Role: ptr(model.RoleDeveloper)})
	mustNoErr(t, err)

	visible := []struct {
		u  model.User
		id int64
	}{
		{cust, toAll.ID}, {cust, toCustomers.ID}, {dev, toAll.ID},
		{target, toAll.ID}, {target, toCustomers.ID}, {target, toTarget.ID},
	}
	for _, c := range visible {
		it, err := st.LinkItem(ctx, c.u.Viewer(), c.id)
		mustNoErr(t, err)
		if !it.BaselineAt.Equal(promotion) {
			t.Errorf("user %d link %d baseline = %v, want the promotion %v", c.u.ID, c.id, it.BaselineAt, promotion)
		}
	}
	it, err := st.LinkItem(ctx, cust.Viewer(), toAll.ID)
	mustNoErr(t, err)
	if it.UseCount != 1 {
		t.Errorf("use count after the return = %d, want 1", it.UseCount)
	}
	// The dev is not a recipient of the customer and user shares.
	if n := countRows(t, st, `SELECT COUNT(*) FROM link_user_state WHERE user_id = ? AND link_id IN (?, ?)`,
		dev.ID, toCustomers.ID, toTarget.ID); n != 0 {
		t.Errorf("state rows for links the dev cannot see: %d", n)
	}

	// Day 201: only the owner's own three links are due.
	clk.Set(promotion.Add(day))
	if n, err := st.ArchiveInactive(ctx, 90); err != nil || n != 3 {
		t.Fatalf("day 201: %d archived, %v; want 3 (the owner's own links)", n, err)
	}
	// 91 days after the promotion the recipients' periods end.
	clk.Set(promotion.Add(91 * day))
	if n, _ := st.ArchiveInactive(ctx, 90); n != len(visible) {
		t.Errorf("day 291: %d archived, want %d", n, len(visible))
	}
}

func TestArchiveInactiveFallbackStateRow(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	l := seedLink(t, st, u, "")
	// Simulate an older own link without a state row.
	_, err := st.db.Exec(`DELETE FROM link_user_state WHERE link_id = ?`, l.ID)
	mustNoErr(t, err)
	clk.Set(testutil.T0.Add(91 * day))
	n, err := st.ArchiveInactive(ctx, 90)
	if err != nil || n != 1 {
		t.Fatalf("fallback: %d, %v", n, err)
	}
}

func TestPurge(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	now := clk.Now()

	// Sessions: absolute expiry, idle expiry (720 minutes) and a live one.
	_, err := st.CreateSession(ctx, model.NewSession{TokenHash: make32(1), UserID: u.ID, ExpiresAt: now.Add(time.Hour)})
	mustNoErr(t, err)
	_, err = st.CreateSession(ctx, model.NewSession{TokenHash: make32(2), UserID: u.ID, ExpiresAt: now.Add(100 * time.Hour)})
	mustNoErr(t, err)
	mustNoErr(t, st.CreateChallenge(ctx, make32(3), u.ID, "", now.Add(5*time.Minute)))
	// Throttle rows: stale, stale but locked, fresh.
	reserve := func(r model.ThrottleRow) (model.ThrottleRow, error) { r.Failures++; return r, nil }
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: "ip:stale"}}, reserve))
	lockUntil := now.Add(48 * time.Hour)
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: "u:locked"}},
		func(r model.ThrottleRow) (model.ThrottleRow, error) {
			r.Failures, r.LockedUntil = 5, &lockUntil
			return r, nil
		}))
	mustNoErr(t, st.SetPendingTOTP(ctx, u.ID, "v1:pending", now.Add(10*time.Minute)))
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventLogout}))
	// An orphan icon and a used one.
	l := seedLink(t, st, u, "")
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: l.ID, URL: l.URL, Status: model.FetchOK,
		Icon: &model.IconBlob{Data: []byte("used"), ContentType: "image/png"}}))
	_, err = upsertIconQ(ctx, st.db, &model.IconBlob{Data: []byte("orphan"), ContentType: "image/png"}, formatTime(now))
	mustNoErr(t, err)

	// Nothing is old yet except the orphan icon.
	res, err := st.Purge(ctx, 180*day)
	mustNoErr(t, err)
	if res != (model.PurgeResult{Icons: 1}) {
		t.Fatalf("first purge = %+v", res)
	}

	clk.Advance(25 * time.Hour)
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: "ip:fresh"}}, reserve))
	_, err = st.CreateSession(ctx, model.NewSession{TokenHash: make32(4), UserID: u.ID, ExpiresAt: clk.Now().Add(time.Hour)})
	mustNoErr(t, err)
	res, err = st.Purge(ctx, 180*day)
	mustNoErr(t, err)
	want := model.PurgeResult{Sessions: 2, Challenges: 1, Throttle: 1, PendingTOTP: 1}
	if res != want {
		t.Fatalf("second purge = %+v, want %+v", res, want)
	}
	if n := countRows(t, st, `SELECT COUNT(*) FROM login_throttle`); n != 2 {
		t.Errorf("throttle rows = %d (locked and fresh stay)", n)
	}
	if n := countRows(t, st, `SELECT COUNT(*) FROM sessions`); n != 1 {
		t.Errorf("sessions = %d", n)
	}

	// Audit retention, explicit and from the setting.
	clk.Advance(180 * day)
	res, err = st.Purge(ctx, 0) // falls back to auditRetentionDays (180)
	mustNoErr(t, err)
	if res.Audit != 1 {
		t.Errorf("audit purge = %+v", res)
	}
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventLogout}))
	clk.Advance(31 * day)
	res, _ = st.Purge(ctx, 30*day)
	if res.Audit != 1 {
		t.Errorf("explicit retention purge = %+v", res)
	}
	if n := countRows(t, st, `SELECT COUNT(*) FROM icons`); n != 1 {
		t.Errorf("icons = %d (the used icon stays)", n)
	}
}
