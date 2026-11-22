package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

func TestAuditAppendAndFilter(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	admin := seedUser(t, st, model.RoleAdmin)
	u := seedUser(t, st, model.RoleCustomer)

	add := func(e model.AuditEntry) {
		t.Helper()
		clk.Advance(time.Minute)
		mustNoErr(t, st.AppendAudit(ctx, e))
	}
	add(model.AuditEntry{Event: model.EventLoginFailure, Success: false, IP: "203.0.113.9",
		Details: map[string]any{model.DetailReason: model.ReasonUnknownUser}})
	add(model.AuditEntry{Event: model.EventLoginSuccess, Success: true,
		Actor:   model.AuditActor{ID: &u.ID, Username: u.Username},
		Details: map[string]any{model.DetailMethod: model.MethodPassword}})
	add(model.AuditEntry{Event: model.EventUserUpdated, Success: true,
		Actor:   model.AuditActor{ID: &admin.ID, Username: admin.Username},
		Target:  &model.AuditTarget{Type: model.TargetUser, ID: &u.ID, Name: strings.Repeat("n", 80)},
		Details: map[string]any{model.DetailChanged: []string{"role"}}})
	explicit := testutil.T0.Add(-time.Hour)
	add(model.AuditEntry{At: explicit, Event: model.EventSettingsUpdated, Success: true,
		Actor:  model.AuditActor{ID: &admin.ID, Username: admin.Username},
		Target: &model.AuditTarget{Type: model.TargetSettings}})

	all, total, err := st.Audit(ctx, model.AuditFilter{}, model.PageReq{Limit: 50})
	mustNoErr(t, err)
	if total != 4 || len(all) != 4 || all[0].Event != model.EventSettingsUpdated {
		t.Fatalf("audit = %d %+v", total, all)
	}
	if !all[0].At.Equal(explicit) || all[0].Target == nil || all[0].Target.ID != nil || len(all[0].Details) != 0 {
		t.Errorf("explicit at / empty details = %+v", all[0])
	}
	upd := all[1]
	if upd.Target == nil || upd.Target.Type != model.TargetUser || *upd.Target.ID != u.ID ||
		len(upd.Target.Name) != model.AuditNameMaxLen || upd.Actor.ID == nil || *upd.Actor.ID != admin.ID {
		t.Errorf("user.updated = %+v", upd)
	}
	if changed, ok := upd.Details[model.DetailChanged].([]any); !ok || len(changed) != 1 || changed[0] != "role" {
		t.Errorf("details = %#v", upd.Details)
	}
	fail := all[3]
	if fail.Actor.ID != nil || fail.Actor.Username != "" || fail.Target != nil || fail.Success || fail.IP != "203.0.113.9" {
		t.Errorf("anonymous failure = %+v", fail)
	}

	from, to := testutil.T0.Add(2*time.Minute), testutil.T0.Add(3*time.Minute)
	cases := []struct {
		name string
		f    model.AuditFilter
		want int
	}{
		{"exact", model.AuditFilter{Event: model.EventLoginSuccess}, 1},
		{"prefix", model.AuditFilter{Event: "auth.*"}, 2},
		{"bare star", model.AuditFilter{Event: "*"}, 4},
		{"actor", model.AuditFilter{ActorID: &admin.ID}, 2},
		{"failed", model.AuditFilter{Success: ptr(false)}, 1},
		{"succeeded", model.AuditFilter{Success: ptr(true)}, 3},
		{"from-to inclusive", model.AuditFilter{From: &from, To: &to}, 2},
		{"none", model.AuditFilter{Event: "nothing"}, 0},
	}
	for _, c := range cases {
		items, n, err := st.Audit(ctx, c.f, model.PageReq{Limit: 50})
		mustNoErr(t, err)
		if n != c.want || len(items) != c.want {
			t.Errorf("%s: total %d items %d, want %d", c.name, n, len(items), c.want)
		}
	}
	page, n, err := st.Audit(ctx, model.AuditFilter{}, model.PageReq{Limit: 3, Offset: 3})
	mustNoErr(t, err)
	if n != 4 || len(page) != 1 || page[0].Event != model.EventLoginFailure {
		t.Errorf("page = %d %+v", n, page)
	}
	page, n, _ = st.Audit(ctx, model.AuditFilter{}, model.PageReq{Limit: 3, Offset: 30})
	if n != 4 || len(page) != 0 || page == nil {
		t.Errorf("past last page = %d %v", n, page)
	}

	if err := st.AppendAudit(ctx, model.AuditEntry{Event: strings.Repeat("e", 65)}); err == nil {
		t.Error("event length CHECK not enforced")
	}
	if err := st.AppendAudit(ctx, model.AuditEntry{Event: "x", Details: map[string]any{"bad": func() {}}}); err == nil {
		t.Error("unencodable details accepted")
	}
}

