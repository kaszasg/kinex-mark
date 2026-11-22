package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// ExportLinks returns the owner's links with the owner's tag names, oldest
// first (GET /api/links/export).
func (s *Store) ExportLinks(ctx context.Context, ownerID int64) ([]model.ExportedLink, error) {
	var out []model.ExportedLink
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		tagNames, err := tagNamesByLinkQ(ctx, tx, ownerID)
		if err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT id, url, title, description, created_at
			FROM links WHERE owner_id = :uid ORDER BY created_at, id`, sql.Named("uid", ownerID))
		if err != nil {
			return fmt.Errorf("export links: %w", err)
		}
		out = []model.ExportedLink{}
		for rows.Next() {
			var id int64
			var l model.ExportedLink
			var created string
			if err := rows.Scan(&id, &l.URL, &l.Title, &l.Description, &created); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan export link: %w", err)
			}
			if l.CreatedAt, err = parseTime(created); err != nil {
				_ = rows.Close()
				return err
			}
			l.Tags = namesOf(tagNames, id)
			out = append(out, l)
		}
		if err := closeRows(rows); err != nil {
			return fmt.Errorf("export links: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ImportLinks creates the owner's links from bookmark entries in one
// transaction (5.9.11) and returns the counts and the new link ids. The
// caller normalises the URLs and drops (and counts) invalid entries; the
// store still skips an entry whose URL or host does not fit the column
// limits and counts it as invalid. An URL the owner already has, or that
// appears earlier in the same import, is a duplicate. When the new links
// would exceed model.MaxLinksPerUser nothing is created and the error is
// model.ErrLimitReached. Missing tags are created with the neutral color
// while the tag limit allows. Title and description lose control and bidi
// characters and are cut to the column limits; a non-empty title is manual.
// New links are pending (next_fetch_at = now) for the sweeper. A link keeps
// the entry's AddedAt as its creation time when it lies between the Unix
// epoch and now (importCreatedAt), but its inactivity baseline is
// the import time, so old bookmarks are not archived at once.
func (s *Store) ImportLinks(ctx context.Context, ownerID int64, entries []model.ImportEntry) (model.ImportResult, []int64, error) {
	result := model.ImportResult{Total: len(entries)}
	ids := []int64{}
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := userRoleQ(ctx, tx, ownerID); err != nil {
			return err
		}
		existing, err := ownURLsQ(ctx, tx, ownerID)
		if err != nil {
			return err
		}
		var fresh []model.ImportEntry
		for _, e := range entries {
			switch {
			case !validImportURL(e.URL, e.Host):
				result.Invalid++
			case existing[e.URL]:
				result.Duplicates++
			default:
				existing[e.URL] = true
				fresh = append(fresh, e)
			}
		}
		count, err := ownLinkCountQ(ctx, tx, ownerID)
		if err != nil {
			return err
		}
		if count+len(fresh) > model.MaxLinksPerUser {
			return model.ErrLimitReached
		}
		tags, tagCount, err := ownTagKeysQ(ctx, tx, ownerID)
		if err != nil {
			return err
		}
		now := s.now()
		nowText := formatTime(now)
		for _, e := range fresh {
			title := cleanImportText(e.Title, false, model.TitleMaxRunes)
			description := cleanImportText(e.Description, true, model.DescriptionMaxLen)
			created := importCreatedAt(e.AddedAt, now)
			id, err := insertLinkQ(ctx, tx, ownerID, e.URL, e.Host, title, description, created, nowText)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO link_user_state (user_id, link_id,
					baseline_at, updated_at)
				VALUES (:uid, :id, :now, :now)`,
				sql.Named("uid", ownerID), sql.Named("id", id), sql.Named("now", nowText)); err != nil {
				return fmt.Errorf("insert link state: %w", err)
			}
			var tagIDs []int64
			linkTags := map[int64]bool{}
			for _, name := range e.Tags {
				if len(tagIDs) >= model.MaxTagsPerLink {
					break
				}
				name = truncateRunes(cleanImportText(name, false, model.TagNameMaxLen), model.TagNameMaxLen)
				if name == "" {
					continue
				}
				key := model.TagNameKey(name)
				tagID, ok := tags[key]
				if !ok {
					if tagCount >= model.MaxTagsPerUser {
						continue
					}
					if tagID, err = insertTagQ(ctx, tx, ownerID, name, model.DefaultTagColor, nowText); err != nil {
						return err
					}
					tags[key] = tagID
					tagCount++
				}
				if !linkTags[tagID] {
					linkTags[tagID] = true
					tagIDs = append(tagIDs, tagID)
				}
			}
			if err := insertLinkTagsQ(ctx, tx, id, tagIDs, nowText); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		result.Created = len(ids)
		return nil
	})
	if err != nil {
		return model.ImportResult{}, nil, err
	}
	return result, ids, nil
}

