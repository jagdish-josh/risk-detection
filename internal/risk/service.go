package risk

import (
	"context"
	"errors"
	"log"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TransactionRepository interface - abstraction to avoid circular dependency
type TransactionRepository interface {
	GetByID(id uuid.UUID) (interface{}, error)
}

type service struct {
	repo            TransactionRiskRepository
	transactionRepo TransactionRepository
}

func NewService(repo TransactionRiskRepository, transactionRepo TransactionRepository) Service {
	return &service{
		repo:            repo,
		transactionRepo: transactionRepo,
	}
}

func (s *service) CalculateRisk(tx interface{}) (*TransactionRisk, error) {


	// Use reflection to extract transaction fields
	val := reflect.ValueOf(tx)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var userID uuid.UUID
	var amount float64
	var txTime time.Time
	var txID uuid.UUID

	// Extract UserID
	if userIDField := val.FieldByName("UserID"); userIDField.IsValid() {
		userID = userIDField.Interface().(uuid.UUID)
	}

	// Extract Amount
	if amountField := val.FieldByName("Amount"); amountField.IsValid() {
		amount = amountField.Interface().(float64)
	}

	// Extract TransactionTime
	if timeField := val.FieldByName("TransactionTime"); timeField.IsValid() {
		txTime = timeField.Interface().(time.Time)
	}

	// Extract ID
	if idField := val.FieldByName("ID"); idField.IsValid() {
		txID = idField.Interface().(uuid.UUID)
	}

	var result TransactionRisk
	result.TransactionID = txID
	result.Decision = "ALLOW"

	// Calculate risk score
	riskScore, err := s.transactionAmountRisk(context.Background(), userID, amount, txTime)
	if err != nil {
		return nil, err
	}

	// Fetch behavior to update it
	behavior, err := s.repo.GetBehaviorByUserID(context.Background(), userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// If behavior doesn't exist, create it
	if behavior == nil {
		if err := s.CreateUserBehavior(context.Background(), userID); err != nil {
			log.Printf("failed to create user behavior: %v", err)
			return nil, err
		}
		// Fetch the newly created behavior
		behavior, err = s.repo.GetBehaviorByUserID(context.Background(), userID)
		if err != nil {
			return nil, err
		}
	}

	// Update behavior after transaction
	if err := s.UpdateUserBehaviorAfterTransaction(context.Background(), behavior, amount, txTime); err != nil {
		log.Printf("unable to update user behavior: %v", err)
		return nil, err
	}

	result.RiskScore = int(riskScore)
	result.RiskLevel = "HIGH"
	result.EvaluatedAt = time.Now()

	return &result, nil
}

func (s *service) transactionAmountRisk(
	ctx context.Context,
	userID uuid.UUID,
	amount float64,
	txTime time.Time,
) (int32, error) {

	behavior, err := s.repo.GetBehaviorByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.CreateUserBehavior(context.Background(), userID)
			return 20, nil
		}
		return 0, err
	}

	// New user → conservative risk
	if behavior == nil || behavior.TotalTransactions == 0 {
		return 20, nil
	}

	var (
		riskScore int32
		wg        sync.WaitGroup
		mu        sync.Mutex
	)

	// ---- Rule 1: Relative Amount (avg) ----
	wg.Add(1)
	go func() {
		defer wg.Done()

		if behavior.AvgTransactionAmount > 0 {
			ratio := amount / behavior.AvgTransactionAmount

			if ratio > 10 {
				mu.Lock()
				riskScore += 35
				mu.Unlock()
			} else if ratio > 5 {
				mu.Lock()
				riskScore += 20
				mu.Unlock()
			}
		}
	}()

	// ---- Rule 2: Z-score ----
	wg.Add(1)
	go func() {
		defer wg.Done()

		if behavior.AmountStdDev > 0 {
			z := (amount - behavior.AvgTransactionAmount) / behavior.AmountStdDev

			if z > 3 {
				mu.Lock()
				riskScore += 30
				mu.Unlock()
			} else if z > 2 {
				mu.Lock()
				riskScore += 20
				mu.Unlock()
			}
		}
	}()

	// ---- Rule 3: EMA deviation ----
	wg.Add(1)
	go func() {
		defer wg.Done()

		if behavior.RecentAvgAmount > 0 {
			ratio := amount / behavior.RecentAvgAmount

			if ratio > 4 {
				mu.Lock()
				riskScore += 10
				mu.Unlock()
			}
		}
	}()

	// ---- Rule 4: Sudden jump (velocity) ----
	wg.Add(1)
	go func() {
		defer wg.Done()

		if behavior.LastTransactionAmount > 0 {
			velocity :=
				(amount - behavior.LastTransactionAmount) /
					behavior.LastTransactionAmount

			if velocity > 3 {
				mu.Lock()
				riskScore += 20
				mu.Unlock()
			}
		}
	}()

	// ---- Rule 5: High value boundary (p95) ----
	wg.Add(1)
	go func() {
		defer wg.Done()

		if behavior.HighValueThreshold > 0 {
			if amount > 2*behavior.HighValueThreshold {
				mu.Lock()
				riskScore += 30
				mu.Unlock()
			} else if amount > behavior.HighValueThreshold {
				mu.Lock()
				riskScore += 20
				mu.Unlock()
			}
		}
	}()

	// ---- Rule 6: Back-to-back transactions ----
	wg.Add(1)
	go func() {
		defer wg.Done()

		if behavior.LastTransactionTime != nil {
			diff := txTime.Sub(*behavior.LastTransactionTime)

			if diff.Seconds() < 30 {
				mu.Lock()
				riskScore += 20
				mu.Unlock()
			}
		}
	}()

	wg.Wait()

	// ---- Clamp final score ----
	if riskScore > 100 {
		riskScore = 100
	}

	return riskScore, nil
}

