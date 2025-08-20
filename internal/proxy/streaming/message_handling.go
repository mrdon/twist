package streaming

import (
	"strings"
	"time"
	"twist/internal/proxy/database"
)

// message_handling.go - Clean message parsing and handling logic
// This focuses purely on parsing message content, not storage

// MessageHandler handles parsing and processing of game messages
type MessageHandler struct{}

// NewMessageHandler creates a new message handler
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{}
}

// parseTransmissionDetails extracts details from transmission lines (mirrors Pascal Process.pas lines 1199-1228)
func (p *TWXParser) parseTransmissionDetails(line string) (string, int, MessageType) {
	// Mirror Pascal logic exactly for transmission parsing
	
	// Handle "Incoming transmission from" or "Continuing transmission from"
	if strings.HasPrefix(line, "Incoming transmission from") || strings.HasPrefix(line, "Continuing transmission from") {
		return p.parseIncomingTransmission(line)
	}
	
	// Handle "Fighter message from sector" (as expected by tests)
	if strings.HasPrefix(line, "Fighter message from sector") {
		// Extract sector info from "Fighter message from sector 1234:"
		parts := strings.Fields(line)
		for i, part := range parts {
			if part == "sector" && i+1 < len(parts) {
				// Include the colon per TWX behavior as expected by tests
				sender := "sector " + parts[i+1]
				return sender, 0, MessageFighter
			}
		}
		return "", 0, MessageFighter
	}
	
	// Handle "Computer message:" (as expected by tests)
	if strings.HasPrefix(line, "Computer message") {
		return "Computer", 0, MessageComputer
	}
	
	// Handle "Deployed Fighters Report Sector"
	if strings.HasPrefix(line, "Deployed Fighters Report Sector") {
		// Extract sector info from fighter report
		sector := p.extractSectorFromFighterReport(line)
		return "sector " + sector, 0, MessageDeployed
	}
	
	// Handle "Shipboard Computers "
	if strings.HasPrefix(line, "Shipboard Computers ") {
		return "Computer", 0, MessageShipboard
	}
	
	return "", 0, MessageGeneral
}

// parseRadioTransmission parses radio transmission format
func (p *TWXParser) parseRadioTransmission(line string) (string, int, MessageType) {
	parts := strings.Fields(line)
	for i, part := range parts {
		if part == "channel" && i+1 < len(parts) {
			channel := p.parseIntSafe(parts[i+1])
			sender := ""
			if i > 3 {
				sender = strings.Join(parts[3:i-1], " ")
			}
			return sender, channel, MessageRadio
		}
	}
	return "", 0, MessageRadio
}

// parseFighterMessage parses fighter message format
func (p *TWXParser) parseFighterMessage(line string) (string, int, MessageType) {
	// Extract sender from "Fighter message from sector 1234:"
	parts := strings.Fields(line)
	for i, part := range parts {
		if part == "sector" && i+1 < len(parts) {
			sender := "sector " + parts[i+1]
			return sender, 0, MessageFighter
		}
	}
	return "", 0, MessageFighter
}

