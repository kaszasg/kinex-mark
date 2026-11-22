package auth

import (
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)

func at(d time.Duration) time.Time { return t0.Add(d) }

func ptr(t time.Time) *time.Time { return &t }

func TestThrottleRules(t *testing.T) {
	t.Parallel()
	if UserRule != (ThrottleRule{5, 15 * time.Minute, 15 * time.Minute, 24 * time.Hour, true}) {
		t.Fatalf("UserRule = %+v", UserRule)
	}
	if IPRule != (ThrottleRule{30, 15 * time.Minute, 15 * time.Minute, 15 * time.Minute, false}) {
		t.Fatalf("IPRule = %+v", IPRule)
	}
}

func TestThrottleCheck(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		state       ThrottleState
		now         time.Time
		wantBlocked bool
		wantRetry   time.Duration
	}{
		{"empty state", ThrottleState{}, t0, false, 0},
		{"below threshold", ThrottleState{Failures: 4, WindowStart: t0}, at(time.Minute), false, 0},
		{"threshold reached by reservations", ThrottleState{Failures: 5, WindowStart: t0}, at(time.Minute), true, 14 * time.Minute},
		{"above threshold", ThrottleState{Failures: 7, WindowStart: t0}, at(10 * time.Minute), true, 5 * time.Minute},
		{"window expired", ThrottleState{Failures: 5, WindowStart: t0}, at(15 * time.Minute), false, 0},
		{"locked", ThrottleState{WindowStart: t0, LockedUntil: ptr(at(15 * time.Minute)), Lockouts: 1}, at(5 * time.Minute), true, 10 * time.Minute},
		{"lock expired", ThrottleState{WindowStart: t0, LockedUntil: ptr(at(15 * time.Minute)), Lockouts: 1}, at(15 * time.Minute), false, 0},
		{"zero window start with failures", ThrottleState{Failures: 9}, t0, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			blocked, retry := UserRule.Check(c.state, c.now)
			if blocked != c.wantBlocked || retry != c.wantRetry {
				t.Fatalf("Check = %v, %v; want %v, %v", blocked, retry, c.wantBlocked, c.wantRetry)
			}
		})
	}
}

func TestThrottleReserve(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		state ThrottleState
		now   time.Time
		want  ThrottleState
	}{
		{"first attempt", ThrottleState{}, t0, ThrottleState{Failures: 1, WindowStart: t0}},
		{"inside window", ThrottleState{Failures: 2, WindowStart: t0}, at(14 * time.Minute), ThrottleState{Failures: 3, WindowStart: t0}},
		{"window expired resets first", ThrottleState{Failures: 4, WindowStart: t0}, at(15 * time.Minute), ThrottleState{Failures: 1, WindowStart: at(15 * time.Minute)}},
		{
			"expired lock is cleared",
			ThrottleState{WindowStart: t0, LockedUntil: ptr(at(15 * time.Minute)), Lockouts: 1},
			at(20 * time.Minute),
			ThrottleState{Failures: 1, WindowStart: at(20 * time.Minute), Lockouts: 1},
		},
		{
			"active lock is kept",
			ThrottleState{Failures: 0, WindowStart: t0, LockedUntil: ptr(at(15 * time.Minute)), Lockouts: 1},
			at(time.Minute),
			ThrottleState{Failures: 1, WindowStart: t0, LockedUntil: ptr(at(15 * time.Minute)), Lockouts: 1},
		},
		{
			"lockouts kept within 24 hours after the lock",
			ThrottleState{WindowStart: t0, LockedUntil: ptr(at(2 * time.Hour)), Lockouts: 3},
			at(26*time.Hour - time.Second),
			ThrottleState{Failures: 1, WindowStart: at(26*time.Hour - time.Second), Lockouts: 3},
		},
		{
			"lockouts reset 24 hours after the lock",
			ThrottleState{WindowStart: t0, LockedUntil: ptr(at(2 * time.Hour)), Lockouts: 3},
			at(26 * time.Hour),
			ThrottleState{Failures: 1, WindowStart: at(26 * time.Hour)},
		},
		{
			"lockouts kept within 24 hours after the window",
			ThrottleState{Failures: 2, WindowStart: t0, Lockouts: 2},
			at(24*time.Hour + 14*time.Minute),
			ThrottleState{Failures: 1, WindowStart: at(24*time.Hour + 14*time.Minute), Lockouts: 2},
		},
		{
			"lockouts reset 24 hours after the window",
			ThrottleState{Failures: 2, WindowStart: t0, Lockouts: 2},
			at(24*time.Hour + 15*time.Minute),
			ThrottleState{Failures: 1, WindowStart: at(24*time.Hour + 15*time.Minute)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertState(t, UserRule.Reserve(c.state, c.now), c.want)
		})
	}
}

