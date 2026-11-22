package store

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// storeCalls invokes every public method once with plausible arguments and
// returns their errors by name.
func storeCalls(st *Store, u model.User, linkID, tagID, shareID int64) map[string]error {
	v := u.Viewer()
	now := time.Now()
	errs := map[string]error{}
	add := func(name string, err error) { errs[name] = err }
	two := func(name string) func(any, error) { return func(_ any, err error) { add(name, err) } }
	three := func(name string) func(any, any, error) { return func(_, _ any, err error) { add(name, err) } }

	two("KeyCheck")(st.KeyCheck(ctx))
	add("SetKeyCheck", st.SetKeyCheck(ctx, "x"))
	two("Settings")(st.Settings(ctx))
	two("UpdateSettings")(st.UpdateSettings(ctx, model.SettingsPatch{DefaultArchiveDays: ptr(5)}, u.ID))
	two("BumpPrivacyNotice")(st.BumpPrivacyNotice(ctx, u.ID))
	two("CountUsers")(st.CountUsers(ctx))
	two("CreateUser")(st.CreateUser(ctx, model.NewUser{Username: "brand-new", Role: model.RoleCustomer, PasswordHash: "h"}))
	two("UserByID")(st.UserByID(ctx, u.ID))
	two("Credentials")(st.Credentials(ctx, u.UsernameKey))
	three("AdminUsers")(st.AdminUsers(ctx, model.UserFilter{}, model.PageReq{Limit: 10}))
	two("AdminUser")(st.AdminUser(ctx, u.ID))
	two("UpdateUser")(st.UpdateUser(ctx, u.ID, model.UserPatch{DisplayName: ptr("x")}))
	add("SetPassword", st.SetPassword(ctx, u.ID, "h", false))
	add("RecordLogin", st.RecordLogin(ctx, u.ID))
	two("Directory")(st.Directory(ctx, 0, "", 10))
	two("OwnSharedLinkCount")(st.OwnSharedLinkCount(ctx, u.ID))
	add("SetPendingTOTP", st.SetPendingTOTP(ctx, u.ID, "v1:x", now.Add(time.Hour)))
	_, _, err := st.PendingTOTP(ctx, u.ID)
	add("PendingTOTP", err)
	add("EnableTOTP", st.EnableTOTP(ctx, u.ID, "v1:x", 1, []string{hash64('a')}))
	add("DisableTOTP", st.DisableTOTP(ctx, u.ID))
	two("AdvanceTOTPStep")(st.AdvanceTOTPStep(ctx, u.ID, 5))
	add("ReplaceRecoveryCodes", st.ReplaceRecoveryCodes(ctx, u.ID, []string{hash64('b')}))
	two("UseRecoveryCode")(st.UseRecoveryCode(ctx, u.ID, hash64('b')))
	two("RecoveryCodesRemaining")(st.RecoveryCodesRemaining(ctx, u.ID))
	two("CreateSession")(st.CreateSession(ctx, model.NewSession{TokenHash: make32(77), UserID: u.ID, ExpiresAt: now.Add(time.Hour)}))
	three("SessionByToken")(st.SessionByToken(ctx, make32(77)))
	add("TouchSession", st.TouchSession(ctx, 1))
	two("Sessions")(st.Sessions(ctx, u.ID))
	add("DeleteSession", st.DeleteSession(ctx, u.ID, 1))
	add("DeleteSessionByToken", st.DeleteSessionByToken(ctx, make32(77)))
	two("DeleteUserSessions")(st.DeleteUserSessions(ctx, u.ID, 0))
	add("CreateChallenge", st.CreateChallenge(ctx, make32(78), u.ID, "", now.Add(time.Hour)))
	two("ChallengeAttempt")(st.ChallengeAttempt(ctx, make32(78), 5))
	add("DeleteChallenge", st.DeleteChallenge(ctx, make32(78)))
	add("ThrottleUpdate", st.ThrottleUpdate(ctx, []model.ThrottleKey{{Key: "ip:x"}},
		func(r model.ThrottleRow) (model.ThrottleRow, error) { r.Failures++; return r, nil }))
	add("ClearThrottle", st.ClearThrottle(ctx, "ip:x"))
	add("ClearUserThrottle", st.ClearUserThrottle(ctx, u.ID))
	two("Preferences")(st.Preferences(ctx, u.ID))
	two("UpdatePreferences")(st.UpdatePreferences(ctx, u.ID, model.PreferencesPatch{Theme: ptr(model.ThemeLight)}))
	two("CreateLink")(st.CreateLink(ctx, model.NewLink{OwnerID: u.ID, URL: "https://new.example/", Host: "new.example",
		TagIDs: []int64{tagID}}))
	two("Link")(st.Link(ctx, v, linkID))
	two("Links")(st.Links(ctx, v, model.LinkQuery{Limit: 10}))
	two("LinkItem")(st.LinkItem(ctx, v, linkID))
	two("UpdateLink")(st.UpdateLink(ctx, v, linkID, model.LinkPatch{Title: ptr("t")}))
	add("SetLinkTags", st.SetLinkTags(ctx, v, linkID, []int64{tagID}))
	add("SetPinned", st.SetPinned(ctx, v, linkID, true))
	add("SetArchived", st.SetArchived(ctx, v, linkID, true))
	two("RecordVisit")(st.RecordVisit(ctx, v, linkID))
	two("DueFetches")(st.DueFetches(ctx, 10))
	two("BeginFetch")(st.BeginFetch(ctx, linkID, time.Minute))
	add("ApplyFetchResult", st.ApplyFetchResult(ctx, model.FetchResult{LinkID: linkID, URL: linkURL(st, linkID),
		Status: model.FetchOK, Icon: &model.IconBlob{Data: []byte("i"), ContentType: "image/png"}}))
	two("LinkIcon")(st.LinkIcon(ctx, v, linkID))
	add("ResetFetch", st.ResetFetch(ctx, v, linkID))
	two("Shares")(st.Shares(ctx, v, linkID))
	two("CreateShare")(st.CreateShare(ctx, v, linkID, shareRole(model.RoleCustomer)))
	add("DeleteShare", st.DeleteShare(ctx, v, linkID, shareID))
	two("Tags")(st.Tags(ctx, u.ID, u.Role))
	two("CreateTag")(st.CreateTag(ctx, u.ID, "Another", ""))
	two("UpdateTag")(st.UpdateTag(ctx, u.ID, tagID, model.TagPatch{Name: ptr("Renamed")}))
	add("AppendAudit", st.AppendAudit(ctx, model.AuditEntry{Event: model.EventLogout}))
	three("Audit")(st.Audit(ctx, model.AuditFilter{}, model.PageReq{Limit: 10}))
	add("RecordConsent", st.RecordConsent(ctx, u.ID, model.ConsentPrivacyNotice, 9))
	two("Consents")(st.Consents(ctx, u.ID))
	two("ExportUser")(st.ExportUser(ctx, u.ID))
	two("ExportLinks")(st.ExportLinks(ctx, u.ID))
	three("ImportLinks")(st.ImportLinks(ctx, u.ID, []model.ImportEntry{{URL: "https://imp.example/", Host: "imp.example",
		Tags: []string{"Imported"}}}))
	two("ArchiveInactive")(st.ArchiveInactive(ctx, 1))
	two("Purge")(st.Purge(ctx, time.Hour))
	add("DeleteTag", st.DeleteTag(ctx, u.ID, tagID))
	add("DeleteLink", st.DeleteLink(ctx, v, linkID))
	add("DeleteUser", st.DeleteUser(ctx, u.ID, model.AuditEntry{Event: model.EventUserSelfDeleted}))
	add("Ping", st.Ping(ctx))
	return errs
}

