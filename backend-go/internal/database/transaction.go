package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// WithTransaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func WithTransaction(ctx context.Context, fn func(*gorm.DB) error) error {
	tx := DB.WithContext(ctx).Begin()

	// Ensure rollback on panic
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Error().
				Interface("panic", r).
				Msg("Transaction rolled back due to panic")
			panic(r) // Re-throw panic after rollback
		}
	}()

	// Check if transaction started successfully
	if err := tx.Error; err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute the function
	if err := fn(tx); err != nil {
		tx.Rollback()
		log.Debug().Err(err).Msg("Transaction rolled back")
		return err
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Msg("Transaction committed successfully")
	return nil
}

// WithTransactionRetry executes a function within a transaction with retry logic
// Useful for handling deadlocks and serialization errors
func WithTransactionRetry(ctx context.Context, maxRetries int, fn func(*gorm.DB) error) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Info().
				Int("attempt", attempt).
				Int("max_retries", maxRetries).
				Msg("Retrying transaction")
		}

		err := WithTransaction(ctx, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable (deadlock, serialization failure, etc.)
		if !isRetryableError(err) {
			return err
		}

		// Don't retry on last attempt
		if attempt == maxRetries {
			break
		}
	}

	log.Error().
		Err(lastErr).
		Int("attempts", maxRetries+1).
		Msg("Transaction failed after retries")

	return fmt.Errorf("transaction failed after %d attempts: %w", maxRetries+1, lastErr)
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// serialization_failure, deadlock_detected
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	return false
}

// IsUniqueViolation reports whether err is a Postgres unique_violation (23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
