package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const mod = "github.com/kaszasg/kinex-mark/"

// writeProfile stores a coverage profile in a temporary file and returns its path.
func writeProfile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func runCheck(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// A profile in which internal/a has 9/10 statements covered, internal/b 7/10,
// and the excluded and foreign packages are uncovered.
const sampleProfile = `mode: atomic
github.com/kaszasg/kinex-mark/internal/a/a.go:1.1,2.1 5 3
github.com/kaszasg/kinex-mark/internal/a/a.go:3.1,4.1 4 1
github.com/kaszasg/kinex-mark/internal/a/b.go:1.1,2.1 1 0
github.com/kaszasg/kinex-mark/internal/b/b.go:1.1,2.1 7 2
github.com/kaszasg/kinex-mark/internal/b/b.go:3.1,4.1 3 0
github.com/kaszasg/kinex-mark/internal/testutil/t.go:1.1,2.1 50 0
github.com/kaszasg/kinex-mark/internal/storetest/s.go:1.1,2.1 50 0
github.com/kaszasg/kinex-mark/cmd/kinex-mark/main.go:1.1,2.1 40 0
github.com/kaszasg/kinex-mark/tools/langcheck/main.go:1.1,2.1 40 0
`

func TestParseProfileMergesBlocksLargerCountWins(t *testing.T) {
	t.Parallel()
	profile := `mode: atomic
x/p/f.go:1.1,2.1 2 0
x/p/f.go:3.1,4.1 3 4
mode: atomic
x/p/f.go:1.1,2.1 2 5
x/p/f.go:3.1,4.1 3 0
`
	blocks, err := parseProfile(strings.NewReader(profile))
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if got := blocks["x/p/f.go:1.1,2.1"]; got.count != 5 || got.statements != 2 {
		t.Errorf("first block = %+v, want count 5, statements 2", got)
	}
	if got := blocks["x/p/f.go:3.1,4.1"]; got.count != 4 || got.statements != 3 {
		t.Errorf("second block = %+v, want count 4, statements 3", got)
	}
}

func TestParseProfileErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		profile string
		wantErr string
	}{
		{"empty", "", "missing"},
		{"no header", "x/p/f.go:1.1,2.1 1 1\n", "missing"},
		{"too few fields", "mode: set\nx/p/f.go:1.1,2.1\n", "malformed"},
		{"one field", "mode: set\nnothing\n", "malformed"},
		{"bad count", "mode: set\nx/p/f.go:1.1,2.1 1 z\n", "malformed count"},
		{"negative count", "mode: set\nx/p/f.go:1.1,2.1 1 -1\n", "malformed count"},
		{"bad statements", "mode: set\nx/p/f.go:1.1,2.1 z 1\n", "malformed statement count"},
		{"no colon", "mode: set\nx/p/f.go 1 1\n", "malformed position"},
		{"bad range", "mode: set\nx/p/f.go:1.1-2.1 1 1\n", "malformed position"},
		{"bad position", "mode: set\nx/p/f.go:1,2.1 1 1\n", "malformed position"},
		{"letters in position", "mode: set\nx/p/f.go:a.1,2.1 1 1\n", "malformed position"},
		{"empty column", "mode: set\nx/p/f.go:1.,2.1 1 1\n", "malformed position"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseProfile(strings.NewReader(tt.profile))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseProfileSkipsBlankLinesAndAcceptsSpacesInFileNames(t *testing.T) {
	t.Parallel()
	blocks, err := parseProfile(strings.NewReader("\nmode: count\n\nx/my dir/f.go:1.1,2.1 3 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := blocks["x/my dir/f.go:1.1,2.1"]; got.statements != 3 || got.count != 1 {
		t.Fatalf("block = %+v", got)
	}
}

func TestRunPassesAndPrintsPerPackageLines(t *testing.T) {
	t.Parallel()
	p := writeProfile(t, sampleProfile)
	code, out, errOut := runCheck(t, "-profile", p, "-total", "80", "-package", "70")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, errOut)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], mod+"internal/a ") || !strings.Contains(lines[0], "90.0%") || !strings.Contains(lines[0], "(9/10 statements)") || !strings.HasSuffix(lines[0], "ok") {
		t.Errorf("line for package a = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], mod+"internal/b ") || !strings.Contains(lines[1], "70.0%") {
		t.Errorf("line for package b = %q", lines[1])
	}
	if lines[2] != "total: 80.0% (threshold 80%)" {
		t.Errorf("total line = %q", lines[2])
	}
	for _, skipped := range []string{"testutil", "storetest", "cmd/", "tools/"} {
		if strings.Contains(out, skipped) {
			t.Errorf("output mentions %q although it is outside the gate:\n%s", skipped, out)
		}
	}
	if errOut != "" {
		t.Errorf("unexpected stderr %q", errOut)
	}
}

func TestRunFailsOnPackageThreshold(t *testing.T) {
	t.Parallel()
	p := writeProfile(t, sampleProfile)
	code, out, errOut := runCheck(t, "-profile", p, "-total", "80", "-package", "75")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out, "70.0%") || !strings.Contains(out, "FAIL") {
		t.Errorf("stdout does not mark the failing package:\n%s", out)
	}
	want := "coverage gate failed: packages below 75%: " + mod + "internal/b"
	if strings.TrimSpace(errOut) != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

func TestRunFailsOnTotalThreshold(t *testing.T) {
	t.Parallel()
	p := writeProfile(t, sampleProfile)
	code, out, errOut := runCheck(t, "-profile", p, "-total", "80.5", "-package", "70")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out, "total: 80.0% (threshold 80.5%)") {
		t.Errorf("stdout = %q", out)
	}
	if !strings.Contains(errOut, "total coverage 80.0% is below 80.5%") || strings.Contains(errOut, "packages below") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestRunDefaultsUseCoverageOutAndFail(t *testing.T) {
	// Not parallel: it changes the working directory.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "coverage.out"), []byte(sampleProfile), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	code, _, errOut := runCheck(t)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 with the default thresholds 85/80", code)
	}
	if !strings.Contains(errOut, "below 85%") || !strings.Contains(errOut, "packages below 80%: "+mod+"internal/b") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestRunExcludeFlagReplacesDefault(t *testing.T) {
	t.Parallel()
	p := writeProfile(t, sampleProfile)
	// With b excluded and testutil no longer excluded, testutil (0%) fails.
	code, out, errOut := runCheck(t, "-profile", p, "-total", "0", "-package", "50", "-exclude", " "+mod+"internal/b/ ,,")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if strings.Contains(out, "internal/b") {
		t.Errorf("excluded package listed:\n%s", out)
	}
	if !strings.Contains(errOut, mod+"internal/storetest, "+mod+"internal/testutil") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestRunExcludeCoversSubpackages(t *testing.T) {
	t.Parallel()
	profile := "mode: set\n" +
		mod + "internal/a/a.go:1.1,2.1 1 1\n" +
		mod + "internal/x/y/z.go:1.1,2.1 1 0\n" +
		mod + "internal/xy/z.go:1.1,2.1 1 1\n"
	p := writeProfile(t, profile)
	code, out, errOut := runCheck(t, "-profile", p, "-exclude", mod+"internal/x")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, errOut)
	}
	if strings.Contains(out, "internal/x/y") || !strings.Contains(out, "internal/xy") {
		t.Errorf("subpackage exclusion is wrong:\n%s", out)
	}
}

