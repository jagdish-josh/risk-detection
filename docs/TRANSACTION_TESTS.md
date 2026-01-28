# Transaction Package Test Suite

## Overview
Comprehensive test cases for the transaction package covering handler, service, and repository layers with extensive edge case coverage following Go testing standards.

## Test Files

### 1. handler_test.go
Unit tests for the TransactionHandler with mocked dependencies.

#### HandleTransaction Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestHandleTransaction_Success` | Verify successful transaction creation and risk calculation | Standard happy path |
| `TestHandleTransaction_MissingUserID` | Validate authentication enforcement | Missing context value |
| `TestHandleTransaction_InvalidUserIDFormat` | Verify UUID validation | Malformed UUID string |
| `TestHandleTransaction_InvalidRequestBody` | Test JSON parsing error handling | Malformed JSON input |
| `TestHandleTransaction_MissingRequiredField_Amount` | Validate binding constraints | Missing required field |
| `TestHandleTransaction_MissingRequiredField_TransactionType` | Validate binding constraints | Missing required field |
| `TestHandleTransaction_InvalidAmount_Zero` | Test validation rules | Zero amount (violates gt=0) |
| `TestHandleTransaction_InvalidAmount_Negative` | Test validation rules | Negative amount (violates gt=0) |
| `TestHandleTransaction_ServiceError` | Handle service layer failures | Risk calculation error |
| `TestHandleTransaction_LargeAmount` | High-value transaction handling | Amount: 999999999.99 |
| `TestHandleTransaction_SmallAmount` | Low-value transaction handling | Amount: 0.01 |
| `TestHandleTransaction_OptionalReceiverID_Nil` | Test optional fields | Nil pointer for deposit |

#### GetTransactions Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestGetTransactions_Success` | Retrieve transactions successfully | Normal pagination |
| `TestGetTransactions_MissingUserID` | Enforce authentication | No user_id in context |
| `TestGetTransactions_InvalidUserID` | UUID validation | Invalid format |
| `TestGetTransactions_DefaultLimitAndOffset` | Default parameter handling | Query params omitted |
| `TestGetTransactions_CustomOffsetAndLimit` | Custom pagination | offset=5, limit=20 |
| `TestGetTransactions_InvalidOffsetAndLimit` | Invalid parameter parsing | Non-numeric values |
| `TestGetTransactions_ServiceError` | Service failure handling | Database error |
| `TestGetTransactions_EmptyResult` | Empty result set | No transactions |
| `TestGetTransactions_LargeOffset` | Boundary condition | offset=10000 |

#### Total Handler Tests: **21**

### 2. service_test.go
Unit tests for the TransactionService with mocked repository and risk service.

#### CalculateRiskMatrix Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestCalculateRiskMatrix_Success_Allow` | Risk decision: ALLOW | Maps to COMPLETED status |
| `TestCalculateRiskMatrix_Success_Flag` | Risk decision: FLAG | Maps to FLAGGED status |
| `TestCalculateRiskMatrix_Success_Block` | Risk decision: BLOCK | Maps to BLOCKED status |
| `TestCalculateRiskMatrix_RepositoryCreateError` | Handle DB creation failure | Transaction not saved |
| `TestCalculateRiskMatrix_RiskServiceError` | Handle risk calculation failure | Service returns error |
| `TestCalculateRiskMatrix_RiskServiceReturnsNil` | Handle unexpected nil result | No risk data |
| `TestCalculateRiskMatrix_UpdateStatusError` | Handle status update failure | Update operation fails |
| `TestCalculateRiskMatrix_UnknownDecision` | Handle unknown decision | Maps to PENDING |
| `TestCalculateRiskMatrix_NilTransaction` | Handle nil input | Nil transaction object |

