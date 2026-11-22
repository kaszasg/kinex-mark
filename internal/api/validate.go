package api

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// forbiddenRune reports whether r is rejected in text input (5.1): C0
// controls (except '\n' in multi-line fields), U+007F, C1 controls, and the
// bidi controls (Unicode Bidi_Control): the marks ALM, LRM and RLM, the
// embeddings, overrides and isolates.
func forbiddenRune(r rune, multiline bool) bool {
	switch {
	case r == '\n':
		return !multiline
	case r <= 0x1F, r == 0x7F, r >= 0x80 && r <= 0x9F:
		return true
	case r == 0x061C, r == 0x200E, r == 0x200F, r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069:
		return true
	}
	return false
}

// hasForbiddenRune reports whether s contains a forbidden rune or invalid UTF-8.
func hasForbiddenRune(s string, multiline bool) bool {
	if !utf8.ValidString(s) {
		return true
	}
	for _, r := range s {
		if forbiddenRune(r, multiline) {
			return true
		}
	}
	return false
}

// checkText trims raw and checks it against the text rules of 5.1. It
// returns the cleaned value and a field code ("" when valid). In multi-line
// fields "\r\n" is normalised to "\n" first. Order: required (empty while
// minRunes > 0), invalid (forbidden rune), too_short, too_long.
func checkText(raw string, minRunes, maxRunes int, multiline bool) (string, string) {
	s := raw
	if multiline {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}
	s = strings.TrimSpace(s)
	if s == "" {
		if minRunes > 0 {
			return s, fieldRequired
		}
		return s, ""
	}
	if hasForbiddenRune(s, multiline) {
		return s, fieldInvalid
	}
	n := utf8.RuneCountInString(s)
	if n < minRunes {
		return s, fieldTooShort
	}
	if n > maxRunes {
		return s, fieldTooLong
	}
	return s, ""
}

// checkUsername validates a username (5.12): required, 3-64 characters,
// [A-Za-z0-9._-] only.
func checkUsername(raw string) (string, string) {
	s := strings.TrimSpace(raw)
	switch {
	case s == "":
		return s, fieldRequired
	case utf8.RuneCountInString(s) < model.UsernameMinLen:
		return s, fieldTooShort
	case utf8.RuneCountInString(s) > model.UsernameMaxLen:
		return s, fieldTooLong
	case !model.ValidUsername(s):
		return s, fieldInvalidFormat
	}
	return s, ""
}

// isEmailShaped reports whether s looks like an e-mail address: one "@",
// a non-empty local part of at most 64 bytes, a dotted domain, no spaces or
// forbidden runes, at most 254 bytes. Deliverability is not checked.
func isEmailShaped(s string) bool {
	if len(s) > model.PrivacyControllerEmailMaxLen || hasForbiddenRune(s, false) || strings.ContainsAny(s, " \t") {
		return false
	}
	at := strings.IndexByte(s, '@')
	if at <= 0 || at != strings.LastIndexByte(s, '@') || at > 64 {
		return false
	}
	domain := s[at+1:]
	dot := strings.LastIndexByte(domain, '.')
	return dot > 0 && dot < len(domain)-1 && !strings.HasPrefix(domain, ".") && !strings.Contains(domain, "..")
}

// checkIDList validates a list of ids from a request body: at most max
// entries (too_long), all positive (invalid), no repeats (duplicate).
func checkIDList(ids []int64, max int) string {
	if len(ids) > max {
		return fieldTooLong
	}
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id < 1 {
			return fieldInvalid
		}
		if seen[id] {
			return fieldDuplicate
		}
		seen[id] = true
	}
	return ""
}

// fieldErrors collects field codes; the first code of a field wins.
type fieldErrors map[string]string

// add records code for field unless code is empty or the field already failed.
func (f fieldErrors) add(field, code string) {
	if code == "" {
		return
	}
	if _, ok := f[field]; !ok {
		f[field] = code
	}
}

// text checks a text field with checkText and records its code.
func (f fieldErrors) text(field, raw string, minRunes, maxRunes int, multiline bool) string {
	s, code := checkText(raw, minRunes, maxRunes, multiline)
	f.add(field, code)
	return s
}

// intRange records out_of_range when v is outside r.
func (f fieldErrors) intRange(field string, v int, r model.IntRange) {
	if !r.Contains(v) {
		f.add(field, fieldOutOfRange)
	}
}

