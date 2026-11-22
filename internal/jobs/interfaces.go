// Package jobs runs the background work: automatic archiving, purge and the
// metadata fetch queue with its workers (plan chapter 7, 3.3.9).
package jobs

import (
	"context"
	"time"

	"github.com/kaszasg/kinex-mark/internal/fetch"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Store is the part of store.Store the jobs use. The jobs package never
// imports the store; internal/app asserts the implementation at compile time.
type Store interface {
	Settings(ctx context.Context) (model.SystemSettings, error)
	ArchiveInactive(ctx context.Context, defaultDays int) (int, error)
	Purge(ctx context.Context, auditRetention time.Duration) (model.PurgeResult, error)
	DueFetches(ctx context.Context, limit int) ([]int64, error)
	BeginFetch(ctx context.Context, linkID int64, lease time.Duration) (model.FetchJob, error)
	ApplyFetchResult(ctx context.Context, r model.FetchResult) error
}

// Fetcher downloads the metadata of one page (*fetch.Fetcher implements it).
// acceptLanguage is the owner's locale code ("hu", "en", "de"); the fetcher
// builds the Accept-Language header from it (6.8).
type Fetcher interface {
	Fetch(ctx context.Context, rawURL, acceptLanguage string) (fetch.Result, error)
}

// Ticker is the part of *time.Ticker the scheduler uses; tests inject a
// manually driven implementation through Options.NewTicker.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// realTicker adapts *time.Ticker to Ticker.
type realTicker struct{ t *time.Ticker }

func (r realTicker) C() <-chan time.Time { return r.t.C }
func (r realTicker) Stop()               { r.t.Stop() }

// NewRealTicker returns a Ticker backed by time.NewTicker.
func NewRealTicker(d time.Duration) Ticker { return realTicker{time.NewTicker(d)} }
