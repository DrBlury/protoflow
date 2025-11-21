package transport

import (
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
)

func TestNewKafkaPublisherReturnsError(t *testing.T) {

	orig := KafkaPublisherFactory
	t.Cleanup(func() { KafkaPublisherFactory = orig })

	KafkaPublisherFactory = func(cfg kafka.PublisherConfig, _ watermill.LoggerAdapter) (message.Publisher, error) {
		return nil, errors.New("pub")
	}

	if _, err := newKafkaPublisher([]string{"broker"}, watermill.NopLogger{}); err == nil {
		t.Fatal("expected error when kafka publisher creation fails")
	}
}

func TestNewKafkaSubscriberReturnsError(t *testing.T) {

	orig := KafkaSubscriberFactory
	t.Cleanup(func() { KafkaSubscriberFactory = orig })

	KafkaSubscriberFactory = func(cfg kafka.SubscriberConfig, _ watermill.LoggerAdapter) (message.Subscriber, error) {
		return nil, errors.New("sub")
	}

	if _, err := newKafkaSubscriber("group", []string{"broker"}, watermill.NopLogger{}); err == nil {
		t.Fatal("expected error when kafka subscriber creation fails")
	}
}
