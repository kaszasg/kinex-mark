package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFormatParseTimeRoundTrip(t *testing.T) {
	t.Parallel()
	cases := []time.Time{
		time.Date(2026, 11, 21, 9, 15, 0, 123_000_000, time.UTC),
		time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC),
		time.Date(1999, 12, 31, 23, 59, 59, 999_000_000, time.FixedZone("CET", 3600)),
	}
	for _, in := range cases {
		s := FormatTime(in)
		if len(s) != len("2026-11-21T09:15:00.123Z") {
			t.Errorf("FormatTime(%v) = %q: not fixed width", in, s)
		}
		if !strings.HasSuffix(s, "Z") {
			t.Errorf("FormatTime(%v) = %q: not UTC", in, s)
		}
		out, err := ParseTime(s)
		if err != nil {
			t.Fatalf("ParseTime(%q): %v", s, err)
		}
		if !out.Equal(in) || out.Location() != time.UTC {
			t.Errorf("round trip %v -> %q -> %v", in, s, out)
		}
	}
	if got := FormatTime(time.Date(2026, 11, 21, 9, 15, 0, 0, time.UTC)); got != "2026-11-21T09:15:00.000Z" {
		t.Errorf("FormatTime keeps three digits: got %q", got)
	}
}

func TestParseTimeAcceptsRFC3339AndRejectsGarbage(t *testing.T) {
	t.Parallel()
	got, err := ParseTime("2026-11-21T09:15:00.5+01:00")
	if err != nil {
		t.Fatalf("ParseTime: %v", err)
	}
	if want := time.Date(2026, 11, 21, 8, 15, 0, 500_000_000, time.UTC); !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("got %v, want %v", got, want)
	}
	if _, err := ParseTime("yesterday"); err == nil {
		t.Error("ParseTime accepted garbage")
	}
}

func TestTimeLexicalOrderIsChronological(t *testing.T) {
	t.Parallel()
	a := FormatTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	b := FormatTime(time.Date(2026, 1, 1, 0, 0, 0, 1_000_000, time.UTC))
	c := FormatTime(time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC))
	if !(a < b && b < c) {
		t.Errorf("lexical order broken: %q %q %q", a, b, c)
	}
}

func TestTimePtrHelpers(t *testing.T) {
	t.Parallel()
	if FormatTimePtr(nil) != nil {
		t.Error("FormatTimePtr(nil) != nil")
	}
	if got, err := ParseTimePtr(nil); got != nil || err != nil {
		t.Errorf("ParseTimePtr(nil) = %v, %v", got, err)
	}
	now := time.Date(2026, 3, 4, 5, 6, 7, 8_000_000, time.UTC)
	s := FormatTimePtr(&now)
	back, err := ParseTimePtr(s)
	if err != nil || back == nil || !back.Equal(now) {
		t.Errorf("pointer round trip: %v, %v", back, err)
	}
	bad := "nope"
	if _, err := ParseTimePtr(&bad); err == nil {
		t.Error("ParseTimePtr accepted garbage")
	}
}

func TestRolePermissions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		role                      Role
		valid, share, list, admin bool
	}{
		{RoleAdmin, true, true, true, true},
		{RoleDeveloper, true, true, true, false},
		{RoleCustomer, true, false, false, false},
		{Role("root"), false, false, false, false},
		{Role(""), false, false, false, false},
	}
	for _, c := range cases {
		if c.role.Valid() != c.valid || c.role.CanShare() != c.share ||
			c.role.CanListUsers() != c.list || c.role.IsAdmin() != c.admin {
			t.Errorf("role %q: got valid=%v share=%v list=%v admin=%v", c.role,
				c.role.Valid(), c.role.CanShare(), c.role.CanListUsers(), c.role.IsAdmin())
		}
	}
	if len(Roles) != 3 {
		t.Errorf("Roles = %v", Roles)
	}
	for _, r := range Roles {
		if !r.Valid() {
			t.Errorf("listed role %q invalid", r)
		}
	}
}