#### GetTransactions Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestGetTransactions_Success` | Retrieve transactions with total count | Multiple results |
| `TestGetTransactions_EmptyResult` | Empty transaction list | No user transactions |
| `TestGetTransactions_RepositoryError` | Handle query failure | DB connection error |
| `TestGetTransactions_CountError` | Handle count query failure | Count operation fails |
| `TestGetTransactions_NegativeOffset` | Validation and correction | offset < 0 → 0 |
| `TestGetTransactions_NegativeLimit` | Validation and correction | limit < 0 → 10 |
| `TestGetTransactions_ZeroLimit` | Validation and correction | limit == 0 → 10 |
| `TestGetTransactions_LargeOffsetAndLimit` | Large pagination values | offset=10000, limit=1000 |
| `TestGetTransactions_ContextCancelled` | Handle context cancellation | Cancelled context |

#### MapDecisionToStatus Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestMapDecisionToStatus_Allow` | ALLOW → COMPLETED | Standard mapping |
| `TestMapDecisionToStatus_Flag` | FLAG → FLAGGED | Standard mapping |
| `TestMapDecisionToStatus_Block` | BLOCK → BLOCKED | Standard mapping |
| `TestMapDecisionToStatus_Unknown` | Unknown → PENDING | Fallback behavior |
| `TestMapDecisionToStatus_Empty` | Empty string → PENDING | Fallback behavior |
| `TestMapDecisionToStatus_CaseSensitive` | Case sensitivity check | "allow" vs "ALLOW" |

#### Total Service Tests: **27**

### 3. repository_test.go
Integration tests for the TransactionRepository with actual database operations.

#### GetByID Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestGetByID_Success` | Retrieve transaction by ID | Valid ID |
| `TestGetByID_NotFound` | Handle non-existent ID | Record not found error |
| `TestGetByID_InvalidUUID` | Handle zero UUID | Invalid UUID value |

#### Create Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestCreate_Success` | Insert new transaction | Valid data |
| `TestCreate_DuplicateID` | Primary key constraint | Duplicate UUID |
| `TestCreate_InvalidAmount_Negative` | Amount validation | Negative value |
| `TestCreate_EmptyDeviceID` | NOT NULL constraint | Empty string |

#### UpdateStatusByID Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestUpdateStatusByID_Success` | Update transaction status | Valid status |
| `TestUpdateStatusByID_NonExistentID` | Update non-existent record | No error in GORM |
| `TestUpdateStatusByID_AllValidStatuses` | All valid status values | PENDING, COMPLETED, FLAGGED, BLOCKED |
| `TestUpdateStatusByID_InvalidStatus` | CHECK constraint violation | Invalid status value |

#### CountTransactionFrequency Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestCountTransactionFrequency_Success` | Count within time window | Recent transactions |
| `TestCountTransactionFrequency_OutsideWindow` | Exclude old transactions | Outside 60-min window |
| `TestCountTransactionFrequency_ZeroDuration` | Zero-minute window | duration=0 |
| `TestCountTransactionFrequency_NegativeDuration` | Negative duration handling | duration=-60 |
| `TestCountTransactionFrequency_ContextCancelled` | Context cancellation | Cancelled context |

#### GetTransactions Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestGetTransactions_Success` | Retrieve multiple transactions | Multiple records |
| `TestGetTransactions_WithPagination` | Offset and limit applied | Pagination correctness |
| `TestGetTransactions_EmptyResult` | No transactions for user | Empty result set |
| `TestGetTransactions_ContextCancelled` | Context cancellation | Cancelled context |
| `TestGetTransactions_OrderByCreatedAtDesc` | Correct sort order | DESC ordering verification |

#### CountTotalTransaction Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestCountTotalTransaction_Success` | Count user's transactions | Per-user isolation |
| `TestCountTotalTransaction_NoTransactions` | Count zero transactions | Empty result |
| `TestCountTotalTransaction_ContextCancelled` | Context cancellation | Cancelled context |

#### Security Tests

| Test Case | Purpose | Edge Cases |
|-----------|---------|-----------|
| `TestRepository_SQLInjectionPrevention_UserID` | SQL injection prevention | Parameterized queries |
| `TestRepository_ParameterizedQueries` | Query parameterization | Verify safe query patterns |

#### Total Repository Tests: **35**

## Running Tests

### Run All Tests
```bash
go test ./internal/transaction/... -v
```

### Run Specific Test File
```bash
go test -v ./internal/transaction -run TestHandleTransaction
go test -v ./internal/transaction -run TestService
go test -v ./internal/transaction -run TestRepository
```

