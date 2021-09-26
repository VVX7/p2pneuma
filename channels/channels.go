package channels

import (
	"github.com/preludeorg/pneuma/util"
	"sync"
	"time"
)

var agentConfigOpChanOnce sync.Once
var AgentConfigOpsChannel chan AgentConfigOp

var beaconOpChanOnce sync.Once
var BeaconOpsChannel chan BeaconOp

var cacheOpChanOnce sync.Once
var CacheOpsChannel chan CacheOp

var connectionOpChanOnce sync.Once
var ConnectionOpsChannel chan ConnectionOp

var envelopeOnce sync.Once
var Envelopes chan *util.Envelope

func InitChannels() {
	agentConfigOpChanOnce.Do(func() {
		AgentConfigOpsChannel = make(chan AgentConfigOp)
	})

	beaconOpChanOnce.Do(func() {
		BeaconOpsChannel = make(chan BeaconOp)
	})

	cacheOpChanOnce.Do(func() {
		CacheOpsChannel = make(chan CacheOp)
	})

	connectionOpChanOnce.Do(func() {
		ConnectionOpsChannel = make(chan ConnectionOp)
	})

	envelopeOnce.Do(func() {
		Envelopes = make(chan *util.Envelope)
	})
}

func InitChannelGoroutines() {

}

type AgentConfigOp struct {
	Type           string
	Config         *util.AgentConfig
	ResponseStatus chan bool
	ResponseConfig chan *util.AgentConfig
}

// BeaconOp instruct the Beacon state goroutine to perform some operation on the Beacon.
type BeaconOp struct {
	Type           string
	Contact        string
	Links          []util.Instruction
	ResponseStatus chan bool
	ResponseBeacon chan *util.Beacon
	ResponseLinks  chan []util.Instruction
}

type CacheOp struct {
	Type           string
	State          string
	Sent           bool
	Link           string
	ResponseStatus chan bool
	ResponseLinks  chan map[string]util.CachedLink
}

type ConnectionOp struct {
	Type                string
	Name                string
	Connection          *util.Connection
	ConnectionType      string
	ResponseStatus      chan bool
	ResponseConnections chan map[string]*util.Connection
}

func ReadAgentConfig() *util.AgentConfig {
	b := AgentConfigOp{
		Type:           "read",
		ResponseConfig: make(chan *util.AgentConfig),
	}
	AgentConfigOpsChannel <- b
	return <-b.ResponseConfig
}

func WriteAgentConfig(agent *util.AgentConfig) bool {
	b := AgentConfigOp{
		Type:           "write",
		Config:         agent,
		ResponseStatus: make(chan bool),
	}
	AgentConfigOpsChannel <- b
	return <-b.ResponseStatus
}

// ReadBeacon reads and return the Beacon.
func ReadBeacon(contact string) *util.Beacon {
	b := BeaconOp{
		Type:           "read",
		Contact:        contact,
		ResponseBeacon: make(chan *util.Beacon),
	}
	BeaconOpsChannel <- b
	return <-b.ResponseBeacon
}

// RefreshBeacon calls refreshBeacon on the Beacon.
func RefreshBeacon() bool {
	b := BeaconOp{
		Type:           "refresh",
		ResponseStatus: make(chan bool),
	}
	BeaconOpsChannel <- b
	return <-b.ResponseStatus
}

// AppendBeaconLinks appends Links to the Beacon.
func AppendBeaconLinks(links []util.Instruction) bool {
	b := BeaconOp{
		Type:           "append",
		Links:          links,
		ResponseStatus: make(chan bool),
	}
	BeaconOpsChannel <- b
	return <-b.ResponseStatus
}

// TrimBeaconLinks removes 'complete' and 'sent' Links from the Beacon.
func TrimBeaconLinks() bool {
	b := BeaconOp{
		Type:           "trim",
		ResponseStatus: make(chan bool),
	}
	BeaconOpsChannel <- b
	return <-b.ResponseStatus
}

func ReadCacheLinks() map[string]util.CachedLink {
	b := CacheOp{
		Type:          "read",
		ResponseLinks: make(chan map[string]util.CachedLink),
	}
	CacheOpsChannel <- b
	return <-b.ResponseLinks
}

func WriteCacheLink(state string, sent bool, instruction string) bool {
	b := CacheOp{
		Type:           "write",
		Link:           instruction,
		State:          state,
		Sent:           sent,
		ResponseStatus: make(chan bool),
	}
	CacheOpsChannel <- b
	return <-b.ResponseStatus
}

func ReadConnections() map[string]*util.Connection {
	b := ConnectionOp{
		Type:                "read",
		ResponseConnections: make(chan map[string]*util.Connection),
	}
	ConnectionOpsChannel <- b
	return <-b.ResponseConnections
}

func RefreshConnections() bool {
	b := ConnectionOp{
		Type:           "refresh",
		ResponseStatus: make(chan bool),
	}
	ConnectionOpsChannel <- b
	return <-b.ResponseStatus
}

// TrimSentLinks removes links already sent to Operator.
func TrimSentLinks(beacon *util.Beacon) {
	// Read the LinkCache.
	cacheLinks := ReadCacheLinks()
	unsentLinks := make([]util.Instruction, 0)

	// If not sent and complete then keep link in beacon.
	for _, link := range beacon.Links {
		if lcache, ok := cacheLinks[link.ID]; ok {
			if !lcache.Sent {
				if lcache.State == "complete" {
					unsentLinks = append(unsentLinks, link)
				}
			}
		}
	}
	// Update the beacon to only sent completed, unsent Links
	beacon.Links = unsentLinks
}

//TrimLinkCache removes old links from the link cache.
func TrimLinkCache(cache *map[string]util.CachedLink) {
	now := time.Now()
	agent := ReadAgentConfig()
	for k, v := range *cache {
		diff := v.Time.Sub(now).Seconds()
		if diff >= float64(agent.CommandTimeout) {
			delete(*cache, k)
		}
	}
}
