package components

import (
	"fmt"
	"strings"
	"twist/internal/scripting"
	"twist/internal/theme"

	"github.com/rivo/tview"
)

// StatusComponent manages the bottom status bar
type StatusComponent struct {
	wrapper       *tview.TextView
	scriptManager ScriptManagerInterface
	connected     bool
	serverAddress string
}

// ScriptManagerInterface defines the interface for script management
type ScriptManagerInterface interface {
	GetEngine() *scripting.Engine
	LoadAndRunScript(filename string) error
	Stop() error
	GetStatus() map[string]interface{}
}

// NewStatusComponent creates a new status bar component
func NewStatusComponent() *StatusComponent {
	// Create status bar as TextView with traditional styling
	statusBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetWrap(false)
	
	// Set background and text color to match traditional status bars
	statusBar.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	statusBar.SetTextColor(tview.Styles.PrimaryTextColor)
	
	// Set initial status
	statusBar.SetText(" Scripts: 0 active | Status: Ready | F1=Help")

	return &StatusComponent{
		wrapper: statusBar,
	}
}

// GetWrapper returns the status bar TextView
func (sc *StatusComponent) GetWrapper() *tview.TextView {
	return sc.wrapper
}

// SetScriptManager sets the script manager interface
func (sc *StatusComponent) SetScriptManager(sm ScriptManagerInterface) {
	sc.scriptManager = sm
	sc.UpdateStatus()
}

// SetConnectionStatus sets the connection status
func (sc *StatusComponent) SetConnectionStatus(connected bool, serverAddress string) {
	sc.connected = connected
	sc.serverAddress = serverAddress
	sc.UpdateStatus()
}

// UpdateStatus updates the status bar display
func (sc *StatusComponent) UpdateStatus() {
	var statusText strings.Builder
	
	// Get theme colors for status bar
	currentTheme := theme.Current()
	statusColors := currentTheme.StatusColors()
	
	// Set the overall status bar to normal foreground color
	sc.wrapper.SetTextColor(statusColors.Foreground)
	
	// Build status text with colored connection status
	statusText.WriteString(" ")
	if sc.connected {
		// Add green "Connected" part
		statusText.WriteString(fmt.Sprintf("[%s]Connected[-] to %s", 
			statusColors.ConnectedFg.String(), sc.serverAddress))
	} else if sc.serverAddress != "" {
		// Add connecting color for server address
		statusText.WriteString(fmt.Sprintf("[%s]%s[-]", 
			statusColors.ConnectingFg.String(), sc.serverAddress))
	} else {
		// Add red "Disconnected" part
		statusText.WriteString(fmt.Sprintf("[%s]Disconnected[-]", 
			statusColors.DisconnectedFg.String()))
	}
	
	// Script status - handle case where script manager is not available
	if sc.scriptManager == nil {
		statusText.WriteString(" | Scripts: Not available")
	} else {
		engine := sc.scriptManager.GetEngine()
		if engine == nil {
			statusText.WriteString(" | Scripts: Engine error")
		} else {
			allScripts := engine.ListScripts()
			runningScripts := engine.GetRunningScripts()
			
			totalCount := len(allScripts)
			runningCount := len(runningScripts)
			
			statusText.WriteString(" | Scripts: ")
			statusText.WriteString(fmt.Sprintf("%d active", runningCount))
			
			if totalCount > runningCount {
				statusText.WriteString(fmt.Sprintf(", %d stopped", totalCount-runningCount))
			}
		}
	}
	
	
	statusText.WriteString(" | F1=Help")
	
	sc.wrapper.SetText(statusText.String())
}