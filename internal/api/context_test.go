package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestAuthContextRoundTrip(t *testing.T) {
	t.Parallel()
	user := model.User{ID: 12, Username: "dev", Role: model.RoleDeveloper, Active: true}
	session := model.Session{ID: 3, UserID: 12, CreatedAt: time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req = req.WithContext(withAuth(req.Context(), authInfo{User: user, Session: session}))

	a, ok := authFrom(req)
	if !ok || a.User.ID != 12 || a.Session.ID != 3 {
		t.Fatalf("authFrom = %+v, %v", a, ok)
	}
	if got := userFrom(req); got.ID != user.ID || got.Role != user.Role {
		t.Errorf("userFrom = %+v", got)
	}
	if got := sessionFrom(req); got.ID != session.ID || !got.CreatedAt.Equal(session.CreatedAt) {
		t.Errorf("sessionFrom = %+v", got)
	}
	if got := viewerFrom(req); got != (model.Viewer{UserID: 12, Role: model.RoleDeveloper}) {
		t.Errorf("viewerFrom = %+v", got)
	}
}

func TestAuthContextMissing(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/public/config", nil)
	if _, ok := authFrom(req); ok {
		t.Error("authFrom reported an identity on an anonymous request")
	}
	if userFrom(req).ID != 0 || sessionFrom(req).ID != 0 || viewerFrom(req) != (model.Viewer{}) {
		t.Error("anonymous accessors must return zero values")
	}

	// A foreign value under another key does not leak in.
	type otherKey struct{}
	req = req.WithContext(context.WithValue(req.Context(), otherKey{}, authInfo{User: model.User{ID: 9}}))
	if _, ok := authFrom(req); ok {
		t.Error("authFrom read a value stored under a different key")
	}
}
