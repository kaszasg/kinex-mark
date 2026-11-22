package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// visibleCTE lists the links a user (:uid, role :role) may see (4.7): own
// links, and links of an admin or developer owner shared with everyone, with
// the user's role or with the user personally. Every visibility-dependent
// query starts with it.
const visibleCTE = `WITH vis AS (
    SELECT l.id AS link_id, NULL AS via, NULL AS shared_at
    FROM links l
    WHERE l.owner_id = :uid
    UNION ALL
    SELECT s.link_id, s.target_type, s.created_at
    FROM link_shares s
    JOIN links l ON l.id = s.link_id
    JOIN users o ON o.id = l.owner_id AND o.role IN ('admin', 'developer')
    WHERE l.owner_id <> :uid
      AND (s.target_type = 'all'
           OR (s.target_type = 'role' AND s.target_role = :role)
           OR (s.target_type = 'user' AND s.target_user_id = :uid))
),
v AS (
    SELECT link_id,
           MIN(shared_at) AS shared_at,
           CASE WHEN SUM(via = 'user') > 0 THEN 'user'
                WHEN SUM(via = 'role') > 0 THEN 'role'
                WHEN SUM(via = 'all')  > 0 THEN 'all'
           END AS via
    FROM vis
    GROUP BY link_id
)
`

// viewerCreatedAt is the creation time of the viewer's account (:uid). A
// shared link's inactivity period starts at the earliest effective share,
// but never before the recipient's account existed: a user cannot have left
// a link unused before the account was created (7.2; the functional
// requirements of BASE-SYSTEM-PROMPT.md). A link that becomes visible through a role change gets a
// state row with the time of the change (roleVisibilityUpsert).
const viewerCreatedAt = `(SELECT vu.created_at FROM users vu WHERE vu.id = :uid)`

// linkColumns are the columns of scanLink (4.8 plus the detail columns).
const linkColumns = `l.id, l.url, l.host, l.title, l.title_source, l.description, l.fetch_status,
       l.owner_id, o.username, o.display_name,
       ic.sha256,
       coalesce(st.pinned, 0)    AS pinned,
       coalesce(st.use_count, 0) AS use_count,
       st.last_used_at, st.archived_at, st.archive_reason,
       coalesce(st.baseline_at,
                CASE WHEN l.owner_id = :uid THEN l.created_at
                     ELSE max(v.shared_at, ` + viewerCreatedAt + `) END) AS baseline_at,
       v.via, v.shared_at,
       (SELECT json_group_array(lt.tag_id)
          FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
         WHERE lt.link_id = l.id AND t.owner_id = :uid) AS tag_ids,
       CASE WHEN l.owner_id = :uid
            THEN (SELECT COUNT(*) FROM link_shares s2 WHERE s2.link_id = l.id)
            ELSE 0 END AS share_count,
       l.created_at, l.updated_at,
       l.fetch_error, l.fetched_at, l.fetch_attempts`

// linkFrom joins a visible link with its owner, icon and the viewer's state.
const linkFrom = `
FROM v
JOIN links l ON l.id = v.link_id
JOIN users o ON o.id = l.owner_id
LEFT JOIN icons ic ON ic.id = l.icon_id
LEFT JOIN link_user_state st ON st.user_id = :uid AND st.link_id = l.id`

