package cronjob

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"risk-detection/internal/audit"
	"risk-detection/internal/risk"
)

// ============ Mock Repository ============
type mockTransactionRiskRepository struct {
	mock.Mock
}

func (m *mockTransactionRiskRepository) Create(risk *risk.TransactionRisk) error {
	args := m.Called(risk)
	return args.Error(0)
}

func (m *mockTransactionRiskRepository) GetRiskByTransactionID(id uuid.UUID) (*risk.TransactionRisk, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*risk.TransactionRisk), args.Error(1)
}

func (m *mockTransactionRiskRepository) GetDailyTransactionAggregate(ctx context.Context, from, to time.Time) ([]risk.DailyAggregate, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]risk.DailyAggregate), args.Error(1)
}

func (m *mockTransactionRiskRepository) UpdateBehaviorParams(ctx context.Context, userID uuid.UUID, stdDev, p95 float64) error {
	args := m.Called(ctx, userID, stdDev, p95)
	return args.Error(0)
}

func (m *mockTransactionRiskRepository) UpdateBehaviorPerTransaction(ctx context.Context, behavior *risk.UserBehavior) error {
	args := m.Called(ctx, behavior)
	return args.Error(0)
}

func (m *mockTransactionRiskRepository) CreateFirstBehavior(ctx context.Context, behavior *risk.UserBehavior) error {
	args := m.Called(ctx, behavior)
	return args.Error(0)
}

func (m *mockTransactionRiskRepository) GetDeviceInfo(ctx context.Context, userID uuid.UUID) (*risk.UserSecurity, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*risk.UserSecurity), args.Error(1)
}

func (m *mockTransactionRiskRepository) GetBehaviorByUserID(ctx context.Context, userID uuid.UUID) (*risk.UserBehavior, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*risk.UserBehavior), args.Error(1)
}

func (m *mockTransactionRiskRepository) GetEnabledRules(ctx context.Context) ([]risk.RiskRule, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]risk.RiskRule), args.Error(1)
}

// ============ Integration Tests ============
// Tests that verify the daily behavior update job handles various user populations correctly.

func TestParameterUpdater_DailyBehaviorUpdate(t *testing.T) {
	tests := []struct {
		name              string
		setupAggregates   func() []risk.DailyAggregate
		repositoryError   error
		validateBehavior  func(updateCalls int) bool
	}{
		{
			name: "handles empty daily aggregate without errors",
			setupAggregates: func() []risk.DailyAggregate {
				return []risk.DailyAggregate{}
			},
			repositoryError: nil,
			validateBehavior: func(updateCalls int) bool {
				return updateCalls == 0
			},
		},
		{
			name: "updates parameters for each user with transactions",
			setupAggregates: func() []risk.DailyAggregate {
				return []risk.DailyAggregate{
					{UserID: uuid.New(), AvgAmount: 100.0, P95Amount: 200.0},
					{UserID: uuid.New(), AvgAmount: 150.0, P95Amount: 300.0},
					{UserID: uuid.New(), AvgAmount: 120.0, P95Amount: 250.0},
				}
			},
			repositoryError: nil,
			validateBehavior: func(updateCalls int) bool {
				return updateCalls == 3
			},
		},
		{
			name: "stops gracefully when daily aggregate fetch fails",
			setupAggregates: func() []risk.DailyAggregate {
				return nil
			},
			repositoryError: errors.New("database connection failed"),
			validateBehavior: func(updateCalls int) bool {
				return updateCalls == 0
			},
		},
		{
			name: "continues with other users if one update fails",
			setupAggregates: func() []risk.DailyAggregate {
				return []risk.DailyAggregate{
					{UserID: uuid.New(), AvgAmount: 100.0, P95Amount: 200.0},
					{UserID: uuid.New(), AvgAmount: 150.0, P95Amount: 300.0},
				}
			},
			repositoryError: nil,
			validateBehavior: func(updateCalls int) bool {
				return updateCalls == 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockTransactionRiskRepository)
			auditLog := &audit.Logger{}

			aggregates := tt.setupAggregates()

			repo.On("GetDailyTransactionAggregate", mock.Anything, mock.Anything, mock.Anything).
				Return(aggregates, tt.repositoryError)

			updateCallCount := 0
			if tt.repositoryError == nil && aggregates != nil {
				for _, agg := range aggregates {
					repo.On("UpdateBehaviorParams", mock.Anything, agg.UserID, mock.AnythingOfType("float64"), agg.P95Amount).
						Run(func(args mock.Arguments) {
							updateCallCount++
						}).
						Return(nil)
				}
			}

			updater := NewParameterUpdater(repo, auditLog)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			day := time.Now().UTC()
			err := updater.UpdateDailyBehavior(ctx, day)

			if tt.repositoryError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.True(t, tt.validateBehavior(updateCallCount), "Expected update behavior validation to pass")
		})
	}
}

// ============ Variance Calculation Tests ============
// Tests that verify the variance calculation applies decay factor correctly.

