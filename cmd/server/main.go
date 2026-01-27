package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"risk-detection/internal/auth"
	"risk-detection/internal/db"
	"risk-detection/internal/risk"
	"risk-detection/internal/risk/cronjob"
	customrouter "risk-detection/internal/router"
	"risk-detection/internal/transaction"
    "risk-detection/internal/audit"

	"github.com/gin-gonic/gin"
)

func main() {

	router := gin.New()

	DB, err := db.Connect()

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

    auditLogger, err := audit.NewLogger("internal/audit/file.log")
    defer auditLogger.Close()

    if err != nil {
        log.Fatal(err)
    }

	authRepo := auth.NewRepository(DB)
	authService := auth.NewService(authRepo, jwtSecret, time.Hour)
	authHandler := auth.NewHandler(authService)

	transactionRepo := transaction.NewRepository(DB)

	riskRepo := risk.NewRepository(DB)
	riskService, err := risk.NewService(riskRepo, transactionRepo, auditLogger)
    if err !=  nil {
        log.Fatal("unble to load rules in risks")
    }

	cronjob.NewParameterUpdater(riskRepo)

	transactionService := transaction.NewService(transactionRepo, riskService, auditLogger)
	transactionHandler := transaction.NewHandler(transactionService)

	customrouter.RegisterRoutes(router, authHandler, transactionHandler, jwtSecret)

	fmt.Println("Connected to database")
	router.Run()
}
