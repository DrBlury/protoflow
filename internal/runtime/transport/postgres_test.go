package transport

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	_ "github.com/lib/pq"
)

// skipIfNoPostgres skips the test if PostgreSQL is not available.
// Set PROTOFLOW_TEST_POSTGRES_URL to run these tests.
// Example: export PROTOFLOW_TEST_POSTGRES_URL="postgres://postgres:postgres@localhost:5432/protoflow_test?sslmode=disable"
func skipIfNoPostgres(t *testing.T) string {
	t.Helper()
	// For CI/CD or local testing, you can use Docker:
	// docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=protoflow_test postgres:16
	connStr := "postgres://postgres:postgres@localhost:5432/protoflow_test?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	// Clean up schema for clean tests
	if _, err := db.Exec("DROP SCHEMA IF EXISTS protoflow_test CASCADE"); err != nil {
		t.Logf("Warning: failed to clean up test schema: %v", err)
	}

	return connStr
}

func TestPostgresTransport_NewTransport(t *testing.T) {
	connStr := skipIfNoPostgres(t)

	cfg := PostgresConfig{
		ConnectionString: connStr,
		SchemaName:       "protoflow_test",
	}

	transport, err := NewPostgresTransport(cfg, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	defer transport.Close()

	// Verify schema was created
	var exists bool
	err = transport.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = 'protoflow_test' AND table_name = 'messages')
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check schema: %v", err)
	}
	if !exists {
		t.Fatal("messages table was not created")
	}
}

