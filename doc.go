// Package protoflow is a small layer on top of Watermill that wires routers,
// publishers, subscribers, and middleware for protobuf- or JSON-driven services.
// It reads the target transport (Kafka, RabbitMQ, or AWS SNS/SQS) from Config,
// bootstraps the Watermill router, and registers the default middleware chain
// for correlation IDs, logging, validation, outbox persistence, tracing,
// retries, and poison queue forwarding.
//
// Service hosts the router and exposes typed helpers: RegisterProtoHandler and
// RegisterJSONHandler take care of marshaling, metadata cloning, and optional
// protobuf validation, while Service.PublishProto lets HTTP/RPC handlers emit
// events without touching low-level Watermill APIs. A minimal setup therefore involves
// filling Config, creating a Service, registering handlers, and calling Start;
// see README.md for a copy/paste quick start snippet.
//
// When you need more control, ServiceDependencies exposes well-scoped hooks:
// bring your own OutboxStore, ProtoValidator, middleware registrations, or even
// an entire TransportFactory to plug in custom brokers. The README organises
// these knobs by topic so you can dive into the exact setting you want to
// adjust without rereading the whole guide.
package protoflow
