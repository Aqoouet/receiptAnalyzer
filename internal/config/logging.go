package config

import (
	"log"
	"os"
	"path/filepath"
)

// SetupLogging configures logging to write to logs directory
func SetupLogging(serviceName string) error {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		return err
	}

	// Open log file
	logFile, err := os.OpenFile(
		filepath.Join("logs", serviceName+".log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return err
	}

	// Set log output to both file and stdout
	log.SetOutput(logFile)

	return nil
}