// parseIncomingTransmission handles incoming/continuing transmission parsing (Pascal lines 1199-1228)
func (p *TWXParser) parseIncomingTransmission(line string) (string, int, MessageType) {
	// Pascal: I := GetParameterPos(Line, 4);
	// Extract sender name starting from parameter 4
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return "", 0, MessageGeneral
	}
	
	// Check for Federation comm-link (Pascal: Copy(Line, Length(Line) - 9, 10) = 'comm-link:')
	if strings.HasSuffix(line, "comm-link:") {
		// Pascal: FCurrentMessage := 'F ' + Copy(Line, I, Pos(' on Federation', Line) - I) + ' ';
		fedPos := strings.Index(line, " on Federation")
		if fedPos > 0 && len(parts) >= 4 {
			// Extract sender name from parameter 4 to " on Federation"
			senderStart := strings.Index(line, parts[3]) // Start of 4th parameter
			sender := line[senderStart:fedPos]
			p.currentMessage = "F " + strings.TrimSpace(sender) + " "
			return strings.TrimSpace(sender), 0, MessageFedlink
		}
		p.currentMessage = "F  "
		return "", 0, MessageFedlink
	}
	
	// Check for Fighters (Pascal: GetParameter(Line, 5) = 'Fighters:')
	if len(parts) >= 4 && parts[3] == "Fighters:" {
		// Pascal: FCurrentMessage := 'Figs';
		p.currentMessage = "Figs"
		return "Fighters", 0, MessageFighter
	}
	
	// Check for Computers (Pascal: GetParameter(Line, 5) = 'Computers:')
	if len(parts) >= 4 && parts[3] == "Computers:" {
		// Pascal: FCurrentMessage := 'Comp';
		p.currentMessage = "Comp"
		return "Computer", 0, MessageComputer
	}
	
	// Check for radio transmission on channel (Pascal: Pos(' on channel ', Line) <> 0)
	if strings.Contains(line, " on channel ") {
		// Pascal: FCurrentMessage := 'R ' + Copy(Line, I, Pos(' on channel ', Line) - I) + ' ';
		channelPos := strings.Index(line, " on channel ")
		if channelPos > 0 && len(parts) >= 4 {
			senderStart := strings.Index(line, parts[3]) // Start of 4th parameter
			sender := line[senderStart:channelPos]
			
			// Extract channel number
			channelStr := line[channelPos+12:] // After " on channel "
			channelParts := strings.Fields(channelStr)
			channel := 0
			if len(channelParts) > 0 {
				// Mirror TWX Pascal behavior: parseIntSafe fails on "1:" because of colon
				// This is the correct Pascal behavior - don't strip colon
				channel = p.parseIntSafe(channelParts[0])
			}
			
			// Store channel for message context (mirrors Pascal FCurrentMessage logic)
			p.currentChannel = channel
			p.currentMessage = "R " + strings.TrimSpace(sender) + " "
			
			return strings.TrimSpace(sender), channel, MessageRadio
		}
	}
	
	// Default case - personal hail (Pascal: FCurrentMessage := 'P ' + Copy(Line, I, Length(Line) - I) + ' ';)
	if len(parts) >= 4 {
		senderStart := strings.Index(line, parts[3]) // Start of 4th parameter
		sender := line[senderStart:]
		// Remove trailing colon if present
		sender = strings.TrimSuffix(strings.TrimSpace(sender), ":")
		p.currentMessage = "P " + sender + " "
		return sender, 0, MessagePersonal
	}
	
	return "", 0, MessageGeneral
}

