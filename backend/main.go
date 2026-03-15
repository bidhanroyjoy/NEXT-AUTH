package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env definitions
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it")
	}

	// Change the Gin mode for production
	// gin.SetMode(gin.ReleaseMode)

	InitDB()

	r := gin.Default()

	// CORS Setup
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // Next.js default port
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true, // Needs to be true to accept HttpOnly Cookies
	}))

	// Basic route for testing
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "up and running",
		})
	})

	api := r.Group("/api/auth")
	api.Use(CSRFMiddleware())
	{
		api.POST("/register", Register)
		api.POST("/verify-email", VerifyOTP)
		api.POST("/login", Login)

		// Token Management
		api.POST("/refresh", RefreshToken)

		// Password Reset
		api.POST("/forgot-password", ForgotPassword)
		api.POST("/reset-password", ResetPassword)
	}

	// 2FA Routes (requires authentication)
	twoFA := r.Group("/api/auth/2fa")
	twoFA.Use(CSRFMiddleware())
	twoFA.Use(RequireAuth())
	{
		twoFA.GET("/status", Get2FAStatus)
		twoFA.POST("/setup", Setup2FA)
		twoFA.POST("/verify-setup", VerifySetup2FA)
		twoFA.POST("/disable", Disable2FA)
	}

	log.Println("Server running on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
