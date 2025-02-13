// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package p2p provides the peer-to-peer abstractions used
// across different protocols in Bee.
package p2p

import (
	"context"
	"io"
	"time"

	"github.com/holisticode/bee/pkg/bzz"
	"github.com/holisticode/bee/pkg/swarm"
	"github.com/libp2p/go-libp2p-core/network"
	ma "github.com/multiformats/go-multiaddr"
)

// ReachabilityStatus represents the node reachability status.
type ReachabilityStatus network.Reachability

// String implements the fmt.Stringer interface.
func (rs ReachabilityStatus) String() string {
	return network.Reachability(rs).String()
}

const (
	ReachabilityStatusUnknown = ReachabilityStatus(network.ReachabilityUnknown)
	ReachabilityStatusPublic  = ReachabilityStatus(network.ReachabilityPublic)
	ReachabilityStatusPrivate = ReachabilityStatus(network.ReachabilityPrivate)
)

// Service provides methods to handle p2p Peers and Protocols.
type Service interface {
	AddProtocol(ProtocolSpec) error
	// Connect to a peer but do not notify topology about the established connection.
	Connect(ctx context.Context, addr ma.Multiaddr) (address *bzz.Address, err error)
	Disconnecter
	Peers() []Peer
	BlocklistedPeers() ([]Peer, error)
	Addresses() ([]ma.Multiaddr, error)
	SetPickyNotifier(PickyNotifier)
	Halter
}

type Disconnecter interface {
	Disconnect(overlay swarm.Address, reason string) error
	Blocklister
}

type Blocklister interface {
	// Blocklist will disconnect a peer and put it on a blocklist (blocking in & out connections) for provided duration
	// Duration 0 is treated as an infinite duration.
	Blocklist(overlay swarm.Address, duration time.Duration, reason string) error
}

type Halter interface {
	// Halt new incoming connections while shutting down
	Halt()
}

// PickyNotifier can decide whether a peer should be picked
type PickyNotifier interface {
	Picker
	Notifier
	ReachabilityUpdater
	ReachableNotifier
}

type Picker interface {
	Pick(Peer) bool
}

type ReachableNotifier interface {
	Reachable(swarm.Address, ReachabilityStatus)
}

type Reacher interface {
	Connected(swarm.Address, ma.Multiaddr)
	Disconnected(swarm.Address)
}

type ReachabilityUpdater interface {
	UpdateReachability(ReachabilityStatus)
}

type Notifier interface {
	Connected(context.Context, Peer, bool) error
	Disconnected(Peer)
	Announce(ctx context.Context, peer swarm.Address, fullnode bool) error
	AnnounceTo(ctx context.Context, addressee, peer swarm.Address, fullnode bool) error
}

// DebugService extends the Service with method used for debugging.
type DebugService interface {
	Service
	SetWelcomeMessage(val string) error
	GetWelcomeMessage() string
}

// Streamer is able to create a new Stream.
type Streamer interface {
	NewStream(ctx context.Context, address swarm.Address, h Headers, protocol, version, stream string) (Stream, error)
}

type StreamerDisconnecter interface {
	Streamer
	Disconnecter
}

// Pinger interface is used to ping a underlay address which is not yet known to the bee node.
// It uses libp2p's default ping protocol. This is different from the PingPong protocol as this
// is meant to be used before we know a particular underlay and we can consider it useful
type Pinger interface {
	Ping(ctx context.Context, addr ma.Multiaddr) (rtt time.Duration, err error)
}

type StreamerPinger interface {
	Streamer
	Pinger
}

// Stream represent a bidirectional data Stream.
type Stream interface {
	io.ReadWriter
	io.Closer
	ResponseHeaders() Headers
	Headers() Headers
	FullClose() error
	Reset() error
}

// ProtocolSpec defines a collection of Stream specifications with handlers.
type ProtocolSpec struct {
	Name          string
	Version       string
	StreamSpecs   []StreamSpec
	ConnectIn     func(context.Context, Peer) error
	ConnectOut    func(context.Context, Peer) error
	DisconnectIn  func(Peer) error
	DisconnectOut func(Peer) error
}

// StreamSpec defines a Stream handling within the protocol.
type StreamSpec struct {
	Name    string
	Handler HandlerFunc
	Headler HeadlerFunc
}

// Peer holds information about a Peer.
type Peer struct {
	Address         swarm.Address
	FullNode        bool
	EthereumAddress []byte
}

// HandlerFunc handles a received Stream from a Peer.
type HandlerFunc func(context.Context, Peer, Stream) error

// HandlerMiddleware decorates a HandlerFunc by returning a new one.
type HandlerMiddleware func(HandlerFunc) HandlerFunc

// HeadlerFunc is returning response headers based on the received request
// headers.
type HeadlerFunc func(Headers, swarm.Address) Headers

// Headers represents a collection of p2p header key value pairs.
type Headers map[string][]byte

// Common header names.
const (
	HeaderNameTracingSpanContext = "tracing-span-context"
)

// NewSwarmStreamName constructs a libp2p compatible stream name out of
// protocol name and version and stream name.
func NewSwarmStreamName(protocol, version, stream string) string {
	return "/swarm/" + protocol + "/" + version + "/" + stream
}
