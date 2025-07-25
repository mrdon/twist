package telnet

import (
	"log"
	"os"
)

// Telnet command constants
const (
	IAC  = 0xFF // Interpret As Command
	DONT = 0xFE // Don't use option
	DO   = 0xFD // Use option
	WONT = 0xFC // Won't use option
	WILL = 0xFB // Will use option
	SB   = 0xFA // Subnegotiation Begin
	SE   = 0xF0 // Subnegotiation End
)

// Telnet option constants
const (
	ECHO             = 0x01
	SUPPRESS_GO_AHEAD = 0x03
	TERMINAL_TYPE    = 0x18
	NAWS            = 0x1F // Negotiate About Window Size
)

// Handler manages telnet protocol negotiation
type Handler struct {
	writer func([]byte) error
	logger *log.Logger
}

// NewHandler creates a new telnet protocol handler
func NewHandler(writer func([]byte) error) *Handler {
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	
	logger := log.New(logFile, "[TELNET] ", log.LstdFlags|log.Lshortfile)
	
	return &Handler{
		writer: writer,
		logger: logger,
	}
}

// SendInitialNegotiation sends the initial telnet option negotiations
func (h *Handler) SendInitialNegotiation() error {
	h.logger.Println("Sending initial telnet negotiation")
	
	// Send basic telnet client capabilities
	commands := [][]byte{
		{IAC, WILL, TERMINAL_TYPE},    // We support terminal type
		{IAC, WILL, NAWS},             // We support window size negotiation
		{IAC, DO, ECHO},               // Server should handle echo
		{IAC, WILL, SUPPRESS_GO_AHEAD}, // We support suppress go ahead
		{IAC, DO, SUPPRESS_GO_AHEAD},   // Server should suppress go ahead
	}
	
	for _, cmd := range commands {
		h.logger.Printf("Sending: IAC %02x %02x", cmd[1], cmd[2])
		if err := h.writer(cmd); err != nil {
			return err
		}
	}
	
	return nil
}

// ProcessData filters telnet commands from incoming data and returns clean text
func (h *Handler) ProcessData(data []byte) []byte {
	var result []byte
	i := 0
	
	for i < len(data) {
		if data[i] == IAC && i+1 < len(data) {
			cmd := data[i+1]
			h.logger.Printf("Received telnet command: IAC %02x", cmd)
			
			switch cmd {
			case DONT, DO, WONT, WILL:
				// Three-byte commands: IAC + command + option
				if i+2 < len(data) {
					option := data[i+2]
					h.handleNegotiation(cmd, option)
					i += 3
				} else {
					// Incomplete command, skip what we have
					i = len(data)
				}
				
			case SB:
				// Subnegotiation: skip until IAC SE
				i += 2
				for i < len(data) {
					if data[i] == IAC && i+1 < len(data) && data[i+1] == SE {
						i += 2 // Skip IAC SE
						break
					}
					i++
				}
				
			case SE:
				// Subnegotiation end (should be handled above)
				i += 2
				
			case IAC:
				// Escaped IAC (0xFF 0xFF represents literal 0xFF)
				result = append(result, IAC)
				i += 2
				
			default:
				// Other two-byte commands
				h.logger.Printf("Unknown telnet command: %02x", cmd)
				i += 2
			}
		} else {
			// Regular data byte
			result = append(result, data[i])
			i++
		}
	}
	
	if len(result) > 0 {
		h.logger.Printf("Filtered data: %q", string(result))
	}
	
	return result
}

// handleNegotiation processes telnet option negotiations
func (h *Handler) handleNegotiation(cmd byte, option byte) {
	h.logger.Printf("Negotiation: %02x %02x", cmd, option)
	
	var response []byte
	
	switch cmd {
	case DO: // Server wants us to enable option
		switch option {
		case ECHO:
			// We don't want to echo, let server handle it
			response = []byte{IAC, WONT, ECHO}
		case SUPPRESS_GO_AHEAD:
			// We support suppress go ahead
			response = []byte{IAC, WILL, SUPPRESS_GO_AHEAD}
		case TERMINAL_TYPE:
			// We support terminal type
			response = []byte{IAC, WILL, TERMINAL_TYPE}
		case NAWS:
			// We support window size negotiation
			response = []byte{IAC, WILL, NAWS}
		default:
			// Don't support unknown options
			response = []byte{IAC, WONT, option}
		}
		
	case DONT: // Server doesn't want us to use option
		// Acknowledge by saying we won't
		response = []byte{IAC, WONT, option}
		
	case WILL: // Server will enable option
		switch option {
		case ECHO:
			// Good, server will handle echo
			response = []byte{IAC, DO, ECHO}
		case SUPPRESS_GO_AHEAD:
			// Good, server will suppress go ahead
			response = []byte{IAC, DO, SUPPRESS_GO_AHEAD}
		default:
			// Don't care about other options server enables
			response = []byte{IAC, DONT, option}
		}
		
	case WONT: // Server won't enable option
		// Acknowledge
		response = []byte{IAC, DONT, option}
	}
	
	if response != nil {
		h.logger.Printf("Responding: IAC %02x %02x", response[1], response[2])
		if err := h.writer(response); err != nil {
			h.logger.Printf("Failed to send response: %v", err)
		}
	}
}