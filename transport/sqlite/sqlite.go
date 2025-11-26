// Package sqlite provides a SQLite-based transport for protoflow.
package sqlite

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

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "sqlite"

const (
	// DefaultPollInterval is the default interval for polling new messages.
	DefaultPollInterval = 100 * time.Millisecond
	// DefaultMaxRetries is the default number of retries before moving to DLQ.
	DefaultMaxRetries = 3
)

func init() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.SQLiteCapabilities)
}

// Build creates a new SQLite transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	config := Config{
		FilePath: cfg.GetSQLiteFile(),
	}

	t, err := New(config, logger)
	if err != nil {
		return transport.Transport{}, err
	}

	return transport.Transport{
		Publisher:  t,
		Subscriber: t,
	}, nil
}

// Capabilities returns the capabilities of this transport.
func Capabilities() transport.Capabilities {
	return transport.SQLiteCapabilities
}

// Config holds SQLite-specific configuration.
type Config struct {
	// FilePath is the path to the SQLite database file.
	// Use ":memory:" for an in-memory database (useful for testing).
	FilePath string
	// PollInterval is the interval for polling new messages.
	PollInterval time.Duration
	// MaxRetries is the number of times to retry a message before giving up.
	MaxRetries int
}

func (c Config) withDefaults() Config {
	if c.FilePath == "" {
		c.FilePath = "protoflow_queue.db"
	}
	if c.PollInterval <= 0 {
		c.PollInterval = DefaultPollInterval
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = DefaultMaxRetries
	}
	return c
}

// Transport implements both Publisher and Subscriber interfaces for SQLite.
type Transport struct {
	db     *sql.DB
	config Config
	logger watermill.LoggerAdapter

	subscriptions map[string]chan *message.Message
	subMu         sync.RWMutex

	closed     bool
	closedMu   sync.RWMutex
	closedChan chan struct{}
	wg         sync.WaitGroup
}

