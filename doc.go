// Package protoflow is a small layer on top of Watermill that wires routers,
// publishers, subscribers, and middleware for protobuf- or JSON-driven services.
// It reads the target transport (Kafka, RabbitMQ, AWS SNS/SQS, NATS, HTTP, I/O,
// SQLite, PostgreSQL, or Go Channels) from Config, bootstraps the Watermill router, and
// registers the default middleware chain for correlation IDs, logging, validation,
// outbox persistence, tracing, retries, and poison queue forwarding.
//
// Service hosts the router and exposes typed helpers: RegisterProtoHandler and
// RegisterJSONHandler take care of marshaling, metadata cloning, and optional
// protobuf validation, while Service.PublishProto lets HTTP/RPC handlers emit
// events without touching low-level Watermill APIs. A minimal setup therefore involves
// filling Config, creating a Service, registering handlers, and calling Start;
// see README.md for a copy/paste quick start snippet.
//
// # Transports
//
// Protoflow supports 9 message transports out of the box:
//   - channel: In-memory Go channels for testing
//   - kafka: High-throughput streaming with consumer groups
//   - rabbitmq: AMQP-based durable queues
//   - aws: AWS SNS/SQS with LocalStack support
//   - nats: High-performance messaging
//   - http: Request/response messaging
//   - io: File-based persistence
//   - sqlite: Embedded persistent queue with delayed messages and DLQ management
//   - postgres: Production-ready PostgreSQL queue with SKIP LOCKED and DLQ
//
// # Middleware
//
// The default middleware chain includes correlation ID injection, structured logging,
// protobuf validation, outbox persistence, OpenTelemetry tracing, Prometheus metrics,
// retry with exponential backoff, poison queue forwarding, and panic recovery.
// Custom middleware can be added via ServiceDependencies.Middlewares.
//
// # Job Hooks
//
// JobHooksMiddleware provides OnJobStart, OnJobDone, and OnJobError callbacks for
// custom logging, metrics collection, and alerting around handler execution.
//
// When you need more control, ServiceDependencies exposes well-scoped hooks:
// bring your own OutboxStore, ProtoValidator, middleware registrations, or even
// an entire TransportFactory to plug in custom brokers. The README organises
// these knobs by topic so you can dive into the exact setting you want to
// adjust without rereading the whole guide.
package protoflow
