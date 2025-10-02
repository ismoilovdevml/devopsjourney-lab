package database

import (
	"os"

	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/models"
	"github.com/rs/zerolog/log"
)

// SeedAdminUser creates default admin user from environment variables
func SeedAdminUser() error {
	// Check if admin seeding is enabled
	if os.Getenv("SEED_ADMIN") != "true" {
		log.Info().Msg("Admin seeding disabled (SEED_ADMIN != true)")
		return nil
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	// Validate required fields
	if adminUsername == "" || adminEmail == "" || adminPassword == "" {
		log.Warn().Msg("Admin credentials not provided in ENV, skipping admin seed")
		return nil
	}

	// Check if admin already exists
	var existingAdmin models.User
	err := DB.Where("username = ? OR email = ?", adminUsername, adminEmail).First(&existingAdmin).Error

	if err == nil {
		// Admin exists, update if needed
		if existingAdmin.Role != "admin" {
			DB.Model(&existingAdmin).Update("role", "admin")
			log.Info().Str("username", adminUsername).Msg("Existing user promoted to admin")
		} else {
			log.Info().Str("username", adminUsername).Msg("Admin user already exists")
		}
		return nil
	}

	// Create new admin user
	hashedPassword, err := auth.HashPassword(adminPassword)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash admin password")
		return err
	}

	admin := models.User{
		Username: adminUsername,
		Email:    adminEmail,
		Password: hashedPassword,
		Role:     "admin",
		Active:   true,
	}

	if err := DB.Create(&admin).Error; err != nil {
		log.Error().Err(err).Msg("Failed to create admin user")
		return err
	}

	log.Info().
		Str("username", adminUsername).
		Str("email", adminEmail).
		Msg("✅ Admin user created successfully")

	return nil
}