func assertState(t *testing.T, got, want ThrottleState) {
	t.Helper()
	lockEqual := (got.LockedUntil == nil) == (want.LockedUntil == nil) &&
		(got.LockedUntil == nil || got.LockedUntil.Equal(*want.LockedUntil))
	if got.Failures != want.Failures || !got.WindowStart.Equal(want.WindowStart) || got.Lockouts != want.Lockouts || !lockEqual {
		t.Fatalf("state = %s, want %s", formatState(got), formatState(want))
	}
}

func formatState(s ThrottleState) string {
	lock := "nil"
	if s.LockedUntil != nil {
		lock = s.LockedUntil.Format(time.RFC3339)
	}
	return "{failures " + itoa(s.Failures) + ", window " + s.WindowStart.Format(time.RFC3339) +
		", locked " + lock + ", lockouts " + itoa(s.Lockouts) + "}"
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestThrottleConfirm(t *testing.T) {
	t.Parallel()
	now := at(5 * time.Minute)
	cases := []struct {
		name  string
		rule  ThrottleRule
		state ThrottleState
		want  ThrottleState
	}{
		{"below threshold keeps the count", UserRule, ThrottleState{Failures: 4, WindowStart: t0}, ThrottleState{Failures: 4, WindowStart: t0}},
		{"threshold locks", UserRule, ThrottleState{Failures: 5, WindowStart: t0}, ThrottleState{WindowStart: now, LockedUntil: ptr(now.Add(15 * time.Minute)), Lockouts: 1}},
		{"second lock doubles", UserRule, ThrottleState{Failures: 5, WindowStart: t0, Lockouts: 1}, ThrottleState{WindowStart: now, LockedUntil: ptr(now.Add(30 * time.Minute)), Lockouts: 2}},
		{"third lock doubles again", UserRule, ThrottleState{Failures: 5, WindowStart: t0, Lockouts: 2}, ThrottleState{WindowStart: now, LockedUntil: ptr(now.Add(time.Hour)), Lockouts: 3}},
		{"escalation capped at 24 hours", UserRule, ThrottleState{Failures: 5, WindowStart: t0, Lockouts: 7}, ThrottleState{WindowStart: now, LockedUntil: ptr(now.Add(24 * time.Hour)), Lockouts: 8}},
		{"huge lockout count stays capped", UserRule, ThrottleState{Failures: 5, WindowStart: t0, Lockouts: 1000}, ThrottleState{WindowStart: now, LockedUntil: ptr(now.Add(24 * time.Hour)), Lockouts: 1001}},
		{"ip rule does not escalate", IPRule, ThrottleState{Failures: 30, WindowStart: t0, Lockouts: 4}, ThrottleState{WindowStart: now, LockedUntil: ptr(now.Add(15 * time.Minute)), Lockouts: 5}},
		{
			"already locked is unchanged",
			UserRule,
			ThrottleState{Failures: 5, WindowStart: t0, LockedUntil: ptr(at(10 * time.Minute)), Lockouts: 1},
			ThrottleState{Failures: 5, WindowStart: t0, LockedUntil: ptr(at(10 * time.Minute)), Lockouts: 1},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertState(t, c.rule.Confirm(c.state, now), c.want)
		})
	}
}

