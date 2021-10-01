package libp2p

import (
	"context"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	mplex "github.com/libp2p/go-libp2p-mplex"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	tls "github.com/libp2p/go-libp2p-tls"
	yamux "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	"github.com/pkg/errors"
	"github.com/preludeorg/pneuma/util"
	"sync"
)

var defaultAgentPubSubOnce sync.Once
var defaultAgentGossipSubErr error
var DefaultAgentPubSub *AgentPubSub

type MdnsNotifee struct {
	Host    host.Host
	Context context.Context
}

type MessageCache struct {
	Cache []string
}

type AgentPubSub struct {
	Host        host.Host
	Context     context.Context
	Cancel      context.CancelFunc
	Transports  libp2p.Option
	Muxers      libp2p.Option
	Security    libp2p.Option
	ListenAddrs libp2p.Option
	Pubsub      *pubsub.PubSub
	Topic       *pubsub.Topic
	Sub         *pubsub.Subscription
}

func (m *MdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	err := m.Host.Connect(m.Context, pi)
	if err != nil {
		util.DebugLogf("[HandlePeerFound] %s", err)
	}
}

func initJoinPubsubTopic(ps *pubsub.PubSub, psTopic string) (*pubsub.Topic, error) {
	topic, err := ps.Join(psTopic)
	if err != nil {
		util.DebugLogf("[%s] failed to join libp2p PubSub Topic.", err)
		return topic, errors.Wrap(err, "[initJoinPubsubTopic] ")
	}
	util.DebugLogf("[%s] joined libp2p PubSub Topic.", psTopic)
	return topic, nil
}

func initNewGossipSub(ctx context.Context, h host.Host) (*pubsub.PubSub, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		util.DebugLogf("[%s] failed to create libp2p GossipSub.", err)
		return ps, errors.Wrap(err, "[initNewGossipSub] ")
	}
	util.DebugLog("Created libp2p PubSub Topic.")
	return ps, err
}

func initPubsubSubscribe(topic *pubsub.Topic) (*pubsub.Subscription, error) {
	sub, err := topic.Subscribe()
	if err != nil {
		util.DebugLogf("[%s] failed to create libp2p PubSub Subscription.", err)
		return sub, errors.Wrap(err, "[initPubsubSubscribe] ")
	}
	util.DebugLog("Created libp2p PubSub Subscription.")
	return sub, err
}

func initListenAddrs(addrs ...string) libp2p.Option {
	listenAddrs := libp2p.ListenAddrStrings(addrs...)
	util.DebugLogf("[%s] added addrs to libp2p listenAddrs.", addrs)
	return listenAddrs
}

func initTlsSecurity() libp2p.Option {
	security := libp2p.Security(tls.ID, tls.New)
	return security
}

func initMuxers() libp2p.Option {
	muxers := libp2p.ChainOptions(
		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
	)
	return muxers
}

func initTransports() libp2p.Option {
	transports := libp2p.ChainOptions(
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(ws.New),
	)
	return transports
}

func initHost(ctx context.Context, opts ...libp2p.Option) (host.Host, error) {
	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		util.DebugLogf("[%s] failed to create libp2p PubSub Subscription.", err)
		return h, errors.Wrap(err, "[initHost] ")
	}
	return h, nil
}

//func InitAgentP2PObject(opts ...libp2p.Option) (host.Host, error) {
//	// takes args from a beacon to construct a new p2p object
//	ctx, cancel := context.WithCancel(context.Background())
//	//
//	h, err := initHost(ctx, opts...)
//	if err != nil {
//		cancel()
//		return h, errors.Wrap(err,"[InitAgentP2PObject] ")
//	}
//	return h, nil
//}

func InitDefaultAgentGossipSub(pubsubTopic string) (*AgentPubSub, error) {
	defaultAgentPubSubOnce.Do(func() {
		defaultAgentGossipSubErr := defaultAgentGossipSub(pubsubTopic)
		if defaultAgentGossipSubErr != nil {
			errors.Wrap(defaultAgentGossipSubErr, "[InitDefaultAgentGossipSub] ")
		}
	})
	return DefaultAgentPubSub, defaultAgentGossipSubErr
}

func DefaultAgentPeerDiscovery(DefaultAgentPubSub *AgentPubSub) {
	//	mdns, err := discovery.NewMdnsService(DefaultAgentPubSub.Context, DefaultAgentPubSub.Host, time.Second*10, "")
	m := mdns.NewMdnsService(DefaultAgentPubSub.Host, "")
	m.RegisterNotifee(&MdnsNotifee{Host: DefaultAgentPubSub.Host, Context: DefaultAgentPubSub.Context})
}

func defaultAgentGossipSub(pubsubTopic string) error {
	// inits the default GossipSub object
	ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	transports := initTransports()

	muxers := initMuxers()

	security := initTlsSecurity()

	listenAddrs := initListenAddrs()

	h, err := initHost(ctx, transports, muxers, security, listenAddrs)
	if err != nil {
		cancel()
		errors.Wrap(err, "[initPubsubSubscribe] ")
	}

	ps, err := initNewGossipSub(ctx, h)
	if err != nil {
		cancel()
		errors.Wrap(err, "[initPubsubSubscribe] ")
	}

	topic, err := initJoinPubsubTopic(ps, pubsubTopic)
	if err != nil {
		cancel()
		errors.Wrap(err, "[initPubsubSubscribe] ")
	}
	// TODO: handle close in new func
	//defer topic.Close()

	sub, err := initPubsubSubscribe(topic)
	if err != nil {
		cancel()
		errors.Wrap(err, "[initPubsubSubscribe] ")
	}

	DefaultAgentPubSub = &AgentPubSub{
		Host:        h,
		Context:     ctx,
		Cancel:      cancel,
		Transports:  transports,
		Muxers:      muxers,
		Security:    security,
		ListenAddrs: listenAddrs,
		Pubsub:      ps,
		Topic:       topic,
		Sub:         sub,
	}

	return err
}
