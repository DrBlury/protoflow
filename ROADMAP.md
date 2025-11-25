# Protoflow Roadmap

This document outlines the future direction and planned features for the Protoflow library.

## ✅ Implemented Features

### Transports
- [x] **Go Channels**: In-memory transport for testing and local development
- [x] **Kafka**: Full pub/sub support with consumer groups
- [x] **RabbitMQ**: AMQP-based messaging with durable queues
- [x] **AWS SNS/SQS**: Cloud-native pub/sub with LocalStack support
- [x] **NATS**: High-performance messaging
- [x] **HTTP**: HTTP-based request/response messaging
- [x] **I/O (File)**: File-based transport for simple message persistence

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

### Core Features
- [x] **Type-Safe Handlers**: Generic `RegisterProtoHandler` and `RegisterJSONHandler`
- [x] **Service Logger Abstraction**: Pluggable logging (slog, logrus, zerolog, etc.)
- [x] **Configuration Validation**: Runtime validation with helpful error messages
- [x] **Credential Redaction**: Safe logging of sensitive configuration
- [x] **Graceful Shutdown**: Clean HTTP server and router shutdown
- [x] **Custom Transport Factory**: Bring your own message broker
- [x] **Metadata Propagation**: Automatic metadata handling across handlers
- [x] **WebUI API**: Handler introspection endpoint with configurable CORS

---

## 🚀 Planned Features

### Priority 1: Production Readiness

- [ ] **PostgreSQL Transport**: Transactional job queue using PostgreSQL (inspired by [gue](https://github.com/vgarvardt/gue))
  - Transaction-level locks for exactly-once processing
  - Built-in migrations
  - Job scheduling with `RunAt`
  
- [ ] **SQLite Transport**: Lightweight embedded queue for simple deployments

- [ ] **Rate Limiting Middleware**: Token bucket / sliding window rate limiting
  - Per-handler limits
  - Per-queue limits
  - Configurable overflow behavior

- [ ] **Circuit Breaker Middleware**: Protect downstream services
  - Configurable failure thresholds
  - Half-open state for recovery
  - Per-handler circuit breakers

### Priority 2: Job Queue Features (inspired by [goqueue](https://github.com/saravanasai/goqueue))

- [ ] **Delayed Jobs**: Schedule jobs to run at a specific time
  - `DispatchWithDelay(job, duration)`
  - `DispatchAt(job, time.Time)`

- [ ] **Job Priorities**: Priority queues for urgent work
  - High/Medium/Low priority levels
  - Weighted fair scheduling

- [ ] **Worker Pools**: Configurable concurrent workers
  - `WithWorkerCount(n)` option
  - Graceful scaling

- [ ] **Dead Letter Queue Management**
  - Inspect failed jobs
  - Replay individual jobs
  - Bulk replay with filtering
  - DLQ metrics and alerting

- [ ] **Job Hooks**
  - `OnJobStart` / `OnJobDone` / `OnJobError`
  - Custom logging, metrics, alerting

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
