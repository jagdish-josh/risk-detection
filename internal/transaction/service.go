package transaction

import (
    "fmt"
    "risk-detection/internal/risk"

)

type service struct {
    repo        Repository
    riskService risk.Service
}

func NewService(repo Repository, riskService risk.Service) Service {
    return &service{
        repo:        repo,
        riskService: riskService,
    }
}

func (s *service) CalculateRiskMatrix(tx *Transaction) (*TransactionRiskResponse, error) {
    // Step 1: Save transaction to database
    if err := s.repo.Create(tx); err != nil {
        return nil, fmt.Errorf("failed to create transaction: %w", err)
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