package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// DueFetches returns up to limit pending links whose next_fetch_at has come,
// oldest first (the sweeper, 7.3).
func (s *Store) DueFetches(ctx context.Context, limit int) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM links
		WHERE fetch_status = 'pending' AND next_fetch_at <= :now
		ORDER BY next_fetch_at, id
		LIMIT :limit`, sql.Named("now", formatTime(s.now())), sql.Named("limit", limit))
	if err != nil {
		return nil, fmt.Errorf("list due fetches: %w", err)
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan due fetch: %w", err)
		}
		ids = append(ids, id)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("list due fetches: %w", err)
	}
	return ids, nil
}

// BeginFetch leases a pending, due link for lease by moving its
// next_fetch_at forward (7.3). It returns model.ErrNotFound when the link is
// not pending or is leased by another worker.
func (s *Store) BeginFetch(ctx context.Context, linkID int64, lease time.Duration) (model.FetchJob, error) {
	now := s.now()
	job := model.FetchJob{LinkID: linkID}
	err := s.db.QueryRowContext(ctx, `UPDATE links SET next_fetch_at = :until
		WHERE id = :id AND fetch_status = 'pending' AND next_fetch_at <= :now
		RETURNING url, owner_id`,
		sql.Named("until", formatTime(now.Add(lease))), sql.Named("id", linkID),
		sql.Named("now", formatTime(now))).Scan(&job.URL, &job.OwnerID)
	if err != nil {
		return model.FetchJob{}, notFound(err)
	}
	var locale string
	err = s.db.QueryRowContext(ctx, `SELECT locale FROM user_preferences WHERE user_id = :uid`,
		sql.Named("uid", job.OwnerID)).Scan(&locale)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		job.Locale = model.DefaultLocale
	case err != nil:
		return model.FetchJob{}, fmt.Errorf("read owner locale: %w", err)
	default:
		job.Locale = model.Locale(locale)
	}
	return job, nil
}

// ApplyFetchResult stores the outcome of a fetch in one transaction (7.3).
// The result is discarded (nil error) when the link no longer exists or its
// URL changed since BeginFetch. A temporary error is retried with backoff
// until model.FetchMaxAttempts and changes neither title nor icon. A final
// result replaces the icon (NULL when none) and, for "ok", a non-manual title.
func (s *Store) ApplyFetchResult(ctx context.Context, r model.FetchResult) error {
	if !r.Temporary && r.Status != model.FetchOK && r.Status != model.FetchFailed && r.Status != model.FetchBlocked {
		return fmt.Errorf("apply fetch result: invalid status %q", r.Status)
	}
	return s.withTx(ctx, func(tx *sql.Tx) error {
		var host, title, source, description string
		var attempts int
		err := tx.QueryRowContext(ctx, `SELECT host, title, title_source, description, fetch_attempts
			FROM links WHERE id = :id AND url = :url`,
			sql.Named("id", r.LinkID), sql.Named("url", r.URL)).
			Scan(&host, &title, &source, &description, &attempts)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read link for fetch result: %w", err)
		}
		now := s.now()
		nowText := formatTime(now)
		attempts++
		fetchErr := nullText(r.Error)

		if r.Temporary && attempts < model.FetchMaxAttempts {
			_, err := tx.ExecContext(ctx, `UPDATE links SET fetch_status = 'pending',
					fetch_error = :error, fetch_attempts = :attempts, next_fetch_at = :next
				WHERE id = :id`,
				sql.Named("error", fetchErr), sql.Named("attempts", attempts),
				sql.Named("next", formatTime(now.Add(model.FetchBackoff(attempts)))),
				sql.Named("id", r.LinkID))
			if err != nil {
				return fmt.Errorf("schedule fetch retry: %w", err)
			}
			return nil
		}

		status := r.Status
		if r.Temporary {
			status = model.FetchFailed
		}
		if status == model.FetchOK && model.TitleSource(source) != model.TitleManual {
			if r.Title != "" {
				title, source = truncateRunes(r.Title, model.TitleMaxRunes), string(model.TitleFetched)
			} else {
				title, source = "", string(model.TitleNone)
			}
		}
		iconID, err := upsertIconQ(ctx, tx, r.Icon, nowText)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE links SET fetch_status = :status, fetch_error = :error,
				fetch_attempts = :attempts, next_fetch_at = NULL, fetched_at = :now,
				icon_id = :icon, title = :title, title_source = :source,
				search_text = :search, sort_label = :label
			WHERE id = :id`,
			sql.Named("status", string(status)), sql.Named("error", fetchErr),
			sql.Named("attempts", attempts), sql.Named("now", nowText),
			sql.Named("icon", nullID(iconID)), sql.Named("title", title),
			sql.Named("source", source),
			sql.Named("search", searchText(r.URL, host, title, description)),
			sql.Named("label", sortLabel(host, title, description)),
			sql.Named("id", r.LinkID))
		if err != nil {
			return fmt.Errorf("apply fetch result: %w", err)
		}
		return nil
	})
}

// ResetFetch marks an own link for fetching again (5.9.7). Errors:
// model.ErrNotOwner, model.ErrNotFound.
func (s *Store) ResetFetch(ctx context.Context, v model.Viewer, id int64) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := requireOwnerQ(ctx, tx, v, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE links SET fetch_status = 'pending',
				fetch_error = NULL, fetch_attempts = 0, next_fetch_at = :now
			WHERE id = :id AND owner_id = :uid`,
			sql.Named("now", formatTime(s.now())), sql.Named("id", id),
			sql.Named("uid", v.UserID)); err != nil {
			return fmt.Errorf("reset fetch: %w", err)
		}
		return nil
	})
}