// linkFilter is the WHERE clause of the link list (4.8).
const linkFilter = `
WHERE (:archived = 'include'
       OR (:archived = 'exclude' AND st.archived_at IS NULL)
       OR (:archived = 'only'    AND st.archived_at IS NOT NULL))
  AND (:scope = 'all'
       OR (:scope = 'own'    AND l.owner_id = :uid)
       OR (:scope = 'shared' AND l.owner_id <> :uid))
  AND (:q IS NULL
       OR instr(l.search_text, :q) > 0
       OR EXISTS (SELECT 1 FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
                   WHERE lt.link_id = l.id AND t.owner_id = :uid AND instr(` + foldFunc + `(t.name_key), :q) > 0))
  AND (:tag_ids IS NULL
       OR (:tag_match = 'any' AND EXISTS (
              SELECT 1 FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
               WHERE lt.link_id = l.id AND t.owner_id = :uid
                 AND lt.tag_id IN (SELECT value FROM json_each(:tag_ids))))
       OR (:tag_match = 'all' AND (
              SELECT COUNT(DISTINCT lt.tag_id) FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
               WHERE lt.link_id = l.id AND t.owner_id = :uid
                 AND lt.tag_id IN (SELECT value FROM json_each(:tag_ids)))
              = json_array_length(:tag_ids)))
  AND (:untagged = 0 OR NOT EXISTS (
          SELECT 1 FROM link_tags lt JOIN tags t ON t.id = lt.tag_id
           WHERE lt.link_id = l.id AND t.owner_id = :uid))`

// linkOrder is the ORDER BY of the link list (4.8).
const linkOrder = `
ORDER BY coalesce(st.pinned, 0) DESC,
         CASE WHEN :sort = 'usage' THEN coalesce(st.use_count, 0) END DESC,
         CASE WHEN :sort = 'usage' THEN st.last_used_at END DESC NULLS LAST,
         CASE WHEN :sort = 'title' THEN l.sort_label END ASC,
         l.created_at DESC,
         l.id DESC`

const linksListQuery = visibleCTE + `SELECT ` + linkColumns + `,
       COUNT(*) OVER () AS total,
       SUM(coalesce(st.pinned, 0)) OVER () AS pinned_count,
       SUM(st.archived_at IS NOT NULL) OVER () AS archived_count` +
	linkFrom + linkFilter + linkOrder + `
LIMIT :limit OFFSET :offset`

const linksCountQuery = visibleCTE + `SELECT COUNT(*),
       coalesce(SUM(coalesce(st.pinned, 0)), 0),
       coalesce(SUM(st.archived_at IS NOT NULL), 0)` + linkFrom + linkFilter

const linkByIDQuery = visibleCTE + `SELECT ` + linkColumns + linkFrom + `
WHERE l.id = :id`

// linkRow is a scanned link with the detail-only columns.
type linkRow struct {
	item          model.LinkItem
	titleSource   model.TitleSource
	fetchError    *string
	fetchedAt     sql.NullString
	fetchAttempts int
}

type linkScan struct {
	r                                     linkRow
	titleSource, fetchStatus, baselineAt  string
	createdAt, updatedAt, tagIDs          string
	iconSHA, lastUsed, archivedAt, reason sql.NullString
	via, sharedAt, fetchError             sql.NullString
}

func (ls *linkScan) dest() []any {
	it := &ls.r.item
	return []any{&it.ID, &it.URL, &it.Host, &it.Title, &ls.titleSource, &it.Description,
		&ls.fetchStatus, &it.Owner.ID, &it.Owner.Username, &it.Owner.DisplayName,
		&ls.iconSHA, &it.Pinned, &it.UseCount, &ls.lastUsed, &ls.archivedAt, &ls.reason,
		&ls.baselineAt, &ls.via, &ls.sharedAt, &ls.tagIDs, &it.ShareCount,
		&ls.createdAt, &ls.updatedAt, &ls.fetchError, &ls.r.fetchedAt, &ls.r.fetchAttempts}
}

