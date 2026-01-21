package customrouter

import (
	"risk-detection/internal/auth"
	"risk-detection/internal/transaction"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, authHandler *auth.Handler) {

	//Auth routes
	router.POST("/v1/login", authHandler.Login)

	//api routes
	api := router.Group("/api/v1")
	api.POST("/transaction", transaction.HandleTransaction)
}
