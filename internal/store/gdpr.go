package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// ExportUser builds the personal data export (5.8.1) from one consistent
// snapshot. It never contains secrets or icon data; sharedWithMe carries the
// URL only for links that are still visible to the user (4.7).
func (s *Store) ExportUser(ctx context.Context, userID int64) (model.UserExport, error) {
	var out model.UserExport
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		now := s.now()
		u, err := userByIDQ(ctx, tx, userID)
		if err != nil {
			return err
		}
		out = model.UserExport{
			Format:     model.UserExportFormat,
			ExportedAt: now,
			User: model.ExportUser{
				ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Role: u.Role,
				Active: u.Active, TOTPEnabled: u.TOTPEnabled, CreatedAt: u.CreatedAt,
				LastLoginAt: u.LastLoginAt, PasswordChangedAt: u.PasswordChangedAt,
			},
		}
		if out.Preferences, err = preferencesQ(ctx, tx, userID); err != nil {
			return err
		}
		if out.Consents, err = consentsQ(ctx, tx, userID); err != nil {
			return err
		}
		if out.Sessions, err = sessionsQ(ctx, tx, userID, now); err != nil {
			return err
		}
		if out.Tags, err = exportTagsQ(ctx, tx, userID); err != nil {
			return err
		}
		tagNames, err := tagNamesByLinkQ(ctx, tx, userID)
		if err != nil {
			return err
		}
		if out.Links, err = exportOwnLinksQ(ctx, tx, userID, tagNames); err != nil {
			return err
		}
		if out.SharedWithMe, err = exportSharedQ(ctx, tx, u.Viewer(), tagNames); err != nil {
			return err
		}
		if out.AuditEvents, err = exportAuditQ(ctx, tx, userID); err != nil {
			return err
		}
		out.Security.LockedUntil, err = lockedUntilQ(ctx, tx, userID, now)
		return err
	})
	if err != nil {
		return model.UserExport{}, err
	}
	return out, nil
}

func exportTagsQ(ctx context.Context, q querier, userID int64) ([]model.ExportTag, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, name, color, created_at FROM tags
		WHERE owner_id = :uid ORDER BY name_key, id`, sql.Named("uid", userID))
	if err != nil {
		return nil, fmt.Errorf("export tags: %w", err)
	}
	items := []model.ExportTag{}
	for rows.Next() {
		var t model.ExportTag
		var color, at string
		if err := rows.Scan(&t.ID, &t.Name, &color, &at); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan export tag: %w", err)
		}
		t.Color = model.TagColor(color)
		if t.CreatedAt, err = parseTime(at); err != nil {
			_ = rows.Close()
			return nil, err
		}
		items = append(items, t)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("export tags: %w", err)
	}
	return items, nil
}

// tagNamesByLinkQ maps link id to the names of the user's own tags on it,
// in name order.
func tagNamesByLinkQ(ctx context.Context, q querier, userID int64) (map[int64][]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT lt.link_id, t.name
		FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
		WHERE t.owner_id = :uid
		ORDER BY lt.link_id, t.name_key`, sql.Named("uid", userID))
	if err != nil {
		return nil, fmt.Errorf("read tag names: %w", err)
	}
	out := map[int64][]string{}
	for rows.Next() {
		var linkID int64
		var name string
		if err := rows.Scan(&linkID, &name); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan tag name: %w", err)
		}
		out[linkID] = append(out[linkID], name)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("read tag names: %w", err)
	}
	return out, nil
}

func namesOf(m map[int64][]string, id int64) []string {
	if names := m[id]; names != nil {
		return names
	}
	return []string{}
}

func exportOwnLinksQ(ctx context.Context, q querier, userID int64, tagNames map[int64][]string) ([]model.ExportOwnLink, error) {
	rows, err := q.QueryContext(ctx, `SELECT l.id, l.url, l.title, l.description, l.created_at,
			l.updated_at, coalesce(st.pinned, 0), coalesce(st.use_count, 0), st.last_used_at,
			st.archived_at
		FROM links l
		LEFT JOIN link_user_state st ON st.user_id = :uid AND st.link_id = l.id
		WHERE l.owner_id = :uid
		ORDER BY l.created_at, l.id`, sql.Named("uid", userID))
	if err != nil {
		return nil, fmt.Errorf("export links: %w", err)
	}
	items := []model.ExportOwnLink{}
	index := map[int64]int{}
	for rows.Next() {
		var l model.ExportOwnLink
		var created, updated string
		var lastUsed, archived sql.NullString
		if err := rows.Scan(&l.ID, &l.URL, &l.Title, &l.Description, &created, &updated,
			&l.Pinned, &l.UseCount, &lastUsed, &archived); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan export link: %w", err)
		}
		if err := parseAll(
			func() (err error) { l.CreatedAt, err = parseTime(created); return },
			func() (err error) { l.UpdatedAt, err = parseTime(updated); return },
			func() (err error) { l.LastUsedAt, err = parseNullTime(lastUsed); return },
			func() (err error) { l.ArchivedAt, err = parseNullTime(archived); return },
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
		l.Tags = namesOf(tagNames, l.ID)
		l.Shares = []model.ExportShare{}
		index[l.ID] = len(items)
		items = append(items, l)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("export links: %w", err)
	}

	rows, err = q.QueryContext(ctx, `SELECT s.link_id, s.target_type, s.target_role, tu.username,
			s.created_at
		FROM link_shares s
		JOIN links l ON l.id = s.link_id
		LEFT JOIN users tu ON tu.id = s.target_user_id
		WHERE l.owner_id = :uid
		ORDER BY s.link_id, s.created_at, s.id`, sql.Named("uid", userID))
	if err != nil {
		return nil, fmt.Errorf("export shares: %w", err)
	}
	for rows.Next() {
		var linkID int64
		var sh model.ExportShare
		var targetType, created string
		var role, username sql.NullString
		if err := rows.Scan(&linkID, &targetType, &role, &username, &created); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan export share: %w", err)
		}
		sh.TargetType = model.ShareTargetType(targetType)
		if role.Valid {
			r := model.Role(role.String)
			sh.TargetRole = &r
		}
		sh.TargetUsername = stringPtr(username)
		if sh.CreatedAt, err = parseTime(created); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if i, ok := index[linkID]; ok {
			items[i].Shares = append(items[i].Shares, sh)
		}
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("export shares: %w", err)
	}
	return items, nil
}

