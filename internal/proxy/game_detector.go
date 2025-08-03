package proxy

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"twist/internal/ansi"
	"twist/internal/debug"
	"twist/internal/proxy/database"
	"twist/internal/proxy/scripting"
)

// Token types for game detection
type TokenType int

const (
	TokenError TokenType = iota
	TokenEOF
	TokenText              // Regular text to ignore
	TokenGameMenu          // "Select a game :" pattern
	TokenGameOption        // "<A> Game Name" pattern
	TokenIsolatedLetter    // Single letter game selection
	TokenGameStart         // "Show today's log? (Y/N)" pattern
	TokenGameExit          // Exit patterns
	TokenMainMenu          // Return to main menu patterns
	TokenUserPrompt        // User input prompt patterns like "Your choice: " or "Enter selection: "
)

type GameDetectionState int

const (
	StateIdle GameDetectionState = iota
	StateGameMenuVisible
	StateGameSelected
	StateGameActive  // Combines starting and active - game is running
)

// String returns a string representation of the GameDetectionState
func (s GameDetectionState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateGameMenuVisible:
		return "GameMenuVisible"
	case StateGameSelected:
		return "GameSelected"
	case StateGameActive:
		return "GameActive"
	default:
		return "Unknown"
	}
}

// ConnectionInfo holds server connection details
type ConnectionInfo struct {
	Host string
	Port string
}

// Token represents a lexed token
type Token struct {
	Type     TokenType
	Value    string
	Letter   string // For game options and selections
	GameName string // For game options
	Pos      int    // Position in input
}

// PatternMatcher tracks progress matching a specific pattern
type PatternMatcher struct {
	pattern     string      // Pattern to match
	position    int         // Current position in pattern
	tokenType   TokenType   // Token to emit on match
	buffer      string      // Buffer for this pattern
	isActive    bool        // Whether this matcher is currently active
}

// stateFn represents the state of the scanner as a function that returns the next state
type stateFn func(*GameDetector) stateFn

// GameDetector is a streaming lexer for game detection
type GameDetector struct {
	mu                    sync.RWMutex
	
	// Connection info
	serverHost            string
	serverPort            string
	
	// Streaming input handling
	currentBuffer         string  // Small buffer for current potential match
	patternMatchers       map[string]*PatternMatcher // Active pattern matchers
	
	// ANSI stripping for streaming content
	ansiStripper          *ansi.StreamingStripper
	
	// Detection state
	currentState          GameDetectionState
	selectedGame          string
	gameOptions           map[string]string
	expectingUserInput    bool    // Track when we're expecting user input after a prompt
	
	// Token channel
	tokens                chan Token
	
	// Database management
	currentDatabase       database.Database
	currentScriptManager  *scripting.ScriptManager
	
	// Callbacks
	onDatabaseLoaded      func(db database.Database, scriptManager *scripting.ScriptManager) error
	onDatabaseStateChanged func(gameName, serverHost, serverPort, dbName string, isLoaded bool)
	
	// Timing
	lastActivity          time.Time
	detectionTimeout      time.Duration
}

// NewGameDetector creates a new lexer-based game detector
func NewGameDetector(connInfo ConnectionInfo) *GameDetector {
	l := &GameDetector{
		serverHost:       connInfo.Host,
		serverPort:       connInfo.Port,
		currentState:     StateIdle,
		gameOptions:      make(map[string]string),
		tokens:           make(chan Token, 100), // Buffered channel
		detectionTimeout: time.Minute * 5,
		patternMatchers:  make(map[string]*PatternMatcher),
		ansiStripper:     ansi.NewStreamingStripper(),
	}
	
	// Initialize pattern matchers
	l.initializePatterns()
	
	return l
}