func TestLocales(t *testing.T) {
	t.Parallel()
	for _, l := range Locales {
		if !l.Valid() {
			t.Errorf("listed locale %q invalid", l)
		}
	}
	if Locale("fr").Valid() || !DefaultLocale.Valid() {
		t.Error("locale validity wrong")
	}
}

func TestTagColors(t *testing.T) {
	t.Parallel()
	want := []TagColor{"blue", "ochre", "green", "violet", "rose", "neutral", "contrast"}
	if fmt.Sprint(TagColors) != fmt.Sprint(want) {
		t.Fatalf("TagColors = %v, want %v", TagColors, want)
	}
	for _, c := range TagColors {
		if !c.Valid() {
			t.Errorf("%q invalid", c)
		}
	}
	for _, c := range []TagColor{"", "red", "primary", "Blue"} {
		if c.Valid() {
			t.Errorf("%q accepted", c)
		}
	}
	if !DefaultTagColor.Valid() {
		t.Error("default color invalid")
	}
}

func TestShareTargetTypes(t *testing.T) {
	t.Parallel()
	for _, s := range ShareTargetTypes {
		if !s.Valid() {
			t.Errorf("%q invalid", s)
		}
	}
	if ShareTargetType("group").Valid() {
		t.Error("unknown share type accepted")
	}
}

func assertUnique(t *testing.T, name string, list []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, v := range list {
		if v == "" {
			t.Errorf("%s: empty entry", name)
		}
		if seen[v] {
			t.Errorf("%s: duplicate %q", name, v)
		}
		seen[v] = true
	}
}

func TestAuditEventsUnique(t *testing.T) {
	t.Parallel()
	if len(AuditEvents) != 28 {
		t.Errorf("AuditEvents has %d codes, want 28", len(AuditEvents))
	}
	assertUnique(t, "AuditEvents", AuditEvents)
	for _, e := range AuditEvents {
		if len(e) > AuditEventMaxLen {
			t.Errorf("event %q longer than %d", e, AuditEventMaxLen)
		}
	}
	assertUnique(t, "AuditDetailKeys", AuditDetailKeys)
	if len(AuditDetailKeys) != 15 {
		t.Errorf("AuditDetailKeys has %d keys, want 15", len(AuditDetailKeys))
	}
	keys := map[string]bool{}
	for _, k := range AuditDetailKeys {
		keys[k] = true
	}
	for k, values := range AuditDetailValues {
		if !keys[k] {
			t.Errorf("AuditDetailValues key %q is not a detail key", k)
		}
		assertUnique(t, "AuditDetailValues."+k, values)
	}
	assertUnique(t, "FetchErrors", FetchErrors)
	if len(FetchErrors) != 10 {
		t.Errorf("FetchErrors has %d codes, want 10", len(FetchErrors))
	}
}

func TestValidUsername(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"abc":                   true,
		"Admin.User_1-x":        true,
		strings.Repeat("a", 64): true,
		"ab":                    false,
		strings.Repeat("a", 65): false,
		"with space":            false,
		"slash/name":            false,
		"árvíztűrő":             false,
		"":                      false,
	}
	for in, want := range cases {
		if got := ValidUsername(in); got != want {
			t.Errorf("ValidUsername(%q) = %v, want %v", in, got, want)
		}
	}
	if UsernameKey("Admin") != "admin" || TagNameKey("ÁLOM") != "álom" {
		t.Error("keys are not lower-cased")
	}
}

func TestUserHelpers(t *testing.T) {
	t.Parallel()
	u := User{ID: 7, Username: "dev", DisplayName: "Dev", Role: RoleDeveloper}
	if r := u.Ref(); r != (UserRef{ID: 7, Username: "dev", DisplayName: "Dev"}) {
		t.Errorf("Ref() = %+v", r)
	}
	if v := u.Viewer(); v != (Viewer{UserID: 7, Role: RoleDeveloper}) {
		t.Errorf("Viewer() = %+v", v)
	}
	b, err := json.Marshal(DirectoryUser{UserRef: u.Ref(), Role: u.Role})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"id":7,"username":"dev","displayName":"Dev","role":"developer"}` {
		t.Errorf("DirectoryUser JSON = %s", b)
	}
}

func TestDefaultPreferences(t *testing.T) {
	t.Parallel()
	p := DefaultPreferences(90)
	if p.Locale != LocaleHU || p.Theme != ThemeDark || p.ViewMode != ViewList || p.Sort != SortUsage ||
		p.TagMatch != TagMatchAny || !p.ShowIcon || p.ShowArchived || p.ArchiveAfterDays != nil ||
		p.EffectiveArchiveAfterDays != 90 || p.SystemArchiveAfterDays != 90 {
		t.Errorf("unexpected defaults: %+v", p)
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"archiveAfterDays":null`) {
		t.Errorf("archiveAfterDays must encode as null: %s", b)
	}
}

