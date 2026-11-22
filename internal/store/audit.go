package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// AppendAudit writes an audit entry. A zero At means now; names are cut to
// the column limits and nil details are stored as {}.
func (s *Store) AppendAudit(ctx context.Context, e model.AuditEntry) error {
	return appendAuditQ(ctx, s.db, e, s.now())
}

func appendAuditQ(ctx context.Context, q querier, e model.AuditEntry, now time.Time) error {
	at := e.At
	if at.IsZero() {
		at = now
	}
	details := "{}"
	if len(e.Details) > 0 {
		b, err := json.Marshal(e.Details)
		if err != nil {
			return fmt.Errorf("encode audit details: %w", err)
		}
		details = string(b)
	}
	var targetType, targetID any
	targetName := ""
	if e.Target != nil {
		targetType = string(e.Target.Type)
		targetID = nullInt64(e.Target.ID)
		targetName = truncateRunes(e.Target.Name, model.AuditNameMaxLen)
	}
	_, err := q.ExecContext(ctx, `INSERT INTO audit_log (at, event, success, actor_id, actor_name,
			target_type, target_id, target_name, ip, details)
		VALUES (:at, :event, :success, :actor_id, :actor_name, :target_type, :target_id,
			:target_name, :ip, :details)`,
		sql.Named("at", formatTime(at)), sql.Named("event", e.Event),
		sql.Named("success", e.Success), sql.Named("actor_id", nullInt64(e.Actor.ID)),
		sql.Named("actor_name", truncateRunes(e.Actor.Username, model.AuditNameMaxLen)),
		sql.Named("target_type", targetType), sql.Named("target_id", targetID),
		sql.Named("target_name", targetName),
		sql.Named("ip", truncateBytes(e.IP, model.SessionIPMaxLen)), sql.Named("details", details))
	if err != nil {
		return fmt.Errorf("append audit: %w", err)
	}
	return nil
}

// auditFilter is the WHERE clause of the audit list (5.12).
const auditFilter = `
WHERE (:event IS NULL OR event = :event)
  AND (:prefix IS NULL OR substr(event, 1, length(:prefix)) = :prefix)
  AND (:actor IS NULL OR actor_id = :actor)
  AND (:success IS NULL OR success = :success)
  AND (:from IS NULL OR at >= :from)
  AND (:to IS NULL OR at <= :to)`

// Audit returns one page of the audit log, newest first, and the size of the
// filtered set. Event is an exact code or a prefix ending in "*".
func (s *Store) Audit(ctx context.Context, f model.AuditFilter, p model.PageReq) ([]model.AuditEntry, int, error) {
	var event, prefix any
	switch {
	case strings.HasSuffix(f.Event, "*"):
		if pre := strings.TrimSuffix(f.Event, "*"); pre != "" {
			prefix = pre
		}
	case f.Event != "":
		event = f.Event
	}
	var success any
	if f.Success != nil {
		success = *f.Success
	}
	var from, to any
	if f.From != nil {
		from = formatTime(*f.From)
	}
	if f.To != nil {
		to = formatTime(*f.To)
	}
	args := []any{sql.Named("event", event), sql.Named("prefix", prefix),
		sql.Named("actor", nullInt64(f.ActorID)), sql.Named("success", success),
		sql.Named("from", from), sql.Named("to", to),
		sql.Named("limit", p.Limit), sql.Named("offset", p.Offset)}

	rows, err := s.db.QueryContext(ctx, `SELECT id, at, event, success, actor_id, actor_name,
			target_type, target_id, target_name, ip, details, COUNT(*) OVER () AS total
		FROM audit_log`+auditFilter+`
		ORDER BY id DESC
		LIMIT :limit OFFSET :offset`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit: %w", err)
	}
	items := []model.AuditEntry{}
	total := 0
	for rows.Next() {
		e, err := scanAudit(rows, &total)
		if err != nil {
			_ = rows.Close()
			return nil, 0, err
		}
		items = append(items, e)
	}
	if err := closeRows(rows); err != nil {
		return nil, 0, fmt.Errorf("list audit: %w", err)
	}
	if len(items) == 0 {
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log`+auditFilter,
			args...).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count audit: %w", err)
		}
	}
	return items, total, nil
}

func scanAudit(r rowScanner, extra ...any) (model.AuditEntry, error) {
	var e model.AuditEntry
	var at, details string
	var actorID, targetID sql.NullInt64
	var targetType sql.NullString
	var targetName string
	dest := []any{&e.ID, &at, &e.Event, &e.Success, &actorID, &e.Actor.Username,
		&targetType, &targetID, &targetName, &e.IP, &details}
	if err := r.Scan(append(dest, extra...)...); err != nil {
		return model.AuditEntry{}, fmt.Errorf("scan audit: %w", err)
	}
	var err error
	if e.At, err = parseTime(at); err != nil {
		return model.AuditEntry{}, err
	}
	e.Actor.ID = int64Ptr(actorID)
	if targetType.Valid {
		e.Target = &model.AuditTarget{Type: model.AuditTargetType(targetType.String),
			ID: int64Ptr(targetID), Name: targetName}
	}
	e.Details = map[string]any{}
	if err := json.Unmarshal([]byte(details), &e.Details); err != nil {
		return model.AuditEntry{}, fmt.Errorf("decode audit details: %w", err)
	}
	if e.Details == nil {
		e.Details = map[string]any{}
	}
	return e, nil
}
