# NATS JetStream CloudEvents Example

This example demonstrates protoflow's CloudEvents integration with NATS JetStream, including:

- **CloudEvents v1.0 Compliance**: Standard event format with protoflow extensions
- **Delayed Message Delivery**: Schedule events for future processing
- **Automatic Retries**: Exponential backoff with configurable max attempts
- **Dead Letter Queue**: Automatic routing of failed messages
- **Tracing Context**: Propagation across service boundaries

## Quick Start

### 1. Start NATS JetStream

```bash
docker-compose up -d
```

### 2. Run the Example

```bash
go run main.go
```

### 3. What Happens

1. **order.created** event is published and processed
2. Handler responds by publishing a **payment.initiated** event
3. Payment handler simulates failures and retries with explicit delays
4. **notification.scheduled** event is published with a 5-second delay
5. **order.problematic** event is published and eventually sent to DLQ

## CloudEvents Format

Events follow the [CloudEvents v1.0 specification](https://github.com/cloudevents/spec/blob/v1.0/spec.md):

```json
{
  "specversion": "1.0",
  "type": "order.created",
  "source": "api-gateway",
  "id": "01HX...",
  "time": "2024-01-15T10:30:00Z",
  "datacontenttype": "application/json",
  "data": {
    "order_id": "ORD-001",
    "customer_id": "CUST-123",
    "amount": 99.99,
    "currency": "USD"
  },
  "pf_attempt": 1,
  "pf_max_attempts": 3,
  "pf_trace_id": "trace-xyz-789",
  "pf_correlation_id": "req-abc-123"
}
```

## Protoflow Extensions

| Extension | Type | Description |
|-----------|------|-------------|
| `pf_attempt` | int | Current retry attempt (1-based) |
| `pf_max_attempts` | int | Maximum retry attempts allowed |
| `pf_next_attempt_at` | RFC3339 | Scheduled time for next retry |
| `pf_dead_letter` | bool | Whether event is in DLQ |
| `pf_trace_id` | string | W3C trace ID |
| `pf_parent_id` | string | Parent span ID |
| `pf_delay_ms` | int64 | Delay before processing (ms) |
| `pf_correlation_id` | string | Request correlation ID |

## Handler Return Semantics

```go
// Acknowledge (success)
return nil

// Retry with default delay
return protoflow.ErrRetry

// Retry after specific delay
return protoflow.ErrRetryAfter(5*time.Second, nil)

// Send to DLQ immediately
return protoflow.ErrDeadLetter

// Send to DLQ with reason
return protoflow.ErrDeadLetterWithReason("validation failed", originalErr)
```

## API Examples

### Publishing Events

```go
// Create and publish a CloudEvent
evt := protoflow.NewCloudEvent("order.created", "my-service", orderData)
protoflow.SetCorrelationID(&evt, "request-123")
err := svc.PublishEvent(ctx, evt)

// Publish with delay
err := svc.PublishEventAfter(ctx, evt, 5*time.Minute)

// Convenience API
err := svc.PublishData(ctx, "order.created", "my-service", orderData,
    protoflow.WithCorrelationID("request-123"),
    protoflow.WithSubject("orders/ORD-001"),
)
```

### Consuming Events

```go
err := svc.ConsumeEvents("order.created", func(ctx context.Context, evt protoflow.Event) error {
    // Access event metadata
    log.Printf("Event ID: %s, Type: %s", evt.ID, evt.Type)

    // Access protoflow extensions
    attempt := protoflow.GetAttempt(evt)
    traceID := protoflow.GetTraceID(evt)

    // Parse data
    var order OrderData
    // ... unmarshal evt.Data

    return nil // Acknowledge
})
```

## Transport Capabilities

Check what the transport supports at runtime:

```go
caps := svc.GetTransportCapabilities()
if caps.SupportsDelay {
    // Use native delayed delivery
}
if caps.SupportsNativeDLQ {
    // Transport handles DLQ routing
}
```

## Cleanup

```bash
docker-compose down -v
```
