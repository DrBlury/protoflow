# Handler Guide

Protoflow's typed handlers let you focus on business logic while the library handles unmarshaling, validation, and infrastructure concerns.

## Registering Protobuf Handlers

`RegisterProtoHandler` creates a type-safe handler that:

1. Unmarshals the payload into your protobuf type
2. Provides cloned metadata for safe mutation
3. Validates outgoing messages (if configured)
4. Publishes results to the specified queue

```go
err := protoflow.RegisterProtoHandler(svc, protoflow.ProtoHandlerRegistration[*models.OrderCreated]{
    Name:         "orders-created",
    ConsumeQueue: "orders.created",
    PublishQueue: "orders.audit", // Optional: auto-publish outputs here
    Options: []protoflow.ProtoHandlerOption{
        protoflow.WithPublishMessageTypes(&models.OrderHandled{}),
    },
    Handler: func(ctx context.Context, evt protoflow.ProtoMessageContext[*models.OrderCreated]) ([]protoflow.ProtoMessageOutput, error) {
        // Log with the handler's logger
        evt.Logger.Info("Processing order", protoflow.LogFields{"order_id": evt.Payload.OrderId})

        // Clone metadata for safe mutation
        metadata := evt.CloneMetadata().With("handled_by", "proto")

        // Create response
        msg := &models.OrderHandled{OrderId: evt.Payload.OrderId}

        return []protoflow.ProtoMessageOutput{{
            Message:  msg,
            Metadata: metadata,
        }}, nil
    },
})
```

### Handler Registration Fields

| Field | Required | Description |
|:------|:---------|:------------|
| `Name` | Yes | Unique handler identifier |
| `ConsumeQueue` | Yes | Queue/topic to consume from |
| `PublishQueue` | No | Queue/topic for output messages |
| `Handler` | Yes | Handler function |
| `Options` | No | Handler options (e.g., `WithPublishMessageTypes`) |
| `ValidateOutgoing` | No | Validate outgoing messages |

### Handler Options

- **`WithPublishMessageTypes(...proto.Message)`**: Register additional message types for validation at startup

## Registering JSON Handlers

`RegisterJSONHandler` provides the same type safety for JSON payloads:

```go
type IncomingOrder struct {
    ID       string `json:"id"`
    Customer string `json:"customer"`
}

type OutgoingOrder struct {
    ID       string `json:"id"`
    Status   string `json:"status"`
}

err := protoflow.RegisterJSONHandler(svc, protoflow.JSONHandlerRegistration[*IncomingOrder, *OutgoingOrder]{
    Name:         "json-orders",
    ConsumeQueue: "json.orders",
    PublishQueue: "json.audit",
    Handler: func(ctx context.Context, evt protoflow.JSONMessageContext[*IncomingOrder]) ([]protoflow.JSONMessageOutput[*OutgoingOrder], error) {
        evt.Logger.Info("Processing JSON order", protoflow.LogFields{"id": evt.Payload.ID})
        
        return []protoflow.JSONMessageOutput[*OutgoingOrder]{
            {Message: &OutgoingOrder{ID: evt.Payload.ID, Status: "processed"}},
        }, nil
    },
})
```

**Note:** JSON types must be pointer types (`*IncomingOrder`, not `IncomingOrder`).

## Message Context

Both `ProtoMessageContext` and `JSONMessageContext` embed `MessageContextBase`:

```go
type MessageContextBase struct {
    Metadata protoflow.Metadata      // Incoming message metadata
    Logger   protoflow.ServiceLogger // Handler-scoped logger
}
```

### Available Methods

| Method | Description |
|:-------|:------------|
| `evt.CloneMetadata()` | Returns a deep copy of metadata for safe mutation |
| `evt.Get(key)` | Get a metadata value by key |
| `evt.CorrelationID()` | Get the correlation ID from metadata |
| `evt.Logger` | Access the handler's logger |
| `evt.Payload` | Access the typed message payload |

### Metadata Helpers

```go
// Clone for safe mutation
metadata := evt.CloneMetadata()

// Add values (returns new map)
metadata = metadata.With("key", "value")

// Access values
correlationID := evt.CorrelationID()
value := evt.Get("custom_key")
```

## Publishing Events Outside Handlers

Use `Service.PublishProto` to publish from HTTP handlers, cron jobs, etc.:

```go
metadata := protoflow.Metadata{
    "event_source": "api",
}

err := svc.PublishProto(ctx, "orders.created", &models.OrderCreated{
    OrderId: "123",
}, metadata)
```

## Raw Message Handlers

For full control, use `RegisterMessageHandler` with raw Watermill messages:

```go
protoflow.RegisterMessageHandler(svc, protoflow.MessageHandlerRegistration{
    Name:         "raw-handler",
    ConsumeQueue: "raw.messages",
    PublishQueue: "raw.processed",
    Handler: func(msg *message.Message) ([]*message.Message, error) {
        // Direct access to Watermill message
        return nil, nil
    },
})
```

## Error Handling

### Retryable Errors

Return an error to trigger the retry middleware:

```go
Handler: func(ctx context.Context, evt ...) ([]..., error) {
    if err := doSomething(); err != nil {
        return nil, err // Will be retried
    }
    return nil, nil
}
```

### Non-Retryable Errors

Return `UnprocessableEventError` to send directly to the poison queue:

```go
Handler: func(ctx context.Context, evt ...) ([]..., error) {
    if !isValid(evt.Payload) {
        return nil, &protoflow.UnprocessableEventError{
            // Message goes to poison queue without retry
        }
    }
    return nil, nil
}
```

## Logging in Handlers

Use the context logger for structured logging:

```go
Handler: func(ctx context.Context, evt protoflow.ProtoMessageContext[*models.Order]) ([]protoflow.ProtoMessageOutput, error) {
    // Simple log
    evt.Logger.Info("Processing order", protoflow.LogFields{
        "order_id": evt.Payload.OrderId,
    })

    // Error logging
    if err := process(); err != nil {
        evt.Logger.Error("Processing failed", err, protoflow.LogFields{
            "order_id": evt.Payload.OrderId,
        })
        return nil, err
    }

    return nil, nil
}
```

## Metadata Constants

Use standard metadata keys:

```go
const (
    protoflow.MetadataKeyCorrelationID = "correlation_id"
    protoflow.MetadataKeyEventSchema   = "event_message_schema"
    protoflow.MetadataKeyQueueDepth    = "queue_depth"
    protoflow.MetadataKeyEnqueuedAt    = "enqueued_at"
    protoflow.MetadataKeyTraceID       = "trace_id"
    protoflow.MetadataKeySpanID        = "span_id"
)
```

## Service Logger Adapters

`NewService` requires a `ServiceLogger`. Wrap your existing logger:

```go
// Standard library slog
logger := protoflow.NewSlogServiceLogger(slog.Default())

// Entry-style loggers (logrus, zerolog)
entry := logrus.NewEntry(logrus.StandardLogger())
logger := protoflow.NewEntryServiceLogger(entry)
```

The logger is passed to all handlers via the message context.
