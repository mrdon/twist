package telnet

import ()

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
	
	// SAUCE detection state
	sauceBuffer  []byte
	sauceTarget  []byte
}

// NewHandler creates a new telnet protocol handler
func NewHandler(writer func([]byte) error) *Handler {
	return &Handler{
		writer: writer,
		sauceTarget: []byte{0x1A, 'S', 'A', 'U', 'C', 'E', '0', '0'},
	}
}

// SendInitialNegotiation sends the initial telnet option negotiations
func (h *Handler) SendInitialNegotiation() error {
	
	// If no writer is available, skip telnet negotiation
	if h.writer == nil {
		return nil // Success - no negotiation needed
	}
	
	// Send basic telnet client capabilities
	commands := [][]byte{
		{IAC, WILL, TERMINAL_TYPE},    // We support terminal type
		{IAC, WILL, NAWS},             // We support window size negotiation
		{IAC, DO, ECHO},               // Server should handle echo
		{IAC, WILL, SUPPRESS_GO_AHEAD}, // We support suppress go ahead
		{IAC, DO, SUPPRESS_GO_AHEAD},   // Server should suppress go ahead
	}
	
	for _, cmd := range commands {
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
				i += 2
			}
		} else {
			// Regular data byte
			result = append(result, data[i])
			i++
		}
	}
	
	// Filter out SAUCE records (ANSI art metadata)
	result = h.filterSAUCE(result)
	
	
	return result
}

// filterSAUCE removes SAUCE records (ANSI art metadata) from streaming data
func (h *Handler) filterSAUCE(data []byte) []byte {
	var result []byte
	
	for _, b := range data {
		// Add byte to SAUCE buffer
		h.sauceBuffer = append(h.sauceBuffer, b)
		
		// Check if we're building toward SAUCE header
		if len(h.sauceBuffer) <= len(h.sauceTarget) {
			// Still potentially matching SAUCE header
			if h.sauceBuffer[len(h.sauceBuffer)-1] == h.sauceTarget[len(h.sauceBuffer)-1] {
				// Byte matches, continue building
				if len(h.sauceBuffer) == len(h.sauceTarget) {
					// Complete SAUCE header detected - drop everything in buffer
						h.sauceBuffer = nil
					// From here on, drop all remaining data (SAUCE record continues)
					return result
				}
				// Partial match, don't output yet
				continue
			} else {
				// No match, output buffered data and reset
				result = append(result, h.sauceBuffer...)
				h.sauceBuffer = nil
			}
		} else {
			// We're past SAUCE header length and still buffering = drop data
			// (We're in a SAUCE record, drop everything)
			continue
		}
	}
	
	return result
}

// handleNegotiation processes telnet option negotiations
func (h *Handler) handleNegotiation(cmd byte, option byte) {
	
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
		if err := h.writer(response); err != nil {
			// Failed to send response
		}
	}
}