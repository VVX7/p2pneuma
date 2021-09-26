package sockets

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/preludeorg/pneuma/channels"
	"github.com/preludeorg/pneuma/util"
	"io"
	"net"
	"strings"
)

type UDP struct{}

func init() {
	util.CommunicationChannels["udp"] = UDP{}
}

func (contact UDP) Communicate(agent *util.AgentConfig, name string) (*util.Connection, error) {
	// Dial a UDP connection.
	conn, err := net.Dial("udp", agent.Contact["udp"])
	if err != nil {
		util.DebugLogf("[-] %s is either unavailable or a firewall is blocking traffic.", agent.Contact["udp"])
		return nil, err
	}

	// Create the Envelope Send/Recv channels.
	send := make(chan *util.Envelope)
	recv := make(chan *util.Envelope)
	ctrl := make(chan bool)

	// Create the Connection to be returned to the caller.
	connection := &util.Connection{
		Name:   name,
		Type:   "udp",
		Send:   send,
		Recv:   recv,
		Ctrl:   ctrl,
		IsOpen: true,
		Cleanup: func() {
			util.DebugLogf("[udp] cleaning up connection.")
			close(send)
			conn.Close()
			close(recv)
		},
	}

	// Write socket goroutine reads from the connection Send chan and writes to the socket.
	go func() {
		defer connection.Cleanup()
		for {
			envelope := <-send
			udpBufferedSend(conn, *envelope.Beacon)
			channels.UpdateSentLinks(envelope)
		}
	}()

	// Read socket goroutine reads from the socket and writes to the connection Recv chan.
	go func() {
		defer connection.Cleanup()
		for {
			beacon, err := udpRead(conn)
			if err != nil {
				break
			}
			envelope := util.BuildEnvelope(beacon, connection)
			if envelope != nil {
				recv <- envelope
			}
			util.JitterSleep(agent.CommandJitter, "SILENT")
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

// udpRead reads a message from the connection socket, decrypts it, and returns a Beacon.
func udpRead(conn net.Conn) (*util.Beacon, error) {
	scanner := bufio.NewScanner(conn)
	var beacon util.Beacon
	for scanner.Scan() {
		message := strings.TrimSpace(scanner.Text())
		err := json.Unmarshal([]byte(util.Decrypt(message)), &beacon)
		if err != nil {
			util.DebugLog("[-] Unable to decrypt TCP message.")
			continue
		}
		return &beacon, nil
	}
	return nil, nil
}

// udpBufferedSend
func udpBufferedSend(conn net.Conn, beacon util.Beacon) {
	data, _ := json.Marshal(beacon)
	allData := bytes.NewReader(append(util.Encrypt(data), "\n"...))
	sendBuffer := make([]byte, 1024)
	for {
		_, err := allData.Read(sendBuffer)
		if err == io.EOF {
			return
		}
		if _, err = conn.Write(sendBuffer); err != nil {
			util.DebugLog(err)
		}
	}
}