func TestConsents(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	list, err := st.Consents(ctx, u.ID)
	mustNoErr(t, err)
	if list == nil || len(list) != 0 {
		t.Fatalf("empty consents = %v", list)
	}
	mustNoErr(t, st.RecordConsent(ctx, u.ID, model.ConsentPrivacyNotice, 1))
	clk.Advance(time.Hour)
	mustNoErr(t, st.RecordConsent(ctx, u.ID, model.ConsentPrivacyNotice, 1)) // no-op
	mustNoErr(t, st.RecordConsent(ctx, u.ID, model.ConsentPrivacyNotice, 2))
	list, _ = st.Consents(ctx, u.ID)
	if len(list) != 2 || list[0].Version != 1 || !list[0].AcceptedAt.Equal(testutil.T0) || list[1].Version != 2 {
		t.Errorf("consents = %+v", list)
	}
	if err := st.RecordConsent(ctx, u.ID, "marketing", 1); err == nil {
		t.Error("consent kind CHECK not enforced")
	}
}

func TestExportUser(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	admin := seedUser(t, st, model.RoleAdmin)
	owner := seedUser(t, st, model.RoleDeveloper)
	u := seedUser(t, st, model.RoleDeveloper)
	v := u.Viewer()

	mustNoErr(t, st.RecordConsent(ctx, u.ID, model.ConsentPrivacyNotice, 1))
	_, err := st.CreateSession(ctx, model.NewSession{TokenHash: make32(5), UserID: u.ID,
		ExpiresAt: clk.Now().Add(time.Hour), IP: "192.0.2.10", UserAgent: "Firefox"})
	mustNoErr(t, err)
	mustNoErr(t, st.EnableTOTP(ctx, u.ID, "v1:totp-secret", 1, []string{hash64('r')}))
	tag := seedTag(t, st, u, "Fontos")
	own, err := st.CreateLink(ctx, model.NewLink{OwnerID: u.ID, URL: "https://own.example/", Host: "own.example",
		Title: "Own", TagIDs: []int64{tag.ID}})
	mustNoErr(t, err)
	share(t, st, u, own.ID, shareUser(admin.ID))
	share(t, st, u, own.ID, shareRole(model.RoleCustomer))
	mustNoErr(t, st.ApplyFetchResult(ctx, model.FetchResult{LinkID: own.ID, URL: own.URL, Status: model.FetchOK,
		Icon: &model.IconBlob{Data: []byte("icon-bytes"), ContentType: "image/png"}}))

	// Personal state on two shared links; one share is revoked later.
	kept := seedLink(t, st, owner, "Kept")
	revoked := seedLink(t, st, owner, "Revoked")
	share(t, st, owner, kept.ID, shareAll())
	sh := share(t, st, owner, revoked.ID, shareUser(u.ID))
	mustNoErr(t, st.SetPinned(ctx, v, kept.ID, true))
	_, err = st.RecordVisit(ctx, v, revoked.ID)
	mustNoErr(t, err)
	mustNoErr(t, st.SetLinkTags(ctx, v, revoked.ID, []int64{tag.ID}))
	mustNoErr(t, st.DeleteShare(ctx, owner.Viewer(), revoked.ID, sh.ID))

	// Audit: u as actor, u as target, unrelated.
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventLoginFailure, IP: "192.0.2.10",
		Actor: model.AuditActor{ID: &u.ID, Username: u.Username}, Details: map[string]any{"reason": "credentials"}}))
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventUserPasswordReset, Success: true, IP: "198.51.100.7",
		Actor:  model.AuditActor{ID: &admin.ID, Username: admin.Username},
		Target: &model.AuditTarget{Type: model.TargetUser, ID: &u.ID, Name: u.Username}}))
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventLoginSuccess, Success: true,
		Actor: model.AuditActor{ID: &admin.ID, Username: admin.Username}}))

	lock := clk.Now().Add(30 * time.Minute)
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: "u:export", UserID: &u.ID}},
		func(r model.ThrottleRow) (model.ThrottleRow, error) {
			r.Failures, r.LockedUntil = 5, &lock
			return r, nil
		}))

	exp, err := st.ExportUser(ctx, u.ID)
	mustNoErr(t, err)
	if exp.Format != model.UserExportFormat || !exp.ExportedAt.Equal(clk.Now()) || exp.User.ID != u.ID ||
		exp.User.Username != u.Username || !exp.User.TOTPEnabled || !exp.User.Active {
		t.Errorf("header/user = %+v", exp)
	}
	if exp.Preferences.Locale != model.LocaleHU || len(exp.Consents) != 1 || len(exp.Sessions) != 1 ||
		exp.Sessions[0].UserAgent != "Firefox" || len(exp.Tags) != 1 || exp.Tags[0].Name != "Fontos" {
		t.Errorf("prefs/consents/sessions/tags = %+v", exp)
	}
	if len(exp.Links) != 1 {
		t.Fatalf("links = %+v", exp.Links)
	}
	ol := exp.Links[0]
	if ol.URL != own.URL || len(ol.Tags) != 1 || ol.Tags[0] != "Fontos" || len(ol.Shares) != 2 ||
		ol.Shares[0].TargetUsername == nil || *ol.Shares[0].TargetUsername != admin.Username ||
		ol.Shares[1].TargetRole == nil || *ol.Shares[1].TargetRole != model.RoleCustomer || ol.Shares[1].TargetUsername != nil {
		t.Errorf("own link = %+v", ol)
	}
	if len(exp.SharedWithMe) != 2 {
		t.Fatalf("sharedWithMe = %+v", exp.SharedWithMe)
	}
	for _, s := range exp.SharedWithMe {
		switch s.LinkID {
		case kept.ID:
			if s.URL == nil || *s.URL != kept.URL || !s.Pinned || len(s.Tags) != 0 {
				t.Errorf("kept = %+v", s)
			}
		case revoked.ID:
			if s.URL != nil || s.UseCount != 1 || len(s.Tags) != 1 || s.Tags[0] != "Fontos" {
				t.Errorf("revoked = %+v (URL must be hidden)", s)
			}
		default:
			t.Errorf("unexpected shared link %+v", s)
		}
	}
	if len(exp.AuditEvents) != 2 || exp.AuditEvents[0].Role != model.ExportAuditActor ||
		exp.AuditEvents[1].Role != model.ExportAuditTarget || exp.AuditEvents[1].Event != model.EventUserPasswordReset {
		t.Errorf("audit events = %+v", exp.AuditEvents)
	}
	// The user's own actions keep their IP; the administrator's IP in a
	// target event is another person's data (Art. 15(4)).
	if len(exp.AuditEvents) == 2 && (exp.AuditEvents[0].IP != "192.0.2.10" || exp.AuditEvents[1].IP != "") {
		t.Errorf("audit IPs = %q, %q; want the own IP and \"\"", exp.AuditEvents[0].IP, exp.AuditEvents[1].IP)
	}
	if exp.Security.LockedUntil == nil || !exp.Security.LockedUntil.Equal(lock) {
		t.Errorf("security = %+v", exp.Security)
	}

	// No secrets or icon data anywhere in the JSON.
	b, err := json.Marshal(exp)
	mustNoErr(t, err)
	for _, secret := range []string{"v1:totp-secret", hash64('r'), "icon-bytes", "aWNvbi1ieXRlcw", "hash", revoked.URL, "198.51.100.7"} {
		if strings.Contains(string(b), secret) {
			t.Errorf("export contains %q", secret)
		}
	}

	_, err = st.ExportUser(ctx, 9999)
	wantErr(t, err, model.ErrNotFound)
}

