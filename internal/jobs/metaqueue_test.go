package jobs

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/fetch"
	"github.com/kaszasg/kinex-mark/internal/model"
)

var testIcon = &fetch.Icon{Data: []byte{0x89, 'P', 'N', 'G'}, ContentType: "image/png", SHA256: "abc123"}

func wantBlob() *model.IconBlob {
	return &model.IconBlob{Data: testIcon.Data, ContentType: testIcon.ContentType, SHA256: testIcon.SHA256}
}

func TestEnqueueQueueFull(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{QueueSize: 2}, noFetch)
	if !h.r.Enqueue(1) || !h.r.Enqueue(2) {
		t.Fatal("Enqueue into a free queue returned false")
	}
	if !h.r.Enqueue(1) {
		t.Fatal("Enqueue of a queued link must report true")
	}
	if got := len(h.r.queue); got != 2 {
		t.Fatalf("queue length = %d, want 2 (no duplicate)", got)
	}
	if h.r.Enqueue(3) {
		t.Fatal("Enqueue into a full queue returned true")
	}
	if _, ok := h.r.inflight[3]; ok {
		t.Fatal("rejected link left in the in-flight set")
	}
	// Once a slot is free, the rejected link is accepted.
	<-h.r.queue
	if !h.r.Enqueue(3) {
		t.Fatal("Enqueue after a free slot returned false")
	}
}

func TestEnqueueDedupWhileProcessing(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	h := newHarness(t, Options{Workers: 2}, func(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
		<-release
		return fetch.Result{Title: "Example"}, nil
	})
	h.st.sweepOff = true
	h.st.addPending(1, "https://example.com/", model.LocaleEN)
	h.start(t)
	if !h.r.Enqueue(1) {
		t.Fatal("Enqueue = false")
	}
	await(t, h.f.calls, "fetch")
	// While link 1 is being fetched, enqueueing it again is accepted but not queued.
	if !h.r.Enqueue(1) || len(h.r.queue) != 0 {
		t.Fatalf("duplicate queued: queue length %d", len(h.r.queue))
	}
	close(release)
	await(t, h.processed, "processed")
	if got := h.st.beginCount(); got != 1 {
		t.Fatalf("BeginFetch calls = %d, want 1", got)
	}
	// After processing the link may be queued again; the store lease decides.
	if !h.r.Enqueue(1) {
		t.Fatal("Enqueue after processing = false")
	}
	await(t, h.processed, "second processing")
	if got := h.st.beginCount(); got != 2 {
		t.Fatalf("BeginFetch calls = %d, want 2", got)
	}
	if len(h.st.appliedResults()) != 1 {
		t.Fatal("a link that is no longer pending was fetched again")
	}
}

// TestTransientBackoff: a transient error is retried after 1 minute, then
// after 10 minutes, and the third failure is final (7.3). The sweeper picks
// the link up only when it is due again.
func TestTransientBackoff(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{Workers: 1}, func(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
		return fetch.Result{}, &fetch.Error{Code: model.FetchErrTimeout, Temporary: true}
	})
	h.st.addPending(1, "https://slow.example.com/", model.LocaleDE)
	h.start(t)
	sweep := h.tk.get(t, DefaultSweepInterval)

	// Attempt 1 from the immediate sweep.
	if n := await(t, h.st.dueCalls, "first sweep"); n != 1 {
		t.Fatalf("first sweep due = %d", n)
	}
	await(t, h.processed, "attempt 1")
	if l := h.st.link(1); l.status != model.FetchPending || l.attempts != 1 || !l.next.Equal(h.clk.Now().Add(time.Minute)) {
		t.Fatalf("after attempt 1: %+v", l)
	}

	steps := []struct {
		advance time.Duration
		due     int
	}{
		{59 * time.Second, 0}, // not due before the 1 minute backoff
		{time.Second, 1},      // attempt 2
		{10*time.Minute - time.Second, 0},
		{time.Second, 1}, // attempt 3
	}
	for i, s := range steps {
		h.clk.Advance(s.advance)
		tick(t, sweep)
		if n := await(t, h.st.dueCalls, "sweep"); n != s.due {
			t.Fatalf("step %d: due = %d, want %d", i, n, s.due)
		}
		if s.due == 1 {
			await(t, h.processed, "attempt")
		}
		if i == 1 {
			if l := h.st.link(1); l.attempts != 2 || !l.next.Equal(h.clk.Now().Add(10*time.Minute)) {
				t.Fatalf("after attempt 2: %+v", l)
			}
		}
	}
	l := h.st.link(1)
	if l.status != model.FetchFailed || l.next != nil || l.attempts != 3 || l.errCode != model.FetchErrTimeout {
		t.Fatalf("after attempt 3: %+v", l)
	}
	for i, r := range h.st.appliedResults() {
		want := model.FetchResult{LinkID: 1, URL: "https://slow.example.com/", Status: model.FetchFailed, Error: model.FetchErrTimeout, Temporary: true}
		if !reflect.DeepEqual(r, want) {
			t.Errorf("result %d = %+v, want %+v", i, r, want)
		}
	}
	// A final link is never due again.
	h.clk.Advance(time.Hour)
	tick(t, sweep)
	if n := await(t, h.st.dueCalls, "final sweep"); n != 0 {
		t.Fatalf("failed link due again: %d", n)
	}
}

