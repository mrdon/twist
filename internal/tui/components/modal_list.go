package components

import (
	"twist/internal/theme"
	
	"github.com/rivo/tview"
)

// ModalList represents a DOS-style modal list component
type ModalList struct {
	modal    *tview.Modal
	list     *tview.List
	title    string
	items    []string
	callback func(string)
}

// NewModalList creates a new DOS-style modal list
func NewModalList(title string, items []string, callback func(string)) *ModalList {
	ml := &ModalList{
		title:    title,
		items:    items,
		callback: callback,
	}

	ml.setupComponents()
	return ml
}

// setupComponents initializes the modal and list components
func (ml *ModalList) setupComponents() {
	// Create the list using theme factory for convenience
	ml.list = theme.NewList()
	
	// Set title
	ml.list.SetTitle(" " + ml.title + " ")
	ml.list.SetTitleAlign(tview.AlignLeft)

	// Add all items to the list
	for _, item := range ml.items {
		ml.list.AddItem(item, "", 0, func() {
			if ml.callback != nil {
				selectedIndex := ml.list.GetCurrentItem()
				if selectedIndex >= 0 && selectedIndex < len(ml.items) {
					ml.callback(ml.items[selectedIndex])
				}
			}
		})
	}

	// Create a flex container to center the list
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(ml.list, len(ml.items)+4, 0, true). // Height based on item count + border
			AddItem(nil, 0, 1, false), 0, 1, true).
		AddItem(nil, 0, 1, false)

	// Apply theme colors for modal overlay
	currentTheme := theme.Current()
	flex.SetBackgroundColor(currentTheme.TerminalColors().Background)

	// Create modal wrapper using theme factory
	ml.modal = theme.NewModal()
	
	// Replace modal's content with our custom flex
	ml.modal.SetText("")
	ml.modal.AddButtons([]string{})
	
	// We need to manually handle the modal content
	// This is a bit of a hack since tview.Modal doesn't easily allow custom content
	// Instead, we'll return the flex directly and handle it as a page
}

// GetView returns the main view component
func (ml *ModalList) GetView() tview.Primitive {
	// Create a flex container to center the list
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(ml.list, len(ml.items)+4, 0, true). // Height based on item count + border
			AddItem(nil, 0, 1, false), 0, 1, true).
		AddItem(nil, 0, 1, false)

	// Apply theme colors for modal overlay effect
	currentTheme := theme.Current()
	flex.SetBackgroundColor(currentTheme.TerminalColors().Background)
	
	return flex
}

// GetList returns the internal list component
func (ml *ModalList) GetList() *tview.List {
	return ml.list
}

// SetDoneFunc sets the function to call when the modal should close
func (ml *ModalList) SetDoneFunc(handler func()) {
	ml.list.SetDoneFunc(handler)
}

// Focus sets focus to the list
func (ml *ModalList) Focus() {
	// The list will be focused when the view is displayed
}