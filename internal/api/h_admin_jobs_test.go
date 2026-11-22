package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/kaszasg/kinex-mark/internal/model"
)

func TestAdminJobsRunArchive(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := e.loginAs(t, model.RoleAdmin)
	e.jobs.mu.Lock()
	e.jobs.archiveCount = 7
	e.jobs.mu.Unlock()
	res := c.post("/api/admin/jobs/archive", nil)
	assertJSON(t, expectStatus(t, res, http.StatusOK), `{"archived":7}`)
	e.jobs.mu.Lock()
	calls := e.jobs.archiveCalls
	e.jobs.archiveErr = errors.New("archive failed")
	e.jobs.mu.Unlock()
	if calls != 1 {
		t.Errorf("RunArchive calls %d", calls)
	}
	expectError(t, c.post("/api/admin/jobs/archive", nil), http.StatusInternalServerError, codeInternalError)
	if e.logs.count("archive failed") == 0 {
		t.Errorf("failure not logged:\n%s", e.logs)
	}

	dc, _ := e.loginAs(t, model.RoleDeveloper)
	expectError(t, dc.post("/api/admin/jobs/archive", nil), http.StatusForbidden, codeForbidden)
}
