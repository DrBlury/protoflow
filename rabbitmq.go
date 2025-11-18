package protoflow

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
)

var (
	amqpConnectionFactory = func(config amqp.ConnectionConfig, logger watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
		return amqp.NewConnection(config, logger)
	}
	amqpPublisherFactory = func(config amqp.Config, logger watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Publisher, error) {
		return amqp.NewPublisherWithConnection(config, logger, conn)
	}
	amqpSubscriberFactory = func(config amqp.Config, logger watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Subscriber, error) {
		return amqp.NewSubscriberWithConnection(config, logger, conn)
	}
)

func (s *Service) setupAmpq(conf *Config, logger watermill.LoggerAdapter) (*amqp.ConnectionWrapper, amqp.Config) {
	ampqConfig := amqp.NewDurablePubSubConfig(
		conf.RabbitMQURL,
		amqp.GenerateQueueNameTopicNameWithSuffix("-queueSuffix"),
	)
	ampqConn, err := amqpConnectionFactory(amqp.ConnectionConfig{
		AmqpURI:   conf.RabbitMQURL,
		TLSConfig: nil,
		Reconnect: amqp.DefaultReconnectConfig(),
	}, logger)
	if err != nil {
		panic(err)
	}
	return ampqConn, ampqConfig
}

func (s *Service) createRabbitMQPublisher(config amqp.Config, conn *amqp.ConnectionWrapper, logger watermill.LoggerAdapter) {
	publisher, err := amqpPublisherFactory(
		config,
		logger,
		conn,
	)
	if err != nil {
		panic(err)
	}
	s.publisher = publisher
}

func (s *Service) createRabbitMQSubscriber(config amqp.Config, conn *amqp.ConnectionWrapper, logger watermill.LoggerAdapter) {
	subscriber, err := amqpSubscriberFactory(
		config,
		logger,
		conn,
	)
	if err != nil {
		panic(err)
	}
	s.subscriber = subscriber
}
