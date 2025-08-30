package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"twist/internal/log"
	_ "twist/internal/proxy" // Import proxy package to register Connect implementation
	"twist/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set up global panic handler first
	defer func() {
		if r := recover(); r != nil {
			log.Error("GLOBAL PANIC recovered", "error", r, "stack", string(debug.Stack()))
			fmt.Fprintf(os.Stderr, "Application crashed. See twist_debug.log for details.\n")
			os.Exit(1)
		}
	}()
	
	// Configure debug logging to file for main application
	if err := log.SetFileOutput("twist_debug.log"); err != nil {
		fmt.Printf("Warning: Could not configure debug logging to file: %v\n", err)
	}

	// Set up signal handlers to catch segfaults and other crashes
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGSEGV, syscall.SIGABRT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	go func() {
		sig := <-signalChan
		log.Error("SIGNAL RECEIVED", "signal", sig.String(), "stack", string(debug.Stack()))
		fmt.Fprintf(os.Stderr, "Application received signal %s. See twist_debug.log for details.\n", sig.String())
		os.Exit(1)
	}()

	// Add a deadlock detector - log every 30 seconds that we're alive
	go func() {
		for {
			time.Sleep(30 * time.Second)
			log.Debug("HEARTBEAT: Application is alive")
		}
	}()

	// Check if we have a proper TTY
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Println("Trade Wars 2002 Client")
		fmt.Println("This application requires a terminal/TTY to run properly.")
		fmt.Println("Please run this in a proper terminal environment.")
		os.Exit(1)
	}

	// Get script name from command line arguments (default to empty string)
	var scriptName string
	if len(os.Args) > 1 {
		scriptName = os.Args[1]
	}

	// Initialize and run the tview application
	app := tui.NewApplication()
	app.SetVersionInfo(version, commit, date)
	app.SetInitialScript(scriptName)
	if err := app.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
