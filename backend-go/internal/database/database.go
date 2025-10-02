package database

import (
	"fmt"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/models"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect establishes database connection
func Connect(cfg *configs.DatabaseConfig) error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Info().Msg("Database connected successfully")
	return nil
}

// Migrate runs database migrations
func Migrate() error {
	log.Info().Msg("Running database migrations...")

	err := DB.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.Progress{},
	)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Info().Msg("Database migrations completed")
	return nil
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// HealthCheck checks database connection
func HealthCheck() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
