package proxy

import (
	"os"
	"strings"
	"testing"
	"time"
	"twist/internal/proxy/database"
	"twist/internal/proxy/scripting"
)

// TestGameDetector_BasicFlow tests the complete game detection flow
func TestGameDetector_BasicFlow(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	// Initial state should be idle
	if gd.GetState() != StateIdle {
		t.Errorf("Expected StateIdle initially, got %v", gd.GetState())
	}

	// Step 1: Process game menu trigger
	gd.ProcessLine("Select a game :")
	if gd.GetState() != StateGameMenuVisible {
		t.Errorf("Expected StateGameMenuVisible after menu trigger, got %v", gd.GetState())
	}

	// Step 2: Process game options
	gd.ProcessLine("<A> Trade Wars 2002\n")
	gd.ProcessLine("<B> Another Game\n")

	// Step 3: User selects game (server shows prompt, user types input)
	gd.ProcessLine("Your choice: ")  // Server output
	gd.ProcessUserInput("A")         // User input
	if gd.GetState() != StateGameSelected {
		t.Errorf("Expected StateGameSelected after selection, got %v", gd.GetState())
	}
	if gd.GetCurrentGame() != "Trade Wars 2002" {
		t.Errorf("Expected game 'Trade Wars 2002', got %q", gd.GetCurrentGame())
	}

	// Step 4: Game starts
	gd.ProcessLine("Show today's log? (Y/N)")
	if gd.GetState() != StateGameActive {
		t.Errorf("Expected StateGameActive after game start, got %v", gd.GetState())
	}

	// Step 5: Game exit
	gd.ProcessLine("Goodbye")
	if gd.GetState() != StateIdle {
		t.Errorf("Expected StateIdle after game exit, got %v", gd.GetState())
	}
	if gd.GetCurrentGame() != "" {
		t.Errorf("Expected empty game after exit, got %q", gd.GetCurrentGame())
	}
}

// TestGameDetector_ChunkSplitting tests streaming across chunk boundaries
func TestGameDetector_ChunkSplitting(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	testCases := []struct {
		name   string
		chunks []string
		expectedState GameDetectionState
		expectedGame  string
	}{
		{
			name: "Menu trigger split across chunks",
			chunks: []string{"Select a ga", "me :"},
			expectedState: StateGameMenuVisible,
		},
		{
			name: "Game option split across chunks", 
			chunks: []string{"Select a game :", "<A> Trade Wa", "rs 2002\n", "A"},
			expectedState: StateGameSelected,
			expectedGame: "Trade Wars 2002",
		},
		{
			name: "Game start pattern split",
			chunks: []string{"Select a game :", "<A> Test Game\n", "A", "Show today's log", "? (Y/N)"},
			expectedState: StateGameActive,
			expectedGame: "Test Game",
		},
		{
			name: "Exit pattern split",
			chunks: []string{"Select a game :", "<A> Test Game\n", "A", "Show today's log? (Y/N)", "Good", "bye"},
			expectedState: StateIdle,
			expectedGame: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gd.resetGameState()
			
			for _, chunk := range tc.chunks {
				gd.ProcessLine(chunk)
			}
			
			if gd.GetState() != tc.expectedState {
				t.Errorf("Expected state %v, got %v", tc.expectedState, gd.GetState())
			}
			
			if tc.expectedGame != "" && gd.GetCurrentGame() != tc.expectedGame {
				t.Errorf("Expected game %q, got %q", tc.expectedGame, gd.GetCurrentGame())
			}
			
			if tc.expectedGame == "" && gd.GetCurrentGame() != "" {
				t.Errorf("Expected empty game, got %q", gd.GetCurrentGame())
			}
		})
	}
}

// TestGameDetector_ANSISequences tests handling of ANSI codes
func TestGameDetector_ANSISequences(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	// Test with ANSI sequences mixed in
	gd.ProcessLine("\x1b[36mSelect a game :\x1b[37m")
	if gd.GetState() != StateGameMenuVisible {
		t.Errorf("Expected StateGameMenuVisible with ANSI, got %v", gd.GetState())
	}

	gd.ProcessLine("\x1b[31m<A>\x1b[32m Trade Wars 2002\x1b[0m\n")
	gd.ProcessLine("\x1b[1mA\x1b[0m")
	
	if gd.GetState() != StateGameSelected {
		t.Errorf("Expected StateGameSelected with ANSI, got %v", gd.GetState())
	}
	if gd.GetCurrentGame() != "Trade Wars 2002" {
		t.Errorf("Expected game 'Trade Wars 2002', got %q", gd.GetCurrentGame())
	}
}

