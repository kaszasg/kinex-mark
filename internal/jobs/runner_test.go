package jobs

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/fetch"
	"github.com/kaszasg/kinex-mark/internal/model"
)

func noFetch(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
	return fetch.Result{}, nil
}

func TestNewDefaults(t *testing.T) {
	t.Parallel()
	r := New(nil, nil, clock.NewFake(time.Time{}), Options{})
	if r.opts.Workers != DefaultWorkers || r.opts.QueueSize != DefaultQueueSize {
		t.Errorf("workers/queue = %d/%d", r.opts.Workers, r.opts.QueueSize)
	}
	if r.opts.Interval != time.Hour || r.opts.SweepInterval != time.Minute {
		t.Errorf("interval/sweep = %v/%v", r.opts.Interval, r.opts.SweepInterval)
	}
	if cap(r.queue) != 256 {
		t.Errorf("queue capacity = %d, want 256", cap(r.queue))
	}
	if r.log == nil || r.opts.NewTicker == nil {
		t.Error("logger or ticker factory not defaulted")
	}
	// The discard logger accepts records.
	r.log.Info("discarded")
}

func TestNewKeepsOptions(t *testing.T) {
	t.Parallel()
	r := New(nil, nil, clock.NewFake(time.Time{}), Options{Workers: 5, QueueSize: 3, Interval: 2 * time.Hour, SweepInterval: 30 * time.Second})
	if r.opts.Workers != 5 || cap(r.queue) != 3 || r.opts.Interval != 2*time.Hour || r.opts.SweepInterval != 30*time.Second {
		t.Fatalf("options not kept: %+v, capacity %d", r.opts, cap(r.queue))
	}
}

func TestRealTicker(t *testing.T) {
	t.Parallel()
	tk := NewRealTicker(time.Hour)
	if tk.C() == nil {
		t.Fatal("C() is nil")
	}
	tk.Stop()
}

func TestScheduleRunsAtOnceAndOnEveryTick(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{Workers: 1}, noFetch)
	h.st.settings.DefaultArchiveDays = 30
	h.st.settings.AuditRetentionDays = 60
	cancel := h.start(t)

	// First cycle at once: archive, then purge; the sweeper also runs at once.
	if got := await(t, h.st.archiveCalls, "first archive"); got != 30 {
		t.Errorf("ArchiveInactive defaultDays = %d, want 30", got)
	}
	if got := await(t, h.st.purgeCalls, "first purge"); got != 60*24*time.Hour {
		t.Errorf("Purge retention = %v, want 1440h", got)
	}
	await(t, h.st.dueCalls, "first sweep")

	maint := h.tk.get(t, DefaultInterval)
	sweep := h.tk.get(t, DefaultSweepInterval)
	for i := 0; i < 2; i++ {
		tick(t, maint)
		await(t, h.st.archiveCalls, "archive after tick")
		await(t, h.st.purgeCalls, "purge after tick")
		tick(t, sweep)
		await(t, h.st.dueCalls, "sweep after tick")
	}
	h.st.mu.Lock()
	order := append([]string(nil), h.st.order...)
	h.st.mu.Unlock()
	want := []string{"archive", "purge", "archive", "purge", "archive", "purge"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("maintenance order = %v, want %v", order, want)
	}

	cancel()
	for _, tk := range []*fakeTicker{maint, sweep} {
		select {
		case <-tk.stopped:
		default:
			t.Errorf("ticker %v not stopped after shutdown", tk.period)
		}
	}
	if got := len(h.log.find(t, "archive run")); got != 3 {
		t.Errorf("archive run log records = %d, want 3", got)
	}
	if got := len(h.log.find(t, "purge run")); got != 3 {
		t.Errorf("purge run log records = %d, want 3", got)
	}
}

func TestScheduleRetriesAfterError(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{Workers: 1}, noFetch)
	h.st.archiveErr = errors.New("database is locked")
	h.start(t)
	await(t, h.st.archiveCalls, "failing archive")
	// The purge still runs after a failed archive.
	await(t, h.st.purgeCalls, "purge after failed archive")
	h.st.mu.Lock()
	h.st.archiveErr = nil
	h.st.mu.Unlock()
	tick(t, h.tk.get(t, DefaultInterval))
	await(t, h.st.archiveCalls, "retried archive")
	await(t, h.st.purgeCalls, "purge after retry")
	if got := len(h.log.find(t, "archive run failed")); got != 1 {
		t.Errorf("archive run failed records = %d, want 1", got)
	}
}

