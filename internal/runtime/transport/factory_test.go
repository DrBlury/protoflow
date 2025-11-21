package transport

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill-aws/sns"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/drblury/protoflow/internal/runtime/config"
)

func TestKafkaTransportReturnsPublisherAndSubscriber(t *testing.T) {
	origPub := KafkaPublisherFactory
	origSub := KafkaSubscriberFactory
	t.Cleanup(func() {
		KafkaPublisherFactory = origPub
		KafkaSubscriberFactory = origSub
	})

	pub := &testPublisher{}
	sub := &testSubscriber{}

	KafkaPublisherFactory = func(cfg kafka.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
		if len(cfg.Brokers) != 1 || cfg.Brokers[0] != "broker" {
			t.Fatalf("unexpected brokers: %#v", cfg.Brokers)
		}
		return pub, nil
	}
	KafkaSubscriberFactory = func(cfg kafka.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
		if cfg.ConsumerGroup != "group" {
			t.Fatalf("unexpected consumer group %s", cfg.ConsumerGroup)
		}
		return sub, nil
	}

	transport, err := kafkaTransport(&config.Config{KafkaBrokers: []string{"broker"}, KafkaConsumerGroup: "group"}, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("unexpected kafka transport error: %v", err)
	}
	if transport.Publisher != pub || transport.Subscriber != sub {
		t.Fatal("expected kafka transport to expose publisher and subscriber")
	}
}

func TestRabbitTransportReturnsPublisherAndSubscriber(t *testing.T) {
	origConn := AmqpConnectionFactory
	origPub := AmqpPublisherFactory
	origSub := AmqpSubscriberFactory
	t.Cleanup(func() {
		AmqpConnectionFactory = origConn
		AmqpPublisherFactory = origPub
		AmqpSubscriberFactory = origSub
	})

	conn := &amqp.ConnectionWrapper{}
	pub := &testPublisher{}
	sub := &testSubscriber{}

	AmqpConnectionFactory = func(cfg amqp.ConnectionConfig, logger watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
		if cfg.AmqpURI == "" {
			t.Fatal("expected AMQP URI to be set")
		}
		return conn, nil
	}
	AmqpPublisherFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, c *amqp.ConnectionWrapper) (message.Publisher, error) {
		if c != conn {
			t.Fatalf("unexpected connection instance")
		}
		return pub, nil
	}
	AmqpSubscriberFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, c *amqp.ConnectionWrapper) (message.Subscriber, error) {
		if c != conn {
			t.Fatalf("unexpected connection instance")
		}
		return sub, nil
	}

	transport, err := rabbitTransport(&config.Config{RabbitMQURL: "amqp://guest"}, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("unexpected rabbit transport error: %v", err)
	}
	if transport.Publisher != pub || transport.Subscriber != sub {
		t.Fatal("expected rabbit transport components to be returned")
	}
}

func TestAwsTransportUsesCustomFactories(t *testing.T) {
	origLoader := AWSDefaultConfigLoader
	origTopic := SNSTopicResolverFactory
	origPub := SNSPublisherFactory
	origSub := SNSSubscriberFactory
	t.Cleanup(func() {
		AWSDefaultConfigLoader = origLoader
		SNSTopicResolverFactory = origTopic
		SNSPublisherFactory = origPub
		SNSSubscriberFactory = origSub
	})

	AWSDefaultConfigLoader = func(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}
	SNSTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		return origTopic(accountID, region)
	}

	pub := &testPublisher{}
	sub := &testSubscriber{}

	SNSPublisherFactory = func(cfg sns.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
		if cfg.TopicResolver == nil {
			t.Fatal("topic resolver must be set")
		}
		return pub, nil
	}
	SNSSubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
		return sub, nil
	}

	conf := &config.Config{PubSubSystem: "aws", AWSAccountID: "000000000000", AWSRegion: "eu-west-1"}
	transport, err := awsTransport(context.Background(), conf, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("unexpected aws transport error: %v", err)
	}
	if transport.Publisher != pub || transport.Subscriber != sub {
		t.Fatal("expected aws transport components to be returned")
	}
}

