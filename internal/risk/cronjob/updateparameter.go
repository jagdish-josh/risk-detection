package cronjob

import (
	"context"
	"log"
	"math"
	"time"

	// "github.com/google/uuid"
	"risk-detection/internal/audit"
	"risk-detection/internal/risk"
)

const (
	// decay factor approximates 6-month rolling window
	varianceDecay = 0.995
)

type ParameterUpdater struct {
	repo     risk.TransactionRiskRepository
	auditLog *audit.Logger
}

func NewParameterUpdater(repo risk.TransactionRiskRepository, auditLog *audit.Logger) *ParameterUpdater {
	return &ParameterUpdater{
		repo:     repo,
		auditLog: auditLog,
	}
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
		log.Println("update daily trigger")
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

			p.auditLog.Log(audit.AuditLog{
				EventType:  audit.EventUserBehaviorUpdated,
				Action:     "UPDATE",
				EntityType: "user_behavior",
				EntityID:   r.UserID.String(),
				ActorType:  "SYSTEM",
				Status:     "FAILURE",
			})
		}
		newValues := map[string]interface{}{
			"amount_std_dev":       stdDev,
			"high_value_treshould": r.P95Amount,
		}
		p.auditLog.Log(audit.AuditLog{
			EventType:  audit.EventUserBehaviorUpdated,
			Action:     "UPDATE",
			EntityType: "user_behavior",
			EntityID:   r.UserID.String(),
			ActorType:  "SYSTEM",
			NewValues:  newValues,
			Status:     "SUCCESS",
		})
	}

	return nil
}
