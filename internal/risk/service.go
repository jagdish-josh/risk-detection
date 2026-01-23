package risk

import (
    "fmt"

)

// TransactionRepository interface - abstraction to avoid circular dependency
type TransactionRepository interface {
   
}

type service struct {
    repo                        TransactionRiskRepository
    transactionRepo             TransactionRepository
}

func NewService(repo TransactionRiskRepository, transactionRepo TransactionRepository) Service {
    return &service{
        repo:            repo,
        transactionRepo: transactionRepo,
    }
}

func (s *service) CalculateRisk(tx interface{}) (*TransactionRisk, error) {
    
    fmt.Printf("%+v\n", tx)

	var result TransactionRisk
	result.Decision = "ALLOW"
	result.RiskScore = 25
	result.RiskLevel = "HIGH"

    
    return &result, nil ///need to change result later
}