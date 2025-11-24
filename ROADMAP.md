# Protoflow Roadmap

This document outlines the future direction and planned features for the Protoflow library.

## Planned Features

### Transports

- [x] **Go Channels Transport**: In-memory transport for testing and local development.
- [x] **I/O Transport**: File-based transport for simple message persistence.
- [x] **HTTP Transport**: Support for HTTP-based messaging.
- [x] **NATS JetStream**: High-performance messaging with NATS.
- [ ] **PostgreSQL LISTEN/NOTIFY Transport**: Leverage PostgreSQL's built-in pub/sub capabilities.
- [ ] **Google Cloud Pub/Sub**: Native support for GCP Pub/Sub.
- [ ] **SQLite Transport**: Lightweight SQL-based transport for simple use cases.

### Middleware

- [ ] **Rate Limiting**: Built-in rate limiting middleware.
- [ ] **Circuit Breaker**: Resilience patterns for failing downstream services.
- [x] **Metrics**: First-class Prometheus/OpenTelemetry metrics integration (beyond basic tracing).
<!-- - [ ] **Dead Letter Queue (DLQ) Management**: Tools for inspecting and replaying DLQ messages. -->

### Developer Experience

<!-- - [ ] **CLI Tool**: A `protoflow` CLI for scaffolding new services and handlers. -->
<!-- - [ ] **Schema Registry**: Integration with schema registries (e.g., Confluent, AWS Glue) for validation. -->

## Contribution

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) (to be created) for details.
