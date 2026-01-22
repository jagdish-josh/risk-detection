package transaction

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type TransactionHandler struct{

} 

func NewTransactionHandler() *TransactionHandler {
	return &TransactionHandler{}
}

// HandleTransaction handles the /api/v1/transection endpoint.
func HandleTransaction(c *gin.Context) {
	

	
	c.JSON(http.StatusOK, gin.H{"message": "transaction endpoint"})
}
