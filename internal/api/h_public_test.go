package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestHealthz(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	// No session, no custom header, no Origin: the health check is a plain GET.
	c := e.client(t).withHeader(headerKinexRequest, "").withHeader("Origin", "")

	res := c.get("/healthz")
	assertJSON(t, expectStatus(t, res, http.StatusOK), `{"status":"ok"}`)
	expectJSONContentType(t, res)
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control %q", res.Header.Get("Cache-Control"))
	}

	res = c.do(http.MethodHead, "/healthz", nil)
	if body := expectStatus(t, res, http.StatusOK); len(body) != 0 {
		t.Errorf("HEAD body %q", body)
	}
}

func TestHealthzDatabaseDown(t *testing.T) {
	t.Parallel()
	fail := map[string]bool{"Ping": true}
	e := newEnv(t, withDeps(func(d *Deps) { d.Store = faultStore{Store: d.Store, fail: fail} }))
	expectError(t, e.client(t).get("/healthz"), http.StatusInternalServerError, codeInternalError)
	out := e.logs.String()
	if !strings.Contains(out, `msg="request failed"`) || !strings.Contains(out, errInjected.Error()) || !strings.Contains(out, `route="GET /healthz"`) {
		t.Fatalf("failure log:\n%s", out)
	}
}

func TestPublicConfig(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := e.client(t).withHeader(headerKinexRequest, "")

	const defaults = `{
		"appName": "Kinex Mark",
		"version": "test",
		"locales": ["hu", "en", "de"],
		"defaultLocale": "hu",
		"tagColors": ["blue", "ochre", "green", "violet", "rose", "neutral", "contrast"],
		"passwordPolicy": {"minLength": 12, "maxLength": 128, "minDistinct": 6},
		"privacy": {
			"controllerName": "", "controllerAddress": "", "controllerEmail": "", "dpoContact": "",
			"supervisoryAuthority": "", "noticeVersion": 1, "updatedAt": null,
			"auditRetentionDays": 180, "sessionIdleMinutes": 720, "sessionAbsoluteHours": 168
		}
	}`
	res := c.get("/api/public/config")
	assertJSON(t, expectStatus(t, res, http.StatusOK), defaults)
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control %q", res.Header.Get("Cache-Control"))
	}

	// The privacy block follows the system settings.
	name, authority, retention, idle := "Kinexium Kft.", "Nemzeti Adatvédelmi és Információszabadság Hatóság", 365, 60
	if _, err := e.store.UpdateSettings(e.ctx(), model.SettingsPatch{
		PrivacyControllerName:       &name,
		PrivacySupervisoryAuthority: &authority,
		AuditRetentionDays:          &retention,
		SessionIdleMinutes:          &idle,
	}, 0); err != nil {
		t.Fatal(err)
	}
	e.clock.Advance(time.Minute)
	bumped, err := e.store.BumpPrivacyNotice(e.ctx(), 0)
	if err != nil {
		t.Fatal(err)
	}
	type privacy struct {
		ControllerName       string  `json:"controllerName"`
		SupervisoryAuthority string  `json:"supervisoryAuthority"`
		NoticeVersion        int     `json:"noticeVersion"`
		UpdatedAt            *string `json:"updatedAt"`
		AuditRetentionDays   int     `json:"auditRetentionDays"`
		SessionIdleMinutes   int     `json:"sessionIdleMinutes"`
	}
	got := decodeStatus[struct {
		Privacy privacy `json:"privacy"`
	}](t, c.get("/api/public/config"), http.StatusOK).Privacy
	if got.ControllerName != name || got.SupervisoryAuthority != authority || got.NoticeVersion != 2 ||
		got.AuditRetentionDays != retention || got.SessionIdleMinutes != idle || got.UpdatedAt == nil {
		t.Fatalf("privacy %+v", got)
	}
	if at, err := model.ParseTime(*got.UpdatedAt); err != nil || !at.Equal(*bumped.PrivacyNoticeUpdatedAt) {
		t.Fatalf("updatedAt %q (%v), want %v", *got.UpdatedAt, err, bumped.PrivacyNoticeUpdatedAt)
	}
}

func TestPublicConfigVersionDefault(t *testing.T) {
	t.Parallel()
	e := newEnv(t, withDeps(func(d *Deps) { d.Version = "" }))
	got := decodeStatus[map[string]any](t, e.client(t).get("/api/public/config"), http.StatusOK)
	if got["version"] != "dev" {
		t.Fatalf("version %v", got["version"])
	}
}

func TestPublicConfigSettingsFailure(t *testing.T) {
	t.Parallel()
	fail := map[string]bool{"Settings": true}
	e := newEnv(t, withDeps(func(d *Deps) { d.Store = faultStore{Store: d.Store, fail: fail} }))
	expectError(t, e.client(t).get("/api/public/config"), http.StatusInternalServerError, codeInternalError)
}
