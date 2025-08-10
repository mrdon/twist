// Package interfaces contains shared interface definitions to avoid circular dependencies
package interfaces

// ScriptInfo represents information about a running script
type ScriptInfo interface {
	GetID() string
	GetName() string
	GetFilename() string
	IsRunning() bool
}

// ScriptEngine represents the script execution engine
type ScriptEngine interface {
	GetRunningScripts() []ScriptInfo
	GetAllScripts() []ScriptInfo
	GetScriptCount() int
	GetRunningScriptCount() int
	GetStatus() map[string]interface{}
}

// ScriptManager represents the script management system
type ScriptManager interface {
	LoadAndRunScript(filename string) error
	Stop() error
	GetStatus() map[string]interface{}
	GetEngine() ScriptEngine
	HasScriptWaitingForInput() (string, string)
	ResumeScriptWithInput(scriptID, input string) error
}

// ProxyInterface defines methods for proxy operations
type ProxyInterface interface {
	SendInput(input string)
	SendOutput(output string)
	GetScriptManager() ScriptManager
}