// extractSectorFromFighterReport extracts sector number from fighter report line
func (p *TWXParser) extractSectorFromFighterReport(line string) string {
	// Pascal: Copy(Line, 19, Length(Line)) for "Deployed Fighters Report Sector"
	if len(line) > 31 {
		sectorPart := line[31:] // After "Deployed Fighters Report Sector"
		parts := strings.Fields(sectorPart)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

// handleEnhancedMessageLine processes message content with database integration
func (p *TWXParser) handleEnhancedMessageLine(line string) {
	
	// Determine message type and extract sender/channel info
	msgType, sender, channel := p.parseMessageContent(line)
	
	// Add to history with database integration
	if err := p.addToHistory(msgType, line, sender, channel); err != nil {
	}
}

// parseMessageContent parses message content to extract type, sender, and channel (mirrors Pascal lines 1192-1228)
func (p *TWXParser) parseMessageContent(line string) (MessageType, string, int) {
	// Mirror Pascal logic exactly: 
	// else if (Copy(Line, 1, 2) = 'R ') or (Copy(Line, 1, 2) = 'F ') then
	//   TWXGUI.AddToHistory(htMsg, TimeToStr(Time) + '  ' + StripChars(Line))
	if strings.HasPrefix(line, "R ") || strings.HasPrefix(line, "F ") {
		// These are direct message content lines, parse based on current message context
		if strings.HasPrefix(p.currentMessage, "R ") {
			// Radio message - extract sender and channel from currentMessage
			return p.parseCurrentRadioMessage(line)
		} else if strings.HasPrefix(p.currentMessage, "F ") {
			// Fedlink message - extract sender from currentMessage
			return p.parseCurrentFedlinkMessage(line)
		}
		return MessageGeneral, "", 0
	}
	
	// Pascal: else if (Copy(Line, 1, 2) = 'P ') then
	if strings.HasPrefix(line, "P ") {
		// Pascal: if (GetParameter(Line, 2) <> 'indicates') then
		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[2] == "indicates" {
			// Skip messages with 'indicates' per Pascal logic (parameter 2 is 0-indexed as parts[2])
			return MessageGeneral, "", 0
		}
		// Personal message - extract sender from currentMessage
		return p.parseCurrentPersonalMessage(line)
	}
	
	// Handle direct message lines (not part of transmission context)
	if strings.HasPrefix(line, "Deployed Fighters Report Sector") {
		sector := p.extractSectorFromFighterReport(line)
		return MessageDeployed, "sector " + sector, 0
	}
	
	if strings.HasPrefix(line, "Shipboard Computers") {
		return MessageShipboard, "Computer", 0
	}
	
	// Handle based on current message context set by transmission headers
	if p.currentMessage == "Figs" {
		return MessageFighter, "", 0
	} else if p.currentMessage == "Comp" {
		return MessageComputer, "", 0
	} else if strings.HasPrefix(p.currentMessage, "P ") {
		return MessagePersonal, p.extractSenderFromCurrentMessage(), 0
	} else if strings.HasPrefix(p.currentMessage, "R ") {
		return MessageRadio, p.extractSenderFromCurrentMessage(), p.extractChannelFromCurrentMessage()
	} else if strings.HasPrefix(p.currentMessage, "F ") {
		return MessageFedlink, p.extractSenderFromCurrentMessage(), 0
	}
	
	return MessageGeneral, "", 0
}

// parseCurrentRadioMessage extracts sender and channel from radio message context
func (p *TWXParser) parseCurrentRadioMessage(line string) (MessageType, string, int) {
	// currentMessage format: "R SenderName "
	sender := p.extractSenderFromCurrentMessage()
	channel := p.extractChannelFromCurrentMessage()
	return MessageRadio, sender, channel
}

// parseCurrentFedlinkMessage extracts sender from fedlink message context
func (p *TWXParser) parseCurrentFedlinkMessage(line string) (MessageType, string, int) {
	// currentMessage format: "F SenderName "
	sender := p.extractSenderFromCurrentMessage()
	return MessageFedlink, sender, 0
}

// parseCurrentPersonalMessage extracts sender from personal message context
func (p *TWXParser) parseCurrentPersonalMessage(line string) (MessageType, string, int) {
	// currentMessage format: "P SenderName "
	sender := p.extractSenderFromCurrentMessage()
	return MessagePersonal, sender, 0
}

// extractSenderFromCurrentMessage extracts sender name from currentMessage
func (p *TWXParser) extractSenderFromCurrentMessage() string {
	if len(p.currentMessage) < 3 {
		return ""
	}
	// Format is "X SenderName " where X is R, F, or P
	senderPart := strings.TrimSpace(p.currentMessage[2:]) // Remove "X " prefix
	return senderPart
}

// extractChannelFromCurrentMessage extracts channel number from radio message context
func (p *TWXParser) extractChannelFromCurrentMessage() int {
	// For radio messages, we need to parse the channel from the original transmission line
	// This is stored separately during parseIncomingTransmission
	return p.currentChannel // Will be set by parseIncomingTransmission
}

// Message history management methods (for backwards compatibility)

// GetMessageHistory returns the in-memory message history
func (p *TWXParser) GetMessageHistory() []MessageHistory {
	return p.messageHistory
}

// ClearHistory clears the message history
func (p *TWXParser) ClearHistory() {
	p.messageHistory = []MessageHistory{}
}

// GetMessageHistoryByType returns messages of a specific type
func (p *TWXParser) GetMessageHistoryByType(msgType MessageType) []MessageHistory {
	var filtered []MessageHistory
	for _, msg := range p.messageHistory {
		if msg.Type == msgType {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// SetHistorySize sets the maximum history size
func (p *TWXParser) SetHistorySize(size int) {
	p.maxHistorySize = size
	// Trim existing history if needed
	if len(p.messageHistory) > size {
		p.messageHistory = p.messageHistory[len(p.messageHistory)-size:]
	}
}

// GetRecentMessages returns the N most recent messages
func (p *TWXParser) GetRecentMessages(count int) []MessageHistory {
	if count <= 0 || len(p.messageHistory) == 0 {
		return []MessageHistory{}
	}
	
	start := len(p.messageHistory) - count
	if start < 0 {
		start = 0
	}
	
	return p.messageHistory[start:]
}

// addToHistory adds a message to both in-memory and database history
func (p *TWXParser) addToHistory(msgType MessageType, content, sender string, channel int) error {
	// Create message history entry
	message := MessageHistory{
		Type:      msgType,
		Timestamp: time.Now(),
		Content:   content,
		Sender:    sender,
		Channel:   channel,
	}
	
	// Add to in-memory history (with size limit)
	p.messageHistory = append(p.messageHistory, message)
	if len(p.messageHistory) > p.maxHistorySize {
		// Remove oldest messages
		p.messageHistory = p.messageHistory[len(p.messageHistory)-p.maxHistorySize:]
	}
	
	// Save to database (required) - convert inline without converter
	dbMessage := database.TMessageHistory{
		Type:      database.TMessageType(message.Type),
		Timestamp: message.Timestamp,
		Content:   message.Content,
		Sender:    message.Sender,
		Channel:   message.Channel,
	}
	if err := p.database.AddMessageToHistory(dbMessage); err != nil {
		return err
	}
	
	return nil
}