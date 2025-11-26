package transport

// Capabilities describes the features supported by a transport backend.
// Use this to introspect what operations are available at runtime.
type Capabilities struct {
	// SupportsDelay indicates the transport can natively delay message delivery.
	// When false, delayed delivery must be emulated by the application.
	SupportsDelay bool

	// SupportsNativeDLQ indicates the transport has built-in dead letter queue support.
	// When false, protoflow will handle DLQ routing at the application level.
	SupportsNativeDLQ bool

	// SupportsOrdering indicates the transport guarantees message ordering.
	// When true, messages within a partition/stream are delivered in order.
	SupportsOrdering bool

	// SupportsTracing indicates the transport propagates tracing headers natively.
	SupportsTracing bool

	// SupportsBatching indicates the transport can batch multiple messages.
	SupportsBatching bool

	// SupportsAck indicates the transport supports explicit message acknowledgment.
	SupportsAck bool

	// SupportsNack indicates the transport supports negative acknowledgment (redelivery).
	SupportsNack bool

	// SupportsPriority indicates the transport supports message priority queues.
	SupportsPriority bool

	// SupportsPartitioning indicates the transport supports message partitioning.
	SupportsPartitioning bool

	// MaxMessageSize is the maximum message size in bytes (0 = unlimited/unknown).
	MaxMessageSize int64

	// MaxDelayDuration is the maximum delay duration supported (0 = unlimited/unknown).
	MaxDelayDuration int64

	// Name is the human-readable name of the transport.
	Name string

	// Version is the transport/driver version.
	Version string
}

// RequiresDelayEmulation returns true if the transport needs application-level
// delay handling because it doesn't support native delayed delivery.
func (c Capabilities) RequiresDelayEmulation() bool {
	return !c.SupportsDelay
}

// RequiresDLQEmulation returns true if the transport needs application-level
// DLQ routing because it doesn't support native dead letter queues.
func (c Capabilities) RequiresDLQEmulation() bool {
	return !c.SupportsNativeDLQ
}

// SupportsReliableDelivery returns true if the transport supports at-least-once
// delivery semantics (ack + nack).
func (c Capabilities) SupportsReliableDelivery() bool {
	return c.SupportsAck && c.SupportsNack
}

// Predefined capability sets for common transports.
var (
	// ChannelCapabilities for in-memory Go channel transport.
	ChannelCapabilities = Capabilities{
		Name:              "channel",
		SupportsDelay:     false,
		SupportsNativeDLQ: false,
		SupportsOrdering:  true,
		SupportsTracing:   false,
		SupportsBatching:  false,
		SupportsAck:       true,
		SupportsNack:      true,
		SupportsPriority:  false,
	}

	// KafkaCapabilities for Apache Kafka transport.
	KafkaCapabilities = Capabilities{
		Name:                 "kafka",
		SupportsDelay:        false,
		SupportsNativeDLQ:    false,
		SupportsOrdering:     true,
		SupportsTracing:      true,
		SupportsBatching:     true,
		SupportsAck:          true,
		SupportsNack:         false,
		SupportsPriority:     false,
		SupportsPartitioning: true,
		MaxMessageSize:       1048576, // Default 1MB
	}

	// RabbitMQCapabilities for RabbitMQ/AMQP transport.
	RabbitMQCapabilities = Capabilities{
		Name:              "rabbitmq",
		SupportsDelay:     true,
		SupportsNativeDLQ: true,
		SupportsOrdering:  true,
		SupportsTracing:   true,
		SupportsBatching:  false,
		SupportsAck:       true,
		SupportsNack:      true,
		SupportsPriority:  true,
	}

	// NATSCapabilities for NATS Core transport.
	NATSCapabilities = Capabilities{
		Name:              "nats",
		SupportsDelay:     false,
		SupportsNativeDLQ: false,
		SupportsOrdering:  false,
		SupportsTracing:   true,
		SupportsBatching:  false,
		SupportsAck:       false,
		SupportsNack:      false,
		SupportsPriority:  false,
		MaxMessageSize:    1048576, // Default 1MB
	}

	// NATSJetStreamCapabilities for NATS JetStream transport.
	NATSJetStreamCapabilities = Capabilities{
		Name:              "nats-jetstream",
		SupportsDelay:     true,
		SupportsNativeDLQ: true,
		SupportsOrdering:  true,
		SupportsTracing:   true,
		SupportsBatching:  true,
		SupportsAck:       true,
		SupportsNack:      true,
		SupportsPriority:  false,
		MaxMessageSize:    1048576, // Default 1MB
	}

	// AWSCapabilities for AWS SNS/SQS transport.
	AWSCapabilities = Capabilities{
		Name:              "aws",
		SupportsDelay:     true,
		SupportsNativeDLQ: true,
		SupportsOrdering:  true,
		SupportsTracing:   true,
		SupportsBatching:  true,
		SupportsAck:       true,
		SupportsNack:      true,
		SupportsPriority:  false,
		MaxMessageSize:    262144, // 256KB
		MaxDelayDuration:  900000, // 15 minutes in ms
	}

	// SQLiteCapabilities for SQLite-based transport.
	SQLiteCapabilities = Capabilities{
		Name:              "sqlite",
		SupportsDelay:     true,
		SupportsNativeDLQ: true,
		SupportsOrdering:  true,
		SupportsTracing:   false,
		SupportsBatching:  true,
		SupportsAck:       true,
		SupportsNack:      true,
		SupportsPriority:  false,
	}

	// PostgresCapabilities for PostgreSQL-based transport.
	PostgresCapabilities = Capabilities{
		Name:              "postgres",
		SupportsDelay:     true,
		SupportsNativeDLQ: true,
		SupportsOrdering:  true,
		SupportsTracing:   false,
		SupportsBatching:  true,
		SupportsAck:       true,
		SupportsNack:      true,
		SupportsPriority:  true,
	}

	// HTTPCapabilities for HTTP-based transport.
	HTTPCapabilities = Capabilities{
		Name:              "http",
		SupportsDelay:     false,
		SupportsNativeDLQ: false,
		SupportsOrdering:  false,
		SupportsTracing:   true,
		SupportsBatching:  false,
		SupportsAck:       false,
		SupportsNack:      false,
		SupportsPriority:  false,
	}

	// IOCapabilities for file-based I/O transport.
	IOCapabilities = Capabilities{
		Name:              "io",
		SupportsDelay:     false,
		SupportsNativeDLQ: false,
		SupportsOrdering:  true,
		SupportsTracing:   false,
		SupportsBatching:  false,
		SupportsAck:       false,
		SupportsNack:      false,
		SupportsPriority:  false,
	}
)

// GetCapabilities returns the capabilities for a transport by name.
// Uses the registry to look up capabilities registered by each transport package.
// Returns a zero Capabilities struct if the transport is unknown.
func GetCapabilities(transportName string) Capabilities {
	return DefaultRegistry.GetCapabilities(transportName)
}
