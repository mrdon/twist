package tui

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Menu bar styles
var (
	menuBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	menuItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230"))

	menuItemActiveStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("205")).
		Foreground(lipgloss.Color("0")).
		Bold(true)

	menuKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Underline(true)
)

// MenuBar represents the top menu bar component
type MenuBar struct {
	width           int
	logger          *log.Logger
	activeMenu      int  // -1 = none, 0 = File, 1 = Edit, etc.
	dropdownVisible bool
	selectedItem    int // Selected item in dropdown
}

// NewMenuBar creates a new menu bar
func NewMenuBar(logger *log.Logger) MenuBar {
	return MenuBar{
		logger:          logger,
		activeMenu:      -1,
		dropdownVisible: false,
		selectedItem:    0,
	}
}

// Update handles menu bar input and returns whether the key was handled
func (mb MenuBar) Update(msg tea.Msg) (MenuBar, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		mb.width = msg.Width
		return mb, nil, false
	case tea.KeyMsg:
		mb, cmd, handled := mb.handleKeyPress(msg)
		return mb, cmd, handled
	case tea.MouseMsg:
		mb, cmd, handled := mb.handleMouseEvent(msg)
		return mb, cmd, handled
	}
	return mb, nil, false
}

// handleKeyPress processes keyboard input for the menu bar
// Returns (updated menubar, command, handled)
func (mb MenuBar) handleKeyPress(msg tea.KeyMsg) (MenuBar, tea.Cmd, bool) {
	// If dropdown is visible, handle navigation keys
	if mb.dropdownVisible {
		switch msg.String() {
		case "esc":
			mb.logger.Printf("MenuBar: Closing dropdown")
			mb.dropdownVisible = false
			mb.activeMenu = -1
			return mb, nil, true
		case "up":
			mb.logger.Printf("MenuBar: Moving up in dropdown")
			if mb.selectedItem > 0 {
				mb.selectedItem--
			}
			return mb, nil, true
		case "down":
			mb.logger.Printf("MenuBar: Moving down in dropdown")
			menuItems := mb.getMenuItems(mb.activeMenu)
			if mb.selectedItem < len(menuItems)-1 {
				mb.selectedItem++
			}
			return mb, nil, true
		case "enter":
			mb.logger.Printf("MenuBar: Selected menu item %d", mb.selectedItem)
			// TODO: Execute menu action
			mb.dropdownVisible = false
			mb.activeMenu = -1
			return mb, nil, true
		case "left":
			mb.logger.Printf("MenuBar: Moving to previous menu")
			mb.activeMenu = (mb.activeMenu - 1 + 5) % 5
			mb.selectedItem = 0
			return mb, nil, true
		case "right":
			mb.logger.Printf("MenuBar: Moving to next menu")
			mb.activeMenu = (mb.activeMenu + 1) % 5
			mb.selectedItem = 0
			return mb, nil, true
		}
	}

	// Handle any key combination that involves Alt
	if msg.Alt || strings.HasPrefix(msg.String(), "alt") {
		mb.logger.Printf("MenuBar: Alt key detected - %q", msg.String())
		
		switch msg.String() {
		case "alt+f", "alt+F":
			mb.logger.Printf("MenuBar: File menu activated")
			mb.activeMenu = 0
			mb.dropdownVisible = true
			mb.selectedItem = 0
			return mb, nil, true
		case "alt+e", "alt+E":
			mb.logger.Printf("MenuBar: Edit menu activated")
			mb.activeMenu = 1
			mb.dropdownVisible = true
			mb.selectedItem = 0
			return mb, nil, true
		case "alt+v", "alt+V":
			mb.logger.Printf("MenuBar: View menu activated")
			mb.activeMenu = 2
			mb.dropdownVisible = true
			mb.selectedItem = 0
			return mb, nil, true
		case "alt+t", "alt+T":
			mb.logger.Printf("MenuBar: Terminal menu activated")
			mb.activeMenu = 3
			mb.dropdownVisible = true
			mb.selectedItem = 0
			return mb, nil, true
		case "alt+h", "alt+H":
			mb.logger.Printf("MenuBar: Help menu activated")
			mb.activeMenu = 4
			mb.dropdownVisible = true
			mb.selectedItem = 0
			return mb, nil, true
		case "alt":
			mb.logger.Printf("MenuBar: Alt key alone pressed - showing File menu")
			mb.activeMenu = 0
			mb.dropdownVisible = true
			mb.selectedItem = 0
			return mb, nil, true
		default:
			mb.logger.Printf("MenuBar: Unhandled Alt combination - %q", msg.String())
			return mb, nil, true // Still consume it since it's Alt-related
		}
	}
	
	// Not an Alt key, let other components handle it
	return mb, nil, false
}

