package channel

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drblury/protoflow/transport"
)

func TestRegister(t *testing.T) {
	transport.DefaultRegistry = transport.NewRegistry()
	Register()

	caps := transport.GetCapabilities(TransportName)
	assert.Equal(t, "channel", caps.Name)
	assert.True(t, caps.SupportsOrdering)
	assert.True(t, caps.SupportsAck)
	assert.True(t, caps.SupportsNack)
	assert.False(t, caps.SupportsDelay)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	assert.Equal(t, transport.ChannelCapabilities, caps)
	assert.Equal(t, "channel", caps.Name)
}

func TestBuild(t *testing.T) {
	t.Run("creates transport with default factory", func(t *testing.T) {
		cfg := &mockConfig{}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.NotNil(t, tr.Publisher)
		assert.NotNil(t, tr.Subscriber)
	})

	t.Run("uses custom factory", func(t *testing.T) {
		originalFactory := Factory
		defer func() { Factory = originalFactory }()

		mockPub := &mockPublisher{}
		mockSub := &mockSubscriber{}
		Factory = func(cfg gochannel.Config, logger watermill.LoggerAdapter) (message.Publisher, message.Subscriber) {
			return mockPub, mockSub
		}

		cfg := &mockConfig{}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.Equal(t, mockPub, tr.Publisher)
		assert.Equal(t, mockSub, tr.Subscriber)
	})
}

func TestTransportName(t *testing.T) {
	assert.Equal(t, "channel", TransportName)
}

type mockConfig struct{}

func (m *mockConfig) GetPubSubSystem() string       { return "channel" }
func (m *mockConfig) GetKafkaBrokers() []string     { return nil }
func (m *mockConfig) GetKafkaConsumerGroup() string { return "" }
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
