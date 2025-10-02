package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init initializes the logger
func Init(environment string) {
	// Set time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Pretty logging for development
	if environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		})
	}

	// Set log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if environment == "development" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().Str("environment", environment).Msg("Logger initialized")
}
