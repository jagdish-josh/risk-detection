package transaction

import (
	"context"
	"fmt"

	"risk-detection/internal/audit"
	"risk-detection/internal/risk"

	"github.com/google/uuid"
)

type service struct {
	repo        Repository
	riskService risk.Service
	auditLog    *audit.Logger
}

func NewService(repo Repository, riskService risk.Service, auditLog *audit.Logger) Service {
	return &service{
		repo:        repo,
		riskService: riskService,
		auditLog:    auditLog,
	}
}

func (s *service) CalculateRiskMatrix(tx *Transaction) (*TransactionRiskResponse, error) {
	// Step 1: Save transaction to database
	if err := s.repo.Create(tx); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	if s.auditLog != nil {

		//transaction creation log
		s.auditLog.Log(audit.AuditLog{
			EventType:  audit.EventTransactionCreated,
			ActorID:    tx.ID.String(),
			EntityType: "users",
			EntityID:   tx.UserID.String(),
			Status:     "SUCCESS",
			IPAddress:  tx.IPAddress,
			DeviceID:   tx.DeviceID,
		})
	}

	// Step 2: Calculate risk score from risk service
	riskResult, err := s.riskService.CalculateRisk(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate risk: %w", err)
	}
	if riskResult == nil {
		return nil, fmt.Errorf("risk calculation returned nil result")
	}

	// Step 3: Update transaction status based on risk decision
	newStatus := s.mapDecisionToStatus(riskResult.Decision)
	if err := s.repo.UpdateStatusByID(tx.ID, newStatus); err != nil {
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	if s.auditLog != nil {
	s.auditLog.Log(audit.AuditLog{
		EventType:  audit.EventTransactionUpdated,
		Action:     "UPDATE",
		EntityType: "transactions",
		EntityID:   tx.ID.String(),
		ActorID:    tx.UserID.String(),
		OldValues: map[string]interface{}{
			"Status": newStatus,
		},
		NewValues: map[string]interface{}{
			"Status": "PENDING",
		},
		Status: "SUCCESS",
	})}

	// Step 4: Return formatted risk response to handler
	response := &TransactionRiskResponse{
		TransactionID: tx.ID,
		RiskScore:     riskResult.RiskScore,
		RiskLevel:     riskResult.RiskLevel,
		Decision:      riskResult.Decision,
		EvaluatedAt:   riskResult.EvaluatedAt,
	}

	return response, nil
}
func (s *service) GetTransactions(
	ctx context.Context,
	userID uuid.UUID,
	offset int,
	limit int,
) ([]*Transaction, int64, error) {

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	transactions, err := s.repo.GetTransactions(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountTotalTransaction(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// mapDecisionToStatus converts risk decision to transaction status
func (s *service) mapDecisionToStatus(decision string) string {
	switch decision {
	case "ALLOW":
		return "COMPLETED"
	case "FLAG":
		return "FLAGGED"
	case "BLOCK":
		return "BLOCKED"
	default:
		return "PENDING"
	}
}
