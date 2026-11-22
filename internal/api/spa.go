package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// immutableCache is the Cache-Control of hashed build assets and fonts.
const immutableCache = "public, max-age=31536000, immutable"

// spa serves the embedded SvelteKit build (TECHNOLOGY.md 4.7 with the
// fonts/ extension, 6.9).
type spa struct {
	ui    fs.FS
	index []byte
	files http.Handler
}

// newSPA reads index.html into memory; New fails when it is missing.
func newSPA(ui fs.FS) (*spa, error) {
	if ui == nil {
		return nil, errors.New("api: UI file system is required")
	}
	index, err := fs.ReadFile(ui, "index.html")
	if err != nil {
		return nil, fmt.Errorf("api: read index.html: %w", err)
	}
	return &spa{ui: ui, index: index, files: http.FileServerFS(ui)}, nil
}

// ServeHTTP serves an existing file (immutable caching under _app/immutable/
// and fonts/), 404 for a missing _app/ file, and index.html with no-cache
// for every other path and method.
func (h *spa) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if info, err := fs.Stat(h.ui, name); err == nil && !info.IsDir() && name != "index.html" {
		if strings.HasPrefix(name, "_app/immutable/") || strings.HasPrefix(name, "fonts/") {
			w.Header().Set("Cache-Control", immutableCache)
		}
		h.files.ServeHTTP(w, r)
		return
	}
	// A missing versioned file gets 404 rather than the HTML page.
	if strings.HasPrefix(name, "_app/") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(h.index)
}

// inlineScriptHashes returns the CSP source ('sha256-<base64>') of every
// inline <script> element (one without a src attribute) of an HTML
// document, in document order and without repeats (6.9). It uses the
// x/net/html tokenizer only (never html.Parse, see A-close note 1). The
// hashed text is the raw element content with line breaks normalised to
// "\n", which is what the browser hashes after input preprocessing.
func inlineScriptHashes(doc []byte) ([]string, error) {
	return readScriptHashes(bytes.NewReader(doc))
}

// readScriptHashes is inlineScriptHashes over a reader.
func readScriptHashes(rd io.Reader) ([]string, error) {
	z := html.NewTokenizer(rd)
	var hashes []string
	seen := map[string]bool{}
	inInline := false
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if errors.Is(z.Err(), io.EOF) {
				return hashes, nil
			}
			return nil, fmt.Errorf("tokenize index.html: %w", z.Err())
		case html.StartTagToken:
			name, hasAttr := z.TagName()
			inInline = false
			if string(name) == "script" {
				inInline = !hasSrcAttr(z, hasAttr)
			}
		case html.TextToken:
			if !inInline {
				continue
			}
			inInline = false
			body := normalizeNewlines(z.Raw())
			if len(body) == 0 {
				continue
			}
			sum := sha256.Sum256(body)
			src := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
			if !seen[src] {
				seen[src] = true
				hashes = append(hashes, src)
			}
		default:
			inInline = false
		}
	}
}

// hasSrcAttr reports whether the current start tag has a src attribute.
func hasSrcAttr(z *html.Tokenizer, more bool) bool {
	for more {
		var key []byte
		key, _, more = z.TagAttr()
		if string(key) == "src" {
			return true
		}
	}
	return false
}

// normalizeNewlines converts "\r\n" and lone "\r" to "\n" (HTML input
// stream preprocessing).
func normalizeNewlines(b []byte) []byte {
	if !bytes.ContainsRune(b, '\r') {
		return b
	}
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
}
