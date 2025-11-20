# Protoflow

Protoflow is a thin productivity layer on top of [Watermill](https://watermill.io/) that helps you build protobuf or JSON event-driven services that run on Kafka, RabbitMQ, or AWS SNS/SQS. It wires the router, publisher, subscriber, default middleware stack, and typed handler helpers so you can focus on your domain logic instead of plumbing.

## Features

- Strongly typed handler registrations for protobuf (`RegisterProtoHandler`) and JSON (`RegisterJSONHandler`) payloads.
- Built-in router wiring for Kafka, RabbitMQ, and AWS SNS/SQS transports selected via configuration.
- Default middleware chain that injects correlation IDs, logs payloads, validates protobufs, stores outgoing messages in an outbox, traces with OpenTelemetry, retries with backoff, and routes poison messages.
- Extension points for plugging in your own `ProtoValidator`, `OutboxStore`, and custom middleware registrations.
- Helper utilities for publishing protobuf events (`PublishProto`/`Service.PublishProto`) and cloning metadata safely.

## Installation

```bash
go get github.com/drblury/protoflow
```

Go 1.25+ is recommended because the module itself targets Go 1.25.4 in `go.mod`.

## Examples

Each directory under `examples/` is a runnable scenario that you can execute with `go run ./examples/<name>`:

- `examples/simple` registers an untyped handler via `RegisterMessageHandler` and works directly with Watermill messages.
- `examples/json` wires up a typed JSON handler that forwards enriched metadata.
- `examples/proto` showcases protobuf handlers backed by the generated files in `models/`.
- `examples/full` demonstrates protobuf, JSON, and raw handlers alongside custom middleware, a validator, an in-memory outbox, and a periodic publisher.

Use these as blueprints and adjust the hardcoded broker configuration to match your environment.

## Usage

### Registering JSON handlers

If your payloads are JSON instead of protobuf, use `RegisterJSONHandler`:

```go
err := protoflow.RegisterJSONHandler(svc, protoflow.JSONHandlerRegistration[*IncomingOrder, *OutgoingOrder]{
    Name:               "json-order-handler",
    ConsumeQueue:       "orders.json",
    PublishQueue:       "orders.json.out",
    ConsumeMessageType: &IncomingOrder{},
    Handler: func(ctx context.Context, evt protoflow.JSONMessageContext[*IncomingOrder]) ([]protoflow.JSONMessageOutput[*OutgoingOrder], error) {
        response := &OutgoingOrder{ID: evt.Payload.ID}
        return []protoflow.JSONMessageOutput[*OutgoingOrder]{{
            Message:  response,
            Metadata: evt.CloneMetadata(),
        }}, nil
    },
})
```

### Producing events

Use the helper to publish protobuf messages outside of handlers (for example, from HTTP endpoints):

```go
metadata := protoflow.Metadata{"event_source": "api"}
if err := svc.PublishProto(ctx, "orders.created", &models.OrderCreated{OrderId: "123"}, metadata); err != nil {
    logger.Error("publish failed", "err", err)
}
```

### Logging

`NewService` expects a `ServiceLogger`. You can obtain one by wrapping:

- a standard library `slog.Logger` via `protoflow.NewSlogServiceLogger`
- any Watermill `LoggerAdapter` via `protoflow.NewWatermillServiceLogger`
- entry-style loggers (for example loggers that expose `WithField`/`WithError` chains) via `protoflow.NewEntryServiceLogger`

```go
entry := customLogger.WithContext(ctx) // implements the Entry-style API
svc := protoflow.NewService(cfg, protoflow.NewEntryServiceLogger(entry), ctx, protoflow.ServiceDependencies{})
```

This lets consumers plug Protoflow into existing logging stacks without having to retool to slog or Watermill-specific adapters.

## Configuration reference

`Config` selects the transport and holds per-transport settings:

```go
cfg := &protoflow.Config{
    PubSubSystem:       "kafka",            // or "rabbitmq" / "aws"
    KafkaBrokers:       []string{"broker"},
    KafkaConsumerGroup: "group",
    RabbitMQURL:        "amqp://guest:guest@localhost",
    AWSRegion:          "eu-west-1",
    AWSAccountID:       "123456789012",
    AWSEndpoint:        "http://localhost:4566", // optional (LocalStack)
    PoisonQueue:        "events.poison",
    RetryMaxRetries:    5,
}
```

Only the fields required by the selected `PubSubSystem` are used. The retry-related settings feed into the default retry middleware.

## Service dependencies and middleware

`ServiceDependencies` lets you inject optional collaborators:

- `Validator` (`ProtoValidator`) validates protobuf payloads in the `ProtoValidateMiddleware` and optionally outgoing events. Implementations just need to provide a `Validate(value any) error` method (wrapping `protovalidate.Validator`, `go-playground/validator`, etc.).
- `Outbox` (`OutboxStore`) stores emitted events before they are forwarded.
- `Middlewares` (`[]MiddlewareRegistration`) are appended after the default chain.
- `DisableDefaultMiddlewares` skips the built-in middleware stack so you can assemble your own.

The default middleware order is:

1. Correlation ID
2. Message logger
3. Proto validation
4. Outbox persistence
5. OpenTelemetry tracer
6. Retry with exponential backoff
7. Poison queue forwarding
8. Panic recoverer

You can register additional middleware by supplying `ServiceDependencies.Middlewares` or by calling `Service.RegisterMiddleware` manually.

## Local development tips

- AWS SNS/SQS: set `Config.AWSEndpoint` to your LocalStack URL to reuse the same code locally.
- Kafka: ensure `KafkaConsumerGroup` is unique per service instance.
- RabbitMQ: the connection is reused for both publisher and subscriber; supply TLS information via `amqp.ConnectionConfig` if needed.

## Testing

The repository comes with unit tests that exercise service wiring, middleware, and handler helpers. Run them with:

```bash
go test ./...
```

## Contributing

1. Fork the repo and create a feature branch.
2. Run `go test ./...` before opening a PR.
3. Keep the README and package docs up to date when you add new features.

## License

Protoflow is available under the [MIT License](LICENSE).