func (ls *linkScan) row(uid int64) (linkRow, error) {
	r := ls.r
	it := &r.item
	r.titleSource = model.TitleSource(ls.titleSource)
	r.fetchError = stringPtr(ls.fetchError)
	it.FetchStatus = model.FetchStatus(ls.fetchStatus)
	it.IconSHA256 = ls.iconSHA.String
	it.IsOwner = it.Owner.ID == uid
	if ls.reason.Valid {
		reason := model.ArchiveReason(ls.reason.String)
		it.ArchiveReason = &reason
	}
	if ls.via.Valid {
		via := model.ShareTargetType(ls.via.String)
		it.SharedVia = &via
	}
	var err error
	if it.LastUsedAt, err = parseNullTime(ls.lastUsed); err != nil {
		return linkRow{}, err
	}
	if it.ArchivedAt, err = parseNullTime(ls.archivedAt); err != nil {
		return linkRow{}, err
	}
	if it.SharedAt, err = parseNullTime(ls.sharedAt); err != nil {
		return linkRow{}, err
	}
	if it.BaselineAt, err = parseTime(ls.baselineAt); err != nil {
		return linkRow{}, err
	}
	if it.CreatedAt, err = parseTime(ls.createdAt); err != nil {
		return linkRow{}, err
	}
	if it.UpdatedAt, err = parseTime(ls.updatedAt); err != nil {
		return linkRow{}, err
	}
	it.TagIDs = []int64{}
	if err := json.Unmarshal([]byte(ls.tagIDs), &it.TagIDs); err != nil {
		return linkRow{}, fmt.Errorf("decode tag ids: %w", err)
	}
	if it.TagIDs == nil {
		it.TagIDs = []int64{}
	}
	sort.Slice(it.TagIDs, func(i, j int) bool { return it.TagIDs[i] < it.TagIDs[j] })
	return r, nil
}

// viewerArgs are the named arguments of visibleCTE.
func viewerArgs(v model.Viewer) []any {
	return []any{sql.Named("uid", v.UserID), sql.Named("role", string(v.Role))}
}

// linkRowQ loads one visible link; model.ErrNotFound when it does not exist
// or is not visible (indistinguishable on purpose, IDOR).
func linkRowQ(ctx context.Context, q querier, v model.Viewer, id int64) (linkRow, error) {
	var ls linkScan
	err := q.QueryRowContext(ctx, linkByIDQuery,
		append(viewerArgs(v), sql.Named("id", id))...).Scan(ls.dest()...)
	if err != nil {
		return linkRow{}, notFound(err)
	}
	return ls.row(v.UserID)
}

