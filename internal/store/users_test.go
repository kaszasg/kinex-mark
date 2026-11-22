package store

import (
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

func TestCreateUserAndLookups(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)

	n, err := st.CountUsers(ctx)
	if err != nil || n != 0 {
		t.Fatalf("CountUsers empty = %d, %v", n, err)
	}
	u, err := st.CreateUser(ctx, model.NewUser{
		Username: "Anna.K", DisplayName: "Kovács Anna", Role: model.RoleDeveloper,
		PasswordHash: "h1", MustChangePassword: true, Locale: model.LocaleDE,
	})
	mustNoErr(t, err)
	if u.ID == 0 || u.Username != "Anna.K" || u.UsernameKey != "anna.k" || u.Role != model.RoleDeveloper ||
		!u.MustChangePassword || !u.Active || u.TOTPEnabled || u.LastLoginAt != nil {
		t.Fatalf("created user = %+v", u)
	}
	if !u.CreatedAt.Equal(testutil.T0) || !u.PasswordChangedAt.Equal(testutil.T0) {
		t.Errorf("timestamps = %v %v", u.CreatedAt, u.PasswordChangedAt)
	}
	prefs, err := st.Preferences(ctx, u.ID)
	mustNoErr(t, err)
	if prefs.Locale != model.LocaleDE || prefs.Theme != model.ThemeDark {
		t.Errorf("preferences = %+v", prefs)
	}

	_, err = st.CreateUser(ctx, model.NewUser{Username: "ANNA.k", Role: model.RoleCustomer, PasswordHash: "h"})
	wantDuplicate(t, err, model.DuplicateUsername, 0)

	_, err = st.CreateUser(ctx, model.NewUser{Username: "x", Role: model.RoleCustomer, PasswordHash: "h"})
	if err == nil {
		t.Fatal("CHECK on username length not enforced")
	}
	_, err = st.CreateUser(ctx, model.NewUser{Username: "badlocale", Role: model.RoleCustomer,
		PasswordHash: "h", Locale: "fr"})
	if err == nil {
		t.Fatal("CHECK on locale not enforced")
	}
	if n, _ := st.CountUsers(ctx); n != 1 {
		t.Fatalf("CountUsers = %d, want 1 (failed inserts rolled back)", n)
	}

	got, err := st.UserByID(ctx, u.ID)
	mustNoErr(t, err)
	if got.Username != u.Username {
		t.Errorf("UserByID = %+v", got)
	}
	_, err = st.UserByID(ctx, 9999)
	wantErr(t, err, model.ErrNotFound)

	c, err := st.Credentials(ctx, "anna.k")
	mustNoErr(t, err)
	if c.User.ID != u.ID || c.PasswordHash != "h1" || c.TOTPSecret != "" || c.TOTPLastStep != 0 {
		t.Errorf("Credentials = %+v", c)
	}
	_, err = st.Credentials(ctx, "nobody")
	wantErr(t, err, model.ErrNotFound)

	clk.Advance(time.Hour)
	mustNoErr(t, st.SetPassword(ctx, u.ID, "h2", false))
	mustNoErr(t, st.RecordLogin(ctx, u.ID))
	got, _ = st.UserByID(ctx, u.ID)
	if got.MustChangePassword || !got.PasswordChangedAt.Equal(clk.Now()) || got.LastLoginAt == nil ||
		!got.LastLoginAt.Equal(clk.Now()) {
		t.Errorf("after SetPassword/RecordLogin = %+v", got)
	}
	c, _ = st.Credentials(ctx, "anna.k")
	if c.PasswordHash != "h2" {
		t.Errorf("hash = %q", c.PasswordHash)
	}
	wantErr(t, st.SetPassword(ctx, 9999, "h", true), model.ErrNotFound)
	wantErr(t, st.RecordLogin(ctx, 9999), model.ErrNotFound)
}

