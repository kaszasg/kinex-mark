package store

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestSharesCRUD(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	owner := seedUser(t, st, model.RoleDeveloper)
	peer := seedUser(t, st, model.RoleDeveloper)
	cust := seedUser(t, st, model.RoleCustomer)
	inactive := seedUser(t, st, model.RoleCustomer)
	_, err := st.UpdateUser(ctx, inactive.ID, model.UserPatch{Active: ptr(false)})
	mustNoErr(t, err)
	l := seedLink(t, st, owner, "Doc")
	ov := owner.Viewer()

	all := share(t, st, owner, l.ID, shareAll())
	if all.TargetType != model.ShareAll || all.TargetRole != nil || all.TargetUser != nil || !all.CreatedAt.Equal(clk.Now()) {
		t.Errorf("share all = %+v", all)
	}
	clk.Advance(time.Second)
	byRole := share(t, st, owner, l.ID, shareRole(model.RoleCustomer))
	if byRole.TargetRole == nil || *byRole.TargetRole != model.RoleCustomer {
		t.Errorf("share role = %+v", byRole)
	}
	clk.Advance(time.Second)
	byUser := share(t, st, owner, l.ID, shareUser(cust.ID))
	if byUser.TargetUser == nil || byUser.TargetUser.ID != cust.ID || byUser.TargetUser.Username != cust.Username {
		t.Errorf("share user = %+v", byUser)
	}

	_, err = st.CreateShare(ctx, ov, l.ID, shareAll())
	wantDuplicate(t, err, model.DuplicateShare, all.ID)
	_, err = st.CreateShare(ctx, ov, l.ID, shareRole(model.RoleCustomer))
	wantDuplicate(t, err, model.DuplicateShare, byRole.ID)
	_, err = st.CreateShare(ctx, ov, l.ID, shareUser(cust.ID))
	wantDuplicate(t, err, model.DuplicateShare, byUser.ID)

	invalid := []model.ShareTarget{
		{TargetType: model.ShareAll, TargetRole: ptr(model.RoleAdmin)},
		{TargetType: model.ShareRole},
		{TargetType: model.ShareRole, TargetRole: ptr(model.Role("root"))},
		{TargetType: model.ShareRole, TargetRole: ptr(model.RoleAdmin), TargetUserID: &cust.ID},
		{TargetType: model.ShareUser},
		shareUser(owner.ID),
		shareUser(inactive.ID),
		shareUser(9999),
		{TargetType: model.ShareUser, TargetUserID: &cust.ID, TargetRole: ptr(model.RoleAdmin)},
		{TargetType: "group"},
	}
	for i, target := range invalid {
		_, err := st.CreateShare(ctx, ov, l.ID, target)
		if !errors.Is(err, model.ErrConflict) {
			t.Errorf("invalid target %d: %v", i, err)
		}
	}

	list, err := st.Shares(ctx, ov, l.ID)
	mustNoErr(t, err)
	if len(list) != 3 || list[0].ID != all.ID || list[2].ID != byUser.ID {
		t.Errorf("shares = %+v", list)
	}

	// Non-owners: visible link is ErrNotOwner, invisible is ErrNotFound.
	_, err = st.Shares(ctx, cust.Viewer(), l.ID)
	wantErr(t, err, model.ErrNotOwner)
	_, err = st.CreateShare(ctx, peer.Viewer(), l.ID, shareAll())
	wantErr(t, err, model.ErrNotOwner)
	private := seedLink(t, st, owner, "Private")
	_, err = st.Shares(ctx, peer.Viewer(), private.ID)
	wantErr(t, err, model.ErrNotFound)
	_, err = st.Shares(ctx, ov, 9999)
	wantErr(t, err, model.ErrNotFound)

	// Nested id rule (4.7): a share id under another link is not found.
	peerLink := seedLink(t, st, peer, "Peer")
	wantErr(t, st.DeleteShare(ctx, peer.Viewer(), peerLink.ID, all.ID), model.ErrNotFound)
	wantErr(t, st.DeleteShare(ctx, ov, private.ID, all.ID), model.ErrNotFound)
	wantErr(t, st.DeleteShare(ctx, cust.Viewer(), l.ID, all.ID), model.ErrNotOwner)
	wantErr(t, st.DeleteShare(ctx, peer.Viewer(), private.ID, all.ID), model.ErrNotFound)
	wantErr(t, st.DeleteShare(ctx, ov, 9999, all.ID), model.ErrNotFound)
	if n := countRows(t, st, `SELECT COUNT(*) FROM link_shares WHERE id = ?`, all.ID); n != 1 {
		t.Fatal("share removed through a foreign route")
	}
	mustNoErr(t, st.DeleteShare(ctx, ov, l.ID, all.ID))
	wantErr(t, st.DeleteShare(ctx, ov, l.ID, all.ID), model.ErrNotFound)

	// Deleting the target user cascades the personal share.
	admin := seedUser(t, st, model.RoleAdmin)
	mustNoErr(t, st.DeleteUser(ctx, cust.ID, model.AuditEntry{Event: model.EventUserDeleted, Success: true,
		Actor: model.AuditActor{ID: &admin.ID, Username: admin.Username}}))
	list, _ = st.Shares(ctx, ov, l.ID)
	if len(list) != 1 || list[0].ID != byRole.ID {
		t.Errorf("shares after target deletion = %+v", list)
	}
}

