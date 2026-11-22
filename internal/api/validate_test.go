package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestCheckText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		in        string
		min, max  int
		multiline bool
		want      string
		code      string
	}{
		{"trimmed", "  hello  ", 1, 10, false, "hello", ""},
		{"required", "   ", 1, 10, false, "", fieldRequired},
		{"optional empty", "", 0, 10, false, "", ""},
		{"too short", "ab", 3, 10, false, "ab", fieldTooShort},
		{"too long", "abcdef", 1, 5, false, "abcdef", fieldTooLong},
		{"rune count, not bytes", "árvíz", 1, 5, false, "árvíz", ""},
		{"newline rejected in single line", "a\nb", 0, 10, false, "a\nb", fieldInvalid},
		{"newline allowed in multi line", "a\nb", 0, 10, true, "a\nb", ""},
		{"crlf normalised", "a\r\nb", 0, 10, true, "a\nb", ""},
		{"lone cr rejected", "a\rb", 0, 10, true, "a\rb", fieldInvalid},
		{"tab rejected", "a\tb", 0, 10, true, "a\tb", fieldInvalid},
		{"nul rejected", "a\x00b", 0, 10, false, "a\x00b", fieldInvalid},
		{"del rejected", "a\x7fb", 0, 10, false, "a\x7fb", fieldInvalid},
		{"c1 rejected", "a\u0085b", 0, 10, false, "a\u0085b", fieldInvalid},
		{"lrm rejected", "a‎b", 0, 10, false, "a‎b", fieldInvalid},
		{"rlm rejected", "a‏b", 0, 10, false, "a‏b", fieldInvalid},
		{"rlo rejected", "a‮b", 0, 10, false, "a‮b", fieldInvalid},
		{"lre rejected", "a‪b", 0, 10, false, "a‪b", fieldInvalid},
		{"isolate rejected", "a⁦b", 0, 10, false, "a⁦b", fieldInvalid},
		{"pdi rejected", "a⁩b", 0, 10, false, "a⁩b", fieldInvalid},
		{"alm rejected", "a؜b", 0, 10, false, "a؜b", fieldInvalid},
		{"invalid utf8", "a\xffb", 0, 10, false, "a\xffb", fieldInvalid},
		{"zwj allowed", "a‍b", 0, 10, false, "a‍b", ""},
	}
	for _, c := range cases {
		got, code := checkText(c.in, c.min, c.max, c.multiline)
		if got != c.want || code != c.code {
			t.Errorf("%s: checkText(%q) = %q, %q; want %q, %q", c.name, c.in, got, code, c.want, c.code)
		}
	}
}

func TestCheckUsername(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"admin":                 "",
		" Dev.User_1-x ":        "",
		"":                      fieldRequired,
		"ab":                    fieldTooShort,
		strings.Repeat("a", 65): fieldTooLong,
		"bad name":              fieldInvalidFormat,
		"a/b/c":                 fieldInvalidFormat,
		"álmos":                 fieldInvalidFormat,
	}
	for in, want := range cases {
		if _, code := checkUsername(in); code != want {
			t.Errorf("checkUsername(%q) = %q, want %q", in, code, want)
		}
	}
	if got, _ := checkUsername("  admin "); got != "admin" {
		t.Errorf("not trimmed: %q", got)
	}
}