// Erased account data must not stay readable in the database files (free
// space of app.db, old page images in app.db-wal), which are also what a
// backup of the data directory copies (6.11, 6.12).
func TestDeleteUserLeavesNoRemnants(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := Open(ctx, dir, Options{Clock: testutil.NewClock()})
	mustNoErr(t, err)
	t.Cleanup(func() { _ = st.Close() })
	seedUser(t, st, model.RoleAdmin)
	keep := seedUser(t, st, model.RoleCustomer)
	seedLink(t, st, keep, "Kept link")

	needles := []string{"erasure.probe", "Erasure Probe Display", "probe-password-hash-value",
		"https://erase.example/private?token=probe-secret", "Probe tag name", "Probe user agent"}
	u, err := st.CreateUser(ctx, model.NewUser{Username: needles[0], DisplayName: needles[1],
		Role: model.RoleDeveloper, PasswordHash: needles[2]})
	mustNoErr(t, err)
	l, err := st.CreateLink(ctx, model.NewLink{OwnerID: u.ID, URL: needles[3], Host: "erase.example", Title: "Private"})
	mustNoErr(t, err)
	share(t, st, u, l.ID, shareAll())
	seedTag(t, st, u, needles[4])
	_, err = st.CreateSession(ctx, model.NewSession{TokenHash: make32(7), UserID: u.ID, UserAgent: needles[5],
		ExpiresAt: testutil.T0.Add(time.Hour)})
	mustNoErr(t, err)
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventLoginSuccess, Success: true,
		Actor: model.AuditActor{ID: &u.ID, Username: u.Username}}))

	mustNoErr(t, st.DeleteUser(ctx, u.ID, model.AuditEntry{Event: model.EventUserDeleted, Success: true,
		Actor:  model.AuditActor{ID: &u.ID, Username: u.Username},
		Target: &model.AuditTarget{Type: model.TargetUser, ID: &u.ID, Name: u.Username}}))

	scan := func(when string) {
		t.Helper()
		for _, name := range []string{DBFileName, DBFileName + "-wal"} {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			mustNoErr(t, err)
			for _, n := range needles {
				if c := bytes.Count(data, []byte(n)); c > 0 {
					t.Errorf("%s: %s still holds %q %d times", when, name, n, c)
				}
			}
		}
	}
	scan("after DeleteUser")
	mustNoErr(t, st.Close())
	scan("after Close")
	// The kept data is still there.
	data, err := os.ReadFile(filepath.Join(dir, DBFileName))
	mustNoErr(t, err)
	if !bytes.Contains(data, []byte(keep.Username)) {
		t.Error("the remaining user's data is missing from app.db")
	}
}

