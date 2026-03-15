package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
)

// HashPassword generates a bcrypt hash from the given password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14) // 14 is the cost
	return string(bytes), err
}

// CheckPasswordHash compares a raw password with a bcrypt hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateOTP produces a 6-digit random string for OTP
func GenerateOTP() (string, error) {
	// Secure random number generation for OTP
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	// Format as a 6 digit string, left-padded with zeros if needed
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// SendEmail sends an email using SMTP via gomail.
// Returns (isMock, err). isMock=true means email was logged, not actually sent.
func SendEmail(to, subject, body string) (bool, error) {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	fromEmail := os.Getenv("SMTP_FROM")

	if smtpHost == "" || smtpPortStr == "" || smtpUser == "" || smtpPass == "" ||
		strings.HasPrefix(smtpPass, "REPLACE") || strings.HasPrefix(smtpPass, "your_") {
		log.Printf("\n--- MOCK EMAIL (SMTP not configured) ---\nTo: %s\nSubject: %s\nBody: %s\n------------------\n", to, subject, body)
		return true, nil
	}

	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		log.Printf("Invalid SMTP_PORT value: %s", smtpPortStr)
		return false, fmt.Errorf("invalid SMTP port: %s", smtpPortStr)
	}

	if fromEmail == "" {
		fromEmail = smtpUser
	}

	m := gomail.NewMessage()
	m.SetHeader("From", fromEmail)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)

	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
		return false, err
	}
	log.Printf("Email successfully sent to %s", to)
	return false, nil
}

// JWT Claims struct
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateTokens creates both an access token and a refresh token
func GenerateTokens(userID uint, email string) (accessToken, refreshToken string, err error) {
	// Read secrets securely from env variables
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if accessSecret == "" {
		accessSecret = "fallback_access_secret" // fallback for dev
	}
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = "fallback_refresh_secret" // fallback for dev
	}

	// 1. Generate Access Token (short-lived)
	accessExpirationTime := time.Now().Add(15 * time.Minute) // 15 mins expiry
	accessClaims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessTokenRaw := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenRaw.SignedString([]byte(accessSecret))
	if err != nil {
		return "", "", err
	}

	// 2. Generate Refresh Token (long-lived)
	refreshExpirationTime := time.Now().Add(7 * 24 * time.Hour) // 7 days expiry
	refreshClaims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	refreshTokenRaw := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenRaw.SignedString([]byte(refreshSecret))

	return accessToken, refreshToken, err
}

// ValidateToken parses a JWT token string
func ValidateToken(tokenStr string, isRefresh bool) (*Claims, error) {
	secretKey := os.Getenv("JWT_ACCESS_SECRET")
	if isRefresh {
		secretKey = os.Getenv("JWT_REFRESH_SECRET")
	}

	if secretKey == "" {
		if isRefresh {
			secretKey = "fallback_refresh_secret"
		} else {
			secretKey = "fallback_access_secret"
		}
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// GenerateTOTPSecret creates a new TOTP secret for a user
func GenerateTOTPSecret(email string) (secret string, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "FoodiAuth",
		AccountName: email,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// ValidateTOTP checks if a TOTP code is valid for the given secret
func ValidateTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}
