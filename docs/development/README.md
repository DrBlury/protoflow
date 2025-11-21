# Development Guide

Use these notes when running Protoflow locally, executing tests, or contributing changes.

## Taskfile workflows

`taskfile.yml` defines repeatable commands so you do not have to remember long `go` invocations.

Common targets:

| Command | Description |
| --- | --- |
| `task lint` | Runs `golangci-lint` with the repo configuration. |
| `task test` | Executes `go test ./...` and surfaces failures quickly. |
| `task examples:<name>` | Runs one of the runnable scenarios from `examples/`. |

Run `task --list` to discover all available tasks.

## Local broker tips

- **AWS SNS/SQS**: point `Config.AWSEndpoint` at your LocalStack URL and omit credentials for the default provider chain.
- **Kafka**: pick a unique `KafkaConsumerGroup` per service instance to avoid partitions getting stuck.
- **RabbitMQ**: the same AMQP connection backs both publisher and subscriber; supply TLS config via `amqp.ConnectionConfig` if needed.

## Testing

Unit tests cover service wiring, middleware, transports, and typed handler helpers. Run them with either `task test` or directly with:

```bash
go test ./...
```

Use `go test ./internal/runtime/...` when iterating on a specific subsystem.

When you need code coverage, exclude the runnable examples so their `main` packages do not skew the report:

```bash
go test $(go list ./... | grep -v '/examples/') -coverprofile=coverage.out -covermode=atomic
```

The examples are still build-tested via `go test ./examples/...` whenever you want to verify them explicitly.
