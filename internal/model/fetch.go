package model

import "time"

// Metadata fetch error codes (links.fetch_error, 7.3).
const (
	FetchErrTimeout          = "timeout"
	FetchErrDNS              = "dns"
	FetchErrConnection       = "connection"
	FetchErrTLS              = "tls"
	FetchErrHTTPStatus       = "http_status"
	FetchErrTooLarge         = "too_large"
	FetchErrNotHTML          = "not_html"
	FetchErrTooManyRedirects = "too_many_redirects"
	FetchErrBlockedAddress   = "blocked_address"
	FetchErrBlockedPort      = "blocked_port"
)

// FetchErrors lists the fetch error codes in the order of the 7.3 table.
var FetchErrors = []string{
	FetchErrTimeout, FetchErrDNS, FetchErrConnection, FetchErrTLS, FetchErrHTTPStatus,
	FetchErrTooLarge, FetchErrNotHTML, FetchErrTooManyRedirects, FetchErrBlockedAddress,
	FetchErrBlockedPort,
}

// Metadata fetch scheduling (7.3).
const (
	FetchLease       = 2 * time.Minute
	FetchMaxAttempts = 3
	FetchSweepBatch  = 50
)

// FetchBackoff is the retry delay after the n-th transient failure (n >= 1).
func FetchBackoff(attempts int) time.Duration {
	if attempts <= 1 {
		return time.Minute
	}
	return 10 * time.Minute
}

// FetchJob is a leased metadata fetch (Store.BeginFetch).
type FetchJob struct {
	LinkID  int64
	URL     string
	OwnerID int64
	Locale  Locale // the owner's locale, for Accept-Language
}

// FetchResult is the outcome of one fetch attempt (Store.ApplyFetchResult).
// The store applies it only while the link still has URL (7.3).
type FetchResult struct {
	LinkID int64
	URL    string // the URL that was fetched (FetchJob.URL)
	// Status is the final state: FetchOK, FetchFailed or FetchBlocked. With
	// Temporary set the store retries with backoff and writes no title or icon
	// until the attempts run out.
	Status    FetchStatus
	Error     string // fetch error code, "" when none (not_html keeps Status FetchOK)
	Temporary bool
	Title     string    // cleaned page title, may be ""
	Icon      *IconBlob // nil when no valid icon was found
}
