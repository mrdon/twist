package display

import (
	"fmt"
	"strings"

	"twist/internal/debug"
)

const (
	// ANSI color codes matching TWX terminal menu formatting
	MENU_LIGHT = "\x1b[37m" // ANSI_15 - white
	MENU_MID   = "\x1b[36m" // ANSI_10 - cyan
	MENU_DARK  = "\x1b[32m" // ANSI_2 - dark green

	// Additional ANSI codes for terminal menus
	ANSI_RESET     = "\x1b[0m"
	ANSI_BOLD      = "\x1b[1m"
	ANSI_DIM       = "\x1b[2m"
	ANSI_UNDERLINE = "\x1b[4m"

	// Background colors
	ANSI_BG_BLACK = "\x1b[40m"
	ANSI_BG_BLUE  = "\x1b[44m"

	// Cursor control
	ANSI_CLEAR_LINE     = "\x1b[2K"
	ANSI_CLEAR_SCREEN   = "\x1b[2J"
	ANSI_HOME_CURSOR    = "\x1b[H"
	ANSI_SAVE_CURSOR    = "\x1b[s"
	ANSI_RESTORE_CURSOR = "\x1b[u"
)

func FormatMenuPrompt(prompt, line string) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in FormatMenuPrompt: %v", r)
		}
	}()

	if prompt == "" {
		prompt = "Selection"
	}

	formatted := fmt.Sprintf("%s%s%s: %s%s%s",
		MENU_LIGHT,
		prompt,
		ANSI_RESET,
		MENU_MID,
		line,
		ANSI_RESET)

	return formatted
}

func FormatMenuTitle(title string) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in FormatMenuTitle: %v", r)
		}
	}()

	separator := strings.Repeat("-", len(title))

	formatted := fmt.Sprintf("%s%s%s%s\r\n%s%s%s%s\r\n",
		ANSI_BOLD,
		MENU_LIGHT,
		title,
		ANSI_RESET,
		MENU_DARK,
		separator,
		ANSI_RESET,
		"\r\n")

	return formatted
}

func FormatMenuOption(hotkey rune, description string, enabled bool) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in FormatMenuOption: %v", r)
		}
	}()

	var color string
	if enabled {
		color = MENU_LIGHT
	} else {
		color = MENU_DARK + ANSI_DIM
	}

	formatted := fmt.Sprintf("%s(%s%c%s)%s%s%s",
		MENU_MID,
		MENU_LIGHT+ANSI_BOLD,
		hotkey,
		MENU_MID+ANSI_RESET,
		color,
		description,
		ANSI_RESET)

	return formatted
}

func FormatHelpText(text string) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in FormatHelpText: %v", r)
		}
	}()

	lines := strings.Split(text, "\n")
	var formatted strings.Builder

	formatted.WriteString(MENU_DARK + "=== Help ===" + ANSI_RESET + "\r\n")

	for _, line := range lines {
		formatted.WriteString(MENU_MID + line + ANSI_RESET + "\r\n")
	}

	formatted.WriteString(MENU_DARK + "============" + ANSI_RESET + "\r\n")

	return formatted.String()
}

func FormatErrorMessage(message string) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in FormatErrorMessage: %v", r)
		}
	}()

	return fmt.Sprintf("\x1b[31m%s%s%s\r\n", // Red color
		ANSI_BOLD,
		message,
		ANSI_RESET)
}

func FormatSuccessMessage(message string) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in FormatSuccessMessage: %v", r)
		}
	}()

	return fmt.Sprintf("\x1b[32m%s%s%s\r\n", // Green color
		ANSI_BOLD,
		message,
		ANSI_RESET)
}

func ClearMenuLine() string {
	return ANSI_CLEAR_LINE + "\r"
}

func ClearMenuScreen() string {
	return ANSI_CLEAR_SCREEN + ANSI_HOME_CURSOR
}

func SaveCursorPosition() string {
	return ANSI_SAVE_CURSOR
}

func RestoreCursorPosition() string {
	return ANSI_RESTORE_CURSOR
}

func MoveCursorUp(lines int) string {
	return fmt.Sprintf("\x1b[%dA", lines)
}

func MoveCursorDown(lines int) string {
	return fmt.Sprintf("\x1b[%dB", lines)
}

func MoveCursorToColumn(col int) string {
	return fmt.Sprintf("\x1b[%dG", col)
}

func FormatMenuSeparator(width int) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in FormatMenuSeparator: %v", r)
		}
	}()

	if width <= 0 {
		width = 40
	}

	separator := strings.Repeat("-", width)
	return fmt.Sprintf("%s%s%s\r\n", MENU_DARK, separator, ANSI_RESET)
}

func FormatInputPrompt(prompt string) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in FormatInputPrompt: %v", r)
		}
	}()

	return fmt.Sprintf("\r\n%s%s%s: ", MENU_LIGHT, prompt, ANSI_RESET)
}

func FormatBreadcrumb(path string) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in FormatBreadcrumb: %v", r)
		}
	}()

	return fmt.Sprintf("%s%s%s%s\r\n",
		MENU_DARK,
		ANSI_DIM,
		path,
		ANSI_RESET)
}

func WrapText(text string, width int) []string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in WrapText: %v", r)
		}
	}()

	if width <= 0 {
		width = 78
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+len(word)+1 <= width {
			currentLine.WriteString(" " + word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

func StripANSI(text string) string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in StripANSI: %v", r)
		}
	}()

	// Simple ANSI escape sequence removal
	// This is a basic implementation - for production use, consider a more robust solution
	result := strings.ReplaceAll(text, ANSI_RESET, "")
	result = strings.ReplaceAll(result, ANSI_BOLD, "")
	result = strings.ReplaceAll(result, ANSI_DIM, "")
	result = strings.ReplaceAll(result, ANSI_UNDERLINE, "")
	result = strings.ReplaceAll(result, MENU_LIGHT, "")
	result = strings.ReplaceAll(result, MENU_MID, "")
	result = strings.ReplaceAll(result, MENU_DARK, "")
	result = strings.ReplaceAll(result, ANSI_BG_BLACK, "")
	result = strings.ReplaceAll(result, ANSI_BG_BLUE, "")

	return result
}
