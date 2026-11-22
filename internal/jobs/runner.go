package jobs

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/kaszasg/kinex-mark/internal/clock"
)

// Scheduling defaults (7.1, 7.3).
const (
	DefaultWorkers       = 2
	DefaultQueueSize     = 256
	DefaultInterval      = time.Hour   // archive, then purge
	DefaultSweepInterval = time.Minute // metadata fetch sweeper
)

// Options configures a Runner. Zero values take the defaults above.
type Options struct {
	Workers       int           // metadata fetch workers (APP_FETCH_WORKERS)
	QueueSize     int           // capacity of the fetch queue
	Interval      time.Duration // period of the archive and purge run
	SweepInterval time.Duration // period of the fetch sweeper
	Log           *slog.Logger  // nil discards the log
	// NewTicker creates the scheduling tickers; nil uses NewRealTicker.
	// Tests inject a manually driven ticker, so no test sleeps.
	NewTicker func(d time.Duration) Ticker
}

// Runner owns the background goroutines: the archive and purge scheduler,
// the fetch sweeper and the fetch workers (7.1).
type Runner struct {
	st    Store
	f     Fetcher
	clk   clock.Clock
	opts  Options
	log   *slog.Logger
	queue chan int64

	mu       sync.Mutex
	inflight map[int64]struct{} // queued or being processed (7.3)
	exited   chan struct{}      // created by Start, closed when its goroutines have exited

	maintMu sync.Mutex // serialises archive and purge runs (scheduler and admin endpoint)
	wg      sync.WaitGroup

	// onProcessed, when set (tests only), runs after a worker has finished a
	// link and removed it from the in-flight set.
	onProcessed func(linkID int64)
}

// New returns a Runner. Enqueue works before Start: queued links wait for
// the workers.
func New(st Store, f Fetcher, clk clock.Clock, opts Options) *Runner {
	if opts.Workers <= 0 {
		opts.Workers = DefaultWorkers
	}
	if opts.QueueSize <= 0 {
		opts.QueueSize = DefaultQueueSize
	}
	if opts.Interval <= 0 {
		opts.Interval = DefaultInterval
	}
	if opts.SweepInterval <= 0 {
		opts.SweepInterval = DefaultSweepInterval
	}
	if opts.NewTicker == nil {
		opts.NewTicker = NewRealTicker
	}
	log := opts.Log
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Runner{
		st:       st,
		f:        f,
		clk:      clk,
		opts:     opts,
		log:      log,
		queue:    make(chan int64, opts.QueueSize),
		inflight: map[int64]struct{}{},
	}
}

// Start launches the scheduler (archive, then purge: at once and every
// Interval), the sweeper (at once and every SweepInterval) and Workers fetch
// workers. They stop when ctx is done; Wait then waits for them. A second
// call is a no-op.
func (r *Runner) Start(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.exited != nil {
		return
	}
	r.exited = make(chan struct{})

	r.goRun(func() { r.every(ctx, r.opts.Interval, r.maintain) })
	r.goRun(func() { r.every(ctx, r.opts.SweepInterval, r.sweep) })
	for i := 0; i < r.opts.Workers; i++ {
		r.goRun(func() { r.work(ctx) })
	}
	go func(exited chan struct{}) {
		r.wg.Wait()
		close(exited)
	}(r.exited)
}

// goRun starts fn as a goroutine tracked by Wait.
func (r *Runner) goRun(fn func()) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		fn()
	}()
}

// every runs fn at once, then on every tick, until ctx is done.
func (r *Runner) every(ctx context.Context, period time.Duration, fn func(context.Context)) {
	t := r.opts.NewTicker(period)
	defer t.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		fn(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C():
		}
	}
}

// maintain is one scheduled cycle: archive, then purge (7.1). Failures are
// logged by RunArchive and RunPurge; the next cycle retries.
func (r *Runner) maintain(ctx context.Context) {
	_, _ = r.RunArchive(ctx)
	if ctx.Err() != nil {
		return
	}
	_, _ = r.RunPurge(ctx)
}

// Wait waits up to timeout for the goroutines started by Start to exit after
// their context is done, and reports whether they did. A timeout of zero or
// less only checks. Without Start it returns true.
func (r *Runner) Wait(timeout time.Duration) bool {
	r.mu.Lock()
	exited := r.exited
	r.mu.Unlock()
	if exited == nil {
		return true
	}
	if timeout <= 0 {
		select {
		case <-exited:
			return true
		default:
			return false
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-exited:
		return true
	case <-timer.C:
		return false
	}
}
