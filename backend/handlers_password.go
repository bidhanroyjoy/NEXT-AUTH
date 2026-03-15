package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RefreshToken validates a refresh token and returns a new access token
func RefreshToken(c *gin.Context) {
	// Typically, the refresh token is stored in an HttpOnly cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token missing"})
		return
	}

	claims, err := ValidateToken(refreshToken, true)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Generate new access token
	accessToken, _, err := GenerateTokens(claims.UserID, claims.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate new token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

// ForgotPassword generates an OTP, saves it, and sends it via email mock
func ForgotPassword(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var user User
	if err := DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// To prevent email enumeration, return a generic success message
		c.JSON(http.StatusOK, gin.H{"message": "If this email is registered, an OTP has been sent."})
		return
	}

	// Generate OTP
	otpCode, _ := GenerateOTP()
	otp := OTP{
		UserID:    user.ID,
		Code:      otpCode,
		Purpose:   "ForgotPassword",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	DB.Create(&otp)

	// Send Email with OTP
	if err := SendEmail(user.Email, "Password Reset", fmt.Sprintf("Your password reset code is: %s", otpCode)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send password reset email. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If this email is registered, an OTP has been sent."})
}

// ResetPassword verifies the OTP and updates the password
func ResetPassword(c *gin.Context) {
	var input struct {
		Email       string `json:"email" binding:"required,email"`
		OTP         string `json:"otp" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	var user User
	if err := DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate OTP
	var otp OTP
	if err := DB.Where("user_id = ? AND code = ? AND purpose = ? AND expires_at > ?", user.ID, input.OTP, "ForgotPassword", time.Now()).First(&otp).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// Hash new password
	hashedPassword, err := HashPassword(input.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Update User record
	user.PasswordHash = hashedPassword
	user.FailedLoginAttempts = 0 // Reset attempts on manual password reset
	DB.Save(&user)

	// Invalidate the OTP
	DB.Where("user_id = ? AND purpose = ?", user.ID, "ForgotPassword").Delete(&OTP{})

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
