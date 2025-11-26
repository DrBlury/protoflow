// Package postgres provides a PostgreSQL-based transport for protoflow.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "postgres"

const (
	// DefaultPollInterval is the default interval for polling new messages.
	DefaultPollInterval = 100 * time.Millisecond
	// DefaultMaxRetries is the default number of retries before moving to DLQ.
	DefaultMaxRetries = 3
	// DefaultLockTimeout is the default duration a message is locked during processing.
	DefaultLockTimeout = 30 * time.Second
)

func init() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.PostgresCapabilities)
	transport.RegisterWithCapabilities("postgresql", Build, transport.PostgresCapabilities) // Alias
}

// Build creates a new PostgreSQL transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	config := Config{
		ConnectionString: cfg.GetPostgresURL(),
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
	return transport.PostgresCapabilities
}

// Config holds PostgreSQL-specific configuration.
type Config struct {
	// ConnectionString is the PostgreSQL connection string.
	ConnectionString string
	// PollInterval is the interval for polling new messages.
	PollInterval time.Duration
	// MaxRetries is the number of times to retry a message before moving to DLQ.
	MaxRetries int
	// LockTimeout is how long a message stays locked during processing.
	LockTimeout time.Duration
	// SchemaName is the schema to use for tables. Defaults to "protoflow".
	SchemaName string
	// MaxOpenConns sets the maximum number of open connections to the database.
	MaxOpenConns int
	// MaxIdleConns sets the maximum number of idle connections.
	MaxIdleConns int
}

func (c Config) withDefaults() Config {
	if c.PollInterval <= 0 {
		c.PollInterval = DefaultPollInterval
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = DefaultMaxRetries
	}
	if c.LockTimeout <= 0 {
		c.LockTimeout = DefaultLockTimeout
	}
	if c.SchemaName == "" {
		c.SchemaName = "protoflow"
	}
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = 10
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 5
	}
	return c
}

// Transport implements both Publisher and Subscriber interfaces for PostgreSQL.
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

