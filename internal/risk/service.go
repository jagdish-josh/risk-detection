package risk


type service struct {
	repo TransactionRiskRepository
}

func NewService(repo TransactionRiskRepository) Service {
	return &service{repo: repo}
}


func (s *service)CalculateRisk(transactionID string, amount float64) (*TransactionRisk, error){

	return nil, nil//for now returning nil<--------------------

}


