package components

import (
	"twist/internal/theme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// BurstInputDialog represents a dialog for entering burst commands
type BurstInputDialog struct {
	form           *tview.Form
	callback       func(string)
	cancelCallback func()
}

// NewBurstInputDialog creates a new burst input dialog
func NewBurstInputDialog(callback func(string), cancelCallback func()) *BurstInputDialog {
	bid := &BurstInputDialog{
		callback:       callback,
		cancelCallback: cancelCallback,
	}

	bid.setupComponents()
	return bid
}

// setupComponents initializes the dialog components
func (bid *BurstInputDialog) setupComponents() {
	// Create the form using theme factory
	bid.form = theme.NewForm()

	// Set title and border
	bid.form.SetTitle(" Burst Command ")
	bid.form.SetTitleAlign(tview.AlignCenter)
	bid.form.SetBorder(true)
	bid.form.SetBorderPadding(2, 2, 2, 2) // top, bottom, left, right padding

	// Add help text as a text view
	helpText := "Enter burst text to send to server.\nUse '*' character for ENTER (e.g., 'lt1*' lists trader #1)\nExamples: 'bp100*' (buy 100 product), 'sp50*' (sell 50 product), 'tw1234*' (transwarp to sector 1234)"
	bid.form.AddTextView("Help", helpText, 0, 3, true, false)

	// Add burst command input field
	bid.form.AddInputField("Burst Command:", "", 60, nil, nil)

	// Add buttons (Send first for easy access, Cancel second)
	bid.form.AddButton("Send", func() {
		burstText := bid.form.GetFormItem(1).(*tview.InputField).GetText()
		if burstText != "" && bid.callback != nil {
			bid.callback(burstText)
		}
	})

	bid.form.AddButton("Cancel", func() {
		if bid.cancelCallback != nil {
			bid.cancelCallback()
		}
	})

	// Set up escape key handling
	bid.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			if bid.cancelCallback != nil {
				bid.cancelCallback()
			}
			return nil // Consume the event
		}
		return event // Pass through other keys
	})

	// Set focus to the input field
	bid.form.SetFocus(1)
}

// SetDoneFunc sets a function to call when the dialog should be closed
func (bid *BurstInputDialog) SetDoneFunc(handler func()) InputDialog {
	// This is used by the main app for ESC key handling consistency
	bid.form.SetCancelFunc(handler)
	return bid
}

// GetView returns the main view component
func (bid *BurstInputDialog) GetView() tview.Primitive {
	// Create a flex container with proper proportional centering
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false). // Top spacer (proportional)
		AddItem(tview.NewFlex().
						AddItem(nil, 0, 1, false).               // Left spacer (proportional)
						AddItem(bid.form, 70, 0, true).          // Fixed width for form (70 chars for longer input)
						AddItem(nil, 0, 1, false), 12, 0, true). // Fixed height (12 rows for help text)
		AddItem(nil, 0, 1, false) // Bottom spacer (proportional)

	// Apply theme colors for modal overlay effect
	currentTheme := theme.Current()
	flex.SetBackgroundColor(currentTheme.DialogColors().Background)

	return flex
}

// GetForm returns the underlying tview.Form for display
func (bid *BurstInputDialog) GetForm() *tview.Form {
	return bid.form
}
