package schedule

import (
	"context"
	"log/slog"
	"time"
)

// tick is the default scheduled job: it logs the current time so a running
// scheduler visibly proves the pipeline works before real jobs are added.
func tick(ctx context.Context) error {
	slog.Info("schedule tick", "time", time.Now().Format(time.RFC3339))
	return nil
}
