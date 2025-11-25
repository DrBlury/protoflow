package transport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/drblury/protoflow/internal/runtime/config"
)

const (
	// DefaultSQLitePollInterval is the default interval for polling new messages.
	DefaultSQLitePollInterval = 100 * time.Millisecond
	// DefaultSQLiteMaxRetries is the default number of retries before moving to DLQ.
	DefaultSQLiteMaxRetries = 3
)

// SQLiteConfig holds SQLite-specific configuration.
type SQLiteConfig struct {
	// FilePath is the path to the SQLite database file.
	// Use ":memory:" for an in-memory database (useful for testing).
	FilePath string
	// PollInterval is the interval for polling new messages.
	PollInterval time.Duration
	// MaxRetries is the number of times to retry a message before giving up.
	MaxRetries int
}

func (c SQLiteConfig) withDefaults() SQLiteConfig {
	if c.FilePath == "" {
		c.FilePath = "protoflow_queue.db"
	}
	if c.PollInterval <= 0 {
		c.PollInterval = DefaultSQLitePollInterval
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = DefaultSQLiteMaxRetries
	}
	return c
}

// SQLiteTransport implements both Publisher and Subscriber interfaces for SQLite.
type SQLiteTransport struct {
	db     *sql.DB
	config SQLiteConfig
	logger watermill.LoggerAdapter

	subscriptions map[string]chan *message.Message
	subMu         sync.RWMutex

	closed     bool
	closedMu   sync.RWMutex
	closedChan chan struct{}
	wg         sync.WaitGroup
}