// TestWorkerOutcomes runs each outcome class through a worker and checks the
// exact ApplyFetchResult argument and the resulting link state.
func TestWorkerOutcomes(t *testing.T) {
	t.Parallel()
	const link = "https://example.com/page"
	tests := []struct {
		name       string
		res        fetch.Result
		err        error
		want       model.FetchResult
		wantStatus model.FetchStatus
	}{
		{
			name:       "success",
			res:        fetch.Result{FinalURL: link, Title: "Example page", Icon: testIcon},
			want:       model.FetchResult{Status: model.FetchOK, Title: "Example page", Icon: wantBlob()},
			wantStatus: model.FetchOK,
		},
		{
			name:       "success without icon",
			res:        fetch.Result{FinalURL: link, Title: ""},
			want:       model.FetchResult{Status: model.FetchOK},
			wantStatus: model.FetchOK,
		},
		{
			name:       "final tls error",
			err:        &fetch.Error{Code: model.FetchErrTLS},
			want:       model.FetchResult{Status: model.FetchFailed, Error: model.FetchErrTLS},
			wantStatus: model.FetchFailed,
		},
		{
			name:       "blocked address",
			err:        &fetch.Error{Code: model.FetchErrBlockedAddress},
			want:       model.FetchResult{Status: model.FetchBlocked, Error: model.FetchErrBlockedAddress},
			wantStatus: model.FetchBlocked,
		},
		{
			name:       "blocked port is never temporary",
			err:        &fetch.Error{Code: model.FetchErrBlockedPort, Temporary: true},
			want:       model.FetchResult{Status: model.FetchBlocked, Error: model.FetchErrBlockedPort},
			wantStatus: model.FetchBlocked,
		},
		{
			name:       "not_html keeps the code with status ok",
			res:        fetch.Result{FinalURL: link, Icon: testIcon, PageErr: &fetch.Error{Code: model.FetchErrNotHTML}},
			want:       model.FetchResult{Status: model.FetchOK, Error: model.FetchErrNotHTML, Icon: wantBlob()},
			wantStatus: model.FetchOK,
		},
		{
			name:       "page 404 with favicon fallback",
			res:        fetch.Result{FinalURL: link, Title: "ignored", Icon: testIcon, PageErr: &fetch.Error{Code: model.FetchErrHTTPStatus}},
			want:       model.FetchResult{Status: model.FetchFailed, Error: model.FetchErrHTTPStatus, Icon: wantBlob()},
			wantStatus: model.FetchFailed,
		},
		{
			name:       "page 503 is temporary",
			res:        fetch.Result{FinalURL: link, PageErr: &fetch.Error{Code: model.FetchErrHTTPStatus, Temporary: true}},
			want:       model.FetchResult{Status: model.FetchFailed, Error: model.FetchErrHTTPStatus, Temporary: true},
			wantStatus: model.FetchPending,
		},
		{
			name:       "too many redirects",
			err:        fmt.Errorf("wrapped: %w", &fetch.Error{Code: model.FetchErrTooManyRedirects}),
			want:       model.FetchResult{Status: model.FetchFailed, Error: model.FetchErrTooManyRedirects},
			wantStatus: model.FetchFailed,
		},
		{
			name:       "unknown error is a temporary connection error",
			err:        errors.New("something unexpected"),
			want:       model.FetchResult{Status: model.FetchFailed, Error: model.FetchErrConnection, Temporary: true},
			wantStatus: model.FetchPending,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, Options{Workers: 1}, func(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
				return tc.res, tc.err
			})
			h.st.sweepOff = true
			h.st.addPending(42, link, model.LocaleHU)
			h.start(t)
			h.r.Enqueue(42)
			call := await(t, h.f.calls, "fetch")
			if call.url != link || call.lang != "hu" {
				t.Errorf("Fetch(%q, %q), want (%q, %q)", call.url, call.lang, link, "hu")
			}
			got := await(t, h.st.applyCalls, "apply")
			want := tc.want
			want.LinkID, want.URL = 42, link
			if !reflect.DeepEqual(got, want) {
				t.Errorf("ApplyFetchResult(%+v)\nwant            %+v", got, want)
			}
			await(t, h.processed, "processed")
			if l := h.st.link(42); l.status != tc.wantStatus {
				t.Errorf("link status = %q, want %q", l.status, tc.wantStatus)
			}
		})
	}
}

