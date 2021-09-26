package channels

import "github.com/preludeorg/pneuma/util"

// InitAgentConfigManager manages read/write ops on the AgentConfig.
func InitAgentConfigManager(agent *util.AgentConfig) {
	// Loop read/write ops on the config.
	for {
		op := <-AgentConfigOpsChannel
		switch {
		case op.Type == "read":
			//
			op.ResponseConfig <- agent
		case op.Type == "write":
			//
			agent = op.Config
			op.ResponseStatus <- true
		default:
			util.DebugLogf("[AgentConfig goroutine] unknown op type: %#v\n", op)
		}
	}
}