// NewSQLiteTransport creates a new SQLite-based transport.
func NewSQLiteTransport(cfg SQLiteConfig, logger watermill.LoggerAdapter) (*SQLiteTransport, error) {
	cfg = cfg.withDefaults()

	db, err := sql.Open("sqlite3", cfg.FilePath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Set connection pool settings for better concurrency
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes well
	db.SetMaxIdleConns(1)

	t := &SQLiteTransport{
		db:            db,
		config:        cfg,
		logger:        logger,
		subscriptions: make(map[string]chan *message.Message),
		closedChan:    make(chan struct{}),
	}

	if err := t.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return t, nil
}

func (t *SQLiteTransport) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT NOT NULL UNIQUE,
		topic TEXT NOT NULL,
		payload BLOB NOT NULL,
		metadata TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		available_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		locked_until TIMESTAMP,
		retry_count INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending'
	);

	CREATE INDEX IF NOT EXISTS idx_messages_topic_status ON messages(topic, status, available_at);
	CREATE INDEX IF NOT EXISTS idx_messages_uuid ON messages(uuid);

	CREATE TABLE IF NOT EXISTS dead_letter_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT NOT NULL,
		original_topic TEXT NOT NULL,
		payload BLOB NOT NULL,
		metadata TEXT,
		error_message TEXT,
		failed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		retry_count INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_dlq_topic ON dead_letter_queue(original_topic);
	`
	_, err := t.db.Exec(schema)
	return err
}

// Publish publishes a message to the specified topic.
func (t *SQLiteTransport) Publish(topic string, messages ...*message.Message) error {
	t.closedMu.RLock()
	if t.closed {
		t.closedMu.RUnlock()
		return fmt.Errorf("transport is closed")
	}
	t.closedMu.RUnlock()

	tx, err := t.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO messages (uuid, topic, payload, metadata, available_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, msg := range messages {
		metadata, err := json.Marshal(msg.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		availableAt := time.Now().UTC()
		// Support delayed messages via metadata
		if delayStr := msg.Metadata.Get("protoflow_delay"); delayStr != "" {
			if delay, err := time.ParseDuration(delayStr); err == nil {
				availableAt = availableAt.Add(delay)
			}
		}

		_, err = stmt.Exec(msg.UUID, topic, msg.Payload, string(metadata), availableAt)
		if err != nil {
			return fmt.Errorf("failed to insert message: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Subscribe subscribes to messages from the specified topic.
func (t *SQLiteTransport) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	t.closedMu.RLock()
	if t.closed {
		t.closedMu.RUnlock()
		return nil, fmt.Errorf("transport is closed")
	}
	t.closedMu.RUnlock()

	msgChan := make(chan *message.Message)

	t.subMu.Lock()
	t.subscriptions[topic] = msgChan
	t.subMu.Unlock()

	t.wg.Add(1)
	go t.pollMessages(ctx, topic, msgChan)

	return msgChan, nil
}

func (t *SQLiteTransport) pollMessages(ctx context.Context, topic string, msgChan chan *message.Message) {
	defer t.wg.Done()
	defer close(msgChan)

	ticker := time.NewTicker(t.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.closedChan:
			return
		case <-ticker.C:
			t.processAvailableMessages(ctx, topic, msgChan)
		}
	}
}

func (t *SQLiteTransport) processAvailableMessages(ctx context.Context, topic string, msgChan chan *message.Message) {
	// Lock and fetch a message
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("failed to begin transaction", err, nil)
		}
		return
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	lockUntil := now.Add(30 * time.Second)

	row := tx.QueryRowContext(ctx, `
		SELECT id, uuid, payload, metadata
		FROM messages
		WHERE topic = ?
		AND status = 'pending'
		AND available_at <= ?
		AND (locked_until IS NULL OR locked_until < ?)
		ORDER BY available_at ASC
		LIMIT 1
	`, topic, now, now)

	var id int64
	var uuid string
	var payload []byte
	var metadataStr string

	if err := row.Scan(&id, &uuid, &payload, &metadataStr); err != nil {
		if err == sql.ErrNoRows {
			return // No messages available
		}
		if t.logger != nil {
			t.logger.Error("failed to scan message", err, nil)
		}
		return
	}

	// Lock the message
	_, err = tx.ExecContext(ctx, `
		UPDATE messages SET locked_until = ? WHERE id = ?
	`, lockUntil, id)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("failed to lock message", err, nil)
		}
		return
	}

	if err := tx.Commit(); err != nil {
		if t.logger != nil {
			t.logger.Error("failed to commit lock", err, nil)
		}
		return
	}

	// Parse metadata
	metadata := make(message.Metadata)
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			if t.logger != nil {
				t.logger.Error("failed to unmarshal metadata", err, nil)
			}
		}
	}

	msg := message.NewMessage(uuid, payload)
	msg.Metadata = metadata

	// Send message and wait for ack/nack
	select {
	case msgChan <- msg:
		// Wait for ack/nack from consumer
		select {
		case <-msg.Acked():
			t.ackMessage(id)
		case <-msg.Nacked():
			t.nackMessage(id, topic)
		case <-ctx.Done():
			// Context cancelled, unlock the message for retry
			t.unlockMessage(id)
		case <-t.closedChan:
			t.unlockMessage(id)
		}
	case <-ctx.Done():
		t.unlockMessage(id)
	case <-t.closedChan:
		t.unlockMessage(id)
	}
}

func (t *SQLiteTransport) ackMessage(id int64) {
	_, err := t.db.Exec(`DELETE FROM messages WHERE id = ?`, id)
	if err != nil && t.logger != nil {
		t.logger.Error("failed to ack message", err, nil)
	}
}

func (t *SQLiteTransport) nackMessage(id int64, topic string) {
	// Check retry count and potentially move to DLQ
	var retryCount int
	err := t.db.QueryRow(`SELECT retry_count FROM messages WHERE id = ?`, id).Scan(&retryCount)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("failed to get retry count", err, nil)
		}
		return
	}

	if retryCount >= t.config.MaxRetries {
		// Move to dead letter queue
		_, err = t.db.Exec(`
			INSERT INTO dead_letter_queue (uuid, original_topic, payload, metadata, error_message, retry_count)
			SELECT uuid, topic, payload, metadata, 'max retries exceeded', retry_count
			FROM messages WHERE id = ?
		`, id)
		if err != nil && t.logger != nil {
			t.logger.Error("failed to move message to DLQ", err, nil)
		}

		// Delete from main queue
		_, err = t.db.Exec(`DELETE FROM messages WHERE id = ?`, id)
		if err != nil && t.logger != nil {
			t.logger.Error("failed to delete message after DLQ move", err, nil)
		}
	} else {
		// Increment retry count and unlock for retry with backoff
		backoffSeconds := 1 * (retryCount + 1) // Linear backoff: 1s, 2s, 3s...
		availableAt := time.Now().UTC().Add(time.Duration(backoffSeconds) * time.Second)
		_, err = t.db.Exec(`
			UPDATE messages
			SET retry_count = retry_count + 1,
			    locked_until = NULL,
			    available_at = ?
			WHERE id = ?
		`, availableAt, id)
		if err != nil && t.logger != nil {
			t.logger.Error("failed to nack message", err, nil)
		}
	}
}

func (t *SQLiteTransport) unlockMessage(id int64) {
	_, err := t.db.Exec(`UPDATE messages SET locked_until = NULL WHERE id = ?`, id)
	if err != nil && t.logger != nil {
		t.logger.Error("failed to unlock message", err, nil)
	}
}

// Close closes the transport and releases resources.
func (t *SQLiteTransport) Close() error {
	t.closedMu.Lock()
	if t.closed {
		t.closedMu.Unlock()
		return nil
	}
	t.closed = true
	close(t.closedChan)
	t.closedMu.Unlock()

	t.wg.Wait()

	t.subMu.Lock()
	for _, ch := range t.subscriptions {
		// Channels are closed by pollMessages goroutine
		_ = ch
	}
	t.subscriptions = nil
	t.subMu.Unlock()

	return t.db.Close()
}

// GetDB returns the underlying database connection for advanced use cases.
func (t *SQLiteTransport) GetDB() *sql.DB {
	return t.db
}

// GetPendingCount returns the number of pending messages for a topic.
func (t *SQLiteTransport) GetPendingCount(topic string) (int64, error) {
	var count int64
	err := t.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE topic = ? AND status = 'pending'
	`, topic).Scan(&count)
	return count, err
}

