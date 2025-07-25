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
	
	// Connection state
	connected     bool
	serverAddress string
	
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
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	// Initialize terminal buffer - TUI owns this
	term := terminal.NewTerminal(80, 24)
	
	// Initialize proxy with the terminal as a writer
	proxyInstance := proxy.New(term)

	model := Model{
		proxy:         proxyInstance,
		list:          l,
		terminalView:  vp,
		textInput:     ti,
		connected:     false,
		serverAddress: "twgs.geekm0nkey.com:23",
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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Calculate panel dimensions
		leftWidth := (msg.Width - 84) / 2 // 84 = 80 + 4 for middle panel
		contentHeight := msg.Height - 4   // Account for chat panel
		
		// Resize components
		m.list.SetSize(leftWidth-4, contentHeight-4)
		m.terminalView.Width = 80
		m.terminalView.Height = contentHeight - 4
		
		// Resize terminal buffer
		m.terminal.Resize(80, contentHeight-4)

	case tea.KeyMsg:
		// Handle different input modes
		switch m.inputMode {
		case InputModeMenu:
			return m.handleMenuInput(msg)
		case InputModeAddress:
			return m.handleAddressInput(msg)
		case InputModeTerminal:
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
		}

	case DisconnectMsg:
		m.proxy.Disconnect()
		m.connected = false
		m.terminal.Write([]byte("Disconnected\r\n"))
		m.inputMode = InputModeMenu

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
		// Update terminal viewport content
		terminalLines := m.convertTerminalCellsToLipgloss(80, m.terminalView.Height)
		content := strings.Join(terminalLines, "\n")
		m.terminalView.SetContent(content)
		
		// Continue listening for more updates
		cmds = append(cmds, m.listenForTerminalUpdates())
		
	case TickMsg:
		// Legacy tick handler - no longer needed with message-based updates
		cmds = append(cmds, tickCmd())
	}

	// Update components based on input mode
	var cmd tea.Cmd
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
	switch msg.String() {
	case "enter":
		address := m.textInput.Value()
		if address != "" {
			m.serverAddress = address
			m.inputMode = InputModeMenu
			m.textInput.Blur()
			return m, func() tea.Msg { return ConnectMsg{Address: address} }
		}
	case "esc":
		m.inputMode = InputModeMenu
		m.textInput.Blur()
	}
	return m, nil
}

func (m Model) handleTerminalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.inputMode = InputModeMenu
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

// convertTerminalCellsToLipgloss converts terminal buffer cells to lipgloss styled lines
func (m Model) convertTerminalCellsToLipgloss(width, height int) []string {
	cells := m.terminal.GetCells()
	var styledLines []string
	
	for y := 0; y < height && y < len(cells); y++ {
		var lineBuilder strings.Builder
		
		// Group consecutive cells with same styling
		currentStyle := lipgloss.NewStyle()
		var currentText strings.Builder
		
		for x := 0; x < width && x < len(cells[y]); x++ {
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
			
			// If style changed, render accumulated text and start new group
			if !stylesEqual(currentStyle, cellStyle) {
				if currentText.Len() > 0 {
					lineBuilder.WriteString(currentStyle.Render(currentText.String()))
					currentText.Reset()
				}
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
	
	// Pad to height
	for len(styledLines) < height {
		styledLines = append(styledLines, "")
	}
	
	return styledLines
}

// stylesEqual compares two lipgloss styles for equality (simplified)
func stylesEqual(a, b lipgloss.Style) bool {
	// This is a simplified comparison - lipgloss doesn't expose style internals easily
	// For now, we'll group characters individually to ensure proper styling
	return false
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