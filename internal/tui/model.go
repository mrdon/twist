package tui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"twist/internal/proxy"
	"twist/internal/terminal"
)

type Model struct {
	proxy         *proxy.Proxy
	width         int
	height        int
	
	// Bubble Tea components
	list          list.Model
	terminalView  viewport.Model
	textInput     textinput.Model
	menuBar       MenuBar
	
	// Connection state
	connected     bool
	serverAddress string
	firstConnection bool  // Track if this is the first connection
	
	// Input handling
	inputMode     InputMode
	
	// Terminal buffer
	terminal      *terminal.Terminal
	
	// Chat content
	chatLines     []string
	
	// Logger for debugging
	logger        *log.Logger
	
	// Terminal update notification
	terminalUpdateChan chan struct{}
}

type InputMode int

const (
	InputModeMenu InputMode = iota
	InputModeTerminal
	InputModeAddress
)

// MenuItem represents a menu item for the list component
type MenuItem struct {
	title string
	desc  string
}

func (i MenuItem) Title() string       { return i.title }
func (i MenuItem) Description() string { return i.desc }
func (i MenuItem) FilterValue() string { return i.title }


type ConnectMsg struct {
	Address string
}

type DisconnectMsg struct{}

type OutputMsg struct {
	Content string
}

type ErrorMsg struct {
	Error error
}

type TickMsg struct{}

type TerminalUpdateMsg struct{}

// Generate tick messages for batched updates
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*16, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

