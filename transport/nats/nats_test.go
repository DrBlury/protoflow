package nats

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drblury/protoflow/transport"
)

func TestRegister(t *testing.T) {
	transport.DefaultRegistry = transport.NewRegistry()
	Register()

	caps := transport.GetCapabilities(TransportName)
	assert.Equal(t, "nats", caps.Name)
	assert.False(t, caps.SupportsDelay)
	assert.False(t, caps.SupportsNativeDLQ)
	assert.True(t, caps.SupportsTracing)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	assert.Equal(t, transport.NATSCapabilities, caps)
	assert.Equal(t, "nats", caps.Name)
}

func TestTransportName(t *testing.T) {
	assert.Equal(t, "nats", TransportName)
}

func TestBuild(t *testing.T) {
	t.Run("creates transport with mocked factories", func(t *testing.T) {
		originalPubFactory := PublisherFactory
		originalSubFactory := SubscriberFactory
		defer func() {
			PublisherFactory = originalPubFactory
			SubscriberFactory = originalSubFactory
		}()

		mockPub := &mockPublisher{}
		mockSub := &mockSubscriber{}

		PublisherFactory = func(config nats.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return mockPub, nil
		}
		SubscriberFactory = func(config nats.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			return mockSub, nil
		}

		cfg := &mockConfig{natsURL: "nats://localhost:4222"}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.Equal(t, mockPub, tr.Publisher)
		assert.Equal(t, mockSub, tr.Subscriber)
	})

	t.Run("returns error when publisher factory fails", func(t *testing.T) {
		originalPubFactory := PublisherFactory
		defer func() { PublisherFactory = originalPubFactory }()

		PublisherFactory = func(config nats.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return nil, errors.New("publisher error")
		}

		cfg := &mockConfig{natsURL: "nats://localhost:4222"}
		_, err := Build(context.Background(), cfg, watermill.NopLogger{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publisher error")
	})

	t.Run("returns error when subscriber factory fails", func(t *testing.T) {
		originalPubFactory := PublisherFactory
		originalSubFactory := SubscriberFactory
		defer func() {
			PublisherFactory = originalPubFactory
			SubscriberFactory = originalSubFactory
		}()

		PublisherFactory = func(config nats.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return &mockPublisher{}, nil
		}
		SubscriberFactory = func(config nats.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			return nil, errors.New("subscriber error")
		}

		cfg := &mockConfig{natsURL: "nats://localhost:4222"}
		_, err := Build(context.Background(), cfg, watermill.NopLogger{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "subscriber error")
	})
}

type mockConfig struct {
	natsURL string
}

func (m *mockConfig) GetPubSubSystem() string       { return "nats" }
func (m *mockConfig) GetKafkaBrokers() []string     { return nil }
func (m *mockConfig) GetKafkaConsumerGroup() string { return "" }
func (m *mockConfig) GetRabbitMQURL() string        { return "" }
func (m *mockConfig) GetNATSURL() string            { return m.natsURL }
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