func TestStartTwiceIsNoop(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{Workers: 1}, noFetch)
	ctx, cancel := context.WithCancel(context.Background())
	h.r.Start(ctx)
	h.r.Start(ctx)
	await(t, h.tk.created, "ticker 1")
	await(t, h.tk.created, "ticker 2")
	await(t, h.st.purgeCalls, "first purge")
	await(t, h.st.dueCalls, "first sweep")
	cancel()
	if !h.r.Wait(eventTimeout) {
		t.Fatal("Wait = false")
	}
	if extra := len(h.tk.created); extra != 0 {
		t.Fatalf("second Start created %d more tickers", extra)
	}
}

func TestWaitWithoutStart(t *testing.T) {
	t.Parallel()
	r := New(nil, nil, clock.NewFake(time.Time{}), Options{})
	if !r.Wait(0) || !r.Wait(time.Second) {
		t.Fatal("Wait without Start must report true")
	}
}

func TestEveryStopsOnCancelledContext(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{}, noFetch)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	h.r.every(ctx, time.Minute, func(context.Context) { called = true })
	if called {
		t.Fatal("fn ran on a cancelled context")
	}
	tk := await(t, h.tk.created, "ticker")
	select {
	case <-tk.stopped:
	default:
		t.Fatal("ticker not stopped")
	}
}

func TestMaintainSkipsPurgeAfterShutdown(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{}, noFetch)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.r.maintain(ctx)
	await(t, h.st.archiveCalls, "archive")
	select {
	case <-h.st.purgeCalls:
		t.Fatal("purge ran after the context ended")
	default:
	}
}

// TestShutdownInterruptsFetch: on shutdown the workers abort the running
// fetch, write no result and exit within the Wait timeout (7.5).
func TestShutdownInterruptsFetch(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{Workers: 2}, func(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
		<-ctx.Done()
		return fetch.Result{}, &fetch.Error{Code: model.FetchErrTimeout, Temporary: true, Err: ctx.Err()}
	})
	h.st.sweepOff = true
	h.st.addPending(1, "https://example.com/", model.LocaleHU)
	cancel := h.start(t)
	h.r.Enqueue(1)
	await(t, h.f.calls, "fetch start")
	if h.r.Wait(0) {
		t.Fatal("Wait(0) = true while workers run")
	}
	cancel()
	if got := h.st.appliedResults(); len(got) != 0 {
		t.Fatalf("ApplyFetchResult called on shutdown: %+v", got)
	}
	if l := h.st.link(1); l.status != model.FetchPending || l.attempts != 0 {
		t.Fatalf("link changed on shutdown: %+v", l)
	}
}

// TestWaitTimesOut: a worker that ignores the context keeps Wait false until
// it returns; its late result is not written.
func TestWaitTimesOut(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	h := newHarness(t, Options{Workers: 1}, func(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
		<-release
		return fetch.Result{Title: "late"}, nil
	})
	h.st.sweepOff = true
	h.st.addPending(1, "https://example.com/", model.LocaleEN)
	ctx, cancel := context.WithCancel(context.Background())
	h.r.Start(ctx)
	h.r.Enqueue(1)
	await(t, h.f.calls, "fetch start")
	cancel()
	if h.r.Wait(0) {
		t.Fatal("Wait(0) = true while a worker is stuck")
	}
	// The timeout path itself: the stuck worker cannot exit before it expires.
	if h.r.Wait(time.Millisecond) {
		t.Fatal("Wait(1ms) = true while a worker is stuck")
	}
	close(release)
	if !h.r.Wait(eventTimeout) {
		t.Fatal("Wait = false after the worker returned")
	}
	if !h.r.Wait(0) {
		t.Fatal("Wait(0) = false after every goroutine exited")
	}
	if got := h.st.appliedResults(); len(got) != 0 {
		t.Fatalf("result written after shutdown: %+v", got)
	}
}
