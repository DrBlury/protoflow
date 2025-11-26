// Package transport defines the core interfaces and types for protoflow transports.
// Each transport implementation (kafka, rabbitmq, aws, etc.) should be in its own
// sub-package and register itself with the transport registry.
package transport

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Transport combines a publisher and subscriber pair produced by a factory.
type Transport struct {
	Publisher  message.Publisher
	Subscriber message.Subscriber
}

// Builder is the function signature for creating a transport from config.
// Each transport package should provide a Builder function that can be registered.
type Builder func(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error)

// Config provides the configuration values needed by transports.
// This interface allows transports to access only the config they need
// without depending on the full config package.
type Config interface {
	// GetPubSubSystem returns the transport type name.
	GetPubSubSystem() string

	// Kafka
	GetKafkaBrokers() []string
	GetKafkaConsumerGroup() string

	// RabbitMQ
	GetRabbitMQURL() string

	// NATS
	GetNATSURL() string

	// HTTP
	GetHTTPServerAddress() string
	GetHTTPPublisherURL() string

	// IO
	GetIOFile() string

	// SQLite
	GetSQLiteFile() string

	// PostgreSQL
	GetPostgresURL() string

	// AWS
	GetAWSRegion() string
	GetAWSAccountID() string
	GetAWSAccessKeyID() string
	GetAWSSecretAccessKey() string
	GetAWSEndpoint() string
}

// CapabilitiesProvider is implemented by transports that can report their capabilities.
type CapabilitiesProvider interface {
	Capabilities() Capabilities
}

// DLQManager is implemented by transports that support DLQ management.
type DLQManager interface {
	GetDLQCount(topic string) (int64, error)
	ReplayDLQMessage(dlqID int64) error
	ReplayAllDLQ(topic string) (int64, error)
	PurgeDLQ(topic string) (int64, error)
}

// DLQLister is implemented by transports that can list DLQ messages.
type DLQLister interface {
	ListDLQMessages(topic string, limit, offset int) ([]DLQMessage, error)
}

// DLQMessage represents a message in the dead letter queue.
// This is used by transports that implement DLQLister.
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

// QueueIntrospector is implemented by transports that can report queue statistics.
type QueueIntrospector interface {
	GetPendingCount(topic string) (int64, error)
}

// DelayedPublisher is implemented by transports that support delayed message delivery.
type DelayedPublisher interface {
	PublishWithDelay(topic string, delay int64, messages ...*message.Message) error
}
