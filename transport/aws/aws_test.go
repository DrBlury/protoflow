package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-aws/sns"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drblury/protoflow/transport"
)

func TestRegister(t *testing.T) {
	transport.DefaultRegistry = transport.NewRegistry()
	Register()

	caps := transport.GetCapabilities(TransportName)
	assert.Equal(t, "aws", caps.Name)
	assert.True(t, caps.SupportsDelay)
	assert.True(t, caps.SupportsNativeDLQ)
	assert.True(t, caps.SupportsTracing)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	assert.Equal(t, transport.AWSCapabilities, caps)
	assert.Equal(t, "aws", caps.Name)
}

func TestTransportName(t *testing.T) {
	assert.Equal(t, "aws", TransportName)
}

func TestBuild(t *testing.T) {
	t.Run("creates transport with mocked factories", func(t *testing.T) {
		originalConfigLoader := DefaultConfigLoader
		originalTopicResolver := TopicResolverFactory
		originalPubFactory := PublisherFactory
		originalSubFactory := SubscriberFactory
		defer func() {
			DefaultConfigLoader = originalConfigLoader
			TopicResolverFactory = originalTopicResolver
			PublisherFactory = originalPubFactory
			SubscriberFactory = originalSubFactory
		}()

		mockPub := &mockPublisher{}
		mockSub := &mockSubscriber{}

		DefaultConfigLoader = func(ctx context.Context, opts ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{Region: "us-east-1"}, nil
		}
		TopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
			return &sns.GenerateArnTopicResolver{}, nil
		}
		PublisherFactory = func(cfg sns.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return mockPub, nil
		}
		SubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			return mockSub, nil
		}

		cfg := &mockConfig{
			awsRegion:    "us-east-1",
			awsAccountID: "123456789012",
		}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.Equal(t, mockPub, tr.Publisher)
		assert.Equal(t, mockSub, tr.Subscriber)
	})

	t.Run("returns error when config loader fails", func(t *testing.T) {
		originalConfigLoader := DefaultConfigLoader
		defer func() { DefaultConfigLoader = originalConfigLoader }()

		DefaultConfigLoader = func(ctx context.Context, opts ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, errors.New("config error")
		}

		cfg := &mockConfig{awsRegion: "us-east-1"}
		_, err := Build(context.Background(), cfg, watermill.NopLogger{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config error")
	})

	t.Run("returns error when publisher factory fails", func(t *testing.T) {
		originalConfigLoader := DefaultConfigLoader
		originalTopicResolver := TopicResolverFactory
		originalPubFactory := PublisherFactory
		defer func() {
			DefaultConfigLoader = originalConfigLoader
			TopicResolverFactory = originalTopicResolver
			PublisherFactory = originalPubFactory
		}()

		DefaultConfigLoader = func(ctx context.Context, opts ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{Region: "us-east-1"}, nil
		}
		TopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
			return &sns.GenerateArnTopicResolver{}, nil
		}
		PublisherFactory = func(cfg sns.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return nil, errors.New("publisher error")
		}

		cfg := &mockConfig{awsRegion: "us-east-1", awsAccountID: "123456789012"}
		_, err := Build(context.Background(), cfg, watermill.NopLogger{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publisher error")
	})

	t.Run("returns error when subscriber factory fails", func(t *testing.T) {
		originalConfigLoader := DefaultConfigLoader
		originalTopicResolver := TopicResolverFactory
		originalPubFactory := PublisherFactory
		originalSubFactory := SubscriberFactory
		defer func() {
			DefaultConfigLoader = originalConfigLoader
			TopicResolverFactory = originalTopicResolver
			PublisherFactory = originalPubFactory
			SubscriberFactory = originalSubFactory
		}()

		DefaultConfigLoader = func(ctx context.Context, opts ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{Region: "us-east-1"}, nil
		}
		TopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
			return &sns.GenerateArnTopicResolver{}, nil
		}
		PublisherFactory = func(cfg sns.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return &mockPublisher{}, nil
		}
		SubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			return nil, errors.New("subscriber error")
		}

		cfg := &mockConfig{awsRegion: "us-east-1", awsAccountID: "123456789012"}
		_, err := Build(context.Background(), cfg, watermill.NopLogger{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "subscriber error")
	})
}

func TestResolveAccountAndRegion(t *testing.T) {
	t.Run("uses config values", func(t *testing.T) {
		cfg := &mockConfig{
			awsAccountID: "123456789012",
			awsRegion:    "us-west-2",
		}
		accountID, region := resolveAccountAndRegion(cfg, watermill.NopLogger{}, "us-east-1")
		assert.Equal(t, "123456789012", accountID)
		assert.Equal(t, "us-west-2", region)
	})

	t.Run("uses fallback region when config region empty", func(t *testing.T) {
		cfg := &mockConfig{awsAccountID: "123456789012"}
		accountID, region := resolveAccountAndRegion(cfg, watermill.NopLogger{}, "us-east-1")
		assert.Equal(t, "123456789012", accountID)
		assert.Equal(t, "us-east-1", region)
	})

	t.Run("uses localstack default when endpoint set and account empty", func(t *testing.T) {
		cfg := &mockConfig{awsEndpoint: "http://localhost:4566"}
		accountID, _ := resolveAccountAndRegion(cfg, watermill.NopLogger{}, "us-east-1")
		assert.Equal(t, localstackAccountID, accountID)
	})

	t.Run("returns empty values for nil config", func(t *testing.T) {
		accountID, region := resolveAccountAndRegion(nil, watermill.NopLogger{}, "us-east-1")
		assert.Equal(t, "", accountID)
		assert.Equal(t, "us-east-1", region)
	})
}

func TestAwsEndpointURL(t *testing.T) {
	t.Run("returns nil for nil config", func(t *testing.T) {
		url, err := awsEndpointURL(nil)
		assert.NoError(t, err)
		assert.Nil(t, url)
	})

	t.Run("returns nil for empty endpoint", func(t *testing.T) {
		cfg := &mockConfig{}
		url, err := awsEndpointURL(cfg)
		assert.NoError(t, err)
		assert.Nil(t, url)
	})

	t.Run("parses valid endpoint", func(t *testing.T) {
		cfg := &mockConfig{awsEndpoint: "http://localhost:4566"}
		url, err := awsEndpointURL(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, url)
		assert.Equal(t, "localhost:4566", url.Host)
	})
}

type mockConfig struct {
	awsRegion          string
	awsAccountID       string
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsEndpoint        string
}

func (m *mockConfig) GetPubSubSystem() string       { return "aws" }
func (m *mockConfig) GetKafkaBrokers() []string     { return nil }
func (m *mockConfig) GetKafkaConsumerGroup() string { return "" }
func (m *mockConfig) GetRabbitMQURL() string        { return "" }
func (m *mockConfig) GetNATSURL() string            { return "" }
func (m *mockConfig) GetHTTPServerAddress() string  { return "" }
func (m *mockConfig) GetHTTPPublisherURL() string   { return "" }
func (m *mockConfig) GetIOFile() string             { return "" }
func (m *mockConfig) GetSQLiteFile() string         { return "" }
func (m *mockConfig) GetPostgresURL() string        { return "" }
func (m *mockConfig) GetAWSRegion() string          { return m.awsRegion }
func (m *mockConfig) GetAWSAccountID() string       { return m.awsAccountID }
func (m *mockConfig) GetAWSAccessKeyID() string     { return m.awsAccessKeyID }
func (m *mockConfig) GetAWSSecretAccessKey() string { return m.awsSecretAccessKey }
func (m *mockConfig) GetAWSEndpoint() string        { return m.awsEndpoint }

type mockPublisher struct{}

func (m *mockPublisher) Publish(topic string, messages ...*message.Message) error { return nil }
func (m *mockPublisher) Close() error                                             { return nil }

type mockSubscriber struct{}

func (m *mockSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return make(chan *message.Message), nil
}
func (m *mockSubscriber) Close() error { return nil }
