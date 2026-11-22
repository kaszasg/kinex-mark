package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func decodeErrorBody(t *testing.T, rec *httptest.ResponseRecorder) errorBody {
	t.Helper()
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return body
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusCreated, map[string]int{"id": 7})
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q", cc)
	}
	if rec.Body.String() != "{\"id\":7}\n" {
		t.Errorf("body = %q", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	rec.Header().Set("Cache-Control", "private, max-age=604800")
	writeJSON(rec, http.StatusOK, nil)
	if cc := rec.Header().Get("Cache-Control"); cc != "private, max-age=604800" {
		t.Errorf("existing Cache-Control overwritten: %q", cc)
	}
}

func TestWriteNoContent(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writeNoContent(rec)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("got %d %q %q", rec.Code, rec.Body.String(), rec.Header().Get("Cache-Control"))
	}
	rec = httptest.NewRecorder()
	rec.Header().Set("Cache-Control", "no-cache")
	writeNoContent(rec)
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Error("existing Cache-Control overwritten")
	}
}

func TestWriteErrorShapes(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writeError(rec, newError(codeNotFound))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"error":"not found","code":"not_found"}` {
		t.Errorf("body = %s", got)
	}

	rec = httptest.NewRecorder()
	writeValidation(rec, map[string]string{"url": fieldNotAllowed})
	body := decodeErrorBody(t, rec)
	if rec.Code != http.StatusBadRequest || body.Code != codeValidationFailed || body.Error != "validation failed" ||
		body.Fields["url"] != fieldNotAllowed {
		t.Errorf("validation: %d %+v", rec.Code, body)
	}

	rec = httptest.NewRecorder()
	writeError(rec, nil)
	if rec.Code != http.StatusInternalServerError || decodeErrorBody(t, rec).Code != codeInternalError {
		t.Errorf("nil error: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	writeError(rec, newError("no_such_code"))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("unknown code status = %d", rec.Code)
	}
}

func TestTooManyAttempts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   time.Duration
		want int
	}{
		{0, 1},
		{-time.Second, 1},
		{1500 * time.Millisecond, 2},
		{15 * time.Minute, 900},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		writeError(rec, tooManyAttempts(c.in))
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("%v: status %d", c.in, rec.Code)
		}
		if got := rec.Header().Get("Retry-After"); got != fmt.Sprint(c.want) {
			t.Errorf("%v: Retry-After = %q, want %d", c.in, got, c.want)
		}
		body := decodeErrorBody(t, rec)
		if body.Code != codeTooManyAttempts || body.Details[detailRetryAfterSeconds] != float64(c.want) {
			t.Errorf("%v: body %+v", c.in, body)
		}
	}
}

func TestErrorHelpers(t *testing.T) {
	t.Parallel()
	e := currentPasswordError()
	if e.Status != http.StatusBadRequest || e.Code != codeCurrentPasswordInvalid || e.Fields["currentPassword"] != fieldIncorrect {
		t.Errorf("currentPasswordError = %+v", e)
	}
	q := invalidQueryError(map[string]string{"limit": fieldInvalid})
	if q.Status != http.StatusBadRequest || q.Code != codeInvalidQuery || q.Fields["limit"] != fieldInvalid {
		t.Errorf("invalidQueryError = %+v", q)
	}
	l := settingLocked([]string{"allowPrivateFetch"})
	if l.Status != http.StatusConflict || fmt.Sprint(l.Details[detailKeys]) != "[allowPrivateFetch]" {
		t.Errorf("settingLocked = %+v", l)
	}
	if limitReached(limitLinks).Code != codeLinkLimitReached || limitReached(limitTags).Code != codeTagLimitReached {
		t.Error("limitReached mapping")
	}

	base := newError(codeLinkDuplicate)
	withOne := base.withDetail(detailLinkID, int64(1))
	withTwo := withOne.withDetail("extra", true)
	if base.Details != nil || len(withOne.Details) != 1 || len(withTwo.Details) != 2 {
		t.Errorf("withDetail mutated its receiver: %v %v %v", base.Details, withOne.Details, withTwo.Details)
	}
	if !strings.Contains(base.Error(), codeLinkDuplicate) {
		t.Errorf("Error() = %q", base.Error())
	}
}

func TestErrorSpecsAreConsistent(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, s := range errorSpecs {
		if seen[s.code] {
			t.Errorf("duplicate code %q", s.code)
		}
		seen[s.code] = true
		if s.status < 400 || s.status > 599 || s.message == "" {
			t.Errorf("bad spec %+v", s)
		}
		if strings.ToLower(s.message) != s.message && !strings.Contains(s.message, "JSON") {
			t.Errorf("message %q should be lower case", s.message)
		}
		if e := newError(s.code); e.Code != s.code || e.Status != s.status {
			t.Errorf("newError(%q) = %+v", s.code, e)
		}
	}
	if len(errorCodes()) != len(errorSpecs) || len(errorSpecs) != 40 {
		t.Errorf("expected 40 codes, got %d", len(errorSpecs))
	}
	if newError(codeHostMismatch).Status != 421 || newError(codePayloadTooLarge).Status != 413 ||
		newError(codeNotImplemented).Status != 501 {
		t.Error("unexpected statuses")
	}
}

func TestMapModelError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		err     error
		code    string
		status  int
		details map[string]any
	}{
		{model.ErrNotFound, codeNotFound, 404, nil},
		{fmt.Errorf("get link: %w", model.ErrNotFound), codeNotFound, 404, nil},
		{model.ErrNotOwner, codeNotOwner, 403, nil},
		{model.ErrLastAdmin, codeLastAdmin, 409, nil},
		{&model.DuplicateError{Kind: model.DuplicateLink, ExistingID: 4}, codeLinkDuplicate, 409, map[string]any{detailLinkID: int64(4)}},
		{&model.DuplicateError{Kind: model.DuplicateTag, ExistingID: 5}, codeTagDuplicate, 409, map[string]any{detailTagID: int64(5)}},
		{&model.DuplicateError{Kind: model.DuplicateShare}, codeShareDuplicate, 409, nil},
		{fmt.Errorf("create: %w", &model.DuplicateError{Kind: model.DuplicateUsername}), codeUsernameTaken, 409, nil},
	}
	for _, c := range cases {
		got, ok := mapModelError(c.err)
		if !ok || got.Code != c.code || got.Status != c.status || fmt.Sprint(got.Details) != fmt.Sprint(c.details) {
			t.Errorf("mapModelError(%v) = %+v, %v", c.err, got, ok)
		}
	}
	for _, err := range []error{nil, model.ErrLimitReached, model.ErrConflict, errors.New("disk full"),
		&model.DuplicateError{Kind: "other"}} {
		if got, ok := mapModelError(err); ok {
			t.Errorf("mapModelError(%v) mapped to %+v", err, got)
		}
	}
}

func TestWriteFailure(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/links/1", nil)
	req.Pattern = "GET /api/links/{id}"

	rec := httptest.NewRecorder()
	writeFailure(rec, req, log, fmt.Errorf("wrapped: %w", newError(codeNotOwner)))
	if rec.Code != http.StatusForbidden || decodeErrorBody(t, rec).Code != codeNotOwner {
		t.Errorf("apiError: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	writeFailure(rec, req, log, model.ErrNotFound)
	if rec.Code != http.StatusNotFound {
		t.Errorf("model error: %d", rec.Code)
	}
	if logs.Len() != 0 {
		t.Errorf("expected errors are logged: %s", logs.String())
	}

	rec = httptest.NewRecorder()
	writeFailure(rec, req, log, errors.New("disk full at /secret/path"))
	body := decodeErrorBody(t, rec)
	if rec.Code != http.StatusInternalServerError || body.Code != codeInternalError || body.Error != "internal error" {
		t.Errorf("internal: %d %+v", rec.Code, body)
	}
	if strings.Contains(rec.Body.String(), "disk full") {
		t.Error("internal detail leaked to the client")
	}
	if !strings.Contains(logs.String(), "request failed") || !strings.Contains(logs.String(), "disk full") ||
		!strings.Contains(logs.String(), "GET /api/links/{id}") {
		t.Errorf("log = %s", logs.String())
	}

	rec = httptest.NewRecorder()
	writeFailure(rec, req, nil, errors.New("no logger"))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("nil logger: %d", rec.Code)
	}
}

func TestReadJSON(t *testing.T) {
	t.Parallel()
	type body struct {
		Title string `json:"title"`
	}
	cases := []struct {
		name string
		in   string
		ok   bool
	}{
		{"valid", `{"title":"a"}`, true},
		{"trailing data ignored", `{"title":"a"} xyz`, true},
		{"empty", ``, false},
		{"malformed", `{"title":`, false},
		{"unknown field", `{"title":"a","x":1}`, false},
		{"wrong type", `{"title":5}`, false},
		{"too large", `{"title":"` + strings.Repeat("a", maxBodyBytes) + `"}`, false},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/x", strings.NewReader(c.in))
		var dst body
		ok := readJSON(rec, req, &dst)
		if ok != c.ok {
			t.Errorf("%s: ok = %v", c.name, ok)
			continue
		}
		if !ok {
			if rec.Code != http.StatusBadRequest || decodeErrorBody(t, rec).Code != codeInvalidJSON {
				t.Errorf("%s: %d %s", c.name, rec.Code, rec.Body.String())
			}
		} else if dst.Title != "a" {
			t.Errorf("%s: decoded %+v", c.name, dst)
		}
	}
}

func TestListHelpers(t *testing.T) {
	t.Parallel()
	if got := nonNil[int](nil); got == nil || len(got) != 0 {
		t.Errorf("nonNil(nil) = %#v", got)
	}
	if got := nonNil([]int{1}); len(got) != 1 {
		t.Errorf("nonNil kept %v", got)
	}
	rec := httptest.NewRecorder()
	writeList[model.Tag](rec, nil)
	if strings.TrimSpace(rec.Body.String()) != `{"items":[]}` || rec.Code != http.StatusOK {
		t.Errorf("writeList(nil) = %d %s", rec.Code, rec.Body.String())
	}
}
