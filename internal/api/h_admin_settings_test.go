package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestAdminSettingsGet(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := e.loginAs(t, model.RoleAdmin)
	res := c.get("/api/admin/settings")
	got := decodeStatus[adminSettingsResponse](t, res, http.StatusOK)
	def := model.DefaultSystemSettings()
	if got.Defaults != def || got.Settings.DefaultArchiveDays != def.DefaultArchiveDays || got.LockedByEnv == nil || len(got.LockedByEnv) != 0 {
		t.Errorf("settings %+v", got)
	}
	if keys := strings.Join(jsonKeys(t, readBody(t, res)), ","); keys != "defaults,lockedByEnv,settings" {
		t.Errorf("keys %s", keys)
	}
	dc, _ := e.loginAs(t, model.RoleDeveloper)
	expectError(t, dc.get("/api/admin/settings"), http.StatusForbidden, codeForbidden)
}

func TestAdminSettingsUpdate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, admin := e.loginAs(t, model.RoleAdmin)
	patch := map[string]any{
		"defaultArchiveDays":       30,
		"sessionIdleMinutes":       60,
		"allowPrivateFetch":        true,
		"privacyControllerName":    "  Kinex Kft.  ",
		"privacyControllerAddress": "1111 Budapest\nFő utca 1.",
		"privacyControllerEmail":   "dpo@example.hu",
	}
	got := decodeStatus[adminSettingsResponse](t, c.patch("/api/admin/settings", patch), http.StatusOK)
	s := got.Settings
	if s.DefaultArchiveDays != 30 || s.SessionIdleMinutes != 60 || !s.AllowPrivateFetch || s.PrivacyControllerName != "Kinex Kft." ||
		s.PrivacyControllerAddress != "1111 Budapest\nFő utca 1." || s.PrivacyControllerEmail != "dpo@example.hu" {
		t.Errorf("settings %+v", s)
	}
	a := authExpectAudit(t, e, model.EventSettingsUpdated, 1)[0]
	assertJSON(t, mustJSON(t, a.Details), `{"changed":["defaultArchiveDays","sessionIdleMinutes","allowPrivateFetch",
		"privacyControllerName","privacyControllerAddress","privacyControllerEmail"]}`)
	if a.Target == nil || a.Target.Type != model.TargetSettings || a.Actor.ID == nil || *a.Actor.ID != admin.ID {
		t.Errorf("target %+v", a.Target)
	}
	// An empty patch changes nothing and is not audited.
	decodeStatus[adminSettingsResponse](t, c.patch("/api/admin/settings", map[string]any{}), http.StatusOK)
	authExpectAudit(t, e, model.EventSettingsUpdated, 1)
	// Empty texts clear a value.
	got = decodeStatus[adminSettingsResponse](t, c.patch("/api/admin/settings", map[string]any{"privacyControllerEmail": ""}), http.StatusOK)
	if got.Settings.PrivacyControllerEmail != "" {
		t.Errorf("email %q", got.Settings.PrivacyControllerEmail)
	}

	expectFields(t, c.patch("/api/admin/settings", map[string]any{
		"defaultArchiveDays": 3651, "auditRetentionDays": 29, "sessionIdleMinutes": 14, "sessionAbsoluteHours": 2161,
	}), map[string]string{
		"defaultArchiveDays": fieldOutOfRange, "auditRetentionDays": fieldOutOfRange,
		"sessionIdleMinutes": fieldOutOfRange, "sessionAbsoluteHours": fieldOutOfRange,
	})
	expectFields(t, c.patch("/api/admin/settings", map[string]any{
		"privacySupervisoryAuthority": strings.Repeat("é", 301),
		"privacyControllerName":       "line\nbreak",
		"privacyDpoContact":           strings.Repeat("x", 301),
		"privacyControllerEmail":      "no-at-sign",
	}), map[string]string{
		"privacySupervisoryAuthority": fieldTooLong, "privacyControllerName": fieldInvalid,
		"privacyDpoContact": fieldTooLong, "privacyControllerEmail": fieldInvalidFormat,
	})
	// 300 runes are fine.
	decodeStatus[adminSettingsResponse](t, c.patch("/api/admin/settings", map[string]any{"privacySupervisoryAuthority": strings.Repeat("é", 300)}), http.StatusOK)
	expectError(t, c.patch("/api/admin/settings", map[string]any{"privacyNoticeVersion": 9}), http.StatusBadRequest, codeInvalidJSON)
	expectError(t, c.patch("/api/admin/settings", map[string]any{"defaultArchiveDays": "30"}), http.StatusBadRequest, codeInvalidJSON)
}

