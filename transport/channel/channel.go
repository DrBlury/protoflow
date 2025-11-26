// Package channel provides an in-memory Go channel transport for protoflow.
// This transport is useful for testing and local development.
package channel

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "channel"

// Factory allows overriding the channel creation for testing.
var Factory = func(cfg gochannel.Config, logger watermill.LoggerAdapter) (message.Publisher, message.Subscriber) {
	pubSub := gochannel.NewGoChannel(cfg, logger)
	return pubSub, pubSub
}

func init() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.ChannelCapabilities)
}

// Build creates a new Go channel transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	pub, sub := Factory(gochannel.Config{}, logger)
	return transport.Transport{
		Publisher:  pub,
		Subscriber: sub,
	}, nil
}

// Capabilities returns the capabilities of this transport.
func Capabilities() transport.Capabilities {
	return transport.ChannelCapabilities
}