// initializePatterns sets up all the pattern matchers
func (l *GameDetector) initializePatterns() {
	patterns := map[string]TokenType{
		"Select a game :":           TokenGameMenu,
		"Show today's log?":         TokenGameStart,  // Match the question, ignore the options after
		"Goodbye":                   TokenGameExit,
		"Thank you for playing":     TokenGameExit,
		"Connection terminated":     TokenGameExit,
		"Disconnected":             TokenGameExit,
		"TWGS v":                   TokenMainMenu,
		"TradeWars Game Server":    TokenMainMenu,
		"Your choice: ":            TokenUserPrompt,
		"Enter selection: ":        TokenUserPrompt,
		"Choice: ":                 TokenUserPrompt,
		"Enter your choice: ":      TokenUserPrompt,
		"Please enter your choice: ": TokenUserPrompt,
		"Selection: ":              TokenUserPrompt,
	}
	
	for pattern, tokenType := range patterns {
		l.patternMatchers[pattern] = &PatternMatcher{
			pattern:   pattern,
			position:  0,
			tokenType: tokenType,
			buffer:    "",
			isActive:  false,
		}
	}
}

// ProcessChunk processes raw data chunks for game detection
func (l *GameDetector) ProcessChunk(data []byte) {
	if len(data) == 0 {
		return
	}
	
	text := string(data)
	l.ProcessLine(text)
}

// ProcessUserInput processes user input separately from server output
// This should be called when the user types something, not for server output
func (l *GameDetector) ProcessUserInput(input string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Update activity timestamp
	l.lastActivity = time.Now()
	
	// Only process isolated letters from user input in game menu state
	if l.currentState == StateGameMenuVisible && len(input) == 1 {
		// Extract the character and convert to uppercase
		char := rune(input[0])
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') {
			letterStr := strings.ToUpper(string(char))
			if _, exists := l.gameOptions[letterStr]; exists {
				debug.Log("GameDetector: User input letter detected: %s", letterStr)
				l.emitIsolatedLetterToken(letterStr)
				l.processTokens()
			}
		}
	}
}

// ProcessLine processes server output character by character with minimal buffering  
func (l *GameDetector) ProcessLine(text string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Update activity timestamp
	l.lastActivity = time.Now()
	
	// Strip ANSI codes before processing using streaming stripper
	cleanText := l.ansiStripper.StripChunk(text)
	
	
	// Process each character
	for _, char := range cleanText {
		l.processCharacter(char)
	}
	
	// Process any emitted tokens
	l.processTokens()
	
	// Check for timeout
	l.checkTimeout()
}

// processCharacter handles a single character through state-appropriate pattern matchers
func (l *GameDetector) processCharacter(char rune) {
	
	// Always check for exit and main menu patterns (can happen in any state)
	l.checkPattern("Goodbye", char)
	l.checkPattern("Thank you for playing", char)
	l.checkPattern("Connection terminated", char)
	l.checkPattern("Disconnected", char)
	l.checkPattern("TWGS v", char)
	l.checkPattern("TradeWars Game Server", char)
	
	// Always check for user prompt patterns in game menu states
	if l.currentState == StateGameMenuVisible {
		l.checkPattern("Your choice: ", char)
		l.checkPattern("Enter selection: ", char)
		l.checkPattern("Choice: ", char)
		l.checkPattern("Enter your choice: ", char)
		l.checkPattern("Please enter your choice: ", char)
		l.checkPattern("Selection: ", char)
	}
	
	// State-specific pattern matching
	switch l.currentState {
	case StateIdle:
		// Look for game menu pattern AND game options (some servers send options first)
		l.checkPattern("Select a game :", char)
		l.processGameOptionPattern(char)          // <X> Game Name format - auto-transition to menu state
		
	case StateGameMenuVisible:
		// Look for game options from server output 
		l.processGameOptionPattern(char)          // <X> Game Name format
		// Note: isolated letters are now handled via ProcessUserInput(), not from server output
		
	case StateGameSelected:
		// Look for game start pattern (log prompt)
		l.checkPattern("Show today's log?", char)
		
	case StateGameActive:
		// Game is active - only exit/menu patterns (handled above)
		// Don't process any game selection patterns
		
	default:
		// Unknown state, be conservative and check basic patterns
		l.checkPattern("Select a game :", char)
	}
}

