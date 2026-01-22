package risk

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)


type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) TransactionRiskRepository {
	return &repository{db: db}
}


func (r *repository) Create(risk *TransactionRisk) error {
	return r.db.Create(risk).Error
}

func (r *repository) GetByTransactionID(id uuid.UUID) (*TransactionRisk, error) {
	var risk TransactionRisk
	if err := r.db.First(&risk, "transaction_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &risk, nil
}
