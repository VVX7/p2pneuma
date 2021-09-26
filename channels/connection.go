package channels

import (
	"github.com/preludeorg/pneuma/util"
)

// InitConnectionManager manages read/write ops on the Connection map.
func InitConnectionManager() {
	// Reads the AgentConfig and returns a slice of Connections for each Contact with a target.
	agent := ReadAgentConfig()
	connections := make(map[string]*util.Connection)
	util.RefreshConnections(agent, connections)

	for {
		op := <-ConnectionOpsChannel
		switch {
		// Read ops return a slice of pointers to Connections.
		case op.Type == "read":
			op.ResponseConnections <- connections
		// Close ops closes a Connection of the specified name.
		case op.Type == "close":
			connections[op.Name].Cleanup()
			op.ResponseStatus <- true
		// Refresh ops restart closed connections.
		case op.Type == "refresh":
			util.RemoveClosedConnection(connections)
			util.RefreshConnections(agent, connections)
			op.ResponseStatus <- true
		}
	}
}
