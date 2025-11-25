# Transport Comparison Guide

This guide provides a comprehensive comparison of all supported transport backends in Protoflow, including their capabilities, performance characteristics, and recommended use cases.

## Feature Comparison Matrix

| Feature | Channel | Kafka | RabbitMQ | AWS SQS | NATS | HTTP | File I/O | SQLite | PostgreSQL |
|---------|---------|-------|----------|---------|------|------|----------|--------|------------|
| **Message Persistence** | ❌ | ✅ | ✅ | ✅ | ✅* | ❌ | ✅ | ✅ | ✅ |
| **Delayed Messages** | ❌ | ❌ | ✅ | ❌** | ❌ | ❌ | ❌ | ✅ | ✅ |
| **Message Ordering** | ✅ | ✅*** | ✅ | ❌ | ✅ | N/A | ✅ | ✅ | ✅ |
| **Consumer Groups** | ❌ | ✅ | ✅ | ✅ | ✅ | N/A | ❌ | ❌ | ✅**** |
| **Dead Letter Queue** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **DLQ Inspection** | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ |
| **DLQ Replay** | ❌ | Manual | ✅ | ✅ | Manual | ❌ | ❌ | ✅ | ✅ |
| **At-Least-Once Delivery** | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ |
| **Exactly-Once (via transactions)** | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| **Horizontal Scaling** | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |
| **Cloud-Native** | ❌ | ⚪ | ⚪ | ✅ | ⚪ | ✅ | ❌ | ❌ | ✅ |
| **Embedded/No Dependencies** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ❌ |
| **Message Prioritization** | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅***** | ✅***** |
| **Queue Introspection** | ❌ | Limited | ✅ | ✅ | Limited | ❌ | ❌ | ✅ | ✅ |
| **Message TTL** | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **SKIP LOCKED Support** | N/A | N/A | N/A | N/A | N/A | N/A | N/A | ❌ | ✅ |

**Legend:**

- ✅ Fully supported
- ⚪ Possible with managed services
- ❌ Not supported
- NATS requires JetStream for persistence
- ** SQS delay queues only support up to 15 minutes via visibility timeout
- *** Kafka maintains order within partitions only
- **** PostgreSQL supports concurrent consumers via SKIP LOCKED
- ***** SQLite/PostgreSQL can be extended for priority via custom queries

## Transport Details

### Go Channel (`channel`)

**Best For:** Testing, local development, single-process applications

```go
cfg := &protoflow.Config{
    PubSubSystem: "channel",
}
```

**Pros:**

- Zero external dependencies
- Fastest possible throughput
- Simplest setup

**Cons:**

- No persistence (messages lost on restart)
- Single process only
- No consumer groups

---

### Apache Kafka (`kafka`)

**Best For:** High-throughput event streaming, event sourcing, real-time analytics

```go
cfg := &protoflow.Config{
    PubSubSystem:       "kafka",
    KafkaBrokers:       []string{"localhost:9092"},
    KafkaConsumerGroup: "my-service",
}
```

**Pros:**

- Extremely high throughput (millions of messages/sec)
- Strong ordering guarantees within partitions
- Built-in replication and fault tolerance
- Message retention for replay

**Cons:**

- Complex operational overhead
- No native delayed message support
- Ordering only guaranteed within a partition

**Delayed Messages Workaround:**

Kafka doesn't support delayed messages natively. Alternatives:

1. Use a separate "delay" topic with a consumer that waits
2. Use Kafka Streams with a state store for scheduling
3. Combine with a scheduler service

---

### RabbitMQ (`rabbitmq`)

**Best For:** Traditional messaging, complex routing, enterprise integration

```go
cfg := &protoflow.Config{
    PubSubSystem: "rabbitmq",
    RabbitMQURL:  "amqp://guest:guest@localhost:5672/",
}
```

**Pros:**

- Mature, battle-tested broker
- Native delayed messages (via plugin or dead-letter exchange)
- Flexible routing patterns
- Message prioritization

**Cons:**

- Lower throughput than Kafka
- More complex to scale horizontally
- Plugin required for delayed messages

**Delayed Messages:**

```go
// Set TTL and dead-letter exchange for delayed processing
msg.Metadata.Set("x-delay", "60000") // 60 seconds (requires rabbitmq_delayed_message_exchange plugin)
```

---

### AWS SQS (`aws`)

**Best For:** AWS-native applications, serverless architectures

```go
cfg := &protoflow.Config{
    PubSubSystem: "aws",
    AWSRegion:    "us-east-1",
    AWSAccountID: "123456789012",
}
```

**Pros:**

- Fully managed, no operational overhead
- Automatic scaling
- Pay-per-use pricing
- Native AWS integration

**Cons:**

