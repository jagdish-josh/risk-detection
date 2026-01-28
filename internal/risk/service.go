package risk

import (
	"context"
	"errors"
	"log"
	"math"
	"reflect"
	"risk-detection/internal/audit"
	"sync"
	"time"
	"runtime/debug"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TransactionRepository interface - abstraction to avoid circular dependency
type TransactionRepository interface {
	CountTransactionFrequency(tx context.Context, userID uuid.UUID, duration int32) (float64, error)
}

type service struct {
	repo            TransactionRiskRepository
	transactionRepo TransactionRepository
	rules           map[string]RiskRule
	mu              sync.RWMutex
	auditLog        *audit.Logger
}

func NewService(repo TransactionRiskRepository, transactionRepo TransactionRepository, auditLog *audit.Logger) (Service, error) {

	s := &service{
		repo:            repo,
		transactionRepo: transactionRepo,
		auditLog:        auditLog,
		rules:           make(map[string]RiskRule),
	}
	if err := s.ReloadRules(context.Background()); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *service) ReloadRules(ctx context.Context) error {
	rules, err := s.repo.GetEnabledRules(ctx)
	if err != nil {
		return err
	}

	m := make(map[string]RiskRule)
	for _, r := range rules {
		m[r.Name] = r
	}

	s.mu.Lock()
	s.rules = m
	s.mu.Unlock()

	return nil
}

func (s *service) CalculateRisk(tx interface{}) (*TransactionRisk, error) {

	txdto, err := ExtractTxContext(tx)
	if err != nil {
		log.Printf("unable to extract transaction context: %v", err)
		return nil, err
	}

	var result TransactionRisk
	result.TransactionID = txdto.TxID

	// Calculate risk score
	riskScore1, err := s.transactionAmountRisk(context.Background(), txdto.UserID, txdto.Amount, txdto.TxTime)
	if err != nil {
		return nil, err
	}

	riskScore2, err := s.transactionDeviceRisk(context.Background(), txdto.UserID, txdto.DeviceID, txdto.IPAddress)
	if err != nil {
		return nil, err
	}
	riskScore3, err := s.transactionFrequencyRisk(context.Background(), txdto.UserID)

	if err != nil {
		return nil, err
	}
	totalRisk := 0

	if rule, ok := s.getRule("TRANSACTION_AMOUNT_RISK"); ok {
		totalRisk += applyRule(int(riskScore1), rule)
	}

	if rule, ok := s.getRule("NEW_DEVICE_RISK"); ok {
		totalRisk += applyRule(int(riskScore2), rule)
	}

	if rule, ok := s.getRule("TRANSACTION_FREQUENCY_RISK"); ok {
		totalRisk += applyRule(int(riskScore3), rule)
	}

	result.RiskScore = int(totalRisk)
	result.RiskLevel = calculateRiskLevel(result.RiskScore)
	result.Decision = riskDesion(result.RiskScore)
	result.EvaluatedAt = time.Now()

	if s.repo.Create(&result) != nil {
		log.Printf("unable to save risk matrix")

	}
	s.auditLog.Log(audit.AuditLog{
		EventType:  audit.EventRiskEvaluated,
		Action:     "EVALUATE",
		EntityType: "risk_evaluations",
		EntityID:   result.TransactionID.String(),
		ActorType:  "SYSTEM",
		RiskScore:  &result.RiskScore,
		RiskLevel:  &result.RiskLevel,
		Decision:   &result.Decision,
		Status:     "SUCCESS",
	})

	// Fetch behavior to update it
	behavior, err := s.repo.GetBehaviorByUserID(context.Background(), txdto.UserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// If behavior doesn't exist, create it
	if behavior == nil {
		if err := s.CreateUserBehavior(context.Background(), txdto.UserID); err != nil {
			log.Printf("failed to create user behavior: %v", err)
			return &result, nil // Return result even if behavior creation fails
		}
		// Fetch the newly created behavior
		behavior, err = s.repo.GetBehaviorByUserID(context.Background(), txdto.UserID)
		if err != nil {
			log.Printf("unable to fetch behavior after creation: %v", err)
			return &result, nil // Return result even if fetch fails
		}
	}

	// Update behavior after transaction only if behavior exists
	if behavior != nil {
		if err := s.UpdateUserBehaviorAfterTransaction(context.Background(), behavior, txdto.Amount, txdto.TxID, txdto.TxTime); err != nil {
			log.Printf("unable to update user behavior: %v", err)
			// Continue even if behavior update fails - risk already calculated
		}
	}

	return &result, nil
}

func ExtractTxContext(tx any) (TransactionDTO, error) {
	var dto TransactionDTO

	if tx == nil {
		return dto, errors.New("nil transaction input")
	}

	val := reflect.ValueOf(tx)
	if !val.IsValid() {
		return dto, errors.New("nil transaction")
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return dto, errors.New("nil transaction pointer")
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return dto, errors.New("transaction must be a struct")
	}

	if f := val.FieldByName("ID"); f.IsValid() && f.CanInterface() {
		dto.TxID, _ = f.Interface().(uuid.UUID)
	}

	if f := val.FieldByName("UserID"); f.IsValid() && f.CanInterface() {
		dto.UserID, _ = f.Interface().(uuid.UUID)
	}

	if f := val.FieldByName("Amount"); f.IsValid() && f.CanInterface() {
		dto.Amount, _ = f.Interface().(float64)
	}

	if f := val.FieldByName("TransactionTime"); f.IsValid() && f.CanInterface() {
		dto.TxTime, _ = f.Interface().(time.Time)
	}

	if f := val.FieldByName("DeviceID"); f.IsValid() && f.CanInterface() {
		dto.DeviceID, _ = f.Interface().(string)
	}

	if f := val.FieldByName("IpAdress"); f.IsValid() && f.CanInterface() {
		dto.IPAddress, _ = f.Interface().(string)
	}

	return dto, nil
}

func applyRule(rawScore int, rule RiskRule) int {
	if !rule.Enabled {
		return 0
	}

	return (rule.Weight * rawScore) / 100
}
func (s *service) getRule(name string) (RiskRule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.rules[name]
	return r, ok
}
func calculateRiskLevel(riskScore int) string {
	if riskScore <= 30 {
		return "LOW"
	} else if riskScore <= 70 {
		return "MEDIUM"
	}
	return "HIGH"
}
func riskDesion(riskScore int) string {
	if riskScore <= 30 {
		return "ALLOW"
	} else if riskScore <= 70 {
		return "FLAG"
	}
	return "BLOCK"
}

func safeGo(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC][transactionAmountRisk][GOROUTINE] %v\n%s",
					r, debug.Stack())
			}
		}()
		fn()
	}()
}

