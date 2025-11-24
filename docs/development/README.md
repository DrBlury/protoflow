# Development Guide

This guide explains how to set up your environment, run tests, and contribute to Protoflow.

## Taskfile Workflows

We use `task` (Taskfile) to automate common development tasks.

| Command | Description |
| :--- | :--- |
| `task lint` | Run `golangci-lint` to check code quality. |
| `task test` | Run the full test suite (`go test ./...`). |
| `task examples:<name>` | Run a specific example (e.g., `task examples:simple`). |

*Run `task --list` to see all available commands.*

## Local Broker Configuration

Running brokers locally can require specific configurations:

- **AWS SNS/SQS**: Point `Config.AWSEndpoint` to your LocalStack URL. The default provider chain handles credentials (or use dummy values).
- **Kafka**: Use a unique `KafkaConsumerGroup` for each service instance to avoid rebalancing issues during development.
- **RabbitMQ**: The same AMQP connection is used for both publishing and subscribing. If TLS is required, supply it via `amqp.ConnectionConfig`.

## Testing

Our test suite covers service wiring, middleware, transports, and helpers.

**Run all tests:**

```bash
task test
# OR
go test ./...
```

**Run specific tests:**

```bash
go test ./internal/runtime/...
```

**Check coverage:**
To see library coverage excluding examples:

```bash
go test $(go list ./... | grep -v '/examples/') -coverprofile=coverage.out -covermode=atomic
```

*Note: Examples are build-tested via `go test ./examples/...` to ensure they remain functional.*