// seedForCalls prepares a user with a link, a tag and a share.
func seedForCalls(t *testing.T, st *Store) (model.User, int64, int64, int64) {
	t.Helper()
	u := seedUser(t, st, model.RoleDeveloper)
	seedUser(t, st, model.RoleAdmin) // u is not the last admin
	l := seedLink(t, st, u, "")
	tag := seedTag(t, st, u, "Tag")
	sh := share(t, st, u, l.ID, shareAll())
	return u, l.ID, tag.ID, sh.ID
}

// TestEveryMethodSucceeds runs storeCalls on a healthy store and expects no
// error from any method.
func TestEveryMethodSucceeds(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	u, linkID, tagID, shareID := seedForCalls(t, st)
	for name, err := range storeCalls(st, u, linkID, tagID, shareID) {
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestEveryMethodReportsDatabaseErrors closes the database and checks that
// every method returns an error instead of a zero value.
func TestEveryMethodReportsDatabaseErrors(t *testing.T) {
	t.Parallel()
	st, _ := newTestStore(t)
	u, linkID, tagID, shareID := seedForCalls(t, st)
	mustNoErr(t, st.db.Close())
	for name, err := range storeCalls(st, u, linkID, tagID, shareID) {
		if err == nil {
			t.Errorf("%s: no error on a closed database", name)
			continue
		}
		if errors.Is(err, model.ErrNotFound) {
			t.Errorf("%s: database error reported as not found: %v", name, err)
		}
	}
}

// TestInjectedStatementFailures makes single statements fail with a trigger
// and checks that the error surfaces and the transaction rolls back.
func TestInjectedStatementFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		trigger string // "<BEFORE|AFTER> <INSERT|UPDATE|DELETE> ON <table>"
		call    func(st *Store, u model.User, linkID, tagID, shareID int64) error
	}{
		{"CreateUser preferences", "BEFORE INSERT ON user_preferences", func(st *Store, _ model.User, _, _, _ int64) error {
			_, err := st.CreateUser(ctx, model.NewUser{Username: "late", Role: model.RoleCustomer, PasswordHash: "h"})
			return err
		}},
		{"CreateLink state", "BEFORE INSERT ON link_user_state", func(st *Store, u model.User, _, _, _ int64) error {
			_, err := st.CreateLink(ctx, model.NewLink{OwnerID: u.ID, URL: "https://z.example/", Host: "z.example"})
			return err
		}},
		{"CreateLink tags", "BEFORE INSERT ON link_tags", func(st *Store, u model.User, _, tagID, _ int64) error {
			_, err := st.CreateLink(ctx, model.NewLink{OwnerID: u.ID, URL: "https://z.example/", Host: "z.example",
				TagIDs: []int64{tagID}})
			return err
		}},
		{"SetLinkTags clear", "BEFORE DELETE ON link_tags", func(st *Store, u model.User, linkID, tagID, _ int64) error {
			if err := st.SetLinkTags(ctx, u.Viewer(), linkID, []int64{tagID}); err != nil {
				return err
			}
			return st.SetLinkTags(ctx, u.Viewer(), linkID, nil)
		}},
		{"UpdateLink", "BEFORE UPDATE ON links", func(st *Store, u model.User, linkID, _, _ int64) error {
			_, err := st.UpdateLink(ctx, u.Viewer(), linkID, model.LinkPatch{Title: ptr("x")})
			return err
		}},
		{"DeleteLink", "BEFORE DELETE ON links", func(st *Store, u model.User, linkID, _, _ int64) error {
			return st.DeleteLink(ctx, u.Viewer(), linkID)
		}},
		{"ResetFetch", "BEFORE UPDATE ON links", func(st *Store, u model.User, linkID, _, _ int64) error {
			return st.ResetFetch(ctx, u.Viewer(), linkID)
		}},
		{"ApplyFetchResult retry", "BEFORE UPDATE ON links", func(st *Store, _ model.User, linkID, _, _ int64) error {
			return st.ApplyFetchResult(ctx, model.FetchResult{LinkID: linkID, URL: linkURL(st, linkID),
				Status: model.FetchFailed, Temporary: true, Error: model.FetchErrDNS})
		}},
		{"ApplyFetchResult final", "BEFORE UPDATE ON links", func(st *Store, _ model.User, linkID, _, _ int64) error {
			return st.ApplyFetchResult(ctx, model.FetchResult{LinkID: linkID, URL: linkURL(st, linkID), Status: model.FetchOK})
		}},
		{"ApplyFetchResult icon", "BEFORE INSERT ON icons", func(st *Store, _ model.User, linkID, _, _ int64) error {
			return st.ApplyFetchResult(ctx, model.FetchResult{LinkID: linkID, URL: linkURL(st, linkID), Status: model.FetchOK,
				Icon: &model.IconBlob{Data: []byte("i"), ContentType: "image/png"}})
		}},
		{"BeginFetch locale", "BEFORE UPDATE ON links", func(st *Store, _ model.User, linkID, _, _ int64) error {
			_, err := st.BeginFetch(ctx, linkID, time.Minute)
			return err
		}},
		{"SetPinned", "BEFORE UPDATE ON link_user_state", func(st *Store, u model.User, linkID, _, _ int64) error {
			return st.SetPinned(ctx, u.Viewer(), linkID, true)
		}},
		{"RecordVisit", "BEFORE UPDATE ON link_user_state", func(st *Store, u model.User, linkID, _, _ int64) error {
			_, err := st.RecordVisit(ctx, u.Viewer(), linkID)
			return err
		}},
		{"CreateShare", "BEFORE INSERT ON link_shares", func(st *Store, u model.User, linkID, _, _ int64) error {
			_, err := st.CreateShare(ctx, u.Viewer(), linkID, shareRole(model.RoleAdmin))
			return err
		}},
		{"DeleteShare", "BEFORE DELETE ON link_shares", func(st *Store, u model.User, linkID, _, shareID int64) error {
			return st.DeleteShare(ctx, u.Viewer(), linkID, shareID)
		}},
		{"UpdateTag", "BEFORE UPDATE ON tags", func(st *Store, u model.User, _, tagID, _ int64) error {
			_, err := st.UpdateTag(ctx, u.ID, tagID, model.TagPatch{Name: ptr("New name")})
			return err
		}},
		{"CreateTag", "BEFORE INSERT ON tags", func(st *Store, u model.User, _, _, _ int64) error {
			_, err := st.CreateTag(ctx, u.ID, "Fresh", "")
			return err
		}},
		{"UpdateUser", "BEFORE UPDATE ON users", func(st *Store, u model.User, _, _, _ int64) error {
			_, err := st.UpdateUser(ctx, u.ID, model.UserPatch{DisplayName: ptr("x")})
			return err
		}},
		{"DeleteUser audit insert", "BEFORE INSERT ON audit_log", func(st *Store, u model.User, _, _, _ int64) error {
			return st.DeleteUser(ctx, u.ID, model.AuditEntry{Event: model.EventUserSelfDeleted})
		}},
		{"DeleteUser pseudonymise", "BEFORE UPDATE ON audit_log", func(st *Store, u model.User, _, _, _ int64) error {
			if err := st.AppendAudit(ctx, model.AuditEntry{Event: model.EventLogout,
				Actor: model.AuditActor{ID: &u.ID, Username: u.Username}}); err != nil {
				return err
			}
			return st.DeleteUser(ctx, u.ID, model.AuditEntry{Event: model.EventUserSelfDeleted})
		}},
		{"DeleteUser delete", "BEFORE DELETE ON users", func(st *Store, u model.User, _, _, _ int64) error {
			return st.DeleteUser(ctx, u.ID, model.AuditEntry{Event: model.EventUserSelfDeleted})
		}},
		{"EnableTOTP codes", "BEFORE INSERT ON recovery_codes", func(st *Store, u model.User, _, _, _ int64) error {
			return st.EnableTOTP(ctx, u.ID, "v1:x", 1, []string{hash64('a')})
		}},
		{"EnableTOTP update", "BEFORE UPDATE ON users", func(st *Store, u model.User, _, _, _ int64) error {
			return st.EnableTOTP(ctx, u.ID, "v1:x", 1, nil)
		}},
		{"DisableTOTP codes", "BEFORE DELETE ON recovery_codes", func(st *Store, u model.User, _, _, _ int64) error {
			if err := st.ReplaceRecoveryCodes(ctx, u.ID, []string{hash64('c')}); err != nil {
				return err
			}
			return st.DisableTOTP(ctx, u.ID)
		}},
		{"CreateChallenge insert", "BEFORE INSERT ON login_challenges", func(st *Store, u model.User, _, _, _ int64) error {
			return st.CreateChallenge(ctx, make32(1), u.ID, "", time.Now().Add(time.Hour))
		}},
		{"CreateSession", "BEFORE INSERT ON sessions", func(st *Store, u model.User, _, _, _ int64) error {
			_, err := st.CreateSession(ctx, model.NewSession{TokenHash: make32(1), UserID: u.ID, ExpiresAt: time.Now()})
			return err
		}},
		{"UpdateSettings reset", "BEFORE DELETE ON system_settings", func(st *Store, u model.User, _, _, _ int64) error {
			if _, err := st.UpdateSettings(ctx, model.SettingsPatch{DefaultArchiveDays: ptr(30)}, u.ID); err != nil {
				return err
			}
			_, err := st.UpdateSettings(ctx, model.SettingsPatch{DefaultArchiveDays: ptr(90)}, u.ID)
			return err
		}},
		{"BumpPrivacyNotice", "BEFORE INSERT ON system_settings", func(st *Store, u model.User, _, _, _ int64) error {
			_, err := st.BumpPrivacyNotice(ctx, u.ID)
			return err
		}},
		{"ImportLinks state", "BEFORE INSERT ON link_user_state", func(st *Store, u model.User, _, _, _ int64) error {
			_, _, err := st.ImportLinks(ctx, u.ID, []model.ImportEntry{{URL: "https://imp.example/", Host: "imp.example"}})
			return err
		}},
		{"ImportLinks link", "BEFORE INSERT ON links", func(st *Store, u model.User, _, _, _ int64) error {
			_, _, err := st.ImportLinks(ctx, u.ID, []model.ImportEntry{{URL: "https://imp.example/", Host: "imp.example"}})
			return err
		}},
		{"ImportLinks tag", "BEFORE INSERT ON tags", func(st *Store, u model.User, _, _, _ int64) error {
			_, _, err := st.ImportLinks(ctx, u.ID, []model.ImportEntry{{URL: "https://imp.example/", Host: "imp.example",
				Tags: []string{"New"}}})
			return err
		}},
		{"ImportLinks link tags", "BEFORE INSERT ON link_tags", func(st *Store, u model.User, _, _, _ int64) error {
			_, _, err := st.ImportLinks(ctx, u.ID, []model.ImportEntry{{URL: "https://imp.example/", Host: "imp.example",
				Tags: []string{"Tag"}}})
			return err
		}},
		{"ArchiveInactive shared rows", "BEFORE INSERT ON link_user_state", func(st *Store, u model.User, linkID, _, _ int64) error {
			seedUserNoT(st, model.RoleCustomer)
			_, err := st.ArchiveInactive(ctx, 1)
			return err
		}},
		{"ArchiveInactive update", "BEFORE UPDATE ON link_user_state", func(st *Store, _ model.User, _, _, _ int64) error {
			// Replace the state row with an old one (the trigger blocks only UPDATE).
			if _, err := st.db.Exec(`DELETE FROM link_user_state`); err != nil {
				return err
			}
			if _, err := st.db.Exec(`INSERT INTO link_user_state (user_id, link_id, baseline_at, updated_at)
				SELECT owner_id, id, '2000-01-01T00:00:00.000Z', '2000-01-01T00:00:00.000Z' FROM links`); err != nil {
				return err
			}
			_, err := st.ArchiveInactive(ctx, 1)
			return err
		}},
		{"Purge audit", "BEFORE DELETE ON audit_log", func(st *Store, _ model.User, _, _, _ int64) error {
			if err := st.AppendAudit(ctx, model.AuditEntry{At: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Event: "x"}); err != nil {
				return err
			}
			_, err := st.Purge(ctx, time.Hour)
			return err
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			st, _ := newTestStore(t)
			u, linkID, tagID, shareID := seedForCalls(t, st)
			name := "inject_" + strings.NewReplacer(" ", "_").Replace(c.name)
			_, err := st.db.Exec(`CREATE TRIGGER ` + name + ` ` + c.trigger +
				` BEGIN SELECT RAISE(ABORT, 'injected failure'); END`)
			mustNoErr(t, err)
			err = c.call(st, u, linkID, tagID, shareID)
			if err == nil || !strings.Contains(err.Error(), "injected failure") {
				t.Fatalf("error = %v, want the injected failure", err)
			}
		})
	}
}

