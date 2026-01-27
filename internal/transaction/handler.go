package transaction

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TransactionHandler struct {
    service Service
}

func NewHandler(service Service) *TransactionHandler {
    return &TransactionHandler{
        service: service,
    }
}

// HandleTransaction handles the /api/v1/transaction endpoint.
func (h *TransactionHandler) HandleTransaction(c *gin.Context) {
    // Get user ID from JWT middleware
    userIDStr, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
        return
    }

    // Convert userID to UUID
    userID, err := uuid.Parse(userIDStr.(string))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id format"})
        return
    }

    // Bind request body - ensure Content-Type is application/json
    var req TransactionRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "invalid request body",
            "details": err.Error(),
        })
        return
    }

    // Create transaction object
    transaction := Transaction{
        UserID:          userID,
        TransactionType: req.TransactionType,
        ReceiverID:      req.ReceiverID,
        Amount:          req.Amount,
        DeviceID:        req.DeviceID,
        IPAddress:       c.ClientIP(),
        TransactionStatus:          "PENDING",
        TransactionTime: req.TransactionTime,
    }

    // Call service to calculate risk
    riskResult, err := h.service.CalculateRiskMatrix(&transaction)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "risk_result": riskResult,
    })
}

func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	ctx := c.Request.Context()

	// Example: userID from Gin context (set by auth middleware)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	 userID, err := uuid.Parse(userIDValue.(string))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id format"})
        return
    }

	// Query params
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	transactions, total, err := h.service.GetTransactions(
		ctx,
		userID,
		offset,
		limit,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  transactions,
		"total": total,
		"meta": gin.H{
			"offset": offset,
			"limit":  limit,
		},
	})
}