// TestGameDetector_ProcessChunk tests raw byte processing
func TestGameDetector_ProcessChunk(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	// Test processing raw bytes
	chunk1 := []byte("Select a ga")
	chunk2 := []byte("me :")
	
	gd.ProcessChunk(chunk1)
	if gd.GetState() != StateIdle {
		t.Errorf("Expected StateIdle after partial pattern, got %v", gd.GetState())
	}
	
	gd.ProcessChunk(chunk2)
	if gd.GetState() != StateGameMenuVisible {
		t.Errorf("Expected StateGameMenuVisible after complete pattern, got %v", gd.GetState())
	}

	// Test empty chunks
	gd.ProcessChunk([]byte{})
	gd.ProcessChunk(nil)
	if gd.GetState() != StateGameMenuVisible {
		t.Errorf("State should not change with empty chunks, got %v", gd.GetState())
	}
}

// TestGameDetector_StateProtection tests state-based pattern filtering
func TestGameDetector_StateProtection(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	// Complete game flow to active state
	gd.ProcessLine("Select a game :")
	gd.ProcessLine("<A> Test Game\n")
	gd.ProcessLine("A")
	gd.ProcessLine("Show today's log? (Y/N)")
	
	if gd.GetState() != StateGameActive {
		t.Fatalf("Expected StateGameActive, got %v", gd.GetState())
	}

	// Game menu pattern should not trigger in active state
	originalGame := gd.GetCurrentGame()
	gd.ProcessLine("Select a game :")
	gd.ProcessLine("<B> Another Game\n")
	gd.ProcessLine("B")
	
	if gd.GetState() != StateGameActive {
		t.Errorf("Game detection should not interfere with active game, got %v", gd.GetState())
	}
	if gd.GetCurrentGame() != originalGame {
		t.Errorf("Active game should not change, got %q", gd.GetCurrentGame())
	}

	// Exit should still work
	gd.ProcessLine("Goodbye")
	if gd.GetState() != StateIdle {
		t.Errorf("Expected StateIdle after exit, got %v", gd.GetState())
	}
}

// TestGameDetector_MainMenuReturn tests main menu detection
func TestGameDetector_MainMenuReturn(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	// Set up active game
	gd.ProcessLine("Select a game :")
	gd.ProcessLine("<A> Test Game\n")
	gd.ProcessLine("A") 
	gd.ProcessLine("Show today's log? (Y/N)")
	
	if gd.GetState() != StateGameActive {
		t.Fatalf("Expected StateGameActive, got %v", gd.GetState())
	}

	// Main menu patterns should reset state
	testCases := []string{
		"TWGS v1.0",
		"TradeWars Game Server",
	}

	for _, pattern := range testCases {
		t.Run("MainMenu_"+pattern, func(t *testing.T) {
			// Reset to active state
			gd.resetGameState()
			gd.ProcessLine("Select a game :")
			gd.ProcessLine("<A> Test Game\n")
			gd.ProcessLine("A")
			gd.ProcessLine("Show today's log? (Y/N)")
			
			// Process main menu pattern
			gd.ProcessLine(pattern)
			
			if gd.GetState() != StateIdle {
				t.Errorf("Expected StateIdle after main menu pattern %q, got %v", pattern, gd.GetState())
			}
		})
	}
}

// TestGameDetector_Timeout tests timeout functionality
func TestGameDetector_Timeout(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	// Set detection timeout to very short for testing
	gd.SetDetectionTimeout(time.Millisecond * 10)

	// Start game detection
	gd.ProcessLine("Select a game :")
	if gd.GetState() != StateGameMenuVisible {
		t.Fatalf("Expected StateGameMenuVisible, got %v", gd.GetState())
	}

	// Wait for timeout
	time.Sleep(time.Millisecond * 20)
	
	// Trigger timeout check without updating activity
	gd.CheckTimeoutManual()
	
	if gd.GetState() != StateIdle {
		t.Errorf("Expected StateIdle after timeout, got %v", gd.GetState())
	}
}

