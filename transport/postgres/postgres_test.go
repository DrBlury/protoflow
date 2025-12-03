package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/drblury/protoflow/transport"
)

func TestRegister(t *testing.T) {
	transport.DefaultRegistry = transport.NewRegistry()
	Register()

	// Test main name
	caps := transport.GetCapabilities(TransportName)
	assert.Equal(t, "postgres", caps.Name)
	assert.True(t, caps.SupportsDelay)
	assert.True(t, caps.SupportsNativeDLQ)
	assert.False(t, caps.SupportsTracing)

	// Test alias
	capsAlias := transport.GetCapabilities("postgresql")
	assert.Equal(t, "postgres", capsAlias.Name)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	assert.Equal(t, transport.PostgresCapabilities, caps)
	assert.Equal(t, "postgres", caps.Name)
}

func TestTransportName(t *testing.T) {
	assert.Equal(t, "postgres", TransportName)
}

func TestConfig_withDefaults(t *testing.T) {
	t.Run("empty config gets defaults", func(t *testing.T) {
		cfg := Config{}
		result := cfg.withDefaults()

		assert.Equal(t, DefaultPollInterval, result.PollInterval)
		assert.Equal(t, DefaultMaxRetries, result.MaxRetries)
		assert.Equal(t, DefaultLockTimeout, result.LockTimeout)
		assert.Equal(t, "protoflow", result.SchemaName)
	})

	t.Run("custom values preserved", func(t *testing.T) {
		cfg := Config{
			ConnectionString: "postgres://localhost:5432/test",
			PollInterval:     200,
			MaxRetries:       5,
			LockTimeout:      60,
			SchemaName:       "custom",
			MaxOpenConns:     10,
			MaxIdleConns:     5,
		}
		result := cfg.withDefaults()

		assert.Equal(t, "postgres://localhost:5432/test", result.ConnectionString)
		assert.Equal(t, cfg.PollInterval, result.PollInterval)
		assert.Equal(t, 5, result.MaxRetries)
		assert.Equal(t, cfg.LockTimeout, result.LockTimeout)
		assert.Equal(t, "custom", result.SchemaName)
	})

	t.Run("negative values get defaults", func(t *testing.T) {
		cfg := Config{
			PollInterval: -1,
			MaxRetries:   -1,
			LockTimeout:  -1,
		}
		result := cfg.withDefaults()

		assert.Equal(t, DefaultPollInterval, result.PollInterval)
		assert.Equal(t, DefaultMaxRetries, result.MaxRetries)
		assert.Equal(t, DefaultLockTimeout, result.LockTimeout)
	})
}

func TestDLQMessage(t *testing.T) {
	now := time.Now()
	msg := transport.DLQMessage{
		ID:            1,
		UUID:          "test-uuid",
		OriginalTopic: "test-topic",
		Payload:       []byte("test payload"),
		Metadata:      map[string]string{"key": "value"},
		ErrorMessage:  "test error",
		FailedAt:      now,
		RetryCount:    3,
	}

	assert.Equal(t, int64(1), msg.ID)
	assert.Equal(t, "test-uuid", msg.UUID)
	assert.Equal(t, "test-topic", msg.OriginalTopic)
	assert.Equal(t, []byte("test payload"), msg.Payload)
	assert.Equal(t, "value", msg.Metadata["key"])
	assert.Equal(t, "test error", msg.ErrorMessage)
	assert.Equal(t, 3, msg.RetryCount)
}
