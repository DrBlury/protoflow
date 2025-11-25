package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/drblury/protoflow"
)

// This example demonstrates how to use job lifecycle hooks (OnJobStart, OnJobDone, OnJobError)
// to add custom behavior around handler execution.

var (
	ErrSimulatedFailure = errors.New("simulated failure for demonstration")
	jobCounter          atomic.Int64
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := protoflow.NewSlogServiceLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := &protoflow.Config{
		PubSubSystem: "channel", // In-memory for this example
		PoisonQueue:  "hooks.poison",
	}

	// Create custom hooks for this application
	customHooks := createApplicationHooks(logger)

	// Build middleware chain with our hooks
	middlewares := []protoflow.MiddlewareRegistration{
		protoflow.CorrelationIDMiddleware(),
		protoflow.LogMessagesMiddleware(nil),
		protoflow.JobHooksMiddleware(customHooks), // Add our custom hooks
		protoflow.RetryMiddleware(protoflow.RetryMiddlewareConfig{
			MaxRetries:      2,
			InitialInterval: 100 * time.Millisecond,
		}),
		protoflow.RecovererMiddleware(),
	}

	svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{
		DisableDefaultMiddlewares: true,
		Middlewares:               middlewares,
	})

	// Register a handler that processes jobs
	err := protoflow.RegisterMessageHandler(svc, protoflow.MessageHandlerRegistration{
		Name:         "job-processor",
		ConsumeQueue: "hooks.jobs",
		PublishQueue: "hooks.results",
		Handler:      createJobHandler(logger),
	})
	if err != nil {
		panic(err)
	}

	// Start a goroutine to publish sample messages
	go publishSampleJobs(ctx, svc)

	// Run until we've processed a few messages
	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()

	if err := svc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("service stopped with error", err, nil)
	}

	logger.Info("Example completed", protoflow.LogFields{
		"total_jobs_processed": jobCounter.Load(),
	})
}

// createApplicationHooks demonstrates how to create custom hooks for your application.
func createApplicationHooks(logger protoflow.ServiceLogger) protoflow.JobHooks {
	// Track job metrics in memory (in production, you'd use Prometheus)
	var (
		startedJobs   atomic.Int64
		completedJobs atomic.Int64
		failedJobs    atomic.Int64
	)

	return protoflow.JobHooks{
		OnJobStart: func(ctx protoflow.JobContext) {
			count := startedJobs.Add(1)
			logger.Info("🚀 Job started", protoflow.LogFields{
				"job_number":   count,
				"message_uuid": ctx.MessageUUID,
				"handler":      ctx.HandlerName,
				"topic":        ctx.Topic,
				"retry_count":  ctx.RetryCount,
			})

			// Example: You could notify a job tracking system here
			// jobTracker.RecordStart(ctx.MessageUUID)
		},

		OnJobDone: func(ctx protoflow.JobContext) {
			count := completedJobs.Add(1)
			logger.Info("✅ Job completed successfully", protoflow.LogFields{
				"completed_count": count,
				"message_uuid":    ctx.MessageUUID,
				"duration_ms":     ctx.Duration.Milliseconds(),
			})

			// Example: Update job status in database
			// db.UpdateJobStatus(ctx.MessageUUID, "completed")
		},

		OnJobError: func(ctx protoflow.JobContext, err error) {
			count := failedJobs.Add(1)
			logger.Error("❌ Job failed", err, protoflow.LogFields{
				"failed_count": count,
				"message_uuid": ctx.MessageUUID,
				"retry_count":  ctx.RetryCount,
				"duration_ms":  ctx.Duration.Milliseconds(),
			})

			// Example: Send alert for critical failures
			if ctx.RetryCount >= 2 {
				logger.Info("🚨 ALERT: Job exhausted retries", protoflow.LogFields{
					"message_uuid": ctx.MessageUUID,
				})
				// alerting.SendSlackNotification(...)
			}
		},
	}
}

// createJobHandler creates a handler that simulates job processing
func createJobHandler(logger protoflow.ServiceLogger) func(msg *message.Message) ([]*message.Message, error) {
	return func(msg *message.Message) ([]*message.Message, error) {
		jobCounter.Add(1)

		// Simulate some work
		time.Sleep(50 * time.Millisecond)

		// Simulate occasional failures (every 3rd message)
		if msg.Metadata.Get("simulate_error") == "true" {
			return nil, ErrSimulatedFailure
		}

		// Return a result message
		result := message.NewMessage(protoflow.CreateULID(), []byte(fmt.Sprintf("processed: %s", msg.UUID)))
		result.Metadata = message.Metadata{
			"event_source":         "hooks_example",
			"event_message_schema": "hooks.ResultV1",
			"original_message_id":  msg.UUID,
		}
		return []*message.Message{result}, nil
	}
}

func publishSampleJobs(ctx context.Context, svc *protoflow.Service) {
	time.Sleep(500 * time.Millisecond) // Wait for router to start

	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg := message.NewMessage(protoflow.CreateULID(), []byte(fmt.Sprintf("job-%d", i)))
		msg.Metadata = message.Metadata{
			"event_source":         "hooks_example",
			"event_message_schema": "hooks.JobV1",
			"protoflow_handler":    "job-processor",
			"protoflow_topic":      "hooks.jobs",
		}

		// Make every 3rd message fail
		if i%3 == 2 {
			msg.Metadata.Set("simulate_error", "true")
		}

		if err := svc.Publish(ctx, "hooks.jobs", msg); err != nil {
			slog.Error("failed to publish", "error", err)
		}

		time.Sleep(200 * time.Millisecond)
	}
}
