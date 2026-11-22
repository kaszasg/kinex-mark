package store

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func hash64(c byte) string { return strings.Repeat(string(c), 64) }

func TestTOTPLifecycle(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)

	_, _, err := st.PendingTOTP(ctx, u.ID)
	wantErr(t, err, model.ErrNotFound)
	_, _, err = st.PendingTOTP(ctx, 9999)
	wantErr(t, err, model.ErrNotFound)
	wantErr(t, st.SetPendingTOTP(ctx, 9999, "v1:x", clk.Now()), model.ErrNotFound)

	expires := clk.Now().Add(10 * time.Minute)
	mustNoErr(t, st.SetPendingTOTP(ctx, u.ID, "v1:pending", expires))
	sealed, exp, err := st.PendingTOTP(ctx, u.ID)
	mustNoErr(t, err)
	if sealed != "v1:pending" || !exp.Equal(expires) {
		t.Errorf("PendingTOTP = %q %v", sealed, exp)
	}
	clk.Advance(10 * time.Minute)
	_, _, err = st.PendingTOTP(ctx, u.ID)
	wantErr(t, err, model.ErrNotFound)

	codes := []string{hash64('a'), hash64('b'), hash64('c')}
	mustNoErr(t, st.EnableTOTP(ctx, u.ID, "v1:secret", 100, codes))
	got, _ := st.UserByID(ctx, u.ID)
	if !got.TOTPEnabled {
		t.Fatal("TOTP not enabled")
	}
	c, _ := st.Credentials(ctx, got.UsernameKey)
	if c.TOTPSecret != "v1:secret" || c.TOTPLastStep != 100 {
		t.Errorf("credentials = %+v", c)
	}
	_, _, err = st.PendingTOTP(ctx, u.ID)
	wantErr(t, err, model.ErrNotFound)
	wantErr(t, st.EnableTOTP(ctx, u.ID, "v1:other", 1, nil), model.ErrConflict)
	wantErr(t, st.EnableTOTP(ctx, 9999, "v1:other", 1, nil), model.ErrNotFound)

	// Replay protection: only a newer step is accepted.
	for _, c := range []struct {
		step int64
		ok   bool
	}{{100, false}, {99, false}, {101, true}, {101, false}, {103, true}} {
		ok, err := st.AdvanceTOTPStep(ctx, u.ID, c.step)
		mustNoErr(t, err)
		if ok != c.ok {
			t.Errorf("AdvanceTOTPStep(%d) = %v, want %v", c.step, ok, c.ok)
		}
	}

	n, err := st.RecoveryCodesRemaining(ctx, u.ID)
	if err != nil || n != 3 {
		t.Fatalf("remaining = %d, %v", n, err)
	}
	ok, err := st.UseRecoveryCode(ctx, u.ID, hash64('a'))
	if err != nil || !ok {
		t.Fatalf("first use = %v, %v", ok, err)
	}
	ok, _ = st.UseRecoveryCode(ctx, u.ID, hash64('a'))
	if ok {
		t.Error("recovery code used twice")
	}
	ok, _ = st.UseRecoveryCode(ctx, u.ID, hash64('z'))
	if ok {
		t.Error("unknown recovery code accepted")
	}
	if n, _ := st.RecoveryCodesRemaining(ctx, u.ID); n != 2 {
		t.Errorf("remaining after use = %d", n)
	}

	mustNoErr(t, st.ReplaceRecoveryCodes(ctx, u.ID, []string{hash64('d'), hash64('d'), hash64('e')}))
	if n, _ := st.RecoveryCodesRemaining(ctx, u.ID); n != 2 {
		t.Errorf("remaining after replace = %d", n)
	}
	wantErr(t, st.ReplaceRecoveryCodes(ctx, 9999, nil), model.ErrNotFound)

	mustNoErr(t, st.SetPendingTOTP(ctx, u.ID, "v1:again", clk.Now().Add(time.Minute)))
	mustNoErr(t, st.DisableTOTP(ctx, u.ID))
	got, _ = st.UserByID(ctx, u.ID)
	c, _ = st.Credentials(ctx, got.UsernameKey)
	if got.TOTPEnabled || c.TOTPSecret != "" || c.TOTPLastStep != 0 {
		t.Errorf("after disable = %+v %+v", got, c)
	}
	if n, _ := st.RecoveryCodesRemaining(ctx, u.ID); n != 0 {
		t.Errorf("recovery codes survive disable: %d", n)
	}
	_, _, err = st.PendingTOTP(ctx, u.ID)
	wantErr(t, err, model.ErrNotFound)
	mustNoErr(t, st.DisableTOTP(ctx, u.ID)) // idempotent
	wantErr(t, st.DisableTOTP(ctx, 9999), model.ErrNotFound)
}

