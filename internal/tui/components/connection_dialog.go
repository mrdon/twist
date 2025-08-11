package components

import (
	"twist/internal/theme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ConnectionDialog represents a dialog for connecting to a server
type ConnectionDialog struct {
	form           *tview.Form
	callback       func(string)
	cancelCallback func()
}

// NewConnectionDialog creates a new connection dialog
func NewConnectionDialog(callback func(string), cancelCallback func()) *ConnectionDialog {
	cd := &ConnectionDialog{
		callback:       callback,
		cancelCallback: cancelCallback,
	}

	cd.setupComponents()
	return cd
}

// setupComponents initializes the dialog components
func (cd *ConnectionDialog) setupComponents() {
	// Create the form using theme factory
	cd.form = theme.NewForm()

	// Set title and border
	cd.form.SetTitle(" Connect to Server ")
	cd.form.SetTitleAlign(tview.AlignCenter)
	cd.form.SetBorder(true)
	cd.form.SetBorderPadding(2, 2, 2, 2) // top, bottom, left, right padding

	// Add server address field with default value
	cd.form.AddInputField("Server Address:", "192.168.1.182:23", 40, nil, nil)

	// Add buttons (Connect first for easy tab access, Cancel second)
	cd.form.AddButton("Connect", func() {
		serverAddress := cd.form.GetFormItem(0).(*tview.InputField).GetText()
		if cd.callback != nil {
			cd.callback(serverAddress)
		}
	})

	cd.form.AddButton("Cancel", func() {
		if cd.cancelCallback != nil {
			cd.cancelCallback()
		}
	})

	// Center the buttons
	cd.form.SetButtonsAlign(tview.AlignCenter)

	// Set focus to the input field
	cd.form.SetFocus(0)

	// Set up input capture to handle Enter key
	cd.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			// Activate the Connect button (first button, index 0)
			serverAddress := cd.form.GetFormItem(0).(*tview.InputField).GetText()
			if cd.callback != nil {
				cd.callback(serverAddress)
			}
			return nil // Consume the event
		}
		return event // Allow other keys to be processed normally
	})
}

// GetView returns the main view component
func (cd *ConnectionDialog) GetView() tview.Primitive {
	// Create a flex container with proper proportional centering
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false). // Top spacer (proportional)
		AddItem(tview.NewFlex().
						AddItem(nil, 0, 1, false).               // Left spacer (proportional)
						AddItem(cd.form, 60, 0, true).           // Fixed width for form (60 chars)
						AddItem(nil, 0, 1, false), 10, 0, true). // Fixed height (10 rows)
		AddItem(nil, 0, 1, false) // Bottom spacer (proportional)

	// Apply theme colors for modal overlay effect
	currentTheme := theme.Current()
	flex.SetBackgroundColor(currentTheme.DialogColors().Background)

	return flex
}

// GetForm returns the internal form component
func (cd *ConnectionDialog) GetForm() *tview.Form {
	return cd.form
}

// SetDoneFunc sets the function to call when ESC is pressed
func (cd *ConnectionDialog) SetDoneFunc(handler func()) {
	cd.form.SetCancelFunc(handler)
}
