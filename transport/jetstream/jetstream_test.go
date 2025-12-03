package jetstream

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/drblury/protoflow/transport"
)

func TestRegister(t *testing.T) {
	transport.DefaultRegistry = transport.NewRegistry()
	Register()

	caps := transport.GetCapabilities(TransportName)
	assert.Equal(t, "nats-jetstream", caps.Name)
	assert.True(t, caps.SupportsDelay)
	assert.True(t, caps.SupportsNativeDLQ)
	assert.True(t, caps.SupportsTracing)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	assert.Equal(t, transport.NATSJetStreamCapabilities, caps)
	assert.Equal(t, "nats-jetstream", caps.Name)
}

func TestTransportName(t *testing.T) {
	assert.Equal(t, "nats-jetstream", TransportName)
}

func TestConfig_withDefaults(t *testing.T) {
	t.Run("empty config gets defaults", func(t *testing.T) {
		cfg := Config{}
		result := cfg.withDefaults()

		assert.Equal(t, "PROTOFLOW", result.StreamName)
		assert.Equal(t, DefaultMaxDeliver, result.MaxDeliver)
		assert.Equal(t, DefaultAckWait, result.AckWait)
		assert.Equal(t, 1, result.Replicas)
	})

	t.Run("custom values preserved", func(t *testing.T) {
		cfg := Config{
			URL:             "nats://localhost:4222",
			StreamName:      "CUSTOM",
			MaxDeliver:      5,
			AckWait:         60,
			Replicas:        3,
			RetentionPolicy: "workqueue",
		}
		result := cfg.withDefaults()

		assert.Equal(t, "nats://localhost:4222", result.URL)
		assert.Equal(t, "CUSTOM", result.StreamName)
		assert.Equal(t, 5, result.MaxDeliver)
		assert.Equal(t, cfg.AckWait, result.AckWait)
		assert.Equal(t, 3, result.Replicas)
		assert.Equal(t, "workqueue", result.RetentionPolicy)
	})

	t.Run("negative values get defaults", func(t *testing.T) {
		cfg := Config{
			MaxDeliver: -1,
			AckWait:    -1,
			Replicas:   -1,
		}
		result := cfg.withDefaults()

		assert.Equal(t, DefaultMaxDeliver, result.MaxDeliver)
		assert.Equal(t, DefaultAckWait, result.AckWait)
		assert.Equal(t, 1, result.Replicas)
	})
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "pf_delay_ms", MetadataDelay)
	assert.Equal(t, 3, DefaultMaxDeliver)
}
