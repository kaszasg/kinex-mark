package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// linkTestTime returns T0 plus the given number of days, in a non-UTC zone
// so that the conversion to UTC is observable.
func linkTestTime(days int) time.Time {
	zone := time.FixedZone("CET", 3600)
	return time.Date(2026, 1, 5, 11, 0, 0, 0, zone).AddDate(0, 0, days)
}

func linkTestPtr[T any](v T) *T { return &v }

func TestLinkDTOToLinkItem(t *testing.T) {
	t.Parallel()
	const sha = "3fa2c1d0e5b64a7f9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b"
	base := linkTestTime(0)
	later := linkTestTime(10)
	earlier := linkTestTime(-3)
	archivedAt := linkTestTime(20)
	auto := model.ArchiveAuto

	cases := []struct {
		name         string
		item         model.LinkItem
		days         int
		wantInactive time.Time
		wantDue      *time.Time
		wantIcon     *string
	}{
		{
			name:         "never used: baseline counts, due after the period",
			item:         model.LinkItem{ID: 7, BaselineAt: base},
			days:         90,
			wantInactive: base.UTC(),
			wantDue:      linkTestPtr(base.UTC().AddDate(0, 0, 90)),
		},
		{
			name:         "last use later than the baseline wins",
			item:         model.LinkItem{ID: 7, BaselineAt: base, LastUsedAt: &later},
			days:         30,
			wantInactive: later.UTC(),
			wantDue:      linkTestPtr(later.UTC().AddDate(0, 0, 30)),
		},
		{
			name:         "last use before the baseline (restored link) keeps the baseline",
			item:         model.LinkItem{ID: 7, BaselineAt: base, LastUsedAt: &earlier},
			days:         30,
			wantInactive: base.UTC(),
			wantDue:      linkTestPtr(base.UTC().AddDate(0, 0, 30)),
		},
		{
			name:         "pinned link is never due",
			item:         model.LinkItem{ID: 7, BaselineAt: base, Pinned: true},
			days:         90,
			wantInactive: base.UTC(),
		},
		{
			name:         "archived link is never due",
			item:         model.LinkItem{ID: 7, BaselineAt: base, ArchivedAt: &archivedAt, ArchiveReason: &auto},
			days:         90,
			wantInactive: base.UTC(),
		},
		{
			name:         "zero days means never",
			item:         model.LinkItem{ID: 7, BaselineAt: base},
			days:         0,
			wantInactive: base.UTC(),
		},
		{
			name:         "icon url from the first 8 hex characters",
			item:         model.LinkItem{ID: 12, BaselineAt: base, IconSHA256: sha},
			days:         0,
			wantInactive: base.UTC(),
			wantIcon:     linkTestPtr("/api/links/12/icon?v=3fa2c1d0"),
		},
		{
			name:         "short hash is used as a whole",
			item:         model.LinkItem{ID: 12, BaselineAt: base, IconSHA256: "abc"},
			days:         0,
			wantInactive: base.UTC(),
			wantIcon:     linkTestPtr("/api/links/12/icon?v=abc"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toLinkItem(tc.item, tc.days)
			if !got.InactiveSince.Equal(tc.wantInactive) || got.InactiveSince.Location() != time.UTC {
				t.Errorf("inactiveSince = %v, want %v in UTC", got.InactiveSince, tc.wantInactive)
			}
			switch {
			case tc.wantDue == nil && got.ArchiveDueAt != nil:
				t.Errorf("archiveDueAt = %v, want nil", *got.ArchiveDueAt)
			case tc.wantDue != nil && (got.ArchiveDueAt == nil || !got.ArchiveDueAt.Equal(*tc.wantDue)):
				t.Errorf("archiveDueAt = %v, want %v", got.ArchiveDueAt, *tc.wantDue)
			}
			switch {
			case tc.wantIcon == nil && got.IconURL != nil:
				t.Errorf("iconUrl = %q, want nil", *got.IconURL)
			case tc.wantIcon != nil && (got.IconURL == nil || *got.IconURL != *tc.wantIcon):
				t.Errorf("iconUrl = %v, want %q", got.IconURL, *tc.wantIcon)
			}
			if got.TagIDs == nil {
				t.Error("tagIds is nil, want an empty list")
			}
		})
	}
}

