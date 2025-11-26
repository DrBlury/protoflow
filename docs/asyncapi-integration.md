# AsyncAPI Integration

Protoflow supports AsyncAPI for event-driven API documentation and tooling integration.

## Overview

[AsyncAPI](https://www.asyncapi.com/) is a specification for defining asynchronous APIs. Protoflow can:

- Validate event types against AsyncAPI definitions
- Generate Go structs from JSON Schema
- Auto-discover topics from AsyncAPI files
- Provide runtime validation

## AsyncAPI Document Structure

```yaml
asyncapi: "2.6.0"
info:
  title: Order Events API
  version: "1.0.0"
  description: Event-driven order processing API

servers:
  production:
    url: nats://events.example.com:4222
    protocol: nats
    description: Production NATS server

defaultContentType: application/cloudevents+json

channels:
  order.created:
    description: New order has been placed
    subscribe:
      operationId: onOrderCreated
      message:
        $ref: "#/components/messages/OrderCreated"

  order.confirmed:
    description: Order has been confirmed
    subscribe:
      message:
        $ref: "#/components/messages/OrderConfirmed"

  payment.processed:
    description: Payment has been processed
    publish:
      message:
        $ref: "#/components/messages/PaymentProcessed"

components:
  messages:
    OrderCreated:
      name: OrderCreated
      title: Order Created Event
      contentType: application/json
      traits:
        - $ref: "#/components/messageTraits/CloudEventHeaders"
      payload:
        $ref: "#/components/schemas/OrderPayload"

    OrderConfirmed:
      name: OrderConfirmed
      title: Order Confirmed Event
      contentType: application/json
      traits:
        - $ref: "#/components/messageTraits/CloudEventHeaders"
      payload:
        $ref: "#/components/schemas/OrderConfirmPayload"

    PaymentProcessed:
      name: PaymentProcessed
      title: Payment Processed Event
      contentType: application/json
      payload:
        $ref: "#/components/schemas/PaymentPayload"

  schemas:
    OrderPayload:
      type: object
      required:
        - order_id
        - customer_id
        - items
        - total
      properties:
        order_id:
          type: string
          format: uuid
          description: Unique order identifier
        customer_id:
          type: string
          description: Customer identifier
        items:
          type: array
          items:
            $ref: "#/components/schemas/OrderItem"
        total:
          type: number
          format: float
          description: Order total amount
        currency:
          type: string
          default: USD

    OrderItem:
      type: object
      properties:
        sku:
          type: string
        quantity:
          type: integer
          minimum: 1
        price:
          type: number

    OrderConfirmPayload:
      type: object
      properties:
        order_id:
          type: string
        confirmed_at:
          type: string
          format: date-time
        estimated_delivery:
          type: string
          format: date-time

    PaymentPayload:
      type: object
      properties:
        payment_id:
          type: string
        order_id:
          type: string
        status:
          type: string
          enum: [pending, completed, failed]
        amount:
          type: number

  messageTraits:
    CloudEventHeaders:
      headers:
        type: object
        properties:
          ce_specversion:
            type: string
            const: "1.0"
          ce_type:
            type: string
          ce_source:
            type: string
          ce_id:
            type: string
            format: uuid
          ce_time:
            type: string
            format: date-time
```

## Using AsyncAPI with Protoflow

### Loading AsyncAPI Definitions

```go
import "github.com/drblury/protoflow/asyncapi"

// Load AsyncAPI document
spec, err := asyncapi.LoadFile("asyncapi.yaml")
if err != nil {
    log.Fatal(err)
}

// Get available channels
channels := spec.Channels()
for name, channel := range channels {
    fmt.Printf("Channel: %s - %s\n", name, channel.Description)
}
```

### Validating Event Types

```go
// Validate that an event type exists in the spec
if !spec.HasChannel("order.created") {
    log.Warn("Unknown event type: order.created")
}

// Validate event payload against schema
evt := protoflow.NewCloudEvent("order.created", "order-service", orderData)
if err := spec.ValidatePayload("order.created", evt.Data); err != nil {
    log.Error("Invalid payload", err)
}
```

### Auto-Registering Handlers

```go
// Register handlers for all subscribed channels
for channelName, channel := range spec.Channels() {
    if channel.Subscribe != nil {
        err := svc.ConsumeEvents(channelName, createHandler(channelName))
        if err != nil {
            log.Fatal(err)
        }
    }
}
```

## Generating Go Types

Use `asyncapi-codegen` or similar tools to generate Go structs:

```bash
# Using asyncapi-codegen
asyncapi-codegen -i asyncapi.yaml -o events/types.go -p events
```

Generated code:

```go
package events

type OrderPayload struct {
    OrderID    string      `json:"order_id"`
    CustomerID string      `json:"customer_id"`
    Items      []OrderItem `json:"items"`
    Total      float64     `json:"total"`
    Currency   string      `json:"currency,omitempty"`
}

type OrderItem struct {
    SKU      string  `json:"sku"`
    Quantity int     `json:"quantity"`
    Price    float64 `json:"price"`
}

type PaymentPayload struct {
    PaymentID string  `json:"payment_id"`
    OrderID   string  `json:"order_id"`
    Status    string  `json:"status"`
    Amount    float64 `json:"amount"`
}
```

## CloudEvents Mapping

Map AsyncAPI channels to CloudEvents event types:

| AsyncAPI Channel | CloudEvents Type |
|-----------------|------------------|
| `order.created` | `order.created` |
| `order.confirmed` | `order.confirmed` |
| `payment.processed` | `payment.processed` |

The channel name becomes the CloudEvents `type` attribute.

## Topic Validation

Validate that configured topics exist:

```go
func validateTopics(spec *asyncapi.Spec, cfg *protoflow.Config) error {
    // Get topics from config or handler registrations
    topics := []string{"order.created", "payment.processed"}

    for _, topic := range topics {
        if !spec.HasChannel(topic) {
            return fmt.Errorf("topic %q not defined in AsyncAPI spec", topic)
        }
    }
    return nil
}
```

## Runtime Schema Validation

Enable runtime payload validation:

```go
svc.ConsumeEvents("order.created", func(ctx context.Context, evt protoflow.Event) error {
    // Validate against AsyncAPI schema
    if err := spec.ValidatePayload("order.created", evt.Data); err != nil {
        log.Error("Schema validation failed", err)
        return protoflow.ErrDeadLetterWithReason("schema validation failed", err)
    }

    // Process valid event...
    return nil
})
```

## Best Practices

1. **Single source of truth**: Keep AsyncAPI document as the contract
2. **Version in event type**: Use `order.created.v2` for breaking changes
3. **Generate types**: Use code generation for type safety
4. **Validate early**: Validate payloads at publish time when possible
5. **Document extensions**: Include protoflow extensions in the spec

## Example: Complete Integration

```go
package main

import (
    "context"
    "log"

    "github.com/drblury/protoflow"
    "github.com/drblury/protoflow/asyncapi"
    "myapp/events" // Generated types
)

func main() {
    ctx := context.Background()

    // Load AsyncAPI spec
    spec, err := asyncapi.LoadFile("asyncapi.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Create service
    cfg := &protoflow.Config{
        PubSubSystem: "nats-jetstream",
        NATSURL:      "nats://localhost:4222",
    }
    svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{})

    // Register handlers from spec
    registerHandlers(svc, spec)

    // Start
    svc.Start(ctx)
}

func registerHandlers(svc *protoflow.Service, spec *asyncapi.Spec) {
    // Handler for order.created
    svc.ConsumeEvents("order.created", func(ctx context.Context, evt protoflow.Event) error {
        // Parse into generated type
        var order events.OrderPayload
        if err := json.Unmarshal(toJSON(evt.Data), &order); err != nil {
            return protoflow.ErrDeadLetterWithReason("invalid payload", err)
        }

        log.Printf("Processing order %s", order.OrderID)

        // Publish confirmation
        confirm := events.OrderConfirmPayload{
            OrderID:     order.OrderID,
            ConfirmedAt: time.Now(),
        }

        confirmEvt := protoflow.NewCloudEvent("order.confirmed", "order-service", confirm)
        return svc.PublishEvent(ctx, confirmEvt)
    })
}
```

## Tools and Resources

- [AsyncAPI Specification](https://www.asyncapi.com/docs/reference/specification/v2.6.0)
- [AsyncAPI Generator](https://github.com/asyncapi/generator)
- [AsyncAPI Studio](https://studio.asyncapi.com/) - Visual editor
- [asyncapi-codegen](https://github.com/lerenn/asyncapi-codegen) - Go code generator
