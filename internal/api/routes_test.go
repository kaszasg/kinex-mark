package api

import (
	"net/http"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// accessRejectionCodes are the codes of the global and per-route checks. A
// permitted request must never end with one of them, whatever the handler
// does with the (deliberately minimal) request.
var accessRejectionCodes = map[string]bool{
	codeUnauthenticated:        true,
	codeForbidden:              true,
	codeShareNotAllowed:        true,
	codePasswordChangeRequired: true,
	codeConsentRequired:        true,
	codeCSRFRejected:           true,
	codeHostMismatch:           true,
	codeEndpointNotFound:       true,
}

// TestRoutePermissionMatrix is 9.2.3 group 1: every row of the route table
// with an anonymous, a customer, a developer and an admin client. The
// boundary is 401 for anonymous callers of non-public routes and the row's
// deny code (403) for roles below its level (5.13).
func TestRoutePermissionMatrix(t *testing.T) {
	t.Parallel()
	for _, rt := range routes() {
		t.Run(rt.key(), func(t *testing.T) {
			t.Parallel()
			// A fresh server per route: some permitted calls change shared
			// state (privacy notice bump, sign-out).
			e := newEnv(t)
			path := concretePath(rt.pattern)
			var body any
			if isUnsafeMethod(rt.method) {
				body = map[string]any{}
			}

			res := e.client(t).do(rt.method, path, body)
			if rt.needsSession() {
				expectError(t, res, http.StatusUnauthorized, codeUnauthenticated)
			} else {
				expectPermitted(t, res)
			}

			for _, role := range model.Roles {
				c, _ := e.loginAs(t, role)
				res := c.do(rt.method, path, body)
				if !rt.permits(role) {
					expectError(t, res, http.StatusForbidden, rt.deny)
					continue
				}
				expectPermitted(t, res)
			}
		})
	}
}

// expectPermitted fails when res is a rejection by an access check.
func expectPermitted(t *testing.T, res *http.Response) {
	t.Helper()
	body := readBody(t, res)
	if res.StatusCode == http.StatusUnauthorized {
		t.Fatalf("%s %s: 401 %s", res.Request.Method, res.Request.URL.Path, body)
	}
	if res.Header.Get("Content-Type") == "application/json; charset=utf-8" && len(body) > 0 && body[0] == '{' {
		var eb errorBody
		decodeInto(t, body, &eb)
		if accessRejectionCodes[eb.Code] {
			t.Fatalf("%s %s: rejected with %d %s", res.Request.Method, res.Request.URL.Path, res.StatusCode, eb.Code)
		}
	}
}
