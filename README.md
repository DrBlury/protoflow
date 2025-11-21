# Protoflow

[![Go Reference](https://pkg.go.dev/badge/github.com/drblury/protoflow.svg)](https://pkg.go.dev/github.com/drblury/protoflow)
[![pkg.go.dev](https://img.shields.io/badge/pkg.go.dev-docs-00ADD8?logo=go&logoColor=white)](https://pkg.go.dev/github.com/drblury/protoflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/drblury/protoflow)](https://goreportcard.com/report/github.com/drblury/protoflow)
[![CI](https://github.com/DrBlury/protoflow/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/DrBlury/protoflow/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/DrBlury/protoflow/branch/main/graph/badge.svg)](https://codecov.io/gh/DrBlury/protoflow)
[![Release](https://img.shields.io/github/v/release/DrBlury/protoflow?display_name=tag)](https://github.com/DrBlury/protoflow/releases)
[![License](https://img.shields.io/github/license/DrBlury/protoflow)](LICENSE)

Protoflow is a thin productivity layer on top of [Watermill](https://watermill.io/) that wires routers, publishers, subscribers, middleware, and typed handler helpers so you can build protobuf or JSON services without reimplementing the plumbing for Kafka, RabbitMQ, or AWS SNS/SQS. It keeps the domain-facing API in the root module while the heavy lifting lives under `internal/runtime/`.

## Feature highlights

- Typed handler registrations for protobuf (`RegisterProtoHandler`) and JSON (`RegisterJSONHandler`).
- Built-in router wiring for Kafka, RabbitMQ, and AWS SNS/SQS selected via configuration.
- Default middleware chain with correlation IDs, structured logging, proto validation, outbox persistence, OpenTelemetry traces, retries, poison queue routing, and panic recovery.
- Extension points for custom validators, outbox stores, middleware registrations, and transport factories.
- Publishing helpers so services emit protobuf events safely from any component.

## Quick start

1. Install the module: `go get github.com/drblury/protoflow` (Go 1.25+).
2. Pick a transport in `protoflow.Config`.
3. Create a `Service`, register handlers, then call `Start`.

```go
cfg := &protoflow.Config{
    PubSubSystem:       "kafka",
    KafkaBrokers:       []string{"localhost:9092"},
    KafkaConsumerGroup: "orders-service",
    PoisonQueue:        "orders.poison",
}

logger := protoflow.NewSlogServiceLogger(slog.Default())
svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{})

must(protoflow.RegisterProtoHandler(svc, protoflow.ProtoHandlerRegistration[*models.OrderCreated]{
    Name:         "orders-created",
    ConsumeQueue: "orders.created",
    Handler: func(ctx context.Context, evt protoflow.ProtoMessageContext[*models.OrderCreated]) ([]protoflow.ProtoMessageOutput, error) {
        return nil, nil
    },
}))

go func() { _ = svc.Start(ctx) }()
```

Set `PublishQueue` whenever a handler should emit follow-up events. For typed handler patterns, metadata helpers, and publishing utilities see the [Handlers guide](docs/handlers/README.md).

## Logging

`protoflow.ServiceLogger` is the single logging contract the router, middleware, and transports rely on. Wrap whichever logger you already use and supply it to `NewService`:

- `protoflow.NewSlogServiceLogger` adapts a `log/slog` logger, mapping levels onto Watermill so structured fields stay intact.
- `protoflow.NewEntryServiceLogger` takes any entry-style logger that satisfies the generic `EntryLoggerAdapter[T]` constraint (for example logrus, zerolog’s contextual logger, or your own facade) and turns it into a `ServiceLogger` without touching Watermill APIs.

```go
type contextualLogger interface {
    WithField(key string, value any) contextualLogger
    WithError(err error) contextualLogger
    Info(args ...any)
    Debug(args ...any)
    Error(args ...any)
    Trace(args ...any)
}

svc := protoflow.NewService(cfg,
    protoflow.NewEntryServiceLogger(contextualLoggerInstance),
    ctx,
    protoflow.ServiceDependencies{},
)
```

Under the hood Protoflow converts `ServiceLogger` back into a Watermill `LoggerAdapter`, so you only have to think about one logging abstraction. More details live in the [Handlers guide](docs/handlers/README.md#logging-adapters).

## Documentation

Deeper guides live under `docs/` so the README stays focused:

- [Handlers](docs/handlers/README.md) — protobuf and JSON handlers, metadata cloning, publishing helpers, logging adapters.
- [Configuration](docs/configuration/README.md) — transport knobs, middleware stack, dependency injection, custom transport factories.
- [Development](docs/development/README.md) — Taskfile commands, local broker tips, and testing instructions.

`doc.go` carries the package-level API docs published on pkg.go.dev.

## Examples

`examples/` contains runnable scenarios you can launch with `go run ./examples/<name>`:

- `simple` – raw Watermill handler registration.
- `json` – typed JSON handlers with metadata enrichment.
- `proto` – protobuf handlers backed by generated models.
- `full` – hybrid example with custom middleware, validators, an in-memory outbox, and a periodic publisher.

Use them as blueprints and swap the broker configuration for your environment.

## Development workflow

`taskfile.yml` defines repeatable tasks—`task lint`, `task test`, and transport-specific helpers—so contributors share the same local workflow. See the [development guide](docs/development/README.md) for the full command list plus local broker hints.

Run the full test suite with `go test ./...` (or `task test`) before sending changes.

## Contribution guidelines

1. Fork the repo and branch from `main`.
2. Run `task lint` and `task test` (or the equivalent commands) before opening a PR.
3. Add or update docs inside `docs/` and package comments when you ship new features.
4. Keep commits focused; attach context in PR descriptions so reviewers understand the transport, middleware, or handler surface you touched.

## License

Protoflow is available under the [MIT License](LICENSE).
