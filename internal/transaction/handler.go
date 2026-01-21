package transaction

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleTransaction handles the /api/v1/transection endpoint.
func HandleTransaction(c *gin.Context) {
	

	
	c.JSON(http.StatusOK, gin.H{"message": "transaction endpoint"})
}
