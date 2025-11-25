# Configuration Guide

Protoflow centralizes broker wiring, middleware, and dependencies via `protoflow.Config` and `protoflow.ServiceDependencies`.

## Transport Selection

Set `Config.PubSubSystem` to select a transport:

### Go Channel (`"channel"`)

In-memory transport for testing and local development. No configuration needed.

```go
cfg := &protoflow.Config{
    PubSubSystem: "channel",
    PoisonQueue:  "failed.messages",
}
```

### Kafka (`"kafka"`)

| Field | Purpose |
|:------|:--------|
| `KafkaBrokers` | List of brokers (e.g., `[]string{"localhost:9092"}`) |
| `KafkaConsumerGroup` | Consumer group name |
| `KafkaClientID` | Optional producer identifier |

```go
cfg := &protoflow.Config{
    PubSubSystem:       "kafka",
    KafkaBrokers:       []string{"localhost:9092"},
    KafkaConsumerGroup: "my-service",
    PoisonQueue:        "failed.messages",
}
```

### RabbitMQ (`"rabbitmq"`)

| Field | Purpose |
|:------|:--------|
| `RabbitMQURL` | Connection URL (e.g., `amqp://user:pass@host:5672/vhost`) |

```go
cfg := &protoflow.Config{
    PubSubSystem: "rabbitmq",
    RabbitMQURL:  "amqp://guest:guest@localhost:5672/",
    PoisonQueue:  "failed.messages",
}
```

**Note:** Credentials in URLs are automatically redacted in logs.

### AWS SNS/SQS (`"aws"`)

| Field | Purpose |
|:------|:--------|
| `AWSRegion` | AWS Region (required) |
| `AWSAccountID` | Account ID for ARNs |
| `AWSEndpoint` | Custom endpoint for LocalStack |
| `AWSAccessKeyID` | Explicit credentials (optional) |
| `AWSSecretAccessKey` | Explicit credentials (optional) |

```go
cfg := &protoflow.Config{
    PubSubSystem: "aws",
    AWSRegion:    "us-east-1",
    AWSAccountID: "123456789012",
    AWSEndpoint:  "http://localhost:4566", // LocalStack
    PoisonQueue:  "failed-messages",
}
```

### NATS (`"nats"`)

| Field | Purpose |
|:------|:--------|
| `NATSURL` | Connection URL (e.g., `nats://localhost:4222`) |

```go
cfg := &protoflow.Config{
    PubSubSystem: "nats",
    NATSURL:      "nats://localhost:4222",
    PoisonQueue:  "failed.messages",
}
```

### HTTP (`"http"`)

| Field | Purpose |
|:------|:--------|
| `HTTPServerAddress` | Server address for receiving messages |
| `HTTPPublisherURL` | Base URL for publishing messages |

```go
cfg := &protoflow.Config{
    PubSubSystem:      "http",
    HTTPServerAddress: ":8080",
    HTTPPublisherURL:  "http://localhost:8080",
    PoisonQueue:       "failed.messages",
}
```

### File I/O (`"io"`)

| Field | Purpose |
|:------|:--------|
| `IOFile` | File path for message persistence (default: `messages.log`) |

```go
cfg := &protoflow.Config{
    PubSubSystem: "io",
    IOFile:       "/var/log/messages.log",
    PoisonQueue:  "failed.messages",
}
```

### SQLite (`"sqlite"`)

Embedded database transport with delayed message support and built-in DLQ management.

| Field | Purpose |
|:------|:--------|
| `SQLiteFile` | Path to SQLite database file (use `:memory:` for in-memory) |

```go
cfg := &protoflow.Config{
    PubSubSystem: "sqlite",
    SQLiteFile:   "/var/lib/myapp/queue.db",
    PoisonQueue:  "failed.messages",
}
```

**Delayed Messages:** SQLite supports scheduling messages for future processing:

```go
msg := message.NewMessage(protoflow.CreateULID(), payload)
msg.Metadata.Set("protoflow_delay", "30s")  // Process after 30 seconds
msg.Metadata.Set("protoflow_delay", "5m")   // Process after 5 minutes
```

**Note:** See [Transport Comparison Guide](../transports/README.md) for detailed feature comparison.

## Common Configuration

| Field | Description |
|:------|:------------|
| `PoisonQueue` | Dead letter queue for failed messages |
| `RetryMaxRetries` | Max retry attempts (default: 5) |
| `RetryInitialInterval` | Initial retry delay (default: 1s) |
| `RetryMaxInterval` | Max retry backoff (default: 16s) |
| `MetricsEnabled` | Enable Prometheus metrics |
| `MetricsPort` | Port for `/metrics` endpoint |
| `WebUIEnabled` | Enable handler introspection API |
| `WebUIPort` | Port for WebUI API (default: 8081) |
| `WebUICORSAllowedOrigins` | Allowed origins for CORS (e.g., `["*"]` or `["https://example.com"]`) |

## Configuration Validation

Protoflow validates configuration at service creation:

```go
// Panics on invalid config
svc := protoflow.NewService(cfg, logger, ctx, deps)

// Returns error on invalid config
svc, err := protoflow.TryNewService(cfg, logger, ctx, deps)
if err != nil {
    var validationErr protoflow.ConfigValidationError
    if errors.As(err, &validationErr) {
        // Handle validation error
    }
}
```

You can also validate manually:

```go
if err := protoflow.ValidateConfig(cfg); err != nil {
    log.Fatal(err)
}
```

## Service Dependencies

`protoflow.ServiceDependencies` injects custom behavior:

```go
svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{
    Validator:                 myProtoValidator,
    Outbox:                    myOutboxStore,
    Middlewares:               []protoflow.MiddlewareRegistration{...},
    DisableDefaultMiddlewares: false,
    TransportFactory:          myCustomFactory,
    ErrorClassifier:           myErrorClassifier,
})
```

### Available Dependencies

| Field | Type | Purpose |
|:------|:-----|:--------|
| `Validator` | `ProtoValidator` | Custom protobuf validation |
| `Outbox` | `OutboxStore` | Persist events before publishing |
| `Middlewares` | `[]MiddlewareRegistration` | Additional middleware |
| `DisableDefaultMiddlewares` | `bool` | Skip default middleware stack |
| `TransportFactory` | `TransportFactory` | Custom transport implementation |
| `ErrorClassifier` | `ErrorClassifier` | Classify errors for retry logic |

## Custom Transports

Implement `TransportFactory` to add your own transport:

```go
type gcpPubSubFactory struct {
    client *pubsub.Client
}

func (f *gcpPubSubFactory) Build(ctx context.Context, conf *protoflow.Config, logger watermill.LoggerAdapter) (protoflow.Transport, error) {
    pub := newGCPPublisher(f.client, logger)
    sub := newGCPSubscriber(f.client, conf, logger)
    return protoflow.Transport{Publisher: pub, Subscriber: sub}, nil
}

svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{
    TransportFactory: &gcpPubSubFactory{client: client},
})
```

## Default Middleware Order

1. Correlation ID injector
2. Message logger
3. Proto validation
4. Outbox persistence
5. OpenTelemetry tracing
6. Prometheus metrics
7. Retry with exponential backoff
8. Poison queue forwarder
9. Panic recoverer

Override by setting `DisableDefaultMiddlewares: true` and providing your own `Middlewares` slice.
