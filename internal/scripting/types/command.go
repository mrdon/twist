package types

// ParameterType represents the type of a command parameter
type ParameterType int

const (
	ParamValue ParameterType = iota // Input parameter (value or variable reference)
	ParamVar                        // Output parameter (variable that receives result)
)

// CommandParam represents a parameter to a script command
type CommandParam struct {
	Type       ParameterType
	Value      *Value
	VarName    string
	IsVariable bool
}

// Command represents a compiled script command
type Command struct {
	Name       string
	Params     []*CommandParam
	LineNumber int
	ScriptID   string
}

// CommandDef represents a command definition
type CommandDef struct {
	Name       string
	MinParams  int
	MaxParams  int // -1 for unlimited
	ParamTypes []ParameterType
	ReturnType ParameterType
	Handler    CommandHandler
}

// CommandHandler is the function signature for command implementations
type CommandHandler func(vm VMInterface, params []*CommandParam) error

// VMInterface defines the interface that command handlers use to interact with the VM
type VMInterface interface {
	// Variable management
	GetVariable(name string) *Value
	SetVariable(name string, value *Value)
	GetVarParam(name string) *VarParam
	SetVarParam(name string, varParam *VarParam)
	
	// Script control
	Goto(label string) error
	Gosub(label string) error
	Return() error
	Halt() error
	Pause() error
	
	// Output
	Echo(message string) error
	ClientMessage(message string) error
	
	// Input
	GetInput(prompt string) (string, error)
	WaitFor(text string) error
	
	// Network
	Send(data string) error
	
	// Game interface
	GetGameInterface() GameInterface
	
	// Script management
	GetCurrentScript() ScriptInterface
	LoadAdditionalScript(filename string) (ScriptInterface, error)
	StopScript(scriptID string) error
	GetScriptManager() interface{} // Returns the script manager for advanced operations
	
	// Trigger management
	SetTrigger(trigger TriggerInterface) error
	KillTrigger(triggerID string) error
	GetActiveTriggersCount() int
	KillAllTriggers()
	
	// Error handling
	Error(message string) error
	
	// Processing filters (for testing)
	ProcessInput(filter string) error
	ProcessOutput(filter string) error
}

// GameInterface defines the interface for interacting with the game
type GameInterface interface {
	// Database access
	GetSector(index int) (SectorData, error)
	SetSectorParameter(sector int, name, value string) error
	GetSectorParameter(sector int, name string) (string, error)
	GetDatabase() interface{} // Returns the underlying database for script management
	
	// Navigation
	GetCourse(from, to int) ([]int, error)
	GetDistance(from, to int) (int, error)
	GetAllCourses(from int) (map[int][]int, error)
	GetNearestWarps(sector int, count int) ([]int, error)
	
	// Current state
	GetCurrentSector() int
	GetCurrentPrompt() string
	
	// Network
	SendCommand(cmd string) error
	GetLastOutput() string
	
	// Script variable persistence
	SaveScriptVariable(name string, value *Value) error
	LoadScriptVariable(name string) (*Value, error)
}

// SectorData represents sector information
type SectorData struct {
	Number        int
	Warps         []int
	NavHaz        int
	Constellation string
	Beacon        string
	Density       int
	Anomaly       bool
	Explored      int
	HasPort       bool
	PortName      string
	PortClass     int
	Ships         []ShipData
	Traders       []TraderData
	Planets       []PlanetData
}

// ShipData represents ship information
type ShipData struct {
	Name     string
	Owner    string
	ShipType string
	Fighters int
}

// TraderData represents trader information
type TraderData struct {
	Name     string
	ShipType string
	ShipName string
	Fighters int
}

// PlanetData represents planet information
type PlanetData struct {
	Name string
}

// ScriptInterface defines the interface for script objects
type ScriptInterface interface {
	GetID() string
	GetFilename() string
	GetName() string
	IsRunning() bool
	IsSystem() bool
	Stop() error
}