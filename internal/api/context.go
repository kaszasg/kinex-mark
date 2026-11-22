package api

import (
	"context"
	"net/http"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// authInfo is the authenticated identity of a request, stored in the
// request context by the authenticate middleware.
type authInfo struct {
	User    model.User
	Session model.Session
}

// authKey is the context key of authInfo.
type authKey struct{}

// withAuth returns ctx carrying the authenticated identity.
func withAuth(ctx context.Context, a authInfo) context.Context {
	return context.WithValue(ctx, authKey{}, a)
}

// authFrom returns the authenticated identity of r, if any.
func authFrom(r *http.Request) (authInfo, bool) {
	a, ok := r.Context().Value(authKey{}).(authInfo)
	return a, ok
}

// userFrom returns the authenticated user; the zero User on public routes.
func userFrom(r *http.Request) model.User {
	a, _ := authFrom(r)
	return a.User
}

// sessionFrom returns the current session; the zero Session on public routes.
func sessionFrom(r *http.Request) model.Session {
	a, _ := authFrom(r)
	return a.Session
}

// viewerFrom returns the visibility identity of the authenticated user; the
// zero Viewer (user id 0, which owns and sees nothing) on public routes.
func viewerFrom(r *http.Request) model.Viewer {
	return userFrom(r).Viewer()
}