func TestRunPrefixFlag(t *testing.T) {
	t.Parallel()
	p := writeProfile(t, sampleProfile)
	code, out, errOut := runCheck(t, "-profile", p, "-prefix", mod+"tools/", "-total", "0", "-package", "0")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, errOut)
	}
	if !strings.Contains(out, mod+"tools/langcheck") || strings.Contains(out, "internal/") {
		t.Errorf("stdout = %q", out)
	}
}

func TestRunMarkdown(t *testing.T) {
	t.Parallel()
	p := writeProfile(t, sampleProfile)
	code, out, _ := runCheck(t, "-profile", p, "-total", "85", "-package", "75", "-markdown")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	for _, want := range []string{
		"| Package | Statements | Covered | Coverage | Status |",
		"|---|---:|---:|---:|---|",
		"| `" + mod + "internal/a` | 10 | 9 | 90.0% | ok |",
		"| `" + mod + "internal/b` | 10 | 7 | 70.0% | **FAIL** |",
		"| **total** | 20 | 16 | **80.0%** | **FAIL** |",
		"total: 80.0% (threshold 85%), package threshold 75%",
		"**coverage gate failed: total coverage 80.0% is below 85%; packages below 75%: " + mod + "internal/b**",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown misses %q:\n%s", want, out)
		}
	}
}