func TestBuildResultTypedNilError(t *testing.T) {
	t.Parallel()
	var fe *fetch.Error
	got := buildResult(model.FetchJob{LinkID: 1, URL: "https://a.example/"}, fetch.Result{}, fe)
	if got.Status != model.FetchFailed || got.Error != model.FetchErrConnection || !got.Temporary {
		t.Fatalf("typed nil error: %+v", got)
	}
}

// TestFailureLogHasNoURL: fetch failures are logged at Debug level with the
// link id, the code and the host only (7.3).
func TestFailureLogHasNoURL(t *testing.T) {
	t.Parallel()
	const secretURL = "https://intranet.example.com/private/report?token=s3cr3t#frag"
	h := newHarness(t, Options{Workers: 1}, func(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
		inner := &url.Error{Op: "Get", URL: rawURL, Err: errors.New("dial tcp 10.0.0.5:443: connection refused")}
		return fetch.Result{}, &fetch.Error{Code: model.FetchErrConnection, Temporary: true, Err: inner}
	})
	h.st.sweepOff = true
	h.st.addPending(7, secretURL, model.LocaleEN)
	h.start(t)
	h.r.Enqueue(7)
	await(t, h.processed, "processed")

	recs := h.log.find(t, "metadata fetch failed")
	if len(recs) != 1 {
		t.Fatalf("metadata fetch failed records = %d, want 1", len(recs))
	}
	rec := recs[0]
	wantKeys := map[string]bool{"time": true, "level": true, "msg": true, "link_id": true, "code": true, "host": true}
	for k := range rec {
		if !wantKeys[k] {
			t.Errorf("unexpected log attribute %q = %v", k, rec[k])
		}
	}
	if rec["level"] != "DEBUG" || rec["link_id"] != float64(7) || rec["code"] != "connection" || rec["host"] != "intranet.example.com" {
		t.Errorf("record = %v", rec)
	}
	all := h.log.String()
	for _, leak := range []string{"/private", "report", "token", "s3cr3t", "frag", "dial tcp", "10.0.0.5", "https://"} {
		if strings.Contains(all, leak) {
			t.Errorf("log contains %q:\n%s", leak, all)
		}
	}
	for _, rec := range h.log.records(t) {
		if rec["level"] != "DEBUG" && strings.HasPrefix(rec["msg"].(string), "metadata") {
			t.Errorf("metadata record above Debug level: %v", rec)
		}
	}
}

func TestSuccessIsNotLogged(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{Workers: 1}, func(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
		return fetch.Result{Title: "Fine"}, nil
	})
	h.st.sweepOff = true
	h.st.addPending(1, "https://example.com/?q=1", model.LocaleEN)
	h.start(t)
	h.r.Enqueue(1)
	await(t, h.processed, "processed")
	if recs := h.log.find(t, "metadata fetch failed"); len(recs) != 0 {
		t.Fatalf("success logged as failure: %v", recs)
	}
}

