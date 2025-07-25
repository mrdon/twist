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

	// Calculate panel dimensions - middle panel fixed at 80 chars
	middleWidth := 84 // 80 content + 4 for borders/padding
	remainingWidth := m.width - middleWidth
	leftWidth := remainingWidth / 2
	rightWidth := remainingWidth - leftWidth
	contentHeight := m.height - 4 // Reserve space for chat panel

	// Render panels based on current mode
	var leftPanel, middlePanel, rightPanel string

	// Left panel - always shows menu or info
	if m.inputMode == InputModeMenu {
		leftPanel = panelStyle.
			Width(leftWidth-2).
			Height(contentHeight-2).
			Render(m.list.View())
	} else {
		leftPanel = panelStyle.
			Width(leftWidth-2).
			Height(contentHeight-2).
			Render(m.renderTraderInfo())
	}

	// Middle panel - shows terminal, connection prompt, or address input
	switch m.inputMode {
	case InputModeAddress:
		middlePanel = panelStyle.
			Width(middleWidth-2).
			Height(contentHeight-2).
			Render(m.renderConnectionPrompt())
	case InputModeTerminal:
		if m.connected {
			middlePanel = panelStyle.
				Width(middleWidth-2).
				Height(contentHeight-2).
				Render(m.terminalView.View())
		} else {
			middlePanel = panelStyle.
				Width(middleWidth-2).
				Height(contentHeight-2).
				Render("Not connected")
		}
	default:
		middlePanel = panelStyle.
			Width(middleWidth-2).
			Height(contentHeight-2).
			Render(titleStyle.Render("Terminal") + "\n\nPress Enter on 'Connect to Server' to begin")
	}

	// Right panel - sector info
	rightPanel = panelStyle.
		Width(rightWidth-2).
		Height(contentHeight-2).
		Render(m.renderSectorInfo())

	// Create top row
	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		middlePanel,
		rightPanel,
	)

	// Create chat panel
	chatPanel := m.renderChatPanel(m.width-4, 2)
	chatRow := chatPanelStyle.
		Width(m.width-2).
		Height(4).
		Render(chatPanel)

	// Combine panels
	view := lipgloss.JoinVertical(lipgloss.Left, topRow, chatRow)

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

