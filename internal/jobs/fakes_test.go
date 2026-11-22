package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/fetch"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// eventTimeout guards channel waits against deadlocks. Tests never sleep:
// every wait ends as soon as the awaited event arrives.
const eventTimeout = 10 * time.Second

// await receives one value from ch or fails the test.
func await[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(eventTimeout):
		t.Fatalf("timed out waiting for %s", what)
		panic("unreachable")
	}
}

// fakeLink is the fetch state of one link in fakeStore.
type fakeLink struct {
	url      string
	locale   model.Locale
	status   model.FetchStatus
	next     *time.Time // next_fetch_at; nil when not scheduled
	attempts int
	errCode  string
}

// fakeStore implements Store in memory. Its fetch bookkeeping follows plan
// 7.3 (lease, backoff with model.FetchBackoff, final states), so the runner
// can be tested end to end without SQLite. Every call is reported on a
// buffered channel for synchronisation.
type fakeStore struct {
	clk *clock.Fake

	mu          sync.Mutex
	settings    model.SystemSettings
	settingsErr error
	archiveN    int
	archiveErr  error
	purgeRes    model.PurgeResult
	purgeErr    error
	dueErr      error
	beginErr    error
	applyErr    error
	sweepOff    bool // DueFetches returns nothing
	links       map[int64]*fakeLink
	applied     []model.FetchResult
	begins      []int64
	order       []string // "archive" and "purge" in call order

	archiveCalls chan int           // defaultDays
	purgeCalls   chan time.Duration // auditRetention
	dueCalls     chan int           // number of ids returned
	applyCalls   chan model.FetchResult
}

func newFakeStore(clk *clock.Fake) *fakeStore {
	return &fakeStore{
		clk:          clk,
		settings:     model.DefaultSystemSettings(),
		links:        map[int64]*fakeLink{},
		archiveCalls: make(chan int, 100),
		purgeCalls:   make(chan time.Duration, 100),
		dueCalls:     make(chan int, 100),
		applyCalls:   make(chan model.FetchResult, 100),
	}
}

// addPending adds a link that is due now.
func (s *fakeStore) addPending(id int64, rawURL string, locale model.Locale) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clk.Now()
	s.links[id] = &fakeLink{url: rawURL, locale: locale, status: model.FetchPending, next: &now}
}

func (s *fakeStore) link(id int64) fakeLink {
	s.mu.Lock()
	defer s.mu.Unlock()
	return *s.links[id]
}

func (s *fakeStore) beginCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.begins)
}

func (s *fakeStore) appliedResults() []model.FetchResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]model.FetchResult(nil), s.applied...)
}

func (s *fakeStore) Settings(ctx context.Context) (model.SystemSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings, s.settingsErr
}

func (s *fakeStore) ArchiveInactive(ctx context.Context, defaultDays int) (int, error) {
	s.mu.Lock()
	s.order = append(s.order, "archive")
	n, err := s.archiveN, s.archiveErr
	s.mu.Unlock()
	s.archiveCalls <- defaultDays
	return n, err
}

func (s *fakeStore) Purge(ctx context.Context, auditRetention time.Duration) (model.PurgeResult, error) {
	s.mu.Lock()
	s.order = append(s.order, "purge")
	res, err := s.purgeRes, s.purgeErr
	s.mu.Unlock()
	s.purgeCalls <- auditRetention
	return res, err
}

