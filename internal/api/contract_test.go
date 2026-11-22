package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// update rewrites the golden files:
// go test ./internal/api -run TestContractGolden -update
// Other test files of this package reuse this flag instead of defining their own.
var update = flag.Bool("update", false, "rewrite the golden files in testdata")

const contractGolden = "testdata/contract.json"

// contract is the backend/frontend contract golden (9.6). The frontend's
// contract.spec.ts checks that every listed code has a translation.
type contract struct {
	ErrorCodes        []string            `json:"errorCodes"`
	FieldCodes        []string            `json:"fieldCodes"`
	AuditEvents       []string            `json:"auditEvents"`
	AuditDetailKeys   []string            `json:"auditDetailKeys"`
	AuditDetailValues map[string][]string `json:"auditDetailValues"`
	FetchErrors       []string            `json:"fetchErrors"`
	TagColors         []model.TagColor    `json:"tagColors"`
	Roles             []model.Role        `json:"roles"`
	Locales           []model.Locale      `json:"locales"`
}

func currentContract() contract {
	return contract{
		ErrorCodes:        errorCodes(),
		FieldCodes:        fieldCodes,
		AuditEvents:       model.AuditEvents,
		AuditDetailKeys:   model.AuditDetailKeys,
		AuditDetailValues: model.AuditDetailValues,
		FetchErrors:       model.FetchErrors,
		TagColors:         model.TagColors,
		Roles:             model.Roles,
		Locales:           model.Locales,
	}
}

func TestContractGolden(t *testing.T) {
	got, err := json.MarshalIndent(currentContract(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	if *update {
		if err := os.MkdirAll(filepath.Dir(contractGolden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(contractGolden, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(contractGolden)
	if err != nil {
		t.Fatalf("read golden (run with -update to create it): %v", err)
	}
	// Tolerate a CRLF checkout of the golden file.
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is out of date; run: go test ./internal/api -run TestContractGolden -update\ngot:\n%s", contractGolden, got)
	}
}

func TestContractListsAreUnique(t *testing.T) {
	t.Parallel()
	c := currentContract()
	lists := map[string][]string{
		"errorCodes":      c.ErrorCodes,
		"fieldCodes":      c.FieldCodes,
		"auditEvents":     c.AuditEvents,
		"auditDetailKeys": c.AuditDetailKeys,
		"fetchErrors":     c.FetchErrors,
	}
	for name, list := range lists {
		seen := map[string]bool{}
		for _, v := range list {
			if seen[v] {
				t.Errorf("%s: duplicate %q", name, v)
			}
			seen[v] = true
		}
	}
	if len(c.FieldCodes) != 16 {
		t.Errorf("fieldCodes has %d entries, want 16", len(c.FieldCodes))
	}
	for _, code := range c.ErrorCodes {
		if code == "network_error" {
			t.Error("the client-only network_error must not be in the server list")
		}
	}
}

// nilStore satisfies Store for tests that never reach the store.
type nilStore struct{ Store }

// concretePath fills the path wildcards of a route pattern.
func concretePath(pattern string) string {
	r := strings.NewReplacer("{id}", "1", "{shareId}", "2")
	return r.Replace(pattern)
}

func TestRouteTable(t *testing.T) {
	t.Parallel()
	table := routes()
	seen := map[string]bool{}
	for _, rt := range table {
		if seen[rt.key()] {
			t.Errorf("duplicate route %s", rt.key())
		}
		seen[rt.key()] = true
		if rt.handle == nil {
			t.Errorf("%s: no handler", rt.key())
		}
		if rt.pattern != "/healthz" && !strings.HasPrefix(rt.pattern, "/api/") {
			t.Errorf("%s: pattern outside /api/", rt.key())
		}
		switch rt.access {
		case accessSharer:
			if rt.deny != codeShareNotAllowed && rt.deny != codeForbidden {
				t.Errorf("%s: sharer route deny code %q", rt.key(), rt.deny)
			}
		case accessAdmin:
			if rt.deny != codeForbidden {
				t.Errorf("%s: admin route deny code %q", rt.key(), rt.deny)
			}
		default:
			if rt.deny != "" {
				t.Errorf("%s: deny code %q on a level every role passes", rt.key(), rt.deny)
			}
		}
		if strings.HasPrefix(rt.pattern, "/api/links/{id}/shares") && rt.deny != codeShareNotAllowed {
			t.Errorf("%s: share routes deny with share_not_allowed", rt.key())
		}
		if strings.HasPrefix(rt.pattern, "/api/admin/") && rt.access != accessAdmin {
			t.Errorf("%s: admin route not admin-only", rt.key())
		}
		if rt.access.String() == "?" {
			t.Errorf("%s: unknown access level", rt.key())
		}
	}
	// 5.6: 2, 5.7: 5, 5.8: 14, 5.9: 19, 5.10: 4, 5.11: 1, 5.12: 14.
	if len(table) != 59 {
		t.Errorf("route table has %d routes, want 59", len(table))
	}

	h, err := New(minimalDeps(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, rt := range table {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(rt.method, concretePath(rt.pattern), nil))
		if rec.Code == http.StatusNotFound && strings.Contains(rec.Body.String(), codeEndpointNotFound) {
			t.Errorf("%s is not registered", rt.key())
		}
	}
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/nope"},
		{http.MethodPut, "/api/links"},
		{http.MethodPost, "/api/auth/me"},
		{http.MethodGet, "/api/"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(c.method, c.path, nil)
		req.Header.Set(headerKinexRequest, "1")
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"code":"endpoint_not_found"`) {
			t.Errorf("%s %s: %d %s", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
}

func TestRoutePermissions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		access                     access
		admin, developer, customer bool
		session, gated             bool
	}{
		{accessPublic, true, true, true, false, false},
		{accessExempt, true, true, true, true, false},
		{accessUser, true, true, true, true, true},
		{accessSharer, true, true, false, true, true},
		{accessAdmin, true, false, false, true, true},
	}
	for _, c := range cases {
		rt := route{access: c.access}
		if rt.permits(model.RoleAdmin) != c.admin || rt.permits(model.RoleDeveloper) != c.developer ||
			rt.permits(model.RoleCustomer) != c.customer || rt.needsSession() != c.session || rt.gated() != c.gated {
			t.Errorf("access %s: unexpected permissions", c.access)
		}
	}
	if access(99).String() != "?" {
		t.Error("unknown access level string")
	}
}

// minimalDeps returns the smallest Deps that New accepts; the store is a
// nilStore, so only paths that never reach it may be exercised.
func minimalDeps(t testing.TB) Deps {
	return Deps{Store: nilStore{}, Hasher: testHasher, Keyring: newTestKeyring(t), UI: testUI()}
}

func TestNewRequiresStore(t *testing.T) {
	t.Parallel()
	if _, err := New(Deps{}); err == nil {
		t.Error("New accepted missing dependencies")
	}
	cases := map[string]func(*Deps){
		"store":      func(d *Deps) { d.Store = nil },
		"hasher":     func(d *Deps) { d.Hasher = nil },
		"keyring":    func(d *Deps) { d.Keyring = nil },
		"ui":         func(d *Deps) { d.UI = nil },
		"index.html": func(d *Deps) { d.UI = fstest.MapFS{"robots.txt": {Data: []byte("x")}} },
	}
	for name, drop := range cases {
		d := minimalDeps(t)
		drop(&d)
		if _, err := New(d); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("missing %s: err = %v", name, err)
		}
	}
	if _, err := New(minimalDeps(t)); err != nil {
		t.Errorf("minimal deps: %v", err)
	}
}
