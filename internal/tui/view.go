package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Panel styles
	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1)

	chatPanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1)

	// Text styles
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	statusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
)

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Debug the actual terminal dimensions
	if m.logger != nil {
		m.logger.Printf("VIEW DEBUG - Terminal dimensions: width=%d, height=%d", m.width, m.height)
	}

	// Let Bubble Tea handle all layout calculations
	if m.logger != nil {
		m.logger.Printf("VIEW DEBUG - Terminal dimensions: width=%d, height=%d", m.width, m.height)
	}

	// Render panels based on current mode
	var leftPanel, middlePanel, rightPanel string

	// Left panel - always shows menu or info (with border)  
	if m.inputMode == InputModeMenu {
		leftPanel = panelStyle.Render(m.list.View())
	} else {
		leftPanel = panelStyle.Render(m.renderTraderInfo())
	}

	// Middle panel - shows terminal, connection prompt, or address input
	switch m.inputMode {
	case InputModeAddress:
		middlePanel = lipgloss.NewStyle().
			Width(80).
			Padding(1).
			Render(m.renderConnectionPrompt())
	case InputModeTerminal:
		if m.connected {
			// No border for terminal content, but with padding and fixed width
			middlePanel = lipgloss.NewStyle().
				Width(80).
				Padding(1).
				Render(m.terminalView.View())
		} else {
			middlePanel = lipgloss.NewStyle().
				Width(80).
				Padding(1).
				Render("Not connected")
		}
	default:
		middlePanel = lipgloss.NewStyle().
			Width(80).
			Padding(1).
			Render(titleStyle.Render("Terminal") + "\n\nPress Enter on 'Connect to Server' to begin")
	}

	// Right panel - sector info (with border)
	rightPanel = panelStyle.Render(m.renderSectorInfo())

	// Create menu bar
	menuBar := m.menuBar.View()
	
	// Create main content row
	mainRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		middlePanel,
		rightPanel,
	)
	
	// Combine menu bar and main content
	view := lipgloss.JoinVertical(lipgloss.Left, menuBar, mainRow)

	// Debug final view dimensions
	if m.logger != nil {
		viewLines := strings.Split(view, "\n")
		m.logger.Printf("VIEW DEBUG - Final view has %d lines, expected terminal height was %d", 
			len(viewLines), m.height)
		if len(viewLines) > 0 {
			m.logger.Printf("VIEW DEBUG - First line length: %d, last line length: %d", 
				len(viewLines[0]), len(viewLines[len(viewLines)-1]))
		}
	}

	return view
}

func (m Model) renderTraderInfo() string {
	content := []string{
		titleStyle.Render("Trader Info"),
		"",
		"Sector:     5379",
		"Turns:      150", 
		"Experience: 1087",
		"Alignment:  -33",
		"Credits:    142,439",
		"",
		titleStyle.Render("Holds"),
		"",
		"Total:      150",
		"Fuel Ore:   0",
		"Organics:   0",
		"Equipment:  150", 
		"Colonists:  0",
		"Empty:      0",
		"",
		titleStyle.Render("Quick Query"),
		"",
		"[Input field here]",
		"",
		titleStyle.Render("Stats"),
		"",
		"Profit:     0",
	}

	return strings.Join(content, "\n")
}

func (m Model) renderConnectionPrompt() string {
	content := []string{
		titleStyle.Render("Connect to Server"),
		"",
		"Enter server address:",
		m.textInput.View(),
		"",
		statusStyle.Render("Press Enter to connect, Esc to cancel"),
	}
	return strings.Join(content, "\n")
}

func (m Model) renderSectorInfo() string {
	content := []string{
		titleStyle.Render("Sector 5379"),
		"",
		"Port:       Class 9 (Stardock)",
		"Density:    101 Fighters",
		"NavHaz:     0%",
		"Anom:       No",
		"",
		titleStyle.Render("Visual Map"),
		"",
		"        *     *",
		"    *       *   *",
		"  *   * [5379] *",
		"    *   *   *",
		"      *   *",
		"",
		titleStyle.Render("Notepad"),
		"",
		"Notes will appear here...",
	}

	return strings.Join(content, "\n")
}

func (m Model) renderChatPanel(width, height int) string {
	content := []string{
		titleStyle.Render("Communications"),
	}

	// Add recent chat lines
	startIdx := len(m.chatLines) - (height - 1)
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(m.chatLines) && len(content) < height; i++ {
		line := m.chatLines[i]
		if len(line) > width {
			line = line[:width-3] + "..."
		}
		content = append(content, line)
	}

	// Pad to height
	for len(content) < height {
		content = append(content, "")
	}

	return strings.Join(content, "\n")
}

