package jobs

import (
	"context"
	"errors"
	"net/url"

	"github.com/kaszasg/kinex-mark/internal/fetch"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Enqueue queues a link for a metadata fetch without blocking (7.3). It
// returns false when the queue is full; the link then stays pending and the
// next sweep picks it up. A link that is already queued or being processed
// is not queued twice, and Enqueue reports true for it.
func (r *Runner) Enqueue(linkID int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.inflight[linkID]; ok {
		return true
	}
	select {
	case r.queue <- linkID:
		r.inflight[linkID] = struct{}{}
		return true
	default:
		return false
	}
}

// release removes a processed link from the in-flight set.
func (r *Runner) release(linkID int64) {
	r.mu.Lock()
	delete(r.inflight, linkID)
	r.mu.Unlock()
}

// sweep queues up to model.FetchSweepBatch due links (7.3). The sweeper
// logs at Debug level only.
func (r *Runner) sweep(ctx context.Context) {
	ids, err := r.st.DueFetches(ctx, model.FetchSweepBatch)
	if err != nil {
		if ctx.Err() == nil {
			r.log.Debug("metadata sweep failed", "err", err)
		}
		return
	}
	queued := 0
	for _, id := range ids {
		if r.Enqueue(id) {
			queued++
		}
	}
	if len(ids) > 0 {
		r.log.Debug("metadata sweep", "due", len(ids), "queued", queued)
	}
}

// work is one fetch worker: it processes queued links until ctx is done.
func (r *Runner) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-r.queue:
			r.process(ctx, id)
			r.release(id)
			if r.onProcessed != nil {
				r.onProcessed(id)
			}
		}
	}
}

// process runs the three worker steps of 7.3: lease, fetch without holding
// a database connection, store the result. When ctx ends during the fetch
// nothing is written: the link stays pending and is picked up again after
// the lease expires (7.5).
//
// Logging rule (7.3, 6.12): only the link id, the error code and the host
// are logged; never the URL, its query string or the fetch error text.
func (r *Runner) process(ctx context.Context, linkID int64) {
	job, err := r.st.BeginFetch(ctx, linkID, model.FetchLease)
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) && ctx.Err() == nil {
			r.log.Debug("metadata fetch lease failed", "link_id", linkID, "err", err)
		}
		return
	}
	res, fetchErr := r.f.Fetch(ctx, job.URL, string(job.Locale))
	if ctx.Err() != nil {
		return
	}
	result := buildResult(job, res, fetchErr)
	if result.Error != "" {
		r.log.Debug("metadata fetch failed", "link_id", linkID, "code", result.Error, "host", hostOf(job.URL))
	}
	if err := r.st.ApplyFetchResult(ctx, result); err != nil && ctx.Err() == nil {
		r.log.Debug("metadata fetch result not stored", "link_id", linkID, "err", err)
	}
}

// buildResult maps a fetch outcome to the store input (7.3 table):
//   - a fetch error or a page error with a blocked_* code: FetchBlocked, final;
//   - any other error: FetchFailed, Temporary as reported (the store retries
//     with backoff while attempts remain);
//   - not_html: FetchOK with the code kept in Error and an empty title;
//   - success: FetchOK with the title and icon.
//
// An error that is not a *fetch.Error counts as a temporary connection error.
// The icon of a page error comes from the /favicon.ico fallback (6.8).
func buildResult(job model.FetchJob, res fetch.Result, fetchErr error) model.FetchResult {
	out := model.FetchResult{LinkID: job.LinkID, URL: job.URL}
	var fe *fetch.Error
	switch {
	case fetchErr != nil:
		if !errors.As(fetchErr, &fe) || fe == nil {
			fe = &fetch.Error{Code: model.FetchErrConnection, Temporary: true}
		}
	case res.PageErr != nil:
		fe = res.PageErr
		out.Icon = toIconBlob(res.Icon)
	default:
		out.Status = model.FetchOK
		out.Title = res.Title
		out.Icon = toIconBlob(res.Icon)
		return out
	}
	out.Error = fe.Code
	switch fe.Code {
	case model.FetchErrNotHTML:
		out.Status = model.FetchOK
	case model.FetchErrBlockedAddress, model.FetchErrBlockedPort:
		out.Status = model.FetchBlocked
	default:
		out.Status = model.FetchFailed
		out.Temporary = fe.Temporary
	}
	return out
}

// toIconBlob converts a fetched icon; nil stays nil.
func toIconBlob(icon *fetch.Icon) *model.IconBlob {
	if icon == nil {
		return nil
	}
	return &model.IconBlob{Data: icon.Data, ContentType: icon.ContentType, SHA256: icon.SHA256}
}

// hostOf returns the host name of a URL for logging, or "" when it does not parse.
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