// New creates a new PostgreSQL-based transport.
func New(cfg Config, logger watermill.LoggerAdapter) (*Transport, error) {
	if cfg.ConnectionString == "" {
		return nil, fmt.Errorf("PostgreSQL connection string is required")
	}

	cfg = cfg.withDefaults()

	db, err := sql.Open("postgres", cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

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
	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	_, err := t.db.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, t.config.SchemaName))
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	schema := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %[1]s.messages (
		id BIGSERIAL PRIMARY KEY,
		uuid TEXT NOT NULL UNIQUE,
		topic TEXT NOT NULL,
		payload BYTEA NOT NULL,
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		available_at TIMESTAMPTZ DEFAULT NOW(),
		locked_until TIMESTAMPTZ,
		retry_count INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending'
	);

	CREATE INDEX IF NOT EXISTS idx_messages_topic_status_available
		ON %[1]s.messages(topic, status, available_at)
		WHERE status = 'pending';

	CREATE INDEX IF NOT EXISTS idx_messages_uuid ON %[1]s.messages(uuid);
	CREATE INDEX IF NOT EXISTS idx_messages_locked_until ON %[1]s.messages(locked_until)
		WHERE locked_until IS NOT NULL;

	CREATE TABLE IF NOT EXISTS %[1]s.dead_letter_queue (
		id BIGSERIAL PRIMARY KEY,
		uuid TEXT NOT NULL,
		original_topic TEXT NOT NULL,
		payload BYTEA NOT NULL,
		metadata JSONB DEFAULT '{}',
		error_message TEXT,
		failed_at TIMESTAMPTZ DEFAULT NOW(),
		retry_count INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_dlq_topic ON %[1]s.dead_letter_queue(original_topic);
	CREATE INDEX IF NOT EXISTS idx_dlq_failed_at ON %[1]s.dead_letter_queue(failed_at);
	`, t.config.SchemaName)

	_, err = t.db.Exec(schema)
	return err
}

// Publish publishes messages to the specified topic.
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

	stmt, err := tx.Prepare(fmt.Sprintf(`
		INSERT INTO %s.messages (uuid, topic, payload, metadata, available_at)
		VALUES ($1, $2, $3, $4, $5)
	`, t.config.SchemaName))
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

		_, err = stmt.Exec(msg.UUID, topic, msg.Payload, metadata, availableAt)
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

func (t *Transport) fetchAndLockMessage(ctx context.Context, topic string) (int64, *message.Message, bool) {
	now := time.Now().UTC()
	lockUntil := now.Add(t.config.LockTimeout)

	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`
		UPDATE %[1]s.messages
		SET locked_until = $1
		WHERE id = (
			SELECT id FROM %[1]s.messages
			WHERE topic = $2
			AND status = 'pending'
			AND available_at <= $3
			AND (locked_until IS NULL OR locked_until < $3)
			ORDER BY available_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, uuid, payload, metadata
	`, t.config.SchemaName)

	var id int64
	var uuid string
	var payload []byte
	var metadataJSON []byte

	err := t.db.QueryRowContext(ctx, query, lockUntil, topic, now).Scan(&id, &uuid, &payload, &metadataJSON)
	if err != nil {
		if err != sql.ErrNoRows && t.logger != nil {
			t.logger.Error("failed to fetch and lock message", err, nil)
		}
		return 0, nil, false
	}

	metadata := make(message.Metadata)
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil && t.logger != nil {
			t.logger.Error("failed to unmarshal metadata", err, nil)
		}
	}

	msg := message.NewMessage(uuid, payload)
	msg.Metadata = metadata
	return id, msg, true
}

func (t *Transport) handleMessageResult(ctx context.Context, id int64, topic string, msg *message.Message) {
	select {
	case <-msg.Acked():
		t.ackMessage(ctx, id)
	case <-msg.Nacked():
		t.nackMessage(ctx, id, topic)
	case <-ctx.Done():
		t.unlockMessage(ctx, id)
	case <-t.closedChan:
		t.unlockMessage(ctx, id)
	}
}

func (t *Transport) processAvailableMessages(ctx context.Context, topic string, msgChan chan *message.Message) {
	id, msg, found := t.fetchAndLockMessage(ctx, topic)
	if !found {
		return
	}

	select {
	case msgChan <- msg:
		t.handleMessageResult(ctx, id, topic, msg)
	case <-ctx.Done():
		t.unlockMessage(ctx, id)
	case <-t.closedChan:
		t.unlockMessage(ctx, id)
	}
}

func (t *Transport) ackMessage(ctx context.Context, id int64) {
	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`DELETE FROM %s.messages WHERE id = $1`, t.config.SchemaName)
	_, err := t.db.ExecContext(ctx, query, id)
	if err != nil && t.logger != nil {
		t.logger.Error("failed to ack message", err, nil)
	}
}

func (t *Transport) nackMessage(ctx context.Context, id int64, topic string) {
	var retryCount int
	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`SELECT retry_count FROM %s.messages WHERE id = $1`, t.config.SchemaName)
	err := t.db.QueryRowContext(ctx, query, id).Scan(&retryCount)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("failed to get retry count", err, nil)
		}
		return
	}

	if retryCount >= t.config.MaxRetries {
		// #nosec G201 - schema name is validated/sanitized via withDefaults()
		moveToDLQ := fmt.Sprintf(`
			WITH moved AS (
				DELETE FROM %[1]s.messages WHERE id = $1
				RETURNING uuid, topic, payload, metadata, retry_count
			)
			INSERT INTO %[1]s.dead_letter_queue (uuid, original_topic, payload, metadata, error_message, retry_count)
			SELECT uuid, topic, payload, metadata, 'max retries exceeded', retry_count FROM moved
		`, t.config.SchemaName)

		_, err = t.db.ExecContext(ctx, moveToDLQ, id)
		if err != nil && t.logger != nil {
			t.logger.Error("failed to move message to DLQ", err, nil)
		}
	} else {
		backoffSeconds := 1 << retryCount
		availableAt := time.Now().UTC().Add(time.Duration(backoffSeconds) * time.Second)
		// #nosec G201 - schema name is validated/sanitized via withDefaults()
		query := fmt.Sprintf(`
			UPDATE %s.messages
			SET retry_count = retry_count + 1,
			    locked_until = NULL,
			    available_at = $1
			WHERE id = $2
		`, t.config.SchemaName)
		_, err = t.db.ExecContext(ctx, query, availableAt, id)
		if err != nil && t.logger != nil {
			t.logger.Error("failed to nack message", err, nil)
		}
	}
}

func (t *Transport) unlockMessage(ctx context.Context, id int64) {
	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`UPDATE %s.messages SET locked_until = NULL WHERE id = $1`, t.config.SchemaName)
	_, err := t.db.ExecContext(ctx, query, id)
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
	return transport.PostgresCapabilities
}

// GetDB returns the underlying database connection for advanced use cases.
func (t *Transport) GetDB() *sql.DB {
	return t.db
}

// GetPendingCount returns the number of pending messages for a topic.
func (t *Transport) GetPendingCount(topic string) (int64, error) {
	var count int64
	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s.messages
		WHERE topic = $1 AND status = 'pending'
	`, t.config.SchemaName)
	err := t.db.QueryRow(query, topic).Scan(&count)
	return count, err
}

