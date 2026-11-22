package model

import "time"

// Session is a server-side sign-in session (5.5 Session without the
// computed "current" flag). The token itself is never stored, only its hash.
type Session struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	ExpiresAt  time.Time `json:"expiresAt"` // absolute expiry; idle expiry is derived from LastSeenAt (4.3)
	IP         string    `json:"ip"`
	UserAgent  string    `json:"userAgent"`
}

// Session column limits (4.3).
const (
	SessionIPMaxLen        = 45
	SessionUserAgentMaxLen = 255
)

// NewSession is the input of Store.CreateSession.
type NewSession struct {
	TokenHash []byte // SHA-256 of the session token, 32 bytes
	UserID    int64
	ExpiresAt time.Time // absolute expiry
	IP        string    // at most SessionIPMaxLen bytes
	UserAgent string    // truncated to SessionUserAgentMaxLen bytes
}

// Challenge is a pending second sign-in step (login_challenges).
type Challenge struct {
	UserID    int64
	Attempts  int // attempts used, including the one just reserved
	ExpiresAt time.Time
}

// ThrottleRow is a login_throttle row (4.3, 6.7).
type ThrottleRow struct {
	Key         string
	UserID      *int64 // set on the "u:" row of an existing account
	Failures    int
	WindowStart time.Time
	LockedUntil *time.Time
	Lockouts    int
	UpdatedAt   time.Time
}

// ThrottleKey selects a throttle row for Store.ThrottleUpdate. UserID is set
// only for the "u:" key of an existing account.
type ThrottleKey struct {
	Key    string
	UserID *int64
}
