# Configuration Guide

Protoflow centralizes broker wiring, middleware, and extension points behind `protoflow.Config` plus `protoflow.ServiceDependencies`. Use this guide to select transports, tweak defaults, and plug in custom collaborators.

## Transport selection

Set `Config.PubSubSystem` to one of the built-in transports and fill the matching fields.

### Kafka (`"kafka"`)

| Field | Purpose |
| --- | --- |
| `KafkaBrokers` | Broker list (e.g. `[]string{"localhost:9092"}`) |
| `KafkaConsumerGroup` | Consumer group name for handler subscriptions |
| `KafkaClientID` | Optional producer identifier |

### RabbitMQ (`"rabbitmq"`)

| Field | Purpose |
| --- | --- |
| `RabbitMQURL` | Connection URL, e.g. `amqp://user:pass@host:5672/vhost` |

### AWS SNS/SQS (`"aws"`)

| Field | Purpose |
| --- | --- |
| `AWSRegion` | Region used for SNS and SQS clients |
| `AWSAccountID` | Used when composing ARNs; defaults to `000000000000` for LocalStack |
| `AWSEndpoint` | Override endpoint for LocalStack or private gateways |
| `AWSAccessKeyID` / `AWSSecretAccessKey` | Optional explicit credentials (otherwise default AWS chain is used) |

## Common knobs

| Field | Description |
| --- | --- |
| `PoisonQueue` | Queue/topic that receives messages after retries are exhausted |
| `RetryMaxRetries` | Maximum retry attempts (default: exponential backoff config) |
| `RetryInitialInterval` | First retry delay |
| `RetryMaxInterval` | Upper bound for retry backoff |

Leave zero values to keep Protoflow defaults.

## Middleware and dependencies

`protoflow.ServiceDependencies` lets you bring your own collaborators and customize middleware ordering.

- `Validator` (`ProtoValidator`) — runs inside `ProtoValidateMiddleware` for incoming and outgoing protobufs.
- `Outbox` (`OutboxStore`) — stores emitted events before forwarding them to the transport.
- `Middlewares` (`[]MiddlewareRegistration`) — append or prepend middleware to the default stack.
- `DisableDefaultMiddlewares` — skip correlation IDs, logging, validation, outbox, OpenTelemetry, retries, poison queue, and panic recovery so you can register everything manually.

Default middleware order:

1. Correlation ID injector
2. Message logger
3. Proto validation
4. Outbox persistence
5. OpenTelemetry tracing
6. Retry with exponential backoff
7. Poison queue forwarder
8. Panic recoverer

## Custom transports

`ServiceDependencies.TransportFactory` replaces the built-in transport picker. The factory receives the context, resolved config, and the Watermill logger adapter. Return a `protoflow.Transport` with both publisher and subscriber instances.

```go
type gcppubsubFactory struct{ client *pubsub.Client }

func (f gcppubsubFactory) Build(ctx context.Context, conf *protoflow.Config, logger watermill.LoggerAdapter) (protoflow.Transport, error) {
    pub := newPubSubPublisher(f.client, logger)
    sub := newPubSubSubscriber(f.client, conf, logger)
    return protoflow.Transport{Publisher: pub, Subscriber: sub}, nil
}

svc := protoflow.NewService(cfg, logger, ctx,
    protoflow.ServiceDependencies{TransportFactory: gcppubsubFactory{client: client}},
)
```

This pattern lets you integrate brokers such as GCP Pub/Sub, NATS, Redis Streams, or in-memory transports for tests without forking Protoflow.
