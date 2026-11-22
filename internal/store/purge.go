package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Purge removes expired data statement by statement (7.4), so that the
// single connection is never held for long. auditRetention <= 0 falls back
// to the auditRetentionDays setting.
func (s *Store) Purge(ctx context.Context, auditRetention time.Duration) (model.PurgeResult, error) {
	var res model.PurgeResult
	settings, err := s.Settings(ctx)
	if err != nil {
		return res, err
	}
	if auditRetention <= 0 {
		auditRetention = time.Duration(settings.AuditRetentionDays) * 24 * time.Hour
	}
	now := s.now()
	steps := []struct {
		name  string
		query string
		args  []any
		count *int64
	}{
		{"sessions", `DELETE FROM sessions WHERE expires_at < :now OR last_seen_at < :idle_cutoff`,
			[]any{sql.Named("now", formatTime(now)), sql.Named("idle_cutoff",
				formatTime(now.Add(-time.Duration(settings.SessionIdleMinutes)*time.Minute)))},
			&res.Sessions},
		{"challenges", `DELETE FROM login_challenges WHERE expires_at < :now`,
			[]any{sql.Named("now", formatTime(now))}, &res.Challenges},
		{"throttle", `DELETE FROM login_throttle
			WHERE (locked_until IS NULL OR locked_until < :now) AND updated_at < :day_ago`,
			[]any{sql.Named("now", formatTime(now)), sql.Named("day_ago", formatTime(now.Add(-24*time.Hour)))},
			&res.Throttle},
		{"pending totp", `UPDATE users SET totp_pending_secret = NULL, totp_pending_expires_at = NULL
			WHERE totp_pending_expires_at < :now`,
			[]any{sql.Named("now", formatTime(now))}, &res.PendingTOTP},
		{"audit", `DELETE FROM audit_log WHERE at < :audit_cutoff`,
			[]any{sql.Named("audit_cutoff", formatTime(now.Add(-auditRetention)))}, &res.Audit},
		{"icons", `DELETE FROM icons WHERE NOT EXISTS (SELECT 1 FROM links WHERE links.icon_id = icons.id)`,
			nil, &res.Icons},
	}
	for _, step := range steps {
		r, err := s.db.ExecContext(ctx, step.query, step.args...)
		if err != nil {
			return res, fmt.Errorf("purge %s: %w", step.name, err)
		}
		if *step.count, err = r.RowsAffected(); err != nil {
			return res, fmt.Errorf("purge %s: %w", step.name, err)
		}
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA optimize`); err != nil {
		return res, fmt.Errorf("purge optimize: %w", err)
	}
	return res, nil
}
