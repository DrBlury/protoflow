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

// This example demonstrates the SQLite transport with features like:
// - Persistent message queue stored in a local database
// - Delayed message processing (schedule jobs for the future)
// - Dead letter queue management
// - Queue introspection (pending/DLQ counts)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := protoflow.NewSlogServiceLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// SQLite configuration - messages persist to a local file
	cfg := &protoflow.Config{
		PubSubSystem: "sqlite",
		SQLiteFile:   "example_queue.db", // File-based persistence
		PoisonQueue:  "sqlite.poison",
		// Note: For testing, use ":memory:" for an in-memory database
	}

	svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{})

	// Register a handler for immediate jobs
	err := protoflow.RegisterMessageHandler(svc, protoflow.MessageHandlerRegistration{
		Name:         "immediate-processor",
		ConsumeQueue: "sqlite.jobs",
		Handler:      createHandler(logger, "immediate"),
	})
	if err != nil {
		panic(err)
	}

	// Register a handler for delayed/scheduled jobs
	err = protoflow.RegisterMessageHandler(svc, protoflow.MessageHandlerRegistration{
		Name:         "scheduled-processor",
		ConsumeQueue: "sqlite.scheduled",
		Handler:      createHandler(logger, "scheduled"),
	})
	if err != nil {
		panic(err)
	}

	// Publish some sample messages
	go publishExampleMessages(ctx, svc, logger)

	// Run for a bit to demonstrate features
	go func() {
		time.Sleep(10 * time.Second)
		cancel()
	}()

	if err := svc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("service stopped with error", err, nil)
	}

	// Clean up the database file
	os.Remove("example_queue.db")
	os.Remove("example_queue.db-wal")
	os.Remove("example_queue.db-shm")

	logger.Info("SQLite example completed", nil)
}

func createHandler(logger protoflow.ServiceLogger, handlerType string) func(msg *message.Message) ([]*message.Message, error) {
	return func(msg *message.Message) ([]*message.Message, error) {
		logger.Info(fmt.Sprintf("📥 [%s] Processing message", handlerType), protoflow.LogFields{
			"message_id":   msg.UUID,
			"payload":      string(msg.Payload),
			"processed_at": time.Now().Format(time.RFC3339),
		})
		return nil, nil
	}
}

func publishExampleMessages(ctx context.Context, svc *protoflow.Service, logger protoflow.ServiceLogger) {
	time.Sleep(500 * time.Millisecond) // Wait for router to start

	// 1. Publish immediate jobs
	logger.Info("📤 Publishing immediate jobs...", nil)
	for i := 0; i < 3; i++ {
		msg := message.NewMessage(protoflow.CreateULID(), []byte(fmt.Sprintf("immediate-job-%d", i)))
		msg.Metadata = message.Metadata{
			"event_source":         "sqlite_example",
			"event_message_schema": "sqlite.JobV1",
		}
		if err := svc.Publish(ctx, "sqlite.jobs", msg); err != nil {
			logger.Error("failed to publish", err, nil)
		}
	}

	// 2. Publish delayed jobs - these will only be processed after the delay
	logger.Info("📤 Publishing delayed jobs (will process in 3 seconds)...", nil)
	for i := 0; i < 2; i++ {
		msg := message.NewMessage(protoflow.CreateULID(), []byte(fmt.Sprintf("scheduled-job-%d", i)))
		msg.Metadata = message.Metadata{
			"event_source":         "sqlite_example",
			"event_message_schema": "sqlite.ScheduledJobV1",
			// The magic: set protoflow_delay to schedule future processing
			"protoflow_delay": "3s", // Will be available for processing after 3 seconds
		}
		if err := svc.Publish(ctx, "sqlite.scheduled", msg); err != nil {
			logger.Error("failed to publish", err, nil)
		}
	}

	logger.Info("⏰ Immediate jobs will process now, scheduled jobs in ~3 seconds", nil)

	// Wait and show that delayed jobs are processed later
	time.Sleep(5 * time.Second)
}
