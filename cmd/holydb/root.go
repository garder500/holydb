package holydb

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/garder500/holydb/internal/server"
)

// Execute is the main entry point for the holydb command
func Execute() error {
	// Parse any global flags (none currently) and use flag.Args() for positional args.
	flag.Parse()
	args := flag.Args()

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
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		addr := fs.String("addr", ":8080", "address to listen on")
		root := fs.String("root", ".", "storage root directory")
		background := fs.Bool("background", false, "run server in background (detached)")
		// parse remaining args (skip the command itself)
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		// If requested, start detached background process and exit parent.
		if *background && os.Getenv("HOLYDB_BACKGROUND") != "1" {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			args := []string{exe, "serve", "--addr=" + *addr, "--root=" + *root, "--background=false"}
			// redirect stdio to /dev/null
			devnull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
			if err != nil {
				return err
			}
			attr := &os.ProcAttr{
				Dir:   "",
				Env:   append(os.Environ(), "HOLYDB_BACKGROUND=1"),
				Files: []*os.File{devnull, devnull, devnull},
				Sys:   &syscall.SysProcAttr{Setsid: true},
			}
			proc, err := os.StartProcess(exe, args, attr)
			if err != nil {
				devnull.Close()
				return err
			}
			devnull.Close()
			fmt.Printf("started background process pid=%d\n", proc.Pid)
			return nil
		}
		fmt.Printf("Starting server on %s, root=%s\n", *addr, *root)
		return server.Run(server.Config{Addr: *addr, Root: *root})
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
	fmt.Println("  serve    Start HTTP storage server (see flags below)")
	fmt.Println("")
	fmt.Println("Use 'holydb [command] --help' for more information about a command.")
	fmt.Println("")
	fmt.Println("Serve flags:")
	fmt.Println("  --addr=:8080    Address to listen on (default :8080)")
	fmt.Println("  --root=./data   Storage root directory (default .)")
}

func showVersion() {
	fmt.Println("HolyDB version 0.1.0")
}
