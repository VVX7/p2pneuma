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

type TCP struct{}

func init() {
	util.CommunicationChannels["tcp"] = TCP{}
}

func (contact TCP) Communicate(agent *util.AgentConfig, name string) (*util.Connection, error) {
	// Split the TCP target into host and port.
	//host, port, err := net.SplitHostPort(agent.Contact["tcp"])
	//if err != nil {
	//	util.DebugLogf("[-] %s Unable to parse the ip:port of the TCP listening post.", agent.TCP)
	//}

	// Dial a TCP connection.
	conn, err := net.Dial("tcp", agent.Contact["tcp"])
	if err != nil {
		util.DebugLogf("[-] %s is either unavailable or a firewall is blocking traffic.", agent.Contact["tcp"])
		return nil, err
	}

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
			util.DebugLogf("[tcp] cleaning up connection.")
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
			bufferedSend(conn, *envelope.Beacon)
			// get the cache
			c := channels.ReadCacheLinks()
			for _, link := range envelope.Beacon.Links {
				if l, b := c[link.ID]; b {
					if l.State == "complete" {
						if l.Sent == false {
							_ = channels.WriteCacheLink("complete", true, link.ID)
						}
					}
				}
			}
		}
	}()

	// Read socket goroutine reads from the socket and writes to the connection Recv chan.
	go func() {
		defer connection.Cleanup()
		for {
			beacon, err := tcpRead(conn)
			if err != nil {
				break
			}

			if beacon != nil {
				envelope := util.BuildEnvelope(beacon, connection)
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

// tcpRead reads a message from the connection socket, decrypts it, and returns a Beacon.
func tcpRead(conn net.Conn) (*util.Beacon, error) {
	scanner := bufio.NewScanner(conn)
	var beacon util.Beacon
	for scanner.Scan() {
		message := strings.TrimSpace(scanner.Text())
		err := json.Unmarshal([]byte(util.Decrypt(message)), &beacon)
		if err != nil {
			util.DebugLog("[TCP] Unable to decrypt message.")
			continue
		}
		return &beacon, nil
	}
	return nil, nil
}

// bufferedSend sends a beacon to the connection socket.
func bufferedSend(conn net.Conn, beacon util.Beacon) {
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