func TestAdminSettingsLockedByEnv(t *testing.T) {
	t.Parallel()
	off := false
	e := newEnv(t, withConfig(func(c *config.Config) { c.AllowPrivateFetch = &off }))
	c, _ := e.loginAs(t, model.RoleAdmin)
	// The stored value is true, the environment decides: false.
	on := true
	if _, err := e.store.UpdateSettings(e.ctx(), model.SettingsPatch{AllowPrivateFetch: &on}, 0); err != nil {
		t.Fatal(err)
	}
	got := decodeStatus[adminSettingsResponse](t, c.get("/api/admin/settings"), http.StatusOK)
	if got.Settings.AllowPrivateFetch || len(got.LockedByEnv) != 1 || got.LockedByEnv[0] != model.SettingAllowPrivateFetch {
		t.Errorf("settings %+v", got)
	}
	// Locked keys are refused before validation, with the key list.
	b := expectError(t, c.patch("/api/admin/settings", map[string]any{"allowPrivateFetch": true, "defaultArchiveDays": -1}),
		http.StatusConflict, codeSettingLocked)
	keys, _ := b.Details[detailKeys].([]any)
	if len(keys) != 1 || keys[0] != model.SettingAllowPrivateFetch {
		t.Errorf("details %v", b.Details)
	}
	// Other keys still change.
	decodeStatus[adminSettingsResponse](t, c.patch("/api/admin/settings", map[string]any{"defaultArchiveDays": 10}), http.StatusOK)
}

func TestAdminSettingsBumpPrivacyNotice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, admin := e.loginAs(t, model.RoleAdmin)
	user, _ := e.loginAs(t, model.RoleCustomer)
	expectStatus(t, user.get("/api/me/sessions"), http.StatusOK)

	got := decodeStatus[adminSettingsResponse](t, c.post("/api/admin/settings/privacy-notice/bump", nil), http.StatusOK)
	if got.Settings.PrivacyNoticeVersion != 2 || got.Settings.PrivacyNoticeUpdatedAt == nil {
		t.Errorf("settings %+v", got.Settings)
	}
	a := authExpectAudit(t, e, model.EventPrivacyNoticeBumped, 1)[0]
	if a.Details[model.DetailVersion] != float64(2) || a.Actor.ID == nil || *a.Actor.ID != admin.ID {
		t.Errorf("audit %+v", a)
	}
	// Everyone, the admin included, must acknowledge the new version.
	expectError(t, user.get("/api/me/sessions"), http.StatusForbidden, codeConsentRequired)
	expectError(t, c.get("/api/admin/settings"), http.StatusForbidden, codeConsentRequired)
	me := decodeStatus[meResponse](t, user.get("/api/auth/me"), http.StatusOK)
	if !me.PrivacyConsentRequired || me.PrivacyNoticeVersion != 2 {
		t.Errorf("me %+v", me)
	}
	expectStatus(t, user.post("/api/me/privacy/consent", map[string]int{"version": 2}), http.StatusNoContent)
	expectStatus(t, user.get("/api/me/sessions"), http.StatusOK)
}

func TestAdminSettingsFaults(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	call := func(method, path string, body any) func() func() *http.Response {
		return func() func() *http.Response {
			c, _ := e.loginAs(t, model.RoleAdmin)
			return func() *http.Response { return c.do(method, path, body) }
		}
	}
	// The guard reads the settings once (consent gate) before the handler.
	runAccountFaults(t, fs, []accountFault{
		{name: "get", method: "Settings", nth: 2, prepare: call(http.MethodGet, "/api/admin/settings", nil)},
		{name: "update", method: "UpdateSettings", nth: 1, prepare: call(http.MethodPatch, "/api/admin/settings", map[string]int{"defaultArchiveDays": 5})},
		{name: "bump", method: "BumpPrivacyNotice", nth: 1, prepare: call(http.MethodPost, "/api/admin/settings/privacy-notice/bump", nil)},
	})
}
