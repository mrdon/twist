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

// connectImpl is implemented in proxy package to avoid circular dependency
// This will be injected at runtime
var connectImpl func(string, TuiAPI) ProxyAPI

// SetConnectImpl allows the proxy package to register its implementation
func SetConnectImpl(impl func(string, TuiAPI) ProxyAPI) {
	connectImpl = impl
}