package theme

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"twist/internal/components"
)

// ThemedComponents provides convenience factory functions for creating themed components
// while still allowing manual styling using theme properties
type ThemedComponents struct {
	theme Theme
}

// NewThemedComponents creates a new themed components factory
func NewThemedComponents(theme Theme) *ThemedComponents {
	return &ThemedComponents{theme: theme}
}

// NewList creates a new list with theme applied
func (tc *ThemedComponents) NewList() *tview.List {
	list := tview.NewList()
	colors := tc.theme.DialogColors()
	border := tc.theme.BorderStyle()
	
	list.SetBackgroundColor(colors.Background)
	list.SetMainTextColor(colors.Foreground)
	list.SetSelectedTextColor(colors.SelectedFg)
	list.SetSelectedBackgroundColor(colors.SelectedBg)
	list.SetBorderColor(colors.Border)
	list.SetTitleColor(colors.Title)
	list.SetBorder(true)
	list.SetBorderPadding(border.Padding, border.Padding, border.Padding, border.Padding)
	
	return list
}

// NewModal creates a new modal with theme applied
func (tc *ThemedComponents) NewModal() *tview.Modal {
	modal := tview.NewModal()
	colors := tc.theme.DialogColors()
	
	modal.SetBackgroundColor(colors.Background)
	modal.SetTextColor(colors.Foreground)
	modal.SetButtonBackgroundColor(colors.ButtonBg)
	modal.SetButtonTextColor(colors.ButtonFg)
	
	return modal
}

// NewTextView creates a new text view with theme applied
func (tc *ThemedComponents) NewTextView() *tview.TextView {
	textView := tview.NewTextView()
	colors := tc.theme.TerminalColors()
	border := tc.theme.BorderStyle()
	
	textView.SetBackgroundColor(colors.Background)
	textView.SetTextColor(colors.Foreground)
	textView.SetBorderColor(colors.Border)
	textView.SetTitleColor(border.TitleColor)
	// Border styling applied via theme colors
	
	return textView
}

// NewInputField creates a new input field with theme applied
func (tc *ThemedComponents) NewInputField() *tview.InputField {
	input := tview.NewInputField()
	colors := tc.theme.TerminalColors()
	
	input.SetBackgroundColor(colors.Background)
	input.SetFieldBackgroundColor(colors.Background)
	input.SetFieldTextColor(colors.Foreground)
	input.SetLabelColor(colors.Foreground)
	
	return input
}

// NewTable creates a new table with theme applied
func (tc *ThemedComponents) NewTable() *tview.Table {
	table := tview.NewTable()
	colors := tc.theme.PanelColors()
	
	table.SetBackgroundColor(colors.Background)
	table.SetBorderColor(colors.Border)
	table.SetTitleColor(colors.Title)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(colors.HeaderBg).
		Foreground(colors.HeaderFg))
	
	return table
}

// NewFlex creates a new flex with theme applied (typically for overlays)
func (tc *ThemedComponents) NewFlex() *tview.Flex {
	flex := tview.NewFlex()
	colors := tc.theme.TerminalColors()
	
	flex.SetBackgroundColor(colors.Background)
	
	return flex
}

// NewMenuList creates a new list specifically styled for menus
func (tc *ThemedComponents) NewMenuList() *tview.List {
	list := tview.NewList()
	colors := tc.theme.MenuColors()
	
	list.SetBackgroundColor(colors.Background)
	list.SetMainTextColor(colors.Foreground)
	list.SetSelectedTextColor(colors.SelectedFg)
	list.SetSelectedBackgroundColor(colors.SelectedBg)
	list.SetBorderColor(colors.Foreground)
	list.SetBorder(true)
	
	return list
}

// NewTwistMenu creates a new TwistMenu with themed styling and custom borders
func (tc *ThemedComponents) NewTwistMenu() *components.TwistMenu {
	colors := tc.theme.MenuColors()
	borderStyle := tc.theme.MenuBorderStyle()
	borderChars := components.NewSimpleBorderChars(borderStyle)
	
	menu := components.NewTwistMenu(borderChars)
	
	// Set the overall background color
	menu.SetBackgroundColor(colors.Background)
	
	// Set the main text style with explicit background color for unselected items
	mainStyle := tcell.StyleDefault.
		Foreground(colors.Foreground).
		Background(colors.Background)
	menu.SetMainTextStyle(mainStyle)
	
	// Set selected item colors
	menu.SetSelectedTextColor(colors.SelectedFg)
	menu.SetSelectedBackgroundColor(colors.SelectedBg)
	
	// Set border styling
	menu.SetBorderColor(colors.Foreground)
	menu.SetBorder(true)
	
	return menu
}

