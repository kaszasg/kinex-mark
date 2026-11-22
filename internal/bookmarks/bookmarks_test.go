package bookmarks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func ts(sec int64) *time.Time {
	t := time.Unix(sec, 0).UTC()
	return &t
}

// A browser export: folders (nested), a toolbar folder, TAGS, ADD_DATE,
// <DD> descriptions for entries and for a folder, upper-case markup.
const browserExport = `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<!-- This is an automatically generated file. -->
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks Menu</H1>
<DL><p>
    <DT><A HREF="https://top.example/" ADD_DATE="1700000000" LAST_MODIFIED="1700000001">Top &amp; level</A>
    <DD>Top description
with two lines
    <DT><H3 ADD_DATE="1600000000" PERSONAL_TOOLBAR_FOLDER="true">Work</H3>
    <DD>Folder description that belongs to no link
    <DL><p>
        <DT><A HREF="https://work.example/a" TAGS="docs, Go ,,docs">Work A</A>
        <DT><H3>Deep</H3>
        <DL><p>
            <DT><A HREF="https://deep.example/" ADD_DATE="bogus" TAGS="Deep">Deep link</A>
            <DD>Deep &lt;description&gt;
        </DL><p>
        <DT><A HREF="https://work.example/b" ADD_DATE="99999999999999">Work B <b>bold</b></A>
    </DL><p>
    <DT><A HREF=" https://after.example/ ">After folders</A>
    <DT><A NAME="anchor-without-href">Not a link</A>
    <DT><A HREF="">Empty URL</A>
</DL><p>
`

