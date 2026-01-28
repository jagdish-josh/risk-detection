package transaction

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockService is a mock implementation of the Service interface
type MockService struct {
	mock.Mock
}

func (m *MockService) CalculateRiskMatrix(tx *Transaction) (*TransactionRiskResponse, error) {
	args := m.Called(tx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TransactionRiskResponse), args.Error(1)
}

func (m *MockService) GetTransactions(ctx context.Context, userID uuid.UUID, offset int, limit int) ([]*Transaction, int64, error) {
	args := m.Called(ctx, userID, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*Transaction), args.Get(1).(int64), args.Error(2)
}

// Helper function to create a test request
func createTestContext(userID string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/transaction", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	if userID != "" {
		c.Set("user_id", userID)
	}
	return c, w
}

// ============ HandleTransaction Tests ============

func TestHandleTransaction_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userID := uuid.New()
	receiverID := uuid.New()

	req := TransactionRequest{
		TransactionType: "TRANSFER",
		ReceiverID:      &receiverID,
		Amount:          100.50,
		DeviceID:        "device123",
		TransactionTime: time.Now(),
	}

	reqBody, _ := json.Marshal(req)
	c, w := createTestContext(userID.String(), reqBody)

	mockService.On("CalculateRiskMatrix", mock.MatchedBy(func(tx *Transaction) bool {
		return tx.UserID == userID && tx.Amount == 100.50
	})).Return(&TransactionRiskResponse{
		TransactionID: uuid.New(),
		RiskScore:     20,
		RiskLevel:     "LOW",
		Decision:      "ALLOW",
		EvaluatedAt:   time.Now(),
	}, nil)

	handler.HandleTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "risk_result")
	mockService.AssertExpectations(t)
}

func TestHandleTransaction_MissingUserID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	req := TransactionRequest{
		TransactionType: "TRANSFER",
		Amount:          100.50,
		DeviceID:        "device123",
		TransactionTime: time.Now(),
	}

	reqBody, _ := json.Marshal(req)
	c, w := createTestContext("", reqBody) // No user ID

	handler.HandleTransaction(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "user_id not found in context", response["error"])
}

func TestHandleTransaction_InvalidUserIDFormat(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	req := TransactionRequest{
		TransactionType: "TRANSFER",
		Amount:          100.50,
		DeviceID:        "device123",
		TransactionTime: time.Now(),
	}

	reqBody, _ := json.Marshal(req)
	c, w := createTestContext("invalid-uuid", reqBody)

	handler.HandleTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "invalid user id format", response["error"])
}

func TestHandleTransaction_InvalidRequestBody(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userID := uuid.New()
	c, w := createTestContext(userID.String(), []byte(`{"invalid json`))

	handler.HandleTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "invalid request body", response["error"])
}

func TestHandleTransaction_MissingRequiredField_Amount(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userID := uuid.New()
	req := TransactionRequest{
		TransactionType: "TRANSFER",
		DeviceID:        "device123",
		TransactionTime: time.Now(),
	}

	reqBody, _ := json.Marshal(req)
	c, w := createTestContext(userID.String(), reqBody)

	handler.HandleTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleTransaction_InvalidAmount_Zero(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userID := uuid.New()
	req := TransactionRequest{
		TransactionType: "TRANSFER",
		Amount:          0,
		DeviceID:        "device123",
		TransactionTime: time.Now(),
	}

	reqBody, _ := json.Marshal(req)
	c, w := createTestContext(userID.String(), reqBody)

	handler.HandleTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleTransaction_InvalidAmount_Negative(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userID := uuid.New()
	req := TransactionRequest{
		TransactionType: "TRANSFER",
		Amount:          -100.50,
		DeviceID:        "device123",
		TransactionTime: time.Now(),
	}

	reqBody, _ := json.Marshal(req)
	c, w := createTestContext(userID.String(), reqBody)

	handler.HandleTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleTransaction_ServiceError(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userID := uuid.New()
	receiverID := uuid.New()

	req := TransactionRequest{
		TransactionType: "TRANSFER",
		ReceiverID:      &receiverID,
		Amount:          100.50,
		DeviceID:        "device123",
		TransactionTime: time.Now(),
	}

	reqBody, _ := json.Marshal(req)
	c, w := createTestContext(userID.String(), reqBody)

	mockService.On("CalculateRiskMatrix", mock.Anything).Return(nil, assert.AnError)

	handler.HandleTransaction(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "error")
	mockService.AssertExpectations(t)
}

// ============ GetTransactions Tests ============

func TestGetTransactions_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userID := uuid.New()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/transactions?offset=0&limit=10", nil)

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", userID.String())

	transactions := []*Transaction{
		{
			ID:     uuid.New(),
			UserID: userID,
			Amount: 100.00,
		},
	}

	mockService.On("GetTransactions", mock.Anything, userID, 0, 10).Return(transactions, int64(1), nil)

	handler.GetTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "data")
	assert.Contains(t, response, "total")
	mockService.AssertExpectations(t)
}

func TestGetTransactions_MissingUserID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/transactions", nil)

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.GetTransactions(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetTransactions_InvalidUserID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/transactions", nil)

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", "invalid-uuid")

	handler.GetTransactions(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNewHandler_NotNil(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	assert.NotNil(t, handler)
}
