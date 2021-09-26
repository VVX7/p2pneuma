package util

import "sync"

// Envelope defines a wrapper for passing Beacon structs to handlers.
type Envelope struct {
	ID             int
	Beacon         *Beacon
	Type           string
	ConnectionName string
	Connection     *Connection
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
	var t string
	if conn.Type == "p2p" {
		t = "p2pGeneric"
	} else {
		t = "executorGeneric"
	}
	e := &Envelope{
		Type:       t,
		Beacon:     beacon,
		Connection: conn,
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