// handleMouseEvent processes mouse input for the menu bar
func (mb MenuBar) handleMouseEvent(msg tea.MouseMsg) (MenuBar, tea.Cmd, bool) {
	if msg.Type == tea.MouseLeft {
		// Check if clicking on menu bar (row 0)
		if msg.Y == 0 {
			mb.logger.Printf("MenuBar: Mouse click at (%d, %d)", msg.X, msg.Y)
			
			// Determine which menu was clicked based on X position
			menuNames := []string{"File", "Edit", "View", "Terminal", "Help"}
			// Calculate actual positions based on text lengths, accounting for menu bar padding
			menuPositions := []int{1} // Start with left padding
			pos := 1 // Start with left padding
			for i := 0; i < len(menuNames)-1; i++ {
				pos += len(menuNames[i]) + 2 // +2 for spacing
				menuPositions = append(menuPositions, pos)
			}
			
			for i, pos := range menuPositions {
				if msg.X >= pos && msg.X < pos+len(menuNames[i])+2 {
					mb.logger.Printf("MenuBar: Clicked on %s menu", menuNames[i])
					mb.activeMenu = i
					mb.dropdownVisible = true
					mb.selectedItem = 0
					return mb, nil, true
				}
			}
		}
		
		// Check if clicking on dropdown (if visible)
		if mb.dropdownVisible && msg.Y > 0 {
			menuItems := mb.getMenuItems(mb.activeMenu)
			if msg.Y <= len(menuItems)+1 { // +1 for border
				itemIndex := msg.Y - 2 // Adjust for border
				if itemIndex >= 0 && itemIndex < len(menuItems) && menuItems[itemIndex] != "---" {
					mb.logger.Printf("MenuBar: Clicked on dropdown item %d", itemIndex)
					mb.selectedItem = itemIndex
					// TODO: Execute menu action
					mb.dropdownVisible = false
					mb.activeMenu = -1
					return mb, nil, true
				}
			}
		}
	}
	
	return mb, nil, false
}

// View renders the menu bar
func (mb MenuBar) View() string {
	// Define menu items with Alt key shortcuts
	menuItems := []string{
		menuItemStyle.Render(menuKeyStyle.Render("F") + "ile"),
		menuItemStyle.Render(menuKeyStyle.Render("E") + "dit"),
		menuItemStyle.Render(menuKeyStyle.Render("V") + "iew"),
		menuItemStyle.Render(menuKeyStyle.Render("T") + "erminal"),
		menuItemStyle.Render(menuKeyStyle.Render("H") + "elp"),
	}
	
	menuContent := strings.Join(menuItems, "  ")
	
	menuBar := menuBarStyle.
		Width(mb.width).
		Render(menuContent)

	// If dropdown is visible, add it below the menu bar
	if mb.dropdownVisible {
		dropdown := mb.renderDropdown()
		return menuBar + "\n" + dropdown
	}

	return menuBar
}

// getMenuItems returns the items for a specific menu
func (mb MenuBar) getMenuItems(menuIndex int) []string {
	switch menuIndex {
	case 0: // File
		return []string{"New", "Open...", "Save", "Save As...", "---", "Exit"}
	case 1: // Edit
		return []string{"Undo", "Redo", "---", "Cut", "Copy", "Paste", "---", "Find..."}
	case 2: // View
		return []string{"Full Screen", "Zoom In", "Zoom Out", "---", "Show Terminal", "Show Chat"}
	case 3: // Terminal
		return []string{"Connect...", "Disconnect", "---", "Clear Screen", "Reset"}
	case 4: // Help
		return []string{"About", "Help Contents", "---", "Keyboard Shortcuts"}
	default:
		return []string{}
	}
}

// renderDropdown creates the dropdown menu
func (mb MenuBar) renderDropdown() string {
	if !mb.dropdownVisible || mb.activeMenu < 0 {
		return ""
	}

	menuItems := mb.getMenuItems(mb.activeMenu)
	
	// Calculate position - place dropdown under the appropriate menu item
	// Account for menu bar left padding (1 space) + menu text lengths + spacing
	menuStartPos := 1 // Start with menu bar left padding
	menuTexts := []string{"File", "Edit", "View", "Terminal", "Help"}
	for i := 0; i < mb.activeMenu; i++ {
		menuStartPos += len(menuTexts[i]) + 2 // +2 for "  " spacing between menus
	}
	mb.logger.Printf("MenuBar dropdown positioning: activeMenu=%d, menuStartPos=%d (including padding), menuText=%s", 
		mb.activeMenu, menuStartPos, menuTexts[mb.activeMenu])
	
	var dropdownLines []string
	
	for i, item := range menuItems {
		if item == "---" {
			dropdownLines = append(dropdownLines, strings.Repeat("─", 20))
		} else {
			var line string
			if i == mb.selectedItem {
				// Highlight selected item
				line = menuItemActiveStyle.Render(" " + item + strings.Repeat(" ", 18-len(item)))
			} else {
				line = menuItemStyle.Render(" " + item + strings.Repeat(" ", 18-len(item)))
			}
			dropdownLines = append(dropdownLines, line)
		}
	}
	
	// Create bordered dropdown
	dropdownContent := strings.Join(dropdownLines, "\n")
	dropdown := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("0")).
		Width(20).
		Render(dropdownContent)
	
	// Add spacing to position the dropdown
	spacing := strings.Repeat(" ", menuStartPos)
	return spacing + dropdown
}

