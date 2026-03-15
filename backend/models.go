package main

import (
	"time"

	"gorm.io/gorm"
)

// User represents the user model in the database
type User struct {
	gorm.Model                          // Adds ID, CreatedAt, UpdatedAt, DeletedAt
	Email               string `gorm:"uniqueIndex;not null"` // User's email, must be unique
	PasswordHash        string `gorm:"not null"`             // Bcrypt hash of the password
	IsEmailVerified     bool   `gorm:"default:false"`        // Track if email is verified
	FailedLoginAttempts int    `gorm:"default:0"`            // Track failed attempts for captcha trigger
	TwoFASecret         string `gorm:"default:''"`           // TOTP secret for Google Authenticator
	Is2FAEnabled        bool   `gorm:"default:false"`        // Whether 2FA is active
}

// OTP represents One-Time Passwords sent to users for verification or password reset
type OTP struct {
	ID        uint      `gorm:"primaryKey"`                 // Unique ID
	UserID    uint      `gorm:"not null"`                   // The associated user
	Code      string    `gorm:"not null"`                   // The OTP string
	Purpose   string    `gorm:"not null"`                   // E.g., "EmailVerification" or "ForgotPassword"
	ExpiresAt time.Time `gorm:"not null"`                   // Expiration time of the OTP
	CreatedAt time.Time `gorm:"autoCreateTime"`             // Creation time
}
