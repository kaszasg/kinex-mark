// Package app wires the packages together and owns the process lifecycle:
// data directory, key ring, store, first admin, background jobs, HTTP
// server, health check and logger (3.3.11). It is the only package that
// imports web.
package app

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kaszasg/kinex-mark/internal/api"
	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/fetch"
	"github.com/kaszasg/kinex-mark/internal/jobs"
	"github.com/kaszasg/kinex-mark/internal/model"
	"github.com/kaszasg/kinex-mark/internal/store"
	"github.com/kaszasg/kinex-mark/web"
)

// The api and jobs packages reach the database and each other through their
// own interfaces; these assertions prove the wiring at compile time (3.3).
var (
	_ api.Store    = (*store.Store)(nil)
	_ jobs.Store   = (*store.Store)(nil)
	_ api.Jobs     = (*jobs.Runner)(nil)
	_ jobs.Fetcher = (*fetch.Fetcher)(nil)
)

const (
	// jobsWaitTimeout bounds the wait for the background goroutines (7.5).
	jobsWaitTimeout = 5 * time.Second
	// policyTimeout bounds the settings read behind the fetch policy.
	policyTimeout = 5 * time.Second
	// drainTimeout bounds the wait for the database connection at shutdown.
	drainTimeout = 5 * time.Second
	// hasherConcurrency bounds parallel PBKDF2 derivations.
	hasherConcurrency = 4
)

// hasherIterations is the PBKDF2 cost of new password hashes. Tests lower it.
var hasherIterations = auth.DefaultIterations

// errKeyMismatch stops the start when the key ring does not belong to the
// database (3.3.11 step 5, 6.12).
var errKeyMismatch = errors.New("secret key does not match this database: restore the original key file or APP_SECRET_KEY")

// Run starts the server and blocks until ctx ends, a termination signal
// arrives or the server fails. A clean shutdown returns nil.
func Run(ctx context.Context, cfg config.Config, version string, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, kr, err := openData(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer closeStore(st, log)

	hasher := auth.NewHasher(hasherIterations, hasherConcurrency)
	if err := bootstrapAdmin(ctx, st, hasher, cfg, log); err != nil {
		return err
	}

	fetcher := fetch.New(fetch.Options{
		Timeout:   cfg.FetchTimeout,
		UserAgent: userAgent(version),
		Policy:    fetchPolicy(st, cfg, log),
	})
	// Enqueue works before Start, so the handlers can hold the runner while
	// the jobs start only once the listener is open.
	runner := jobs.New(st, fetcher, clock.Real{}, jobs.Options{Workers: cfg.FetchWorkers, Log: log})

	ui, err := web.Files()
	if err != nil {
		return fmt.Errorf("open embedded ui: %w", err)
	}
	handler, err := api.New(api.Deps{
		Store:   st,
		Jobs:    runner,
		Hasher:  hasher,
		Keyring: kr,
		Clock:   clock.Real{},
		Log:     log,
		Config:  cfg,
		Version: version,
		UI:      ui,
	})
	if err != nil {
		return fmt.Errorf("build http handler: %w", err)
	}
	srv, ln, err := listen(cfg, handler, log)
	if err != nil {
		return err
	}

	// The jobs outlive the signal context: they stop only after the HTTP
	// server has shut down, and the store closes after them (7.5 steps 3-5).
	jobsCtx, stopJobs := context.WithCancel(context.WithoutCancel(ctx))
	runner.Start(jobsCtx)
	defer func() {
		stopJobs()
		if !runner.Wait(jobsWaitTimeout) {
			log.Warn("background jobs did not stop in time", "timeout", jobsWaitTimeout)
		}
	}()

	log.Info("server started", "addr", ln.Addr().String(), "data", cfg.DataDir, "version", version, "tls", srv.TLSConfig != nil)
	return serve(ctx, srv, ln, log)
}

// openData runs the start steps shared by Run and RunAdminCommand (3.3.11
// steps 2-5): data directory, key ring, store and key check. On success the
// caller owns the store and must close it.
func openData(ctx context.Context, cfg config.Config, log *slog.Logger) (*store.Store, *auth.Keyring, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o750); err != nil {
		return nil, nil, fmt.Errorf("create data dir: %w", err)
	}
	keyFile := cfg.SecretKeyFile
	if keyFile == "" {
		keyFile = filepath.Join(cfg.DataDir, config.SecretKeyFileName)
	}
	kr, created, err := auth.LoadOrCreateKeyring(cfg.SecretKey, keyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("load secret key: %w", err)
	}
	if created {
		log.Info("secret key created", "path", keyFile)
	}
	st, err := store.Open(ctx, cfg.DataDir, store.Options{Clock: clock.Real{}})
	if err != nil {
		return nil, nil, fmt.Errorf("open store: %w", err)
	}
	if err := verifyKey(ctx, st, kr); err != nil {
		st.Close()
		if created && errors.Is(err, errKeyMismatch) {
			// The key was generated a moment ago for a database that already
			// has one: remove it, so restoring the original is not blocked.
			if rmErr := os.Remove(keyFile); rmErr == nil {
				log.Info("secret key removed", "path", keyFile)
			}
		}
		return nil, nil, err
	}
	return st, kr, nil
}

