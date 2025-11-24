# Handler Guide

Protoflow's typed handlers allow you to focus on your domain logic while the library handles unmarshaling, validation, and infrastructure concerns.

## Registering Protobuf Handlers

`RegisterProtoHandler` wires up a strongly-typed handler that:

1. Unmarshals the payload.
2. Clones metadata.
3. Validates outgoing messages.
4. Ensures type safety.

```go
err := protoflow.RegisterProtoHandler(svc, protoflow.ProtoHandlerRegistration[*models.OrderCreated]{
    Name:          "orders-created",
    ConsumeQueue:  "orders.created",
    PublishQueue:  "orders.audit", // Optional: auto-publish outputs here
    Options: []protoflow.ProtoHandlerOption{
        // Register extra types for validation safety
        protoflow.WithPublishMessageTypes(&models.OrderHandled{}),
    },
    Handler: func(ctx context.Context, evt protoflow.ProtoMessageContext[*models.OrderCreated]) ([]protoflow.ProtoMessageOutput, error) {
        // 0. Log something
        evt.Logger.Info("Processing order", protoflow.LogFields{"order_id": evt.Payload.OrderId})

        // 1. Clone metadata and add a tag
        metadata := evt.CloneMetadata().With("handled_by", "proto")

        // 2. Create your response
        msg := &models.OrderHandled{OrderId: evt.Payload.OrderId}

        // 3. Return it!
        return []protoflow.ProtoMessageOutput{ {
            Message:  msg,
            Metadata: metadata,
        } }, nil
    },
})
```

**Tips:**

- **`PublishQueue`**: If set, every message returned by the handler is published to this queue.
- **`WithPublishMessageTypes`**: Validates message types at startup.
- **Return `nil, err`**: Triggers the retry middleware.

## Registering JSON Handlers

`RegisterJSONHandler` works similarly to the protobuf version but for JSON payloads.

```go
err := protoflow.RegisterJSONHandler(svc, protoflow.JSONHandlerRegistration[*IncomingOrder, *OutgoingOrder]{
    Name:         "json-orders",
    ConsumeQueue: "json.orders",
    PublishQueue: "json.audit",
    Handler: func(ctx context.Context, evt protoflow.JSONMessageContext[*IncomingOrder]) ([]protoflow.JSONMessageOutput[*OutgoingOrder], error) {
        evt.Logger.Info("Processing JSON order", protoflow.LogFields{"id": evt.Payload.ID})
        return []protoflow.JSONMessageOutput[*OutgoingOrder]{
            {Message: &OutgoingOrder{ID: evt.Payload.ID}},
        }, nil
    },
})
```

### Metadata Helpers

- `evt.CloneMetadata()` returns a deep copy suitable for mutation.
- `Metadata.With(key, value)` returns a new metadata map without mutating the original.

## Producing Events Outside Handlers

You can publish protobuf messages from other components (e.g., HTTP handlers, cron jobs) by calling `Service.PublishProto` directly:

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