func TestUpdateUserAndLastAdmin(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	admin := seedUser(t, st, model.RoleAdmin)
	dev := seedUser(t, st, model.RoleDeveloper)

	_, err := st.UpdateUser(ctx, admin.ID, model.UserPatch{Role: ptr(model.RoleDeveloper)})
	wantErr(t, err, model.ErrLastAdmin)
	_, err = st.UpdateUser(ctx, admin.ID, model.UserPatch{Active: ptr(false)})
	wantErr(t, err, model.ErrLastAdmin)

	u, err := st.UpdateUser(ctx, admin.ID, model.UserPatch{DisplayName: ptr("Main Admin")})
	mustNoErr(t, err)
	if u.DisplayName != "Main Admin" || u.Role != model.RoleAdmin {
		t.Errorf("display update = %+v", u)
	}
	same, err := st.UpdateUser(ctx, admin.ID, model.UserPatch{})
	mustNoErr(t, err)
	if same.DisplayName != "Main Admin" {
		t.Errorf("empty patch changed the user: %+v", same)
	}

	// Promote the developer: now the first admin may be demoted.
	_, err = st.UpdateUser(ctx, dev.ID, model.UserPatch{Role: ptr(model.RoleAdmin)})
	mustNoErr(t, err)
	u, err = st.UpdateUser(ctx, admin.ID, model.UserPatch{Role: ptr(model.RoleCustomer), Active: ptr(false)})
	mustNoErr(t, err)
	if u.Role != model.RoleCustomer || u.Active {
		t.Errorf("demoted = %+v", u)
	}
	// The remaining admin is the last one again.
	_, err = st.UpdateUser(ctx, dev.ID, model.UserPatch{Active: ptr(false)})
	wantErr(t, err, model.ErrLastAdmin)
	// Changing an inactive user's role is not restricted.
	_, err = st.UpdateUser(ctx, admin.ID, model.UserPatch{Role: ptr(model.RoleDeveloper)})
	mustNoErr(t, err)

	_, err = st.UpdateUser(ctx, 9999, model.UserPatch{DisplayName: ptr("x")})
	wantErr(t, err, model.ErrNotFound)
	_, err = st.UpdateUser(ctx, dev.ID, model.UserPatch{Role: ptr(model.Role("root"))})
	if err == nil {
		t.Error("invalid role accepted")
	}
}