func linkURL(st *Store, id int64) string {
	var url string
	_ = st.db.QueryRow(`SELECT url FROM links WHERE id = ?`, id).Scan(&url)
	return url
}

// seedUserNoT creates a user without a *testing.T (inside injected cases).
func seedUserNoT(st *Store, role model.Role) model.User {
	u, _ := st.CreateUser(ctx, model.NewUser{Username: "nt-" + itoa64(userSeq.Add(1)), Role: role, PasswordHash: "h"})
	return u
}

// TestCorruptRowsAreReported stores unparsable timestamps and checks that
// readers report them instead of returning zero times.
func TestCorruptRowsAreReported(t *testing.T) {
	t.Parallel()
	st, clk := newTestStore(t)
	u, linkID, tagID, _ := seedForCalls(t, st)
	_, err := st.CreateSession(ctx, model.NewSession{TokenHash: make32(3), UserID: u.ID, ExpiresAt: clk.Now().Add(time.Hour)})
	mustNoErr(t, err)
	mustNoErr(t, st.RecordConsent(ctx, u.ID, model.ConsentPrivacyNotice, 1))
	mustNoErr(t, st.AppendAudit(ctx, model.AuditEntry{Event: "x", Actor: model.AuditActor{ID: &u.ID}}))
	_, err = st.RecordVisit(ctx, u.Viewer(), linkID)
	mustNoErr(t, err)

	corrupt := []string{
		`UPDATE users SET last_login_at = 'garbage'`,
		`UPDATE sessions SET last_seen_at = 'garbage'`,
		`UPDATE consent_records SET accepted_at = 'garbage'`,
		`UPDATE audit_log SET at = 'garbage'`,
		`UPDATE link_user_state SET last_used_at = 'garbage'`,
		`UPDATE tags SET created_at = 'garbage'`,
		`UPDATE link_shares SET created_at = 'garbage'`,
		`UPDATE links SET fetched_at = 'garbage'`,
	}
	for _, q := range corrupt {
		_, err := st.db.Exec(q)
		mustNoErr(t, err)
	}
	checks := map[string]error{}
	_, checks["UserByID"] = st.UserByID(ctx, u.ID)
	_, checks["Sessions"] = st.Sessions(ctx, u.ID)
	_, checks["Consents"] = st.Consents(ctx, u.ID)
	_, _, checks["Audit"] = st.Audit(ctx, model.AuditFilter{}, model.PageReq{Limit: 10})
	_, checks["Links"] = st.Links(ctx, u.Viewer(), model.LinkQuery{Limit: 10})
	_, checks["Link"] = st.Link(ctx, u.Viewer(), linkID)
	_, checks["Tags"] = st.Tags(ctx, u.ID, u.Role)
	_, checks["ExportUser"] = st.ExportUser(ctx, u.ID)
	_, checks["linkStateQ"] = linkStateQ(ctx, st.db, u.ID, linkID)
	_, checks["sharesQ"] = sharesQ(ctx, st.db, linkID)
	_, checks["UpdateTag"] = st.UpdateTag(ctx, u.ID, tagID, model.TagPatch{})
	for name, err := range checks {
		if err == nil {
			t.Errorf("%s accepted a corrupt timestamp", name)
		}
	}
}
