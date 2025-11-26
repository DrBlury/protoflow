// Package http provides an HTTP transport for protoflow.
package http

import (
	"context"
	nethttp "net/http"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-http/v2/pkg/http"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "http"

// PublisherFactory allows overriding the publisher creation for testing.
var PublisherFactory = func(config http.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
	return http.NewPublisher(config, logger)
}

// SubscriberFactory allows overriding the subscriber creation for testing.
var SubscriberFactory = func(addr string, config http.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
	return http.NewSubscriber(addr, config, logger)
}

func init() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.HTTPCapabilities)
}

// Build creates a new HTTP transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	serverAddr := cfg.GetHTTPServerAddress()
	publisherURL := cfg.GetHTTPPublisherURL()

	publisher, err := PublisherFactory(
		http.PublisherConfig{
			MarshalMessageFunc: func(topic string, msg *message.Message) (*nethttp.Request, error) {
				url := publisherURL + topic
				return http.DefaultMarshalMessageFunc(url, msg)
			},
		},
		logger,
	)
	if err != nil {
		return transport.Transport{}, err
	}

	subscriber, err := SubscriberFactory(
		serverAddr,
		http.SubscriberConfig{
			UnmarshalMessageFunc: http.DefaultUnmarshalMessageFunc,
		},
		logger,
	)
	if err != nil {
		return transport.Transport{}, err
	}

	// Start HTTP server in background if subscriber is the right type
	go func() {
		if s, ok := subscriber.(*http.Subscriber); ok {
			if err := s.StartHTTPServer(); err != nil {
				logger.Error("Failed to start HTTP subscriber server", err, nil)
			}
		}
	}()

	return transport.Transport{
		Publisher:  publisher,
		Subscriber: subscriber,
	}, nil
}

// Capabilities returns the capabilities of this transport.
func Capabilities() transport.Capabilities {
	return transport.HTTPCapabilities
}
