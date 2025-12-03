package sqlite

import (
	"context"
	"os"
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
	assert.Equal(t, "sqlite", caps.Name)
	assert.True(t, caps.SupportsDelay)
	assert.True(t, caps.SupportsNativeDLQ)
	assert.True(t, caps.SupportsAck)
	assert.True(t, caps.SupportsNack)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	assert.Equal(t, transport.SQLiteCapabilities, caps)
	assert.Equal(t, "sqlite", caps.Name)
}

func TestConfig_withDefaults(t *testing.T) {
	t.Run("empty config gets defaults", func(t *testing.T) {
		cfg := Config{}
		result := cfg.withDefaults()

		assert.Equal(t, "protoflow_queue.db", result.FilePath)
		assert.Equal(t, DefaultPollInterval, result.PollInterval)
		// MaxRetries defaults only if < 0, so 0 stays 0
		assert.Equal(t, 0, result.MaxRetries)
	})

	t.Run("custom values preserved", func(t *testing.T) {
		cfg := Config{
			FilePath:     "custom.db",
			PollInterval: 200 * time.Millisecond,
			MaxRetries:   5,
		}
		result := cfg.withDefaults()

		assert.Equal(t, "custom.db", result.FilePath)
		assert.Equal(t, 200*time.Millisecond, result.PollInterval)
		assert.Equal(t, 5, result.MaxRetries)
	})

	t.Run("negative poll interval gets default", func(t *testing.T) {
		cfg := Config{PollInterval: -1}
		result := cfg.withDefaults()
		assert.Equal(t, DefaultPollInterval, result.PollInterval)
	})

	t.Run("negative max retries gets default", func(t *testing.T) {
		cfg := Config{MaxRetries: -1}
		result := cfg.withDefaults()
		assert.Equal(t, DefaultMaxRetries, result.MaxRetries)
	})

	t.Run("zero max retries preserved", func(t *testing.T) {
		cfg := Config{MaxRetries: 0}
		result := cfg.withDefaults()
		assert.Equal(t, 0, result.MaxRetries)
	})
}