// TestGameDetector_DatabaseLoading tests database creation
func TestGameDetector_DatabaseLoading(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	var callbackCalled bool
	gd.SetDatabaseLoadedCallback(func(db database.Database, sm *scripting.ScriptManager) error {
		callbackCalled = true
		return nil
	})

	// Complete flow to trigger database loading
	gd.ProcessLine("Select a game :")
	gd.ProcessLine("<A> Test Game\n")
	gd.ProcessLine("A")
	gd.ProcessLine("Show today's log? (Y/N)")

	// Give callback time to execute
	time.Sleep(time.Millisecond * 10)

	if !callbackCalled {
		t.Error("Database loaded callback was not called")
	}

	// Verify database was created
	if gd.GetCurrentDatabase() == nil {
		t.Error("Expected database to be loaded")
	}

	if gd.GetCurrentScriptManager() == nil {
		t.Error("Expected script manager to be created")
	}

	// Clean up database file
	os.Remove("localhost_23_test_game.db")
}

// TestGameDetector_ConcurrentAccess tests thread safety
func TestGameDetector_ConcurrentAccess(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	// Run concurrent operations
	done := make(chan bool, 3)

	// Goroutine 1: Process lines
	go func() {
		for i := 0; i < 100; i++ {
			gd.ProcessLine("Select a game :")
			gd.ProcessLine("<A> Game\n")
			gd.ProcessLine("A")
			gd.resetGameState()
		}
		done <- true
	}()

	// Goroutine 2: Read state
	go func() {
		for i := 0; i < 100; i++ {
			_ = gd.GetState()
			_ = gd.GetCurrentGame()
			_ = gd.IsGameActive()
		}
		done <- true
	}()

	// Goroutine 3: Server info
	go func() {
		for i := 0; i < 100; i++ {
			_ = gd.GetServerInfo()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should not crash and should end in a valid state
	finalState := gd.GetState()
	validStates := []GameDetectionState{StateIdle, StateGameMenuVisible, StateGameSelected, StateGameActive}
	stateValid := false
	for _, validState := range validStates {
		if finalState == validState {
			stateValid = true
			break
		}
	}
	if !stateValid {
		t.Errorf("Invalid final state after concurrent access: %v", finalState)
	}
}

// TestGameDetector_EdgeCases tests various edge cases
func TestGameDetector_EdgeCases(t *testing.T) {
	connInfo := ConnectionInfo{Host: "localhost", Port: "23"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	testCases := []struct {
		name     string
		input    string
		expected GameDetectionState
	}{
		{
			name:     "Empty input",
			input:    "",
			expected: StateIdle,
		},
		{
			name:     "Whitespace only",
			input:    "   \t\n  ",
			expected: StateIdle,
		},
		{
			name:     "Very long line",
			input:    strings.Repeat("X", 10000) + "Select a game :",
			expected: StateGameMenuVisible,
		},
		{
			name:     "Binary data mixed with pattern",
			input:    string([]byte{0x00, 0x01, 0x02}) + "Select a game :",
			expected: StateGameMenuVisible,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gd.resetGameState()
			gd.ProcessLine(tc.input)
			
			if gd.GetState() != tc.expected {
				t.Errorf("Expected state %v, got %v", tc.expected, gd.GetState())
			}
		})
	}
}

// TestGameDetector_RealWorldScenarios tests realistic game connection scenarios
func TestGameDetector_RealWorldScenarios(t *testing.T) {
	connInfo := ConnectionInfo{Host: "example.com", Port: "2323"}
	gd := NewGameDetector(connInfo)
	defer gd.Close()

	t.Run("CompleteSessionWithNoise", func(t *testing.T) {
		// Simulate real session with server headers, ANSI, etc.
		chunks := []string{
			"\x1b[2J\x1b[H", // Clear screen
			"Welcome to TradeWars Game Server\n",
			"\x1b[36mSelect a game :\x1b[37m\n",
			"\x1b[31m<A>\x1b[32m Trade Wars 2002 \x1b[33m[Active]\x1b[37m\n",
			"<B> Another Game [Inactive]\n", 
			"<Q> Quit\n",
			"Your choice: A\n",
			"Loading Trade Wars 2002...\n",
			"Show today's log? (Y/N) [N]: ",
			"Welcome to the game!\n",
			"You are in sector 1.\n",
			"Command: Q\n",
			"Goodbye\n",
		}

		for _, chunk := range chunks {
			gd.ProcessLine(chunk)
		}

		if gd.GetState() != StateIdle {
			t.Errorf("Expected StateIdle at end of session, got %v", gd.GetState())
		}
	})

	t.Run("UserChangesGame", func(t *testing.T) {
		gd.resetGameState()
		
		// User starts to select one game
		gd.ProcessLine("Select a game :")
		gd.ProcessLine("<A> Game One\n")
		gd.ProcessLine("<B> Game Two\n")
		
		// But then sees the menu again (maybe they pressed wrong key)
		gd.ProcessLine("Select a game :")
		gd.ProcessLine("<A> Game One\n")
		gd.ProcessLine("<B> Game Two\n")
		gd.ProcessUserInput("B") // User types B
		
		if gd.GetCurrentGame() != "Game Two" {
			t.Errorf("Expected 'Game Two', got %q", gd.GetCurrentGame())
		}
	})

	t.Run("ServerMenuTextShouldNotTriggerGameSelection", func(t *testing.T) {
		gd.resetGameState()
		
		// Simulate server displaying game menu with various letters in the content
		gd.ProcessLine("Select a game :\n")
		gd.ProcessLine("<A> Alien Retribution\n")
		gd.ProcessLine("<B> Star Control II\n") 
		gd.ProcessLine("<E> Stock (9600Baud)\n")
		gd.ProcessLine("Found game option via lexer: A -> Alien Retribution\n")
		gd.ProcessLine("Found game option via lexer: B -> Star Control II\n")
		gd.ProcessLine("Found game option via lexer: E -> Stock (9600Baud)\n")
		gd.ProcessLine("Some server text with letters like A and B and E scattered throughout\n")
		
		// Verify no game was incorrectly selected from server output
		if gd.GetState() != StateGameMenuVisible {
			t.Errorf("Expected StateGameMenuVisible, got %v", gd.GetState())
		}
		if gd.GetCurrentGame() != "" {
			t.Errorf("Expected no game selected, but got %q", gd.GetCurrentGame())
		}
		
		// Now simulate actual user input with proper prompt context
		gd.ProcessLine("Your choice: ")  // Server shows prompt
		gd.ProcessUserInput("E")         // User types selection
		
		// Should now detect the game selection
		if gd.GetState() != StateGameSelected {
			t.Errorf("Expected StateGameSelected after user input, got %v", gd.GetState())
		}
		if gd.GetCurrentGame() != "Stock (9600Baud)" {
			t.Errorf("Expected 'Stock (9600Baud)', got %q", gd.GetCurrentGame())
		}
	})

	t.Run("NewlineServerTextShouldNotTriggerGameSelection", func(t *testing.T) {
		gd.resetGameState()
		
		// Simulate the exact scenario from the debug log
		gd.ProcessLine("Select a game :\n")
		gd.ProcessLine("<A> Alien Retribution\n")
		gd.ProcessLine("<E> Stock (9600Baud)\n")
		gd.ProcessLine("Some server text with E\n")  // This "E" at start of line should NOT be detected
		gd.ProcessLine("E")  // This should also not be detected yet
		
		// Verify no game was selected from server output
		if gd.GetState() != StateGameMenuVisible {
			t.Errorf("Expected StateGameMenuVisible, got %v", gd.GetState())
		}
		if gd.GetCurrentGame() != "" {
			t.Errorf("Expected no game selected from server text, but got %q", gd.GetCurrentGame())
		}
		
		// Now the server sends the actual prompt  
		gd.ProcessLine("Enter your choice: ")  // Server output
		// And then the actual user input
		gd.ProcessUserInput("A")               // User input
		
		// Should now detect the correct game selection
		if gd.GetState() != StateGameSelected {
			t.Errorf("Expected StateGameSelected after user input, got %v", gd.GetState())
		}
		if gd.GetCurrentGame() != "Alien Retribution" {
			t.Errorf("Expected 'Alien Retribution', got %q", gd.GetCurrentGame())
		}
	})
}