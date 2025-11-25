# 📚 Protoflow Documentation

Welcome to the Protoflow documentation. This is where you'll find detailed guides on handlers, configuration, and development workflows.

## 🗺️ Documentation Map

- [**Handlers Guide**](handlers/README.md) 🧠
  - Type-safe handlers for Protobuf and JSON messages
  - Metadata manipulation and publishing patterns
  - Using the `ServiceLogger` in handlers

- [**Configuration Guide**](configuration/README.md) ⚙️
  - Transport configuration (Kafka, RabbitMQ, AWS, NATS, HTTP, IO, Channel)
  - Middleware customization
  - Dependency injection (validators, outbox stores)

- [**Development Guide**](development/README.md) 🛠️
  - Local development setup
  - Running tests with coverage
  - Taskfile workflows

## 📦 What's Included

### Transports
- **Go Channels**: In-memory transport for testing
- **Kafka**: High-throughput streaming with consumer groups
- **RabbitMQ**: AMQP-based durable queues
- **AWS SNS/SQS**: Cloud-native pub/sub with LocalStack support
- **NATS**: High-performance messaging
- **HTTP**: Request/response messaging
- **File I/O**: Simple file-based persistence

### Middleware Stack
- Correlation ID injection
- Structured message logging
- Protobuf validation
- Outbox pattern for reliability
- OpenTelemetry distributed tracing
- Prometheus metrics
- Retry with exponential backoff
- Poison queue (dead letter)
- Panic recovery

### Core APIs
- `NewService` / `TryNewService`: Service creation
- `RegisterProtoHandler`: Type-safe protobuf handlers
- `RegisterJSONHandler`: Type-safe JSON handlers
- `PublishProto`: Direct event publishing
- `ServiceLogger`: Pluggable logging abstraction

## 🔗 Related Resources

- **`examples/`**: Runnable examples (`simple`, `json`, `proto`, `full`)
- **`internal/runtime/`**: Implementation source code
- **[ROADMAP.md](../ROADMAP.md)**: Future development plans
