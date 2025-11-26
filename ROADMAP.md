# Protoflow Roadmap

This document outlines the future direction and planned features for the Protoflow library.

## ✅ Implemented Features

### Transports

- [x] **Go Channels**: In-memory transport for testing and local development
- [x] **Kafka**: Full pub/sub support with consumer groups
- [x] **RabbitMQ**: AMQP-based messaging with durable queues
- [x] **AWS SNS/SQS**: Cloud-native pub/sub with LocalStack support
- [x] **NATS**: High-performance messaging
- [x] **NATS JetStream**: Persistent streaming with delayed delivery support
- [x] **HTTP**: HTTP-based request/response messaging
- [x] **I/O (File)**: File-based transport for simple message persistence
- [x] **SQLite**: Lightweight embedded queue for simple deployments with built-in DLQ
- [x] **PostgreSQL**: Production-ready queue with SKIP LOCKED and DLQ management

### Event Model

- [x] **CloudEvents v1.0**: Standardized event format as canonical model
  - Full CloudEvents v1.0 spec compliance
  - Watermill message conversion utilities
  - Protoflow extension attributes for reliability semantics

- [x] **Transport Capabilities**: Introspection API for transport features
  - `SupportsDelay`, `SupportsNativeDLQ`, `SupportsOrdering`, `SupportsTracing`
  - Per-transport capability functions

- [x] **Handler Error Types**: Structured error returns for message lifecycle
  - `ErrRetry` - Retry with backoff
  - `ErrRetryAfter(duration)` - Retry after specific delay
  - `ErrDeadLetter` - Send to dead letter queue

### Middleware

- [x] **Correlation ID**: Automatic request tracing across services
- [x] **Structured Logging**: Debug logging with metadata
- [x] **Proto Validation**: Schema validation for protobuf messages
- [x] **Outbox Pattern**: Reliable message delivery via outbox store
- [x] **OpenTelemetry Tracing**: Distributed tracing with span propagation
- [x] **Prometheus Metrics**: Request counts, latencies, and custom metrics
- [x] **Retry with Backoff**: Configurable exponential backoff
- [x] **Poison Queue**: Dead letter queue for failed messages
- [x] **Panic Recovery**: Graceful error handling for panics
- [x] **Job Hooks**: `OnJobStart`, `OnJobDone`, `OnJobError` callbacks for custom logging, metrics, alerting

### Core Features

- [x] **Type-Safe Handlers**: Generic `RegisterProtoHandler` and `RegisterJSONHandler`
- [x] **Service Logger Abstraction**: Pluggable logging (slog, logrus, zerolog, etc.)
- [x] **Configuration Validation**: Runtime validation with helpful error messages
- [x] **Credential Redaction**: Safe logging of sensitive configuration
- [x] **Graceful Shutdown**: Clean HTTP server and router shutdown
- [x] **Custom Transport Factory**: Bring your own message broker
- [x] **Metadata Propagation**: Automatic metadata handling across handlers
- [x] **WebUI API**: Handler introspection endpoint with configurable CORS
- [x] **DLQ Metrics**: Prometheus metrics for dead letter queue monitoring

---

## 🚀 Planned Features

### Priority 1: Production Readiness

- [x] **PostgreSQL Transport**: Transactional job queue using PostgreSQL
  - SKIP LOCKED for efficient concurrent consumers
  - Built-in schema migrations
  - Job scheduling with delayed messages
  - Full DLQ management (list, replay, purge)

- [ ] **Rate Limiting Middleware**: Token bucket / sliding window rate limiting
  - Per-handler limits
  - Per-queue limits
  - Configurable overflow behavior

- [ ] **Circuit Breaker Middleware**: Protect downstream services
  - Configurable failure thresholds
  - Half-open state for recovery
  - Per-handler circuit breakers

### Priority 2: Job Queue Features (inspired by [goqueue](https://github.com/saravanasai/goqueue))

