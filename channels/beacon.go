package channels

import (
	"github.com/preludeorg/pneuma/util"
)

// InitBeaconManager goroutine manages read/write ops on the Beacon.
func InitBeaconManager() {
	// Init the Beacon from AgentConfig to set default values.
	agent := ReadAgentConfig()
	beacon := agent.BuildBeacon("tcp")
	// Loop read/write ops on the beacon.
	for {
		op := <-BeaconOpsChannel
		switch {
		// op.Contact string is used to select the contact target from the AgentConfig.
		// Instead of mutating the beacon, which should be initiated by a write op, read returns a copy.
		case op.Type == "read":
			b := beacon
			b.Target = agent.Contact[op.Contact]
			op.ResponseBeacon <- &b
		// Calls RefreshBeacon to update the Beacon with the current AgentConfig.
		case op.Type == "refresh":
			a := ReadAgentConfig()
			util.RefreshBeacon(a, &beacon, "tcp")
			op.ResponseStatus <- true
		// Append Links to the Beacon.
		case op.Type == "append":
			for _, link := range op.Links {
				beacon.Links = append(beacon.Links, link)
			}
			op.ResponseStatus <- true
		// Removes complete and sent links from the Beacon.
		case op.Type == "trim":
			TrimSentLinks(&beacon)
			op.ResponseStatus <- true
		default:
			util.DebugLogf("[Beacon goroutine] unknown op type: %#v\n", op)
		}
	}
}
