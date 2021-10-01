package sockets

import (
	"context"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/preludeorg/pneuma/channels"
	"github.com/preludeorg/pneuma/sockets/libp2p"
	"github.com/preludeorg/pneuma/sockets/libp2p/protos"
	"github.com/preludeorg/pneuma/util"
)

type P2P struct{}

func init() {
	util.CommunicationChannels["p2p"] = P2P{}
}

func (contact P2P) Communicate(agent *util.AgentConfig, name string) (*util.Connection, error) {
	// Init the libp2p GossipSub context.
	p2p, err := libp2p.InitDefaultAgentGossipSub(util.DefaultPubSubTopic)
	if err != nil {
		util.DebugLogf("[P2P Communicate] %s", err)
	}

	//libp2p.DefaultAgentPeerDiscovery(p2p)
	m := mdns.NewMdnsService(p2p.Host, "")
	m.RegisterNotifee(&libp2p.MdnsNotifee{Host: p2p.Host, Context: p2p.Context})

	// Create the Envelope Send/Recv channels.
	send := make(chan *util.Envelope)
	recv := make(chan *util.Envelope)
	ctrl := make(chan bool)

	// Create the Connection to be returned to the caller.
	connection := &util.Connection{
		Name:   name,
		Type:   "p2p",
		Send:   send,
		Recv:   recv,
		Ctrl:   ctrl,
		IsOpen: true,
		Cleanup: func() {
			util.DebugLogf("[p2p] cleaning up connection.")
			close(send)
			//conn.Close()
			close(recv)
		},
	}

	// Send a beacon to the GossipSub topic.
	go func() {
		defer connection.Cleanup()
		for {
			envelope := <-send
			// TODO: handle route or other types of messages
			// Handles a beacon p2p message
			data, _ := json.Marshal(*envelope.Beacon)
			sendBeacon(agent, p2p.Context, p2p.Topic, string(data))
			channels.UpdateSentLinks(envelope)
		}
	}()

	// Read a message from the GossipSub topic.
	go func() {
		defer connection.Cleanup()
		agentConfig := channels.ReadAgentConfig()

		for {
			msg, err := p2p.Sub.Next(p2p.Context)
			if err != nil {
				util.DebugLogf("[P2P Communicate] ", err)
				continue
			}

			// Unmarshall the message into a proto3 EventWrapper struct.
			req := &protos.EventWrapper{}
			err = proto.Unmarshal(msg.Data, req)
			if err != nil {
				util.DebugLogf("[P2P Communicate] ", err)
				continue
			}

			var beacon util.Beacon

			// Handle the message by its message type.
			switch req.Msg.(type) {
			case *protos.EventWrapper_B:
				// Handles beacon messages.
				// Agent name indicates the agent that should execute the message.
				// Drop the message if it's not for this agent.
				if agentConfig.Name == req.GetB().Agent {
					body := []byte(req.GetB().GetMessage())
					if err := json.Unmarshal(body, &beacon); err != nil || len(beacon.Links) == 0 {
						//
						envelope := util.BuildP2PEnvelope("P2PExecutor", "", &beacon, connection)
						recv <- envelope
						util.JitterSleep(agent.CommandJitter, "")
					}
				}
			case *protos.EventWrapper_R:
				// Handles route messages.
				msg := req.GetR().GetRoute()
				envelope := util.BuildP2PEnvelope("P2PRoute", msg, &beacon, connection)
				recv <- envelope
				util.JitterSleep(agent.CommandJitter, "")
			}
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

func sendBeacon(agent *util.AgentConfig, ctx context.Context, topic *pubsub.Topic, msg string) {
	// Sends a beacon as proto3 to GossipSub topic.
	m := &protos.SendBeacon{
		Agent:   agent.Name,
		Message: msg,
	}

	req := &protos.EventWrapper{
		Msg: &protos.EventWrapper_B{B: m},
	}

	msgBytes, err := proto.Marshal(req)
	if err != nil {
		return
	}
	err = topic.Publish(ctx, msgBytes)
}