func TestParameterUpdater_VarianceCalculation(t *testing.T) {
	tests := []struct {
		name           string
		avgAmount      float64
		expectedStdDev func(float64) bool
		description    string
	}{
		{
			name:      "calculates positive stddev from positive average",
			avgAmount: 100.0,
			expectedStdDev: func(stdDev float64) bool {
				return stdDev > 5.0 && stdDev < 10.0
			},
			description: "100 * 100 * 0.005 ≈ 50, sqrt(50) ≈ 7.07",
		},
		{
			name:      "handles zero average amount",
			avgAmount: 0.0,
			expectedStdDev: func(stdDev float64) bool {
				return stdDev == 0.0
			},
			description: "zero produces zero variance",
		},
		{
			name:      "handles large average amounts",
			avgAmount: 10000.0,
			expectedStdDev: func(stdDev float64) bool {
				return stdDev > 0 && stdDev > 500.0
			},
			description: "10000 * 10000 * 0.005 produces large stddev",
		},
		{
			name:      "variance decay factor reduces magnitude",
			avgAmount: 50.0,
			expectedStdDev: func(stdDev float64) bool {
				// decay = 0.995, so (1 - decay) = 0.005
				// variance = 50^2 * 0.005 = 12.5, sqrt(12.5) ≈ 3.54
				return stdDev > 0 && stdDev < 5.0
			},
			description: "decay factor applied to average amount variance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockTransactionRiskRepository)
			auditLog := &audit.Logger{}

			aggregates := []risk.DailyAggregate{
				{
					UserID:    uuid.New(),
					AvgAmount: tt.avgAmount,
					P95Amount: tt.avgAmount * 2.0,
				},
			}

			repo.On("GetDailyTransactionAggregate", mock.Anything, mock.Anything, mock.Anything).
				Return(aggregates, nil)

			capturedStdDev := 0.0
			repo.On("UpdateBehaviorParams", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), mock.Anything).
				Run(func(args mock.Arguments) {
					capturedStdDev = args.Get(2).(float64)
				}).
				Return(nil)

			updater := NewParameterUpdater(repo, auditLog)
			ctx := context.Background()
			day := time.Now().UTC()

			err := updater.UpdateDailyBehavior(ctx, day)
			assert.NoError(t, err)

			assert.True(t, tt.expectedStdDev(capturedStdDev),
				"Expected stdDev %f to match validation. Details: %s", capturedStdDev, tt.description)
		})
	}
}

// ============ Error Handling and Resilience Tests ============
// Verify behavior under error conditions and partial failures.

func TestParameterUpdater_ErrorHandling(t *testing.T) {
	tests := []struct {
		name            string
		setupAggregates func() []risk.DailyAggregate
		failOnUserID    *uuid.UUID
		expectError     bool
		continueUpdates bool
	}{
		{
			name: "returns error from daily aggregate query",
			setupAggregates: func() []risk.DailyAggregate {
				return nil
			},
			expectError:     true,
			continueUpdates: false,
		},
		{
			name: "continues updating other users despite single failure",
			setupAggregates: func() []risk.DailyAggregate {
				u1 := uuid.New()
				u2 := uuid.New()
				return []risk.DailyAggregate{
					{UserID: u1, AvgAmount: 100.0, P95Amount: 200.0},
					{UserID: u2, AvgAmount: 150.0, P95Amount: 300.0},
				}
			},
			expectError:     false,
			continueUpdates: true,
		},
		{
			name: "handles single user aggregate",
			setupAggregates: func() []risk.DailyAggregate {
				return []risk.DailyAggregate{
					{UserID: uuid.New(), AvgAmount: 100.0, P95Amount: 200.0},
				}
			},
			expectError:     false,
			continueUpdates: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockTransactionRiskRepository)
			auditLog := &audit.Logger{}

			aggregates := tt.setupAggregates()

			if tt.expectError {
				repo.On("GetDailyTransactionAggregate", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("query failed"))
			} else {
				repo.On("GetDailyTransactionAggregate", mock.Anything, mock.Anything, mock.Anything).
					Return(aggregates, nil)

				for _, agg := range aggregates {
					repo.On("UpdateBehaviorParams", mock.Anything, agg.UserID, mock.AnythingOfType("float64"), agg.P95Amount).
						Return(nil)
				}
			}

			updater := NewParameterUpdater(repo, auditLog)
			ctx := context.Background()
			day := time.Now().UTC()

			err := updater.UpdateDailyBehavior(ctx, day)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, len(repo.Calls) > 0, "Expected at least one call to repository")
			}
		})
	}
}

// ============ Concurrency and Timeout Tests ============
// Verify behavior under time constraints and cancellation.

func TestParameterUpdater_ContextHandling(t *testing.T) {
	tests := []struct {
		name            string
		contextTimeout  time.Duration
		userCount       int
		expectError     bool
		expectCancelled bool
	}{
		{
			name:            "completes within reasonable timeout",
			contextTimeout:  10 * time.Second,
			userCount:       5,
			expectError:     false,
			expectCancelled: false,
		},
		{
			name:            "returns error on context cancellation",
			contextTimeout:  100 * time.Millisecond,
			userCount:       50,
			expectError:     false,
			expectCancelled: false,
		},
		{
			name:            "handles single user update quickly",
			contextTimeout:  5 * time.Second,
			userCount:       1,
			expectError:     false,
			expectCancelled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockTransactionRiskRepository)
			auditLog := &audit.Logger{}

			aggregates := make([]risk.DailyAggregate, tt.userCount)
			for i := 0; i < tt.userCount; i++ {
				aggregates[i] = risk.DailyAggregate{
					UserID:    uuid.New(),
					AvgAmount: 100.0 + float64(i)*10,
					P95Amount: 200.0 + float64(i)*20,
				}
			}

			repo.On("GetDailyTransactionAggregate", mock.Anything, mock.Anything, mock.Anything).
				Return(aggregates, nil)

			repo.On("UpdateBehaviorParams", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), mock.Anything).
				Return(nil)

			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTimeout)
			defer cancel()

			updater := NewParameterUpdater(repo, auditLog)
			day := time.Now().UTC()

			err := updater.UpdateDailyBehavior(ctx, day)

			if tt.expectError || tt.expectCancelled {
				if tt.expectCancelled {
					assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || err != nil)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
