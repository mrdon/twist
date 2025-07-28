package components

import (
	"fmt"
	"strings"
	"twist/internal/scripting"

	"github.com/rivo/tview"
)

// StatusComponent manages the bottom status bar
type StatusComponent struct {
	wrapper       *tview.TextView
	scriptManager ScriptManagerInterface
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

// UpdateStatus updates the status bar display
func (sc *StatusComponent) UpdateStatus() {
	if sc.scriptManager == nil {
		sc.wrapper.SetText(" Scripts: Unavailable | Status: Error | F1=Help")
		return
	}

	engine := sc.scriptManager.GetEngine()
	if engine == nil {
		sc.wrapper.SetText(" Scripts: Engine Error | Status: Error | F1=Help")
		return
	}

	allScripts := engine.ListScripts()
	runningScripts := engine.GetRunningScripts()
	
	totalCount := len(allScripts)
	runningCount := len(runningScripts)
	
	var statusText strings.Builder
	statusText.WriteString(" Scripts: ")
	
	statusText.WriteString(fmt.Sprintf("%d active", runningCount))
	
	if totalCount > runningCount {
		statusText.WriteString(fmt.Sprintf(", %d stopped", totalCount-runningCount))
	}
	
	statusText.WriteString(" | Status: ")
	
	if runningCount > 0 {
		statusText.WriteString("Running")
	} else if totalCount > 0 {
		statusText.WriteString("Loaded")
	} else {
		statusText.WriteString("Ready")
	}
	
	statusText.WriteString(" | F1=Help")
	
	sc.wrapper.SetText(statusText.String())
}