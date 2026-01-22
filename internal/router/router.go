package customrouter

import (
	"risk-detection/internal/auth"
	"risk-detection/internal/transaction"
	"risk-detection/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine,
	 authHandler *auth.Handler,
	 transactionHandler *transaction.TransactionHandler,
	 jwtSecret string,
	 ) {

	//Auth routes
	router.POST("/v1/login", authHandler.Login)

	//api routes
	api := router.Group("/api/v1")


 	api.Use(middleware.JWTAuthMiddleware(jwtSecret))
	
	api.POST("/transaction", transaction.HandleTransaction)
}
