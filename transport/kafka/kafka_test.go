package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drblury/protoflow/transport"
)

func TestRegister(t *testing.T) {
	transport.DefaultRegistry = transport.NewRegistry()
	Register()

	caps := transport.GetCapabilities(TransportName)
	assert.Equal(t, "kafka", caps.Name)
	assert.False(t, caps.SupportsDelay)
	assert.False(t, caps.SupportsNativeDLQ)
	assert.True(t, caps.SupportsTracing)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	assert.Equal(t, transport.KafkaCapabilities, caps)
	assert.Equal(t, "kafka", caps.Name)
}

func TestTransportName(t *testing.T) {
	assert.Equal(t, "kafka", TransportName)
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

		PublisherFactory = func(cfg kafka.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			assert.Equal(t, []string{"localhost:9092"}, cfg.Brokers)
			return mockPub, nil
		}
		SubscriberFactory = func(cfg kafka.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			assert.Equal(t, []string{"localhost:9092"}, cfg.Brokers)
			assert.Equal(t, "test-group", cfg.ConsumerGroup)
			return mockSub, nil
		}

		cfg := &mockConfig{
			brokers:       []string{"localhost:9092"},
			consumerGroup: "test-group",
		}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.Equal(t, mockPub, tr.Publisher)
		assert.Equal(t, mockSub, tr.Subscriber)
	})

	t.Run("returns error when publisher factory fails", func(t *testing.T) {
		originalPubFactory := PublisherFactory
		defer func() { PublisherFactory = originalPubFactory }()

		PublisherFactory = func(cfg kafka.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return nil, errors.New("publisher error")
		}

		cfg := &mockConfig{brokers: []string{"localhost:9092"}}
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

		PublisherFactory = func(cfg kafka.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return &mockPublisher{}, nil
		}
		SubscriberFactory = func(cfg kafka.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			return nil, errors.New("subscriber error")
		}

		cfg := &mockConfig{brokers: []string{"localhost:9092"}}
		_, err := Build(context.Background(), cfg, watermill.NopLogger{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "subscriber error")
	})
}

type mockConfig struct {
	brokers       []string
	consumerGroup string
}

func (m *mockConfig) GetPubSubSystem() string       { return "kafka" }
func (m *mockConfig) GetKafkaBrokers() []string     { return m.brokers }
func (m *mockConfig) GetKafkaConsumerGroup() string { return m.consumerGroup }
func (m *mockConfig) GetRabbitMQURL() string        { return "" }
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
