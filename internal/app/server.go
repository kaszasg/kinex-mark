package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/kaszasg/kinex-mark/internal/config"
)

// HTTP server limits (TECHNOLOGY.md 4.1, 3.3.11 step 9).
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 2 * time.Minute
	maxHeaderBytes    = 64 << 10
	shutdownTimeout   = 10 * time.Second
)

// newHTTPServer returns the server with the fixed limits. The server's own
// error messages (for example failed TLS handshakes) go to the application
// log at Warn level, so every line shares one format.
func newHTTPServer(addr string, handler http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}

// listen prepares the server for handler: it loads the TLS certificate when
// one is configured (TLS 1.2 or newer) and opens the listener on cfg.Addr.
// Opening the listener first lets the "server started" line carry the real
// address (also for port 0) and keeps the background jobs from starting when
// the port is busy.
func listen(cfg config.Config, handler http.Handler, log *slog.Logger) (*http.Server, net.Listener, error) {
	srv := newHTTPServer(cfg.Addr, handler, log)
	if cfg.TLSCertFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return nil, nil, fmt.Errorf("load tls certificate: %w", err)
		}
		srv.TLSConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
	}
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, nil, fmt.Errorf("listen: %w", err)
	}
	return srv, ln, nil
}

// serve serves on ln until the server fails or ctx ends, then shuts the
// server down gracefully within shutdownTimeout (TECHNOLOGY.md 4.1, 7.5
// steps 1-2). http.ErrServerClosed is not an error.
func serve(ctx context.Context, srv *http.Server, ln net.Listener, log *slog.Logger) error {
	errs := make(chan error, 1)
	go func() {
		if srv.TLSConfig != nil {
			errs <- srv.ServeTLS(ln, "", "")
			return
		}
		errs <- srv.Serve(ln)
	}()

	select {
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
