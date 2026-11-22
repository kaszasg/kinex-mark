package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Preferences returns the user's preferences with the effective and the
// system archive period; model.ErrNotFound when the user does not exist.
func (s *Store) Preferences(ctx context.Context, userID int64) (model.Preferences, error) {
	return preferencesQ(ctx, s.db, userID)
}

func preferencesQ(ctx context.Context, q querier, userID int64) (model.Preferences, error) {
	settings, err := settingsQ(ctx, q)
	if err != nil {
		return model.Preferences{}, err
	}
	var p model.Preferences
	var locale string
	var days sql.NullInt64
	err = q.QueryRowContext(ctx, `SELECT locale, theme, view_mode, show_icon, show_url, show_title,
			show_description, show_archived, sort, tag_match, archive_after_days
		FROM user_preferences WHERE user_id = :uid`, sql.Named("uid", userID)).
		Scan(&locale, &p.Theme, &p.ViewMode, &p.ShowIcon, &p.ShowURL, &p.ShowTitle,
			&p.ShowDescription, &p.ShowArchived, &p.Sort, &p.TagMatch, &days)
	if err != nil {
		return model.Preferences{}, notFound(err)
	}
	p.Locale = model.Locale(locale)
	p.SystemArchiveAfterDays = settings.DefaultArchiveDays
	p.EffectiveArchiveAfterDays = settings.DefaultArchiveDays
	if days.Valid {
		d := int(days.Int64)
		p.ArchiveAfterDays = &d
		p.EffectiveArchiveAfterDays = d
	}
	return p, nil
}

// UpdatePreferences applies a partial update; absent fields stay unchanged
// and an explicit null archiveAfterDays restores the system default.
func (s *Store) UpdatePreferences(ctx context.Context, userID int64, p model.PreferencesPatch) (model.Preferences, error) {
	var locale any
	if p.Locale != nil {
		locale = string(*p.Locale)
	}
	var days any
	if p.ArchiveAfterDays.Set && p.ArchiveAfterDays.Value != nil {
		days = *p.ArchiveAfterDays.Value
	}
	var out model.Preferences
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE user_preferences SET
				locale = coalesce(:locale, locale),
				theme = coalesce(:theme, theme),
				view_mode = coalesce(:view_mode, view_mode),
				show_icon = coalesce(:show_icon, show_icon),
				show_url = coalesce(:show_url, show_url),
				show_title = coalesce(:show_title, show_title),
				show_description = coalesce(:show_description, show_description),
				show_archived = coalesce(:show_archived, show_archived),
				sort = coalesce(:sort, sort),
				tag_match = coalesce(:tag_match, tag_match),
				archive_after_days = CASE WHEN :days_set THEN :days ELSE archive_after_days END,
				updated_at = :now
			WHERE user_id = :uid`,
			sql.Named("locale", locale), sql.Named("theme", optString(p.Theme)),
			sql.Named("view_mode", optString(p.ViewMode)), sql.Named("show_icon", optBool(p.ShowIcon)),
			sql.Named("show_url", optBool(p.ShowURL)), sql.Named("show_title", optBool(p.ShowTitle)),
			sql.Named("show_description", optBool(p.ShowDescription)),
			sql.Named("show_archived", optBool(p.ShowArchived)), sql.Named("sort", optString(p.Sort)),
			sql.Named("tag_match", optString(p.TagMatch)),
			sql.Named("days_set", p.ArchiveAfterDays.Set), sql.Named("days", days),
			sql.Named("now", formatTime(s.now())), sql.Named("uid", userID))
		if err != nil {
			return fmt.Errorf("update preferences: %w", err)
		}
		if err := affectedOne(res); err != nil {
			return err
		}
		out, err = preferencesQ(ctx, tx, userID)
		return err
	})
	if err != nil {
		return model.Preferences{}, err
	}
	return out, nil
}

// optString binds an optional string (nil: NULL).
func optString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

// optBool binds an optional bool (nil: NULL).
func optBool(p *bool) any {
	if p == nil {
		return nil
	}
	return *p
}