// GetDLQCount returns the number of messages in the dead letter queue for a topic.
func (t *SQLiteTransport) GetDLQCount(topic string) (int64, error) {
	var count int64
	err := t.db.QueryRow(`
		SELECT COUNT(*) FROM dead_letter_queue
		WHERE original_topic = ?
	`, topic).Scan(&count)
	return count, err
}

// ReplayDLQMessage moves a message from DLQ back to the main queue.
func (t *SQLiteTransport) ReplayDLQMessage(dlqID int64) error {
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO messages (uuid, topic, payload, metadata, retry_count)
		SELECT uuid || '-replay-' || ?, original_topic, payload, metadata, 0
		FROM dead_letter_queue WHERE id = ?
	`, time.Now().UnixNano(), dlqID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM dead_letter_queue WHERE id = ?`, dlqID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ReplayAllDLQ moves all messages from DLQ back to the main queue for a topic.
func (t *SQLiteTransport) ReplayAllDLQ(topic string) (int64, error) {
	tx, err := t.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO messages (uuid, topic, payload, metadata, retry_count)
		SELECT uuid || '-replay-' || ?, original_topic, payload, metadata, 0
		FROM dead_letter_queue WHERE original_topic = ?
	`, time.Now().UnixNano(), topic)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()

	_, err = tx.Exec(`DELETE FROM dead_letter_queue WHERE original_topic = ?`, topic)
	if err != nil {
		return 0, err
	}

	return affected, tx.Commit()
}

// PurgeDLQ removes all messages from the dead letter queue for a topic.
func (t *SQLiteTransport) PurgeDLQ(topic string) (int64, error) {
	result, err := t.db.Exec(`DELETE FROM dead_letter_queue WHERE original_topic = ?`, topic)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ListDLQMessages returns messages from the dead letter queue with pagination.
func (t *SQLiteTransport) ListDLQMessages(topic string, limit, offset int) ([]DLQMessage, error) {
	rows, err := t.db.Query(`
		SELECT id, uuid, original_topic, payload, metadata, error_message, failed_at, retry_count
		FROM dead_letter_queue
		WHERE original_topic = ?
		ORDER BY failed_at DESC
		LIMIT ? OFFSET ?
	`, topic, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []DLQMessage
	for rows.Next() {
		var msg DLQMessage
		var metadataStr string
		if err := rows.Scan(&msg.ID, &msg.UUID, &msg.OriginalTopic, &msg.Payload, &metadataStr, &msg.ErrorMessage, &msg.FailedAt, &msg.RetryCount); err != nil {
			return nil, err
		}
		if metadataStr != "" {
			json.Unmarshal([]byte(metadataStr), &msg.Metadata)
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// DLQMessage represents a message in the dead letter queue.
type DLQMessage struct {
	ID            int64             `json:"id"`
	UUID          string            `json:"uuid"`
	OriginalTopic string            `json:"original_topic"`
	Payload       []byte            `json:"payload"`
	Metadata      map[string]string `json:"metadata"`
	ErrorMessage  string            `json:"error_message"`
	FailedAt      time.Time         `json:"failed_at"`
	RetryCount    int               `json:"retry_count"`
}

// sqliteTransport builds a SQLite transport from config.
func sqliteTransport(conf *config.Config, logger watermill.LoggerAdapter) (Transport, error) {
	cfg := SQLiteConfig{
		FilePath: conf.SQLiteFile,
	}

	transport, err := NewSQLiteTransport(cfg, logger)
	if err != nil {
		return Transport{}, err
	}

	return Transport{
		Publisher:  transport,
		Subscriber: transport,
	}, nil
}