func New() Model {
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	
	logger := log.New(logFile, "[TUI] ", log.LstdFlags|log.Lshortfile)
	logger.Println("TUI initialized")
	
	// Initialize list component
	items := []list.Item{
		MenuItem{title: "Connect to Server", desc: "Connect to a Trade Wars server"},
		MenuItem{title: "Disconnect", desc: "Disconnect from current server"},
		MenuItem{title: "Quit", desc: "Exit the application"},
	}
	
	// Create a custom delegate for better styling
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("205")).
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("205")).
		Foreground(lipgloss.Color("241")).
		Padding(0, 0, 0, 1)
	
	l := list.New(items, delegate, 0, 0)
	l.Title = "Main Menu"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)
	
	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Enter server address..."
	
	// Initialize viewport for terminal content
	vp := viewport.New(80, 24)
	// Remove border to show all content

	// Initialize terminal buffer - TUI owns this (50 lines + scrollback)
	term := terminal.NewTerminal(80, 50)
	
	// Initialize proxy with the terminal as a writer
	proxyInstance := proxy.New(term)

	model := Model{
		proxy:         proxyInstance,
		list:          l,
		terminalView:  vp,
		textInput:     ti,
		menuBar:       NewMenuBar(logger),
		connected:     false,
		serverAddress: "twgs.geekm0nkey.com:23",
		firstConnection: true,  // Initialize as first connection
		inputMode:     InputModeMenu,
		terminal:      term,
		chatLines:     []string{"Chat messages will appear here"},
		logger:        logger,
		terminalUpdateChan: make(chan struct{}, 100),
	}
	
	// Set up terminal update callback
	term.SetUpdateCallback(func() {
		logger.Printf("Terminal update callback triggered")
		select {
		case model.terminalUpdateChan <- struct{}{}:
			logger.Printf("Sent terminal update message")
		default:
			logger.Printf("Terminal update channel full, skipping")
		}
	})
	
	return model
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.listenForErrors(),
		m.listenForTerminalUpdates(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Debug all messages
	m.logger.Printf("Update() called with message type: %T", msg)

	// Always update menu bar first (it handles Alt key filtering)
	var menuBarHandled bool
	var cmd tea.Cmd
	m.logger.Printf("Calling MenuBar.Update() with message: %T", msg)
	m.menuBar, cmd, menuBarHandled = m.menuBar.Update(msg)
	m.logger.Printf("MenuBar returned handled=%t", menuBarHandled)
	cmds = append(cmds, cmd)
	
	// If menu bar handled the message, don't process it further
	if menuBarHandled {
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Let Bubble Tea handle component sizing automatically
		// Only set the terminal to standard 80 columns
		m.terminalView.Width = 80
		m.terminalView.Height = msg.Height
		
		// Resize terminal buffer to match viewport
		m.terminal.Resize(80, msg.Height)

	case tea.KeyMsg:
		// Debug log current mode and key
		m.logger.Printf("KeyMsg received - Mode: %d, Key: %q", m.inputMode, msg.String())
		// Handle different input modes
		switch m.inputMode {
		case InputModeMenu:
			m.logger.Printf("Handling menu input")
			return m.handleMenuInput(msg)
		case InputModeAddress:
			m.logger.Printf("Handling address input")
			return m.handleAddressInput(msg)
		case InputModeTerminal:
			m.logger.Printf("Handling terminal input")
			return m.handleTerminalInput(msg)
		}

	case ConnectMsg:
		m.logger.Printf("Attempting to connect to %s", msg.Address)
		err := m.proxy.Connect(msg.Address)
		if err != nil {
			m.logger.Printf("Connection failed: %v", err)
			// Connection failed, don't have access to terminal yet
			// Error will be shown in the menu view
		} else {
			m.logger.Printf("Successfully connected to %s", msg.Address)
			m.connected = true
			connText := fmt.Sprintf("Connected to %s\r\n", msg.Address)
			m.terminal.Write([]byte(connText))
			m.inputMode = InputModeTerminal
			
			// Ensure viewport starts at bottom for new connections
			m.terminalView.GotoBottom()
			m.logger.Printf("Set viewport to bottom on initial connection")
		}

	case DisconnectMsg:
		m.proxy.Disconnect()
		m.connected = false
		m.terminal.Write([]byte("Disconnected\r\n"))
		m.inputMode = InputModeMenu
		m.firstConnection = true  // Reset for next connection

	case OutputMsg:
		// This case is no longer used with the streaming pipeline
		m.logger.Printf("Received legacy output: %q", msg.Content)

	case ErrorMsg:
		m.logger.Printf("Received error: %v", msg.Error)
		// Write error to terminal buffer
		errorText := fmt.Sprintf("Error: %v\r\n", msg.Error)
		m.terminal.Write([]byte(errorText))
		
		// Continue listening for more errors
		cmds = append(cmds, m.listenForErrors())
		
	case TerminalUpdateMsg:
		m.logger.Printf("Processing TerminalUpdateMsg - updating viewport")
		// Update terminal viewport content - include scrollback + current buffer for scrolling
		cells := m.terminal.GetAllCells()
		
		// Debug the raw terminal buffer for the last few lines
		if len(cells) >= 3 {
			lastCellLine := len(cells) - 1
			for lineIdx := lastCellLine - 2; lineIdx <= lastCellLine; lineIdx++ {
				if lineIdx >= 0 && lineIdx < len(cells) {
					rawLine := ""
					for _, cell := range cells[lineIdx] {
						if cell.Char != 0 {
							rawLine += string(cell.Char)
						}
					}
					m.logger.Printf("Raw cells for line %d: %q", lineIdx, strings.TrimRight(rawLine, " "))
				}
			}
		}
		
		// Convert all cells to styled lines - don't limit by height since we want all content
		allTerminalLines := m.convertAllTerminalCellsToLipgloss(cells, 80)
		m.logger.Printf("Converted %d lines from terminal cells", len(allTerminalLines))
		
		// Safety check - ensure we have content
		if len(allTerminalLines) == 0 {
			allTerminalLines = []string{"No terminal content available"}
			m.logger.Printf("Warning: Empty terminal content, using placeholder")
		}
		
		// Calculate valid scroll bounds after content update
		// For N total lines and H viewport height:
		// - Lines are indexed 0 to N-1
		// - At offset 0, we see lines 0 to H-1  
		// - At max offset, we should see lines (N-H) to N-1
		// - So maxOffset = N - H, but ensure it's not negative
		maxOffset := len(allTerminalLines) - m.terminalView.Height
		if maxOffset < 0 {
			maxOffset = 0
		}
		m.logger.Printf("Content updated - TotalLines: %d, ViewportHeight: %d, MaxOffset: %d", 
			len(allTerminalLines), m.terminalView.Height, maxOffset)
		
		// Always show all content - let the terminal buffer handle the memory management
		// The terminal buffer already has its own scrollback limit (1000 lines in buffer.go)
		content := strings.Join(allTerminalLines, "\n")
		m.terminalView.SetContent(content)
		
		// Auto-scroll to bottom of content
		m.terminalView.GotoBottom()
		m.logger.Printf("Set terminal content: %d total lines, offset: %d", 
			len(allTerminalLines), m.terminalView.YOffset)
		
		// Debug viewport bounds to check for off-by-one errors
		newOffset := m.terminalView.YOffset
		visibleStart := newOffset
		visibleEnd := newOffset + m.terminalView.Height - 1
		actualMaxOffset := len(allTerminalLines) - m.terminalView.Height
		m.logger.Printf("Auto-scrolled to bottom: YOffset=%d, ViewportHeight=%d, TotalLines=%d", 
			newOffset, m.terminalView.Height, len(allTerminalLines))
		m.logger.Printf("Expected MaxOffset=%d, Actual YOffset after GotoBottom=%d", actualMaxOffset, newOffset)
		m.logger.Printf("Visible range: lines %d-%d of %d total (should see line %d)", 
			visibleStart, visibleEnd, len(allTerminalLines)-1, len(allTerminalLines)-1)
		
		// Debug first few lines content to verify they're there
		if len(allTerminalLines) >= 3 {
			for i := 0; i < 3; i++ {
				line := allTerminalLines[i]
				if len(line) > 80 { line = line[:80] + "..." }
				m.logger.Printf("Line %d: %q", i, line)
			}
		}
		
		// Debug: Check what the viewport actually thinks it's displaying
		viewportContent := m.terminalView.View()
		viewportLines := strings.Split(viewportContent, "\n")
		m.logger.Printf("Viewport reports %d lines in View(), last line: %q", 
			len(viewportLines), 
			func() string {
				if len(viewportLines) > 0 {
					lastLine := viewportLines[len(viewportLines)-1]
					if len(lastLine) > 100 { return lastLine[:100] + "..." }
					return lastLine
				}
				return "NO_LINES"
			}())
		
		// Log the last few lines to see what should be visible
		if len(allTerminalLines) >= 3 {
			lastLine := len(allTerminalLines) - 1
			truncate := func(s string, maxLen int) string {
				if len(s) > maxLen { return s[:maxLen] }
				return s
			}
			m.logger.Printf("Last 3 lines: [%d]=%q (len=%d), [%d]=%q (len=%d), [%d]=%q (len=%d)", 
				lastLine-2, truncate(allTerminalLines[lastLine-2], 50), len(allTerminalLines[lastLine-2]),
				lastLine-1, truncate(allTerminalLines[lastLine-1], 50), len(allTerminalLines[lastLine-1]),
				lastLine, truncate(allTerminalLines[lastLine], 50), len(allTerminalLines[lastLine]))
			
			// Check for pause prompt in ALL lines (last 10 to be thorough)
			pauseFound := false
			searchStart := len(allTerminalLines) - 10
			if searchStart < 0 { searchStart = 0 }
			
			for i := searchStart; i < len(allTerminalLines); i++ {
				line := allTerminalLines[i]
				// Remove ANSI codes for easier searching
				cleanLine := strings.ReplaceAll(line, "\x1b", "ESC")
				if strings.Contains(cleanLine, "Pause") || strings.Contains(cleanLine, "Press") || strings.Contains(cleanLine, "continue") {
					m.logger.Printf("*** PAUSE FOUND in line %d: FULL LINE: %q", i, line)
					pauseFound = true
				}
				// Also log dotted lines
				if strings.Contains(cleanLine, "·") || strings.Contains(cleanLine, "...") {
					maxLen := 50
					if len(cleanLine) < maxLen { maxLen = len(cleanLine) }
					m.logger.Printf("*** DOTS FOUND in line %d: %q", i, cleanLine[:maxLen])
				}
			}
			if !pauseFound {
				m.logger.Printf("*** PAUSE NOT FOUND in last 10 lines (total: %d)", len(allTerminalLines))
			}
		}
		
		// Continue listening for more updates
		cmds = append(cmds, m.listenForTerminalUpdates())
		
	case TickMsg:
		// Legacy tick handler - no longer needed with message-based updates
		cmds = append(cmds, tickCmd())
	}

	// Update underlying components for non-key messages
	switch m.inputMode {
	case InputModeMenu:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	case InputModeAddress:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	case InputModeTerminal:
		m.terminalView, cmd = m.terminalView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}


func (m Model) handleMenuInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "enter":
		selected := m.list.SelectedItem()
		if item, ok := selected.(MenuItem); ok {
			switch item.title {
			case "Connect to Server":
				m.inputMode = InputModeAddress
				m.textInput.SetValue(m.serverAddress)
				m.textInput.Focus()
			case "Disconnect":
				return m, func() tea.Msg { return DisconnectMsg{} }
			case "Quit":
				return m, tea.Quit
			}
		}
	case "esc":
		if m.connected {
			m.inputMode = InputModeTerminal
		}
	}
	return m, nil
}