// minImportCreatedAt is the earliest creation time an import keeps (the Unix
// epoch, as for the Netscape ADD_DATE). A negative year in UTC would be
// written as "-0001-…", which model.ParseTime cannot read back.
var minImportCreatedAt = time.Unix(0, 0).UTC()

// importCreatedAt returns the stored creation time of an imported link: the
// entry's AddedAt when it lies in [1970-01-01T00:00:00Z, now), else now.
func importCreatedAt(added *time.Time, now time.Time) string {
	if added == nil || added.Before(minImportCreatedAt) || !added.Before(now) {
		return formatTime(now)
	}
	return formatTime(*added)
}

func validImportURL(url, host string) bool {
	return len(url) >= 9 && len(url) <= model.URLMaxBytes &&
		utf8.RuneCountInString(host) >= 1 && utf8.RuneCountInString(host) <= model.HostMaxLen
}

// ownURLsQ returns the set of the owner's link URLs.
func ownURLsQ(ctx context.Context, q querier, ownerID int64) (map[string]bool, error) {
	rows, err := q.QueryContext(ctx, `SELECT url FROM links WHERE owner_id = :uid`,
		sql.Named("uid", ownerID))
	if err != nil {
		return nil, fmt.Errorf("read link urls: %w", err)
	}
	out := map[string]bool{}
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan link url: %w", err)
		}
		out[url] = true
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("read link urls: %w", err)
	}
	return out, nil
}

// ownTagKeysQ maps the owner's tag name keys to ids and returns their count.
func ownTagKeysQ(ctx context.Context, q querier, ownerID int64) (map[string]int64, int, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, name_key FROM tags WHERE owner_id = :uid`,
		sql.Named("uid", ownerID))
	if err != nil {
		return nil, 0, fmt.Errorf("read tags: %w", err)
	}
	out := map[string]int64{}
	for rows.Next() {
		var id int64
		var key string
		if err := rows.Scan(&id, &key); err != nil {
			_ = rows.Close()
			return nil, 0, fmt.Errorf("scan tag: %w", err)
		}
		out[key] = id
	}
	if err := closeRows(rows); err != nil {
		return nil, 0, fmt.Errorf("read tags: %w", err)
	}
	return out, len(out), nil
}

// cleanImportText applies the text rule of 6.8 to imported text as a second
// line of defence behind fetch.CleanText (the store cannot import fetch):
// C0 and C1 controls, U+007F and the bidi controls (the marks ALM, LRM and
// RLM, the embeddings, overrides and isolates) are removed (a newline is kept when keepNewlines), runs of spaces collapse,
// the result is trimmed and cut to max runes.
func cleanImportText(s string, keepNewlines bool, max int) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		switch {
		case r == '\n' && keepNewlines:
			b.WriteRune('\n')
			space = false
			continue
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if !space {
				b.WriteRune(' ')
			}
			space = true
			continue
		case r < 0x20, r >= 0x7f && r <= 0x9f, r == 0x061c, r == 0x200e, r == 0x200f,
			r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
			continue
		}
		b.WriteRune(r)
		space = false
	}
	out := strings.TrimSpace(b.String())
	if keepNewlines {
		lines := strings.Split(out, "\n")
		for i, l := range lines {
			lines[i] = strings.TrimSpace(l)
		}
		out = strings.TrimSpace(strings.Join(lines, "\n"))
	}
	return strings.TrimSpace(truncateRunes(out, max))
}
