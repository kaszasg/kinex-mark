package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Personal state upserts (4.9). Each runs after the visibility check in the
// same transaction.
const (
	visitUpsert = `INSERT INTO link_user_state (user_id, link_id, use_count, last_used_at, baseline_at, updated_at)
VALUES (:uid, :id, 1, :now, :now, :now)
ON CONFLICT (user_id, link_id) DO UPDATE SET
    use_count = use_count + 1, last_used_at = :now,
    archived_at = NULL, archive_reason = NULL, updated_at = :now`

	pinUpsert = `INSERT INTO link_user_state (user_id, link_id, pinned, baseline_at, updated_at)
VALUES (:uid, :id, :pinned, :now, :now)
ON CONFLICT (user_id, link_id) DO UPDATE SET
    pinned = :pinned, archived_at = NULL, archive_reason = NULL, baseline_at = :now, updated_at = :now`

	archiveUpsert = `INSERT INTO link_user_state (user_id, link_id, archived_at, archive_reason, baseline_at, updated_at)
VALUES (:uid, :id, :now, 'manual', :now, :now)
ON CONFLICT (user_id, link_id) DO UPDATE SET
    pinned = 0, archived_at = :now, archive_reason = 'manual', updated_at = :now`

	restoreUpsert = `INSERT INTO link_user_state (user_id, link_id, baseline_at, updated_at)
VALUES (:uid, :id, :now, :now)
ON CONFLICT (user_id, link_id) DO UPDATE SET
    archived_at = NULL, archive_reason = NULL, baseline_at = :now, updated_at = :now`
)

// stateWrite checks that link id is visible to v and runs one upsert.
func (s *Store) stateWrite(ctx context.Context, v model.Viewer, id int64, query string, extra ...any) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := requireVisibleQ(ctx, tx, v, id); err != nil {
			return err
		}
		args := append([]any{sql.Named("uid", v.UserID), sql.Named("id", id),
			sql.Named("now", formatTime(s.now()))}, extra...)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("write link state: %w", err)
		}
		return nil
	})
}

// SetPinned pins or unpins a visible link for v. Pinning ends an archived
// state; both restart the inactivity period (4.9).
func (s *Store) SetPinned(ctx context.Context, v model.Viewer, id int64, pinned bool) error {
	return s.stateWrite(ctx, v, id, pinUpsert, sql.Named("pinned", pinned))
}

// SetArchived archives (manually) or restores a visible link for v.
// Archiving ends a pinned state; restoring restarts the inactivity period.
func (s *Store) SetArchived(ctx context.Context, v model.Viewer, id int64, archived bool) error {
	if archived {
		return s.stateWrite(ctx, v, id, archiveUpsert)
	}
	return s.stateWrite(ctx, v, id, restoreUpsert)
}

// RecordVisit counts one use of a visible link (open or copy, 5.9.5); it also
// restores an archived link. It returns the new state.
func (s *Store) RecordVisit(ctx context.Context, v model.Viewer, id int64) (model.LinkState, error) {
	var out model.LinkState
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		if err := requireVisibleQ(ctx, tx, v, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, visitUpsert, sql.Named("uid", v.UserID), sql.Named("id", id),
			sql.Named("now", formatTime(s.now()))); err != nil {
			return fmt.Errorf("record visit: %w", err)
		}
		var err error
		out, err = linkStateQ(ctx, tx, v.UserID, id)
		return err
	})
	if err != nil {
		return model.LinkState{}, err
	}
	return out, nil
}

// linkStateQ reads a stored personal state row.
func linkStateQ(ctx context.Context, q querier, userID, linkID int64) (model.LinkState, error) {
	var st model.LinkState
	var lastUsed, archivedAt, reason sql.NullString
	var baseline string
	err := q.QueryRowContext(ctx, `SELECT pinned, use_count, last_used_at, archived_at,
			archive_reason, baseline_at
		FROM link_user_state WHERE user_id = :uid AND link_id = :id`,
		sql.Named("uid", userID), sql.Named("id", linkID)).
		Scan(&st.Pinned, &st.UseCount, &lastUsed, &archivedAt, &reason, &baseline)
	if err != nil {
		return model.LinkState{}, notFound(err)
	}
	if reason.Valid {
		r := model.ArchiveReason(reason.String)
		st.ArchiveReason = &r
	}
	if st.LastUsedAt, err = parseNullTime(lastUsed); err != nil {
		return model.LinkState{}, err
	}
	if st.ArchivedAt, err = parseNullTime(archivedAt); err != nil {
		return model.LinkState{}, err
	}
	if st.BaselineAt, err = parseTime(baseline); err != nil {
		return model.LinkState{}, err
	}
	return st, nil
}

// SetLinkTags replaces v's own tags on a visible link. Tags of other users
// on the same link (for example the owner's) are not touched (4.4). A tag
// that is not v's own is model.ErrConflict.
func (s *Store) SetLinkTags(ctx context.Context, v model.Viewer, id int64, tagIDs []int64) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := requireVisibleQ(ctx, tx, v, id); err != nil {
			return err
		}
		ids, err := checkOwnTagsQ(ctx, tx, v.UserID, tagIDs)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM link_tags WHERE link_id = :id
			AND tag_id IN (SELECT id FROM tags WHERE owner_id = :uid)`,
			sql.Named("id", id), sql.Named("uid", v.UserID)); err != nil {
			return fmt.Errorf("clear link tags: %w", err)
		}
		return insertLinkTagsQ(ctx, tx, id, ids, formatTime(s.now()))
	})
}
