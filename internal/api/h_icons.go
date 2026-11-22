package api

import (
	"net/http"
	"strconv"
	"strings"
)

// Response headers of the icon endpoint (5.9.9). The CSP replaces the page
// policy: an SVG icon opened directly runs no script and loads nothing.
const (
	iconCacheControl = "private, max-age=604800"
	iconCSP          = "default-src 'none'; style-src 'unsafe-inline'; sandbox"
	iconDisposition  = `inline; filename="icon"`
)

// linkIcon answers GET /api/links/{id}/icon: the stored icon of a visible
// link with its own cache and security headers; 304 when If-None-Match
// carries the current ETag. The ?v= parameter is only a cache buster.
func (s *server) linkIcon(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	icon, err := s.deps.Store.LinkIcon(r.Context(), viewerFrom(r), id)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	etag := `"` + icon.SHA256 + `"`
	h := w.Header()
	h.Set("Cache-Control", iconCacheControl)
	h.Set("ETag", etag)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", iconCSP)
	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Type", icon.ContentType)
	h.Set("Content-Disposition", iconDisposition)
	h.Set("Content-Length", strconv.Itoa(len(icon.Data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(icon.Data)
}

// etagMatches reports whether an If-None-Match header value matches etag:
// "*" or any listed tag, weak tags compared by their opaque part (RFC 9110
// 13.1.2 weak comparison).
func etagMatches(header, etag string) bool {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "*" || strings.TrimPrefix(part, "W/") == etag {
			return true
		}
	}
	return false
}
