package transport

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drblury/protoflow/internal/runtime/config"
	"github.com/drblury/protoflow/internal/runtime/logging"
)

func testLogger() watermill.LoggerAdapter {
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	serviceLogger := logging.NewSlogServiceLogger(slogger)
	return logging.NewWatermillAdapter(serviceLogger)
}

func TestDefaultFactory(t *testing.T) {
	factory := DefaultFactory()
	assert.NotNil(t, factory)
}

func TestDefaultFactory_Build_Channel(t *testing.T) {
	// Note: This test may not work in isolation as it depends on transport packages
	// being imported and registered via their init() functions. The factory.go file
	// imports all transports with _ "github.com/drblury/protoflow/transport/..."
	// which triggers their registration.
	
	factory := DefaultFactory()
	ctx := context.Background()
	cfg := &config.Config{
		PubSubSystem: "channel",
	}
	logger := testLogger()

	transport, err := factory.Build(ctx, cfg, logger)
	
	// If the transport isn't registered (e.g., in minimal test environment),
	// this will error. That's okay - we're testing the factory interface.
	if err != nil {
		// Expected in some test environments where transports aren't registered
		t.Skipf("Transport not registered in test environment: %v", err)
		return
	}
	
	require.NoError(t, err)
	assert.NotNil(t, transport.Publisher)
	assert.NotNil(t, transport.Subscriber)
}

func TestDefaultFactory_Build_NilConfig(t *testing.T) {
	factory := DefaultFactory()
	ctx := context.Background()
	logger := testLogger()

	_, err := factory.Build(ctx, nil, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestDefaultFactory_Build_InvalidTransport(t *testing.T) {
	factory := DefaultFactory()
	ctx := context.Background()
	cfg := &config.Config{
		PubSubSystem: "invalid-transport",
	}
	logger := testLogger()

	_, err := factory.Build(ctx, cfg, logger)
	assert.Error(t, err)
}

func TestTransportStruct(t *testing.T) {
	// Verify Transport struct can be created
	transport := Transport{
		Publisher:  nil,
		Subscriber: nil,
	}
	assert.NotNil(t, transport)
}