func TestAdminUsersAndDirectory(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	mk := func(name, display string, role model.Role) model.User {
		u, err := st.CreateUser(ctx, model.NewUser{Username: name, DisplayName: display, Role: role, PasswordHash: "h"})
		mustNoErr(t, err)
		return u
	}
	admin := mk("admin", "Rendszergazda", model.RoleAdmin)
	agnes := mk("agnes", "Ágnes Nagy", model.RoleDeveloper)
	bela := mk("bela", "Béla", model.RoleCustomer)
	cecil := mk("cecil", "Cecil", model.RoleCustomer)
	_, err := st.UpdateUser(ctx, cecil.ID, model.UserPatch{Active: ptr(false)})
	mustNoErr(t, err)

	// Counters for agnes: two links (one shared), one live session, one lock.
	l1 := seedLink(t, st, agnes, "")
	seedLink(t, st, agnes, "")
	share(t, st, agnes, l1.ID, shareAll())
	_, err = st.CreateSession(ctx, model.NewSession{TokenHash: make32(1), UserID: agnes.ID,
		ExpiresAt: clk.Now().Add(time.Hour)})
	mustNoErr(t, err)
	_, err = st.CreateSession(ctx, model.NewSession{TokenHash: make32(2), UserID: agnes.ID,
		ExpiresAt: clk.Now().Add(-time.Hour)})
	mustNoErr(t, err)
	lockUntil := clk.Now().Add(15 * time.Minute)
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: "u:agnes", UserID: &agnes.ID}},
		func(r model.ThrottleRow) (model.ThrottleRow, error) {
			r.Failures, r.LockedUntil = 5, &lockUntil
			return r, nil
		}))

	all, total, err := st.AdminUsers(ctx, model.UserFilter{}, model.PageReq{Limit: 50})
	mustNoErr(t, err)
	if total != 4 || len(all) != 4 || all[0].Username != "admin" || all[1].Username != "agnes" {
		t.Fatalf("all users = %d %+v", total, all)
	}
	a := all[1]
	if a.LinkCount != 2 || a.SharedLinkCount != 1 || a.SessionCount != 1 || a.LockedUntil == nil ||
		!a.LockedUntil.Equal(lockUntil) {
		t.Errorf("agnes counters = %+v", a)
	}
	if all[0].LockedUntil != nil {
		t.Errorf("admin lockedUntil = %v", all[0].LockedUntil)
	}

	cases := []struct {
		name string
		f    model.UserFilter
		want []string
	}{
		{"accented display name", model.UserFilter{Query: "ágnes"}, []string{"agnes"}},
		{"upper-case query", model.UserFilter{Query: "BÉLA"}, []string{"bela"}},
		{"username part", model.UserFilter{Query: "ec"}, []string{"cecil"}},
		{"role", model.UserFilter{Role: model.RoleCustomer}, []string{"bela", "cecil"}},
		{"active", model.UserFilter{Status: model.UserStatusActive}, []string{"admin", "agnes", "bela"}},
		{"inactive", model.UserFilter{Status: model.UserStatusInactive}, []string{"cecil"}},
		{"no match", model.UserFilter{Query: "zzz"}, nil},
	}
	for _, c := range cases {
		items, n, err := st.AdminUsers(ctx, c.f, model.PageReq{Limit: 50})
		mustNoErr(t, err)
		if n != len(c.want) || len(items) != len(c.want) {
			t.Errorf("%s: total %d items %d, want %d", c.name, n, len(items), len(c.want))
			continue
		}
		for i, name := range c.want {
			if items[i].Username != name {
				t.Errorf("%s: item %d = %s, want %s", c.name, i, items[i].Username, name)
			}
		}
	}

	page, n, err := st.AdminUsers(ctx, model.UserFilter{}, model.PageReq{Limit: 2, Offset: 2})
	mustNoErr(t, err)
	if n != 4 || len(page) != 2 || page[0].Username != "bela" {
		t.Errorf("page 2 = %d %+v", n, page)
	}
	page, n, err = st.AdminUsers(ctx, model.UserFilter{}, model.PageReq{Limit: 2, Offset: 10})
	mustNoErr(t, err)
	if n != 4 || len(page) != 0 || page == nil {
		t.Errorf("past last page = %d %v", n, page)
	}

	one, err := st.AdminUser(ctx, agnes.ID)
	mustNoErr(t, err)
	if one.Username != "agnes" || one.LinkCount != 2 {
		t.Errorf("AdminUser = %+v", one)
	}
	_, err = st.AdminUser(ctx, 9999)
	wantErr(t, err, model.ErrNotFound)

	// Lock expiry hides lockedUntil.
	clk.Advance(time.Hour)
	one, _ = st.AdminUser(ctx, agnes.ID)
	if one.LockedUntil != nil || one.SessionCount != 0 {
		t.Errorf("after expiry = %+v", one)
	}

	dir, err := st.Directory(ctx, admin.ID, "", 50)
	mustNoErr(t, err)
	if len(dir) != 2 || dir[0].Username != "agnes" || dir[0].Role != model.RoleDeveloper || dir[1].Username != "bela" {
		t.Errorf("directory = %+v", dir)
	}
	dir, _ = st.Directory(ctx, admin.ID, "ÁGN", 50)
	if len(dir) != 1 || dir[0].DisplayName != "Ágnes Nagy" {
		t.Errorf("directory search = %+v", dir)
	}
	dir, _ = st.Directory(ctx, agnes.ID, "", 1)
	if len(dir) != 1 || dir[0].Username != "admin" {
		t.Errorf("directory limit = %+v", dir)
	}
	_ = bela

	cnt, err := st.OwnSharedLinkCount(ctx, agnes.ID)
	if err != nil || cnt != 1 {
		t.Errorf("OwnSharedLinkCount = %d, %v", cnt, err)
	}
}

func make32(b byte) []byte {
	h := make([]byte, 32)
	for i := range h {
		h[i] = b
	}
	return h
}
