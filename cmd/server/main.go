package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"risk-detection/internal/auth"
	"risk-detection/internal/db"
	customrouter "risk-detection/internal/router"

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


	authRepo := auth.NewRepository(DB) 
	authService := auth.NewService(authRepo, jwtSecret, time.Hour)
	authHandler := auth.NewHandler(authService)

	customrouter.RegisterRoutes(router, authHandler)

	fmt.Println("Connected to database")
	router.Run()

}
