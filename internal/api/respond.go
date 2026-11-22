package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

// maxBodyBytes is the JSON request body limit (5.1, TECHNOLOGY.md 4.4).
const maxBodyBytes = 64 << 10

// errorBody is the JSON error shape (5.2).
type errorBody struct {
	Error   string            `json:"error"`
	Code    string            `json:"code"`
	Fields  map[string]string `json:"fields,omitempty"`
	Details map[string]any    `json:"details,omitempty"`
}

// writeJSON writes v as JSON with a trailing newline. Every /api/ response
// is "Cache-Control: no-store" unless the handler already set a policy
// (the icon endpoint, 5.9.9).
func writeJSON(w http.ResponseWriter, status int, v any) {
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	if h.Get("Cache-Control") == "" {
		h.Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeNoContent writes an empty 204 response.
func writeNoContent(w http.ResponseWriter) {
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeError writes an error response; for 429 it also sets Retry-After.
func writeError(w http.ResponseWriter, e *apiError) {
	if e == nil {
		e = newError(codeInternalError)
	}
	if e.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(e.RetryAfter))
	}
	writeJSON(w, e.Status, errorBody{
		Error:   e.Message,
		Code:    e.Code,
		Fields:  e.Fields,
		Details: e.Details,
	})
}

// writeValidation writes 400 validation_failed with the field codes.
func writeValidation(w http.ResponseWriter, fields map[string]string) {
	writeError(w, validationError(fields))
}

// writeFailure writes the response for an error returned by a dependency:
// an *apiError as is, a mapped model error (mapModelError), and anything
// else as 500 internal_error with the detail logged only (5.2).
func writeFailure(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	var ae *apiError
	if errors.As(err, &ae) {
		writeError(w, ae)
		return
	}
	if mapped, ok := mapModelError(err); ok {
		writeError(w, mapped)
		return
	}
	if log != nil {
		log.Error("request failed", "err", err, "route", r.Pattern)
	}
	writeError(w, newError(codeInternalError))
}

// readJSON decodes the request body into dst: at most maxBodyBytes, unknown
// fields rejected, a single Decode call. Any failure (malformed, empty, too
// large, unknown field) writes 400 invalid_json and returns false.
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, newError(codeInvalidJSON))
		return false
	}
	return true
}

// nonNil returns s, or an empty slice when s is nil, so that lists encode
// as [] and never as null (5.1).
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// listResponse is the unpaginated list shape {"items": [...]} (5.4).
type listResponse[T any] struct {
	Items []T `json:"items"`
}

// writeList writes {"items": [...]} with status 200.
func writeList[T any](w http.ResponseWriter, items []T) {
	writeJSON(w, http.StatusOK, listResponse[T]{Items: nonNil(items)})
}
