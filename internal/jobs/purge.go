package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// RunPurge removes expired sessions, challenges, throttle rows and pending
// TOTP secrets, audit rows older than auditRetentionDays and orphan icons
// (7.4, 6.13). Runs never overlap with each other or with an archive run.
func (r *Runner) RunPurge(ctx context.Context) (model.PurgeResult, error) {
	r.maintMu.Lock()
	defer r.maintMu.Unlock()
	start := r.clk.Now()
	res, err := r.purge(ctx)
	if err != nil {
		r.log.Error("purge run failed", "err", err)
		return model.PurgeResult{}, err
	}
	r.log.Info("purge run",
		"sessions", res.Sessions,
		"challenges", res.Challenges,
		"throttle", res.Throttle,
		"pending_totp", res.PendingTOTP,
		"audit", res.Audit,
		"icons", res.Icons,
		"duration", r.clk.Now().Sub(start),
	)
	return res, nil
}

func (r *Runner) purge(ctx context.Context) (model.PurgeResult, error) {
	settings, err := r.st.Settings(ctx)
	if err != nil {
		return model.PurgeResult{}, fmt.Errorf("purge: read settings: %w", err)
	}
	retention := time.Duration(settings.AuditRetentionDays) * 24 * time.Hour
	res, err := r.st.Purge(ctx, retention)
	if err != nil {
		return model.PurgeResult{}, fmt.Errorf("purge: %w", err)
	}
	return res, nil
}
