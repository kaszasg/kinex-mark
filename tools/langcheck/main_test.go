package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// writeTree creates the files (slash separated path -> content) under a new temporary root.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runCheck(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestForbiddenMatchesThePlanList(t *testing.T) {
	t.Parallel()
	const planList = "áéíóöőúüűÁÉÍÓÖŐÚÜŰäöüßÄÖÜ"
	for _, r := range planList {
		if !slices.Contains(forbidden, r) {
			t.Errorf("forbidden misses %q", r)
		}
	}
	for _, r := range forbidden {
		if !strings.ContainsRune(planList, r) {
			t.Errorf("forbidden has extra rune %q", r)
		}
	}
}

func TestOwnSourcePassesItself(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := check("tools/langcheck/main.go", data); len(got) != 0 {
		t.Fatalf("main.go contains non-English characters: %v", got)
	}
}

func TestCheckReportsDistinctCharactersPerLine(t *testing.T) {
	t.Parallel()
	data := []byte("clean line\r\n// őőé and ő again\r\nñ — č ł are allowed\nßäÄ\n")
	got := check("x.go", data)
	want := []string{
		`x.go:2: non-English character "ő"`,
		`x.go:2: non-English character "é"`,
		`x.go:4: non-English character "ß"`,
		`x.go:4: non-English character "ä"`,
		`x.go:4: non-English character "Ä"`,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d findings %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i].String() != want[i] {
			t.Errorf("finding %d = %q, want %q", i, got[i].String(), want[i])
		}
	}
}

func TestCheckToleratesBOMAndInvalidUTF8(t *testing.T) {
	t.Parallel()
	data := []byte("\xef\xbb\xbf#Requires -Version 5.1\n\xff\xfe broken\n")
	if got := check("build.ps1", data); len(got) != 0 {
		t.Fatalf("unexpected findings %v", got)
	}
}

func TestRunReportsFindingsAndRespectsScope(t *testing.T) {
	t.Parallel()
	const bad = "// előre\n"
	root := writeTree(t, map[string]string{
		// Checked.
		"cmd/kinex-mark/main.go":              "package main\n" + bad,
		"internal/x/x.go":                     "package x\n\n" + bad,
		"internal/q/q.sql":                    "-- tábla\n",
		"tools/fonts/build_fonts.py":          "# betűk\n",
		"web/src/App.svelte":                  "<p>Szia, világ</p>\n",
		"web/src/app.css":                     "/* színek */\n",
		"web/src/app.html":                    "<!-- fő -->\n",
		"web/src/lib/test/other.ts":           bad,
		"web/src/lib/x.js":                    bad,
		"web/e2e/support/helper.mjs":          bad,
		"web/scripts/sync-app-css.mjs":        bad,
		"web/vite.config.ts":                  bad,
		"web/embed.go":                        "package web\n" + bad,
		"graphics/generate_graphics.py":       "# kép\n",
		"Makefile":                            "# Straße\n",
		"Dockerfile":                          "# Ärger\n",
		"compose.yaml":                        "# hätte\n",
		"build.ps1":                           "\xef\xbb\xbf# főág\r\n",
		".github/workflows/ci.yml":            "# élő\n",
		".github/dependabot.yml":              "version: 2\n",
		"internal/clean/clean.go":             "package clean\n",
		"web/src/lib/components/Clean.svelte": "<p>Hello</p>\n",
		// Skipped.
		"internal/x/x_test.go":                  bad,
		"internal/x/testdata/data.go":           bad,
		"internal/q/q.txt":                      bad,
		"internal/q/README.md":                  bad,
		"web/src/lib/i18n/messages/hu/index.ts": bad,
		"web/src/lib/i18n/messages/hu/links.ts": bad,
		"web/src/lib/test/fixtures.ts":          bad,
		"web/src/lib/a.spec.ts":                 bad,
		"web/src/lib/a.test.ts":                 bad,
		"web/src/node_modules/pkg/index.js":     bad,
		"web/src/.svelte-kit/generated/root.js": bad,
		"web/src/build/index.js":                bad,
		"web/build/index.html":                  bad,
		"web/e2e/fixtures/site/index.html":      bad,
		"web/e2e/specs/links.spec.ts":           bad,
		"web/e2e/specs/helper.ts":               bad,
		"web/other/deep.ts":                     bad,
		"web/package.json":                      bad,
		"graphics/sub/nested.py":                bad,
		"graphics/prompts.json":                 bad,
		"ai/BASE-SYSTEM-PLAN.md":                bad,
		"README.md":                             bad,
		"other.go":                              bad,
	})

	code, out, errOut := runCheck(t, "-root", root)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", code, errOut)
	}
	want := []string{
		`cmd/kinex-mark/main.go:2: non-English character "ő"`,
		`internal/q/q.sql:1: non-English character "á"`,
		`internal/x/x.go:3: non-English character "ő"`,
		`tools/fonts/build_fonts.py:1: non-English character "ű"`,
		`web/src/App.svelte:1: non-English character "á"`,
		`web/src/app.css:1: non-English character "í"`,
		`web/src/app.html:1: non-English character "ő"`,
		`web/src/lib/test/other.ts:1: non-English character "ő"`,
		`web/src/lib/x.js:1: non-English character "ő"`,
		`web/e2e/support/helper.mjs:1: non-English character "ő"`,
		`web/scripts/sync-app-css.mjs:1: non-English character "ő"`,
		`.github/workflows/ci.yml:1: non-English character "é"`,
		`.github/workflows/ci.yml:1: non-English character "ő"`,
		`web/vite.config.ts:1: non-English character "ő"`,
		`web/embed.go:2: non-English character "ő"`,
		`graphics/generate_graphics.py:1: non-English character "é"`,
		`Makefile:1: non-English character "ß"`,
		`build.ps1:1: non-English character "ő"`,
		`build.ps1:1: non-English character "á"`,
		`Dockerfile:1: non-English character "Ä"`,
		`compose.yaml:1: non-English character "ä"`,
	}
	got := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("stdout:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if !strings.Contains(errOut, "langcheck: 21 non-English characters in 19 of 22 files") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestRunCleanTree(t *testing.T) {
	t.Parallel()
	root := writeTree(t, map[string]string{
		"internal/a/a.go":       "package a\n",
		"web/src/app.html":      "<html></html>\n",
		"web/src/lib/x_test.go": "// ő is test data here\n",
		"Makefile":              "all:\n",
	})
	code, out, errOut := runCheck(t, "-root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	if out != "langcheck: 3 files checked, no non-English characters\n" || errOut != "" {
		t.Errorf("stdout = %q, stderr = %q", out, errOut)
	}
}

