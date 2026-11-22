package api

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestIconServing(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)
	d := linkTestCreateURL(t, c, "https://icon.example/")
	icon := linkTestIconBlob("image/png", []byte("\x89PNG\r\n\x1a\nicon-bytes"))
	linkTestFetched(t, e, d.ID, "Ikon", icon)
	detail := decode[LinkDetail](t, c.get(linkTestPath(d.ID)))
	if detail.IconURL == nil {
		t.Fatal("no iconUrl after the fetch")
	}
	etag := `"` + icon.SHA256 + `"`

	res := c.get(*detail.IconURL)
	body := expectStatus(t, res, http.StatusOK)
	if string(body) != string(icon.Data) {
		t.Errorf("body = %q", body)
	}
	want := map[string]string{
		"Content-Type":            "image/png",
		"Cache-Control":           "private, max-age=604800",
		"ETag":                    etag,
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'none'; style-src 'unsafe-inline'; sandbox",
		"Content-Disposition":     `inline; filename="icon"`,
		"Content-Length":          strconv.Itoa(len(icon.Data)),
	}
	for k, v := range want {
		if got := res.Header.Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	if n := len(res.Header.Values("Content-Security-Policy")); n != 1 {
		t.Errorf("%d CSP headers, want exactly the icon policy", n)
	}

	t.Run("conditional requests", func(t *testing.T) {
		for _, inm := range []string{etag, "W/" + etag, "*", `"other", ` + etag, ` "x" ,W/` + etag} {
			res := c.withHeader("If-None-Match", inm).get(linkTestPath(d.ID, "/icon"))
			if res.StatusCode != http.StatusNotModified {
				t.Errorf("If-None-Match %s: status %d, want 304", inm, res.StatusCode)
				continue
			}
			if len(readBody(t, res)) != 0 || res.Header.Get("ETag") != etag ||
				res.Header.Get("Cache-Control") != "private, max-age=604800" {
				t.Errorf("304 headers/body = %v", res.Header)
			}
		}
		for _, inm := range []string{`"other"`, icon.SHA256, `W/"x"`} {
			expectStatus(t, c.withHeader("If-None-Match", inm).get(linkTestPath(d.ID, "/icon")), http.StatusOK)
		}
	})

	t.Run("head and cache buster", func(t *testing.T) {
		res := c.do(http.MethodHead, linkTestPath(d.ID, "/icon?v=whatever"), nil)
		if res.StatusCode != http.StatusOK || len(readBody(t, res)) != 0 || res.Header.Get("ETag") != etag {
			t.Errorf("HEAD = %d %v", res.StatusCode, res.Header)
		}
	})

	t.Run("svg keeps the sandbox policy", func(t *testing.T) {
		svg := linkTestCreateURL(t, c, "https://svg.example/")
		blob := linkTestIconBlob("image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`))
		linkTestFetched(t, e, svg.ID, "", blob)
		res := c.get(linkTestPath(svg.ID, "/icon"))
		expectStatus(t, res, http.StatusOK)
		if res.Header.Get("Content-Type") != "image/svg+xml" ||
			res.Header.Get("Content-Security-Policy") != "default-src 'none'; style-src 'unsafe-inline'; sandbox" {
			t.Errorf("svg headers = %v", res.Header)
		}
	})

	t.Run("no icon", func(t *testing.T) {
		plain := linkTestCreateURL(t, c, "https://plain.example/")
		res := c.get(linkTestPath(plain.ID, "/icon"))
		expectError(t, res, http.StatusNotFound, codeNotFound)
		if res.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("404 Cache-Control = %q", res.Header.Get("Cache-Control"))
		}
		expectError(t, c.get("/api/links/abc/icon"), http.StatusBadRequest, codeInvalidID)
		expectError(t, c.get(linkTestPath(424242, "/icon")), http.StatusNotFound, codeNotFound)
	})

	t.Run("url change drops the icon", func(t *testing.T) {
		up := decode[LinkDetail](t, c.patch(linkTestPath(d.ID), map[string]any{"url": "https://moved.example/"}))
		if up.IconURL != nil {
			t.Errorf("iconUrl after url change = %v", *up.IconURL)
		}
		expectError(t, c.get(linkTestPath(d.ID, "/icon")), http.StatusNotFound, codeNotFound)
	})

	t.Run("anonymous", func(t *testing.T) {
		expectError(t, e.client(t).get(linkTestPath(d.ID, "/icon")), http.StatusUnauthorized, codeUnauthenticated)
	})
}
