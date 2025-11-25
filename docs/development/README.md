# Development Guide

This guide covers local development setup, testing, and contributing to Protoflow.

## Prerequisites

- **Go 1.23+**: Required for generics and standard library features
- **Task**: Optional but recommended for running common commands

Install Task:
```bash
# macOS
brew install go-task

# Linux
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

# Or see https://taskfile.dev/installation/
```

## Taskfile Commands

| Command | Description |
|:--------|:------------|
| `task lint` | Run golangci-lint |
| `task test` | Run full test suite |
| `task examples:simple` | Run the simple example |
| `task examples:json` | Run the JSON example |
| `task examples:proto` | Run the proto example |
| `task examples:full` | Run the full example |

Run `task --list` to see all available commands.

## Running Tests

### Full Test Suite

```bash
task test
# OR
go test ./...
```

### Specific Package

```bash
go test ./internal/runtime/...
go test ./internal/runtime/transport/...
go test ./internal/runtime/handlers/...
```

### With Coverage

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Coverage Excluding Examples

```bash
go test $(go list ./... | grep -v '/examples/') -coverprofile=coverage.out -covermode=atomic
```

## Project Structure

```
protoflow/
├── libapi.go              # Public API exports
├── doc.go                 # Package documentation
├── docs/                  # Documentation
│   ├── configuration/
│   ├── development/
│   └── handlers/
├── examples/              # Runnable examples
│   ├── simple/
│   ├── json/
│   ├── proto/
│   └── full/
└── internal/runtime/      # Implementation
    ├── config/            # Configuration
    ├── errors/            # Sentinel errors
    ├── handlers/          # Handler implementations
    ├── ids/               # ULID generation
    ├── jsoncodec/         # JSON encoding
    ├── logging/           # Logger abstractions
    ├── metadata/          # Metadata handling
    └── transport/         # Transport implementations
```

## Local Broker Setup

### Go Channels (No Setup)

```go
cfg := &protoflow.Config{
    PubSubSystem: "channel",
}
```

### Kafka (Docker)

```bash
docker run -d --name kafka \
  -p 9092:9092 \
  -e KAFKA_BROKER_ID=1 \
  -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  confluentinc/cp-kafka:latest
```

### RabbitMQ (Docker)

```bash
docker run -d --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  rabbitmq:3-management
```

### AWS LocalStack

```bash
docker run -d --name localstack \
  -p 4566:4566 \
  -e SERVICES=sns,sqs \
  localstack/localstack
```

Then configure:
```go
cfg := &protoflow.Config{
    PubSubSystem: "aws",
    AWSRegion:    "us-east-1",
    AWSEndpoint:  "http://localhost:4566",
    AWSAccountID: "000000000000",
}
```

### NATS (Docker)

```bash
docker run -d --name nats \
  -p 4222:4222 \
  nats:latest
```

## Code Style

- Run `task lint` (or `golangci-lint run`) before committing
- Follow standard Go conventions
- Keep functions focused and testable
- Add tests for new functionality

## Contributing

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes
4. Run `task lint` and `task test`
5. Submit a pull request

### Good First Issues

- Add tests for uncovered code paths
- Improve documentation
- Add transport-specific integration tests
- Enhance examples

## Debugging Tips

### Enable Debug Logging

```go
handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})
logger := protoflow.NewSlogServiceLogger(slog.New(handler))
```

### Inspect Handler Stats

Enable WebUI to see handler statistics:

```go
cfg := &protoflow.Config{
    WebUIEnabled:            true,
    WebUIPort:               8081,
    WebUICORSAllowedOrigins: []string{"*"},
}
```

Then access: `http://localhost:8081/api/handlers`

### Trace Message Flow

The correlation ID middleware automatically adds tracing:

```go
// Access in handlers
correlationID := evt.CorrelationID()
```