func (m Model) handleAddressInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.logger.Printf("Address input - Key: %q, Current value: %q", msg.String(), m.textInput.Value())
	
	switch msg.String() {
	case "enter":
		address := m.textInput.Value()
		m.logger.Printf("Enter pressed - address: %q", address)
		if address != "" {
			m.serverAddress = address
			m.inputMode = InputModeMenu
			m.textInput.Blur()
			return m, func() tea.Msg { return ConnectMsg{Address: address} }
		} else {
			m.logger.Printf("Address is empty, not connecting")
		}
	case "esc":
		m.logger.Printf("Esc pressed, returning to menu")
		m.inputMode = InputModeMenu
		m.textInput.Blur()
	}
	return m, nil
}

func (m Model) handleTerminalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Debug logging for all key presses
	m.logger.Printf("Terminal input - Key: %q, Type: %d, Alt: %t", 
		msg.String(), msg.Type, msg.Alt)
	
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.inputMode = InputModeMenu
	case "shift+pgup":
		m.logger.Printf("Shift+PgUp detected - scrolling up 10 lines")
		m.terminalView.LineUp(10)
		return m, nil
	case "shift+pgdown":
		m.logger.Printf("Shift+PgDown detected - scrolling down 10 lines")
		m.terminalView.LineDown(10)
		return m, nil
	case "shift+up":
		m.logger.Printf("Shift+Up detected - scrolling up 1 line")
		totalLines := m.terminalView.TotalLineCount()
		currentOffset := m.terminalView.YOffset
		m.logger.Printf("Before scroll - YOffset: %d, YPosition: %d, TotalLines: %d, ViewportHeight: %d", 
			currentOffset, m.terminalView.YPosition, totalLines, m.terminalView.Height)
		
		// Only scroll up if not already at top
		if currentOffset > 0 {
			m.terminalView.LineUp(1)
		} else {
			m.logger.Printf("Already at top, cannot scroll up further")
		}
		
		newOffset := m.terminalView.YOffset
		m.logger.Printf("After scroll - YOffset: %d, YPosition: %d", newOffset, m.terminalView.YPosition)
		
		// Debug: If we think we're at the top, show what line is actually visible
		if newOffset == 0 {
			m.logger.Printf("At YOffset=0 - should be showing first line of content")
			// Check what the viewport is actually displaying
			viewportContent := m.terminalView.View()
			viewportLines := strings.Split(viewportContent, "\n")
			if len(viewportLines) > 0 {
				firstDisplayed := viewportLines[0]
				if len(firstDisplayed) > 80 { firstDisplayed = firstDisplayed[:80] + "..." }
				m.logger.Printf("VIEWPORT SHOWS as first line: %q", firstDisplayed)
			}
		}
		return m, nil
	case "shift+down":
		m.logger.Printf("Shift+Down detected - scrolling down 1 line")
		totalLines := m.terminalView.TotalLineCount()
		currentOffset := m.terminalView.YOffset
		maxOffset := totalLines - m.terminalView.Height
		if maxOffset < 0 {
			maxOffset = 0
		}
		m.logger.Printf("Before scroll - YOffset: %d, YPosition: %d, TotalLines: %d, ViewportHeight: %d, MaxOffset: %d", 
			currentOffset, m.terminalView.YPosition, totalLines, m.terminalView.Height, maxOffset)
		
		// Only scroll down if not already at bottom
		if currentOffset < maxOffset {
			m.terminalView.LineDown(1)
		} else {
			m.logger.Printf("Already at bottom, cannot scroll down further")
		}
		m.logger.Printf("After scroll - YOffset: %d, YPosition: %d", m.terminalView.YOffset, m.terminalView.YPosition)
		return m, nil
	case "pgup":
		m.logger.Printf("PgUp without shift - sending to proxy")
		if m.connected {
			m.proxy.SendInput("\x1b[5~")
		}
		return m, nil
	case "pgdown":
		m.logger.Printf("PgDown without shift - sending to proxy")
		if m.connected {
			m.proxy.SendInput("\x1b[6~")
		}
		return m, nil
	case "up":
		m.logger.Printf("Up without shift - sending to proxy")
		if m.connected {
			m.proxy.SendInput("\x1b[A")
		}
		return m, nil
	case "down":
		m.logger.Printf("Down without shift - sending to proxy")
		if m.connected {
			m.proxy.SendInput("\x1b[B")
		}
		return m, nil
	case "enter":
		// Send carriage return + line feed for telnet
		if m.connected {
			m.logger.Printf("Sending enter key")
			m.proxy.SendInput("\r\n")
		}
	default:
		// Send individual key presses to the proxy
		if m.connected {
			keyStr := msg.String()
			m.logger.Printf("Sending key: %q", keyStr)
			
			// Handle special keys that need conversion
			switch keyStr {
			case "space":
				m.proxy.SendInput(" ")
			case "tab":
				m.proxy.SendInput("\t")
			case "backspace":
				m.proxy.SendInput("\b")
			default:
				// Send regular characters as-is
				if len(keyStr) == 1 {
					m.proxy.SendInput(keyStr)
				}
			}
		}
	}
	return m, nil
}

