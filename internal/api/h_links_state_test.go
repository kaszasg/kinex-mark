package api

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestLinkStatePinArchive(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	d := linkTestCreateURL(t, c, "https://state.example/")
	p := linkTestPath(d.ID)

	e.clock.Advance(6 * time.Hour)
	pinned := decode[LinkItem](t, c.post(p+"/pin", nil))
	if !pinned.Pinned || pinned.ArchiveDueAt != nil || pinned.ArchivedAt != nil {
		t.Errorf("pinned = %+v", pinned)
	}
	// Idempotent.
	if again := decode[LinkItem](t, c.post(p+"/pin", nil)); !again.Pinned {
		t.Error("second pin unpinned the link")
	}

	e.clock.Advance(6 * time.Hour)
	unpinned := decode[LinkItem](t, c.del(p+"/pin"))
	now := e.clock.Now()
	if unpinned.Pinned || !unpinned.InactiveSince.Equal(now) || unpinned.ArchiveDueAt == nil ||
		!unpinned.ArchiveDueAt.Equal(now.AddDate(0, 0, 90)) {
		t.Errorf("unpinned = pinned %v inactive %v due %v, want the period restarted at %v",
			unpinned.Pinned, unpinned.InactiveSince, unpinned.ArchiveDueAt, now)
	}

	archived := decode[LinkItem](t, c.post(p+"/archive", nil))
	if archived.ArchivedAt == nil || archived.ArchiveReason == nil || *archived.ArchiveReason != model.ArchiveManual ||
		archived.ArchiveDueAt != nil || archived.Pinned {
		t.Errorf("archived = %+v", archived)
	}
	if again := decode[LinkItem](t, c.post(p+"/archive", nil)); again.ArchivedAt == nil {
		t.Error("second archive restored the link")
	}

	// Pinning an archived link restores it (4.9).
	e.clock.Advance(time.Hour)
	repinned := decode[LinkItem](t, c.post(p+"/pin", nil))
	if !repinned.Pinned || repinned.ArchivedAt != nil || repinned.ArchiveReason != nil {
		t.Errorf("pin of an archived link = %+v", repinned)
	}

	decode[LinkItem](t, c.post(p+"/archive", nil))
	e.clock.Advance(time.Hour)
	restored := decode[LinkItem](t, c.del(p+"/archive"))
	if restored.ArchivedAt != nil || restored.Pinned || !restored.InactiveSince.Equal(e.clock.Now()) {
		t.Errorf("restored = %+v", restored)
	}
	if again := decode[LinkItem](t, c.del(p+"/archive")); again.ArchivedAt != nil {
		t.Error("second restore archived the link")
	}

	for _, path := range []string{"/pin", "/archive", "/visit", "/tags"} {
		method := http.MethodPost
		if path == "/tags" {
			method = http.MethodPut
		}
		expectError(t, c.do(method, "/api/links/x"+path, map[string]any{"tagIds": []int64{}}), http.StatusBadRequest, codeInvalidID)
		expectError(t, c.do(method, linkTestPath(424242, path), map[string]any{"tagIds": []int64{}}), http.StatusNotFound, codeNotFound)
	}
	expectError(t, c.del("/api/links/0/pin"), http.StatusBadRequest, codeInvalidID)
	expectError(t, c.del(linkTestPath(424242, "/archive")), http.StatusNotFound, codeNotFound)
}