// GetDLQCount returns the number of messages in the dead letter queue for a topic.
func (t *Transport) GetDLQCount(topic string) (int64, error) {
	var count int64
	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s.dead_letter_queue
		WHERE original_topic = $1
	`, t.config.SchemaName)
	err := t.db.QueryRow(query, topic).Scan(&count)
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

	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`
		WITH replayed AS (
			DELETE FROM %[1]s.dead_letter_queue WHERE id = $1
			RETURNING uuid, original_topic, payload, metadata
		)
		INSERT INTO %[1]s.messages (uuid, topic, payload, metadata, retry_count)
		SELECT uuid || '-replay-' || extract(epoch from now())::bigint, original_topic, payload, metadata, 0
		FROM replayed
	`, t.config.SchemaName)

	result, err := tx.Exec(query, dlqID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("DLQ message with id %d not found", dlqID)
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

	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`
		WITH replayed AS (
			DELETE FROM %[1]s.dead_letter_queue WHERE original_topic = $1
			RETURNING uuid, original_topic, payload, metadata
		)
		INSERT INTO %[1]s.messages (uuid, topic, payload, metadata, retry_count)
		SELECT uuid || '-replay-' || extract(epoch from now())::bigint || '-' || row_number() OVER (), original_topic, payload, metadata, 0
		FROM replayed
	`, t.config.SchemaName)

	result, err := tx.Exec(query, topic)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()
	return affected, tx.Commit()
}

// PurgeDLQ removes all messages from the dead letter queue for a topic.
func (t *Transport) PurgeDLQ(topic string) (int64, error) {
	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`DELETE FROM %s.dead_letter_queue WHERE original_topic = $1`, t.config.SchemaName)
	result, err := t.db.Exec(query, topic)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ListDLQMessages returns messages from the dead letter queue with pagination.
func (t *Transport) ListDLQMessages(topic string, limit, offset int) ([]transport.DLQMessage, error) {
	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`
		SELECT id, uuid, original_topic, payload, metadata, error_message, failed_at, retry_count
		FROM %s.dead_letter_queue
		WHERE original_topic = $1
		ORDER BY failed_at DESC
		LIMIT $2 OFFSET $3
	`, t.config.SchemaName)

	rows, err := t.db.Query(query, topic, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []transport.DLQMessage
	for rows.Next() {
		var msg transport.DLQMessage
		var metadataJSON []byte
		if err := rows.Scan(&msg.ID, &msg.UUID, &msg.OriginalTopic, &msg.Payload, &metadataJSON, &msg.ErrorMessage, &msg.FailedAt, &msg.RetryCount); err != nil {
			return nil, err
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &msg.Metadata); err != nil {
				if t.logger != nil {
					t.logger.Error("failed to unmarshal metadata", err, nil)
				}
			}
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// CleanupExpiredLocks unlocks messages that have been locked longer than the lock timeout.
func (t *Transport) CleanupExpiredLocks() (int64, error) {
	// #nosec G201 - schema name is validated/sanitized via withDefaults()
	query := fmt.Sprintf(`
		UPDATE %s.messages
		SET locked_until = NULL
		WHERE locked_until IS NOT NULL AND locked_until < NOW()
	`, t.config.SchemaName)
	result, err := t.db.Exec(query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// VacuumTables runs VACUUM on the message tables to reclaim space.
func (t *Transport) VacuumTables() error {
	if _, err := t.db.Exec(fmt.Sprintf(`VACUUM %s.messages`, t.config.SchemaName)); err != nil {
		return err
	}
	if _, err := t.db.Exec(fmt.Sprintf(`VACUUM %s.dead_letter_queue`, t.config.SchemaName)); err != nil {
		return err
	}
	return nil
}
