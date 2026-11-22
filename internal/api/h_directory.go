package api

import (
	"net/http"
)

// User directory for sharing (5.11).

const (
	// directoryQueryMaxRunes bounds the search text of the directory.
	directoryQueryMaxRunes = 64
	// directoryLimit is the maximum number of directory entries.
	directoryLimit = 50
)

// searchDirectory is GET /api/users/directory?q=: active users other than
// the caller whose username or display name contains q, by username, at
// most directoryLimit.
func (s *server) searchDirectory(w http.ResponseWriter, r *http.Request) {
	q := newQuery(r)
	text := q.text("q", directoryQueryMaxRunes)
	if q.writeIfInvalid(w) {
		return
	}
	items, err := s.deps.Store.Directory(r.Context(), userFrom(r).ID, text, directoryLimit)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeList(w, items)
}
