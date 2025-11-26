// Package kafka provides a Kafka transport for protoflow.
package kafka

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "kafka"

// PublisherFactory allows overriding the publisher creation for testing.
var PublisherFactory = func(cfg kafka.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
	return kafka.NewPublisher(cfg, logger)
}

// SubscriberFactory allows overriding the subscriber creation for testing.
var SubscriberFactory = func(cfg kafka.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
	return kafka.NewSubscriber(cfg, logger)
}

func init() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.KafkaCapabilities)
}

// Build creates a new Kafka transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	brokers := cfg.GetKafkaBrokers()
	consumerGroup := cfg.GetKafkaConsumerGroup()

	publisher, err := PublisherFactory(
		kafka.PublisherConfig{
			Brokers:   brokers,
			Marshaler: kafka.DefaultMarshaler{},
		},
		logger,
	)
	if err != nil {
		return transport.Transport{}, err
	}

	subscriber, err := SubscriberFactory(
		kafka.SubscriberConfig{
			Brokers:       brokers,
			Unmarshaler:   kafka.DefaultMarshaler{},
			ConsumerGroup: consumerGroup,
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
	return transport.KafkaCapabilities
}
