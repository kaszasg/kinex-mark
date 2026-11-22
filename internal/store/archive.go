package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Automatic archiving statements (7.2), run per user in one transaction.
const (
	// missing state rows for shared links, baseline = earliest effective share,
	// but not before the recipient's account existed (viewerCreatedAt)
	archiveSharedStateInsert = `INSERT INTO link_user_state (user_id, link_id, baseline_at, updated_at)
SELECT :uid, s.link_id, max(MIN(s.created_at), ` + viewerCreatedAt + `), :now
FROM link_shares s
JOIN links l ON l.id = s.link_id
JOIN users o ON o.id = l.owner_id AND o.role IN ('admin', 'developer')
WHERE l.owner_id <> :uid
  AND (s.target_type = 'all'
       OR (s.target_type = 'role' AND s.target_role = :role)
       OR (s.target_type = 'user' AND s.target_user_id = :uid))
GROUP BY s.link_id
ON CONFLICT (user_id, link_id) DO NOTHING`

	// roleVisibilityUpsert runs when a user's role changes from :old_role to
	// :role: the links shared with the new role that were not visible before
	// (no share with everyone, the old role or the user) start their
	// inactivity period now. An existing state row (from an earlier period of
	// visibility) keeps its pin, use and archive data; only a baseline older
	// than now moves forward.
	roleVisibilityUpsert = `INSERT INTO link_user_state (user_id, link_id, baseline_at, updated_at)
SELECT :uid, s.link_id, :now, :now
FROM link_shares s
JOIN links l ON l.id = s.link_id
JOIN users o ON o.id = l.owner_id AND o.role IN ('admin', 'developer')
WHERE l.owner_id <> :uid
  AND s.target_type = 'role' AND s.target_role = :role
  AND NOT EXISTS (
      SELECT 1 FROM link_shares s2
      WHERE s2.link_id = s.link_id
        AND (s2.target_type = 'all'
             OR (s2.target_type = 'role' AND s2.target_role = :old_role)
             OR (s2.target_type = 'user' AND s2.target_user_id = :uid)))
GROUP BY s.link_id
ON CONFLICT (user_id, link_id) DO UPDATE SET
    baseline_at = max(baseline_at, excluded.baseline_at),
    updated_at = excluded.updated_at`

	// ownerSharesReturnUpsert runs when the owner :uid gets a sharing role
	// back (customer to developer or admin): the shares that were dormant
	// while the owner was a customer (4.7) become visible again, so every
	// recipient starts the inactivity period of those links now. As in
	// roleVisibilityUpsert, an existing state row keeps its pin, use and
	// archive data, and only a baseline older than now moves forward.
	ownerSharesReturnUpsert = `INSERT INTO link_user_state (user_id, link_id, baseline_at, updated_at)
SELECT u.id, s.link_id, :now, :now
FROM link_shares s
JOIN links l ON l.id = s.link_id AND l.owner_id = :uid
JOIN users u ON u.id <> :uid
WHERE (s.target_type = 'all'
       OR (s.target_type = 'role' AND s.target_role = u.role)
       OR (s.target_type = 'user' AND s.target_user_id = u.id))
GROUP BY u.id, s.link_id
ON CONFLICT (user_id, link_id) DO UPDATE SET
    baseline_at = max(baseline_at, excluded.baseline_at),
    updated_at = excluded.updated_at`

	// own links get their state row on creation; fallback for older rows
	archiveOwnStateInsert = `INSERT INTO link_user_state (user_id, link_id, baseline_at, updated_at)
SELECT :uid, l.id, l.created_at, :now FROM links l WHERE l.owner_id = :uid
ON CONFLICT (user_id, link_id) DO NOTHING`

	archiveUpdate = `UPDATE link_user_state
SET archived_at = :now, archive_reason = 'auto', updated_at = :now
WHERE user_id = :uid
  AND archived_at IS NULL
  AND pinned = 0
  AND max(baseline_at, coalesce(last_used_at, baseline_at)) < :cutoff
  AND link_id IN (
      SELECT id FROM links WHERE owner_id = :uid
      UNION
      SELECT s.link_id FROM link_shares s
      JOIN links l ON l.id = s.link_id
      JOIN users o ON o.id = l.owner_id AND o.role IN ('admin', 'developer')
      WHERE l.owner_id <> :uid
        AND (s.target_type = 'all'
             OR (s.target_type = 'role' AND s.target_role = :role)
             OR (s.target_type = 'user' AND s.target_user_id = :uid)))`
)

type archiveUser struct {
	id   int64
	role string
	days sql.NullInt64
}

// ArchiveInactive archives, for every active user, the visible links that
// are neither archived nor pinned and whose last activity is older than the
// user's effective period (archive_after_days, else defaultDays; 0 = never).
// It returns the number of archived rows (7.2).
func (s *Store) ArchiveInactive(ctx context.Context, defaultDays int) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT u.id, u.role, p.archive_after_days
		FROM users u LEFT JOIN user_preferences p ON p.user_id = u.id
		WHERE u.is_active = 1`)
	if err != nil {
		return 0, fmt.Errorf("list users for archiving: %w", err)
	}
	var users []archiveUser
	for rows.Next() {
		var u archiveUser
		if err := rows.Scan(&u.id, &u.role, &u.days); err != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("scan user for archiving: %w", err)
		}
		users = append(users, u)
	}
	if err := closeRows(rows); err != nil {
		return 0, fmt.Errorf("list users for archiving: %w", err)
	}

	total := 0
	for _, u := range users {
		days := defaultDays
		if u.days.Valid {
			days = int(u.days.Int64)
		}
		if days <= 0 {
			continue
		}
		n, err := s.archiveUser(ctx, u, days)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func (s *Store) archiveUser(ctx context.Context, u archiveUser, days int) (int, error) {
	n := 0
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		now := s.now()
		args := []any{sql.Named("uid", u.id), sql.Named("role", u.role),
			sql.Named("now", formatTime(now)),
			sql.Named("cutoff", formatTime(now.Add(-time.Duration(days)*24*time.Hour)))}
		if _, err := tx.ExecContext(ctx, archiveSharedStateInsert, args...); err != nil {
			return fmt.Errorf("archive shared state rows: %w", err)
		}
		if _, err := tx.ExecContext(ctx, archiveOwnStateInsert, args...); err != nil {
			return fmt.Errorf("archive own state rows: %w", err)
		}
		res, err := tx.ExecContext(ctx, archiveUpdate, args...)
		if err != nil {
			return fmt.Errorf("archive links: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("archive links: %w", err)
		}
		n = int(affected)
		return nil
	})
	return n, err
}
