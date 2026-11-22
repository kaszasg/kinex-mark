package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// fontAsset describes one self-hosted variable font (plan 2.2 T12).
type fontAsset struct {
	name   string
	size   int
	sha256 string
}

var fontAssets = []fontAsset{
	{
		name:   "HankenGrotesk-Variable.woff2",
		size:   34532,
		sha256: "bcd9aa547e0e3e3299c8dc0113407a3d3e47c23b2d89d25122d1d36a9e16f74c",
	},
	{
		name:   "GeistMono-Variable.woff2",
		size:   23200,
		sha256: "8165ac18be02933145d5a331d11630c6fb6dcefef4284cbae5ada1771c546786",
	},
}

// The SIL OFL 1.1 files start with the copyright notice of the font project.
var licenseFiles = []struct {
	name      string
	firstLine string
}{
	{
		name:      "OFL-Hanken-Grotesk.txt",
		firstLine: "Copyright 2021 The Hanken Grotesk Project Authors (https://github.com/marcologous/hanken-grotesk)",
	},
	{
		name:      "OFL-Geist-Mono.txt",
		firstLine: "Copyright 2024 The Geist Project Authors (https://github.com/vercel/geist-font.git)",
	},
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func normalizeNewlines(data []byte) string {
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func TestFontFilesMatchSizeAndHash(t *testing.T) {
	t.Parallel()
	for _, font := range fontAssets {
		t.Run(font.name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join("static", "fonts", font.name))
			if err != nil {
				t.Fatalf("read font: %v", err)
			}
			if len(data) != font.size {
				t.Errorf("size = %d bytes, want %d", len(data), font.size)
			}
			if got := sha256Hex(data); got != font.sha256 {
				t.Errorf("sha256 = %s, want %s", got, font.sha256)
			}
			if !bytes.HasPrefix(data, []byte("wOF2")) {
				t.Errorf("file does not start with the WOFF2 signature")
			}
		})
	}
}

func TestFontLicenseFiles(t *testing.T) {
	t.Parallel()
	for _, license := range licenseFiles {
		t.Run(license.name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join("static", "fonts", license.name))
			if err != nil {
				t.Fatalf("read license: %v", err)
			}
			text := normalizeNewlines(data)
			first, _, _ := strings.Cut(text, "\n")
			if first != license.firstLine {
				t.Errorf("first line = %q, want %q", first, license.firstLine)
			}
			for _, want := range []string{
				"SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007",
				"PERMISSION & CONDITIONS",
				"OTHER DEALINGS IN THE FONT SOFTWARE.",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("license text lacks %q", want)
				}
			}
		})
	}
}

func TestRobotsTxtDisallowsEverything(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("static", "robots.txt"))
	if err != nil {
		t.Fatalf("read robots.txt: %v", err)
	}
	if got, want := normalizeNewlines(data), "User-agent: *\nDisallow: /\n"; got != want {
		t.Errorf("robots.txt = %q, want %q", got, want)
	}
}

func TestFilesContainsBuild(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		t.Fatalf("index.html is not embedded (run npm run build in web/ first): %v", err)
	}
	if !bytes.Contains(index, []byte(`<html lang="hu" class="dark">`)) {
		t.Errorf("index.html does not come from src/app.html")
	}

	info, err := fs.Stat(files, "fonts")
	if err != nil || !info.IsDir() {
		t.Fatalf("fonts/ directory is not embedded: %v", err)
	}
	for _, font := range fontAssets {
		data, err := fs.ReadFile(files, "fonts/"+font.name)
		if err != nil {
			t.Errorf("fonts/%s is not embedded: %v", font.name, err)
			continue
		}
		if got := sha256Hex(data); got != font.sha256 {
			t.Errorf("embedded fonts/%s sha256 = %s, want %s", font.name, got, font.sha256)
		}
	}
	for _, license := range licenseFiles {
		if _, err := fs.Stat(files, "fonts/"+license.name); err != nil {
			t.Errorf("fonts/%s is not embedded: %v", license.name, err)
		}
	}
	for _, name := range []string{"favicon.svg", "robots.txt"} {
		if _, err := fs.Stat(files, name); err != nil {
			t.Errorf("%s is not embedded: %v", name, err)
		}
	}
}

// The server hashes every inline script for its CSP (plan 6.9): the fallback page holds
// exactly two, the theme bootstrap from src/app.html and the SvelteKit start script.
func TestIndexHasTwoInlineScripts(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	scriptTag := regexp.MustCompile(`<script\b[^>]*>`)
	inline := 0
	for _, tag := range scriptTag.FindAll(index, -1) {
		if !bytes.Contains(tag, []byte("src=")) {
			inline++
		}
	}
	if inline != 2 {
		t.Errorf("inline scripts = %d, want 2", inline)
	}
	if !bytes.Contains(index, []byte("localStorage.setItem('mode-watcher-mode', 'dark')")) {
		t.Errorf("theme bootstrap script is missing")
	}
}
