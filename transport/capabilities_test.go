package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilities_RequiresDelayEmulation(t *testing.T) {
	tests := []struct {
		name          string
		caps          Capabilities
		wantEmulation bool
	}{
		{
			name:          "supports delay",
			caps:          Capabilities{SupportsDelay: true},
			wantEmulation: false,
		},
		{
			name:          "no delay support",
			caps:          Capabilities{SupportsDelay: false},
			wantEmulation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantEmulation, tt.caps.RequiresDelayEmulation())
		})
	}
}

func TestCapabilities_RequiresDLQEmulation(t *testing.T) {
	tests := []struct {
		name          string
		caps          Capabilities
		wantEmulation bool
	}{
		{
			name:          "supports native DLQ",
			caps:          Capabilities{SupportsNativeDLQ: true},
			wantEmulation: false,
		},
		{
			name:          "no native DLQ support",
			caps:          Capabilities{SupportsNativeDLQ: false},
			wantEmulation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantEmulation, tt.caps.RequiresDLQEmulation())
		})
	}
}

func TestCapabilities_SupportsReliableDelivery(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		wantBool bool
	}{
		{
			name: "supports ack and nack",
			caps: Capabilities{
				SupportsAck:  true,
				SupportsNack: true,
			},
			wantBool: true,
		},
		{
			name: "supports ack only",
			caps: Capabilities{
				SupportsAck:  true,
				SupportsNack: false,
			},
			wantBool: false,
		},
		{
			name: "supports nack only",
			caps: Capabilities{
				SupportsAck:  false,
				SupportsNack: true,
			},
			wantBool: false,
		},
		{
			name: "supports neither",
			caps: Capabilities{
				SupportsAck:  false,
				SupportsNack: false,
			},
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantBool, tt.caps.SupportsReliableDelivery())
		})
	}
}

