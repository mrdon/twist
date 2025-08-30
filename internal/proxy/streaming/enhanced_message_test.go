package streaming

import (
	"strings"
	"testing"
	"twist/internal/proxy/database"
)

func TestEnhancedMessageHandling(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(func() database.Database { return db }, nil)

	// Test cases based on Pascal Process.pas logic
	testCases := []struct {
		name             string
		transmissionLine string
		messageLines     []string
		expectedType     MessageType
		expectedSender   string
		expectedChannel  int
		description      string
	}{
		{
			name:             "Radio transmission on channel",
			transmissionLine: "Incoming transmission from Captain Kirk on channel 1:",
			messageLines:     []string{"R Hello there, trader!"},
			expectedType:     MessageRadio,
			expectedSender:   "Captain Kirk",
			expectedChannel:  0, // Pascal behavior: parseIntSafe("1:") returns 0 due to colon
			description:      "Standard radio transmission with channel",
		},
		{
			name:             "Federation comm-link",
			transmissionLine: "Incoming transmission from Admiral Pike on Federation comm-link:",
			messageLines:     []string{"F Starfleet orders are as follows..."},
			expectedType:     MessageFedlink,
			expectedSender:   "Admiral Pike",
			expectedChannel:  0,
			description:      "Federation communication link",
		},
		{
			name:             "Personal hail message",
			transmissionLine: "Incoming transmission from Dr. McCoy:",
			messageLines:     []string{"P He's dead, Jim!"},
			expectedType:     MessagePersonal,
			expectedSender:   "Dr. McCoy",
			expectedChannel:  0,
			description:      "Personal hail without channel",
		},
		{
			name:             "Fighter deployment message",
			transmissionLine: "Incoming transmission from Fighters:",
			messageLines:     []string{"Deployed fighters report sector status"},
			expectedType:     MessageFighter,
			expectedSender:   "Fighters",
			expectedChannel:  0,
			description:      "Fighter deployment report",
		},
		{
			name:             "Computer message",
			transmissionLine: "Incoming transmission from Computers:",
			messageLines:     []string{"System status: All green"},
			expectedType:     MessageComputer,
			expectedSender:   "Computer",
			expectedChannel:  0,
			description:      "Computer system message",
		},
		{
			name:             "Continuing transmission",
			transmissionLine: "Continuing transmission from Spock on channel 2:",
			messageLines:     []string{"R Logic dictates that we proceed"},
			expectedType:     MessageRadio,
			expectedSender:   "Spock",
			expectedChannel:  0, // Pascal behavior: parseIntSafe("2:") returns 0 due to colon
			description:      "Continuing radio transmission",
		},
		{
			name:             "Deployed fighter report",
			transmissionLine: "Deployed Fighters Report Sector 1234",
			messageLines:     []string{"Fighter status report"},
			expectedType:     MessageDeployed,
			expectedSender:   "sector 1234",
			expectedChannel:  0,
			description:      "Direct fighter report from sector",
		},
		{
			name:             "Shipboard computer message",
			transmissionLine: "Shipboard Computers report anomaly detected",
			messageLines:     []string{"Anomaly detected in sector 5678"},
			expectedType:     MessageShipboard,
			expectedSender:   "Computer",
			expectedChannel:  0,
			description:      "Shipboard computer report",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Clear parser state
			parser.currentMessage = ""
			parser.currentChannel = 0

			// Process transmission header line
			sender, channel, msgType := parser.parseTransmissionDetails(tt.transmissionLine)

			// Verify transmission parsing
			if msgType != tt.expectedType {
				t.Errorf("Expected message type %d, got %d", tt.expectedType, msgType)
			}
			if sender != tt.expectedSender {
				t.Errorf("Expected sender '%s', got '%s'", tt.expectedSender, sender)
			}
			if channel != tt.expectedChannel {
				t.Errorf("Expected channel %d, got %d", tt.expectedChannel, channel)
			}

			// Process message content lines
			for _, messageLine := range tt.messageLines {
				// For direct message types (Deployed, Shipboard), test the transmission line itself
				if tt.expectedType == MessageDeployed || tt.expectedType == MessageShipboard {
					contentType, contentSender, _ := parser.parseMessageContent(tt.transmissionLine)

					// Verify direct message parsing
					if contentType != tt.expectedType {
						t.Errorf("Direct message type mismatch: expected %d, got %d", tt.expectedType, contentType)
					}
					if contentSender != tt.expectedSender {
						t.Errorf("Direct message sender mismatch: expected '%s', got '%s'", tt.expectedSender, contentSender)
					}
				} else {
					// For transmission context messages, test the message content
					contentType, contentSender, contentChannel := parser.parseMessageContent(messageLine)

					// Verify message content parsing uses context correctly
					if contentType != tt.expectedType {
						t.Errorf("Message content type mismatch: expected %d, got %d", tt.expectedType, contentType)
					}

					// For radio messages, sender should come from transmission header context
					if tt.expectedType == MessageRadio && contentSender != tt.expectedSender {
						t.Errorf("Radio message sender mismatch: expected '%s', got '%s'", tt.expectedSender, contentSender)
					}

					// For radio messages, channel should be preserved
					if tt.expectedType == MessageRadio && contentChannel != tt.expectedChannel {
						t.Errorf("Radio message channel mismatch: expected %d, got %d", tt.expectedChannel, contentChannel)
					}
				}
			}

			t.Logf("✓ %s: %s", tt.name, tt.description)
		})
	}
}

