package transport

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drblury/protoflow/internal/runtime/config"
)

func TestSQLiteTransport_PublishSubscribe(t *testing.T) {
	logger := watermill.NopLogger{}
	transport, err := NewSQLiteTransport(SQLiteConfig{
		FilePath:     ":memory:",
		PollInterval: 10 * time.Millisecond,
	}, logger)
	require.NoError(t, err)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test-topic"
	msgChan, err := transport.Subscribe(ctx, topic)
	require.NoError(t, err)

	// Give the subscriber time to start
	time.Sleep(100 * time.Millisecond)

	// Publish a message
	msg := message.NewMessage("test-uuid", []byte(`{"test": "payload"}`))
	msg.Metadata.Set("key", "value")
	err = transport.Publish(topic, msg)
	require.NoError(t, err)

	// Receive the message
	select {
	case received := <-msgChan:
		require.NotNil(t, received, "received message should not be nil")
		assert.Equal(t, "test-uuid", received.UUID)
		assert.JSONEq(t, `{"test": "payload"}`, string(received.Payload))
		assert.Equal(t, "value", received.Metadata.Get("key"))
		received.Ack()
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	// Verify message was deleted after ack
	time.Sleep(200 * time.Millisecond)
	count, err := transport.GetPendingCount(topic)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestSQLiteTransport_NackAndRetry(t *testing.T) {
	logger := watermill.NopLogger{}
	transport, err := NewSQLiteTransport(SQLiteConfig{
		FilePath:     ":memory:",
		PollInterval: 50 * time.Millisecond,
		MaxRetries:   2,
	}, logger)
	require.NoError(t, err)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	topic := "test-nack-topic"
	msgChan, err := transport.Subscribe(ctx, topic)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	msg := message.NewMessage("nack-test-uuid", []byte(`{"test": "nack"}`))
	err = transport.Publish(topic, msg)
	require.NoError(t, err)

	// Nack the message multiple times until it goes to DLQ
	nackCount := 0
	timeout := time.After(10 * time.Second)
	for nackCount < 3 {
		select {
		case received := <-msgChan:
			if received != nil {
				received.Nack()
				nackCount++
			}
		case <-timeout:
			t.Logf("Timeout after %d nacks", nackCount)
			goto checkDLQ
		case <-ctx.Done():
			goto checkDLQ
		}
	}

checkDLQ:
	// Verify message moved to DLQ
	time.Sleep(500 * time.Millisecond)
	dlqCount, err := transport.GetDLQCount(topic)
	require.NoError(t, err)
	assert.Equal(t, int64(1), dlqCount, "message should be in DLQ after max retries")
}

func TestSQLiteTransport_DLQOperations(t *testing.T) {
	logger := watermill.NopLogger{}
	transport, err := NewSQLiteTransport(SQLiteConfig{
		FilePath:     ":memory:",
		PollInterval: 50 * time.Millisecond,
		MaxRetries:   0, // Immediately move to DLQ on nack
	}, logger)
	require.NoError(t, err)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topic := "test-dlq-topic"
	msgChan, err := transport.Subscribe(ctx, topic)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	msg := message.NewMessage("dlq-test-uuid", []byte(`{"test": "dlq"}`))
	err = transport.Publish(topic, msg)
	require.NoError(t, err)

	// Nack to move to DLQ
	select {
	case received := <-msgChan:
		require.NotNil(t, received)
		received.Nack()
		// Wait for nack to be processed
		time.Sleep(500 * time.Millisecond)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	// List DLQ messages
	dlqMessages, err := transport.ListDLQMessages(topic, 10, 0)
	require.NoError(t, err)
	require.Len(t, dlqMessages, 1)
	assert.Contains(t, dlqMessages[0].UUID, "dlq-test-uuid")

	// Replay DLQ message
	err = transport.ReplayDLQMessage(dlqMessages[0].ID)
	require.NoError(t, err)

	// Verify DLQ is empty
	dlqCountAfter, err := transport.GetDLQCount(topic)
	require.NoError(t, err)
	assert.Equal(t, int64(0), dlqCountAfter)

	// Verify message is back in queue
	pendingCount, err := transport.GetPendingCount(topic)
	require.NoError(t, err)
	assert.Equal(t, int64(1), pendingCount)
}

func TestSQLiteTransport_DelayedMessages(t *testing.T) {
	logger := watermill.NopLogger{}
	transport, err := NewSQLiteTransport(SQLiteConfig{
		FilePath:     ":memory:",
		PollInterval: 50 * time.Millisecond,
	}, logger)
	require.NoError(t, err)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topic := "test-delay-topic"
	msgChan, err := transport.Subscribe(ctx, topic)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Publish a delayed message
	msg := message.NewMessage("delayed-uuid", []byte(`{"test": "delayed"}`))
	msg.Metadata.Set("protoflow_delay", "500ms")
	start := time.Now()
	err = transport.Publish(topic, msg)
	require.NoError(t, err)

	// Receive the message (should be delayed)
	select {
	case received := <-msgChan:
		require.NotNil(t, received)
		elapsed := time.Since(start)
		assert.True(t, elapsed >= 400*time.Millisecond, "message should be delayed by at least 400ms, got %v", elapsed)
		received.Ack()
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for delayed message")
	}
}

func TestSQLiteConfig_WithDefaults(t *testing.T) {
	// Test that empty config gets defaults (except MaxRetries which defaults to 0)
	cfg := SQLiteConfig{}
	cfg = cfg.withDefaults()

	assert.Equal(t, "protoflow_queue.db", cfg.FilePath)
	assert.Equal(t, DefaultSQLitePollInterval, cfg.PollInterval)
	assert.Equal(t, 0, cfg.MaxRetries) // Zero value is valid - means no retries

	// Test that explicit positive MaxRetries is preserved
	cfg2 := SQLiteConfig{MaxRetries: 5}
	cfg2 = cfg2.withDefaults()
	assert.Equal(t, 5, cfg2.MaxRetries)

	// Test that -1 gets default
	cfg3 := SQLiteConfig{MaxRetries: -1}
	cfg3 = cfg3.withDefaults()
	assert.Equal(t, DefaultSQLiteMaxRetries, cfg3.MaxRetries)
}

func TestSqliteTransport_FromConfig(t *testing.T) {
	conf := &config.Config{
		PubSubSystem: "sqlite",
		SQLiteFile:   ":memory:",
	}

	transport, err := sqliteTransport(conf, watermill.NopLogger{})
	require.NoError(t, err)
	assert.NotNil(t, transport.Publisher)
	assert.NotNil(t, transport.Subscriber)

	// Clean up
	if closer, ok := transport.Publisher.(interface{ Close() error }); ok {
		closer.Close()
	}
}

func TestSQLiteTransport_Close(t *testing.T) {
	transport, err := NewSQLiteTransport(SQLiteConfig{
		FilePath: ":memory:",
	}, watermill.NopLogger{})
	require.NoError(t, err)

	// Close should not error
	err = transport.Close()
	require.NoError(t, err)

	// Double close should not error
	err = transport.Close()
	require.NoError(t, err)

	// Operations after close should error
	err = transport.Publish("topic", message.NewMessage("uuid", []byte("payload")))
	assert.Error(t, err)
}