// checkPattern checks a specific pattern matcher
func (l *GameDetector) checkPattern(pattern string, char rune) {
	if matcher, exists := l.patternMatchers[pattern]; exists {
		l.updatePatternMatcher(matcher, char)
	}
}

// updatePatternMatcher updates a single pattern matcher with the new character
func (l *GameDetector) updatePatternMatcher(matcher *PatternMatcher, char rune) {
	expectedChar := rune(matcher.pattern[matcher.position])
	
	if char == expectedChar {
		// Character matches, advance in pattern
		matcher.position++
		matcher.buffer += string(char)
		matcher.isActive = true
		
		// Check if we've matched the complete pattern
		if matcher.position >= len(matcher.pattern) {
			// Emit token
			token := Token{
				Type:  matcher.tokenType,
				Value: matcher.buffer,
				Pos:   0, // Position tracking simplified for streaming
			}
			l.tokens <- token
			
			// Reset matcher
			matcher.position = 0
			matcher.buffer = ""
			matcher.isActive = false
		}
	} else if matcher.isActive {
		// Pattern broken, reset but check if current char starts a new match
		matcher.position = 0
		matcher.buffer = ""
		matcher.isActive = false
		
		// Check if current character starts this pattern
		if char == rune(matcher.pattern[0]) {
			matcher.position = 1
			matcher.buffer = string(char)
			matcher.isActive = true
		}
	} else if char == rune(matcher.pattern[0]) {
		// Start of potential pattern match
		matcher.position = 1
		matcher.buffer = string(char)
		matcher.isActive = true
	}
}

// gameOptionState tracks parsing of <X> Game Name patterns
type gameOptionState struct {
	state    int    // 0=none, 1=saw<, 2=sawletter, 3=saw>, 4=ingamename
	letter   string
	gameName strings.Builder
}

// alternativeGameOptionState tracks parsing of X - Game Name patterns (Trade Wars style)
type alternativeGameOptionState struct {
	state    int    // 0=none, 1=sawletter, 2=sawspace, 3=sawdash, 4=sawspace2, 5=ingamename
	letter   string
	gameName strings.Builder
}

var gOptionState = &gameOptionState{}
var altOptionState = &alternativeGameOptionState{}

// processGameOptionPattern handles <X> Game Name pattern parsing
func (l *GameDetector) processGameOptionPattern(char rune) {
	
	switch gOptionState.state {
	case 0: // Looking for '<'
		if char == '<' {
			gOptionState.state = 1
		}
	case 1: // Looking for letter after '<'
		if char >= 'A' && char <= 'Z' {
			gOptionState.letter = string(char)
			gOptionState.state = 2
		} else {
			gOptionState.state = 0 // Reset
		}
	case 2: // Looking for '>' after letter
		if char == '>' {
			gOptionState.state = 3
		} else {
			gOptionState.state = 0 // Reset
		}
	case 3: // Skip whitespace, start collecting game name
		if char == ' ' || char == '\t' {
			// Continue waiting
		} else if char == '\n' || char == '\r' || char == '[' {
			// End of game name
			l.emitGameOptionToken(gOptionState.letter, gOptionState.gameName.String())
			gOptionState.reset()
		} else {
			gOptionState.gameName.WriteRune(char)
			gOptionState.state = 4
		}
	case 4: // Collecting game name
		if char == '\n' || char == '\r' || char == '[' {
			// End of game name
			l.emitGameOptionToken(gOptionState.letter, gOptionState.gameName.String())
			gOptionState.reset()
		} else {
			gOptionState.gameName.WriteRune(char)
		}
	}
}

func (g *gameOptionState) reset() {
	g.state = 0
	g.letter = ""
	g.gameName.Reset()
}

