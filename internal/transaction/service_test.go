package transaction

import (
	"context"
	"errors"
	"testing"

	"risk-detection/internal/risk"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetByID(id uuid.UUID) (*Transaction, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Transaction), args.Error(1)
}

func (m *MockRepository) Create(tx *Transaction) error {
	args := m.Called(tx)
	return args.Error(0)
}

func (m *MockRepository) UpdateStatusByID(id uuid.UUID, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockRepository) CountTransactionFrequency(ctx context.Context, userID uuid.UUID, duration int32) (float64, error) {
	args := m.Called(ctx, userID, duration)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRepository) GetTransactions(ctx context.Context, userID uuid.UUID, offset int, limit int) ([]*Transaction, error) {
	args := m.Called(ctx, userID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Transaction), args.Error(1)
}

func (m *MockRepository) CountTotalTransaction(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// MockRiskService is a mock implementation of the risk.Service interface
type MockRiskService struct {
	mock.Mock
}

func (m *MockRiskService) CalculateRisk(tx interface{}) (*risk.TransactionRisk, error) {
	args := m.Called(tx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*risk.TransactionRisk), args.Error(1)
}
//========== GetTransactions Tests ============

func TestService_GetTransactions_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil)

	userID := uuid.New()
	ctx := context.Background()

	transactions := []*Transaction{
		{ID: uuid.New(), UserID: userID, Amount: 100.00},
		{ID: uuid.New(), UserID: userID, Amount: 200.00},
	}

	mockRepo.On("GetTransactions", ctx, userID, 0, 10).Return(transactions, nil)
	mockRepo.On("CountTotalTransaction", ctx, userID).Return(int64(2), nil)

	result, total, err := service.GetTransactions(ctx, userID, 0, 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(2), total)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestGetTransactions_EmptyResult(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil)

	userID := uuid.New()
	ctx := context.Background()

	mockRepo.On("GetTransactions", ctx, userID, 0, 10).Return([]*Transaction{}, nil)
	mockRepo.On("CountTotalTransaction", ctx, userID).Return(int64(0), nil)

	result, total, err := service.GetTransactions(ctx, userID, 0, 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), total)
	assert.Len(t, result, 0)
}

func TestGetTransactions_QueryError(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil)

	userID := uuid.New()
	ctx := context.Background()

	mockRepo.On("GetTransactions", ctx, userID, 0, 10).Return(nil, errors.New("database connection lost"))

	result, total, err := service.GetTransactions(ctx, userID, 0, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(0), total)
}

func TestGetTransactions_CountError(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil)

	userID := uuid.New()
	ctx := context.Background()

	transactions := []*Transaction{{ID: uuid.New(), UserID: userID}}

	mockRepo.On("GetTransactions", ctx, userID, 0, 10).Return(transactions, nil)
	mockRepo.On("CountTotalTransaction", ctx, userID).Return(int64(0), errors.New("count query failed"))

	result, total, err := service.GetTransactions(ctx, userID, 0, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(0), total)
}

func TestGetTransactions_NegativeOffsetAdjustment(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil)

	userID := uuid.New()
	ctx := context.Background()

	// Expect offset to be corrected to 0
	mockRepo.On("GetTransactions", ctx, userID, 0, 10).Return([]*Transaction{}, nil)
	mockRepo.On("CountTotalTransaction", ctx, userID).Return(int64(0), nil)

	_, _, err := service.GetTransactions(ctx, userID, -5, 10)

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "GetTransactions", ctx, userID, 0, 10)
}

func TestGetTransactions_NegativeLimitAdjustment(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil)

	userID := uuid.New()
	ctx := context.Background()

	// Expect limit to be corrected to 10
	mockRepo.On("GetTransactions", ctx, userID, 0, 10).Return([]*Transaction{}, nil)
	mockRepo.On("CountTotalTransaction", ctx, userID).Return(int64(0), nil)

	_, _, err := service.GetTransactions(ctx, userID, 0, -5)

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "GetTransactions", ctx, userID, 0, 10)
}

func TestGetTransactions_ZeroLimitAdjustment(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil)

	userID := uuid.New()
	ctx := context.Background()

	// Expect limit to be corrected to 10
	mockRepo.On("GetTransactions", ctx, userID, 0, 10).Return([]*Transaction{}, nil)
	mockRepo.On("CountTotalTransaction", ctx, userID).Return(int64(0), nil)

	_, _, err := service.GetTransactions(ctx, userID, 0, 0)

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "GetTransactions", ctx, userID, 0, 10)
}

// ============ MapDecisionToStatus Tests ============

func TestMapDecisionToStatus_Allow(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil).(*service)

	status := service.mapDecisionToStatus("ALLOW")
	assert.Equal(t, "COMPLETED", status)
}

func TestMapDecisionToStatus_Flag(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil).(*service)

	status := service.mapDecisionToStatus("FLAG")
	assert.Equal(t, "FLAGGED", status)
}

func TestMapDecisionToStatus_Block(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil).(*service)

	status := service.mapDecisionToStatus("BLOCK")
	assert.Equal(t, "BLOCKED", status)
}

func TestMapDecisionToStatus_Unknown(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil).(*service)

	status := service.mapDecisionToStatus("UNKNOWN_DECISION")
	assert.Equal(t, "PENDING", status)
}

func TestMapDecisionToStatus_Empty(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil).(*service)

	status := service.mapDecisionToStatus("")
	assert.Equal(t, "PENDING", status)
}

func TestMapDecisionToStatus_CaseSensitive(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil).(*service)

	// Lowercase should not match - should return PENDING
	status := service.mapDecisionToStatus("allow")
	assert.Equal(t, "PENDING", status)
}

// ============ Service Initialization Tests ============

func TestNewService_NotNil(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRiskService := new(MockRiskService)

	service := NewService(mockRepo, mockRiskService, nil)

	assert.NotNil(t, service)
}