func (s *service) UpdateUserBehaviorAfterTransaction(
	ctx context.Context,
	behavior *UserBehavior,
	amount float64,
	txTime time.Time,
) error {

	// ---------- 1. Increment transaction count ----------
	behavior.TotalTransactions++

	// ---------- 2. Update Average (incremental mean) ----------
	// newAvg = oldAvg + (x - oldAvg) / n
	delta := amount - behavior.AvgTransactionAmount
	behavior.AvgTransactionAmount += delta / float64(behavior.TotalTransactions)

	// ---------- 3. Update Variance (Welford’s algorithm) ----------
	// variance_acc += delta * (x - newAvg)
	behavior.AmountVarianceAcc += delta * (amount - behavior.AvgTransactionAmount)

	if behavior.TotalTransactions > 1 {
		behavior.AmountVariance =
			behavior.AmountVarianceAcc / float64(behavior.TotalTransactions-1)

		behavior.AmountStdDev =
			math.Sqrt(behavior.AmountVariance)
	} else {
		behavior.AmountVariance = 0
		behavior.AmountStdDev = 0
	}

	// ---------- 4. Update EMA (recent average) ----------
	alpha := behavior.EMASmoothingFactor
	if behavior.RecentAvgAmount == 0 {
		behavior.RecentAvgAmount = amount
	} else {
		behavior.RecentAvgAmount =
			alpha*amount + (1-alpha)*behavior.RecentAvgAmount
	}

	// ---------- 5. Update High Value Threshold (approx p95) ----------
	// Simple adaptive threshold (good enough for online systems)
	if behavior.HighValueThreshold == 0 {
		behavior.HighValueThreshold = amount
	} else if amount > behavior.HighValueThreshold {
		behavior.HighValueThreshold =
			behavior.HighValueThreshold + 0.05*(amount-behavior.HighValueThreshold)
	}

	// ---------- 6. Update last transaction ----------
	behavior.LastTransactionAmount = amount
	behavior.LastTransactionTime = &txTime
	behavior.UpdatedAt = time.Now()

	// ---------- 7. Persist ----------
	err := s.repo.UpdateBehaviorPerTransaction(ctx, behavior)

	if err != nil {
		log.Printf("unable to update behavior parameter: %v", err)
		return err
	}

	return nil
}

func (s *service) CreateUserBehavior(ctx context.Context, userID uuid.UUID) error {
	behavior := &UserBehavior{
		UserID:                userID,
		TotalTransactions:     0,
		AvgTransactionAmount:  0,
		AmountVarianceAcc:     0,
		AmountVariance:        0,
		AmountStdDev:          0,
		RecentAvgAmount:       0,
		EMASmoothingFactor:    0.1,
		LastTransactionAmount: 0,
		LastTransactionTime:   nil,
		HighValueThreshold:    0,
		UpdatedAt:             time.Now(),
	}

	if err := s.repo.CreateFirstBehavior(ctx, behavior); err != nil {
		log.Printf("unable to create user behavior: %v", err)
		return err
	}

	return nil
}