func (s *service) transactionAmountRisk(
	ctx context.Context,
	userID uuid.UUID,
	amount float64,
	txTime time.Time,
) (score int32, err error) {

	// ---- Top-level panic protection ----
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC][transactionAmountRisk][TOP] %v\n%s",
				r, debug.Stack())
			score = 50 // safe fallback score
			err = nil
		}
	}()

	log.Println("transaction amount risk triggered")

	behavior, err := s.repo.GetBehaviorByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = s.CreateUserBehavior(context.Background(), userID)
			return 20, nil
		}
		return 0, err
	}

	// New / empty user
	if behavior == nil || behavior.TotalTransactions == 0 {
		return 20, nil
	}

	var (
		riskScore int32
		wg        sync.WaitGroup
		mu        sync.Mutex
	)

	// ---- Rule 1: Relative Amount (avg) ----
	safeGo(&wg, func() {
		if behavior.AvgTransactionAmount <= 0 {
			return
		}
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
	})

	// ---- Rule 2: Z-score ----
	safeGo(&wg, func() {
		if behavior.AmountStdDev <= 0 {
			return
		}
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
	})

	// ---- Rule 3: EMA deviation ----
	safeGo(&wg, func() {
		if behavior.RecentAvgAmount <= 0 {
			return
		}
		if amount/behavior.RecentAvgAmount > 4 {
			mu.Lock()
			riskScore += 10
			mu.Unlock()
		}
	})

	// ---- Rule 4: Sudden jump (velocity) ----
	safeGo(&wg, func() {
		if behavior.LastTransactionAmount <= 0 {
			return
		}
		velocity :=
			(amount - behavior.LastTransactionAmount) /
				behavior.LastTransactionAmount
		if velocity > 3 {
			mu.Lock()
			riskScore += 20
			mu.Unlock()
		}
	})

	// ---- Rule 5: High value boundary (p95) ----
	safeGo(&wg, func() {
		if behavior.HighValueThreshold <= 0 {
			return
		}
		if amount > 2*behavior.HighValueThreshold {
			mu.Lock()
			riskScore += 30
			mu.Unlock()
		} else if amount > behavior.HighValueThreshold {
			mu.Lock()
			riskScore += 20
			mu.Unlock()
		}
	})

	// ---- Rule 6: Back-to-back transactions ----
	safeGo(&wg, func() {
		if behavior.LastTransactionTime == nil {
			return
		}
		if txTime.Sub(*behavior.LastTransactionTime).Seconds() < 30 {
			mu.Lock()
			riskScore += 20
			mu.Unlock()
		}
	})

	wg.Wait()

	// ---- Clamp final score ----
	if riskScore > 100 {
		riskScore = 100
	}

	return riskScore, nil
}