// closingStore is the part of the store that closeStore uses.
type closingStore interface {
	Ping(ctx context.Context) error
	io.Closer
}

// closeStore closes the store as the last shutdown step (7.5 step 5).
//
// A job cancelled in the middle of a transaction returns its connection
// asynchronously (database/sql rolls the transaction back in its own
// goroutine), and DB.Close does not wait for connections in use. The store
// has a single connection, so a Ping first waits until that connection is
// free; only then does Close release the database file.
func closeStore(st closingStore, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	if err := st.Ping(ctx); err != nil {
		log.Warn("database still busy at shutdown", "err", err)
	}
	if err := st.Close(); err != nil {
		log.Warn("close store failed", "err", err)
	}
}

// keyCheckStore is the part of the store that verifyKey uses.
type keyCheckStore interface {
	KeyCheck(ctx context.Context) (string, error)
	SetKeyCheck(ctx context.Context, v string) error
}

// verifyKey stores the key check value on first use and afterwards refuses a
// key ring that does not belong to the database.
func verifyKey(ctx context.Context, st keyCheckStore, kr *auth.Keyring) error {
	stored, err := st.KeyCheck(ctx)
	if err != nil {
		return fmt.Errorf("read key check: %w", err)
	}
	want := kr.CheckValue()
	if stored == "" {
		if err := st.SetKeyCheck(ctx, want); err != nil {
			return fmt.Errorf("store key check: %w", err)
		}
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(want)) != 1 {
		return errKeyMismatch
	}
	return nil
}

// userAgent is the User-Agent header of the metadata fetcher (6.8).
func userAgent(version string) string {
	return "KinexMark/" + version + " (+metadata fetcher)"
}

// settingsReader is the part of the store that the fetch policy reads.
type settingsReader interface {
	Settings(ctx context.Context) (model.SystemSettings, error)
}

// fetchPolicy returns the fetcher's policy function, read again on every
// dial (6.8). APP_ALLOW_PRIVATE_FETCH, when set, overrides the system
// setting; loopback access comes from APP_ALLOW_LOOPBACK_FETCH only. When the
// setting cannot be read, private addresses stay blocked.
func fetchPolicy(st settingsReader, cfg config.Config, log *slog.Logger) func() fetch.Policy {
	return func() fetch.Policy {
		p := fetch.Policy{AllowLoopback: cfg.AllowLoopbackFetch}
		if cfg.AllowPrivateFetch != nil {
			p.AllowPrivate = *cfg.AllowPrivateFetch
			return p
		}
		ctx, cancel := context.WithTimeout(context.Background(), policyTimeout)
		defer cancel()
		s, err := st.Settings(ctx)
		if err != nil {
			log.Warn("read fetch policy failed, private addresses stay blocked", "err", err)
			return p
		}
		p.AllowPrivate = s.AllowPrivateFetch
		return p
	}
}