func TestIsEmailShaped(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"dpo@example.com":                        true,
		"first.last+tag@sub.example.co.uk":       true,
		"":                                       false,
		"no-at.example.com":                      false,
		"@example.com":                           false,
		"a@b@example.com":                        false,
		"a@example":                              false,
		"a@.example.com":                         false,
		"a@example.com.":                         false,
		"a@exa..mple.com":                        false,
		"a b@example.com":                        false,
		"a@exa‮mple.com":                         false,
		strings.Repeat("a", 65) + "@x.com":       false,
		"a@" + strings.Repeat("b", 250) + ".com": false,
	}
	for in, want := range cases {
		if got := isEmailShaped(in); got != want {
			t.Errorf("isEmailShaped(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCheckIDList(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ids  []int64
		want string
	}{
		{nil, ""},
		{[]int64{1, 2, 3}, ""},
		{[]int64{1, 0}, fieldInvalid},
		{[]int64{-4}, fieldInvalid},
		{[]int64{2, 2}, fieldDuplicate},
		{[]int64{1, 2, 3, 4}, fieldTooLong},
	}
	for _, c := range cases {
		if got := checkIDList(c.ids, 3); got != c.want {
			t.Errorf("checkIDList(%v) = %q, want %q", c.ids, got, c.want)
		}
	}
}

func TestFieldErrors(t *testing.T) {
	t.Parallel()
	f := fieldErrors{}
	if !f.empty() {
		t.Fatal("new fieldErrors not empty")
	}
	f.add("a", "")
	if !f.empty() {
		t.Error("empty code recorded")
	}
	f.add("a", fieldRequired)
	f.add("a", fieldTooLong)
	if f["a"] != fieldRequired {
		t.Errorf("first code must win: %v", f)
	}
	if got := f.text("title", "  x  ", 0, 10, false); got != "x" || f["title"] != "" {
		t.Errorf("text ok: %q %v", got, f)
	}
	f.text("desc", strings.Repeat("y", 11), 0, 10, true)
	f.intRange("days", 3651, model.DefaultArchiveDaysRange)
	f.intRange("ok", 90, model.DefaultArchiveDaysRange)
	f.oneOf("sort", "random", model.SortOrders)
	f.oneOf("view", model.ViewTiles, model.ViewModes)
	want := map[string]string{"a": fieldRequired, "desc": fieldTooLong, "days": fieldOutOfRange, "sort": fieldInvalid}
	if len(f) != len(want) {
		t.Errorf("fields = %v, want %v", f, want)
	}
	for k, v := range want {
		if f[k] != v {
			t.Errorf("field %s = %q, want %q", k, f[k], v)
		}
	}

	rec := httptest.NewRecorder()
	if !f.writeIfAny(rec) || rec.Code != http.StatusBadRequest || decodeErrorBody(t, rec).Fields["days"] != fieldOutOfRange {
		t.Errorf("writeIfAny: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	if (fieldErrors{}).writeIfAny(rec) || rec.Body.Len() != 0 {
		t.Error("writeIfAny wrote for no errors")
	}
}

func TestPathID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	var got int64
	mux.HandleFunc("GET /x/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, "id")
		if ok {
			got = id
			w.WriteHeader(http.StatusOK)
		}
	})
	cases := map[string]int{
		"/x/42":                   http.StatusOK,
		"/x/0":                    http.StatusBadRequest,
		"/x/-1":                   http.StatusBadRequest,
		"/x/abc":                  http.StatusBadRequest,
		"/x/1.5":                  http.StatusBadRequest,
		"/x/99999999999999999999": http.StatusBadRequest,
	}
	for path, want := range cases {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("%s: status %d, want %d", path, rec.Code, want)
		}
		if want == http.StatusBadRequest && decodeErrorBody(t, rec).Code != codeInvalidID {
			t.Errorf("%s: body %s", path, rec.Body.String())
		}
	}
	if got != 42 {
		t.Errorf("parsed id = %d", got)
	}
}

func newQueryFor(rawQuery string) *queryReader {
	return newQuery(httptest.NewRequest(http.MethodGet, "/api/x?"+rawQuery, nil))
}

