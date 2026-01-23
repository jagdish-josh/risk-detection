package transaction

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	
	
)


type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}
func (r *repository) GetByID(id uuid.UUID) (*Transaction, error) {
	var tx Transaction
	if err := r.db.First(&tx, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *repository) Create(tx *Transaction) error {
	return r.db.Create(tx).Error
}

func (r *repository) UpdateStatusByID(id uuid.UUID, status string) error {
	return r.db.
		Model(&Transaction{}).
		Where("id = ?", id).
		Update("transaction_status", status).Error
}