package cronjob

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type BehaviorUpdater interface {
	UpdateDailyBehavior(ctx context.Context, day time.Time) error
}

func StartBehaviorCron(
	ctx context.Context,
	updater BehaviorUpdater,
) {

	c := cron.New(cron.WithLocation(time.UTC))

	// Runs every day at 01:00 UTC
	_, err := c.AddFunc("0 1 * * *", func() {
		day := time.Now().UTC().AddDate(0, 0, -1)

		if err := updater.UpdateDailyBehavior(ctx, day); err != nil {
			log.Printf("[RISK][CRON] behavior update failed: %v", err)
		}
	})

	if err != nil {
		log.Fatalf("failed to start behavior cron: %v", err)
	}

	c.Start()
}
