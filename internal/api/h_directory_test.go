package api

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/storetest"
)

func TestDirectorySearch(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, me := e.loginAs(t, model.RoleDeveloper)
	prefix := authUsername("dir")
	seed := func(suffix, display string, role model.Role, active bool) model.User {
		u := storetest.SeedNewUser(t, e.store, model.NewUser{
			Username: prefix + suffix, DisplayName: display, Role: role, PasswordHash: storetest.SeedPasswordHash,
		})
		if !active {
			off := false
			if _, err := e.store.UpdateUser(e.ctx(), u.ID, model.UserPatch{Active: &off}); err != nil {
				t.Fatal(err)
			}
		}
		return u
	}
	b := seed("b", "Szabó Árpád", model.RoleCustomer, true)
	a := seed("a", "", model.RoleAdmin, true)
	seed("c", "Inactive", model.RoleCustomer, false)

	search := func(q string) []model.DirectoryUser {
		t.Helper()
		return decodeStatus[listResponse[model.DirectoryUser]](t, c.get("/api/users/directory?q="+url.QueryEscape(q)), http.StatusOK).Items
	}
	got := search(strings.ToUpper(prefix))
	if len(got) != 2 || got[0].ID != a.ID || got[1].ID != b.ID || got[1].DisplayName != "Szabó Árpád" || got[0].Role != model.RoleAdmin {
		t.Fatalf("directory %+v", got)
	}
	// Display name match, case-insensitive, accented.
	if got := search("árpád"); len(got) != 1 || got[0].ID != b.ID {
		t.Errorf("display name search %+v", got)
	}
	// The caller is never listed; no query lists everyone else.
	for _, d := range search("") {
		if d.ID == me.ID {
			t.Error("caller listed")
		}
	}
	if got := search(me.Username); len(got) != 0 {
		t.Errorf("own name search %+v", got)
	}
	body := expectStatus(t, c.get("/api/users/directory?q="+url.QueryEscape(prefix+"a")), http.StatusOK)
	assertJSON(t, body, `{"items":[{"id":`+strconv.FormatInt(a.ID, 10)+`,"username":"`+a.Username+`","displayName":"","role":"admin"}]}`)

	for _, q := range []string{strings.Repeat("x", 65), "a\u0007"} {
		res := c.get("/api/users/directory?q=" + url.QueryEscape(q))
		if b := expectError(t, res, http.StatusBadRequest, codeInvalidQuery); b.Fields["q"] != fieldInvalid {
			t.Errorf("fields %v", b.Fields)
		}
	}

	// Customers may not search (5.11).
	cust, _ := e.loginAs(t, model.RoleCustomer)
	expectError(t, cust.get("/api/users/directory"), http.StatusForbidden, codeForbidden)
}

func TestDirectoryLimit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := e.loginAs(t, model.RoleAdmin)
	prefix := authUsername("many")
	for i := 0; i < directoryLimit+3; i++ {
		storetest.SeedNewUser(t, e.store, model.NewUser{
			Username: prefix + "x" + strconv.Itoa(i), Role: model.RoleCustomer, PasswordHash: storetest.SeedPasswordHash,
		})
	}
	got := decodeStatus[listResponse[model.DirectoryUser]](t, c.get("/api/users/directory?q="+prefix), http.StatusOK).Items
	if len(got) != directoryLimit {
		t.Errorf("%d entries, want %d", len(got), directoryLimit)
	}
}

func TestDirectoryFault(t *testing.T) {
	t.Parallel()
	e, fs := accountFaultEnv(t)
	runAccountFaults(t, fs, []accountFault{{
		name: "directory", method: "Directory", nth: 1,
		prepare: func() func() *http.Response {
			c, _ := e.loginAs(t, model.RoleDeveloper)
			return func() *http.Response { return c.get("/api/users/directory?q=a") }
		},
	}})
}
