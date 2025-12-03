package rabbitmq

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drblury/protoflow/transport"
)

func TestRegister(t *testing.T) {
	transport.DefaultRegistry = transport.NewRegistry()
	Register()

	caps := transport.GetCapabilities(TransportName)
	assert.Equal(t, "rabbitmq", caps.Name)
	assert.True(t, caps.SupportsDelay)
	assert.True(t, caps.SupportsNativeDLQ)
	assert.True(t, caps.SupportsTracing)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	assert.Equal(t, transport.RabbitMQCapabilities, caps)
	assert.Equal(t, "rabbitmq", caps.Name)
}

func TestTransportName(t *testing.T) {
	assert.Equal(t, "rabbitmq", TransportName)
}

func TestBuild(t *testing.T) {
	t.Run("creates transport with mocked factories", func(t *testing.T) {
		originalConnFactory := ConnectionFactory
		originalPubFactory := PublisherFactory
		originalSubFactory := SubscriberFactory
		defer func() {
			ConnectionFactory = originalConnFactory
			PublisherFactory = originalPubFactory
			SubscriberFactory = originalSubFactory
		}()

		mockConn := &amqp.ConnectionWrapper{}
		mockPub := &mockPublisher{}
		mockSub := &mockSubscriber{}

		ConnectionFactory = func(cfg amqp.ConnectionConfig, logger watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
			assert.Equal(t, "amqp://guest:guest@localhost:5672/", cfg.AmqpURI)
			return mockConn, nil
		}
		PublisherFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Publisher, error) {
			return mockPub, nil
		}
		SubscriberFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Subscriber, error) {
			return mockSub, nil
		}

		cfg := &mockConfig{rabbitmqURL: "amqp://guest:guest@localhost:5672/"}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.Equal(t, mockPub, tr.Publisher)
		assert.Equal(t, mockSub, tr.Subscriber)
	})

	t.Run("returns error when connection factory fails", func(t *testing.T) {
		originalConnFactory := ConnectionFactory
		defer func() { ConnectionFactory = originalConnFactory }()

		ConnectionFactory = func(cfg amqp.ConnectionConfig, logger watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
			return nil, errors.New("connection error")
		}

		cfg := &mockConfig{rabbitmqURL: "amqp://localhost:5672/"}
		_, err := Build(context.Background(), cfg, watermill.NopLogger{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection error")
	})

	t.Run("returns error when publisher factory fails", func(t *testing.T) {
		originalConnFactory := ConnectionFactory
		originalPubFactory := PublisherFactory
		defer func() {
			ConnectionFactory = originalConnFactory
			PublisherFactory = originalPubFactory
		}()

		ConnectionFactory = func(cfg amqp.ConnectionConfig, logger watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
			return &amqp.ConnectionWrapper{}, nil
		}
		PublisherFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Publisher, error) {
			return nil, errors.New("publisher error")
		}

		cfg := &mockConfig{rabbitmqURL: "amqp://localhost:5672/"}
		_, err := Build(context.Background(), cfg, watermill.NopLogger{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publisher error")
	})

	t.Run("returns error when subscriber factory fails", func(t *testing.T) {
		originalConnFactory := ConnectionFactory
		originalPubFactory := PublisherFactory
		originalSubFactory := SubscriberFactory
		defer func() {
			ConnectionFactory = originalConnFactory
			PublisherFactory = originalPubFactory
			SubscriberFactory = originalSubFactory
		}()

		ConnectionFactory = func(cfg amqp.ConnectionConfig, logger watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
			return &amqp.ConnectionWrapper{}, nil
		}
		PublisherFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Publisher, error) {
			return &mockPublisher{}, nil
		}
		SubscriberFactory = func(cfg amqp.Config, logger watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Subscriber, error) {
			return nil, errors.New("subscriber error")
		}

		cfg := &mockConfig{rabbitmqURL: "amqp://localhost:5672/"}
		_, err := Build(context.Background(), cfg, watermill.NopLogger{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "subscriber error")
	})
}

type mockConfig struct {
	rabbitmqURL string
}

func (m *mockConfig) GetPubSubSystem() string       { return "rabbitmq" }
func (m *mockConfig) GetKafkaBrokers() []string     { return nil }
func (m *mockConfig) GetKafkaConsumerGroup() string { return "" }
func (m *mockConfig) GetRabbitMQURL() string        { return m.rabbitmqURL }
func (m *mockConfig) GetNATSURL() string            { return "" }
func (m *mockConfig) GetHTTPServerAddress() string  { return "" }
func (m *mockConfig) GetHTTPPublisherURL() string   { return "" }
func (m *mockConfig) GetIOFile() string             { return "" }
func (m *mockConfig) GetSQLiteFile() string         { return "" }
func (m *mockConfig) GetPostgresURL() string        { return "" }
func (m *mockConfig) GetAWSRegion() string          { return "" }
func (m *mockConfig) GetAWSAccountID() string       { return "" }
func (m *mockConfig) GetAWSAccessKeyID() string     { return "" }
func (m *mockConfig) GetAWSSecretAccessKey() string { return "" }
func (m *mockConfig) GetAWSEndpoint() string        { return "" }

type mockPublisher struct{}

func (m *mockPublisher) Publish(topic string, messages ...*message.Message) error { return nil }
func (m *mockPublisher) Close() error                                             { return nil }

type mockSubscriber struct{}

func (m *mockSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return make(chan *message.Message), nil
}
func (m *mockSubscriber) Close() error { return nil }
