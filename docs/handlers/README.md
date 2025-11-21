# Handler Guide

Protoflow ships typed helpers on top of Watermill so handlers stay focused on domain logic. This guide walks through protobuf handlers, JSON handlers, and publishing outside handler flows.

## Registering protobuf handlers

`RegisterProtoHandler` wires a strongly typed protobuf handler: it unmarshals payloads, clones metadata, validates outgoing messages, and keeps publisher registrations in sync via generics.

```go
err := protoflow.RegisterProtoHandler(svc, protoflow.ProtoHandlerRegistration[*models.OrderCreated]{
    Name:          "orders-created",
    ConsumeQueue:  "orders.created",
    PublishQueue:  "orders.audit",
    Options: []protoflow.ProtoHandlerOption{
        protoflow.WithPublishMessageTypes(&models.OrderHandled{}),
    },
    Handler: func(ctx context.Context, evt protoflow.ProtoMessageContext[*models.OrderCreated]) ([]protoflow.ProtoMessageOutput, error) {
        metadata := evt.CloneMetadata().With("handled_by", "proto")
        msg := &models.OrderHandled{OrderId: evt.Payload.OrderId}
        return []protoflow.ProtoMessageOutput{ {
            Message:  msg,
            Metadata: metadata,
        } }, nil
    },
})
```

Key points:

- `PublishQueue` is optional; when set, every emitted `ProtoMessageOutput` lands on that queue.
- `WithPublishMessageTypes` registers additional protobuf types for validation before runtime errors can surface.
- Handlers return zero or more outputs; return `nil, err` to trigger the retry middleware.

## Registering JSON handlers

`RegisterJSONHandler` mirrors the protobuf helper but works with JSON payloads:

```go
err := protoflow.RegisterJSONHandler(svc, protoflow.JSONHandlerRegistration[*IncomingOrder, *OutgoingOrder]{
    Name:         "json-order-handler",
    ConsumeQueue: "orders.json",
    PublishQueue: "orders.json.out",
    Handler: func(ctx context.Context, evt protoflow.JSONMessageContext[*IncomingOrder]) ([]protoflow.JSONMessageOutput[*OutgoingOrder], error) {
        response := &OutgoingOrder{ID: evt.Payload.ID}
        return []protoflow.JSONMessageOutput[*OutgoingOrder]{ {
            Message:  response,
            Metadata: evt.Metadata.With("processed_by", "json-handler"),
        } }, nil
    },
})
```

### Metadata helpers

- `evt.CloneMetadata()` returns a deep copy suitable for mutation.
- `Metadata.With(key, value)` is fluent and returns a new metadata map without mutating the original.

## Producing events outside handlers

You can publish protobuf messages from HTTP handlers, cron jobs, or any other component by calling `Service.PublishProto` directly:

```go
metadata := protoflow.Metadata{"event_source": "api"}
if err := svc.PublishProto(ctx, "orders.created", &models.OrderCreated{OrderId: "123"}, metadata); err != nil {
    logger.Error("publish failed", "err", err)
}
```

When you only need raw Watermill access, fall back to `RegisterMessageHandler` and the underlying `Publisher` exposed via `ServiceDependencies`.

## Logging adapters

`NewService` expects a `ServiceLogger`. Wrap whichever logger you already use:

- `protoflow.NewSlogServiceLogger` adapts `log/slog` loggers.
- `protoflow.NewEntryServiceLogger` adapts entry-style loggers that implement `WithField`, `WithError`, etc.

```go
entry := customLogger.WithContext(ctx)
svc := protoflow.NewService(cfg, protoflow.NewEntryServiceLogger(entry), ctx, protoflow.ServiceDependencies{})
```

Adapters make Protoflow cooperate with existing logging stacks while Watermill receives the `watermill.LoggerAdapter` it needs internally.
