# Transport Capabilities

Protoflow provides a standard way for transports to declare their abilities, enabling runtime introspection and conditional behavior.

## Capabilities Structure

```go
type Capabilities struct {
    // Feature support
    SupportsDelay        bool  // Native delayed delivery
    SupportsNativeDLQ    bool  // Built-in dead letter queue
    SupportsOrdering     bool  // Message ordering guarantees
    SupportsTracing      bool  // Native trace context propagation
    SupportsBatching     bool  // Batch publish support
    SupportsAck          bool  // Explicit acknowledgment
    SupportsNack         bool  // Negative acknowledgment (redelivery)
    SupportsPriority     bool  // Priority queues
    SupportsPartitioning bool  // Message partitioning

    // Limits
    MaxMessageSize   int64  // Maximum message size (bytes, 0=unknown)
    MaxDelayDuration int64  // Maximum delay (ms, 0=unlimited)

    // Metadata
    Name    string  // Transport name
    Version string  // Driver/transport version
}
```

## Transport Comparison

| Capability | Channel | Kafka | RabbitMQ | NATS | JetStream | AWS SQS | SQLite | Postgres |
|------------|:-------:|:-----:|:--------:|:----:|:---------:|:-------:|:------:|:--------:|
| Delay | ❌ | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Native DLQ | ❌ | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Ordering | ✅ | ✅¹ | ✅ | ❌ | ✅ | ✅² | ✅ | ✅ |
| Tracing | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Batching | ❌ | ✅ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Ack | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Nack | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Priority | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Partitioning | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

¹ Within partition
² FIFO queues only

## Querying Capabilities

### From Service

```go
caps := svc.GetTransportCapabilities()

if caps.SupportsDelay {
    // Use native delayed delivery
    err := svc.PublishEventAfter(ctx, evt, 5*time.Minute)
} else {
    // Schedule externally or use application-level delay
}
```

### By Transport Name

```go
caps := protoflow.GetCapabilities("nats-jetstream")

fmt.Printf("Transport: %s\n", caps.Name)
fmt.Printf("Supports Delay: %v\n", caps.SupportsDelay)
fmt.Printf("Supports DLQ: %v\n", caps.SupportsNativeDLQ)
```

## Helper Methods

### RequiresDelayEmulation

```go
if caps.RequiresDelayEmulation() {
    // Transport doesn't support native delay
    // Use application-level scheduling
}
```

### RequiresDLQEmulation

```go
if caps.RequiresDLQEmulation() {
    // Transport doesn't support native DLQ
    // Protoflow will handle DLQ routing
}
```

### SupportsReliableDelivery

```go
if caps.SupportsReliableDelivery() {
    // Transport supports at-least-once delivery
    // (both ack and nack available)
}
```

## Transport Details

### Go Channel

In-memory transport for testing and local development.

```go
cfg := &protoflow.Config{
    PubSubSystem: "channel",
}
```

- **Best for**: Unit tests, local development
- **Persistence**: None (in-memory)
- **Max message size**: Unlimited

### Apache Kafka

Distributed streaming platform with high throughput.

```go
cfg := &protoflow.Config{
    PubSubSystem:       "kafka",
    KafkaBrokers:       []string{"localhost:9092"},
    KafkaConsumerGroup: "my-group",
}
```

- **Best for**: High-throughput streaming, log aggregation
- **Ordering**: Within partition (use message key)
- **Max message size**: 1MB (default, configurable)

### RabbitMQ

Flexible message broker with advanced routing.

```go
cfg := &protoflow.Config{
    PubSubSystem: "rabbitmq",
    RabbitMQURL:  "amqp://guest:guest@localhost:5672/",
}
```

- **Best for**: Complex routing, priority queues
- **Delay**: Via delayed-message-exchange plugin or message TTL
- **DLQ**: Native via x-dead-letter-exchange

### NATS Core

Lightweight, high-performance messaging.

```go
cfg := &protoflow.Config{
    PubSubSystem: "nats",
    NATSURL:      "nats://localhost:4222",
}
```

- **Best for**: Microservices, request-reply patterns
- **Delivery**: At-most-once (fire and forget)
- **Max message size**: 1MB (default)

### NATS JetStream

Persistent streaming built on NATS.

```go
cfg := &protoflow.Config{
    PubSubSystem: "nats-jetstream",
    NATSURL:      "nats://localhost:4222",
}
```

- **Best for**: Event sourcing, reliable delivery
- **Ordering**: Within stream
- **DLQ**: Via max delivery attempts

### AWS SNS/SQS

Cloud-native pub/sub with message queuing.

```go
cfg := &protoflow.Config{
    PubSubSystem:       "aws",
    AWSRegion:          "us-east-1",
    AWSAccessKeyID:     "...",
    AWSSecretAccessKey: "...",
}
```

- **Best for**: AWS-native applications
- **Delay**: 0-15 minutes (SQS DelaySeconds)
- **DLQ**: SQS redrive policy

### SQLite

Embedded queue for simple deployments.

```go
cfg := &protoflow.Config{
    PubSubSystem: "sqlite",
    SQLiteFile:   "queue.db",
}
```

- **Best for**: Single-node apps, development, embedded systems
- **Delay**: Via `available_at` column
- **DLQ**: Separate `dead_letter_queue` table

### PostgreSQL

Production-ready transactional queue.

```go
cfg := &protoflow.Config{
    PubSubSystem: "postgres",
    PostgresURL:  "postgres://user:pass@localhost:5432/mydb",
}
```

- **Best for**: Production workloads, transactional guarantees
- **Delay**: Via `scheduled_at` column
- **Concurrency**: SKIP LOCKED for efficient workers

## Conditional Feature Usage

```go
func processOrder(ctx context.Context, svc *protoflow.Service, order Order) error {
    caps := svc.GetTransportCapabilities()

    // Create the event
    evt := protoflow.NewCloudEvent("order.created", "order-service", order)

    // Set max attempts based on transport capabilities
    if caps.SupportsNativeDLQ {
        protoflow.SetMaxAttempts(&evt, 5)
    } else {
        protoflow.SetMaxAttempts(&evt, 3) // Fewer attempts when DLQ is emulated
    }

    // Publish with delay if supported
    if order.ScheduledDelivery.After(time.Now()) && caps.SupportsDelay {
        delay := time.Until(order.ScheduledDelivery)
        return svc.PublishEventAfter(ctx, evt, delay)
    }

    return svc.PublishEvent(ctx, evt)
}
```

## Custom Transport Capabilities

When implementing a custom transport, implement the `CapabilitiesProvider` interface:

```go
type CapabilitiesProvider interface {
    Capabilities() Capabilities
}

type MyTransport struct {
    // ...
}

func (t *MyTransport) Capabilities() protoflow.Capabilities {
    return protoflow.Capabilities{
        Name:              "my-transport",
        SupportsDelay:     true,
        SupportsNativeDLQ: false,
        SupportsOrdering:  true,
        SupportsAck:       true,
        SupportsNack:      true,
        MaxMessageSize:    10 * 1024 * 1024, // 10MB
    }
}
```
