package io

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drblury/protoflow/transport"
)

func TestRegister(t *testing.T) {
	transport.DefaultRegistry = transport.NewRegistry()
	Register()

	caps := transport.GetCapabilities(TransportName)
	assert.Equal(t, "io", caps.Name)
	assert.True(t, caps.SupportsOrdering)
	assert.False(t, caps.SupportsDelay)
	assert.False(t, caps.SupportsAck)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	assert.Equal(t, transport.IOCapabilities, caps)
	assert.Equal(t, "io", caps.Name)
}

func TestTransportName(t *testing.T) {
	assert.Equal(t, "io", TransportName)
}

func TestDefaultFilePath(t *testing.T) {
	assert.Equal(t, "messages.log", DefaultFilePath)
}

func TestBuild(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_messages.log")

	t.Run("creates transport with custom file", func(t *testing.T) {
		cfg := &mockConfig{ioFile: testFile}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.NotNil(t, tr.Publisher)
		assert.NotNil(t, tr.Subscriber)
	})

	t.Run("uses default file path when empty", func(t *testing.T) {
		cfg := &mockConfig{ioFile: ""}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.NotNil(t, tr.Publisher)
		assert.NotNil(t, tr.Subscriber)

		os.Remove(DefaultFilePath)
	})

	t.Run("uses custom publisher factory", func(t *testing.T) {
		originalFactory := PublisherFactory
		defer func() { PublisherFactory = originalFactory }()

		mockPub := &Publisher{filePath: "mock"}
		PublisherFactory = func(filePath string, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return mockPub, nil
		}

		cfg := &mockConfig{ioFile: testFile}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.Equal(t, mockPub, tr.Publisher)
	})

	t.Run("uses custom subscriber factory", func(t *testing.T) {
		originalFactory := SubscriberFactory
		defer func() { SubscriberFactory = originalFactory }()

		mockSub := &Subscriber{filePath: "mock"}
		SubscriberFactory = func(filePath string, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			return mockSub, nil
		}

		cfg := &mockConfig{ioFile: testFile}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.Equal(t, mockSub, tr.Subscriber)
	})
}

func TestPublisher_Publish(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "publish_test.log")

	pub := &Publisher{filePath: testFile, logger: watermill.NopLogger{}}

	t.Run("publishes single message", func(t *testing.T) {
		msg := message.NewMessage("test-uuid-1", []byte("test payload"))
		msg.Metadata.Set("key", "value")

		err := pub.Publish("test.topic", msg)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test-uuid-1")
		assert.Contains(t, string(content), "test.topic")
		// Payload is base64-encoded in JSON, so check for the UUID instead
		assert.Contains(t, string(content), `"key":"value"`)
	})

	t.Run("publishes multiple messages", func(t *testing.T) {
		msg1 := message.NewMessage("multi-uuid-1", []byte("payload 1"))
		msg2 := message.NewMessage("multi-uuid-2", []byte("payload 2"))

		err := pub.Publish("multi.topic", msg1, msg2)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "multi-uuid-1")
		assert.Contains(t, string(content), "multi-uuid-2")
	})
}

func TestPublisher_Close(t *testing.T) {
	pub := &Publisher{}
	err := pub.Close()
	assert.NoError(t, err)
}

func TestSubscriber_Subscribe(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subscribe_test.log")

	pub := &Publisher{filePath: testFile, logger: watermill.NopLogger{}}
	msg := message.NewMessage("sub-uuid-1", []byte("subscribe test"))
	err := pub.Publish("sub.topic", msg)
	require.NoError(t, err)

	sub := &Subscriber{filePath: testFile, logger: watermill.NopLogger{}}

	t.Run("subscribes and receives message", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		msgChan, err := sub.Subscribe(ctx, "sub.topic")
		require.NoError(t, err)

		select {
		case received := <-msgChan:
			assert.Equal(t, "sub-uuid-1", received.UUID)
			assert.EqualValues(t, []byte("subscribe test"), received.Payload)
			received.Ack()
		case <-ctx.Done():
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("filters by topic", func(t *testing.T) {
		msg := message.NewMessage("other-topic-uuid", []byte("other topic"))
		err := pub.Publish("other.topic", msg)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		msgChan, err := sub.Subscribe(ctx, "non.existent.topic")
		require.NoError(t, err)

		select {
		case <-msgChan:
			t.Fatal("should not receive message for different topic")
		case <-ctx.Done():
			// Expected timeout
		}
	})
}

func TestSubscriber_Close(t *testing.T) {
	sub := &Subscriber{}
	err := sub.Close()
	assert.NoError(t, err)
}

func TestStoredMessage_Structure(t *testing.T) {
	sm := storedMessage{
		UUID:     "test-uuid",
		Metadata: map[string]string{"key": "value"},
		Payload:  []byte("test payload"),
		Topic:    "test.topic",
	}

	assert.Equal(t, "test-uuid", sm.UUID)
	assert.Equal(t, "value", sm.Metadata["key"])
	assert.Equal(t, []byte("test payload"), sm.Payload)
	assert.Equal(t, "test.topic", sm.Topic)
}

type mockConfig struct {
	ioFile string
}

func (m *mockConfig) GetPubSubSystem() string       { return "io" }
func (m *mockConfig) GetKafkaBrokers() []string     { return nil }
func (m *mockConfig) GetKafkaConsumerGroup() string { return "" }
func (m *mockConfig) GetRabbitMQURL() string        { return "" }
func (m *mockConfig) GetNATSURL() string            { return "" }
func (m *mockConfig) GetHTTPServerAddress() string  { return "" }
func (m *mockConfig) GetHTTPPublisherURL() string   { return "" }
func (m *mockConfig) GetIOFile() string             { return m.ioFile }
func (m *mockConfig) GetSQLiteFile() string         { return "" }
func (m *mockConfig) GetPostgresURL() string        { return "" }
func (m *mockConfig) GetAWSRegion() string          { return "" }
func (m *mockConfig) GetAWSAccountID() string       { return "" }
func (m *mockConfig) GetAWSAccessKeyID() string     { return "" }
func (m *mockConfig) GetAWSSecretAccessKey() string { return "" }
func (m *mockConfig) GetAWSEndpoint() string        { return "" }