// NewStatusBar creates a new text view styled for status bars
func (tc *ThemedComponents) NewStatusBar() *tview.TextView {
	textView := tview.NewTextView()
	colors := tc.theme.StatusColors()
	
	textView.SetBackgroundColor(colors.Background)
	textView.SetTextColor(colors.Foreground)
	textView.SetDynamicColors(true)
	
	return textView
}

// NewMenuBar creates a new text view styled for menu bars
func (tc *ThemedComponents) NewMenuBar() *tview.TextView {
	textView := tview.NewTextView()
	colors := tc.theme.MenuColors()
	
	textView.SetBackgroundColor(colors.Background)
	textView.SetTextColor(colors.Foreground)
	textView.SetDynamicColors(true)
	
	return textView
}

// NewPanelView creates a new text view styled for side panels
func (tc *ThemedComponents) NewPanelView() *tview.TextView {
	textView := tview.NewTextView()
	colors := tc.theme.PanelColors()
	border := tc.theme.BorderStyle()
	
	textView.SetBackgroundColor(colors.Background)
	textView.SetTextColor(colors.Foreground)
	textView.SetBorderColor(colors.Border)
	textView.SetTitleColor(colors.Title)
	textView.SetBorder(true)
	textView.SetBorderPadding(border.Padding, border.Padding, border.Padding, border.Padding)
	// Border styling applied via theme colors
	
	return textView
}

// NewForm creates a new form with theme applied
func (tc *ThemedComponents) NewForm() *tview.Form {
	form := tview.NewForm()
	colors := tc.theme.DialogColors()
	
	form.SetBackgroundColor(colors.Background)
	form.SetFieldBackgroundColor(colors.FieldBg)
	form.SetFieldTextColor(colors.FieldFg)
	form.SetLabelColor(colors.Foreground)
	form.SetButtonBackgroundColor(colors.ButtonBg)
	form.SetButtonTextColor(colors.ButtonFg)
	form.SetBorderColor(colors.Border)
	form.SetTitleColor(colors.Title)
	
	return form
}

// Global factory instance using current theme
var defaultFactory = &ThemedComponents{}

// updateDefaultFactory updates the global factory with current theme
func updateDefaultFactory() {
	defaultFactory.theme = defaultThemeManager.Current()
}

// Convenience functions using global theme
func NewList() *tview.List {
	updateDefaultFactory()
	return defaultFactory.NewList()
}

func NewModal() *tview.Modal {
	updateDefaultFactory()
	return defaultFactory.NewModal()
}

func NewTextView() *tview.TextView {
	updateDefaultFactory()
	return defaultFactory.NewTextView()
}

func NewInputField() *tview.InputField {
	updateDefaultFactory()
	return defaultFactory.NewInputField()
}

func NewTable() *tview.Table {
	updateDefaultFactory()
	return defaultFactory.NewTable()
}

func NewFlex() *tview.Flex {
	updateDefaultFactory()
	return defaultFactory.NewFlex()
}

func NewMenuList() *tview.List {
	updateDefaultFactory()
	return defaultFactory.NewMenuList()
}

func NewTwistMenu() *components.TwistMenu {
	updateDefaultFactory()
	return defaultFactory.NewTwistMenu()
}

func NewStatusBar() *tview.TextView {
	updateDefaultFactory()
	return defaultFactory.NewStatusBar()
}

func NewMenuBar() *tview.TextView {
	updateDefaultFactory()
	return defaultFactory.NewMenuBar()
}

func NewPanelView() *tview.TextView {
	updateDefaultFactory()
	return defaultFactory.NewPanelView()
}

func NewForm() *tview.Form {
	updateDefaultFactory()
	return defaultFactory.NewForm()
}