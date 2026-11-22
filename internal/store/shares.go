package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kaszasg/kinex-mark/internal/model"
)

const shareColumns = `s.id, s.target_type, s.target_role, s.target_user_id, s.created_at,
	tu.username, tu.display_name`

func scanShare(r rowScanner) (model.Share, error) {
	var sh model.Share
	var targetType, createdAt string
	var role, username, display sql.NullString
	var userID sql.NullInt64
	if err := r.Scan(&sh.ID, &targetType, &role, &userID, &createdAt, &username, &display); err != nil {
		return model.Share{}, err
	}
	sh.TargetType = model.ShareTargetType(targetType)
	if role.Valid {
		rl := model.Role(role.String)
		sh.TargetRole = &rl
	}
	if userID.Valid {
		sh.TargetUser = &model.UserRef{ID: userID.Int64, Username: username.String, DisplayName: display.String}
	}
	var err error
	if sh.CreatedAt, err = parseTime(createdAt); err != nil {
		return model.Share{}, err
	}
	return sh, nil
}

// sharesQ lists the shares of a link, oldest first.
func sharesQ(ctx context.Context, q querier, linkID int64) ([]model.Share, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+shareColumns+`
		FROM link_shares s LEFT JOIN users tu ON tu.id = s.target_user_id
		WHERE s.link_id = :id
		ORDER BY s.created_at, s.id`, sql.Named("id", linkID))
	if err != nil {
		return nil, fmt.Errorf("list shares: %w", err)
	}
	items := []model.Share{}
	for rows.Next() {
		sh, err := scanShare(rows)
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan share: %w", err)
		}
		items = append(items, sh)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("list shares: %w", err)
	}
	return items, nil
}

// Shares lists the shares of an own link. Errors: model.ErrNotOwner,
// model.ErrNotFound. The role permission (share_not_allowed) is checked by
// the handler.
func (s *Store) Shares(ctx context.Context, v model.Viewer, linkID int64) ([]model.Share, error) {
	var out []model.Share
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		if err := requireOwnerQ(ctx, tx, v, linkID); err != nil {
			return err
		}
		var err error
		out, err = sharesQ(ctx, tx, linkID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CreateShare shares an own link (5.9.6). Errors: model.ErrNotOwner,
// model.ErrNotFound, *model.DuplicateError{Kind: "share"} with the existing
// share id, and model.ErrConflict for an invalid target (wrong field
// combination, unknown role, a target user that does not exist, is inactive
// or is the owner).
func (s *Store) CreateShare(ctx context.Context, v model.Viewer, linkID int64, t model.ShareTarget) (model.Share, error) {
	var out model.Share
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		if err := requireOwnerQ(ctx, tx, v, linkID); err != nil {
			return err
		}
		if err := checkShareTargetQ(ctx, tx, v.UserID, t); err != nil {
			return err
		}
		var role any
		if t.TargetRole != nil {
			role = string(*t.TargetRole)
		}
		userID := nullInt64(t.TargetUserID)
		var existing int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM link_shares
			WHERE link_id = :link AND target_type = :type
			  AND coalesce(target_role, '') = coalesce(:role, '')
			  AND coalesce(target_user_id, 0) = coalesce(:user, 0)`,
			sql.Named("link", linkID), sql.Named("type", string(t.TargetType)),
			sql.Named("role", role), sql.Named("user", userID)).Scan(&existing)
		if err == nil {
			return &model.DuplicateError{Kind: model.DuplicateShare, ExistingID: existing}
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check duplicate share: %w", err)
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO link_shares (link_id, target_type, target_role,
				target_user_id, created_at)
			VALUES (:link, :type, :role, :user, :now)`,
			sql.Named("link", linkID), sql.Named("type", string(t.TargetType)),
			sql.Named("role", role), sql.Named("user", userID),
			sql.Named("now", formatTime(s.now())))
		if err != nil {
			return fmt.Errorf("insert share: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("insert share: %w", err)
		}
		out, err = scanShare(tx.QueryRowContext(ctx, `SELECT `+shareColumns+`
			FROM link_shares s LEFT JOIN users tu ON tu.id = s.target_user_id
			WHERE s.id = :id`, sql.Named("id", id)))
		if err != nil {
			return fmt.Errorf("read share: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.Share{}, err
	}
	return out, nil
}

// checkShareTargetQ validates a share target; problems are model.ErrConflict.
func checkShareTargetQ(ctx context.Context, q querier, ownerID int64, t model.ShareTarget) error {
	switch t.TargetType {
	case model.ShareAll:
		if t.TargetRole != nil || t.TargetUserID != nil {
			return fmt.Errorf("share to all takes no target: %w", model.ErrConflict)
		}
	case model.ShareRole:
		if t.TargetRole == nil || !t.TargetRole.Valid() || t.TargetUserID != nil {
			return fmt.Errorf("invalid role share target: %w", model.ErrConflict)
		}
	case model.ShareUser:
		if t.TargetUserID == nil || t.TargetRole != nil || *t.TargetUserID == ownerID {
			return fmt.Errorf("invalid user share target: %w", model.ErrConflict)
		}
		var one int
		err := q.QueryRowContext(ctx, `SELECT 1 FROM users WHERE id = :id AND is_active = 1`,
			sql.Named("id", *t.TargetUserID)).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("unknown share target user: %w", model.ErrConflict)
		}
		if err != nil {
			return fmt.Errorf("check share target: %w", err)
		}
	default:
		return fmt.Errorf("unknown share target type %q: %w", t.TargetType, model.ErrConflict)
	}
	return nil
}

// DeleteShare deletes a share of an own link (4.7): the share must belong to
// linkID and the link to v, in one statement. When nothing is deleted the
// error is model.ErrNotOwner for a visible foreign link and model.ErrNotFound
// otherwise.
func (s *Store) DeleteShare(ctx context.Context, v model.Viewer, linkID, shareID int64) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM link_shares
			WHERE id = :share_id
			  AND link_id = :link_id
			  AND link_id IN (SELECT id FROM links WHERE id = :link_id AND owner_id = :uid)`,
			sql.Named("share_id", shareID), sql.Named("link_id", linkID), sql.Named("uid", v.UserID))
		if err != nil {
			return fmt.Errorf("delete share: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("delete share: %w", err)
		}
		if n > 0 {
			return nil
		}
		var owner int64
		err = tx.QueryRowContext(ctx, `SELECT owner_id FROM links WHERE id = :id`,
			sql.Named("id", linkID)).Scan(&owner)
		if err != nil {
			return notFound(err)
		}
		if owner == v.UserID {
			return model.ErrNotFound
		}
		return notOwnerOrNotFound(ctx, tx, v, linkID)
	})
}