func TestTagsCRUD(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	other := seedUser(t, st, model.RoleCustomer)

	b, err := st.CreateTag(ctx, u.ID, "Backend", model.TagColorBlue)
	mustNoErr(t, err)
	if b.Name != "Backend" || b.Color != model.TagColorBlue || b.LinkCount != 0 || !b.CreatedAt.Equal(clk.Now()) {
		t.Errorf("created = %+v", b)
	}
	a, err := st.CreateTag(ctx, u.ID, "Árlista", "")
	mustNoErr(t, err)
	if a.Color != model.TagColorNeutral {
		t.Errorf("default color = %q", a.Color)
	}
	_, err = st.CreateTag(ctx, u.ID, "BACKEND", model.TagColorRose)
	wantDuplicate(t, err, model.DuplicateTag, b.ID)
	_, err = st.CreateTag(ctx, other.ID, "Backend", "")
	mustNoErr(t, err)
	_, err = st.CreateTag(ctx, u.ID, "Bad", model.TagColor("red"))
	if err == nil {
		t.Error("color CHECK not enforced")
	}
	_, err = st.CreateTag(ctx, 9999, "Orphan", "")
	if err == nil {
		t.Error("tag for a missing user created")
	}

	tags, err := st.Tags(ctx, u.ID, u.Role)
	mustNoErr(t, err)
	if len(tags) != 2 || tags[0].ID != b.ID || tags[1].ID != a.ID {
		// name_key order: "backend" < "árlista" bytewise
		t.Errorf("tags = %+v", tags)
	}

	clk.Advance(time.Minute)
	up, err := st.UpdateTag(ctx, u.ID, b.ID, model.TagPatch{Name: ptr("Back-end"), Color: ptr(model.TagColorGreen)})
	mustNoErr(t, err)
	if up.Name != "Back-end" || up.Color != model.TagColorGreen || !up.UpdatedAt.Equal(clk.Now()) {
		t.Errorf("updated = %+v", up)
	}
	up, err = st.UpdateTag(ctx, u.ID, b.ID, model.TagPatch{Name: ptr("back-END")}) // own name, other case
	mustNoErr(t, err)
	if up.Name != "back-END" {
		t.Errorf("case rename = %+v", up)
	}
	same, err := st.UpdateTag(ctx, u.ID, b.ID, model.TagPatch{})
	mustNoErr(t, err)
	if same.Name != "back-END" {
		t.Errorf("empty patch = %+v", same)
	}
	_, err = st.UpdateTag(ctx, u.ID, b.ID, model.TagPatch{Name: ptr("árlista")})
	wantDuplicate(t, err, model.DuplicateTag, a.ID)
	_, err = st.UpdateTag(ctx, other.ID, b.ID, model.TagPatch{Name: ptr("x")})
	wantErr(t, err, model.ErrNotFound)
	_, err = st.UpdateTag(ctx, 9999, b.ID, model.TagPatch{Name: ptr("x")})
	wantErr(t, err, model.ErrNotFound)

	wantErr(t, st.DeleteTag(ctx, other.ID, b.ID), model.ErrNotFound)
	mustNoErr(t, st.DeleteTag(ctx, u.ID, b.ID))
	wantErr(t, st.DeleteTag(ctx, u.ID, b.ID), model.ErrNotFound)
}

func TestTagLimit(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	now := formatTime(clk.Now())
	tx, err := st.db.Begin()
	mustNoErr(t, err)
	for i := 0; i < model.MaxTagsPerUser; i++ {
		_, err := tx.Exec(`INSERT INTO tags (owner_id, name, name_key, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`, u.ID, "t"+strconv.Itoa(i), "t"+strconv.Itoa(i), now, now)
		mustNoErr(t, err)
	}
	mustNoErr(t, tx.Commit())
	_, err = st.CreateTag(ctx, u.ID, "One more", "")
	wantErr(t, err, model.ErrLimitReached)
	_, err = st.CreateTag(ctx, u.ID, "T1", "")
	wantDuplicate(t, err, model.DuplicateTag, 2)
}
