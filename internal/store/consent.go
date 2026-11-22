package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// RecordConsent records that the user acknowledged a privacy notice version;
// recording the same version again is a no-op.
func (s *Store) RecordConsent(ctx context.Context, userID int64, kind string, version int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO consent_records (user_id, kind, version, accepted_at)
		VALUES (:uid, :kind, :version, :now)
		ON CONFLICT (user_id, kind, version) DO NOTHING`,
		sql.Named("uid", userID), sql.Named("kind", kind), sql.Named("version", version),
		sql.Named("now", formatTime(s.now())))
	if err != nil {
		return fmt.Errorf("record consent: %w", err)
	}
	return nil
}

// Consents lists the user's acknowledgements, oldest first.
func (s *Store) Consents(ctx context.Context, userID int64) ([]model.Consent, error) {
	return consentsQ(ctx, s.db, userID)
}

func consentsQ(ctx context.Context, q querier, userID int64) ([]model.Consent, error) {
	rows, err := q.QueryContext(ctx, `SELECT kind, version, accepted_at FROM consent_records
		WHERE user_id = :uid ORDER BY accepted_at, id`, sql.Named("uid", userID))
	if err != nil {
		return nil, fmt.Errorf("list consents: %w", err)
	}
	items := []model.Consent{}
	for rows.Next() {
		var c model.Consent
		var at string
		if err := rows.Scan(&c.Kind, &c.Version, &at); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan consent: %w", err)
		}
		if c.AcceptedAt, err = parseTime(at); err != nil {
			_ = rows.Close()
			return nil, err
		}
		items = append(items, c)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("list consents: %w", err)
	}
	return items, nil
}
