package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Register handles user registration, creates unverified user and OTP
func Register(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Check if user already exists
	var existingUser User
	if err := DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
		return
	}

	// Create user (unverified initially)
	user := User{
		Email:           input.Email,
		PasswordHash:    hashedPassword,
		IsEmailVerified: false,
	}

	if err := DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}

	// Generate OTP
	otpCode, _ := GenerateOTP()
	otp := OTP{
		UserID:    user.ID,
		Code:      otpCode,
		Purpose:   "EmailVerification",
		ExpiresAt: time.Now().Add(10 * time.Minute), // OTP valid for 10 mins
	}

	if err := DB.Create(&otp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate OTP"})
		return
	}

	// Send Email with OTP
	if err := SendEmail(user.Email, "Verify your email", fmt.Sprintf("Your OTP code is: %s", otpCode)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration successful but failed to send OTP email. Please try again."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Registration successful. Please verify your email using the OTP sent."})
}

// VerifyOTP verifies the OTP submitted for email registration
func VerifyOTP(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
		OTP   string `json:"otp" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Find the user first
	var user User
	if err := DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if already verified
	if user.IsEmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is already verified"})
		return
	}

	// Find valid OTP record
	var otp OTP
	if err := DB.Where("user_id = ? AND code = ? AND purpose = ? AND expires_at > ?", user.ID, input.OTP, "EmailVerification", time.Now()).First(&otp).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// OTP matched, mark user as verified
	user.IsEmailVerified = true
	DB.Save(&user)

	// Clean up all EmailVerification OTPs for this user
	DB.Where("user_id = ? AND purpose = ?", user.ID, "EmailVerification").Delete(&OTP{})

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully. You can now login."})
}

// Login handles user authentication, JWT generation, and Captcha triggers
func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Captcha  string `json:"captcha"`    // Captcha response from client
		TOTPCode string `json:"totp_code"`  // Google Authenticator code (optional)
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login data"})
		return
	}

	var user User
	if err := DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Enforcement of Email Verification
	if !user.IsEmailVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email not verified. Please verify first."})
		return
	}

	// CAPTCHA Validation Logic
	if user.FailedLoginAttempts >= 3 {
		// Mock CAPTCHA check: assume missing or invalid "captcha" means test fails
		if input.Captcha == "" || input.Captcha != "VALID_CAPTCHA_MOCK" { // Replace with real captcha verification
			c.JSON(http.StatusForbidden, gin.H{
				"error":           "Too many failed attempts. Captcha required.",
				"captchaRequired": true,
			})
			return
		}
	}

	// Password Check
	if !CheckPasswordHash(input.Password, user.PasswordHash) {
		// Increment failed attempts
		user.FailedLoginAttempts += 1
		DB.Save(&user)

		// Return Captcha requirement if threshold reached just now
		if user.FailedLoginAttempts >= 3 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials. Captcha now required.", "captchaRequired": true})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 2FA Check — if enabled, require TOTP code
	if user.Is2FAEnabled {
		if input.TOTPCode == "" {
			// Password is correct but 2FA code is needed
			c.JSON(http.StatusForbidden, gin.H{
				"error":        "2FA code required",
				"requires_2fa": true,
			})
			return
		}
		// Validate the TOTP code
		if !ValidateTOTP(user.TwoFASecret, input.TOTPCode) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        "Invalid 2FA code",
				"requires_2fa": true,
			})
			return
		}
	}

	// Successful Login: Reset failed attempts back to 0
	user.FailedLoginAttempts = 0
	DB.Save(&user)

	// Generate JWTs
	accessToken, refreshToken, err := GenerateTokens(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	// Set Refresh Token and CSRF tokens as HttpOnly cookies for security
	c.SetCookie("refresh_token", refreshToken, 7*24*3600, "/", "localhost", false, true) // HttpOnly true

	c.JSON(http.StatusOK, gin.H{
		"message":      "Login successful",
		"access_token": accessToken,
		// refresh_token typically handled via HTTPOnly cookie automatically set above, but returning here for client flexibility
	})
}