func TestDeleteUserPseudonymises(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	admin := seedUser(t, st, model.RoleAdmin)
	u, err := st.CreateUser(ctx, model.NewUser{Username: "Zoltan.K", DisplayName: "Zoltan Kiss",
		Role: model.RoleDeveloper, PasswordHash: "h"})
	mustNoErr(t, err)
	rcpt := seedUser(t, st, model.RoleCustomer)

	// Data that must cascade.
	l := seedLink(t, st, u, "Shared by Zoltan")
	share(t, st, u, l.ID, shareAll())
	mustNoErr(t, st.SetPinned(ctx, rcpt.Viewer(), l.ID, true))
	seedTag(t, st, u, "Tag")
	mustNoErr(t, st.RecordConsent(ctx, u.ID, model.ConsentPrivacyNotice, 1))
	_, err = st.CreateSession(ctx, model.NewSession{TokenHash: make32(8), UserID: u.ID, ExpiresAt: testutil.T0.Add(time.Hour)})
	mustNoErr(t, err)
	mustNoErr(t, st.CreateChallenge(ctx, make32(9), u.ID, "", testutil.T0.Add(time.Minute)))
	mustNoErr(t, st.ReplaceRecoveryCodes(ctx, u.ID, []string{hash64('q')}))
	mustNoErr(t, st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: "u:zoltan", UserID: &u.ID}},
		func(r model.ThrottleRow) (model.ThrottleRow, error) { r.Failures = 1; return r, nil }))
	other := seedLink(t, st, rcpt, "Recipient's own")
	_, err = st.RecordVisit(ctx, u.Viewer(), l.ID)
	mustNoErr(t, err)

	// Audit rows naming the user.
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventLoginSuccess, Success: true,
		Actor: model.AuditActor{ID: &u.ID, Username: u.Username}}))
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventUserUpdated, Success: true,
		Actor:   model.AuditActor{ID: &admin.ID, Username: admin.Username},
		Target:  &model.AuditTarget{Type: model.TargetUser, ID: &u.ID, Name: u.Username},
		Details: map[string]any{model.DetailChanged: []string{"displayName"}}}))
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventUserPasswordReset, Success: true,
		Actor:   model.AuditActor{Username: "zoltan.k"}, // an actor_id IS NULL row with the name
		Details: map[string]any{model.DetailVia: model.ViaCLI}}))
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: model.EventLoginSuccess, Success: true,
		Actor: model.AuditActor{ID: &admin.ID, Username: admin.Username}}))

	// The last admin cannot be deleted.
	wantErr(t, st.DeleteUser(ctx, admin.ID, model.AuditEntry{Event: model.EventUserSelfDeleted}), model.ErrLastAdmin)
	wantErr(t, st.DeleteUser(ctx, 9999, model.AuditEntry{Event: model.EventUserDeleted}), model.ErrNotFound)
	if n := countRows(t, st, `SELECT COUNT(*) FROM audit_log WHERE event = ?`, model.EventUserSelfDeleted); n != 0 {
		t.Fatal("audit row of a refused deletion was kept")
	}

	mustNoErr(t, st.DeleteUser(ctx, u.ID, model.AuditEntry{Event: model.EventUserDeleted, Success: true,
		Actor:  model.AuditActor{ID: &admin.ID, Username: admin.Username},
		Target: &model.AuditTarget{Type: model.TargetUser, ID: &u.ID, Name: u.Username}}))

	_, err = st.UserByID(ctx, u.ID)
	wantErr(t, err, model.ErrNotFound)
	for _, q := range []string{
		`SELECT COUNT(*) FROM user_preferences WHERE user_id = ?`,
		`SELECT COUNT(*) FROM sessions WHERE user_id = ?`,
		`SELECT COUNT(*) FROM login_challenges WHERE user_id = ?`,
		`SELECT COUNT(*) FROM recovery_codes WHERE user_id = ?`,
		`SELECT COUNT(*) FROM login_throttle WHERE user_id = ?`,
		`SELECT COUNT(*) FROM consent_records WHERE user_id = ?`,
		`SELECT COUNT(*) FROM tags WHERE owner_id = ?`,
		`SELECT COUNT(*) FROM links WHERE owner_id = ?`,
		`SELECT COUNT(*) FROM link_user_state WHERE user_id = ?`,
		`SELECT COUNT(*) FROM audit_log WHERE actor_id = ?`,
	} {
		if n := countRows(t, st, q, u.ID); n != 0 {
			t.Errorf("%s = %d after deletion", q, n)
		}
	}
	// Shared links vanish for recipients, with their personal state.
	if n := countRows(t, st, `SELECT COUNT(*) FROM link_user_state WHERE link_id = ?`, l.ID); n != 0 {
		t.Errorf("recipient state on a deleted link: %d", n)
	}
	page, _ := st.Links(ctx, rcpt.Viewer(), model.LinkQuery{Archived: model.ArchivedInclude, Limit: 10})
	if page.Total != 1 || page.Items[0].ID != other.ID {
		t.Errorf("recipient links = %+v", page.Items)
	}

	// No audit column names the user any more; the new row is pseudonymised too.
	rows, err := st.db.Query(`SELECT event, actor_name, target_name, details FROM audit_log`)
	mustNoErr(t, err)
	alias := "deleted-user-" + itoa64(u.ID)
	sawDeleted, aliases := false, 0
	for rows.Next() {
		var event, actor, target, details string
		mustNoErr(t, rows.Scan(&event, &actor, &target, &details))
		for _, col := range []string{actor, target, details} {
			low := strings.ToLower(col)
			if strings.Contains(low, "zoltan") {
				t.Errorf("%s row still names the user: %q", event, col)
			}
		}
		if event == model.EventUserDeleted {
			sawDeleted = true
			if target != alias || actor != admin.Username {
				t.Errorf("user.deleted row = actor %q target %q", actor, target)
			}
		}
		if actor == alias || target == alias {
			aliases++
		}
	}
	mustNoErr(t, rows.Close())
	if !sawDeleted || aliases != 4 {
		t.Errorf("user.deleted written %v, pseudonymised rows %d (want 4)", sawDeleted, aliases)
	}
}

