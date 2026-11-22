package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// searchText is links.search_text (4.4), folded by foldText so that the
// search ignores case and accents.
func searchText(url, host, title, description string) string {
	return foldText(url + "\n" + host + "\n" + title + "\n" + description)
}

// sortLabel is links.sort_label: the display label (description, else
// title, else host) folded by foldText (4.4), so that a label starting with
// an a acute sorts among the a's and not after z.
func sortLabel(host, title, description string) string {
	switch {
	case description != "":
		return foldText(description)
	case title != "":
		return foldText(title)
	}
	return foldText(host)
}

// titleSourceOf returns the source of a user-entered title (4.4).
func titleSourceOf(title string) model.TitleSource {
	if title == "" {
		return model.TitleNone
	}
	return model.TitleManual
}

// ownLinkCountQ counts the owner's links.
func ownLinkCountQ(ctx context.Context, q querier, ownerID int64) (int, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE owner_id = :uid`,
		sql.Named("uid", ownerID)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count links: %w", err)
	}
	return n, nil
}

// duplicateLinkQ returns a *model.DuplicateError when the owner already has
// url (excluding link except).
func duplicateLinkQ(ctx context.Context, q querier, ownerID int64, url string, except int64) error {
	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM links
		WHERE owner_id = :uid AND url = :url AND id <> :except`,
		sql.Named("uid", ownerID), sql.Named("url", url), sql.Named("except", except)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check duplicate link: %w", err)
	}
	return &model.DuplicateError{Kind: model.DuplicateLink, ExistingID: id}
}

// checkOwnTagsQ verifies that every tag id belongs to userID. An unknown or
// foreign id, or more than model.MaxTagsPerLink ids, is reported as
// model.ErrConflict, which the handlers map to validation_failed with the
// field code "unknown" (or "too_long").
func checkOwnTagsQ(ctx context.Context, q querier, userID int64, tagIDs []int64) ([]int64, error) {
	ids := uniqueIDs(tagIDs)
	if len(ids) > model.MaxTagsPerLink {
		return nil, fmt.Errorf("more than %d tags: %w", model.MaxTagsPerLink, model.ErrConflict)
	}
	for _, id := range ids {
		var one int
		err := q.QueryRowContext(ctx, `SELECT 1 FROM tags WHERE id = :id AND owner_id = :uid`,
			sql.Named("id", id), sql.Named("uid", userID)).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("unknown tag %d: %w", id, model.ErrConflict)
		}
		if err != nil {
			return nil, fmt.Errorf("check tag: %w", err)
		}
	}
	return ids, nil
}

