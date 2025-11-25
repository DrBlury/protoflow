package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/drblury/protoflow"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// This example demonstrates DLQ (Dead Letter Queue) metrics collection.
// It shows how to:
// - Track messages sent to the DLQ using protoflow.DLQMetrics
// - Monitor DLQ depth per topic
// - Track message replays and purges
// - Export metrics to Prometheus

var (
	// Simulated error for demonstration
	ErrProcessingFailed = errors.New("message processing failed")
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := protoflow.NewSlogServiceLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Create DLQ metrics collector using protoflow's built-in type
	dlqMetrics := protoflow.NewDLQMetrics(prometheus.DefaultRegisterer)
	dlqMetrics.Register()

	// Start Prometheus metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logger.Info("📊 Prometheus metrics available at http://localhost:2112/metrics", nil)
		if err := http.ListenAndServe(":2112", nil); err != nil {
			logger.Error("metrics server error", err, nil)
		}
	}()

	cfg := &protoflow.Config{
		PubSubSystem: "channel",
		PoisonQueue:  "dlq.poison",
	}

	// Build middleware chain with DLQ tracking
	middlewares := []protoflow.MiddlewareRegistration{
		protoflow.CorrelationIDMiddleware(),
		dlqTrackingMiddleware(dlqMetrics), // Track messages going to DLQ
		protoflow.RetryMiddleware(protoflow.RetryMiddlewareConfig{
			MaxRetries:      2,
			InitialInterval: 100 * time.Millisecond,
		}),
		protoflow.PoisonQueueMiddleware(nil),
		protoflow.RecovererMiddleware(),
	}

	svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{
		DisableDefaultMiddlewares: true,
		Middlewares:               middlewares,
	})

	// Register a handler that occasionally fails
	err := protoflow.RegisterMessageHandler(svc, protoflow.MessageHandlerRegistration{
		Name:         "flaky-processor",
		ConsumeQueue: "dlq.incoming",
		Handler:      createFlakyHandler(logger),
	})
	if err != nil {
		panic(err)
	}

	// Publish test messages
	go publishTestMessages(ctx, svc, logger)

	// Periodically print DLQ metrics
	go printMetricsPeriodically(ctx, dlqMetrics, logger)

	// Run for demonstration
	go func() {
		time.Sleep(15 * time.Second)
		cancel()
	}()

	if err := svc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("service stopped with error", err, nil)
	}

	// Print final metrics
	printSnapshot(dlqMetrics, logger)
}

// createFlakyHandler creates a handler that fails for certain messages
func createFlakyHandler(logger protoflow.ServiceLogger) func(msg *message.Message) ([]*message.Message, error) {
	return func(msg *message.Message) ([]*message.Message, error) {
		// Simulate failures for messages marked as "should_fail"
		if msg.Metadata.Get("should_fail") == "true" {
			logger.Info("❌ Handler failing (simulated)", protoflow.LogFields{
				"message_id": msg.UUID,
			})
			return nil, ErrProcessingFailed
		}

		logger.Info("✅ Handler succeeded", protoflow.LogFields{
			"message_id": msg.UUID,
		})
		return nil, nil
	}
}

func publishTestMessages(ctx context.Context, svc *protoflow.Service, logger protoflow.ServiceLogger) {
	time.Sleep(500 * time.Millisecond)

	logger.Info("📤 Publishing test messages (some will fail)...", nil)

	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg := message.NewMessage(protoflow.CreateULID(), []byte(fmt.Sprintf("message-%d", i)))
		msg.Metadata = message.Metadata{
			"event_source":         "dlq_metrics_example",
			"event_message_schema": "dlq.TestMessageV1",
			"protoflow_handler":    "flaky-processor",
			"protoflow_topic":      "dlq.incoming",
		}

		// Make every 3rd message fail
		if i%3 == 0 {
			msg.Metadata.Set("should_fail", "true")
		}

		if err := svc.Publish(ctx, "dlq.incoming", msg); err != nil {
			logger.Error("failed to publish", err, nil)
		}

		time.Sleep(300 * time.Millisecond)
	}
}

func printMetricsPeriodically(ctx context.Context, metrics *protoflow.DLQMetrics, logger protoflow.ServiceLogger) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			printSnapshot(metrics, logger)
		}
	}
}

func printSnapshot(metrics *protoflow.DLQMetrics, logger protoflow.ServiceLogger) {
	snapshot := metrics.GetSnapshot()
	logger.Info("📊 DLQ Metrics Snapshot", protoflow.LogFields{
		"total_messages": snapshot.TotalMessages,
		"total_replayed": snapshot.TotalReplayed,
		"total_purged":   snapshot.TotalPurged,
		"topics_tracked": len(snapshot.TopicMetrics),
	})

	for topic, tm := range snapshot.TopicMetrics {
		logger.Info(fmt.Sprintf("   Topic: %s", topic), protoflow.LogFields{
			"current":  tm.MessagesCurrent,
			"received": tm.MessagesReceived,
			"replayed": tm.MessagesReplayed,
		})
	}
}

// dlqTrackingMiddleware creates middleware that tracks messages going to DLQ
func dlqTrackingMiddleware(metrics *protoflow.DLQMetrics) protoflow.MiddlewareRegistration {
	return protoflow.MiddlewareRegistration{
		Name: "dlq_tracking",
		Builder: func(s *protoflow.Service) (message.HandlerMiddleware, error) {
			return func(h message.HandlerFunc) message.HandlerFunc {
				return func(msg *message.Message) ([]*message.Message, error) {
					msgs, err := h(msg)

					// Track if message was sent to DLQ (poison queue middleware sets this)
					if msg.Metadata.Get("protoflow_dlq_sent") == "true" {
						topic := msg.Metadata.Get("protoflow_topic")
						handler := msg.Metadata.Get("protoflow_handler")
						metrics.RecordMessageToDLQ(topic, handler, 0, 0)
					}

					return msgs, err
				}
			}, nil
		},
	}
}