// oneOf records invalid when v is not one of allowed.
func (f fieldErrors) oneOf(field, v string, allowed []string) {
	for _, a := range allowed {
		if v == a {
			return
		}
	}
	f.add(field, fieldInvalid)
}

// empty reports whether no field failed.
func (f fieldErrors) empty() bool { return len(f) == 0 }

// writeIfAny writes 400 validation_failed when a field failed and reports
// whether it did.
func (f fieldErrors) writeIfAny(w http.ResponseWriter) bool {
	if f.empty() {
		return false
	}
	writeValidation(w, f)
	return true
}

// parseID parses a path id: a base-10 int64 of at least 1.
func parseID(s string) (int64, bool) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// pathID reads the path value name as an id; on failure it writes 400
// invalid_id and returns false (5.1).
func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, ok := parseID(r.PathValue(name))
	if !ok {
		writeError(w, newError(codeInvalidID))
		return 0, false
	}
	return id, true
}

// queryReader parses query parameters and collects the names of invalid ones.
type queryReader struct {
	values url.Values
	errs   fieldErrors
}

// newQuery returns a reader for the request's query string.
func newQuery(r *http.Request) *queryReader {
	return &queryReader{values: r.URL.Query(), errs: fieldErrors{}}
}

// raw returns the first value of name and whether the parameter is present
// with a non-empty value.
func (q *queryReader) raw(name string) (string, bool) {
	v := q.values.Get(name)
	return v, v != ""
}

// invalid marks a parameter invalid.
func (q *queryReader) invalid(name string) { q.errs.add(name, fieldInvalid) }

// has reports whether the parameter is present with a non-empty value.
func (q *queryReader) has(name string) bool {
	_, ok := q.raw(name)
	return ok
}

// text returns the trimmed value (at most maxRunes, no forbidden runes).
func (q *queryReader) text(name string, maxRunes int) string {
	v, ok := q.raw(name)
	if !ok {
		return ""
	}
	s, code := checkText(v, 0, maxRunes, false)
	if code != "" {
		q.invalid(name)
		return ""
	}
	return s
}

// enum returns the value when it is one of allowed, def when absent.
func (q *queryReader) enum(name, def string, allowed []string) string {
	v, ok := q.raw(name)
	if !ok {
		return def
	}
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	q.invalid(name)
	return def
}

// boolean returns the value of a "true"/"false" parameter, nil when absent.
func (q *queryReader) boolean(name string) *bool {
	v, ok := q.raw(name)
	if !ok {
		return nil
	}
	switch v {
	case "true":
		b := true
		return &b
	case "false":
		b := false
		return &b
	}
	q.invalid(name)
	return nil
}

// id returns a positive id parameter, nil when absent.
func (q *queryReader) id(name string) *int64 {
	v, ok := q.raw(name)
	if !ok {
		return nil
	}
	id, valid := parseID(v)
	if !valid {
		q.invalid(name)
		return nil
	}
	return &id
}

// intIn returns an integer parameter within [min, max], def when absent.
func (q *queryReader) intIn(name string, def, min, max int) int {
	v, ok := q.raw(name)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		q.invalid(name)
		return def
	}
	return n
}

// ids returns a comma-separated list of distinct positive ids, at most max
// entries; empty items are ignored, nil when absent.
func (q *queryReader) ids(name string, max int) []int64 {
	v, ok := q.raw(name)
	if !ok {
		return nil
	}
	var out []int64
	seen := map[int64]bool{}
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, valid := parseID(part)
		if !valid {
			q.invalid(name)
			return nil
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	if len(out) > max {
		q.invalid(name)
		return nil
	}
	return out
}

// time returns an RFC 3339 timestamp parameter in UTC, nil when absent.
func (q *queryReader) time(name string) *time.Time {
	v, ok := q.raw(name)
	if !ok {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		q.invalid(name)
		return nil
	}
	t = t.UTC()
	return &t
}

// page returns limit and offset (5.4): limit 1..maxLimit (default
// defLimit), offset 0..model.MaxOffset (default 0).
func (q *queryReader) page(defLimit, maxLimit int) model.PageReq {
	return model.PageReq{
		Limit:  q.intIn("limit", defLimit, 1, maxLimit),
		Offset: q.intIn("offset", 0, 0, model.MaxOffset),
	}
}

// writeIfInvalid writes 400 invalid_query when a parameter failed and
// reports whether it did.
func (q *queryReader) writeIfInvalid(w http.ResponseWriter) bool {
	if q.errs.empty() {
		return false
	}
	writeError(w, invalidQueryError(q.errs))
	return true
}
