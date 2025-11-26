# Protoflow CloudEvents Extensions

Protoflow defines a set of CloudEvents extension attributes (prefixed with `pf_`) to express reliability semantics across different transport backends.

## Extension Reference

| Extension | Type | Default | Description |
|-----------|------|---------|-------------|
| `pf_attempt` | int | 0 | Current retry attempt number (1-based) |
| `pf_max_attempts` | int | 3 | Maximum retry attempts allowed |
| `pf_next_attempt_at` | RFC3339/unix | - | Scheduled time for next retry |
| `pf_dead_letter` | bool | false | Event has been moved to DLQ |
| `pf_trace_id` | string | - | Distributed trace ID (W3C compatible) |
| `pf_parent_id` | string | - | Parent span ID for trace correlation |
| `pf_delay_ms` | int64 | 0 | Delay before processing (milliseconds) |
| `pf_event_version` | string | - | Optional event schema version |
| `pf_original_topic` | string | - | Original topic before DLQ routing |
| `pf_error_message` | string | - | Last error message (DLQ events) |
| `pf_correlation_id` | string | - | Request correlation identifier |

## Retry Extensions

### pf_attempt

Current retry attempt number, starting from 1 on first processing.

```go
// Get current attempt
attempt := protoflow.GetAttempt(evt)

// Set attempt (typically done by middleware)
protoflow.SetAttempt(&evt, 2)

// Increment and get new value
newAttempt := protoflow.IncrementAttempt(&evt)
```

### pf_max_attempts

Maximum number of retry attempts before sending to DLQ.

```go
// Get max attempts (returns default if not set)
maxAttempts := protoflow.GetMaxAttempts(evt) // Default: 3

// Set max attempts
protoflow.SetMaxAttempts(&evt, 5)

// Check if exceeded
if protoflow.ExceedsMaxAttempts(evt) {
    // Send to DLQ
}
```

### pf_next_attempt_at

Scheduled time for the next retry attempt. Used by transports that support delayed redelivery.

```go
// Get next attempt time
nextTime := protoflow.GetNextAttemptAt(evt)

// Set absolute time
protoflow.SetNextAttemptAt(&evt, time.Now().Add(5*time.Minute))

// Set relative delay
protoflow.SetNextAttemptAfter(&evt, 30*time.Second)
```

## Dead Letter Extensions

### pf_dead_letter

Boolean flag indicating the event has been moved to a dead letter queue.

```go
// Check if dead-lettered
if protoflow.IsDeadLetter(evt) {
    // Handle DLQ event
}

// Mark as dead-lettered
protoflow.SetDeadLetter(&evt, true)
```

### pf_original_topic

Stores the original topic/event type when an event is moved to DLQ.

```go
// Get original topic
originalTopic := protoflow.GetOriginalTopic(evt)

// Set original topic
protoflow.SetOriginalTopic(&evt, "order.created")
```

### pf_error_message

Last error message when an event failed processing.

```go
// Get error message
errMsg := protoflow.GetErrorMessage(evt)

// Set error message
protoflow.SetErrorMessage(&evt, "payment gateway timeout")
```

## Tracing Extensions

### pf_trace_id

Distributed trace identifier, compatible with W3C Trace Context.

```go
// Get trace ID
traceID := protoflow.GetTraceID(evt)

// Set trace ID
protoflow.SetTraceID(&evt, "0af7651916cd43dd8448eb211c80319c")
```

### pf_parent_id

Parent span ID for trace correlation across services.

```go
// Get parent ID
parentID := protoflow.GetParentID(evt)

// Set parent ID
protoflow.SetParentID(&evt, "b7ad6b7169203331")
```

### pf_correlation_id

Request correlation identifier for tracking requests across service boundaries.

```go
// Get correlation ID
correlationID := protoflow.GetCorrelationID(evt)

// Set correlation ID
protoflow.SetCorrelationID(&evt, "req-abc-123")
```

## Delay Extensions

### pf_delay_ms

Delay in milliseconds before the event should be processed.

```go
// Get delay in milliseconds
delayMs := protoflow.GetDelayMs(evt)

// Set delay in milliseconds
protoflow.SetDelayMs(&evt, 5000) // 5 seconds

// Get as time.Duration
delay := protoflow.GetDelay(evt)

// Set from time.Duration
protoflow.SetDelay(&evt, 5*time.Minute)
```

## Version Extensions

### pf_event_version

Optional version number for the event schema.

```go
// Get event version
version := protoflow.GetEventVersion(evt)

// Set event version
protoflow.SetEventVersion(&evt, "2.0")
```

## Helper Functions

### PrepareForRetry

Prepares an event for retry by incrementing attempt counter and setting next attempt time.

```go
protoflow.PrepareForRetry(&evt, 30*time.Second)
// Equivalent to:
// protoflow.IncrementAttempt(&evt)
// protoflow.SetNextAttemptAfter(&evt, 30*time.Second)
```

### PrepareForDLQ

Prepares an event for dead letter queue routing.

```go
protoflow.PrepareForDLQ(&evt, "order.created", err)
// Sets:
// - pf_dead_letter = true
// - pf_original_topic = "order.created"
// - pf_error_message = err.Error()
```

### DLQTopic

Returns the dead letter queue topic name for an event type.

```go
dlqTopic := protoflow.DLQTopic("order.created")
// Returns: "order.created.dead"
```

### CopyTracingContext

Copies tracing extensions from source to destination event.

```go
protoflow.CopyTracingContext(srcEvt, &dstEvt)
// Copies: pf_trace_id, pf_parent_id, pf_correlation_id
```

## Example Event with Extensions

```json
{
  "specversion": "1.0",
  "type": "order.created",
  "source": "order-service",
  "id": "01HX7QDXKS8MN4VKXQJF3J6YRP",
  "time": "2024-01-15T10:30:00Z",
  "data": {
    "order_id": "ORD-001"
  },
  "pf_attempt": 2,
  "pf_max_attempts": 5,
  "pf_next_attempt_at": "2024-01-15T10:31:00Z",
  "pf_trace_id": "0af7651916cd43dd8448eb211c80319c",
  "pf_parent_id": "b7ad6b7169203331",
  "pf_correlation_id": "req-abc-123",
  "pf_delay_ms": 5000
}
```

## Transport Considerations

Different transports may handle extensions differently:

| Transport | Native Delay | Native DLQ | Extension Storage |
|-----------|--------------|------------|-------------------|
| NATS JetStream | ✅ Headers | ✅ Max Deliver | Message Headers |
| Kafka | ❌ Emulated | ❌ Emulated | Message Headers |
| RabbitMQ | ✅ TTL/Plugin | ✅ DLX | Message Properties |
| AWS SQS | ✅ DelaySeconds | ✅ Redrive | Message Attributes |
| SQLite | ✅ available_at | ✅ DLQ table | JSON metadata |
| PostgreSQL | ✅ scheduled_at | ✅ DLQ table | JSONB metadata |

When a transport doesn't support native delayed delivery, protoflow emulates it at the application level using the `pf_delay_ms` and `pf_next_attempt_at` extensions.
