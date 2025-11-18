package protoflow

import (
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
)

func TestSetupAmpqPanicsOnConnectionError(t *testing.T) {

	origConn := amqpConnectionFactory
	t.Cleanup(func() { amqpConnectionFactory = origConn })

	amqpConnectionFactory = func(config amqp.ConnectionConfig, _ watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
		return nil, errors.New("conn")
	}

	svc := &Service{Conf: &Config{RabbitMQURL: "amqp://guest"}}
	logger := watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when connection fails")
		}
	}()

	svc.setupAmpq(svc.Conf, logger)
}

func TestCreateRabbitMQPublisherPanicsOnError(t *testing.T) {

	origPub := amqpPublisherFactory
	t.Cleanup(func() { amqpPublisherFactory = origPub })

	amqpPublisherFactory = func(cfg amqp.Config, _ watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Publisher, error) {
		return nil, errors.New("publisher")
	}

	svc := &Service{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when publisher creation fails")
		}
	}()

	svc.createRabbitMQPublisher(amqp.Config{}, &amqp.ConnectionWrapper{}, watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping))
}

func TestCreateRabbitMQSubscriberPanicsOnError(t *testing.T) {

	origSub := amqpSubscriberFactory
	t.Cleanup(func() { amqpSubscriberFactory = origSub })

	amqpSubscriberFactory = func(cfg amqp.Config, _ watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Subscriber, error) {
		return nil, errors.New("subscriber")
	}

	svc := &Service{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when subscriber creation fails")
		}
	}()

	svc.createRabbitMQSubscriber(amqp.Config{}, &amqp.ConnectionWrapper{}, watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping))
}
