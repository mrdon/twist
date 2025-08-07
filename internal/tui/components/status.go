package components

import (
	"fmt"
	"strings"
	"twist/internal/api"
	"twist/internal/theme"

	"github.com/rivo/tview"
)

// StatusComponent manages the bottom status bar
type StatusComponent struct {
	wrapper       *tview.TextView
	proxyAPI      api.ProxyAPI
	connected     bool
	serverAddress string
	gameInfo      *GameInfo // Current active game information
}

// GameInfo holds information about the currently active game
type GameInfo struct {
	GameName   string
	ServerHost string
	ServerPort string
	IsLoaded   bool
}

// NewStatusComponent creates a new status bar component
func NewStatusComponent() *StatusComponent {
	// Create status bar as TextView with traditional styling
	statusBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetWrap(false)
	
	// Get theme colors for status bar
	currentTheme := theme.Current()
	statusColors := currentTheme.StatusColors()
	
	// Set background and text color using themed colors
	statusBar.SetBackgroundColor(statusColors.Background)
	statusBar.SetTextColor(statusColors.Foreground)
	
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

// SetProxyAPI sets the proxy API interface
func (sc *StatusComponent) SetProxyAPI(proxyAPI api.ProxyAPI) {
	sc.proxyAPI = proxyAPI
	sc.UpdateStatus()
}

// SetConnectionStatus sets the connection status
func (sc *StatusComponent) SetConnectionStatus(connected bool, serverAddress string) {
	sc.connected = connected
	sc.serverAddress = serverAddress
	sc.UpdateStatus()
}

// SetGameInfo sets the active game information
func (sc *StatusComponent) SetGameInfo(gameName, serverHost, serverPort string, isLoaded bool) {
	
	if isLoaded {
		sc.gameInfo = &GameInfo{
			GameName:   gameName,
			ServerHost: serverHost,
			ServerPort: serverPort,
			IsLoaded:   isLoaded,
		}
	} else {
		sc.gameInfo = nil
	}
	sc.UpdateStatus()
}

// UpdateStatus updates the status bar display
func (sc *StatusComponent) UpdateStatus() {
	var statusText strings.Builder
	
	// Get theme colors for status bar
	currentTheme := theme.Current()
	statusColors := currentTheme.StatusColors()
	
	// Set the overall status bar colors using theme
	sc.wrapper.SetBackgroundColor(statusColors.Background)
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
	
	// Add active game information if available
	if sc.gameInfo != nil && sc.gameInfo.IsLoaded {
		statusText.WriteString(" | Game: ")
		statusText.WriteString(fmt.Sprintf("[%s]%s[-]", 
			statusColors.ConnectedFg.String(), sc.gameInfo.GameName))
	} else {
	}
	
	// Script status - use ProxyAPI instead of direct script manager access
	if sc.proxyAPI != nil && sc.proxyAPI.IsConnected() {
		scriptStatus := sc.proxyAPI.GetScriptStatus()
		statusText.WriteString(" | Scripts: ")
		statusText.WriteString(fmt.Sprintf("%d active", scriptStatus.ActiveCount))
		
		if scriptStatus.TotalCount > scriptStatus.ActiveCount {
			statusText.WriteString(fmt.Sprintf(", %d stopped", 
				scriptStatus.TotalCount - scriptStatus.ActiveCount))
		}
	} else {
		statusText.WriteString(" | Scripts: Not available")
	}
	
	
	statusText.WriteString(" | F1=Help")
	
	sc.wrapper.SetText(statusText.String())
}