func TestThrottleLockDuration(t *testing.T) {
	t.Parallel()
	want := []time.Duration{
		15 * time.Minute, 30 * time.Minute, time.Hour, 2 * time.Hour, 4 * time.Hour,
		8 * time.Hour, 16 * time.Hour, 24 * time.Hour, 24 * time.Hour,
	}
	for previous, w := range want {
		if got := UserRule.lockDuration(previous); got != w {
			t.Errorf("UserRule.lockDuration(%d) = %v, want %v", previous, got, w)
		}
	}
	uncapped := ThrottleRule{MaxFailures: 1, LockFor: time.Minute}
	if got := uncapped.lockDuration(9); got != time.Minute {
		t.Errorf("rule without MaxLock = %v, want %v", got, time.Minute)
	}
	shortCap := ThrottleRule{MaxFailures: 1, LockFor: time.Hour, MaxLock: time.Minute}
	if got := shortCap.lockDuration(0); got != time.Minute {
		t.Errorf("LockFor above MaxLock = %v, want %v", got, time.Minute)
	}
}

func TestThrottleRefund(t *testing.T) {
	t.Parallel()
	lock := ptr(at(15 * time.Minute))
	cases := []struct {
		name        string
		state, want ThrottleState
	}{
		{"decrements", ThrottleState{Failures: 3, WindowStart: t0}, ThrottleState{Failures: 2, WindowStart: t0}},
		{"not below zero", ThrottleState{WindowStart: t0}, ThrottleState{WindowStart: t0}},
		{"does not lift a lock", ThrottleState{Failures: 1, WindowStart: t0, LockedUntil: lock, Lockouts: 1}, ThrottleState{WindowStart: t0, LockedUntil: lock, Lockouts: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := UserRule.Refund(c.state)
			assertState(t, got, c.want)
			if blocked, _ := UserRule.Check(got, at(time.Minute)); c.state.LockedUntil != nil && !blocked {
				t.Fatal("a refund unlocked the key")
			}
		})
	}
}

// attempt runs one login attempt the way the api does: Check, Reserve, then
// Confirm (failure) or Refund (success). It reports whether the attempt was
// allowed to verify.
func attempt(rule ThrottleRule, s *ThrottleState, now time.Time, success bool) bool {
	if blocked, _ := rule.Check(*s, now); blocked {
		return false
	}
	*s = rule.Reserve(*s, now)
	if success {
		*s = rule.Refund(*s)
	} else {
		*s = rule.Confirm(*s, now)
	}
	return true
}

