package main

import (
	"fmt"
	"net/http"
	"time"
)

// Register handles user registration, creates unverified user and OTP
func Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := readJSON(r, &input); err != nil || input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	if len(input.Password) < 6 {
		writeError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Check if user already exists
	var existingUser User
	if err := DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		writeError(w, http.StatusConflict, "User with this email already exists")
		return
	}

	// Hash password
	hashedPassword, err := HashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not hash password")
		return
	}

	// Create user (unverified initially)
	user := User{
		Email:           input.Email,
		PasswordHash:    hashedPassword,
		IsEmailVerified: false,
	}

	if err := DB.Create(&user).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create user")
		return
	}

	// Generate OTP
	otpCode, _ := GenerateOTP()
	otp := OTP{
		UserID:    user.ID,
		Code:      otpCode,
		Purpose:   "EmailVerification",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := DB.Create(&otp).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Could not generate OTP")
		return
	}

	// Send Email with OTP
	isMock, err := SendEmail(user.Email, "Verify your email", fmt.Sprintf("Your OTP code is: %s", otpCode))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Registration successful but failed to send OTP email. Please try again.")
		return
	}

	response := map[string]interface{}{
		"message": "Registration successful. Please verify your email using the OTP sent.",
	}
	if isMock {
		response["otp"] = otpCode
		response["message"] = "Registration successful. SMTP not configured — use the OTP shown below."
	}
	writeJSON(w, http.StatusCreated, response)
}

// VerifyOTP verifies the OTP submitted for email registration
func VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}

	if err := readJSON(r, &input); err != nil || input.Email == "" || input.OTP == "" {
		writeError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	// Find the user
	var user User
	if err := DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	if user.IsEmailVerified {
		writeError(w, http.StatusBadRequest, "Email is already verified")
		return
	}

	// Find valid OTP record
	var otp OTP
	if err := DB.Where("user_id = ? AND code = ? AND purpose = ? AND expires_at > ?", user.ID, input.OTP, "EmailVerification", time.Now()).First(&otp).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid or expired OTP")
		return
	}

	// OTP matched, mark user as verified
	user.IsEmailVerified = true
	DB.Save(&user)

	// Clean up OTPs
	DB.Where("user_id = ? AND purpose = ?", user.ID, "EmailVerification").Delete(&OTP{})

	writeJSON(w, http.StatusOK, map[string]string{"message": "Email verified successfully. You can now login."})
}

// Login handles user authentication, JWT generation, and Captcha triggers
func Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Captcha  string `json:"captcha"`
		TOTPCode string `json:"totp_code"`
	}

	if err := readJSON(r, &input); err != nil || input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "Invalid login data")
		return
	}

	var user User
	if err := DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Email verification check
	if !user.IsEmailVerified {
		writeError(w, http.StatusForbidden, "Email not verified. Please verify first.")
		return
	}

	// CAPTCHA check
	if user.FailedLoginAttempts >= 3 {
		if input.Captcha == "" || input.Captcha != "VALID_CAPTCHA_MOCK" {
			writeJSON(w, http.StatusForbidden, map[string]interface{}{
				"error":           "Too many failed attempts. Captcha required.",
				"captchaRequired": true,
			})
			return
		}
	}

	// Password check
	if !CheckPasswordHash(input.Password, user.PasswordHash) {
		user.FailedLoginAttempts += 1
		DB.Save(&user)

		if user.FailedLoginAttempts >= 3 {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"error":           "Invalid credentials. Captcha now required.",
				"captchaRequired": true,
			})
			return
		}
		writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// 2FA check
	if user.Is2FAEnabled {
		if input.TOTPCode == "" {
			writeJSON(w, http.StatusForbidden, map[string]interface{}{
				"error":        "2FA code required",
				"requires_2fa": true,
			})
			return
		}
		if !ValidateTOTP(user.TwoFASecret, input.TOTPCode) {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"error":        "Invalid 2FA code",
				"requires_2fa": true,
			})
			return
		}
	}

	// Successful login
	user.FailedLoginAttempts = 0
	DB.Save(&user)

	accessToken, refreshToken, err := GenerateTokens(user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// Set refresh token as HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Login successful",
		"access_token": accessToken,
	})
}
