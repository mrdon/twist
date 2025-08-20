package streaming

import "twist/internal/proxy/database"

// EventType represents different types of events that can be fired
type EventType int

const (
	EventTextLine EventType = iota
	EventText
	EventTrigger
	EventAutoText
	EventSectorComplete
	EventParseComplete
	EventStateChange
	EventMessageReceived
	EventDatabaseUpdate
)

// Event represents a generic event in the system
type Event struct {
	Type      EventType
	Data      interface{}
	Source    string
	Timestamp int64
}

// EventHandler defines the signature for event handling functions
type EventHandler func(event Event)

// IEventBus defines the interface for event communication
type IEventBus interface {
	Subscribe(eventType EventType, handler EventHandler) string
	Unsubscribe(eventType EventType, subscriptionID string)
	Fire(event Event)
	FireAsync(event Event)
}

// IModExtractor defines the main parser interface (Pascal: IModExtractor)
type IModExtractor interface {
	// Core parsing methods
	ProcessInBound(data string)
	ProcessOutBound(data string) bool

	// State management
	GetCurrentSector() int
	GetCurrentDisplay() DisplayType
	SetCurrentDisplay(display DisplayType)

	// Event integration
	SetEventBus(bus IEventBus)
	GetEventBus() IEventBus

	// Script integration points
	FireTextEvent(line string, outbound bool)
	FireTextLineEvent(line string, outbound bool)
	ActivateTriggers()
	FireAutoTextEvent(line string, outbound bool)

	// Database integration
	GetDatabase() database.Database
	SetDatabase(db database.Database)
}

// ITWXModule defines the base interface for all TWX modules
type ITWXModule interface {
	// Module lifecycle
	Initialize() error
	Shutdown() error

	// Event handling
	OnEvent(event Event)

	// Module identification
	GetModuleName() string
	GetModuleVersion() string
}

// IScriptInterpreter defines the interface for script execution
type IScriptInterpreter interface {
	// Script event firing (mirrors Pascal TWXInterpreter)
	TextEvent(line string, outbound bool)
	TextLineEvent(line string, outbound bool)
	ActivateTriggers()
	AutoTextEvent(line string, outbound bool)

	// Script management
	LoadScript(filename string) error
	UnloadScript(name string) error
	ExecuteScript(name string, params map[string]interface{}) error
}

// Observer pattern interfaces
type IObserver interface {
	Update(subject ISubject, event Event)
	GetObserverID() string
}

type ISubject interface {
	Attach(observer IObserver)
	Detach(observerID string)
	Notify(event Event)
}

// IGameStateManager manages overall game state
type IGameStateManager interface {
	// State queries
	IsInGame() bool
	GetCurrentGame() string
	GetCurrentSector() int

	// State changes
	OnGameStart(gameName string)
	OnGameExit()
	OnSectorChange(sectorNum int)

	// Event integration
	SetEventBus(bus IEventBus)
}