- Limited delay support (max 15 minutes visibility timeout)
- No strict ordering (FIFO queues have limitations)
- Higher latency than dedicated brokers
- AWS lock-in

**Delayed Messages:**

SQS doesn't support true delayed messages, but you can use:

1. Visibility timeout (up to 12 hours, but affects retries)
2. Message timers in SQS (up to 15 minutes)
3. Step Functions for longer delays

---

### NATS (`nats`)

**Best For:** Low-latency messaging, microservices communication, edge computing

```go
cfg := &protoflow.Config{
    PubSubSystem: "nats",
    NATSURL:      "nats://localhost:4222",
}
```

**Pros:**

- Extremely low latency
- Simple deployment
- Request-reply pattern support
- Lightweight resource usage

**Cons:**

- No delayed messages
- Requires JetStream for persistence
- Simpler feature set than Kafka/RabbitMQ

---

### HTTP (`http`)

**Best For:** Webhooks, request-response patterns, API-based integration

```go
cfg := &protoflow.Config{
    PubSubSystem:      "http",
    HTTPServerAddress: ":8080",
    HTTPPublisherURL:  "http://localhost:8080",
}
```

**Pros:**

- Universal compatibility
- Easy debugging with standard HTTP tools
- Works through firewalls

**Cons:**

- No persistence
- Synchronous processing only
- No consumer groups or scaling

---

### File I/O (`io`)

**Best For:** Simple persistence, debugging, development

```go
cfg := &protoflow.Config{
    PubSubSystem: "io",
    IOFile:       "/var/log/messages.log",
}
```

**Pros:**

- Simple file-based persistence
- Easy to inspect messages
- No external dependencies

**Cons:**

- Single consumer only
- No delayed messages
- Limited scalability

---

### SQLite (`sqlite`)

**Best For:** Embedded applications, single-node services, prototyping, edge deployments

```go
cfg := &protoflow.Config{
    PubSubSystem: "sqlite",
    SQLiteFile:   "queue.db", // Use ":memory:" for in-memory
}
```

**Pros:**

- ✅ **Native delayed message support** - Schedule jobs for future processing
- Persistent, ACID-compliant storage
- Built-in DLQ with inspection and replay
- No external dependencies
- Queue introspection (pending count, DLQ count)
- Transactional message handling

**Cons:**

- Single-node only (no clustering)
- Lower throughput than dedicated message brokers
- Not suitable for high-volume distributed systems

**Delayed Messages:**

```go
msg := message.NewMessage(protoflow.CreateULID(), payload)
msg.Metadata.Set("protoflow_delay", "30s")  // Process after 30 seconds
msg.Metadata.Set("protoflow_delay", "5m")   // Process after 5 minutes
msg.Metadata.Set("protoflow_delay", "1h")   // Process after 1 hour
```

**DLQ Management:**

SQLite transport provides programmatic DLQ access:

- `ListDLQMessages(topic, limit, offset)` - Browse failed messages
- `ReplayDLQMessage(dlqID)` - Retry a single message
- `ReplayAllDLQ(topic)` - Retry all messages for a topic
- `PurgeDLQ(topic)` - Delete all DLQ messages for a topic
- `GetDLQCount(topic)` - Get DLQ message count
- `GetPendingCount(topic)` - Get pending message count

---

### PostgreSQL (`postgres`)

**Best For:** Production workloads, multi-node services, teams already using PostgreSQL

```go
cfg := &protoflow.Config{
    PubSubSystem: "postgres",
    PostgresURL:  "postgres://user:pass@localhost:5432/dbname?sslmode=disable",
}
```

**Pros:**

- ✅ **Native delayed message support** - Schedule jobs for future processing
- ✅ **SKIP LOCKED** - Efficient concurrent consumer support without blocking
- Production-ready ACID compliance
- Built-in DLQ with inspection and replay
- Horizontal scaling with multiple consumers
- Exponential backoff retry
- Connection pooling
- Works with existing PostgreSQL infrastructure

**Cons:**

- Requires PostgreSQL server
- Higher latency than dedicated message brokers
- Polling-based (configurable poll interval)

**Delayed Messages:**

```go
msg := message.NewMessage(protoflow.CreateULID(), payload)
msg.Metadata.Set("protoflow_delay", "30s")  // Process after 30 seconds
msg.Metadata.Set("protoflow_delay", "5m")   // Process after 5 minutes
msg.Metadata.Set("protoflow_delay", "1h")   // Process after 1 hour
```

**DLQ Management:**

PostgreSQL transport provides the same DLQ API as SQLite:

- `ListDLQMessages(topic, limit, offset)` - Browse failed messages
- `ReplayDLQMessage(dlqID)` - Retry a single message
- `ReplayAllDLQ(topic)` - Retry all messages for a topic
- `PurgeDLQ(topic)` - Delete all DLQ messages for a topic
- `GetDLQCount(topic)` - Get DLQ message count
- `GetPendingCount(topic)` - Get pending message count

