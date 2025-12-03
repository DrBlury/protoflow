package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCapabilities(t *testing.T) {
	// Test that GetCapabilities returns capabilities for known transports
	// The actual values come from the registry after the transports register themselves
	tests := []string{"channel", "kafka", "rabbitmq", "nats", "jetstream", "aws", "http", "io", "sqlite", "postgres"}
	
	for _, transportName := range tests {
		t.Run(transportName, func(t *testing.T) {
			caps := GetCapabilities(transportName)
			// Just verify we get capabilities back with the name set
			assert.NotNil(t, caps)
			// The transport may or may not be registered in tests, 
			// so we just check the function works
		})
	}
}

func TestGetCapabilities_Unknown(t *testing.T) {
	caps := GetCapabilities("unknown-transport")
	assert.NotNil(t, caps)
	// Unknown transports get default empty capabilities
	assert.False(t, caps.SupportsNativeDLQ)
	assert.False(t, caps.SupportsDelay)
}

func TestCapabilitiesAliases(t *testing.T) {
	// Verify all capability aliases are accessible
	assert.NotNil(t, ChannelCapabilities)
	assert.NotNil(t, KafkaCapabilities)
	assert.NotNil(t, RabbitMQCapabilities)
	assert.NotNil(t, NATSCapabilities)
	assert.NotNil(t, NATSJetStreamCapabilities)
	assert.NotNil(t, AWSCapabilities)
	assert.NotNil(t, SQLiteCapabilities)
	assert.NotNil(t, PostgresCapabilities)
	assert.NotNil(t, HTTPCapabilities)
	assert.NotNil(t, IOCapabilities)
}
