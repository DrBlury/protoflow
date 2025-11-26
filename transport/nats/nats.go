// Package nats provides a NATS Core transport for protoflow.
package nats

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "nats"

// PublisherFactory allows overriding the publisher creation for testing.
var PublisherFactory = func(cfg nats.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
	return nats.NewPublisher(cfg, logger)
}

// SubscriberFactory allows overriding the subscriber creation for testing.
var SubscriberFactory = func(cfg nats.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
	return nats.NewSubscriber(cfg, logger)
}

// Register registers the NATS transport with the default registry.
// This should be called from an init() function in an importing package,
// or explicitly before using the transport.
func Register() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.NATSCapabilities)
}

// Build creates a new NATS transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	url := cfg.GetNATSURL()
	marshaler := &nats.NATSMarshaler{}

	publisher, err := PublisherFactory(
		nats.PublisherConfig{
			URL:       url,
			Marshaler: marshaler,
		},
		logger,
	)
	if err != nil {
		return transport.Transport{}, err
	}

	subscriber, err := SubscriberFactory(
		nats.SubscriberConfig{
			URL:         url,
			Unmarshaler: marshaler,
		},
		logger,
	)
	if err != nil {
		return transport.Transport{}, err
	}

	return transport.Transport{
		Publisher:  publisher,
		Subscriber: subscriber,
	}, nil
}

// Capabilities returns the capabilities of this transport.
func Capabilities() transport.Capabilities {
	return transport.NATSCapabilities
}