func TestSessions(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	other := seedUser(t, st, model.RoleCustomer)

	long := strings.Repeat("é", 200) // 400 bytes
	s1, err := st.CreateSession(ctx, model.NewSession{TokenHash: make32(1), UserID: u.ID,
		ExpiresAt: clk.Now().Add(168 * time.Hour), IP: "203.0.113.5", UserAgent: long})
	mustNoErr(t, err)
	if s1.ID == 0 || s1.IP != "203.0.113.5" || len(s1.UserAgent) > model.SessionUserAgentMaxLen ||
		!s1.CreatedAt.Equal(clk.Now()) || !s1.LastSeenAt.Equal(clk.Now()) {
		t.Errorf("session = %+v (ua %d bytes)", s1, len(s1.UserAgent))
	}
	clk.Advance(time.Minute)
	s2, err := st.CreateSession(ctx, model.NewSession{TokenHash: make32(2), UserID: u.ID,
		ExpiresAt: clk.Now().Add(time.Hour)})
	mustNoErr(t, err)
	_, err = st.CreateSession(ctx, model.NewSession{TokenHash: make32(3), UserID: other.ID,
		ExpiresAt: clk.Now().Add(time.Hour)})
	mustNoErr(t, err)
	_, err = st.CreateSession(ctx, model.NewSession{TokenHash: make32(1), UserID: u.ID,
		ExpiresAt: clk.Now().Add(time.Hour)})
	if err == nil {
		t.Error("duplicate token hash accepted")
	}
	_, err = st.CreateSession(ctx, model.NewSession{TokenHash: []byte("short"), UserID: u.ID,
		ExpiresAt: clk.Now()})
	if err == nil {
		t.Error("token hash length CHECK not enforced")
	}

	sess, user, err := st.SessionByToken(ctx, make32(2))
	mustNoErr(t, err)
	if sess.ID != s2.ID || user.ID != u.ID || sess.UserID != u.ID {
		t.Errorf("SessionByToken = %+v %+v", sess, user)
	}
	_, _, err = st.SessionByToken(ctx, make32(9))
	wantErr(t, err, model.ErrNotFound)

	clk.Advance(time.Minute)
	mustNoErr(t, st.TouchSession(ctx, s1.ID))
	sess, _, _ = st.SessionByToken(ctx, make32(1))
	if !sess.LastSeenAt.Equal(clk.Now()) {
		t.Errorf("TouchSession: last seen %v", sess.LastSeenAt)
	}
	wantErr(t, st.TouchSession(ctx, 9999), model.ErrNotFound)

	list, err := st.Sessions(ctx, u.ID)
	mustNoErr(t, err)
	if len(list) != 2 || list[0].ID != s2.ID || list[1].ID != s1.ID {
		t.Errorf("Sessions newest first = %+v", list)
	}
	clk.Advance(2 * time.Hour) // s2 is past its absolute expiry
	list, _ = st.Sessions(ctx, u.ID)
	if len(list) != 1 || list[0].ID != s1.ID {
		t.Errorf("Sessions without expired = %+v", list)
	}

	// Another user's session id is not found.
	wantErr(t, st.DeleteSession(ctx, other.ID, s1.ID), model.ErrNotFound)
	mustNoErr(t, st.DeleteSession(ctx, u.ID, s2.ID))
	wantErr(t, st.DeleteSession(ctx, u.ID, s2.ID), model.ErrNotFound)

	mustNoErr(t, st.DeleteSessionByToken(ctx, make32(1)))
	mustNoErr(t, st.DeleteSessionByToken(ctx, make32(1))) // no row: not an error
	_, _, err = st.SessionByToken(ctx, make32(1))
	wantErr(t, err, model.ErrNotFound)

	for i := byte(10); i < 14; i++ {
		_, err := st.CreateSession(ctx, model.NewSession{TokenHash: make32(i), UserID: u.ID,
			ExpiresAt: clk.Now().Add(time.Hour)})
		mustNoErr(t, err)
	}
	keep, _, _ := st.SessionByToken(ctx, make32(10))
	n, err := st.DeleteUserSessions(ctx, u.ID, keep.ID)
	if err != nil || n != 3 {
		t.Errorf("DeleteUserSessions except = %d, %v", n, err)
	}
	n, _ = st.DeleteUserSessions(ctx, u.ID, 0)
	if n != 1 {
		t.Errorf("DeleteUserSessions all = %d", n)
	}
	if c := countRows(t, st, `SELECT COUNT(*) FROM sessions WHERE user_id = ?`, other.ID); c != 1 {
		t.Errorf("other user's sessions = %d", c)
	}
}