### Run with Coverage
```bash
go test ./internal/transaction/... -v -cover
go test ./internal/transaction/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Short Mode (skip integration tests)
```bash
go test ./internal/transaction/... -v -short
```

### Run Specific Test
```bash
go test -v ./internal/transaction -run TestCalculateRiskMatrix_Success_Allow
```

## Testing Standards Followed

### 1. **Naming Conventions**
- Test functions follow Go standard: `Test<FunctionName>_<Scenario>`
- Clear, descriptive names indicating what is being tested
- Example: `TestHandleTransaction_InvalidAmount_Negative`

### 2. **Arrange-Act-Assert Pattern**
```go
// Arrange: Setup test data and mocks
mockService := new(MockService)

// Act: Execute the function being tested
handler.HandleTransaction(c)

// Assert: Verify the results
assert.Equal(t, http.StatusOK, w.Code)
```

### 3. **Mocking Strategy**
- **Handler Tests**: Mock Service interface
- **Service Tests**: Mock Repository and RiskService interfaces
- **Repository Tests**: Use actual database (integration tests)
- All mocks implement the same interfaces as production code

### 4. **Edge Case Coverage**

#### Input Validation
- Nil/empty values
- Invalid formats (UUIDs, JSON)
- Boundary values (zero, negative, very large)
- Missing required fields
- Invalid field constraints

#### Error Handling
- Database errors
- Service errors
- Context cancellation
- Unexpected nil returns
- Network failures

#### Business Logic
- All decision paths (ALLOW, FLAG, BLOCK)
- Status mapping
- Pagination boundaries
- Per-user data isolation
- Time-based filtering

#### Security
- SQL injection prevention
- Input validation
- Authentication enforcement
- Parameter binding safety

### 5. **Context Handling**
- Background context
- Cancelled contexts
- Context timeout
- Context deadline exceeded

### 6. **Assertions**
- Using `github.com/stretchr/testify/assert` for clear assertions
- Verifying both positive and negative cases
- Checking error messages and types
- Validating complete response structures

### 7. **Concurrency Considerations**
- Mock call order verification
- Expectation matching with `mock.MatchedBy` for complex objects
- Assertion of all expectations

## Mock Implementation Details

### Handler Mocks
```go
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
```

### Service Mocks
```go
type MockRepository struct {
    mock.Mock
}

type MockRiskService struct {
    mock.Mock
}

type MockAuditLogger struct {
    mock.Mock
}
```

## Expected Behavior

### Handler Layer
- **Status Codes**: 200 (success), 400 (bad request), 401 (unauthorized), 500 (server error)
- **Request Validation**: JSON binding, UUID parsing, field constraints
- **Response Format**: Consistent JSON responses with error details
- **Authentication**: User ID from context (JWT middleware)

### Service Layer
- **Decision Mapping**: ALLOW→COMPLETED, FLAG→FLAGGED, BLOCK→BLOCKED, Unknown→PENDING
- **Audit Logging**: All operations logged
- **Error Propagation**: Errors wrapped with context
- **Parameter Validation**: Limit/offset normalization

### Repository Layer
- **Query Safety**: Parameterized queries, no SQL injection
- **Constraint Validation**: CHECK, NOT NULL, PRIMARY KEY
- **Data Isolation**: Per-user query filtering
- **Sort Order**: CreatedAt DESC (most recent first)

## Coverage Goals

- **Handler**: >95% line coverage
- **Service**: >98% line coverage
- **Repository**: >90% line coverage (integration tests may have skips)

## Dependencies

```go
- github.com/stretchr/testify/assert
- github.com/stretchr/testify/mock
- gorm.io/gorm
- github.com/gin-gonic/gin
- github.com/google/uuid
```

## Future Improvements

1. **Table-Driven Tests**: Refactor repetitive test cases
2. **Fixtures**: Create reusable test data builders
3. **Benchmarks**: Add performance benchmarks for critical paths
4. **Concurrent Tests**: Add race detector tests
5. **Integration Tests**: Full end-to-end tests with real database
