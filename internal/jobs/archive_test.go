package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestRunArchiveUsesConfiguredDefault(t *testing.T) {
	t.Parallel()
	for _, days := range []int{90, 0, 3650} {
		h := newHarness(t, Options{}, noFetch)
		h.st.settings.DefaultArchiveDays = days
		h.st.archiveN = 4
		n, err := h.r.RunArchive(context.Background())
		if err != nil || n != 4 {
			t.Fatalf("days %d: RunArchive = %d, %v; want 4, nil", days, n, err)
		}
		if got := await(t, h.st.archiveCalls, "archive"); got != days {
			t.Errorf("ArchiveInactive defaultDays = %d, want %d", got, days)
		}
		recs := h.log.find(t, "archive run")
		if len(recs) != 1 || recs[0]["level"] != "INFO" || recs[0]["archived"] != float64(4) {
			t.Errorf("archive run log = %v", recs)
		}
	}
}

func TestRunArchiveErrors(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	tests := []struct {
		name        string
		settingsErr error
		archiveErr  error
		wantPrefix  string
	}{
		{"settings", boom, nil, "archive: read settings: boom"},
		{"archive", nil, boom, "archive: boom"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, Options{}, noFetch)
			h.st.settingsErr, h.st.archiveErr = tc.settingsErr, tc.archiveErr
			n, err := h.r.RunArchive(context.Background())
			if n != 0 || !errors.Is(err, boom) || err.Error() != tc.wantPrefix {
				t.Fatalf("RunArchive = %d, %v; want 0, %q", n, err, tc.wantPrefix)
			}
			recs := h.log.find(t, "archive run failed")
			if len(recs) != 1 || recs[0]["level"] != "ERROR" {
				t.Fatalf("error log = %v", recs)
			}
			if len(h.log.find(t, "archive run")) != 0 {
				t.Fatal("success record logged for a failed run")
			}
		})
	}
}

func TestRunPurgeUsesAuditRetention(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{}, noFetch)
	h.st.settings.AuditRetentionDays = 30
	want := model.PurgeResult{Sessions: 1, Challenges: 2, Throttle: 3, PendingTOTP: 4, Audit: 5, Icons: 6}
	h.st.purgeRes = want
	got, err := h.r.RunPurge(context.Background())
	if err != nil || got != want {
		t.Fatalf("RunPurge = %+v, %v", got, err)
	}
	if ret := await(t, h.st.purgeCalls, "purge"); ret != 30*24*time.Hour {
		t.Errorf("auditRetention = %v, want 720h", ret)
	}
	recs := h.log.find(t, "purge run")
	if len(recs) != 1 || recs[0]["level"] != "INFO" {
		t.Fatalf("purge run log = %v", recs)
	}
	for key, n := range map[string]float64{"sessions": 1, "challenges": 2, "throttle": 3, "pending_totp": 4, "audit": 5, "icons": 6} {
		if recs[0][key] != n {
			t.Errorf("purge run %s = %v, want %v", key, recs[0][key], n)
		}
	}
}

func TestRunPurgeErrors(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	tests := []struct {
		name        string
		settingsErr error
		purgeErr    error
		want        string
	}{
		{"settings", boom, nil, "purge: read settings: boom"},
		{"purge", nil, boom, "purge: boom"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, Options{}, noFetch)
			h.st.settingsErr, h.st.purgeErr = tc.settingsErr, tc.purgeErr
			h.st.purgeRes = model.PurgeResult{Sessions: 9}
			res, err := h.r.RunPurge(context.Background())
			if res != (model.PurgeResult{}) || !errors.Is(err, boom) || err.Error() != tc.want {
				t.Fatalf("RunPurge = %+v, %v; want zero, %q", res, err, tc.want)
			}
			if recs := h.log.find(t, "purge run failed"); len(recs) != 1 || recs[0]["level"] != "ERROR" {
				t.Fatalf("error log = %v", recs)
			}
		})
	}
}
