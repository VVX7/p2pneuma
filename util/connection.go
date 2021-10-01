package util

// Connection defines a contacts message channels and state.
type Connection struct {
	Name string
	Type string
	Send chan *Envelope
	Recv chan *Envelope
	Ctrl chan bool
	//Once    *sync.Once
	//Mutex   *sync.RWMutex
	IsOpen  bool
	Cleanup func()
}

// RemoveClosedConnection removes closed Connections from the Connection map.
func RemoveClosedConnection(connections map[string]*Connection) {
	for k, conn := range connections {
		if !conn.IsOpen {
			delete(connections, k)
		}
	}
}

// RefreshConnections updates the Connection map with the current active Connections.
func RefreshConnections(agent *AgentConfig, connections map[string]*Connection) {
	for contact, _ := range agent.Contact {
		active := false
		for _, conn := range connections {
			if contact == conn.Type {
				active = true
			}
		}
		if !active {
			name := PickName(8)
			conn, err := CommunicationChannels[contact].Communicate(agent, name)
			if err != nil {
				DebugLogf("[%s] Error initializing connection.", err)
				continue
			}
			connections[name] = conn
		}
	}
}