**Maintenance Operations:**

```go
// Cleanup expired locks (for crashed consumers)
transport.CleanupExpiredLocks()

// Reclaim space after high-volume processing
transport.VacuumTables()
```

---

## Use Case Recommendations

### Testing & Development

**Recommended:** Channel or SQLite (in-memory)

```go
cfg := &protoflow.Config{
    PubSubSystem: "channel",  // Or "sqlite" with SQLiteFile: ":memory:"
}
```

### Single-Node Production with Delayed Jobs

**Recommended:** SQLite

```go
cfg := &protoflow.Config{
    PubSubSystem: "sqlite",
    SQLiteFile:   "/var/lib/myapp/queue.db",
}
```

### High-Throughput Event Streaming

**Recommended:** Kafka

```go
cfg := &protoflow.Config{
    PubSubSystem:       "kafka",
    KafkaBrokers:       []string{"kafka1:9092", "kafka2:9092"},
    KafkaConsumerGroup: "my-service",
}
```

### Enterprise Integration with Complex Routing

**Recommended:** RabbitMQ

```go
cfg := &protoflow.Config{
    PubSubSystem: "rabbitmq",
    RabbitMQURL:  "amqp://user:pass@rabbitmq-cluster:5672/",
}
```

### AWS Cloud-Native Applications

**Recommended:** AWS SQS

```go
cfg := &protoflow.Config{
    PubSubSystem: "aws",
    AWSRegion:    "us-east-1",
    AWSAccountID: "123456789012",
}
```

### Low-Latency Microservices

**Recommended:** NATS

```go
cfg := &protoflow.Config{
    PubSubSystem: "nats",
    NATSURL:      "nats://nats-cluster:4222",
}
```

### Edge/Embedded Systems

**Recommended:** SQLite

```go
cfg := &protoflow.Config{
    PubSubSystem: "sqlite",
    SQLiteFile:   "queue.db",
}
```

### Production with Existing PostgreSQL Infrastructure

**Recommended:** PostgreSQL

```go
cfg := &protoflow.Config{
    PubSubSystem: "postgres",
    PostgresURL:  "postgres://user:pass@postgres-cluster:5432/myapp?sslmode=require",
}
```

---

## Delayed Message Support Summary

| Transport | Delayed Message Support | Max Delay | Configuration |
|-----------|------------------------|-----------|---------------|
| **PostgreSQL** | ✅ Native | Unlimited | `msg.Metadata.Set("protoflow_delay", "30s")` |
| **SQLite** | ✅ Native | Unlimited | `msg.Metadata.Set("protoflow_delay", "30s")` |
| **RabbitMQ** | ✅ Via Plugin | Unlimited | Requires `rabbitmq_delayed_message_exchange` plugin |
| **AWS SQS** | ⚠️ Limited | 15 minutes | Via message timers or visibility timeout |
| **Kafka** | ❌ Not Native | N/A | Requires external scheduler or custom implementation |
| **NATS** | ❌ Not Supported | N/A | Would need external scheduler |
| **Channel** | ❌ Not Supported | N/A | In-memory only |
| **HTTP** | ❌ Not Supported | N/A | Synchronous only |
| **File I/O** | ❌ Not Supported | N/A | Simple append log |

---

## DLQ (Dead Letter Queue) Capabilities

| Transport | DLQ Support | Inspection | Replay | Purge |
|-----------|-------------|------------|--------|-------|
| **PostgreSQL** | ✅ Built-in | ✅ | ✅ | ✅ |
| **SQLite** | ✅ Built-in | ✅ | ✅ | ✅ |
| **RabbitMQ** | ✅ Via DLX | ✅ | ✅ | ✅ |
| **AWS SQS** | ✅ Native | ✅ | ✅ | ✅ |
| **Kafka** | ⚠️ Manual setup | Limited | Manual | Manual |
| **NATS** | ⚠️ Manual setup | Limited | Manual | Manual |
| **Channel** | ✅ In-memory | ❌ | ❌ | ❌ |
| **HTTP** | ✅ Via config | ❌ | ❌ | ❌ |
| **File I/O** | ✅ Via config | ❌ | ❌ | ❌ |

---

## See Also

- [Configuration Guide](configuration/README.md) - Detailed transport configuration
- [Examples](../examples/) - Working examples for each transport
  - [`examples/postgres/`](../examples/postgres/) - PostgreSQL transport with delayed messages
  - [`examples/sqlite/`](../examples/sqlite/) - SQLite transport with delayed messages
  - [`examples/hooks/`](../examples/hooks/) - Job lifecycle hooks
  - [`examples/dlq_metrics/`](../examples/dlq_metrics/) - DLQ metrics collection