func itoa64(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestExportAndImportLinks(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	other := seedUser(t, st, model.RoleCustomer)
	existingTag := seedTag(t, st, u, "Go")
	existing, err := st.CreateLink(ctx, model.NewLink{OwnerID: u.ID, URL: "https://go.dev/", Host: "go.dev",
		Title: "Go", TagIDs: []int64{existingTag.ID}})
	mustNoErr(t, err)

	added := testutil.T0.Add(-365 * 24 * time.Hour)
	future := testutil.T0.Add(time.Hour)
	entries := []model.ImportEntry{
		{URL: "https://go.dev/", Host: "go.dev", Title: "Duplicate of an existing link"},
		{URL: "https://a.example/", Host: "a.example", Title: "A‮ title\x00 with\tcontrols",
			Description: "line 1\r\nline⁦ 2  \x07", Tags: []string{"go", "New tag", "New tag", "  "}, AddedAt: &added},
		{URL: "https://a.example/", Host: "a.example", Title: "Duplicate within the import"},
		{URL: "http://x", Host: "x"}, // too short for the column
		{URL: "https://b.example/", Host: "b.example", Description: strings.Repeat("d", 600), AddedAt: &future},
	}
	res, ids, err := st.ImportLinks(ctx, u.ID, entries)
	mustNoErr(t, err)
	if res.Total != 5 || res.Created != 2 || res.Duplicates != 2 || res.Invalid != 1 || len(ids) != 2 {
		t.Fatalf("result = %+v ids %v", res, ids)
	}
	a, err := st.Link(ctx, u.Viewer(), ids[0])
	mustNoErr(t, err)
	if a.Title != "A title with controls" || a.TitleSource != model.TitleManual || a.Description != "line 1\nline 2" ||
		a.FetchStatus != model.FetchPending || !a.CreatedAt.Equal(added) || !a.BaselineAt.Equal(clk.Now()) {
		t.Errorf("imported a = %+v", a)
	}
	if len(a.Tags) != 2 || a.Tags[0].ID != existingTag.ID || a.Tags[1].Name != "New tag" || a.Tags[1].Color != model.TagColorNeutral {
		t.Errorf("imported tags = %+v", a.Tags)
	}
	b, _ := st.Link(ctx, u.Viewer(), ids[1])
	if len([]rune(b.Description)) != model.DescriptionMaxLen || b.TitleSource != model.TitleNone || !b.CreatedAt.Equal(clk.Now()) {
		t.Errorf("imported b = %+v", b)
	}
	due, _ := st.DueFetches(ctx, 10)
	if len(due) != 3 {
		t.Errorf("due after import = %v", due)
	}
	_, _, err = st.ImportLinks(ctx, 9999, entries)
	wantErr(t, err, model.ErrNotFound)

	exported, err := st.ExportLinks(ctx, u.ID)
	mustNoErr(t, err)
	if len(exported) != 3 || exported[0].URL != "https://a.example/" || exported[1].URL != existing.URL ||
		len(exported[1].Tags) != 1 || exported[1].Tags[0] != "Go" {
		t.Errorf("export = %+v", exported)
	}
	if none, _ := st.ExportLinks(ctx, other.ID); none == nil || len(none) != 0 {
		t.Errorf("empty export = %v", none)
	}

	// Re-importing the export yields only duplicates.
	again := []model.ImportEntry{}
	for _, l := range exported {
		again = append(again, model.ImportEntry{URL: l.URL, Host: "example", Title: l.Title, Tags: l.Tags})
	}
	res, ids, err = st.ImportLinks(ctx, u.ID, again)
	mustNoErr(t, err)
	if res.Created != 0 || res.Duplicates != 3 || len(ids) != 0 {
		t.Errorf("re-import = %+v %v", res, ids)
	}
}

func TestImportTagLimits(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	now := formatTime(clk.Now())
	for i := 0; i < model.MaxTagsPerUser-1; i++ {
		_, err := st.db.Exec(`INSERT INTO tags (owner_id, name, name_key, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`, u.ID, "t"+itoa64(int64(i)), "t"+itoa64(int64(i)), now, now)
		mustNoErr(t, err)
	}
	names := []string{}
	for i := 0; i < 25; i++ {
		names = append(names, "t"+itoa64(int64(i)))
	}
	res, ids, err := st.ImportLinks(ctx, u.ID, []model.ImportEntry{
		{URL: "https://one.example/", Host: "one.example", Tags: []string{"Brand new", "Second new"}},
		{URL: "https://two.example/", Host: "two.example", Tags: names},
	})
	mustNoErr(t, err)
	if res.Created != 2 {
		t.Fatalf("result = %+v", res)
	}
	if n := countRows(t, st, `SELECT COUNT(*) FROM tags WHERE owner_id = ?`, u.ID); n != model.MaxTagsPerUser {
		t.Errorf("tags = %d, want the limit", n)
	}
	one, _ := st.LinkItem(ctx, u.Viewer(), ids[0])
	two, _ := st.LinkItem(ctx, u.Viewer(), ids[1])
	if len(one.TagIDs) != 1 || len(two.TagIDs) != model.MaxTagsPerLink {
		t.Errorf("tags on links: %d and %d", len(one.TagIDs), len(two.TagIDs))
	}
}

// An AddedAt before the Unix epoch (a negative UTC year cannot be read back)
// or not before now is replaced by the import time; the links stay readable.
func TestImportCreatedAtBounds(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u := seedUser(t, st, model.RoleDeveloper)
	recipient := seedUser(t, st, model.RoleCustomer)
	yearMinusOne := time.Date(-1, 12, 31, 23, 30, 0, 0, time.UTC)
	yearOne := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	beforeEpoch := time.Unix(-1, 0).UTC()
	epoch := time.Unix(0, 0).UTC()
	now := clk.Now()
	entries := []model.ImportEntry{
		{URL: "https://minus.example/", Host: "minus.example", AddedAt: &yearMinusOne},
		{URL: "https://one.example/", Host: "one.example", AddedAt: &yearOne},
		{URL: "https://before.example/", Host: "before.example", AddedAt: &beforeEpoch},
		{URL: "https://now.example/", Host: "now.example", AddedAt: &now},
		{URL: "https://epoch.example/", Host: "epoch.example", AddedAt: &epoch},
	}
	_, ids, err := st.ImportLinks(ctx, u.ID, entries)
	mustNoErr(t, err)
	for i, id := range ids {
		want := now
		if i == 4 {
			want = epoch
		}
		it, err := st.LinkItem(ctx, u.Viewer(), id)
		mustNoErr(t, err)
		if !it.CreatedAt.Equal(want) {
			t.Errorf("%s: createdAt = %v, want %v", entries[i].URL, it.CreatedAt, want)
		}
		share(t, st, u, id, shareAll())
	}
	_, err = st.Links(ctx, u.Viewer(), model.LinkQuery{Limit: 50})
	mustNoErr(t, err)
	_, err = st.Links(ctx, recipient.Viewer(), model.LinkQuery{Limit: 50})
	mustNoErr(t, err)
	_, err = st.ExportLinks(ctx, u.ID)
	mustNoErr(t, err)
	_, err = st.ExportUser(ctx, u.ID)
	mustNoErr(t, err)
}

// Repeated tag names of one entry are resolved in linear time (the link's
// tag ids used to be de-duplicated again after every name).
func TestImportManyTagNamesIsLinear(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	u := seedUser(t, st, model.RoleCustomer)
	names := make([]string, 0, 150002)
	for i := 0; i < 150000; i++ {
		names = append(names, "Same")
	}
	names = append(names, "Second", "Third")
	start := time.Now()
	_, ids, err := st.ImportLinks(ctx, u.ID, []model.ImportEntry{
		{URL: "https://many.example/", Host: "many.example", Tags: names},
	})
	elapsed := time.Since(start)
	mustNoErr(t, err)
	it, err := st.LinkItem(ctx, u.Viewer(), ids[0])
	mustNoErr(t, err)
	if len(it.TagIDs) != 3 {
		t.Errorf("tags = %v, want 3", it.TagIDs)
	}
	if elapsed > 2*time.Second {
		t.Errorf("import of one entry with %d tag names took %v", len(names), elapsed)
	}
}

func TestCleanImportText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		keep bool
		max  int
		want string
	}{
		{"  plain  ", false, 10, "plain"},
		{"a\u0085b\u009fc\u007fd", false, 10, "abcd"},
		{"adm؜in", false, 10, "admin"},
		{"x‎y‏z‪‫‬‭‮⁦⁧⁨⁩", false, 10, "xyz"},
		{"a\nb\r\nc", false, 10, "a b c"},
		{"a\nb\r\n  c ", true, 10, "a\nb\nc"},
		{"árvíztűrő", false, 4, "árví"},
		{"a    b", false, 10, "a b"},
	}
	for _, c := range cases {
		if got := cleanImportText(c.in, c.keep, c.max); got != c.want {
			t.Errorf("cleanImportText(%q, %v) = %q, want %q", c.in, c.keep, got, c.want)
		}
	}
}
