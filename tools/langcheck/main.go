// Command langcheck verifies that the source code is written in English.
//
// It walks the source trees of the repository and reports every Hungarian or
// German letter outside the i18n message modules and the test data. User
// visible text belongs in web/src/lib/i18n/messages only.
//
// Output: one "<file>:<line>: non-English character "x"" line per finding and
// exit code 1; exit code 0 when the tree is clean; exit code 2 on I/O errors.
//
// Usage:
//
//	go run ./tools/langcheck [-root .]
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

// Exit codes.
const (
	exitOK       = 0
	exitFindings = 1
	exitError    = 2
)

// forbidden holds the Hungarian and German letters that never appear in
// English source code. They are written as code points so this file passes itself.
var forbidden = []rune{
	// Lower case Hungarian: a, e, i, o acute; o diaeresis; o double acute; u acute; u diaeresis; u double acute.
	0x00E1, 0x00E9, 0x00ED, 0x00F3, 0x00F6, 0x0151, 0x00FA, 0x00FC, 0x0171,
	// Upper case Hungarian, in the same order.
	0x00C1, 0x00C9, 0x00CD, 0x00D3, 0x00D6, 0x0150, 0x00DA, 0x00DC, 0x0170,
	// German letters not covered above: a diaeresis, sharp s, upper case A diaeresis.
	0x00E4, 0x00DF, 0x00C4,
}

// Directories walked recursively, relative to the repository root.
var scanDirs = []string{
	"cmd",
	"internal",
	"tools",
	"web/src",
	"web/e2e/support",
	"web/scripts",
	".github",
}

// Glob patterns for single directory levels.
var scanGlobs = []string{
	"web/*.ts",
	"web/*.go",
	"graphics/*.py",
}

// Single files at the repository root.
var scanFiles = []string{
	"Makefile",
	"build.ps1",
	"Dockerfile",
	"compose.yaml",
}

// Checked file extensions; Makefile and Dockerfile are matched by name.
var extensions = map[string]bool{
	".go": true, ".ts": true, ".svelte": true, ".js": true, ".mjs": true,
	".css": true, ".html": true, ".yml": true, ".yaml": true, ".ps1": true,
	".sql": true, ".py": true,
}

var fileNames = map[string]bool{
	"Makefile":   true,
	"Dockerfile": true,
}

// Directory names skipped wherever they occur.
var skipDirNames = map[string]bool{
	"node_modules": true,
	"build":        true,
	".svelte-kit":  true,
	"testdata":     true,
}

// Directory paths skipped with everything below them.
var skipDirPaths = []string{
	"web/src/lib/i18n/messages",
	"web/e2e/fixtures",
}

// Single files that may contain accented test data.
var skipFilePaths = map[string]bool{
	"web/src/lib/test/fixtures.ts": true,
}

// Test file suffixes: test data may contain accented text.
var skipSuffixes = []string{"_test.go", ".spec.ts", ".test.ts"}

// finding is one non-English character on one line.
type finding struct {
	file string
	line int
	char rune
}

func (f finding) String() string {
	return fmt.Sprintf("%s:%d: non-English character %q", f.file, f.line, string(f.char))
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fset := flag.NewFlagSet("langcheck", flag.ContinueOnError)
	fset.SetOutput(stderr)
	root := fset.String("root", ".", "repository root to check")
	if err := fset.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitError
	}
	if fset.NArg() > 0 {
		fmt.Fprintf(stderr, "langcheck: unexpected arguments: %s\n", strings.Join(fset.Args(), " "))
		return exitError
	}

	files, err := collect(*root)
	if err != nil {
		fmt.Fprintf(stderr, "langcheck: %v\n", err)
		return exitError
	}

	var findings []finding
	badFiles := 0
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(*root, filepath.FromSlash(rel)))
		if err != nil {
			fmt.Fprintf(stderr, "langcheck: %v\n", err)
			return exitError
		}
		found := check(rel, data)
		if len(found) > 0 {
			badFiles++
			findings = append(findings, found...)
		}
	}

	for _, f := range findings {
		fmt.Fprintln(stdout, f)
	}
	if len(findings) > 0 {
		fmt.Fprintf(stderr, "langcheck: %d non-English characters in %d of %d files\n", len(findings), badFiles, len(files))
		return exitFindings
	}
	fmt.Fprintf(stdout, "langcheck: %d files checked, no non-English characters\n", len(files))
	return exitOK
}

// collect returns the slash separated paths, relative to root, of every file
// to check, in a deterministic order and without duplicates.
func collect(root string) ([]string, error) {
	seen := make(map[string]bool)
	var files []string
	add := func(rel string) {
		if !seen[rel] && wanted(rel) {
			seen[rel] = true
			files = append(files, rel)
		}
	}

	for _, dir := range scanDirs {
		base := filepath.Join(root, filepath.FromSlash(dir))
		info, err := os.Stat(base)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if p != base && skipDir(rel) {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type().IsRegular() {
				add(rel)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	for _, pattern := range scanGlobs {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil {
				return nil, err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			rel, err := filepath.Rel(root, m)
			if err != nil {
				return nil, err
			}
			add(filepath.ToSlash(rel))
		}
	}

	for _, name := range scanFiles {
		info, err := os.Stat(filepath.Join(root, name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode().IsRegular() {
			add(name)
		}
	}
	return files, nil
}

// skipDir reports whether the directory rel (slash separated) is excluded.
func skipDir(rel string) bool {
	if skipDirNames[path.Base(rel)] {
		return true
	}
	for _, p := range skipDirPaths {
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}

// wanted reports whether the file rel (slash separated) is checked.
func wanted(rel string) bool {
	base := path.Base(rel)
	if !extensions[path.Ext(base)] && !fileNames[base] {
		return false
	}
	if skipFilePaths[rel] {
		return false
	}
	for _, s := range skipSuffixes {
		if strings.HasSuffix(base, s) {
			return false
		}
	}
	// A file can be reached through a glob inside an excluded directory.
	for dir := path.Dir(rel); dir != "."; dir = path.Dir(dir) {
		if skipDir(dir) {
			return false
		}
	}
	return true
}

// check returns the forbidden characters of data, one finding per distinct
// character per line, in the order of their first occurrence.
func check(rel string, data []byte) []finding {
	var findings []finding
	for i, line := range bytes.Split(data, []byte("\n")) {
		var reported []rune
		for len(line) > 0 {
			r, size := utf8.DecodeRune(line)
			line = line[size:]
			if r < utf8.RuneSelf || !slices.Contains(forbidden, r) || slices.Contains(reported, r) {
				continue
			}
			reported = append(reported, r)
			findings = append(findings, finding{file: rel, line: i + 1, char: r})
		}
	}
	return findings
}