func TestNew(t *testing.T) {
	t.Run("creates transport with in-memory db", func(t *testing.T) {
		cfg := Config{FilePath: ":memory:"}
		tr, err := New(cfg, watermill.NopLogger{})

		require.NoError(t, err)
		require.NotNil(t, tr)
		assert.NotNil(t, tr.db)
		assert.NotNil(t, tr.closedChan)
		assert.False(t, tr.closed)

		err = tr.Close()
		require.NoError(t, err)
	})

	t.Run("creates transport with file db", func(t *testing.T) {
		tmpFile := "test_sqlite_" + time.Now().Format("20060102150405") + ".db"
		defer os.Remove(tmpFile)

		cfg := Config{FilePath: tmpFile}
		tr, err := New(cfg, watermill.NopLogger{})

		require.NoError(t, err)
		require.NotNil(t, tr)

		err = tr.Close()
		require.NoError(t, err)
	})

	t.Run("initializes schema correctly", func(t *testing.T) {
		cfg := Config{FilePath: ":memory:"}
		tr, err := New(cfg, watermill.NopLogger{})
		require.NoError(t, err)
		defer tr.Close()

		var count int
		err = tr.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='messages'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		err = tr.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='dead_letter_queue'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestBuild(t *testing.T) {
	t.Run("builds transport from config", func(t *testing.T) {
		cfg := &mockConfig{sqliteFile: ":memory:"}
		tr, err := Build(context.Background(), cfg, watermill.NopLogger{})

		require.NoError(t, err)
		assert.NotNil(t, tr.Publisher)
		assert.NotNil(t, tr.Subscriber)

		if pub, ok := tr.Publisher.(*Transport); ok {
			pub.Close()
		}
	})
}

func TestTransport_Publish(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	t.Run("publishes single message", func(t *testing.T) {
		msg := message.NewMessage("test-uuid-1", []byte("test payload"))
		err := tr.Publish("test.topic", msg)
		require.NoError(t, err)

		count, err := tr.GetPendingCount("test.topic")
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("publishes multiple messages", func(t *testing.T) {
		msg1 := message.NewMessage("test-uuid-2", []byte("payload 1"))
		msg2 := message.NewMessage("test-uuid-3", []byte("payload 2"))
		err := tr.Publish("test.topic2", msg1, msg2)
		require.NoError(t, err)

		count, err := tr.GetPendingCount("test.topic2")
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("publishes with delay metadata", func(t *testing.T) {
		msg := message.NewMessage("test-uuid-delay", []byte("delayed payload"))
		msg.Metadata.Set("protoflow_delay", "1s")
		err := tr.Publish("test.topic.delayed", msg)
		require.NoError(t, err)

		count, err := tr.GetPendingCount("test.topic.delayed")
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("fails on closed transport", func(t *testing.T) {
		closedTr := newTestTransport(t)
		closedTr.Close()

		msg := message.NewMessage("test-uuid-closed", []byte("test"))
		err := closedTr.Publish("test.topic", msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})
}

func TestTransport_Subscribe(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	t.Run("subscribes to topic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		msgChan, err := tr.Subscribe(ctx, "sub.topic")
		require.NoError(t, err)
		require.NotNil(t, msgChan)

		msg := message.NewMessage("sub-test-1", []byte("subscribe test"))
		err = tr.Publish("sub.topic", msg)
		require.NoError(t, err)

		select {
		case received := <-msgChan:
			assert.Equal(t, "sub-test-1", received.UUID)
			assert.EqualValues(t, []byte("subscribe test"), received.Payload)
			received.Ack()
		case <-ctx.Done():
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("fails on closed transport", func(t *testing.T) {
		closedTr := newTestTransport(t)
		closedTr.Close()

		_, err := closedTr.Subscribe(context.Background(), "test.topic")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})
}

func TestTransport_Close(t *testing.T) {
	t.Run("closes transport", func(t *testing.T) {
		tr := newTestTransport(t)
		err := tr.Close()
		require.NoError(t, err)
		assert.True(t, tr.closed)
	})

	t.Run("idempotent close", func(t *testing.T) {
		tr := newTestTransport(t)
		err := tr.Close()
		require.NoError(t, err)

		err = tr.Close()
		require.NoError(t, err)
	})
}

func TestTransport_GetCapabilities(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	caps := tr.GetCapabilities()
	assert.Equal(t, transport.SQLiteCapabilities, caps)
}

func TestTransport_GetDB(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	db := tr.GetDB()
	assert.NotNil(t, db)
	assert.Equal(t, tr.db, db)
}

func TestTransport_GetPendingCount(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	count, err := tr.GetPendingCount("pending.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	msg := message.NewMessage("pending-1", []byte("test"))
	err = tr.Publish("pending.topic", msg)
	require.NoError(t, err)

	count, err = tr.GetPendingCount("pending.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestTransport_GetDLQCount(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	count, err := tr.GetDLQCount("dlq.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	_, err = tr.db.Exec(`
		INSERT INTO dead_letter_queue (uuid, original_topic, payload, metadata, error_message)
		VALUES ('dlq-uuid-1', 'dlq.topic', 'test', '{}', 'test error')
	`)
	require.NoError(t, err)

	count, err = tr.GetDLQCount("dlq.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestTransport_ReplayDLQMessage(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	result, err := tr.db.Exec(`
		INSERT INTO dead_letter_queue (uuid, original_topic, payload, metadata, error_message, retry_count)
		VALUES ('replay-uuid', 'replay.topic', 'replay payload', '{"key":"value"}', 'error msg', 3)
	`)
	require.NoError(t, err)

	dlqID, err := result.LastInsertId()
	require.NoError(t, err)

	err = tr.ReplayDLQMessage(dlqID)
	require.NoError(t, err)

	count, err := tr.GetPendingCount("replay.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	dlqCount, err := tr.GetDLQCount("replay.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(0), dlqCount)
}

func TestTransport_ReplayAllDLQ(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	for i := 0; i < 3; i++ {
		_, err := tr.db.Exec(`
			INSERT INTO dead_letter_queue (uuid, original_topic, payload, metadata, error_message)
			VALUES (?, 'replay-all.topic', 'payload', '{}', 'error')
		`, "replay-all-uuid-"+string(rune('0'+i)))
		require.NoError(t, err)
	}

	affected, err := tr.ReplayAllDLQ("replay-all.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(3), affected)

	count, err := tr.GetPendingCount("replay-all.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	dlqCount, err := tr.GetDLQCount("replay-all.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(0), dlqCount)
}

func TestTransport_PurgeDLQ(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	for i := 0; i < 3; i++ {
		_, err := tr.db.Exec(`
			INSERT INTO dead_letter_queue (uuid, original_topic, payload, metadata, error_message)
			VALUES (?, 'purge.topic', 'payload', '{}', 'error')
		`, "purge-uuid-"+string(rune('0'+i)))
		require.NoError(t, err)
	}

	affected, err := tr.PurgeDLQ("purge.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(3), affected)

	count, err := tr.GetDLQCount("purge.topic")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestTransport_ListDLQMessages(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	for i := 0; i < 5; i++ {
		_, err := tr.db.Exec(`
			INSERT INTO dead_letter_queue (uuid, original_topic, payload, metadata, error_message, retry_count)
			VALUES (?, 'list.topic', ?, '{}', 'error msg', ?)
		`, "list-uuid-"+string(rune('0'+i)), []byte("payload-"+string(rune('0'+i))), i)
		require.NoError(t, err)
	}

	t.Run("list with pagination", func(t *testing.T) {
		msgs, err := tr.ListDLQMessages("list.topic", 2, 0)
		require.NoError(t, err)
		assert.Len(t, msgs, 2)
	})

	t.Run("list with offset", func(t *testing.T) {
		msgs, err := tr.ListDLQMessages("list.topic", 10, 3)
		require.NoError(t, err)
		assert.Len(t, msgs, 2)
	})

	t.Run("message fields populated", func(t *testing.T) {
		msgs, err := tr.ListDLQMessages("list.topic", 1, 0)
		require.NoError(t, err)
		require.Len(t, msgs, 1)

		msg := msgs[0]
		assert.NotZero(t, msg.ID)
		assert.NotEmpty(t, msg.UUID)
		assert.Equal(t, "list.topic", msg.OriginalTopic)
		assert.NotEmpty(t, msg.Payload)
		assert.NotNil(t, msg.Metadata)
		assert.Equal(t, "error msg", msg.ErrorMessage)
		assert.False(t, msg.FailedAt.IsZero())
	})
}

func TestTransport_MessageAckNack(t *testing.T) {
	tr := newTestTransport(t)
	defer tr.Close()

	t.Run("acked message is removed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		msgChan, err := tr.Subscribe(ctx, "ack.topic")
		require.NoError(t, err)

		msg := message.NewMessage("ack-test-1", []byte("ack test"))
		err = tr.Publish("ack.topic", msg)
		require.NoError(t, err)

		select {
		case received := <-msgChan:
			received.Ack()
			time.Sleep(50 * time.Millisecond)
		case <-ctx.Done():
			t.Fatal("timeout waiting for message")
		}

		count, err := tr.GetPendingCount("ack.topic")
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("nacked message goes to DLQ after max retries", func(t *testing.T) {
		cfg := Config{FilePath: ":memory:", MaxRetries: 0, PollInterval: 50 * time.Millisecond}
		tr, err := New(cfg, watermill.NopLogger{})
		require.NoError(t, err)
		defer tr.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		msgChan, err := tr.Subscribe(ctx, "nack.topic")
		require.NoError(t, err)

		msg := message.NewMessage("nack-test-1", []byte("nack test"))
		err = tr.Publish("nack.topic", msg)
		require.NoError(t, err)

		select {
		case received := <-msgChan:
			received.Nack()
			time.Sleep(100 * time.Millisecond)
		case <-ctx.Done():
			t.Fatal("timeout waiting for message")
		}

		dlqCount, err := tr.GetDLQCount("nack.topic")
		require.NoError(t, err)
		assert.Equal(t, int64(1), dlqCount)
	})
}

func newTestTransport(t *testing.T) *Transport {
	t.Helper()
	cfg := Config{
		FilePath:     ":memory:",
		PollInterval: 50 * time.Millisecond,
		MaxRetries:   3,
	}
	tr, err := New(cfg, watermill.NopLogger{})
	require.NoError(t, err)
	return tr
}

type mockConfig struct {
	sqliteFile string
}

func (m *mockConfig) GetPubSubSystem() string       { return "sqlite" }
func (m *mockConfig) GetKafkaBrokers() []string     { return nil }
func (m *mockConfig) GetKafkaConsumerGroup() string { return "" }
func (m *mockConfig) GetRabbitMQURL() string        { return "" }
func (m *mockConfig) GetNATSURL() string            { return "" }
func (m *mockConfig) GetHTTPServerAddress() string  { return "" }
func (m *mockConfig) GetHTTPPublisherURL() string   { return "" }
func (m *mockConfig) GetIOFile() string             { return "" }
func (m *mockConfig) GetSQLiteFile() string         { return m.sqliteFile }
func (m *mockConfig) GetPostgresURL() string        { return "" }
func (m *mockConfig) GetAWSRegion() string          { return "" }
func (m *mockConfig) GetAWSAccountID() string       { return "" }
func (m *mockConfig) GetAWSAccessKeyID() string     { return "" }
func (m *mockConfig) GetAWSSecretAccessKey() string { return "" }
func (m *mockConfig) GetAWSEndpoint() string        { return "" }
