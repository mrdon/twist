package components

import "github.com/rivo/tview"

// InputDialog defines the interface that all input dialogs must implement
// This provides a consistent way for the TUI to handle different types of input dialogs
type InputDialog interface {
	// GetView returns the main view component for display
	GetView() tview.Primitive
	
	// GetForm returns the underlying form component for focus management
	GetForm() *tview.Form
	
	// SetDoneFunc sets the function to call when the dialog should be closed (ESC key, etc.)
	SetDoneFunc(handler func()) InputDialog
}

// DialogManager provides utilities for managing input dialogs
type DialogManager struct{}

// NewDialogManager creates a new dialog manager
func NewDialogManager() *DialogManager {
	return &DialogManager{}
}

// ValidateDialog checks if a dialog implements the required interface
func (dm *DialogManager) ValidateDialog(dialog interface{}) bool {
	_, ok := dialog.(InputDialog)
	return ok
}

// CastDialog safely casts an interface{} to InputDialog
func (dm *DialogManager) CastDialog(dialog interface{}) (InputDialog, bool) {
	d, ok := dialog.(InputDialog)
	return d, ok
}