// exportSharedQ lists the user's personal state and own tags on links owned
// by others. The URL is included only while the link is visible (5.8.1).
func exportSharedQ(ctx context.Context, q querier, v model.Viewer, tagNames map[int64][]string) ([]model.ExportSharedLink, error) {
	rows, err := q.QueryContext(ctx, visibleCTE+`,
mine AS (
    SELECT link_id FROM link_user_state WHERE user_id = :uid
    UNION
    SELECT lt.link_id FROM link_tags lt JOIN tags t ON t.id = lt.tag_id WHERE t.owner_id = :uid
)
SELECT m.link_id, CASE WHEN v.link_id IS NOT NULL THEN l.url END,
       coalesce(st.pinned, 0), coalesce(st.use_count, 0), st.last_used_at, st.archived_at
FROM mine m
JOIN links l ON l.id = m.link_id
LEFT JOIN v ON v.link_id = m.link_id
LEFT JOIN link_user_state st ON st.user_id = :uid AND st.link_id = m.link_id
WHERE l.owner_id <> :uid
ORDER BY m.link_id`, viewerArgs(v)...)
	if err != nil {
		return nil, fmt.Errorf("export shared links: %w", err)
	}
	items := []model.ExportSharedLink{}
	for rows.Next() {
		var l model.ExportSharedLink
		var url, lastUsed, archived sql.NullString
		if err := rows.Scan(&l.LinkID, &url, &l.Pinned, &l.UseCount, &lastUsed, &archived); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan shared link: %w", err)
		}
		l.URL = stringPtr(url)
		if err := parseAll(
			func() (err error) { l.LastUsedAt, err = parseNullTime(lastUsed); return },
			func() (err error) { l.ArchivedAt, err = parseNullTime(archived); return },
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
		l.Tags = namesOf(tagNames, l.LinkID)
		items = append(items, l)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("export shared links: %w", err)
	}
	return items, nil
}

// exportAuditQ lists the audit events in which the user is the actor or the
// target user, oldest first. The IP address is exported only for the user's
// own actions (failed sign-ins against the account included, they carry the
// account as actor): in an event where the user is only the target, the IP
// is the acting administrator's, personal data of another person that the
// copy must not disclose (GDPR Art. 15(4)); it is exported as "".
func exportAuditQ(ctx context.Context, q querier, userID int64) ([]model.ExportAuditEvent, error) {
	rows, err := q.QueryContext(ctx, `SELECT at, event, success,
			CASE WHEN actor_id = :uid THEN ip ELSE '' END,
			CASE WHEN actor_id = :uid THEN 'actor' ELSE 'target' END
		FROM audit_log
		WHERE actor_id = :uid OR (target_type = 'user' AND target_id = :uid)
		ORDER BY id`, sql.Named("uid", userID))
	if err != nil {
		return nil, fmt.Errorf("export audit: %w", err)
	}
	items := []model.ExportAuditEvent{}
	for rows.Next() {
		var e model.ExportAuditEvent
		var at string
		if err := rows.Scan(&at, &e.Event, &e.Success, &e.IP, &e.Role); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan export audit: %w", err)
		}
		if e.At, err = parseTime(at); err != nil {
			_ = rows.Close()
			return nil, err
		}
		items = append(items, e)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("export audit: %w", err)
	}
	return items, nil
}

// lockedUntilQ returns the active sign-in lock of an account, if any.
func lockedUntilQ(ctx context.Context, q querier, userID int64, now time.Time) (*time.Time, error) {
	var locked sql.NullString
	err := q.QueryRowContext(ctx, `SELECT MAX(locked_until) FROM login_throttle
		WHERE user_id = :uid AND locked_until > :now`,
		sql.Named("uid", userID), sql.Named("now", formatTime(now))).Scan(&locked)
	if err != nil {
		return nil, fmt.Errorf("read lock: %w", err)
	}
	return parseNullTime(locked)
}

// parseAll runs parse steps until the first error.
func parseAll(steps ...func() error) error {
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}
