package main

import (
	"net/http"
)

// Setup2FA generates a TOTP secret and returns the provisioning URI for QR code
func Setup2FA(w http.ResponseWriter, r *http.Request) {
	userID := parseUint(r.Header.Get("X-User-ID"))
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	if user.Is2FAEnabled {
		writeError(w, http.StatusBadRequest, "2FA is already enabled")
		return
	}

	// Generate TOTP secret
	secret, otpauthURL, err := GenerateTOTPSecret(user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate 2FA secret")
		return
	}

	// Store the secret (not yet enabled)
	user.TwoFASecret = secret
	DB.Save(&user)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"secret":      secret,
		"otpauth_url": otpauthURL,
		"message":     "Scan the QR code with Google Authenticator, then verify with a code.",
	})
}

// VerifySetup2FA verifies the first TOTP code to confirm 2FA setup
func VerifySetup2FA(w http.ResponseWriter, r *http.Request) {
	userID := parseUint(r.Header.Get("X-User-ID"))
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input struct {
		Code string `json:"code"`
	}
	if err := readJSON(r, &input); err != nil || input.Code == "" {
		writeError(w, http.StatusBadRequest, "Please provide the 6-digit code")
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	if user.TwoFASecret == "" {
		writeError(w, http.StatusBadRequest, "Please setup 2FA first")
		return
	}

	if user.Is2FAEnabled {
		writeError(w, http.StatusBadRequest, "2FA is already enabled")
		return
	}

	// Validate the TOTP code
	if !ValidateTOTP(user.TwoFASecret, input.Code) {
		writeError(w, http.StatusUnauthorized, "Invalid code. Please try again.")
		return
	}

	// Enable 2FA
	user.Is2FAEnabled = true
	DB.Save(&user)

	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA has been enabled successfully!"})
}

// Disable2FA disables 2FA for the user (requires current TOTP code)
func Disable2FA(w http.ResponseWriter, r *http.Request) {
	userID := parseUint(r.Header.Get("X-User-ID"))
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input struct {
		Code string `json:"code"`
	}
	if err := readJSON(r, &input); err != nil || input.Code == "" {
		writeError(w, http.StatusBadRequest, "Please provide the 6-digit code")
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	if !user.Is2FAEnabled {
		writeError(w, http.StatusBadRequest, "2FA is not enabled")
		return
	}

	// Validate the TOTP code before disabling
	if !ValidateTOTP(user.TwoFASecret, input.Code) {
		writeError(w, http.StatusUnauthorized, "Invalid code")
		return
	}

	// Disable 2FA
	user.Is2FAEnabled = false
	user.TwoFASecret = ""
	DB.Save(&user)

	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA has been disabled successfully"})
}

// Get2FAStatus returns whether 2FA is enabled for the user
func Get2FAStatus(w http.ResponseWriter, r *http.Request) {
	userID := parseUint(r.Header.Get("X-User-ID"))
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"is_2fa_enabled": user.Is2FAEnabled})
}
