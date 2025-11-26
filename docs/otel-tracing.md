# OpenTelemetry Tracing

Protoflow integrates with OpenTelemetry for distributed tracing across services and message brokers.

## Overview

Protoflow's tracing support:

- Propagates trace context via CloudEvents extensions
- Maps to W3C Trace Context standard
- Works across retries and delayed delivery
- Integrates with existing OTEL infrastructure

## CloudEvents ↔ W3C Trace Context Mapping

| CloudEvents Extension | W3C Header | Description |
|----------------------|------------|-------------|
| `pf_trace_id` | `traceparent` (trace-id part) | 32-character hex trace ID |
| `pf_parent_id` | `traceparent` (parent-id part) | 16-character hex span ID |

## Basic Setup

### 1. Configure OpenTelemetry

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func initTracer() (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(context.Background(),
        otlptracegrpc.WithEndpoint("localhost:4317"),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName("my-service"),
        )),
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### 2. Enable Tracer Middleware

```go
svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{})

// TracerMiddleware is included in DefaultMiddlewares()
// It automatically creates spans for message processing
```

### 3. Propagate Trace Context

```go
// When publishing, set trace context
evt := protoflow.NewCloudEvent("order.created", "order-service", data)
protoflow.SetTraceID(&evt, "0af7651916cd43dd8448eb211c80319c")
protoflow.SetParentID(&evt, "b7ad6b7169203331")

// Or extract from current span
span := trace.SpanFromContext(ctx)
if span.SpanContext().IsValid() {
    protoflow.SetTraceID(&evt, span.SpanContext().TraceID().String())
    protoflow.SetParentID(&evt, span.SpanContext().SpanID().String())
}

err := svc.PublishEvent(ctx, evt)
```

### 4. Access Trace Context in Handlers

```go
svc.ConsumeEvents("order.created", func(ctx context.Context, evt protoflow.Event) error {
    // Get trace context from event
    traceID := protoflow.GetTraceID(evt)
    parentID := protoflow.GetParentID(evt)

    // Create child span
    tracer := otel.Tracer("order-handler")
    ctx, span := tracer.Start(ctx, "process-order")
    defer span.End()

    // Add event attributes
    span.SetAttributes(
        attribute.String("event.id", evt.ID),
        attribute.String("event.type", evt.Type),
    )

    // Process...
    return nil
})
```

## Automatic Span Creation

The `TracerMiddleware` automatically:

1. Extracts trace context from incoming messages
2. Creates a processing span
3. Adds standard attributes
4. Records errors and exceptions

```go
// Span attributes added automatically:
// - messaging.system: transport name
// - messaging.destination: topic/queue
// - messaging.message_id: event ID
// - messaging.operation: "process"
```

## Trace Context Across Retries

Trace context is preserved across retries:

```go
svc.ConsumeEvents("payment.process", func(ctx context.Context, evt protoflow.Event) error {
    // Same trace_id across all retry attempts
    traceID := protoflow.GetTraceID(evt)
    attempt := protoflow.GetAttempt(evt)

    log.Printf("Processing attempt %d, trace: %s", attempt, traceID)

    if shouldRetry {
        return protoflow.ErrRetryAfter(5*time.Second, err)
    }
    return nil
})
```

## Trace Context Across Services

When publishing response events, copy the trace context:

```go
svc.ConsumeEvents("order.created", func(ctx context.Context, evt protoflow.Event) error {
    // Process order...

    // Create response event with same trace context
    responseEvt := protoflow.NewCloudEvent("order.confirmed", "order-service", data)
    protoflow.CopyTracingContext(evt, &responseEvt)

    return svc.PublishEvent(ctx, responseEvt)
})
```

## Correlation IDs

In addition to trace IDs, use correlation IDs for business-level request tracking:

```go
// At API gateway
evt := protoflow.NewCloudEvent("order.created", "api-gateway", data)
protoflow.SetCorrelationID(&evt, requestID) // From HTTP header
protoflow.SetTraceID(&evt, traceID)         // From OTEL

// In downstream handlers
correlationID := protoflow.GetCorrelationID(evt)
log.Printf("Processing request: %s", correlationID)
```

## Integration with Jaeger

```yaml
# docker-compose.yml
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # UI
      - "14268:14268"  # Collector HTTP
      - "4317:4317"    # OTLP gRPC
```

```go
// Configure OTEL to export to Jaeger
exporter, err := otlptracegrpc.New(context.Background(),
    otlptracegrpc.WithEndpoint("localhost:4317"),
    otlptracegrpc.WithInsecure(),
)
```

## Integration with Datadog

```go
import "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentelemetry"

func initTracer() {
    provider := opentelemetry.NewTracerProvider()
    otel.SetTracerProvider(provider)
}
```

## Transport-Specific Considerations

| Transport | Trace Propagation |
|-----------|-------------------|
| Kafka | Message headers |
| RabbitMQ | Message properties |
| NATS/JetStream | Message headers |
| AWS SQS | Message attributes |
| SQLite/Postgres | JSON metadata |

## Best Practices

1. **Always propagate trace context** when publishing response events
2. **Use correlation IDs** for business-level tracking alongside trace IDs
3. **Add meaningful attributes** to spans (order ID, customer ID, etc.)
4. **Record errors** with appropriate error types and messages
5. **Sample appropriately** in production to control costs

```go
// Example: Adding business context to spans
span.SetAttributes(
    attribute.String("order.id", order.ID),
    attribute.String("customer.id", order.CustomerID),
    attribute.Float64("order.amount", order.Amount),
)

// Record errors with context
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return protoflow.ErrRetryAfter(5*time.Second, err)
}
```

## Debugging Traces

Enable verbose logging to see trace propagation:

```go
logger := protoflow.NewSlogServiceLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})))
```

Look for log entries with `trace_id` and `span_id` fields.