// Links returns one page of the links visible to v with the filtered total
// and counts (4.8).
func (s *Store) Links(ctx context.Context, v model.Viewer, q model.LinkQuery) (model.LinkPage, error) {
	var tagIDs any
	if len(q.TagIDs) > 0 {
		b, err := json.Marshal(q.TagIDs)
		if err != nil {
			return model.LinkPage{}, fmt.Errorf("encode tag ids: %w", err)
		}
		tagIDs = string(b)
	}
	args := append(viewerArgs(v),
		sql.Named("archived", orDefault(q.Archived, model.ArchivedExclude)),
		sql.Named("scope", orDefault(q.Scope, model.ScopeAll)),
		sql.Named("q", nullText(foldText(q.Q))),
		sql.Named("tag_ids", tagIDs),
		sql.Named("tag_match", orDefault(q.TagMatch, model.TagMatchAny)),
		sql.Named("untagged", q.Untagged),
		sql.Named("sort", orDefault(q.Sort, model.SortUsage)),
		sql.Named("limit", q.Limit), sql.Named("offset", q.Offset))

	page := model.LinkPage{Items: []model.LinkItem{}, Limit: q.Limit, Offset: q.Offset}
	rows, err := s.db.QueryContext(ctx, linksListQuery, args...)
	if err != nil {
		return model.LinkPage{}, fmt.Errorf("list links: %w", err)
	}
	for rows.Next() {
		var ls linkScan
		dest := append(ls.dest(), &page.Total, &page.Counts.Pinned, &page.Counts.Archived)
		if err := rows.Scan(dest...); err != nil {
			_ = rows.Close()
			return model.LinkPage{}, fmt.Errorf("scan link: %w", err)
		}
		r, err := ls.row(v.UserID)
		if err != nil {
			_ = rows.Close()
			return model.LinkPage{}, err
		}
		page.Items = append(page.Items, r.item)
	}
	if err := closeRows(rows); err != nil {
		return model.LinkPage{}, fmt.Errorf("list links: %w", err)
	}
	if len(page.Items) == 0 {
		// No row carries the window counts (empty set or offset past the end).
		err := s.db.QueryRowContext(ctx, linksCountQuery, args...).
			Scan(&page.Total, &page.Counts.Pinned, &page.Counts.Archived)
		if err != nil {
			return model.LinkPage{}, fmt.Errorf("count links: %w", err)
		}
	}
	return page, nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// LinkItem returns one visible link; model.ErrNotFound when it is not visible.
func (s *Store) LinkItem(ctx context.Context, v model.Viewer, id int64) (model.LinkItem, error) {
	r, err := linkRowQ(ctx, s.db, v, id)
	if err != nil {
		return model.LinkItem{}, err
	}
	return r.item, nil
}

// Link returns the detail view of one visible link; model.ErrNotFound when
// it is not visible.
func (s *Store) Link(ctx context.Context, v model.Viewer, id int64) (model.LinkDetail, error) {
	return linkDetailQ(ctx, s.db, v, id)
}

func linkDetailQ(ctx context.Context, q querier, v model.Viewer, id int64) (model.LinkDetail, error) {
	r, err := linkRowQ(ctx, q, v, id)
	if err != nil {
		return model.LinkDetail{}, err
	}
	d := model.LinkDetail{
		LinkItem:      r.item,
		TitleSource:   r.titleSource,
		FetchError:    r.fetchError,
		FetchAttempts: r.fetchAttempts,
		Tags:          []model.Tag{},
		Shares:        []model.Share{},
	}
	if d.FetchedAt, err = parseNullTime(r.fetchedAt); err != nil {
		return model.LinkDetail{}, err
	}
	if len(r.item.TagIDs) > 0 {
		tags, err := tagsQ(ctx, q, v.UserID, v.Role)
		if err != nil {
			return model.LinkDetail{}, err
		}
		want := map[int64]bool{}
		for _, id := range r.item.TagIDs {
			want[id] = true
		}
		for _, t := range tags {
			if want[t.ID] {
				d.Tags = append(d.Tags, t)
			}
		}
	}
	if r.item.IsOwner && v.Role.CanShare() {
		if d.Shares, err = sharesQ(ctx, q, id); err != nil {
			return model.LinkDetail{}, err
		}
	}
	return d, nil
}

// isVisibleQ reports whether link id is visible to v.
func isVisibleQ(ctx context.Context, q querier, v model.Viewer, id int64) (bool, error) {
	var one int
	err := q.QueryRowContext(ctx, visibleCTE+`SELECT 1 FROM v WHERE v.link_id = :id`,
		append(viewerArgs(v), sql.Named("id", id))...).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check visibility: %w", err)
	}
	return true, nil
}

// requireVisibleQ returns model.ErrNotFound unless link id is visible to v.
func requireVisibleQ(ctx context.Context, q querier, v model.Viewer, id int64) error {
	ok, err := isVisibleQ(ctx, q, v, id)
	if err != nil {
		return err
	}
	if !ok {
		return model.ErrNotFound
	}
	return nil
}

// requireOwnerQ checks an owner-only operation (4.7): nil for the owner,
// model.ErrNotOwner when the link is visible but not own, otherwise
// model.ErrNotFound.
func requireOwnerQ(ctx context.Context, q querier, v model.Viewer, id int64) error {
	var owner int64
	err := q.QueryRowContext(ctx, `SELECT owner_id FROM links WHERE id = :id`,
		sql.Named("id", id)).Scan(&owner)
	if err != nil {
		return notFound(err)
	}
	if owner == v.UserID {
		return nil
	}
	return notOwnerOrNotFound(ctx, q, v, id)
}

// notOwnerOrNotFound distinguishes a visible foreign link (model.ErrNotOwner)
// from an invisible or missing one (model.ErrNotFound).
func notOwnerOrNotFound(ctx context.Context, q querier, v model.Viewer, id int64) error {
	visible, err := isVisibleQ(ctx, q, v, id)
	if err != nil {
		return err
	}
	if visible {
		return model.ErrNotOwner
	}
	return model.ErrNotFound
}