// processAlternativeGameOptionPattern handles X - Game Name pattern parsing (Trade Wars style)
func (l *GameDetector) processAlternativeGameOptionPattern(char rune) {
	switch altOptionState.state {
	case 0: // Looking for letter at start of line
		if char >= 'A' && char <= 'Z' {
			altOptionState.letter = string(char)
			altOptionState.state = 1
		}
	case 1: // Looking for space after letter
		if char == ' ' {
			altOptionState.state = 2
		} else {
			altOptionState.reset() // Reset if no space
		}
	case 2: // Looking for dash
		if char == '-' {
			altOptionState.state = 3
		} else {
			altOptionState.reset() // Reset if no dash
		}
	case 3: // Looking for space after dash
		if char == ' ' {
			altOptionState.state = 4
		} else {
			altOptionState.reset() // Reset if no space
		}
	case 4: // Collecting game name
		if char == '\n' || char == '\r' {
			// End of game name
			gameName := strings.TrimSpace(altOptionState.gameName.String())
			if gameName != "" {
				l.emitGameOptionToken(altOptionState.letter, gameName)
			}
			altOptionState.reset()
		} else {
			altOptionState.gameName.WriteRune(char)
		}
	}
}

func (g *alternativeGameOptionState) reset() {
	g.state = 0
	g.letter = ""
	g.gameName.Reset()
}

// isolatedLetterState tracks context for isolated letter detection
type isolatedLetterState struct {
	prevChar rune
	prevPrevChar rune
}

var iLetterState = &isolatedLetterState{}

// processIsolatedLetter is no longer used - isolated letters are handled via ProcessUserInput
// This method is kept for backward compatibility but does nothing
func (l *GameDetector) processIsolatedLetter(char rune) {
	// Update state tracking for any remaining usage
	iLetterState.prevPrevChar = iLetterState.prevChar
	iLetterState.prevChar = char
}

// isValidPrecedingChar checks if the preceding character is valid for isolated letters
func (l *GameDetector) isValidPrecedingChar(prevChar rune) bool {
	validChars := []rune{' ', ':', '>', '\n', '\r', '\t', '[', '(', ')'}
	for _, validChar := range validChars {
		if prevChar == validChar {
			return true
		}
	}
	return prevChar == 0 // Start of input
}


// emitGameOptionToken emits a game option token
func (l *GameDetector) emitGameOptionToken(letter, gameName string) {
	cleanGameName := strings.TrimSpace(gameName)
	
	// Auto-transition to game menu state if we detect game options while idle
	if l.currentState == StateIdle {
		debug.Log("GameDetector: Auto-transitioning to StateGameMenuVisible due to game option detection")
		l.currentState = StateGameMenuVisible
		// Only initialize gameOptions if it's nil/empty to preserve existing options
		if l.gameOptions == nil {
			l.gameOptions = make(map[string]string)
		}
	}
	
	token := Token{
		Type:     TokenGameOption,
		Value:    "<" + letter + "> " + cleanGameName,
		Letter:   letter,
		GameName: cleanGameName,
		Pos:      0,
	}
	l.tokens <- token
}

// emitIsolatedLetterToken emits an isolated letter token
func (l *GameDetector) emitIsolatedLetterToken(letter string) {
	token := Token{
		Type:  TokenIsolatedLetter,
		Value: letter,
		Pos:   0,
	}
	l.tokens <- token
}

// processTokens handles emitted tokens and updates game detection state
func (l *GameDetector) processTokens() {
	for {
		select {
		case token := <-l.tokens:
			l.handleToken(token)
		default:
			return // No more tokens
		}
	}
}