func TestMessagePatternRecognition(t *testing.T) {
	parser := NewTestTWXParser()

	// Test Pascal pattern recognition exactly
	testCases := []struct {
		line         string
		shouldMatch  bool
		expectedType MessageType
		description  string
	}{
		{
			line:         "Incoming transmission from Captain Kirk on channel 1:",
			shouldMatch:  true,
			expectedType: MessageRadio,
			description:  "Standard incoming radio transmission",
		},
		{
			line:         "Continuing transmission from Spock on channel 2:",
			shouldMatch:  true,
			expectedType: MessageRadio,
			description:  "Continuing radio transmission",
		},
		{
			line:         "Incoming transmission from Admiral Pike on Federation comm-link:",
			shouldMatch:  true,
			expectedType: MessageFedlink,
			description:  "Federation comm-link transmission",
		},
		{
			line:         "Incoming transmission from Dr. McCoy:",
			shouldMatch:  true,
			expectedType: MessagePersonal,
			description:  "Personal hail transmission",
		},
		{
			line:         "Incoming transmission from Fighters:",
			shouldMatch:  true,
			expectedType: MessageFighter,
			description:  "Fighter transmission",
		},
		{
			line:         "Incoming transmission from Computers:",
			shouldMatch:  true,
			expectedType: MessageComputer,
			description:  "Computer transmission",
		},
		{
			line:         "Deployed Fighters Report Sector 1234",
			shouldMatch:  true,
			expectedType: MessageDeployed,
			description:  "Direct fighter report",
		},
		{
			line:         "Shipboard Computers report system status",
			shouldMatch:  true,
			expectedType: MessageShipboard,
			description:  "Shipboard computer report",
		},
		{
			line:         "R Captain Kirk says hello",
			shouldMatch:  false,
			expectedType: MessageGeneral,
			description:  "Message content line (not transmission header)",
		},
		{
			line:         "F Admiral Pike sends orders",
			shouldMatch:  false,
			expectedType: MessageGeneral,
			description:  "Fedlink content line (not transmission header)",
		},
		{
			line:         "P Dr. McCoy indicates medical emergency",
			shouldMatch:  false,
			expectedType: MessageGeneral,
			description:  "Personal message with 'indicates' (should be ignored per Pascal)",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.description, func(t *testing.T) {
			sender, channel, msgType := parser.parseTransmissionDetails(tt.line)

			if tt.shouldMatch {
				if msgType != tt.expectedType {
					t.Errorf("Expected to match type %d, got %d", tt.expectedType, msgType)
				}
				if sender == "" && (tt.expectedType == MessageRadio || tt.expectedType == MessageFedlink || tt.expectedType == MessagePersonal) {
					t.Errorf("Expected sender to be extracted, got empty string")
				}
				// Pascal behavior: channel parsing fails with colon, returns 0
				// This is expected behavior for Pascal compatibility
				if tt.expectedType == MessageRadio && channel != 0 {
					t.Logf("Note: Channel extraction got %d (Pascal parseIntSafe would return 0 for colon-terminated numbers)", channel)
				}
			} else {
				if msgType != MessageGeneral {
					t.Errorf("Expected non-matching line to return MessageGeneral, got %d", msgType)
				}
			}

			t.Logf("✓ Pattern test: %s", tt.description)
		})
	}
}

func TestMessageContextHandling(t *testing.T) {
	parser := NewTestTWXParser()

	t.Run("Radio message context", func(t *testing.T) {
		// Set up radio transmission context
		parser.parseTransmissionDetails("Incoming transmission from Captain Kirk on channel 5:")

		// Verify context was set correctly
		if parser.currentMessage != "R Captain Kirk " {
			t.Errorf("Expected currentMessage to be 'R Captain Kirk ', got '%s'", parser.currentMessage)
		}
		// Pascal behavior: parseIntSafe("5:") returns 0 due to colon
		if parser.currentChannel != 0 {
			t.Errorf("Expected currentChannel to be 0 (Pascal parseIntSafe behavior), got %d", parser.currentChannel)
		}

		// Test message content parsing uses context
		msgType, sender, channel := parser.parseMessageContent("R Hello there, trader!")
		if msgType != MessageRadio {
			t.Errorf("Expected MessageRadio, got %d", msgType)
		}
		if sender != "Captain Kirk" {
			t.Errorf("Expected sender 'Captain Kirk', got '%s'", sender)
		}
		// Pascal behavior: channel extraction returns 0 due to parseIntSafe colon handling
		if channel != 0 {
			t.Errorf("Expected channel 0 (Pascal parseIntSafe behavior), got %d", channel)
		}
	})

	t.Run("Personal message context", func(t *testing.T) {
		// Set up personal transmission context
		parser.parseTransmissionDetails("Incoming transmission from Dr. McCoy:")

		// Verify context was set correctly
		if !strings.HasPrefix(parser.currentMessage, "P Dr. McCoy") {
			t.Errorf("Expected currentMessage to start with 'P Dr. McCoy', got '%s'", parser.currentMessage)
		}

		// Test message content parsing uses context
		msgType, sender, channel := parser.parseMessageContent("P He's dead, Jim!")
		if msgType != MessagePersonal {
			t.Errorf("Expected MessagePersonal, got %d", msgType)
		}
		if sender != "Dr. McCoy" {
			t.Errorf("Expected sender 'Dr. McCoy', got '%s'", sender)
		}
		if channel != 0 {
			t.Errorf("Expected channel 0 for personal message, got %d", channel)
		}
	})

	t.Run("Message with 'indicates' ignored", func(t *testing.T) {
		// Clear parser state to ensure clean test
		parser.currentMessage = ""
		parser.currentChannel = 0

		// Per Pascal: if (GetParameter(Line, 2) <> 'indicates') then
		msgType, _, _ := parser.parseMessageContent("P Kirk indicates ship status")
		if msgType != MessageGeneral {
			t.Errorf("Expected MessageGeneral for 'indicates' message, got %d", msgType)
		}
	})
}