func (s *fakeStore) DueFetches(ctx context.Context, limit int) ([]int64, error) {
	s.mu.Lock()
	var ids []int64
	err := s.dueErr
	if err == nil && !s.sweepOff {
		now := s.clk.Now()
		for id, l := range s.links {
			if l.status == model.FetchPending && l.next != nil && !l.next.After(now) {
				ids = append(ids, id)
			}
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		if len(ids) > limit {
			ids = ids[:limit]
		}
	}
	s.mu.Unlock()
	s.dueCalls <- len(ids)
	return ids, err
}

func (s *fakeStore) BeginFetch(ctx context.Context, linkID int64, lease time.Duration) (model.FetchJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.begins = append(s.begins, linkID)
	if s.beginErr != nil {
		return model.FetchJob{}, s.beginErr
	}
	l, ok := s.links[linkID]
	now := s.clk.Now()
	if !ok || l.status != model.FetchPending || l.next == nil || l.next.After(now) {
		return model.FetchJob{}, model.ErrNotFound
	}
	until := now.Add(lease)
	l.next = &until
	return model.FetchJob{LinkID: linkID, URL: l.url, OwnerID: 7, Locale: l.locale}, nil
}

func (s *fakeStore) ApplyFetchResult(ctx context.Context, r model.FetchResult) error {
	s.mu.Lock()
	s.applied = append(s.applied, r)
	err := s.applyErr
	if l, ok := s.links[r.LinkID]; ok && err == nil && l.url == r.URL {
		now := s.clk.Now()
		l.errCode = r.Error
		switch {
		case r.Temporary:
			l.attempts++
			if l.attempts < model.FetchMaxAttempts {
				next := now.Add(model.FetchBackoff(l.attempts))
				l.next = &next
			} else {
				l.status, l.next = model.FetchFailed, nil
			}
		default:
			l.status, l.next = r.Status, nil
		}
	}
	s.mu.Unlock()
	s.applyCalls <- r
	return err
}

// fetchCall records one Fetch invocation.
type fetchCall struct {
	url, lang string
}

// fakeFetcher calls fn and reports every call on calls.
type fakeFetcher struct {
	fn    func(ctx context.Context, rawURL, lang string) (fetch.Result, error)
	calls chan fetchCall
}

func newFakeFetcher(fn func(ctx context.Context, rawURL, lang string) (fetch.Result, error)) *fakeFetcher {
	return &fakeFetcher{fn: fn, calls: make(chan fetchCall, 100)}
}

func (f *fakeFetcher) Fetch(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
	f.calls <- fetchCall{url: rawURL, lang: lang}
	return f.fn(ctx, rawURL, lang)
}

// fakeTicker is driven by the test: tick blocks until the loop receives.
type fakeTicker struct {
	period  time.Duration
	ch      chan time.Time
	stopped chan struct{}
	once    sync.Once
}

func (t *fakeTicker) C() <-chan time.Time { return t.ch }
func (t *fakeTicker) Stop()               { t.once.Do(func() { close(t.stopped) }) }

// tickers is an Options.NewTicker implementation that hands every created
// ticker to the test.
type tickers struct {
	mu      sync.Mutex
	byDur   map[time.Duration]*fakeTicker
	created chan *fakeTicker
}

func newTickers() *tickers {
	return &tickers{byDur: map[time.Duration]*fakeTicker{}, created: make(chan *fakeTicker, 10)}
}

func (ts *tickers) New(d time.Duration) Ticker {
	t := &fakeTicker{period: d, ch: make(chan time.Time), stopped: make(chan struct{})}
	ts.mu.Lock()
	ts.byDur[d] = t
	ts.mu.Unlock()
	ts.created <- t
	return t
}

func (ts *tickers) get(t *testing.T, d time.Duration) *fakeTicker {
	t.Helper()
	ts.mu.Lock()
	defer ts.mu.Unlock()
	tk, ok := ts.byDur[d]
	if !ok {
		t.Fatalf("no ticker with period %v", d)
	}
	return tk
}

// tick delivers one tick to tk.
func tick(t *testing.T, tk *fakeTicker) {
	t.Helper()
	select {
	case tk.ch <- time.Time{}:
	case <-time.After(eventTimeout):
		t.Fatalf("ticker %v: tick not received", tk.period)
	}
}

// syncBuffer is a goroutine-safe log sink.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// records decodes the JSON log lines.
func (b *syncBuffer) records(t *testing.T) []map[string]any {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(b.buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// find returns the log records with the given message.
func (b *syncBuffer) find(t *testing.T, msg string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, rec := range b.records(t) {
		if rec["msg"] == msg {
			out = append(out, rec)
		}
	}
	return out
}

// newLogger returns a Debug-level JSON logger writing to a fresh buffer.
func newLogger() (*slog.Logger, *syncBuffer) {
	buf := &syncBuffer{}
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), buf
}

// harness bundles a runner with its fakes.
type harness struct {
	clk       *clock.Fake
	st        *fakeStore
	f         *fakeFetcher
	tk        *tickers
	log       *syncBuffer
	r         *Runner
	processed chan int64
}

func newHarness(t *testing.T, opts Options, fn func(ctx context.Context, rawURL, lang string) (fetch.Result, error)) *harness {
	t.Helper()
	clk := clock.NewFake(time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC))
	h := &harness{
		clk:       clk,
		st:        newFakeStore(clk),
		f:         newFakeFetcher(fn),
		tk:        newTickers(),
		processed: make(chan int64, 100),
	}
	var log *slog.Logger
	log, h.log = newLogger()
	opts.Log = log
	opts.NewTicker = h.tk.New
	h.r = New(h.st, h.f, clk, opts)
	h.r.onProcessed = func(id int64) { h.processed <- id }
	return h
}

// start starts the runner and waits for both tickers; the returned cancel
// stops it and checks that Wait succeeds.
func (h *harness) start(t *testing.T) (cancel func()) {
	t.Helper()
	ctx, stop := context.WithCancel(context.Background())
	h.r.Start(ctx)
	await(t, h.tk.created, "first ticker")
	await(t, h.tk.created, "second ticker")
	stopped := false
	cancel = func() {
		if stopped {
			return
		}
		stopped = true
		stop()
		if !h.r.Wait(eventTimeout) {
			t.Error("runner goroutines did not exit")
		}
	}
	t.Cleanup(cancel)
	return cancel
}
