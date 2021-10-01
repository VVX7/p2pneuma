package sockets

import (
	"github.com/preludeorg/pneuma/channels"
	"github.com/preludeorg/pneuma/commands"
	"github.com/preludeorg/pneuma/util"
	"strings"
	"sync"
)

type RPCHandler func(*util.Beacon, *util.Connection)

type ExecutorHandler func(*util.Beacon)

type P2PHandler func(*util.Beacon, *util.Connection)

var (
	rpcHandlers = map[string]RPCHandler{
		"rpcCd": cdHandler,
	}
)

var (
	executorHandlers = map[string]ExecutorHandler{
		"executor": executorHandler,
	}
)

var (
	p2pHandlers = map[string]P2PHandler{
		"p2pExecutor": p2pExecutorHandler,
		"p2pC2Bridge": p2pC2BridgeHandler,
	}
)

func GetRPCHandlers() map[string]RPCHandler {
	return rpcHandlers
}

func GetExecutorHandlers() map[string]ExecutorHandler {
	return executorHandlers
}

func GetP2PHandlers() map[string]P2PHandler {
	return p2pHandlers
}

// EnvelopeHandler passes an envelope to the correct message handler.
func EnvelopeHandler(envelope *util.Envelope) {
	// Init the envelope handlers.
	rpcHandlers := GetRPCHandlers()
	executorHandlers := GetExecutorHandlers()
	p2pHandlers := GetP2PHandlers()

	// Read the active Connections.
	//connections := util.ReadConnections()

	// Check by name if the envelope connection still exists in the active connections
	// and fall through to any connection of the same connection type if it does not.
	//var connection *util.Connection
	//if conn, ok := connections[envelope.ConnectionName]; ok {
	//	connection = conn
	//} else {
	//	for k, v := range connections {
	//		if envelope.Type == v.Type {
	//			envelope.ConnectionName = k
	//			connection = v
	//		}
	//	}
	//}

	// Pass the envelope to its handler.
	// For now, this is always the executorHandler or p2pHandler.
	// TODO: add RPC handler to support Sliver-like RPC calls.
	if handler, ok := rpcHandlers[envelope.Type]; ok {
		handler(envelope.Beacon, envelope.Connection)
	} else if handler, ok := executorHandlers[envelope.Type]; ok {
		handler(envelope.Beacon)
	} else if handler, ok := p2pHandlers[envelope.Type]; ok {
		handler(envelope.Beacon, envelope.Connection)
	} else {
		util.DebugLogf("[%s] Unknown envelope type.", envelope.Type)
	}
}

func cdHandler(beacon *util.Beacon, conn *util.Connection) {
	// TODO: implement Sliver-like RPC handlers
}

// p2pExecutorHandler passes a beacon to the executors.
func p2pExecutorHandler(beacon *util.Beacon, conn *util.Connection) {
	// Falls through to the executorHandler.
	executorHandler(beacon)
}

// p2pC2BridgeHandler passes a beacon to active Connection.Send channels.
// This forwards beacons from p2p-only nodes to Operator.
func p2pC2BridgeHandler(beacon *util.Beacon, conn *util.Connection) {
	// Get the active connections.
	connections := channels.ReadConnections()
	// Wait for each contact's EventLoop to complete before sending the next beacon.
	var wg sync.WaitGroup
	// Forward the beacon to each connection.Send chan
	for _, conn := range connections {
		wg.Add(1)

		// Construct the Envelope that holds the Beacon and the Connection.
		envelope := util.BuildEnvelope(beacon, conn)

		// EnvelopeForwarder passes the beacon to the connection send channel.
		go util.EnvelopeForwarder(conn, envelope, &wg)
	}
	wg.Wait()

}

// executorHandler calls runLinks on a Beacon and executes each Link.
// This is equivalent to the original Pneuma execution method via respond calling runLinks.
func executorHandler(beacon *util.Beacon) {
	// Copy the Beacon and remove Links.
	tmpBeacon := channels.ReadBeacon("tcp")
	tmpBeacon.Links = tmpBeacon.Links[:0]

	// Execute each Beacon Link adding the result to responseBeacon.
	agent := channels.ReadAgentConfig()
	runLinks(beacon, tmpBeacon, agent, "\r\n")

	// Update the AgentConfig.
	_ = channels.RefreshBeacon()

	// Update the channel Beacon by appending the results of the executor.
	_ = channels.AppendBeaconLinks(tmpBeacon.Links)
}

// runLinks handles each Link in the Beacon, calling an executor or downloading a payload as needed.
func runLinks(beacon *util.Beacon, tmpBeacon *util.Beacon, agent *util.AgentConfig, delimiter string) {
	for _, link := range beacon.Links {
		// Set link state to executing.
		_ = channels.WriteCacheLink("executing", false, link.ID)

		// Download the payload.
		var payloadPath string
		var payloadErr error
		if len(link.Payload) > 0 {
			payloadPath, payloadErr = requestPayload(link.Payload)
		}

		// Perform execution.
		if payloadErr == nil {
			response, status, pid := commands.RunCommand(link.Request, link.Executor, payloadPath, agent)
			link.Response = strings.TrimSpace(response) + delimiter
			link.Status = status
			link.Pid = pid
		} else {
			payloadErrorResponse(payloadErr, agent, &link)
		}

		// After execution set the Link status to complete.
		_ = channels.WriteCacheLink("complete", false, link.ID)

		// Add each completed Link to the tmpBeacon.
		tmpBeacon.Links = append(tmpBeacon.Links, link)
	}
}
