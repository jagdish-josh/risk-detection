package risk

import(
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

type TransactionRiskRepository interface {
	Create(risk *TransactionRisk) error
	GetByTransactionID(id uuid.UUID) (*TransactionRisk, error)
}

type Service interface {
	CalculateRisk(transactionID string, amount float64) (*TransactionRisk, error)
}