func TestChallenges(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)

	mustNoErr(t, st.CreateChallenge(ctx, make32(1), u.ID, "198.51.100.7", clk.Now().Add(5*time.Minute)))
	mustNoErr(t, st.CreateChallenge(ctx, make32(2), u.ID, "198.51.100.7", clk.Now().Add(5*time.Minute)))
	// One challenge per user: the first one is gone.
	_, err := st.ChallengeAttempt(ctx, make32(1), 5)
	wantErr(t, err, model.ErrNotFound)

	for i := 1; i <= 5; i++ {
		c, err := st.ChallengeAttempt(ctx, make32(2), 5)
		mustNoErr(t, err)
		if c.UserID != u.ID || c.Attempts != i || !c.ExpiresAt.Equal(clk.Now().Add(5*time.Minute)) {
			t.Fatalf("attempt %d = %+v", i, c)
		}
	}
	_, err = st.ChallengeAttempt(ctx, make32(2), 5)
	wantErr(t, err, model.ErrNotFound)

	mustNoErr(t, st.CreateChallenge(ctx, make32(3), u.ID, "", clk.Now().Add(5*time.Minute)))
	clk.Advance(5 * time.Minute)
	_, err = st.ChallengeAttempt(ctx, make32(3), 5)
	wantErr(t, err, model.ErrNotFound)

	mustNoErr(t, st.CreateChallenge(ctx, make32(4), u.ID, "", clk.Now().Add(5*time.Minute)))
	mustNoErr(t, st.DeleteChallenge(ctx, make32(4)))
	_, err = st.ChallengeAttempt(ctx, make32(4), 5)
	wantErr(t, err, model.ErrNotFound)
	mustNoErr(t, st.DeleteChallenge(ctx, make32(4)))
}

func TestChallengeAttemptConcurrent(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	mustNoErr(t, st.CreateChallenge(ctx, make32(7), u.ID, "", clk.Now().Add(5*time.Minute)))

	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, notFound := 0, 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := st.ChallengeAttempt(ctx, make32(7), 5)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case errors.Is(err, model.ErrNotFound):
				notFound++
			default:
				t.Errorf("ChallengeAttempt: %v", err)
			}
		}()
	}
	wg.Wait()
	if ok > 5 || ok+notFound != 10 {
		t.Fatalf("reservations: %d ok, %d not found", ok, notFound)
	}
	if ok != 5 {
		t.Errorf("reservations = %d, want exactly 5", ok)
	}
}

