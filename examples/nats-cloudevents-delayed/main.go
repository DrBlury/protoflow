package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/drblury/protoflow"
)

// This example demonstrates CloudEvents with NATS JetStream including:
// - Publishing CloudEvents with the new API
// - Delayed message delivery
// - Automatic retry with exponential backoff
// - Dead letter queue handling
// - Tracing context propagation

// OrderData represents the payload of an order event.
type OrderData struct {
	OrderID    string  `json:"order_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
}

// PaymentData represents the payload of a payment event.
type PaymentData struct {
	PaymentID string  `json:"payment_id"`
	OrderID   string  `json:"order_id"`
	Status    string  `json:"status"`
	Amount    float64 `json:"amount"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := protoflow.NewSlogServiceLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	// Configure NATS JetStream
	cfg := &protoflow.Config{
		PubSubSystem: "nats-jetstream",
		NATSURL:      getEnv("NATS_URL", "nats://localhost:4222"),
		PoisonQueue:  "protoflow.dlq",
	}

	svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{})

	// Register CloudEvents handlers
	registerHandlers(svc, logger)

	// Publish example events
	go publishExampleEvents(ctx, svc, logger)

	// Start the service
	logger.Info("Starting NATS JetStream CloudEvents example...", nil)
	if err := svc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Service stopped with error", err, nil)
	}

	logger.Info("Example completed", nil)
}

func registerHandlers(svc *protoflow.Service, logger protoflow.ServiceLogger) {
	// Handler for order.created events
	err := svc.ConsumeEvents("order.created", func(ctx context.Context, evt protoflow.Event) error {
		logger.Info("Processing order.created event", protoflow.LogFields{
			"event_id":   evt.ID,
			"event_type": evt.Type,
			"source":     evt.Source,
			"attempt":    protoflow.GetAttempt(evt),
		})

		// Parse the order data
		var order OrderData
		if data, ok := evt.Data.(map[string]any); ok {
			order.OrderID, _ = data["order_id"].(string)
			order.CustomerID, _ = data["customer_id"].(string)
			order.Amount, _ = data["amount"].(float64)
			order.Currency, _ = data["currency"].(string)
		}

		logger.Info("Order details", protoflow.LogFields{
			"order_id":    order.OrderID,
			"customer_id": order.CustomerID,
			"amount":      order.Amount,
			"currency":    order.Currency,
		})

		// Simulate some processing
		time.Sleep(100 * time.Millisecond)

		// Publish a payment event in response
		paymentEvt := protoflow.NewCloudEvent("payment.initiated", "order-service", PaymentData{
			PaymentID: protoflow.NewEventID(),
			OrderID:   order.OrderID,
			Status:    "pending",
			Amount:    order.Amount,
		})
		// Copy tracing context
		protoflow.CopyTracingContext(evt, &paymentEvt)

		if err := svc.PublishEvent(ctx, paymentEvt); err != nil {
			logger.Error("Failed to publish payment event", err, nil)
		}

		return nil // Acknowledge
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register order.created handler: %v", err))
	}

	// Handler for payment.initiated events
	err = svc.ConsumeEvents("payment.initiated", func(ctx context.Context, evt protoflow.Event) error {
		logger.Info("Processing payment.initiated event", protoflow.LogFields{
			"event_id":   evt.ID,
			"event_type": evt.Type,
			"attempt":    protoflow.GetAttempt(evt),
		})

		// Simulate payment processing with occasional failures
		attempt := protoflow.GetAttempt(evt)
		if attempt < 2 {
			logger.Info("Payment processing failed, will retry", protoflow.LogFields{
				"attempt": attempt,
			})
			// Return retry error with explicit delay
			return protoflow.ErrRetryAfter(2*time.Second, fmt.Errorf("payment gateway timeout"))
		}

		logger.Info("Payment processed successfully", protoflow.LogFields{
			"event_id": evt.ID,
		})

		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register payment.initiated handler: %v", err))
	}

	// Handler for delayed notifications
	err = svc.ConsumeEvents("notification.scheduled", func(ctx context.Context, evt protoflow.Event) error {
		logger.Info("Processing scheduled notification", protoflow.LogFields{
			"event_id":   evt.ID,
			"event_type": evt.Type,
			"delay_ms":   protoflow.GetDelayMs(evt),
		})
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register notification.scheduled handler: %v", err))
	}

	// Handler for events that should go to DLQ
	err = svc.ConsumeEvents("order.problematic", func(ctx context.Context, evt protoflow.Event) error {
		logger.Info("This handler always fails for demonstration", protoflow.LogFields{
			"event_id": evt.ID,
			"attempt":  protoflow.GetAttempt(evt),
		})

		// Simulate a permanently failing scenario
		if protoflow.GetAttempt(evt) >= protoflow.GetMaxAttempts(evt) {
			return protoflow.ErrDeadLetterWithReason("order validation permanently failed", nil)
		}

		return protoflow.ErrRetry
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register order.problematic handler: %v", err))
	}
}

func publishExampleEvents(ctx context.Context, svc *protoflow.Service, logger protoflow.ServiceLogger) {
	// Wait for the service to start
	time.Sleep(2 * time.Second)

	logger.Info("Publishing example CloudEvents...", nil)

	// 1. Publish a regular order event
	orderEvt := protoflow.NewCloudEvent("order.created", "api-gateway", OrderData{
		OrderID:    "ORD-001",
		CustomerID: "CUST-123",
		Amount:     99.99,
		Currency:   "USD",
	})
	protoflow.SetCorrelationID(&orderEvt, "req-abc-123")
	protoflow.SetTraceID(&orderEvt, "trace-xyz-789")

	if err := svc.PublishEvent(ctx, orderEvt); err != nil {
		logger.Error("Failed to publish order event", err, nil)
	} else {
		logger.Info("Published order.created event", protoflow.LogFields{
			"event_id": orderEvt.ID,
		})
	}

	// 2. Publish a delayed notification (5 seconds delay)
	notifEvt := protoflow.NewCloudEvent("notification.scheduled", "order-service", map[string]any{
		"message":   "Your order has been confirmed!",
		"recipient": "customer@example.com",
	})
	if err := svc.PublishEventAfter(ctx, notifEvt, 5*time.Second); err != nil {
		logger.Error("Failed to publish delayed notification", err, nil)
	} else {
		logger.Info("Published delayed notification (5s)", protoflow.LogFields{
			"event_id": notifEvt.ID,
		})
	}

	// 3. Publish a problematic event that will end up in DLQ
	problemEvt := protoflow.NewCloudEvent("order.problematic", "api-gateway", map[string]any{
		"order_id": "ORD-BAD",
		"error":    "This order has invalid data",
	})
	protoflow.SetMaxAttempts(&problemEvt, 2) // Only allow 2 attempts

	if err := svc.PublishEvent(ctx, problemEvt); err != nil {
		logger.Error("Failed to publish problematic event", err, nil)
	} else {
		logger.Info("Published problematic event (will go to DLQ)", protoflow.LogFields{
			"event_id": problemEvt.ID,
		})
	}

	// 4. Use the convenience PublishData API
	if err := svc.PublishData(ctx, "order.updated", "order-service", map[string]any{
		"order_id": "ORD-001",
		"status":   "processing",
	},
		protoflow.WithCorrelationID("req-abc-123"),
		protoflow.WithSubject("orders/ORD-001"),
	); err != nil {
		logger.Error("Failed to publish with PublishData", err, nil)
	} else {
		logger.Info("Published using PublishData convenience API", nil)
	}

	logger.Info("All example events published!", nil)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
