package transaction

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"context"
	"time"
	
	
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

func (r *repository) CountTransactionFrequency(
	ctx context.Context,
	userID uuid.UUID,
	duration int32,
) (float64, error) {

	var count int64

	fromTime := time.Now().Add(-time.Duration(duration) * time.Minute)

	err := r.db.WithContext(ctx).
		Model(&Transaction{}).
		Where("user_id = ?", userID).
		Where("created_at >= ?", fromTime).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return float64(count), nil
}

func (r *repository) GetTransactions(
	ctx context.Context,
	userID uuid.UUID,
	offset1 int,
	offset2 int,
) ([]*Transaction, error) {

	var transactions []*Transaction

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset1).
		Limit(offset2).
		Find(&transactions).
		Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *repository) CountTotalTransaction(
	ctx context.Context,
	userID uuid.UUID,
) (int64, error) {

	var count int64

	err := r.db.WithContext(ctx).
		Model(&Transaction{}).
		Where("user_id = ?", userID).
		Count(&count).
		Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

