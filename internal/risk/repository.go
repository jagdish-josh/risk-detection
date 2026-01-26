package risk

import (
	"context"
	"errors"
	"time"

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

func (r *repository) GetRiskByTransactionID(id uuid.UUID) (*TransactionRisk, error) {
	var risk TransactionRisk
	if err := r.db.First(&risk, "transaction_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &risk, nil
}

func (r *repository) GetBehaviorByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (*UserBehavior, error) {

	var behavior UserBehavior

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&behavior).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // new user
		}
		return nil, err
	}

	return &behavior, nil
}
func (r *repository) GetDailyTransactionAggregate(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]DailyAggregate, error) {

	var result []DailyAggregate

	err := r.db.WithContext(ctx).
		Raw(`
			SELECT
				user_id,
				COUNT(*) AS txn_count,
				AVG(amount) AS avg_amount,
				PERCENTILE_CONT(0.95)
					WITHIN GROUP (ORDER BY amount) AS p95_amount
			FROM transactions
			WHERE transaction_time >= ?
			  AND transaction_time < ?
			GROUP BY user_id
		`, from, to).
		Scan(&result).Error

	return result, err
}

func (r *repository) UpdateBehaviorParams(
	ctx context.Context,
	userID uuid.UUID,
	stdDev float64,
	p95 float64,
) error {

	return r.db.WithContext(ctx).
		Table("user_behavior").
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"amount_std_dev":       stdDev,
			"high_value_threshold": p95,
			"updated_at":           time.Now(),
		}).Error
}

func (r *repository) UpdateBehaviorPerTransaction(ctx context.Context, behavior *UserBehavior) error {
	return r.db.WithContext(ctx).
		Table("user_behavior").
		Where("user_id = ?", behavior.UserID).
		Updates(behavior).Error
}

func (r *repository) CreateFirstBehavior(ctx context.Context, behavior *UserBehavior) error {
	return r.db.WithContext(ctx).Create(behavior).Error
}

func (r *repository) GetDeviceInfo(ctx context.Context,  userID uuid.UUID)(*UserSecurity, error){
	var userSecurity UserSecurity
	if err := r.db.First(&userSecurity, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &userSecurity, nil
}

