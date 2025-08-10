package main

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	_ "twist/internal/proxy" // Import proxy package to register Connect implementation
	"twist/internal/tui"
)

func main() {
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
	app.SetInitialScript(scriptName)
	if err := app.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
