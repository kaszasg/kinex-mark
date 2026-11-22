package storetest

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/store"
)

// SeedPasswordHash is the password hash of seeded users. It is not a valid
// auth.Hasher encoding, so no password signs a seeded user in; tests that
// sign in with a password create the user with SeedNewUser and a real hash.
const SeedPasswordHash = "seed-no-password"

var seq atomic.Int64

// SeedUser creates an active user of the given role with a unique username
// ("<role>-<n>") and acknowledges the current privacy notice, so that the
// user passes the password-change and consent gates (6.2).
func SeedUser(t testing.TB, st *store.Store, role model.Role) model.User {
	t.Helper()
	u := SeedNewUser(t, st, model.NewUser{
		Username:     fmt.Sprintf("%s-%d", role, seq.Add(1)),
		Role:         role,
		PasswordHash: SeedPasswordHash,
	})
	ctx := context.Background()
	settings, err := st.Settings(ctx)
	if err != nil {
		t.Fatalf("storetest: read settings: %v", err)
	}
	if err := st.RecordConsent(ctx, u.ID, model.ConsentPrivacyNotice, settings.PrivacyNoticeVersion); err != nil {
		t.Fatalf("storetest: record consent: %v", err)
	}
	return u
}

// SeedNewUser creates a user from n as given (no consent is recorded).
func SeedNewUser(t testing.TB, st *store.Store, n model.NewUser) model.User {
	t.Helper()
	u, err := st.CreateUser(context.Background(), n)
	if err != nil {
		t.Fatalf("storetest: create user %q: %v", n.Username, err)
	}
	return u
}
