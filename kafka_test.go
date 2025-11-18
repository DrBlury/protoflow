package protoflow

import (
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
)

func TestCreateKafkaPublisherPanicsOnError(t *testing.T) {

	orig := kafkaPublisherFactory
	t.Cleanup(func() { kafkaPublisherFactory = orig })

	kafkaPublisherFactory = func(cfg kafka.PublisherConfig, _ watermill.LoggerAdapter) (message.Publisher, error) {
		return nil, errors.New("pub")
	}

	svc := &Service{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when kafka publisher creation fails")
		}
	}()

	svc.createKafkaPublisher([]string{"broker"}, watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping))
}

func TestCreateKafkaSubscriberPanicsOnError(t *testing.T) {

	orig := kafkaSubscriberFactory
	t.Cleanup(func() { kafkaSubscriberFactory = orig })

	kafkaSubscriberFactory = func(cfg kafka.SubscriberConfig, _ watermill.LoggerAdapter) (message.Subscriber, error) {
		return nil, errors.New("sub")
	}

	svc := &Service{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when kafka subscriber creation fails")
		}
	}()

	svc.createKafkaSubscriber("group", []string{"broker"}, watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping))
}
