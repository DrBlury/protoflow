// Package protoflow exposes a small framework on top of Watermill that wires message
// routers, publishers, subscribers, and middleware for protobuf and JSON driven
// services. It selects the desired Pub/Sub transport (Kafka, RabbitMQ, or AWS
// SNS/SQS) from Config, bootstraps the Watermill router, and registers a
// middleware chain that adds correlation IDs, logging, protobuf validation,
// outbox persistence, tracing, retries, and poison queue forwarding.
//
// The Service type hosts the router and offers helpers for registering typed
// handlers via RegisterProtoHandler or RegisterJSONHandler. Typed registrations
// automatically marshal/unmarshal payloads, clone metadata safely, and can
// validate outgoing protobuf events when a ProtoValidator is provided. Service
// also exposes PublishProto so your HTTP or RPC handlers can emit events without
// touching the lower-level Watermill APIs.
//
// Advanced users can extend the default middleware chain, plug in an OutboxStore
// to persist outgoing messages, or provide a ProtoValidator to enforce message
// contracts. See README.md for a full walkthrough.
package protoflow