// handleToken processes a single token and updates detection state
func (l *GameDetector) handleToken(token Token) {
	switch token.Type {
	case TokenGameMenu:
		// Detect game menu regardless of current state (could be returning from a game)
		debug.Log("Game menu detected via lexer (was in state: %v)", l.currentState)
		l.currentState = StateGameMenuVisible
		// Only reset gameOptions if we don't already have options (preserve auto-detected ones)
		if len(l.gameOptions) == 0 {
			l.gameOptions = make(map[string]string)
		}
		
	case TokenGameOption:
		if l.currentState == StateGameMenuVisible {
			l.gameOptions[token.Letter] = token.GameName
			debug.Log("Found game option via lexer: %s -> %s", token.Letter, token.GameName)
		}
		
	case TokenIsolatedLetter:
		if l.currentState == StateGameMenuVisible {
			if gameName, exists := l.gameOptions[token.Value]; exists {
				debug.Log("Game selected via lexer: %s (%s)", token.Value, gameName)
				l.selectedGame = gameName
				l.currentState = StateGameSelected
			}
		}
		
	case TokenGameStart:
		if l.currentState == StateGameSelected {
			debug.Log("Game starting detected via lexer for: %s", l.selectedGame)
			l.currentState = StateGameActive
			if err := l.loadGameDatabase(); err != nil {
				debug.Log("Error loading database: %v", err)
			}
		}
		
	case TokenGameExit:
		if l.currentState == StateGameActive || l.currentState == StateGameSelected {
			debug.Log("Game exit detected via lexer for: %s", l.selectedGame)
			l.resetGameState()
		}
		
	case TokenMainMenu:
		if l.currentState == StateGameActive {
			debug.Log("Main menu detected via lexer, resetting game state")
			l.resetGameState()
		}
		
	case TokenUserPrompt:
		// A user prompt was detected - we're now expecting user input
		debug.Log("User prompt detected: %s", token.Value)
		l.expectingUserInput = true
	}
}

// resetGameState clears current game state
func (l *GameDetector) resetGameState() {
	debug.Log("Resetting game state via lexer")
	
	// Notify about database being unloaded if one is currently loaded
	if l.currentDatabase != nil && l.onDatabaseStateChanged != nil {
		currentGame := l.selectedGame
		if currentGame == "" {
			currentGame = "Unknown Game"
		}
		currentDbName := l.createDatabaseName(currentGame)
		go func() {
			l.onDatabaseStateChanged(currentGame, l.serverHost, l.serverPort, currentDbName, false)
		}()
	}
	
	l.currentState = StateIdle
	l.selectedGame = ""
	l.gameOptions = make(map[string]string)
	l.expectingUserInput = false
	
	// Reset pattern matchers
	for _, matcher := range l.patternMatchers {
		matcher.position = 0
		matcher.buffer = ""
		matcher.isActive = false
	}
	
	// Reset state machines
	gOptionState.reset()
	altOptionState.reset()
	iLetterState.prevChar = 0
	iLetterState.prevPrevChar = 0
	
	// Reset ANSI stripper state
	l.ansiStripper.Reset()
}

// Helper functions

// isAlphaNum checks if character is alphanumeric
func (l *GameDetector) isAlphaNum(char byte) bool {
	return (char >= 'A' && char <= 'Z') || 
		   (char >= 'a' && char <= 'z') || 
		   (char >= '0' && char <= '9')
}


// checkTimeout resets state if no activity for too long
func (l *GameDetector) checkTimeout() {
	if time.Since(l.lastActivity) > l.detectionTimeout {
		debug.Log("Game detection timeout via lexer, resetting state")
		l.resetGameState()
	}
}

// CheckTimeoutManual allows manual timeout checking for tests
func (l *GameDetector) CheckTimeoutManual() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.checkTimeout()
}

// resetGameState clears current game state

// Database management functions (reused from original)