// New creates a new SQLite-based transport.
func New(cfg Config, logger watermill.LoggerAdapter) (*Transport, error) {
	cfg = cfg.withDefaults()

	db, err := sql.Open("sqlite3", cfg.FilePath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	t := &Transport{
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

func (t *Transport) initSchema() error {
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
func (t *Transport) Publish(topic string, messages ...*message.Message) error {
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
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			if t.logger != nil {
				t.logger.Error("failed to rollback transaction", err, nil)
			}
		}
	}()

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
func (t *Transport) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
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

func (t *Transport) pollMessages(ctx context.Context, topic string, msgChan chan *message.Message) {
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

type fetchedMessage struct {
	id       int64
	uuid     string
	payload  []byte
	metadata string
}

func (t *Transport) fetchAndLockMessage(ctx context.Context, topic string) (*fetchedMessage, bool) {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("failed to begin transaction", err, nil)
		}
		return nil, false
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			if t.logger != nil {
				t.logger.Error("failed to rollback transaction", err, nil)
			}
		}
	}()

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

	var fm fetchedMessage
	if err := row.Scan(&fm.id, &fm.uuid, &fm.payload, &fm.metadata); err != nil {
		if err != sql.ErrNoRows && t.logger != nil {
			t.logger.Error("failed to scan message", err, nil)
		}
		return nil, false
	}

	if _, err = tx.ExecContext(ctx, `UPDATE messages SET locked_until = ? WHERE id = ?`, lockUntil, fm.id); err != nil {
		if t.logger != nil {
			t.logger.Error("failed to lock message", err, nil)
		}
		return nil, false
	}

	if err := tx.Commit(); err != nil {
		if t.logger != nil {
			t.logger.Error("failed to commit lock", err, nil)
		}
		return nil, false
	}

	return &fm, true
}

func (t *Transport) handleMessageResult(ctx context.Context, id int64, topic string, msg *message.Message) {
	select {
	case <-msg.Acked():
		t.ackMessage(id)
	case <-msg.Nacked():
		t.nackMessage(id, topic)
	case <-ctx.Done():
		t.unlockMessage(id)
	case <-t.closedChan:
		t.unlockMessage(id)
	}
}

func (t *Transport) processAvailableMessages(ctx context.Context, topic string, msgChan chan *message.Message) {
	fm, found := t.fetchAndLockMessage(ctx, topic)
	if !found {
		return
	}

	metadata := make(message.Metadata)
	if fm.metadata != "" {
		if err := json.Unmarshal([]byte(fm.metadata), &metadata); err != nil && t.logger != nil {
			t.logger.Error("failed to unmarshal metadata", err, nil)
		}
	}

	msg := message.NewMessage(fm.uuid, fm.payload)
	msg.Metadata = metadata

	select {
	case msgChan <- msg:
		t.handleMessageResult(ctx, fm.id, topic, msg)
	case <-ctx.Done():
		t.unlockMessage(fm.id)
	case <-t.closedChan:
		t.unlockMessage(fm.id)
	}
}

func (t *Transport) ackMessage(id int64) {
	_, err := t.db.Exec(`DELETE FROM messages WHERE id = ?`, id)
	if err != nil && t.logger != nil {
		t.logger.Error("failed to ack message", err, nil)
	}
}

func (t *Transport) nackMessage(id int64, topic string) {
	var retryCount int
	err := t.db.QueryRow(`SELECT retry_count FROM messages WHERE id = ?`, id).Scan(&retryCount)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("failed to get retry count", err, nil)
		}
		return
	}

	if retryCount >= t.config.MaxRetries {
		_, err = t.db.Exec(`
			INSERT INTO dead_letter_queue (uuid, original_topic, payload, metadata, error_message, retry_count)
			SELECT uuid, topic, payload, metadata, 'max retries exceeded', retry_count
			FROM messages WHERE id = ?
		`, id)
		if err != nil && t.logger != nil {
			t.logger.Error("failed to move message to DLQ", err, nil)
		}

		_, err = t.db.Exec(`DELETE FROM messages WHERE id = ?`, id)
		if err != nil && t.logger != nil {
			t.logger.Error("failed to delete message after DLQ move", err, nil)
		}
	} else {
		backoffSeconds := 1 * (retryCount + 1)
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

func (t *Transport) unlockMessage(id int64) {
	_, err := t.db.Exec(`UPDATE messages SET locked_until = NULL WHERE id = ?`, id)
	if err != nil && t.logger != nil {
		t.logger.Error("failed to unlock message", err, nil)
	}
}

// Close closes the transport and releases resources.
func (t *Transport) Close() error {
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
	t.subscriptions = nil
	t.subMu.Unlock()

	return t.db.Close()
}

// GetCapabilities returns the capabilities of this transport instance.
func (t *Transport) GetCapabilities() transport.Capabilities {
	return transport.SQLiteCapabilities
}

// GetDB returns the underlying database connection for advanced use cases.
func (t *Transport) GetDB() *sql.DB {
	return t.db
}

// GetPendingCount returns the number of pending messages for a topic.
func (t *Transport) GetPendingCount(topic string) (int64, error) {
	var count int64
	err := t.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE topic = ? AND status = 'pending'
	`, topic).Scan(&count)
	return count, err
}

// GetDLQCount returns the number of messages in the dead letter queue for a topic.
func (t *Transport) GetDLQCount(topic string) (int64, error) {
	var count int64
	err := t.db.QueryRow(`
		SELECT COUNT(*) FROM dead_letter_queue
		WHERE original_topic = ?
	`, topic).Scan(&count)
	return count, err
}

// ReplayDLQMessage moves a message from DLQ back to the main queue.
func (t *Transport) ReplayDLQMessage(dlqID int64) error {
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			if t.logger != nil {
				t.logger.Error("failed to rollback transaction", err, nil)
			}
		}
	}()

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
func (t *Transport) ReplayAllDLQ(topic string) (int64, error) {
	tx, err := t.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			if t.logger != nil {
				t.logger.Error("failed to rollback transaction", err, nil)
			}
		}
	}()

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
func (t *Transport) PurgeDLQ(topic string) (int64, error) {
	result, err := t.db.Exec(`DELETE FROM dead_letter_queue WHERE original_topic = ?`, topic)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ListDLQMessages returns messages from the dead letter queue with pagination.
func (t *Transport) ListDLQMessages(topic string, limit, offset int) ([]transport.DLQMessage, error) {
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

	var messages []transport.DLQMessage
	for rows.Next() {
		var msg transport.DLQMessage
		var metadataStr string
		if err := rows.Scan(&msg.ID, &msg.UUID, &msg.OriginalTopic, &msg.Payload, &metadataStr, &msg.ErrorMessage, &msg.FailedAt, &msg.RetryCount); err != nil {
			return nil, err
		}
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &msg.Metadata); err != nil {
				if t.logger != nil {
					t.logger.Error("failed to unmarshal metadata", err, nil)
				}
			}
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}