func TestRunEmptyRootAndFilesInPlaceOfDirectories(t *testing.T) {
	t.Parallel()
	// "cmd" and ".github" are files here, and "Makefile" is a directory: all are ignored.
	root := writeTree(t, map[string]string{
		"cmd":            "ő",
		".github":        "ő",
		"Makefile/x.txt": "ő",
	})
	if err := os.MkdirAll(filepath.Join(root, "web", "dir.ts"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCheck(t, "-root", root)
	if code != 0 || out != "langcheck: 0 files checked, no non-English characters\n" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
}

func TestRunDefaultRootIsWorkingDirectory(t *testing.T) {
	// Not parallel: it changes the working directory.
	root := writeTree(t, map[string]string{"internal/a/a.go": "// tűz\n"})
	t.Chdir(root)
	code, out, _ := runCheck(t)
	if code != 1 || out != "internal/a/a.go:1: non-English character \"ű\"\n" {
		t.Fatalf("code = %d, stdout = %q", code, out)
	}
}

func TestRunUsageErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unknown flag", []string{"-nope"}, "flag provided but not defined"},
		{"positional argument", []string{"extra"}, "unexpected arguments: extra"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, _, errOut := runCheck(t, tt.args...)
			if code != 2 || !strings.Contains(errOut, tt.wantErr) {
				t.Fatalf("code = %d, stderr = %q", code, errOut)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	t.Parallel()
	code, _, errOut := runCheck(t, "-h")
	if code != 0 || !strings.Contains(errOut, "-root") {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
}

func TestWantedAndSkipDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rel  string
		want bool
	}{
		{"internal/a/a.go", true},
		{"internal/a/a_test.go", false},
		{"internal/store/migrations/0001_users.sql", true},
		{"web/src/routes/+page.svelte", true},
		{"web/src/lib/x.spec.ts", false},
		{"web/src/lib/x.test.ts", false},
		{"web/src/lib/test/fixtures.ts", false},
		{"web/src/lib/test/setup.ts", true},
		{"web/src/lib/i18n/messages/en/index.ts", false},
		{"web/src/lib/i18n/format.ts", true},
		{"web/e2e/fixtures/site/a.html", false},
		{"internal/api/testdata/contract.go", false},
		{"web/node_modules/x/index.js", false},
		{"Makefile", true},
		{"Dockerfile", true},
		{"build.ps1", true},
		{"README.md", false},
		{"web/static/favicon.svg", false},
		{"web/package.json", false},
	}
	for _, tt := range tests {
		if got := wanted(tt.rel); got != tt.want {
			t.Errorf("wanted(%q) = %v, want %v", tt.rel, got, tt.want)
		}
	}
}