func uniqueIDs(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := []int64{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// insertLinkTagsQ attaches tags to a link (existing pairs are kept).
func insertLinkTagsQ(ctx context.Context, q querier, linkID int64, tagIDs []int64, now string) error {
	for _, id := range tagIDs {
		if _, err := q.ExecContext(ctx, `INSERT INTO link_tags (link_id, tag_id, created_at)
			VALUES (:link, :tag, :now) ON CONFLICT (link_id, tag_id) DO NOTHING`,
			sql.Named("link", linkID), sql.Named("tag", id), sql.Named("now", now)); err != nil {
			return fmt.Errorf("insert link tag: %w", err)
		}
	}
	return nil
}

// CreateLink creates a link marked for metadata fetching, the owner's state
// row and its tags. Errors: *model.DuplicateError{Kind: "link"} with the
// existing id, model.ErrLimitReached at model.MaxLinksPerUser links, and
// model.ErrConflict for a tag that is not the owner's.
func (s *Store) CreateLink(ctx context.Context, n model.NewLink) (model.LinkDetail, error) {
	var out model.LinkDetail
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		if err := duplicateLinkQ(ctx, tx, n.OwnerID, n.URL, 0); err != nil {
			return err
		}
		count, err := ownLinkCountQ(ctx, tx, n.OwnerID)
		if err != nil {
			return err
		}
		if count >= model.MaxLinksPerUser {
			return model.ErrLimitReached
		}
		tagIDs, err := checkOwnTagsQ(ctx, tx, n.OwnerID, n.TagIDs)
		if err != nil {
			return err
		}
		role, err := userRoleQ(ctx, tx, n.OwnerID)
		if err != nil {
			return err
		}
		now := formatTime(s.now())
		id, err := insertLinkQ(ctx, tx, n.OwnerID, n.URL, n.Host, n.Title, n.Description, now, now)
		if err != nil {
			return err
		}
		pinned := 0
		if n.Pinned {
			pinned = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO link_user_state (user_id, link_id, pinned,
				baseline_at, updated_at)
			VALUES (:uid, :id, :pinned, :now, :now)`,
			sql.Named("uid", n.OwnerID), sql.Named("id", id), sql.Named("pinned", pinned),
			sql.Named("now", now)); err != nil {
			return fmt.Errorf("insert link state: %w", err)
		}
		if err := insertLinkTagsQ(ctx, tx, id, tagIDs, now); err != nil {
			return err
		}
		out, err = linkDetailQ(ctx, tx, model.Viewer{UserID: n.OwnerID, Role: role}, id)
		return err
	})
	if err != nil {
		return model.LinkDetail{}, err
	}
	return out, nil
}

// insertLinkQ inserts a pending link (fetch fields reset, next_fetch_at = now, 4.4).
func insertLinkQ(ctx context.Context, q querier, ownerID int64, url, host, title, description,
	createdAt, now string) (int64, error) {
	res, err := q.ExecContext(ctx, `INSERT INTO links (owner_id, url, host, title, title_source,
			description, search_text, sort_label, fetch_status, fetch_error, fetch_attempts,
			next_fetch_at, created_at, updated_at)
		VALUES (:uid, :url, :host, :title, :source, :description, :search, :label,
			'pending', NULL, 0, :now, :created, :now)`,
		sql.Named("uid", ownerID), sql.Named("url", url), sql.Named("host", host),
		sql.Named("title", title), sql.Named("source", string(titleSourceOf(title))),
		sql.Named("description", description),
		sql.Named("search", searchText(url, host, title, description)),
		sql.Named("label", sortLabel(host, title, description)),
		sql.Named("now", now), sql.Named("created", createdAt))
	if err != nil {
		return 0, fmt.Errorf("insert link: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert link: %w", err)
	}
	return id, nil
}

// userRoleQ returns the current role of a user.
func userRoleQ(ctx context.Context, q querier, id int64) (model.Role, error) {
	var role string
	err := q.QueryRowContext(ctx, `SELECT role FROM users WHERE id = :id`, sql.Named("id", id)).Scan(&role)
	if err != nil {
		return "", notFound(err)
	}
	return model.Role(role), nil
}

// UpdateLink changes the URL, title or description of an own link (5.9.4).
// A URL change recomputes host, search text and sort label, drops the icon
// and a fetched title, and marks the link for fetching. An emptied title
// becomes "none"; when it was manual the link is fetched again. Errors:
// model.ErrNotOwner, model.ErrNotFound, *model.DuplicateError{Kind: "link"}.
func (s *Store) UpdateLink(ctx context.Context, v model.Viewer, id int64, p model.LinkPatch) (model.LinkDetail, error) {
	var out model.LinkDetail
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		if err := requireOwnerQ(ctx, tx, v, id); err != nil {
			return err
		}
		var url, host, title, source, description string
		err := tx.QueryRowContext(ctx, `SELECT url, host, title, title_source, description
			FROM links WHERE id = :id`, sql.Named("id", id)).
			Scan(&url, &host, &title, &source, &description)
		if err != nil {
			return notFound(err)
		}
		titleSource := model.TitleSource(source)
		refetch := false
		urlChanged := p.URL != nil && *p.URL != url
		if urlChanged {
			if err := duplicateLinkQ(ctx, tx, v.UserID, *p.URL, id); err != nil {
				return err
			}
			url = *p.URL
			if p.Host != nil {
				host = *p.Host
			}
			if titleSource == model.TitleFetched {
				title, titleSource = "", model.TitleNone
			}
			refetch = true
		}
		if p.Title != nil {
			if *p.Title == "" && titleSource == model.TitleManual {
				refetch = true
			}
			title, titleSource = *p.Title, titleSourceOf(*p.Title)
		}
		if p.Description != nil {
			description = *p.Description
		}
		now := formatTime(s.now())
		_, err = tx.ExecContext(ctx, `UPDATE links SET url = :url, host = :host, title = :title,
				title_source = :source, description = :description, search_text = :search,
				sort_label = :label, updated_at = :now,
				icon_id = CASE WHEN :url_changed THEN NULL ELSE icon_id END,
				fetched_at = CASE WHEN :url_changed THEN NULL ELSE fetched_at END,
				fetch_status = CASE WHEN :refetch THEN 'pending' ELSE fetch_status END,
				fetch_error = CASE WHEN :refetch THEN NULL ELSE fetch_error END,
				fetch_attempts = CASE WHEN :refetch THEN 0 ELSE fetch_attempts END,
				next_fetch_at = CASE WHEN :refetch THEN :now ELSE next_fetch_at END
			WHERE id = :id`,
			sql.Named("url", url), sql.Named("host", host), sql.Named("title", title),
			sql.Named("source", string(titleSource)), sql.Named("description", description),
			sql.Named("search", searchText(url, host, title, description)),
			sql.Named("label", sortLabel(host, title, description)), sql.Named("now", now),
			sql.Named("url_changed", urlChanged), sql.Named("refetch", refetch), sql.Named("id", id))
		if err != nil {
			return fmt.Errorf("update link: %w", err)
		}
		out, err = linkDetailQ(ctx, tx, v, id)
		return err
	})
	if err != nil {
		return model.LinkDetail{}, err
	}
	return out, nil
}

// DeleteLink deletes an own link; tags, shares and every user's state
// cascade. Errors: model.ErrNotOwner, model.ErrNotFound.
func (s *Store) DeleteLink(ctx context.Context, v model.Viewer, id int64) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := requireOwnerQ(ctx, tx, v, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM links WHERE id = :id AND owner_id = :uid`,
			sql.Named("id", id), sql.Named("uid", v.UserID)); err != nil {
			return fmt.Errorf("delete link: %w", err)
		}
		return nil
	})
}