func TestParseNetscapeBrowserExport(t *testing.T) {
	t.Parallel()
	got, err := ParseNetscape(strings.NewReader(browserExport), 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{
		{URL: "https://top.example/", Title: "Top & level", Description: "Top description\nwith two lines", AddedAt: ts(1700000000)},
		{URL: "https://work.example/a", Title: "Work A", Tags: []string{"docs", "Go", "Work"}},
		{URL: "https://deep.example/", Title: "Deep link", Description: "Deep <description>", Tags: []string{"Deep"}},
		{URL: "https://work.example/b", Title: "Work B bold", Tags: []string{"Work"}},
		{URL: "https://after.example/", Title: "After folders"},
		{URL: "", Title: "Empty URL"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseNetscape =\n%+v\nwant\n%+v", got, want)
	}
}

func TestParseNetscapeLenientMarkup(t *testing.T) {
	t.Parallel()
	// No doctype, unclosed elements, lower-case tags: still a bookmark list.
	in := `<dl><dt><h3>Folder</h3><dl><dt><a href="https://a.example/" tags="x">A<dt><a href="https://b.example/">B`
	got, err := ParseNetscape(strings.NewReader(in), 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{
		{URL: "https://a.example/", Title: "A", Tags: []string{"x", "Folder"}},
		{URL: "https://b.example/", Title: "B", Tags: []string{"Folder"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	// An empty export with the doctype is valid and has no entries.
	got, err = ParseNetscape(strings.NewReader("<!DOCTYPE NETSCAPE-Bookmark-file-1>\n<DL><p>\n</DL><p>\n"), 0)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty export: %v, %v", got, err)
	}
}

func TestParseNetscapeInvalid(t *testing.T) {
	t.Parallel()
	for name, in := range map[string]string{
		"empty":           "",
		"plain text":      "just some text, no bookmarks",
		"html page":       `<!doctype html><html><body><p><a href="https://x.example/">outside a list</a></p></body></html>`,
		"json":            `{"format":"kinex-mark-links/1","links":[]}`,
		"binary":          "\x00\x01\x02\xff\xfe",
		"other doctype":   `<!DOCTYPE html><dl></dl>`,
		"broken markup":   `<<<>>> </a </dl`,
		"a without href":  `<dl><dt><a name="x">x</a></dl>`,
		"unclosed folder": `<h3>Folder`,
	} {
		if _, err := ParseNetscape(strings.NewReader(in), 0); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
}

func TestParseNetscapeLimit(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	b.WriteString("<!DOCTYPE NETSCAPE-Bookmark-file-1><DL><p>\n")
	for i := 0; i < 3; i++ {
		b.WriteString(`<DT><A HREF="https://x.example/">x</A>` + "\n")
	}
	in := b.String()
	if got, err := ParseNetscape(strings.NewReader(in), 3); err != nil || len(got) != 3 {
		t.Fatalf("limit 3: %d entries, %v", len(got), err)
	}
	if _, err := ParseNetscape(strings.NewReader(in), 2); !errors.Is(err, ErrTooManyEntries) {
		t.Fatalf("limit 2: err = %v", err)
	}
	// The limit also applies to an entry left open at the end of the file
	// and to an entry closed by a list end.
	if _, err := ParseNetscape(strings.NewReader(in+`<DT><A HREF="https://y.example/">open`), 3); !errors.Is(err, ErrTooManyEntries) {
		t.Fatalf("open entry at EOF: err = %v", err)
	}
	if _, err := ParseNetscape(strings.NewReader(in+`<DT><A HREF="https://y.example/">open</DL>`), 3); !errors.Is(err, ErrTooManyEntries) {
		t.Fatalf("entry closed by </DL>: err = %v", err)
	}
	if _, err := ParseNetscape(strings.NewReader(in+`<DT><A HREF="https://y.example/">open<DT><A HREF="https://z.example/">`), 3); !errors.Is(err, ErrTooManyEntries) {
		t.Fatalf("entry closed by the next <A>: err = %v", err)
	}
}

func TestParseAddDate(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]*time.Time{
		"1700000000":     ts(1700000000),
		" 1 ":            ts(1),
		"0":              nil,
		"-5":             nil,
		"1.5":            nil,
		"":               nil,
		"253402300799":   ts(253402300799),
		"253402300800":   nil,
		"99999999999999": nil,
	} {
		got := parseAddDate(in)
		if (got == nil) != (want == nil) || got != nil && !got.Equal(*want) {
			t.Errorf("parseAddDate(%q) = %v, want %v", in, got, want)
		}
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("disk on fire") }

func TestParseReadError(t *testing.T) {
	t.Parallel()
	_, err := ParseNetscape(io.MultiReader(strings.NewReader("<DL><DT><A HREF=x>"), errReader{}), 0)
	if err == nil || !strings.Contains(err.Error(), "disk on fire") || errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	if _, err := ParseJSON(errReader{}, 0); !errors.Is(err, ErrInvalid) {
		t.Fatalf("json read error: %v", err)
	}
}

var sample = []Entry{
	{
		URL:         "https://example.com/?a=1&b=<2>",
		Title:       `Tom & Jerry "quoted" <tag> 'single'`,
		Description: "First line\nSecond line with \u00e1rv\u00edzt\u0171r\u0151",
		Tags:        []string{"cartoons", "Kids & Family", "\u00e9kezet"},
		AddedAt:     ts(1767607200),
	},
	{URL: "http://plain.example/", Title: "", Description: "", Tags: nil, AddedAt: nil},
	{URL: "https://second.example/path", Title: "Second", Tags: []string{"one"}},
}

func TestNetscapeRoundTrip(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := WriteNetscape(&buf, sample); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"<!DOCTYPE NETSCAPE-Bookmark-file-1>\n",
		"<TITLE>Kinex Mark</TITLE>\n<H1>Kinex Mark</H1>\n",
		`<DT><A HREF="https://example.com/?a=1&amp;b=&lt;2&gt;" ADD_DATE="1767607200" TAGS="cartoons,Kids &amp; Family,` + "\u00e9kezet" + `">Tom &amp; Jerry &#34;quoted&#34; &lt;tag&gt; &#39;single&#39;</A>`,
		`<DT><A HREF="http://plain.example/"></A>`,
		"<DD>First line\nSecond line",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	got, err := ParseNetscape(strings.NewReader(out), len(sample))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, sample) {
		t.Fatalf("round trip =\n%+v\nwant\n%+v", got, sample)
	}
	// An empty export reads back as an empty list.
	buf.Reset()
	if err := WriteNetscape(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if got, err := ParseNetscape(&buf, 0); err != nil || len(got) != 0 {
		t.Fatalf("empty round trip: %v, %v", got, err)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	t.Parallel()
	exported := time.Date(2026, 11, 21, 9, 15, 0, 123000000, time.FixedZone("CET", 3600))
	var buf bytes.Buffer
	if err := WriteJSON(&buf, exported, sample); err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if raw["format"] != "kinex-mark-links/1" || raw["exportedAt"] != "2026-11-21T08:15:00.123Z" {
		t.Fatalf("header = %v %v", raw["format"], raw["exportedAt"])
	}
	links := raw["links"].([]any)
	second := links[1].(map[string]any)
	if tags, ok := second["tags"].([]any); !ok || len(tags) != 0 || second["addedAt"] != nil {
		t.Fatalf("empty tags and addedAt must be [] and null: %v", second)
	}
	first := links[0].(map[string]any)
	if first["addedAt"] != "2026-01-05T10:00:00.000Z" {
		t.Fatalf("addedAt = %v", first["addedAt"])
	}
	for _, k := range []string{"url", "title", "description", "tags", "addedAt"} {
		if _, ok := first[k]; !ok {
			t.Errorf("link lacks %q", k)
		}
	}
	if !strings.HasSuffix(buf.String(), "}\n") {
		t.Error("missing trailing newline")
	}

	got, err := ParseJSON(bytes.NewReader(buf.Bytes()), len(sample))
	if err != nil {
		t.Fatal(err)
	}
	want := append([]Entry(nil), sample...)
	want[1].Tags = []string{} // read back as an empty list
	for i := range got {
		if got[i].Tags == nil {
			got[i].Tags = []string{}
		}
		if want[i].Tags == nil {
			want[i].Tags = []string{}
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip =\n%+v\nwant\n%+v", got, want)
	}
}

func TestParseJSON(t *testing.T) {
	t.Parallel()
	valid := `{"format":"kinex-mark-links/1","exportedAt":"x","extra":true,"links":[
		{"url":"https://a.example/","title":"A","description":"d","tags":[" t1 ","t1","",null],"addedAt":"2026-01-05T11:00:00+01:00","unknown":1},
		{"url":"https://b.example/","tags":null,"addedAt":null}]}` + "\n\n"
	got, err := ParseJSON(strings.NewReader(valid), 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{
		{URL: "https://a.example/", Title: "A", Description: "d", Tags: []string{"t1"}, AddedAt: ts(1767607200)},
		{URL: "https://b.example/"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseJSON =\n%+v\nwant\n%+v", got, want)
	}
	if _, err := ParseJSON(strings.NewReader(valid), 1); !errors.Is(err, ErrTooManyEntries) {
		t.Fatalf("limit: %v", err)
	}
	for name, in := range map[string]string{
		"empty":          "",
		"not json":       "<!DOCTYPE NETSCAPE-Bookmark-file-1>",
		"wrong format":   `{"format":"kinex-mark-links/2","links":[]}`,
		"no format":      `{"links":[]}`,
		"missing links":  `{"format":"kinex-mark-links/1"}`,
		"null links":     `{"format":"kinex-mark-links/1","links":null}`,
		"links object":   `{"format":"kinex-mark-links/1","links":{}}`,
		"url number":     `{"format":"kinex-mark-links/1","links":[{"url":5}]}`,
		"bad addedAt":    `{"format":"kinex-mark-links/1","links":[{"url":"https://a.example/","addedAt":"yesterday"}]}`,
		"trailing data":  `{"format":"kinex-mark-links/1","links":[]} {}`,
		"trailing junk":  `{"format":"kinex-mark-links/1","links":[]} x`,
		"array root":     `[]`,
		"truncated json": `{"format":"kinex-mark-links/1","links":[`,
	} {
		if _, err := ParseJSON(strings.NewReader(in), 0); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
	// An empty list is a valid file.
	if got, err := ParseJSON(strings.NewReader(`{"format":"kinex-mark-links/1","links":[]}`), 0); err != nil || len(got) != 0 {
		t.Fatalf("empty list: %v, %v", got, err)
	}
}

// An addedAt before the Unix epoch counts as missing, as a Netscape ADD_DATE
// <= 0 does: a negative UTC year cannot be stored and read back.
func TestParseJSONAddedAtBeforeEpoch(t *testing.T) {
	t.Parallel()
	in := `{"format":"kinex-mark-links/1","links":[
		{"url":"https://year-zero.example/","addedAt":"0000-01-01T00:30:00+01:00"},
		{"url":"https://year-one.example/","addedAt":"0001-01-01T00:00:00Z"},
		{"url":"https://before.example/","addedAt":"1969-12-31T23:59:59.999Z"},
		{"url":"https://epoch.example/","addedAt":"1970-01-01T01:00:00+01:00"}]}`
	got, err := ParseJSON(strings.NewReader(in), 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range got[:3] {
		if e.AddedAt != nil {
			t.Errorf("%s: addedAt = %v, want nil", e.URL, e.AddedAt)
		}
	}
	if got[3].AddedAt == nil || !got[3].AddedAt.Equal(time.Unix(0, 0)) {
		t.Errorf("epoch: addedAt = %v", got[3].AddedAt)
	}
}

// manyTags returns n distinct tag names.
func manyTags(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("t%d", i)
	}
	return names
}

// One entry may carry about 190k tags within the 2 MiB import cap. Parsing
// must stay linear and keep at most MaxTagsPerEntry names (it used to
// compare every name with every kept one: about a minute of CPU).
func TestParseManyTagsIsBounded(t *testing.T) {
	t.Parallel()
	names := manyTags(200000)
	b, err := json.Marshal(map[string]any{"format": JSONFormat,
		"links": []map[string]any{{"url": "https://a.example/", "tags": append(names, "t0", " t1 ")}}})
	if err != nil {
		t.Fatal(err)
	}
	html := `<!DOCTYPE NETSCAPE-Bookmark-file-1><DL><DT><A HREF="https://a.example/" TAGS="` +
		strings.Join(names, ",") + `">A</A></DL>`
	for name, parse := range map[string]func() ([]Entry, error){
		"json":     func() ([]Entry, error) { return ParseJSON(bytes.NewReader(b), 0) },
		"netscape": func() ([]Entry, error) { return ParseNetscape(strings.NewReader(html), 0) },
	} {
		start := time.Now()
		got, err := parse()
		elapsed := time.Since(start)
		if err != nil || len(got) != 1 {
			t.Fatalf("%s: %d entries, %v", name, len(got), err)
		}
		if !reflect.DeepEqual(got[0].Tags, names[:MaxTagsPerEntry]) {
			t.Errorf("%s: kept %d tags, want the first %d", name, len(got[0].Tags), MaxTagsPerEntry)
		}
		if elapsed > 2*time.Second {
			t.Errorf("%s: parsing took %v", name, elapsed)
		}
	}
}

func TestAddTags(t *testing.T) {
	t.Parallel()
	got := addTags(nil, " a ", "b", "a", "", "  ")
	got = addTags(got, "b", "c")
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("addTags = %q", got)
	}
	full := manyTags(MaxTagsPerEntry)
	if got := addTags(full, "extra"); len(got) != MaxTagsPerEntry {
		t.Fatalf("a full list grew to %d", len(got))
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestWriteErrors(t *testing.T) {
	t.Parallel()
	if err := WriteNetscape(failWriter{}, sample); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("WriteNetscape err = %v", err)
	}
	if err := WriteJSON(failWriter{}, time.Unix(0, 0), sample); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("WriteJSON err = %v", err)
	}
}
