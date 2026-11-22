package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// metaKeyCheck is the only app_meta key: the keyring check value (4.5).
const metaKeyCheck = "key_check"

// KeyCheck returns the stored keyring check value, or "" when none is stored yet.
func (s *Store) KeyCheck(ctx context.Context) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM app_meta WHERE key = :key`, sql.Named("key", metaKeyCheck)).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read key check: %w", err)
	}
	return v, nil
}

// SetKeyCheck stores the keyring check value.
func (s *Store) SetKeyCheck(ctx context.Context, v string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO app_meta (key, value) VALUES (:key, :value)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`,
		sql.Named("key", metaKeyCheck), sql.Named("value", v))
	if err != nil {
		return fmt.Errorf("write key check: %w", err)
	}
	return nil
}

// Settings returns the system settings: the stored non-default values merged
// over model.DefaultSystemSettings (4.5).
func (s *Store) Settings(ctx context.Context) (model.SystemSettings, error) {
	return settingsQ(ctx, s.db)
}

func settingsQ(ctx context.Context, q querier) (model.SystemSettings, error) {
	settings := model.DefaultSystemSettings()
	rows, err := q.QueryContext(ctx, `SELECT key, value FROM system_settings`)
	if err != nil {
		return settings, fmt.Errorf("read settings: %w", err)
	}
	stored := map[string]json.RawMessage{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			_ = rows.Close()
			return settings, fmt.Errorf("scan setting: %w", err)
		}
		stored[key] = json.RawMessage(value)
	}
	if err := rows.Close(); err != nil {
		return settings, fmt.Errorf("read settings: %w", err)
	}
	if err := rows.Err(); err != nil {
		return settings, fmt.Errorf("read settings: %w", err)
	}
	if len(stored) == 0 {
		return settings, nil
	}
	merged, err := json.Marshal(stored)
	if err != nil {
		return settings, fmt.Errorf("merge settings: %w", err)
	}
	// Unknown keys are ignored, so a setting removed in a later version does
	// not break reading.
	if err := json.Unmarshal(merged, &settings); err != nil {
		return model.DefaultSystemSettings(), fmt.Errorf("decode settings: %w", err)
	}
	return settings, nil
}

// UpdateSettings applies a partial update. A value equal to the built-in
// default removes the stored row, so only non-default values are kept (4.5).
// by is the acting admin (0: none).
func (s *Store) UpdateSettings(ctx context.Context, p model.SettingsPatch, by int64) (model.SystemSettings, error) {
	values, err := toJSONFields(p)
	if err != nil {
		return model.SystemSettings{}, fmt.Errorf("encode settings patch: %w", err)
	}
	var out model.SystemSettings
	err = s.withTx(ctx, func(tx *sql.Tx) error {
		if err := writeSettings(ctx, tx, values, by, s.now()); err != nil {
			return err
		}
		out, err = settingsQ(ctx, tx)
		return err
	})
	if err != nil {
		return model.SystemSettings{}, err
	}
	return out, nil
}

// BumpPrivacyNotice increments the privacy notice version and records the
// time, so every user has to acknowledge the new notice (consent_required).
func (s *Store) BumpPrivacyNotice(ctx context.Context, by int64) (model.SystemSettings, error) {
	var out model.SystemSettings
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		cur, err := settingsQ(ctx, tx)
		if err != nil {
			return err
		}
		now := s.now()
		version, _ := json.Marshal(cur.PrivacyNoticeVersion + 1)
		at, _ := json.Marshal(formatTime(now))
		values := map[string]json.RawMessage{
			model.SettingPrivacyNoticeVersion:   version,
			model.SettingPrivacyNoticeUpdatedAt: at,
		}
		if err := writeSettings(ctx, tx, values, by, now); err != nil {
			return err
		}
		out, err = settingsQ(ctx, tx)
		return err
	})
	if err != nil {
		return model.SystemSettings{}, err
	}
	return out, nil
}

// writeSettings stores each value, or deletes its row when it equals the default.
func writeSettings(ctx context.Context, tx *sql.Tx, values map[string]json.RawMessage, by int64, now time.Time) error {
	defaults, err := toJSONFields(model.DefaultSystemSettings())
	if err != nil {
		return fmt.Errorf("encode default settings: %w", err)
	}
	for key, value := range values {
		if def, ok := defaults[key]; ok && bytes.Equal(def, value) {
			if _, err := tx.ExecContext(ctx, `DELETE FROM system_settings WHERE key = :key`,
				sql.Named("key", key)); err != nil {
				return fmt.Errorf("reset setting %s: %w", key, err)
			}
			continue
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO system_settings (key, value, updated_at, updated_by)
			VALUES (:key, :value, :now, :by)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value,
				updated_at = excluded.updated_at, updated_by = excluded.updated_by`,
			sql.Named("key", key), sql.Named("value", string(value)),
			sql.Named("now", formatTime(now)), sql.Named("by", nullID(by)))
		if err != nil {
			return fmt.Errorf("write setting %s: %w", key, err)
		}
	}
	return nil
}

// toJSONFields encodes a struct to its JSON fields (omitempty drops the
// absent fields of a patch).
func toJSONFields(v any) (map[string]json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(b, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}
