package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Setup2FA generates a TOTP secret and returns the provisioning URI for QR code
func Setup2FA(c *gin.Context) {
	// Get user from JWT (set by AuthMiddleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.Is2FAEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is already enabled"})
		return
	}

	// Generate TOTP secret
	secret, otpauthURL, err := GenerateTOTPSecret(user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate 2FA secret"})
		return
	}

	// Store the secret (not yet enabled)
	user.TwoFASecret = secret
	DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"secret":      secret,
		"otpauth_url": otpauthURL,
		"message":     "Scan the QR code with Google Authenticator, then verify with a code.",
	})
}

// VerifySetup2FA verifies the first TOTP code to confirm 2FA setup
func VerifySetup2FA(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide the 6-digit code"})
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.TwoFASecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please setup 2FA first"})
		return
	}

	if user.Is2FAEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is already enabled"})
		return
	}

	// Validate the TOTP code
	if !ValidateTOTP(user.TwoFASecret, input.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code. Please try again."})
		return
	}

	// Enable 2FA
	user.Is2FAEnabled = true
	DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "2FA has been enabled successfully!"})
}

// Disable2FA disables 2FA for the user (requires current TOTP code)
func Disable2FA(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide the 6-digit code"})
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !user.Is2FAEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is not enabled"})
		return
	}

	// Validate the TOTP code before disabling
	if !ValidateTOTP(user.TwoFASecret, input.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}

	// Disable 2FA
	user.Is2FAEnabled = false
	user.TwoFASecret = ""
	DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "2FA has been disabled successfully"})
}

// Get2FAStatus returns whether 2FA is enabled for the user
func Get2FAStatus(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_2fa_enabled": user.Is2FAEnabled})
}
