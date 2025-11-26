// Package rabbitmq provides a RabbitMQ/AMQP transport for protoflow.
package rabbitmq

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "rabbitmq"

// ConnectionFactory allows overriding the connection creation for testing.
var ConnectionFactory = func(cfg amqp.ConnectionConfig, logger watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
	return amqp.NewConnection(cfg, logger)
}

// PublisherFactory allows overriding the publisher creation for testing.
var PublisherFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Publisher, error) {
	return amqp.NewPublisherWithConnection(cfg, logger, conn)
}

// SubscriberFactory allows overriding the subscriber creation for testing.
var SubscriberFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Subscriber, error) {
	return amqp.NewSubscriberWithConnection(cfg, logger, conn)
}

// Register registers the RabbitMQ transport with the default registry.
// This should be called from an init() function in an importing package,
// or explicitly before using the transport.
func Register() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.RabbitMQCapabilities)
}

// Build creates a new RabbitMQ transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	url := cfg.GetRabbitMQURL()

	amqpConfig := amqp.NewDurablePubSubConfig(
		url,
		amqp.GenerateQueueNameTopicName,
	)

	conn, err := ConnectionFactory(amqp.ConnectionConfig{
		AmqpURI:   url,
		TLSConfig: nil,
		Reconnect: amqp.DefaultReconnectConfig(),
	}, logger)
	if err != nil {
		return transport.Transport{}, err
	}

	publisher, err := PublisherFactory(amqpConfig, logger, conn)
	if err != nil {
		return transport.Transport{}, err
	}

	subscriber, err := SubscriberFactory(amqpConfig, logger, conn)
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
	return transport.RabbitMQCapabilities
}