func TestRunMarkdownPassing(t *testing.T) {
	t.Parallel()
	p := writeProfile(t, sampleProfile)
	code, out, _ := runCheck(t, "-profile", p, "-total", "50", "-package", "50", "-markdown")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if strings.Contains(out, "FAIL") || strings.Contains(out, "failed") {
		t.Errorf("passing markdown reports a failure:\n%s", out)
	}
	if !strings.Contains(out, "| **total** | 20 | 16 | **80.0%** | ok |") {
		t.Errorf("markdown total row missing:\n%s", out)
	}
}

func TestRunUncoveredPackageWithoutTestsFails(t *testing.T) {
	t.Parallel()
	// go test lists packages without tests with count 0 blocks.
	profile := "mode: atomic\n" +
		mod + "internal/a/a.go:1.1,2.1 99 1\n" +
		mod + "internal/notest/n.go:1.1,2.1 1 0\n"
	p := writeProfile(t, profile)
	code, _, errOut := runCheck(t, "-profile", p)
	if code != 1 || !strings.Contains(errOut, mod+"internal/notest") {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
}

func TestRunZeroStatementPackageNeverFails(t *testing.T) {
	t.Parallel()
	profile := "mode: set\n" +
		mod + "internal/a/a.go:1.1,2.1 4 1\n" +
		mod + "internal/empty/e.go:1.1,1.2 0 0\n"
	p := writeProfile(t, profile)
	code, out, errOut := runCheck(t, "-profile", p, "-total", "100", "-package", "100")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	if !strings.Contains(out, "n/a") {
		t.Errorf("zero statement package not shown as n/a:\n%s", out)
	}
}

func TestRunThresholdComparisonIsExact(t *testing.T) {
	t.Parallel()
	// 7999/10000 = 79.99%: displayed rounded down as 79.9% and fails at 80.
	profile := "mode: set\n" +
		mod + "internal/a/a.go:1.1,2.1 7999 1\n" +
		mod + "internal/a/a.go:3.1,4.1 2001 0\n"
	p := writeProfile(t, profile)
	code, out, _ := runCheck(t, "-profile", p, "-total", "80", "-package", "0")
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(out, "total: 79.9% (threshold 80%)") {
		t.Errorf("stdout = %q", out)
	}
	code, _, _ = runCheck(t, "-profile", p, "-total", "79.99", "-package", "0")
	if code != 0 {
		t.Fatalf("code = %d at exactly the threshold, want 0", code)
	}
}

func TestRunNoMatchingPackages(t *testing.T) {
	t.Parallel()
	p := writeProfile(t, "mode: set\nexample.com/other/x.go:1.1,2.1 1 1\n")
	code, out, errOut := runCheck(t, "-profile", p)
	if code != 1 || out != "" || !strings.Contains(errOut, "no coverage data") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
}

func TestRunUsageErrors(t *testing.T) {
	t.Parallel()
	good := writeProfile(t, sampleProfile)
	bad := writeProfile(t, "mode: set\nbroken line here\n")
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"missing file", []string{"-profile", filepath.Join(t.TempDir(), "none.out")}, "covercheck:"},
		{"malformed profile", []string{"-profile", bad}, "line 2"},
		{"unknown flag", []string{"-nope"}, "flag provided but not defined"},
		{"bad number", []string{"-total", "x"}, "invalid value"},
		{"threshold too high", []string{"-profile", good, "-total", "101"}, "between 0 and 100"},
		{"negative threshold", []string{"-profile", good, "-package", "-1"}, "between 0 and 100"},
		{"positional argument", []string{"-profile", good, "extra"}, "unexpected arguments: extra"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, _, errOut := runCheck(t, tt.args...)
			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if !strings.Contains(errOut, tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", errOut, tt.wantErr)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	t.Parallel()
	code, _, errOut := runCheck(t, "-h")
	if code != 0 || !strings.Contains(errOut, "-markdown") {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
}

func TestPercentRoundsDown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		covered, statements int64
		want                string
	}{
		{0, 0, "n/a"},
		{0, 3, "0.0%"},
		{2, 3, "66.6%"},
		{1, 3, "33.3%"},
		{3, 3, "100.0%"},
		{873, 1000, "87.3%"},
	}
	for _, tt := range tests {
		if got := percent(packageStat{covered: tt.covered, statements: tt.statements}); got != tt.want {
			t.Errorf("percent(%d/%d) = %q, want %q", tt.covered, tt.statements, got, tt.want)
		}
	}
}