func TestLinkStateVisit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	d := linkTestCreateURL(t, c, "https://visit.example/")
	p := linkTestPath(d.ID)

	for i := 1; i <= 3; i++ {
		e.clock.Advance(time.Minute)
		res := c.post(p+"/visit", nil)
		body := expectStatus(t, res, http.StatusOK)
		st := decode[LinkState](t, res)
		if st.UseCount != int64(i) || st.LastUsedAt == nil || !st.LastUsedAt.Equal(e.clock.Now()) ||
			!st.InactiveSince.Equal(e.clock.Now()) || st.ArchiveDueAt == nil {
			t.Errorf("visit %d = %+v", i, st)
		}
		if i == 1 {
			want := "archiveDueAt,archiveReason,archivedAt,inactiveSince,lastUsedAt,pinned,useCount"
			if got := jsonKeys(t, body); !slices.Equal(got, []string{
				"archiveDueAt", "archiveReason", "archivedAt", "inactiveSince", "lastUsedAt", "pinned", "useCount",
			}) {
				t.Errorf("LinkState keys = %v, want %s", got, want)
			}
		}
	}
	if item := decode[LinkDetail](t, c.get(p)); item.UseCount != 3 {
		t.Errorf("useCount = %d after three visits", item.UseCount)
	}

	// A visit restores an archived link.
	decode[LinkItem](t, c.post(p+"/archive", nil))
	st := decode[LinkState](t, c.post(p+"/visit", nil))
	if st.ArchivedAt != nil || st.UseCount != 4 {
		t.Errorf("visit of an archived link = %+v", st)
	}
	// Pinned: never due.
	decode[LinkItem](t, c.post(p+"/pin", nil))
	if st := decode[LinkState](t, c.post(p+"/visit", nil)); !st.Pinned || st.ArchiveDueAt != nil {
		t.Errorf("visit of a pinned link = %+v", st)
	}
}

func TestLinkStateTags(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	t1 := linkTestCreateTag(t, c, "Egy")
	t2 := linkTestCreateTag(t, c, "Kettő")
	d := linkTestCreateURL(t, c, "https://tags.example/")
	p := linkTestPath(d.ID) + "/tags"

	item := decode[LinkItem](t, c.put(p, map[string]any{"tagIds": []int64{t2.ID, t1.ID}}))
	if got := slices.Sorted(slices.Values(item.TagIDs)); !slices.Equal(got, []int64{t1.ID, t2.ID}) {
		t.Errorf("tagIds = %v", item.TagIDs)
	}
	// Idempotent replacement.
	item = decode[LinkItem](t, c.put(p, map[string]any{"tagIds": []int64{t1.ID}}))
	if !slices.Equal(item.TagIDs, []int64{t1.ID}) {
		t.Errorf("tagIds = %v, want [%d]", item.TagIDs, t1.ID)
	}
	if again := decode[LinkItem](t, c.put(p, map[string]any{"tagIds": []int64{t1.ID}})); !slices.Equal(again.TagIDs, []int64{t1.ID}) {
		t.Errorf("repeated put = %v", again.TagIDs)
	}
	detail := decode[LinkDetail](t, c.get(linkTestPath(d.ID)))
	if len(detail.Tags) != 1 || detail.Tags[0].Name != "Egy" {
		t.Errorf("detail tags = %+v", detail.Tags)
	}
	item = decode[LinkItem](t, c.put(p, map[string]any{"tagIds": []int64{}}))
	if item.TagIDs == nil || len(item.TagIDs) != 0 {
		t.Errorf("cleared tagIds = %v, want []", item.TagIDs)
	}

	many := make([]int64, 21)
	for i := range many {
		many[i] = int64(i + 1)
	}
	expectFields(t, c.put(p, map[string]any{"tagIds": many}), map[string]string{"tagIds": "too_long"})
	expectFields(t, c.put(p, map[string]any{"tagIds": []int64{t1.ID, t1.ID}}), map[string]string{"tagIds": "duplicate"})
	expectFields(t, c.put(p, map[string]any{"tagIds": []int64{-1}}), map[string]string{"tagIds": "invalid"})
	expectFields(t, c.put(p, map[string]any{"tagIds": []int64{424242}}), map[string]string{"tagIds": "unknown"})
	expectFields(t, c.put(p, map[string]any{}), map[string]string{"tagIds": "required"})
	expectFields(t, c.put(p, map[string]any{"tagIds": nil}), map[string]string{"tagIds": "required"})
	expectError(t, c.put(p, map[string]any{"tags": []int64{}}), http.StatusBadRequest, codeInvalidJSON)
	if after := decode[LinkDetail](t, c.get(linkTestPath(d.ID))); len(after.TagIDs) != 0 {
		t.Errorf("rejected requests changed the tags: %v", after.TagIDs)
	}
}
