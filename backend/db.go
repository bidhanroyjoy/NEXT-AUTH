package main

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB initializes the database connection and runs auto-migrations
func InitDB() {
	// Remove .env dependency and Postgres requirements for simpler setup
	var err error

	// Connect to SQLite database file
	DB, err = gorm.Open(sqlite.Open("auth.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate model
	err = DB.AutoMigrate(&User{}, &OTP{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database connection established and models migrated.")
}
