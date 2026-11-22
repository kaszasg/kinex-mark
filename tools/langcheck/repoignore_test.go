package main

// Repository hygiene checks for the ignore files of the repository root.
// They live next to langcheck because both guard what may enter the
// repository; the tests read the real files two levels above this package.

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// repoRoot is the repository root relative to this package directory.
var repoRoot = filepath.Join("..", "..")

// ignoreLines returns the non-empty, non-comment lines of a root ignore file.
func ignoreLines(t *testing.T, name string) map[string]bool {
	t.Helper()
	f, err := os.Open(filepath.Join(repoRoot, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines := make(map[string]bool)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line != "" && line[0] != '#' {
			lines[line] = true
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}

// Paths the Playwright MCP may write to: snapshots, console logs, downloads
// (recovery codes, data exports) and screenshots. The web variant covers an
// MCP server started from the web directory.
var mcpOutputPaths = []string{
	".playwright-mcp/page-2026-11-21T10-00-00.yml",
	".playwright-mcp/kinex-mark-recovery-codes.txt",
	"web/.playwright-mcp/export.json",
}

func TestGitignoreCoversPlaywrightMCPOutput(t *testing.T) {
	t.Parallel()
	// Unanchored on purpose, so it matches in every directory.
	if !ignoreLines(t, ".gitignore")[".playwright-mcp/"] {
		t.Fatal(`.gitignore lacks the unanchored ".playwright-mcp/" entry`)
	}

	git, err := exec.LookPath("git")
	if err != nil {
		t.Log("git not found: checked the .gitignore entry only")
		return
	}
	checkIgnore := func(rel string) (bool, error) {
		// --no-index: the paths need not exist and must not be tracked.
		err := exec.Command(git, "-C", repoRoot, "check-ignore", "-q", "--no-index", rel).Run()
		var exitErr *exec.ExitError
		switch {
		case err == nil:
			return true, nil
		case errors.As(err, &exitErr) && exitErr.ExitCode() == 1:
			return false, nil
		default:
			return false, err
		}
	}
	if err := exec.Command(git, "-C", repoRoot, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		t.Log("not a git work tree: checked the .gitignore entry only")
		return
	}
	for _, rel := range mcpOutputPaths {
		ignored, err := checkIgnore(rel)
		if err != nil {
			t.Fatalf("git check-ignore %s: %v", rel, err)
		}
		if !ignored {
			t.Errorf("%s is not ignored by git", rel)
		}
	}
	// Guard against an over-broad pattern.
	for _, rel := range []string{"internal/api/api.go", "web/e2e/support/server.ts", "web/playwright.config.ts"} {
		ignored, err := checkIgnore(rel)
		if err != nil {
			t.Fatalf("git check-ignore %s: %v", rel, err)
		}
		if ignored {
			t.Errorf("%s is ignored by git, want tracked", rel)
		}
	}
}

func TestDockerignoreCoversPlaywrightMCPOutput(t *testing.T) {
	t.Parallel()
	// "**/" matches zero or more directories in .dockerignore, so this keeps
	// the root and the web/ output out of the build context.
	if !ignoreLines(t, ".dockerignore")["**/.playwright-mcp"] {
		t.Fatal(`.dockerignore lacks the "**/.playwright-mcp" entry`)
	}
}
