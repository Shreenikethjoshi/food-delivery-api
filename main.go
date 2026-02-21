package main

import (
	"log"
	"net/http"
	"os"

	"food-delivery-api/config"
	"food-delivery-api/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin mode
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize database
	config.InitDB()

	// Create Gin router with default middleware (logger + recovery)
	r := gin.Default()

	// CORS middleware for frontend integration
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "Food Delivery Order Management API",
			"version": "1.0.0",
		})
	})

	// Welcome
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "🍔 Welcome to the Food Delivery Order Management API",
			"docs":    "/api/state-machine",
			"health":  "/health",
			"roles":   []string{"customer", "restaurant", "driver", "admin"},
		})
	})

	// Register all routes
	routes.SetupRoutes(r)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 Server running on http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
