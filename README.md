# Protoflow

Protoflow is a thin productivity layer on top of [Watermill](https://watermill.io/) that helps you build protobuf or JSON event-driven services that run on Kafka, RabbitMQ, or AWS SNS/SQS. It wires the router, publisher, subscriber, default middleware stack, and typed handler helpers so you can focus on your domain logic instead of plumbing.

## Features

- Strongly typed handler registrations for protobuf (`RegisterProtoHandler`) and JSON (`RegisterJSONHandler`) payloads.
- Built-in router wiring for Kafka, RabbitMQ, and AWS SNS/SQS transports selected via configuration.
- Default middleware chain that injects correlation IDs, logs payloads, validates protobufs, stores outgoing messages in an outbox, traces with OpenTelemetry, retries with backoff, and routes poison messages.
- Extension points for plugging in your own `ProtoValidator`, `OutboxStore`, and custom middleware registrations.
- `TransportFactory` hook so you can supply custom brokers (GCP Pub/Sub, NATS, Redis, etc.) without forking the service wiring.
- Helper utilities for publishing protobuf events (`Service.PublishProto`) and cloning metadata safely.

## Quick start

1. Install the module: `go get github.com/drblury/protoflow`.
2. Fill a `protoflow.Config` with the transport you run locally (Kafka, RabbitMQ, or AWS SNS/SQS).
3. Create a `Service`, register the handlers you need, then call `Start`.

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

Set `PublishQueue` on the handler when it should emit events.

Start with the defaults above, then drop into the configuration guide below whenever you need to tune a specific transport, middleware, or dependency.

## Repository layout

- `internal/runtime/` hosts the implementation (service, transports, middleware, handlers, logging adapters). Keeping it internal lets the public API stay stable.
- `exports_*.go` re-export selected runtime symbols so dependants import `github.com/drblury/protoflow` without worrying about the internal structure.
- `examples/` showcases end-to-end setups (simple/raw, JSON, protobuf, hybrid “full”) you can copy into your own services.

Use the runtime package as the reference if you need deeper extension points; stick to the re-exports for regular application code.

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

### Registering protobuf handlers

`RegisterProtoHandler` wires a strongly typed protobuf handler that automatically unmarshals payloads, clones metadata, and accepts an optional list of outgoing messages to publish.

```go
err := protoflow.RegisterProtoHandler(svc, protoflow.ProtoHandlerRegistration[*models.OrderCreated]{
    Name:          "proto-order-handler",
    ConsumeQueue:  "orders.proto",
    PublishQueue:  "orders.proto.audit",
    Options: []protoflow.ProtoHandlerOption{
        protoflow.WithPublishMessageTypes(&models.OrderHandled{}),
    },
    Handler: func(ctx context.Context, evt protoflow.ProtoMessageContext[*models.OrderCreated]) ([]protoflow.ProtoMessageOutput, error) {
        metadata := evt.CloneMetadata().With("handled_by", "proto")
        msg := &models.OrderHandled{OrderId: evt.Payload.OrderId}
        return []protoflow.ProtoMessageOutput{{
            Message:  msg,
            Metadata: metadata,
        }}, nil
    },
})
```

Outgoing events automatically land on the handler's `PublishQueue`. Call `WithPublishMessageTypes` when the handler can emit additional proto types so they are registered for validation. The generics keep payloads type-safe end-to-end; use `RegisterMessageHandler` if you prefer to work with raw Watermill messages.

### Registering JSON handlers

If your payloads are JSON instead of protobuf, use `RegisterJSONHandler`:

```go
err := protoflow.RegisterJSONHandler(svc, protoflow.JSONHandlerRegistration[*IncomingOrder, *OutgoingOrder]{
    Name:               "json-order-handler",
    ConsumeQueue:       "orders.json",
    PublishQueue:       "orders.json.out",
    Handler: func(ctx context.Context, evt protoflow.JSONMessageContext[*IncomingOrder]) ([]protoflow.JSONMessageOutput[*OutgoingOrder], error) {
        response := &OutgoingOrder{ID: evt.Payload.ID}
        return []protoflow.JSONMessageOutput[*OutgoingOrder]{{
            Message:  response,
            Metadata: evt.Metadata.With("processed_by", "json-handler"),
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
- entry-style loggers (for example loggers that expose `WithField`/`WithError` chains) via `protoflow.NewEntryServiceLogger`

```go
entry := customLogger.WithContext(ctx) // implements the Entry-style API
svc := protoflow.NewService(cfg, protoflow.NewEntryServiceLogger(entry), ctx, protoflow.ServiceDependencies{})
```

This lets consumers plug Protoflow into existing logging stacks without having to learn Watermill's logging abstractions—the service internally adapts your logger where necessary.

## Configuration guide

Use the sections below to tweak only the knobs you care about.

### Transport selection

Set `Config.PubSubSystem` to one of the supported transports and populate its fields:

#### Kafka (`"kafka"`)

- `KafkaBrokers`: list of broker addresses.
- `KafkaConsumerGroup`: consumer group for handlers.
- `KafkaClientID`: optional identifier for produced messages.

#### RabbitMQ (`"rabbitmq"`)

- `RabbitMQURL`: connection URL (`amqp://user:pass@host:port/vhost`). The same connection backs both publisher and subscriber.

#### AWS SNS/SQS (`"aws"`)

- `AWSRegion`: required region for both SNS and SQS.
- `AWSAccountID`: used to build ARNs. Leave empty when pointing to LocalStack to fall back to `000000000000`.
- `AWSEndpoint`: optional URL for LocalStack or private endpoints.
- `AWSAccessKeyID` / `AWSSecretAccessKey`: optional explicit credentials; otherwise the default AWS chain is used.

### Common knobs

- `PoisonQueue`: destination for messages that still fail after all retries.
- `RetryMaxRetries`, `RetryInitialInterval`, `RetryMaxInterval`: tune the default retry middleware. Zero values keep library defaults.

### Middleware and dependencies

`ServiceDependencies` lets you bring your own collaborators:

- `Validator` (`ProtoValidator`) validates protobuf payloads in `ProtoValidateMiddleware` and outgoing events.
- `Outbox` (`OutboxStore`) stores emitted events before forwarding them.
- `Middlewares` (`[]MiddlewareRegistration`) append to the default chain.
- `DisableDefaultMiddlewares` skips the built-in middleware stack so you can assemble your own.

Default middleware order:

1. Correlation ID
2. Message logger
3. Proto validation
4. Outbox persistence
5. OpenTelemetry tracer
6. Retry with exponential backoff
7. Poison queue forwarding
8. Panic recoverer

You can register additional middleware via `ServiceDependencies.Middlewares` or `Service.RegisterMiddleware`.

### Custom transports

`ServiceDependencies.TransportFactory` lets you replace the built-in Kafka/RabbitMQ/AWS wiring with your own implementation. The factory receives the request context, resolved `Config`, and the `watermill.LoggerAdapter`, and must return a `protoflow.Transport` containing both a publisher and a subscriber. This makes it straightforward to plug in GCP Pub/Sub, NATS, Redis streams, in-memory brokers for tests, or anything else you need without editing `Service`.

```go
type gcppubsubFactory struct{ client *pubsub.Client }

func (f gcppubsubFactory) Build(ctx context.Context, conf *protoflow.Config, logger watermill.LoggerAdapter) (protoflow.Transport, error) {
    pub := newPubSubPublisher(f.client, logger)
    sub := newPubSubSubscriber(f.client, conf, logger)
    return protoflow.Transport{Publisher: pub, Subscriber: sub}, nil
}

svc := protoflow.NewService(cfg, logger,
    ctx,
    protoflow.ServiceDependencies{TransportFactory: gcppubsubFactory{client: client}},
)
```

If you do not supply a factory, Protoflow falls back to the default implementation located in `internal/runtime/transport_factory.go` that selects Kafka, RabbitMQ, or AWS SNS/SQS based on `Config.PubSubSystem`.

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

### Future ideas

The list below balances quick wins with longer-term bets.

#### High leverage (low-to-medium effort)

1. **Transport factory registry** – package a registry plus bundled factories (GCP Pub/Sub, NATS, Redis streams) so most teams can adopt new brokers by importing a module instead of writing glue code.
2. **Stateful outbox adapters** – provide ready-made `OutboxStore` implementations for Postgres, DynamoDB, and Redis along with a polling forwarder. Medium lift, high value because durable outboxes are hard to get right.
3. **Observability starter pack** – ship middleware presets that emit RED metrics and OpenTelemetry spans with sensible defaults. Mostly wiring work with clear payoff.

#### Longer-term explorations (higher effort)

1. **Schema registry integration** – resolve protobuf/JSON schemas from Confluent or AWS Glue to validate compatibility before handlers start. High complexity due to registry APIs and caching.
2. **Testing harness** – bundle an in-memory transport plus snapshot helpers so teams can test handler flows without brokers. Requires careful parity with production transports.
3. **Code generation CLI** – add a `protoflow generate` command that scaffolds handlers and service skeletons. Valuable but involves compiler plugins and thoughtful ergonomics.

## License

Protoflow is available under the [MIT License](LICENSE).
