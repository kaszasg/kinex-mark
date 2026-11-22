package api

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// maxRequestBytes bounds every request body. It equals the import limit
// (5.9.11), the largest body any endpoint accepts; JSON endpoints apply the
// smaller maxBodyBytes in readJSON on top of it.
const maxRequestBytes = model.MaxImportBytes

// statusWriter records the status code written through it.
type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader records the first status code.
func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

// Write records an implicit 200.
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap gives http.ResponseController access to the underlying writer.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// code returns the recorded status (200 when nothing was written).
func (w *statusWriter) code() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

// recoverer turns a handler panic into 500 internal_error and logs the
// panic value with the stack (6.1). http.ErrAbortHandler is passed on.
func (s *server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			s.log.Error("panic", "err", v, "stack", string(debug.Stack()))
			if sw.status == 0 {
				writeError(sw, newError(codeInternalError))
			}
		}()
		next.ServeHTTP(sw, r)
	})
}

// accessLog writes one Debug line per request: method, the matched route
// pattern, status and duration (6.1). The path and query string are not
// logged, because the query may hold personal data.
func (s *server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.log.Enabled(r.Context(), slog.LevelDebug) {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		s.log.Debug("request",
			"method", r.Method,
			"route", r.Pattern,
			"status", sw.code(),
			"duration", time.Since(start))
	})
}

// limitBody caps every request body at maxRequestBytes.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		}
		next.ServeHTTP(w, r)
	})
}
