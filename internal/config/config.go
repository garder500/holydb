// Package config handles configuration for HolyDB
package config

import (
	"os"
	"path/filepath"
)

// Config holds the configuration for HolyDB
type Config struct {
	DataDir string
	LogDir  string
	Debug   bool
}

// Default returns a default configuration
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".holydb", "data")
	logDir := filepath.Join(homeDir, ".holydb", "logs")

	return &Config{
		DataDir: dataDir,
		LogDir:  logDir,
		Debug:   false,
	}
}

// Load loads configuration from environment or defaults
func Load() *Config {
	cfg := Default()

	if dataDir := os.Getenv("HOLYDB_DATA_DIR"); dataDir != "" {
		cfg.DataDir = dataDir
	}

	if logDir := os.Getenv("HOLYDB_LOG_DIR"); logDir != "" {
		cfg.LogDir = logDir
	}

	if os.Getenv("HOLYDB_DEBUG") == "true" {
		cfg.Debug = true
	}

	return cfg
}
