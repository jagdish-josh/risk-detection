package customrouter

import (
	"risk-detection/internal/auth"
	"risk-detection/internal/transaction"

	"github.com/gin-gonic/gin"
)


func RegisterRoutes(router *gin.Engine) {

	//Auth routes
	router.POST("/v1/login", auth.Login)

	//api routes
	api := router.Group("/api/v1")
	api.POST("/transaction", transaction.HandleTransaction)
}
