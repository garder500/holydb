package holydb

import (
	"fmt"
	"os"
)

// Execute is the main entry point for the holydb command
func Execute() error {
	args := os.Args[1:]

	if len(args) == 0 {
		return runDefault()
	}

	command := args[0]
	switch command {
	case "help", "-h", "--help":
		showHelp()
		return nil
	case "version", "-v", "--version":
		showVersion()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func runDefault() error {
	fmt.Println("HolyDB - A Database System")
	fmt.Println("Version: 0.1.0")
	fmt.Println("")
	fmt.Println("Use 'holydb help' for available commands.")
	return nil
}

func showHelp() {
	fmt.Println("HolyDB - A Database System")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  holydb [command]")
	fmt.Println("")
	fmt.Println("Available Commands:")
	fmt.Println("  help     Show help information")
	fmt.Println("  version  Show version information")
	fmt.Println("")
	fmt.Println("Use 'holydb [command] --help' for more information about a command.")
}

func showVersion() {
	fmt.Println("HolyDB version 0.1.0")
}