// LinkIcon returns the icon of a visible link; model.ErrNotFound when the
// link is not visible or has no icon.
func (s *Store) LinkIcon(ctx context.Context, v model.Viewer, id int64) (model.IconBlob, error) {
	var ic model.IconBlob
	err := s.db.QueryRowContext(ctx, visibleCTE+`SELECT ic.data, ic.content_type, ic.sha256
		FROM v JOIN links l ON l.id = v.link_id JOIN icons ic ON ic.id = l.icon_id
		WHERE v.link_id = :id`, append(viewerArgs(v), sql.Named("id", id))...).
		Scan(&ic.Data, &ic.ContentType, &ic.SHA256)
	if err != nil {
		return model.IconBlob{}, notFound(err)
	}
	return ic, nil
}

// upsertIconQ stores an icon deduplicated by content and returns its id. An
// icon with a type or size the icons table does not accept returns 0.
func upsertIconQ(ctx context.Context, q querier, ic *model.IconBlob, now string) (int64, error) {
	if ic == nil || len(ic.Data) == 0 || len(ic.Data) > model.MaxIconBytes || !validIconType(ic.ContentType) {
		return 0, nil
	}
	sum := sha256.Sum256(ic.Data)
	hash := hex.EncodeToString(sum[:])
	if _, err := q.ExecContext(ctx, `INSERT INTO icons (sha256, content_type, data, created_at)
		VALUES (:sha, :type, :data, :now) ON CONFLICT (sha256) DO NOTHING`,
		sql.Named("sha", hash), sql.Named("type", ic.ContentType), sql.Named("data", ic.Data),
		sql.Named("now", now)); err != nil {
		return 0, fmt.Errorf("insert icon: %w", err)
	}
	var id int64
	if err := q.QueryRowContext(ctx, `SELECT id FROM icons WHERE sha256 = :sha`,
		sql.Named("sha", hash)).Scan(&id); err != nil {
		return 0, fmt.Errorf("read icon: %w", err)
	}
	return id, nil
}

func validIconType(t string) bool {
	for _, ok := range model.IconContentTypes {
		if t == ok {
			return true
		}
	}
	return false
}

// truncateRunes shortens s to at most n runes.
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}
