package protoflow

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
)

var (
	kafkaPublisherFactory = func(config kafka.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
		return kafka.NewPublisher(config, logger)
	}
	kafkaSubscriberFactory = func(config kafka.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
		return kafka.NewSubscriber(config, logger)
	}
)

func (s *Service) createKafkaPublisher(brokers []string, logger watermill.LoggerAdapter) {
	kafkaPublisher, err := kafkaPublisherFactory(
		kafka.PublisherConfig{
			Brokers:   brokers,
			Marshaler: kafka.DefaultMarshaler{},
		},
		logger,
	)
	if err != nil {
		panic(err)
	}

	s.publisher = kafkaPublisher
}

func (s *Service) createKafkaSubscriber(consumerGroup string, brokers []string, logger watermill.LoggerAdapter) {
	kafkaSubscriber, err := kafkaSubscriberFactory(
		kafka.SubscriberConfig{
			Brokers:       brokers,
			Unmarshaler:   kafka.DefaultMarshaler{},
			ConsumerGroup: consumerGroup,
		},
		logger,
	)
	if err != nil {
		panic(err)
	}

	s.subscriber = kafkaSubscriber
}
