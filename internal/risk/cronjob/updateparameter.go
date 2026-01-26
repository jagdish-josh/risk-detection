package cronjob

import (
	"context"
	"log"
	"math"
	"time"

	// "github.com/google/uuid"
	"risk-detection/internal/risk"
)

const (
	// decay factor approximates 6-month rolling window
	varianceDecay = 0.995
)



type ParameterUpdater struct {
	repo risk.TransactionRiskRepository
}

func NewParameterUpdater(repo risk.TransactionRiskRepository) *ParameterUpdater {
	return &ParameterUpdater{repo: repo}
}

func (p *ParameterUpdater) UpdateDailyBehavior(
	ctx context.Context,
	day time.Time,
) error {

	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)

	rows, err := p.repo.GetDailyTransactionAggregate(ctx, from, to)
	if err != nil {
		return err
	}

	for _, r := range rows {

		// Approximate rolling variance using decay
		variance := (r.AvgAmount * r.AvgAmount) * (1 - varianceDecay)
		stdDev := math.Sqrt(variance)

		err := p.repo.UpdateBehaviorParams(
			ctx,
			r.UserID,
			stdDev,
			r.P95Amount,
		)

		if err != nil {
			log.Printf(
				"[RISK][CRON] failed for user %s: %v",
				r.UserID, err,
			)
		}
	}

	return nil
}
