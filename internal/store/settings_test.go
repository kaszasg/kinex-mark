package store

import (
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestKeyCheck(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	v, err := st.KeyCheck(ctx)
	if err != nil || v != "" {
		t.Fatalf("empty KeyCheck = %q, %v", v, err)
	}
	mustNoErr(t, st.SetKeyCheck(ctx, "abc"))
	mustNoErr(t, st.SetKeyCheck(ctx, "def"))
	if v, _ := st.KeyCheck(ctx); v != "def" {
		t.Errorf("KeyCheck = %q", v)
	}
}

func TestSettingsMergeAndUpdate(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	admin := seedUser(t, st, model.RoleAdmin)

	s, err := st.Settings(ctx)
	mustNoErr(t, err)
	if s != model.DefaultSystemSettings() {
		t.Fatalf("defaults = %+v", s)
	}

	s, err = st.UpdateSettings(ctx, model.SettingsPatch{
		DefaultArchiveDays:          ptr(30),
		AllowPrivateFetch:           ptr(true),
		PrivacyControllerName:       ptr("Kinexium Kft."),
		PrivacySupervisoryAuthority: ptr("NAIH"),
	}, admin.ID)
	mustNoErr(t, err)
	if s.DefaultArchiveDays != 30 || !s.AllowPrivateFetch || s.PrivacyControllerName != "Kinexium Kft." ||
		s.AuditRetentionDays != 180 || s.PrivacyNoticeVersion != 1 {
		t.Fatalf("updated = %+v", s)
	}
	if n := countRows(t, st, `SELECT COUNT(*) FROM system_settings WHERE updated_by = ?`, admin.ID); n != 4 {
		t.Errorf("stored rows = %d", n)
	}

	// Setting a value back to its default removes the row.
	s, err = st.UpdateSettings(ctx, model.SettingsPatch{DefaultArchiveDays: ptr(90), AllowPrivateFetch: ptr(false)}, 0)
	mustNoErr(t, err)
	if s.DefaultArchiveDays != 90 || s.AllowPrivateFetch {
		t.Errorf("reset = %+v", s)
	}
	if n := countRows(t, st, `SELECT COUNT(*) FROM system_settings`); n != 2 {
		t.Errorf("rows after reset = %d, want 2", n)
	}
	s, _ = st.UpdateSettings(ctx, model.SettingsPatch{}, admin.ID)
	if s.PrivacySupervisoryAuthority != "NAIH" {
		t.Errorf("empty patch = %+v", s)
	}

	// Bump the privacy notice twice.
	clk.Advance(time.Hour)
	s, err = st.BumpPrivacyNotice(ctx, admin.ID)
	mustNoErr(t, err)
	if s.PrivacyNoticeVersion != 2 || s.PrivacyNoticeUpdatedAt == nil || !s.PrivacyNoticeUpdatedAt.Equal(clk.Now()) {
		t.Fatalf("bump = %+v", s)
	}
	s, _ = st.BumpPrivacyNotice(ctx, 0)
	if s.PrivacyNoticeVersion != 3 {
		t.Errorf("second bump = %d", s.PrivacyNoticeVersion)
	}
	again, _ := st.Settings(ctx)
	if again.PrivacyNoticeVersion != 3 || again.PrivacyControllerName != "Kinexium Kft." {
		t.Errorf("reread = %+v", again)
	}

	// Unknown stored keys are ignored; undecodable values are an error.
	_, err = st.db.Exec(`INSERT INTO system_settings (key, value, updated_at) VALUES ('removedSetting', '1', 'x')`)
	mustNoErr(t, err)
	if _, err := st.Settings(ctx); err != nil {
		t.Errorf("unknown key broke Settings: %v", err)
	}
	_, err = st.db.Exec(`INSERT INTO system_settings (key, value, updated_at) VALUES ('defaultArchiveDays', '"many"', 'x')
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`)
	mustNoErr(t, err)
	if _, err := st.Settings(ctx); err == nil {
		t.Error("wrongly typed setting accepted")
	}
	if _, err := st.Preferences(ctx, admin.ID); err == nil {
		t.Error("Preferences ignored a settings error")
	}
	if _, err := st.Purge(ctx, time.Hour); err == nil {
		t.Error("Purge ignored a settings error")
	}
}

func TestPreferences(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)

	p, err := st.Preferences(ctx, u.ID)
	mustNoErr(t, err)
	want := model.DefaultPreferences(90)
	if p.Locale != want.Locale || p.Theme != want.Theme || p.ViewMode != want.ViewMode || !p.ShowIcon ||
		!p.ShowURL || !p.ShowTitle || !p.ShowDescription || p.ShowArchived || p.Sort != want.Sort ||
		p.TagMatch != want.TagMatch || p.ArchiveAfterDays != nil || p.EffectiveArchiveAfterDays != 90 ||
		p.SystemArchiveAfterDays != 90 {
		t.Fatalf("defaults = %+v", p)
	}
	_, err = st.Preferences(ctx, 9999)
	wantErr(t, err, model.ErrNotFound)

	p, err = st.UpdatePreferences(ctx, u.ID, model.PreferencesPatch{
		Locale: ptr(model.LocaleEN), Theme: ptr(model.ThemeLight), ViewMode: ptr(model.ViewTiles),
		ShowIcon: ptr(false), ShowURL: ptr(false), ShowTitle: ptr(false), ShowDescription: ptr(false),
		ShowArchived: ptr(true), Sort: ptr(model.SortTitle), TagMatch: ptr(model.TagMatchAll),
		ArchiveAfterDays: model.NullableInt{Set: true, Value: ptr(14)},
	})
	mustNoErr(t, err)
	if p.Locale != model.LocaleEN || p.Theme != model.ThemeLight || p.ViewMode != model.ViewTiles || p.ShowIcon ||
		p.ShowURL || p.ShowTitle || p.ShowDescription || !p.ShowArchived || p.Sort != model.SortTitle ||
		p.TagMatch != model.TagMatchAll || p.ArchiveAfterDays == nil || *p.ArchiveAfterDays != 14 ||
		p.EffectiveArchiveAfterDays != 14 || p.SystemArchiveAfterDays != 90 {
		t.Fatalf("updated = %+v", p)
	}

	// Absent fields stay; the archive period stays while absent.
	p, err = st.UpdatePreferences(ctx, u.ID, model.PreferencesPatch{Theme: ptr(model.ThemeSystem)})
	mustNoErr(t, err)
	if p.Theme != model.ThemeSystem || p.Locale != model.LocaleEN || p.ArchiveAfterDays == nil || *p.ArchiveAfterDays != 14 {
		t.Fatalf("partial = %+v", p)
	}
	// An explicit null restores the system default, which follows the setting.
	_, err = st.UpdateSettings(ctx, model.SettingsPatch{DefaultArchiveDays: ptr(45)}, 0)
	mustNoErr(t, err)
	p, err = st.UpdatePreferences(ctx, u.ID, model.PreferencesPatch{ArchiveAfterDays: model.NullableInt{Set: true}})
	mustNoErr(t, err)
	if p.ArchiveAfterDays != nil || p.EffectiveArchiveAfterDays != 45 || p.SystemArchiveAfterDays != 45 {
		t.Fatalf("null period = %+v", p)
	}
	// 0 means never and is kept as an explicit value.
	p, _ = st.UpdatePreferences(ctx, u.ID, model.PreferencesPatch{ArchiveAfterDays: model.NullableInt{Set: true, Value: ptr(0)}})
	if p.ArchiveAfterDays == nil || p.EffectiveArchiveAfterDays != 0 {
		t.Errorf("zero period = %+v", p)
	}

	_, err = st.UpdatePreferences(ctx, u.ID, model.PreferencesPatch{Theme: ptr("neon")})
	if err == nil {
		t.Error("theme CHECK not enforced")
	}
	_, err = st.UpdatePreferences(ctx, 9999, model.PreferencesPatch{Theme: ptr(model.ThemeDark)})
	wantErr(t, err, model.ErrNotFound)
}