func (s *service) transactionDeviceRisk(ctx context.Context, userID uuid.UUID, txDeviceID string, txIpAddress string) (int32, error) {
	log.Println("device security risk triggered")

	// Return 0 if no device ID provided
	if txDeviceID == "" {
		return 0, nil
	}

	deviceInfo, err := s.repo.GetDeviceInfo(context.Background(), userID)
	if err != nil {
		log.Printf("unable to get device information: %v", err)
		// Return moderate risk if device info not found
		return 20, nil
	}

	// Nil check for deviceInfo
	if deviceInfo == nil {
		return 20, nil
	}

	deviceID := deviceInfo.DeviceID

	if deviceID == txDeviceID {
		return 0, nil
	}
	return 100, nil
}

func (s *service) transactionFrequencyRisk(ctx context.Context, userID uuid.UUID) (float64, error) {
	log.Println("high frequency in short duration risk triggered")

	// Check if transaction repo is nil
	if s.transactionRepo == nil {
		log.Printf("transaction repository is nil, skipping frequency risk check")
		return 0, nil
	}

	count, err := s.transactionRepo.CountTransactionFrequency(ctx, userID, 5)
	if err != nil {
		log.Printf("unable to count frequency: %v", err)
		return 0, nil
	}
	if count == 0 {
		return 90, nil
	}

	// Convert int64 to float64 and apply risk calculation
	riskScore := float64(count-1) * 20

	if riskScore > 100 {
		riskScore = 100
	}
	return riskScore, nil
}

func (s *service) UpdateUserBehaviorAfterTransaction(ctx context.Context, behavior *UserBehavior, amount float64, txID uuid.UUID, txTime time.Time) error {

	// ---------- Capture OLD values for audit ----------
	oldValues := map[string]interface{}{
		"total_transactions":      behavior.TotalTransactions,
		"avg_transaction_amount":  behavior.AvgTransactionAmount,
		"amount_variance":         behavior.AmountVariance,
		"amount_std_dev":          behavior.AmountStdDev,
		"recent_avg_amount":       behavior.RecentAvgAmount,
		"high_value_threshold":    behavior.HighValueThreshold,
		"last_transaction_amount": behavior.LastTransactionAmount,
		"last_transaction_time":   behavior.LastTransactionTime,
	}

	// ---------- 1. Increment transaction count ----------
	behavior.TotalTransactions++

	// ---------- 2. Update Average (incremental mean) ----------
	delta := amount - behavior.AvgTransactionAmount
	behavior.AvgTransactionAmount += delta / float64(behavior.TotalTransactions)

	// ---------- 3. Update Variance (Welford’s algorithm) ----------
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

	// ---------- 5. Update High Value Threshold ----------
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

	// ---------- Persist ----------
	if err := s.repo.UpdateBehaviorPerTransaction(ctx, behavior); err != nil {
		log.Printf("unable to update behavior parameter: %v", err)
		return err
	}

	// ---------- Capture NEW values for audit ----------
	newValues := map[string]interface{}{
		"total_transactions":      behavior.TotalTransactions,
		"avg_transaction_amount":  behavior.AvgTransactionAmount,
		"amount_variance":         behavior.AmountVariance,
		"amount_std_dev":          behavior.AmountStdDev,
		"recent_avg_amount":       behavior.RecentAvgAmount,
		"high_value_threshold":    behavior.HighValueThreshold,
		"last_transaction_amount": behavior.LastTransactionAmount,
		"last_transaction_time":   behavior.LastTransactionTime,
	}

	// ---------- Audit log ----------
	s.auditLog.Log(audit.AuditLog{
		EventType:     audit.EventUserBehaviorUpdated,
		Action:        "UPDATE",
		EntityType:    "user_behavior",
		EntityID:      behavior.UserID.String(),
		ActorType:     "SYSTEM",
		TransactionID: txID.String(),
		OldValues:     oldValues,
		NewValues:     newValues,
		Status:        "SUCCESS",
	})

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
	s.auditLog.Log(audit.AuditLog{

		EventType:  audit.EventUserBehaviorCreated,
		Action:     "CREATE",
		EntityType: "user_behavior",
		EntityID:   userID.String(),
		ActorType:  "SYSTEM",
		NewValues: map[string]interface{}{
			"total_transactions":   1,
			"ema_smoothing_factor": 0.1,
		},
	})
	return nil
}
