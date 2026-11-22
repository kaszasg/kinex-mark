package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// tagsQuery lists the user's tags by name with the number of visible links
// carrying each tag and how many of those are not archived for the user
// (5.10).
const tagsQuery = visibleCTE + `SELECT t.id, t.name, t.color, t.created_at, t.updated_at,
       (SELECT COUNT(*) FROM link_tags lt JOIN v ON v.link_id = lt.link_id
         WHERE lt.tag_id = t.id) AS link_count,
       (SELECT COUNT(*) FROM link_tags lt JOIN v ON v.link_id = lt.link_id
          LEFT JOIN link_user_state st ON st.user_id = :uid AND st.link_id = lt.link_id
         WHERE lt.tag_id = t.id AND st.archived_at IS NULL) AS active_link_count
FROM tags t
WHERE t.owner_id = :uid AND (:id IS NULL OR t.id = :id)
ORDER BY t.name_key, t.id`

func scanTag(r rowScanner) (model.Tag, error) {
	var t model.Tag
	var color, createdAt, updatedAt string
	if err := r.Scan(&t.ID, &t.Name, &color, &createdAt, &updatedAt, &t.LinkCount, &t.ActiveLinkCount); err != nil {
		return model.Tag{}, err
	}
	t.Color = model.TagColor(color)
	var err error
	if t.CreatedAt, err = parseTime(createdAt); err != nil {
		return model.Tag{}, err
	}
	if t.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return model.Tag{}, err
	}
	return t, nil
}

func tagsQ(ctx context.Context, q querier, userID int64, role model.Role) ([]model.Tag, error) {
	rows, err := q.QueryContext(ctx, tagsQuery,
		sql.Named("uid", userID), sql.Named("role", string(role)), sql.Named("id", nil))
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	items := []model.Tag{}
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		items = append(items, t)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return items, nil
}

func tagQ(ctx context.Context, q querier, userID int64, role model.Role, id int64) (model.Tag, error) {
	t, err := scanTag(q.QueryRowContext(ctx, tagsQuery,
		sql.Named("uid", userID), sql.Named("role", string(role)), sql.Named("id", id)))
	if err != nil {
		return model.Tag{}, notFound(err)
	}
	return t, nil
}

// Tags lists the user's own tags by name; role is the user's current role
// (visibility of the counted links).
func (s *Store) Tags(ctx context.Context, userID int64, role model.Role) ([]model.Tag, error) {
	return tagsQ(ctx, s.db, userID, role)
}

// duplicateTagQ returns a *model.DuplicateError when the user already has a
// tag with the same name key (excluding tag except).
func duplicateTagQ(ctx context.Context, q querier, userID int64, name string, except int64) error {
	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM tags
		WHERE owner_id = :uid AND name_key = :key AND id <> :except`,
		sql.Named("uid", userID), sql.Named("key", model.TagNameKey(name)),
		sql.Named("except", except)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check duplicate tag: %w", err)
	}
	return &model.DuplicateError{Kind: model.DuplicateTag, ExistingID: id}
}

// CreateTag creates a tag ("" color: neutral). Errors:
// *model.DuplicateError{Kind: "tag"} with the existing id and
// model.ErrLimitReached at model.MaxTagsPerUser tags.
func (s *Store) CreateTag(ctx context.Context, userID int64, name string, color model.TagColor) (model.Tag, error) {
	if color == "" {
		color = model.DefaultTagColor
	}
	var out model.Tag
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		if err := duplicateTagQ(ctx, tx, userID, name, 0); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE owner_id = :uid`,
			sql.Named("uid", userID)).Scan(&n); err != nil {
			return fmt.Errorf("count tags: %w", err)
		}
		if n >= model.MaxTagsPerUser {
			return model.ErrLimitReached
		}
		id, err := insertTagQ(ctx, tx, userID, name, color, formatTime(s.now()))
		if err != nil {
			return err
		}
		role, err := userRoleQ(ctx, tx, userID)
		if err != nil {
			return err
		}
		out, err = tagQ(ctx, tx, userID, role, id)
		return err
	})
	if err != nil {
		return model.Tag{}, err
	}
	return out, nil
}

func insertTagQ(ctx context.Context, q querier, userID int64, name string, color model.TagColor, now string) (int64, error) {
	res, err := q.ExecContext(ctx, `INSERT INTO tags (owner_id, name, name_key, color, created_at, updated_at)
		VALUES (:uid, :name, :key, :color, :now, :now)`,
		sql.Named("uid", userID), sql.Named("name", name), sql.Named("key", model.TagNameKey(name)),
		sql.Named("color", string(color)), sql.Named("now", now))
	if err != nil {
		return 0, fmt.Errorf("insert tag: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert tag: %w", err)
	}
	return id, nil
}

// UpdateTag renames or recolors an own tag. Errors: model.ErrNotFound (not
// the user's tag), *model.DuplicateError{Kind: "tag"}.
func (s *Store) UpdateTag(ctx context.Context, userID, id int64, p model.TagPatch) (model.Tag, error) {
	var out model.Tag
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		role, err := userRoleQ(ctx, tx, userID)
		if err != nil {
			return err
		}
		if _, err := tagQ(ctx, tx, userID, role, id); err != nil {
			return err
		}
		var name, key any
		if p.Name != nil {
			if err := duplicateTagQ(ctx, tx, userID, *p.Name, id); err != nil {
				return err
			}
			name, key = *p.Name, model.TagNameKey(*p.Name)
		}
		var color any
		if p.Color != nil {
			color = string(*p.Color)
		}
		if p.Name != nil || p.Color != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE tags SET name = coalesce(:name, name),
					name_key = coalesce(:key, name_key), color = coalesce(:color, color),
					updated_at = :now
				WHERE id = :id AND owner_id = :uid`,
				sql.Named("name", name), sql.Named("key", key), sql.Named("color", color),
				sql.Named("now", formatTime(s.now())), sql.Named("id", id),
				sql.Named("uid", userID)); err != nil {
				return fmt.Errorf("update tag: %w", err)
			}
		}
		out, err = tagQ(ctx, tx, userID, role, id)
		return err
	})
	if err != nil {
		return model.Tag{}, err
	}
	return out, nil
}

// DeleteTag deletes an own tag; it is removed from every link.
// model.ErrNotFound when it is not the user's tag.
func (s *Store) DeleteTag(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tags WHERE id = :id AND owner_id = :uid`,
		sql.Named("id", id), sql.Named("uid", userID))
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	return affectedOne(res)
}
