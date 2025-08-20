package holydb

import (
	"testing"
)

func TestExecute(t *testing.T) {
	// Test that Execute doesn't panic and returns without error for default case
	// Note: This is a basic test since Execute() has side effects (prints to stdout)
	err := runDefault()
	if err != nil {
		t.Errorf("runDefault() returned error: %v", err)
	}
}

func TestShowVersion(t *testing.T) {
	// Test that showVersion doesn't panic
	// Note: This function prints to stdout but shouldn't return an error
	showVersion()
}

func TestShowHelp(t *testing.T) {
	// Test that showHelp doesn't panic
	// Note: This function prints to stdout but shouldn't return an error
	showHelp()
}
