package api

import (
	"net/http"
	"strings"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// Admin audit log (5.12).

const (
	// auditDefaultLimit and auditMaxLimit page the audit list (5.4).
	auditDefaultLimit = 50
	auditMaxLimit     = 200
)

// validAuditEventFilter reports whether v is an event filter: an event code
// or a prefix ending in "*" ("auth.*"), lower-case letters, digits, '.',
// '_' only.
func validAuditEventFilter(v string) bool {
	v = strings.TrimSuffix(v, "*")
	if v == "" || len(v) > model.AuditEventMaxLen {
		return false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '.' && c != '_' {
			return false
		}
	}
	return true
}

// adminListAudit is GET /api/admin/audit?event=&actorId=&success=&from=&to=,
// newest first.
func (s *server) adminListAudit(w http.ResponseWriter, r *http.Request) {
	q := newQuery(r)
	filter := model.AuditFilter{
		ActorID: q.id("actorId"),
		Success: q.boolean("success"),
		From:    q.time("from"),
		To:      q.time("to"),
	}
	if v, ok := q.raw("event"); ok {
		if validAuditEventFilter(v) {
			filter.Event = v
		} else {
			q.invalid("event")
		}
	}
	page := q.page(auditDefaultLimit, auditMaxLimit)
	if filter.From != nil && filter.To != nil && filter.To.Before(*filter.From) {
		q.invalid("to")
	}
	if q.writeIfInvalid(w) {
		return
	}
	items, total, err := s.deps.Store.Audit(r.Context(), filter, page)
	if err != nil {
		writeFailure(w, r, s.log, err)
		return
	}
	for i := range items {
		if items[i].Details == nil {
			items[i].Details = map[string]any{}
		}
	}
	writeJSON(w, http.StatusOK, model.NewPage(items, total, page))
}
