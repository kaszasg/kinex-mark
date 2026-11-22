package jobs

import (
	"context"
	"fmt"
)

// RunArchive archives the inactive links of every active user with the
// defaultArchiveDays system setting as the default period (7.2) and returns
// the number of archived rows. The scheduler and POST /api/admin/jobs/archive
// call it; runs never overlap with each other or with a purge.
func (r *Runner) RunArchive(ctx context.Context) (int, error) {
	r.maintMu.Lock()
	defer r.maintMu.Unlock()
	start := r.clk.Now()
	n, err := r.archive(ctx)
	if err != nil {
		r.log.Error("archive run failed", "err", err)
		return 0, err
	}
	r.log.Info("archive run", "archived", n, "duration", r.clk.Now().Sub(start))
	return n, nil
}

func (r *Runner) archive(ctx context.Context) (int, error) {
	settings, err := r.st.Settings(ctx)
	if err != nil {
		return 0, fmt.Errorf("archive: read settings: %w", err)
	}
	n, err := r.st.ArchiveInactive(ctx, settings.DefaultArchiveDays)
	if err != nil {
		return 0, fmt.Errorf("archive: %w", err)
	}
	return n, nil
}
