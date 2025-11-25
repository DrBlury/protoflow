# Protoflow

[![Go Reference](https://pkg.go.dev/badge/github.com/drblury/protoflow.svg)](https://pkg.go.dev/github.com/drblury/protoflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/drblury/protoflow)](https://goreportcard.com/report/github.com/drblury/protoflow)
[![CI](https://github.com/DrBlury/protoflow/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/DrBlury/protoflow/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/DrBlury/protoflow/branch/main/graph/badge.svg)](https://codecov.io/gh/DrBlury/protoflow)
[![Latest Tag](https://img.shields.io/github/v/tag/DrBlury/protoflow?sort=semver&label=latest%20tag)](https://github.com/DrBlury/protoflow/tags)
[![License](https://img.shields.io/github/license/DrBlury/protoflow)](LICENSE)

**Stop writing plumbing. Start shipping features.**

Protoflow is a productivity layer for [Watermill](https://watermill.io/) that simplifies event-driven architecture. It manages routers, publishers, subscribers, and middleware so you can focus on your domain logic.

Whether you are using Protobufs or JSON, Protoflow provides a type-safe, production-ready foundation for Kafka, RabbitMQ, AWS SNS/SQS, NATS, HTTP, and Go Channels.

## Feature Highlights

- **Type-Safe Handlers**: Generic `RegisterProtoHandler` and `RegisterJSONHandler` helpers keep your code clean.
- **7 Built-in Transports**: Kafka, RabbitMQ, AWS SNS/SQS, NATS, HTTP, File I/O, and Go Channels - switch with a single config change.
- **Batteries Included**: Default middleware stack with correlation IDs, structured logging, protobuf validation, outbox pattern, OpenTelemetry tracing, Prometheus metrics, retries with exponential backoff, poison queues, and panic recovery.
- **Pluggable Logging**: Bring your own logger (slog, logrus, zerolog) via `ServiceLogger` abstraction.
- **Safe Configuration**: Built-in validation with credential redaction in logs.
- **Graceful Lifecycle**: Clean shutdown of HTTP servers and message routers.
- **Extensible**: Custom transport factories, middleware, validators, and outbox stores.

## Quick Start

1. **Install**: `go get github.com/drblury/protoflow` (Go 1.23+).
2. **Configure**: Set up `protoflow.Config`.
3. **Launch**: Create a `Service`, register your handlers, and `Start`.

\`\`\`go
// 1. Configure your transport (Kafka, RabbitMQ, AWS, NATS, HTTP, IO, Channel)
cfg := &protoflow.Config{
    PubSubSystem: "channel", // Use in-memory channel for testing
    PoisonQueue:  "orders.poison",
}

// 2. Use your preferred logger (slog, logrus, zerolog, etc.)
logger := protoflow.NewSlogServiceLogger(slog.Default())
svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{})

// 3. Register a strongly-typed handler
must(protoflow.RegisterProtoHandler(svc, protoflow.ProtoHandlerRegistration[*models.OrderCreated]{
    Name:         "orders-created",
    ConsumeQueue: "orders.created",
    Handler: func(ctx context.Context, evt protoflow.ProtoMessageContext[*models.OrderCreated]) ([]protoflow.ProtoMessageOutput, error) {
        evt.Logger.Info("Order received", protoflow.LogFields{"id": evt.Payload.OrderId})
        return nil, nil
    },
}))

// 4. Start the service
go func() { _ = svc.Start(ctx) }()
\`\`\`

Want to emit events? Need metadata handling? Check out the [Handlers Guide](docs/handlers/README.md).

## Supported Transports

| Transport | Config Value | Use Case |
|-----------|--------------|----------|
| Go Channels | `"channel"` | Testing, local development |
| Kafka | `"kafka"` | High-throughput streaming |
| RabbitMQ | `"rabbitmq"` | Traditional message queuing |
| AWS SNS/SQS | `"aws"` | Cloud-native pub/sub |
| NATS | `"nats"` | High-performance messaging |
| HTTP | `"http"` | Request/response patterns |
| File I/O | `"io"` | Simple message persistence |

## Default Middleware Stack

1. **Correlation ID**: Injects tracing IDs into message metadata
2. **Message Logging**: Debug logging with payload and metadata
3. **Proto Validation**: Schema validation for protobuf messages
4. **Outbox Pattern**: Reliable message delivery via OutboxStore
5. **OpenTelemetry Tracing**: Distributed tracing with span propagation
6. **Prometheus Metrics**: Request counts and latencies
7. **Retry with Backoff**: Configurable exponential backoff
8. **Poison Queue**: Dead letter queue for unprocessable messages
9. **Panic Recovery**: Converts panics to errors for graceful handling

## Logging

`protoflow.ServiceLogger` unifies logging across the router, middleware, and transports. Wrap your favorite logger and pass it to `NewService`:

- **`protoflow.NewSlogServiceLogger`**: Adapts `log/slog` (standard library).
- **`protoflow.NewEntryServiceLogger`**: Adapts structured loggers (logrus, zerolog) via `EntryLoggerAdapter[T]`.

\`\`\`go
svc := protoflow.NewService(cfg,
    protoflow.NewEntryServiceLogger(myLogrusEntry),
    ctx,
    protoflow.ServiceDependencies{},
)
\`\`\`

## Error Handling

Protoflow provides `TryNewService` for error-returning service creation:

\`\`\`go
svc, err := protoflow.TryNewService(cfg, logger, ctx, deps)
if err != nil {
    // Handle protoflow.ConfigValidationError, etc.
}
\`\`\`

## Documentation

- [**Handlers Guide**](docs/handlers/README.md): Typed handlers, metadata, and publishing.
- [**Configuration Guide**](docs/configuration/README.md): Transports, middleware, and dependency injection.
- [**Development Guide**](docs/development/README.md): Local development, testing, and Taskfile workflow.

## Examples

Check out `examples/` for runnable code:

- `simple`: Basic Protoflow usage with Go Channels.
- `json`: Typed JSON handlers with metadata enrichment.
- `proto`: Protobuf handlers with generated models.
- `full`: Comprehensive example with custom middleware, validators, and outbox.

Run them with: `go run ./examples/<name>`

## Development Workflow

We use `task` (Taskfile) to manage development:

- `task lint`: Run golangci-lint
- `task test`: Run full test suite

See the [Development Guide](docs/development/README.md) for details.

## Contributing

1. Fork the repo and branch from `main`.
2. Run `task lint` and `task test` before opening a PR.
3. Add or update docs in `docs/` for new features.
4. Keep commits focused with context in PR descriptions.

## License

Protoflow is available under the [MIT License](LICENSE).
