package store

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

// The store's own tests cannot import internal/storetest (import cycle), so
// they use these helpers (3.3.12).

var ctx = context.Background()

var userSeq atomic.Int64

// newTestStore opens a store in t.TempDir() with a fake clock at testutil.T0.
func newTestStore(t *testing.T) (*Store, *clock.Fake) {
	t.Helper()
	clk := testutil.NewClock()
	st, err := Open(ctx, t.TempDir(), Options{Clock: clk})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st, clk
}

// seedUser creates an active user with a unique username.
func seedUser(t *testing.T, st *Store, role model.Role) model.User {
	t.Helper()
	u, err := st.CreateUser(ctx, model.NewUser{
		Username:     fmt.Sprintf("%s-%d", role, userSeq.Add(1)),
		DisplayName:  "Display " + string(role),
		Role:         role,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return u
}

// seedLink creates a link for owner with a unique URL.
func seedLink(t *testing.T, st *Store, owner model.User, title string) model.LinkDetail {
	t.Helper()
	n := userSeq.Add(1)
	d, err := st.CreateLink(ctx, model.NewLink{
		OwnerID: owner.ID,
		URL:     fmt.Sprintf("https://example.com/page-%d", n),
		Host:    "example.com",
		Title:   title,
	})
	if err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	return d
}

func seedTag(t *testing.T, st *Store, owner model.User, name string) model.Tag {
	t.Helper()
	tag, err := st.CreateTag(ctx, owner.ID, name, "")
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	return tag
}

func share(t *testing.T, st *Store, owner model.User, linkID int64, target model.ShareTarget) model.Share {
	t.Helper()
	sh, err := st.CreateShare(ctx, owner.Viewer(), linkID, target)
	if err != nil {
		t.Fatalf("CreateShare: %v", err)
	}
	return sh
}

func shareAll() model.ShareTarget { return model.ShareTarget{TargetType: model.ShareAll} }

func shareRole(r model.Role) model.ShareTarget {
	return model.ShareTarget{TargetType: model.ShareRole, TargetRole: &r}
}

func shareUser(id int64) model.ShareTarget {
	return model.ShareTarget{TargetType: model.ShareUser, TargetUserID: &id}
}

func ptr[T any](v T) *T { return &v }

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func wantErr(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("error = %v, want %v", err, target)
	}
}

func wantDuplicate(t *testing.T, err error, kind string, id int64) {
	t.Helper()
	var dup *model.DuplicateError
	if !errors.As(err, &dup) {
		t.Fatalf("error = %v, want *DuplicateError", err)
	}
	if dup.Kind != kind || dup.ExistingID != id {
		t.Fatalf("duplicate = %+v, want kind %q id %d", dup, kind, id)
	}
}

func countRows(t *testing.T, st *Store, query string, args ...any) int {
	t.Helper()
	var n int
	if err := st.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

func sameTime(a, b time.Time) bool { return a.Equal(b) }

func TestHelpers(t *testing.T) {
	t.Parallel()
	if got := truncateBytes("abécd", 3); got != "ab" {
		t.Errorf("truncateBytes = %q", got)
	}
	if got := truncateBytes("abc", 5); got != "abc" {
		t.Errorf("truncateBytes short = %q", got)
	}
	if got := truncateRunes("árvíz", 3); got != "árv" {
		t.Errorf("truncateRunes = %q", got)
	}
	if nullID(0) != nil || nullID(5) != int64(5) {
		t.Error("nullID")
	}
	if nullInt64(nil) != nil || nullInt64(ptr(int64(2))) != int64(2) {
		t.Error("nullInt64")
	}
	if optString(nil) != nil || optString(ptr("x")) != "x" || optBool(nil) != nil || optBool(ptr(true)) != true {
		t.Error("opt helpers")
	}
	if orDefault("", "a") != "a" || orDefault("b", "a") != "b" {
		t.Error("orDefault")
	}
	if userStatus("weird") != model.UserStatusAll || userStatus(model.UserStatusInactive) != model.UserStatusInactive {
		t.Error("userStatus")
	}
	if sortLabel("Host.COM", "", "") != "host.com" || sortLabel("h", "Title", "") != "title" ||
		sortLabel("h", "Title", "Desc") != "desc" {
		t.Error("sortLabel")
	}
	if got := searchText("U", "H", "T", "D"); got != "u\nh\nt\nd" {
		t.Errorf("searchText = %q", got)
	}
	if got := sortLabel("h", "", "Árvíztűrő Tükörfúrógép"); got != "arvizturo tukorfurogep" {
		t.Errorf("sortLabel folds accents: %q", got)
	}
	for in, want := range map[string]string{
		"": "", "ABC": "abc", "Ősz Øre Straße": "osz øre straße", "Crème Brûlée": "creme brulee", "한국어": "한국어",
	} {
		if got := foldText(in); got != want {
			t.Errorf("foldText(%q) = %q, want %q", in, got, want)
		}
	}
	if v, err := foldValue(nil, []driver.Value{"ÁGNES"}); err != nil || v != "agnes" {
		t.Errorf("foldValue string = %v, %v", v, err)
	}
	if v, _ := foldValue(nil, []driver.Value{[]byte("Ébred")}); v != "ebred" {
		t.Errorf("foldValue bytes = %v", v)
	}
	if v, _ := foldValue(nil, []driver.Value{int64(3)}); v != int64(3) {
		t.Errorf("foldValue int = %v", v)
	}
	v, err := lowerValue(nil, []driver.Value{"ÁGNES"})
	if err != nil || v != "ágnes" {
		t.Errorf("lowerValue string = %v, %v", v, err)
	}
	if v, _ := lowerValue(nil, []driver.Value{[]byte("AB")}); v != "ab" {
		t.Errorf("lowerValue bytes = %v", v)
	}
	if v, _ := lowerValue(nil, []driver.Value{int64(3)}); v != int64(3) {
		t.Errorf("lowerValue int = %v", v)
	}
	if !strings.Contains(visibleCTE, "o.role IN ('admin', 'developer')") {
		t.Error("visibleCTE lost the owner role rule")
	}
}
