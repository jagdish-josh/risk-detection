package transaction

import (
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
	// TODO: Implement risk matrix calculation using riskService
	return nil, nil
}
