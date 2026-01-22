package transaction

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID              uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index"`
	TransactionType string    `gorm:"type:varchar(20);not null"`
	ReceiverID      *uuid.UUID `gorm:"type:uuid"`
	Amount          float64   `gorm:"type:numeric(12,2);not null"`

	DeviceID        string    `gorm:"type:varchar(255);not null"`
	IPAddress       string    `gorm:"type:varchar(45);not null"`

	Status string `gorm:"type:varchar(20);not null;check:status IN ('PENDING','COMPLETED','FLAGGED','BLOCKED')"`

	TransactionTime time.Time `gorm:"type:timestamptz;not null"`
	CreatedAt       time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt       time.Time `gorm:"type:timestamptz;not null;default:now()"`

}

type TransactionRiskResponse struct {
	TransactionID uuid.UUID `json:"transaction_id"`

	RiskScore int    `json:"risk_score"`
	RiskLevel string `json:"risk_level"` 
	Decision  string `json:"decision"`   

	EvaluatedAt time.Time `json:"evaluated_at"`
}

type Repository interface {
	GetByID(id uuid.UUID) (*Transaction, error)
	Create(tx *Transaction) error
	UpdateStatusByID(id uuid.UUID, status string) error
	
}

type Service interface {
	CalculateRiskMatrix(tx *Transaction) (*TransactionRiskResponse, error)

}

