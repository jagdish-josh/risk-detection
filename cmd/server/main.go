package main

import (
	"fmt"
	"log"
	"risk-detection/internal/db"
	customrouter "risk-detection/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {

	router := gin.New()

	_, err := db.Connect()

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	customrouter.RegisterRoutes(router)
	

  fmt.Println("Connected to database")
  router.Run()

	

	
}