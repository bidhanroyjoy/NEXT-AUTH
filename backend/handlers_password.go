package main

import (
	"fmt"
	"net/http"
	"time"
)

// RefreshToken validates a refresh token and returns a new access token
func RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := getCookie(r, "refresh_token")
	if refreshToken == "" {
		writeError(w, http.StatusUnauthorized, "Refresh token missing")
		return
	}

	claims, err := ValidateToken(refreshToken, true)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	// Generate new access token
	accessToken, _, err := GenerateTokens(claims.UserID, claims.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not generate new token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"access_token": accessToken})
}

// ForgotPassword generates an OTP, saves it, and sends it via email
func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	if err := readJSON(r, &input); err != nil || input.Email == "" {
		writeError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	var user User
	if err := DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// Prevent email enumeration
		writeJSON(w, http.StatusOK, map[string]string{"message": "If this email is registered, an OTP has been sent."})
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
	isMock, err := SendEmail(user.Email, "Password Reset", fmt.Sprintf("Your password reset code is: %s", otpCode))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to send password reset email. Please try again.")
		return
	}

	response := map[string]interface{}{
		"message": "If this email is registered, an OTP has been sent.",
	}
	if isMock {
		response["otp"] = otpCode
		response["message"] = "SMTP not configured — use the OTP shown below."
	}
	writeJSON(w, http.StatusOK, response)
}

// ResetPassword verifies the OTP and updates the password
func ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email       string `json:"email"`
		OTP         string `json:"otp"`
		NewPassword string `json:"new_password"`
	}

	if err := readJSON(r, &input); err != nil || input.Email == "" || input.OTP == "" || len(input.NewPassword) < 6 {
		writeError(w, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	var user User
	if err := DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate OTP
	var otp OTP
	if err := DB.Where("user_id = ? AND code = ? AND purpose = ? AND expires_at > ?", user.ID, input.OTP, "ForgotPassword", time.Now()).First(&otp).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid or expired OTP")
		return
	}

	// Hash new password
	hashedPassword, err := HashPassword(input.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	// Update user
	user.PasswordHash = hashedPassword
	user.FailedLoginAttempts = 0
	DB.Save(&user)

	// Invalidate OTPs
	DB.Where("user_id = ? AND purpose = ?", user.ID, "ForgotPassword").Delete(&OTP{})

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully"})
}
