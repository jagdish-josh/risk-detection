package risk

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TransactionRisk struct {
	TransactionID uuid.UUID `gorm:"type:uuid;primaryKey"`

	RiskScore int    `gorm:"not null"`
	RiskLevel string `gorm:"type:varchar(20);not null"`
	Decision  string `gorm:"type:varchar(10);not null;check:decision IN ('ALLOW','FLAG','BLOCK')"`

	EvaluatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

type UserBehavior struct {
	UserID            uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id"`
	TotalTransactions int64     `gorm:"column:total_transactions;not null;default:0"`

	AvgTransactionAmount float64 `gorm:"column:avg_transaction_amount;type:numeric(18,2);not null;default:0"`
	AmountVarianceAcc    float64 `gorm:"column:amount_variance_acc;type:numeric(18,4);not null;default:0"`
	AmountVariance       float64 `gorm:"column:amount_variance;type:numeric(18,4);not null;default:0"`
	AmountStdDev         float64 `gorm:"column:amount_std_dev;type:numeric(18,2);not null;default:0"`

	RecentAvgAmount    float64 `gorm:"column:recent_avg_amount;type:numeric(18,2);not null;default:0"`
	EMASmoothingFactor float64 `gorm:"column:ema_smoothing_factor;type:numeric(5,4);not null;default:0.1"`

	LastTransactionAmount float64    `gorm:"column:last_transaction_amount;type:numeric(18,2)"`
	LastTransactionTime   *time.Time `gorm:"column:last_transaction_time"`

	HighValueThreshold float64 `gorm:"column:high_value_threshold;type:numeric(18,2)"` //p95

	// ---- Metadata ----
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

type UserSecurity struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	DeviceID  string    `gorm:"type:varchar(255);not null"`
	IPAddress string    `gorm:"type:varchar(45);not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

type DailyAggregate struct {
	UserID    uuid.UUID
	TxnCount  int64
	AvgAmount float64
	P95Amount float64
}

type TransactionRiskRepository interface {
	Create(risk *TransactionRisk) error
	GetRiskByTransactionID(id uuid.UUID) (*TransactionRisk, error)
	GetBehaviorByUserID(ctx context.Context, userID uuid.UUID) (*UserBehavior, error)
	GetDailyTransactionAggregate(ctx context.Context, from time.Time, to time.Time) ([]DailyAggregate, error)
	UpdateBehaviorParams(ctx context.Context, userID uuid.UUID, stdDev float64, p95 float64) error
	UpdateBehaviorPerTransaction(ctx context.Context, behavior *UserBehavior) error
	CreateFirstBehavior(ctx context.Context, behavior *UserBehavior) error
	GetDeviceInfo(ctx context.Context,  userID uuid.UUID)(*UserSecurity, error)
}

type Service interface {
	CalculateRisk(tx interface{}) (*TransactionRisk, error)
}

func (UserBehavior) TableName() string {
	return "user_behavior"
}
func (UserSecurity) TableName() string {
	return "user_security"
}
