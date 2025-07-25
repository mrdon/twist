package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
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

	// Initialize the TUI first so it can own the terminal buffer
	tuiModel := tui.New()

	// Start the Bubble Tea program with fallback options
	var p *tea.Program
	if isatty.IsTerminal(os.Stdin.Fd()) {
		// Remove mouse capture to allow native terminal selection
		p = tea.NewProgram(tuiModel, tea.WithAltScreen())
	} else {
		// Fallback for non-interactive environments
		p = tea.NewProgram(tuiModel)
	}

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}