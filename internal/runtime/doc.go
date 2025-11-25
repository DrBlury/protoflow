/*
Package runtime provides the core event processing infrastructure for protoflow.

# Architecture Overview

The runtime package implements a message-driven architecture built on top of
Watermill. It provides typed handlers for Protocol Buffers and JSON messages,
along with a middleware chain for cross-cutting concerns.

# Package Structure

The runtime package is organized into the following components:

## Core Service (service.go)

The Service struct is the central orchestrator that wires together:
  - Message router (Watermill)
  - Publisher and subscriber connections
  - Middleware chain
  - HTTP servers for metrics and WebUI
  - Proto message registry for validation

## Handler Registration (registration*.go)

Handler registration files provide typed wrappers for message handlers:
  - registration.go: Raw Watermill handlers and base registration logic
  - registration_json.go: Typed JSON message handlers
  - registration_proto.go: Typed Protocol Buffer message handlers

## Middleware (middleware.go)

The middleware system provides composable message processing stages:
  - CorrelationID: Ensures message traceability
  - LogMessages: Debug logging of message payloads
  - ProtoValidate: Schema validation for protobuf messages
  - Outbox: Transactional outbox pattern support
  - Tracer: OpenTelemetry distributed tracing
  - Metrics: Prometheus metrics collection
  - Retry: Exponential backoff retry logic
  - PoisonQueue: Dead letter queue for failed messages
  - Recoverer: Panic recovery

## Stats & Monitoring (models.go, resources.go)

Extended metrics collection for handler performance:
  - Latency percentiles (p50, p95, p99)
  - Throughput tracking
  - Error categorization
  - Resource usage sampling
  - Backlog estimation

## Publishing (publisher.go)

Utilities for emitting proto-based events with proper metadata.

## WebUI (webui.go)

HTTP API for introspecting handler state and statistics.

# Sub-packages

  - config/: Service configuration with validation
  - errors/: Sentinel errors and error types
  - handlers/: Message context types and handler building
  - ids/: ULID generation for message IDs
  - jsoncodec/: JSON marshaling utilities
  - logging/: Logger interface and adapters
  - metadata/: Message metadata utilities
  - transport/: Pub/sub transport implementations (Kafka, RabbitMQ, AWS, NATS, etc.)

# Usage Example

	cfg := &protoflow.Config{
		PubSubSystem:   "kafka",
		KafkaBrokers:   []string{"localhost:9092"},
		MetricsEnabled: true,
		MetricsPort:    9090,
	}

	svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{})

	protoflow.RegisterProtoHandler(svc, protoflow.ProtoHandlerRegistration[*pb.OrderCreated]{
		Name:         "order-processor",
		ConsumeQueue: "orders.created",
		PublishQueue: "orders.processed",
		Handler:      processOrder,
	})

	svc.Start(ctx)
*/
package runtime
