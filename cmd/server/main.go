package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"risk-detection/internal/audit"
	"risk-detection/internal/auth"
	"risk-detection/internal/db"
	"risk-detection/internal/risk"
	"risk-detection/internal/risk/cronjob"
	customrouter "risk-detection/internal/router"
	"risk-detection/internal/transaction"

	"github.com/gin-gonic/gin"
)

func main() {

	router := gin.New()
	ctx := context.Background()

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
	authService := auth.NewService(authRepo, auditLogger, jwtSecret, time.Hour)
	authHandler := auth.NewHandler(authService)

	transactionRepo := transaction.NewRepository(DB)

	riskRepo := risk.NewRepository(DB)
	riskService, err := risk.NewService(riskRepo, transactionRepo, auditLogger)
    if err !=  nil {
        log.Fatal("unble to load rules in risks")
    }

	updater := cronjob.NewParameterUpdater(riskRepo ,auditLogger)
	cronjob.StartBehaviorCron(ctx, updater)

	transactionService := transaction.NewService(transactionRepo, riskService, auditLogger)
	transactionHandler := transaction.NewHandler(transactionService)

	customrouter.RegisterRoutes(router, authHandler, transactionHandler, jwtSecret)

	fmt.Println("Connected to database")
	router.Run()
}