func TestQueryReaderValidValues(t *testing.T) {
	t.Parallel()
	q := newQueryFor("q=%20Hello%20&sort=title&untagged=true&archived=false&actorId=5" +
		"&tags=3,%205,,3&from=2026-01-05T10:00:00%2B01:00&limit=20&offset=40")
	if got := q.text("q", 200); got != "Hello" {
		t.Errorf("text = %q", got)
	}
	if got := q.enum("sort", model.SortUsage, model.SortOrders); got != model.SortTitle {
		t.Errorf("enum = %q", got)
	}
	if got := q.enum("tagMatch", model.TagMatchAny, model.TagMatchModes); got != model.TagMatchAny {
		t.Errorf("enum default = %q", got)
	}
	if b := q.boolean("untagged"); b == nil || !*b {
		t.Errorf("boolean true = %v", b)
	}
	if b := q.boolean("archived"); b == nil || *b {
		t.Errorf("boolean false = %v", b)
	}
	if b := q.boolean("missing"); b != nil {
		t.Errorf("boolean missing = %v", b)
	}
	if id := q.id("actorId"); id == nil || *id != 5 {
		t.Errorf("id = %v", id)
	}
	if id := q.id("missing"); id != nil {
		t.Errorf("missing id = %v", id)
	}
	if ids := q.ids("tags", 20); len(ids) != 2 || ids[0] != 3 || ids[1] != 5 {
		t.Errorf("ids = %v", ids)
	}
	if ids := q.ids("missing", 20); ids != nil {
		t.Errorf("missing ids = %v", ids)
	}
	from := q.time("from")
	if from == nil || !from.Equal(time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)) || from.Location() != time.UTC {
		t.Errorf("time = %v", from)
	}
	if q.time("to") != nil {
		t.Error("missing time not nil")
	}
	if p := q.page(200, 500); p.Limit != 20 || p.Offset != 40 {
		t.Errorf("page = %+v", p)
	}
	if !q.has("q") || q.has("missing") || q.text("missing", 5) != "" {
		t.Error("has/text on missing parameter")
	}
	rec := httptest.NewRecorder()
	if q.writeIfInvalid(rec) {
		t.Errorf("unexpected invalid params: %v", q.errs)
	}
}

func TestQueryReaderDefaults(t *testing.T) {
	t.Parallel()
	q := newQueryFor("")
	if p := q.page(50, 200); p.Limit != 50 || p.Offset != 0 {
		t.Errorf("default page = %+v", p)
	}
	if !q.errs.empty() {
		t.Errorf("errors = %v", q.errs)
	}
}

func TestQueryReaderInvalidValues(t *testing.T) {
	t.Parallel()
	q := newQueryFor("q=" + strings.Repeat("a", 201) + "&sort=random&untagged=yes&actorId=0" +
		"&tags=1,x&from=yesterday&limit=0&offset=1000001&scope=")
	q.text("q", 200)
	q.enum("sort", model.SortUsage, model.SortOrders)
	q.enum("scope", model.ScopeAll, model.LinkScopes) // empty value counts as absent
	q.boolean("untagged")
	q.id("actorId")
	q.ids("tags", 20)
	q.time("from")
	q.page(200, 500)
	want := []string{"q", "sort", "untagged", "actorId", "tags", "from", "limit", "offset"}
	if len(q.errs) != len(want) {
		t.Errorf("errs = %v, want keys %v", q.errs, want)
	}
	for _, k := range want {
		if q.errs[k] != fieldInvalid {
			t.Errorf("param %s = %q", k, q.errs[k])
		}
	}
	rec := httptest.NewRecorder()
	if !q.writeIfInvalid(rec) {
		t.Fatal("writeIfInvalid reported no error")
	}
	body := decodeErrorBody(t, rec)
	if rec.Code != http.StatusBadRequest || body.Code != codeInvalidQuery || body.Fields["limit"] != fieldInvalid {
		t.Errorf("response %d %+v", rec.Code, body)
	}

	tooMany := newQueryFor("tags=1,2,3")
	if ids := tooMany.ids("tags", 2); ids != nil || tooMany.errs["tags"] != fieldInvalid {
		t.Errorf("too many ids accepted: %v", ids)
	}
	ctrl := newQueryFor("q=a%0Ab")
	if ctrl.text("q", 10); ctrl.errs["q"] != fieldInvalid {
		t.Error("control character accepted in a query parameter")
	}
}
