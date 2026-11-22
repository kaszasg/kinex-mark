package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/kaszasg/kinex-mark/internal/model"
)

const sessionColumns = `se.id, se.user_id, se.created_at, se.last_seen_at, se.expires_at, se.ip, se.user_agent`

type sessionScan struct {
	s                               model.Session
	createdAt, lastSeenAt, expireAt string
}

func (ss *sessionScan) dest() []any {
	return []any{&ss.s.ID, &ss.s.UserID, &ss.createdAt, &ss.lastSeenAt, &ss.expireAt,
		&ss.s.IP, &ss.s.UserAgent}
}

func (ss *sessionScan) session() (model.Session, error) {
	out := ss.s
	var err error
	if out.CreatedAt, err = parseTime(ss.createdAt); err != nil {
		return model.Session{}, err
	}
	if out.LastSeenAt, err = parseTime(ss.lastSeenAt); err != nil {
		return model.Session{}, err
	}
	if out.ExpiresAt, err = parseTime(ss.expireAt); err != nil {
		return model.Session{}, err
	}
	return out, nil
}

// truncateBytes shortens s to at most n bytes on a rune boundary.
func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// CreateSession stores a new session (only the token hash is stored).
func (s *Store) CreateSession(ctx context.Context, n model.NewSession) (model.Session, error) {
	now := formatTime(s.now())
	var out model.Session
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `INSERT INTO sessions (token_hash, user_id, created_at,
				last_seen_at, expires_at, ip, user_agent)
			VALUES (:hash, :uid, :now, :now, :expires, :ip, :ua)`,
			sql.Named("hash", n.TokenHash), sql.Named("uid", n.UserID), sql.Named("now", now),
			sql.Named("expires", formatTime(n.ExpiresAt)),
			sql.Named("ip", truncateBytes(n.IP, model.SessionIPMaxLen)),
			sql.Named("ua", truncateBytes(n.UserAgent, model.SessionUserAgentMaxLen)))
		if err != nil {
			return fmt.Errorf("insert session: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("insert session: %w", err)
		}
		var ss sessionScan
		if err := tx.QueryRowContext(ctx, `SELECT `+sessionColumns+` FROM sessions se WHERE se.id = :id`,
			sql.Named("id", id)).Scan(ss.dest()...); err != nil {
			return fmt.Errorf("read session: %w", err)
		}
		out, err = ss.session()
		return err
	})
	if err != nil {
		return model.Session{}, err
	}
	return out, nil
}

// SessionByToken loads a session and its user with one query; it returns
// model.ErrNotFound for an unknown token hash. Expiry is checked by the caller
// against the current settings (6.2).
func (s *Store) SessionByToken(ctx context.Context, hash []byte) (model.Session, model.User, error) {
	var ss sessionScan
	var us userScan
	dest := append(ss.dest(), us.dest()...)
	err := s.db.QueryRowContext(ctx, `SELECT `+sessionColumns+`, `+userColumns+`
		FROM sessions se JOIN users u ON u.id = se.user_id
		WHERE se.token_hash = :hash`, sql.Named("hash", hash)).Scan(dest...)
	if err != nil {
		return model.Session{}, model.User{}, notFound(err)
	}
	sess, err := ss.session()
	if err != nil {
		return model.Session{}, model.User{}, err
	}
	u, err := us.user()
	if err != nil {
		return model.Session{}, model.User{}, err
	}
	return sess, u, nil
}

// TouchSession records activity on a session (last_seen_at = now).
func (s *Store) TouchSession(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = :now WHERE id = :id`,
		sql.Named("now", formatTime(s.now())), sql.Named("id", id))
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return affectedOne(res)
}

// Sessions lists the user's sessions that have not reached their absolute
// expiry, newest first.
func (s *Store) Sessions(ctx context.Context, userID int64) ([]model.Session, error) {
	return sessionsQ(ctx, s.db, userID, s.now())
}

func sessionsQ(ctx context.Context, q querier, userID int64, now time.Time) ([]model.Session, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+sessionColumns+` FROM sessions se
		WHERE se.user_id = :uid AND se.expires_at > :now
		ORDER BY se.created_at DESC, se.id DESC`,
		sql.Named("uid", userID), sql.Named("now", formatTime(now)))
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	items := []model.Session{}
	for rows.Next() {
		var ss sessionScan
		if err := rows.Scan(ss.dest()...); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sess, err := ss.session()
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		items = append(items, sess)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return items, nil
}

// DeleteSession deletes one of the user's sessions; model.ErrNotFound when
// the session does not exist or belongs to someone else.
func (s *Store) DeleteSession(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = :id AND user_id = :uid`,
		sql.Named("id", id), sql.Named("uid", userID))
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return affectedOne(res)
}

// DeleteSessionByToken deletes the session with the token hash, if any.
func (s *Store) DeleteSessionByToken(ctx context.Context, hash []byte) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = :hash`,
		sql.Named("hash", hash)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteUserSessions deletes the user's sessions except exceptID (0: all)
// and returns how many were deleted.
func (s *Store) DeleteUserSessions(ctx context.Context, userID, exceptID int64) (int, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = :uid AND id <> :except`,
		sql.Named("uid", userID), sql.Named("except", exceptID))
	if err != nil {
		return 0, fmt.Errorf("delete sessions: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete sessions: %w", err)
	}
	return int(n), nil
}

// CreateChallenge stores a second-step sign-in challenge and deletes the
// user's other open challenges, so that one challenge per user is alive.
func (s *Store) CreateChallenge(ctx context.Context, hash []byte, userID int64, ip string, expires time.Time) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM login_challenges WHERE user_id = :uid`,
			sql.Named("uid", userID)); err != nil {
			return fmt.Errorf("delete challenges: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO login_challenges (token_hash, user_id, ip,
				created_at, expires_at)
			VALUES (:hash, :uid, :ip, :now, :expires)`,
			sql.Named("hash", hash), sql.Named("uid", userID),
			sql.Named("ip", truncateBytes(ip, model.SessionIPMaxLen)),
			sql.Named("now", formatTime(s.now())), sql.Named("expires", formatTime(expires))); err != nil {
			return fmt.Errorf("insert challenge: %w", err)
		}
		return nil
	})
}

// ChallengeAttempt reserves one attempt atomically before verification:
//
//	UPDATE login_challenges SET attempts = attempts + 1
//	 WHERE token_hash = :h AND attempts < :max AND expires_at > :now RETURNING user_id, attempts.
//
// It returns model.ErrNotFound when the challenge is unknown, expired or out
// of attempts.
func (s *Store) ChallengeAttempt(ctx context.Context, hash []byte, maxAttempts int) (model.Challenge, error) {
	var c model.Challenge
	var expires string
	err := s.db.QueryRowContext(ctx, `UPDATE login_challenges SET attempts = attempts + 1
		WHERE token_hash = :hash AND attempts < :max AND expires_at > :now
		RETURNING user_id, attempts, expires_at`,
		sql.Named("hash", hash), sql.Named("max", maxAttempts),
		sql.Named("now", formatTime(s.now()))).Scan(&c.UserID, &c.Attempts, &expires)
	if err != nil {
		return model.Challenge{}, notFound(err)
	}
	if c.ExpiresAt, err = parseTime(expires); err != nil {
		return model.Challenge{}, err
	}
	return c, nil
}

// DeleteChallenge deletes a challenge, if it exists.
func (s *Store) DeleteChallenge(ctx context.Context, hash []byte) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM login_challenges WHERE token_hash = :hash`,
		sql.Named("hash", hash)); err != nil {
		return fmt.Errorf("delete challenge: %w", err)
	}
	return nil
}
