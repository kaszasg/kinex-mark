package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

func TestNewStoreAndSeed(t *testing.T) {
	clk := testutil.NewClock()
	clk.Advance(time.Hour)
	st := NewStore(t, WithClock(clk), WithDir(t.TempDir()))
	ctx := context.Background()

	a := SeedUser(t, st, model.RoleAdmin)
	b := SeedUser(t, st, model.RoleAdmin)
	if a.Username == b.Username || a.Role != model.RoleAdmin || !a.Active || a.MustChangePassword {
		t.Fatalf("seeded users = %+v, %+v", a, b)
	}
	if !a.CreatedAt.Equal(clk.Now()) {
		t.Errorf("CreatedAt = %v, want the injected clock %v", a.CreatedAt, clk.Now())
	}
	consents, err := st.Consents(ctx, a.ID)
	if err != nil || len(consents) != 1 || consents[0].Version != 1 {
		t.Errorf("consents = %+v, %v", consents, err)
	}

	c := SeedNewUser(t, st, model.NewUser{Username: "plain", Role: model.RoleCustomer, PasswordHash: "h"})
	consents, _ = st.Consents(ctx, c.ID)
	if len(consents) != 0 {
		t.Errorf("SeedNewUser recorded consent: %+v", consents)
	}
	if n, _ := st.CountUsers(ctx); n != 3 {
		t.Errorf("CountUsers = %d", n)
	}
}

func TestNewStoreDefaultClock(t *testing.T) {
	st := NewStore(t)
	u := SeedUser(t, st, model.RoleCustomer)
	if !u.CreatedAt.Equal(testutil.T0) {
		t.Errorf("CreatedAt = %v, want testutil.T0", u.CreatedAt)
	}
}
