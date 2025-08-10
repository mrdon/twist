package api

// Connect creates a new proxy instance and returns a connected ProxyAPI
func Connect(address string, tuiAPI TuiAPI) ProxyAPI {
	// Ensure implementation is registered
	if connectImpl == nil {
		panic("Connect implementation not registered - proxy package may not be imported")
	}
	// Delegate to implementation registered by proxy package
	return connectImpl(address, tuiAPI)
}

// ConnectWithScript creates a new proxy instance with initial script and returns a connected ProxyAPI
func ConnectWithScript(address string, tuiAPI TuiAPI, scriptName string) ProxyAPI {
	// Ensure implementation is registered
	if connectWithScriptImpl == nil {
		panic("ConnectWithScript implementation not registered - proxy package may not be imported")
	}
	// Delegate to implementation registered by proxy package
	return connectWithScriptImpl(address, tuiAPI, scriptName)
}

// connectImpl is implemented in proxy package to avoid circular dependency
// This will be injected at runtime
var connectImpl func(string, TuiAPI) ProxyAPI
var connectWithScriptImpl func(string, TuiAPI, string) ProxyAPI

// SetConnectImpl allows the proxy package to register its implementation
func SetConnectImpl(impl func(string, TuiAPI) ProxyAPI) {
	connectImpl = impl
}

// SetConnectWithScriptImpl allows the proxy package to register its implementation
func SetConnectWithScriptImpl(impl func(string, TuiAPI, string) ProxyAPI) {
	connectWithScriptImpl = impl
}