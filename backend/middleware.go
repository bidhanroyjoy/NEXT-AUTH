package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateRandomBytes returns securely generated random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded securely generated random string.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

// CSRFMiddleware handles generating and validating CSRF tokens
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For safe methods, generate a token if not present
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			csrfToken, _ := c.Cookie("csrf_token")
			if csrfToken == "" {
				newToken, _ := GenerateRandomString(32)
				c.SetCookie("csrf_token", newToken, 3600, "/", "localhost", false, false) // Not HttpOnly
			}
			c.Next()
			return
		}

		// For state-mutating methods, validate the token, but skip for certain auth routes
		// like login and register where the user is just establishing their session
		path := c.Request.URL.Path
		if path == "/api/auth/login" || path == "/api/auth/register" || path == "/api/auth/verify-email" || path == "/api/auth/forgot-password" || path == "/api/auth/reset-password" {
			c.Next()
			return
		}

		csrfCookie, err := c.Cookie("csrf_token")
		csrfHeader := c.GetHeader("X-CSRF-Token")

		if err != nil || csrfHeader == "" || csrfCookie != csrfHeader {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token mismatch or missing"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAuth middleware to validate access token
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing or invalid"})
			c.Abort()
			return
		}

		tokenStr := authHeader[7:]
		claims, err := ValidateToken(tokenStr, false)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Next()
	}
}
