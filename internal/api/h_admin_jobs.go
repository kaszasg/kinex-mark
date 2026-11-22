package api

import "net/http"

// Admin job trigger (5.12).

// archiveResponse is the body of POST /api/admin/jobs/archive.
type archiveResponse struct {
	Archived int `json:"archived"`
}

// adminRunArchive is POST /api/admin/jobs/archive: the automatic archiving
// job runs now (7.2) and the number of archived links is returned.
func (s *server) adminRunArchive(w http.ResponseWriter, r *http.Request) {
	n, err := s.deps.Jobs.RunArchive(r.Context())
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, archiveResponse{Archived: n})
}