func TestLeaseNotFoundSkipsSilently(t *testing.T) {
	t.Parallel()
	h := newHarness(t, Options{Workers: 1}, noFetch)
	h.st.sweepOff = true
	h.start(t)
	h.r.Enqueue(99) // unknown link: BeginFetch returns ErrNotFound
	await(t, h.processed, "processed")
	if len(h.f.calls) != 0 || len(h.st.appliedResults()) != 0 {
		t.Fatal("a link without a lease was fetched")
	}
	if recs := h.log.find(t, "metadata fetch lease failed"); len(recs) != 0 {
		t.Fatalf("ErrNotFound logged: %v", recs)
	}
}

func TestStoreErrorsAreLoggedAtDebug(t *testing.T) {
	t.Parallel()
	t.Run("lease", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, Options{Workers: 1}, noFetch)
		h.st.sweepOff = true
		h.st.beginErr = errors.New("database is locked")
		h.start(t)
		h.r.Enqueue(5)
		await(t, h.processed, "processed")
		recs := h.log.find(t, "metadata fetch lease failed")
		if len(recs) != 1 || recs[0]["level"] != "DEBUG" || recs[0]["link_id"] != float64(5) {
			t.Fatalf("lease failure log = %v", recs)
		}
		if len(h.f.calls) != 0 {
			t.Fatal("fetched without a lease")
		}
	})
	t.Run("apply", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, Options{Workers: 1}, noFetch)
		h.st.sweepOff = true
		h.st.applyErr = errors.New("database is locked")
		h.st.addPending(6, "https://example.com/", model.LocaleEN)
		h.start(t)
		h.r.Enqueue(6)
		await(t, h.processed, "processed")
		recs := h.log.find(t, "metadata fetch result not stored")
		if len(recs) != 1 || recs[0]["level"] != "DEBUG" || recs[0]["link_id"] != float64(6) {
			t.Fatalf("apply failure log = %v", recs)
		}
	})
	t.Run("sweep", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, Options{Workers: 1}, noFetch)
		h.st.dueErr = errors.New("database is locked")
		cancel := h.start(t)
		await(t, h.st.dueCalls, "sweep")
		cancel()
		recs := h.log.find(t, "metadata sweep failed")
		if len(recs) != 1 || recs[0]["level"] != "DEBUG" {
			t.Fatalf("sweep failure log = %v", recs)
		}
	})
}

// TestSweepQueueFull: when the queue is full the sweep leaves the rest
// pending for the next run.
func TestSweepQueueFull(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	h := newHarness(t, Options{Workers: 1, QueueSize: 1}, func(ctx context.Context, rawURL, lang string) (fetch.Result, error) {
		<-release
		return fetch.Result{Title: "t"}, nil
	})
	// Link 1 occupies the worker, link 2 the only queue slot, link 3 is rejected.
	h.st.sweepOff = true
	for id := int64(1); id <= 3; id++ {
		h.st.addPending(id, fmt.Sprintf("https://example.com/%d", id), model.LocaleEN)
	}
	cancel := h.start(t)
	await(t, h.st.dueCalls, "disabled first sweep")
	h.r.Enqueue(1)
	await(t, h.f.calls, "fetch 1")
	h.st.mu.Lock()
	h.st.sweepOff = false
	h.st.mu.Unlock()
	tick(t, h.tk.get(t, DefaultSweepInterval))
	if n := await(t, h.st.dueCalls, "sweep"); n != 2 {
		t.Fatalf("due = %d, want 2 (link 1 is leased)", n)
	}
	close(release)
	await(t, h.processed, "link 1")
	await(t, h.processed, "link 2")
	cancel()
	recs := h.log.find(t, "metadata sweep")
	if len(recs) != 1 || recs[0]["due"] != float64(2) || recs[0]["queued"] != float64(1) {
		t.Fatalf("sweep log = %v", recs)
	}
	if l := h.st.link(3); l.status != model.FetchPending {
		t.Fatalf("rejected link 3 = %+v, want pending", l)
	}
}

func TestHostOf(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"https://Example.com:8443/a?b": "Example.com",
		"http://[::1]:80/":             "::1",
		"://bad":                       "",
	} {
		if got := hostOf(in); got != want {
			t.Errorf("hostOf(%q) = %q, want %q", in, got, want)
		}
	}
}