func TestLinkDTOCopiesFieldsInUTC(t *testing.T) {
	t.Parallel()
	used := linkTestTime(1)
	archived := linkTestTime(2)
	shared := linkTestTime(-1)
	via := model.ShareRole
	manual := model.ArchiveManual
	m := model.LinkItem{
		ID: 3, URL: "https://example.com/", Host: "example.com", Title: "Example",
		Description: "Leírás", FetchStatus: model.FetchOK, TagIDs: []int64{4, 5},
		UseCount: 9, LastUsedAt: &used, ArchivedAt: &archived, ArchiveReason: &manual,
		IsOwner: false, Owner: model.UserRef{ID: 2, Username: "owner", DisplayName: "Owner"},
		ShareCount: 0, SharedVia: &via, SharedAt: &shared,
		CreatedAt: linkTestTime(-5), UpdatedAt: linkTestTime(-4), BaselineAt: shared,
	}
	got := toLinkItem(m, 90)
	if got.ID != 3 || got.URL != m.URL || got.Host != m.Host || got.Title != m.Title ||
		got.Description != m.Description || got.FetchStatus != model.FetchOK || got.UseCount != 9 ||
		got.IsOwner || got.Owner != m.Owner || got.ShareCount != 0 || *got.SharedVia != via ||
		*got.ArchiveReason != manual || len(got.TagIDs) != 2 {
		t.Fatalf("fields not copied: %+v", got)
	}
	for name, ts := range map[string]time.Time{
		"lastUsedAt": *got.LastUsedAt, "archivedAt": *got.ArchivedAt, "sharedAt": *got.SharedAt,
		"createdAt": got.CreatedAt, "updatedAt": got.UpdatedAt,
	} {
		if ts.Location() != time.UTC {
			t.Errorf("%s is not UTC: %v", name, ts)
		}
	}
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"archiveDueAt", "archiveReason", "archivedAt", "createdAt", "description", "fetchStatus", "host",
		"iconUrl", "id", "inactiveSince", "isOwner", "lastUsedAt", "owner", "pinned", "shareCount",
		"sharedAt", "sharedVia", "tagIds", "title", "updatedAt", "url", "useCount",
	}
	if keys := strings.Join(jsonKeys(t, body), ","); keys != strings.Join(want, ",") {
		t.Errorf("LinkItem keys = %s\nwant %s", keys, strings.Join(want, ","))
	}
	if !strings.Contains(string(body), `"createdAt":"2025-12-31T10:00:00Z"`) {
		t.Errorf("createdAt not encoded in UTC: %s", body)
	}
}

func TestLinkDTODetailAndState(t *testing.T) {
	t.Parallel()
	base := linkTestTime(0)
	d := toLinkDetail(model.LinkDetail{
		LinkItem:    model.LinkItem{ID: 1, BaselineAt: base},
		TitleSource: model.TitleManual, FetchAttempts: 2,
	}, 10)
	if d.Tags == nil || d.Shares == nil {
		t.Fatalf("nil lists in detail: tags %v shares %v", d.Tags, d.Shares)
	}
	if d.TitleSource != model.TitleManual || d.FetchAttempts != 2 || d.FetchedAt != nil || d.FetchError != nil {
		t.Errorf("detail fields = %+v", d)
	}
	body, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	keys := jsonKeys(t, body)
	for _, k := range []string{"titleSource", "tags", "fetchError", "fetchedAt", "fetchAttempts", "shares", "inactiveSince", "iconUrl"} {
		found := false
		for _, have := range keys {
			found = found || have == k
		}
		if !found {
			t.Errorf("LinkDetail JSON lacks %q: %s", k, body)
		}
	}

	used := linkTestTime(3)
	st := toLinkState(model.LinkState{UseCount: 1, LastUsedAt: &used, BaselineAt: base}, 7)
	if !st.InactiveSince.Equal(used) || st.ArchiveDueAt == nil || !st.ArchiveDueAt.Equal(used.AddDate(0, 0, 7)) {
		t.Errorf("state = %+v", st)
	}
	pinned := toLinkState(model.LinkState{Pinned: true, BaselineAt: base}, 7)
	if pinned.ArchiveDueAt != nil {
		t.Errorf("pinned state due = %v", pinned.ArchiveDueAt)
	}
	body, err = json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if keys := strings.Join(jsonKeys(t, body), ","); keys != "archiveDueAt,archiveReason,archivedAt,inactiveSince,lastUsedAt,pinned,useCount" {
		t.Errorf("LinkState keys = %s", keys)
	}
}

func TestLinkDTOPage(t *testing.T) {
	t.Parallel()
	p := toLinkPage(model.LinkPage{Total: 3, Limit: 99, Offset: 98, Counts: model.LinkCounts{Pinned: 1, Archived: 2}},
		model.LinkQuery{Limit: 2, Offset: 1}, 0)
	if p.Items == nil || len(p.Items) != 0 || p.Total != 3 || p.Limit != 2 || p.Offset != 1 || p.Counts.Archived != 2 {
		t.Errorf("page = %+v", p)
	}
}