func (l *GameDetector) loadGameDatabase() error {
	// Notify about database being unloaded if one is currently loaded
	if l.currentDatabase != nil && l.onDatabaseStateChanged != nil {
		currentGame := l.selectedGame
		if currentGame == "" {
			// If we're replacing a database, use the previous game name if available
			currentGame = "Unknown Game"
		}
		currentDbName := l.createDatabaseName(currentGame)
		go func() {
			l.onDatabaseStateChanged(currentGame, l.serverHost, l.serverPort, currentDbName, false)
		}()
	}
	
	if l.currentDatabase != nil {
		l.currentDatabase.CloseDatabase()
	}
	
	if l.currentScriptManager != nil {
		l.currentScriptManager.Stop()
	}
	
	dbName := l.createDatabaseName(l.selectedGame)
	debug.Log("Loading database via lexer: %s", dbName)
	
	db := database.NewDatabase()
	
	if err := db.CreateDatabase(dbName); err != nil {
		if err := db.OpenDatabase(dbName); err != nil {
			return fmt.Errorf("failed to load database %s: %w", dbName, err)
		}
	}
	
	scriptManager := scripting.NewScriptManager(db)
	
	l.currentDatabase = db
	l.currentScriptManager = scriptManager
	
	// Notify about database state change (loaded)
	if l.onDatabaseStateChanged != nil {
		go func() {
			l.onDatabaseStateChanged(l.selectedGame, l.serverHost, l.serverPort, dbName, true)
		}()
	} else {
		debug.Log("GameDetector: onDatabaseStateChanged callback is nil - cannot notify TUI")
	}
	
	if l.onDatabaseLoaded != nil {
		go func() {
			if err := l.onDatabaseLoaded(db, scriptManager); err != nil {
				debug.Log("Error in database loaded callback: %v", err)
			}
		}()
	}
	
	return nil
}

func (l *GameDetector) createDatabaseName(gameName string) string {
	host := sanitizeForFilename(l.serverHost)
	port := sanitizeForFilename(l.serverPort)
	game := sanitizeForFilename(gameName)
	
	return fmt.Sprintf("%s_%s_%s.db", host, port, game)
}

// sanitizeForFilename removes or replaces characters that are not safe for filenames
func sanitizeForFilename(input string) string {
	// Replace unsafe characters with underscores
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " ", "."}
	result := input
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Remove consecutive underscores
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}
	// Trim underscores from start and end
	result = strings.Trim(result, "_")
	return strings.ToLower(result)
}

// reset is an alias for resetGameState for test compatibility
func (l *GameDetector) reset() {
	l.resetGameState()
}

// Public interface methods (matching original GameDetector)

func (l *GameDetector) SetDetectionTimeout(timeout time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.detectionTimeout = timeout
}

func (l *GameDetector) SetDatabaseLoadedCallback(callback func(db database.Database, scriptManager *scripting.ScriptManager) error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onDatabaseLoaded = callback
}

func (l *GameDetector) SetDatabaseStateChangedCallback(callback func(gameName, serverHost, serverPort, dbName string, isLoaded bool)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onDatabaseStateChanged = callback
}

func (l *GameDetector) GetCurrentGame() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.selectedGame
}

func (l *GameDetector) GetState() GameDetectionState {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.currentState
}

func (l *GameDetector) IsGameActive() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.currentState == StateGameActive
}

func (l *GameDetector) GetCurrentDatabase() database.Database {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.currentDatabase
}

func (l *GameDetector) GetCurrentScriptManager() *scripting.ScriptManager {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.currentScriptManager
}

func (l *GameDetector) GetServerInfo() ConnectionInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return ConnectionInfo{
		Host: l.serverHost,
		Port: l.serverPort,
	}
}

func (l *GameDetector) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Notify about database being unloaded
	if l.currentDatabase != nil && l.onDatabaseStateChanged != nil {
		currentGame := l.selectedGame
		if currentGame == "" {
			currentGame = "Unknown Game"
		}
		currentDbName := l.createDatabaseName(currentGame)
		go func() {
			l.onDatabaseStateChanged(currentGame, l.serverHost, l.serverPort, currentDbName, false)
		}()
	}
	
	if l.currentScriptManager != nil {
		l.currentScriptManager.Stop()
	}
	
	if l.currentDatabase != nil {
		return l.currentDatabase.CloseDatabase()
	}
	
	return nil
}