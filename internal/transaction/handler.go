package transaction

import (
	"github.com/gin-gonic/gin"
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

	userID := c.GetString("user_id")
	role := c.GetString("role")
	email := c.GetString("email")

	



	c.JSON(200, gin.H{
		"user_id": userID,
		"role":    role,
		"email":   email,
	})
}
