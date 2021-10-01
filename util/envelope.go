package util

import "sync"

// Envelope defines a wrapper for passing Beacon structs to handlers.
type Envelope struct {
	//Agent          string
	Beacon         *Beacon
	Connection     *Connection
	ConnectionName string
	P2PMessage     string
	Type           string
}

// EnvelopeForwarder passes the beacon to the connection send channel.
func EnvelopeForwarder(conn *Connection, envelope *Envelope, wg *sync.WaitGroup) {
	defer wg.Done()
	conn.Send <- envelope
}

// BuildEnvelope creates an envelope.
// Envelope.Type is a placeholder that falls through to the executors/p2p handlers, but we should
// consider sending a beacon type value from Operator to indicate RPC, tunnel, control messages, etc.
func BuildEnvelope(beacon *Beacon, conn *Connection) *Envelope {
	e := &Envelope{
		Type:       "executor",
		Beacon:     beacon,
		Connection: conn,
	}
	return e
}

func BuildP2PEnvelope(p2pType string, p2pMessage string, beacon *Beacon, conn *Connection) *Envelope {
	e := &Envelope{
		//Agent:      agentName,
		Beacon:     beacon,
		Connection: conn,
		P2PMessage: p2pMessage,
		Type:       p2pType,
	}
	return e
}

func BuildSingleLinkEnvelope(envelope *Envelope, link Instruction) *Envelope {
	var e *Envelope
	e = envelope
	l := []Instruction{link}
	e.Beacon.Links = l
	return e
}
