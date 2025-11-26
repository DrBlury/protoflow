# Roadmap: Delayed Delivery

This document outlines the delayed delivery capabilities across protoflow transports and the roadmap for improvements.

## Current Status

| Transport | Delay Support | Implementation | Max Delay |
|-----------|---------------|----------------|-----------|
| SQLite | ✅ Native | `available_at` column | Unlimited |
| PostgreSQL | ✅ Native | `scheduled_at` column | Unlimited |
| NATS JetStream | ✅ Native | NAK with delay | Unlimited |
| AWS SQS | ✅ Native | DelaySeconds | 15 minutes |
| RabbitMQ | ✅ Native | TTL + DLX or plugin | Varies |
| Kafka | ❌ Emulated | Application-level | N/A |
| NATS Core | ❌ Not supported | N/A | N/A |
| Channel | ❌ Not supported | N/A | N/A |

## Usage

### Using PublishEventAfter

```go
// Publish with delay
evt := protoflow.NewCloudEvent("notification.send", "notification-service", data)
err := svc.PublishEventAfter(ctx, evt, 5*time.Minute)
```

### Using pf_delay_ms Extension

```go
evt := protoflow.NewCloudEvent("reminder.send", "reminder-service", data)
protoflow.SetDelay(&evt, 24*time.Hour)
err := svc.PublishEvent(ctx, evt)
```

### Using Metadata (Legacy)

```go
msg := message.NewMessage(protoflow.CreateULID(), payload)
msg.Metadata["protoflow_delay"] = "5m"
err := svc.Publish(ctx, "topic", msg)
```

## Transport Implementations

### SQLite

Messages are stored with an `available_at` timestamp:

```sql
CREATE TABLE messages (
    id INTEGER PRIMARY KEY,
    uuid TEXT NOT NULL,
    topic TEXT NOT NULL,
    payload BLOB NOT NULL,
    available_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- ...
);

-- Only fetch available messages
SELECT * FROM messages
WHERE status = 'pending'
AND available_at <= CURRENT_TIMESTAMP;
```

### PostgreSQL

Similar to SQLite with optimized indexing:

```sql
CREATE TABLE protoflow_messages (
    id BIGSERIAL PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),
    -- ...
);

CREATE INDEX idx_scheduled ON protoflow_messages(topic, scheduled_at)
WHERE status = 'pending';
```

### NATS JetStream

Uses NAK with delay for redelivery:

```go
// In consumer
if !isReady(msg) {
    msg.NakWithDelay(remainingDelay)
    continue
}
```

Alternatively, use message headers:

```go
headers := nats.Header{}
headers.Set("pf_delay_until", strconv.FormatInt(delayUntil, 10))
```

### AWS SQS

Uses SQS DelaySeconds (limited to 15 minutes):

```go
_, err := sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
    QueueUrl:     &queueURL,
    MessageBody:  &body,
    DelaySeconds: int32(delay.Seconds()),
})
```

For delays > 15 minutes, use scheduled Lambda or Step Functions.

### RabbitMQ

Two approaches:

**Option 1: Delayed Message Exchange Plugin**

```go
// Requires rabbitmq_delayed_message_exchange plugin
headers := amqp.Table{
    "x-delay": int64(delay.Milliseconds()),
}
```

**Option 2: Per-Message TTL**

```go
// Publish to delay queue with TTL, DLX routes to target
publishing := amqp.Publishing{
    Expiration: strconv.FormatInt(delay.Milliseconds(), 10),
    // ...
}
```

## Roadmap

### Phase 1: Current (v1.x)

- [x] SQLite delayed delivery
- [x] PostgreSQL delayed delivery
- [x] NATS JetStream delayed delivery
- [x] CloudEvents `pf_delay_ms` extension
- [x] `PublishEventAfter` API

### Phase 2: Enhanced (v2.x)

- [ ] AWS SQS delayed delivery integration
- [ ] RabbitMQ delayed message exchange support
- [ ] Kafka delay emulation via compacted topics
- [ ] Scheduled job API for long delays

```go
// Proposed API for scheduled jobs
svc.Schedule(ctx, "notification.send", data, time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC))
```

### Phase 3: Advanced (v3.x)

- [ ] Cron-style recurring events
- [ ] Cancel/reschedule delayed events
- [ ] Delay queue management UI
- [ ] Metrics for delayed message visibility

```go
// Proposed API for recurring events
svc.ScheduleRecurring(ctx, "report.generate", data, "0 9 * * MON")

// Cancel a scheduled event
svc.CancelScheduled(ctx, eventID)
```

## Best Practices

### 1. Check Transport Capabilities

```go
caps := svc.GetTransportCapabilities()
if !caps.SupportsDelay {
    // Use external scheduler or warn
    log.Warn("Transport doesn't support delay, using immediate delivery")
}
```

### 2. Handle Long Delays

For delays longer than transport limits:

```go
const maxSQSDelay = 15 * time.Minute

func publishWithDelay(svc *protoflow.Service, evt protoflow.Event, delay time.Duration) error {
    if delay > maxSQSDelay {
        // Store in database and use scheduler
        return scheduleForLater(evt, time.Now().Add(delay))
    }
    return svc.PublishEventAfter(ctx, evt, delay)
}
```

### 3. Idempotent Handlers

Delayed messages may be delivered multiple times:

```go
svc.ConsumeEvents("reminder.send", func(ctx context.Context, evt protoflow.Event) error {
    // Check if already processed
    if isProcessed(evt.ID) {
        return protoflow.ErrSkip
    }

    // Process and mark as done
    if err := process(evt); err != nil {
        return err
    }

    return markProcessed(evt.ID)
})
```

### 4. Monitor Delay Queues

Track delayed message metrics:

```go
// Prometheus metrics
delayedMessagesGauge.WithLabelValues(topic).Set(float64(count))
delayHistogram.WithLabelValues(topic).Observe(delay.Seconds())
```

## Known Limitations

| Transport | Limitation | Workaround |
|-----------|------------|------------|
| AWS SQS | Max 15 min delay | Use Step Functions for longer |
| Kafka | No native delay | Use time-bucketed topics |
| NATS Core | No delay support | Upgrade to JetStream |
| RabbitMQ | Requires plugin | Use TTL + DLX pattern |

## Related Documentation

- [Transport Capabilities](transport-capabilities.md)
- [Protoflow Extensions](protoflow-extensions.md)
- [CloudEvents Model](cloudevents-model.md)
