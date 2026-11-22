// Command covercheck enforces the Go statement coverage gate.
//
// It reads a coverage profile written by "go test -coverprofile", merges the
// blocks by key (the larger count wins), computes statement coverage per
// package and in total for the packages under -prefix that are not excluded,
// and exits with code 1 when the total or any package is below its threshold.
//
// Usage:
//
//	go run ./tools/covercheck -profile coverage.out -total 85 -package 80 [-markdown]
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultPrefix  = "github.com/kaszasg/kinex-mark/internal/"
	defaultExclude = "github.com/kaszasg/kinex-mark/internal/testutil,github.com/kaszasg/kinex-mark/internal/storetest"
	modePrefix     = "mode:"
)

// Exit codes.
const (
	exitOK    = 0
	exitBelow = 1 // the gate failed
	exitUsage = 2 // invalid flags or an unreadable profile
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// options holds the parsed command line.
type options struct {
	profile  string
	total    float64
	pkg      float64
	prefix   string
	exclude  []string
	markdown bool
}

// block is one merged coverage block.
type block struct {
	statements int64
	count      int64
}

// packageStat is the coverage of one package.
type packageStat struct {
	name       string
	statements int64
	covered    int64
}

// result is the outcome of the gate.
type result struct {
	packages []packageStat
	total    packageStat
	failed   []string // packages below the package threshold
	totalLow bool
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseFlags(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}

	f, err := os.Open(opts.profile)
	if err != nil {
		fmt.Fprintf(stderr, "covercheck: %v\n", err)
		return exitUsage
	}
	defer f.Close()

	blocks, err := parseProfile(f)
	if err != nil {
		fmt.Fprintf(stderr, "covercheck: %s: %v\n", opts.profile, err)
		return exitUsage
	}

	res := evaluate(blocks, opts)
	if len(res.packages) == 0 {
		fmt.Fprintf(stderr, "covercheck: no coverage data for packages under %s\n", opts.prefix)
		return exitBelow
	}

	if opts.markdown {
		writeMarkdown(stdout, res, opts)
	} else {
		writeText(stdout, res, opts)
	}

	if res.totalLow || len(res.failed) > 0 {
		fmt.Fprintln(stderr, failureMessage(res, opts))
		return exitBelow
	}
	return exitOK
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	fs := flag.NewFlagSet("covercheck", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts options
	var exclude string
	fs.StringVar(&opts.profile, "profile", "coverage.out", "coverage profile written by go test -coverprofile")
	fs.Float64Var(&opts.total, "total", 85, "minimum total statement coverage in percent")
	fs.Float64Var(&opts.pkg, "package", 80, "minimum statement coverage of every package in percent")
	fs.StringVar(&opts.prefix, "prefix", defaultPrefix, "only packages whose import path starts with this prefix are gated")
	fs.StringVar(&exclude, "exclude", defaultExclude, "comma separated import paths of packages (and their subpackages) to skip")
	fs.BoolVar(&opts.markdown, "markdown", false, "print a Markdown table instead of plain text")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "covercheck: unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return opts, errors.New("unexpected arguments")
	}
	if opts.total < 0 || opts.total > 100 || opts.pkg < 0 || opts.pkg > 100 {
		fmt.Fprintln(stderr, "covercheck: thresholds must be between 0 and 100")
		return opts, errors.New("invalid threshold")
	}
	for _, p := range strings.Split(exclude, ",") {
		p = strings.TrimSuffix(strings.TrimSpace(p), "/")
		if p != "" {
			opts.exclude = append(opts.exclude, p)
		}
	}
	return opts, nil
}

// parseProfile reads a coverage profile and merges duplicated blocks.
// The key of a block is "file:startLine.startCol,endLine.endCol".
func parseProfile(r io.Reader) (map[string]block, error) {
	blocks := make(map[string]block)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	sawMode := false
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, modePrefix) {
			// Concatenated profiles repeat the mode line.
			sawMode = true
			continue
		}
		if !sawMode {
			return nil, fmt.Errorf("line %d: missing %q header", lineNo, modePrefix)
		}
		key, stmts, count, err := parseBlock(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if prev, ok := blocks[key]; ok {
			if count < prev.count {
				count = prev.count
			}
			if stmts < prev.statements {
				stmts = prev.statements
			}
		}
		blocks[key] = block{statements: stmts, count: count}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !sawMode {
		return nil, fmt.Errorf("missing %q header", modePrefix)
	}
	return blocks, nil
}

// parseBlock splits "file:12.2,14.3 2 1" into its key, statement count and hit count.
func parseBlock(line string) (key string, stmts, count int64, err error) {
	i := strings.LastIndexByte(line, ' ')
	if i < 0 {
		return "", 0, 0, fmt.Errorf("malformed block %q", line)
	}
	count, err = strconv.ParseInt(line[i+1:], 10, 64)
	if err != nil || count < 0 {
		return "", 0, 0, fmt.Errorf("malformed count in %q", line)
	}
	rest := line[:i]
	j := strings.LastIndexByte(rest, ' ')
	if j < 0 {
		return "", 0, 0, fmt.Errorf("malformed block %q", line)
	}
	stmts, err = strconv.ParseInt(rest[j+1:], 10, 64)
	if err != nil || stmts < 0 {
		return "", 0, 0, fmt.Errorf("malformed statement count in %q", line)
	}
	key = rest[:j]
	colon := strings.LastIndexByte(key, ':')
	if colon <= 0 || !validRange(key[colon+1:]) {
		return "", 0, 0, fmt.Errorf("malformed position in %q", line)
	}
	return key, stmts, count, nil
}

// validRange reports whether s has the form "line.col,line.col".
func validRange(s string) bool {
	start, end, ok := strings.Cut(s, ",")
	return ok && validPos(start) && validPos(end)
}

func validPos(s string) bool {
	l, c, ok := strings.Cut(s, ".")
	return ok && isDigits(l) && isDigits(c)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// evaluate aggregates the blocks per package and applies the thresholds.
func evaluate(blocks map[string]block, opts options) result {
	byPkg := make(map[string]*packageStat)
	for key, b := range blocks {
		file := key[:strings.LastIndexByte(key, ':')]
		pkg := path.Dir(file)
		// The trailing slash lets the prefix ".../internal/" match ".../internal" itself.
		if !strings.HasPrefix(pkg+"/", opts.prefix) {
			continue
		}
		if excluded(pkg, opts.exclude) {
			continue
		}
		st := byPkg[pkg]
		if st == nil {
			st = &packageStat{name: pkg}
			byPkg[pkg] = st
		}
		st.statements += b.statements
		if b.count > 0 {
			st.covered += b.statements
		}
	}

	var res result
	res.total.name = "total"
	for _, st := range byPkg {
		res.packages = append(res.packages, *st)
		res.total.statements += st.statements
		res.total.covered += st.covered
	}
	sort.Slice(res.packages, func(i, j int) bool { return res.packages[i].name < res.packages[j].name })
	for _, st := range res.packages {
		if below(st, opts.pkg) {
			res.failed = append(res.failed, st.name)
		}
	}
	res.totalLow = below(res.total, opts.total)
	return res
}

func excluded(pkg string, exclude []string) bool {
	for _, e := range exclude {
		if pkg == e || strings.HasPrefix(pkg, e+"/") {
			return true
		}
	}
	return false
}

// below compares exactly: covered/statements*100 < threshold.
// A package without statements has nothing to cover and never fails.
func below(st packageStat, threshold float64) bool {
	if st.statements == 0 {
		return false
	}
	return float64(st.covered)*100 < threshold*float64(st.statements)
}

// percent formats the coverage rounded down to one decimal, so a value
// shown as reaching the threshold never fails the gate.
func percent(st packageStat) string {
	if st.statements == 0 {
		return "n/a"
	}
	tenths := st.covered * 1000 / st.statements
	return fmt.Sprintf("%d.%d%%", tenths/10, tenths%10)
}

func threshold(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64) + "%"
}

func writeText(w io.Writer, res result, opts options) {
	width := 0
	for _, st := range res.packages {
		if len(st.name) > width {
			width = len(st.name)
		}
	}
	for _, st := range res.packages {
		status := "ok"
		if below(st, opts.pkg) {
			status = "FAIL"
		}
		fmt.Fprintf(w, "%-*s  %7s  (%d/%d statements)  %s\n", width, st.name, percent(st), st.covered, st.statements, status)
	}
	fmt.Fprintf(w, "total: %s (threshold %s)\n", percent(res.total), threshold(opts.total))
}

func writeMarkdown(w io.Writer, res result, opts options) {
	fmt.Fprintln(w, "### Go coverage")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Package | Statements | Covered | Coverage | Status |")
	fmt.Fprintln(w, "|---|---:|---:|---:|---|")
	for _, st := range res.packages {
		status := "ok"
		if below(st, opts.pkg) {
			status = "**FAIL**"
		}
		fmt.Fprintf(w, "| `%s` | %d | %d | %s | %s |\n", st.name, st.statements, st.covered, percent(st), status)
	}
	status := "ok"
	if res.totalLow {
		status = "**FAIL**"
	}
	fmt.Fprintf(w, "| **total** | %d | %d | **%s** | %s |\n", res.total.statements, res.total.covered, percent(res.total), status)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "total: %s (threshold %s), package threshold %s\n", percent(res.total), threshold(opts.total), threshold(opts.pkg))
	if res.totalLow || len(res.failed) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "**%s**\n", failureMessage(res, opts))
	}
}

func failureMessage(res result, opts options) string {
	var parts []string
	if res.totalLow {
		parts = append(parts, fmt.Sprintf("total coverage %s is below %s", percent(res.total), threshold(opts.total)))
	}
	if len(res.failed) > 0 {
		parts = append(parts, fmt.Sprintf("packages below %s: %s", threshold(opts.pkg), strings.Join(res.failed, ", ")))
	}
	return "coverage gate failed: " + strings.Join(parts, "; ")
}