func TestThrottleUpdate(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	uKey := model.ThrottleKey{Key: "u:" + hash64('f'), UserID: &u.ID}
	ipKey := model.ThrottleKey{Key: "ip:203.0.113.5"}

	// A check-only pass on unknown keys stores nothing.
	var seen []model.ThrottleRow
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{uKey, ipKey},
		func(r model.ThrottleRow) (model.ThrottleRow, error) {
			seen = append(seen, r)
			return r, nil
		}))
	if len(seen) != 2 || seen[0].Key != uKey.Key || seen[0].UserID == nil || *seen[0].UserID != u.ID ||
		seen[1].UserID != nil || seen[0].Failures != 0 || !seen[0].WindowStart.Equal(clk.Now()) {
		t.Fatalf("fresh rows = %+v", seen)
	}
	if n := countRows(t, st, `SELECT COUNT(*) FROM login_throttle`); n != 0 {
		t.Fatalf("check-only pass stored %d rows", n)
	}

	reserve := func(r model.ThrottleRow) (model.ThrottleRow, error) {
		r.Failures++
		return r, nil
	}
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{uKey, ipKey}, reserve))
	clk.Advance(time.Minute)
	lock := clk.Now().Add(15 * time.Minute)
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: uKey.Key}, ipKey},
		func(r model.ThrottleRow) (model.ThrottleRow, error) {
			r.Failures++
			if r.Key == uKey.Key {
				r.LockedUntil = &lock
				r.Lockouts = 1
			}
			return r, nil
		}))
	var got model.ThrottleRow
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: uKey.Key}},
		func(r model.ThrottleRow) (model.ThrottleRow, error) {
			got = r
			return r, nil
		}))
	if got.Failures != 2 || got.Lockouts != 1 || got.LockedUntil == nil || !got.LockedUntil.Equal(lock) ||
		got.UserID == nil || *got.UserID != u.ID || !got.UpdatedAt.Equal(clk.Now()) {
		t.Errorf("stored row = %+v (user id kept from the first write)", got)
	}

	// fn errors roll back every key.
	boom := errors.New("blocked")
	err := st.ThrottleUpdate(ctx, []model.ThrottleKey{ipKey, uKey},
		func(r model.ThrottleRow) (model.ThrottleRow, error) {
			if r.Key == uKey.Key {
				return r, boom
			}
			r.Failures = 99
			return r, nil
		})
	if !errors.Is(err, boom) {
		t.Fatalf("fn error = %v", err)
	}
	if n := countRows(t, st, `SELECT failures FROM login_throttle WHERE key = ?`, ipKey.Key); n != 2 {
		t.Errorf("ip failures after rollback = %d, want 2", n)
	}

	mustNoErr(t, st.ClearThrottle(ctx, ipKey.Key))
	if n := countRows(t, st, `SELECT COUNT(*) FROM login_throttle WHERE key = ?`, ipKey.Key); n != 0 {
		t.Errorf("ClearThrottle left %d rows", n)
	}
	mustNoErr(t, st.ClearUserThrottle(ctx, u.ID))
	if n := countRows(t, st, `SELECT COUNT(*) FROM login_throttle`); n != 0 {
		t.Errorf("ClearUserThrottle left %d rows", n)
	}

	// The CHECK on the key length holds.
	err = st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: strings.Repeat("k", 101)}}, reserve)
	if err == nil {
		t.Error("key length CHECK not enforced")
	}
}

func TestThrottleUpdateConcurrent(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	keys := []model.ThrottleKey{{Key: "u:concurrent"}, {Key: "ip:192.0.2.1"}}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := st.ThrottleUpdate(ctx, keys, func(r model.ThrottleRow) (model.ThrottleRow, error) {
				r.Failures++
				return r, nil
			})
			if err != nil {
				t.Errorf("ThrottleUpdate: %v", err)
			}
		}()
	}
	wg.Wait()
	for _, k := range keys {
		if n := countRows(t, st, `SELECT failures FROM login_throttle WHERE key = ?`, k.Key); n != 20 {
			t.Errorf("%s failures = %d, want exactly 20 (no lost reservation)", k.Key, n)
		}
	}
}
