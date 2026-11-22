// Package storetest provides a real store in a temporary directory and seed
// helpers for the tests of packages outside internal/store (api, jobs, app).
// Only _test.go files import it; it is excluded from the coverage gate.
//
// The store's own tests (package store) cannot import it, because that would
// be an import cycle; they use their own newTestStore and seedUser helpers.
package storetest

import (
	"context"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/store"
	"github.com/kaszasg/kinex-mark/internal/testutil"
)

// Option configures NewStore.
type Option func(*options)

type options struct {
	clock clock.Clock
	dir   string
}

// WithClock makes the store use c for every timestamp it writes. Without it
// the store gets its own fake clock set to testutil.T0.
func WithClock(c clock.Clock) Option {
	return func(o *options) { o.clock = c }
}

// WithDir opens the store in dir instead of a fresh t.TempDir().
func WithDir(dir string) Option {
	return func(o *options) { o.dir = dir }
}

// NewStore opens a migrated store in a temporary directory and closes it
// when the test ends.
func NewStore(t testing.TB, opts ...Option) *store.Store {
	t.Helper()
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.clock == nil {
		o.clock = testutil.NewClock()
	}
	if o.dir == "" {
		o.dir = t.TempDir()
	}
	st, err := store.Open(context.Background(), o.dir, store.Options{Clock: o.clock})
	if err != nil {
		t.Fatalf("storetest: open store: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("storetest: close store: %v", err)
		}
	})
	return st
}