// listenForOutput is no longer needed with streaming pipeline
// Terminal updates happen directly through the pipeline

func (m Model) listenForTerminalUpdates() tea.Cmd {
	return func() tea.Msg {
		// Block waiting for terminal updates
		m.logger.Printf("Waiting for terminal update...")
		<-m.terminalUpdateChan
		m.logger.Printf("Received terminal update, sending TerminalUpdateMsg")
		return TerminalUpdateMsg{}
	}
}

func (m Model) listenForErrors() tea.Cmd {
	return func() tea.Msg {
		// Block waiting for errors - no default case  
		err := <-m.proxy.GetErrorChan()
		return ErrorMsg{Error: err}
	}
}

// convertAllTerminalCellsToLipgloss converts ALL terminal buffer cells to lipgloss styled lines
func (m Model) convertAllTerminalCellsToLipgloss(cells [][]terminal.Cell, width int) []string {
	var styledLines []string
	
	for y := 0; y < len(cells); y++ {
		var lineBuilder strings.Builder
		
		// Group consecutive characters with same styling to avoid excessive ANSI codes
		currentStyle := lipgloss.NewStyle()
		var currentText strings.Builder
		
		for x := 0; x < len(cells[y]); x++ {
			cell := cells[y][x]
			
			// Create style for this cell
			cellStyle := lipgloss.NewStyle().
				Foreground(m.ansiToLipglossColor(cell.Foreground)).
				Background(m.ansiToLipglossColor(cell.Background))
			
			if cell.Bold {
				cellStyle = cellStyle.Bold(true)
			}
			if cell.Underline {
				cellStyle = cellStyle.Underline(true)
			}
			
			// Check if style changed or we have accumulated enough
			styleChanged := (x == 0) || !stylesEqual(currentStyle, cellStyle)
			
			if styleChanged && currentText.Len() > 0 {
				// Render accumulated text with previous style
				lineBuilder.WriteString(currentStyle.Render(currentText.String()))
				currentText.Reset()
			}
			
			if styleChanged {
				currentStyle = cellStyle
			}
			
			// Add character to current group
			if cell.Char == 0 {
				currentText.WriteRune(' ')
			} else {
				currentText.WriteRune(cell.Char)
			}
		}
		
		// Render final group
		if currentText.Len() > 0 {
			lineBuilder.WriteString(currentStyle.Render(currentText.String()))
		}
		
		styledLines = append(styledLines, strings.TrimRight(lineBuilder.String(), " "))
	}
	
	// Don't pad - return all actual content
	return styledLines
}

// stylesEqual compares two lipgloss styles for equality (simplified)
func stylesEqual(a, b lipgloss.Style) bool {
	// Simple comparison by comparing rendered output of a test character
	// This is not perfect but works for our use case
	testA := a.Render("X")
	testB := b.Render("X")
	return testA == testB
}

// ansiToLipglossColor converts ANSI color codes to lipgloss colors
func (m Model) ansiToLipglossColor(colorCode int) lipgloss.Color {
	colors := []string{
		"0",   // Black
		"1",   // Red
		"2",   // Green
		"3",   // Yellow
		"4",   // Blue
		"5",   // Magenta
		"6",   // Cyan
		"7",   // White
	}
	
	if colorCode >= 0 && colorCode < len(colors) {
		return lipgloss.Color(colors[colorCode])
	}
	
	return lipgloss.Color("7") // Default to white
}