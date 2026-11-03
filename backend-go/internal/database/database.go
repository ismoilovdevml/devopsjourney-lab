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

	// Backfill legacy NULLs before AutoMigrate adds NOT NULL constraints.
	if DB.Migrator().HasTable(&models.Progress{}) {
		if err := DB.Exec(`UPDATE progresses SET time_spent = COALESCE(time_spent, 0),
			attempts = COALESCE(attempts, 1), passed = COALESCE(passed, false)
			WHERE time_spent IS NULL OR attempts IS NULL OR passed IS NULL`).Error; err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	err := DB.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.Progress{},
		&models.CheckAttempt{},
		&models.ActivityLog{},
	)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	for _, stmt := range postMigrate {
		if err := DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	log.Info().Msg("Database migrations completed")
	return nil
}

// postMigrate holds idempotent data fixes and indexes GORM tags cannot express.
var postMigrate = []string{
	// Older versions inserted one progress row per passed check; keep the best row per user+task,
	// carrying over the earliest completion time and the number of rows as attempts.
	`UPDATE progresses p SET completed_at = d.first_completed, attempts = d.n
	   FROM (SELECT user_id, task_id, COUNT(*) AS n,
	                MIN(completed_at) FILTER (WHERE passed AND completed_at > '1971-01-01') AS first_completed
	           FROM progresses GROUP BY user_id, task_id HAVING COUNT(*) > 1) d
	  WHERE p.user_id = d.user_id AND p.task_id = d.task_id`,
	`DELETE FROM progresses a USING progresses b
	  WHERE a.user_id = b.user_id AND a.task_id = b.task_id
	    AND (a.passed::int, a.score, a.id) < (b.passed::int, b.score, b.id)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_progress_user_task ON progresses (user_id, task_id)`,
	`UPDATE progresses SET completed_at = NULL WHERE passed = false AND completed_at IS NOT NULL`,
	`UPDATE progresses SET completed_at = NULL WHERE completed_at < '1971-01-01'`,
	`CREATE INDEX IF NOT EXISTS idx_progress_passed_completed ON progresses (passed, completed_at)`,
	`UPDATE activity_logs SET level = 'INFO' WHERE level IS NULL OR level = ''`,
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