func TestPredefinedCapabilities(t *testing.T) {
	// Test that all predefined capability sets are properly configured
	t.Run("ChannelCapabilities", func(t *testing.T) {
		assert.Equal(t, "channel", ChannelCapabilities.Name)
		assert.True(t, ChannelCapabilities.SupportsOrdering)
		assert.True(t, ChannelCapabilities.SupportsAck)
		assert.True(t, ChannelCapabilities.SupportsNack)
		assert.False(t, ChannelCapabilities.SupportsDelay)
		assert.False(t, ChannelCapabilities.SupportsNativeDLQ)
	})

	t.Run("KafkaCapabilities", func(t *testing.T) {
		assert.Equal(t, "kafka", KafkaCapabilities.Name)
		assert.True(t, KafkaCapabilities.SupportsOrdering)
		assert.True(t, KafkaCapabilities.SupportsPartitioning)
		assert.True(t, KafkaCapabilities.SupportsBatching)
		assert.False(t, KafkaCapabilities.SupportsDelay)
		assert.Greater(t, KafkaCapabilities.MaxMessageSize, int64(0))
	})

	t.Run("RabbitMQCapabilities", func(t *testing.T) {
		assert.Equal(t, "rabbitmq", RabbitMQCapabilities.Name)
		assert.True(t, RabbitMQCapabilities.SupportsDelay)
		assert.True(t, RabbitMQCapabilities.SupportsNativeDLQ)
		assert.True(t, RabbitMQCapabilities.SupportsPriority)
	})

	t.Run("NATSCapabilities", func(t *testing.T) {
		assert.Equal(t, "nats", NATSCapabilities.Name)
		assert.False(t, NATSCapabilities.SupportsDelay)
		assert.False(t, NATSCapabilities.SupportsNativeDLQ)
		assert.False(t, NATSCapabilities.SupportsAck)
	})

	t.Run("NATSJetStreamCapabilities", func(t *testing.T) {
		assert.Equal(t, "nats-jetstream", NATSJetStreamCapabilities.Name)
		assert.True(t, NATSJetStreamCapabilities.SupportsDelay)
		assert.True(t, NATSJetStreamCapabilities.SupportsNativeDLQ)
		assert.True(t, NATSJetStreamCapabilities.SupportsOrdering)
	})

	t.Run("AWSCapabilities", func(t *testing.T) {
		assert.Equal(t, "aws", AWSCapabilities.Name)
		assert.True(t, AWSCapabilities.SupportsDelay)
		assert.True(t, AWSCapabilities.SupportsNativeDLQ)
		assert.Greater(t, AWSCapabilities.MaxMessageSize, int64(0))
		assert.Greater(t, AWSCapabilities.MaxDelayDuration, int64(0))
	})

	t.Run("SQLiteCapabilities", func(t *testing.T) {
		assert.Equal(t, "sqlite", SQLiteCapabilities.Name)
		assert.True(t, SQLiteCapabilities.SupportsDelay)
		assert.True(t, SQLiteCapabilities.SupportsNativeDLQ)
		assert.True(t, SQLiteCapabilities.SupportsOrdering)
	})

	t.Run("PostgresCapabilities", func(t *testing.T) {
		assert.Equal(t, "postgres", PostgresCapabilities.Name)
		assert.True(t, PostgresCapabilities.SupportsDelay)
		assert.True(t, PostgresCapabilities.SupportsNativeDLQ)
		assert.True(t, PostgresCapabilities.SupportsPriority)
	})

	t.Run("HTTPCapabilities", func(t *testing.T) {
		assert.Equal(t, "http", HTTPCapabilities.Name)
		assert.False(t, HTTPCapabilities.SupportsDelay)
		assert.False(t, HTTPCapabilities.SupportsNativeDLQ)
		assert.True(t, HTTPCapabilities.SupportsTracing)
	})

	t.Run("IOCapabilities", func(t *testing.T) {
		assert.Equal(t, "io", IOCapabilities.Name)
		assert.False(t, IOCapabilities.SupportsDelay)
		assert.False(t, IOCapabilities.SupportsNativeDLQ)
		assert.True(t, IOCapabilities.SupportsOrdering)
	})
}

func TestGetCapabilities_PackageLevel(t *testing.T) {
	// Test the package-level GetCapabilities function
	// Note: This relies on the DefaultRegistry which may be empty in tests
	caps := GetCapabilities("nonexistent")
	assert.Equal(t, "nonexistent", caps.Name)
}

func TestCapabilities_ZeroValue(t *testing.T) {
	// Test that zero value is safe
	var caps Capabilities
	assert.False(t, caps.SupportsDelay)
	assert.False(t, caps.SupportsNativeDLQ)
	assert.False(t, caps.SupportsOrdering)
	assert.True(t, caps.RequiresDelayEmulation())
	assert.True(t, caps.RequiresDLQEmulation())
	assert.False(t, caps.SupportsReliableDelivery())
}

func TestCapabilities_FeatureCombinations(t *testing.T) {
	t.Run("reliable with ordering", func(t *testing.T) {
		caps := Capabilities{
			SupportsAck:      true,
			SupportsNack:     true,
			SupportsOrdering: true,
		}
		assert.True(t, caps.SupportsReliableDelivery())
		assert.True(t, caps.RequiresDelayEmulation()) // No delay support set
	})

	t.Run("delayed with DLQ", func(t *testing.T) {
		caps := Capabilities{
			SupportsDelay:     true,
			SupportsNativeDLQ: true,
		}
		assert.False(t, caps.RequiresDelayEmulation())
		assert.False(t, caps.RequiresDLQEmulation())
	})

	t.Run("minimal capabilities", func(t *testing.T) {
		caps := Capabilities{
			Name: "minimal",
		}
		assert.True(t, caps.RequiresDelayEmulation())
		assert.True(t, caps.RequiresDLQEmulation())
		assert.False(t, caps.SupportsReliableDelivery())
	})
}
