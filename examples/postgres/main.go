// Package main demonstrates using the PostgreSQL transport with protoflow.
//
// PostgreSQL provides a robust, production-ready message queue with features like:
// - Delayed messages via metadata
// - Dead letter queue (DLQ) for failed messages
// - Automatic retry with exponential backoff
// - SKIP LOCKED for efficient concurrent consumers
//
// Start PostgreSQL:
//
//	docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=protoflow postgres:16
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/drblury/protoflow"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := protoflow.NewSlogServiceLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Get PostgreSQL URL from environment or use default
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgres://postgres:postgres@localhost:5432/protoflow?sslmode=disable"
	}

	// PostgreSQL configuration - messages persist to a PostgreSQL database
	cfg := &protoflow.Config{
		PubSubSystem: "postgres",
		PostgresURL:  postgresURL,
		PoisonQueue:  "postgres.poison",
	}

	svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{})

	// Register a handler for orders
	err := protoflow.RegisterMessageHandler(svc, protoflow.MessageHandlerRegistration{
		Name:         "order-processor",
		ConsumeQueue: "postgres.orders",
		Handler:      createOrderHandler(logger),
	})
	if err != nil {
		panic(err)
	}

	// Publish some sample messages
	go publishExampleMessages(ctx, svc, logger)

	// Run for a bit to demonstrate features
	go func() {
		time.Sleep(15 * time.Second)
		cancel()
	}()

	if err := svc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("service stopped with error", err, nil)
	}

	logger.Info("PostgreSQL example completed", nil)
}

func createOrderHandler(logger protoflow.ServiceLogger) func(msg *message.Message) ([]*message.Message, error) {
	return func(msg *message.Message) ([]*message.Message, error) {
		logger.Info("📥 Processing order", protoflow.LogFields{
			"uuid":    msg.UUID,
			"payload": string(msg.Payload),
		})

		// Simulate some processing
		time.Sleep(100 * time.Millisecond)

		// Simulate failures for messages containing "fail"
		if string(msg.Payload) == "fail-order" {
			return nil, fmt.Errorf("simulated order processing failure")
		}

		logger.Info("✅ Order processed successfully", protoflow.LogFields{
			"uuid": msg.UUID,
		})
		return nil, nil
	}
}

func publishExampleMessages(ctx context.Context, svc *protoflow.Service, logger protoflow.ServiceLogger) {
	time.Sleep(500 * time.Millisecond) // Wait for service to start

	// Publish immediate orders
	orders := []string{"order-001", "order-002", "fail-order", "order-003"}
	for _, order := range orders {
		msg := message.NewMessage(order, []byte(order))
		if err := svc.Publish(ctx, "postgres.orders", msg); err != nil {
			logger.Error("Failed to publish order", err, protoflow.LogFields{"order": order})
		} else {
			logger.Info("📤 Published order", protoflow.LogFields{"order": order})
		}
	}

	// Publish a delayed order (will be processed in 5 seconds)
	logger.Info("📤 Publishing delayed order (5s delay)", nil)
	delayedMsg := message.NewMessage("delayed-order", []byte("delayed-order-data"))
	delayedMsg.Metadata.Set("protoflow_delay", "5s")
	if err := svc.Publish(ctx, "postgres.orders", delayedMsg); err != nil {
		logger.Error("Failed to publish delayed order", err, nil)
	}
}