func TestDefaultFactoryBuild(t *testing.T) {
	factory := DefaultFactory()

	t.Run("unsupported system", func(t *testing.T) {
		if _, err := factory.Build(context.Background(), &config.Config{PubSubSystem: "unknown"}, watermill.NopLogger{}); err == nil {
			t.Fatal("expected error for unknown pubsub system")
		}
	})

	t.Run("kafka", func(t *testing.T) {
		origPub := KafkaPublisherFactory
		origSub := KafkaSubscriberFactory
		pub := &testPublisher{}
		sub := &testSubscriber{}
		KafkaPublisherFactory = func(cfg kafka.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return pub, nil
		}
		KafkaSubscriberFactory = func(cfg kafka.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			return sub, nil
		}
		defer func() {
			KafkaPublisherFactory = origPub
			KafkaSubscriberFactory = origSub
		}()

		transport, err := factory.Build(context.Background(), &config.Config{PubSubSystem: "kafka", KafkaBrokers: []string{"broker"}, KafkaConsumerGroup: "group"}, watermill.NopLogger{})
		if err != nil {
			t.Fatalf("unexpected error building kafka transport: %v", err)
		}
		if transport.Publisher != pub || transport.Subscriber != sub {
			t.Fatal("expected kafka transport to reuse stub publishers")
		}
	})

	t.Run("rabbitmq", func(t *testing.T) {
		origConn := AmqpConnectionFactory
		origPub := AmqpPublisherFactory
		origSub := AmqpSubscriberFactory
		conn := &amqp.ConnectionWrapper{}
		pub := &testPublisher{}
		sub := &testSubscriber{}
		AmqpConnectionFactory = func(cfg amqp.ConnectionConfig, logger watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
			return conn, nil
		}
		AmqpPublisherFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, c *amqp.ConnectionWrapper) (message.Publisher, error) {
			return pub, nil
		}
		AmqpSubscriberFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, c *amqp.ConnectionWrapper) (message.Subscriber, error) {
			return sub, nil
		}
		defer func() {
			AmqpConnectionFactory = origConn
			AmqpPublisherFactory = origPub
			AmqpSubscriberFactory = origSub
		}()

		transport, err := factory.Build(context.Background(), &config.Config{PubSubSystem: "rabbitmq", RabbitMQURL: "amqp://guest"}, watermill.NopLogger{})
		if err != nil {
			t.Fatalf("unexpected error building rabbitmq transport: %v", err)
		}
		if transport.Publisher != pub || transport.Subscriber != sub {
			t.Fatal("expected rabbitmq transport to reuse stub components")
		}
	})

	t.Run("aws", func(t *testing.T) {
		origLoader := AWSDefaultConfigLoader
		origTopic := SNSTopicResolverFactory
		origPub := SNSPublisherFactory
		origSub := SNSSubscriberFactory
		pub := &testPublisher{}
		sub := &testSubscriber{}
		AWSDefaultConfigLoader = func(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{Region: "us-east-1"}, nil
		}
		SNSTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
			return origTopic(accountID, region)
		}
		SNSPublisherFactory = func(cfg sns.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return pub, nil
		}
		SNSSubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			return sub, nil
		}
		defer func() {
			AWSDefaultConfigLoader = origLoader
			SNSTopicResolverFactory = origTopic
			SNSPublisherFactory = origPub
			SNSSubscriberFactory = origSub
		}()

		transport, err := factory.Build(context.Background(), &config.Config{PubSubSystem: "aws", AWSAccountID: "000000000000", AWSRegion: "us-east-1"}, watermill.NopLogger{})
		if err != nil {
			t.Fatalf("unexpected error building aws transport: %v", err)
		}
		if transport.Publisher != pub || transport.Subscriber != sub {
			t.Fatal("expected aws transport to reuse stub components")
		}
	})
}

func TestDefaultFactoryRequiresConfig(t *testing.T) {
	if _, err := (defaultFactory{}).Build(context.Background(), nil, watermill.NopLogger{}); err == nil {
		t.Fatal("expected error when config nil")
	}
}
