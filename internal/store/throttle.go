package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// ThrottleUpdate reads, computes and writes the throttle rows of keys in one
// write transaction (6.7). fn receives the current row of each key in order
// (a fresh row with zero failures and WindowStart = now when the key has no
// row yet) and returns the row to store. An error from fn rolls the whole
// transaction back and is returned unchanged.
//
// The single database connection serialises the transactions, so concurrent
// callers always see each other's reservations.
func (s *Store) ThrottleUpdate(ctx context.Context, keys []model.ThrottleKey,
	fn func(row model.ThrottleRow) (model.ThrottleRow, error)) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		now := s.now()
		for _, k := range keys {
			row, existed, err := throttleRowQ(ctx, tx, k, now)
			if err != nil {
				return err
			}
			next, err := fn(row)
			if err != nil {
				return err
			}
			if !existed && next.Failures == 0 && next.LockedUntil == nil && next.Lockouts == 0 {
				// Nothing to remember for a key that has no row.
				continue
			}
			userID := next.UserID
			if userID == nil {
				userID = row.UserID
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO login_throttle (key, user_id, failures,
					window_started_at, locked_until, lockouts, updated_at)
				VALUES (:key, :uid, :failures, :window, :locked, :lockouts, :now)
				ON CONFLICT (key) DO UPDATE SET user_id = excluded.user_id,
					failures = excluded.failures, window_started_at = excluded.window_started_at,
					locked_until = excluded.locked_until, lockouts = excluded.lockouts,
					updated_at = excluded.updated_at`,
				sql.Named("key", k.Key), sql.Named("uid", nullInt64(userID)),
				sql.Named("failures", next.Failures), sql.Named("window", formatTime(next.WindowStart)),
				sql.Named("locked", formatNullTime(next.LockedUntil)),
				sql.Named("lockouts", next.Lockouts), sql.Named("now", formatTime(now)))
			if err != nil {
				return fmt.Errorf("write throttle row: %w", err)
			}
		}
		return nil
	})
}

// throttleRowQ reads the row of k, or returns a fresh one (existed false).
func throttleRowQ(ctx context.Context, q querier, k model.ThrottleKey, now time.Time) (model.ThrottleRow, bool, error) {
	var uid sql.NullInt64
	var window, updated string
	var locked sql.NullString
	row := model.ThrottleRow{Key: k.Key}
	err := q.QueryRowContext(ctx, `SELECT user_id, failures, window_started_at, locked_until,
			lockouts, updated_at
		FROM login_throttle WHERE key = :key`, sql.Named("key", k.Key)).
		Scan(&uid, &row.Failures, &window, &locked, &row.Lockouts, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		row.UserID = k.UserID
		row.WindowStart = now
		row.UpdatedAt = now
		return row, false, nil
	}
	if err != nil {
		return model.ThrottleRow{}, false, fmt.Errorf("read throttle row: %w", err)
	}
	row.UserID = int64Ptr(uid)
	if k.UserID != nil {
		row.UserID = k.UserID
	}
	if row.WindowStart, err = parseTime(window); err != nil {
		return model.ThrottleRow{}, false, err
	}
	if row.LockedUntil, err = parseNullTime(locked); err != nil {
		return model.ThrottleRow{}, false, err
	}
	if row.UpdatedAt, err = parseTime(updated); err != nil {
		return model.ThrottleRow{}, false, err
	}
	return row, true, nil
}

// ClearThrottle deletes the throttle row of a key.
func (s *Store) ClearThrottle(ctx context.Context, key string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM login_throttle WHERE key = :key`,
		sql.Named("key", key)); err != nil {
		return fmt.Errorf("clear throttle: %w", err)
	}
	return nil
}

// ClearUserThrottle deletes every throttle row of an account (unlock, 6.7).
func (s *Store) ClearUserThrottle(ctx context.Context, userID int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM login_throttle WHERE user_id = :uid`,
		sql.Named("uid", userID)); err != nil {
		return fmt.Errorf("clear user throttle: %w", err)
	}
	return nil
}