- [x] **Delayed Jobs**: Schedule jobs to run at a specific time (via SQLite transport)
  - Set `protoflow_delay` metadata for delayed processing
  - Supports durations like `30s`, `5m`, `1h`

- [ ] **Job Priorities**: Priority queues for urgent work
  - High/Medium/Low priority levels
  - Weighted fair scheduling

- [ ] **Worker Pools**: Configurable concurrent workers
  - `WithWorkerCount(n)` option
  - Graceful scaling

- [x] **Dead Letter Queue Management** (via SQLite transport)
  - Inspect failed jobs (`ListDLQMessages`)
  - Replay individual jobs (`ReplayDLQMessage`)
  - Bulk replay with filtering (`ReplayAllDLQ`)
  - DLQ metrics and alerting (`DLQMetrics`)

### Priority 3: Observability & Operations

- [ ] **Enhanced Metrics**
  - Queue depth gauges
  - Processing latency histograms
  - Error rate counters by error type
  - Worker utilization metrics

- [ ] **Health Checks**
  - `/health` and `/ready` endpoints
  - Transport connectivity checks
  - Downstream dependency checks

- [ ] **Structured Error Types**
  - Retryable vs non-retryable errors
  - Error classification middleware
  - Error aggregation and reporting

### Priority 4: Cloud & Enterprise

- [ ] **Google Cloud Pub/Sub**: Native GCP support

- [ ] **Redis Streams Transport**: Alternative to Kafka for simpler deployments

- [ ] **Azure Service Bus**: Native Azure messaging support

- [ ] **Schema Registry Integration**
  - Confluent Schema Registry
  - AWS Glue Schema Registry
  - Schema evolution validation

### Priority 5: Developer Experience

- [ ] **CLI Tool** (`protoflow`)
  - `protoflow init` - scaffold new service
  - `protoflow handler add` - generate handler boilerplate
  - `protoflow dlq list/replay` - DLQ management

- [ ] **WebUI Dashboard**
  - Real-time handler stats
  - Message flow visualization
  - DLQ browser with replay

---

## 💡 Feature Requests

Have an idea? Open an issue with the `enhancement` label!

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Good First Issues

- Add unit tests for uncovered code paths
- Improve documentation and examples
- Add transport-specific integration tests

---

## 📚 Documentation & Examples

The following examples are available in the `examples/` directory:

| Example | Description |
|---------|-------------|
| `simple/` | Basic untyped handler with custom retry middleware |
| `json/` | Type-safe JSON handler with metadata |
| `proto/` | Type-safe Protobuf handler with validation |
| `full/` | Complete example with multiple handlers and custom middleware |
| `hooks/` | Job lifecycle hooks (OnJobStart, OnJobDone, OnJobError) |
| `sqlite/` | SQLite transport with delayed message scheduling |
| `postgres/` | PostgreSQL transport with delayed messages and DLQ |
| `dlq_metrics/` | DLQ metrics collection with Prometheus |
| `nats-cloudevents-delayed/` | NATS JetStream with CloudEvents and delayed delivery |

Documentation guides:

- [CloudEvents Model](docs/cloudevents-model.md) - CloudEvents v1.0 integration
- [Protoflow Extensions](docs/protoflow-extensions.md) - Reliability extension attributes
- [Transport Capabilities](docs/transport-capabilities.md) - Capability introspection API
- [OpenTelemetry Tracing](docs/otel-tracing.md) - Distributed tracing integration
- [AsyncAPI Integration](docs/asyncapi-integration.md) - AsyncAPI spec generation
- [Delayed Delivery Roadmap](docs/roadmap-delayed-delivery.md) - Future delayed delivery plans
- [Transport Comparison Guide](docs/transports/README.md) - Feature matrix for all transports
- [Configuration Guide](docs/configuration/README.md) - Transport and middleware configuration
- [Handlers Guide](docs/handlers/README.md) - Type-safe handler patterns
