package config

import (
	"os"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}

	if cfg.LogDir == "" {
		t.Error("LogDir should not be empty")
	}

	if cfg.Debug {
		t.Error("Debug should be false by default")
	}
}

func TestLoad(t *testing.T) {
	// Save original env vars
	originalDataDir := os.Getenv("HOLYDB_DATA_DIR")
	originalLogDir := os.Getenv("HOLYDB_LOG_DIR")
	originalDebug := os.Getenv("HOLYDB_DEBUG")

	// Clean up after test
	defer func() {
		os.Setenv("HOLYDB_DATA_DIR", originalDataDir)
		os.Setenv("HOLYDB_LOG_DIR", originalLogDir)
		os.Setenv("HOLYDB_DEBUG", originalDebug)
	}()

	// Test with environment variables
	os.Setenv("HOLYDB_DATA_DIR", "/custom/data")
	os.Setenv("HOLYDB_LOG_DIR", "/custom/logs")
	os.Setenv("HOLYDB_DEBUG", "true")

	cfg := Load()

	if cfg.DataDir != "/custom/data" {
		t.Errorf("Expected DataDir '/custom/data', got '%s'", cfg.DataDir)
	}

	if cfg.LogDir != "/custom/logs" {
		t.Errorf("Expected LogDir '/custom/logs', got '%s'", cfg.LogDir)
	}

	if !cfg.Debug {
		t.Error("Debug should be true when HOLYDB_DEBUG=true")
	}
}
