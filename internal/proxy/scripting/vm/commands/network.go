package commands

import (
	"fmt"
	"net"
	"sync"
	"time"
	"twist/internal/proxy/scripting/types"
)

// NetworkConnection manages network connections for VMs
type NetworkConnection struct {
	conn net.Conn
	host string
	port string
	mu   sync.RWMutex
}

// Global connection manager for VMs
var connectionManager = struct {
	connections map[types.VMInterface]*NetworkConnection
	mu          sync.RWMutex
}{
	connections: make(map[types.VMInterface]*NetworkConnection),
}

// RegisterNetworkCommands registers network commands with the VM
func RegisterNetworkCommands(vm CommandRegistry) {
	vm.RegisterCommand("CONNECT", 2, 2, []types.ParameterType{types.ParamValue, types.ParamValue}, cmdConnect)
	vm.RegisterCommand("DISCONNECT", 0, 0, []types.ParameterType{}, cmdDisconnect)
}

// getConnection gets or creates a network connection for a VM
func getConnection(vm types.VMInterface) *NetworkConnection {
	connectionManager.mu.RLock()
	conn, exists := connectionManager.connections[vm]
	connectionManager.mu.RUnlock()
	
	if !exists {
		connectionManager.mu.Lock()
		conn = &NetworkConnection{}
		connectionManager.connections[vm] = conn
		connectionManager.mu.Unlock()
	}
	
	return conn
}

// IsConnected checks if the VM has an active connection
func IsConnected(vm types.VMInterface) bool {
	conn := getConnection(vm)
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.conn != nil
}

// GetConnectionInfo returns connection host and port
func GetConnectionInfo(vm types.VMInterface) (string, string) {
	conn := getConnection(vm)
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.host, conn.port
}

// cmdConnect connects to a host
func cmdConnect(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 2 {
		return vm.Error("CONNECT requires exactly 2 parameters: host, port")
	}

	host := GetParamString(vm, params[0])
	port := GetParamString(vm, params[1])
	
	// Validate parameters
	if host == "" {
		return vm.Error("CONNECT requires a non-empty host parameter")
	}
	if port == "" {
		return vm.Error("CONNECT requires a non-empty port parameter")
	}
	
	// Get or create connection for this VM
	netConn := getConnection(vm)
	
	netConn.mu.Lock()
	defer netConn.mu.Unlock()
	
	// Close existing connection if any
	if netConn.conn != nil {
		netConn.conn.Close()
	}
	
	// Establish new connection with timeout
	address := fmt.Sprintf("%s:%s", host, port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return vm.Error(fmt.Sprintf("Failed to connect to %s: %v", address, err))
	}
	
	netConn.conn = conn
	netConn.host = host
	netConn.port = port
	
	// Store connection info in VM variables for script access
	vm.SetVariable("__connected", &types.Value{Type: types.NumberType, Number: 1})
	vm.SetVariable("__connectHost", &types.Value{Type: types.StringType, String: host})
	vm.SetVariable("__connectPort", &types.Value{Type: types.StringType, String: port})
	
	return nil
}

// cmdDisconnect disconnects from the current host
func cmdDisconnect(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 0 {
		return vm.Error("DISCONNECT requires no parameters")
	}

	netConn := getConnection(vm)
	
	netConn.mu.Lock()
	defer netConn.mu.Unlock()
	
	if netConn.conn != nil {
		netConn.conn.Close()
		netConn.conn = nil
		netConn.host = ""
		netConn.port = ""
	}
	
	// Clear connection info in VM variables
	vm.SetVariable("__connected", &types.Value{Type: types.NumberType, Number: 0})
	vm.SetVariable("__connectHost", &types.Value{Type: types.StringType, String: ""})
	vm.SetVariable("__connectPort", &types.Value{Type: types.StringType, String: ""})
	
	return nil
}


// CleanupConnections closes all connections for cleanup (useful for tests)
func CleanupConnections() {
	connectionManager.mu.Lock()
	defer connectionManager.mu.Unlock()
	
	for _, conn := range connectionManager.connections {
		conn.mu.Lock()
		if conn.conn != nil {
			conn.conn.Close()
			conn.conn = nil
		}
		conn.mu.Unlock()
	}
	
	// Clear the map
	connectionManager.connections = make(map[types.VMInterface]*NetworkConnection)
}