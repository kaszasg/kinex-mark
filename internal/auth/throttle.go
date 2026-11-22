package auth

import (
	"net/netip"
	"time"
)

// ThrottleState is the counter of one login_throttle row.
type ThrottleState struct {
	Failures    int        // reserved or confirmed failures in the current window
	WindowStart time.Time  // start of the current counting window
	LockedUntil *time.Time // set while (or after) the key is locked
	Lockouts    int        // number of locks so far, drives escalation
}

// ThrottleRule is the policy of one throttle key family (plan 6.7).
type ThrottleRule struct {
	MaxFailures int           // failures per window that trigger a lock
	Window      time.Duration // counting window, measured from WindowStart
	LockFor     time.Duration // first lock duration
	MaxLock     time.Duration // upper bound of an escalated lock
	Escalate    bool          // double the lock duration on every new lock
}

// UserRule throttles a username key: 5 failures in 15 minutes lock it for
// 15 minutes, every further lock doubles the duration up to 24 hours.
var UserRule = ThrottleRule{5, 15 * time.Minute, 15 * time.Minute, 24 * time.Hour, true}

// IPRule throttles a client address key: 30 failures in 15 minutes lock it
// for 15 minutes, without escalation.
var IPRule = ThrottleRule{30, 15 * time.Minute, 15 * time.Minute, 15 * time.Minute, false}

// LockoutResetAfter is the quiet period after which the lockout counter (and
// with it the escalation) starts again from zero. The period runs from the
// later of the end of the last counting window (an upper bound of the last
// failure) and the end of the last lock.
const LockoutResetAfter = 24 * time.Hour

func (r ThrottleRule) locked(s ThrottleState, now time.Time) bool {
	return s.LockedUntil != nil && now.Before(*s.LockedUntil)
}

func (r ThrottleRule) windowEnd(s ThrottleState) time.Time {
	return s.WindowStart.Add(r.Window)
}

func (r ThrottleRule) windowOpen(s ThrottleState, now time.Time) bool {
	return !s.WindowStart.IsZero() && now.Before(r.windowEnd(s))
}

// Check reports whether a new attempt must be refused: the key is locked, or
// the current window already holds MaxFailures attempts (reserved or failed).
// retryAfter is the time until the lock or the window ends.
func (r ThrottleRule) Check(s ThrottleState, now time.Time) (blocked bool, retryAfter time.Duration) {
	if r.locked(s, now) {
		return true, s.LockedUntil.Sub(now)
	}
	if r.windowOpen(s, now) && s.Failures >= r.MaxFailures {
		return true, r.windowEnd(s).Sub(now)
	}
	return false, 0
}

// Reserve counts an attempt as a failure before the credential is verified.
// An expired window or lock starts a new window first, and after
// LockoutResetAfter without failures the lockout counter returns to zero.
func (r ThrottleRule) Reserve(s ThrottleState, now time.Time) ThrottleState {
	if s.Lockouts > 0 {
		quietFrom := r.windowEnd(s)
		if s.LockedUntil != nil && s.LockedUntil.After(quietFrom) {
			quietFrom = *s.LockedUntil
		}
		if !now.Before(quietFrom.Add(LockoutResetAfter)) {
			s.Lockouts = 0
		}
	}
	lockExpired := s.LockedUntil != nil && !now.Before(*s.LockedUntil)
	if lockExpired {
		s.LockedUntil = nil
	}
	if lockExpired || !r.windowOpen(s, now) {
		s.Failures = 0
		s.WindowStart = now
	}
	s.Failures++
	return s
}

// Confirm records a verified failure. The reservation already counted it, so
// Confirm only acts when the window holds MaxFailures failures: it locks the
// key (escalating with every lock when the rule says so) and starts a new,
// empty window. A key that is already locked is left unchanged, so concurrent
// confirmations do not stack locks.
func (r ThrottleRule) Confirm(s ThrottleState, now time.Time) ThrottleState {
	if r.locked(s, now) || s.Failures < r.MaxFailures {
		return s
	}
	until := now.Add(r.lockDuration(s.Lockouts))
	s.LockedUntil = &until
	s.Lockouts++
	s.Failures = 0
	s.WindowStart = now
	return s
}

// Refund returns the reservation of a verified success: failures - 1, never
// below zero. It does not lift a lock.
func (r ThrottleRule) Refund(s ThrottleState) ThrottleState {
	if s.Failures > 0 {
		s.Failures--
	}
	return s
}

// lockDuration returns the length of the next lock after previous locks:
// LockFor, doubled per previous lock when escalating, capped at MaxLock.
func (r ThrottleRule) lockDuration(previous int) time.Duration {
	d := r.LockFor
	if r.Escalate {
		for i := 0; i < previous && d < r.MaxLock; i++ {
			d *= 2
		}
	}
	if r.MaxLock > 0 && d > r.MaxLock {
		d = r.MaxLock
	}
	return d
}

// throttleUsernamePurpose is the MAC purpose of username throttle keys.
const throttleUsernamePurpose = "throttle-username"

// UserThrottleKey returns the throttle key of a username key (the lower-cased
// username): "u:" + 64 hex characters. The value is the same for existing and
// unknown names, and the typed name cannot be read back from it.
func UserThrottleKey(k *Keyring, usernameKey string) string {
	return "u:" + k.MAC(throttleUsernamePurpose, []byte(usernameKey))
}

// IPThrottleKey returns the throttle key of a client address. IPv4-mapped
// IPv6 addresses are unmapped first; an IPv4 address keys on itself (/32), an
// IPv6 address on its /64 prefix, for example "ip:2001:db8:1:2::/64".
func IPThrottleKey(ip netip.Addr) string {
	if !ip.IsValid() {
		return "ip:invalid"
	}
	ip = ip.Unmap().WithZone("")
	if ip.Is4() {
		return "ip:" + ip.String()
	}
	prefix, _ := ip.Prefix(64) // cannot fail: a valid IPv6 address without zone
	return "ip:" + prefix.String()
}
