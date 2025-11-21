package transport

import (
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/drblury/protoflow/internal/runtime/config"
)

func TestSetupAmqpReturnsError(t *testing.T) {

	origConn := AmqpConnectionFactory
	t.Cleanup(func() { AmqpConnectionFactory = origConn })

	AmqpConnectionFactory = func(config amqp.ConnectionConfig, _ watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
		return nil, errors.New("conn")
	}

	_, _, err := setupAmqp(&config.Config{RabbitMQURL: "amqp://guest"}, watermill.NopLogger{})
	if err == nil {
		t.Fatal("expected error when connection fails")
	}
}

func TestNewRabbitMQPublisherReturnsError(t *testing.T) {

	origPub := AmqpPublisherFactory
	t.Cleanup(func() { AmqpPublisherFactory = origPub })

	AmqpPublisherFactory = func(cfg amqp.Config, _ watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Publisher, error) {
		return nil, errors.New("publisher")
	}

	if _, err := newRabbitMQPublisher(amqp.Config{}, &amqp.ConnectionWrapper{}, watermill.NopLogger{}); err == nil {
		t.Fatal("expected error when publisher creation fails")
	}
}

func TestNewRabbitMQSubscriberReturnsError(t *testing.T) {

	origSub := AmqpSubscriberFactory
	t.Cleanup(func() { AmqpSubscriberFactory = origSub })

	AmqpSubscriberFactory = func(cfg amqp.Config, _ watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Subscriber, error) {
		return nil, errors.New("subscriber")
	}

	if _, err := newRabbitMQSubscriber(amqp.Config{}, &amqp.ConnectionWrapper{}, watermill.NopLogger{}); err == nil {
		t.Fatal("expected error when subscriber creation fails")
	}
}
