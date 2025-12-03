package transport

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock config for testing
type mockConfig struct {
	pubSubSystem string
}

func (m *mockConfig) GetPubSubSystem() string       { return m.pubSubSystem }
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

// Mock publisher and subscriber
type mockPublisher struct{}

func (m *mockPublisher) Publish(topic string, messages ...*message.Message) error {
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

type mockSubscriber struct{}

func (m *mockSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	ch := make(chan *message.Message)
	close(ch)
	return ch, nil
}

func (m *mockSubscriber) Close() error {
	return nil
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	assert.NotNil(t, reg)
	assert.NotNil(t, reg.builders)
	assert.NotNil(t, reg.capabilities)
	assert.Empty(t, reg.Names())
}

func TestRegistry_Register(t *testing.T) {
	reg := NewRegistry()

	builder := func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
		return Transport{
			Publisher:  &mockPublisher{},
			Subscriber: &mockSubscriber{},
		}, nil
	}

	reg.Register("test-transport", builder)
	assert.True(t, reg.Has("test-transport"))
	assert.Contains(t, reg.Names(), "test-transport")
}

func TestRegistry_RegisterWithCapabilities(t *testing.T) {
	reg := NewRegistry()

	builder := func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
		return Transport{
			Publisher:  &mockPublisher{},
			Subscriber: &mockSubscriber{},
		}, nil
	}

	caps := Capabilities{
		Name:              "test-transport",
		SupportsDelay:     true,
		SupportsNativeDLQ: true,
	}

	reg.RegisterWithCapabilities("test-transport", builder, caps)

	assert.True(t, reg.Has("test-transport"))
	retrievedCaps := reg.GetCapabilities("test-transport")
	assert.Equal(t, "test-transport", retrievedCaps.Name)
	assert.True(t, retrievedCaps.SupportsDelay)
	assert.True(t, retrievedCaps.SupportsNativeDLQ)
}

func TestRegistry_GetCapabilities_Unknown(t *testing.T) {
	reg := NewRegistry()
	caps := reg.GetCapabilities("unknown")
	assert.Equal(t, "unknown", caps.Name)
	assert.False(t, caps.SupportsDelay)
	assert.False(t, caps.SupportsNativeDLQ)
}

func TestRegistry_Build(t *testing.T) {
	reg := NewRegistry()

	builder := func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
		return Transport{
			Publisher:  &mockPublisher{},
			Subscriber: &mockSubscriber{},
		}, nil
	}

	reg.Register("test-transport", builder)

	cfg := &mockConfig{pubSubSystem: "test-transport"}
	ctx := context.Background()

	transport, err := reg.Build(ctx, cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, transport.Publisher)
	assert.NotNil(t, transport.Subscriber)
}

func TestRegistry_Build_NilConfig(t *testing.T) {
	reg := NewRegistry()
	ctx := context.Background()

	_, err := reg.Build(ctx, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestRegistry_Build_UnknownTransport(t *testing.T) {
	reg := NewRegistry()
	cfg := &mockConfig{pubSubSystem: "unknown-transport"}
	ctx := context.Background()

	_, err := reg.Build(ctx, cfg, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown transport")
}

func TestRegistry_Build_BuilderError(t *testing.T) {
	reg := NewRegistry()

	expectedErr := errors.New("builder error")
	builder := func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
		return Transport{}, expectedErr
	}

	reg.Register("failing-transport", builder)
	cfg := &mockConfig{pubSubSystem: "failing-transport"}
	ctx := context.Background()

	_, err := reg.Build(ctx, cfg, nil)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestRegistry_Has(t *testing.T) {
	reg := NewRegistry()

	builder := func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
		return Transport{}, nil
	}

	assert.False(t, reg.Has("test-transport"))

	reg.Register("test-transport", builder)
	assert.True(t, reg.Has("test-transport"))
	assert.False(t, reg.Has("other-transport"))
}

func TestRegistry_Names(t *testing.T) {
	reg := NewRegistry()

	builder := func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
		return Transport{}, nil
	}

	assert.Empty(t, reg.Names())

	reg.Register("transport1", builder)
	reg.Register("transport2", builder)
	reg.Register("transport3", builder)

	names := reg.Names()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "transport1")
	assert.Contains(t, names, "transport2")
	assert.Contains(t, names, "transport3")
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewRegistry()

	builder := func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
		return Transport{
			Publisher:  &mockPublisher{},
			Subscriber: &mockSubscriber{},
		}, nil
	}

	// Register multiple transports concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				reg.Register("transport", builder)
				reg.Has("transport")
				reg.Names()
				reg.GetCapabilities("transport")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.True(t, reg.Has("transport"))
}

func TestGlobalRegistry(t *testing.T) {
	// Test that DefaultRegistry exists
	assert.NotNil(t, DefaultRegistry)

	// Note: We can't test the global Register functions without
	// potentially affecting other tests, since they share the
	// global DefaultRegistry
}

func TestBuildWithDefaultRegistry(t *testing.T) {
	// This tests the package-level Build function
	// We create a new test registry to avoid affecting global state

	cfg := &mockConfig{pubSubSystem: "nonexistent"}
	ctx := context.Background()

	// Should fail with unknown transport
	_, err := Build(ctx, cfg, nil)
	assert.Error(t, err)
}

func TestPackageLevelRegister(t *testing.T) {
	// Test the package-level Register function
	builder := func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
		return Transport{
			Publisher:  &mockPublisher{},
			Subscriber: &mockSubscriber{},
		}, nil
	}

	// Register a transport
	Register("test-pkg-transport", builder)

	// Verify it was registered in the default registry
	assert.True(t, DefaultRegistry.Has("test-pkg-transport"))
}

func TestPackageLevelRegisterWithCapabilities(t *testing.T) {
	// Test the package-level RegisterWithCapabilities function
	builder := func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
		return Transport{
			Publisher:  &mockPublisher{},
			Subscriber: &mockSubscriber{},
		}, nil
	}

	caps := Capabilities{
		Name:          "test-pkg-caps-transport",
		SupportsDelay: true,
	}

	// Register a transport with capabilities
	RegisterWithCapabilities("test-pkg-caps-transport", builder, caps)

	// Verify it was registered
	assert.True(t, DefaultRegistry.Has("test-pkg-caps-transport"))
	retrievedCaps := DefaultRegistry.GetCapabilities("test-pkg-caps-transport")
	assert.Equal(t, "test-pkg-caps-transport", retrievedCaps.Name)
	assert.True(t, retrievedCaps.SupportsDelay)
}