func TestPostgresTransport_PublishSubscribe(t *testing.T) {
	connStr := skipIfNoPostgres(t)

	cfg := PostgresConfig{
		ConnectionString: connStr,
		SchemaName:       "protoflow_test",
		PollInterval:     50 * time.Millisecond,
	}

	transport, err := NewPostgresTransport(cfg, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe first
	msgChan, err := transport.Subscribe(ctx, "test-topic")
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	// Publish a message
	testPayload := []byte("hello postgres")
	msg := message.NewMessage(watermill.NewUUID(), testPayload)
	msg.Metadata.Set("test-key", "test-value")

	if err := transport.Publish("test-topic", msg); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Wait for message
	select {
	case received := <-msgChan:
		if string(received.Payload) != string(testPayload) {
			t.Errorf("payload mismatch: got %s, want %s", received.Payload, testPayload)
		}
		if received.Metadata.Get("test-key") != "test-value" {
			t.Errorf("metadata mismatch: got %s, want test-value", received.Metadata.Get("test-key"))
		}
		received.Ack()
	case <-ctx.Done():
		t.Fatal("timeout waiting for message")
	}

	// Verify message was deleted after ack
	time.Sleep(100 * time.Millisecond)
	count, err := transport.GetPendingCount("test-topic")
	if err != nil {
		t.Fatalf("failed to get pending count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 pending messages, got %d", count)
	}
}

func TestPostgresTransport_DelayedMessages(t *testing.T) {
	connStr := skipIfNoPostgres(t)

	cfg := PostgresConfig{
		ConnectionString: connStr,
		SchemaName:       "protoflow_test",
		PollInterval:     50 * time.Millisecond,
	}

	transport, err := NewPostgresTransport(cfg, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgChan, err := transport.Subscribe(ctx, "delayed-topic")
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	// Publish a delayed message
	msg := message.NewMessage(watermill.NewUUID(), []byte("delayed"))
	msg.Metadata.Set("protoflow_delay", "500ms")

	publishTime := time.Now()
	if err := transport.Publish("delayed-topic", msg); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Wait for message
	select {
	case received := <-msgChan:
		elapsed := time.Since(publishTime)
		if elapsed < 400*time.Millisecond {
			t.Errorf("message received too early: %v", elapsed)
		}
		received.Ack()
	case <-ctx.Done():
		t.Fatal("timeout waiting for delayed message")
	}
}

func TestPostgresTransport_NackAndRetry(t *testing.T) {
	connStr := skipIfNoPostgres(t)

	cfg := PostgresConfig{
		ConnectionString: connStr,
		SchemaName:       "protoflow_test",
		PollInterval:     50 * time.Millisecond,
		MaxRetries:       2, // Will go to DLQ after 2 retries
	}

	transport, err := NewPostgresTransport(cfg, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	msgChan, err := transport.Subscribe(ctx, "retry-topic")
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	// Publish a message
	msg := message.NewMessage(watermill.NewUUID(), []byte("will-fail"))
	if err := transport.Publish("retry-topic", msg); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Nack the message multiple times until it goes to DLQ
	nackedCount := 0
	for {
		select {
		case received := <-msgChan:
			nackedCount++
			received.Nack()
			if nackedCount > 3 {
				t.Fatal("message should have moved to DLQ by now")
			}
		case <-time.After(5 * time.Second):
			// Should have moved to DLQ
			dlqCount, err := transport.GetDLQCount("retry-topic")
			if err != nil {
				t.Fatalf("failed to get DLQ count: %v", err)
			}
			if dlqCount != 1 {
				t.Errorf("expected 1 message in DLQ, got %d", dlqCount)
			}
			return
		case <-ctx.Done():
			t.Fatal("context cancelled")
		}
	}
}

// setupDLQTestTransport creates a transport configured for DLQ testing with MaxRetries=0.
func setupDLQTestTransport(t *testing.T, topic string) (*PostgresTransport, context.Context, context.CancelFunc, <-chan *message.Message) {
	t.Helper()
	connStr := skipIfNoPostgres(t)

	cfg := PostgresConfig{
		ConnectionString: connStr,
		SchemaName:       "protoflow_test",
		PollInterval:     50 * time.Millisecond,
		MaxRetries:       0,
	}

	transport, err := NewPostgresTransport(cfg, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	msgChan, err := transport.Subscribe(ctx, topic)
	if err != nil {
		transport.Close()
		cancel()
		t.Fatalf("failed to subscribe: %v", err)
	}

	return transport, ctx, cancel, msgChan
}

// publishAndNackToDLQ publishes a message and nacks it to move to DLQ.
func publishAndNackToDLQ(t *testing.T, transport *PostgresTransport, msgChan <-chan *message.Message, topic string) {
	t.Helper()
	msg := message.NewMessage(watermill.NewUUID(), []byte("dlq-test"))
	msg.Metadata.Set("custom", "metadata")
	if err := transport.Publish(topic, msg); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	select {
	case received := <-msgChan:
		received.Nack()
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
	time.Sleep(200 * time.Millisecond)
}

func TestPostgresTransport_DLQOperations(t *testing.T) {
	transport, _, cancel, msgChan := setupDLQTestTransport(t, "dlq-topic")
	defer transport.Close()
	defer cancel()

	publishAndNackToDLQ(t, transport, msgChan, "dlq-topic")

	t.Run("CheckDLQCount", func(t *testing.T) {
		count, err := transport.GetDLQCount("dlq-topic")
		if err != nil {
			t.Fatalf("failed to get DLQ count: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 message in DLQ, got %d", count)
		}
	})

	t.Run("ListVerifyAndReplay", func(t *testing.T) {
		dlqMsgs, err := transport.ListDLQMessages("dlq-topic", 10, 0)
		if err != nil {
			t.Fatalf("failed to list DLQ messages: %v", err)
		}
		if len(dlqMsgs) != 1 {
			t.Fatalf("expected 1 DLQ message, got %d", len(dlqMsgs))
		}
		if dlqMsgs[0].Metadata["custom"] != "metadata" {
			t.Errorf("metadata not preserved in DLQ")
		}

		if err := transport.ReplayDLQMessage(dlqMsgs[0].ID); err != nil {
			t.Fatalf("failed to replay message: %v", err)
		}

		count, err := transport.GetDLQCount("dlq-topic")
		if err != nil {
			t.Fatalf("failed to get DLQ count: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 messages in DLQ after replay, got %d", count)
		}

		pending, err := transport.GetPendingCount("dlq-topic")
		if err != nil {
			t.Fatalf("failed to get pending count: %v", err)
		}
		if pending != 1 {
			t.Errorf("expected 1 pending message after replay, got %d", pending)
		}
	})
}

// nackMessages receives and nacks the specified number of messages.
func nackMessages(t *testing.T, msgChan <-chan *message.Message, count int) {
	t.Helper()
	nackedCount := 0
	for nackedCount < count {
		select {
		case received := <-msgChan:
			received.Nack()
			nackedCount++
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout after nacking %d messages", nackedCount)
		}
	}
}

func TestPostgresTransport_ReplayAllDLQ(t *testing.T) {
	transport, _, cancel, msgChan := setupDLQTestTransport(t, "replay-all-topic")
	defer transport.Close()
	defer cancel()

	// Publish multiple messages
	for i := 0; i < 3; i++ {
		msg := message.NewMessage(watermill.NewUUID(), []byte("batch"))
		if err := transport.Publish("replay-all-topic", msg); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}
	}

	nackMessages(t, msgChan, 3)
	time.Sleep(300 * time.Millisecond)

	count, err := transport.GetDLQCount("replay-all-topic")
	if err != nil {
		t.Fatalf("failed to get DLQ count: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 messages in DLQ, got %d", count)
	}

	replayed, err := transport.ReplayAllDLQ("replay-all-topic")
	if err != nil {
		t.Fatalf("failed to replay all: %v", err)
	}
	if replayed != 3 {
		t.Errorf("expected 3 replayed, got %d", replayed)
	}

	count, err = transport.GetDLQCount("replay-all-topic")
	if err != nil {
		t.Fatalf("failed to get DLQ count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 messages in DLQ, got %d", count)
	}
}

func TestPostgresTransport_PurgeDLQ(t *testing.T) {
	connStr := skipIfNoPostgres(t)

	cfg := PostgresConfig{
		ConnectionString: connStr,
		SchemaName:       "protoflow_test",
		PollInterval:     50 * time.Millisecond,
		MaxRetries:       0,
	}

	transport, err := NewPostgresTransport(cfg, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	defer transport.Close()

	// Insert directly into DLQ for testing
	_, err = transport.db.Exec(`
		INSERT INTO protoflow_test.dead_letter_queue (uuid, original_topic, payload, error_message)
		VALUES ('test-1', 'purge-topic', 'payload1', 'error'),
		       ('test-2', 'purge-topic', 'payload2', 'error')
	`)
	if err != nil {
		t.Fatalf("failed to insert DLQ messages: %v", err)
	}

	// Purge
	purged, err := transport.PurgeDLQ("purge-topic")
	if err != nil {
		t.Fatalf("failed to purge DLQ: %v", err)
	}
	if purged != 2 {
		t.Errorf("expected 2 purged, got %d", purged)
	}

	// Verify empty
	count, err := transport.GetDLQCount("purge-topic")
	if err != nil {
		t.Fatalf("failed to get DLQ count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 messages after purge, got %d", count)
	}
}

func TestPostgresTransport_CleanupExpiredLocks(t *testing.T) {
	connStr := skipIfNoPostgres(t)

	cfg := PostgresConfig{
		ConnectionString: connStr,
		SchemaName:       "protoflow_test",
	}

	transport, err := NewPostgresTransport(cfg, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	defer transport.Close()

	// Insert a message with an expired lock
	expiredTime := time.Now().Add(-1 * time.Hour)
	_, err = transport.db.Exec(`
		INSERT INTO protoflow_test.messages (uuid, topic, payload, locked_until)
		VALUES ('expired-lock', 'cleanup-topic', 'payload', $1)
	`, expiredTime)
	if err != nil {
		t.Fatalf("failed to insert message: %v", err)
	}

	// Cleanup
	cleaned, err := transport.CleanupExpiredLocks()
	if err != nil {
		t.Fatalf("failed to cleanup locks: %v", err)
	}
	if cleaned != 1 {
		t.Errorf("expected 1 cleaned, got %d", cleaned)
	}

	// Verify lock is cleared
	var lockedUntil sql.NullTime
	err = transport.db.QueryRow(`
		SELECT locked_until FROM protoflow_test.messages WHERE uuid = 'expired-lock'
	`).Scan(&lockedUntil)
	if err != nil {
		t.Fatalf("failed to query message: %v", err)
	}
	if lockedUntil.Valid {
		t.Error("locked_until should be NULL after cleanup")
	}
}

func TestPostgresTransport_ClosedTransport(t *testing.T) {
	connStr := skipIfNoPostgres(t)

	cfg := PostgresConfig{
		ConnectionString: connStr,
		SchemaName:       "protoflow_test",
	}

	transport, err := NewPostgresTransport(cfg, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}

	// Close the transport
	if err := transport.Close(); err != nil {
		t.Fatalf("failed to close transport: %v", err)
	}

	// Try to publish - should fail
	msg := message.NewMessage(watermill.NewUUID(), []byte("test"))
	if err := transport.Publish("topic", msg); err == nil {
		t.Error("expected error publishing to closed transport")
	}

	// Try to subscribe - should fail
	_, err = transport.Subscribe(context.Background(), "topic")
	if err == nil {
		t.Error("expected error subscribing to closed transport")
	}

	// Double close should be safe
	if err := transport.Close(); err != nil {
		t.Errorf("double close should not error: %v", err)
	}
}

func TestPostgresTransport_MissingConnectionString(t *testing.T) {
	cfg := PostgresConfig{}
	_, err := NewPostgresTransport(cfg, watermill.NopLogger{})
	if err == nil {
		t.Error("expected error with empty connection string")
	}
}

func TestPostgresConfig_Defaults(t *testing.T) {
	cfg := PostgresConfig{
		ConnectionString: "postgres://localhost/test",
	}
	cfg = cfg.withDefaults()

	if cfg.PollInterval != DefaultPostgresPollInterval {
		t.Errorf("expected default poll interval %v, got %v", DefaultPostgresPollInterval, cfg.PollInterval)
	}
	if cfg.MaxRetries != DefaultPostgresMaxRetries {
		t.Errorf("expected default max retries %d, got %d", DefaultPostgresMaxRetries, cfg.MaxRetries)
	}
	if cfg.LockTimeout != DefaultPostgresLockTimeout {
		t.Errorf("expected default lock timeout %v, got %v", DefaultPostgresLockTimeout, cfg.LockTimeout)
	}
	if cfg.SchemaName != "protoflow" {
		t.Errorf("expected default schema 'protoflow', got %s", cfg.SchemaName)
	}
	if cfg.MaxOpenConns != 10 {
		t.Errorf("expected default max open conns 10, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("expected default max idle conns 5, got %d", cfg.MaxIdleConns)
	}
}