func TestThrottleReservationsBoundConcurrentAttempts(t *testing.T) {
	t.Parallel()
	// 20 simultaneous attempts: each reserves before any verification ends,
	// so only MaxFailures of them reach the password check.
	var s ThrottleState
	allowed := 0
	for range 20 {
		if blocked, _ := UserRule.Check(s, t0); !blocked {
			s = UserRule.Reserve(s, t0)
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("%d attempts passed the reservation, want 5", allowed)
	}
	for range allowed {
		s = UserRule.Confirm(s, t0)
	}
	if s.LockedUntil == nil || !s.LockedUntil.Equal(at(15*time.Minute)) || s.Lockouts != 1 {
		t.Fatalf("after 5 confirmed failures: %s, want a single 15 minute lock", formatState(s))
	}
}

func TestThrottleRefundFreesReservation(t *testing.T) {
	t.Parallel()
	var s ThrottleState
	for range 5 {
		s = UserRule.Reserve(s, t0)
	}
	if blocked, _ := UserRule.Check(s, t0); !blocked {
		t.Fatal("five reservations do not block")
	}
	s = UserRule.Refund(s)
	if blocked, _ := UserRule.Check(s, t0); blocked {
		t.Fatal("a refunded reservation still blocks")
	}
}

func TestThrottleEscalationScenario(t *testing.T) {
	t.Parallel()
	var s ThrottleState
	now := t0
	for lock, want := range []time.Duration{15 * time.Minute, 30 * time.Minute, time.Hour} {
		for i := range 5 {
			if !attempt(UserRule, &s, now, false) {
				t.Fatalf("lock %d: attempt %d was blocked early", lock, i)
			}
		}
		if s.LockedUntil == nil || s.LockedUntil.Sub(now) != want {
			t.Fatalf("lock %d: %s, want a lock of %v", lock, formatState(s), want)
		}
		if attempt(UserRule, &s, now.Add(want-time.Second), true) {
			t.Fatalf("lock %d: an attempt passed during the lock", lock)
		}
		now = now.Add(want)
	}
	// A success after the lock refunds its reservation; four more failures do
	// not lock again because the success did not count.
	if !attempt(UserRule, &s, now, true) {
		t.Fatal("attempt after the lock was blocked")
	}
	for range 4 {
		attempt(UserRule, &s, now, false)
	}
	if s.LockedUntil != nil {
		t.Fatalf("locked after 4 failures: %s", formatState(s))
	}
	// A day of silence resets the escalation.
	now = now.Add(LockoutResetAfter + UserRule.Window)
	for range 5 {
		attempt(UserRule, &s, now, false)
	}
	if s.Lockouts != 1 || s.LockedUntil == nil || s.LockedUntil.Sub(now) != 15*time.Minute {
		t.Fatalf("after a quiet day: %s, want a fresh 15 minute lock", formatState(s))
	}
}

func TestThrottleIPRuleThreshold(t *testing.T) {
	t.Parallel()
	var s ThrottleState
	for i := range 30 {
		if !attempt(IPRule, &s, at(time.Duration(i)*time.Second), false) {
			t.Fatalf("attempt %d blocked before the threshold", i)
		}
	}
	if blocked, retry := IPRule.Check(s, at(time.Minute)); !blocked || retry <= 0 {
		t.Fatalf("Check after 30 failures = %v, %v", blocked, retry)
	}
}

var userKeyFormat = regexp.MustCompile(`^u:[0-9a-f]{64}$`)

func TestUserThrottleKey(t *testing.T) {
	t.Parallel()
	kr := mustKeyring(t, 21)
	k := UserThrottleKey(kr, "alice")
	if !userKeyFormat.MatchString(k) || len(k) != 66 {
		t.Fatalf("key %q is not u: plus 64 hex characters", k)
	}
	if UserThrottleKey(kr, "alice") != k {
		t.Fatal("UserThrottleKey is not deterministic")
	}
	if strings.Contains(k, "alice") || strings.Contains(k, "616c696365") {
		t.Fatal("the username can be read from the key")
	}
	if k != "u:"+kr.MAC("throttle-username", []byte("alice")) {
		t.Fatal("the key is not the throttle-username MAC")
	}
	if UserThrottleKey(kr, "alicf") == k {
		t.Fatal("different usernames share a key")
	}
	if UserThrottleKey(mustKeyring(t, 22), "alice") == k {
		t.Fatal("the key does not depend on the server key")
	}
}

func TestIPThrottleKey(t *testing.T) {
	t.Parallel()
	key := func(s string) string { return IPThrottleKey(netip.MustParseAddr(s)) }
	cases := map[string]string{
		"1.2.3.4":              "ip:1.2.3.4",
		"::ffff:1.2.3.4":       "ip:1.2.3.4",
		"2001:db8::1":          "ip:2001:db8::/64",
		"2001:db8:1:2:3:4:5:6": "ip:2001:db8:1:2::/64",
		"fe80::1%eth0":         "ip:fe80::/64",
		"::1":                  "ip:::/64",
	}
	for in, want := range cases {
		if got := key(in); got != want {
			t.Errorf("IPThrottleKey(%s) = %q, want %q", in, got, want)
		}
	}
	if key("1.2.3.4") != key("::ffff:1.2.3.4") {
		t.Error("an IPv4-mapped address has its own key")
	}
	if key("1.2.3.4") == key("1.2.3.5") {
		t.Error("two IPv4 addresses share a key")
	}
	if key("2001:db8::1") != key("2001:db8::2") {
		t.Error("two addresses of one /64 have different keys")
	}
	if key("2001:db8:0:1::1") == key("2001:db8:0:2::1") {
		t.Error("two /64 prefixes share a key")
	}
	if got := IPThrottleKey(netip.Addr{}); got != "ip:invalid" {
		t.Errorf("IPThrottleKey(zero) = %q", got)
	}
	if long := key("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"); len(long) > 100 {
		t.Errorf("key %q exceeds the 100 character column limit", long)
	}
}
