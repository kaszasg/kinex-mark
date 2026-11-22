package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// SetPendingTOTP stores a sealed, not yet confirmed TOTP secret that expires
// at expires (2FA setup, 5.8).
func (s *Store) SetPendingTOTP(ctx context.Context, id int64, sealed string, expires time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET totp_pending_secret = :sealed,
		totp_pending_expires_at = :expires, updated_at = :now WHERE id = :id`,
		sql.Named("sealed", sealed), sql.Named("expires", formatTime(expires)),
		sql.Named("now", formatTime(s.now())), sql.Named("id", id))
	if err != nil {
		return fmt.Errorf("set pending totp: %w", err)
	}
	return affectedOne(res)
}

// PendingTOTP returns the pending TOTP secret and its expiry. It returns
// model.ErrNotFound when the user has no pending secret or it has expired.
func (s *Store) PendingTOTP(ctx context.Context, id int64) (string, time.Time, error) {
	var sealed, expires sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT totp_pending_secret, totp_pending_expires_at
		FROM users WHERE id = :id`, sql.Named("id", id)).Scan(&sealed, &expires)
	if err != nil {
		return "", time.Time{}, notFound(err)
	}
	if !sealed.Valid || !expires.Valid {
		return "", time.Time{}, model.ErrNotFound
	}
	exp, err := parseTime(expires.String)
	if err != nil {
		return "", time.Time{}, err
	}
	if !exp.After(s.now()) {
		return "", time.Time{}, model.ErrNotFound
	}
	return sealed.String, exp, nil
}

// EnableTOTP moves the confirmed secret into place, records the accepted
// step and replaces the recovery codes. It returns model.ErrConflict when
// 2FA is already enabled.
func (s *Store) EnableTOTP(ctx context.Context, id int64, sealed string, step int64, codeHashes []string) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		u, err := userByIDQ(ctx, tx, id)
		if err != nil {
			return err
		}
		if u.TOTPEnabled {
			return fmt.Errorf("totp already enabled: %w", model.ErrConflict)
		}
		now := formatTime(s.now())
		if _, err := tx.ExecContext(ctx, `UPDATE users SET totp_secret = :sealed,
				totp_enabled_at = :now, totp_last_step = :step,
				totp_pending_secret = NULL, totp_pending_expires_at = NULL, updated_at = :now
			WHERE id = :id`,
			sql.Named("sealed", sealed), sql.Named("now", now), sql.Named("step", step),
			sql.Named("id", id)); err != nil {
			return fmt.Errorf("enable totp: %w", err)
		}
		return replaceRecoveryCodes(ctx, tx, id, codeHashes, now)
	})
}

// DisableTOTP removes the TOTP secrets and every recovery code. It is a
// no-op for a user without 2FA.
func (s *Store) DisableTOTP(ctx context.Context, id int64) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE users SET totp_secret = NULL, totp_enabled_at = NULL,
				totp_last_step = 0, totp_pending_secret = NULL, totp_pending_expires_at = NULL,
				updated_at = :now
			WHERE id = :id`, sql.Named("now", formatTime(s.now())), sql.Named("id", id))
		if err != nil {
			return fmt.Errorf("disable totp: %w", err)
		}
		if err := affectedOne(res); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM recovery_codes WHERE user_id = :id`,
			sql.Named("id", id)); err != nil {
			return fmt.Errorf("delete recovery codes: %w", err)
		}
		return nil
	})
}

// AdvanceTOTPStep records step as the last accepted TOTP step. It returns
// false when the step is not newer than the stored one (a replay, 6.5).
func (s *Store) AdvanceTOTPStep(ctx context.Context, id, step int64) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET totp_last_step = :step
		WHERE id = :id AND totp_last_step < :step`, sql.Named("step", step), sql.Named("id", id))
	if err != nil {
		return false, fmt.Errorf("advance totp step: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("advance totp step: %w", err)
	}
	return n == 1, nil
}

// ReplaceRecoveryCodes deletes the user's recovery codes and stores new ones.
func (s *Store) ReplaceRecoveryCodes(ctx context.Context, id int64, codeHashes []string) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := userByIDQ(ctx, tx, id); err != nil {
			return err
		}
		return replaceRecoveryCodes(ctx, tx, id, codeHashes, formatTime(s.now()))
	})
}

func replaceRecoveryCodes(ctx context.Context, tx *sql.Tx, id int64, codeHashes []string, now string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM recovery_codes WHERE user_id = :id`,
		sql.Named("id", id)); err != nil {
		return fmt.Errorf("delete recovery codes: %w", err)
	}
	for _, h := range codeHashes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO recovery_codes (user_id, code_hash, created_at)
			VALUES (:id, :hash, :now) ON CONFLICT (user_id, code_hash) DO NOTHING`,
			sql.Named("id", id), sql.Named("hash", h), sql.Named("now", now)); err != nil {
			return fmt.Errorf("insert recovery code: %w", err)
		}
	}
	return nil
}

// UseRecoveryCode marks an unused recovery code as used. It returns false
// when the code is unknown or already used (6.6).
func (s *Store) UseRecoveryCode(ctx context.Context, id int64, codeHash string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE recovery_codes SET used_at = :now
		WHERE user_id = :id AND code_hash = :hash AND used_at IS NULL`,
		sql.Named("now", formatTime(s.now())), sql.Named("id", id), sql.Named("hash", codeHash))
	if err != nil {
		return false, fmt.Errorf("use recovery code: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("use recovery code: %w", err)
	}
	return n == 1, nil
}

// RecoveryCodesRemaining counts the user's unused recovery codes.
func (s *Store) RecoveryCodesRemaining(ctx context.Context, id int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM recovery_codes
		WHERE user_id = :id AND used_at IS NULL`, sql.Named("id", id)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count recovery codes: %w", err)
	}
	return n, nil
}
