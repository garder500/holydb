package holydb

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/garder500/holydb/internal/server"
)

const versionString = "0.1.0"

// Execute is the main entry point for the holydb command
func Execute() error {
	// We implement a manual lightweight subcommand system using FlagSet.
	args := os.Args[1:]
	if len(args) == 0 { // no command -> root usage
		showHelp()
		return nil
	}

	// support "help" and "help <cmd>"
	if args[0] == "help" {
		if len(args) == 1 {
			showHelp()
			return nil
		}
		return showSubcommandHelp(args[1])
	}

	// root flags (allow: --version)
	if strings.HasPrefix(args[0], "-") { // flags without command
		rootFs := flag.NewFlagSet("holydb", flag.ExitOnError)
		ver := rootFs.Bool("version", false, "show version and exit")
		rootFs.Usage = func() { printRootUsage() }
		if err := rootFs.Parse(args); err != nil {
			return err
		}
		if *ver {
			showVersion()
			return nil
		}
		showHelp()
		return nil
	}

	switch args[0] {
	case "version", "-v", "--version":
		showVersion()
		return nil
	case "serve":
		return execServe(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

// execServe parses serve flags and runs the server.
func execServe(argv []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "address to listen on")
	root := fs.String("root", ".", "storage root directory")
	background := fs.Bool("background", false, "run server in background (detached)")
	fs.Usage = func() { printServeUsage(fs) }
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if *background && os.Getenv("HOLYDB_BACKGROUND") != "1" {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		args := []string{exe, "serve", "--addr=" + *addr, "--root=" + *root, "--background=false"}
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
}

func runDefault() error { // retained for backwards test compatibility
	showHelp()
	return nil
}

func showHelp() { printRootUsage() }

func showVersion() { fmt.Printf("holydb version %s\n", versionString) }

func printRootUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf("HolyDB - A Database System\n\n")
	fmt.Printf("Usage:\n  %s [command] [flags]\n\n", exe)
	fmt.Println("Commands:")
	fmt.Println("  serve      Start HTTP storage server")
	fmt.Println("  version    Show version information")
	fmt.Println("  help       Show help (also: 'help <command>')")
	fmt.Println("")
	fmt.Println("Global Flags:")
	fmt.Println("  -version    Show version and exit")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Printf("  %s serve --addr :9000 --root ./data\n", exe)
	fmt.Printf("  %s --version\n", exe)
	fmt.Printf("  %s help serve\n", exe)
	fmt.Println("")
	fmt.Printf("Run '%s serve -h' for server flags.\n", exe)
}

func printServeUsage(fs *flag.FlagSet) {
	exe := filepath.Base(os.Args[0])
	fmt.Printf("Usage: %s serve [flags]\n\n", exe)
	fmt.Println("Starts the HolyDB storage HTTP server.")
	fmt.Println("Flags:")
	fs.PrintDefaults()
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Printf("  %s serve --addr :8080 --root ./data\n", exe)
	fmt.Printf("  %s serve --background --addr 0.0.0.0:8080\n", exe)
}

func showSubcommandHelp(cmd string) error {
	switch cmd {
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		// duplicate flag definitions for help display only
		fs.String("addr", ":8080", "address to listen on")
		fs.String("root", ".", "storage root directory")
		fs.Bool("background", false, "run server in background (detached)")
		printServeUsage(fs)
		return nil
	case "version":
		fmt.Println("Shows version information.")
		return nil
	case "help":
		showHelp()
		return nil
	default:
		return fmt.Errorf("unknown command for help: %s", cmd)
	}
}
