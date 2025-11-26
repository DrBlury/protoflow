# CloudEvents Model

Protoflow standardizes on [CloudEvents v1.0](https://github.com/cloudevents/spec/blob/v1.0/spec.md) as its canonical event model. Protoflow reliability semantics are expressed via CloudEvents Extensions rather than inventing a custom envelope format.

## Event Structure

```go
type Event struct {
    // Required attributes
    SpecVersion     string            // Always "1.0"
    Type            string            // Event type (e.g., "order.created")
    Source          string            // Event source (e.g., "order-service")
    ID              string            // Unique event ID (ULID)

    // Optional attributes
    Time            time.Time         // When the event occurred
    DataContentType *string           // Content type (e.g., "application/json")
    DataSchema      *string           // Schema URI for data validation
    Subject         *string           // Subject of the event

    // Payload
    Data            any               // Event data (JSON-serializable)
    DataBase64      *string           // Base64-encoded binary data

    // Extensions
    Extensions      map[string]any    // Custom extension attributes
}
```

## JSON Format

CloudEvents are serialized as JSON with extensions flattened into the top-level object:

```json
{
  "specversion": "1.0",
  "type": "order.created",
  "source": "order-service",
  "id": "01HX7QDXKS8MN4VKXQJF3J6YRP",
  "time": "2024-01-15T10:30:00Z",
  "datacontenttype": "application/json",
  "subject": "orders/ORD-001",
  "data": {
    "order_id": "ORD-001",
    "customer_id": "CUST-123",
    "amount": 99.99,
    "currency": "USD"
  },
  "pf_attempt": 1,
  "pf_max_attempts": 3,
  "pf_trace_id": "abc123"
}
```

## Creating Events

### Basic Event

```go
import "github.com/drblury/protoflow"

// Create a new CloudEvent
evt := protoflow.NewCloudEvent("order.created", "order-service", OrderData{
    OrderID:    "ORD-001",
    CustomerID: "CUST-123",
    Amount:     99.99,
})
```

### With Optional Attributes

```go
evt := protoflow.NewCloudEvent("order.created", "order-service", data).
    WithSubject("orders/ORD-001").
    WithDataContentType("application/json").
    WithDataSchema("https://schemas.example.com/order/v1.json")
```

### With Extensions

```go
evt := protoflow.NewCloudEvent("order.created", "order-service", data).
    WithExtension("customer_tier", "premium").
    WithExtension("region", "us-east-1")

// Or use protoflow extension helpers
protoflow.SetCorrelationID(&evt, "req-123")
protoflow.SetTraceID(&evt, "trace-abc")
```

## Event Type Naming Convention

Protoflow recommends the following naming convention:

```
<resource>.<action>[.v<version>]
```

| Event Type | Description |
|------------|-------------|
| `customer.created` | New customer registered |
| `customer.updated` | Customer data changed |
| `customer.updated.v2` | Breaking change to update event |
| `order.placed` | Order submitted |
| `order.confirmed` | Order confirmed |
| `invoice.paid` | Invoice payment received |
| `user.deleted` | User account removed |

### Versioning Strategy

For breaking changes:
1. **Bump version in event type** (recommended): `customer.updated` → `customer.updated.v2`
2. **Use pf_event_version extension** (optional): Keep type same, add `"pf_event_version": "2"`

## Event Validation

Events are validated before publishing:

```go
err := evt.Validate()
if err != nil {
    // Handle validation error
}
```

Required fields:
- `specversion` must be "1.0"
- `type` must not be empty
- `source` must not be empty
- `id` must not be empty (auto-generated if not provided)

## Publishing Events

```go
// Publish immediately
err := svc.PublishEvent(ctx, evt)

// Publish with delay
err := svc.PublishEventAfter(ctx, evt, 5*time.Minute)

// Convenience API
err := svc.PublishData(ctx, "order.created", "order-service", data,
    protoflow.WithCorrelationID("req-123"),
    protoflow.WithSubject("orders/ORD-001"),
)
```

## Consuming Events

```go
err := svc.ConsumeEvents("order.created", func(ctx context.Context, evt protoflow.Event) error {
    // Access event attributes
    log.Printf("Event: %s from %s", evt.Type, evt.Source)

    // Access data (type assertion required)
    if data, ok := evt.Data.(map[string]any); ok {
        orderID := data["order_id"].(string)
    }

    // Access extensions
    correlationID := protoflow.GetCorrelationID(evt)
    attempt := protoflow.GetAttempt(evt)

    return nil // Acknowledge
})
```

## Compatibility with Legacy Code

Protoflow maintains backward compatibility. The new CloudEvents API works alongside existing proto/JSON handlers:

```go
// Legacy proto handler (still works)
protoflow.RegisterProtoHandler(svc, protoflow.ProtoHandlerRegistration[*pb.Order]{
    Name:         "order-handler",
    ConsumeQueue: "orders",
    Handler:      handleOrder,
})

// New CloudEvents handler
svc.ConsumeEvents("order.created", func(ctx context.Context, evt protoflow.Event) error {
    // ...
})
```

Both can coexist in the same service.