func TestPreferencesPatchNullableArchiveDays(t *testing.T) {
	t.Parallel()
	cases := []struct {
		body  string
		set   bool
		value *int
		bad   bool
	}{
		{body: `{}`},
		{body: `{"archiveAfterDays":null}`, set: true},
		{body: `{"archiveAfterDays":30}`, set: true, value: intPtr(30)},
		{body: `{"archiveAfterDays":"x"}`, bad: true},
	}
	for _, c := range cases {
		var p PreferencesPatch
		err := json.Unmarshal([]byte(c.body), &p)
		if c.bad {
			if err == nil {
				t.Errorf("%s: expected error", c.body)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", c.body, err)
		}
		if p.ArchiveAfterDays.Set != c.set {
			t.Errorf("%s: Set = %v", c.body, p.ArchiveAfterDays.Set)
		}
		if (p.ArchiveAfterDays.Value == nil) != (c.value == nil) ||
			(c.value != nil && *p.ArchiveAfterDays.Value != *c.value) {
			t.Errorf("%s: Value = %v", c.body, p.ArchiveAfterDays.Value)
		}
	}
	for _, c := range []struct {
		in   NullableInt
		want string
	}{
		{NullableInt{}, "null"},
		{NullableInt{Set: true}, "null"},
		{NullableInt{Set: true, Value: intPtr(5)}, "5"},
	} {
		b, err := json.Marshal(c.in)
		if err != nil || string(b) != c.want {
			t.Errorf("Marshal(%+v) = %s, %v", c.in, b, err)
		}
	}
}

func intPtr(v int) *int { return &v }

func TestSystemSettingsDefaultsAndPatchKeys(t *testing.T) {
	t.Parallel()
	d := DefaultSystemSettings()
	if d.DefaultArchiveDays != 90 || d.AuditRetentionDays != 180 || d.SessionIdleMinutes != 720 ||
		d.SessionAbsoluteHours != 168 || d.AllowPrivateFetch || d.PrivacyNoticeVersion != 1 {
		t.Errorf("defaults: %+v", d)
	}
	for name, r := range map[string]struct {
		r IntRange
		v int
	}{
		"archive":   {DefaultArchiveDaysRange, d.DefaultArchiveDays},
		"retention": {AuditRetentionDaysRange, d.AuditRetentionDays},
		"idle":      {SessionIdleMinutesRange, d.SessionIdleMinutes},
		"absolute":  {SessionAbsoluteHoursRange, d.SessionAbsoluteHours},
	} {
		if !r.r.Contains(r.v) || r.r.Contains(r.r.Min-1) || r.r.Contains(r.r.Max+1) {
			t.Errorf("%s: range %+v and default %d disagree", name, r.r, r.v)
		}
	}

	if keys := (SettingsPatch{}).Keys(); len(keys) != 0 {
		t.Errorf("empty patch keys = %v", keys)
	}
	days, allow, name := 30, true, "ACME"
	p := SettingsPatch{DefaultArchiveDays: &days, AllowPrivateFetch: &allow, PrivacyControllerName: &name}
	if got := strings.Join(p.Keys(), ","); got != "defaultArchiveDays,allowPrivateFetch,privacyControllerName" {
		t.Errorf("Keys() = %s", got)
	}
	all := SettingsPatch{
		DefaultArchiveDays: &days, AuditRetentionDays: &days, SessionIdleMinutes: &days,
		SessionAbsoluteHours: &days, AllowPrivateFetch: &allow, PrivacyControllerName: &name,
		PrivacyControllerAddress: &name, PrivacyControllerEmail: &name, PrivacyDpoContact: &name,
		PrivacySupervisoryAuthority: &name,
	}
	b, err := json.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	if len(all.Keys()) != len(fields) {
		t.Errorf("Keys() %v does not match JSON fields %v", all.Keys(), fields)
	}
	for _, k := range all.Keys() {
		if _, ok := fields[k]; !ok {
			t.Errorf("key %q is not a JSON field name", k)
		}
	}
}

func TestDuplicateError(t *testing.T) {
	t.Parallel()
	var err error = &DuplicateError{Kind: DuplicateLink, ExistingID: 42}
	wrapped := fmt.Errorf("create link: %w", err)
	var dup *DuplicateError
	if !errors.As(wrapped, &dup) || dup.ExistingID != 42 || dup.Kind != DuplicateLink {
		t.Fatalf("errors.As failed: %v", dup)
	}
	if !strings.Contains(err.Error(), "42") {
		t.Errorf("Error() = %q", err.Error())
	}
	if (&DuplicateError{Kind: DuplicateUsername}).Error() != "duplicate username" {
		t.Error("Error() without id")
	}
}

func TestPage(t *testing.T) {
	t.Parallel()
	p := NewPage[int](nil, 0, PageReq{Limit: 50})
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"items":[],"total":0,"limit":50,"offset":0}` {
		t.Errorf("empty page JSON = %s", b)
	}
	p2 := NewPage([]string{"a"}, 3, PageReq{Limit: 1, Offset: 2})
	if len(p2.Items) != 1 || p2.Total != 3 || p2.Offset != 2 {
		t.Errorf("page = %+v", p2)
	}
}

func TestFetchBackoff(t *testing.T) {
	t.Parallel()
	if FetchBackoff(1) != time.Minute || FetchBackoff(2) != 10*time.Minute || FetchBackoff(0) != time.Minute {
		t.Error("unexpected backoff")
	}
}

func TestLinkJSONShape(t *testing.T) {
	t.Parallel()
	d := LinkDetail{
		LinkItem: LinkItem{
			ID: 1, URL: "https://example.com/", Host: "example.com", FetchStatus: FetchPending,
			TagIDs: []int64{}, BaselineAt: time.Now(), IconSHA256: "abc",
		},
		TitleSource: TitleManual, Tags: []Tag{}, Shares: []Share{},
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"id", "url", "host", "title", "description", "fetchStatus", "tagIds", "pinned",
		"useCount", "lastUsedAt", "archivedAt", "archiveReason", "isOwner", "owner", "shareCount", "sharedVia",
		"sharedAt", "createdAt", "updatedAt", "titleSource", "tags", "fetchError", "fetchedAt", "fetchAttempts", "shares"} {
		if _, ok := m[k]; !ok {
			t.Errorf("LinkDetail JSON lacks %q", k)
		}
	}
	for _, k := range []string{"BaselineAt", "IconSHA256", "baselineAt", "iconSha256"} {
		if _, ok := m[k]; ok {
			t.Errorf("LinkDetail JSON leaks %q", k)
		}
	}
}

func TestAuditEntryJSONShape(t *testing.T) {
	t.Parallel()
	id := int64(3)
	e := AuditEntry{
		ID: 1, At: time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC), Event: EventUserCreated, Success: true,
		Actor:   AuditActor{ID: &id, Username: "admin"},
		Target:  &AuditTarget{Type: TargetUser, ID: &id, Name: "dev"},
		Details: map[string]any{DetailRole: "developer"},
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":1,"at":"2026-01-05T10:00:00Z","event":"user.created","success":true,` +
		`"actor":{"id":3,"username":"admin"},"target":{"type":"user","id":3,"name":"dev"},` +
		`"ip":"","details":{"role":"developer"}}`
	if string(b) != want {
		t.Errorf("AuditEntry JSON =\n%s\nwant\n%s", b, want)
	}
}
